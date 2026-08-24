package tck

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"

	"github.com/cucumber/gherkin/go/v26"
	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages/go/v21"
)

// WHY THIS FILE EXISTS
//
// A row of a Scenario Outline is identified by its parameters. Every row shares
// the outline's name, so a report that identifies a scenario by feature and name
// gives eleven identical entries for the eleven rows of the type-mismatch matrix
// in errors.feature. If one row fails and ten pass, that report cannot say which
// failed, and a consumer keying on feature and name keeps whichever row it read
// last.
//
// godog hands a hook a *godog.Scenario, which is a messages.Pickle. A pickle is
// the already-expanded scenario: its step text has the parameters substituted
// into it, and the row they came from is gone. What survives is AstNodeIds,
// whose last entry, for a pickle compiled from an outline, is the id of the
// Examples TableRow the pickle was expanded from.
//
// So the row is recovered by parsing the same feature files a second time and
// indexing every Examples TableRow by that id.

// exampleRow is one row of an Examples table.
type exampleRow struct {
	// values are the cells keyed by column header, verbatim as strings. Gherkin
	// has no types, so "1" stays "1": coercing it to a number would be this
	// implementation inventing a fact the feature file did not state, and the
	// four language implementations would each invent a different one.
	values map[string]string
	// order is where the row sits among all the rows of its scenario, counting
	// across every Examples block the scenario has. It is carried so the report
	// can list the rows in the order the table declares them rather than in
	// whatever order sorting the parameter values happens to produce.
	order int
}

// exampleIndex resolves a pickle to the Examples row it was expanded from.
type exampleIndex struct {
	// rows is keyed by the id of the TableRow AST node.
	rows map[string]exampleRow
	// outlines names every scenario that is an outline, keyed by feature URI and
	// scenario name. It exists so a failure to resolve a row can be told apart
	// from a scenario that legitimately has none, and reported rather than
	// silently emitting a report with the ambiguity this field exists to remove.
	outlines map[string]bool
}

var (
	exampleIndexOnce sync.Once
	exampleIndexVal  *exampleIndex
	exampleIndexErr  error
)

// scenarioExamples returns the index, building it once per process.
func scenarioExamples() (*exampleIndex, error) {
	exampleIndexOnce.Do(func() {
		exampleIndexVal, exampleIndexErr = buildExampleIndex()
	})
	return exampleIndexVal, exampleIndexErr
}

// buildExampleIndex parses the embedded feature files and indexes their
// Examples rows by AST node id.
//
// The id has to agree with the one godog will report, and godog's ids are not
// intrinsic to a document: they come from a counter (messages.Incrementing) that
// godog creates once per run and shares across every file it parses, so an id
// depends on how many nodes were numbered before it. Reproducing them therefore
// means reproducing godog's whole parse — the same files, in the same order,
// with pickle compilation in between, because compiling pickles draws from the
// same counter.
//
// That is a coupling to godog's internals, and it is a deliberate one: the
// alternative is to fork the pickle compiler. It is not left to be trusted.
// A pickle that comes from an outline and does not resolve is reported as a
// failure by the run, so a godog release that renumbers nodes breaks the build
// loudly instead of quietly emitting reports with the ambiguity removed again.
func buildExampleIndex() (*exampleIndex, error) {
	index := &exampleIndex{
		rows:     map[string]exampleRow{},
		outlines: map[string]bool{},
	}

	paths, err := featureFilePaths()
	if err != nil {
		return nil, err
	}

	newID := (&messages.Incrementing{}).NewId
	for _, path := range paths {
		file, err := assets.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening the embedded feature file %s: %w", path, err)
		}
		document, err := gherkin.ParseGherkinDocumentForLanguage(file, gherkin.DefaultDialect, newID)
		closeErr := file.Close()
		if err != nil {
			return nil, fmt.Errorf("parsing the embedded feature file %s: %w", path, err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("closing the embedded feature file %s: %w", path, closeErr)
		}

		document.Uri = path
		index.collectDocument(document)

		// The result is discarded; the side effect is the point. Compiling the
		// pickles advances the id counter exactly as godog's parse advances it,
		// so the next document's AST nodes are numbered the way godog numbers
		// them.
		_ = gherkin.Pickles(*document, path, newID)
	}

	return index, nil
}

// featureFilePaths lists the embedded feature files in the order godog walks
// them, which is the lexical order fs.WalkDir yields.
func featureFilePaths() ([]string, error) {
	var paths []string
	err := fs.WalkDir(assets, featuresPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".feature") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing the embedded feature files: %w", err)
	}
	// WalkDir already yields lexical order; sorting says so rather than relying
	// on it, since the ids only line up if this order matches godog's.
	sort.Strings(paths)
	return paths, nil
}

func (index *exampleIndex) collectDocument(document *messages.GherkinDocument) {
	if document == nil || document.Feature == nil {
		return
	}
	for _, child := range document.Feature.Children {
		if child == nil {
			continue
		}
		if child.Scenario != nil {
			index.collectScenario(document.Uri, child.Scenario)
		}
		if child.Rule == nil {
			continue
		}
		for _, ruleChild := range child.Rule.Children {
			if ruleChild != nil && ruleChild.Scenario != nil {
				index.collectScenario(document.Uri, ruleChild.Scenario)
			}
		}
	}
}

func (index *exampleIndex) collectScenario(uri string, scenario *messages.Scenario) {
	if len(scenario.Examples) == 0 {
		return
	}
	index.outlines[outlineKey(uri, scenario.Name)] = true

	order := 0
	for _, examples := range scenario.Examples {
		if examples == nil || examples.TableHeader == nil {
			continue
		}
		headers := make([]string, 0, len(examples.TableHeader.Cells))
		for _, cell := range examples.TableHeader.Cells {
			headers = append(headers, cell.Value)
		}

		for _, row := range examples.TableBody {
			if row == nil {
				continue
			}
			values := make(map[string]string, len(headers))
			for i, cell := range row.Cells {
				// A row with more cells than headers is malformed Gherkin the
				// parser would have rejected; guarding costs nothing and keeps a
				// future parser change from panicking here.
				if i >= len(headers) {
					break
				}
				values[headers[i]] = cell.Value
			}
			if len(values) > 0 {
				index.rows[row.Id] = exampleRow{values: values, order: order}
			}
			order++
		}
	}
}

// rowFor resolves the Examples row a pickle was expanded from.
//
// The last AST node id is the TableRow for an outline pickle and the Scenario
// node for an ordinary one, so a lookup that misses is the ordinary case rather
// than an error.
func (index *exampleIndex) rowFor(sc *godog.Scenario) (exampleRow, bool) {
	if sc == nil || len(sc.AstNodeIds) == 0 {
		return exampleRow{}, false
	}
	row, ok := index.rows[sc.AstNodeIds[len(sc.AstNodeIds)-1]]
	return row, ok
}

// isOutline reports whether a scenario name in a feature belongs to a Scenario
// Outline, and therefore must carry an example.
func (index *exampleIndex) isOutline(uri, name string) bool {
	return index.outlines[outlineKey(uri, name)]
}

func outlineKey(uri, name string) string {
	return uri + "\n" + name
}
