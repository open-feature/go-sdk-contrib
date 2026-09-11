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
// One test function and a [Config] literal. Everything else — registering the
// provider with the OpenFeature API, waiting for it to become ready, awaiting
// events, resetting the backend between scenarios, tearing down — belongs to
// the TCK. If you find yourself writing test infrastructure, that is a defect
// in this package rather than something for you to work around.
//
//	func TestMyProviderConformance(t *testing.T) {
//	    control := myBackendControl()
//	    tck.Run(t, tck.Config{
//	        Name:    "my-provider",
//	        Control: control,
//	        NewProvider: func(ctx context.Context) (openfeature.FeatureProvider, error) {
//	            return myprovider.New(control.Address()), nil
//	        },
//	        Capabilities: []tck.Capability{tck.Events, tck.Object},
//	    })
//	}
//
// # Capabilities
//
// Not every provider implements every optional part of the contract. Scenarios
// exercising an optional part carry a Gherkin tag, and a provider declares which
// of those it supports through [Config.Capabilities]. A scenario whose tag was
// not declared is reported as skipped with the reason printed — never as
// passed. See [Capability].
//
// # Adding your own scenarios
//
// A provider with behaviour the specification does not describe — flagd's
// fractional targeting, a vendor's segment rules — can run scenarios of its own
// inside this suite rather than in a harness beside it, through
// [Config.ExtensionFeatures] and [Config.ExtensionSteps]:
//
//	tck.Run(t, tck.Config{
//	    // ... as above ...
//	    ExtensionFeatures: os.DirFS("testdata/tck-extensions"),
//	    ExtensionSteps: func(ctx *godog.ScenarioContext) {
//	        ctx.Step(`^the fractional bucket is "([^"]*)"$`, theBucketIs)
//	    },
//	})
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
// is two fields rather than none.
//
// # Which control path to use
//
// [BackendControl] is the single seam between the scenarios and whatever
// manipulates the backend. Providers with a real backend drive it over the HTTP
// control API defined in the specification, at
// specification/assets/provider-tck/openapi/control-api.yaml; providers
// with no backend at all may use an in-process implementation such as
// [InProcessControl]. The distinction matters and is not a matter of taste —
// see the documentation on [BackendControl].
//
// [Appendix F]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md
package tck
