package service

import (
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
)

type IService interface {
	ResolveBoolean(string, of.EvaluationContext) (*schemaV1.ResolveBooleanResponse, error)
	ResolveString(string, of.EvaluationContext) (*schemaV1.ResolveStringResponse, error)
	ResolveFloat(string, of.EvaluationContext) (*schemaV1.ResolveFloatResponse, error)
	ResolveInt(string, of.EvaluationContext) (*schemaV1.ResolveIntResponse, error)
	ResolveObject(string, of.EvaluationContext) (*schemaV1.ResolveObjectResponse, error)
}
