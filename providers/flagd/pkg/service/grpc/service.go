package grpc_service

import (
	ctx "context"
	"errors"

	models "github.com/open-feature/flagd/pkg/model"
	models2 "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/model"
	"github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
	log "github.com/sirupsen/logrus"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type GRPCServiceConfiguration struct {
	Port int32
	Host string
}

type GRPCService struct {
	Client IGRPCClient
}

func (s *GRPCService) ResolveBoolean(flagKey string, context of.EvaluationContext, options ...service.IServiceOption) (*schemaV1.ResolveBooleanResponse, error) {
	client := s.Client.GetInstance()
	if client == nil {
		return &schemaV1.ResolveBooleanResponse{
			Reason: models.ErrorReason,
		}, errors.New(models2.ConnectionErrorCode)
	}
	ctxF, err := FormatAsStructpb(context)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveBooleanResponse{
			Reason: models.ErrorReason,
		}, errors.New(models.ParseErrorCode)
	}
	res, err := client.ResolveBoolean(ctx.TODO(), &schemaV1.ResolveBooleanRequest{
		FlagKey: flagKey,
		Context: ctxF,
	})
	if err != nil {
		res, ok := ParseError(err)
		if !ok {
			log.Error(err)
			return &schemaV1.ResolveBooleanResponse{
				Reason: models.ErrorReason,
			}, errors.New(models.GeneralErrorCode)
		}
		return &schemaV1.ResolveBooleanResponse{
			Reason: res.Reason,
		}, errors.New(res.ErrorCode)
	}
	return res, nil
}

func (s *GRPCService) ResolveString(flagKey string, context of.EvaluationContext, options ...service.IServiceOption) (*schemaV1.ResolveStringResponse, error) {
	client := s.Client.GetInstance()
	if client == nil {
		return &schemaV1.ResolveStringResponse{
			Reason: models.ErrorReason,
		}, errors.New(models2.ConnectionErrorCode)
	}
	contextF, err := FormatAsStructpb(context)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveStringResponse{
			Reason: models.ErrorReason,
		}, errors.New(models.ParseErrorCode)
	}
	res, err := client.ResolveString(ctx.TODO(), &schemaV1.ResolveStringRequest{
		FlagKey: flagKey,
		Context: contextF,
	})
	if err != nil {
		res, ok := ParseError(err)
		if !ok {
			log.Error(err)
			return &schemaV1.ResolveStringResponse{
				Reason: models.ErrorReason,
			}, errors.New(models.GeneralErrorCode)
		}
		return &schemaV1.ResolveStringResponse{
			Reason: res.Reason,
		}, errors.New(res.ErrorCode)
	}
	return res, nil
}

func (s *GRPCService) ResolveNumber(flagKey string, context of.EvaluationContext, options ...service.IServiceOption) (*schemaV1.ResolveNumberResponse, error) {
	client := s.Client.GetInstance()
	if client == nil {
		return &schemaV1.ResolveNumberResponse{
			Reason: models.ErrorReason,
		}, errors.New(models2.ConnectionErrorCode)
	}
	contextF, err := FormatAsStructpb(context)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveNumberResponse{
			Reason: models.ErrorReason,
		}, errors.New(models.ParseErrorCode)
	}
	res, err := client.ResolveNumber(ctx.TODO(), &schemaV1.ResolveNumberRequest{
		FlagKey: flagKey,
		Context: contextF,
	})
	if err != nil {
		res, ok := ParseError(err)
		if !ok {
			log.Error(err)
			return &schemaV1.ResolveNumberResponse{
				Reason: models.ErrorReason,
			}, errors.New(models.GeneralErrorCode)
		}
		return &schemaV1.ResolveNumberResponse{
			Reason: res.Reason,
		}, errors.New(res.ErrorCode)
	}
	return res, nil
}

func (s *GRPCService) ResolveObject(flagKey string, context of.EvaluationContext, options ...service.IServiceOption) (*schemaV1.ResolveObjectResponse, error) {
	client := s.Client.GetInstance()
	if client == nil {
		return &schemaV1.ResolveObjectResponse{
			Reason: models.ErrorReason,
		}, errors.New(models2.ConnectionErrorCode)
	}
	contextF, err := FormatAsStructpb(context)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveObjectResponse{
			Reason: models.ErrorReason,
		}, errors.New(models.ParseErrorCode)
	}
	res, err := client.ResolveObject(ctx.TODO(), &schemaV1.ResolveObjectRequest{
		FlagKey: flagKey,
		Context: contextF,
	})
	if err != nil {
		res, ok := ParseError(err)
		if !ok {
			log.Error(err)
			return &schemaV1.ResolveObjectResponse{
				Reason: models.ErrorReason,
			}, errors.New(models.GeneralErrorCode)
		}
		return &schemaV1.ResolveObjectResponse{
			Reason: res.Reason,
		}, errors.New(res.ErrorCode)
	}
	return res, nil
}

func ParseError(err error) (*schemaV1.ErrorResponse, bool) {
	st := status.Convert(err)
	details := st.Details()
	if len(details) != 1 {
		log.Errorf("malformed error received by error handler, details received: %d - %v", len(details), details)
		return nil, false
	}
	res, ok := details[0].(*schemaV1.ErrorResponse)
	return res, ok
}

func FormatAsStructpb(evCtx of.EvaluationContext) (*structpb.Struct, error) {
	evCtxM, ok := evCtx.(map[string]interface{})
	if !ok {
		return nil, errors.New("evaluation context is not map[string]interface{}")
	}
	return structpb.NewStruct(evCtxM)
}
