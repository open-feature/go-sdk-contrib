# Flagsmith provider conformance

The [OpenFeature Provider Conformance Suite](../../../tools/provider-tck) run against the Flagsmith
provider, in both of its evaluation modes.

```bash
go test -tags e2e ./...
```

The backend is a container, pulled automatically. Override it with `FLAGSMITH_TESTBED_IMAGE`.

## Status: draft

Out of 52 scenarios: **31 pass, 2 fail, 19 are skipped** because a capability is not declared.
Identical in both modes.

The two failures are the only ones left after `@variants` landed on the base branch, and neither is
a provider bug — see "Why the 2 fail".

It stays a draft because the testbed image lives in a personal namespace
([aepfli/flagsmith-tck-testbed](https://github.com/aepfli/flagsmith-tck-testbed)), and a contrib
repo's CI should not depend on it until it has a permanent home.

> Read the counts from the suite's own summary, not from `go test`. `go test` prints a PASS line for
> every skipped scenario too — godog skips the scenario and the Go subtest passes anyway — so its
> PASS count is 50, not 31. Reading that as the conformance result is exactly the vacuous pass the
> capability gating exists to prevent.

## The two modes

| | Who evaluates | Endpoint |
| --- | --- | --- |
| `flagsmith-remote` | the backend's **Python** engine | `GET /api/v1/flags/` |
| `flagsmith-local` | the SDK's **Go** engine, in-process | `GET /api/v1/environment-document/` |

Flagsmith's evaluation engine is independently reimplemented per language, so running both modes
against a byte-identical document compares two implementations of the same engine directly. Same
shape as GO Feature Flag's one engine in several hosts, except these are separate reimplementations
— which should make divergence *more* likely.

**They do not diverge.** Byte-identical results: same 31 passes, same 2 failures, same 19 skips,
same reasons — including all four of the evaluation-context and targeting scenarios. A negative
result from a test designed to find divergence, worth re-running when the Java and JS adoptions
exist.

## Why the 2 fail

Reading `float-flag` as a **string** returns `"0.5"` rather than `TYPE_MISMATCH`, and `object-flag`
as a string returns the raw JSON text.

Neither is a provider bug. Flagsmith's `feature_state_value` is natively boolean, integer or string
— no float type, no object type — so on this backend both genuinely *are* strings, and asking for
them as strings is a correct request that correctly succeeds. The scenario assumes the backend's
type system distinguishes them.

Unlike the variant case there is no capability to withhold here, and inventing one looks wrong: the
type-mismatch matrix is testing something real, and Flagsmith simply cannot express half of it.
Recorded as an open question rather than declared solved.

## Capabilities

Declared: `@object`, `@large-integers`, `@targeting`.

`@targeting` holds via the one mechanism Flagsmith has for it. A targeting key **is** a Flagsmith
identifier — the provider calls `GetIdentityFlags(targetingKey)` whenever one is present — so the
testbed seeds the canonical rule as an entry in the environment document's `identity_overrides`.
Segments would be the wrong tool: they match on traits, and the canonical rule has none.

`@variants` is withheld, and it is the reason this adoption was worth running. Flagsmith has no
variant concept for a plain feature: a feature state is `enabled` plus `feature_state_value`,
nothing names the value, and the evaluation response carries no variant key at all. The provider
never receives one and no seeding can produce one. That is permitted rather than defective — 2.2.4
makes populating the variant a SHOULD and `types.md` marks the field optional — so it gets no
deviation entry. Before the capability existed these were untagged assertions and this provider
failed ten scenarios for something its author could not fix, with nothing to record it as.

Everything else is withheld, and almost all of it for one reason: the provider implements none of
`Init`, `Shutdown`, `Status` or `EventChannel`, so it is neither an `openfeature.StateHandler` nor
an `openfeature.EventHandler`.

- `@lifecycle`, `@events`, `@stale`, `@configuration-change` — a provider with no observable
  initialisation has no lifecycle to assert against. The SDK synthesises `PROVIDER_READY` on
  registration, so declaring `@lifecycle` would make those scenarios pass without the provider
  having done anything. A vacuous pass is worse than a skip.
- `@unavailable` — the provider cannot fail initialisation because it has no initialisation; it
  reports READY against a dead backend. `NewUnavailableProvider` is deliberately unset.
- `@reinitialization` — follows from the same absence, and 2.5.2 only says a provider SHOULD revert
  to its uninitialized state anyway.

None of these get a `KnownDeviation`. The specification does not require a provider to implement
`StateHandler` or `EventHandler`, so declining them is an option the contract offers, not a defect.

`@numeric-coercion` is the exception: it is withheld because of a **defect**, and it carries a
deviation entry. The two numeric accessors disagree about the wire type — `IntEvaluation` asserts
`float64` (a JSON number) while `FloatEvaluation` asserts `string` and calls `ParseFloat` — so a
Flagsmith integer feature resolves through `GetIntValue` and returns `TYPE_MISMATCH` through
`GetFloatValue`. No seeding satisfies both.

## Consequences worth knowing

- `POST /change`, `/restart` and `/reset` are implemented by the testbed and observed by **nothing**
  in this adoption, because no event or lifecycle capability is declared.
- Local evaluation has a startup race the provider cannot close: the first environment sync happens
  on a background poll with no `Init` to block in, so the SDK reports READY while evaluations still
  return the code default with `GENERAL`. The adoption waits for the first sync before handing the
  provider back — without that the suite is non-deterministic, which would look like flakiness
  rather than like the defect it is.
