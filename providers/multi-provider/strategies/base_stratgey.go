package strategies

import (
	"strings"

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

// StrategyPerProviderContext
type StrategyPerProviderContext struct {
	StrategyEvaluationContext
	Provider     openfeature.FeatureProvider
	ProviderName string
	Status       openfeature.State
}

type ProviderError struct {
	ProviderName string
	Error        openfeature.ResolutionError
}

type ResolutionDetail[T openfeature.Type] struct {
	Value        T
	ProviderName string
	Provider     openfeature.FeatureProvider
	openfeature.ProviderResolutionDetail
}

type FinalResult[T openfeature.Type] struct {
	Provider     openfeature.FeatureProvider
	ProviderName string
	Details      ResolutionDetail[T]
	Errors       []ProviderError
}

// EvaluationStrategy is the base functions needed for a strategy
type EvaluationStrategy interface {
	// ShouldEvaluateThisProvider determines if the provider should be evaluated
	ShouldEvaluateThisProvider(strategyContext StrategyPerProviderContext, evalContext openfeature.EvaluationContext) bool

	// ShouldEvaluateNextProvider determines whether the next provider should be evaluated
	ShouldEvaluateNextProvider(strategyContext StrategyPerProviderContext, evalContext openfeature.EvaluationContext, result ResolutionDetail[openfeature.Type]) bool

	// DetermineFinalResult decides the final result from the evaluated providers
	DetermineFinalResult(strategyContext StrategyEvaluationContext, evalContext openfeature.EvaluationContext, results []ResolutionDetail[openfeature.Type]) FinalResult[openfeature.Type]
}

// BaseEvaluationStrategy Provides default implementations for the methods that can be used to create user defined strategies.
// DetermineFinalResult must be fully implemented for the user to create custom strategies
// ShouldEvaluateThisProvider & ShouldEvaluateNextProvider have default implementations
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

// ShouldEvaluateNextProvider checks if the next provider should be evaluated based on the result of the previous provider
func (s *BaseEvaluationStrategy) ShouldEvaluateNextProvider(strategyContext StrategyPerProviderContext, evalContext openfeature.EvaluationContext, result ResolutionDetail[openfeature.Type]) bool {
	return true
}

// DetermineFinalResult needs to be implemented by the user to properly define custom strategy
func (s *BaseEvaluationStrategy) DetermineFinalResult(strategyContext StrategyEvaluationContext, evalContext openfeature.EvaluationContext, results []ResolutionDetail[openfeature.Type]) FinalResult[openfeature.Type] {
	panic("DetermineFinalResult must be implemented by the custom strategy")
}

// HasError helper function used to determine if a resolution has an error
func HasError[T openfeature.Type](resolution ResolutionDetail[T]) bool {
	return resolution.ResolutionError != (openfeature.ResolutionError{})
}

// HasErrorWithCode helper function to determine if a resolution has a specific error code
func HasErrorWithCode[T openfeature.Type](resolution ResolutionDetail[T], code openfeature.ErrorCode) bool {
	if !HasError(resolution) {
		return false
	}

	return strings.HasPrefix(resolution.ResolutionError.Error(), string(code))
}

// CollectProviderErrors helper function to collate the errors to add to the final result struct
func CollectProviderErrors[T openfeature.Type](resolutions []ResolutionDetail[T]) FinalResult[T] {
	var errs []ProviderError

	for _, resolution := range resolutions {
		if HasError(resolution) {
			errs = append(errs, ProviderError{
				ProviderName: resolution.ProviderName,
				Error:        resolution.ResolutionError,
			})
		}
	}

	return FinalResult[T]{
		Errors: errs,
	}
}

// ResolutionToFinal converts successful resolution to final result
func ResolutionToFinal[T openfeature.Type](resolution ResolutionDetail[T]) FinalResult[T] {
	return FinalResult[T]{
		Provider: resolution.Provider,
		ProviderName: resolution.ProviderName,
		Details:  resolution,
		Errors: []ProviderError{},
	}
}
