package flagd

import (
	"buf.build/gen/go/open-feature/flagd/grpc/go/sync/v1/syncv1grpc"
	schemav1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	v1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/sync/v1"
	"context"
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	evalmock "github.com/open-feature/flagd/core/pkg/eval/mock"
	"github.com/open-feature/flagd/core/pkg/logger"
	flagdModels "github.com/open-feature/flagd/core/pkg/model"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
	"log"
	"net"
	"reflect"
	sync2 "sync"
	"testing"
	"time"
)

const sampleFlagConfig = `{
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

	return nil
}

func (b *bufferedServer) FetchAllFlags(_ context.Context, _ *v1.FetchAllFlagsRequest) (*v1.FetchAllFlagsResponse, error) {
	return b.fetchAllFlagsResponse, b.fetchAllFlagsError
}

// serve serves a bufferedServer. This is a blocking call
func serve(bServer *bufferedServer) {
	server := grpc.NewServer()

	syncv1grpc.RegisterFlagSyncServiceServer(server, bServer)

	if err := server.Serve(bServer.listener); err != nil {
		log.Fatalf("Server exited with error: %v", err)
	}
}

func TestProvider(t *testing.T) {
	port := 8116
	sURL := fmt.Sprintf("localhost:%d", port)
	lis, err := net.Listen("tcp", sURL)

	require.Nil(t, err)

	bufServer := bufferedServer{
		listener: lis,
		mockResponses: []*v1.SyncFlagsResponse{
			{
				FlagConfiguration: sampleFlagConfig,
				State:             v1.SyncState_SYNC_STATE_ALL,
			},
		},
	}

	// start server
	go serve(&bufServer)

	t.Setenv(flagdSourceURIEnvironmentVariableName, sURL)
	t.Setenv(flagdSourceProviderEnvironmentVariableName, syncProviderGrpc)
	t.Setenv(flagdSourceSelectorEnvironmentVariableName, "my-selector")

	prov := NewProvider(context.TODO())

	require.NotNil(t, prov)

	// listen for the events emitted by the provider
	receivedEvents := []of.Event{}
	mtx := sync2.RWMutex{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		for {
			select {
			case event := <-prov.EventChannel():
				mtx.Lock()
				receivedEvents = append(receivedEvents, event)
				mtx.Unlock()
			case <-ctx.Done():
				break
			}
		}
	}(ctx)

	select {
	case <-prov.IsReady():
	case <-time.After(5 * time.Second):
		t.Errorf("timed out waiting for the provider to be ready")
	}

	evaluation := prov.BooleanEvaluation(context.Background(), "myBoolFlag", false, of.FlattenedContext{})

	require.True(t, evaluation.Value)

	require.Eventually(t, func() bool {
		mtx.RLock()
		defer mtx.RUnlock()
		return len(receivedEvents) == 2
	}, 5*time.Second, 1*time.Millisecond)

	require.Equal(t, of.ProviderReady, receivedEvents[0].EventType)
	require.Equal(t, of.ProviderConfigChange, receivedEvents[1].EventType)
}

func TestBooleanEvaluation(t *testing.T) {
	// flag evaluation metadata
	metadata := map[string]interface{}{
		"scope": "flagd-scope",
	}

	metadataStruct, err := structpb.NewStruct(metadata)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		flagKey      string
		defaultValue bool
		evalCtx      map[string]interface{}

		mockOut   *schemav1.ResolveBooleanResponse
		mockError error

		response of.BoolResolutionDetail
	}{
		{
			name:         "happy path",
			flagKey:      "flag",
			defaultValue: true,
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.BoolResolutionDetail{
				Value: true,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant:      "on",
					Reason:       flagdModels.DefaultReason,
					FlagMetadata: map[string]interface{}{},
				},
			},
		},
		{
			name:         "with evaluation metadata",
			flagKey:      "flag-with-metadata",
			defaultValue: true,
			evalCtx:      map[string]interface{}{},
			mockOut: &schemav1.ResolveBooleanResponse{
				Value:    true,
				Variant:  "off",
				Reason:   flagdModels.DefaultReason,
				Metadata: metadataStruct,
			},
			mockError: nil,
			response: of.BoolResolutionDetail{
				Value: true,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "off",
					Reason:  flagdModels.DefaultReason,
					FlagMetadata: map[string]interface{}{
						"scope": "flagd-scope",
					},
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: true,
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveBooleanResponse{
				Reason: flagdModels.DefaultReason,
			},
			mockError: of.NewFlagNotFoundResolutionError(""),
			response: of.BoolResolutionDetail{
				Value: true,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          flagdModels.DefaultReason,
					ResolutionError: of.NewFlagNotFoundResolutionError(""),
					FlagMetadata:    map[string]interface{}{},
				},
			},
		},
		// flagd does not contain a value field in its response for go zero values (false)
		{
			name:         "zero value response",
			flagKey:      "flag",
			defaultValue: true,
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveBooleanResponse{
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant:      "on",
					Reason:       flagdModels.DefaultReason,
					FlagMetadata: map[string]interface{}{},
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			evalMock := evalmock.NewMockIEvaluator(ctrl)
			ctx := context.Background()

			evalMock.EXPECT().ResolveBooleanValue(ctx, "", test.flagKey, test.evalCtx).Return(
				test.mockOut.Value,
				test.mockOut.Variant,
				test.mockOut.Reason,
				test.mockOut.Metadata.AsMap(),
				test.mockError,
			)

			provider := Provider{
				evaluator: evalMock,
			}

			res := provider.BooleanEvaluation(context.Background(), test.flagKey, test.defaultValue, test.evalCtx)

			if res.ResolutionError.Error() != test.response.ResolutionError.Error() {
				t.Errorf("unexpected ResolutionError received, expected %v, got %v", test.response.ResolutionError.Error(), res.ResolutionError.Error())
			}
			if res.Variant != test.response.Variant {
				t.Errorf("unexpected Variant received, expected %v, got %v", test.response.Variant, res.Variant)
			}
			if res.Value != test.response.Value {
				t.Errorf("unexpected Value received, expected %v, got %v", test.response.Value, res.Value)
			}
			if res.Reason != test.response.Reason {
				t.Errorf("unexpected Reason received, expected %v, got %v", test.response.Reason, res.Reason)
			}
			if !reflect.DeepEqual(res.FlagMetadata, test.response.FlagMetadata) {
				t.Errorf("metadata mismatched, expected %v, got %v", test.response.FlagMetadata, res.FlagMetadata)
			}
		})
	}
}

func TestStringEvaluation(t *testing.T) {
	tests := []struct {
		name         string
		flagKey      string
		defaultValue string
		evalCtx      map[string]interface{}

		mockOut   *schemav1.ResolveStringResponse
		mockError error

		response of.StringResolutionDetail
	}{
		{
			name:         "happy path",
			flagKey:      "flag",
			defaultValue: "true",
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveStringResponse{
				Value:   "true",
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.StringResolutionDetail{
				Value: "true",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "on",
					Reason:  flagdModels.DefaultReason,
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: "true",
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveStringResponse{
				Reason: flagdModels.DefaultReason,
			},
			mockError: of.NewFlagNotFoundResolutionError(""),
			response: of.StringResolutionDetail{
				Value: "true",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          flagdModels.DefaultReason,
					ResolutionError: of.NewFlagNotFoundResolutionError(""),
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			evalMock := evalmock.NewMockIEvaluator(ctrl)
			ctx := context.Background()

			evalMock.EXPECT().ResolveStringValue(ctx, "", test.flagKey, test.evalCtx).Return(
				test.mockOut.Value,
				test.mockOut.Variant,
				test.mockOut.Reason,
				test.mockOut.Metadata.AsMap(),
				test.mockError,
			)

			provider := Provider{
				evaluator: evalMock,
			}

			res := provider.StringEvaluation(context.Background(), test.flagKey, test.defaultValue, test.evalCtx)

			if res.ResolutionError.Error() != test.response.ResolutionError.Error() {
				t.Errorf("unexpected ResolutionError received, expected %v, got %v", test.response.ResolutionError.Error(), res.ResolutionError.Error())
			}
			if res.Variant != test.response.Variant {
				t.Errorf("unexpected Variant received, expected %v, got %v", test.response.Variant, res.Variant)
			}
			if res.Value != test.response.Value {
				t.Errorf("unexpected Value received, expected %v, got %v", test.response.Value, res.Value)
			}
			if res.Reason != test.response.Reason {
				t.Errorf("unexpected Reason received, expected %v, got %v", test.response.Reason, res.Reason)
			}
		})
	}
}

func TestFloatEvaluation(t *testing.T) {
	tests := []struct {
		name         string
		flagKey      string
		defaultValue float64
		evalCtx      map[string]interface{}

		mockOut   *schemav1.ResolveFloatResponse
		mockError error

		response of.FloatResolutionDetail
	}{
		{
			name:         "happy path",
			flagKey:      "flag",
			defaultValue: float64(1),
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveFloatResponse{
				Value:   1,
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.FloatResolutionDetail{
				Value: 1,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "on",
					Reason:  flagdModels.DefaultReason,
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: float64(1),
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveFloatResponse{
				Reason: flagdModels.DefaultReason,
			},
			mockError: of.NewFlagNotFoundResolutionError(""),
			response: of.FloatResolutionDetail{
				Value: 1,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          flagdModels.DefaultReason,
					ResolutionError: of.NewFlagNotFoundResolutionError(""),
				},
			},
		},
		// flagd does not contain a value field in its response for go zero values
		{
			name:         "zero value response",
			flagKey:      "flag",
			defaultValue: 1,
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveFloatResponse{
				Variant: "zero",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.FloatResolutionDetail{
				Value: 0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "zero",
					Reason:  flagdModels.DefaultReason,
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			evalMock := evalmock.NewMockIEvaluator(ctrl)
			ctx := context.Background()

			evalMock.EXPECT().ResolveFloatValue(ctx, "", test.flagKey, test.evalCtx).Return(
				test.mockOut.Value,
				test.mockOut.Variant,
				test.mockOut.Reason,
				test.mockOut.Metadata.AsMap(),
				test.mockError,
			)

			provider := Provider{
				evaluator: evalMock,
			}

			res := provider.FloatEvaluation(context.Background(), test.flagKey, test.defaultValue, test.evalCtx)

			if res.ResolutionError.Error() != test.response.ResolutionError.Error() {
				t.Errorf("unexpected ResolutionError received, expected %v, got %v", test.response.ResolutionError.Error(), res.ResolutionError.Error())
			}
			if res.Variant != test.response.Variant {
				t.Errorf("unexpected Variant received, expected %v, got %v", test.response.Variant, res.Variant)
			}
			if res.Value != test.response.Value {
				t.Errorf("unexpected Value received, expected %v, got %v", test.response.Value, res.Value)
			}
			if res.Reason != test.response.Reason {
				t.Errorf("unexpected Reason received, expected %v, got %v", test.response.Reason, res.Reason)
			}
		})
	}
}

func TestIntEvaluation(t *testing.T) {
	tests := []struct {
		name         string
		flagKey      string
		defaultValue int64
		evalCtx      map[string]interface{}

		mockOut   *schemav1.ResolveIntResponse
		mockError error

		response of.IntResolutionDetail
	}{
		{
			name:         "happy path",
			flagKey:      "flag",
			defaultValue: 1,
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveIntResponse{
				Value:   1,
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.IntResolutionDetail{
				Value: 1,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "on",
					Reason:  flagdModels.DefaultReason,
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: 1,
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveIntResponse{
				Reason: flagdModels.DefaultReason,
			},
			mockError: of.NewFlagNotFoundResolutionError(""),
			response: of.IntResolutionDetail{
				Value: 1,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          flagdModels.DefaultReason,
					ResolutionError: of.NewFlagNotFoundResolutionError(""),
				},
			},
		},
		// flagd does not contain a value field in its response for go zero values
		{
			name:         "zero value response",
			flagKey:      "flag",
			defaultValue: 1,
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveIntResponse{
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.IntResolutionDetail{
				Value: 0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "on",
					Reason:  flagdModels.DefaultReason,
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		evalMock := evalmock.NewMockIEvaluator(ctrl)
		ctx := context.Background()

		evalMock.EXPECT().ResolveIntValue(ctx, "", test.flagKey, test.evalCtx).Return(
			test.mockOut.Value,
			test.mockOut.Variant,
			test.mockOut.Reason,
			test.mockOut.Metadata.AsMap(),
			test.mockError,
		)

		provider := Provider{
			evaluator: evalMock,
		}

		res := provider.IntEvaluation(context.Background(), test.flagKey, test.defaultValue, test.evalCtx)

		if res.ResolutionError.Error() != test.response.ResolutionError.Error() {
			t.Errorf("unexpected ResolutionError received, expected %v, got %v", test.response.ResolutionError.Error(), res.ResolutionError.Error())
		}
		if res.Variant != test.response.Variant {
			t.Errorf("unexpected Variant received, expected %v, got %v", test.response.Variant, res.Variant)
		}
		if res.Value != test.response.Value {
			t.Errorf("unexpected Value received, expected %v, got %v", test.response.Value, res.Value)
		}
		if res.Reason != test.response.Reason {
			t.Errorf("unexpected Reason received, expected %v, got %v", test.response.Reason, res.Reason)
		}
	}
}

func TestObjectEvaluation(t *testing.T) {
	tests := []struct {
		name         string
		flagKey      string
		defaultValue map[string]interface{}
		evalCtx      map[string]interface{}

		mockOut   *schemav1.ResolveObjectResponse
		mockError error

		response of.InterfaceResolutionDetail
	}{
		{
			name:    "happy path",
			flagKey: "flag",
			defaultValue: map[string]interface{}{
				"ping": "pong",
			},
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveObjectResponse{
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.InterfaceResolutionDetail{
				Value: map[string]interface{}{
					"this": "that",
				},
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "on",
					Reason:  flagdModels.DefaultReason,
				},
			},
		},
		{
			name:    "error response",
			flagKey: "flag",
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			defaultValue: map[string]interface{}{},
			mockOut: &schemav1.ResolveObjectResponse{
				Reason: flagdModels.DefaultReason,
			},
			mockError: of.NewFlagNotFoundResolutionError(""),
			response: of.InterfaceResolutionDetail{
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          flagdModels.DefaultReason,
					ResolutionError: of.NewFlagNotFoundResolutionError(""),
				},
				Value: map[string]interface{}{},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {

		if test.response.Value != nil {
			f, err := structpb.NewStruct(test.response.Value.(map[string]interface{}))
			if err != nil {
				t.Fatal(err)
			}
			test.mockOut.Value = f
		}

		evalMock := evalmock.NewMockIEvaluator(ctrl)
		ctx := context.Background()

		evalMock.EXPECT().ResolveObjectValue(ctx, "", test.flagKey, test.evalCtx).Return(
			test.mockOut.Value.AsMap(),
			test.mockOut.Variant,
			test.mockOut.Reason,
			test.mockOut.Metadata.AsMap(),
			test.mockError,
		)

		provider := Provider{
			evaluator: evalMock,
		}

		res := provider.ObjectEvaluation(context.Background(), test.flagKey, test.defaultValue, test.evalCtx)

		if res.ResolutionError.Error() != test.response.ResolutionError.Error() {
			t.Errorf("unexpected ResolutionError received, expected %v, got %v", test.response.ResolutionError.Error(), res.ResolutionError.Error())
		}
		if res.Variant != test.response.Variant {
			t.Errorf("unexpected Variant received, expected %v, got %v", test.response.Variant, res.Variant)
		}
		if res.Value != nil && test.mockOut.Value.AsMap() != nil {
			require.Equal(t, test.mockOut.Value.AsMap(), res.Value)
			//t.Errorf("unexpected Value received, expected %v, got %v", test.mockOut.Value.AsMap(), res.Value)
		}
		if res.Reason != test.response.Reason {
			t.Errorf("unexpected Reason received, expected %v, got %v", test.response.Reason, res.Reason)
		}
	}
}

func failingFunc(p *Provider) {
	p.handleConnectionErr(errors.New(""), func() {
		failingFunc(p)
	})
}

func TestProvider_handleConnectionErrEndUpInErrorState(t *testing.T) {

	p := &Provider{
		connectionInfo: connectionInfo{
			state:           0,
			retries:         0,
			maxSyncRetries:  2,
			backoffDuration: 1 * time.Millisecond,
		},
		ofEventChannel: make(chan of.Event),
		logger:         logger.NewLogger(nil, false),
	}

	receivedEvents := []of.Event{}
	mtx := sync2.RWMutex{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		for {
			select {
			case event := <-p.EventChannel():
				mtx.Lock()
				receivedEvents = append(receivedEvents, event)
				mtx.Unlock()
			case <-ctx.Done():
				break
			}
		}
	}(ctx)

	// call handleConnectionError with a function that keeps calling handleConnectionError again
	// this is to verify that eventually we terminate after maxSyncRetries has been reached
	p.handleConnectionErr(errors.New("oops"), func() {
		failingFunc(p)
	})

	// verify that we end up in the error state
	require.Equal(t, stateError, p.connectionInfo.state)
	require.Equal(t, p.connectionInfo.maxSyncRetries, p.connectionInfo.retries)

	require.Eventually(t, func() bool {
		mtx.RLock()
		defer mtx.RUnlock()
		return len(receivedEvents) == 2
	}, 5*time.Second, 1*time.Millisecond)

	require.Equal(t, of.ProviderStale, receivedEvents[0].EventType)
	require.Equal(t, of.ProviderError, receivedEvents[1].EventType)
}

func TestProvider_handleConnectionErrRecoverFromStaleState(t *testing.T) {
	p := &Provider{
		connectionInfo: connectionInfo{
			state:           0,
			retries:         0,
			maxSyncRetries:  2,
			backoffDuration: 1 * time.Millisecond,
		},
		ofEventChannel: make(chan of.Event),
		isReady:        make(chan struct{}),
		logger:         logger.NewLogger(nil, false),
	}

	receivedEvents := []of.Event{}
	mtx := sync2.RWMutex{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		for {
			select {
			case event := <-p.EventChannel():
				mtx.Lock()
				receivedEvents = append(receivedEvents, event)
				mtx.Unlock()
			case <-ctx.Done():
				break
			}
		}
	}(ctx)

	err := errors.New("oops")

	// call handleConnectionError with a function that keeps calling handleConnectionError again
	// this is to verify that eventually we terminate after maxSyncRetries has been reached
	p.handleConnectionErr(err, func() {
		require.Equal(t, stateStale, p.connectionInfo.state)
		require.Equal(t, 1, p.connectionInfo.retries)

		// simulate successful recovery
		p.handleProviderReady()
	})

	// verify that we are in ready state again
	require.Equal(t, stateReady, p.connectionInfo.state)
	require.Equal(t, 0, p.connectionInfo.retries)

	require.Eventually(t, func() bool {
		mtx.RLock()
		defer mtx.RUnlock()
		return len(receivedEvents) == 2
	}, 5*time.Second, 1*time.Millisecond)

	require.Equal(t, of.ProviderStale, receivedEvents[0].EventType)
	require.Equal(t, of.ProviderReady, receivedEvents[1].EventType)
}
