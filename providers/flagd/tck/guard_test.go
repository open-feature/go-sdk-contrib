package tck

// This file has no build tag on purpose. It guards the repository's build
// configuration rather than the provider, it needs neither Docker nor
// -tags=tck, and it is worth nothing if it only runs in the build it is
// protecting. `make test` runs it.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// tckImportPath is the harness. A test that reaches tck.Run, directly or
// through a helper in this package, is a conformance suite; nothing else in
// this module is one.
const tckImportPath = "github.com/open-feature/go-sdk-contrib/tools/tck"

// TestThisModuleRunsTheConformanceSuite fails unless something in this package
// reaches tck.Run.
//
// What it is not: this used to also assert that every such test was named so
// that the Makefile's `-run 'Conformance'` filter selected it, and that nothing
// else was. That half is gone with the filter. `make tck` now runs this module
// and `make e2e` runs the others, so which target a suite lands in follows from
// where its file is and a rename can no longer move it. A guard defending a
// convention nothing selects on is a rule with no consequence.
//
// What remains is the half a directory cannot check for itself. Selecting a
// module says which tests are *offered* to `make tck`; it cannot say that any
// of them still runs the suite. A conformance module whose tests have stopped
// calling tck.Run — an adoption gutted during a refactor, a helper renamed,
// tck.Run dropped for a hand-rolled loop — leaves `make tck` green by running
// nothing, and green-by-vacuum is the one failure mode of this arrangement that
// is invisible from the outside.
func TestThisModuleRunsTheConformanceSuite(t *testing.T) {
	tests, runsSuite, err := scanPackage(".")
	if err != nil {
		t.Fatalf("reading this package's own sources: %v", err)
	}

	var suites []string
	for _, name := range tests {
		if runsSuite[name] {
			suites = append(suites, name)
		}
	}
	sort.Strings(suites)

	if len(suites) == 0 {
		t.Fatal("found no test calling tck.Run in this package; either the conformance suites are " +
			"gone or this guard can no longer recognise them, and in both cases `make tck` selects " +
			"this module and finds nothing to run")
	}

	t.Logf("conformance suites in this module: %s", strings.Join(suites, ", "))
}

// scanPackage parses every _test.go file in dir and returns the names of its
// test functions, plus the set of package-level functions that reach tck.Run.
// Build tags are irrelevant to it, which is the point: it reads the tck-tagged
// suites from an untagged test, which is the only way a guard against a module
// that has stopped running the suite can run in the build the suite is absent
// from.
func scanPackage(dir string) (tests []string, runsSuite map[string]bool, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}

	fset := token.NewFileSet()
	calls := map[string]map[string]bool{}
	runsSuite = map[string]bool{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		local := localNameOf(file, tckImportPath)

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil {
				continue
			}
			name := fn.Name.Name
			if isTestFunc(fn) {
				tests = append(tests, name)
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch fun := call.Fun.(type) {
				case *ast.SelectorExpr:
					pkg, ok := fun.X.(*ast.Ident)
					if ok && local != "" && pkg.Name == local && fun.Sel.Name == "Run" {
						runsSuite[name] = true
					}
				case *ast.Ident:
					if calls[name] == nil {
						calls[name] = map[string]bool{}
					}
					calls[name][fun.Name] = true
				}
				return true
			})
		}
	}

	// This module's two suites factor the call into a helper rather than
	// calling tck.Run themselves, so follow same-package calls to a fixed point
	// rather than only looking inside the test function itself.
	for changed := true; changed; {
		changed = false
		for caller, callees := range calls {
			if runsSuite[caller] {
				continue
			}
			for callee := range callees {
				if runsSuite[callee] {
					runsSuite[caller] = true
					changed = true
					break
				}
			}
		}
	}

	sort.Strings(tests)
	return tests, runsSuite, nil
}

// localNameOf returns the name importPath is bound to in file, honouring an
// alias, or "" if the file does not import it. This package is itself called
// tck, so the harness is imported under that same name and referred to as
// tck.Run below; the package clause does not put the name in scope, and the
// import is what these selectors resolve to.
func localNameOf(file *ast.File, importPath string) string {
	for _, spec := range file.Imports {
		unquoted, err := strconv.Unquote(spec.Path.Value)
		if err != nil || unquoted != importPath {
			continue
		}
		if spec.Name != nil {
			return spec.Name.Name
		}
		return path.Base(unquoted)
	}
	return ""
}

// isTestFunc applies the testing package's own rule for what `go test` will
// run, so that this test and the test binary agree on the set of names.
func isTestFunc(fn *ast.FuncDecl) bool {
	name := fn.Name.Name
	if !strings.HasPrefix(name, "Test") {
		return false
	}
	if rest := name[len("Test"):]; rest != "" && rest[0] >= 'a' && rest[0] <= 'z' {
		return false
	}
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	star, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "testing" && sel.Sel.Name == "T"
}
