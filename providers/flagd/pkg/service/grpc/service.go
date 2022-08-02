package grpc_service

import (
	ctx "context"
	"errors"

	flagdModels "github.com/open-feature/flagd/pkg/model"
	sdkModels "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/model"
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

func (s *GRPCService) ResolveBoolean(flagKey string, context of.EvaluationContext) (*schemaV1.ResolveBooleanResponse, error) {
	client := s.Client.GetInstance()
	if client == nil {
		return &schemaV1.ResolveBooleanResponse{
			Reason: flagdModels.ErrorReason,
		}, errors.New(sdkModels.ConnectionErrorCode)
	}
	ctxF, err := FormatAsStructpb(context)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveBooleanResponse{
			Reason: flagdModels.ErrorReason,
		}, errors.New(flagdModels.ParseErrorCode)
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
				Reason: flagdModels.ErrorReason,
			}, errors.New(flagdModels.GeneralErrorCode)
		}
		return &schemaV1.ResolveBooleanResponse{
			Reason: res.Reason,
		}, errors.New(res.ErrorCode)
	}
	return res, nil
}

func (s *GRPCService) ResolveString(flagKey string, context of.EvaluationContext) (*schemaV1.ResolveStringResponse, error) {
	client := s.Client.GetInstance()
	if client == nil {
		return &schemaV1.ResolveStringResponse{
			Reason: flagdModels.ErrorReason,
		}, errors.New(sdkModels.ConnectionErrorCode)
	}
	contextF, err := FormatAsStructpb(context)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveStringResponse{
			Reason: flagdModels.ErrorReason,
		}, errors.New(flagdModels.ParseErrorCode)
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
				Reason: flagdModels.ErrorReason,
			}, errors.New(flagdModels.GeneralErrorCode)
		}
		return &schemaV1.ResolveStringResponse{
			Reason: res.Reason,
		}, errors.New(res.ErrorCode)
	}
	return res, nil
}

func (s *GRPCService) ResolveNumber(flagKey string, context of.EvaluationContext) (*schemaV1.ResolveNumberResponse, error) {
	client := s.Client.GetInstance()
	if client == nil {
		return &schemaV1.ResolveNumberResponse{
			Reason: flagdModels.ErrorReason,
		}, errors.New(sdkModels.ConnectionErrorCode)
	}
	contextF, err := FormatAsStructpb(context)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveNumberResponse{
			Reason: flagdModels.ErrorReason,
		}, errors.New(flagdModels.ParseErrorCode)
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
				Reason: flagdModels.ErrorReason,
			}, errors.New(flagdModels.GeneralErrorCode)
		}
		return &schemaV1.ResolveNumberResponse{
			Reason: res.Reason,
		}, errors.New(res.ErrorCode)
	}
	return res, nil
}

func (s *GRPCService) ResolveObject(flagKey string, context of.EvaluationContext) (*schemaV1.ResolveObjectResponse, error) {
	client := s.Client.GetInstance()
	if client == nil {
		return &schemaV1.ResolveObjectResponse{
			Reason: flagdModels.ErrorReason,
		}, errors.New(sdkModels.ConnectionErrorCode)
	}
	contextF, err := FormatAsStructpb(context)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveObjectResponse{
			Reason: flagdModels.ErrorReason,
		}, errors.New(flagdModels.ParseErrorCode)
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
				Reason: flagdModels.ErrorReason,
			}, errors.New(flagdModels.GeneralErrorCode)
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
	if evCtx.TargetingKey != "" {
		evCtx.Attributes["TargettingKey"] = evCtx.TargetingKey
	}

	return structpb.NewStruct(evCtx.Attributes)
}
