package controller_test

import (
	"bytes"
	"errors"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/provider_v2/controller"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/provider_v2/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func Test_CollectDataAPI(t *testing.T) {
	type test struct {
		name          string
		wantErr       assert.ErrorAssertionFunc
		options       controller.GoFeatureFlagApiOptions
		roundtripFunc func(req *http.Request) *http.Response
		roundtripErr  error
		wantHeaders   http.Header
		wantReqBody   string
		events        []model.FeatureEvent
	}
	tests := []test{
		{
			name:    "Valid api call",
			wantErr: assert.NoError,
			options: controller.GoFeatureFlagApiOptions{
				Endpoint: "http://localhost:1031",
				APIKey:   "",
			},
			events: []model.FeatureEvent{
				{
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
				{
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
			wantReqBody: "{\"events\":[{\"kind\":\"feature\",\"contextKind\":\"user\",\"userKey\":\"ABCD\",\"creationDate\":1722266324,\"key\":\"random-key\",\"variation\":\"variationA\",\"value\":\"YO\",\"default\":false,\"version\":\"\",\"source\":\"SERVER\"},{\"kind\":\"feature\",\"contextKind\":\"user\",\"userKey\":\"EFGH\",\"creationDate\":1722266324,\"key\":\"random-key\",\"variation\":\"variationA\",\"value\":\"YO\",\"default\":false,\"version\":\"\",\"source\":\"SERVER\"}],\"meta\":{\"openfeature\":\"true\",\"provider\":\"go\"}}",
		},
		{
			name:    "Valid api call with API Key",
			wantErr: assert.NoError,
			options: controller.GoFeatureFlagApiOptions{
				Endpoint: "http://localhost:1031",
				APIKey:   "my-key",
			},
			events: []model.FeatureEvent{
				{
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
				{
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
				headers.Set("Authorization", "Bearer my-key")
				return headers
			}(),
			wantReqBody: "{\"events\":[{\"kind\":\"feature\",\"contextKind\":\"user\",\"userKey\":\"ABCD\",\"creationDate\":1722266324,\"key\":\"random-key\",\"variation\":\"variationA\",\"value\":\"YO\",\"default\":false,\"version\":\"\",\"source\":\"SERVER\"},{\"kind\":\"feature\",\"contextKind\":\"user\",\"userKey\":\"EFGH\",\"creationDate\":1722266324,\"key\":\"random-key\",\"variation\":\"variationA\",\"value\":\"YO\",\"default\":false,\"version\":\"\",\"source\":\"SERVER\"}],\"meta\":{\"openfeature\":\"true\",\"provider\":\"go\"}}",
		},
		{
			name:    "Request failed",
			wantErr: assert.Error,
			options: controller.GoFeatureFlagApiOptions{
				Endpoint: "http://localhost:1031",
			},
			events: []model.FeatureEvent{},
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
			options: controller.GoFeatureFlagApiOptions{
				Endpoint: "http://localhost:1031",
			},
			events: []model.FeatureEvent{},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mrt := MockRoundTripper{RoundTripFunc: tt.roundtripFunc, Err: tt.roundtripErr}
			client := &http.Client{Transport: &mrt}

			options := tt.options
			options.HTTPClient = client
			g := controller.NewGoFeatureFlagAPI(options)
			err := g.CollectData(tt.events)
			tt.wantErr(t, err)

			if err != nil {
				return
			}

			assert.Equal(t, tt.wantHeaders, mrt.GetLastRequest().Header)

			bodyBytes, err := io.ReadAll(mrt.GetLastRequest().Body)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantReqBody, string(bodyBytes))
		})
	}
}

func Test_ConfigurationHasChanged(t *testing.T) {
	t.Run("Initial configuration call", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
			}
		}}
		client := &http.Client{Transport: &mrt}
		options := controller.GoFeatureFlagApiOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		}
		g := controller.NewGoFeatureFlagAPI(options)
		status, err := g.ConfigurationHasChanged()
		require.NoError(t, err)
		assert.Equal(t, controller.FlagConfigurationInitialized, status)
	})

	t.Run("Change in the configuration", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			if req.Header.Get("If-None-Match") == "123456" {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Header:     map[string][]string{},
				}
				resp.Header.Set("ETag", "78910")
				return resp
			}
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
			}
			resp.Header.Set("ETag", "123456")
			return resp
		}}
		client := &http.Client{Transport: &mrt}
		options := controller.GoFeatureFlagApiOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		}
		g := controller.NewGoFeatureFlagAPI(options)
		status, err := g.ConfigurationHasChanged()
		require.NoError(t, err)
		assert.Equal(t, controller.FlagConfigurationInitialized, status)
		status, err = g.ConfigurationHasChanged()
		require.NoError(t, err)
		assert.Equal(t, controller.FlagConfigurationUpdated, status)
	})

	t.Run("No change in the configuration", func(t *testing.T) {
		mrt := MockRoundTripper{RoundTripFunc: func(req *http.Request) *http.Response {
			if req.Header.Get("If-None-Match") == "123456" {
				resp := &http.Response{
					StatusCode: http.StatusNotModified,
				}
				return resp
			}
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
			}
			resp.Header.Set("ETag", "123456")
			return resp
		}}
		client := &http.Client{Transport: &mrt}
		options := controller.GoFeatureFlagApiOptions{
			Endpoint:   "http://localhost:1031",
			HTTPClient: client,
		}
		g := controller.NewGoFeatureFlagAPI(options)
		status, err := g.ConfigurationHasChanged()
		require.NoError(t, err)
		assert.Equal(t, controller.FlagConfigurationInitialized, status)
		status, err = g.ConfigurationHasChanged()
		require.NoError(t, err)
		assert.Equal(t, controller.FlagConfigurationNotChanged, status)
	})
}

type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) *http.Response
	Err           error
	LastRequest   *http.Request
	NumberCall    int
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.LastRequest = req
	m.NumberCall++
	return m.RoundTripFunc(req), m.Err
}

// NewMockClient creates a new http.Client with the mock RoundTripper.
func NewMockClient(roundTripFunc func(req *http.Request) *http.Response, err error) *http.Client {
	return &http.Client{
		Transport: &MockRoundTripper{RoundTripFunc: roundTripFunc, Err: err},
	}
}

// GetLastRequest returns the last request made by the mock client.
func (m *MockRoundTripper) GetLastRequest() *http.Request {
	return m.LastRequest
}
