package service

import (
	"context"

	schemaV1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
)

type IService interface {
	ResolveBoolean(context.Context, string, map[string]interface{}) (*schemaV1.ResolveBooleanResponse, error)
	ResolveString(context.Context, string, map[string]interface{}) (*schemaV1.ResolveStringResponse, error)
	ResolveFloat(context.Context, string, map[string]interface{}) (*schemaV1.ResolveFloatResponse, error)
	ResolveInt(context.Context, string, map[string]interface{}) (*schemaV1.ResolveIntResponse, error)
	ResolveObject(context.Context, string, map[string]interface{}) (*schemaV1.ResolveObjectResponse, error)
	EventStream(ctx context.Context, eventChan chan<- *schemaV1.EventStreamResponse, errorChan chan<- error)
	IsEventStreamAlive() bool
}
