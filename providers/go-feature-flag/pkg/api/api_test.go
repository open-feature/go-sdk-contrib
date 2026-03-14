package api_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRoundTripper intercepts HTTP requests and returns a pre-configured response.
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) *http.Response
	err           error
	lastRequest   *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.lastRequest = req
	if m.err != nil {
		return nil, m.err
	}
	return m.roundTripFunc(req), nil
}

func newMockClient(roundTripFunc func(req *http.Request) *http.Response, err error) *http.Client {
	return &http.Client{Transport: &mockRoundTripper{roundTripFunc: roundTripFunc, err: err}}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "testdata", "flag_config_responses", name))
	require.NoError(t, err, "read fixture %s", name)
	return data
}

func okResponse(body []byte, headers http.Header) func(*http.Request) *http.Response {
	return func(_ *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     headers,
			Body:       io.NopCloser(strings.NewReader(string(body))),
		}
	}
}

func okEmptyResponse() func(*http.Request) *http.Response {
	return okResponse([]byte("{}"), http.Header{})
}

// --- Request behaviour tests ------------------------------------------------

func Test_GetConfiguration_RequestBehaviour(t *testing.T) {
	tests := []struct {
		name   string
		api    func(client *http.Client) api.GoFeatureFlagAPI
		flags  []string
		etag   string
		assert func(t *testing.T, req *http.Request)
	}{
		{
			name: "calls configuration endpoint with POST",
			api: func(c *http.Client) api.GoFeatureFlagAPI {
				return *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
					Endpoint:   "http://localhost:1031",
					HTTPClient: c,
				})
			},
			assert: func(t *testing.T, req *http.Request) {
				require.NotNil(t, req)
				assert.Equal(t, http.MethodPost, req.Method)
				assert.Equal(t, "/v1/flag/configuration", req.URL.Path)
			},
		},
		{
			name: "has API key when set",
			api: func(c *http.Client) api.GoFeatureFlagAPI {
				return *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
					Endpoint:   "http://localhost:1031",
					HTTPClient: c,
					APIKey:     "my-api-key",
				})
			},
			assert: func(t *testing.T, req *http.Request) {
				require.NotNil(t, req)
				assert.Equal(t, "my-api-key", req.Header.Get("X-API-Key"))
			},
		},
		{
			name: "does not set API key when empty",
			api: func(c *http.Client) api.GoFeatureFlagAPI {
				return *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
					Endpoint:   "http://localhost:1031",
					HTTPClient: c,
					APIKey:     "",
				})
			},
			assert: func(t *testing.T, req *http.Request) {
				require.NotNil(t, req)
				assert.Empty(t, req.Header.Get("X-API-Key"))
			},
		},
		{
			name: "has default Content-Type header",
			api: func(c *http.Client) api.GoFeatureFlagAPI {
				return *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
					Endpoint:   "http://localhost:1031",
					HTTPClient: c,
				})
			},
			assert: func(t *testing.T, req *http.Request) {
				require.NotNil(t, req)
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
			},
		},
		{
			name: "has custom headers",
			api: func(c *http.Client) api.GoFeatureFlagAPI {
				return *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
					Endpoint:   "http://localhost:1031",
					HTTPClient: c,
					Headers:    map[string]string{"X-Custom-Header": "custom-value"},
				})
			},
			assert: func(t *testing.T, req *http.Request) {
				require.NotNil(t, req)
				assert.Equal(t, "custom-value", req.Header.Get("X-Custom-Header"))
			},
		},
		{
			name: "has If-None-Match header when etag provided",
			api: func(c *http.Client) api.GoFeatureFlagAPI {
				return *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
					Endpoint:   "http://localhost:1031",
					HTTPClient: c,
				})
			},
			etag: "xxxx",
			assert: func(t *testing.T, req *http.Request) {
				require.NotNil(t, req)
				assert.Equal(t, "xxxx", req.Header.Get("If-None-Match"))
			},
		},
		{
			name: "has flags in body when flags provided",
			api: func(c *http.Client) api.GoFeatureFlagAPI {
				return *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
					Endpoint:   "http://localhost:1031",
					HTTPClient: c,
				})
			},
			flags: []string{"flag1", "flag2"},
			assert: func(t *testing.T, req *http.Request) {
				require.NotNil(t, req)
				bodyBytes, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				assert.JSONEq(t, `{"flags":["flag1","flag2"]}`, string(bodyBytes))
			},
		},
		{
			name: "body is empty object when no flags provided",
			api: func(c *http.Client) api.GoFeatureFlagAPI {
				return *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
					Endpoint:   "http://localhost:1031",
					HTTPClient: c,
				})
			},
			assert: func(t *testing.T, req *http.Request) {
				require.NotNil(t, req)
				bodyBytes, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				assert.JSONEq(t, `{}`, string(bodyBytes))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mrt := &mockRoundTripper{roundTripFunc: okEmptyResponse()}
			client := &http.Client{Transport: mrt}
			a := tt.api(client)

			_, _ = a.GetConfiguration(context.Background(), tt.flags, tt.etag)

			tt.assert(t, mrt.lastRequest)
		})
	}
}

// --- Response parsing tests -------------------------------------------------

func Test_GetConfiguration_ResponseParsing(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		headers http.Header
		assert  func(t *testing.T, got *api.FlagConfigResponse)
	}{
		{
			name:    "returns valid response with ETag and Last-Updated when 200 received",
			fixture: "valid-flag-config.json",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("ETag", `"valid-flag-config.json"`)
				h.Set("Last-Updated", "2015-10-21T07:28:00Z")
				return h
			}(),
			assert: func(t *testing.T, got *api.FlagConfigResponse) {
				require.NotNil(t, got)
				assert.Equal(t, `"valid-flag-config.json"`, got.Etag)
				require.NotNil(t, got.LastUpdated)
				assert.Equal(t, "production", got.EvaluationContextEnrichment["env"])
				assert.Contains(t, got.Flags, "TEST")
				assert.Contains(t, got.Flags, "TEST2")
				testFlag := got.Flags["TEST"]
				assert.NotNil(t, testFlag.DefaultRule)
				assert.NotEmpty(t, testFlag.GetVariations())
			},
		},
		{
			name:    "does not set LastUpdated when Last-Updated header is invalid",
			fixture: "valid-flag-config.json",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("ETag", `"valid-flag-config.json"`)
				h.Set("Last-Updated", "not-a-valid-date")
				return h
			}(),
			assert: func(t *testing.T, got *api.FlagConfigResponse) {
				require.NotNil(t, got)
				assert.Nil(t, got.LastUpdated)
				assert.Equal(t, `"valid-flag-config.json"`, got.Etag)
				assert.Contains(t, got.Flags, "TEST")
				assert.Contains(t, got.Flags, "TEST2")
			},
		},
		{
			name:    "unmarshals all types fixture",
			fixture: "valid-all-types.json",
			headers: http.Header{},
			assert: func(t *testing.T, got *api.FlagConfigResponse) {
				require.NotNil(t, got)
				assert.Greater(t, len(got.Flags), 0)
				assert.Contains(t, got.Flags, "bool_targeting_match")
				assert.Contains(t, got.Flags, "string_key")
				assert.Contains(t, got.Flags, "object_key")
				assert.Equal(t, "production", got.EvaluationContextEnrichment["env"])
			},
		},
		{
			name:    "unmarshals scheduled rollout fixture",
			fixture: "valid-scheduled-rollout.json",
			headers: http.Header{},
			assert: func(t *testing.T, got *api.FlagConfigResponse) {
				require.NotNil(t, got)
				assert.Contains(t, got.Flags, "my-flag")
				assert.Contains(t, got.Flags, "my-flag-scheduled-in-future")
				flag := got.Flags["my-flag"]
				require.NotNil(t, flag.Scheduled)
				assert.Len(t, *flag.Scheduled, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := readFixture(t, tt.fixture)
			a := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
				Endpoint:   "http://localhost:1031",
				HTTPClient: newMockClient(okResponse(fixture, tt.headers), nil),
			})

			got, err := a.GetConfiguration(context.Background(), nil, "")
			require.NoError(t, err)
			tt.assert(t, got)
		})
	}
}

// --- Error cases ------------------------------------------------------------

func Test_GetConfiguration_ErrorCases(t *testing.T) {
	tests := []struct {
		name            string
		api             api.GoFeatureFlagAPI
		flags           []string
		etag            string
		wantErrIs       error
		wantErrContains string
	}{
		{
			name: "returns ErrNotModified when 304 received",
			api: *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
				Endpoint: "http://localhost:1031",
				HTTPClient: newMockClient(func(_ *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusNotModified,
						Body:       io.NopCloser(strings.NewReader("")),
					}
				}, nil),
			}),
			etag:      "my-etag",
			wantErrIs: api.ErrNotModified,
		},
		{
			name: "returns error when 500 received",
			api: *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
				Endpoint: "http://localhost:1031",
				HTTPClient: newMockClient(func(_ *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Status:     "500 Internal Server Error",
						Body:       io.NopCloser(strings.NewReader("internal server error")),
					}
				}, nil),
			}),
			wantErrContains: "500",
		},
		{
			name: "returns error when 401 received",
			api: *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
				Endpoint: "http://localhost:1031",
				HTTPClient: newMockClient(func(_ *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Status:     "401 Unauthorized",
						Body:       io.NopCloser(strings.NewReader("unauthorized")),
					}
				}, nil),
			}),
			wantErrContains: "401",
		},
		{
			name: "returns error when 403 received",
			api: *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
				Endpoint: "http://localhost:1031",
				APIKey:   "invalid-api-key",
				HTTPClient: newMockClient(func(_ *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusForbidden,
						Status:     "403 Forbidden",
						Body:       io.NopCloser(strings.NewReader("forbidden")),
					}
				}, nil),
			}),
			wantErrContains: "403",
		},
		{
			name: "returns error when 400 received",
			api: *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
				Endpoint: "http://localhost:1031",
				HTTPClient: newMockClient(func(_ *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Status:     "400 Bad Request",
						Body:       io.NopCloser(strings.NewReader("bad request")),
					}
				}, nil),
			}),
			wantErrContains: "400",
		},
		{
			name: "returns error when transport fails",
			api: *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
				Endpoint:   "http://localhost:1031",
				HTTPClient: newMockClient(nil, errors.New("connection refused")),
			}),
			wantErrContains: "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.api.GetConfiguration(context.Background(), tt.flags, tt.etag)
			assert.Nil(t, got)
			require.Error(t, err)
			if tt.wantErrIs != nil {
				assert.ErrorIs(t, err, tt.wantErrIs)
			}
			if tt.wantErrContains != "" {
				assert.Contains(t, err.Error(), tt.wantErrContains)
			}
		})
	}
}

// --- CollectData tests ------------------------------------------------------

func Test_CollectDataAPI(t *testing.T) {
	type collectDataOptions struct {
		Endpoint             string
		APIKey               string
		ExporterMetadata     map[string]any
		DataCollectorBaseURL string
	}
	type test struct {
		name          string
		wantErr       assert.ErrorAssertionFunc
		options       collectDataOptions
		roundtripFunc func(req *http.Request) *http.Response
		roundtripErr  error
		wantHeaders   http.Header
		wantReqBody   string
		events        []model.CollectableEvent
		wantPath      string // optional: assert request URL path when non-empty
		wantHost      string // optional: assert request URL host when non-empty
	}
	eventsFixture := []model.CollectableEvent{
		model.FeatureEvent{
			Kind:         "feature",
			ContextKind:  "user",
			UserKey:      "ABCD",
			CreationDate: 1722266324,
			Key:          "random-key",
			Variation:    "variationA",
			Value:        "YO",
			Default:      false,
			Version:      "",
			Source:       "SERVER",
		},
		model.FeatureEvent{
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
		},
	}
	wantReqBody := `{"events":[{"kind":"feature","contextKind":"user","userKey":"ABCD","creationDate":1722266324,"key":"random-key","variation":"variationA","value":"YO","default":false,"version":"","source":"SERVER"},{"kind":"feature","contextKind":"user","userKey":"EFGH","creationDate":1722266324,"key":"random-key","variation":"variationA","value":"YO","default":false,"version":"","source":"SERVER"}],"meta":{"openfeature":true,"provider":"go"}}`

	tests := []test{
		{
			name:    "calls data collector endpoint",
			wantErr: assert.NoError,
			options: collectDataOptions{
				Endpoint:         "http://localhost:1031",
				APIKey:           "",
				ExporterMetadata: map[string]any{},
			},
			events: []model.CollectableEvent{},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":0}`))),
				}
			},
			wantHeaders: func() http.Header {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				return headers
			}(),
			wantReqBody: `{"events":[],"meta":{}}`,
			wantPath:    "/v1/data/collector",
		},
		{
			name:    "Valid api call",
			wantErr: assert.NoError,
			options: collectDataOptions{
				Endpoint:         "http://localhost:1031",
				APIKey:           "",
				ExporterMetadata: map[string]any{"openfeature": true, "provider": "go"},
			},
			events: eventsFixture,
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":1}`))),
				}
			},
			wantHeaders: func() http.Header {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				return headers
			}(),
			wantReqBody: wantReqBody,
		},
		{
			name:    "include events and metadata in request body",
			wantErr: assert.NoError,
			options: collectDataOptions{
				Endpoint:         "http://localhost:1031",
				APIKey:           "",
				ExporterMetadata: map[string]any{"env": "production"},
			},
			events: []model.CollectableEvent{
				model.FeatureEvent{
					Kind:         "feature",
					CreationDate: 1750406145,
					ContextKind:  "user",
					Key:          "TEST",
					UserKey:      "642e135a-1df9-4419-a3d3-3c42e0e67509",
					Default:      false,
					Value:        "toto",
					Variation:    "on",
					Version:      "1.0.0",
					Source:       "SERVER",
				},
			},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":1}`))),
				}
			},
			wantHeaders: func() http.Header {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				return headers
			}(),
			wantReqBody: `{"events":[{"kind":"feature","creationDate":1750406145,"contextKind":"user","key":"TEST","userKey":"642e135a-1df9-4419-a3d3-3c42e0e67509","default":false,"value":"toto","variation":"on","version":"1.0.0","source":"SERVER"}],"meta":{"env":"production"}}`,
		},
		{
			name:    "Valid api call with API Key",
			wantErr: assert.NoError,
			options: collectDataOptions{
				Endpoint:         "http://localhost:1031",
				APIKey:           "my-key",
				ExporterMetadata: map[string]any{"openfeature": true, "provider": "go"},
			},
			events: eventsFixture,
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":1}`))),
				}
			},
			wantHeaders: func() http.Header {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				headers.Set("X-API-Key", "my-key")
				return headers
			}(),
			wantReqBody: wantReqBody,
		},
		{
			name:    "Valid api call with TrackingEvent",
			wantErr: assert.NoError,
			options: collectDataOptions{
				Endpoint:         "http://localhost:1031",
				APIKey:           "",
				ExporterMetadata: map[string]any{"openfeature": true, "provider": "go"},
			},
			events: []model.CollectableEvent{
				model.TrackingEvent{
					Kind:              "tracking",
					ContextKind:       "user",
					UserKey:           "ABCD",
					CreationDate:      1722266324,
					Key:               "clicked-checkout",
					EvaluationContext: map[string]any{"targetingKey": "ABCD"},
					TrackingDetails:   map[string]any{"value": 99.99},
				},
			},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":1}`))),
				}
			},
			wantHeaders: func() http.Header {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				return headers
			}(),
			wantReqBody: `{"events":[{"kind":"tracking","contextKind":"user","userKey":"ABCD","creationDate":1722266324,"key":"clicked-checkout","evaluationContext":{"targetingKey":"ABCD"},"trackingEventDetails":{"value":99.99}}],"meta":{"openfeature":true,"provider":"go"}}`,
		},
		{
			name:    "handle tracking events",
			wantErr: assert.NoError,
			options: collectDataOptions{
				Endpoint:         "http://localhost:1031",
				APIKey:           "",
				ExporterMetadata: map[string]any{"env": "production"},
			},
			events: []model.CollectableEvent{
				model.TrackingEvent{
					Kind:            "tracking",
					CreationDate:    1750406145,
					ContextKind:     "user",
					Key:             "TEST2",
					UserKey:         "642e135a-1df9-4419-a3d3-3c42e0e67509",
					TrackingDetails: map[string]any{"action": "click", "label": "button1", "value": float64(1)},
				},
			},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":1}`))),
				}
			},
			wantHeaders: func() http.Header {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				return headers
			}(),
			wantReqBody: `{"events":[{"kind":"tracking","creationDate":1750406145,"contextKind":"user","key":"TEST2","userKey":"642e135a-1df9-4419-a3d3-3c42e0e67509","evaluationContext":null,"trackingEventDetails":{"action":"click","label":"button1","value":1}}],"meta":{"env":"production"}}`,
		},
		{
			name:    "mixed events: 1 feature event and 1 tracking event",
			wantErr: assert.NoError,
			options: collectDataOptions{
				Endpoint:         "http://localhost:1031",
				APIKey:           "",
				ExporterMetadata: map[string]any{"openfeature": true, "provider": "go"},
			},
			events: []model.CollectableEvent{
				model.FeatureEvent{
					Kind:         "feature",
					ContextKind:  "user",
					UserKey:      "ABCD",
					CreationDate: 1722266324,
					Key:          "random-key",
					Variation:    "variationA",
					Value:        "YO",
					Default:      false,
					Version:      "",
					Source:       "SERVER",
				},
				model.TrackingEvent{
					Kind:              "tracking",
					ContextKind:       "user",
					UserKey:           "ABCD",
					CreationDate:      1722266324,
					Key:               "clicked-checkout",
					EvaluationContext: map[string]any{"targetingKey": "ABCD"},
					TrackingDetails:   map[string]any{"value": 99.99},
				},
			},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":2}`))),
				}
			},
			wantHeaders: func() http.Header {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				return headers
			}(),
			wantReqBody: `{"events":[{"kind":"feature","contextKind":"user","userKey":"ABCD","creationDate":1722266324,"key":"random-key","variation":"variationA","value":"YO","default":false,"version":"","source":"SERVER"},{"kind":"tracking","contextKind":"user","userKey":"ABCD","creationDate":1722266324,"key":"clicked-checkout","evaluationContext":{"targetingKey":"ABCD"},"trackingEventDetails":{"value":99.99}}],"meta":{"openfeature":true,"provider":"go"}}`,
		},
		{
			name:    "uses DataCollectorBaseURL when set",
			wantErr: assert.NoError,
			options: collectDataOptions{
				Endpoint:             "http://relay-proxy:1031",
				DataCollectorBaseURL: "http://collector:9000",
				ExporterMetadata:     map[string]any{},
			},
			events: []model.CollectableEvent{},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":0}`))),
				}
			},
			wantHeaders: func() http.Header {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				return headers
			}(),
			wantReqBody: `{"events":[],"meta":{}}`,
			wantPath:    "/v1/data/collector",
			wantHost:    "collector:9000",
		},
		{
			name:    "Request failed",
			wantErr: assert.Error,
			options: collectDataOptions{
				Endpoint: "http://localhost:1031",
			},
			events: []model.CollectableEvent{},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":1}`))),
				}
			},
			roundtripErr: errors.New("request failed"),
		},
		{
			name:    "Request return 400",
			wantErr: assert.Error,
			options: collectDataOptions{
				Endpoint: "http://localhost:1031",
			},
			events: []model.CollectableEvent{},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
				}
			},
		},
		{
			name:    "returns error on 401",
			wantErr: assert.Error,
			options: collectDataOptions{
				Endpoint: "http://localhost:1031",
			},
			events: []model.CollectableEvent{},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader("Unauthorized")),
				}
			},
		},
		{
			name:    "returns error on 403",
			wantErr: assert.Error,
			options: collectDataOptions{
				Endpoint: "http://localhost:1031",
			},
			events: []model.CollectableEvent{},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader("Forbidden")),
				}
			},
		},
		{
			name:    "returns error on 500",
			wantErr: assert.Error,
			options: collectDataOptions{
				Endpoint: "http://localhost:1031",
			},
			events: []model.CollectableEvent{},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("Internal Server Error")),
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mrt := &mockRoundTripper{roundTripFunc: tt.roundtripFunc, err: tt.roundtripErr}
			client := &http.Client{Transport: mrt}

			a := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
				Endpoint:             tt.options.Endpoint,
				HTTPClient:           client,
				APIKey:               tt.options.APIKey,
				ExporterMetadata:     tt.options.ExporterMetadata,
				DataCollectorBaseURL: tt.options.DataCollectorBaseURL,
			})
			err := a.CollectData(tt.events)
			tt.wantErr(t, err)

			if err != nil {
				return
			}

			if tt.wantPath != "" {
				assert.Equal(t, tt.wantPath, mrt.lastRequest.URL.Path)
			}
			if tt.wantHost != "" {
				assert.Equal(t, tt.wantHost, mrt.lastRequest.URL.Host)
			}
			assert.Equal(t, tt.wantHeaders, mrt.lastRequest.Header)

			bodyBytes, err := io.ReadAll(mrt.lastRequest.Body)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantReqBody, string(bodyBytes))
		})
	}
}
