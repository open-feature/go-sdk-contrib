package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"buf.build/gen/go/open-feature/flagd/connectrpc/go/flagd/evaluation/v1/evaluationv1connect"
	evaluation "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/flagd/evaluation/v1"
	"connectrpc.com/connect"
	"github.com/go-logr/logr"
	flagdService "github.com/open-feature/flagd/core/pkg/service"
	"go.openfeature.dev/contrib/providers/flagd/v2/internal/cache"
	of "go.openfeature.dev/openfeature/v2"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRPCServiceShutdownCleansUpGoroutines(t *testing.T) {
	// At the end of the test, if no other failures have occurred, check for
	// goroutine leaks.
	startingGoroutineCount := runtime.NumGoroutine()
	t.Cleanup(func() {
		if t.Failed() {
			return
		}
		if numGoroutinesAfter := runtime.NumGoroutine(); numGoroutinesAfter > startingGoroutineCount {
			t.Errorf("Goroutines leaked: %d goroutines before, %d goroutines after", startingGoroutineCount, numGoroutinesAfter)
		}
	})

	var log logr.Logger
	cache := cache.NewCacheService(cache.LRUValue, 10, log)
	srv, cfg := runTestServer(t)
	srv.eventStreamResponses <- &evaluation.EventStreamResponse{
		Type: string(flagdService.ProviderReady),
	}
	service := NewService(cfg, cache, log, 3 /*=retries*/)
	if err := service.Init(); err != nil {
		t.Fatal(err)
	}

	// Wait for provider to be ready.
	channel := service.EventChannel()
	select {
	case event := <-channel:
		if event.EventType != of.ProviderReady {
			t.Fatalf("Provider initialization failed. Got event type %s with message %s", event.EventType, event.Message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Provider initialization did not complete within acceptable timeframe")
	}

	service.Shutdown()
}

func TestRPCServiceShutdownDuringEventHandlingCleansUpGoroutines(t *testing.T) {
	checkGoroutineLeaks(t)

	var log logr.Logger
	cache := cache.NewCacheService(cache.LRUValue, 10, log)
	// Run the server. Then, queue up several events so that the service's event
	// streaming goroutine is forced to block while it waits for consumers to
	// handle events. When we shut down the service, it should be able to unblock
	// itself.
	srv, cfg := runTestServer(t)
	srv.eventStreamResponses <- &evaluation.EventStreamResponse{
		Type: string(flagdService.ProviderReady),
	}
	for range 50 {
		srv.eventStreamResponses <- &evaluation.EventStreamResponse{
			Type: string(flagdService.ConfigurationChange),
			Data: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"flags": {
						Kind: &structpb.Value_StructValue{
							StructValue: &structpb.Struct{},
						},
					},
				},
			},
		}
	}

	service := NewService(cfg, cache, log, 3 /*=retries*/)
	if err := service.Init(); err != nil {
		t.Fatal(err)
	}

	// Wait for provider to be ready.
	channel := service.EventChannel()
	select {
	case event := <-channel:
		if event.EventType != of.ProviderReady {
			t.Fatalf("Provider initialization failed. Got event type %s with message %s", event.EventType, event.Message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Provider initialization did not complete within acceptable timeframe")
	}

	// Wait a little bit for the buffer to fill up.
	time.Sleep(100 * time.Millisecond)

	// Shut down immediately, without waiting for the provider to be ready.
	service.Shutdown()
}

func TestRPCServiceShutdownDuringInitRetry(t *testing.T) {
	// TODO: The httptest server seems to leak a persistConn goroutine for
	// a very short duration (<1ms) in this test - it might have something to do
	// with the error returned on the stream rather than a success response.
	// It would be nice to figure out why this is happening and then re-enable
	// the goroutine leak check.

	// checkGoroutineLeaks(t)

	var log logr.Logger
	cache := cache.NewCacheService(cache.LRUValue, 10, log)
	// Run the server. Then, queue up several events so that the service's event
	// streaming goroutine is forced to block while it waits for consumers to
	// handle events. When we shut down the service, it should be able to unblock
	// itself.
	srv, cfg := runTestServer(t)
	srv.eventStreamErrors <- errors.New("server error")

	service := NewService(cfg, cache, log, 3 /*=retries*/)
	// Override the retry delay so that the test will time out if it doesn't
	// respect ctx cancellation while the retry delay is in progress.
	service.retryCounter.currentDelay = 100 * time.Hour
	if err := service.Init(); err != nil {
		t.Fatal(err)
	}

	// Wait a little bit for the event stream goroutine to receive the error
	// from the server.
	time.Sleep(100 * time.Millisecond)

	// The service should now be waiting for the retry delay to expire, which it
	// never will. Calling Shutdown() should cancel the context, unblocking the
	// goroutine, and then wait for the goroutine to exit.
	service.Shutdown()
}

func TestRPCServiceShutdownCancelsEventStreamGoroutine(t *testing.T) {
	checkGoroutineLeaks(t)

	var log logr.Logger
	cache := cache.NewCacheService(cache.LRUValue, 10, log)
	// Run the server. Then, queue up several events so that the service's event
	// streaming goroutine is forced to block while it waits for consumers to
	// handle events. When we shut down the service, it should be able to unblock
	// itself.
	srv, cfg := runTestServer(t)
	srv.eventStreamResponses <- &evaluation.EventStreamResponse{
		Type: string(flagdService.ProviderReady),
	}
	for range 50 {
		srv.eventStreamResponses <- &evaluation.EventStreamResponse{
			Type: string(flagdService.ConfigurationChange),
			Data: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"flags": {
						Kind: &structpb.Value_StructValue{
							StructValue: &structpb.Struct{},
						},
					},
				},
			},
		}
	}

	service := NewService(cfg, cache, log, 3 /*=retries*/)
	if err := service.Init(); err != nil {
		t.Fatal(err)
	}

	// Wait for provider to be ready.
	channel := service.EventChannel()
	select {
	case event := <-channel:
		if event.EventType != of.ProviderReady {
			t.Fatalf("Provider initialization failed. Got event type %s with message %s", event.EventType, event.Message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Provider initialization did not complete within acceptable timeframe")
	}

	// Wait a little bit for the buffer to fill up.
	time.Sleep(100 * time.Millisecond)

	// The service should now be blocked trying to send events on
	// service.EventChannel(), whose buffer is full. Calling Shutdown() should
	// cancel the context, unblocking the goroutine, and then wait for the
	// goroutine to exit.
	service.Shutdown()
}

// At the end of the test, if no other failures have occurred, check for
// goroutine leaks, and fail the test if any were found.
func checkGoroutineLeaks(t *testing.T) {
	startingGoroutineCount := runtime.NumGoroutine()
	t.Cleanup(func() {
		if t.Failed() {
			return
		}
		buf := make([]byte, 1<<20)
		stacklen := runtime.Stack(buf, true)
		if numGoroutinesAfter := runtime.NumGoroutine(); numGoroutinesAfter > startingGoroutineCount {
			t.Errorf("Goroutines leaked: %d goroutines before, %d goroutines after", startingGoroutineCount, numGoroutinesAfter)
			fmt.Fprintf(os.Stderr, "%s\n", buf[:stacklen])
		}
	})
}

type testServer struct {
	evaluationv1connect.UnimplementedServiceHandler
	eventStreamErrors    chan error
	eventStreamResponses chan *evaluation.EventStreamResponse
}

func (f *testServer) EventStream(ctx context.Context, req *connect.Request[evaluation.EventStreamRequest], stream *connect.ServerStream[evaluation.EventStreamResponse]) error {
	for {
		select {
		case rsp := <-f.eventStreamResponses:
			if err := stream.Send(rsp); err != nil {
				return err
			}
		case err := <-f.eventStreamErrors:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

func runTestServer(t *testing.T) (*testServer, Configuration) {
	ts := &testServer{
		eventStreamResponses: make(chan *evaluation.EventStreamResponse, 100),
		eventStreamErrors:    make(chan error, 1),
	}
	mountPath, handler := evaluationv1connect.NewServiceHandler(ts)
	mux := http.NewServeMux()
	mux.Handle(mountPath, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
		// time.Sleep(1 * time.Millisecond)
	})
	hostPort, ok := strings.CutPrefix(server.URL, "http://")
	if !ok {
		t.Fatal("unexpected server URL", server.URL)
	}
	host, portStr, _ := strings.Cut(hostPort, ":")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal("unexpected port", portStr)
	}
	cfg := Configuration{Host: host, Port: uint16(port)}
	return ts, cfg
}
