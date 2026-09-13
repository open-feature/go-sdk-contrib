package e2e

// This file has no build tag on purpose. It guards the repository's build
// configuration rather than the provider, it needs neither Docker nor
// -tags=e2e, and it is worth nothing if it only runs in the build it is
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

// tckFilter is the test-name pattern the Makefile splits on. `make tck` is
// `go test -run 'Conformance'` and `make e2e` is the same sweep with `-skip`
// in its place, so the two targets are the two halves of one filter and every
// test in this package runs in exactly one of them.
//
// That split is a test-name filter rather than a build tag so that both targets
// keep compiling this package against tools/tck -- a conformance suite that has
// quietly stopped building against its own harness is a worse failure than one
// that runs and fails. The cost of choosing a name filter is that a rename can
// move a suite from one target to the other in silence: `make e2e` would start
// pulling Docker images on every pull request and `make tck` would stop running
// the suite, and neither would say anything. This test is that cost paid off.
const tckFilter = "Conformance"

// tckImportPath is the harness. A test that reaches tck.Run, directly or
// through a helper in this package, is a conformance suite; nothing else in
// this package is one.
const tckImportPath = "github.com/open-feature/go-sdk-contrib/tools/tck"

// TestEveryTckSuiteIsSelectedByTheMakeTargetFilter fails unless the tests that
// run the conformance suite are exactly the tests the Makefile's filter picks
// out -- in both directions. A suite whose name loses the pattern stops being
// run by `make tck` and starts being run by `make e2e`; a test that gains the
// pattern without running the suite ends up in the conformance step, where a
// red result would be reported as a conformance failure it is not.
//
// Note that this test must not itself be named for the pattern, and is not.
func TestEveryTckSuiteIsSelectedByTheMakeTargetFilter(t *testing.T) {
	tests, runsSuite, err := scanPackage(".")
	if err != nil {
		t.Fatalf("reading this package's own sources: %v", err)
	}

	var suites, selected []string
	for _, name := range tests {
		if runsSuite[name] {
			suites = append(suites, name)
		}
		if strings.Contains(name, tckFilter) {
			selected = append(selected, name)
		}
	}
	sort.Strings(suites)
	sort.Strings(selected)

	// Without this the whole test passes vacuously the day the scan stops
	// recognising a call -- an import alias, a suite moved behind another
	// helper -- which is the one way a guard like this fails silently.
	if len(suites) == 0 {
		t.Fatalf("found no test calling tck.Run in this package; either the conformance suites are "+
			"gone or %s can no longer recognise them, and in both cases `make tck` is running nothing",
			"conformance_naming_test.go")
	}

	for _, name := range suites {
		if !strings.Contains(name, tckFilter) {
			t.Errorf("%s runs the conformance suite but its name does not contain %q, so `make tck` "+
				"will not run it and `make e2e` will: rename it", name, tckFilter)
		}
	}
	for _, name := range selected {
		if !runsSuite[name] {
			t.Errorf("%s does not run the conformance suite but its name contains %q, so `make tck` "+
				"will run it and report any failure as a conformance failure: rename it", name, tckFilter)
		}
	}

	t.Logf("conformance suites in this package: %s", strings.Join(suites, ", "))
}

// scanPackage parses every _test.go file in dir and returns the names of its
// test functions, plus the set of package-level functions that reach tck.Run.
// Build tags are irrelevant to it, which is the point: it reads the e2e-tagged
// suites from an untagged test.
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

	// A suite is usually reached through a helper -- runConformance here -- so
	// follow same-package calls to a fixed point rather than only looking for
	// tck.Run in the test function itself.
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
// alias, or "" if the file does not import it.
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
