package service

import (
	"github.com/bufbuild/connect-go"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/model"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"golang.org/x/net/context"

	flagdModels "github.com/open-feature/flagd/pkg/model"
	log "github.com/sirupsen/logrus"
	schemaV1 "go.buf.build/open-feature/flagd-connect/open-feature/flagd/schema/v1"
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
	Client iClient
	Config *ServiceConfiguration
}

// ResolveBoolean handles the flag evaluation response from the flagd ResolveBoolean rpc
func (s *Service) ResolveBoolean(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveBooleanResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveBooleanResponse{
			Reason: string(of.ErrorReason),
		}, of.NewProviderNotReadyResolutionError("connection not made")
	}
	evalCtxF, err := structpb.NewStruct(evalCtx)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveBooleanResponse{
			Reason: string(of.ErrorReason),
		}, of.NewParseErrorResolutionError(err.Error())
	}
	res, err := client.ResolveBoolean(ctx, connect.NewRequest(&schemaV1.ResolveBooleanRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	}))
	if err != nil {
		res, ok := parseError(err)
		if !ok {
			log.Error(err)
			return &schemaV1.ResolveBooleanResponse{
				Reason: string(of.ErrorReason),
			}, of.NewGeneralResolutionError(err.Error())
		}
		return &schemaV1.ResolveBooleanResponse{
			Reason: res.Reason,
		}, model.FlagdErrorCodeToResolutionError(res.ErrorCode, "")
	}
	return res.Msg, nil
}

// ResolveString handles the flag evaluation response from the  flagd interface ResolveString rpc
func (s *Service) ResolveString(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveStringResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveStringResponse{
			Reason: flagdModels.ErrorReason,
		}, of.NewProviderNotReadyResolutionError("connection not made")
	}
	evalCtxF, err := structpb.NewStruct(evalCtx)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveStringResponse{
			Reason: flagdModels.ErrorReason,
		}, of.NewParseErrorResolutionError(err.Error())
	}
	res, err := client.ResolveString(ctx, connect.NewRequest(&schemaV1.ResolveStringRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	}))
	if err != nil {
		res, ok := parseError(err)
		if !ok {
			log.Error(err)
			return &schemaV1.ResolveStringResponse{
				Reason: flagdModels.ErrorReason,
			}, of.NewGeneralResolutionError(err.Error())
		}
		return &schemaV1.ResolveStringResponse{
			Reason: res.Reason,
		}, model.FlagdErrorCodeToResolutionError(res.ErrorCode, "")
	}
	return res.Msg, nil
}

// ResolveFloat handles the flag evaluation response from the  flagd interface ResolveFloat rpc
func (s *Service) ResolveFloat(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveFloatResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveFloatResponse{
			Reason: flagdModels.ErrorReason,
		}, of.NewProviderNotReadyResolutionError("connection not made")
	}
	evalCtxF, err := structpb.NewStruct(evalCtx)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveFloatResponse{
			Reason: flagdModels.ErrorReason,
		}, of.NewParseErrorResolutionError(err.Error())
	}
	res, err := client.ResolveFloat(ctx, connect.NewRequest(&schemaV1.ResolveFloatRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	}))
	if err != nil {
		res, ok := parseError(err)
		if !ok {
			log.Error(err)
			return &schemaV1.ResolveFloatResponse{
				Reason: flagdModels.ErrorReason,
			}, of.NewGeneralResolutionError(err.Error())
		}
		return &schemaV1.ResolveFloatResponse{
			Reason: res.Reason,
		}, model.FlagdErrorCodeToResolutionError(res.ErrorCode, "")
	}
	return res.Msg, nil
}

// ResolveInt handles the flag evaluation response from the  flagd interface ResolveNumber rpc
func (s *Service) ResolveInt(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveIntResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveIntResponse{
			Reason: flagdModels.ErrorReason,
		}, of.NewProviderNotReadyResolutionError("connection not made")
	}
	evalCtxF, err := structpb.NewStruct(evalCtx)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveIntResponse{
			Reason: flagdModels.ErrorReason,
		}, of.NewParseErrorResolutionError(err.Error())
	}
	res, err := client.ResolveInt(ctx, connect.NewRequest(&schemaV1.ResolveIntRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	}))
	if err != nil {
		res, ok := parseError(err)
		if !ok {
			log.Error(err)
			return &schemaV1.ResolveIntResponse{
				Reason: flagdModels.ErrorReason,
			}, of.NewGeneralResolutionError(err.Error())
		}
		return &schemaV1.ResolveIntResponse{
			Reason: res.Reason,
		}, model.FlagdErrorCodeToResolutionError(res.ErrorCode, "")
	}
	return res.Msg, nil
}

// ResolveObject handles the flag evaluation response from the  flagd interface ResolveObject rpc
func (s *Service) ResolveObject(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveObjectResponse, error) {
	client := s.Client.Instance()
	if client == nil {
		return &schemaV1.ResolveObjectResponse{
			Reason: flagdModels.ErrorReason,
		}, of.NewProviderNotReadyResolutionError("connection not made")
	}
	evalCtxF, err := structpb.NewStruct(evalCtx)
	if err != nil {
		log.Error(err)
		return &schemaV1.ResolveObjectResponse{
			Reason: flagdModels.ErrorReason,
		}, of.NewParseErrorResolutionError(err.Error())
	}
	res, err := client.ResolveObject(ctx, connect.NewRequest(&schemaV1.ResolveObjectRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	}))
	if err != nil {
		res, ok := parseError(err)
		if !ok {
			log.Error(err)
			return &schemaV1.ResolveObjectResponse{
				Reason: flagdModels.ErrorReason,
			}, of.NewGeneralResolutionError(err.Error())
		}
		return &schemaV1.ResolveObjectResponse{
			Reason: res.Reason,
		}, model.FlagdErrorCodeToResolutionError(res.ErrorCode, "")
	}
	return res.Msg, nil
}
