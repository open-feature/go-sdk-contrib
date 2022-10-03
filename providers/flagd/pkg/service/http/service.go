package http_service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/model"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"io"
	"strconv"

	log "github.com/sirupsen/logrus"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
)

type HTTPServiceConfiguration struct {
	Port     uint16
	Host     string
	Protocol string
}

// HTTPService handles the client side http interface for the flagd server
type HTTPService struct {
	HTTPServiceConfiguration *HTTPServiceConfiguration
	Client                   iHTTPClient
}

// IntDecodeIntermediate is a required intermediate for decoding the int flag values.
// grpc gateway uses the proto3 json spec to encode its payload, this means that int64 values are encoded into a string
// https://developers.google.com/protocol-buffers/docs/proto3#json
type IntDecodeIntermediate struct {
	Value   string
	Variant string
	Reason  string
}

// ResolveBoolean handles the flag evaluation response from the flagd flags/{flagKey}/resolve/boolean endpoint
func (s *HTTPService) ResolveBoolean(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveBooleanResponse, error) {
	url := fmt.Sprintf("%s://%s:%d/flags/%s/resolve/boolean", s.HTTPServiceConfiguration.Protocol, s.HTTPServiceConfiguration.Host, s.HTTPServiceConfiguration.Port, flagKey)
	resMess := schemaV1.ResolveBooleanResponse{}
	err := s.FetchFlag(ctx, url, evalCtx, &resMess)
	if err != nil {
		return &schemaV1.ResolveBooleanResponse{
			Reason: string(of.ErrorReason),
		}, err
	}
	return &resMess, nil
}

// ResolveString handles the flag evaluation response from the flags/{flagKey}/resolve/string endpoint
func (s *HTTPService) ResolveString(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveStringResponse, error) {
	url := fmt.Sprintf("%s://%s:%d/flags/%s/resolve/string", s.HTTPServiceConfiguration.Protocol, s.HTTPServiceConfiguration.Host, s.HTTPServiceConfiguration.Port, flagKey)
	resMess := schemaV1.ResolveStringResponse{}
	err := s.FetchFlag(ctx, url, evalCtx, &resMess)
	if err != nil {
		return &schemaV1.ResolveStringResponse{
			Reason: string(of.ErrorReason),
		}, err
	}
	return &resMess, nil
}

// ResolveFloat handles the flag evaluation response from the flags/{flagKey}/resolve/float endpoint
func (s *HTTPService) ResolveFloat(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveFloatResponse, error) {
	url := fmt.Sprintf("%s://%s:%d/flags/%s/resolve/float", s.HTTPServiceConfiguration.Protocol, s.HTTPServiceConfiguration.Host, s.HTTPServiceConfiguration.Port, flagKey)
	resMess := schemaV1.ResolveFloatResponse{}
	err := s.FetchFlag(ctx, url, evalCtx, &resMess)
	if err != nil {
		return &schemaV1.ResolveFloatResponse{
			Reason: string(of.ErrorReason),
		}, err
	}
	return &resMess, nil
}

// ResolveInt handles the flag evaluation response from the flags/{flagKey}/resolve/int endpoint
func (s *HTTPService) ResolveInt(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveIntResponse, error) {
	url := fmt.Sprintf("%s://%s:%d/flags/%s/resolve/int", s.HTTPServiceConfiguration.Protocol, s.HTTPServiceConfiguration.Host, s.HTTPServiceConfiguration.Port, flagKey)
	intermediate := IntDecodeIntermediate{}
	err := s.FetchFlag(ctx, url, evalCtx, &intermediate)
	if err != nil {
		return &schemaV1.ResolveIntResponse{
			Reason: string(of.ErrorReason),
		}, err
	}
	val, err := strconv.ParseInt(intermediate.Value, 10, 64)
	if err != nil {
		return &schemaV1.ResolveIntResponse{
			Reason: string(of.ErrorReason),
		}, of.NewParseErrorResolutionError(err.Error())
	}
	return &schemaV1.ResolveIntResponse{
		Reason:  intermediate.Reason,
		Value:   val,
		Variant: intermediate.Variant,
	}, nil
}

// ResolveObject handles the flag evaluation response from the flags/{flagKey}/resolve/object endpoint
func (s *HTTPService) ResolveObject(ctx context.Context, flagKey string, evalCtx map[string]interface{}) (*schemaV1.ResolveObjectResponse, error) {
	url := fmt.Sprintf("%s://%s:%d/flags/%s/resolve/object", s.HTTPServiceConfiguration.Protocol, s.HTTPServiceConfiguration.Host, s.HTTPServiceConfiguration.Port, flagKey)
	resMess := schemaV1.ResolveObjectResponse{}
	err := s.FetchFlag(ctx, url, evalCtx, &resMess)
	if err != nil {
		return &schemaV1.ResolveObjectResponse{
			Reason: string(of.ErrorReason),
		}, err
	}
	return &resMess, nil
}

// FetchFlag handles the calling and parsing of the flag evaluation endpoints.
// Argument p should be a pointer to a valid Resolve{type}Response struct for unmarshalling the response, e.g ResolveObjectResponse{}.
func (s *HTTPService) FetchFlag(ctx context.Context, url string, evalCtx map[string]interface{}, p interface{}) error {
	body, err := json.Marshal(evalCtx)
	if err != nil {
		log.Error(err)
		return of.NewParseErrorResolutionError(err.Error())
	}
	res, err := s.Client.Request(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Error(err)
		return of.NewGeneralResolutionError(err.Error())
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error(err)
		return of.NewGeneralResolutionError(err.Error())
	}
	err = json.Unmarshal(b, p)
	if err != nil {
		log.Error(err)
		if res.StatusCode == 200 {
			return of.NewParseErrorResolutionError(err.Error())
		}
	}
	if res.StatusCode == 200 {
		return nil
	}
	if res.StatusCode == 500 {
		return of.NewGeneralResolutionError(fmt.Sprintf("status code 500"))
	}

	errRes := schemaV1.ErrorResponse{}
	err = json.Unmarshal(b, &errRes)
	if err != nil {
		log.Error(err)
		return of.NewParseErrorResolutionError(err.Error())
	}

	if errRes.ErrorCode != "" {
		return model.FlagdErrorCodeToResolutionError(errRes.ErrorCode, "")
	}

	return of.NewGeneralResolutionError("")
}
