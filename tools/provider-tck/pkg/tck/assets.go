package tck

import "embed"

// NOTE ON THE SOURCE OF TRUTH
//
// The conformance artifacts are NOT owned by this repository. They are the
// language-agnostic definitions kept in open-feature/spec under
// specification/assets/provider-tck/, and they reach this package through a git
// submodule of that repository checked out at spec/:
//
//	spec/specification/assets/provider-tck/gherkin/*.feature   the canonical scenarios
//	spec/specification/assets/provider-tck/flags/…             the flag set those scenarios assume
//	spec/specification/assets/provider-tck/openapi/…           the HTTP surface a backend under test exposes
//
// Nothing is copied. The embed directives below read the submodule's files
// directly, so the revision of the specification this suite conforms to is
// recorded by the submodule pin in the tree — one commit id, visible in
// `git submodule status`, updated by moving the pin and by nothing else.
//
// That is the whole point of the arrangement. A vendored copy can be edited in
// place, and an edited copy forks the definition of conformance, which is the
// one thing this suite exists to prevent. Changes belong in open-feature/spec;
// adopting them here means advancing the pin.
//
// See https://github.com/open-feature/spec/issues/417.

// assets carries the conformance artifacts into the compiled package, so the
// feature files are available to godog with no filesystem layout requirement on
// the consuming module. godog reads them through godog.Options.FS, which
// accepts an embed.FS directly.
//
// The names in the FS are the pattern paths verbatim — embed keys files by
// their path relative to this package directory — so an entry is read back as
// "spec/specification/assets/provider-tck/…", not by its base name.
//
//go:embed spec/specification/assets/provider-tck/gherkin/*.feature
//go:embed spec/specification/assets/provider-tck/flags/canonical-flags.json
//go:embed spec/specification/assets/provider-tck/openapi/control-api.yaml
var assets embed.FS

// assetsRoot is the directory within assets holding the conformance artifacts,
// which is the submodule's path plus the location of the artifacts inside it.
const assetsRoot = "spec/specification/assets/provider-tck"

// featuresPath is the directory within assets holding the canonical Gherkin.
const featuresPath = assetsRoot + "/gherkin"

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
	b, err := assets.ReadFile(assetsRoot + "/flags/canonical-flags.json")
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
	b, err := assets.ReadFile(assetsRoot + "/openapi/control-api.yaml")
	if err != nil {
		panic("provider-tck: control API spec missing from embedded assets: " + err.Error())
	}
	return b
}
