# OpenFeature Provider TCK (Go)

A conformance suite any OpenFeature Go provider can adopt to verify that it implements the provider
contract of the specification.

This module is the Go implementation of [Appendix F][appendix-f]. It runs the same Gherkin
scenarios, against the same canonical flag set, driven through the same backend control API, as
every other language's TCK — that shared basis is the point, because "conformant" only means
something if the question is identical everywhere. **What the suite asks, why each capability is
gated, and what a conformance claim is worth are settled in Appendix F**; this file documents the Go
binding of it.

**Status: proof of concept.** The scenario set is a representative subset covering each
architectural mechanism once, not exhaustive coverage. Breaking changes should be expected.
Tracking issue: [open-feature/spec#417][tracking].

## Quick start

One test function and one Docker Compose file, in a module of their own. The suite owns the whole
lifecycle — the stack, the discovered host ports, the control API, provider registration, the
readiness wait, the per-scenario reset and the teardown. **If you find yourself writing test
infrastructure, that is a defect here rather than something for you to work around.**

```go
//go:build tck

func TestMyProviderConformance(t *testing.T) {
	tck.Run(t,
		tck.WithName("my-provider"),
		tck.WithComposeFile("testdata/docker-compose.yaml"),
		tck.WithBackendPorts(8013),
		tck.WithProviderFromEndpoint(func(_ context.Context, e tck.BackendEndpoint) (openfeature.FeatureProvider, error) {
			return myprovider.New(e.Host(), e.Port(8013)), nil
		}),
		tck.WithUnavailableProvider(func(context.Context) (openfeature.FeatureProvider, error) {
			return myprovider.New("localhost", 9999), nil
		}),
		tck.WithCapabilities(tck.Events, tck.Lifecycle, tck.Object, tck.NumericCoercion, tck.LargeIntegers),
	)
}
```

Each scenario becomes a Go subtest, so `-run` selects one the usual way and a failure names a
scenario. The canonical feature files and flag set arrive as an ordinary Go dependency, so
**adopting this module needs no git submodule** — see [The spec module](#the-spec-module).
Configuration is functional options rather than a struct, and a missing or contradictory one is
reported by name before anything starts.

### Where it goes

**`providers/<name>/tck`, a module of its own, beside the provider's e2e suite rather than inside
it.** *Beside* is [Appendix F's reasoning][appendix-f-ci], and in Go the directory is additionally
what selects the suite, so nothing depends on what the tests are called. A *module* rather than a
plain directory, because a directory would fall under the provider's own module and drag
testcontainers, a Compose client and this package into the dependency graph of every application
that imports the provider. `go mod init github.com/you/.../providers/<name>/tck` and a `replace`
back to the provider is the whole of the setup; `make workspace-update` adds it to the workspace.

Give the suite file a **`//go:build tck`** constraint. That is not what selects it and not what
excludes it from `make e2e` — see [Running it](#running-it). Its one job is the untagged build:
without it, `make test` and a bare `go test ./...` in your module would start a Docker stack. Keep
the guard test and any package doc file untagged, so they still compile when the suite does not.

## Options

`tck.WithName` and a provider factory are always required; which factory, and whether a Compose file
is needed, depends on whether the provider has a backend.

| Option | Required | Default | Meaning |
| --- | --- | --- | --- |
| `tck.WithName(name)` | yes | — | the provider's **mode** — `flagd-rpc`, `flagd-in-process` — used in failure messages and reported as the configuration under test |
| `tck.WithProviderFromEndpoint(f)` | one of the two | — | provider factory for the Compose path; receives the discovered `tck.BackendEndpoint` |
| `tck.WithProvider(f)` | one of the two | — | provider factory for the manual path, used with `tck.WithControl` |
| `tck.WithComposeFile(path)` | with `…FromEndpoint` | — | the Compose file, resolved relative to the package directory |
| `tck.WithBackendPorts(ports...)` | with `…FromEndpoint` | — | container-internal ports the *provider* connects to; the control port is exposed automatically and must not be listed |
| `tck.WithBackendService(name)` | no | `backend` | the Compose service hosting both the control API and the backend |
| `tck.WithControlPort(port)` | no | `8080` | container-internal port of the control API |
| `tck.WithAdditionalPorts(service, ports...)` | no | none | extra service → ports, for stacks with more than one service |
| `tck.WithBackendConfiguration(name)` | no | `default` | the *backend's* named flag configuration, passed to `POST /start`; not the provider's mode |
| `tck.WithStartupTimeout(d)` | no | 60s | how long to wait for the stack and its control API |
| `tck.WithControl(c)` | manual path | — | a `tck.BackendControl` of your own — see [Controlling the backend](#controlling-the-backend) |
| `tck.WithUnavailableProvider(f)` | for `@unavailable` | — | a provider pointed at nothing, for the dead-backend scenarios |
| `tck.WithCapabilities(caps...)` | no | `tck.AllCapabilities()` | what this provider supports — see [Declaring capabilities](#declaring-capabilities) |
| `tck.WithKnownDeviations(devs...)` | no | none | required behaviour this provider does not have — see [Known deviations](#known-deviations) |
| `tck.WithEventTimeout(d)` | no | 12s | how long to wait for an expected provider event |
| `tck.WithReadyTimeout(d)` | no | 30s | how long to wait for `READY`; also bounds each direct `Init` and `Shutdown` |
| `tck.WithFeatures(fsys)` | no | none | extension feature files — see [Extending it](#extending-it) |
| `tck.WithSteps(register)` | no | none | extension step definitions |

`tck.BackendEndpoint` is what the factory receives: `Host()` and `Port(internal)`, plus
`ServiceHost`/`ServicePort` for a named service. `Port` panics on a container port that was never
declared, naming the missing option, rather than handing the provider a `0` that surfaces later as
what looks like a provider defect.

`tck.WithEventTimeout` is the knob that most often needs changing: a streaming provider sees a
configuration change in milliseconds, one polling every 30 seconds may need most of a poll interval,
and too low a value reports impatience as a timeout.

**The Compose file must not pin host ports** — Docker assigns them dynamically and the suite
discovers them after startup. The stack starts once per suite and is never restarted, and the suite
never sleeps after a control call; both are [Appendix F's control-API invariants][appendix-f-control]
rather than local policy, so a scenario that is flaky immediately after a control call is a defect in
the backend's control API. Mixing the Compose and manual paths is refused rather than silently
resolved.

## Declaring capabilities

Each scenario exercising an optional part of the contract carries a Gherkin tag, and a provider
declares what it supports. **A scenario whose capability was not declared is reported as skipped
with the reason, never as passed.** The vocabulary, what each tag means, and the six rules for
deciding whether to declare one are [Appendix F's][appendix-f-caps]. This is the Go spelling:

| Capability | Tag | Capability | Tag |
| --- | --- | --- | --- |
| `tck.Events` | `@events` | `tck.NumericCoercion` | `@numeric-coercion` |
| `tck.Lifecycle` | `@lifecycle` | `tck.LargeIntegers` | `@large-integers` |
| `tck.Stale` | `@stale` | `tck.Reinitialization` | `@reinitialization` |
| `tck.ConfigurationChange` | `@configuration-change` | `tck.Targeting` | `@targeting` |
| `tck.Object` | `@object` | `tck.StandardReasons` | `@standard-reasons` |
| `tck.Variants` | `@variants` | `tck.DisabledFlags` | `@disabled-flags` |
| `tck.UnavailableInit` | `@unavailable` | `tck.StringTyping` | `@string-typing` |
| `tck.Caching` | `@caching` — reserved, **not declarable** | | |

Omitting `tck.WithCapabilities` declares `tck.AllCapabilities()`, which excludes the reserved tags.
Narrow it rather than widening it: start from the default, run the suite, and remove only what your
provider genuinely cannot do — a judgement Appendix F makes **per scenario, not per tag**. Passing
the option with no capability at all is a declaration too, and says this provider supports none of
the optional parts.

Two of these are gated by an argument rather than by a gap in the specification, and are the two
most likely to be withheld by a provider that is doing nothing wrong. `@numeric-coercion` borrows
flagd's coercion ADR, which the specification does not define. `@string-typing` asks for
`TYPE_MISMATCH` when a non-string flag is requested as a string, and the only normative statement
nearby is Requirement 1.3.4 — a `SHOULD` on the *client*. A backend that stores flag values as text
satisfies the string accessor for every flag and has no mismatch to report, so it withholds the tag
and stays conformant; `providers/flagsmith` is that case in this repository, and `providers/flagd`
and `providers/ofrep` are the other one.

Appendix F has the implementation, rather than each adopter, refuse two kinds of capability, with
distinct skip reasons: a **reserved** one (`@caching` today, which no scenario carries anywhere) and
an **inexpressible** one, which the language's SDK cannot ask about at all.

**Go has no inexpressible capability, and that is measured rather than assumed.**
`Client.IntValueDetails` takes and returns an `int64` and `Client.FloatValueDetails` a `float64` —
two accessors over two types, which is why the coercion scenarios mean anything here and where
`@large-integers` gets its width. `TestTheIntegerAccessorIsWideEnoughToAskForALargeInteger` and
`TestTheIntegerAndFloatAccessorsAreDistinctTypes` pin both against the SDK's own method signatures,
so a narrowing in a future SDK fails there rather than as an apparent provider defect.
`inexpressibleCapabilities` in `capability.go` is therefore empty, and the refusal it drives is kept
exercised by tests that install an entry and drive every path it feeds.

Two tripwires guard the pinned assets, because both failure modes are silent. **A reserved tag on a
real scenario fails the run**, naming the one line to delete (`reservedCapabilities` in
`capability.go`), so a newly added `@caching` scenario cannot be skipped for a capability nobody may
declare. And `TestEveryCanonicalFeatureFileIsCollected` spells the collected set out rather than
deriving it from the `//go:embed` pattern, so a feature file renamed or moved fails here instead of
shrinking the suite quietly — that is how `reason.feature` arrived, taking the count from 56
scenarios to 65.

### Known deviations

Narrowing `tck.WithCapabilities` says a scenario was not run. It cannot say **why**, and from the
outside a capability withheld by choice and one withheld because it is broken are the same absence.
`tck.WithKnownDeviations` is where that gets said:

```go
tck.WithKnownDeviations(
	tck.TrackedDeviation(
		tck.NumericCoercion,
		"https://github.com/open-feature/flagd/issues/1996",
		"The lossy half of the coercion rule is not enforced: float-flag (0.5) through the "+
			"integer API returns 0 with no error code, rather than TYPE_MISMATCH with the "+
			"code default.",
	),
	tck.UntrackedDeviation(tck.Stale, "…"),
)
```

**What an entry means, which of its two shapes to reach for, and why a failed scenario is not yet a
deviation are [Appendix F's "Rules for declaring"][appendix-f-deviations]** — read them before
writing one. The short version, because it is where adopters go wrong: an entry asserts the provider
fails to do something it is *required* to do, so find the numbered requirement first; prefer
declaring the capability and letting the scenario fail over withholding the tag and explaining the
skip; and `tck.UntrackedDeviation`, for a gap with no issue yet, is still worth declaring.

Enforced here rather than only documented: a deviation with no summary is rejected, so is one naming
a reserved capability, and so is one naming a capability the SDK cannot express — in both of those
cases the entry would explain a skip that says nothing about your provider. The list is empty by
default, which is silence rather than a claim.

## Adopters

| Provider | Suite | Control path |
| --- | --- | --- |
| flagd (RPC and in-process resolvers) | [`providers/flagd/e2e/tck_test.go`](../../providers/flagd/e2e/tck_test.go) | `tck.HTTPControl` against the `flagd-testbed` launchpad |

## Controlling the backend

`tck.BackendControl` is the single seam between the scenarios and whatever manipulates the backend,
which is why the same Gherkin runs unchanged against a containerised backend and against a provider
manipulated in-process.

**If your provider talks to a backend, drive it over the HTTP control API** in
[`control-api.yaml`][control-api], which this module exposes as bytes through `tck.ControlAPISpec()`.
You do not have to write the client: `tck.WithComposeFile` builds a `tck.HTTPControl` for you, and it
is also constructible directly with `tck.NewHTTPControl(tck.HTTPControlOptions{BaseURL: …})`.
`BaseURL` is the only required field and must be built from the **dynamically mapped** host port
discovered after the stack is up. `BackendConfiguration` defaults to
`tck.DefaultBackendConfiguration`, the one name Appendix F requires every backend under test to
serve.

A control must also state which path it took — `tck.ControlAPIHTTP` for the normative HTTP control
API, `tck.ControlAPIInProcess` for the narrow allowance below. Appendix F requires the control to
say so rather than the harness to infer it, there is no default, and both controls shipped here
answer already, so only an author writing a control of their own writes anything:

```go
func (c *myControl) ControlAPI() tck.ControlAPI { return tck.ControlAPIHTTP }
```

### Providers with no backend

An in-memory, environment-variable or file-based provider has nothing to connect to.
`tck.InProcessControl` is the reference for [Appendix F's in-process allowance][appendix-f-nobackend]:
no Compose file, the control supplied directly, and the provider built without an endpoint.

```go
control := tck.NewInProcessControl()

tck.Run(t,
	tck.WithName("in-memory"),
	tck.WithControl(control),
	tck.WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
		return control.NewProvider(), nil
	}),
	tck.WithCapabilities(tck.Events, tck.ConfigurationChange, tck.Object, tck.LargeIntegers),
)
```

**A provider with an external backend must use the control API.** A backend-less control simply does
not implement `tck.ConnectionControl`, so `Stale` and `UnavailableInit` are left undeclared and their
scenarios skip; declaring them anyway fails loudly rather than passing silently.

## Running it

```console
make tck                                 # from the repository root
go test -tags=tck -timeout=20m ./...     # equivalently, from inside a conformance module
```

`make tck` is `go test -count=1 -timeout=20m -tags=tck ./...` over the conformance modules — the
modules named `tck` under a component directory. `make e2e` is the same sweep over the other modules
under `-tags=e2e`, and then those same conformance modules again under `-tags=tck` with an empty
`-run` pattern, which builds them and runs nothing. The two targets are the two halves of one
partition of the module list, and **the directory is the whole of the selector**.

**A red conformance run is not a broken build**, and why an adoption gets a step of its own rather
than a slice of an existing e2e suite is [Appendix F's "Running the suite in CI"][appendix-f-ci].
What is Go's is that two mechanisms do two jobs, and neither substitutes for the other:

| mechanism | job | what breaks without it |
| --- | --- | --- |
| the module path `providers/<name>/tck` | which of `make e2e` and `make tck` runs a suite | `make e2e` runs the conformance suites, and a red e2e build means two different things |
| `//go:build tck` on the suite file | keeps the suite out of any **untagged** build | `make test`, and any bare `go test ./...`, start a Docker stack |

The corollary is counter-intuitive enough to state: **a tag is not an exclusion in this repository**,
and both adoptions were in fact running, red, on every pull request while carrying one. The usual
objection to a build tag — that it takes the suite out of compilation, which Appendix F asks against
— is answered by `make e2e`'s second command building it. Each adoption also carries an untagged test
that fails if nothing in the module reaches `tck.Run`, because a conformance module whose tests have
stopped running the suite makes `make tck` green by running nothing.

The **default build** runs this module's own tests and nothing else: the three self-tests below, the
unit tests over configuration validation, capability gating and extension mounting, and
`httpcontrol_internal_test.go`, which pins the three rules in `control-api.yaml` that constrain the
*sequence* of control calls rather than any single call. None needs Docker; all finish in about a
second.

## Extending it

A provider often has behaviour the specification does not describe and cannot — flagd's `fractional`
targeting, a vendor's segment rules. Those scenarios still need a provider registered per scenario, a
backend reset between them and the event plumbing this suite already owns, so they run *in* the suite
rather than beside it:

```go
tck.Run(t,
	// ... name, provider factory and capabilities as above ...
	tck.WithFeatures(os.DirFS("testdata/tck-extensions")),
	tck.WithSteps(func(ctx *godog.ScenarioContext) {
		ctx.Step(`^the fractional bucket for "([^"]*)" is "([^"]*)"$`, theBucketIs)
	}),
)
```

`tck.WithFeatures` takes any `fs.FS`; every `.feature` file in it is picked up, and a filesystem
holding none is refused rather than quietly running the canonical suite alone. `tck.WithSteps` is
called after the TCK's own step definitions, so an extension step sees the same scenario context and
hooks. It reaches the provider under test through **`tck.ClientFromContext(ctx)`** — the provider is
registered under a suite-scoped domain the adopter never names, and a step that built a client of its
own would be testing a different provider.

Extension features mount under an `extensions/` prefix while the suite runs and the canonical ones
keep their `gherkin/` paths; [Appendix F prescribes that partition][appendix-f-extending] and the
rules that come with it. Both options are optional.

## Go-specific notes

### The spec module

The Gherkin, the canonical flag set and the control API arrive as an ordinary dependency on [the
spec's assets module][assets], which documents itself: why Go consumes it as a module where the other
three languages use a submodule, what a version bump means for a suite that can go red without
anything changing on your side, the shape of its release tag, and — worth reading before you seed a
backend — the properties of the canonical flag set that a seeding step is most likely to break.

Nothing is copied and nothing is generated here: `assets.go` reads the embedded files straight out of
the dependency, so the revision this suite conforms to is the version in `go.mod` and nothing else,
and an adopter needs no submodule of their own.

What follows from that is this repository's rather than the assets module's: **the pin cannot
silently go stale**, which [Appendix F names as a hazard three of four implementations had][appendix-f-impl]
and one of them hit. There is no working tree of the Gherkin to update, and the pinned version
resolves from a read-only module cache verified against `go.sum` before a scenario is collected.

```console
cd tools/tck
go get github.com/open-feature/spec/specification/assets/provider-tck@<commit-or-tag>
```

Until the spec publishes a release, the pin is a pseudo-version naming the exact commit.

### Three places the shared Gherkin needed a decision

Each is documented where it is implemented; the short form, because a reader of a failure message
wants it:

- **"no exception should have been thrown" means nothing panicked.** Go has no exceptions, and the
  `error` an evaluation returns is not one — an errored evaluation correctly returns the code default
  *alongside* a non-nil error. The one returned error that does count is `Init`'s, which is what the
  other languages' `initialize()` throws to say the provider never reverted to its uninitialised
  state.
- **The shutdown steps call `openfeature.StateHandler` directly**, on the instance the scenario
  registered, never through the SDK: replacing the provider would test the SDK's bookkeeping, and the
  SDK compares providers with `reflect.DeepEqual` unless they are pointers, so re-registering the
  same one may be judged no change and initialise nothing. Each direct call is bounded by
  `tck.WithReadyTimeout` (`state.go`, `provider.go`).
- **Providers are registered under a suite-scoped domain, not a per-scenario one.** Registering in a
  domain replaces and shuts down the previous provider, so a fresh domain per scenario would leak a
  connection per scenario for any provider holding one.

### One module, container harness included

The Compose harness is in package `tck` rather than in a `tools/tck/compose` beside it, so an adopter
has one import path and one version to track. The cost is worth naming: `testcontainers-go` and
`docker/compose` are ordinary dependencies of the package, so a provider with no container to start —
in-memory, environment-variable, file-based — still takes those `go.sum` entries and the ~40
transitive pins behind them. It compiles nothing it does not import, and this is a test-only module
that no application binary links, so the cost is confined to `go test` of an adopting module.

The split was weighed and declined: a second module needs its own version and release-please entry,
and a home for `BackendEndpoint` that both modules can see — which is this package, so the second
module would import the first and buy nothing but a second coordinate to publish.

## The self-tests

Three suites run against providers from the SDK itself. They need no Docker and finish in
milliseconds, which makes them the fast canary: when a change breaks both these and a containerised
provider suite, these point at the TCK rather than at a provider.

| Suite | Subject | Why |
| --- | --- | --- |
| `TestInMemoryProvider` | `memprovider.InMemoryProvider` | reference adoption for a backend-less provider |
| `TestControllableProvider` | `tck.ControllableProvider` | the only one exercising the configuration-change path, and the only one implementing `openfeature.StateHandler` |
| `TestMultiProvider` | `multi.Provider` wrapping **one** child | delegation must be transparent, and one child makes any difference from the suite above attributable to the multi-provider alone |

What they leave undeclared follows from `memprovider` rather than from the suite. Only
`TestControllableProvider` declares `@lifecycle`, because it is the only one whose `READY` the SDK
did not manufacture; none declares `@numeric-coercion`, because `memprovider` type-asserts rather
than converts and so refuses `10.0` as an integer as readily as `0.5`. All three declare
`@string-typing`, which is that same type assertion paying off rather than costing: a boolean, a
number or a structure asked for through the string accessor is refused rather than formatted.
Two omissions are defects
rather than absences — `@configuration-change` ([go-sdk#530][gosdk-530]: no update method, which
[Appendix A][appendix-a] requires) and `@disabled-flags` ([go-sdk#552][gosdk-552], fixed by
go-sdk#574 but unreleased: a disabled flag comes back with the caller's default *and* a `GENERAL`
error) — and each is pinned by a test of its own, which is the condition [Appendix F's self-test
carve-out][appendix-f-deviations] attaches to a TCK's own suites withholding a capability to stay
green. An adoption has no such licence.

The flag set they are seeded from is decoded from the specification's `canonical-flags.json` rather
than transcribed, keeping the type each number was written with — `10` an `int64` flag and `10.0` a
`float64` one, the distinction `encoding/json` would erase and the lossless-coercion scenario rests
on.

## Known gaps

The suite's gaps affect every language equally and are listed in [Appendix F's open
questions][appendix-f-gaps] — evaluation-context passthrough beyond the targeting key, per-flag
control operations, `@stale` never exercised without containers, caching, hooks and flag metadata,
and the unfinished coverage of the numbered requirements.

One consequence is this repository's: `providers/flagd/tck` does not turn flagd's RPC LRU cache off,
so the suite really does run against a caching provider while asserting `STATIC` everywhere, and
passes only because the canonical set respects the appendix's constraint that no scenario evaluates
the same flag twice without a configuration change between. The configuration-change scenario
already depends on cache invalidation working, without saying so.

[appendix-a]: https://github.com/open-feature/spec/blob/main/specification/appendix-a-included-utilities.md
[assets]: https://github.com/open-feature/spec/blob/main/specification/assets/provider-tck/README.md
[appendix-f]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md
[appendix-f-caps]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#capabilities-how-a-provider-says-what-it-cannot-do
[appendix-f-ci]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#running-the-suite-in-ci
[appendix-f-control]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#the-control-api
[appendix-f-deviations]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#rules-for-declaring
[appendix-f-extending]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#extending-the-suite
[appendix-f-gaps]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#open-questions
[appendix-f-impl]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#implementing-the-suite-in-a-language
[appendix-f-nobackend]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#providers-with-no-backend
[control-api]: https://github.com/open-feature/spec/blob/main/specification/assets/provider-tck/openapi/control-api.yaml
[gosdk-530]: https://github.com/open-feature/go-sdk/issues/530
[gosdk-552]: https://github.com/open-feature/go-sdk/issues/552
[tracking]: https://github.com/open-feature/spec/issues/417
