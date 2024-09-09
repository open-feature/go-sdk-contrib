package ofrep

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/open-feature/go-sdk/openfeature"

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
			t.Errorf("expected header %s, but got %s", "HEADER", h)
		}

		if v != "VALUE" {
			t.Errorf("expected value %s, but got %s", "VALUE", v)
		}
	})

	t.Run("validate bearer token", func(t *testing.T) {
		c := outbound.Configuration{}

		WithBearerToken("TOKEN")(&c)

		h, v := c.Callbacks[0]()

		if h != "Authorization" {
			t.Errorf("expected header %s, but got %s", "Authorization", h)
		}

		if v != "Bearer TOKEN" {
			t.Errorf("expected value %s, but got %s", "Bearer TOKEN", v)
		}
	})

	t.Run("validate api auth key", func(t *testing.T) {
		c := outbound.Configuration{}

		WithApiKeyAuth("TOKEN")(&c)

		h, v := c.Callbacks[0]()

		if h != "X-API-Key" {
			t.Errorf("expected header %s, but got %s", "X-API-Key", h)
		}

		if v != "TOKEN" {
			t.Errorf("expected value %s, but got %s", "TOKEN", v)
		}
	})
}

func TestWiringE2E(t *testing.T) {
	// mock server with mocked response
	server := httptest.NewServer(
		mockHandler{
			response: "{\"value\":true,\"key\":\"my-flag\",\"reason\":\"STATIC\",\"variant\":\"true\",\"metadata\":{}}",
			t:        t,
		},
	)
	t.Cleanup(server.Close)

	// custom client with reduced timeout
	customClient := &http.Client{
		Timeout: 1 * time.Second,
	}

	provider := NewProvider(server.URL, WithClient(customClient))
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
