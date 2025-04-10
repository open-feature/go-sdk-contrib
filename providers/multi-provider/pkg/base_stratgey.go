package multiprovider

import (
	"github.com/open-feature/go-sdk/openfeature"
)

// RunModeValue indicates whether providers are evaluated sequentially or in parallel.
type RunModeValue string

const (
	RunModeSequential = "sequential"
	RunModeParallel   = "parallel"
)

// StrategyEvaluationContext contains flag-wide info.
type StrategyEvaluationContext struct {
	FlagKey  string
	FlagType openfeature.Type
}

type StrategyPerProviderContext struct {
	StrategyEvaluationContext
	Provider     openfeature.FeatureProvider
	ProviderName string
	Status       openfeature.State
}

type FinalResult[T openfeature.Type] struct {
	Provider     openfeature.FeatureProvider
	ProviderName string
	Details      interface{}
	Errors       []openfeature.ResolutionError
}

func (fr *FinalResult[T]) determineDetails(detailType openfeature.Type) {
	switch detailType {

	case openfeature.Boolean:
		fr.Details = openfeature.BoolResolutionDetail{}
	case openfeature.String:
		fr.Details = openfeature.StringResolutionDetail{}
	case openfeature.Float:
		fr.Details = openfeature.FloatResolutionDetail{}
	case openfeature.Int:
		fr.Details = openfeature.IntResolutionDetail{}
	case openfeature.Object:
		fr.Details = openfeature.InterfaceResolutionDetail{}

	}
}

type EvaluationStrategy interface {
	// ShouldEvaluateThisProvider determines if the provider should be evaluated
	ShouldEvaluateThisProvider(strategyContext StrategyPerProviderContext, evalContext openfeature.EvaluationContext) bool

	// ShouldEvaluateNextProvider determines whether the next provider should be evaluated
	ShouldEvaluateNextProvider(strategyContext StrategyPerProviderContext, evalContext openfeature.EvaluationContext, result openfeature.InterfaceResolutionDetail) bool

	// DetermineFinalResult decides the final result from the evaluated providers
	DetermineFinalResult(strategyContext StrategyEvaluationContext, evalContext openfeature.EvaluationContext, results []openfeature.InterfaceResolutionDetail) FinalResult[openfeature.Type]
}

// BaseEvaluationStrategy Provides default implementations for the methods that can be used to create user defined strategies
type BaseEvaluationStrategy struct {
	RunMode RunModeValue
}

// ShouldEvaluateThisProvider checks if the provider should be evaluated
func (s *BaseEvaluationStrategy) ShouldEvaluateThisProvider(strategyContext StrategyPerProviderContext, evalContext openfeature.EvaluationContext) bool {
	if strategyContext.Status == openfeature.NotReadyState || strategyContext.Status == openfeature.FatalState {
		return false
	}
	return true
}

func (s *BaseEvaluationStrategy) ShouldEvaluateNextProvider(strategyContext StrategyPerProviderContext, evalContext openfeature.EvaluationContext, result openfeature.InterfaceResolutionDetail) bool {
	return true
}

func (s *BaseEvaluationStrategy) DetermineFinalResult(strategyContext StrategyEvaluationContext, evalContext openfeature.EvaluationContext, results []openfeature.InterfaceResolutionDetail) {
}
