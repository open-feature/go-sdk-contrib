package service

import (
	"context"
	of "github.com/open-feature/go-sdk/pkg/openfeature"

	schemaV1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
)

type IService interface {
	ResolveBoolean(ctx context.Context, key string, defaultValue bool,
		evalCtx map[string]interface{}) of.BoolResolutionDetail
	ResolveString(ctx context.Context, key string, defaultValue string,
		evalCtx map[string]interface{}) of.StringResolutionDetail
	ResolveFloat(ctx context.Context, key string, defaultValue float64,
		evalCtx map[string]interface{}) of.FloatResolutionDetail
	ResolveInt(ctx context.Context, key string, defaultValue int64,
		evalCtx map[string]interface{}) of.IntResolutionDetail
	ResolveObject(ctx context.Context, key string, defaultValue interface{},
		evalCtx map[string]interface{}) of.InterfaceResolutionDetail
	EventStream(ctx context.Context, eventChan chan<- *schemaV1.EventStreamResponse, maxAttempts int,
		errChan chan<- error)
}
