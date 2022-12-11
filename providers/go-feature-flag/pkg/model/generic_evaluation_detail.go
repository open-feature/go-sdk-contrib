package model

import of "github.com/open-feature/go-sdk/pkg/openfeature"

type GenericResolutionDetail[T JsonType] struct {
	Value T
	of.ProviderResolutionDetail
}
