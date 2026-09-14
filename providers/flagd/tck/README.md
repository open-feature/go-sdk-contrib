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
by [`testdata/docker-compose.yaml`](testdata/docker-compose.yaml), whose launchpad already implements
the control API.

That Compose file is deliberately **not** the testbed submodule's, which bind-mounts `${FLAGS_DIR}`
(defaulting to its own directory, so an unset value has the launchpad write into the checked-out
submodule) and runs an envoy sidecar only the e2e suites need. This one needs neither and names its
service `backend`, the TCK's default and the name Java's flagd adoption uses. The cost is the image
tag pinned in a second place — bump it here when the submodule moves.

## What each resolver declares

The full reasoning for every row — measured, with the requirement it turns on — is in the comments
beside each declaration in [`tck_test.go`](tck_test.go).

| Capability | RPC | in-process | Why |
| --- | --- | --- | --- |
| `@stale` | **no** | yes | RPC goes straight to `PROVIDER_ERROR` on connection loss and never emits `PROVIDER_STALE`; in-process emits it and escalates after the retry grace period. Requirement 5.1.1 permits both, so no deviation — but an application switching resolver silently stops receiving stale events. |
| `@reinitialization` | no | no | `Init` after `Shutdown` never completes. Requirement 2.5.2 makes reuse permitted, not required, so this skips rather than fails. |
| `@large-integers` | no | no | Not a provider property: `large-integer-flag` is absent from `flagd-testbed`, so nothing can be established. [flagd-testbed#392](https://github.com/open-feature/flagd-testbed/issues/392), named so the withholding does not outlive its reason. |
| `@numeric-coercion` | yes | yes | Declared although one scenario fails — see below. |
| `@disabled-flags` | yes | yes | Expected to split by architecture and does not: flagd answers reason `DISABLED` with an empty variant and a zero value, RPC recognises that pair and keeps the caller's default (`isDefaultOrDisabledFallback`), in-process reads the state out of the synced ruleset. |
| `@standard-reasons` | yes | yes | flagd reports Appendix F's mapping exactly; all nine executed rows of `reason.feature` pass in both. It composes with `@targeting` and `@disabled-flags`, both declared, so none of them skip. |
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

## The tally

**65 scenarios: 61 pass, 4 fail**, the same four in both resolvers. One is the provider's — the lossy
narrowing above. Three are the fixture's: `large-integer-flag` is absent, so its scenario fails with
`FLAG_NOT_FOUND` and the last `@variants` row has no variant to name, and `integral-float-flag` is
absent for the coercion scenario. flagd-testbed#392 adds both flags and all three go green together.
None gets a known-deviation entry, because an entry there would attribute a fixture gap to the
provider.

**Re-run a red result before reading anything into it.** The launchpad's `POST /start` returns before
flagd's file source has finished loading the regenerated flag file, so any scenario can fail with
`FLAG_NOT_FOUND` or reason `ERROR` on a given run. Six consecutive runs against `flagd-testbed:v3.8.0`
gave 2, 2, 3, 3, 17 and 20 failures, with almost disjoint failing sets above three — and the
20-failure run was the hand-rolled container wrapper this module replaced, so the flapping belongs to
the backend. An initialisation to block on makes it less frequent, not absent: flagd's RPC `Init`
waits for the event stream, which flagd serves as soon as it is listening and before its file source
has populated the store. **No sleep is being added to compensate**; that belongs in the testbed.

[appendix-f-ci]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#running-the-suite-in-ci
[appendix-f-rules]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#rules-for-declaring
