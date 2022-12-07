package service

import (
	"errors"
	"github.com/bufbuild/connect-go"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"
	"time"

	schemaV1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	flagdModels "github.com/open-feature/flagd/pkg/model"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/structpb"
)

type ServiceConfiguration struct {
	Port            uint16
	Host            string
	CertificatePath string
	SocketPath      string
}

// Service handles the client side  interface for the flagd server
type Service struct {
	rwMx           *sync.RWMutex
	streamAlive    bool
	Client         iClient
	baseRetryDelay time.Duration
}

func NewService(client iClient, baseStreamRetryDelay *time.Duration) *Service {
	if baseStreamRetryDelay == nil {
		scnd := time.Second
		baseStreamRetryDelay = &scnd
	}
	return &Service{
		rwMx:           &sync.RWMutex{},
		streamAlive:    false,
		Client:         client,
		baseRetryDelay: *baseStreamRetryDelay,
	}
}

const ConnectionError = "connection not made"

// ResolveBoolean handles the flag evaluation response from the flagd ResolveBoolean rpc
func (s *Service) ResolveBoolean(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveBooleanResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveBooleanResponse{
			Reason: string(openfeature.ErrorReason),
		}, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}
	evalCtxF, err := structpb.NewStruct(evalCtx)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveBooleanResponse{
			Reason: string(openfeature.ErrorReason),
		}, openfeature.NewParseErrorResolutionError(err.Error())
	}
	res, err := client.ResolveBoolean(ctx, connect.NewRequest(&schemaV1.ResolveBooleanRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	}))
	if err != nil {
		return &schemaV1.ResolveBooleanResponse{
			Reason: string(openfeature.ErrorReason),
		}, handleError(err)
	}
	return res.Msg, nil
}

// ResolveString handles the flag evaluation response from the  flagd interface ResolveString rpc
func (s *Service) ResolveString(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveStringResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveStringResponse{
			Reason: flagdModels.ErrorReason,
		}, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}
	evalCtxF, err := structpb.NewStruct(evalCtx)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveStringResponse{
			Reason: flagdModels.ErrorReason,
		}, openfeature.NewParseErrorResolutionError(err.Error())
	}
	res, err := client.ResolveString(ctx, connect.NewRequest(&schemaV1.ResolveStringRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	}))
	if err != nil {
		return &schemaV1.ResolveStringResponse{
			Reason: string(openfeature.ErrorReason),
		}, handleError(err)
	}
	return res.Msg, nil
}

// ResolveFloat handles the flag evaluation response from the  flagd interface ResolveFloat rpc
func (s *Service) ResolveFloat(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveFloatResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveFloatResponse{
			Reason: flagdModels.ErrorReason,
		}, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}
	evalCtxF, err := structpb.NewStruct(evalCtx)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveFloatResponse{
			Reason: flagdModels.ErrorReason,
		}, openfeature.NewParseErrorResolutionError(err.Error())
	}
	res, err := client.ResolveFloat(ctx, connect.NewRequest(&schemaV1.ResolveFloatRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	}))
	if err != nil {
		return &schemaV1.ResolveFloatResponse{
			Reason: string(openfeature.ErrorReason),
		}, handleError(err)
	}
	return res.Msg, nil
}

// ResolveInt handles the flag evaluation response from the  flagd interface ResolveNumber rpc
func (s *Service) ResolveInt(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveIntResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveIntResponse{
			Reason: flagdModels.ErrorReason,
		}, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}
	evalCtxF, err := structpb.NewStruct(evalCtx)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveIntResponse{
			Reason: flagdModels.ErrorReason,
		}, openfeature.NewParseErrorResolutionError(err.Error())
	}
	res, err := client.ResolveInt(ctx, connect.NewRequest(&schemaV1.ResolveIntRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	}))
	if err != nil {
		return &schemaV1.ResolveIntResponse{
			Reason: string(openfeature.ErrorReason),
		}, handleError(err)
	}
	return res.Msg, nil
}

// ResolveObject handles the flag evaluation response from the  flagd interface ResolveObject rpc
func (s *Service) ResolveObject(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveObjectResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveObjectResponse{
			Reason: flagdModels.ErrorReason,
		}, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}
	evalCtxF, err := structpb.NewStruct(evalCtx)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveObjectResponse{
			Reason: flagdModels.ErrorReason,
		}, openfeature.NewParseErrorResolutionError(err.Error())
	}
	res, err := client.ResolveObject(ctx, connect.NewRequest(&schemaV1.ResolveObjectRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	}))
	if err != nil {
		return &schemaV1.ResolveObjectResponse{
			Reason: string(openfeature.ErrorReason),
		}, handleError(err)
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

// EventStream emits received events on the given channel
func (s *Service) EventStream(
	ctx context.Context, eventChan chan<- *schemaV1.EventStreamResponse, maxAttempts int, errChan chan<- error,
) {
	delay := s.baseRetryDelay
	var err error
	for i := 1; i <= maxAttempts; i++ {
		log.Infof("attempt %d at connecting to event stream", i)
		i, err = s.eventStream(ctx, eventChan, i, maxAttempts)
		if i == 1 {
			delay = s.baseRetryDelay // reset delay if the connection was successful before failing
		}
		if err != nil && i <= maxAttempts {
			delay = 2 * delay
			log.Infof("connection to event stream failed, sleeping %v", delay)
			time.Sleep(delay)
		}
	}

	if err != nil {
		errChan <- err
	}
}

func (s *Service) eventStream(
	ctx context.Context, eventChan chan<- *schemaV1.EventStreamResponse, attempt, maxAttempts int,
) (int, error) {
	client := s.Client.Instance()
	if client == nil {
		return attempt + 1, openfeature.NewProviderNotReadyResolutionError(ConnectionError)
	}

	stream, err := client.EventStream(ctx, connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		return attempt + 1, err
	}
	log.Info("connected to event stream")
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
