# Flagsmith provider conformance

The [OpenFeature Provider Conformance Suite](../../../tools/provider-tck) run against the Flagsmith
provider, in both of its evaluation modes.

```bash
go test -tags e2e ./...
```

The backend is a container, pulled automatically. Override it with `FLAGSMITH_TESTBED_IMAGE`.

## Status: draft

**The suite is red, and the failures are the point.** Out of 40 scenarios: **17 pass, 12 fail, 11
are skipped** because a capability is not declared. Identical in both modes.

(`go test` prints 28 PASS lines. Eleven of those are the skipped scenarios -- the Go subtest passes
while godog skips the scenario -- so 28 is the subtest count, not the conformance result.) Ten of the twelve fail for a reason the suite cannot currently express — see
"Variant" below — so this is not a list of twelve provider bugs.

Two things keep it a draft:

1. The testbed image lives in a personal namespace
   ([aepfli/flagsmith-tck-testbed](https://github.com/aepfli/flagsmith-tck-testbed)). A contrib
   repo's CI should not depend on it until it has a permanent home.
2. The variant question below wants a decision on the canonical set before anyone treats these
   results as a conformance verdict.

## The two modes

| | Who evaluates | Endpoint |
| --- | --- | --- |
| `flagsmith-remote` | the backend's **Python** engine | `GET /api/v1/flags/` |
| `flagsmith-local` | the SDK's **Go** engine, in-process | `GET /api/v1/environment-document/` |

Flagsmith's evaluation engine is independently reimplemented per language, so running both modes
against a byte-identical document compares two implementations of the same engine directly. Same
shape as GO Feature Flag's one engine in several hosts, except these are separate reimplementations
— which should make divergence *more* likely.

**They do not diverge.** Byte-identical results: same 17 passes, same 12 failures, same 11 skips,
same reasons. A negative result from a test designed to find divergence, worth re-running when the Java
and JS adoptions exist.

## Why the 12 fail

### Variant — 10 of them

Every one fails with `variant was ""`.

Flagsmith has no variant concept for a standard feature. A feature state is `enabled` plus
`feature_state_value`, and nothing names the value; the evaluation response carries no variant key
at all. The provider is not dropping it — it never receives one, and no seeding of the canonical
set can produce one.

That makes this **a finding about the canonical set rather than about Flagsmith.** The set is
expressed in flagd's format and its comment says what matters is "the keys, types, variant names and
resolved values". Variant names are not universally available: a backend can be entirely conformant
and have no such concept, and the evaluation scenarios assert one unconditionally.

Worth settling on [spec#417](https://github.com/open-feature/spec/issues/417): either variant
assertions get a capability gate, the way `@object` and `@large-integers` gate theirs, or the
canonical set stops requiring them. No `KnownDeviation` is recorded, because there is no capability
to hang one on — which is itself the gap.

### Type mismatch — the other 2

Reading `float-flag` as a **string** returns `"0.5"` rather than `TYPE_MISMATCH`, and `object-flag`
as a string returns the raw JSON text.

Neither is a provider bug. Flagsmith's `feature_state_value` is natively boolean, integer or string
— no float type, no object type — so on this backend both genuinely *are* strings, and asking for
them as strings is a correct request that correctly succeeds. The scenario assumes the backend's
type system distinguishes them.

## Capabilities

Declared: `@object`, `@large-integers`.

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
