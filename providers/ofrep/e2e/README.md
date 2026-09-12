# OFREP Provider Conformance Suite

Runs the cross-language [OpenFeature Provider TCK](../../../tools/provider-tck/README.md) against
the [OFREP provider](../) — the same Gherkin scenarios, the same canonical flag set and the same
backend control API that every other language's TCK runs.

```bash
go test -tags=e2e -run TestOFREPConformance -timeout=10m ./...
```

## Backend

The existing `flagd-testbed`, unmodified. flagd serves the OFREP API on container port **8016**
alongside its own protocols, and the testbed's `docker-compose.yaml` already publishes 8016 next to
8013/8015/8080. So the OFREP provider is exercised against a real, conformant OFREP backend seeded
with the canonical flag set, driven by the launchpad control API that is already there — no new
image, no new compose file, and no change to the flagd suites.

The testbed lives in the flagd provider's submodule, so check it out first:

```bash
git submodule update --init --recursive
```

The stack starts once per suite and is never restarted; scenario isolation comes from the control
API. Host ports are read back after the stack is up, because compose assigns them dynamically.

## Capabilities

The OFREP provider is stateless. Its whole method set is `Metadata`, the five typed `*Evaluation`
methods and `Hooks`; it implements neither `openfeature.EventHandler` nor
`openfeature.StateHandler`.

| Capability | Declared | Why |
| --- | --- | --- |
| `@object` | yes | `ObjectEvaluation` passes the decoded JSON object through, and every scalar request against it reports `TYPE_MISMATCH`. |
| `@numeric-coercion` | yes | `ResolveInt` round-trips the decoded `float64` through `int64` and reports `TYPE_MISMATCH` when that is lossy, so `float-flag` requested as an integer is a mismatch rather than a silent `0`. |
| `@variants` | yes | OFREP's evaluation response carries `variant`, and the provider passes it into `ResolutionDetail`. Seven of the eight rows pass; the eighth asks `large-integer-flag`, which the testbed does not serve. |
| `@targeting` | yes | The evaluation context is the OFREP request body, so `targeting-key-flag`'s rule resolves to its `hit` variant for a matching key and `miss` otherwise. All three scenarios pass. |
| `@disabled-flags` | yes | flagd's OFREP endpoint answers `200` with `{"key":…,"reason":"DISABLED"}` and **no `value` member**, and the provider has an explicit `DISABLED` branch in each typed resolver that returns the caller's default with no error. All four rows pass. See below. |
| `@events` | no | No `EventChannel`; the provider can never publish a provider event. |
| `@configuration-change` | no | Follows from `@events`. Values do change on the next evaluation — nothing signals that they did. |
| `@stale` | no | Follows from `@events`. No state handling means no state to transition. |
| `@lifecycle` | no | Nothing to initialise or shut down. `Init`, `Status` and `Shutdown` do not exist on this provider, so the whole of `lifecycle.feature` asserts SDK behaviour rather than provider behaviour. |
| `@unavailable` | no | No `Init` to fail. The SDK reports `READY` unconditionally for a provider with no `StateHandler`, so an unreachable backend never produces the `ERROR` state the scenario asserts. |
| `@large-integers` | no | Not a provider property: `huge-integer-flag` is absent from `flagd-testbed`, so the capability cannot be exercised against this backend at all. See open-feature/flagd-testbed#392. |

`@events` gates `events.feature` and `@lifecycle` gates `lifecycle.feature`, so withholding both,
plus `@large-integers`, skips 9 of the 56 canonical scenarios. The remaining **47 run: 44 pass and
3 fail** — the evaluation and error-code matrix, which is the part that catches cross-language
disagreements.

All three failures are the backend fixture rather than the provider: `flagd-testbed` serves neither
`integral-float-flag` nor `large-integer-flag`, so the two scenarios that ask for them fail with
`FLAG_NOT_FOUND`, and the last `@variants` row fails because a flag that is not there has no variant
to name. open-feature/flagd-testbed#392 adds the flags and all three go green together. None gets a
known-deviation entry, because an entry there would attribute a fixture gap to the provider.

Declaring `@events` would make one more scenario go green for the wrong reason: the SDK synthesises
`PROVIDER_READY` for any provider without a `StateHandler` ("a provider without state handling
capability can be assumed to be ready immediately"), and it does so identically against a backend
that does not exist. Each omission is a gap to close in the provider, not a decision about the
suite.

## `@disabled-flags`, which this provider was expected not to have

The capability is gated on the reasoning that a disabled flag's resolution depends on where the
caller's default is substituted: a provider that evaluates locally holds it, and one whose backend
decides does not, because the default never leaves the process. OFREP is the clearest case of the
second kind — the request body carries the context and the flag key and nothing else — so the
expectation going in was that this provider could not have the capability at all, and that the
right outcome was an undeclared tag with the architecture written down beside it.

It passes, all four rows, on three consecutive runs. The reasoning was right about the server and
wrong about what the capability needs. Probed directly:

```console
$ curl -s -X POST localhost:$OFREP/ofrep/v1/evaluate/flags/disabled-string-flag \
    -H 'Content-Type: application/json' -d '{"context":{}}'
{"key":"disabled-string-flag","reason":"DISABLED","metadata":{}}
```

`200 OK`, no `value` member and no `variant`. The response does not have to carry the caller's
default; it only has to distinguish a disabled flag from a resolved one, and OFREP's `reason` field
does that. This provider then acts on it: each of the five typed resolvers in
`internal/evaluate/flags.go` checks `evalSuccess.Reason == DISABLED` before it type-asserts the
value, and returns `defaultValue` with reason `DISABLED` and no resolution error. Without that
branch a missing `value` would fail the assertion and come back as `TYPE_MISMATCH`, so the branch
is what earns the tag rather than an accident of JSON decoding.

So the capability stays gated for the same reason as before — a backend whose response says nothing
about state leaves a provider no way to answer — but it is not a property OFREP providers lack, and
nothing here needs a known-deviation entry.

## Known asymmetry

`float-flag` requested as an integer is correctly a `TYPE_MISMATCH`, but `integer-flag` requested as
a float returns `10.0` with no error, because JSON has one number type and `10` decodes to
`float64`. That direction now has a scenario of its own — "An integer requested as a float is
widened without loss", under `@numeric-coercion` — and it passes, because widening is the lossless
direction. So the asymmetry is deliberate rather than untested: "strict numeric typing" here is
strict only where being loose would lose information.
