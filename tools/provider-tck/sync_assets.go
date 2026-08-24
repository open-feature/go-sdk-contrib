//go:build ignore

// Command sync_assets copies the conformance artifacts from the open-feature/spec
// submodule into the package, where they are committed and embedded.
//
// # Why the copies are committed
//
// They have to be, for Go specifically. A Go module is distributed as a zip built
// from the VCS tree, and a git submodule appears there only as a gitlink — its
// files are not in the tree. Embedding straight from the submodule works locally
// and in CI and then fails for anyone who runs `go get`, with the module shipping
// an empty directory and the package failing to compile. That is verifiable
// without publishing anything:
//
//	$ git archive HEAD tools/provider-tck | tar -t | grep 'pkg/tck/spec'
//	tools/provider-tck/pkg/tck/spec/
//	$ git archive HEAD tools/provider-tck | tar -t | grep -c '\.feature$'
//	0
//
// So the submodule stays the source of truth and records the spec revision, and
// the generated copies are what ships. The other languages do not need this: a
// wheel, a JAR and an npm package are all built from a working tree where the
// submodule is present.
//
// # Keeping them honest
//
// A committed generated file is a lie waiting to happen, so CI runs this and
// fails if the result differs from what is checked in. Editing the copies by hand
// is therefore caught, which is the property that matters — the whole point of a
// conformance suite is that its definition cannot drift.
//
// Usage:
//
//	go run ./sync_assets.go          # from tools/provider-tck
//	make provider-tck-assets         # from the repository root
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	specRoot = "pkg/tck/spec/specification/assets/provider-tck"
	destRoot = "pkg/tck/assets"
)

// copies maps a directory in the spec repository to its destination in the
// package. The layout is preserved rather than renamed, so a reader comparing the
// two trees sees the same shape.
var copies = []string{"gherkin", "flags", "openapi"}

func main() {
	if _, err := os.Stat(specRoot); err != nil {
		fail("the spec submodule is not checked out at %s: %v\n\n"+
			"Run: git submodule update --init tools/provider-tck/pkg/tck/spec", specRoot, err)
	}

	if err := os.RemoveAll(destRoot); err != nil {
		fail("could not clear %s: %v", destRoot, err)
	}

	total := 0
	for _, dir := range copies {
		n, err := copyTree(filepath.Join(specRoot, dir), filepath.Join(destRoot, dir))
		if err != nil {
			fail("copying %s: %v", dir, err)
		}
		total += n
	}

	fmt.Printf("synced %d conformance artifacts from %s\n", total, specRoot)
}

func copyTree(src, dst string) (int, error) {
	count := 0
	err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		// The spec repo carries a README and a .gitattributes alongside the
		// artifacts; neither is an artifact, and embedding them would put
		// unrelated files in every consumer's binary.
		if filepath.Ext(path) != ".feature" && filepath.Ext(path) != ".json" && filepath.Ext(path) != ".yaml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "sync_assets: "+format+"\n", args...)
	os.Exit(1)
}
