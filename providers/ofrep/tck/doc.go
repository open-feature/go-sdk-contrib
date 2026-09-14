// Package tck runs the OpenFeature Provider Conformance Suite against the OFREP
// provider. See README.md.
//
// It is a module of its own rather than a package inside providers/ofrep: that
// module requires exactly one thing today — the Go SDK — and the conformance
// dependencies (testcontainers, a Compose client, tools/tck) would otherwise
// land in the dependency graph of every application that imports the provider.
//
// It sits at providers/ofrep/tck rather than providers/ofrep/e2e, which is where
// it used to be and which no longer exists: that module held nothing but this
// suite, and naming it e2e said it was a kind of e2e test. It is not — the two
// mean different things by a red result.
//
// The suite itself is behind the tck build tag. The module path is what selects
// it (`make tck` runs it, `make e2e` builds it without running it); the tag is
// what keeps it out of an untagged build, so `make test` and a bare
// `go test ./...` start no containers. This file carries no build tag so that
// the package still has a buildable file when the suite is absent, and
// guard_test.go carries none either, so that the guard on this module still
// running the suite runs in the default build.
package tck
