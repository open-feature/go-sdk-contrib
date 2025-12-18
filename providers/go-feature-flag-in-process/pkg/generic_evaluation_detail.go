package gofeatureflaginprocess

import of "go.openfeature.dev/openfeature/v2"

type GenericResolutionDetail[T JsonType] struct {
	Value T
	of.ProviderResolutionDetail
}
