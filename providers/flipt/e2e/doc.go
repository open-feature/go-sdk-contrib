// Package e2e runs the OpenFeature Provider Conformance Suite against the Flipt
// backend, measuring two providers on the same throwaway stack: the flipt
// provider (pkg/provider, driven over Flipt's gRPC/HTTP SDK) and the OFREP
// provider (providers/ofrep, driven over Flipt's OFREP endpoint).
//
// It is a module of its own rather than a package inside providers/flipt. The
// provider module requires exactly one thing today — the Go SDK — and the suite
// needs testcontainers, a Docker Compose client and the TCK, which would
// otherwise land in the dependency graph of every application that imports the
// provider. Keeping them apart also keeps the replace directives the suite needs
// out of a module that is actually released.
//
// The backend is built from this module's testdata/tck directory: a control
// server (PID 1) on the ghcr.io/flipt-io/flipt:v2 image that starts, stops and
// reseeds the bundled Flipt binary over HTTP, so the TCK can simulate outages
// and flag changes without ever restarting the container. See README.md for the
// seed model and the known deviations.
//
// The suites themselves are behind the e2e build tag, matching the repository's
// `make e2e` target. This file carries no build tag so that the package still
// has a buildable file when that tag is absent.
package e2e
