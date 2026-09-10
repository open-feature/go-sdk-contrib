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

- mapping backend responses onto typed resolution details: value, variant, reason, error code
- keeping the integer and float types distinct rather than coercing between them
- error handling: a type mismatch and an unknown flag return the code default, report the right
  error code, and never take the application down
- lifecycle: reaching `READY`, and settling into `ERROR` against an unreachable backend
- events: `PROVIDER_READY`, `PROVIDER_ERROR`, `PROVIDER_STALE`, `PROVIDER_CONFIGURATION_CHANGED`
- that a signalled configuration change is actually **applied** on re-evaluation, not merely
  signalled

Deliberately out of scope: backend evaluation logic and targeting (that is the backend's contract,
not the provider's — every canonical flag resolves to its default variant), the provider↔backend
wire protocol, and SDK behaviour, which belongs to the SDK's own [Appendix B][appendix-b] suite.

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
		Capabilities: []tck.Capability{tck.Events, tck.Lifecycle, tck.Object, tck.StrictNumericTyping},
	})
}
```

Three fields are required — `Name`, `NewProvider`, `Control` — and everything else has a working
default. `NewProvider` is a factory rather than an instance because each scenario gets its own
provider, and because a provider often cannot be configured before the suite starts: a container
stack's host ports do not exist until it is up.

Each scenario becomes a Go subtest, so `-run` selects one the usual way and failures name a
scenario.

The canonical feature files and flag set are **embedded in this module**, so **adopting it needs no
git submodule of your own** — `go get` this module and everything the suite runs is already inside
it. The submodule described under [The spec submodule](#the-spec-submodule) is a concern of
contributors to *this* module, not of anyone consuming it.

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
ones keep their `assets/gherkin/` paths. That partition is a contract rather than a detail: it stops
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
| `tck.UnavailableInit` | `@unavailable` | reports an error state instead of hanging against a dead backend |
| `tck.StrictNumericTyping` | `@strict-numeric-typing` | does not coerce between integer and float |
| `tck.Targeting` | `@targeting` | reserved; no scenarios yet |
| `tck.Caching` | `@caching` | reserved; no scenarios yet |

Untagged scenarios are mandatory and always run. `Capabilities` defaults to everything — narrow it
rather than widening it: start from the default, run the suite, and remove only what your provider
genuinely cannot do.

`@lifecycle` and `@events` are separate on purpose, and conflating them is the mistake the
vocabulary exists to prevent. The Go SDK synthesises `PROVIDER_READY` for any provider that does not
implement `openfeature.StateHandler`, so a provider with no initialisation passes the readiness
scenario without demonstrating anything — a `NoopProvider` passes it identically. Conversely a
stateless HTTP provider such as OFREP emits no events of its own and cannot declare `@events`, yet
the readiness scenario is not really about events. Declare `@lifecycle` when initialisation actually
reaches something and the client can observe how that went; declare `@events` when the provider
emits events. Neither implies the other.

`@strict-numeric-typing` deserves a note, because unlike the others it is **not** an optional
feature. The specification requires `TYPE_MISMATCH` when the requested type cannot be satisfied, and
narrowing `0.5` to `0` loses information silently — the worst failure mode a feature flag has,
because the application sees a plausible value and no error. It is a capability only so that a
provider with the defect can adopt today and see the gap reported explicitly rather than being
unable to adopt at all. Not declaring it is an admission of a known bug.

## Controlling the backend

`tck.BackendControl` is the single seam between the scenarios and whatever manipulates the backend.
Step definitions never talk to a backend directly, which is why the same Gherkin runs unchanged
against a containerised backend and against a provider manipulated in-process.

**If your provider talks to a backend, drive it over the HTTP control API** in
[`specification/assets/provider-tck/openapi/control-api.yaml`][control-api], which this module
carries at `pkg/tck/spec/specification/assets/provider-tck/openapi/control-api.yaml` and exposes as
bytes through `tck.ControlAPISpec()`. That API is
the normative contract for those providers, and it is what makes a conformance claim portable:
another language's TCK drives the same endpoints against the same stack and must get the same
answers.

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

## The spec submodule

The Gherkin, the canonical flag set and the control API are not owned by this repository. They are
the language-agnostic definitions in [open-feature/spec][spec], and they reach this package through
a git submodule of that repository at `pkg/tck/spec/`. Nothing is copied and nothing is generated:
the `//go:embed` directives in `pkg/tck/assets.go` name paths inside the submodule, so the
specification revision this suite conforms to is recorded by the submodule pin and by nothing else.

**Adopters need no submodule.** The embedded copy is compiled into the package, so a provider
consuming this module gets the assets with it.

**Contributors to this module do.** Without the submodule checked out the embed patterns match no
files and the package does not compile:

```console
git clone --recurse-submodules https://github.com/open-feature/go-sdk-contrib
# or, in an existing clone
git submodule update --init tools/provider-tck/pkg/tck/spec
```

CI checks out with `submodules: recursive` in both the `lint` and `test` jobs for the same reason.

Changing a scenario, a flag or a control endpoint means changing it in `open-feature/spec` first and
then advancing the pin here. Editing the submodule's working tree in place forks the definition of
conformance, which is the one thing this suite exists to prevent.

## Go-specific translation notes

Two places where the shared Gherkin needed a decision rather than a transcription:

**"no exception should have been thrown"** asserts that the evaluation did not **panic**. Go has no
exceptions, and the returned `error` is not one: an errored evaluation correctly returns the code
default *alongside* a non-nil error, which is the normal shape of the API. The behaviour the feature
files forbid — an unhandled failure escaping a flag evaluation and taking the host application down
— is a panic here. Registration is guarded the same way, because a provider that panics out of
`SetProvider` takes the application with it.

**Providers are registered under a suite-scoped domain, not a per-scenario one.** Registering a
provider in a domain replaces and shuts down the previous one; a fresh domain per scenario would
leave every provider of the suite registered and running, which for a provider holding a network
connection means leaking one connection per scenario.

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
    "implementation": "go-sdk-contrib/tools/provider-tck",
    "version": "v0.1.0",
    "specRevision": "dfa16586d91ca020ef1b3b82a7c972d833ff8f29"
  },
  "backend": { "description": "the Go SDK's memprovider.InMemoryProvider, rebuilt per scenario" },
  "declaration": { "declared": ["@events", "@object", "@strict-numeric-typing"] },
  "results": {
    "format": "cucumber-messages",
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

`PROVIDER_TCK_REPORT_DIR` is an environment variable rather than a `Config` field so that emitting a
report is a property of the run and not of the code: CI sets it, a developer running the suite
locally does not, and no adopter changes a line to publish one. Unset means no report, which is not
an error. Several suites in one test binary each write their own pair of files, so flagd's two
resolvers do not collide.

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
    182 "PASSED"
     26 "SKIPPED"
```

### Why this exists in Go before the other languages

Because Go is the language that needs it most. godog counts a capability-gated skip in its **passed**
tally:

```
29 scenarios (29 passed)
```

Five of those twenty-nine did not run. Appendix F is unambiguous that a scenario skipped for an
undeclared capability is reported as skipped with the reason and *never* as passed, and the harness
does say so in a separate log line — but the headline number still says something false, and a
number is what gets read. pytest and jest-cucumber both report skips correctly, so this is a property
of the runner rather than of the suite's design.

The report does not fix godog's summary. It makes the summary stop mattering. The stream above
accounts for all twenty-nine scenarios and reports five of them as `SKIPPED`, each carrying the
capability that gated it in `testStepResult.message`, so a consumer can check the rule instead of
trusting the runner to have applied it.

godog 0.15.1 has no Messages formatter — it registers `cucumber` (the legacy relishapp JSON),
`events`, `junit`, `pretty` and `progress` — so [`messages.go`](./pkg/tck/messages.go) is one,
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

`tck.specRevision` comes from [`revision.go`](./pkg/tck/revision.go), which `sync_assets.go`
generates from the submodule alongside the embedded artifacts. Generating both in the same command is
what keeps them honest: `make provider-tck-assets-check` regenerates and fails on any difference, so
a revision that disagrees with the artifacts beside it cannot be committed.

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
skipped, which is what they had been passing vacuously under `@events` before the tag existed.

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

## Known gaps

- **Evaluation context passthrough is unverifiable.** The scenarios build evaluation contexts but
  cannot assert one *reached* the backend. That needs an echo operation on the control API. Until
  then a provider that silently drops the context passes. `@targeting` is reserved for these.
- **`POST /restart` is unused.** No current scenario needs a bounded outage — the stale scenario
  uses an explicit disconnect and reconnect — so `tck.ConnectionControl` has no `DisconnectFor`.
- **Hooks and flag metadata** are not covered.
- **Caching is not covered, and the suite is quietly exposed to it.** `@caching` is reserved and no
  scenario carries it, but flagd's RPC resolver enables an LRU cache *by default* and rewrites the
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
[report-schema]: https://github.com/open-feature/spec/blob/main/specification/assets/provider-tck/report/conformance-report.schema.json
[cucumber-messages]: https://github.com/cucumber/messages
[spec]: https://github.com/open-feature/spec
[tracking]: https://github.com/open-feature/spec/issues/417
