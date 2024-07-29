package controller_test

import (
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/provider_v2/controller"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCacheXXX(t *testing.T) {
	c := controller.NewCache(10, 1*time.Minute)

	brd := openfeature.BoolResolutionDetail{
		Value: true,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.ResolutionError{},
			Reason:          "TARGETING_MATCH",
			Variant:         "varA",
			FlagMetadata:    nil,
		},
	}
	evalCtx := openfeature.FlattenedContext{
		"targetingKey": "5e83aec4-0559-415a-82a9-f2d751ba47c0",
	}
	err := c.Set("flag", evalCtx, brd)
	assert.NoError(t, err)

	got, err := c.Get("flag", evalCtx)
	assert.NoError(t, err)
	assert.Equal(t, brd, got)

	assert.IsType(t, openfeature.BoolResolutionDetail{}, got)
}
