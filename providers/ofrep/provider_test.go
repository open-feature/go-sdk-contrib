package ofrep

import (
	"context"
	"fmt"
	"github.com/open-feature/go-sdk/openfeature"
	"net/http"
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/outbound"
)

func TestConfigurations(t *testing.T) {
	t.Run("validate header provider", func(t *testing.T) {
		c := outbound.Configuration{}

		WithHeaderProvider(func() (key string, value string) {
			return "HEADER", "VALUE"
		})(&c)

		h, v := c.Callbacks[0]()

		if h != "HEADER" {
			t.Errorf(fmt.Sprintf("expected header %s, but got %s", "HEADER", h))
		}

		if v != "VALUE" {
			t.Errorf(fmt.Sprintf("expected value %s, but got %s", "VALUE", v))
		}
	})

	t.Run("validate bearer token", func(t *testing.T) {
		c := outbound.Configuration{}

		WithBearerToken("TOKEN")(&c)

		h, v := c.Callbacks[0]()

		if h != "Authorization" {
			t.Errorf(fmt.Sprintf("expected header %s, but got %s", "Authorization", h))
		}

		if v != "Bearer TOKEN" {
			t.Errorf(fmt.Sprintf("expected value %s, but got %s", "Bearer TOKEN", v))
		}
	})

	t.Run("validate api auth key", func(t *testing.T) {
		c := outbound.Configuration{}

		WithApiKeyAuth("TOKEN")(&c)

		h, v := c.Callbacks[0]()

		if h != "X-API-Key" {
			t.Errorf(fmt.Sprintf("expected header %s, but got %s", "X-API-Key", h))
		}

		if v != "TOKEN" {
			t.Errorf(fmt.Sprintf("expected value %s, but got %s", "TOKEN", v))
		}
	})
}

func TestWiringE2E(t *testing.T) {
	// mock server with mocked response
	host := "localhost:18182"

	server := http.Server{
		Addr: host,
		Handler: mockHandler{
			response: "{\"value\":true,\"key\":\"my-flag\",\"reason\":\"STATIC\",\"variant\":\"true\",\"metadata\":{}}",
			t:        t,
		},
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			t.Logf("error starting mock server: %v", err)
			return
		}
	}()

	// time for server to be ready
	<-time.After(3 * time.Second)

	// custom client with reduced timeout
	customClient := &http.Client{
		Timeout: 1 * time.Second,
	}

	provider := NewProvider(fmt.Sprintf("http://%s", host), WithClient(customClient))
	booleanEvaluation := provider.BooleanEvaluation(context.Background(), "flag", false, nil)

	if booleanEvaluation.Value != true {
		t.Errorf("expected %v, but got %v", true, booleanEvaluation.Value)
	}

	if booleanEvaluation.Variant != "true" {
		t.Errorf("expected %v, but got %v", "true", booleanEvaluation.Variant)
	}

	if booleanEvaluation.Reason != openfeature.StaticReason {
		t.Errorf("expected %v, but got %v", openfeature.StaticReason, booleanEvaluation.Reason)
	}

	if booleanEvaluation.Error() != nil {
		t.Errorf("expected no errors, but got %v", booleanEvaluation.Error())
	}

	err := server.Shutdown(context.Background())
	if err != nil {
		t.Errorf("error shuttting down mock server: %v", err)
	}
}

type mockHandler struct {
	response string
	t        *testing.T
}

func (r mockHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)
	_, err := resp.Write([]byte(r.response))
	if err != nil {
		r.t.Logf("error wriging bytes: %v", err)
	}
}
