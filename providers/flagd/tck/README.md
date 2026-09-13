# flagd Provider Conformance Suite

This module runs the cross-language [OpenFeature Provider TCK](../../../tools/tck/README.md) against
the flagd provider — the same Gherkin scenarios, canonical flag set and backend control API that
every other language's TCK runs. It answers a different question from the
[e2e suites next door](../e2e/README.md): not "does flagd work?" but "does the flagd provider
implement the provider contract the same way every other provider does?".

It is a **sibling** of `providers/flagd/e2e`, not a package inside it, and a **module** of its own
rather than a directory. Both halves are deliberate:

- The two suites mean different things by a red result. An e2e suite is expected green, so a failure
  there is a regression; this suite fails scenarios by design wherever a known deviation is declared
  below, and that failure is correct output until the defect is fixed upstream. One signal cannot
  carry both meanings. Filing this under `e2e/` said it was a kind of e2e test, which is the
  conflation a step of its own exists to undo.
- A plain directory would fall under `providers/flagd`'s module and put testcontainers, a Compose
  client and `tools/tck` into the dependency graph of every application that imports the flagd
  provider. The same argument applies to `providers/flagd/e2e`, which is why this is not in there
  either: after the move that module requires neither `tools/tck` nor the spec assets, and the spec
  pin they carried lives here instead.

## What it runs

- **Subjects**: `TestFlagdRPCConformance` and `TestFlagdInProcessConformance`. The two resolvers are
  separate suites because they are separately conformant.
- **Backend**: the unmodified `flagd-testbed` image, described by
  [`testdata/docker-compose.yaml`](testdata/docker-compose.yaml). The TCK drives its
  launchpad through the standardised control API, which the launchpad already implements.
- **Adopter-written infrastructure**: none. The suite owns the container lifecycle — it starts the
  stack, discovers the dynamically mapped host ports, builds the HTTP control against the launchpad
  and tears down after the last scenario. `tck_test.go` names the Compose file, names the
  container-internal port each resolver connects to, and hands over a factory. The hand-rolled
  wrapper it replaces went through `tests/flagd/testframework.NewFlagdContainer` with a temporary
  bind-mounted flags directory, built the control client itself and looked up ports by name.
- **Isolation**: the stack starts once per suite and is never restarted. Scenario isolation comes
  from the control API, because container orchestrators cannot reliably preserve dynamically mapped
  host ports across a restart.
- **Relationship to the e2e suites**: none. They are untouched, they keep using the testbed
  submodule's own Compose file, and so is `flagd-testbed` itself.

The Compose file here is deliberately **not** the testbed submodule's. That one bind-mounts
`${FLAGS_DIR}` — defaulting to its own directory, so an unset value has the launchpad write into the
checked-out submodule — and runs an envoy sidecar that exists for the TLS and permission-denied
scenarios of the e2e suites. The conformance suite needs neither. What it does need is a service
called `backend`, which is the TCK's default and the same name Java's flagd adoption uses, so the two
languages' stacks differ in nothing a reader has to reconcile. The cost is that the image tag is
pinned in two places; bump it here as well when the submodule moves.

Two differences between the resolvers show up as capability declarations rather than as failures:

| | RPC | in-process |
| --- | --- | --- |
| emits `PROVIDER_STALE` on connection loss | **no** — goes straight to `PROVIDER_ERROR` | yes, then escalates to `PROVIDER_ERROR` after the retry grace period |
| `@stale` scenario | skipped, with the reason reported | runs |

That gap is a real behavioural difference between two modes of the same provider: an application
that switches from in-process to RPC silently stops receiving stale events. `tck.Stale` is withheld
from the RPC suite so the scenario is reported as skipped rather than failed, and it should be
declared as soon as the RPC resolver emits `PROVIDER_STALE`.

Both resolvers declare `@disabled-flags`, and that one is a difference that turned out **not** to
exist. The capability is gated because what a disabled flag resolves to depends on where the
caller's default is substituted, so the RPC resolver — which asks flagd to resolve every flag —
looked like the side that could not have it. It can: flagd answers with reason `DISABLED`, an empty
variant and a zero value, and the provider recognises that pair and keeps the caller's default
(`isDefaultOrDisabledFallback` in `pkg/service/rpc/service.go`). The in-process resolver reads the
state out of the ruleset it synced and arrives at the same answer. All four rows pass in both.

Both resolvers also declare `@standard-reasons`, which is new in spec `c342461a` and is the one
capability here whose scenarios did not exist before. flagd reports `STATIC` for a rule-less flag,
`TARGETING_MATCH` for a matching rule, `DEFAULT` for a rule that exists and did not match,
`DISABLED` for a disabled flag and `ERROR` for a failed evaluation — Appendix F's mapping exactly.
All nine executed rows of `reason.feature` pass in both resolvers, which is why it is declared;
withholding it would cost nothing in coverage of `MUST`s, so declaring it is a claim about the
vocabulary rather than a convenience. It composes with `@targeting` and `@disabled-flags`, both
declared here, so all six of its scenarios run rather than three of them skipping.

Both resolvers declare `@numeric-coercion`, and one of its three scenarios fails. That combination
is deliberate, and it is the only known deviation either suite records. flagd **does** coerce —
`integer-flag` (10) requested as a float comes back as 10 with reason `STATIC` and no error code, in
both resolvers — and it gets the other direction wrong: `float-flag` (0.5) requested as an integer
comes back as **0 with no error code at all**, rather than `TYPE_MISMATCH` with the caller's default,
so the fractional part is discarded silently. That failure carries a known-deviation entry naming
[open-feature/flagd#1996](https://github.com/open-feature/flagd/issues/1996), the issue that
implements flagd's numeric-coercion ADR.

**Which way this goes is settled in Appendix F's
["Rules for declaring"](https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#rules-for-declaring)
rather than decided here**, and the rule is one sentence: declare a capability when at least one
scenario gating it can actually be put to the provider, and withhold it only when none can — the
unit of the decision is the *scenario*, not the tag. Both of this suite's answers fall out of it.
`@numeric-coercion` has three scenarios and this backend can still be asked two of them, so it is
declared. `@large-integers` has one, and the backend serves no flag for it, so nothing about it can
be established and it is withheld. What follows is that rule applied to measurements, not a second
argument for it.

Withholding the tag would turn that failure into three skips, and a skip cannot say which of "does
not coerce" and "coerces, and loses information one way round" is true — the passing widening
scenario is exactly that distinction. A withheld capability that *also* carries a deviation is the
combination
[Appendix F](https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md)
tells adopters to avoid, because it asserts a defect at something the suite never asked the provider
to do. Declaring and letting the scenario fail is the shape to prefer; the Java adoption reached the
same conclusion on the same evidence. This file did it the other way round until pass 7.

The price of declaring is a second failure that is not flagd's: `integral-float-flag` is absent from
`flagd-testbed` v3.8.0, so the remaining lossless scenario fails with `FLAG_NOT_FOUND`. It is the
same fixture gap as `large-integer-flag` below, it is named in the deviation summary so a reader is
not left counting it against the provider, and it is accepted rather than used as a reason to
withhold.

Both suites run **65 scenarios: 61 pass and 4 fail.** The four failures are the same set in both
resolvers. **One is the provider's** — the lossy narrowing above. **Three are the fixture's:**
`large-integer-flag` is absent from `flagd-testbed`, so "A large integer resolves without loss of
precision" fails with `FLAG_NOT_FOUND` and the last `@variants` row has no variant to name; and
`integral-float-flag` is absent for the coercion scenario just described.
[open-feature/flagd-testbed#392](https://github.com/open-feature/flagd-testbed/issues/392) adds both
flags and all three go green together. None of the three gets its own known-deviation entry, because
the gap is in the fixture and an entry there would attribute it to the provider — which is the first
of the two consequences the appendix states alongside the rule above. The second is why
`@large-integers` is withheld *with* that issue named: a capability withheld for a backend gap is
temporary in a way one withheld by choice is not, and a withholding with no note saying why outlives
its reason.

**Re-run a red result before reading anything into it.** The launchpad's `POST /start` returns
before flagd's file source has finished loading the regenerated flag file, so any scenario can fail
with `FLAG_NOT_FOUND` or reason `ERROR` on a given run. Six consecutive runs against
`flagd-testbed:v3.8.0` produced 2, 2, 3, 3, 17 and 20 failures, and the three runs with more than
three had almost disjoint failing sets — every extra failure `FLAG_NOT_FOUND` on a flag the testbed
definitely has. The 20-failure run was the hand-rolled container wrapper this file replaces and the
17 was the harness, so the flapping belongs to the backend and not to either of them.

That contradicts what this file used to say, which was that a provider with an initialisation to
block on does not hit the race. It hits it less often than a stateless one, not never: flagd's RPC
`Init` waits for the event stream, which flagd serves as soon as it is listening and before its file
source has populated the store. The OFREP suite documents the race in full.

**No sleep is being added to compensate.** The control API's promise is that a command has taken
effect when it returns, and a suite that sleeps instead of holding it to that promise stops being
able to detect when it breaks. This belongs in the testbed.

**These two suites have a step of their own, and it is the one command to run them:**

```bash
make tck                                 # from the repository root
go test -tags=tck -timeout=20m ./...     # equivalently, from here
```

`make tck` is `go test -count=1 -timeout=20m -tags=tck ./...` over the conformance modules, which
are the modules named `tck` under a component directory — this one and `providers/ofrep/tck`.
`make e2e` is the same sweep over every other module under `-tags=e2e`, followed by these same
modules again under `-tags=tck` with an empty `-run` pattern, which builds them and runs nothing. So
these two suites do not run on a pull request and no pull request starts a Docker stack for them,
and both still compile on every one.

Why an adoption suite is excluded rather than gating a merge, and why it gets a step of its own
rather than a slice of an existing e2e suite, is the same argument in every language and is settled
in Appendix F's
["Running the suite in CI"](https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#running-the-suite-in-ci)
rather than restated here. Its shape, because it is what decides the mechanism below: a red
`make tck` says *conformance* failed, where the same scenarios inside `make e2e` would only say *a
test* failed — and these two suites do carry failures by design, four of them today, each covered by
a declared deviation above. That is correct output from a conformance run and it would be a
regression in an e2e run.

The mechanism is Go's, and it is worth stating exactly, because the obvious single answers are all
wrong on their own and this suite had tried two of them. What works is two mechanisms doing two
jobs — the directory decides which target runs the suite, the build tag keeps it out of an untagged
build — and neither substitutes for the other:

- **A build tag, `//go:build tck`, and what makes it safe is the target rather than the tag.** The
  standing objection to a build tag is that it takes `tck_test.go` out of the build, and the appendix
  asks for the opposite: the suite must keep *compiling* in whatever a pull request builds, even when
  it does not run, so a signature change in `tools/tck` cannot rot it unnoticed. That objection holds
  only while nothing in the pipeline builds *with* the tag. `make e2e`'s second command does exactly
  that — this module under `-tags=tck` with an empty `-run` pattern — excluding the **run** while
  keeping the **build**. Note what the tag is *not* doing: it is not what keeps these suites out of
  `make e2e`, because `make e2e` would apply `-tags=tck` as readily as it applies `-tags=e2e`, and
  both suites were in fact running, red, on every pull request before any gate existed, under a tag.
  What it is for is the untagged build: without it, `make test` and a bare `go test ./...` here would
  start a Docker stack. The file carried `//go:build e2e` until it moved, which was right while it
  was a package inside the e2e suite and a leftover the moment it was not.
- **Not an environment variable.** These suites used to skip unless `TCK_RUN` was set. That is gone:
  the target does the same job without hiding the exclusion inside a test function, and it is the
  step the appendix asks for, which an environment variable is not. Keeping both would have left two
  mechanisms for one exclusion and the next person to touch the pipeline guessing which is
  load-bearing.
- **`testing.Short()` is still checked**, and it is not the exclusion. Neither it nor an environment
  variable is sufficient alone, and the reason is the default each one picks: `-short` defaults the
  wrong way, since without the flag the suite *runs* and every pipeline would have to remember to opt
  out, while a variable defaults to off but is invisible from the build — an exclusion nobody can see
  is the appendix's second mistake. What the short-mode skip is now is the one guard left for a
  developer who runs `go test -tags=tck ./providers/flagd/tck/` by hand, which is a deliberate gap:
  naming this package and asking for the tag is asking for it.
- **Not a test-name filter either, any more.** `make tck` was `-run 'Conformance'` and `make e2e` was
  `-skip 'Conformance'` while these suites still lived in `providers/flagd/e2e`. That worked, but it
  made the *name* of a test load-bearing: a suite renamed without the word in it would silently start
  running under `make e2e` and stop running under `make tck`, so this package carried a test that
  parsed it and checked the correspondence in both directions. The directory does that job now, and a
  file is in it or it is not. The two test names still end in `Conformance`; that is now a
  convenience for `-run` and for reading a failure line, not a contract, and nothing asserts it.
- **`guard_test.go` is what is left of that test, and it holds the half a directory cannot.**
  Selecting a module says which tests are *offered* to `make tck`; it cannot say that any of them
  still runs the suite. A conformance module whose tests have stopped calling `tck.Run` leaves
  `make tck` green by running nothing. The guard carries no build tag — which is the point, since it
  has to run in the build the suite is absent from — so `make test` runs it with neither Docker nor
  `-tags=tck`, and it parses `tck_test.go` off disk rather than importing it.
- It is written down here, in the harness's own README and in `CONTRIBUTING.md` next to the two
  targets it now sits beside, which is the appendix's second mistake avoided.
