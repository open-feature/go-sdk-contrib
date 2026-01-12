package process

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"buf.build/gen/go/open-feature/flagd/grpc/go/flagd/sync/v1/syncv1grpc"
	v1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/flagd/sync/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TestSelectorHeader verifies that the flagd-selector header is sent correctly in gRPC metadata
func TestSelectorHeader(t *testing.T) {
	tests := []struct {
		name          string
		selector      string
		expectHeader  bool
		expectedValue string
	}{
		{
			name:          "selector header is sent when configured",
			selector:      "source=database,app=myapp",
			expectHeader:  true,
			expectedValue: "source=database,app=myapp",
		},
		{
			name:          "no selector header when selector is empty",
			selector:      "",
			expectHeader:  false,
			expectedValue: "",
		},
		{
			name:          "selector header with complex value",
			selector:      "source=test,environment=production,region=us-east",
			expectHeader:  true,
			expectedValue: "source=test,environment=production,region=us-east",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			port := findFreePort(t)
			listen, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
			if err != nil {
				t.Fatalf("Failed to create listener: %v", err)
			}
			defer func() {
				// Listener will be closed by GracefulStop, so ignore "use of closed network connection" errors
				_ = listen.Close()
			}()

			headerReceived := make(chan string, 1)
			mockServer := &selectorHeaderCapturingServer{
				headerReceived: headerReceived,
				mockResponse: &v1.SyncFlagsResponse{
					FlagConfiguration: flagRsp,
				},
			}

			grpcServer := grpc.NewServer()
			syncv1grpc.RegisterFlagSyncServiceServer(grpcServer, mockServer)

			serverDone := make(chan struct{})
			go func() {
				defer close(serverDone)
				if err := grpcServer.Serve(listen); err != nil {
					t.Logf("Server exited: %v", err)
				}
			}()
			defer func() {
				grpcServer.GracefulStop()
				<-serverDone
			}()

			inProcessService := NewInProcessService(Configuration{
				Host:              "localhost",
				Port:              port,
				Selector:          tt.selector,
				TLSEnabled:        false,
				RetryBackOffMaxMs: 2000,
				RetryBackOffMs:    1000,
			})

			// when
			err = inProcessService.Init()
			if err != nil {
				t.Fatalf("Failed to initialize service: %v", err)
			}
			defer inProcessService.Shutdown()

			// Wait for provider to be ready
			select {
			case <-inProcessService.events:
				// Provider ready event
			case <-time.After(2 * time.Second):
				t.Fatal("Timeout waiting for provider ready event")
			}

			// then - verify the flagd-selector header
			select {
			case receivedSelector := <-headerReceived:
				if tt.expectHeader {
					if receivedSelector != tt.expectedValue {
						t.Errorf("Expected selector header to be %q, but got %q", tt.expectedValue, receivedSelector)
					}
				} else {
					if receivedSelector != "" {
						t.Errorf("Expected no selector header, but got %q", receivedSelector)
					}
				}
			case <-time.After(3 * time.Second):
				if tt.expectHeader {
					t.Fatal("Timeout waiting for flagd-selector header")
				}
			}
		})
	}
}

// findFreePort finds an available port for testing
func findFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("Failed to close listener: %v", err)
	}
	return port
}

// selectorHeaderCapturingServer captures the flagd-selector header from incoming requests
type selectorHeaderCapturingServer struct {
	syncv1grpc.UnimplementedFlagSyncServiceServer
	headerReceived chan string
	mockResponse   *v1.SyncFlagsResponse
	mu             sync.Mutex
}

func (s *selectorHeaderCapturingServer) SyncFlags(req *v1.SyncFlagsRequest, stream syncv1grpc.FlagSyncService_SyncFlagsServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.captureHeader(stream.Context())

	// Send mock response
	if err := stream.Send(s.mockResponse); err != nil {
		return err
	}

	// Keep stream open briefly
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (s *selectorHeaderCapturingServer) FetchAllFlags(ctx context.Context, req *v1.FetchAllFlagsRequest) (*v1.FetchAllFlagsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Extract and capture the flagd-selector header
	s.captureHeader(ctx)

	return &v1.FetchAllFlagsResponse{
		FlagConfiguration: flagRsp,
	}, nil
}

func (s *selectorHeaderCapturingServer) GetMetadata(ctx context.Context, req *v1.GetMetadataRequest) (*v1.GetMetadataResponse, error) {
	return &v1.GetMetadataResponse{}, nil
}

func (s *selectorHeaderCapturingServer) captureHeader(ctx context.Context) {
	md, _ := metadata.FromIncomingContext(ctx)
	headerValue := ""
	if values := md.Get("flagd-selector"); len(values) > 0 {
		headerValue = values[0]
	}
	select {
	case s.headerReceived <- headerValue:
	default:
		// Channel is full, which is acceptable in this test.
	}
}
