package service

import (
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/go-logr/logr"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"golang.org/x/net/context"
	"sync"
	"time"

	schemaV1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type ServiceConfiguration struct {
	Port            uint16
	Host            string
	CertificatePath string
	SocketPath      string
	TLSEnabled      bool
}

// Service handles the client side  interface for the flagd server
type Service struct {
	rwMx           *sync.RWMutex
	streamAlive    bool
	Client         iClient
	baseRetryDelay time.Duration
	logger         logr.Logger
}

func NewService(client iClient, logger logr.Logger, baseStreamRetryDelay *time.Duration) *Service {
	if baseStreamRetryDelay == nil {
		scnd := time.Second
		baseStreamRetryDelay = &scnd
	}
	return &Service{
		rwMx:           &sync.RWMutex{},
		streamAlive:    false,
		Client:         client,
		baseRetryDelay: *baseStreamRetryDelay,
		logger:         logger,
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

// ResolveBoolean handles the flag evaluation response from the flagd ResolveBoolean rpc
func (s *Service) ResolveBoolean(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveBooleanResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveBooleanResponse{
			Reason: string(openfeature.ErrorReason),
		}, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}

	resp, err := resolve[schemaV1.ResolveBooleanRequest, schemaV1.ResolveBooleanResponse](
		ctx, s.logger, client.ResolveBoolean, flagKey, evalCtx,
	)
	if err != nil {
		return &schemaV1.ResolveBooleanResponse{
			Reason: string(openfeature.ErrorReason),
		}, err
	}

	return resp, nil
}

// ResolveString handles the flag evaluation response from the  flagd interface ResolveString rpc
func (s *Service) ResolveString(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveStringResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveStringResponse{
			Reason: string(openfeature.ErrorReason),
		}, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}

	resp, err := resolve[schemaV1.ResolveStringRequest, schemaV1.ResolveStringResponse](
		ctx, s.logger, client.ResolveString, flagKey, evalCtx,
	)
	if err != nil {
		return &schemaV1.ResolveStringResponse{
			Reason: string(openfeature.ErrorReason),
		}, err
	}

	return resp, nil
}

// ResolveFloat handles the flag evaluation response from the  flagd interface ResolveFloat rpc
func (s *Service) ResolveFloat(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveFloatResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveFloatResponse{
			Reason: string(openfeature.ErrorReason),
		}, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}

	resp, err := resolve[schemaV1.ResolveFloatRequest, schemaV1.ResolveFloatResponse](
		ctx, s.logger, client.ResolveFloat, flagKey, evalCtx,
	)
	if err != nil {
		return &schemaV1.ResolveFloatResponse{
			Reason: string(openfeature.ErrorReason),
		}, err
	}

	return resp, nil
}

// ResolveInt handles the flag evaluation response from the  flagd interface ResolveNumber rpc
func (s *Service) ResolveInt(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveIntResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveIntResponse{
			Reason: string(openfeature.ErrorReason),
		}, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}

	resp, err := resolve[schemaV1.ResolveIntRequest, schemaV1.ResolveIntResponse](
		ctx, s.logger, client.ResolveInt, flagKey, evalCtx,
	)
	if err != nil {
		return &schemaV1.ResolveIntResponse{
			Reason: string(openfeature.ErrorReason),
		}, err
	}

	return resp, nil
}

// ResolveObject handles the flag evaluation response from the  flagd interface ResolveObject rpc
func (s *Service) ResolveObject(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveObjectResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveObjectResponse{
			Reason: string(openfeature.ErrorReason),
		}, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}

	resp, err := resolve[schemaV1.ResolveObjectRequest, schemaV1.ResolveObjectResponse](
		ctx, s.logger, client.ResolveObject, flagKey, evalCtx,
	)
	if err != nil {
		return &schemaV1.ResolveObjectResponse{
			Reason: string(openfeature.ErrorReason),
		}, err
	}

	return resp, nil
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

// EventStream emits received events on the given channel
func (s *Service) EventStream(
	ctx context.Context, eventChan chan<- *schemaV1.EventStreamResponse, maxAttempts int, errChan chan<- error,
) {
	delay := s.baseRetryDelay
	var err error
	for i := 1; i <= maxAttempts; i++ {
		s.logger.V(logger.Debug).Info("attempt at connection to event stream", "attempt", i)
		i, err = s.eventStream(ctx, eventChan, i)
		if i == 1 {
			delay = s.baseRetryDelay // reset delay if the connection was successful before failing
		}
		if err != nil && i <= maxAttempts {
			delay = 2 * delay
			s.logger.V(logger.Warn).Info(fmt.Sprintf("connection to event stream failed, sleeping %v", delay))
			time.Sleep(delay)
		}
	}

	if err != nil {
		errChan <- err
	}
}

func (s *Service) eventStream(
	ctx context.Context, eventChan chan<- *schemaV1.EventStreamResponse, attempt int) (int, error) {
	client := s.Client.Instance()
	if client == nil {
		return attempt + 1, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}

	stream, err := client.EventStream(ctx, connect.NewRequest(&schemaV1.EventStreamRequest{}))
	if err != nil {
		return attempt + 1, err
	}

	s.logger.V(logger.Info).Info("connected to event stream")
	s.rwMx.Lock()
	s.streamAlive = true
	s.rwMx.Unlock()
	attempt = 0 // reset attempts on successful connection

	for stream.Receive() {
		eventChan <- stream.Msg()
	}
	s.rwMx.Lock()
	s.streamAlive = false
	s.rwMx.Unlock()

	if err := stream.Err(); err != nil {
		return attempt + 1, err
	}
	close(eventChan)

	return attempt + 1, nil
}

func (s *Service) IsEventStreamAlive() bool {
	s.rwMx.RLock()
	defer s.rwMx.RUnlock()

	return s.streamAlive
}
