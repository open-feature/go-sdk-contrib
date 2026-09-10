package tck

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

// ReportDirEnv names the directory a conformance report is written to.
//
// It is an environment variable rather than a Config field so that emitting a
// report is a property of the run and not of the code: CI sets it, a developer
// running the suite locally does not, and no adopter has to change a line to
// publish one. A suite writes <dir>/<name>.json and <dir>/<name>.ndjson, so
// several suites in one test binary — flagd's RPC and in-process resolvers,
// say — each produce their own pair of files without colliding.
//
// Unset means no report, which is the default and is not an error.
const ReportDirEnv = "PROVIDER_TCK_REPORT_DIR"

// reportSchemaVersion is the major version of the report schema this emitter
// produces. See specification/assets/provider-tck/report/.
const reportSchemaVersion = "1"

// Report is one run of the suite against one provider in one configuration.
//
// It is an envelope. It identifies what was tested and what the provider
// claims, and it points at the results; it does not contain them. The results
// are Cucumber Messages, written alongside this document, because per-scenario
// outcomes, tags, Scenario Outline row identity and the executed feature source
// are all already specified there. Restating them here would create a second
// format to maintain and two places for the same fact to disagree.
//
// The field names and shape are fixed by the schema in the specification
// repository; this type is deliberately a transcription of it rather than a
// convenient Go representation, because the point of the format is that four
// languages emit the same thing.
type Report struct {
	SchemaVersion string            `json:"schemaVersion"`
	Provider      ReportProvider    `json:"provider"`
	SDK           ReportSDK         `json:"sdk"`
	TCK           ReportTCK         `json:"tck"`
	Backend       *ReportBackend    `json:"backend,omitempty"`
	Declaration   ReportDeclaration `json:"declaration"`
	Results       ReportResults     `json:"results"`
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
}

type ReportBackend struct {
	Description string `json:"description,omitempty"`
	ControlAPI  string `json:"controlApi,omitempty"`
}

// ReportDeclaration is the capability set the provider claims.
//
// This is an input to reading the results rather than a summary of them, which
// is why it cannot be derived from the results payload and has to be stated
// here. A skipped scenario in the payload says the question was not put to this
// provider; only the declaration says whether that is because the provider
// declines the capability. Given the declaration and a scenario's tags — both
// of which the payload carries — the reason for a skip follows without being
// transported per scenario.
type ReportDeclaration struct {
	// Declared is every capability this configuration declares, as Gherkin tags
	// including the leading at-sign. A scenario tagged with anything absent
	// from this list is expected to be skipped in the results.
	//
	// Never nil: a provider that declares nothing states an empty list, which
	// is a claim, whereas null would be silence.
	Declared []string `json:"declared"`

	// NotApplicable maps a capability that cannot hold for this provider, as
	// opposed to one merely undeclared, to the reason. Nothing populates it
	// yet: Config has no field for it, because no Go provider has needed to
	// distinguish the two. It is transcribed so that a consumer unmarshalling
	// this type sees the whole schema.
	NotApplicable map[string]string `json:"notApplicable,omitempty"`
}

// ReportResults says where the executed results live and in what format.
//
// Referenced rather than inlined because a Messages stream carries the feature
// sources and so is far larger than this envelope, and because a consumer
// deciding whether it cares about a report should not have to fetch a whole run
// to find out.
type ReportResults struct {
	Format string `json:"format"`
	// FormatVersion is the Cucumber Messages release these results were
	// produced against.
	//
	// Messages is versioned and implementations pin different releases -- this
	// one builds against messages/go/v21 while cucumber-jvm ships a
	// considerably later one -- and the message types differ between them.
	// Without recording it a consumer validating this stream has to guess which
	// schema to validate against, and guessing wrong is worse than not checking:
	// a later schema accepts messages this producer could not have emitted,
	// while an earlier one rejects messages that are perfectly valid.
	FormatVersion string `json:"formatVersion,omitempty"`
	// Location is a path relative to this document. It is a bare filename with
	// no separator in it, so the same envelope reads correctly whatever wrote
	// it.
	Location string `json:"location"`
	// Digest covers the results payload byte for byte, as sha256:<hex>, so a
	// consumer can tell that what it fetched is what this envelope describes.
	Digest string `json:"digest,omitempty"`
}

// buildReport assembles the envelope around a results payload already written.
func (r *runner) buildReport(location, digest string) Report {
	declared := r.caps.sorted()
	tags := make([]string, 0, len(declared))
	for _, capability := range declared {
		tags = append(tags, capability.Tag())
	}

	return Report{
		SchemaVersion: reportSchemaVersion,
		Provider: ReportProvider{
			Name:          r.observedProviderName(),
			Language:      "go",
			Configuration: r.cfg.Name,
		},
		SDK: ReportSDK{Name: goSDKModule, Version: sdkVersion()},
		TCK: ReportTCK{
			Implementation: tckImplementation,
			Version:        tckVersion(),
			SpecRevision:   SpecRevision,
		},
		Backend: &ReportBackend{
			Description: r.cfg.Control.Description(),
			ControlAPI:  controlAPIOf(r.cfg.Control),
		},
		Declaration: ReportDeclaration{Declared: tags},
		Results: ReportResults{
			Format:        resultsFormatCucumberMessages,
			FormatVersion: messagesProtocolVersion(),
			Location:      location,
			Digest:        digest,
		},
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

// controlAPIOf reports how the backend was driven, or "" when the control does
// not say.
//
// A control that does not implement the interface leaves the field out, which is
// non-breaking but silent -- and silence here is a small lie by omission: every
// control is either driving a real backend over HTTP or manipulating an
// in-process one, so there is no third case the empty value legitimately
// describes. Two reports of the same in-process provider, one of which says so
// and one of which does not, are harder to compare than either alone.
//
// The runner therefore says so where the adopter will see it, rather than
// quietly emitting a report with a hole in it. See runner.reportControlAPIGap.
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
// One run writes two files: the results payload as Cucumber Messages ndjson,
// and the envelope naming and digesting it. The payload is written first so the
// digest in the envelope is over bytes that exist.
//
// A failure to write is reported as a test failure rather than logged and
// ignored. CI that asked for a report and silently did not get one is how a
// publishing pipeline ends up serving a stale result forever.
func (r *runner) writeReport() {
	dir := strings.TrimSpace(os.Getenv(ReportDirEnv))
	if dir == "" {
		return
	}

	results := r.messagesBytes()
	if len(results) == 0 {
		r.t.Errorf("provider-tck [%s]: the Cucumber Messages formatter produced nothing, so there "+
			"are no results to report; an envelope pointing at an empty payload would be worse "+
			"than no report", r.cfg.Name)
		return
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		r.t.Errorf("provider-tck [%s]: could not create the report directory %s: %v", r.cfg.Name, dir, err)
		return
	}

	base := reportBaseName(r.cfg.Name)
	location := base + ".ndjson"

	resultsPath := filepath.Join(dir, location)
	if err := os.WriteFile(resultsPath, results, 0o644); err != nil {
		r.t.Errorf("provider-tck [%s]: could not write the results payload to %s: %v",
			r.cfg.Name, resultsPath, err)
		return
	}

	sum := sha256.Sum256(results)
	report := r.buildReport(location, "sha256:"+hex.EncodeToString(sum[:]))

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		r.t.Errorf("provider-tck [%s]: could not encode the conformance report: %v", r.cfg.Name, err)
		return
	}
	data = append(data, '\n')

	path := filepath.Join(dir, base+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		r.t.Errorf("provider-tck [%s]: could not write the conformance report to %s: %v",
			r.cfg.Name, path, err)
		return
	}

	r.t.Logf("provider-tck [%s]: conformance report written to %s, results to %s",
		r.cfg.Name, path, resultsPath)
}

// reportBaseName turns a suite name into the stem both report files share.
//
// Suite names are chosen to read well in failure messages rather than to be
// path-safe, so anything that is not obviously safe becomes a hyphen. Without
// this a suite named "flagd/rpc" would silently write outside the directory it
// was given.
func reportBaseName(name string) string {
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
	return cleaned
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
