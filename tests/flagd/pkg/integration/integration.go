package integration

import (
	"context"
	"errors"
	"time"

	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk/pkg/openfeature"
)

var providerOptions []flagd.ProviderOption

// ctxStorageKey is the key used to pass test data across context.Context
type ctxStorageKey struct{}

// ctxClientKey is the key used to pass the openfeature client across context.Context
type ctxClientKey struct{}

func aFlagdProviderIsSet(ctx context.Context) (context.Context, error) {
	pOptions := []flagd.ProviderOption{flagd.WithPort(8013)}
	pOptions = append(pOptions, providerOptions...)
	provider := flagd.NewProvider(pOptions...)
	readyChan := make(chan interface{})

	err := openfeature.SetProvider(provider)
	if err != nil {
		return nil, err
	}

	callBack := func(details openfeature.EventDetails) {
		// emit readiness
		close(readyChan)
	}

	openfeature.AddHandler(openfeature.ProviderReady, &callBack)

	client := openfeature.NewClient("evaluation tests")

	select {
	case <-readyChan:
	case <-time.After(500 * time.Millisecond):
		return ctx, errors.New("provider not ready after 500 milliseconds")
	}

	return context.WithValue(ctx, ctxClientKey{}, client), nil
}
