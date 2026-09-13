// Package tck runs the OpenFeature Provider Conformance Suite against the flagd
// provider, in both of its resolver modes.
//
// It is a sibling of providers/flagd/e2e rather than a package inside it. The
// two suites answer different questions and mean different things by a red
// result: the e2e suites test flagd against flagd's own harness and are expected
// green, while this one tests the provider against the OpenFeature provider
// contract and fails scenarios by design wherever the adoption declares a known
// deviation. It is also what the Makefile selects on — `make tck` runs the
// modules named tck, `make e2e` runs the others — so nothing here depends on
// what its tests are called.
//
// It is a module of its own for the same reason providers/flagd/e2e is: the
// suite needs testcontainers, a Docker Compose client and tools/tck, none of
// which belong in the dependency graph of an application that imports the
// provider, and the replace directives it needs do not belong in a module that
// is released.
//
// The suite itself is behind the tck build tag, so `make e2e` builds it under
// -tags=tck without running it and `make tck` runs it. The tag is not what
// separates those two targets — the module path is — it is what keeps the suite
// out of an untagged build, so `make test` and a bare `go test ./...` start no
// containers. This file carries no build tag so that the package still has a
// buildable file when the suite is absent; guard_test.go carries none either, so
// that the guard on this module still running the suite runs in the default
// build.
package tck
