package service

import (
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/go-logr/logr"
	flagdModels "github.com/open-feature/flagd/core/pkg/model"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/constant"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"golang.org/x/net/context"
	"sync"
	"time"

	of "github.com/open-feature/go-sdk/pkg/openfeature"

	schemaV1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// Service handles the client side  interface for the flagd server
type Service struct {
	cache          *cache.Service
	rwMx           *sync.RWMutex
	streamAlive    bool
	client         Client
	baseRetryDelay time.Duration
	logger         logr.Logger
}

func NewService(client Client, cache *cache.Service, logger logr.Logger) *Service {
	return &Service{
		client: client,
		cache:  cache,
		logger: logger,

		baseRetryDelay: time.Second,
		rwMx:           &sync.RWMutex{},
		streamAlive:    false,
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

// ResolveBoolean handles the flag evaluation response from the flagd ResolveBoolean rpc
func (s *Service) ResolveBoolean(ctx context.Context, key string, defaultValue bool,
	evalCtx map[string]interface{}) of.BoolResolutionDetail {

	if s.cache.IsEnabled() {
		fromCache, ok := s.cache.GetCache().Get(key)
		if ok {
			fromCacheResDetail, ok := fromCache.(openfeature.BoolResolutionDetail)
			if ok {
				fromCacheResDetail.Reason = constant.ReasonCached
				return fromCacheResDetail
			}
		}
	}

	client := s.client.Instance()

	var e of.ResolutionError
	resp, err := resolve[schemaV1.ResolveBooleanRequest, schemaV1.ResolveBooleanResponse](
		ctx, s.logger, client.ResolveBoolean, key, evalCtx,
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
		Value: defaultValue,
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
				fromCacheResDetail.Reason = constant.ReasonCached
				return fromCacheResDetail
			}
		}
	}

	client := s.client.Instance()

	var e of.ResolutionError
	resp, err := resolve[schemaV1.ResolveStringRequest, schemaV1.ResolveStringResponse](
		ctx, s.logger, client.ResolveString, key, evalCtx,
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
		Value: defaultValue,
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
				fromCacheResDetail.Reason = constant.ReasonCached
				return fromCacheResDetail
			}
		}
	}

	client := s.client.Instance()

	var e of.ResolutionError
	resp, err := resolve[schemaV1.ResolveFloatRequest, schemaV1.ResolveFloatResponse](
		ctx, s.logger, client.ResolveFloat, key, evalCtx,
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
		Value: defaultValue,
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
				fromCacheResDetail.Reason = constant.ReasonCached
				return fromCacheResDetail
			}
		}
	}

	client := s.client.Instance()

	var e of.ResolutionError
	resp, err := resolve[schemaV1.ResolveIntRequest, schemaV1.ResolveIntResponse](
		ctx, s.logger, client.ResolveInt, key, evalCtx,
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
		Value: defaultValue,
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
				fromCacheResDetail.Reason = constant.ReasonCached
				return fromCacheResDetail
			}
		}
	}

	client := s.client.Instance()

	var e of.ResolutionError
	resp, err := resolve[schemaV1.ResolveObjectRequest, schemaV1.ResolveObjectResponse](
		ctx, s.logger, client.ResolveObject, key, evalCtx,
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
		Value: defaultValue,
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
	client := s.client.Instance()
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
