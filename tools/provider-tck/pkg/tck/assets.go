package tck

import (
	spectck "github.com/open-feature/spec/specification/assets/provider-tck"
)

// NOTE ON THE SOURCE OF TRUTH
//
// The conformance artifacts are NOT owned by this repository. They are the
// language-agnostic definitions kept in open-feature/spec under
// specification/assets/provider-tck/, and that directory is also a Go module
// whose only content is an embed.FS of them. This package depends on it the
// way it depends on any other module, so the revision of the specification
// this suite conforms to is the one pinned in go.mod -- one version, verified
// against the Go checksum database on every build, advanced by `go get` and by
// nothing else.
//
// That is the whole point of the arrangement. A vendored copy can be edited in
// place, and an edited copy forks the definition of conformance, which is the
// one thing this suite exists to prevent. A module version cannot be edited:
// the same version always resolves to the same bytes, for every consumer.
// Changes belong in open-feature/spec; adopting them here means moving the pin.
//
// A git submodule would not do. Go modules ship as a zip of the VCS tree, in
// which a submodule is only a gitlink, so an embed from a submodule compiles
// in this repository and arrives empty for anyone running `go get`.
//
// See https://github.com/open-feature/spec/issues/417.

// assets carries the conformance artifacts, keyed by their path within the
// spec directory: gherkin/*.feature, flags/canonical-flags.json and
// openapi/control-api.yaml. godog reads the feature files through
// godog.Options.FS, which accepts an embed.FS directly.
var assets = spectck.FS

// featuresPath is the directory within assets holding the canonical Gherkin.
const featuresPath = "gherkin"

// CanonicalFlags returns the canonical flag set as raw JSON, in the flagd
// flag-definition format.
//
// This is the flag set every scenario assumes, and a backend under test must
// serve an equivalent one. The format is not what matters -- the keys, types,
// variant names and resolved values are. Seed them however your backend seeds
// flags.
//
// It is exposed so that an adopting provider can seed a backend directly from
// the canonical definition rather than transcribing it, transcription being the
// usual way the two drift apart.
func CanonicalFlags() []byte {
	b, err := assets.ReadFile("flags/canonical-flags.json")
	if err != nil {
		// Unreachable: the file is embedded at compile time, so a failure here
		// means the spec module's embed directives and this path disagree.
		panic("provider-tck: canonical flag set missing from embedded assets: " + err.Error())
	}
	return b
}

// ControlAPISpec returns the OpenAPI document describing the HTTP control API
// that a containerised backend under test must expose.
//
// It is the normative contract for providers with a real backend. Two of its
// requirements are easy to get wrong and worth reading before implementing a
// testbed: containers are never stopped or restarted to simulate an outage, and
// POST /start resets flag state while POST /restart preserves it.
func ControlAPISpec() []byte {
	b, err := assets.ReadFile("openapi/control-api.yaml")
	if err != nil {
		panic("provider-tck: control API spec missing from embedded assets: " + err.Error())
	}
	return b
}
