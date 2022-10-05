package grpc_service

import (
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/model"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"golang.org/x/net/context"

	flagdModels "github.com/open-feature/flagd/pkg/model"
	log "github.com/sirupsen/logrus"
	schemaV1 "go.buf.build/open-feature/flagd-connect/open-feature/flagd/schema/v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type GRPCServiceConfiguration struct {
	Port            uint16
	Host            string
	CertificatePath string
	SocketPath      string
}

// GRPCService handles the client side grpc interface for the flagd server
type GRPCService struct {
	Client iGRPCClient
}

// ResolveBoolean handles the flag evaluation response from the grpc flagd ResolveBoolean rpc
func (s *GRPCService) ResolveBoolean(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveBooleanResponse, error) {
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
	res, err := client.ResolveBoolean(ctx, &schemaV1.ResolveBooleanRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	})
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
	return res, nil
}

// ResolveString handles the flag evaluation response from the grpc flagd interface ResolveString rpc
func (s *GRPCService) ResolveString(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveStringResponse, error) {
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
	res, err := client.ResolveString(ctx, &schemaV1.ResolveStringRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	})
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
	return res, nil
}

// ResolveFloat handles the flag evaluation response from the grpc flagd interface ResolveFloat rpc
func (s *GRPCService) ResolveFloat(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveFloatResponse, error) {
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
	res, err := client.ResolveFloat(ctx, &schemaV1.ResolveFloatRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	})
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
	return res, nil
}

// ResolveInt handles the flag evaluation response from the grpc flagd interface ResolveNumber rpc
func (s *GRPCService) ResolveInt(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveIntResponse, error) {
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
	res, err := client.ResolveInt(ctx, &schemaV1.ResolveIntRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	})
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
	return res, nil
}

// ResolveObject handles the flag evaluation response from the grpc flagd interface ResolveObject rpc
func (s *GRPCService) ResolveObject(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveObjectResponse, error) {
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
	res, err := client.ResolveObject(ctx, &schemaV1.ResolveObjectRequest{
		FlagKey: flagKey,
		Context: evalCtxF,
	})
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
	return res, nil
}

func parseError(err error) (*schemaV1.ErrorResponse, bool) {
	st := status.Convert(err)
	details := st.Details()
	if len(details) != 1 {
		log.Errorf("malformed error received by error handler, details received: %d - %v", len(details), details)
		return nil, false
	}
	res, ok := details[0].(*schemaV1.ErrorResponse)
	return res, ok
}
