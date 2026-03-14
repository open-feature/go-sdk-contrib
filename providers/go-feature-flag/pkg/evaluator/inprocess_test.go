package evaluator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const initialConfiguration = `{
  "flags": {
    "TEST": {
      "variations": {
        "off": false,
        "on": true
      },
      "defaultRule": {
        "variation": "off"
      }
    }
  },
  "evaluationContextEnrichment": {
    "env": "production"
  }
}`

const updatedConfiguration = `{
  "flags": {
    "TEST": {
      "variations": {
        "off": false,
        "on": true
      },
      "defaultRule": {
        "variation": "on"
      }
    },
    "TEST2": {
      "variations": {
        "disabled": false,
        "enabled": true
      },
      "defaultRule": {
        "variation": "enabled"
      }
    }
  },
  "evaluationContextEnrichment": {
    "env": "staging"
  }
}`

type roundTripResponse func(req *http.Request) (*http.Response, error)

type sequencedRoundTripper struct {
	mu        sync.Mutex
	callCount int
	responses []roundTripResponse
}

func (s *sequencedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	idx := s.callCount
	s.callCount++
	if idx >= len(s.responses) {
		idx = len(s.responses) - 1
	}
	responder := s.responses[idx]
	s.mu.Unlock()
	return responder(req)
}

func (s *sequencedRoundTripper) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.callCount
}

func TestInProcess_PollFailureEmitsStaleAndReadyAfterNotModifiedRecovery(t *testing.T) {
	events := make(chan openfeature.Event, 8)
	transport := &sequencedRoundTripper{
		responses: []roundTripResponse{
			successResponse(initialConfiguration, `"etag-1"`),
			errorResponse(errors.New("relay proxy unavailable")),
			notModifiedResponse(),
			notModifiedResponse(),
		},
	}
	evaluator := newInProcessEvaluatorForTest(10*time.Millisecond, transport, events)
	require.NoError(t, evaluator.Init(context.Background()))
	defer func() {
		require.NoError(t, evaluator.Shutdown(context.Background()))
	}()

	initEvent := requireEvent(t, events, time.Second)
	require.Equal(t, openfeature.ProviderConfigChange, initEvent.EventType)

	staleEvent := requireEvent(t, events, time.Second)
	require.Equal(t, openfeature.ProviderStale, staleEvent.EventType)
	assert.Contains(t, staleEvent.Message, "Configuration refresh failed")
	assert.Contains(t, staleEvent.Message, "relay proxy unavailable")

	readyEvent := requireEvent(t, events, time.Second)
	require.Equal(t, openfeature.ProviderReady, readyEvent.EventType)
	assert.Equal(t, "Configuration refresh recovered", readyEvent.Message)
}

func TestInProcess_PollFailureEmitsConfigChangeAfterChangedRecovery(t *testing.T) {
	events := make(chan openfeature.Event, 8)
	transport := &sequencedRoundTripper{
		responses: []roundTripResponse{
			successResponse(initialConfiguration, `"etag-1"`),
			errorResponse(errors.New("temporary refresh failure")),
			successResponse(updatedConfiguration, `"etag-2"`),
			notModifiedResponse(),
		},
	}
	evaluator := newInProcessEvaluatorForTest(10*time.Millisecond, transport, events)
	require.NoError(t, evaluator.Init(context.Background()))
	defer func() {
		require.NoError(t, evaluator.Shutdown(context.Background()))
	}()

	require.Equal(t, openfeature.ProviderConfigChange, requireEvent(t, events, time.Second).EventType)
	require.Equal(t, openfeature.ProviderStale, requireEvent(t, events, time.Second).EventType)

	recoveryEvent := requireEvent(t, events, time.Second)
	require.Equal(t, openfeature.ProviderConfigChange, recoveryEvent.EventType)
	assert.Equal(t, "Configuration has changed", recoveryEvent.Message)
}

func TestInProcess_DuplicatePollFailuresDoNotEmitDuplicateStaleEvents(t *testing.T) {
	events := make(chan openfeature.Event, 8)
	transport := &sequencedRoundTripper{
		responses: []roundTripResponse{
			successResponse(initialConfiguration, `"etag-1"`),
			errorResponse(errors.New("refresh failed")),
			errorResponse(errors.New("refresh failed again")),
			errorResponse(errors.New("refresh failed yet again")),
		},
	}
	evaluator := newInProcessEvaluatorForTest(10*time.Millisecond, transport, events)
	require.NoError(t, evaluator.Init(context.Background()))

	shutdown := false
	defer func() {
		if !shutdown {
			require.NoError(t, evaluator.Shutdown(context.Background()))
		}
	}()

	require.Equal(t, openfeature.ProviderConfigChange, requireEvent(t, events, time.Second).EventType)
	require.Equal(t, openfeature.ProviderStale, requireEvent(t, events, time.Second).EventType)

	require.Eventually(t, func() bool {
		return transport.CallCount() >= 4
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, evaluator.Shutdown(context.Background()))
	shutdown = true
	assertNoEvent(t, events)
}

func newInProcessEvaluatorForTest(interval time.Duration, rt http.RoundTripper, eventStream chan openfeature.Event) *InProcess {
	return NewInprocessEvaluator(interval, api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
		Endpoint:   "http://localhost:1031",
		HTTPClient: &http.Client{Transport: rt},
	}), eventStream)
}

func successResponse(body, etag string) roundTripResponse {
	return func(_ *http.Request) (*http.Response, error) {
		headers := http.Header{}
		if etag != "" {
			headers.Set("ETag", etag)
		}
		return httpResponse(http.StatusOK, body, headers), nil
	}
}

func notModifiedResponse() roundTripResponse {
	return func(_ *http.Request) (*http.Response, error) {
		return httpResponse(http.StatusNotModified, "", http.Header{}), nil
	}
}

func errorResponse(err error) roundTripResponse {
	return func(_ *http.Request) (*http.Response, error) {
		return nil, err
	}
}

func TestInProcess_getEvaluationContextEnrichment_returnsCopy(t *testing.T) {
	t.Run("getEvaluationContextEnrichment returns a copy, not the internal map", func(t *testing.T) {
		i := &InProcess{
			evaluationContextEnrichment: map[string]any{"env": "production"},
		}
		got := i.getEvaluationContextEnrichment()
		assert.Equal(t, map[string]any{"env": "production"}, got)

		// mutating the returned copy must not affect internal state
		got["env"] = "mutated"
		assert.Equal(t, "production", i.evaluationContextEnrichment["env"])
	})
}

func httpResponse(status int, body string, headers http.Header) *http.Response {
	return &http.Response{
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func requireEvent(t *testing.T, events <-chan openfeature.Event, timeout time.Duration) openfeature.Event {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(timeout):
		t.Fatal("timed out waiting for provider event")
		return openfeature.Event{}
	}
}

func assertNoEvent(t *testing.T, events <-chan openfeature.Event) {
	t.Helper()
	select {
	case event := <-events:
		t.Fatalf("unexpected provider event: %+v", event)
	default:
	}
}
