// Package tck runs the OpenFeature Provider Conformance Suite against the OFREP
// provider.
//
// It is a module of its own rather than a package inside providers/ofrep. The
// provider module requires exactly one thing today — the Go SDK — and the suite
// needs testcontainers, a Docker Compose client and the TCK, which would
// otherwise land in the dependency graph of every application that imports the
// provider. Keeping them apart also keeps the replace directives the suite needs
// out of a module that is actually released.
//
// It sits at providers/ofrep/tck rather than providers/ofrep/e2e, which is where
// it used to be and which no longer exists: that module held nothing but this
// suite, and naming it e2e said it was a kind of e2e test. It is not. An e2e
// suite tests a provider against its backend's own harness and is expected
// green; this one tests the provider against the OpenFeature provider contract
// and fails scenarios by design wherever a known deviation is declared.
//
// The suite itself is behind the tck build tag, so `make e2e` builds it under
// -tags=tck without running it and `make tck` runs it, selecting it by directory
// rather than by test name. The tag is not what separates those two targets --
// the module path is -- it is what keeps the suite out of an untagged build, so
// `make test` and a bare `go test ./...` start no containers. This file carries
// no build tag so that the package still has a buildable file when the suite is
// absent; guard_test.go carries none either, so that the guard on this module
// still running the suite runs in the default build.
package tck
