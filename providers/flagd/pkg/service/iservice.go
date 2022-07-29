package service

import (
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
)

type IServiceOption func(IService)

type IService interface {
	ResolveBoolean(string, of.EvaluationContext, ...IServiceOption) (*schemaV1.ResolveBooleanResponse, error)
	ResolveString(string, of.EvaluationContext, ...IServiceOption) (*schemaV1.ResolveStringResponse, error)
	ResolveNumber(string, of.EvaluationContext, ...IServiceOption) (*schemaV1.ResolveNumberResponse, error)
	ResolveObject(string, of.EvaluationContext, ...IServiceOption) (*schemaV1.ResolveObjectResponse, error)
}
