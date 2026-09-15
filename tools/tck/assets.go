package tck

import (
	spectck "github.com/open-feature/spec/specification/assets/provider-tck"
)

// NOTE ON THE SOURCE OF TRUTH
//
// The conformance artifacts are NOT owned by this repository. They live in
// open-feature/spec under specification/assets/provider-tck/, which is also the
// Go module imported above; its own README documents how Go consumes it, why a
// submodule would not do, and what a version bump means. So the revision this
// suite conforms to is the one pinned in go.mod, and nothing else -- there is no
// copy here to edit, and an edited copy would fork the definition of
// conformance. Changes belong in open-feature/spec; adopting them here means
// moving the pin.

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
// variant names and resolved values are. Read the assets module's README before
// seeding: it names the five properties of this set that a seeding step is most
// likely to break.
//
// It is exposed so that an adopting provider can seed a backend directly from
// the canonical definition rather than transcribing it, transcription being the
// usual way the two drift apart.
func CanonicalFlags() []byte {
	b, err := assets.ReadFile("flags/canonical-flags.json")
	if err != nil {
		// Unreachable: the file is embedded at compile time, so a failure here
		// means the spec module's embed directives and this path disagree.
		panic("tck: canonical flag set missing from embedded assets: " + err.Error())
	}
	return b
}

// ControlAPISpec returns the OpenAPI document describing the HTTP control API
// that a containerised backend under test must expose.
//
// It is the normative contract for providers with a real backend. Appendix F's
// "The control API" states the two invariants a testbed author is most likely to
// break — no container is stopped or restarted to simulate an outage, and no
// state-changing endpoint returns before the new state is being served — and
// why a suite must not paper over a backend that breaks the second.
//
// What is this suite's own rather than the contract's: POST /restart is optional
// and no shipped scenario reaches it. The disconnect/reconnect scenario is an
// unbounded outage, which this suite drives with /stop followed by /start.
// Implement /restart if you want a future caching scenario — which needs its
// flag-state preservation — testable against your backend.
func ControlAPISpec() []byte {
	b, err := assets.ReadFile("openapi/control-api.yaml")
	if err != nil {
		panic("tck: control API spec missing from embedded assets: " + err.Error())
	}
	return b
}
