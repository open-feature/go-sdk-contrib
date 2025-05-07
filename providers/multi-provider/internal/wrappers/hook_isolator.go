package wrappers

import (
	"context"
	"fmt"
	of "github.com/open-feature/go-sdk/openfeature"
	"slices"
	"sync"
)

type (
	// HookIsolator used as a wrapper around a provider that prevents context changes from leaking between providers
	// during evaluation
	HookIsolator struct {
		mu sync.Mutex
		of.FeatureProvider
		hooks           []of.Hook
		capturedContext of.HookContext
		capturedHints   of.HookHints
	}

	// EventHandlingHookIsolator is equivalent to HookIsolator, but also implements [of.EventHandler]
	EventHandlingHookIsolator struct {
		HookIsolator
	}
)

var (
	_ of.FeatureProvider = (*HookIsolator)(nil)
	_ of.Hook            = (*HookIsolator)(nil)
	_ of.EventHandler    = (*EventHandlingHookIsolator)(nil)
)

func IsolateProvider(provider of.FeatureProvider, extraHooks []of.Hook) *HookIsolator {
	return &HookIsolator{
		FeatureProvider: provider,
		hooks:           slices.Concat(provider.Hooks(), extraHooks),
	}
}

func IsolateProviderWithEvents(provider of.FeatureProvider, extraHooks []of.Hook) *EventHandlingHookIsolator {
	return &EventHandlingHookIsolator{*IsolateProvider(provider, extraHooks)}
}

func (h *EventHandlingHookIsolator) EventChannel() <-chan of.Event {
	return h.FeatureProvider.(of.EventHandler).EventChannel()
}

func (h *HookIsolator) Before(ctx context.Context, hookContext of.HookContext, hookHints of.HookHints) (*of.EvaluationContext, error) {
	// Used for capturing the context and hints
	h.mu.Lock()
	defer h.mu.Unlock()
	h.capturedContext = hookContext
	h.capturedHints = hookHints
	// Return copy of original evaluation context so any changes are isolated to each provider's hooks
	evalCtx := h.capturedContext.EvaluationContext()
	return &evalCtx, nil
}

func (h *HookIsolator) After(ctx context.Context, hookContext of.HookContext, flagEvaluationDetails of.InterfaceEvaluationDetails, hookHints of.HookHints) error {
	// Purposely left as a no-op
	return nil
}

func (h *HookIsolator) Error(ctx context.Context, hookContext of.HookContext, err error, hookHints of.HookHints) {
	// Purposely left as a no-op
}

func (h *HookIsolator) Finally(ctx context.Context, hookContext of.HookContext, hookHints of.HookHints) {
	// Purposely left as a no-op
}

func (h *HookIsolator) Metadata() of.Metadata {
	return h.FeatureProvider.Metadata()
}

func (h *HookIsolator) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	completeEval := h.evaluate(ctx, flag, of.Boolean, defaultValue, evalCtx)

	return of.BoolResolutionDetail{
		Value:                    completeEval.Value.(bool),
		ProviderResolutionDetail: toProviderResolutionDetail(completeEval),
	}
}

func (h *HookIsolator) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	completeEval := h.evaluate(ctx, flag, of.String, defaultValue, evalCtx)

	return of.StringResolutionDetail{
		Value:                    completeEval.Value.(string),
		ProviderResolutionDetail: toProviderResolutionDetail(completeEval),
	}
}

func (h *HookIsolator) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	completeEval := h.evaluate(ctx, flag, of.Float, defaultValue, evalCtx)

	return of.FloatResolutionDetail{
		Value:                    completeEval.Value.(float64),
		ProviderResolutionDetail: toProviderResolutionDetail(completeEval),
	}
}

func (h *HookIsolator) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	completeEval := h.evaluate(ctx, flag, of.Int, defaultValue, evalCtx)

	return of.IntResolutionDetail{
		Value:                    completeEval.Value.(int64),
		ProviderResolutionDetail: toProviderResolutionDetail(completeEval),
	}
}

func (h *HookIsolator) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	completeEval := h.evaluate(ctx, flag, of.Object, defaultValue, evalCtx)

	return of.InterfaceResolutionDetail{
		Value:                    completeEval.Value,
		ProviderResolutionDetail: toProviderResolutionDetail(completeEval),
	}
}

func toProviderResolutionDetail(evalDetails of.InterfaceEvaluationDetails) of.ProviderResolutionDetail {
	var resolutionErr of.ResolutionError
	var reason of.Reason
	switch evalDetails.ErrorCode {
	case of.GeneralCode:
		resolutionErr = of.NewGeneralResolutionError(evalDetails.ErrorMessage)
		reason = of.ErrorReason
	case of.FlagNotFoundCode:
		resolutionErr = of.NewFlagNotFoundResolutionError(evalDetails.ErrorMessage)
		reason = of.DefaultReason
	case of.TargetingKeyMissingCode:
		resolutionErr = of.NewTargetingKeyMissingResolutionError(evalDetails.ErrorMessage)
		reason = of.TargetingMatchReason
	case of.TypeMismatchCode:
		resolutionErr = of.NewTypeMismatchResolutionError(evalDetails.ErrorMessage)
		reason = of.ErrorReason
	case of.ParseErrorCode:
		resolutionErr = of.NewParseErrorResolutionError(evalDetails.ErrorMessage)
		reason = of.ErrorReason
	case of.InvalidContextCode:
		resolutionErr = of.NewInvalidContextResolutionError(evalDetails.ErrorMessage)
		reason = of.ErrorReason
	}
	return of.ProviderResolutionDetail{
		ResolutionError: resolutionErr,
		Reason:          reason,
		Variant:         evalDetails.Variant,
		FlagMetadata:    evalDetails.FlagMetadata,
	}
}

func (h *HookIsolator) Hooks() []of.Hook {
	// return self as hook to capture contexts
	return []of.Hook{h}
}

func (h *HookIsolator) evaluate(ctx context.Context, flag string, flagType of.Type, defaultValue interface{}, flatCtx of.FlattenedContext) of.InterfaceEvaluationDetails {
	evalDetails := of.InterfaceEvaluationDetails{
		Value: defaultValue,
		EvaluationDetails: of.EvaluationDetails{
			FlagKey:  flag,
			FlagType: flagType,
		},
	}

	defer func() {
		h.finallyHooks(ctx)
	}()

	evalCtx, err := h.beforeHooks(ctx)
	// Update hook context unconditionally
	h.updateEvalContext(evalCtx)
	if err != nil {
		//h.logger.Error(
		//	err, "before hook", "flag", flag, "defaultValue", defaultValue,
		//	"evaluationContext", flatCtx, "evaluationOptions", options, "type", flagType.String(),
		//)
		err = fmt.Errorf("before hook: %w", err)
		h.errorHooks(ctx, err)
		evalDetails.ResolutionDetail = of.ResolutionDetail{
			Reason:       of.ErrorReason,
			ErrorCode:    of.GeneralCode,
			ErrorMessage: err.Error(),
			FlagMetadata: nil,
		}
		return evalDetails
	}

	// Merge together the passed in flat context and the captured evaluation context and transform back into a flattened
	// context for evaluation
	flatCtx = flattenContext(mergeContexts(h.capturedContext.EvaluationContext(), deepenContext(flatCtx)))

	var resolution of.InterfaceResolutionDetail
	switch flagType {
	case of.Object:
		resolution = h.FeatureProvider.ObjectEvaluation(ctx, flag, defaultValue, flatCtx)
	case of.Boolean:
		defValue := defaultValue.(bool)
		res := h.FeatureProvider.BooleanEvaluation(ctx, flag, defValue, flatCtx)
		resolution.ProviderResolutionDetail = res.ProviderResolutionDetail
		resolution.Value = res.Value
	case of.String:
		defValue := defaultValue.(string)
		res := h.FeatureProvider.StringEvaluation(ctx, flag, defValue, flatCtx)
		resolution.ProviderResolutionDetail = res.ProviderResolutionDetail
		resolution.Value = res.Value
	case of.Float:
		defValue := defaultValue.(float64)
		res := h.FeatureProvider.FloatEvaluation(ctx, flag, defValue, flatCtx)
		resolution.ProviderResolutionDetail = res.ProviderResolutionDetail
		resolution.Value = res.Value
	case of.Int:
		defValue := defaultValue.(int64)
		res := h.FeatureProvider.IntEvaluation(ctx, flag, defValue, flatCtx)
		resolution.ProviderResolutionDetail = res.ProviderResolutionDetail
		resolution.Value = res.Value
	}

	err = resolution.Error()
	if err != nil {
		//h.logger.Error(
		//	err, "flag resolution", "flag", flag, "defaultValue", defaultValue,
		//	"evaluationContext", flatCtx, "evaluationOptions", options, "type", flagType.String(), "errorCode", err,
		//	"errMessage", resolution.ResolutionError.message,
		//)
		err = fmt.Errorf("error code: %w", err)
		h.errorHooks(ctx, err)
		evalDetails.ResolutionDetail = resolution.ResolutionDetail()
		evalDetails.Reason = of.ErrorReason
		return evalDetails
	}
	evalDetails.Value = resolution.Value
	evalDetails.ResolutionDetail = resolution.ResolutionDetail()

	if err := h.afterHooks(ctx, evalDetails); err != nil {
		//h.logger.Error(
		//	err, "after hook", "flag", flag, "defaultValue", defaultValue,
		//	"evaluationContext", flatCtx, "evaluationOptions", options, "type", flagType.String(),
		//)
		err = fmt.Errorf("after hook: %w", err)
		h.errorHooks(ctx, err)
		return evalDetails
	}

	return evalDetails
}

func (h *HookIsolator) beforeHooks(ctx context.Context) (of.EvaluationContext, error) {
	contexts := []of.EvaluationContext{h.capturedContext.EvaluationContext()}
	for _, hook := range h.hooks {
		resultEvalCtx, err := hook.Before(ctx, h.capturedContext, h.capturedHints)
		if resultEvalCtx != nil {
			contexts = append(contexts, *resultEvalCtx)
		}
		if err != nil {
			return mergeContexts(contexts...), err
		}
	}

	return mergeContexts(contexts...), nil
}

func (h *HookIsolator) afterHooks(ctx context.Context, evalDetails of.InterfaceEvaluationDetails) error {
	for _, hook := range h.hooks {
		if err := hook.After(ctx, h.capturedContext, evalDetails, h.capturedHints); err != nil {
			return err
		}
	}

	return nil
}

func (h *HookIsolator) errorHooks(ctx context.Context, err error) {
	for _, hook := range h.hooks {
		hook.Error(ctx, h.capturedContext, err, h.capturedHints)
	}
}

func (h *HookIsolator) finallyHooks(ctx context.Context) {
	for _, hook := range h.hooks {
		hook.Finally(ctx, h.capturedContext, h.capturedHints)
	}
}

// updateEvalContext Returns a new [of.HookContext] with an updated [of.EvaluationContext] value. [of.HookContext] is
// immutable, and this returns a new [of.HookContext] with all other values copied
func (h *HookIsolator) updateEvalContext(evalCtx of.EvaluationContext) {
	hookCtx := of.NewHookContext(
		h.capturedContext.FlagKey(),
		h.capturedContext.FlagType(),
		h.capturedContext.DefaultValue(),
		h.capturedContext.ClientMetadata(),
		h.capturedContext.ProviderMetadata(),
		evalCtx,
	)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.capturedContext = hookCtx
}

func deepenContext(flatCtx of.FlattenedContext) of.EvaluationContext {
	noTargetingKey := make(map[string]interface{})
	for k, v := range flatCtx {
		if k != "targetingKey" {
			noTargetingKey[k] = v
		}
	}
	return of.NewEvaluationContext(flatCtx["targetingKey"].(string), noTargetingKey)
}

func flattenContext(evalCtx of.EvaluationContext) of.FlattenedContext {
	flatCtx := evalCtx.Attributes()
	flatCtx["targetingKey"] = evalCtx.TargetingKey()
	return flatCtx
}

// merges attributes from the given EvaluationContexts with the nth EvaluationContext taking precedence in case
// of any conflicts with the (n+1)th EvaluationContext
func mergeContexts(evaluationContexts ...of.EvaluationContext) of.EvaluationContext {
	if len(evaluationContexts) == 0 {
		return of.EvaluationContext{}
	}
	// create initial values
	attributes := evaluationContexts[0].Attributes()
	targetingKey := evaluationContexts[0].TargetingKey()

	for i := 1; i < len(evaluationContexts); i++ {
		if targetingKey == "" && evaluationContexts[i].TargetingKey() != "" {
			targetingKey = evaluationContexts[i].TargetingKey()
		}

		for k, v := range evaluationContexts[i].Attributes() {
			_, ok := attributes[k]
			if !ok {
				attributes[k] = v
			}
		}
	}

	return of.NewEvaluationContext(targetingKey, attributes)
}
