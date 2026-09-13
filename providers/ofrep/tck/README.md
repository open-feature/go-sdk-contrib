# OFREP Provider Conformance Suite

Runs the cross-language [OpenFeature Provider TCK](../../../tools/tck/README.md) against
the [OFREP provider](../) — the same Gherkin scenarios, the same canonical flag set and the same
backend control API that every other language's TCK runs.

```bash
make tck                                 # from the repository root
go test -tags=tck -timeout=20m ./...     # equivalently, from here
```

Docker is the only prerequisite. There is **no submodule to check out** and no container code in
`tck_test.go`: the suite owns the stack.

This is a **module of its own**, `providers/ofrep/tck`. It was `providers/ofrep/e2e`, a module that
held nothing but this suite, and the name said it was a kind of e2e test — which is the conflation a
separate step exists to undo. An e2e suite tests a provider against its backend's own harness and is
expected green; this one tests the provider against the OpenFeature provider contract and fails
scenarios by design wherever a known deviation is declared. It is a module rather than a directory
under `providers/ofrep` because the provider module requires exactly one thing today, the Go SDK,
and this suite needs testcontainers, a Compose client and `tools/tck`.

**This suite has a step of its own, and `make tck` is it.** That target is
`go test -count=1 -timeout=20m -tags=tck ./...` over the conformance modules, which are the modules
named `tck` under a component directory — this one and `providers/flagd/tck`. `make e2e` is the same
sweep over every other module under `-tags=e2e`, followed by these same modules again under
`-tags=tck` with an empty `-run` pattern, which builds them and runs nothing. So this suite does not
run on a pull request and no pull request starts a Docker stack for it, while it still compiles on
every one. Why an adoption suite is excluded rather
than gating a merge, and why it gets a step of its own rather than a slice of an existing e2e suite,
is settled in Appendix F's
["Running the suite in CI"](https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#running-the-suite-in-ci)
rather than restated here — in short, a red `make tck` says *conformance* failed where the same
scenarios inside `make e2e` would only say *a test* failed, and for a suite that is currently red for
the reasons below, those are very different claims.

The mechanism is Go's, and the obvious single answers are all wrong on their own. What works is two
mechanisms doing two jobs — the directory decides which target runs the suite, the build tag keeps
it out of an untagged build — and neither substitutes for the other:

- **A build tag, `//go:build tck`, and what makes it safe is the target rather than the tag.** The
  standing objection to a build tag is that it takes `tck_test.go` out of the build, and the appendix
  asks for the opposite: the suite must keep *compiling* in whatever a pull request builds, even when
  it does not run, so a signature change in `tools/tck` cannot rot it unnoticed. That objection holds
  only while nothing in the pipeline builds *with* the tag. `make e2e`'s second command does exactly
  that — this module under `-tags=tck` with an empty `-run` pattern — excluding the **run** while
  keeping the **build**. Note what the tag is *not* doing: it is not what keeps this suite out of
  `make e2e`, because `make e2e` would apply `-tags=tck` as readily as it applies `-tags=e2e`, and
  this suite was in fact running, red, on every pull request before any gate existed, under a tag.
  What it is for is the untagged build: without it, `make test` and a bare `go test ./...` here would
  start a Docker stack. The file carried `//go:build e2e` until it moved out of `providers/ofrep/e2e`,
  which was right while the module was called that and a leftover the moment it was not.
- **Not an environment variable.** This suite used to skip unless `TCK_RUN` was set; that is gone,
  because the target does the same job without hiding the exclusion inside a test function, and a
  variable is not the step the appendix asks for.
- **`testing.Short()` is still checked**, and it is not the exclusion. Neither it nor a variable is
  sufficient alone: `-short` defaults the wrong way, since without the flag the suite *runs* and
  every pipeline would have to remember to opt out, while a variable defaults to off but is invisible
  from the build — an exclusion nobody can see is the appendix's second mistake. The short-mode skip
  is now the one guard left for a developer who runs `go test -tags=tck ./providers/ofrep/tck/` by
  hand, which is a deliberate gap: naming this package and asking for the tag is asking for it.
- **Not a test-name filter either, any more.** `make tck` was `-run 'Conformance'` and `make e2e` was
  `-skip 'Conformance'` while this suite still lived in `providers/ofrep/e2e`. That worked, but it
  made the *name* of a test load-bearing: a rename dropping `Conformance` would silently start it
  running under `make e2e` and stop it running under `make tck`, so this package carried a test that
  parsed it and checked the correspondence in both directions. The directory does that job now, and a
  file is in it or it is not. `TestOFREPConformance` keeps its name because `-run` and a failure line
  still read better with it, not because anything selects on it.
- **`guard_test.go` is what is left of that test, and it holds the half a directory cannot.**
  Selecting a module says which tests are *offered* to `make tck`; it cannot say that any of them
  still runs the suite. A conformance module whose tests have stopped calling `tck.Run` leaves
  `make tck` green by running nothing. The guard carries no build tag — which is the point, since it
  has to run in the build the suite is absent from — so `make test` runs it with neither Docker nor
  `-tags=tck`, and it parses `tck_test.go` off disk rather than importing it.

## Backend

The unmodified `flagd-testbed` image, described by
[`testdata/docker-compose.yaml`](testdata/docker-compose.yaml). flagd serves the OFREP API
on container port **8016** alongside its own protocols, and the same image serves the launchpad
control API on **8080**, so the OFREP provider is exercised against a real, conformant OFREP backend
seeded with the canonical flag set — no new image and no change to the flagd suites.

**Adopter-written infrastructure: none.** `tck_test.go` names the Compose file, names the one
container-internal port the provider connects to, and hands over a factory. The suite starts the
stack once, discovers the dynamically mapped host ports, builds the HTTP control against the
launchpad, waits until it accepts commands, constructs a provider per scenario and tears down after
the last one. The module does not even require `testcontainers-go` directly any more — it arrives as
an indirect dependency of `tools/tck`.

What that replaced is worth recording, because this was the last of the eight adoptions across four
languages still driving containers by hand. The old `startTestbed` called
`compose.NewDockerCompose` on the testbed submodule's own Compose file, created a temporary flags
directory and passed it in as `FLAGS_DIR` because that file bind-mounts `${FLAGS_DIR}` and an unset
value defaults to the submodule's own directory, registered two `t.Cleanup`s, declared its own wait
strategy, looked up two mapped ports by string, built the `tck.HTTPControl` itself, hard-coded
`localhost` as the host, and slept two seconds for the launchpad. All of it is gone; the harness
does each of those things, and `endpoint.Host()` is correct where the hard-coded `localhost` was
merely usually correct.

The Compose file here is deliberately **not** the testbed submodule's, for the same reasons the
flagd adoption's is not: no `${FLAGS_DIR}` bind mount, no envoy sidecar, and a service called
`backend`, which is the TCK's default and the name both the flagd adoption here and Java's use. It
publishes only 8016 and 8080, since nothing here speaks flagd's gRPC protocols. The image tag is
pinned in this file and in `providers/flagd/tck/testdata/docker-compose.yaml`; bump both
together, because a cross-provider disagreement is only evidence if both providers answered the same
backend.

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
| `@standard-reasons` | yes | The provider passes the server's `reason` string straight into `ResolutionDetail`, and flagd's OFREP endpoint sends Appendix F's mapping: `STATIC`, `TARGETING_MATCH`, `DEFAULT`, `DISABLED`, and `ERROR` beside an error code. All six scenarios pass — it composes with `@targeting` and `@disabled-flags`, both declared here, so none of them skips. |
| `@events` | no | No `EventChannel`; the provider can never publish a provider event. |
| `@configuration-change` | no | Follows from `@events`. Values do change on the next evaluation — nothing signals that they did. |
| `@stale` | no | Follows from `@events`. No state handling means no state to transition. |
| `@lifecycle` | no | Nothing to initialise or shut down. `Init`, `Status` and `Shutdown` do not exist on this provider, so the whole of `lifecycle.feature` asserts SDK behaviour rather than provider behaviour. |
| `@unavailable` | no | No `Init` to fail. The SDK reports `READY` unconditionally for a provider with no `StateHandler`, so an unreachable backend never produces the `ERROR` state the scenario asserts. |
| `@large-integers` | no | Not a provider property: `huge-integer-flag` is absent from `flagd-testbed`, so the capability cannot be exercised against this backend at all. See open-feature/flagd-testbed#392. |

The two rows that turn on a *backend* gap rather than on a provider property — `@numeric-coercion`
declared, `@large-integers` withheld — are decided by Appendix F's
["Rules for declaring"](https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#rules-for-declaring)
and not argued out here: declare a capability when at least one scenario gating it can actually be
put to the provider, withhold it only when none can, because the unit of the decision is the
*scenario* rather than the tag. `@numeric-coercion` has three scenarios and `flagd-testbed` can
answer two of them; `@large-integers` has one and the testbed serves no flag for it. The appendix's
two consequences apply as written: neither fixture failure gets a known-deviation entry, and the
withholding names open-feature/flagd-testbed#392 so that it is revisited when the backend gains the
flag instead of outliving its reason.

`@events` gates `events.feature` and `@lifecycle` gates `lifecycle.feature`, so withholding both,
plus `@large-integers`, skips 9 of the 65 canonical scenarios. The remaining **56 run** — the
evaluation, reason and error-code matrices, which are the part that catches cross-language
disagreements. The count moved from 47 of 56 when spec `c342461a` added `reason.feature`; the nine
skips are the same nine, because `@standard-reasons` is declared and the two capabilities its
scenarios compose with are declared too.

**Three failures are the floor, and all three are the backend fixture rather than the provider:**
`flagd-testbed` serves neither `integral-float-flag` nor `large-integer-flag`, so the two scenarios
that ask for them fail with `FLAG_NOT_FOUND`, and the last `@variants` row fails because a flag that
is not there has no variant to name. open-feature/flagd-testbed#392 adds the flags and all three go
green together. None gets a known-deviation entry, because an entry there would attribute a fixture
gap to the provider.

**Most runs have more than three, and the number moves.** Eleven consecutive runs against
`flagd-testbed:v3.8.0` on one machine produced 41, 12, 11, 33, 5, 19, 21, 40, 20, 4 and 3 failures
— the 19, 21 and 40 from the hand-rolled container wrapper this suite replaced, which is how we
know the flapping belongs to the backend and not to the harness. Every extra failure is
`FLAG_NOT_FOUND`, or a stale value, on a flag the testbed demonstrably serves: `curl` the OFREP
endpoint directly and `boolean-flag` answers `{"value":true,…,"reason":"STATIC","variant":"on"}`
every time.

`reason.feature` is caught by the same race and in the same way, which is worth saying because a
reason assertion failing looks at first like a vocabulary disagreement. It is not: the 20-failure
run had two rows of "A flag with no targeting rules resolves statically" fail with reason `ERROR`
rather than `STATIC`, which is the `FLAG_NOT_FOUND` above wearing the other field's clothes. The
two quiet runs failed none of them. Judge `@standard-reasons` on whether its scenarios fail
*consistently*, not on whether a red run contains one.

The cause is in the control API rather than here. `POST /start` — which is what isolates each
scenario, because the testbed's launchpad answers `404` to `/reset` — stops flagd, regenerates the
combined flag file, restarts flagd and polls `:8014/readyz` until flagd answers; flagd answers
before its file source has loaded the flags. A stateless provider fires its first evaluation the
instant `/start` returns and races that load, every scenario. **No sleep or retry is being added to
compensate**: the control API's promise is that a command has taken effect when it returns, and a
suite that sleeps instead of holding it to that promise stops being able to detect when it breaks.
So read a red run against the three-failure floor before attributing anything to the provider, and
re-run before concluding.

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
