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
| `@strict-numeric-typing` | yes | `ResolveInt` round-trips the decoded `float64` through `int64` and reports `TYPE_MISMATCH` when that is lossy, so `float-flag` requested as an integer is a mismatch rather than a silent `0`. |
| `@events` | no | No `EventChannel`; the provider can never publish a provider event. |
| `@configuration-change` | no | Follows from `@events`. Values do change on the next evaluation — nothing signals that they did. |
| `@stale` | no | Follows from `@events`. No state handling means no state to transition. |
| `@unavailable` | no | No `Init` to fail. The SDK reports `READY` unconditionally for a provider with no `StateHandler`, so an unreachable backend never produces the `ERROR` state the scenario asserts. |

`@events` is a feature-level tag on both `events.feature` and `lifecycle.feature`, so withholding it
skips 5 scenarios and runs the remaining **24 of 29** — the evaluation and error-code matrix, which
is the part that catches cross-language disagreements.

Declaring `@events` would make one more scenario go green for the wrong reason: the SDK synthesises
`PROVIDER_READY` for any provider without a `StateHandler` ("a provider without state handling
capability can be assumed to be ready immediately"), and it does so identically against a backend
that does not exist. Each omission is a gap to close in the provider, not a decision about the
suite.

## Known asymmetry

`float-flag` requested as an integer is correctly a `TYPE_MISMATCH`, but `integer-flag` requested as
a float returns `10.0` with no error, because JSON has one number type and `10` decodes to
`float64`. No scenario covers that direction, and for a JSON protocol it is arguably right — but it
means "strict numeric typing" here is strict in one direction only.
