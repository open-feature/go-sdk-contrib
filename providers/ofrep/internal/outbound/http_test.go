package outbound

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHttpOutbound(t *testing.T) {
	// given
	key := "flag"
	server := httptest.NewServer(mockHandler{t: t, key: key})
	t.Cleanup(server.Close)

	outbound := NewHttp(Configuration{
		Callbacks: []HeaderCallback{
			func() (string, string) {
				return "Authorization", "Token"
			},
		},
		BaseURI: server.URL,
	})

	// when
	response, err := outbound.Single(t.Context(), key, []byte{})
	if err != nil {
		t.Fatalf("error from request: %v", err)
		return
	}

	// then - expect an ok response
	if response.Status != http.StatusOK {
		t.Errorf("expected 200, but got %d", response.Status)
	}
}

type mockHandler struct {
	key string
	t   *testing.T
}

func (r mockHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		r.t.Logf("invalid request method, expected %s, got %s. test will fail", http.MethodPost, req.Method)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	path := fmt.Sprintf("%s%s", ofrepV1, r.key)
	if req.RequestURI != fmt.Sprintf("%s%s", ofrepV1, r.key) {
		r.t.Logf("invalid request path, expected %s, got %s. test will fail", path, req.RequestURI)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Header.Get("Authorization") == "" {
		r.t.Log("expected non-empty Authorization header, but got empty. test will fail")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	resp.WriteHeader(http.StatusOK)
}
