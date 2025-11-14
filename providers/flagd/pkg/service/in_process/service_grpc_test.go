package process

import (
	"buf.build/gen/go/open-feature/flagd/grpc/go/flagd/sync/v1/syncv1grpc"
	v1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/flagd/sync/v1"
	"context"
	"fmt"
	"github.com/open-feature/go-sdk/openfeature"
	"google.golang.org/grpc"
	"log"
	"net"
	"testing"
	"time"
)

// shared flag for tests
var flagRsp = `{
		"flags": {
		  "myBoolFlag": {
			"state": "ENABLED",
			"variants": {
			  "on": true,
			  "off": false
			},
			"defaultVariant": "on"
		  }
		}
	}`

func TestInProcessProviderEvaluation(t *testing.T) {
	// given
	host := "localhost"
	port := 8090
	scope := "app=myapp"

	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		t.Fatal(err)
	}

	bufServ := &bufferedServer{
		listener: listen,
		mockResponses: []*v1.SyncFlagsResponse{
			{
				FlagConfiguration: flagRsp,
			},
		},
		fetchAllFlagsResponse: nil,
		fetchAllFlagsError:    nil,
	}

	inProcessService := NewInProcessService(Configuration{
		Host:       host,
		Port:       port,
		Selector:   scope,
		TLSEnabled: false,
		RetryBackOffMaxMs: 5000,
		RetryBackOffMs: 1000,
	})

	// when

	// start grpc sync server
	go func() {
		serve(bufServ)
	}()

	// Initialize service
	err = inProcessService.Init()
	if err != nil {
		t.Fatal(err)
	}

	// then

	eventChan := inProcessService.events

	// provider must be ready in acceptable time
	select {
	case event := <-eventChan:
		if event.EventType != openfeature.ProviderReady {
			t.Fatal("Provider initialization failed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Provider initialization did not complete within acceptable timeframe")
	}

	// provider must emit flag change event for the first flag sync
	select {
	case event := <-eventChan:
		if event.EventType != openfeature.ProviderConfigChange {
			t.Fatal("Provider failed to update flag configurations")
		}

		if len(event.ProviderEventDetails.FlagChanges) == 0 {
			t.Fatal("Expected flag changes, but got none")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Provider did not sync flags within an acceptable timeframe")
	}

	// provider must evaluate flag from the grpc source data
	detail := inProcessService.ResolveBoolean(context.Background(), "myBoolFlag", false, make(map[string]interface{}))

	if !detail.Value {
		t.Fatal("Expected true, but got false")
	}

	// check for metadata - scope from grpc sync
	if len(detail.FlagMetadata) == 0 && detail.FlagMetadata["scope"] == "" {
		t.Fatal("Expected scope to be present, but got none")
	}

	if scope != detail.FlagMetadata["scope"] {
		t.Fatalf("Wrong scope value. Expected %s, but got %s", scope, detail.FlagMetadata["scope"])
	}
}

// custom name resolver
func TestInProcessProviderEvaluationEnvoy(t *testing.T) {
	// given
	host := "localhost"
	port := 9211
	scope := "app=myapp"

	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		t.Fatal(err)
	}

	bufServ := &bufferedServer{
		listener: listen,
		mockResponses: []*v1.SyncFlagsResponse{
			{
				FlagConfiguration: flagRsp,
			},
		},
		fetchAllFlagsResponse: nil,
		fetchAllFlagsError:    nil,
	}

	inProcessService := NewInProcessService(Configuration{
		TargetUri: "envoy://localhost:9211/foo.service",
		Selector:   scope,
		TLSEnabled: false,
		RetryBackOffMaxMs: 5000,
		RetryBackOffMs: 1000,
	})

	// when

	// start grpc sync server
	go func() {
		serve(bufServ)
	}()

	// Initialize service
	err = inProcessService.Init()
	if err != nil {
		t.Fatal(err)
	}

	// then

	eventChan := inProcessService.events

	// provider must be ready in acceptable time
	select {
	case event := <-eventChan:
		if event.EventType != openfeature.ProviderReady {
			t.Fatal("Provider initialization failed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Provider initialization did not complete within acceptable timeframe")
	}

	// provider must emit flag change event for the first flag sync
	select {
	case event := <-eventChan:
		if event.EventType != openfeature.ProviderConfigChange {
			t.Fatal("Provider failed to update flag configurations")
		}

		if len(event.ProviderEventDetails.FlagChanges) == 0 {
			t.Fatal("Expected flag changes, but got none")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Provider did not sync flags within an acceptable timeframe")
	}

	// provider must evaluate flag from the grpc source data
	detail := inProcessService.ResolveBoolean(context.Background(), "myBoolFlag", false, make(map[string]interface{}))

	if !detail.Value {
		t.Fatal("Expected true, but got false")
	}

	// check for metadata - scope from grpc sync
	if len(detail.FlagMetadata) == 0 && detail.FlagMetadata["scope"] == "" {
		t.Fatal("Expected scope to be present, but got none")
	}

	if scope != detail.FlagMetadata["scope"] {
		t.Fatalf("Wrong scope value. Expected %s, but got %s", scope, detail.FlagMetadata["scope"])
	}
}


// bufferedServer - a mock grpc service backed by buffered connection
type bufferedServer struct {
	listener              net.Listener
	mockResponses         []*v1.SyncFlagsResponse
	fetchAllFlagsResponse *v1.FetchAllFlagsResponse
	fetchAllFlagsError    error
}

func (b *bufferedServer) SyncFlags(_ *v1.SyncFlagsRequest, stream syncv1grpc.FlagSyncService_SyncFlagsServer) error {
	for _, response := range b.mockResponses {
		err := stream.Send(response)
		if err != nil {
			fmt.Printf("Error with stream: %s", err.Error())
			return err
		}
	}

	// delay EOF
	time.Sleep(2 * time.Second)

	return nil
}

func (b *bufferedServer) FetchAllFlags(_ context.Context, _ *v1.FetchAllFlagsRequest) (*v1.FetchAllFlagsResponse, error) {
	return b.fetchAllFlagsResponse, b.fetchAllFlagsError
}

func (b *bufferedServer) GetMetadata(_ context.Context, _ *v1.GetMetadataRequest) (*v1.GetMetadataResponse, error) {
	return &v1.GetMetadataResponse{}, nil
}

// serve serves a bufferedServer. This is a blocking call
func serve(bServer *bufferedServer) {
	server := grpc.NewServer()

	syncv1grpc.RegisterFlagSyncServiceServer(server, bServer)

	if err := server.Serve(bServer.listener); err != nil {
		log.Fatalf("Server exited with error: %v", err)
	}
}
