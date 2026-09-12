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

One test function and one Docker Compose file. The TCK owns the whole lifecycle: it starts the stack,
discovers its dynamically mapped host ports, drives the backend's control API, registers each
provider with the OpenFeature API under a suite-scoped domain, waits for it to become ready, awaits
events, resets the backend between scenarios, releases the provider at the end and tears the stack
down. **If you find yourself writing test infrastructure, that is a defect here rather than
something for you to work around.**

```go
func TestMyProviderConformance(t *testing.T) {
	tck.Run(t,
		tck.WithName("my-provider"),
		tck.WithComposeFile("testdata/tck/docker-compose.yaml"),
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

Configuration is functional options rather than a struct, so the suite can gain a capability without
every adoption having to be edited, and so a required option is named in one place rather than being
a zero value someone has to remember means "unset". A missing one is reported by name before
anything starts.

Two options are always required — `tck.WithName` and a provider factory — and one more depends on
how the backend is run. The factory is a factory rather than an instance because each scenario gets
its own provider, and because a provider cannot be configured before the stack is up: its host ports
do not exist until then.

Each scenario becomes a Go subtest, so `-run` selects one the usual way and failures name a
scenario.

The canonical feature files and flag set arrive as an ordinary dependency, so **adopting this
module needs no git submodule** — `go get` it and everything the suite runs comes with it. Where
they come from and how the pin moves is described under [The spec module](#the-spec-module).

### The Compose contract

An adopter names a Compose file, says which service and ports to expose, and supplies a factory that
builds a provider from a discovered endpoint. The suite does the rest.

| option | required | default | meaning |
| --- | --- | --- | --- |
| `tck.WithComposeFile(path)` | yes | — | the Compose file, resolved relative to the package directory |
| `tck.WithBackendService(name)` | no | `backend` | the Compose service hosting both the control API and the backend |
| `tck.WithBackendPorts(ports...)` | yes | — | container-internal ports the *provider* connects to. The control port is exposed automatically and must not be listed here |
| `tck.WithControlPort(port)` | no | `8080` | container-internal port of the control API |
| `tck.WithAdditionalPorts(service, ports...)` | no | none | extra service → ports, for stacks with more than one service. Resolved through the endpoint by service name |
| `tck.WithBackendConfiguration(name)` | no | `default` | the *backend's* named flag configuration, passed to `POST /start`. Not the provider's configuration — the report's `configuration` field is the provider's mode and comes from `tck.WithName` |
| `tck.WithStartupTimeout(d)` | no | 60s | how long to wait for the stack and its control API to become reachable |
| `tck.BackendEndpoint` | — | — | what the factory receives: `Host()` and `Port(internal)`, plus `ServiceHost`/`ServicePort` for a named service |

Three rules are not negotiable, because they are the reasons the design is shaped this way.

**The stack must not pin host ports.** Docker assigns them dynamically and the suite discovers them
after startup; a pinned host port makes the suite unrunnable in parallel and collides with whatever
the developer already has listening.

**The stack starts once per suite and is never restarted.** Testcontainers cannot reliably preserve
dynamically mapped host ports across a restart, so a restart would silently invalidate every
provider already pointed at the old port, and the failure would look like a flaky provider. Backend
unavailability is *always* simulated inside the running stack through the control API.

**The control API is the normative contract.** Another language's suite drives the same endpoints
against the same stack and must get the same answers, so do not substitute an in-process control
that reaches an external backend through a side channel — see
[Controlling the backend](#controlling-the-backend).

The suite waits for the control API once, before the first scenario, by probing `GET /healthz`
bounded by the startup timeout; a `404` counts as ready, because that path is optional and the TCP
port wait the stack already passed is the documented fallback. There is **no settle after a control
call**: the control API's promise is that a command has taken effect when it returns, and a suite
that sleeps instead of holding it to that promise stops being able to detect when it breaks. If a
scenario is flaky immediately after a control call, that is a defect in the backend's control API
and worth an issue there rather than a sleep here.

Compose does not replace the manual path. A provider with no backend keeps supplying its own
`tck.WithControl` and building its provider with `tck.WithProvider` — see
[Providers with no backend](#providers-with-no-backend). Mixing the two paths is refused rather than
silently resolved, because either half of the mix would leave a provider pointed at a stack whose
control the suite is not driving.

### Timings

`tck.WithEventTimeout` is the knob that matters. Providers observe backend changes on wildly
different timescales — a streaming provider sees a configuration change in milliseconds, one that
polls every 30 seconds may need most of a poll interval. Set it to comfortably exceed your
worst-case detection latency, or the suite reports timeouts that are really just impatience.

### Adding your own scenarios

A provider often has behaviour the specification does not describe and cannot — flagd's `fractional`
targeting, a vendor's segment rules. Those scenarios still need a provider registered per scenario,
a backend reset between them and the event plumbing this suite already owns, so they run *in* the
suite rather than beside it. Two optional options:

```go
tck.Run(t,
	// ... name, provider factory, control and capabilities as above ...
	tck.WithFeatures(os.DirFS("testdata/tck-extensions")),
	tck.WithSteps(func(ctx *godog.ScenarioContext) {
		ctx.Step(`^the fractional bucket for "([^"]*)" is "([^"]*)"$`, theBucketIs)
	}),
)
```

`tck.WithFeatures` takes any `fs.FS`; every `.feature` file in it is picked up. `tck.WithSteps` is
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

**Both are optional, and a configuration without them runs exactly what it ran before they
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

Untagged scenarios are mandatory and always run. Omitting `tck.WithCapabilities` declares
`tck.AllCapabilities()` — narrow it rather than widening it: start from the default, run the suite,
and remove only what your provider genuinely cannot do. Passing it with no capability at all is a
declaration too, and says this provider supports none of the optional parts.

A **reserved** capability is named by the vocabulary so there is a place for it once scenarios
exist, but it **must not be declared**. No scenario carries the tag, so declaring it cannot be
verified, cannot even produce a skip, and tells a reader of a report only that something was claimed
and nothing examined. `tck.AllCapabilities()` therefore excludes the reserved capabilities, and
naming one in `tck.WithCapabilities` is rejected by configuration validation rather than passed
into a report — an unverifiable claim is a configuration mistake, not a conformance result.
`@caching` is the only reserved tag left: `@targeting` became declarable, with three scenarios, in
spec `26362f85`.

That is easy to reintroduce by accident rather than by intent: an adoption that declares everything
and then removes what it cannot do collects every reserved tag on the way past, which is how a Java
conformance report came to assert `@targeting` and `@caching` as declared — back when both were
reserved.

**The reservation expires by itself, and the suite fails when it should have.** The day the
specification adds the first scenario carrying `@caching`, every adoption would otherwise report
that scenario as skipped for a capability no adopter is permitted to declare: a green suite, a
well-formed report, and a question put and silently withdrawn. Nothing else here would notice —
the scenario *was* collected, so the under-collection guard is satisfied, and a capability-gated
skip is explicitly not a gap. So a reserved tag on a real scenario **fails the run**, with a message
naming the one line to delete (`reservedCapabilities` in `capability.go`).
`TestTheCanonicalScenariosCarryNoReservedTag` is the same check against the pinned assets, so moving
the pin trips it here rather than in an adopter's run.

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

Narrowing `tck.WithCapabilities` says a scenario was not run. It cannot say **why**, and the two
reasons are not alike: a provider with no streaming transport declining `@configuration-change` has made a
decision, while one declining `@numeric-coercion` because it narrows `0.5` to `0` with no error code
has a bug. In the results they are indistinguishable — the same skip, carrying the same reason — so
unless the provider author says which happened, a consumer comparing providers reads a defect as a
design choice. The TCK cannot infer it: from the outside, a capability withheld by choice and one
withheld because it is broken are the same absence.

`tck.WithKnownDeviations` is where that gets said.

```go
tck.WithKnownDeviations(
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
)
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

The rest of this section is the normative wording from
[Appendix F's "Rules for declaring"][appendix-f-deviations], not a Go restatement of it: the field
means the same thing in all four languages, and the appendix is where it is settled. It is repeated
here rather than only linked because it is what an adopter needs at the moment they are writing one.
If the two ever disagree, the appendix wins and this file is wrong.

**An entry says one thing: this provider fails to do something it is required to do.** The
requirement has to be a numbered `MUST`, or a rule the implementation bound itself to elsewhere —
flagd measured against its own accepted numeric-coercion ADR is the worked example. Where the
specification permits the choice, withholding the capability *is* the honest report, and an entry
would assert a defect that does not exist.

**Two shapes are legitimate, and a report's results already tell them apart.**

1. **The capability is declared, the scenario runs, and it fails.** Prefer this. The failure stays
   visible and the deviation says it is known and why.
2. **The capability is withheld, and its scenarios skip.** Legitimate only when the provider cannot
   attempt the behaviour at all, so running the scenario would establish nothing. The deviation then
   explains the absence, so a reader can tell a defect from a design decision.

Withdrawing a capability *in order to* turn a failing scenario into a skip is the failure mode this
field exists to prevent. If the provider attempts the behaviour and gets it wrong, shape 1 is the
honest report.

An entry with **no capability** is also legitimate: the gap is against a mandatory, ungated
scenario, which belongs to no capability and so can name none. Shape 1 — an entry naming a
capability that *is* declared — is likewise how to record that the capability holds while one of the
scenarios it gates does not, which is worth saying precisely because such a scenario may pass for
the wrong reason in one of a provider's modes and hide the gap there.

`summary` is required and `issue` is optional.

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
discovered after the stack is up — a stack under test must not pin host ports.
`BackendConfiguration` defaults to `tck.DefaultBackendConfiguration`, the one name Appendix F
requires every backend under test to serve, and the one that serves the canonical flag set. It is
the backend's config file and not the provider's mode; the report's `configuration` field is the
latter and comes from `tck.WithName`.

If you find yourself writing a control client of your own, that is a defect here rather than
something for you to work around.

Three of its requirements are easy to get wrong:

- **Containers are never stopped or restarted mid-suite.** Unavailability is simulated *inside* the
  running stack — a process kill, a proxy toxic, a socket block. This is portability, not
  preference: container orchestrators assign host ports dynamically and cannot reliably preserve
  them across a restart, so restarting silently invalidates every provider already pointed at the
  old port, and the failure looks like a flaky provider.
- **A state-changing call has taken effect when it returns.** `POST /start`, `/change` and `/reset`
  must not return until the new state is actually being served. That is the *backend's* promise: how
  long the provider under test takes to notice is a property of its transport, and that is what
  `tck.WithEventTimeout` covers. The suite never sleeps after a control call, so a backend that
  returns early makes the provider's detection latency unmeasurable — and turns a control-API
  defect into what looks like a flaky provider.
- **`/start` resets flag state; `/restart` preserves it.** An outage must be observable as a change
  in availability, never as a change in flag values. `/restart` is optional and no shipped scenario
  reaches it — see [Known gaps](#known-gaps).

### A control says which path it took

`tck.BackendControl` requires one more thing of a control than the operations above:

```go
func (c *myControl) ControlAPI() tck.ControlAPI { return tck.ControlAPIHTTP }
```

`tck.ControlAPIHTTP` means the normative HTTP control API; `tck.ControlAPIInProcess` means the narrow
allowance below for a provider with no backend. There is no default and nothing is inferred from the
control's concrete type. The same scenarios passing over the control API and passing through
in-process manipulation of a provider that *does* have a backend are not the same claim, and this is
the only field in the conformance report that separates them — so an absent value would not be "no
claim made" but an unfalsifiable one.

Both controls shipped here answer it already, so an adopter using `tck.WithComposeFile` or
`tck.InProcessControl` writes nothing. The only author who has to state it is the one writing a
control of their own, which is exactly the case where it cannot be guessed.

### Providers with no backend

An in-memory, environment-variable or file-based provider has nothing to connect to and no control
API to expose. Those may control the backend in-process, where flag operations are direct
manipulations of the provider's own state. `tck.InProcessControl` is the reference, and the adoption
is the manual path — no Compose file, the control supplied directly, and the provider built without
an endpoint:

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
nothing is generated: `assets.go` reads the embedded files out of the dependency, so the
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
cd tools/tck
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
`tck.WithReadyTimeout`, so a shutdown that never returns fails its step rather than hanging `go test`.

**Providers are registered under a suite-scoped domain, not a per-scenario one.** Registering a
provider in a domain replaces and shuts down the previous one; a fresh domain per scenario would
leave every provider of the suite registered and running, which for a provider holding a network
connection means leaking one connection per scenario.

And one packaging decision, recorded because its cost is visible in a `go.sum`:

**The Compose harness lives in this module rather than in a second one.** An adopter therefore has
one import path and one version to track, and testcontainers-go and `docker/compose` are ordinary
dependencies of package `tck`. The cost is that a provider with no container to start — in-memory,
environment-variable, file-based — still takes those `go.sum` entries and the ~40 transitive pins
behind them. It compiles nothing it does not import and no application binary links this module, so
the cost is confined to `go test` of an adopting module. A `tools/tck/compose` module was weighed
and declined: it would need its own version and release-please entry, and a home for
`tck.BackendEndpoint` that both modules can see — which is this package, so the second module would
import the first and the split would buy nothing but a second coordinate to publish. After the
Compose decision, containerised adopters are the overwhelming majority, and Java keeps
testcontainers inside its `tck` artifact for the same reason.

## Conformance reports

Set `PROVIDER_TCK_REPORT_DIR` and each suite writes two files: an envelope at `<dir>/<name>.json`,
conforming to the [report schema][report-schema] in the specification, and the results it references
at `<dir>/<name>.ndjson`, which is a [Cucumber Messages][cucumber-messages] stream.

```console
$ PROVIDER_TCK_REPORT_DIR=./reports go test ./...
$ jq . reports/in-memory.json
{
  "schemaVersion": "1",
  "provider": { "name": "InMemoryProvider", "language": "go", "configuration": "in-memory" },
  "sdk": { "name": "github.com/open-feature/go-sdk", "version": "v1.18.0" },
  "tck": {
    "implementation": "go-sdk-contrib/tools/tck",
    "version": "v0.1.0",
    "specRevision": "v0.0.0-20260912135310-93eb1a58d2d2"
  },
  "backend": {
    "description": "the Go SDK's memprovider.InMemoryProvider, rebuilt per scenario",
    "controlApi": "in-process"
  },
  "declaration": { "declared": ["@events", "@large-integers", "@object", "@variants"] },
  "results": {
    "format": "cucumber-messages",
    "formatVersion": "21.0.1",
    "location": "in-memory.ndjson",
    "digest": "sha256:4754d458ac5a1a137080082b7947f8f1eafc5df9f6d54551440da9f8ac6a0dce"
  }
}
```

The envelope says what was tested and what the provider claims. It does not contain the results: a
Messages stream carries the feature sources and so is far larger than the envelope, and a consumer
deciding whether it cares about a report should not have to fetch a whole run to find out.
`results.digest` covers the payload byte for byte, so a consumer can tell that what it fetched is
what the envelope describes.

Two fields next to each other mean different things and are worth reading carefully.
`provider.configuration` is the provider's own **mode** — which of several materially different
configurations of one provider was tested, `flagd-rpc` against `flagd-in-process` — and it comes
from `tck.WithName`. `backend.controlApi` is how the suite drove the backend: `http` for the
normative control API, `in-process` for the narrow allowance made for a provider with no backend.
The backend block is always present and `controlApi` is always set, because `tck.BackendControl`
requires the control to state it. Nothing infers it, and an absent value would not be "no claim
made" but an unfalsifiable one — the same scenarios passing over the control API and passing through
in-process manipulation of a provider that *does* have a backend are not the same claim, and this is
the only field that separates them. The backend's own named flag configuration, if you set one, is
`tck.WithBackendConfiguration` and does not appear in the report at all.

`PROVIDER_TCK_REPORT_DIR` is an environment variable rather than a `Config` field so that emitting a
report is a property of the run and not of the code: CI sets it, a developer running the suite
locally does not, and no adopter changes a line to publish one. Unset means no report, which is not
an error. Several suites in one test binary each write their own pair of files, so flagd's two
resolvers do not collide.

### The canonical set has to have run

A report is a claim that the provider was asked the canonical questions, and nothing in the format
establishes that it was asked all of them. `go test -run` matching one scenario name produces a
green suite and a well-formed report describing a single scenario; so does a mis-wired extension
filesystem. There is no field in the envelope a consumer could read to notice.

So the suite checks itself. After the run it compares what executed against the scenarios the
embedded assets compile to — parsed by godog's own parser, so the expectation is exactly what a full
run would have produced — and fails the test if any of them produced no outcome:

```
tck [in-memory]: 27 canonical scenario(s) did not run, so this is not a conformance run
and its report must not be published:
  - gherkin/errors.feature: Requesting the wrong type returns the code default: 0 of 11
    executed (11 announced but never run, which is what a -run selector or a tag filter leaves behind)
```

A capability-gated scenario is not a gap: it ran the gate and is in the results as `SKIPPED` with
its reason, so the question was put and declined. What this catches is the question that was never
put. Extension scenarios are counted and reported but can never close a gap — an adopter's feature
is an addition to the canonical set, not a substitute for part of it.

### Why the results are Cucumber Messages

Because the alternative was a second format to maintain. The per-scenario outcome, the tags, the
executed feature source and the identity of a Scenario Outline row are all already specified by
Messages, which is maintained, cross-language, schema'd, and emitted natively by cucumber-jvm.
Restating them in the report schema would have meant versioning them and giving the same fact two
places to disagree.

Reading the results needs no bespoke tooling. Every scenario is a `pickle`; every result is a
`testCase` with a `testCaseStarted` and one `testStepFinished` per step; a scenario's outcome is the
most severe of its step results, which is how Cucumber itself derives it.

```console
$ jq -c 'select(.testStepFinished) | .testStepFinished.testStepResult.status' reports/in-memory.ndjson \
    | sort | uniq -c
    270 "PASSED"
    125 "SKIPPED"
```

### Why this exists in Go before the other languages

Because Go is the language that needs it most. godog counts a capability-gated skip in its **passed**
tally:

```
56 scenarios (56 passed)
```

Eighteen of those fifty-six did not run. Appendix F is unambiguous that a scenario skipped for an
undeclared capability is reported as skipped with the reason and *never* as passed, and the harness
does say so in a separate log line — but the headline number still says something false, and a
number is what gets read. pytest and jest-cucumber both report skips correctly, so this is a property
of the runner rather than of the suite's design.

The report does not fix godog's summary. It makes the summary stop mattering. The stream above
accounts for all fifty-six scenarios and reports eighteen of them as `SKIPPED`, each carrying the
capability that gated it in `testStepResult.message`, so a consumer can check the rule instead of
trusting the runner to have applied it.

godog 0.15.1 has no Messages formatter — it registers `cucumber` (the legacy relishapp JSON),
`events`, `junit`, `pretty` and `progress` — so [`messages.go`](./messages.go) is one,
registered through the public `godog.Format` plugin interface. Its JUnit output was not a usable
fallback: a capability-gated skip comes out as `skipped="0"` on the suite, in a non-standard
`<error type="skipped">` element, with the step text where the skip reason should be.

That difference is not incidental. godog's formatter *events* report a gated scenario correctly, as
`Skipped` for every step; its internal *storage* additionally holds a `FAILED` result for the first
step of such a scenario, which is what the built-in formatters read and where the malformed JUnit
comes from. A formatter built on the events is right for the same reason the built-in ones are wrong.

One thing the formatter interface cannot supply: `Skipped(pickle, step, definition)` carries no
error, so the capability that gated a scenario is unrecoverable from the events alone. The reason
therefore comes from the capability gate itself and lands in `TestStepResult.message`.

### What identifies a report

`tck.specRevision` comes from [`revision.go`](./revision.go), and it is the module version
pinned in [`go.mod`](./go.mod) — the assets arrive as a Go module, so the pin *is* the revision.

It is written by hand, which needs a guard rather than an apology. Nothing generates it and nothing
can: a library package's test binary carries no module build information at all, so
`runtime/debug.ReadBuildInfo` reports no dependencies from one, in a Go workspace and in a plain
module alike — and the TCK always runs inside a library test binary, both its own self-tests and an
adopter's `TestConformance`. So `TestSpecRevisionIsRecorded` reads `go.mod` and fails if the constant
disagrees with the pin. Moving the pin without updating the constant breaks the suite's own tests
rather than a consumer's report.

A git tree hash over the artifact directory used to be carried beside it, so that a consumer could
tell which questions a report answers without trusting the recorded commit. The results now carry the
executed feature text itself, in the stream's `source` messages, which answers the same question with
the source rather than with a hash of it — and unlike the hash it cannot be asserted wrongly, because
it is the input godog parsed.

`declaration.declared` is what the provider claims, and it stays in the envelope because it is an
*input* to reading the results rather than a summary of them. A `SKIPPED` scenario says the question
was not put to this provider; only the declaration says whether that is because the provider declines
the capability. Given the declaration and a scenario's tags — which the stream carries — the reason
for a skip follows.

`provider.name` is what the provider reports through its own metadata, not `Config.Name`.
`Config.Name` is chosen to read well in a failure message — `flagd-rpc` — which makes it the
*configuration*, and it is reported as such. One provider with two materially different modes
produces two reports that are not interchangeable. No standard results format has a slot for the
subject under test — Messages records the runner, the runtime and the machine — which is why the
envelope has to name it.

### What identifies a scenario

`feature` and `name` together do not. Every row of a Scenario Outline shares one name, and the
type-mismatch matrix in `errors.feature` is eleven rows, so eleven results carry the same feature and
the same name. Anything that stopped there could not say which row failed, and a consumer keying on
the pair would keep whichever row it read last.

Messages identifies the row exactly, and always has. A pickle's `astNodeIds` end with the id of the
Examples `TableRow` it was expanded from, and the stream carries the `gherkinDocument` those ids
belong to, so the row's cells are recoverable from the stream alone:

```console
$ jq -c 'select(.pickle) | select(.pickle.name | startswith("Requesting the wrong type"))
         | .pickle.astNodeIds' reports/in-memory.ndjson
["25","9"]
["25","10"]
["25","11"]
...
```

A skipped row is identified the same way, which matters because the capability gate stops a scenario
before its first step: the four rows of the `@object` outline would otherwise be four `SKIPPED`
results differing in nothing — exactly as ambiguous as four failures.

The report used to carry the row's parameters in a field of its own. Messages had them all along, as
node ids, so the field was removed — four implementations had each reinvented it independently, one
of them by reverse-engineering how its runner maps a pickle back to a table row.

Removing it removed a coupling as well. Recovering the row from a `*godog.Scenario` meant reparsing
the embedded feature files and reproducing the node ids godog assigns, which come from a counter
godog shares across the files it parses — so reproducing them meant reproducing godog's whole parse.
A formatter is handed the same `gherkinDocument` godog compiled the pickles from, so the ids agree by
construction rather than by imitation.

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

### What a default build runs, and what it does not

The default build executes this module's own tests and nothing else: the three self-tests above, the
unit tests over configuration validation, capability gating and extension mounting, and the
**control-API request-sequence tests** in `httpcontrol_internal_test.go`. None of them needs Docker
and all of them finish in about a second.

That last file is worth naming because of what it replaced. Three rules in `control-api.yaml`
constrain the *sequence* of control calls rather than any single call — `/reset` is preferred over
`/start`, the `/start` fallback is probed once per suite and then remembered, and the scenario after
a disconnect must be prepared with `/start` because `/reset` is specified not to start a stopped
backend. None is visible in the code of one method, and until now all three were reachable only by
starting Docker and reading a container's logs, which means in practice they were unchecked. The
control API is HTTP, so a recording `httptest.Server` is a complete stand-in for a backend — and it
answers things a real launchpad cannot be asked for on demand: a `501`, a single `500`, a `/healthz`
that is unready for exactly three probes.

The **containerised conformance suites are excluded from the default build** — `providers/flagd/e2e`
and `providers/ofrep/e2e` here, and your own adoption if you follow them. They are gated on an
explicit opt-in and a maintainer runs them by hand before merging:

```console
TCK_RUN=1 go test -tags=e2e -timeout=20m -run Conformance ./...
```

**Why an adoption suite is excluded rather than gating a merge** is the same argument in every
language, so it is not restated here: see ["Running the suite in CI"][appendix-f-ci] in Appendix F.
What is Go-specific is where the exclusion lives, and that is this:

- **The exclusion is a runtime skip inside the test function**, reading `TCK_RUN`, and nothing in the
  build re-enables it. That is the first of the two mistakes Appendix F names, and it is the one this
  repository was already making: `make e2e` expands to
  `go list -f '{{.Dir}}/...' -m | xargs -I{} go test -timeout=3m -tags=e2e {}` over every module in
  the workspace, so the `e2e` build tag is applied to everything and a tag is therefore not an
  exclusion here — it is the opposite. Both adoptions were running, red, on every pull request before
  the gate was added.
- **A runtime skip rather than a second build tag** (`//go:build e2e && tck`) also satisfies the
  appendix's requirement that the suite keep compiling when it does not run: the adoption stays
  typechecked against this package under `-tags=e2e`, so a signature change here cannot rot an
  adoption unnoticed. Only the container work is skipped, and the skip message names the variable
  that turns it on.
- **It is written down** — here, and in each adoption's own README — which is the appendix's second
  mistake avoided. No scheduled or path-filtered workflow was added; the three self-tests below still
  run in the default build and are the fast canary.

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
  The specification now marks the endpoint `[OPTIONAL]` for that reason: it had been `[REQUIRED]` on
  the strength of a claim, in its own description, that a TCK used it for the disconnect/reconnect
  scenarios, and no language's TCK ever did. It stays specified because a future `@caching` scenario
  asserting what a stale provider serves *during* an outage needs its flag-state preservation, which
  `/start` on reconnect does not give.
- **Hooks and flag metadata** are not covered. Provider metadata is, but only as far as a non-empty
  name.
- **Caching is not covered, and the constraint that follows is not Go's to state.** `@caching` is
  reserved and therefore not declarable. The reason a provider's own client-side cache can rewrite
  the reason under a scenario's feet — flagd's RPC resolver runs an LRU cache by default and reports
  `CACHED` on a repeat evaluation — used to be written out here, and it is now Appendix F's
  ["Caching" gap entry][appendix-f-gaps], with the constraint on scenario authors spelled out
  there: no scenario may evaluate the same flag twice without a configuration change in between.
  It is a constraint on new scenarios and not a defect in any provider, which is why it belongs with
  the scenarios rather than in one language's README.

  What is left for this file is the Go adoption's own position: `providers/flagd/e2e` does not turn
  the cache off, so the suite here really does run against a caching provider while asserting
  `STATIC` everywhere, and it passes only because the canonical set respects that constraint. One
  consequence is worth recording because the appendix does not: the configuration-change scenario
  already depends on cache invalidation working, without saying so — against flagd RPC it reads
  `changing-flag`, changes it and reads again, which gives the right answer only because the change
  event evicts the entry.

[appendix-a]: https://github.com/open-feature/spec/blob/main/specification/appendix-a-included-utilities.md
[appendix-b]: https://github.com/open-feature/spec/blob/main/specification/appendix-b-gherkin-suites.md
[appendix-f]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md
[appendix-f-ci]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#running-the-suite-in-ci
[appendix-f-deviations]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#rules-for-declaring
[appendix-f-gaps]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#open-questions
[control-api]: https://github.com/open-feature/spec/blob/main/specification/assets/provider-tck/openapi/control-api.yaml
[cucumber-messages]: https://github.com/cucumber/messages
[numeric-coercion-adr]: https://github.com/open-feature/flagd/blob/main/docs/architecture-decisions/numeric-coercion.md
[reinit-fix]: https://github.com/open-feature/spec/commit/fc99d5ace4da472a5fea0595fa4db8034bbbc769
[report-schema]: https://github.com/open-feature/spec/blob/main/specification/assets/provider-tck/report/conformance-report.schema.json
[req-147]: https://github.com/open-feature/spec/blob/main/specification/sections/01-flag-evaluation.md#requirement-147
[req-223]: https://github.com/open-feature/spec/blob/main/specification/sections/02-providers.md#requirement-223
[req-224]: https://github.com/open-feature/spec/blob/main/specification/sections/02-providers.md#requirement-224
[req-225]: https://github.com/open-feature/spec/blob/main/specification/sections/02-providers.md#requirement-225
[req-252]: https://github.com/open-feature/spec/blob/main/specification/sections/02-providers.md#requirement-252
[spec]: https://github.com/open-feature/spec
[tracking]: https://github.com/open-feature/spec/issues/417
