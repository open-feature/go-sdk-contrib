package evaluate

import (
	"context"
	"fmt"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/outbound"
	of "github.com/open-feature/go-sdk/openfeature"
)

type Flags struct {
	resolver resolver
}

func NewFlagsEvaluator(uri string, callback outbound.AuthCallback) *Flags {
	client := outbound.NewOutbound(uri, callback)

	return &Flags{
		resolver: NewOutboundResolver(client),
	}
}

func (h Flags) ResolveBoolean(ctx context.Context, key string, defaultValue bool, evalCtx map[string]interface{}) of.BoolResolutionDetail {
	evalSuccess, resolutionError := h.resolver.resolve(ctx, key, evalCtx)
	if resolutionError != nil {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: *resolutionError,
				Reason:          of.ErrorReason,
			},
		}
	}

	b, ok := evalSuccess.Value.(bool)
	if !ok {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(fmt.Sprintf(
					"resolved value %v is not of boolean type", evalSuccess.Value)),
				Reason: of.ErrorReason,
			},
		}
	}

	return of.BoolResolutionDetail{
		Value: b,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(evalSuccess.Reason),
			Variant:      evalSuccess.Variant,
			FlagMetadata: evalSuccess.Metadata,
		},
	}
}

func (h Flags) ResolveString(ctx context.Context, key string, defaultValue string, evalCtx map[string]interface{}) of.StringResolutionDetail {
	evalSuccess, resolutionError := h.resolver.resolve(ctx, key, evalCtx)
	if resolutionError != nil {
		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: *resolutionError,
				Reason:          of.ErrorReason,
			},
		}
	}

	b, ok := evalSuccess.Value.(string)
	if !ok {
		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(fmt.Sprintf(
					"resolved value %v is not of string type", evalSuccess.Value)),
				Reason: of.ErrorReason,
			},
		}
	}

	return of.StringResolutionDetail{
		Value: b,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(evalSuccess.Reason),
			Variant:      evalSuccess.Variant,
			FlagMetadata: evalSuccess.Metadata,
		},
	}
}

func (h Flags) ResolveFloat(ctx context.Context, key string, defaultValue float64, evalCtx map[string]interface{}) of.FloatResolutionDetail {
	evalSuccess, resolutionError := h.resolver.resolve(ctx, key, evalCtx)
	if resolutionError != nil {
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: *resolutionError,
				Reason:          of.ErrorReason,
			},
		}
	}

	b, ok := evalSuccess.Value.(float64)
	if !ok {
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(fmt.Sprintf(
					"resolved value %v is not of float type", evalSuccess.Value)),
				Reason: of.ErrorReason,
			},
		}
	}

	return of.FloatResolutionDetail{
		Value: b,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(evalSuccess.Reason),
			Variant:      evalSuccess.Variant,
			FlagMetadata: evalSuccess.Metadata,
		},
	}
}

func (h Flags) ResolveInt(ctx context.Context, key string, defaultValue int64, evalCtx map[string]interface{}) of.IntResolutionDetail {
	evalSuccess, resolutionError := h.resolver.resolve(ctx, key, evalCtx)
	if resolutionError != nil {
		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: *resolutionError,
				Reason:          of.ErrorReason,
			},
		}
	}

	b, ok := evalSuccess.Value.(int64)
	if !ok {
		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(fmt.Sprintf(
					"resolved value %v is not of integer type", evalSuccess.Value)),
				Reason: of.ErrorReason,
			},
		}
	}

	return of.IntResolutionDetail{
		Value: b,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(evalSuccess.Reason),
			Variant:      evalSuccess.Variant,
			FlagMetadata: evalSuccess.Metadata,
		},
	}
}

func (h Flags) ResolveObject(ctx context.Context, key string, defaultValue interface{}, evalCtx map[string]interface{}) of.InterfaceResolutionDetail {
	evalSuccess, resolutionError := h.resolver.resolve(ctx, key, evalCtx)
	if resolutionError != nil {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: *resolutionError,
				Reason:          of.ErrorReason,
			},
		}
	}

	return of.InterfaceResolutionDetail{
		Value: evalSuccess.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(evalSuccess.Reason),
			Variant:      evalSuccess.Variant,
			FlagMetadata: evalSuccess.Metadata,
		},
	}
}
