package strategies

import (
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
)

func TestShouldEvaluateThisProvider(t *testing.T) {
	strategy := BaseEvaluationStrategy{}

	tests := []struct {
		status     openfeature.State
		shouldEval bool
	}{
		{openfeature.NotReadyState, false},
		{openfeature.FatalState, false},
		{openfeature.ReadyState, true},
	}

	for _, test := range tests {
		ctx := StrategyPerProviderContext{
			Status: test.status,
		}
		result := strategy.ShouldEvaluateThisProvider(ctx, openfeature.EvaluationContext{})
		assert.Equal(t, test.shouldEval, result)
	}
}

func TestShouldEvaluateNextProvider(t *testing.T) {
	strategy := BaseEvaluationStrategy{}
	ctx := StrategyPerProviderContext{}
	result := ResolutionDetail[openfeature.Type]{}
	assert.True(t, strategy.ShouldEvaluateNextProvider(ctx, openfeature.EvaluationContext{}, result))
}

func TestDetermineFinalResultPanics(t *testing.T) {
	strategy := BaseEvaluationStrategy{}

	assert.Panics(t, func() {
		strategy.DetermineFinalResult(StrategyEvaluationContext{}, openfeature.EvaluationContext{}, nil)
	})
}

func TestHasError(t *testing.T) {
	noError := ResolutionDetail[openfeature.Type]{}
	withError := ResolutionDetail[openfeature.Type]{
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewGeneralResolutionError("something broke"),
		},
	}

	assert.False(t, HasError(noError))
	assert.True(t, HasError(withError))
}

func TestHasErrorWithCode(t *testing.T) {
	resWithCode := ResolutionDetail[openfeature.Type]{
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewParseErrorResolutionError("bad parse"),
		},
	}
	assert.True(t, HasErrorWithCode(resWithCode, openfeature.ParseErrorCode))
	assert.False(t, HasErrorWithCode(resWithCode, openfeature.GeneralCode))
}

func TestCollectProviderErrors(t *testing.T) {
	err1 := openfeature.NewParseErrorResolutionError("bad parse")
	err2 := openfeature.NewFlagNotFoundResolutionError("what flag")

	resolutions := []ResolutionDetail[openfeature.Type]{
		{ProviderName: "prov1"},
		{ProviderName: "prov2", ProviderResolutionDetail: openfeature.ProviderResolutionDetail{ResolutionError: err1}},
		{ProviderName: "prov3", ProviderResolutionDetail: openfeature.ProviderResolutionDetail{ResolutionError: err2}},
	}

	final := CollectProviderErrors(resolutions)
	assert.Len(t, final.Errors, 2)
	assert.Equal(t, "prov2", final.Errors[0].ProviderName)
	assert.Equal(t, "prov3", final.Errors[1].ProviderName)
}

func TestResolutionToFinal(t *testing.T) {
	res := ResolutionDetail[openfeature.Type]{
		Value:        openfeature.Boolean,
		ProviderName: "myProvider",
	}

	final := ResolutionToFinal(res)
	assert.Equal(t, "myProvider", final.ProviderName)
	assert.Equal(t, res, final.Details)
	assert.Empty(t, final.Errors)
}
