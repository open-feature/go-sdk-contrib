package controller_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.openfeature.dev/contrib/providers/go-feature-flag/v2/pkg/controller"
	"go.openfeature.dev/openfeature/v2"
)

func TestCache(t *testing.T) {
	evalCtx := openfeature.FlattenedContext{
		"targetingKey": "5e83aec4-0559-415a-82a9-f2d751ba47c0",
	}

	t.Run("should return a BoolResolutionDetail", func(t *testing.T) {
		c := controller.NewCache(10, 1*time.Minute, false)
		brd := openfeature.BoolResolutionDetail{
			Value: true,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.ResolutionError{},
				Reason:          "TARGETING_MATCH",
				Variant:         "varA",
				FlagMetadata:    nil,
			},
		}
		err := c.Set("flag", evalCtx, brd)
		require.NoError(t, err)
		got, err := c.GetBool("flag", evalCtx)
		require.NoError(t, err)
		assert.Equal(t, &brd, got)
		assert.IsType(t, &openfeature.BoolResolutionDetail{}, got)
	})

	t.Run("should return a StringResolutionDetail", func(t *testing.T) {
		c := controller.NewCache(10, 1*time.Minute, false)
		brd := openfeature.StringResolutionDetail{
			Value: "xxx",
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.ResolutionError{},
				Reason:          "TARGETING_MATCH",
				Variant:         "varA",
				FlagMetadata:    nil,
			},
		}
		err := c.Set("flag", evalCtx, brd)
		require.NoError(t, err)
		got, err := c.GetString("flag", evalCtx)
		require.NoError(t, err)
		assert.Equal(t, &brd, got)
		assert.IsType(t, &openfeature.StringResolutionDetail{}, got)
	})

	t.Run("should return a FloatResolutionDetail", func(t *testing.T) {
		c := controller.NewCache(10, 1*time.Minute, false)
		brd := openfeature.FloatResolutionDetail{
			Value: 1.1,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.ResolutionError{},
				Reason:          "TARGETING_MATCH",
				Variant:         "varA",
				FlagMetadata:    nil,
			},
		}
		err := c.Set("flag", evalCtx, brd)
		require.NoError(t, err)
		got, err := c.GetFloat("flag", evalCtx)
		require.NoError(t, err)
		assert.Equal(t, &brd, got)
		assert.IsType(t, &openfeature.FloatResolutionDetail{}, got)
	})

	t.Run("should return a IntResolutionDetail", func(t *testing.T) {
		c := controller.NewCache(10, 1*time.Minute, false)
		brd := openfeature.IntResolutionDetail{
			Value: 1,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.ResolutionError{},
				Reason:          "TARGETING_MATCH",
				Variant:         "varA",
				FlagMetadata:    nil,
			},
		}
		err := c.Set("flag", evalCtx, brd)
		require.NoError(t, err)
		got, err := c.GetInt("flag", evalCtx)
		require.NoError(t, err)
		assert.Equal(t, &brd, got)
		assert.IsType(t, &openfeature.IntResolutionDetail{}, got)
	})

	t.Run("should return a InterfaceResolutionDetail", func(t *testing.T) {
		c := controller.NewCache(10, 1*time.Minute, false)
		brd := openfeature.ObjectResolutionDetail{
			Value: 1,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.ResolutionError{},
				Reason:          "TARGETING_MATCH",
				Variant:         "varA",
				FlagMetadata:    nil,
			},
		}
		err := c.Set("flag", evalCtx, brd)
		require.NoError(t, err)
		got, err := c.GetInterface("flag", evalCtx)
		require.NoError(t, err)
		assert.Equal(t, &brd, got)
		assert.IsType(t, &openfeature.ObjectResolutionDetail{}, got)
	})

	t.Run("should have a type error for Bool", func(t *testing.T) {
		c := controller.NewCache(10, 1*time.Minute, false)
		brd := openfeature.ObjectResolutionDetail{
			Value: 1,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.ResolutionError{},
				Reason:          "TARGETING_MATCH",
				Variant:         "varA",
				FlagMetadata:    nil,
			},
		}
		err := c.Set("flag", evalCtx, brd)
		require.NoError(t, err)
		_, err = c.GetBool("flag", evalCtx)
		require.Error(t, err)
	})

	t.Run("should have a type error for String", func(t *testing.T) {
		c := controller.NewCache(10, 1*time.Minute, false)
		brd := openfeature.ObjectResolutionDetail{
			Value: 1,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.ResolutionError{},
				Reason:          "TARGETING_MATCH",
				Variant:         "varA",
				FlagMetadata:    nil,
			},
		}
		err := c.Set("flag", evalCtx, brd)
		require.NoError(t, err)
		_, err = c.GetString("flag", evalCtx)
		require.Error(t, err)
	})

	t.Run("should have a type error for Float", func(t *testing.T) {
		c := controller.NewCache(10, 1*time.Minute, false)
		brd := openfeature.ObjectResolutionDetail{
			Value: 1,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.ResolutionError{},
				Reason:          "TARGETING_MATCH",
				Variant:         "varA",
				FlagMetadata:    nil,
			},
		}
		err := c.Set("flag", evalCtx, brd)
		require.NoError(t, err)
		_, err = c.GetFloat("flag", evalCtx)
		require.Error(t, err)
	})

	t.Run("should have a type error for Int", func(t *testing.T) {
		c := controller.NewCache(10, -1, false)
		brd := openfeature.ObjectResolutionDetail{
			Value: 1,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.ResolutionError{},
				Reason:          "TARGETING_MATCH",
				Variant:         "varA",
				FlagMetadata:    nil,
			},
		}
		err := c.Set("flag", evalCtx, brd)
		require.NoError(t, err)
		_, err = c.GetInt("flag", evalCtx)
		require.Error(t, err)
	})

	t.Run("should have a type error for Interface", func(t *testing.T) {
		c := controller.NewCache(10, 1*time.Minute, false)
		brd := openfeature.IntResolutionDetail{
			Value: 1,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.ResolutionError{},
				Reason:          "TARGETING_MATCH",
				Variant:         "varA",
				FlagMetadata:    nil,
			},
		}
		err := c.Set("flag", evalCtx, brd)
		require.NoError(t, err)
		_, err = c.GetInterface("flag", evalCtx)
		require.Error(t, err)
	})

	t.Run("should return nil if cache disabled", func(t *testing.T) {
		c := controller.NewCache(10, 0, true)
		val, err := c.GetBool("flag", evalCtx)
		require.NoError(t, err)
		assert.Nil(t, val)

		val1, err1 := c.GetString("flag", evalCtx)
		require.NoError(t, err1)
		assert.Nil(t, val1)

		val2, err2 := c.GetFloat("flag", evalCtx)
		require.NoError(t, err2)
		assert.Nil(t, val2)

		val3, err3 := c.GetInt("flag", evalCtx)
		require.NoError(t, err3)
		assert.Nil(t, val3)

		val4, err4 := c.GetInterface("flag", evalCtx)
		require.NoError(t, err4)
		assert.Nil(t, val4)
	})

	t.Run("should return nil if cache disabled", func(t *testing.T) {
		c := controller.NewCache(10, 1*time.Minute, true)
		err := c.Set("flag", evalCtx, openfeature.BoolResolutionDetail{})
		require.NoError(t, err)
	})
}
