// Package tck runs the OpenFeature Provider Conformance Suite against the flagd
// provider, in both of its resolver modes. See README.md.
//
// It is a module of its own and a sibling of providers/flagd/e2e rather than a
// package inside it: the two suites mean different things by a red result, and
// the conformance dependencies — testcontainers, a Compose client, tools/tck —
// do not belong in the dependency graph of anything that imports the provider.
//
// The suite itself is behind the tck build tag. The module path is what selects
// it (`make tck` runs it, `make e2e` builds it without running it); the tag is
// what keeps it out of an untagged build, so `make test` and a bare
// `go test ./...` start no containers. This file carries no build tag so that
// the package still has a buildable file when the suite is absent, and
// guard_test.go carries none either, so that the guard on this module still
// running the suite runs in the default build.
package tck
