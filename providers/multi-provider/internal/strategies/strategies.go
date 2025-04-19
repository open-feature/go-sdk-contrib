// Package strategy Resolution strategies are defined within this package
package strategies

import (
	"context"
	of "github.com/open-feature/go-sdk/openfeature"
	"reflect"
)

const (
	MetadataSuccessfulProviderName           = "multiprovider-successful-provider-name"
	MetadataStrategyUsed                     = "multiprovider-strategy-used"
	MetadataFallbackUsed                     = "multiprovider-fallback-used"
	StrategyFirstMatch                       = "strategy-first-match"
	StrategyFirstSuccess                     = "strategy-first-success"
	StrategyComparison                       = "strategy-comparison"
	ReasonAggregated               of.Reason = "AGGREGATED"
	ReasonAggregatedFallback       of.Reason = "AGGREGATED_FALLBACK"
	ErrAggregationNotAllowedText             = "object evaluation not allowed for non-comparable types"
)

type (
	// EvaluationStrategy Defines a strategy to use for resolving the result from multiple providers
	EvaluationStrategy = string
	// Strategy Interface for evaluating providers within the multi-provider.
	Strategy interface {
		Name() EvaluationStrategy
		BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail
		StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail
		FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail
		IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail
		ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail
	}

	// NamedProvider allows for a unique name to be assigned to a provider during a multi-provider set up.
	// The name will be used when reporting errors & results to specify the provider associated.
	NamedProvider struct {
		Name     string
		Provider of.FeatureProvider
	}

	providerError struct {
		providerName string
		err          error
	}

	resultConstraint interface {
		of.BoolResolutionDetail | of.IntResolutionDetail | of.StringResolutionDetail | of.FloatResolutionDetail | of.InterfaceResolutionDetail
	}

	resultWrapper[R resultConstraint] struct {
		name   string
		result *R
		value  any
		detail of.ProviderResolutionDetail
	}
)

var _ error = (*providerError)(nil)

func (p providerError) Error() string {
	return p.providerName + ": " + p.err.Error()
}

// buildDefaultResult Creates a default result using reflection via generics
func buildDefaultResult[R resultConstraint, DV bool | string | int64 | float64 | interface{}](strategy EvaluationStrategy, defaultValue DV, err error) resultWrapper[R] {
	result := *new(R)
	details := of.ProviderResolutionDetail{
		ResolutionError: of.NewFlagNotFoundResolutionError(err.Error()),
		Reason:          of.DefaultReason,
		FlagMetadata:    of.FlagMetadata{MetadataSuccessfulProviderName: "none", MetadataStrategyUsed: strategy},
	}
	switch reflect.TypeOf(result).Name() {
	case "BoolResolutionDetail":
		r := any(result).(of.BoolResolutionDetail)
		r.Value = any(defaultValue).(bool)
		r.ProviderResolutionDetail = details
		result = any(r).(R)
	case "StringResolutionDetail":
		r := any(result).(of.StringResolutionDetail)
		r.Value = any(defaultValue).(string)
		r.ProviderResolutionDetail = details
		result = any(r).(R)
	case "IntResolutionDetail":
		r := any(result).(of.IntResolutionDetail)
		r.Value = any(defaultValue).(int64)
		r.ProviderResolutionDetail = details
		result = any(r).(R)
	case "FloatResolutionDetail":
		r := any(result).(of.FloatResolutionDetail)
		r.Value = any(defaultValue).(float64)
		r.ProviderResolutionDetail = details
		result = any(r).(R)
	default:
		r := any(result).(of.InterfaceResolutionDetail)
		r.Value = defaultValue
		r.ProviderResolutionDetail = details
		result = any(r).(R)
	}

	return resultWrapper[R]{result: &result}
}

func setFlagMetadata(strategyUsed EvaluationStrategy, successProviderName string, metadata of.FlagMetadata) of.FlagMetadata {
	if metadata == nil {
		metadata = make(of.FlagMetadata)
	}
	metadata[MetadataSuccessfulProviderName] = successProviderName
	metadata[MetadataStrategyUsed] = strategyUsed
	return metadata
}
