// Package strategy Resolution strategies are defined within this package
package strategies

import (
	"context"
	of "github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"reflect"
	"regexp"
	"strings"
)

const (
	MetadataSuccessfulProviderName = "multiprovider-successful-provider-name"
	MetadataStrategyUsed           = "multiprovider-strategy-used"
	// StrategyFirstMatch First provider whose response that is not FlagNotFound will be returned. This is executed
	// sequentially, and not in parallel.
	StrategyFirstMatch EvaluationStrategy = "first-match"
	// StrategyFirstSuccess First provider response that is not an error will be returned. This is executed in parallel
	StrategyFirstSuccess EvaluationStrategy = "first-success"
	// StrategyComparison All providers are called in parallel. If all responses agree the value will be returned.
	// Otherwise, the value from the designated fallback provider's response will be returned. The fallback provider
	// will be assigned to the first provider registered. (NOT YET IMPLEMENTED, SUBJECT TO CHANGE)
	StrategyComparison EvaluationStrategy = "comparison"
)

// EvaluationStrategy Defines a strategy to use for resolving the result from multiple providers
type EvaluationStrategy = string

// Strategy Interface for evaluating providers within the multi-provider.
type Strategy interface {
	Name() EvaluationStrategy
	BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail
	StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail
	FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail
	IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail
	ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail
}

type resultConstraint interface {
	of.BoolResolutionDetail | of.IntResolutionDetail | of.StringResolutionDetail | of.FloatResolutionDetail | of.InterfaceResolutionDetail
}

type resultWrapper[R resultConstraint] struct {
	result *R
}

// buildDefaultResult Creates a default result using reflection via generics
func buildDefaultResult[R resultConstraint, DV bool | string | int64 | float64 | interface{}](strategy EvaluationStrategy, defaultValue DV, err error) resultWrapper[R] {
	result := *new(R)
	var resolutionErr of.ResolutionError
	if err == nil {
		resolutionErr = of.NewFlagNotFoundResolutionError("not found in any provider")
	} else {
		resolutionErr = of.NewGeneralResolutionError(cleanErrorMessage(err.Error()))
	}
	details := of.ProviderResolutionDetail{
		ResolutionError: resolutionErr,
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

func cleanErrorMessage(msg string) string {
	codeRegex := strings.Join([]string{
		string(of.ProviderNotReadyCode),
		string(of.ProviderFatalCode),
		string(of.FlagNotFoundCode),
		string(of.ParseErrorCode),
		string(of.TypeMismatchCode),
		string(of.TargetingKeyMissingCode),
		string(of.GeneralCode),
	}, "|")
	re := regexp.MustCompile("(?:" + codeRegex + "): (.*)")
	matches := re.FindSubmatch([]byte(msg))
	if len(matches) == 1 {
		return msg
	}

	return strings.TrimSpace(string(matches[1]))
}
