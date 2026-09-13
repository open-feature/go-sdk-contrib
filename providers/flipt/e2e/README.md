# Flipt conformance (e2e)

Runs the [TCK](https://github.com/open-feature/spec) conformance suite against a
throwaway Flipt instance. Two providers are measured against the same backend:

- the flipt-native provider (`pkg/provider`, driven over Flipt's gRPC/HTTP SDK)
- the OFREP provider (`providers/ofrep`, driven over Flipt's OFREP endpoint)

## Run

From this directory:

```sh
go test -tags=e2e ./...
```

That runs the flipt-native suite. The OFREP suite is red-heavy (every red a
declared deviation) and needs its own stack, so it is opt-in:

```sh
OFREP_TCK_RUN=1 go test -tags=e2e -run TestOFREPConformance ./...
```

Docker is required. The suite builds the `testdata/tck` testbed, brings up the
compose stack `testdata/tck/docker-compose.yaml` with testcontainers, and drives
it through the control API.

## Testbed

The `testdata/tck` image runs a small control server (PID 1) that spawns the
bundled Flipt binary at boot, seeds the canonical flag set, and lets the TCK
restart, stop and reconfigure the backend over HTTP:

- `:8080` control API (`/start`, `/restart`, `/stop`, `/change`, `/reset`, `/healthz`)
- `:8888` Flipt backend (evaluation API)

The image builds on `ghcr.io/flipt-io/flipt:local` — a local build carrying the
disabled-boolean guard (`FLAG_DISABLED` for disabled boolean flags), which
upstream `v2` lacks. That tag exists only where it was built; the stack does
not build anywhere else until the fix ships upstream and the base moves to a
published tag.

### Seed model

The baseline is translated at boot from `flags/canonical-flags.json` out of the
embedded conformance assets (`tck.FS`, see `cmd/control/seed.go`) — there is no
hardcoded flag table. The translation is mechanical: `ENABLED`/`DISABLED` become
the Flipt state, each variant becomes a Flipt variant, and the default variant
name resolves to its Flipt key. What the JSON cannot express stays hand-coded:
the targeting segment and rule (the JSON's JsonLogic has no Flipt equivalent),
the boolean threshold rollouts (the canonical set has no rollout concept), and
the key rendering itself.

Flipt's management API accepts attachment values only as JSON objects and
forbids empty variant keys, so the canonical scalar values cannot be attached.
Scalar flags therefore store their values as the variant keys themselves
(rendered verbatim, so `9007199254740991` and `10.0` survive exactly);
`object-flag` is the one flag that uses real attachments keyed by canonical
name. Consequently Flipt cannot represent the canonical `string-zero-flag`
(empty string); it is seeded with a `"zero"` key instead.

The testbed pins the same assets revision the suite runs
(`tools/tck/go.mod`); advance both pins together or the seed skews from the
scenarios.

## Known deviations

A deviation here is a scenario that fails and is declared known through
`tck.WithKnownDeviations` — the failure stays visible so a consumer can tell a
real gap from a design choice. The declarations in `tck_test.go` are the source
of truth; below is the shape of each.

### flipt-native (`TestFliptConformance`): 47 passed, 2 failed, 16 skipped (of 65)

Counts are true outcomes, not godog's summary line. Godog counts a scenario
skipped in the before-scenario hook as passed — its step-status enum has
`Passed` as the zero value, so a pickle whose steps never ran keeps it — which
is why its tally reads "65 scenarios (63 passed, 2 failed)" while 16 scenarios
were skipped. The tck's own skip report and the step tally (303 passed,
2 failed, 115 skipped of 420) are the reliable measures.

- **string-zero-flag** (mandatory falsy-value row): the empty string cannot be
  seeded as a Flipt variant key, so the flag resolves to `"zero"`.
- **A targeting rule that does not match reports the default**
  (`@standard-reasons`): the rule's non-matching row reports `STATIC` instead of
  `DEFAULT`, because Flipt's evaluation response does not distinguish a rule
  that matched nothing from a rule-less flag. The matching `TARGETING_MATCH`
  row passes.

`@variants` and `@events` are currently withheld, so the variant-name rows and
the two events scenarios skip. Disabled flags are fully green:
`disabled-boolean-flag` resolves `false` with reason `DISABLED` against the
fixed backend (see Testbed).

Lossless numeric coercion and large integers are declared and green:
`integral-float-flag` (`10.0`) narrows to `10` without loss while `0.5` as an
integer is still `TYPE_MISMATCH`, and `huge-integer-flag` (2⁵³−1) resolves
exactly — the integer transform parses integers first, so exact values never
round-trip through a float64 that cannot represent them.

Declared capabilities: `@object`, `@targeting`, `@disabled-flags`,
`@standard-reasons`, `@numeric-coercion`, `@large-integers`.

### OFREP (`TestOFREPConformance`): 31 passed, 18 failed, 16 skipped (of 65)

The OFREP provider layers its own mapping over Flipt's OFREP endpoint. On top of
the seed model it inherits, the mapping itself has known gaps, all declared as
deviations in `tck_test.go`:

- Flipt's raw evaluation reasons are passed through unremapped
  (`@standard-reasons`): statically-resolved flags report `DEFAULT`
  (`string-flag`) or `TARGETING_MATCH` (`boolean-flag`) instead of `STATIC`,
  and the integer/float static rows additionally carry their string-value
  `TYPE_MISMATCH` so their reason is `ERROR`. The matching, non-matching and
  disabled `@targeting`/`@disabled-flags` reason rows pass.
- Flipt's OFREP endpoint reports variant keys as JSON strings; the OFREP
  provider does not coerce them to the numeric accessors, so Integer/Float
  requests return `TYPE_MISMATCH` with the code default.
- Lossless `@numeric-coercion` halves fail for the same reason (`10.0` as
  Integer, `10` as Float); the lossy half (`0.5` as Integer → `TYPE_MISMATCH`)
  passes.
- `huge-integer-flag` (`@large-integers`) fails for the same string-value
  reason.
- `object-flag` resolves to its default variant key rather than the attachment.

`@variants` and `@events` are currently withheld, so the variant-name rows and
the two events scenarios skip. Disabled flags are fully green against the
fixed backend (see Testbed).

Declared capabilities: `@object`, `@targeting`, `@disabled-flags`,
`@standard-reasons`, `@numeric-coercion`, `@large-integers`.