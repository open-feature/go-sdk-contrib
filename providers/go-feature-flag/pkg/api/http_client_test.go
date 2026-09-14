package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultHTTPClient(t *testing.T) {
	t.Run("returns a non-nil client", func(t *testing.T) {
		got := api.DefaultHTTPClient()
		require.NotNil(t, got)
	})

	t.Run("client properties", func(t *testing.T) {
		tests := []struct {
			name  string
			check func(t *testing.T, client *http.Client)
		}{
			{
				name: "timeout is 10 seconds",
				check: func(t *testing.T, client *http.Client) {
					t.Helper()
					assert.Equal(t, 10000*time.Millisecond, client.Timeout)
				},
			},
			{
				name: "transport is not nil",
				check: func(t *testing.T, client *http.Client) {
					t.Helper()
					require.NotNil(t, client.Transport)
				},
			},
			{
				name: "transport is an *http.Transport",
				check: func(t *testing.T, client *http.Client) {
					t.Helper()
					_, ok := client.Transport.(*http.Transport)
					assert.True(t, ok, "expected Transport to be *http.Transport")
				},
			},
			{
				name: "TLS handshake timeout is 10 seconds",
				check: func(t *testing.T, client *http.Client) {
					t.Helper()
					transport, ok := client.Transport.(*http.Transport)
					require.True(t, ok)
					assert.Equal(t, 10000*time.Millisecond, transport.TLSHandshakeTimeout)
				},
			},
			{
				name: "idle connection timeout is 90 seconds",
				check: func(t *testing.T, client *http.Client) {
					t.Helper()
					transport, ok := client.Transport.(*http.Transport)
					require.True(t, ok)
					assert.Equal(t, 90*time.Second, transport.IdleConnTimeout)
				},
			},
		}

		client := api.DefaultHTTPClient()
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tt.check(t, client)
			})
		}
	})

	t.Run("each call returns a distinct client instance", func(t *testing.T) {
		first := api.DefaultHTTPClient()
		second := api.DefaultHTTPClient()
		assert.NotSame(t, first, second)
	})
}
