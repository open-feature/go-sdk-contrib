package gofeatureflaginprocess

import of "github.com/open-feature/go-sdk/openfeature"

type GenericResolutionDetail[T JsonType] struct {
	Value T
	of.ProviderResolutionDetail
}
