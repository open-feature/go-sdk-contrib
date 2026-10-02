package manager_test

import (
	"context"
	"encoding/json"
	"fmt"
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

	t.Run("Should flush in the background when max items reached", func(t *testing.T) {
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

		assert.NoError(t, collector.AddEvent(eventExample))
		assert.NoError(t, collector.AddEvent(eventExample))
		assert.Equal(t, 0, mrt.getNumberCall())

		assert.NoError(t, collector.AddEvent(eventExample))
		assert.Eventually(t, func() bool { return mrt.getNumberCall() == 1 }, time.Second, 5*time.Millisecond)

		assert.NoError(t, collector.AddEvent(eventExample))
		assert.NoError(t, collector.SendData(context.Background()))
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

		collector := manager.NewDataCollectorManager(g, 5, 10*time.Minute)
		assert.NoError(t, collector.AddEvent(eventExample))
		assert.NoError(t, collector.AddEvent(trackingEventExample))
		assert.Error(t, collector.SendData(context.Background()))

		var sent int
		mrt.RoundTripFunc = func(req *http.Request) *http.Response {
			var body struct {
				Events []json.RawMessage `json:"events"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)
			sent = len(body.Events)
			return &http.Response{StatusCode: http.StatusOK}
		}
		assert.NoError(t, collector.SendData(context.Background()))
		assert.Equal(t, 2, sent)
	})

	t.Run("Should drop the oldest events when the queue is full and the relay is down", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{StatusCode: http.StatusServiceUnavailable}
		}}
		client := &http.Client{Transport: &mrt}
		g := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		})

		collector := manager.NewDataCollectorManager(g, 3, 10*time.Minute)
		for i := range 10 {
			e := eventExample
			e.Variation = fmt.Sprintf("v%d", i)
			assert.NoError(t, collector.AddEvent(e))
		}
		assert.Equal(t, 0, mrt.getNumberCall())

		var variations []string
		mrt.RoundTripFunc = func(req *http.Request) *http.Response {
			var body struct {
				Events []model.FeatureEvent `json:"events"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)
			for _, e := range body.Events {
				variations = append(variations, e.Variation)
			}
			return &http.Response{StatusCode: http.StatusOK}
		}
		assert.NoError(t, collector.SendData(context.Background()))
		assert.Equal(t, []string{"v7", "v8", "v9"}, variations)
	})

	t.Run("AddEvent should not wait for an in-flight send", func(t *testing.T) {
		release := make(chan struct{})
		inFlight := make(chan struct{})
		var once sync.Once
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			once.Do(func() { close(inFlight) })
			<-release
			return &http.Response{StatusCode: http.StatusOK}
		}}
		client := &http.Client{Transport: &mrt}
		g := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		})

		collector := manager.NewDataCollectorManager(g, 2, 10*time.Minute)
		require.NoError(t, collector.AddEvent(eventExample))
		go func() { _ = collector.SendData(context.Background()) }()
		<-inFlight
		defer close(release)

		done := make(chan struct{})
		go func() {
			_ = collector.AddEvent(eventExample)
			_ = collector.AddEvent(eventExample)
			_ = collector.AddEvent(eventExample)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("AddEvent blocked while a send to the relay was in flight")
		}
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
