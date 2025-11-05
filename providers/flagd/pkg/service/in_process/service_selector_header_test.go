package process

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"buf.build/gen/go/open-feature/flagd/grpc/go/flagd/sync/v1/syncv1grpc"
	v1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/flagd/sync/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Test that the flagd-selector header is sent in gRPC metadata
func TestSelectorHeaderIsSent(t *testing.T) {
// given
host := "localhost"
port := 8091
selector := "source=test,app=selector-test"
headerReceived := make(chan string, 1)

listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
if err != nil {
t.Fatal(err)
}

// Mock server that captures the flagd-selector header
mockServer := &selectorHeaderCapturingServer{
listener:       listen,
headerReceived: headerReceived,
mockResponse: &v1.SyncFlagsResponse{
FlagConfiguration: flagRsp,
},
}

inProcessService := NewInProcessService(Configuration{
Host:       host,
Port:       port,
Selector:   selector,
TLSEnabled: false,
})

// when
go func() {
server := grpc.NewServer()
syncv1grpc.RegisterFlagSyncServiceServer(server, mockServer)
if err := server.Serve(mockServer.listener); err != nil {
t.Logf("Server exited with error: %v", err)
}
}()

// Initialize service
err = inProcessService.Init()
if err != nil {
t.Fatal(err)
}

// then - verify that the flagd-selector header was sent
select {
case receivedSelector := <-headerReceived:
if receivedSelector != selector {
t.Fatalf("Expected selector header to be %q, but got %q", selector, receivedSelector)
}
case <-time.After(3 * time.Second):
t.Fatal("Timeout waiting for flagd-selector header to be received")
}

inProcessService.Shutdown()
}

// Mock server that captures the flagd-selector header from incoming requests
type selectorHeaderCapturingServer struct {
listener       net.Listener
headerReceived chan string
mockResponse   *v1.SyncFlagsResponse
}

func (s *selectorHeaderCapturingServer) SyncFlags(req *v1.SyncFlagsRequest, stream syncv1grpc.FlagSyncService_SyncFlagsServer) error {
// Extract metadata from context
md, ok := metadata.FromIncomingContext(stream.Context())
if ok {
// Check for flagd-selector header
if values := md.Get("flagd-selector"); len(values) > 0 {
s.headerReceived <- values[0]
} else {
s.headerReceived <- ""
}
} else {
s.headerReceived <- ""
}

// Send mock response
err := stream.Send(s.mockResponse)
if err != nil {
return err
}

// Keep stream open for a bit
time.Sleep(1 * time.Second)
return nil
}

func (s *selectorHeaderCapturingServer) FetchAllFlags(ctx context.Context, req *v1.FetchAllFlagsRequest) (*v1.FetchAllFlagsResponse, error) {
// Extract metadata from context
md, ok := metadata.FromIncomingContext(ctx)
if ok {
// Check for flagd-selector header
if values := md.Get("flagd-selector"); len(values) > 0 {
s.headerReceived <- values[0]
} else {
s.headerReceived <- ""
}
} else {
s.headerReceived <- ""
}

return &v1.FetchAllFlagsResponse{
FlagConfiguration: flagRsp,
}, nil
}

func (s *selectorHeaderCapturingServer) GetMetadata(ctx context.Context, req *v1.GetMetadataRequest) (*v1.GetMetadataResponse, error) {
return &v1.GetMetadataResponse{}, nil
}
