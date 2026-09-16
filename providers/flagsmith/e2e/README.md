# Flagsmith provider conformance

The [OpenFeature Provider Conformance Suite](../../../tools/provider-tck) run against the Flagsmith
provider, in both of its evaluation modes.

```bash
go test -tags e2e ./...
```

The backend is a container, pulled automatically. Override it with `FLAGSMITH_TESTBED_IMAGE`.

## Status: draft

Out of 65 scenarios: **45 pass, 2 fail, 18 are skipped** because a capability is not declared.
Identical in both modes.

Both failures carry a `KnownDeviation` and one of them is a genuine provider defect — see "Why the 2
fail".

It stays a draft because the testbed image lives in a personal namespace
([aepfli/flagsmith-tck-testbed](https://github.com/aepfli/flagsmith-tck-testbed)), and a contrib
repo's CI should not depend on it until it has a permanent home.

> Read the counts from the suite's own summary, not from `go test`. `go test` prints a PASS line for
> every skipped scenario too — godog skips the scenario and the Go subtest passes anyway — so it
> prints `65 scenarios (63 passed, 2 failed)` and the honest pass count is 45. Reading godog's
> number as the conformance result is exactly the vacuous pass the capability gating exists to
> prevent; the real count is on the line underneath, `18 scenario(s) skipped because a capability
> was not declared`.

## The two modes

| | Who evaluates | Endpoint |
| --- | --- | --- |
| `flagsmith-remote` | the backend's **Python** engine | `GET /api/v1/flags/` |
| `flagsmith-local` | the SDK's **Go** engine, in-process | `GET /api/v1/environment-document/` |

Flagsmith's evaluation engine is independently reimplemented per language, so running both modes
against a byte-identical document compares two implementations of the same engine directly. Same
shape as GO Feature Flag's one engine in several hosts, except these are separate reimplementations
— which should make divergence *more* likely.

**They do not diverge.** Byte-identical results: same 45 passes, same 2 failures, same 18 skips,
same reasons — including all four of the evaluation-context and targeting scenarios. A negative
result from a test designed to find divergence, worth re-running when the Java and JS adoptions
exist.

## Why the 2 fail

`integral-float-flag` requested as an **integer** resolves to the code default, and a targeting rule
that does **not** match still reports `TARGETING_MATCH`. Both are under `KnownDeviation` entries in
[`tck_test.go`](tck_test.go): the first is the two numeric accessors disagreeing about the wire type,
the second is a real provider defect that `@standard-reasons` is what made visible.

Two other failures stood here until spec `d47a66eb` and are now **skips**: `float-flag` read as a
string returned `"0.5"`, and `object-flag` returned its raw JSON text. Neither was a provider bug —
Flagsmith's `feature_state_value` is natively boolean, integer or string, so on this backend both
genuinely *are* strings and asking for them as strings is a correct request that correctly succeeds.
That was recorded as an open question for the suite, and the suite has answered it: those rows sit
behind `@fully-typed-values`, which this adoption withholds.

**The finer capability arrived, and it recovered two honest passes.** `boolean-flag` and
`integer-flag` read as strings *do* report `TYPE_MISMATCH`, because those types really are native to
`feature_state_value` — so Flagsmith is partially typed rather than untyped. Until spec `bda599f1`
all four rows sat behind one tag and this adoption had to withhold it whole, giving up two passes it
had earned; this file recorded that as a granularity cost in the capability rather than a fact about
the provider. `bda599f1` split it, so `@string-typing` is now declared and answered, and only the
float and structured rows skip. That is why the tally moved from 43/2/20 to 45/2/18: nothing about
the provider changed, the question got asked at the right granularity.

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

`@fully-typed-values` is withheld for a reason that is neither of those: it is a fact about the
backend's type system rather than about the provider or about the SDK. `@string-typing`, the half of
that question this backend *can* answer, is declared and passes. See "Why the 2 fail" above for the
measurement and for why the split matters.

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
