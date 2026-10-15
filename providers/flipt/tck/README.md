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

Docker must be available. The suite starts `testdata/tck/docker-compose.yaml` and drives it through the control API on `:8080`. Flipt serves on `:8888`. The image builds on `docker.flipt.io/flipt/flipt:v2-nightly`.

## What Flipt supports

The flipt-native suite declares object values, targeting, disabled flags, lossless numeric coercion, and large integers. It withholds standard reasons, variants, events, and lifecycle.

## How Flipt reasons differ

Flipt returns the same `DEFAULT` reason for a flag with no rules and for a rule that matched nothing. The provider reports `STATIC` for both cases. It therefore cannot claim the standard `DEFAULT` reason.

Flipt returns `UNKNOWN` when a flag has no default variant. The provider then returns the caller default with reason `DEFAULT`. The flag `no-default-flag` covers this path.

`testdata/features/flipt-reason.feature` mirrors the skipped standard reason rows with the actual Flipt behavior. Update these copies by hand when the spec pin in `tools/tck/go.mod` changes `reason.feature`.
