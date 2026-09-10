package tck

import (
	"embed"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

// The extension mechanism is two fields and one overlay filesystem. These tests
// pin the filesystem, because godog reaches it through an interface that only
// forwards Open — see storage.FS — so a missing method is a runtime failure
// deep inside the parser rather than a compile error here.

func extensionFixture() fs.FS {
	return fstest.MapFS{
		"vendor.feature":        &fstest.MapFile{Data: []byte("Feature: vendor\n")},
		"nested/more.feature":   &fstest.MapFile{Data: []byte("Feature: nested\n")},
		"notes.md":              &fstest.MapFile{Data: []byte("not a feature\n")},
		"nested/also-notes.txt": &fstest.MapFile{Data: []byte("nor this\n")},
	}
}

func TestNoExtensionsUsesTheEmbeddedAssetsUnchanged(t *testing.T) {
	set, err := (&Config{}).featureSources()
	if err != nil {
		t.Fatalf("featureSources: %v", err)
	}

	if len(set.paths) != 1 || set.paths[0] != featuresPath {
		t.Errorf("paths = %v, want exactly [%s]", set.paths, featuresPath)
	}
	// Identity, not merely equivalence: handing godog the embedded assets
	// themselves is what makes "no extensions behaves exactly as before" a
	// property of the code rather than of a test.
	got, ok := set.fsys.(embed.FS)
	if !ok || got != assets {
		t.Errorf("a Config with no extensions handed godog a %T rather than the embedded assets",
			set.fsys)
	}
}

func TestExtensionFeaturesAreMountedBesideTheCanonicalOnes(t *testing.T) {
	set, err := (&Config{ExtensionFeatures: extensionFixture()}).featureSources()
	if err != nil {
		t.Fatalf("featureSources: %v", err)
	}

	if len(set.paths) != 2 || set.paths[0] != featuresPath || set.paths[1] != extensionsRoot {
		t.Fatalf("paths = %v, want [%s %s]", set.paths, featuresPath, extensionsRoot)
	}

	// Everything godog walks must be reachable through the overlay, from both
	// mount points, with the canonical paths unchanged.
	walked := map[string]bool{}
	for _, root := range set.paths {
		err := fs.WalkDir(set.fsys, root, func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() {
				walked[name] = true
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}

	for _, want := range []string{
		featuresPath + "/evaluation.feature",
		extensionsRoot + "/vendor.feature",
		extensionsRoot + "/nested/more.feature",
	} {
		if !walked[want] {
			t.Errorf("%s is not reachable through the overlay; walked %v", want, keys(walked))
		}
	}

	// Reading through the overlay must return the mounted file's bytes.
	data, err := fs.ReadFile(set.fsys, extensionsRoot+"/vendor.feature")
	if err != nil {
		t.Fatalf("reading an extension feature: %v", err)
	}
	if !strings.Contains(string(data), "Feature: vendor") {
		t.Errorf("the overlay served %q for the extension feature", data)
	}

	canonical, err := fs.ReadFile(set.fsys, featuresPath+"/evaluation.feature")
	if err != nil {
		t.Fatalf("reading a canonical feature through the overlay: %v", err)
	}
	embedded, err := fs.ReadFile(assets, featuresPath+"/evaluation.feature")
	if err != nil {
		t.Fatalf("reading a canonical feature from the assets: %v", err)
	}
	if string(canonical) != string(embedded) {
		t.Error("the overlay changed the bytes of a canonical feature file")
	}
}

// TestOverlayIsUsableThroughOpenAlone is the property godog actually depends
// on: it wraps the filesystem in a type that forwards Open and nothing else, so
// fs.WalkDir has to work through the directory handles Open returns.
func TestOverlayIsUsableThroughOpenAlone(t *testing.T) {
	set, err := (&Config{ExtensionFeatures: extensionFixture()}).featureSources()
	if err != nil {
		t.Fatalf("featureSources: %v", err)
	}

	opener := openOnlyFS{set.fsys}

	found := 0
	if err := fs.WalkDir(opener, extensionsRoot, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(name, ".feature") {
			found++
		}
		return nil
	}); err != nil {
		t.Fatalf("walking the overlay through Open alone: %v", err)
	}
	if found != 2 {
		t.Errorf("found %d extension features through Open alone, want 2", found)
	}
}

// openOnlyFS hides every optional filesystem interface, the way godog's
// storage.FS wrapper does.
type openOnlyFS struct{ inner fs.FS }

func (o openOnlyFS) Open(name string) (fs.File, error) { return o.inner.Open(name) }

func TestExtensionFilesystemWithoutFeaturesIsRefused(t *testing.T) {
	_, err := (&Config{ExtensionFeatures: fstest.MapFS{
		"README.md": &fstest.MapFile{Data: []byte("nothing to run\n")},
	}}).featureSources()

	if err == nil {
		t.Fatal("an extension filesystem holding no .feature file was accepted")
	}
	if !strings.Contains(err.Error(), "no .feature file") {
		t.Errorf("the error does not say what is wrong with the filesystem: %v", err)
	}
}

// TestCanonicalAndExtensionURIsArePartitioned pins the fact a consumer of a
// report relies on.
func TestCanonicalAndExtensionURIsArePartitioned(t *testing.T) {
	found := 0
	err := fs.WalkDir(assets, featuresPath, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".feature") {
			return nil
		}
		found++
		if !isCanonicalFeature(name) {
			t.Errorf("the canonical feature %s is not recognised as canonical", name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the embedded assets: %v", err)
	}
	if found == 0 {
		t.Fatal("the embedded assets hold no feature file")
	}
	if isCanonicalFeature(extensionsRoot + "/vendor.feature") {
		t.Error("an extension feature is recognised as canonical")
	}
}

// TestAnExtensionCannotReachTheCanonicalPrefix is the shadowing constraint at
// the level it is enforced.
//
// The Java TCK's failure was that a same-named file in a second classpath root
// replaced the canonical one and the suite went green on the adopter's version.
// Here the two roots are mounted at disjoint prefixes, so the property to pin is
// that nothing an adopter supplies is reachable under the canonical prefix —
// not even a filesystem deliberately shaped to look like the embedded assets.
func TestAnExtensionCannotReachTheCanonicalPrefix(t *testing.T) {
	// An extension filesystem doing its worst: the canonical layout, verbatim,
	// with different contents.
	hostile := fstest.MapFS{
		"evaluation.feature": &fstest.MapFile{Data: []byte("Feature: not the canonical one\n")},
		featuresPath + "/evaluation.feature": &fstest.MapFile{
			Data: []byte("Feature: not the canonical one either\n"),
		},
	}

	set, err := (&Config{ExtensionFeatures: hostile}).featureSources()
	if err != nil {
		t.Fatalf("featureSources: %v", err)
	}

	// Every canonical path still resolves to the embedded bytes.
	err = fs.WalkDir(assets, featuresPath, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		through, readErr := fs.ReadFile(set.fsys, name)
		if readErr != nil {
			t.Errorf("reading %s through the overlay: %v", name, readErr)
			return nil
		}
		embedded, readErr := fs.ReadFile(assets, name)
		if readErr != nil {
			return readErr
		}
		if string(through) != string(embedded) {
			t.Errorf("the extension filesystem displaced the canonical %s", name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the embedded assets: %v", err)
	}

	// The hostile copies are still present, under the extension prefix, where
	// they are additions rather than replacements.
	for _, name := range []string{
		extensionsRoot + "/evaluation.feature",
		extensionsRoot + "/" + featuresPath + "/evaluation.feature",
	} {
		data, readErr := fs.ReadFile(set.fsys, name)
		if readErr != nil {
			t.Errorf("the extension file %s is not reachable at all: %v", name, readErr)
			continue
		}
		if !strings.Contains(string(data), "not the canonical one") {
			t.Errorf("%s served %q", name, data)
		}
		if isCanonicalFeature(name) {
			t.Errorf("%s is classified as canonical", name)
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
