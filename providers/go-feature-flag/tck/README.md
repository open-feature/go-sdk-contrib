# GO Feature Flag Provider Conformance Suite

Runs the cross-language [OpenFeature Provider TCK](../../../tools/tck/README.md) against the
[GO Feature Flag provider](../), once per evaluation type: `INPROCESS` pulls the flag configuration
from the relay proxy and evaluates it with the GO Feature Flag core module, `REMOTE` asks the relay
proxy over OFREP. They are separate suites because they are separately conformant, and any
difference between the two results is one an application would see when it switches evaluation
type.

```bash
make tck                                 # from the repository root
go test -tags=tck -timeout=20m ./...     # equivalently, from here
```

Docker is the only prerequisite.

`providers/go-feature-flag-in-process` is not covered. It is a different provider rather than an
older mode of this one: an OpenFeature facade over the embedded GO Feature Flag Go module, with its
own retrievers and no relay proxy, so it has nothing to point at this backend. In-process
evaluation against a relay proxy is this provider's `INPROCESS` mode.

## Backend

[aepfli/goff-tck-testbed][testbed]: the unmodified relay proxy, with a launchpad implementing the
control API that writes the relay proxy's flag file and forces a retriever refresh. The
configuration source is faked; the evaluation never is. Described by
[`testdata/tck/docker-compose.yaml`](testdata/tck/docker-compose.yaml), with the image pinned to
`0.1.0` and both ports left to Docker: **1031** is the relay proxy, which both evaluation types use,
and **8080** is the control API. No API key is needed.

The provider is configured for the suite in two ways, both in [`tck_test.go`](tck_test.go):

- `FlagChangePollingInterval` is 500ms rather than two minutes, against which no event scenario can
  complete.
- `DisableCache` is set. Every OFREP response from the relay proxy says `gofeatureflag_cacheable:
  true` and a cache hit rewrites the reason to `CACHED`, so with the cache on the remote suite would
  be asserting this client's cache rather than the resolution.

## Results

Against `ghcr.io/aepfli/goff-tck-testbed:0.1.0`, 65 canonical scenarios each. The counts are the
scenario summary, not the `--- PASS` lines, which godog prints for skipped scenarios too.

| Mode | Passed | Failed | Skipped | Withheld | Known deviations |
| --- | --- | --- | --- | --- | --- |
| `INPROCESS` | 63 | 2 | 0 | none | `@numeric-coercion`, `@configuration-change` |
| `REMOTE` | 60 | 0 | 5 | `@configuration-change`, `@stale`, `@unavailable` | none |

The first run declared everything in both modes: in-process gave 63/2/0, remote 60/5/0, with the
five remote failures exactly the three withheld capabilities' scenarios.

### In-process: two deviations

**`@numeric-coercion` — a float is narrowed to an integer.** `float-flag` (0.5) requested through
`IntEvaluation` resolves to `0` with reason `STATIC`, variant `half` and no error, where the scenario
expects `TYPE_MISMATCH` and the code default. The narrowing is the core module's:
`evaluation.Evaluate` converts any `float64` with `int(value)` when the expected type is `"int"`,
and the in-process evaluator asks for `"int"` and passes the result through. The two lossless
scenarios pass.

**`@configuration-change` — the event names no flags.** `PROVIDER_CONFIGURATION_CHANGED` arrives
within one poll and the new value is served, but the event carries only a message and an empty
`FlagChanges`, so the payload assertion fails. The evaluator has the old and the new flag map in hand
when it swaps them.

Neither is tracked in an issue yet, so both are `tck.UntrackedDeviation`.

Not asserted by any scenario, measured separately: `Init` pushes a `PROVIDER_CONFIGURATION_CHANGED`
for the first load onto the event channel before it returns. The SDK forwards the provider's channel
on a different goroutine from the one that emits `PROVIDER_READY`, so a handler registered before
the provider sees that event before `READY` in some runs and after it in others (two of five in a
probe).

### Remote: three capabilities not attempted

Remote evaluation is the OFREP provider underneath, and with the cache off nothing else reaches the
backend:

| Capability | Why it is withheld |
| --- | --- |
| `@configuration-change`, `@stale` | The remote evaluator never writes to the event channel. Its only poll belongs to the cache and just purges it on a changed ETag. |
| `@unavailable` | `Init` returns at once with the cache off and ignores the result of its one request with the cache on, so against a closed port the provider is `READY` in under a millisecond. Evaluations then report `GENERAL` with the code default. |

`@events` and `@lifecycle` stay declared and pass, matching the Java remote mode. Whether a remote
mode should emit events and fail init at all is [java-sdk-contrib#1948][java-1948].

## Compared with Java

The Java adoption ([java-sdk-contrib#1947][java-1947]) runs the same image against the same
provider design. In-process gives Java 64 passed and 1 skipped, the skip being `@large-integers`,
which Java cannot ask. Remote gives Java four withheld capabilities and one known failure.

| | Go | Java |
| --- | --- | --- |
| In-process `float-flag` as integer | `0`, `STATIC`, no error — **fails** | `TYPE_MISMATCH` — passes |
| In-process change event payload | empty `FlagChanges` — **fails** | names `changing-flag` — passes |
| In-process `@large-integers` | passes | skipped (32-bit accessor) |
| Remote `@disabled-flags` | passes | fails, `FLAG_NOT_FOUND` ([java-sdk-contrib#1850][java-1850]) |
| Remote `@numeric-coercion` | passes | withheld, `integer-flag` as float returns the default |
| Remote `@configuration-change`, `@stale`, `@unavailable` | withheld | withheld |

The float row is the most useful one. Both providers run the same engine, but they reach it
differently. Go links the core module and calls `evaluation.Evaluate` with an expected type, which
narrows. Java runs the WASM build untyped and type-checks the result itself, so it refuses. The
lossy-coercion rule is decided by the provider's glue around the engine, not by the engine, and the
two remote rows are the same kind of difference: each language's own OFREP provider is what
decides them.

[testbed]: https://github.com/aepfli/goff-tck-testbed
[java-1947]: https://github.com/open-feature/java-sdk-contrib/pull/1947
[java-1948]: https://github.com/open-feature/java-sdk-contrib/issues/1948
[java-1850]: https://github.com/open-feature/java-sdk-contrib/issues/1850
