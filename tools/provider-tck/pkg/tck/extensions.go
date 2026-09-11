package tck

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/open-feature/go-sdk/openfeature"
)

// WHY EXTENSIONS ARE CONFIGURED RATHER THAN DISCOVERED
//
// An adopter with provider-specific behaviour — flagd's fractional targeting,
// say — needs its own scenarios to run inside the TCK's backend lifecycle: the
// same per-scenario provider, the same PrepareScenario reset, the same event
// plumbing. Running them in a parallel harness means reimplementing all of
// that, and the two harnesses then drift.
//
// The other TCK implementations discover extensions by convention: Java scans
// the classpath for glue packages, Python picks up a conftest.py beside the
// feature files. Go has no runtime scanning — a step definition is a function
// that has to be called, and a feature file has to be named — so extension here
// is explicit configuration. The goal is that it stays small: two fields, and
// no test infrastructure of the adopter's own.
//
// See Config.ExtensionFeatures and Config.ExtensionSteps.

// ClientFromContext returns the OpenFeature client for the provider under test,
// for use from a step definition registered through Config.ExtensionSteps.
//
// This is the seam that makes an extension step worth running inside the TCK
// rather than beside it. The provider is registered under a suite-scoped domain
// the adopter never sees, so without this an extension step could observe the
// lifecycle the TCK set up but not use it, and would end up registering a
// second provider of its own — which is the parallel harness the extension
// mechanism exists to avoid.
//
// It reports an error before the scenario has registered a provider, which in
// the canonical vocabulary means before a "Given a stable provider" step. Put
// one in a Background, as the canonical features do.
func ClientFromContext(ctx context.Context) (*openfeature.Client, error) {
	state, err := stateFrom(ctx)
	if err != nil {
		return nil, err
	}
	return state.requireClient()
}

// extensionsRoot is the path prefix an adopter's feature files appear under
// while the suite runs.
//
// It is a prefix rather than the adopter's own directory name because the
// canonical assets and the extension filesystem are two roots presented to
// godog as one, and because it partitions the URI space: everything the run
// executed under gherkin/ is canonical, everything under extensions/ is
// not. That partition is what the coverage check and a reader of the Cucumber
// Messages stream both key on, so it is a contract rather than a detail.
const extensionsRoot = "extensions"

// featureSet is what godog is pointed at: one filesystem and the paths within
// it to parse.
//
// Both come from one place because they have to agree, and because passing
// canonical and extension features through two different godog mechanisms is
// not an option — see featureSources.
type featureSet struct {
	fsys  fs.FS
	paths []string
}

// featureSources assembles the canonical assets and the adopter's extensions
// into a single filesystem for godog.
//
// With no extensions configured this returns the embedded assets and the
// canonical path unchanged, so a Config without them behaves exactly as it did
// before this existed.
//
// godog 0.15.1 also has Options.FeatureContents, which would let extension
// features be read and handed over as bytes, and that is the obvious shortcut.
// It is not safe here: godog parses FeatureContents and Paths in two separate
// calls, each with its own freshly seeded id generator, so the AST node, pickle
// and pickle-step ids of the two sets collide. This package keys a scenario's
// skip reason and its Messages test case by pickle id, and the report's whole
// claim to identify a Scenario Outline row rests on those ids being unique
// within a run. One filesystem and one parse keeps them so.
func (c *Config) featureSources() (featureSet, error) {
	canonical := featureSet{fsys: assets, paths: []string{featuresPath}}

	if c.ExtensionFeatures == nil {
		return canonical, nil
	}

	found, err := countFeatureFiles(c.ExtensionFeatures)
	if err != nil {
		return featureSet{}, fmt.Errorf("ExtensionFeatures could not be read: %w", err)
	}
	if found == 0 {
		return featureSet{}, errors.New(
			"ExtensionFeatures contains no .feature file: a filesystem that turns out to be empty " +
				"is the way extension wiring fails silently, so it is refused rather than run as " +
				"the canonical suite alone. Check the directory passed to os.DirFS, or leave the " +
				"field unset")
	}

	return featureSet{
		fsys: overlayFS{roots: []overlayRoot{
			// The canonical root keeps its own path so the URIs in the results
			// are the ones every previous run produced.
			{prefix: featuresPath, fsys: assets, base: featuresPath},
			{prefix: extensionsRoot, fsys: c.ExtensionFeatures, base: "."},
		}},
		paths: []string{featuresPath, extensionsRoot},
	}, nil
}

// countFeatureFiles reports how many .feature files a filesystem holds.
func countFeatureFiles(fsys fs.FS) (int, error) {
	count := 0
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(name, ".feature") {
			count++
		}
		return nil
	})
	return count, err
}

// overlayFS presents several filesystems as one, each mounted at its own path
// prefix.
//
// It exists because godog takes a single fs.FS and a list of paths within it,
// while the suite has two sources: the canonical assets embedded in this
// package and whatever the adopter supplies. Mounting them side by side means
// one parse, one id space, and no chance of an extension file shadowing a
// canonical one.
//
// Only the operations godog performs are implemented, which is Open on a path
// under a mount plus the ReadDir and Stat that fs.WalkDir needs. There is no
// synthetic root directory: nothing walks from ".", because the paths handed to
// godog are the mount points themselves.
type overlayFS struct {
	roots []overlayRoot
}

// overlayRoot mounts fsys at prefix, serving it from base within fsys.
//
// base exists so a mount can be an identity: the canonical assets are embedded
// under gherkin and are served at gherkin, which keeps the URIs in the results
// stable. An adopter's filesystem is rooted at "." and served at extensions.
type overlayRoot struct {
	prefix string
	fsys   fs.FS
	base   string
}

// resolve maps an overlay path onto the filesystem and path that serve it.
func (o overlayFS) resolve(name string) (fs.FS, string, error) {
	if !fs.ValidPath(name) {
		return nil, "", fs.ErrInvalid
	}
	for _, root := range o.roots {
		switch {
		case name == root.prefix:
			return root.fsys, root.base, nil
		case strings.HasPrefix(name, root.prefix+"/"):
			rest := name[len(root.prefix)+1:]
			if root.base == "." {
				return root.fsys, rest, nil
			}
			return root.fsys, path.Join(root.base, rest), nil
		}
	}
	return nil, "", fs.ErrNotExist
}

func (o overlayFS) Open(name string) (fs.File, error) {
	fsys, target, err := o.resolve(name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return fsys.Open(target)
}

func (o overlayFS) ReadDir(name string) ([]fs.DirEntry, error) {
	fsys, target, err := o.resolve(name)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}
	return fs.ReadDir(fsys, target)
}

func (o overlayFS) Stat(name string) (fs.FileInfo, error) {
	fsys, target, err := o.resolve(name)
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}
	return fs.Stat(fsys, target)
}

// isCanonicalFeature reports whether a feature URI from the run is one of the
// canonical assets rather than an adopter's extension.
//
// The URI is the path godog parsed the file from, so this is the partition
// established by featureSources and nothing more subtle.
func isCanonicalFeature(uri string) bool {
	return strings.HasPrefix(uri, featuresPath+"/")
}
