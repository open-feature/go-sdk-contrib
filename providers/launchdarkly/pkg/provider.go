package launchdarkly

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldreason"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/open-feature/go-sdk/openfeature"

	ld "github.com/launchdarkly/go-server-sdk/v6"
)

var errKeyMissing = errors.New("key and targetingKey attributes are missing, at least 1 required")

// Scream at compile time if Provider does not implement FeatureProvider
var _ openfeature.FeatureProvider = (*Provider)(nil)

type Option func(*options)

// options contains all the optional arguments supported by Provider.
type options struct {
	kindAttr string
	l        Logger
}

// WithLogger sets a logger implementation. By default a noop logger is used.
func WithLogger(l Logger) Option {
	return func(o *options) {
		o.l = l
	}
}

// WithKindAttr sets the name of the LaunchDarkly Context kind attribute to recognize.
// By default, "kind" is used.
func WithKindAttr(name string) Option {
	return func(o *options) {
		o.kindAttr = name
	}
}

// Provider implements the FeatureProvider interface for LaunchDarkly.
type Provider struct {
	options
	client *ld.LDClient
}

// NewProvider creates a new LaunchDarkly OpenFeature Provider instance.
func NewProvider(ldclient *ld.LDClient, opts ...Option) *Provider {
	p := &Provider{
		client: ldclient,
		options: options{
			l:        &NoOpLogger{},
			kindAttr: "kind",
		},
	}

	for _, opt := range opts {
		opt(&p.options)
	}
	return p
}

// Metadata returns metadata about the provider.
func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "LaunchDarkly",
	}
}

// toMultiLDContext takes an OpenFeature context and maps it to a LaunchDarkly's multi context kind.
func (p *Provider) toMultiLDContext(evalCtx openfeature.FlattenedContext) (ldcontext.Context, error) {
	ldCtx := ldcontext.NewMultiBuilder()

	// each top level attribute is a kind, and must have a map or struct as value
	for key, attrs := range evalCtx {
		// skip the kind attribute since it is implicit with multi-contexts
		if key == p.kindAttr {
			continue
		}

		p.l.Debug("mapping %q context kind", key)
		if innerCtx, ok := attrs.(map[string]any); ok {
			ctx, err := p.mapContext(ldcontext.Kind(key), openfeature.FlattenedContext(innerCtx))
			if err != nil {
				return ldCtx.Build(), err
			}
			ldCtx.Add(ctx)
		} else {
			p.l.Warn("multi-context: unexpected type in top-level attribute: %s", key)
		}
	}

	return ldCtx.Build(), nil
}

// mapContext maps OpenFeature evaluation context values, and builds the LaunchDarkly context with them.
func (p *Provider) mapContext(kind ldcontext.Kind, evalCtx openfeature.FlattenedContext) (ldcontext.Context, error) {
	var emptyCtx ldcontext.Context
	// Give targetingKey precedence over a key attribute, if both are found.
	key, ok := evalCtx[openfeature.TargetingKey].(string)
	if !ok {
		ldKey, ok := evalCtx["key"].(string)
		if !ok || strings.Trim(ldKey, " ") == "" {
			return emptyCtx, errKeyMissing
		}
		key = ldKey
	}

	ldCtx := ldcontext.NewBuilder(key).Kind(kind)
	if anon, ok := evalCtx["anonymous"].(bool); ok {
		ldCtx.Anonymous(anon)
	}

	if privateAttrs, ok := evalCtx["privateAttributes"].([]string); ok {
		ldCtx.Private(privateAttrs...)
	}

	skipList := []string{
		openfeature.TargetingKey,
		p.kindAttr,
		"key",
		"privateAttributes",
		"anonymous",
	}

	for key, value := range evalCtx {
		// skip attributes that were already added to the context
		if slices.Contains(skipList, key) {
			continue
		}

		ldValue := ldvalue.CopyArbitraryValue(value)
		ldCtx.SetValue(key, ldValue)
	}

	return ldCtx.Build(), nil
}

// toLDContext returns a LaunchDarkly evaluation context, following similar
// conventions adopted by the public LaunchDarkly OpenFeature providers written
// for other programming languages.
func (p *Provider) toLDContext(evalCtx openfeature.FlattenedContext) (ldcontext.Context, error) {
	kind := ldcontext.DefaultKind

	k, ok := evalCtx[p.kindAttr].(string)
	if ok && strings.Trim(k, " ") != "" {
		kind = ldcontext.Kind(k)
	} else {
		p.l.Warn("no context kind set, setting %q by default", kind)
	}

	if kind == ldcontext.MultiKind {
		p.l.Debug("multi context detected")
		return p.toMultiLDContext(evalCtx)
	}

	p.l.Debug("single context detected")
	return p.mapContext(kind, evalCtx)
}

// toReason maps LaunchDarkly flag evaluation reasons to OpenFeatures'
func (p *Provider) toReason(reasonKind ldreason.EvalReasonKind) openfeature.Reason {
	switch reasonKind {
	case ldreason.EvalReasonOff:
		return openfeature.DisabledReason
	case ldreason.EvalReasonTargetMatch:
		return openfeature.TargetingMatchReason
	case ldreason.EvalReasonError:
		return openfeature.ErrorReason
	case ldreason.EvalReasonRuleMatch,
		ldreason.EvalReasonPrerequisiteFailed,
		ldreason.EvalReasonFallthrough:
		fallthrough
	default:
		return openfeature.Reason(reasonKind)
	}
}

// toResolutionError maps LaunchDarkly flag resolution errors to OpenFeatures'
func (p *Provider) toResolutionError(errorKind ldreason.EvalErrorKind, reason string) openfeature.ResolutionError {
	switch errorKind {
	case ldreason.EvalErrorClientNotReady:
		return openfeature.NewProviderNotReadyResolutionError(reason)
	case ldreason.EvalErrorFlagNotFound:
		return openfeature.NewFlagNotFoundResolutionError(reason)
	case ldreason.EvalErrorMalformedFlag:
		return openfeature.NewParseErrorResolutionError(reason)
	case ldreason.EvalErrorUserNotSpecified:
		return openfeature.NewTargetingKeyMissingResolutionError(reason)
	case ldreason.EvalErrorWrongType:
		return openfeature.NewTypeMismatchResolutionError(reason)
	default:
		return openfeature.NewGeneralResolutionError(reason)
	}
}

// toProviderResolutionDetail maps LaunchDarkly's flag resolution details to
// OPen
func (p *Provider) toProviderResolutionDetail(detail ldreason.EvaluationDetail) openfeature.ProviderResolutionDetail {
	p.l.Debug("launchdarkly evaluation detail: %v", detail)

	ofDetail := openfeature.ProviderResolutionDetail{
		Reason: p.toReason(detail.Reason.GetKind()),
	}

	if errorKind := detail.Reason.GetErrorKind(); errorKind != "" {
		ofDetail.ResolutionError = p.toResolutionError(errorKind, fmt.Sprintf("LaunchDarkly returned %s", detail.Reason))
	}

	if detail.VariationIndex.IsDefined() {
		ofDetail.Variant = detail.VariationIndex.String()
	}

	return ofDetail
}

// transformContext encapsulates common logic that validates the OpenFeature evaluation
// contest and translates it to LaunchDarkly's. Callers are supposed to only use
// ldCtx if an error is not returned. When an error is returned, a non-empty OpenFeature
// ResolutionDetail is returned as well with additional details.
// This function also handles context cancellations.
func (p *Provider) transformContext(ctx context.Context, evalCtx openfeature.FlattenedContext) (ldcontext.Context, openfeature.InterfaceResolutionDetail, error) {
	ldCtx, err := p.toLDContext(evalCtx)
	emptyDetail := openfeature.InterfaceResolutionDetail{}

	if err != nil {
		errMsg := err.Error()
		detail := openfeature.InterfaceResolutionDetail{
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewInvalidContextResolutionError(errMsg),
				Reason:          openfeature.ErrorReason,
			},
		}
		if errors.Is(err, errKeyMissing) {
			detail.ProviderResolutionDetail.ResolutionError = openfeature.NewTargetingKeyMissingResolutionError(errMsg)
		}
		return ldCtx, detail, err
	}

	// handle context cancellation before issuing any network calls.
	if err := ctx.Err(); err != nil {
		return ldCtx, openfeature.InterfaceResolutionDetail{
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
				Reason:          openfeature.ErrorReason,
			},
		}, err
	}

	return ldCtx, emptyDetail, nil
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flagKey string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	ldCtx, errDetail, err := p.transformContext(ctx, evalCtx)
	if err != nil {
		return openfeature.BoolResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: errDetail.ProviderResolutionDetail,
		}
	}

	value, detail, err := p.client.BoolVariationDetail(flagKey, ldCtx, defaultValue)
	if err != nil {
		p.l.Error("boolean evaluation: %s", err)
	}

	return openfeature.BoolResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: p.toProviderResolutionDetail(detail),
	}
}

// StringEvaluation evaluates a string feature flag and returns the result.
func (p *Provider) StringEvaluation(ctx context.Context, flagKey string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	ldCtx, errDetail, err := p.transformContext(ctx, evalCtx)
	if err != nil {
		return openfeature.StringResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: errDetail.ProviderResolutionDetail,
		}
	}

	value, detail, err := p.client.StringVariationDetail(flagKey, ldCtx, defaultValue)
	if err != nil {
		p.l.Error("string evaluation: %s", err)
	}

	return openfeature.StringResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: p.toProviderResolutionDetail(detail),
	}
}

// FloatEvaluation evaluates a float feature flag and returns the result.
func (p *Provider) FloatEvaluation(ctx context.Context, flagKey string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	ldCtx, errDetail, err := p.transformContext(ctx, evalCtx)
	if err != nil {
		return openfeature.FloatResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: errDetail.ProviderResolutionDetail,
		}
	}

	value, detail, err := p.client.Float64VariationDetail(flagKey, ldCtx, defaultValue)
	if err != nil {
		p.l.Error("float evaluation: %s", err)
	}

	return openfeature.FloatResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: p.toProviderResolutionDetail(detail),
	}
}

// IntEvaluation evaluates an integer feature flag and returns the result.
func (p *Provider) IntEvaluation(ctx context.Context, flagKey string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	ldCtx, errDetail, err := p.transformContext(ctx, evalCtx)
	if err != nil {
		return openfeature.IntResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: errDetail.ProviderResolutionDetail,
		}
	}

	value, detail, err := p.client.IntVariationDetail(flagKey, ldCtx, int(defaultValue))
	if err != nil {
		p.l.Error("int evaluation: %s", err)
	}

	return openfeature.IntResolutionDetail{
		Value:                    int64(value),
		ProviderResolutionDetail: p.toProviderResolutionDetail(detail),
	}
}

// ObjectEvaluation evaluates an object feature flag and returns the result.
func (p *Provider) ObjectEvaluation(ctx context.Context, flagKey string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	ldCtx, errDetail, err := p.transformContext(ctx, evalCtx)
	if err != nil {
		return openfeature.InterfaceResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: errDetail.ProviderResolutionDetail,
		}
	}

	value, detail, err := p.client.JSONVariationDetail(flagKey, ldCtx, ldvalue.CopyArbitraryValue(defaultValue))
	if err != nil {
		p.l.Error("object evaluation: %s", err)
	}

	return openfeature.InterfaceResolutionDetail{
		Value:                    value.AsArbitraryValue(),
		ProviderResolutionDetail: p.toProviderResolutionDetail(detail),
	}
}

// Hooks returns any hooks implemented by the provider. Not supported by LaunchDarkly.
func (p *Provider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}
