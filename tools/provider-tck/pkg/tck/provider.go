package tck

import (
	"context"
	"errors"
	"maps"
	"sync"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// eventBuffer is the depth of a ControllableProvider's event channel. The suite
// produces at most one event per scenario, so anything above one is slack; the
// buffer exists so that emitting never blocks the caller.
const eventBuffer = 16

// ErrProviderShutdown is returned by ControllableProvider mutators called after
// the provider has been shut down.
var ErrProviderShutdown = errors.New("provider has been shut down")

// ControllableProvider is an in-memory provider whose flag set can be changed
// at runtime, emitting PROVIDER_CONFIGURATION_CHANGED when it is.
//
// # Why this type exists
//
// [Appendix A] of the specification requires an SDK's in-memory provider to
// "support a means of updating the flag set, resulting in the emission of
// PROVIDER_CONFIGURATION_CHANGED events". The Go SDK's
// memprovider.InMemoryProvider does not: it has no update method, does not
// implement openfeature.EventHandler, and does not implement
// openfeature.StateHandler. The JavaScript and Java SDKs both provide this
// (putConfiguration and updateFlag respectively).
//
// The TCK therefore ships the missing piece, wrapping the SDK's provider rather
// than reimplementing it: every resolution decision — variants, reasons, type
// matching, the FLAG_NOT_FOUND and TYPE_MISMATCH mapping — is still made by
// memprovider.InMemoryProvider. Only the update-and-emit behaviour is added
// here. That keeps this an honest reference for what the SDK's provider should
// grow, rather than a second implementation that could disagree with it.
//
// Tracked as a Go SDK gap; see the README's "Findings" section.
//
// # Pointer receiver, deliberately
//
// Every method is on the pointer receiver, and this type must always be used as
// a pointer. The SDK compares providers with reflect.DeepEqual unless the
// dynamic type is a pointer (see providerReference.equals in the SDK). Two
// distinct value-typed providers holding equal flag sets would compare as the
// same provider, and the per-scenario replacement the TCK relies on for
// isolation would silently not happen.
//
// [Appendix A]: https://github.com/open-feature/spec/blob/main/specification/appendix-a-included-utilities.md
type ControllableProvider struct {
	// mu guards every field below. It is held across event emission, which is
	// safe because emission is non-blocking.
	mu sync.Mutex

	// inner is rebuilt on every update rather than mutated in place.
	//
	// memprovider.InMemoryProvider keeps the map it was constructed with by
	// reference and reads it without synchronisation, so mutating that map
	// while an evaluation is in flight is a data race — one the race detector
	// would report against the SDK rather than against this code. Rebuilding
	// over a copied map keeps every reader on an immutable snapshot.
	inner memprovider.InMemoryProvider

	// flags is the current flag set, owned by this provider after construction.
	flags map[string]memprovider.InMemoryFlag

	events chan openfeature.Event
	closed bool
}

// Compile-time proof that this type implements the three provider interfaces
// the TCK depends on. The SDK treats StateHandler and EventHandler as opt-in,
// so a missing method is not a compile error anywhere else — it just silently
// stops the provider from ever emitting an event.
var (
	_ openfeature.FeatureProvider = (*ControllableProvider)(nil)
	_ openfeature.StateHandler    = (*ControllableProvider)(nil)
	_ openfeature.EventHandler    = (*ControllableProvider)(nil)
)

// NewControllableProvider returns a provider serving a copy of the given flag
// set.
//
// The caller keeps ownership of the map it passes; the provider copies it.
func NewControllableProvider(flags map[string]memprovider.InMemoryFlag) *ControllableProvider {
	owned := maps.Clone(flags)
	if owned == nil {
		owned = map[string]memprovider.InMemoryFlag{}
	}
	return &ControllableProvider{
		inner:  memprovider.NewInMemoryProvider(owned),
		flags:  owned,
		events: make(chan openfeature.Event, eventBuffer),
	}
}

// Metadata implements openfeature.FeatureProvider.
func (p *ControllableProvider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{Name: "TckControllableProvider"}
}

// snapshot returns the provider to evaluate against. The returned value shares
// an immutable flag map with no writer, so evaluation happens outside the lock.
func (p *ControllableProvider) snapshot() memprovider.InMemoryProvider {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.inner
}

// BooleanEvaluation implements openfeature.FeatureProvider.
func (p *ControllableProvider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	inner := p.snapshot()
	return inner.BooleanEvaluation(ctx, flag, defaultValue, flatCtx)
}

// StringEvaluation implements openfeature.FeatureProvider.
func (p *ControllableProvider) StringEvaluation(ctx context.Context, flag string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	inner := p.snapshot()
	return inner.StringEvaluation(ctx, flag, defaultValue, flatCtx)
}

// FloatEvaluation implements openfeature.FeatureProvider.
func (p *ControllableProvider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	inner := p.snapshot()
	return inner.FloatEvaluation(ctx, flag, defaultValue, flatCtx)
}

// IntEvaluation implements openfeature.FeatureProvider.
func (p *ControllableProvider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	inner := p.snapshot()
	return inner.IntEvaluation(ctx, flag, defaultValue, flatCtx)
}

// ObjectEvaluation implements openfeature.FeatureProvider.
func (p *ControllableProvider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	inner := p.snapshot()
	return inner.ObjectEvaluation(ctx, flag, defaultValue, flatCtx)
}

// Hooks implements openfeature.FeatureProvider.
func (p *ControllableProvider) Hooks() []openfeature.Hook {
	inner := p.snapshot()
	return inner.Hooks()
}

// Init implements openfeature.StateHandler.
//
// An in-memory provider has nothing to connect to, so initialisation always
// succeeds. The interface is implemented for Shutdown, which the SDK calls when
// this provider is replaced and which is what closes the event channel.
func (p *ControllableProvider) Init(openfeature.EvaluationContext) error { return nil }

// Shutdown implements openfeature.StateHandler.
//
// Closing the event channel is what ends the SDK's listener goroutine for this
// provider. It is safe to call more than once.
func (p *ControllableProvider) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	close(p.events)
}

// EventChannel implements openfeature.EventHandler.
func (p *ControllableProvider) EventChannel() <-chan openfeature.Event {
	return p.events
}

// UpdateFlag replaces one flag and emits PROVIDER_CONFIGURATION_CHANGED naming
// it.
func (p *ControllableProvider) UpdateFlag(key string, flag memprovider.InMemoryFlag) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrProviderShutdown
	}

	updated := maps.Clone(p.flags)
	updated[key] = flag
	p.flags = updated
	p.inner = memprovider.NewInMemoryProvider(updated)

	return p.emitLocked([]string{key})
}

// Flag returns the flag currently registered under key.
func (p *ControllableProvider) Flag(key string) (memprovider.InMemoryFlag, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	flag, ok := p.flags[key]
	return flag, ok
}

// emitLocked publishes a configuration-change event. The caller holds p.mu.
//
// The send is non-blocking. A provider that was created but never registered
// with the OpenFeature API has nobody draining its channel, and a blocking send
// there would hang the scenario with no indication of why; a full buffer is
// reported instead, which cannot happen for the one-event-per-scenario the
// suite produces.
func (p *ControllableProvider) emitLocked(changed []string) error {
	event := openfeature.Event{
		ProviderName: p.Metadata().Name,
		EventType:    openfeature.ProviderConfigChange,
		ProviderEventDetails: openfeature.ProviderEventDetails{
			Message:     "flag configuration changed",
			FlagChanges: changed,
		},
	}

	select {
	case p.events <- event:
		return nil
	default:
		return errors.New("provider event buffer is full: nothing is draining the event channel, " +
			"which usually means the provider was never registered with the OpenFeature API")
	}
}
