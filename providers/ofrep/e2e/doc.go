// Package e2e runs the OpenFeature Provider Conformance Suite against the OFREP
// provider.
//
// It is a module of its own rather than a package inside providers/ofrep. The
// provider module requires exactly one thing today — the Go SDK — and the suite
// needs testcontainers, a Docker Compose client and the TCK, which would
// otherwise land in the dependency graph of every application that imports the
// provider. Keeping them apart also keeps the replace directives the suite needs
// out of a module that is actually released.
//
// The suite itself is behind the e2e build tag, matching the repository's
// `make e2e` target. This file carries no build tag so that the package still
// has a buildable file when that tag is absent.
package e2e
