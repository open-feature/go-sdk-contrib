# Flipt conformance (tck)

This suite tests two providers against a throwaway Flipt backend:

* the flipt-native provider in `pkg/provider`
* the OFREP provider in `providers/ofrep`

## Run

You run the flipt-native suite from this directory:

```sh
go test -tags=tck ./...
```

You run the OFREP suite only on demand:

```sh
OFREP_TCK_RUN=1 go test -tags=tck -run TestOFREPConformance ./...
```

Docker must be available. The suite starts `testdata/tck/docker-compose.yaml` and drives it through the control API on `:8080`. Flipt serves on `:8888`. The Compose file pulls the pinned image `ghcr.io/flipt-io/openfeature-tck:0.1.0`; set `FLIPT_TESTBED_IMAGE` to test a locally built tag.

## What Flipt supports

The flipt-native suite declares object values, targeting, disabled flags, lossless numeric coercion, and large integers. It withholds standard reasons, variants, events, and lifecycle.

## How Flipt reasons differ

Flipt returns the same `DEFAULT` reason for a flag with no rules and for a rule that matched nothing. The provider reports `STATIC` for both cases. It therefore cannot claim the standard `DEFAULT` reason.

Flipt returns `UNKNOWN` when a flag has no default variant. The provider then returns the caller default with reason `DEFAULT`. The flag `no-default-flag` covers this path.

The `spec` module in `flipt-io/openfeature-tck` holds
`extensions/flipt-reason.feature`. It mirrors the skipped standard reason rows
with the actual Flipt behavior. Update that file by hand when the spec pin in
`tools/tck/go.mod` changes `reason.feature`. The Java adoption keeps its own
copy and needs the same update.

The module uses a local `replace` for
`github.com/flipt-io/openfeature-tck/spec` until the shared repository is
published and tagged.
