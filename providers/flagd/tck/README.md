# flagd Provider Conformance Suite

This module runs the cross-language [OpenFeature Provider TCK](../../../tools/tck/README.md) against
the flagd provider. It answers a different question from the [e2e suites next door](../e2e/README.md)
— not "does flagd work?" but "does the flagd provider implement the provider contract the same way
every other provider does?" — which is why it is a module of its own and why a red result here means
something different. The layout and the exclusion mechanism are in the [harness
README](../../../tools/tck/README.md#running-it); the reasoning is [Appendix F's][appendix-f-ci].

```bash
make tck                                 # from the repository root
go test -tags=tck -timeout=20m ./...     # equivalently, from here
```

Docker is the only prerequisite; there is no submodule to check out and no container code in
`tck_test.go`.

## What it runs

`TestFlagdRPCConformance` and `TestFlagdInProcessConformance`, separate suites because the two
resolvers are separately conformant. Both run against the unmodified `flagd-testbed` image described
by [`tests/flagd-testbed/docker-compose.yaml`](../../../tests/flagd-testbed/docker-compose.yaml),
whose launchpad already implements the control API. That file is shared with the other conformance
adoptions rather than copied per module, so the image tag cannot drift between suites whose results
are only comparable if both answered the same backend; its own header says why it is not the testbed
submodule's Compose file.

## What each resolver declares

Measured, with the requirement each row turns on. [`tck_test.go`](tck_test.go) carries the short form
beside each declaration.

| Capability | RPC | in-process | Why |
| --- | --- | --- | --- |
| `@stale` | **no** | yes | RPC goes straight to `PROVIDER_ERROR` on connection loss and never emits `PROVIDER_STALE`; in-process emits it and escalates after the retry grace period. Requirement 5.1.1 permits both, so no deviation — but an application switching resolver silently stops receiving stale events. |
| `@reinitialization` | no | no | `Init` after `Shutdown` never completes. Requirement 2.5.2 makes reuse permitted, not required, so this skips rather than fails. |
| `@large-integers` | no | no | Not a provider property: `large-integer-flag` is absent from `flagd-testbed`, so nothing can be established. [flagd-testbed#392][testbed-392], named so the withholding does not outlive its reason. |
| `@numeric-coercion` | yes | yes | Declared although one scenario fails — see below. |
| `@disabled-flags` | yes | yes | Expected to split by architecture and does not: flagd answers reason `DISABLED` with an empty variant and a zero value, RPC recognises that pair and keeps the caller's default (`isDefaultOrDisabledFallback`), in-process reads the state out of the synced ruleset. |
| `@standard-reasons` | yes | yes | flagd reports Appendix F's mapping exactly; all nine executed rows of `reason.feature` pass in both. It composes with `@targeting` and `@disabled-flags`, both declared, so none of them skip. |
| `@string-typing` | yes | yes | flagd's definitions carry a type per flag, so a non-string flag asked for through the string accessor is a genuine mismatch. Gates the boolean and integer rows; both pass in both resolvers. |
| `@fully-typed-values` | yes | yes | The same question for `float-flag` and `object-flag`, split out by spec `bda599f1` for backends that type booleans and integers natively and keep floats and structures as text. flagd is not one — a flag definition states its own type, floats and objects included — so both tags are declared and all four scenarios keep running. |
| everything else | yes | yes | |

The two rows that turn on a backend gap rather than a provider property follow [Appendix F's first
rule for declaring][appendix-f-rules] rather than a judgement made here — the unit is the scenario,
not the tag. `@numeric-coercion` has three scenarios and this backend can answer two;
`@large-integers` has one and it cannot be asked at all.

## The one known deviation

flagd **does** coerce — `integer-flag` (10) requested as a float comes back as 10 with reason
`STATIC` and no error code, in both resolvers — and gets the other direction wrong: `float-flag`
(0.5) requested as an integer comes back as **0 with no error code at all** rather than
`TYPE_MISMATCH` with the caller's default, so the fractional part is discarded silently. So the
capability is declared, the scenario fails, and a `tck.TrackedDeviation` naming
[flagd#1996](https://github.com/open-feature/flagd/issues/1996) explains it. Withholding the tag
would replace that failure with three skips that cannot distinguish "does not coerce" from "coerces,
and loses information one way round". The price is a second failure that is not flagd's —
`integral-float-flag` is absent from the testbed — named in the deviation summary.

## flagd-testbed extensions

Both suites also run flagd-testbed's [`tck-extensions`][tck-extensions] scenarios, the flagd-specific
ones the canonical set does not ask, pinned in `go.mod`. Each suite selects the scenarios tagged
with its resolver and drops `@fractional-v2`, since both resolvers bucket with v3. The flagd steps
(`an option …`, the two envoy steps) are in [`extensions_test.go`](extensions_test.go).

## The tally

| | RPC | in-process |
| --- | --- | --- |
| canonical (65) | 62 pass, 3 skip | 62 pass, 2 skip, 1 fail |
| extensions run | 262: 259 pass, 3 fail | 268: 263 pass, 5 fail |
| extensions deselected | 37 | 31 |

godog's summary counts every skipped and deselected scenario as passed; read the suite's own skip
line under it.

Failures, none of them wiring:

- **Canonical, in-process:** the lossy narrowing above.
- **Provider forbidden, both:** the TCK registers with `SetNamedProviderWithContextAndWait`, so
  `PROVIDER_ERROR` fires before the scenario attaches its handler, and go-sdk replays a late
  handler for `ErrorState` but not for `FatalState`.
- **Connection via TargetUri, RPC:** the RPC resolver does not dial an `envoy://` target; the e2e
  suite excludes `@targetURI` for RPC for the same reason.
- **Resolves Object details with caching, both:** undefined step. The expected value is JSON with
  inner quotes, which the canonical `"([^"]*)"` value step cannot match.
- **Three metadata scenarios, in-process:** integer metadata comes back as `float64`.

[testbed-392]: https://github.com/open-feature/flagd-testbed/pull/392
[tck-extensions]: https://github.com/open-feature/flagd-testbed/tree/main/tck-extensions

[appendix-f-ci]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#running-the-suite-in-ci
[appendix-f-rules]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#rules-for-declaring
