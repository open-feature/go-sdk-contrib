package ofrep

import (
	"fmt"
	"testing"

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
