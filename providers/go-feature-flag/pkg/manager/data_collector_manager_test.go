package manager_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/manager"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) *http.Response
	Err           error
	mu            sync.Mutex
	NumberCall    int
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	m.NumberCall++
	m.mu.Unlock()
	return m.RoundTripFunc(req), m.Err
}

func (m *MockRoundTripper) getNumberCall() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.NumberCall
}

func Test_DataCollectorManager(t *testing.T) {
	eventExample := model.FeatureEvent{
		Kind:         "feature",
		ContextKind:  "user",
		UserKey:      "EFGH",
		CreationDate: 1722266324,
		Key:          "random-key",
		Variation:    "variationA",
		Value:        "YO",
		Default:      false,
		Version:      "",
		Source:       "SERVER",
	}
	trackingEventExample := model.TrackingEvent{
		Kind:              "tracking",
		ContextKind:       "user",
		UserKey:           "EFGH",
		CreationDate:      1722266324,
		Key:               "clicked-checkout",
		EvaluationContext: map[string]any{"targetingKey": "EFGH"},
		TrackingDetails:   map[string]any{"value": 99.99},
	}
	t.Run("Should collect only once if there is no event in queue", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
			}
		}, Err: nil}
		client := &http.Client{Transport: &mrt}
		g := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		})

		collector := manager.NewDataCollectorManager(g, 100, 100*time.Millisecond)
		collector.Start()
		defer collector.Stop(context.Background())
		_ = collector.AddEvent(eventExample)

		time.Sleep(300 * time.Millisecond)
		assert.Equal(t, 1, mrt.getNumberCall())
	})

	t.Run("Should collect multiple times if we are adding events in between intervals", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
			}
		}, Err: nil}
		client := &http.Client{Transport: &mrt}
		g := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		})

		collector := manager.NewDataCollectorManager(g, 100, 100*time.Millisecond)
		collector.Start()
		defer collector.Stop(context.Background())
		_ = collector.AddEvent(eventExample)
		_ = collector.AddEvent(eventExample)
		_ = collector.AddEvent(eventExample)
		time.Sleep(120 * time.Millisecond)
		_ = collector.AddEvent(eventExample)
		time.Sleep(120 * time.Millisecond)
		_ = collector.AddEvent(eventExample)
		time.Sleep(120 * time.Millisecond)
		assert.Equal(t, 3, mrt.getNumberCall())
	})

	t.Run("Should flush and continue adding when max items reached", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
			}
		}, Err: nil}
		client := &http.Client{Transport: &mrt}
		g := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		})

		collector := manager.NewDataCollectorManager(g, 3, 10*time.Minute)
		collector.Start()
		defer collector.Stop(context.Background())

		// Fill the queue to max
		err := collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		assert.Equal(t, 0, mrt.getNumberCall())

		// 4th event triggers a flush, then gets appended
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		assert.Equal(t, 1, mrt.getNumberCall())

		// Flush the remaining 1 event
		err = collector.SendData(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 2, mrt.getNumberCall())
	})

	t.Run("Should not remove items if saveData failed", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
			}
		}, Err: nil}
		client := &http.Client{Transport: &mrt}
		g := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		})

		collector := manager.NewDataCollectorManager(g, 5, 100*time.Millisecond)
		collector.Start()
		defer collector.Stop(context.Background())
		err := collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(trackingEventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		// Wait until the data collector sends the data (and failed)
		time.Sleep(180 * time.Millisecond)

		// Queue is still full after failed flush; AddEvent attempts another flush which also fails
		err = collector.AddEvent(eventExample)
		assert.Error(t, err)

		// The background ticker called once, then AddEvent attempted a flush once more
		assert.Equal(t, 2, mrt.getNumberCall())
	})

	t.Run("Should collect tracking events", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
			}
		}, Err: nil}
		client := &http.Client{Transport: &mrt}
		g := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		})

		collector := manager.NewDataCollectorManager(g, 100, 100*time.Millisecond)
		collector.Start()
		defer collector.Stop(context.Background())
		err := collector.AddEvent(trackingEventExample)
		require.NoError(t, err)

		err = collector.SendData(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 1, mrt.getNumberCall())
	})

	t.Run("Should flush buffered events on Stop", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{StatusCode: http.StatusOK}
		}}
		client := &http.Client{Transport: &mrt}
		g := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		})

		collector := manager.NewDataCollectorManager(g, 100, 10*time.Minute) // long interval, won't tick
		collector.Start()
		_ = collector.AddEvent(eventExample)
		_ = collector.AddEvent(eventExample)
		collector.Stop(context.Background()) // must flush the 2 buffered events
		assert.Equal(t, 1, mrt.getNumberCall())
	})

	t.Run("Should collect mixed feature and tracking events", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
			}
		}, Err: nil}
		client := &http.Client{Transport: &mrt}
		g := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		})

		collector := manager.NewDataCollectorManager(g, 100, 100*time.Millisecond)
		collector.Start()
		defer collector.Stop(context.Background())
		err := collector.AddEvent(eventExample)
		require.NoError(t, err)
		err = collector.AddEvent(trackingEventExample)
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		err = collector.SendData(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 1, mrt.getNumberCall())
	})
}
