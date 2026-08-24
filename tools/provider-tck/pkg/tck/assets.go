package tck

import "embed"

// NOTE ON THE SOURCE OF TRUTH
//
// The files under assets/ are NOT owned by this repository. They are copies of
// the language-agnostic conformance artifacts defined in open-feature/spec
// under specification/assets/provider-tck/:
//
//	assets/features/*.feature            the canonical scenarios
//	assets/flags/canonical-flags.json    the flag set those scenarios assume
//	assets/openapi/control-api.yaml      the HTTP surface a backend under test exposes
//
// They are vendored here so that adopting this TCK never requires a consumer to
// check out a git submodule of their own. A follow-up change will source them
// from the spec repository as a submodule and copy them in at build time, the
// way providers/flagd already consumes open-feature/spec. Until then, changes
// belong in open-feature/spec first and are copied here — editing them locally
// forks the definition of conformance, which is the one thing this suite exists
// to prevent.
//
// See https://github.com/open-feature/spec/issues/417.

// assets carries the conformance artifacts into the compiled package, so the
// feature files are available to godog with no filesystem layout requirement on
// the consuming module. godog reads them through godog.Options.FS, which
// accepts an embed.FS directly.
//
//go:embed assets/features/*.feature
//go:embed assets/flags/canonical-flags.json
//go:embed assets/openapi/control-api.yaml
var assets embed.FS

// featuresPath is the directory within assets holding the canonical Gherkin.
const featuresPath = "assets/features"

// CanonicalFlags returns the canonical flag set as raw JSON, in the flagd
// flag-definition format.
//
// This is the flag set every scenario assumes, and a backend under test must
// serve an equivalent one. The format is not what matters — the keys, types,
// variant names and resolved values are. Seed them however your backend seeds
// flags.
//
// It is exposed so that an adopting provider can seed a backend directly from
// the canonical definition rather than transcribing it, transcription being the
// usual way the two drift apart.
func CanonicalFlags() []byte {
	b, err := assets.ReadFile("assets/flags/canonical-flags.json")
	if err != nil {
		// Unreachable: the file is embedded at compile time, so a failure here
		// means the embed directive above and the file layout disagree.
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
	b, err := assets.ReadFile("assets/openapi/control-api.yaml")
	if err != nil {
		panic("provider-tck: control API spec missing from embedded assets: " + err.Error())
	}
	return b
}
