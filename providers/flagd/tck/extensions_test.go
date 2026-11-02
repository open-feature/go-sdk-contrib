//go:build tck

package tck

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/cucumber/godog"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/v2/testframework"
	"github.com/open-feature/go-sdk-contrib/tools/tck"
)

// Steps for flagd-testbed's tck-extensions scenarios that are about flagd
// rather than about providers in general. A scenario records provider options
// in its step context, and the TCK's own "a stable provider" step hands that
// context to the provider factory, which applies them.

const (
	envoyService       = "envoy"
	envoyRoutingPort   = 9211
	envoyForbiddenPort = 9212

	// The extension features name testbed files by the path a Java adoption
	// mounts the testbed at; this module checks it out here instead.
	featureTestbedDir = "test-harness/"
	testbedDir        = "../flagd-testbed/"
)

type scenarioOptionsKey struct{}

// scenarioOptions is what a scenario asked of the next provider it creates.
type scenarioOptions struct {
	options []flagd.ProviderOption

	// Resolved against the stack by the factory, which has the endpoint.
	targetURI string
	forbidden bool
}

func optionsFrom(ctx context.Context) scenarioOptions {
	if o, ok := ctx.Value(scenarioOptionsKey{}).(scenarioOptions); ok {
		return o
	}
	return scenarioOptions{}
}

func registerFlagdSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^an option "([^"]*)" of type "([^"]*)" with value "([^"]*)"$`, anOption)
	ctx.Step(`^a target URI "([^"]*)" routed through envoy$`, aTargetURIRoutedThroughEnvoy)
	ctx.Step(`^the provider connects to envoy's forbidden listener$`, theProviderConnectsToTheForbiddenListener)
}

// anOption sets a ProviderConfiguration field by the testbed's option name,
// the way the flagd e2e suites do.
func anOption(ctx context.Context, option, valueType, value string) (context.Context, error) {
	if option == "certPath" {
		value = strings.Replace(value, featureTestbedDir, testbedDir, 1)
	}

	field := testframework.ToFieldName(option)
	if _, ok := reflect.TypeOf(flagd.ProviderConfiguration{}).FieldByName(field); !ok {
		return ctx, fmt.Errorf("flagd.ProviderConfiguration has no field %q for option %q", field, option)
	}

	converter := testframework.NewValueConverter()
	set := func(p *flagd.ProviderConfiguration) {
		target := reflect.ValueOf(p).Elem().FieldByName(field)
		target.Set(converter.ConvertToReflectValue(valueType, value, target.Type()))
	}

	o := optionsFrom(ctx)
	o.options = append(append([]flagd.ProviderOption{}, o.options...), set)
	return context.WithValue(ctx, scenarioOptionsKey{}, o), nil
}

func aTargetURIRoutedThroughEnvoy(ctx context.Context, targetURI string) (context.Context, error) {
	o := optionsFrom(ctx)
	o.targetURI = targetURI
	return context.WithValue(ctx, scenarioOptionsKey{}, o), nil
}

func theProviderConnectsToTheForbiddenListener(ctx context.Context) (context.Context, error) {
	o := optionsFrom(ctx)
	o.forbidden = true
	return context.WithValue(ctx, scenarioOptionsKey{}, o), nil
}

// stackEndpoint holds the running stack's endpoint for the unavailable-provider
// factory, which the TCK does not give one. Every stable provider records it,
// and canonical scenarios build several before any extension scenario runs.
var stackEndpoint struct {
	sync.Mutex
	endpoint *tck.BackendEndpoint
}

func recordEndpoint(endpoint tck.BackendEndpoint) {
	stackEndpoint.Lock()
	defer stackEndpoint.Unlock()
	stackEndpoint.endpoint = &endpoint
}

func recordedEndpoint() (tck.BackendEndpoint, error) {
	stackEndpoint.Lock()
	defer stackEndpoint.Unlock()
	if stackEndpoint.endpoint == nil {
		return tck.BackendEndpoint{}, errors.New("no stable provider has run yet, so the stack's endpoint is unknown")
	}
	return *stackEndpoint.endpoint, nil
}

// scenarioProviderOptions resolves what the scenario recorded against the
// stack. A target URI or the forbidden listener replaces host and port; the
// port of a target-URI provider stays closed so it cannot fall back to them.
func scenarioProviderOptions(o scenarioOptions, endpoint tck.BackendEndpoint) ([]flagd.ProviderOption, error) {
	options := o.options

	if o.targetURI != "" || o.forbidden {
		host, err := endpoint.ServiceHost(envoyService)
		if err != nil {
			return nil, err
		}

		if o.forbidden {
			port, err := endpoint.ServicePort(envoyService, envoyForbiddenPort)
			if err != nil {
				return nil, err
			}
			options = append(options, flagd.WithHost(host), flagd.WithPort(uint16(port)))
		}

		if o.targetURI != "" {
			port, err := endpoint.ServicePort(envoyService, envoyRoutingPort)
			if err != nil {
				return nil, err
			}
			uri := strings.Replace(o.targetURI, "localhost", host, 1)
			uri = strings.Replace(uri, "<port>", strconv.Itoa(port), 1)
			options = append(options, flagd.WithTargetUri(uri), flagd.WithPort(unavailablePort))
		}
	}

	return options, nil
}
