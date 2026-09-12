// Package tck implements the OpenFeature Provider Conformance Suite (TCK) for Go.
//
// The suite answers one question: does this provider map its backend onto the
// OpenFeature provider contract correctly? It is the Go implementation of
// [Appendix F] of the OpenFeature specification, and it runs the same Gherkin
// scenarios, against the same canonical flag set, that every other language's
// TCK runs. That shared basis is the whole point — "conformant" only means
// something if the question is identical everywhere.
//
// # What a provider author writes
//
// One test function and a Docker Compose file. Everything else — starting the
// stack, discovering its host ports, driving the backend's control API,
// registering the provider with the OpenFeature API, waiting for it to become
// ready, awaiting events, resetting the backend between scenarios, tearing down
// — belongs to the TCK. If you find yourself writing test infrastructure, that
// is a defect in this package rather than something for you to work around.
//
//	func TestMyProviderConformance(t *testing.T) {
//	    tck.Run(t,
//	        tck.WithName("my-provider"),
//	        tck.WithComposeFile("testdata/tck/docker-compose.yaml"),
//	        tck.WithBackendPorts(8013),
//	        tck.WithProviderFromEndpoint(func(_ context.Context, e tck.BackendEndpoint) (openfeature.FeatureProvider, error) {
//	            return myprovider.New(e.Host(), e.Port(8013)), nil
//	        }),
//	        tck.WithUnavailableProvider(func(context.Context) (openfeature.FeatureProvider, error) {
//	            return myprovider.New("localhost", 9999), nil
//	        }),
//	        tck.WithCapabilities(tck.Events, tck.Object),
//	    )
//	}
//
// See [WithComposeFile]. A provider with no backend to contain — in-memory,
// in-process — supplies its own control and builds its provider without an
// endpoint instead, through [WithControl] and [WithProvider].
//
// # Options rather than a struct
//
// Every setting is one [Option], so the suite can gain a capability without
// every adoption having to be edited, and so that a required setting is named in
// one place rather than being a zero value someone has to remember means
// "unset". A missing required option is reported by name before anything starts.
//
// # Capabilities
//
// Not every provider implements every optional part of the contract. Scenarios
// exercising an optional part carry a Gherkin tag, and a provider declares which
// of those it supports through [WithCapabilities]. A scenario whose tag was
// not declared is reported as skipped with the reason printed — never as
// passed. See [Capability].
//
// # Adding your own scenarios
//
// A provider with behaviour the specification does not describe — flagd's
// fractional targeting, a vendor's segment rules — can run scenarios of its own
// inside this suite rather than in a harness beside it, through
// [WithFeatures] and [WithSteps]:
//
//	tck.Run(t,
//	    // ... as above ...
//	    tck.WithFeatures(os.DirFS("testdata/tck-extensions")),
//	    tck.WithSteps(func(ctx *godog.ScenarioContext) {
//	        ctx.Step(`^the fractional bucket is "([^"]*)"$`, theBucketIs)
//	    }),
//	)
//
// Extension scenarios get the same provider registration, readiness wait and
// per-scenario backend reset the canonical ones get, and an extension step
// reaches the provider under test with [ClientFromContext]. They are
// distinguishable from canonical scenarios in the conformance report: a result
// whose feature URI starts with "gherkin/" is canonical, one under
// "extensions/" is the adopter's.
//
// Java and Python discover extensions by convention — a classpath scan, a
// conftest.py — because those languages can scan. Go cannot, so extension here
// is two options rather than none.
//
// # Which control path to use
//
// [BackendControl] is the single seam between the scenarios and whatever
// manipulates the backend. Providers with a real backend drive it over the HTTP
// control API defined in the specification, at
// specification/assets/provider-tck/openapi/control-api.yaml; providers
// with no backend at all may use an in-process implementation such as
// [InProcessControl]. The distinction matters and is not a matter of taste —
// see the documentation on [BackendControl] — and a control states which of the
// two it is, through [ControlAPI], rather than leaving it to be inferred.
//
// # One module, container harness included
//
// The Compose harness lives in this package rather than in a second module
// beside it, so an adopter has one import path and one version to track. The
// cost is visible and worth naming: testcontainers-go and docker/compose are
// ordinary dependencies of package tck, so a provider with no container to
// start — in-memory, environment-variable, file-based — still takes those
// go.sum entries and the ~40 transitive pins behind them. It compiles nothing
// it does not import, and this is a test-only module that no application binary
// links, so the cost is confined to `go test` of an adopting module.
//
// The alternative was weighed and declined. A tools/tck/compose module would
// need its own version and release-please entry, and a home for
// [BackendEndpoint] that both modules can see — which is this package, so the
// second module would import the first and the split would buy nothing but a
// second coordinate to publish. After the Compose decision, containerised
// adopters are the overwhelming majority, and Java keeps testcontainers in its
// tck artifact for the same reason.
//
// [Appendix F]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md
package tck
