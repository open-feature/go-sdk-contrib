package http_service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	models "github.com/open-feature/flagd/pkg/model"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
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
func (s *HTTPService) ResolveBoolean(flagKey string, context of.EvaluationContext) (*schemaV1.ResolveBooleanResponse, error) {
	url := fmt.Sprintf("%s://%s:%d/flags/%s/resolve/boolean", s.HTTPServiceConfiguration.Protocol, s.HTTPServiceConfiguration.Host, s.HTTPServiceConfiguration.Port, flagKey)
	resMess := schemaV1.ResolveBooleanResponse{}
	err := s.FetchFlag(url, context, &resMess)
	if err != nil {
		return &schemaV1.ResolveBooleanResponse{
			Reason: models.ErrorReason,
		}, err
	}
	return &resMess, nil
}

// ResolveString handles the flag evaluation response from the flags/{flagKey}/resolve/string endpoint
func (s *HTTPService) ResolveString(flagKey string, context of.EvaluationContext) (*schemaV1.ResolveStringResponse, error) {
	url := fmt.Sprintf("%s://%s:%d/flags/%s/resolve/string", s.HTTPServiceConfiguration.Protocol, s.HTTPServiceConfiguration.Host, s.HTTPServiceConfiguration.Port, flagKey)
	resMess := schemaV1.ResolveStringResponse{}
	err := s.FetchFlag(url, context, &resMess)
	if err != nil {
		return &schemaV1.ResolveStringResponse{
			Reason: models.ErrorReason,
		}, err
	}
	return &resMess, nil
}

// ResolveFloat handles the flag evaluation response from the flags/{flagKey}/resolve/float endpoint
func (s *HTTPService) ResolveFloat(flagKey string, context of.EvaluationContext) (*schemaV1.ResolveFloatResponse, error) {
	url := fmt.Sprintf("%s://%s:%d/flags/%s/resolve/float", s.HTTPServiceConfiguration.Protocol, s.HTTPServiceConfiguration.Host, s.HTTPServiceConfiguration.Port, flagKey)
	resMess := schemaV1.ResolveFloatResponse{}
	err := s.FetchFlag(url, context, &resMess)
	if err != nil {
		return &schemaV1.ResolveFloatResponse{
			Reason: models.ErrorReason,
		}, err
	}
	return &resMess, nil
}

// ResolveInt handles the flag evaluation response from the flags/{flagKey}/resolve/int endpoint
func (s *HTTPService) ResolveInt(flagKey string, context of.EvaluationContext) (*schemaV1.ResolveIntResponse, error) {
	url := fmt.Sprintf("%s://%s:%d/flags/%s/resolve/int", s.HTTPServiceConfiguration.Protocol, s.HTTPServiceConfiguration.Host, s.HTTPServiceConfiguration.Port, flagKey)
	intermediate := IntDecodeIntermediate{}
	err := s.FetchFlag(url, context, &intermediate)
	if err != nil {
		return &schemaV1.ResolveIntResponse{
			Reason: models.ErrorReason,
		}, err
	}
	val, err := strconv.ParseInt(intermediate.Value, 10, 64)
	if err != nil {
		return &schemaV1.ResolveIntResponse{
			Reason: models.ErrorReason,
		}, errors.New(models.ParseErrorCode)
	}
	return &schemaV1.ResolveIntResponse{
		Reason:  intermediate.Reason,
		Value:   val,
		Variant: intermediate.Variant,
	}, nil
}

// ResolveObject handles the flag evaluation response from the flags/{flagKey}/resolve/object endpoint
func (s *HTTPService) ResolveObject(flagKey string, context of.EvaluationContext) (*schemaV1.ResolveObjectResponse, error) {
	url := fmt.Sprintf("%s://%s:%d/flags/%s/resolve/object", s.HTTPServiceConfiguration.Protocol, s.HTTPServiceConfiguration.Host, s.HTTPServiceConfiguration.Port, flagKey)
	resMess := schemaV1.ResolveObjectResponse{}
	err := s.FetchFlag(url, context, &resMess)
	if err != nil {
		return &schemaV1.ResolveObjectResponse{
			Reason: models.ErrorReason,
		}, err
	}
	return &resMess, nil
}

// FetchFlag handles the calling and parsing of the flag evaluation endpoints.
// Argument p should be a pointer to a valid Resolve{type}Response struct for unmarshalling the response, e.g ResolveObjectResponse{}.
func (s *HTTPService) FetchFlag(url string, ctx of.EvaluationContext, p interface{}) error {
	body, err := json.Marshal(flattenContext(ctx))
	if err != nil {
		log.Error(err)
		return errors.New(models.ParseErrorCode)
	}
	res, err := s.Client.Request("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Error(err)
		return errors.New(models.GeneralErrorCode)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error(err)
		return errors.New(models.GeneralErrorCode)
	}
	err = json.Unmarshal(b, p)
	if err != nil {
		log.Error(err)
		if res.StatusCode == 200 {
			return errors.New(models.ParseErrorCode)
		}
	}
	if res.StatusCode == 200 {
		return nil
	}
	if res.StatusCode == 500 {
		return errors.New(models.GeneralErrorCode)
	}

	errRes := schemaV1.ErrorResponse{}
	err = json.Unmarshal(b, &errRes)
	if err != nil {
		log.Error(err)
		return errors.New(models.ParseErrorCode)
	}

	if errRes.ErrorCode != "" {
		return errors.New(errRes.ErrorCode)
	}

	return errors.New(models.GeneralErrorCode)
}

func flattenContext(ctx of.EvaluationContext) map[string]interface{} {
	if ctx.Attributes == nil {
		ctx.Attributes = map[string]interface{}{}
	}
	if ctx.TargetingKey != "" {
		ctx.Attributes["TargetingKey"] = ctx.TargetingKey
	}
	return ctx.Attributes
}
