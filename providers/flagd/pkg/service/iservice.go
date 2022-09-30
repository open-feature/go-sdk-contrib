package service

import (
	"context"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
)

type IService interface {
	ResolveBoolean(context.Context, string, map[string]interface{}) (*schemaV1.ResolveBooleanResponse, error)
	ResolveString(context.Context, string, map[string]interface{}) (*schemaV1.ResolveStringResponse, error)
	ResolveFloat(context.Context, string, map[string]interface{}) (*schemaV1.ResolveFloatResponse, error)
	ResolveInt(context.Context, string, map[string]interface{}) (*schemaV1.ResolveIntResponse, error)
	ResolveObject(context.Context, string, map[string]interface{}) (*schemaV1.ResolveObjectResponse, error)
}
