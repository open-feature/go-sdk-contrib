package tck

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sort"

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
//
// It is typed as the interface and left assignable so that the integrity
// checks over it can be driven against a filesystem a test controls -- the
// embedded one cannot be mutated, and a digest check that is never shown a
// changed asset is a check nobody has evidence for. Same reasoning as
// inexpressibleCapabilities in capability.go. Nothing outside a test reassigns
// it.
var assets fs.FS = spectck.FS

// featuresPath is the directory within assets holding the canonical Gherkin.
const featuresPath = "gherkin"

// assetsModulePath is the module the conformance artifacts come from, named so
// that a failure message can tell an adopter which dependency to inspect.
const assetsModulePath = "github.com/open-feature/spec/specification/assets/provider-tck"

// canonicalAssetsDigest is the SHA-256 of every embedded conformance artifact,
// and it is this suite's revision check.
//
// Why a digest rather than the pin: the obvious check is to compare the pin in
// go.mod against the revision this package says it conforms to, and that check
// has two holes which Appendix F requires closed.
//
// The first is that it cannot always be performed. It has to read go.mod, and
// there is no guarantee the file is there: not in an unpacked distribution, not
// in a linked worktree, and not in a consumer whose working directory is its
// own module rather than this one. A check that reports nothing when it cannot
// do its job is absent exactly where it is needed -- which is also where stale
// assets are most likely -- so skipping is not an option and failing on a
// missing go.mod would break every legitimate adopter. A digest over bytes that
// are embedded at compile time can always be computed, so the question does not
// arise.
//
// The second is that reading *this module's* go.mod answers the wrong question
// for an adopter. An adoption is a module of its own: it pins the assets itself,
// usually indirectly, and resolves them by module-version selection across its
// whole graph. So the assets an adoption's run actually parses need not be the
// ones this module's go.mod names, and the revision a conformance report
// publishes would then name a revision the run did not use -- a report that is
// structurally valid and semantically wrong. The digest is computed from the
// bytes the run is about to parse, in the process that parses them, so it
// cannot disagree with the run.
//
// That is why verifyCanonicalAssets is called from Run rather than from a test
// in this package: the guarantee has to hold where the scenarios execute, and a
// guarantee that holds only in this module's own test suite does not cover the
// run whose results are being published.
//
// Moving the pin changes this constant. TestTheCanonicalAssetsMatchTheirDigest
// prints the value to paste in, which is the one step a pin move needs.
const canonicalAssetsDigest = "47bd565367f60d1df83c211aa799d54c1955e17232423aee7f304004ea8df681"

// assetsDigest fingerprints the embedded conformance artifacts.
//
// The path and the length of each file go into the hash alongside its contents,
// so that renaming a file or moving bytes between two of them changes the
// digest. Paths are sorted, because fs.WalkDir's order is not part of its
// contract and a fingerprint that depended on it would be unstable for reasons
// that say nothing about the assets.
//
// The artifacts are normalised to LF where they are committed, and embedded
// bytes get no line-ending translation, so this value is the same on every
// platform.
func assetsDigest() (string, error) {
	var paths []string
	err := fs.WalkDir(assets, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			paths = append(paths, name)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking the embedded assets: %w", err)
	}
	if len(paths) == 0 {
		return "", errors.New("the embedded assets hold no file at all, so a digest over them " +
			"would be the digest of nothing and would match every empty set")
	}
	sort.Strings(paths)

	hash := sha256.New()
	for _, name := range paths {
		body, err := fs.ReadFile(assets, name)
		if err != nil {
			return "", fmt.Errorf("reading the embedded asset %s: %w", name, err)
		}
		fmt.Fprintf(hash, "%s\n%d\n", name, len(body))
		hash.Write(body)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// verifyCanonicalAssets fails when the artifacts compiled into this binary are
// not the ones this package was written against.
//
// Called from Run, so it is in force for every adoption rather than only for
// this module's tests. It cannot be skipped: there is no condition under which
// it declines to answer, which is the property the pin comparison lacked.
func verifyCanonicalAssets() error {
	got, err := assetsDigest()
	if err != nil {
		// Not a soft failure. An integrity check that cannot be performed is
		// reported as a failure rather than passed over, because the state it
		// cannot rule out is exactly the one it exists to catch.
		return fmt.Errorf(
			"the conformance assets could not be fingerprinted, so this run cannot say which "+
				"revision of the specification it answers: %w", err)
	}
	if got != canonicalAssetsDigest {
		return fmt.Errorf(
			"the conformance assets compiled into this binary are not the ones this suite was "+
				"written against.\n  expected digest: %s\n  actual digest:   %s\n"+
				"The scenarios, the canonical flag set or the control API document differ from the "+
				"pinned revision, so every number this run produces is about a different question "+
				"set than the one it would report. Two ways this happens: the pin in tools/tck's "+
				"go.mod moved without canonicalAssetsDigest in assets.go being updated with it -- "+
				"run TestTheCanonicalAssetsMatchTheirDigest, which prints the value to paste in -- "+
				"or this module's assets are being replaced or resolved to a different version by "+
				"the module doing the running, in which case `go list -m %s` from that module says "+
				"which one it got",
			canonicalAssetsDigest, got, assetsModulePath)
	}
	return nil
}

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
	b, err := fs.ReadFile(assets, "flags/canonical-flags.json")
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
	b, err := fs.ReadFile(assets, "openapi/control-api.yaml")
	if err != nil {
		panic("tck: control API spec missing from embedded assets: " + err.Error())
	}
	return b
}
