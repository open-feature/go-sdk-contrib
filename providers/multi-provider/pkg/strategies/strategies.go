// Package strategies Resolution strategies are defined within this package
//
//go:generate go run go.uber.org/mock/mockgen -source=strategies.go -destination=../../pkg/strategies/strategy_mock.go -package=strategies
package strategies

import (
	"context"
	of "github.com/open-feature/go-sdk/openfeature"
	"reflect"
	"regexp"
	"strings"
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

	resultConstraint interface {
		of.BoolResolutionDetail | of.IntResolutionDetail | of.StringResolutionDetail | of.FloatResolutionDetail | of.InterfaceResolutionDetail
	}

	resultWrapper[R resultConstraint] struct {
		name   string
		result *R
		value  any
		detail of.ProviderResolutionDetail
	}

	evaluator[R resultConstraint] func(ctx context.Context, p *NamedProvider) resultWrapper[R]
)

// buildDefaultResult Creates a default result using reflection via generics
func buildDefaultResult[R resultConstraint, DV bool | string | int64 | float64 | interface{}](strategy EvaluationStrategy, defaultValue DV, err error) resultWrapper[R] {
	result := *new(R)
	var rErr of.ResolutionError
	var reason of.Reason
	if err != nil {
		rErr = of.NewGeneralResolutionError(cleanErrorMessage(err.Error()))
		reason = of.ErrorReason
	} else {
		rErr = of.NewFlagNotFoundResolutionError("not found in any provider")
		reason = of.DefaultReason
	}
	details := of.ProviderResolutionDetail{
		ResolutionError: rErr,
		Reason:          reason,
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

	return resultWrapper[R]{result: &result, detail: details}
}

func setFlagMetadata(strategyUsed EvaluationStrategy, successProviderName string, metadata of.FlagMetadata) of.FlagMetadata {
	if metadata == nil {
		metadata = make(of.FlagMetadata)
	}
	metadata[MetadataSuccessfulProviderName] = successProviderName
	metadata[MetadataStrategyUsed] = strategyUsed
	return metadata
}

func cleanErrorMessage(msg string) string {
	codeRegex := strings.Join([]string{
		string(of.ProviderNotReadyCode),
		//string(of.ProviderFatalCode), // TODO: not available until go-sdk 14
		string(of.FlagNotFoundCode),
		string(of.ParseErrorCode),
		string(of.TypeMismatchCode),
		string(of.TargetingKeyMissingCode),
		string(of.GeneralCode),
	}, "|")
	re := regexp.MustCompile("(?:" + codeRegex + "): (.*)")
	matches := re.FindSubmatch([]byte(msg))
	matchCount := len(matches)
	switch matchCount {
	case 0, 1:
		return msg
	default:
		return strings.TrimSpace(string(matches[1]))
	}
}

// mergeFlagTags Merges flag metadata together into a single FlagMetadata instance by performing a shallow merge
func mergeFlagTags(tags ...of.FlagMetadata) of.FlagMetadata {
	size := len(tags)
	switch size {
	case 0:
		return make(of.FlagMetadata)
	case 1:
		return tags[0]
	default:
		merged := make(of.FlagMetadata)
		for _, t := range tags {
			for key, value := range t {
				merged[key] = value
			}
		}
		return merged
	}
}
