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
// # Which control path to use
//
// [BackendControl] is the single seam between the scenarios and whatever
// manipulates the backend. Providers with a real backend drive it over the HTTP
// control API defined in the spec submodule, at
// spec/specification/assets/provider-tck/openapi/control-api.yaml; providers
// with no backend at all may use an in-process implementation such as
// [InProcessControl]. The distinction matters and is not a matter of taste —
// see the documentation on [BackendControl].
//
// [Appendix F]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md
package tck
