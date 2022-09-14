package service

import (
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
)

type IService interface {
	ResolveBoolean(string, map[string]interface{}) (*schemaV1.ResolveBooleanResponse, error)
	ResolveString(string, map[string]interface{}) (*schemaV1.ResolveStringResponse, error)
	ResolveFloat(string, map[string]interface{}) (*schemaV1.ResolveFloatResponse, error)
	ResolveInt(string, map[string]interface{}) (*schemaV1.ResolveIntResponse, error)
	ResolveObject(string, map[string]interface{}) (*schemaV1.ResolveObjectResponse, error)
}
