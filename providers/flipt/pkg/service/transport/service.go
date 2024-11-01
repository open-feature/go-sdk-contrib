package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	offlipt "github.com/open-feature/go-sdk-contrib/providers/flipt/pkg/service"
	of "github.com/open-feature/go-sdk/openfeature"
	flipt "go.flipt.io/flipt/rpc/flipt"
	"go.flipt.io/flipt/rpc/flipt/evaluation"
	sdk "go.flipt.io/flipt/sdk/go"
	sdkgrpc "go.flipt.io/flipt/sdk/go/grpc"
	sdkhttp "go.flipt.io/flipt/sdk/go/http"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	requestID   = "requestID"
	defaultAddr = "http://localhost:8080"
)

// Service is a Transport service.
type Service struct {
	client            offlipt.Client
	address           string
	certificatePath   string
	unaryInterceptors []grpc.UnaryClientInterceptor
	once              sync.Once
	tokenProvider     sdk.ClientTokenProvider
	grpcDialOptions   []grpc.DialOption
}

// Option is a service option.
type Option func(*Service)

// WithAddress sets the address for the remote Flipt gRPC API.
func WithAddress(address string) Option {
	return func(s *Service) {
		s.address = address
	}
}

// WithCertificatePath sets the certificate path for the service.
func WithCertificatePath(certificatePath string) Option {
	return func(s *Service) {
		s.certificatePath = certificatePath
	}
}

// WithUnaryClientInterceptor sets the provided unary client interceptors
// to be applied to the established gRPC client connection.
func WithUnaryClientInterceptor(unaryInterceptors ...grpc.UnaryClientInterceptor) Option {
	return func(s *Service) {
		s.unaryInterceptors = unaryInterceptors
	}
}

// WithClientTokenProvider sets the token provider for auth to support client
// auth needs.
func WithClientTokenProvider(tokenProvider sdk.ClientTokenProvider) Option {
	return func(s *Service) {
		s.tokenProvider = tokenProvider
	}
}

// WithGRPCDialOptions sets the provided DialOption
// to be applied when establishing gRPC client connection.
func WithGRPCDialOptions(dialOptions ...grpc.DialOption) Option {
	return func(s *Service) {
		s.grpcDialOptions = append(s.grpcDialOptions, dialOptions...)
	}
}

// New creates a new Transport service.
func New(opts ...Option) *Service {
	s := &Service{
		address:           defaultAddr,
		unaryInterceptors: []grpc.UnaryClientInterceptor{},
		grpcDialOptions: []grpc.DialOption{
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Service) connect() (*grpc.ClientConn, error) {
	var (
		err         error
		credentials = insecure.NewCredentials()
	)

	if s.certificatePath != "" {
		credentials, err = loadTLSCredentials(s.certificatePath)
		if err != nil {
			// TODO: log error?
			credentials = insecure.NewCredentials()
		}
	}

	address := s.address

	if strings.HasPrefix(s.address, "unix://") {
		address = "passthrough:///" + s.address
	}

	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials),
		grpc.WithChainUnaryInterceptor(s.unaryInterceptors...),
	}
	dialOptions = append(dialOptions, s.grpcDialOptions...)

	conn, err := grpc.NewClient(address, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("dialing %w", err)
	}

	return conn, nil
}

func (s *Service) instance() (offlipt.Client, error) {
	type fclient struct {
		*sdk.Flipt
		*sdk.Evaluation
	}

	if s.client != nil {
		return s.client, nil
	}

	var err error

	s.once.Do(func() {
		u, uerr := url.Parse(s.address)
		if uerr != nil {
			err = fmt.Errorf("connecting %w", uerr)
		}

		opts := []sdk.Option{}

		if s.tokenProvider != nil {
			opts = append(opts, sdk.WithClientTokenProvider(s.tokenProvider))
		}

		hclient := sdk.New(sdkhttp.NewTransport(s.address), opts...)
		if u.Scheme == "https" || u.Scheme == "http" {
			s.client = &fclient{
				hclient.Flipt(),
				hclient.Evaluation(),
			}

			return
		}

		conn, cerr := s.connect()
		if cerr != nil {
			err = fmt.Errorf("connecting %w", cerr)
		}

		gclient := sdk.New(sdkgrpc.NewTransport(conn), opts...)
		s.client = &fclient{
			gclient.Flipt(),
			gclient.Evaluation(),
		}
	})

	return s.client, err
}

// GetFlag returns a flag if it exists for the given namespace/flag key pair.
func (s *Service) GetFlag(ctx context.Context, namespaceKey, flagKey string) (*flipt.Flag, error) {
	conn, err := s.instance()
	if err != nil {
		return nil, err
	}

	flag, err := conn.GetFlag(ctx, &flipt.GetFlagRequest{
		Key:          flagKey,
		NamespaceKey: namespaceKey,
	})
	if err != nil {
		return nil, gRPCToOpenFeatureError(err)
	}

	return flag, nil
}

// Boolean evaluates a boolean type flag with the given context and namespace/flag key pair.
func (s *Service) Boolean(ctx context.Context, namespaceKey, flagKey string, evalCtx map[string]interface{}) (*evaluation.BooleanEvaluationResponse, error) {
	if evalCtx == nil {
		return nil, of.NewInvalidContextResolutionError("evalCtx is nil")
	}

	ec := convertMapInterface(evalCtx)

	targetingKey := ec[of.TargetingKey]
	if targetingKey == "" {
		return nil, of.NewTargetingKeyMissingResolutionError("targetingKey is missing")
	}

	conn, err := s.instance()
	if err != nil {
		return nil, err
	}

	ber, err := conn.Boolean(ctx, &evaluation.EvaluationRequest{FlagKey: flagKey, NamespaceKey: namespaceKey, EntityId: targetingKey, RequestId: ec[requestID], Context: ec})
	if err != nil {
		return nil, gRPCToOpenFeatureError(err)
	}

	return ber, nil
}

// Evaluate evaluates a variant type flag with the given context and namespace/flag key pair.
func (s *Service) Evaluate(ctx context.Context, namespaceKey, flagKey string, evalCtx map[string]interface{}) (*evaluation.VariantEvaluationResponse, error) {
	if evalCtx == nil {
		return nil, of.NewInvalidContextResolutionError("evalCtx is nil")
	}

	ec := convertMapInterface(evalCtx)

	targetingKey := ec[of.TargetingKey]
	if targetingKey == "" {
		return nil, of.NewTargetingKeyMissingResolutionError("targetingKey is missing")
	}

	conn, err := s.instance()
	if err != nil {
		return nil, err
	}

	resp, err := conn.Variant(ctx, &evaluation.EvaluationRequest{FlagKey: flagKey, NamespaceKey: namespaceKey, EntityId: targetingKey, RequestId: ec[requestID], Context: ec})
	if err != nil {
		return nil, gRPCToOpenFeatureError(err)
	}

	return resp, nil
}

func convertMapInterface(m map[string]interface{}) map[string]string {
	ee := make(map[string]string)
	for k, v := range m {
		ee[k] = fmt.Sprintf("%v", v)
	}

	return ee
}

func loadTLSCredentials(serverCertPath string) (credentials.TransportCredentials, error) {
	pemServerCA, err := os.ReadFile(serverCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, errors.New("failed to add server CA's certificate")
	}

	config := &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS12,
	}

	return credentials.NewTLS(config), nil
}

func gRPCToOpenFeatureError(err error) of.ResolutionError {
	s, ok := status.FromError(err)
	if !ok {
		return of.NewGeneralResolutionError("internal error: " + err.Error())
	}

	switch s.Code() {
	case codes.NotFound:
		return of.NewFlagNotFoundResolutionError(s.Message())
	case codes.InvalidArgument:
		return of.NewInvalidContextResolutionError(s.Message())
	case codes.Unavailable:
		return of.NewProviderNotReadyResolutionError(s.Message())
	}

	return of.NewGeneralResolutionError(s.Message())
}
