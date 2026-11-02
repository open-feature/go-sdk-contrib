// Package tck runs the OpenFeature Provider Conformance Suite against the OFREP
// provider. See README.md.
//
// It is a module of its own rather than a package inside providers/ofrep, which
// requires exactly one thing today — the Go SDK — so that the conformance
// dependencies (testcontainers, a Compose client, tools/tck) stay out of the
// dependency graph of every application that imports the provider.
//
// The suite is behind the tck build tag, so `make test` and a bare
// `go test ./...` start no containers. This file and guard_test.go carry no tag
// on purpose: the package needs a buildable file when the suite is absent, and
// the guard has to run in the build the suite is absent from.
package tck
