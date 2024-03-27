package outbound

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestHttpOutbound(t *testing.T) {
	// given
	host := "localhost:18181"
	key := "flag"

	server := http.Server{
		Addr: host,
		Handler: mockHandler{
			t:   t,
			key: key,
		},
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			t.Logf("error starting mock server: %v", err)
			return
		}
	}()

	<-time.After(3 * time.Second)

	outbound := NewHttp(Configuration{
		Callbacks: []HeaderCallback{
			func() (string, string) {
				return "Authorization", "Token"
			},
		},
		BaseURI: fmt.Sprintf("http://%s", host),
	})

	// when
	response, err := outbound.PostSingle(context.Background(), key, []byte{})
	if err != nil {
		t.Fatalf("error from request: %v", err)
		return
	}

	// then - expect an ok response
	if response.StatusCode != http.StatusOK {
		t.Errorf("expected 200, but got %d", response.StatusCode)
	}

	// cleanup
	err = server.Shutdown(context.Background())
	if err != nil {
		t.Errorf("error shuttting down mock server: %v", err)
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
