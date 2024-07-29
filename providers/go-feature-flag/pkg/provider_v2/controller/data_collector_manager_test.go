package controller_test

import (
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/provider_v2/controller"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/provider_v2/model"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

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
	t.Run("Should collect only once if there is no event in queue", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
			}
		}, Err: nil}
		client := &http.Client{Transport: &mrt}
		g := controller.NewGoFeatureFlagAPI(controller.GoFeatureFlagApiOptions{
			HTTPClient: client,
		})

		collector := controller.NewDataCollectorManager(*g, 100, 100*time.Millisecond)
		collector.Start()
		defer collector.Stop()
		_ = collector.AddEvent(eventExample)

		time.Sleep(300 * time.Millisecond)
		assert.Equal(t, 1, mrt.NumberCall)
	})

	t.Run("Should collect multiple times if we are adding events in between intervals", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
			}
		}, Err: nil}
		client := &http.Client{Transport: &mrt}
		g := controller.NewGoFeatureFlagAPI(controller.GoFeatureFlagApiOptions{
			HTTPClient: client,
		})

		collector := controller.NewDataCollectorManager(*g, 100, 100*time.Millisecond)
		collector.Start()
		defer collector.Stop()
		_ = collector.AddEvent(eventExample)
		_ = collector.AddEvent(eventExample)
		_ = collector.AddEvent(eventExample)
		time.Sleep(120 * time.Millisecond)
		_ = collector.AddEvent(eventExample)
		time.Sleep(120 * time.Millisecond)
		_ = collector.AddEvent(eventExample)
		time.Sleep(120 * time.Millisecond)
		assert.Equal(t, 3, mrt.NumberCall)
	})

	t.Run("Should stop adding events if max items reached", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
			}
		}, Err: nil}
		client := &http.Client{Transport: &mrt}
		g := controller.NewGoFeatureFlagAPI(controller.GoFeatureFlagApiOptions{
			HTTPClient: client,
		})

		collector := controller.NewDataCollectorManager(*g, 5, 10*time.Minute)
		collector.Start()
		defer collector.Stop()
		err := collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.Error(t, err)
		err = collector.AddEvent(eventExample)
		assert.Error(t, err)

		_ = collector.SendData()
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
	})

	t.Run("Should not remove items if saveData failed", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
			}
		}, Err: nil}
		client := &http.Client{Transport: &mrt}
		g := controller.NewGoFeatureFlagAPI(controller.GoFeatureFlagApiOptions{
			HTTPClient: client,
		})

		collector := controller.NewDataCollectorManager(*g, 5, 100*time.Millisecond)
		collector.Start()
		defer collector.Stop()
		err := collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		err = collector.AddEvent(eventExample)
		assert.NoError(t, err)
		// Wait until the data collector sends the data (and failed)
		time.Sleep(180 * time.Millisecond)

		// Should error because the data collector is full
		err = collector.AddEvent(eventExample)
		assert.Error(t, err)

		// Should have tried only once to call the API
		assert.Equal(t, 1, mrt.NumberCall)
	})
}
