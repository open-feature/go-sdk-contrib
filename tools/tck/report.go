package tck

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"
)

// ReportDirEnv names the directory a conformance report is written to.
//
// It is an environment variable rather than a Config field so that emitting a
// report is a property of the run and not of the code: CI sets it, a developer
// running the suite locally does not, and no adopter has to change a line to
// publish one. A suite writes <dir>/<name>.json, so several suites in one test
// binary — flagd's RPC and in-process resolvers, say — each produce their own
// file without colliding.
//
// Unset means no report, which is the default and is not an error.
const ReportDirEnv = "PROVIDER_TCK_REPORT_DIR"

// reportSchemaVersion is the major version of the report schema this emitter
// produces. See specification/assets/provider-tck/report/.
const reportSchemaVersion = "1"

// Outcome is the result of one scenario, or of one capability.
//
// There are four rather than two because "did not run" is not one thing.
// A capability the provider chose not to declare is a different statement from
// one the language makes impossible — @strict-numeric-typing cannot hold in a
// language with no integer type — and reporting both as "not declared" would
// show a whole language as missing something none of its providers can have.
type Outcome string

const (
	OutcomePassed        Outcome = "passed"
	OutcomeFailed        Outcome = "failed"
	OutcomeNotDeclared   Outcome = "not-declared"
	OutcomeNotApplicable Outcome = "not-applicable"
)

// Report is one run of the suite against one provider in one configuration.
//
// The field names and shape are fixed by the schema in the specification
// repository; this type is deliberately a transcription of it rather than a
// convenient Go representation, because the point of the format is that four
// languages emit the same thing.
type Report struct {
	SchemaVersion string                      `json:"schemaVersion"`
	Provider      ReportProvider              `json:"provider"`
	SDK           ReportSDK                   `json:"sdk"`
	TCK           ReportTCK                   `json:"tck"`
	Backend       *ReportBackend              `json:"backend,omitempty"`
	Capabilities  map[string]ReportCapability `json:"capabilities"`
	Scenarios     []ReportScenario            `json:"scenarios"`
}

type ReportProvider struct {
	Name          string `json:"name"`
	Version       string `json:"version,omitempty"`
	Language      string `json:"language"`
	Configuration string `json:"configuration,omitempty"`
}

type ReportSDK struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ReportTCK struct {
	Implementation string `json:"implementation"`
	Version        string `json:"version"`
	SpecRevision   string `json:"specRevision"`
	SpecRelease    string `json:"specRelease,omitempty"`
	AssetsTree     string `json:"assetsTree,omitempty"`
}

type ReportBackend struct {
	Description string `json:"description,omitempty"`
	ControlAPI  string `json:"controlApi,omitempty"`
}

type ReportCapability struct {
	State  Outcome `json:"state"`
	Reason string  `json:"reason,omitempty"`
}

type ReportScenario struct {
	Feature    string   `json:"feature"`
	Name       string   `json:"name"`
	Tags       []string `json:"tags,omitempty"`
	Outcome    Outcome  `json:"outcome"`
	Reason     string   `json:"reason,omitempty"`
	DurationMs float64  `json:"durationMs,omitempty"`
}

// scenarioRecord is what the runner accumulates as scenarios execute.
type scenarioRecord struct {
	feature  string
	name     string
	tags     []string
	outcome  Outcome
	reason   string
	duration time.Duration
}

// buildReport assembles the report from what the run observed.
//
// The per-scenario list is the load-bearing part. Appendix F requires that a
// scenario skipped for an undeclared capability is never reported as passed,
// and godog's own summary does exactly that — it counts capability skips in its
// passed tally, so the headline number says something false. Emitting the
// outcome of every scenario individually makes the rule checkable by a consumer
// instead of dependent on each runner's summary being trustworthy.
func (r *runner) buildReport() Report {
	providerName := r.observedProviderName()
	r.mu.Lock()
	records := make([]scenarioRecord, len(r.records))
	copy(records, r.records)
	r.mu.Unlock()

	sort.Slice(records, func(i, j int) bool {
		if records[i].feature != records[j].feature {
			return records[i].feature < records[j].feature
		}
		return records[i].name < records[j].name
	})

	scenarios := make([]ReportScenario, 0, len(records))
	// failed counts, per capability, the scenarios gating on it that failed, so a
	// capability is reported as passed only when everything gating on it passed and
	// a failure can say how much failed.
	failed := map[Capability]int{}
	// exercised counts the scenarios gating on each capability at all. A capability
	// no scenario carries cannot have been demonstrated, and reporting it as passed
	// would claim conformance the suite never tested -- which is the same vacuous
	// green the capability vocabulary exists to prevent.
	exercised := map[Capability]int{}

	for _, rec := range records {
		scenarios = append(scenarios, ReportScenario{
			Feature:    rec.feature,
			Name:       rec.name,
			Tags:       rec.tags,
			Outcome:    rec.outcome,
			Reason:     rec.reason,
			DurationMs: float64(rec.duration.Microseconds()) / 1000.0,
		})
		for _, tag := range rec.tags {
			capability, gates := CapabilityForTag(tag)
			if !gates {
				continue
			}
			exercised[capability]++
			if rec.outcome == OutcomeFailed {
				failed[capability]++
			}
		}
	}

	capabilities := map[string]ReportCapability{}
	for _, capability := range AllCapabilities() {
		switch {
		case !r.caps.has(capability):
			capabilities[capability.Tag()] = ReportCapability{
				State: OutcomeNotDeclared,
				Reason: fmt.Sprintf(
					"not declared by this provider's configuration; the %s scenarios were skipped and did not contribute to this result",
					capability.Tag()),
			}
		case exercised[capability] == 0:
			// Declared, but no scenario in the suite gates on it. Saying nothing is
			// the only honest answer: the suite asked no question, so it has none
			// to report. Claiming passed would be a green result for an untested
			// claim, which is precisely what this suite exists to make impossible.
		case failed[capability] > 0:
			capabilities[capability.Tag()] = ReportCapability{
				State: OutcomeFailed,
				Reason: fmt.Sprintf(
					"%d of %d scenarios carrying %s failed; the per-scenario results say which, and why",
					failed[capability], exercised[capability], capability.Tag()),
			}
		default:
			capabilities[capability.Tag()] = ReportCapability{State: OutcomePassed}
		}
	}

	return Report{
		SchemaVersion: reportSchemaVersion,
		Provider: ReportProvider{
			Name:          providerName,
			Language:      "go",
			Configuration: r.cfg.Name,
		},
		SDK: ReportSDK{Name: goSDKModule, Version: sdkVersion()},
		TCK: ReportTCK{
			Implementation: tckImplementation,
			Version:        tckVersion(),
			SpecRevision:   SpecRevision,
			AssetsTree:     AssetsTree,
		},
		Backend: &ReportBackend{
			Description: r.cfg.Control.Description(),
			ControlAPI:  controlAPIOf(r.cfg.Control),
		},
		Capabilities: capabilities,
		Scenarios:    scenarios,
	}
}

// observedProviderName is what the provider called itself, falling back to the
// suite name when no scenario ever registered one -- which happens when every
// scenario was skipped, and is worth reporting as the suite name rather than as
// an empty string the schema would reject.
func (r *runner) observedProviderName() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.providerName != "" {
		return r.providerName
	}
	return r.cfg.Name
}

// controlAPIReporter is implemented by a BackendControl that knows which kind
// of control the schema should record.
//
// It is an optional interface rather than a method on BackendControl because
// adding a method would break every existing implementation for the sake of one
// string, and a control that does not implement it simply omits the field.
type controlAPIReporter interface {
	// ControlAPI reports "http" for the normative control API or "in-process"
	// for the narrow allowance made for providers with no backend.
	ControlAPI() string
}

func controlAPIOf(control BackendControl) string {
	if reporter, ok := control.(controlAPIReporter); ok {
		return reporter.ControlAPI()
	}
	return ""
}

const (
	goSDKModule       = "github.com/open-feature/go-sdk"
	tckImplementation = "go-sdk-contrib/tools/provider-tck"
	tckModule         = "github.com/open-feature/go-sdk-contrib/tools/provider-tck"
)

// writeReport emits the report if ReportDirEnv is set.
//
// A failure to write is reported as a test failure rather than logged and
// ignored. CI that asked for a report and silently did not get one is how a
// publishing pipeline ends up serving a stale result forever.
func (r *runner) writeReport() {
	dir := strings.TrimSpace(os.Getenv(ReportDirEnv))
	if dir == "" {
		return
	}

	report := r.buildReport()

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		r.t.Errorf("provider-tck [%s]: could not encode the conformance report: %v", r.cfg.Name, err)
		return
	}
	data = append(data, '\n')

	if err := os.MkdirAll(dir, 0o755); err != nil {
		r.t.Errorf("provider-tck [%s]: could not create the report directory %s: %v", r.cfg.Name, dir, err)
		return
	}

	path := filepath.Join(dir, reportFileName(r.cfg.Name))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		r.t.Errorf("provider-tck [%s]: could not write the conformance report to %s: %v", r.cfg.Name, path, err)
		return
	}

	r.t.Logf("provider-tck [%s]: conformance report written to %s", r.cfg.Name, path)
}

// reportFileName turns a suite name into a filename.
//
// Suite names are chosen to read well in failure messages rather than to be
// path-safe, so anything that is not obviously safe becomes a hyphen. Without
// this a suite named "flagd/rpc" would silently write outside the directory it
// was given.
func reportFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	cleaned := strings.Trim(b.String(), "-.")
	if cleaned == "" {
		cleaned = "report"
	}
	return cleaned + ".json"
}

// sdkVersion reports the go-sdk version this binary was built against.
//
// Read from the build info rather than declared, because a declared version is
// a second place to be wrong: the report would keep claiming 1.14.0 after a
// dependency bump moved the actual code underneath it.
func sdkVersion() string {
	return moduleVersion(goSDKModule)
}

func tckVersion() string {
	return moduleVersion(tckModule)
}

func moduleVersion(path string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	if info.Main.Path == path && info.Main.Version != "" {
		return info.Main.Version
	}
	for _, dep := range info.Deps {
		if dep == nil || dep.Path != path {
			continue
		}
		// A replace directive means the code being run is not the version the
		// requirement names, and saying so is more useful than either version
		// alone.
		if dep.Replace != nil && dep.Replace.Version != "" {
			return dep.Replace.Version
		}
		if dep.Version != "" {
			return dep.Version
		}
	}
	return "unknown"
}

// featureName turns a Gherkin document's URI into the bare feature name the
// schema asks for: "errors", not "features/errors.feature".
func featureName(uri string) string {
	base := filepath.Base(filepath.ToSlash(uri))
	return strings.TrimSuffix(base, filepath.Ext(base))
}
