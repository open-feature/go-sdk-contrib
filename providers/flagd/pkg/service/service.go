package service

import (
	schemaConnectV1 "buf.build/gen/go/open-feature/flagd/bufbuild/connect-go/schema/v1/schemav1connect"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	otelconnect "github.com/bufbuild/connect-opentelemetry-go"
	"github.com/go-logr/logr"
	flagdModels "github.com/open-feature/flagd/core/pkg/model"
	flagdService "github.com/open-feature/flagd/core/pkg/service"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/cache"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"golang.org/x/net/context"
	"net"
	"net/http"
	"os"
	"time"

	of "github.com/open-feature/go-sdk/pkg/openfeature"

	schemaV1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	ReasonCached = "CACHED"
)

type Configuration struct {
	Port            uint16
	Host            string
	CertificatePath string
	SocketPath      string
	TLSEnabled      bool
	OtelInterceptor bool
}

// Service handles the client side  interface for the flagd server
type Service struct {
	cfg            Configuration
	baseRetryDelay time.Duration
	cache          *cache.Service
	events         chan of.Event
	logger         logr.Logger
	maxRetries     int

	client     schemaConnectV1.ServiceClient
	cancelHook context.CancelFunc
}

func NewService(cfg Configuration, cache *cache.Service, logger logr.Logger, retries int) *Service {
	return &Service{
		cfg:        cfg,
		cache:      cache,
		logger:     logger,
		maxRetries: retries,

		events:         make(chan of.Event, 1),
		baseRetryDelay: time.Second,
	}
}

const ConnectionError = "connection not made"

type resolutionRequestConstraints interface {
	schemaV1.ResolveBooleanRequest | schemaV1.ResolveStringRequest | schemaV1.ResolveIntRequest |
		schemaV1.ResolveFloatRequest | schemaV1.ResolveObjectRequest
}

type resolutionResponseConstraints interface {
	schemaV1.ResolveBooleanResponse | schemaV1.ResolveStringResponse | schemaV1.ResolveIntResponse |
		schemaV1.ResolveFloatResponse | schemaV1.ResolveObjectResponse
}

func (s *Service) Init() error {
	var err error
	s.client, err = newClient(s.cfg)
	if err != nil {
		return err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	go func() {
		s.startEventStream(ctx)
	}()

	s.cancelHook = cancelFunc

	return nil
}

func (s *Service) Shutdown() {
	s.cancelHook()
}

// ResolveBoolean handles the flag evaluation response from the flagd ResolveBoolean rpc
func (s *Service) ResolveBoolean(ctx context.Context, key string, defaultValue bool,
	evalCtx map[string]interface{}) of.BoolResolutionDetail {

	if s.cache.IsEnabled() {
		fromCache, ok := s.cache.GetCache().Get(key)
		if ok {
			fromCacheResDetail, ok := fromCache.(openfeature.BoolResolutionDetail)
			if ok {
				fromCacheResDetail.Reason = ReasonCached
				return fromCacheResDetail
			}
		}
	}

	var e of.ResolutionError
	resp, err := resolve[schemaV1.ResolveBooleanRequest, schemaV1.ResolveBooleanResponse](
		ctx, s.logger, s.client.ResolveBoolean, key, evalCtx,
	)
	if err != nil {
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
			},
		}
	}

	detail := of.BoolResolutionDetail{
		Value: resp.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: e,
			Reason:          of.Reason(resp.Reason),
			Variant:         resp.Variant,
			FlagMetadata:    resp.Metadata.AsMap(),
		},
	}

	if s.cache.IsEnabled() && detail.Reason == flagdModels.StaticReason {
		s.cache.GetCache().Add(key, detail)
	}

	return detail
}

// ResolveString handles the flag evaluation response from the  flagd interface ResolveString rpc
func (s *Service) ResolveString(ctx context.Context, key string, defaultValue string,
	evalCtx map[string]interface{}) of.StringResolutionDetail {

	if s.cache.IsEnabled() {
		fromCache, ok := s.cache.GetCache().Get(key)
		if ok {
			fromCacheResDetail, ok := fromCache.(openfeature.StringResolutionDetail)
			if ok {
				fromCacheResDetail.Reason = ReasonCached
				return fromCacheResDetail
			}
		}
	}

	var e of.ResolutionError
	resp, err := resolve[schemaV1.ResolveStringRequest, schemaV1.ResolveStringResponse](
		ctx, s.logger, s.client.ResolveString, key, evalCtx,
	)
	if err != nil {
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
			},
		}
	}

	detail := of.StringResolutionDetail{
		Value: resp.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: e,
			Reason:          of.Reason(resp.Reason),
			Variant:         resp.Variant,
			FlagMetadata:    resp.Metadata.AsMap(),
		},
	}

	if s.cache.IsEnabled() && detail.Reason == flagdModels.StaticReason {
		s.cache.GetCache().Add(key, detail)
	}

	return detail
}

// ResolveFloat handles the flag evaluation response from the  flagd interface ResolveFloat rpc
func (s *Service) ResolveFloat(ctx context.Context, key string, defaultValue float64,
	evalCtx map[string]interface{}) of.FloatResolutionDetail {

	if s.cache.IsEnabled() {
		fromCache, ok := s.cache.GetCache().Get(key)
		if ok {
			fromCacheResDetail, ok := fromCache.(openfeature.FloatResolutionDetail)
			if ok {
				fromCacheResDetail.Reason = ReasonCached
				return fromCacheResDetail
			}
		}
	}

	var e of.ResolutionError
	resp, err := resolve[schemaV1.ResolveFloatRequest, schemaV1.ResolveFloatResponse](
		ctx, s.logger, s.client.ResolveFloat, key, evalCtx,
	)
	if err != nil {
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
			},
		}
	}

	detail := of.FloatResolutionDetail{
		Value: resp.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: e,
			Reason:          of.Reason(resp.Reason),
			Variant:         resp.Variant,
			FlagMetadata:    resp.Metadata.AsMap(),
		},
	}

	if s.cache.IsEnabled() && detail.Reason == flagdModels.StaticReason {
		s.cache.GetCache().Add(key, detail)
	}

	return detail
}

// ResolveInt handles the flag evaluation response from the  flagd interface ResolveNumber rpc
func (s *Service) ResolveInt(ctx context.Context, key string, defaultValue int64,
	evalCtx map[string]interface{}) of.IntResolutionDetail {

	if s.cache.IsEnabled() {
		fromCache, ok := s.cache.GetCache().Get(key)
		if ok {
			fromCacheResDetail, ok := fromCache.(openfeature.IntResolutionDetail)
			if ok {
				fromCacheResDetail.Reason = ReasonCached
				return fromCacheResDetail
			}
		}
	}

	var e of.ResolutionError
	resp, err := resolve[schemaV1.ResolveIntRequest, schemaV1.ResolveIntResponse](
		ctx, s.logger, s.client.ResolveInt, key, evalCtx,
	)
	if err != nil {
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
			},
		}
	}

	detail := of.IntResolutionDetail{
		Value: resp.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: e,
			Reason:          of.Reason(resp.Reason),
			Variant:         resp.Variant,
			FlagMetadata:    resp.Metadata.AsMap(),
		},
	}

	if s.cache.IsEnabled() && detail.Reason == flagdModels.StaticReason {
		s.cache.GetCache().Add(key, detail)
	}

	return detail
}

// ResolveObject handles the flag evaluation response from the  flagd interface ResolveObject rpc
func (s *Service) ResolveObject(ctx context.Context, key string, defaultValue interface{},
	evalCtx map[string]interface{}) of.InterfaceResolutionDetail {
	if s.cache.IsEnabled() {
		fromCache, ok := s.cache.GetCache().Get(key)
		if ok {
			fromCacheResDetail, ok := fromCache.(openfeature.InterfaceResolutionDetail)
			if ok {
				fromCacheResDetail.Reason = ReasonCached
				return fromCacheResDetail
			}
		}
	}

	var e of.ResolutionError
	resp, err := resolve[schemaV1.ResolveObjectRequest, schemaV1.ResolveObjectResponse](
		ctx, s.logger, s.client.ResolveObject, key, evalCtx,
	)
	if err != nil {
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
			},
		}
	}

	detail := of.InterfaceResolutionDetail{
		Value: resp.Value.AsMap(),
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: e,
			Reason:          of.Reason(resp.Reason),
			Variant:         resp.Variant,
			FlagMetadata:    resp.Metadata.AsMap(),
		},
	}

	if s.cache.IsEnabled() && detail.Reason == flagdModels.StaticReason {
		s.cache.GetCache().Add(key, detail)
	}

	return detail
}

func resolve[req resolutionRequestConstraints, resp resolutionResponseConstraints](
	ctx context.Context, logger logr.Logger,
	resolver func(context.Context, *connect.Request[req]) (*connect.Response[resp], error),
	flagKey string, evalCtx map[string]interface{},
) (*resp, error) {
	evalCtxF, err := structpb.NewStruct(evalCtx)
	if err != nil {
		logger.Error(err, "struct from evaluation context")
		return nil, openfeature.NewParseErrorResolutionError(err.Error())
	}

	res, err := resolver(ctx, connect.NewRequest(&req{
		FlagKey: flagKey,
		Context: evalCtxF,
	}))
	if err != nil {
		return nil, handleError(err)
	}

	return res.Msg, nil
}

func handleError(err error) openfeature.ResolutionError {
	connectErr := &connect.Error{}
	errors.As(err, &connectErr)
	switch connectErr.Code() {
	case connect.CodeUnavailable:
		return openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	case connect.CodeNotFound:
		return openfeature.NewFlagNotFoundResolutionError(err.Error())
	case connect.CodeInvalidArgument:
		return openfeature.NewTypeMismatchResolutionError(err.Error())
	case connect.CodeDataLoss:
		return openfeature.NewParseErrorResolutionError(err.Error())
	}
	return openfeature.NewGeneralResolutionError(err.Error())
}

func (s *Service) EventChannel() <-chan of.Event {
	return s.events
}

// startEventStream - starts listening to flagd event stream with retries.
// If retrying is exhausted, an event with openfeature.ProviderError is emitted.
func (s *Service) startEventStream(ctx context.Context) {
	var delay = s.baseRetryDelay
	var attempts = 1

	// wraps connection with retry attempts
	for attempts <= s.maxRetries {
		s.logger.V(logger.Debug).Info("connecting to event stream")
		err := s.streamClient(ctx)
		if err != nil {
			// error in stream handler, purge cache if available and retry
			s.logger.V(logger.Warn).Info("connection to event stream failed, attempting again")
			if s.cache.IsEnabled() {
				s.cache.GetCache().Purge()
			}
			delay = 2 * delay
			attempts++
		} else {
			// no errors means planned connection termination
			attempts = 0
			delay = s.baseRetryDelay
		}

		time.Sleep(delay)
	}

	// retry attempts exhausted. Disable cache and emit error event
	s.cache.Disable()
	s.events <- of.Event{
		ProviderName: "flagd",
		EventType:    of.ProviderError,
		ProviderEventDetails: of.ProviderEventDetails{
			Message: "grpc connection establishment failed",
		},
	}
}

// streamClient opens the event stream and distribute streams to appropriate handlers
func (s *Service) streamClient(ctx context.Context) error {
	stream, err := s.client.EventStream(ctx, connect.NewRequest(&schemaV1.EventStreamRequest{}))
	if err != nil {
		return err
	}

	s.logger.V(logger.Info).Info("connected to event stream")

	for stream.Receive() {
		switch stream.Msg().Type {
		case string(flagdService.ConfigurationChange):
			s.handleConfigurationChangeEvent(stream.Msg())
		case string(flagdService.ProviderReady):
			s.handleReadyEvent()
		case string(flagdService.Shutdown):
			// this is considered as a non-error
			return nil
		case string(flagdService.KeepAlive):
		default:
			// do nothing
		}
	}

	if err := stream.Err(); err != nil {
		return err
	}

	return nil
}

func (s *Service) handleConfigurationChangeEvent(event *schemaV1.EventStreamResponse) {
	if !s.cache.IsEnabled() {
		return
	}

	if event.Data == nil {
		// purge cache and return
		s.cache.GetCache().Purge()
		return
	}

	flagsVal, ok := event.Data.AsMap()["flags"]
	if !ok {
		// purge cache and return
		s.cache.GetCache().Purge()
		return
	}

	flags, ok := flagsVal.(map[string]interface{})
	if !ok {
		// purge cache and return
		s.cache.GetCache().Purge()
		return
	}

	var keys []string

	for flagKey := range flags {
		s.cache.GetCache().Remove(flagKey)
		keys = append(keys, flagKey)
	}

	s.events <- of.Event{
		ProviderName: "flagd",
		EventType:    of.ProviderConfigChange,
		ProviderEventDetails: of.ProviderEventDetails{
			Message:     "flags changed",
			FlagChanges: keys,
		},
	}
}

func (s *Service) handleReadyEvent() {
	s.events <- of.Event{
		ProviderName: "flagd",
		EventType:    of.ProviderReady,
	}
}

// newClient is a helper to derive schemaConnectV1.ServiceClient
func newClient(cfg Configuration) (schemaConnectV1.ServiceClient, error) {
	var dialContext func(ctx context.Context, network string, addr string) (net.Conn, error)
	var tlsConfig *tls.Config
	url := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	// socket
	if cfg.SocketPath != "" {
		dialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", cfg.SocketPath)
		}
	}
	// tls
	if cfg.TLSEnabled {
		url = fmt.Sprintf("https://%s:%d", cfg.Host, cfg.Port)
		tlsConfig = &tls.Config{}
		if cfg.CertificatePath != "" {
			caCert, err := os.ReadFile(cfg.CertificatePath)
			if err != nil {
				return nil, err
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, errors.New("error appending provider certificate file. please check and try again")
			}
			tlsConfig.RootCAs = caCertPool
		}
	}

	// build options
	var options []connect.ClientOption

	if cfg.OtelInterceptor {
		options = append(options, connect.WithInterceptors(
			otelconnect.NewInterceptor(),
		))
	}

	return schemaConnectV1.NewServiceClient(
		&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
				DialContext:     dialContext,
			},
		},
		url,
		options...,
	), nil
}
