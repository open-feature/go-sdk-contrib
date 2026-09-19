// Package tck implements the OpenFeature Provider Conformance Suite (TCK) for Go.
//
// The suite answers one question: does this provider map its backend onto the
// OpenFeature provider contract correctly? It is the Go implementation of
// [Appendix F] of the OpenFeature specification, and it runs the same Gherkin
// scenarios, against the same canonical flag set, that every other language's
// TCK runs. That shared basis is the whole point — "conformant" only means
// something if the question is identical everywhere. Appendix F is the contract;
// this package documents the Go binding of it, and [the README] the rest.
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
//	        tck.WithComposeFile("testdata/docker-compose.yaml"),
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
// Put it in a module of its own, at providers/<name>/tck, beside the provider's
// e2e suite rather than inside it, and give the file a //go:build tck
// constraint. The directory is what selects the suite — `make tck` runs those
// modules and `make e2e` runs the others — and the tag is what keeps it out of
// an untagged build, so `make test` and a bare `go test ./...` start no
// containers. A conformance run and an e2e run mean different things by a red
// result, which is why they are separate.
//
// A provider with no backend to contain — in-memory, in-process — supplies its
// own control and builds its provider without an endpoint instead, through
// [WithControl] and [WithProvider]. See [BackendControl] for which path fits,
// which is not a matter of taste.
//
// Every setting is one [Option]; [Capability] is how a provider declares the
// optional parts of the contract it supports; [WithFeatures] and [WithSteps]
// run scenarios of your own inside this suite.
//
// [Appendix F]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md
// [the README]: https://github.com/open-feature/go-sdk-contrib/blob/main/tools/tck/README.md
package tck
