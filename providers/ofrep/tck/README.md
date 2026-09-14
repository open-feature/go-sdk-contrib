# OFREP Provider Conformance Suite

Runs the cross-language [OpenFeature Provider TCK](../../../tools/tck/README.md) against the
[OFREP provider](../). It asks whether the provider implements the provider contract the same way
every other provider does, rather than whether it works — which is why it is a module of its own
(formerly `providers/ofrep/e2e`, which held nothing else) and why a red result here means something
different from a red e2e run. The layout and the exclusion mechanism are in the [harness
README](../../../tools/tck/README.md#running-it); the reasoning is [Appendix F's][appendix-f-ci].

```bash
make tck                                 # from the repository root
go test -tags=tck -timeout=20m ./...     # equivalently, from here
```

Docker is the only prerequisite; there is no submodule to check out and no container code in
`tck_test.go`.

## Backend

The unmodified `flagd-testbed` image, described by
[`tests/flagd-testbed/docker-compose.yaml`](../../../tests/flagd-testbed/docker-compose.yaml). flagd
serves the OFREP API on container port **8016** alongside its own protocols and the launchpad control
API on **8080**, so the provider is exercised against a real, conformant OFREP backend seeded with the
canonical flag set, with no new image. That Compose file is one file for every conformance adoption
in this repository rather than a copy per module, because a cross-provider disagreement is only
evidence if both providers answered the same backend; this suite asks the harness for 8016 and the
flagd suites ask for 8013 and 8015.

## What it declares

The OFREP provider is stateless: `Metadata`, the five typed `*Evaluation` methods and `Hooks`, and
neither `openfeature.EventHandler` nor `openfeature.StateHandler`. The full reasoning for each row is
beside the declaration in [`tck_test.go`](tck_test.go).

| Capability | Declared | Why |
| --- | --- | --- |
| `@object` | yes | `ObjectEvaluation` passes the decoded JSON object through, and every scalar request against it reports `TYPE_MISMATCH`. |
| `@numeric-coercion` | yes | `ResolveInt` round-trips the decoded `float64` through `int64` and reports `TYPE_MISMATCH` when lossy. The other direction is deliberately loose — `integer-flag` as a float returns `10.0`, JSON having one number type — and has a passing scenario of its own, widening being lossless. |
| `@variants` | yes | The response carries `variant` and the provider passes it into `ResolutionDetail`; seven of eight rows pass, the eighth asking for a flag the testbed does not serve. |
| `@targeting` | yes | The evaluation context is the OFREP request body, so `targeting-key-flag` resolves `hit` for a matching key and `miss` otherwise. |
| `@standard-reasons` | yes | The provider passes the server's `reason` through and flagd's OFREP endpoint sends Appendix F's mapping. Composes with `@targeting` and `@disabled-flags`, both declared, so none of its scenarios skips. |
| `@disabled-flags` | yes | Expected to be impossible here and is not — see below. |
| `@events` | no | No `EventChannel`; the provider can never publish a provider event. |
| `@configuration-change`, `@stale` | no | Follow from `@events`. Values do change on the next evaluation; nothing signals that they did. |
| `@lifecycle`, `@unavailable` | no | Nothing to initialise or shut down, so `lifecycle.feature` would assert SDK behaviour: the SDK synthesises `PROVIDER_READY` for a provider with no `StateHandler`, even against a backend that does not exist. |
| `@large-integers` | no | Not a provider property: `large-integer-flag` is absent from `flagd-testbed`, so nothing can be established — [flagd-testbed#392][testbed-392]. |

The two rows turning on a *backend* gap rather than a provider property follow [Appendix F's first
rule for declaring][appendix-f-rules], where the unit is the scenario and not the tag:
`@numeric-coercion` has three scenarios and the testbed can answer two, `@large-integers` has one and
it cannot be asked at all. The withheld capabilities skip 9 of the 65 canonical scenarios; the
remaining **56 run**, which is the part that catches cross-language disagreements.

**`@disabled-flags` was expected to be impossible here.** The capability is gated because a disabled
flag's resolution depends on where the caller's default is substituted, and OFREP is the clearest
case of a backend that decides. All four rows pass anyway: flagd's OFREP endpoint answers `200` with
`{"key":…,"reason":"DISABLED"}` and no `value` member, and a response only has to distinguish a
disabled flag from a resolved one rather than carry the default. Each typed resolver in
`internal/evaluate/flags.go` checks the reason before it type-asserts — without that branch the
missing `value` would come back as `TYPE_MISMATCH`, so the branch earns the tag rather than JSON
decoding doing it by accident.

## The tally

**Three failures are the floor, and all three are the backend fixture**: the two flags the pinned
`flagd-testbed` image does not serve, which [flagd-testbed#392][testbed-392] adds. None gets a
known-deviation entry, because that would attribute a fixture gap to the provider; no provider
deviation is recorded at all.

**Most runs have more than three, and the number moves.** Eleven consecutive runs against
`flagd-testbed:v3.8.0` gave 41, 12, 11, 33, 5, 19, 21, 40, 20, 4 and 3 failures — three of the worst
from the hand-rolled container wrapper this suite replaced, which is how we know the flapping belongs
to the backend. Every extra failure is `FLAG_NOT_FOUND`, a stale value, or a `reason` of `ERROR`, on
a flag the testbed demonstrably serves. It is the launchpad's `POST /start` returning before the
flags are evaluable, which a provider with no initialisation to block on races on every scenario;
[flagd-testbed#394][testbed-394] measures it and fixes it. **No sleep or retry is being added to
compensate.** Read a red run against the floor, judge `@standard-reasons` on whether its scenarios
fail *consistently*, and re-run before concluding.

[testbed-392]: https://github.com/open-feature/flagd-testbed/pull/392
[testbed-394]: https://github.com/open-feature/flagd-testbed/pull/394

[appendix-f-ci]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#running-the-suite-in-ci
[appendix-f-rules]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#rules-for-declaring
