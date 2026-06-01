package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	schemaConnectV2 "buf.build/gen/go/open-feature/flagd/connectrpc/go/flagd/evaluation/v2/evaluationv2connect"
	schemaV2 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/flagd/evaluation/v2"
	"connectrpc.com/connect"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/header"
)

// TestSelectorInterceptor_Unary verifies the flagd-selector header is added
// to outgoing unary requests when a selector is configured, and omitted when
// it is not.
func TestSelectorInterceptor_Unary(t *testing.T) {
	tests := []struct {
		name          string
		selector      string
		expectedValue string
	}{
		{"sends header when configured", "source=db,app=myapp", "source=db,app=myapp"},
		{"omits header when empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := newSelectorInterceptor(tt.selector)
			req := connect.NewRequest(&schemaV2.ResolveBooleanRequest{FlagKey: "k"})

			wrapped := interceptor.WrapUnary(func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				got := req.Header().Get(header.Selector)
				if got != tt.expectedValue {
					t.Errorf("expected header %q, got %q", tt.expectedValue, got)
				}
				return connect.NewResponse(&schemaV2.ResolveBooleanResponse{}), nil
			})

			if _, err := wrapped(context.Background(), req); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestSelectorInterceptor_E2E spins up a real connect server and verifies the
// flagd-selector header arrives on the server side via a unary call.
func TestSelectorInterceptor_E2E(t *testing.T) {
	tests := []struct {
		name         string
		selector     string
		expectHeader bool
	}{
		{"with selector", "source=db", true},
		{"without selector", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			received := make(chan string, 1)
			mux := http.NewServeMux()
			path, handler := schemaConnectV2.NewServiceHandler(&headerCapturingHandler{received: received})
			mux.Handle(path, handler)

			srv := httptest.NewUnstartedServer(mux)
			srv.EnableHTTP2 = true
			srv.StartTLS()
			defer srv.Close()

			httpClient := srv.Client()
			var opts []connect.ClientOption
			if tt.selector != "" {
				opts = append(opts, connect.WithInterceptors(newSelectorInterceptor(tt.selector)))
			}
			client := schemaConnectV2.NewServiceClient(httpClient, srv.URL, opts...)

			_, err := client.ResolveBoolean(context.Background(), connect.NewRequest(&schemaV2.ResolveBooleanRequest{FlagKey: "k"}))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := <-received
			if tt.expectHeader && got != tt.selector {
				t.Errorf("expected header %q, got %q", tt.selector, got)
			}
			if !tt.expectHeader && got != "" {
				t.Errorf("expected no header, got %q", got)
			}
		})
	}
}

type headerCapturingHandler struct {
	schemaConnectV2.UnimplementedServiceHandler
	received chan string
}

func (h *headerCapturingHandler) ResolveBoolean(_ context.Context, req *connect.Request[schemaV2.ResolveBooleanRequest]) (*connect.Response[schemaV2.ResolveBooleanResponse], error) {
	h.received <- req.Header().Get(header.Selector)
	return connect.NewResponse(&schemaV2.ResolveBooleanResponse{}), nil
}
