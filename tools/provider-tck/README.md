# OpenFeature Provider TCK (Go)

A conformance suite any OpenFeature Go provider can adopt to verify that it implements the provider
contract of the specification.

OpenFeature's central promise is that swapping providers does not change application behaviour.
Nothing verifies that today, and every provider tests differently — so "implements the provider
contract" is an unverified claim, and a behavioural difference between two providers is discovered
by the application that trips over it.

This module is the Go implementation of [Appendix F][appendix-f]. It runs the same Gherkin
scenarios, against the same canonical flag set, driven through the same backend control API, as
every other language's TCK. That shared basis is the point: "conformant" only means something if the
question is identical everywhere.

Tracking issue: [open-feature/spec#417][tracking].

## Status

**Proof of concept.** The scenario set is a representative subset covering each architectural
mechanism once, not exhaustive coverage. Breaking changes should be expected.

## What it tests

- mapping backend responses onto typed resolution details: value, reason, error code, and no error
  message on a normal evaluation — plus the variant, under `@variants`, because a variant is a
  `SHOULD` and some backends have no such concept
- the values most often mistaken for an absence — `false`, `0` and `""` — resolving as values
- under `@disabled-flags`, that a flag disabled in the management system resolves to the caller's
  default with no error, which only a provider that evaluates locally can do at all
- that supplying an evaluation context to an untargeted resolution is harmless, and under
  `@targeting` that a matching one actually reaches the backend and changes the answer
- integer precision: 2^31 − 1 for every provider, and 2^53 − 1 under `@large-integers`
- keeping the integer and float types distinct, and under `@numeric-coercion` converting between
  them only when nothing is lost
- error handling: a type mismatch and an unknown flag return the code default, report the right
  error code, and never take the application down
- identity: a non-empty metadata name
- lifecycle: reaching `READY`, settling into `ERROR` against an unreachable backend, and a shutdown
  that is idempotent, reversible by initialising again, and prompt when the backend is gone
- events: `PROVIDER_READY`, `PROVIDER_ERROR`, `PROVIDER_STALE`, `PROVIDER_CONFIGURATION_CHANGED`
- that a signalled configuration change is actually **applied** on re-evaluation, not merely
  signalled

Deliberately out of scope: backend evaluation logic, bucketing and rule-language correctness (that
is the backend's contract, not the provider's — every canonical flag that is enabled and untargeted
resolves to its default variant whatever the context), the provider↔backend wire protocol, and SDK
behaviour, which belongs to the SDK's own [Appendix B][appendix-b] suite.

`targeting-key-flag` is the one exception to the "whatever the context" half. It carries a single
rule and is there to show that the evaluation context *reached* the backend rather than to test how
the backend evaluated it. Its rule is stated as behaviour — resolve `hit` for one specific targeting
key, `miss` otherwise — not as a syntax, so a backend expresses it however it expresses targeting.

The four `disabled-*` flags are the exception to the "default variant" half, and the only flags in
the set whose state is not `ENABLED`. They resolve to nothing: the caller's default stands in and no
variant is named. Every other scenario assumes a flag serves its own value, so enabling one of
these — or disabling anything else — breaks those quietly rather than loudly.

## Adopting it

One test function and one struct literal. The TCK owns the whole lifecycle: it registers each
provider with the OpenFeature API under a suite-scoped domain, waits for it to become ready, awaits
events, resets the backend between scenarios and releases the provider at the end. **If you find
yourself writing test infrastructure, that is a defect here rather than something for you to work
around.**

```go
func TestMyProviderConformance(t *testing.T) {
	control := myBackendControl()

	tck.Run(t, tck.Config{
		Name:    "my-provider",
		Control: control,
		NewProvider: func(ctx context.Context) (openfeature.FeatureProvider, error) {
			return myprovider.New(control.Address()), nil
		},
		Capabilities: []tck.Capability{tck.Events, tck.Lifecycle, tck.Object, tck.NumericCoercion, tck.LargeIntegers},
	})
}
```

Three fields are required — `Name`, `NewProvider`, `Control` — and everything else has a working
default. `NewProvider` is a factory rather than an instance because each scenario gets its own
provider, and because a provider often cannot be configured before the suite starts: a container
stack's host ports do not exist until it is up.

Each scenario becomes a Go subtest, so `-run` selects one the usual way and failures name a
scenario.

The canonical feature files and flag set arrive as an ordinary dependency, so **adopting this
module needs no git submodule** — `go get` it and everything the suite runs comes with it. Where
they come from and how the pin moves is described under [The spec module](#the-spec-module).

### Timings

`Config.EventTimeout` is the knob that matters. Providers observe backend changes on wildly
different timescales — a streaming provider sees a configuration change in milliseconds, one that
polls every 30 seconds may need most of a poll interval. Set it to comfortably exceed your
worst-case detection latency, or the suite reports timeouts that are really just impatience.

### Adding your own scenarios

A provider often has behaviour the specification does not describe and cannot — flagd's `fractional`
targeting, a vendor's segment rules. Those scenarios still need a provider registered per scenario,
a backend reset between them and the event plumbing this suite already owns, so they run *in* the
suite rather than beside it. Two optional fields:

```go
tck.Run(t, tck.Config{
	// ... Name, NewProvider, Control, Capabilities as above ...
	ExtensionFeatures: os.DirFS("testdata/tck-extensions"),
	ExtensionSteps: func(ctx *godog.ScenarioContext) {
		ctx.Step(`^the fractional bucket for "([^"]*)" is "([^"]*)"$`, theBucketIs)
	},
})
```

`ExtensionFeatures` is any `fs.FS`; every `.feature` file in it is picked up. `ExtensionSteps` is
called during scenario initialisation, after the TCK's own step definitions, so an extension step
sees the same scenario context and the same hooks. It reaches the provider under test through
`tck.ClientFromContext(ctx)` — the provider is registered under a suite-scoped domain the adopter
never names, and a step that built a client of its own would be testing a different provider.

An extension feature can use the canonical steps, and usually should: `Given a stable provider` in a
`Background` is what puts the scenario in the same lifecycle phase as the canonical ones.

Extension features are mounted under an `extensions/` prefix while the suite runs, and the canonical
ones keep their `gherkin/` paths. That partition is a contract rather than a detail: it stops
an extension file shadowing a canonical one, and it is how a result in the conformance report is
told apart from a canonical result. A filesystem holding no `.feature` file is refused rather than
quietly running the canonical suite alone, that being how mis-wired extensions otherwise go
unnoticed.

**Both fields are optional, and a `Config` without them runs exactly what it ran before they
existed.**

Where Java and Python discover extensions by convention — a classpath scan, a `conftest.py` beside
the feature files — Go has no runtime scanning, so this is configuration. The design rule across the
four languages is convention where the language can scan and configuration where it cannot; the
constraint is that the configuration stays small.

## Capabilities

Not every provider implements every optional part of the contract. A provider backed by a static
file has no meaningful notion of going stale; one without a streaming transport cannot emit
configuration-change events. Each scenario exercising an optional part carries a Gherkin tag, and a
provider declares what it supports.

**A scenario whose capability was not declared is reported as skipped, with the reason printed —
never as passed.** A conformance suite that quietly goes green on scenarios it did not run is worse
than no suite at all, so the skips and their reasons are printed next to the result rather than left
to be inferred from a scenario count.

| Capability | Tag | Meaning |
| --- | --- | --- |
| `tck.Events` | `@events` | emits lifecycle events at all |
| `tck.Lifecycle` | `@lifecycle` | performs an initialisation that reaches its backend, with an observable outcome |
| `tck.Stale` | `@stale` | enters `STALE` and emits `PROVIDER_STALE` on backend loss |
| `tck.ConfigurationChange` | `@configuration-change` | detects configuration changes and emits `PROVIDER_CONFIGURATION_CHANGED` |
| `tck.Object` | `@object` | supports structured flag values |
| `tck.Variants` | `@variants` | names the variant it resolved, which [Requirement 2.2.4][req-224] makes a `SHOULD` and `types.md` types as optional |
| `tck.DisabledFlags` | `@disabled-flags` | resolves a flag disabled in the management system to the code default, with no error |
| `tck.UnavailableInit` | `@unavailable` | reports an error state instead of hanging against a dead backend |
| `tck.NumericCoercion` | `@numeric-coercion` | coerces between integer and float only when lossless, else `TYPE_MISMATCH` |
| `tck.LargeIntegers` | `@large-integers` | resolves integers up to 2^53 − 1 exactly; undeclarable where the SDK's integer accessor is 32-bit |
| `tck.Reinitialization` | `@reinitialization` | can be initialised again after `shutdown`, which [Requirement 2.5.2][req-252] permits rather than requires |
| `tck.Targeting` | `@targeting` | resolves a flag differently for a matching evaluation context |
| `tck.Caching` | `@caching` | reserved; **not declarable** — no scenarios yet |

Untagged scenarios are mandatory and always run. `Capabilities` defaults to `tck.AllCapabilities()`
— narrow it rather than widening it: start from the default, run the suite, and remove only what
your provider genuinely cannot do.

A **reserved** capability is named by the vocabulary so there is a place for it once scenarios
exist, but it **must not be declared**. No scenario carries the tag, so declaring it cannot be
verified, cannot even produce a skip, and tells a reader of a report only that something was claimed
and nothing examined. `tck.AllCapabilities()` therefore excludes the reserved capabilities, and
naming one in `Capabilities` is rejected by configuration validation rather than passed into a
report — an unverifiable claim is a configuration mistake, not a conformance result. `@caching` is
the only reserved tag left: `@targeting` became declarable, with three scenarios, in spec
`26362f85`.

That is easy to reintroduce by accident rather than by intent: an adoption that declares everything
and then removes what it cannot do collects every reserved tag on the way past, which is how a Java
conformance report came to assert `@targeting` and `@caching` as declared — back when both were
reserved.

**Declare a capability only on evidence from running the suite, never from reading the provider's
source.** Source inspection is unreliable in both directions and demonstrably so: flagd's RPC
resolver clears its own initialised flag on shutdown, which reads as support for reuse, and then
fails to initialise again because the transport underneath cannot restart. Start from the default,
run, and withdraw what actually fails.

`@variants` is the case that earned this tag its existence. Every evaluation scenario used to assert
a variant, which reads as obviously correct until a backend with no variant concept for a plain flag
is put under test: its response carries no such key, the provider never receives one, and no seeding
can produce one. Ten scenarios failed a conformant provider for something its author could not fix,
with nothing to record as a known deviation because there was no capability to hang one on. The
value and reason assertions stay untagged, because [Requirement 2.2.3][req-223] makes the value a
`MUST`. The `reason` field is *not* modelled this way even though [Requirement 2.2.5][req-225] is
also a `SHOULD` — Appendix F records that as a deliberate narrowing rather than an oversight.

`@disabled-flags` is gated for a reason no other capability here has: the answer depends on **where
the substitution happens**, not on provider quality. A provider that evaluates locally — flagd's
in-process resolver, an in-memory flag set — holds the flag's state and can hand back the value the
caller passed in. A provider whose backend decides cannot, and OFREP is the clean case: the caller's
default never leaves the process, so the server has never seen it and no response it could send
would carry it. The same flag cannot behave the same way across those two architectures and neither
of them is wrong.

Nothing in the specification says what a provider owes a disabled flag either. [Requirement
1.4.7][req-147] is about the SDK propagating whatever reason arrived, and [Requirement
2.2.5][req-225] only lists `DISABLED` among the reason strings a provider may use — so, like
`@numeric-coercion` below, the behaviour is stated by Appendix F and gated rather than required. The
four scenarios assert the value and the absence of an error and **not** the reason: each row's
caller default differs from the flag's configured value, so a provider that ignores the state is
caught on the value alone, which rests on [Requirement 2.2.3][req-223], a `MUST`. Pinning reason
`DISABLED` would rest on 2.2.5, a `SHOULD` that permits *"some other string"*. It does not compose
with `@variants`, and that is not an omission: a disabled flag has resolved no variant, so there is
no name for a variant assertion to be about.

`@lifecycle` and `@events` are separate on purpose, and conflating them is the mistake the
vocabulary exists to prevent. The Go SDK synthesises `PROVIDER_READY` for any provider that does not
implement `openfeature.StateHandler`, so a provider with no initialisation passes the readiness
scenario without demonstrating anything — a `NoopProvider` passes it identically. Conversely a
stateless HTTP provider such as OFREP emits no events of its own and cannot declare `@events`, yet
the readiness scenario is not really about events. Declare `@lifecycle` when initialisation actually
reaches something and the client can observe how that went; declare `@events` when the provider
emits events. Neither implies the other.

`@reinitialization` is separate from `@lifecycle` for a subtler reason, and it is worth knowing how
it came to be separate, because the mistake behind it is easy to repeat. [Requirement 2.5.2][req-252]
says a provider **SHOULD** revert to its uninitialized state after `shutdown`, and its supporting
text adds that *"some providers **may** allow reinitialization from this state"*. Reuse is therefore
**permitted, not required**: a provider that releases its client on shutdown and refuses to be
started again is exercising a choice the specification offers it. Leave the tag undeclared and the
scenario is skipped — that is the whole of what is needed, and in particular it is **not** a known
deviation, because nothing is deviating.

The scenario was untagged and so mandatory until [spec `fc99d5ac`][reinit-fix]. Run against flagd's
RPC resolver it failed, was written down as a known deviation, and was one step from being filed as
a defect against a provider doing nothing wrong. **A false failure is the mirror image of a vacuous
pass**, and this suite cares about both.

What the tag buys is the other direction: a provider that does offer reuse has somewhere to be held
to it, because "`Shutdown` releases the client and `Init` returns early because an initialised flag
was never cleared" is easy to write and leaves the provider evaluating against a closed connection
rather than failing outright. Reverting the state is not separately observable — a provider that
reverts but refuses reuse presents exactly as one that did neither — so a gated reuse scenario is
the only assertion the requirement admits.

`@numeric-coercion` deserves a note, because it is the one capability here that **the specification
does not define**. OpenFeature has a single numeric type on purpose — `number` is "a numeric value of
unspecified type or size", and languages **may** differentiate between integers and floats "as idioms
dictate" — so no requirement says what a provider must do when a value does not fit the accessor it
was asked through. That gap is
[open-feature/spec#430](https://github.com/open-feature/spec/issues/430).

The rule this tag is tested against is therefore **borrowed, not normative**: lossless coercion is
permitted, lossy coercion must fail — `10.0` requested as an integer must succeed, `0.5` must not.
It comes from flagd's [numeric coercion ADR][numeric-coercion-adr]
([open-feature/flagd#1996](https://github.com/open-feature/flagd/issues/1996)), which is scoped to
flagd's own implementations, and the tag took that name — it was `@strict-numeric-typing` — because
flagd's testbed is gaining `@numeric-coercion` scenarios and two vocabularies for one observable
property is worse than one borrowed name. **A provider that behaves differently is not violating the
specification.** Withholding this capability may be a deliberate choice or a tracked defect; a
report's `knownDeviations` is where the second is recorded.

What remains true is that flagd narrows `0.5` to `0` with no error code at all, in Go and in Java and
in both resolvers, so an application sees a plausible value and no signal — which is what
flagd#1996 fixes.

Both halves of the rule have scenarios. The lossy half asks for `float-flag` (`0.5`) as an integer
and expects `TYPE_MISMATCH`; the lossless half asks for `integral-float-flag` (`10.0`) as an integer
and for `integer-flag` (`10`) as a float, and expects both to succeed. A provider declaring the tag
has to satisfy all three — rejecting every float is an easy way to pass the first, and the other
two are what stop it. The SDK's `memprovider.InMemoryProvider` is exactly such a provider: it
type-asserts and never converts between `int64` and `float64`, which is why none of the self-tests
declares the capability.

**Accessor width** is the related property the ADR distinguishes, and it has its own tag because it
is a property of the SDK rather than of the provider. Every language's integer accessor can ask for
2^31 − 1, so that precision scenario is untagged. Only some can ask for 2^53 − 1: Go's
`ResolveIntValue` is `int64`, so every Go provider can, and declaring `@large-integers` says the
value survives the trip — anything routed through a 32-bit integer, or through a float and back
with rounding, changes it. Java's accessor is a 32-bit `Integer`, and a provider there leaves the
tag undeclared. Nothing above 2^53 − 1 is asked for.

### Known deviations

Narrowing `Capabilities` says a scenario was not run. It cannot say **why**, and the two reasons are
not alike: a provider with no streaming transport declining `@configuration-change` has made a
decision, while one declining `@numeric-coercion` because it narrows `0.5` to `0` with no error code
has a bug. In the results they are indistinguishable — the same skip, carrying the same reason — so
unless the provider author says which happened, a consumer comparing providers reads a defect as a
design choice. The TCK cannot infer it: from the outside, a capability withheld by choice and one
withheld because it is broken are the same absence.

`Config.KnownDeviations` is where that gets said.

```go
KnownDeviations: []tck.KnownDeviation{
	tck.TrackedDeviation(
		tck.NumericCoercion,
		"https://github.com/open-feature/flagd/issues/1996",
		"The lossy half of the coercion rule is not enforced: float-flag (0.5) through the "+
			"integer API returns 0 with no error code, rather than TYPE_MISMATCH with the "+
			"code default.",
	),
	tck.UntrackedDeviation(
		tck.Stale,
		"The provider never leaves READY when its stream drops: the reconnect loop swallows "+
			"the transport error instead of emitting PROVIDER_STALE, so an application sees "+
			"last-known values with no signal that they are last-known.",
	),
},
```

**Check the requirement before you write one.** A scenario failed is not yet a deviation, and a
capability you cannot satisfy is not yet a defect — first find the numbered requirement the scenario
maps to and read what it actually says. Three rules in this suite have now been found asserted more
strongly than the specification states them: `@numeric-coercion` is borrowed from an ADR and is not
normative at all, `@large-integers` is a property of the SDK's accessor rather than of the provider,
and `@reinitialization` is [explicitly permitted rather than required][req-252] — that last one
reached a written-down deviation against a provider doing nothing wrong before anyone looked the
requirement up. When the specification allows a provider to decline the behaviour, the answer is a
gate, not a deviation.

`tck.UntrackedDeviation` is for a gap with no issue behind it yet, and is worth declaring even so:
naming the defect is what separates it from a capability withheld by choice, and a declaration that
merely omits the tag cannot say which of the two happened. Move it to `tck.TrackedDeviation` as soon
as there is an issue to point at, and delete the entry once the defect is fixed.

Two shapes are legitimate besides the obvious one. An entry with **no capability** is a gap against
a mandatory scenario, which belongs to no capability and so can name none. An entry naming a
capability that **is declared** covers the case where the capability holds but one of the scenarios
it gates does not — worth recording precisely because such a scenario may pass for the wrong reason
in one of a provider's modes and hide the gap there.

Empty by default, which is silence rather than a claim. A deviation with no summary is rejected by
configuration validation, because it records that something is broken without saying what and is
then worth less than the bare skip it accompanies; so is one naming a reserved capability, because
no scenario carries that tag and nothing was skipped for it to explain.

## Controlling the backend

`tck.BackendControl` is the single seam between the scenarios and whatever manipulates the backend.
Step definitions never talk to a backend directly, which is why the same Gherkin runs unchanged
against a containerised backend and against a provider manipulated in-process.

**If your provider talks to a backend, drive it over the HTTP control API** in
[`specification/assets/provider-tck/openapi/control-api.yaml`][control-api], which this module
exposes as bytes through `tck.ControlAPISpec()`. That API is
the normative contract for those providers, and it is what makes a conformance claim portable:
another language's TCK drives the same endpoints against the same stack and must get the same
answers.

You do not have to write the client. `tck.HTTPControl` is one, and it ships here rather than in an
adoption precisely because every adoption needs the same one:

```go
control, err := tck.NewHTTPControl(tck.HTTPControlOptions{
    BaseURL: fmt.Sprintf("http://localhost:%d", launchpad.MappedPort()),
})
```

`BaseURL` is the only required field, and it must be built from the **dynamically mapped** host port
discovered after the stack is up — a stack under test must not pin host ports. `Configuration`
defaults to `tck.DefaultConfiguration`, the one name Appendix F requires every backend under test to
serve, and the one that serves the canonical flag set.

If you find yourself writing a control client of your own, that is a defect here rather than
something for you to work around.

Two of its requirements are easy to get wrong:

- **Containers are never stopped or restarted mid-suite.** Unavailability is simulated *inside* the
  running stack — a process kill, a proxy toxic, a socket block. This is portability, not
  preference: container orchestrators assign host ports dynamically and cannot reliably preserve
  them across a restart, so restarting silently invalidates every provider already pointed at the
  old port, and the failure looks like a flaky provider.
- **`/start` resets flag state; `/restart` preserves it.** An outage must be observable as a change
  in availability, never as a change in flag values.

### Providers with no backend

An in-memory, environment-variable or file-based provider has nothing to connect to and no control
API to expose. Those may control the backend in-process, where flag operations are direct
manipulations of the provider's own state. `tck.InProcessControl` is the reference.

This is a narrow allowance and the obvious thing to abuse. **A provider with an external backend
must use the control API.** Reaching into an external backend from inside the test process — a
test-only admin client, a shared database handle, a hook inside the provider — produces a suite that
passes while proving nothing, because the path it exercised is not the path the contract describes.

Connection-dependent scenarios have no meaning without a connection, so a backend-less control
simply does not implement `tck.ConnectionControl`, leaves `Stale` and `UnavailableInit` undeclared,
and those scenarios are skipped with their reason. Declaring the capability anyway fails loudly
rather than silently passing — that is a test-configuration bug, not a provider defect.

## The spec module

The Gherkin, the canonical flag set and the control API are not owned by this repository. They are
the language-agnostic definitions in [open-feature/spec][spec], and that directory is also a Go
module, `github.com/open-feature/spec/specification/assets/provider-tck`, whose only content is an
`embed.FS` of them. This package depends on it like on any other module. Nothing is copied and
nothing is generated: `pkg/tck/assets.go` reads the embedded files out of the dependency, so the
specification revision this suite conforms to is the version pinned in `go.mod` and nothing else,
and `go.sum` guarantees that version always resolves to the same bytes.

**Adopters need no submodule.** The dependency is fetched with this module, so a provider consuming
it gets the assets with it.

**Contributors need none either.** There is no submodule to initialise; a plain clone builds.

A git submodule would not work here, which is why the other language TCKs use one and this one
does not: a Go module is distributed as a zip of the VCS tree, in which a submodule is only a
gitlink, so an embed from a submodule compiles in this repository and arrives empty for anyone
running `go get`.

Changing a scenario, a flag or a control endpoint means changing it in `open-feature/spec` first and
then moving the pin here, by tag or by commit:

```console
cd tools/provider-tck
go get github.com/open-feature/spec/specification/assets/provider-tck@<commit-or-tag>
```

Nested Go modules are tagged with their path as a prefix, so the tag form is
`specification/assets/provider-tck/vX.Y.Z`; until the spec publishes one, the pin is a
pseudo-version naming the exact commit.

## Go-specific translation notes

Three places where the shared Gherkin needed a decision rather than a transcription:

**"no exception should have been thrown"** asserts that nothing the scenario asked of the provider
**panicked**: the evaluation, and any `Shutdown` or `Init` the lifecycle steps called. Go has no
exceptions, and the `error` an evaluation returns is not one: an errored evaluation correctly
returns the code default *alongside* a non-nil error, which is the normal shape of the API. The
behaviour the feature files forbid — an unhandled failure escaping the provider and taking the host
application down — is a panic here. Registration is guarded the same way, because a provider that
panics out of `SetProvider` takes the application with it. The one returned error that does count is
`Init`'s: a provider that cannot be initialised again after a shutdown has not reverted to its
uninitialised state, which is what the other languages' `initialize()` throws to say.

**The shutdown steps call the provider's `openfeature.StateHandler` directly**, on the very
instance the scenario registered, and never through the SDK. Replacing the provider would test the
SDK's bookkeeping as much as the provider, which Appendix B already covers; and the SDK compares
providers with `reflect.DeepEqual` unless they are pointers, so re-registering the same one may be
judged no change and initialise nothing. Because the SDK is never told about the direct shutdown,
the client still routes to that instance and still holds it as `READY`, which is what lets the
evaluation after "the provider is initialized again" reach it. A provider that does not implement
`openfeature.StateHandler` has nothing to call, and the steps are no-ops — the `@lifecycle` gate
keeps such a provider out of those scenarios anyway. Each direct call is bounded by
`Config.ReadyTimeout`, so a shutdown that never returns fails its step rather than hanging `go test`.

**Providers are registered under a suite-scoped domain, not a per-scenario one.** Registering a
provider in a domain replaces and shuts down the previous one; a fresh domain per scenario would
leave every provider of the suite registered and running, which for a provider holding a network
connection means leaking one connection per scenario.

## The self-tests

Three suites run against providers from the SDK itself. They need no Docker and finish in
milliseconds, which makes them the fast canary: when a change breaks both these and a containerised
provider suite, these say so immediately and point at the TCK rather than at a provider.

| Suite | Subject | Why |
| --- | --- | --- |
| `TestInMemoryProvider` | `memprovider.InMemoryProvider` | reference adoption for a backend-less provider |
| `TestControllableProvider` | `tck.ControllableProvider` | the only suite that exercises the configuration-change path — see below |
| `TestMultiProvider` | `multi.Provider` wrapping one child | delegation must be transparent |

Only `TestControllableProvider` declares `@lifecycle`, because `tck.ControllableProvider` is the only
one of the three that implements `openfeature.StateHandler` and therefore the only one whose `READY`
the SDK did not manufacture. The other two leave it undeclared and report the readiness scenario as
skipped, which is what they had been passing vacuously under `@events` before the tag existed. It is
also, for the same reason, the only suite that exercises the shutdown steps without Docker.

None of the three declares `@numeric-coercion`. Every resolution decision in all three is made by
`memprovider.InMemoryProvider`, which type-asserts rather than coerces: it correctly refuses `0.5` as
an integer, but equally refuses `10.0` as an integer and `10` as a float, and the lossless scenarios
are what the capability requires beyond the lossy one. All three declare `@large-integers`, because
the `int64` a flag was seeded with is what comes back.

None of the three declares `@disabled-flags` either, and that one is a defect rather than an
absence — see the finding below. An in-memory provider is the architecture that *can* satisfy the
capability, because the caller's default never has to leave the process.

The flag set the three are seeded from is decoded from the specification's `canonical-flags.json`
rather than transcribed, and the decoding keeps the type each number was written with: `10` is an
`int64` flag and `10.0` a `float64` one. That distinction is what the lossless-coercion scenario
rests on, and it is the one `encoding/json` on its own would erase.

`TestMultiProvider` wraps exactly one child on purpose. That is the interesting configuration rather
than a degenerate one: the correct answer is precisely what `TestControllableProvider` already
asserts about the child alone, so any difference between the two suites is attributable to the
multi-provider and nothing else — a variant that does not survive the hop, a reason rewritten to
`DEFAULT`, an error code flattened to `GENERAL`, an event that never reaches the client.

## Findings

### The Go SDK's in-memory provider cannot update its flag set

[Appendix A][appendix-a] requires an SDK's in-memory provider to *"support a means of updating the
`flag set`, resulting in the emission of `PROVIDER_CONFIGURATION_CHANGED` events"*. The Go SDK's
`memprovider.InMemoryProvider` does not: it exposes no update method, does not implement
`openfeature.EventHandler`, and does not implement `openfeature.StateHandler`.

| SDK | update method | emits `PROVIDER_CONFIGURATION_CHANGED` |
| --- | --- | --- |
| JavaScript | `putConfiguration()` | yes |
| Java | `updateFlag()` | yes |
| **Go** | **none** | **no** |

The knock-on is not limited to this suite: Appendix A also requires SDK end-to-end tests to use the
in-memory provider, so the Go SDK's own [Appendix B][appendix-b] suite cannot cover
configuration-change events either.

`TestInMemoryProvider` therefore leaves `ConfigurationChange` undeclared and reports the scenario as
skipped with its reason. `tck.ControllableProvider` supplies the missing behaviour by **wrapping**
the SDK's provider rather than reimplementing it — every resolution decision is still made by
`memprovider` — so it doubles as a reference for what the SDK's provider should grow.

### The Go SDK's in-memory provider reports a disabled flag as an error

`memprovider.InMemoryProvider` returns the caller's default value for a flag whose `state` is
`DISABLED`, which is the half of the behaviour that matters most — but it attaches a `GENERAL`
resolution error to it while setting the reason to `DISABLED`
([`Resolve`](https://github.com/open-feature/go-sdk/blob/main/openfeature/memprovider/in_memory_provider.go)).
Those two contradict each other. [Requirement 2.2.5][req-225] lists `DISABLED` among the reason
strings a resolution that *worked* may carry, and an error code beside it tells the application
something went wrong when nothing did.

Measured, not inferred: with `@disabled-flags` declared, all four rows of the outline fail on `the
error-code should be ""` in all three self-test suites, and only on that step — the value assertion
passes, because the default really is what comes back.

So the three suites leave the capability undeclared and the scenarios are reported as skipped. This
is the odd one out among their omissions, because it is the *only* capability in the vocabulary an
in-memory provider is architecturally guaranteed to be able to satisfy: there is no backend to ask,
so the default is right there. `TestCanonicalFlagSetDisabledFlagsCarryAnError` pins the current
behaviour and fails when the SDK stops attaching the error, at which point the fix is to declare the
capability rather than to relax the assertion.

## Known gaps

- **Evaluation context passthrough is verified only for the targeting key.** `targeting-key-flag`
  resolves differently for a matching context, so a provider that drops the context is caught by the
  resolved value itself — that is what the three `@targeting` scenarios do, and no echo operation is
  needed for it. What is still unverified is that the *whole* context arrives intact: a provider
  that forwards the targeting key and silently discards every other attribute passes. Closing that
  needs either an echo operation on the control API or a second canonical flag whose rule keys on a
  custom attribute.
- **`POST /restart` is unused.** No current scenario needs a bounded outage — the stale scenario
  uses an explicit disconnect and reconnect — so `tck.ConnectionControl` has no `DisconnectFor`.
- **Hooks and flag metadata** are not covered. Provider metadata is, but only as far as a non-empty
  name.
- **Caching is not covered, and the suite is quietly exposed to it.** `@caching` is reserved and
  therefore not declarable, but flagd's RPC resolver enables an LRU cache *by default* and rewrites the
  reason to `CACHED` on a hit. The adoption does not turn it off, so the suite already runs against a
  caching provider while asserting `STATIC` everywhere. It passes only because no scenario evaluates
  the same flag twice in a way that hits the cache — so a scenario added later that does will fail
  against flagd RPC with `CACHED`, and the failure will look like a provider defect rather than a
  test-design one. Note also that the configuration-change scenario already depends on cache
  invalidation working without saying so: against flagd RPC it reads `changing-flag`, changes it, and
  reads again, which only gives the right answer because the change event evicts the entry.

[appendix-a]: https://github.com/open-feature/spec/blob/main/specification/appendix-a-included-utilities.md
[appendix-b]: https://github.com/open-feature/spec/blob/main/specification/appendix-b-gherkin-suites.md
[appendix-f]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md
[control-api]: https://github.com/open-feature/spec/blob/main/specification/assets/provider-tck/openapi/control-api.yaml
[numeric-coercion-adr]: https://github.com/open-feature/flagd/blob/main/docs/architecture-decisions/numeric-coercion.md
[reinit-fix]: https://github.com/open-feature/spec/commit/fc99d5ace4da472a5fea0595fa4db8034bbbc769
[req-147]: https://github.com/open-feature/spec/blob/main/specification/sections/01-flag-evaluation.md#requirement-147
[req-223]: https://github.com/open-feature/spec/blob/main/specification/sections/02-providers.md#requirement-223
[req-224]: https://github.com/open-feature/spec/blob/main/specification/sections/02-providers.md#requirement-224
[req-225]: https://github.com/open-feature/spec/blob/main/specification/sections/02-providers.md#requirement-225
[req-252]: https://github.com/open-feature/spec/blob/main/specification/sections/02-providers.md#requirement-252
[spec]: https://github.com/open-feature/spec
[tracking]: https://github.com/open-feature/spec/issues/417
