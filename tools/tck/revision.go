package tck

// SpecRevision is the revision of the open-feature/spec conformance assets this
// suite ran against.
//
// It is the module version pinned in go.mod, not a bare commit id, because the
// assets now arrive as a Go module rather than through a submodule and a
// generated copy. A pseudo-version names the commit anyway -- the trailing
// twelve characters are its prefix -- and once the assets are released under a
// tag this reads as that tag instead.
//
// It is written by hand. Nothing generates it any more, and nothing can: a
// library package's test binary carries no module build information at all
// (runtime/debug.ReadBuildInfo reports no dependencies from one, in workspace
// mode and in plain module mode alike), and the TCK always runs inside one --
// both its own self-tests and an adopter's TestConformance. So the value is
// duplicated from go.mod on purpose, and TestSpecRevisionIsRecorded fails if the
// two disagree.
const SpecRevision = "v0.0.0-20260912211427-ccdb88790bb4"

// SpecModulePath is the module the conformance assets come from.
//
// Exported so the revision guard can find the pin without restating the path,
// and so a consumer reading a report can tell which module a SpecRevision
// belongs to.
const SpecModulePath = "github.com/open-feature/spec/specification/assets/provider-tck"
