package api_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/consts"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/testutils/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CollectDataAPI(t *testing.T) {
	type test struct {
		name          string
		wantErr       assert.ErrorAssertionFunc
		options       gofeatureflag.ProviderOptions
		roundtripFunc func(req *http.Request) *http.Response
		roundtripErr  error
		wantHeaders   http.Header
		wantReqBody   string
		events        []model.ExportableEvent
	}
	tests := []test{
		{
			name:    "Valid api call",
			wantErr: assert.NoError,
			options: gofeatureflag.ProviderOptions{
				Endpoint:         "http://localhost:1031",
				APIKey:           "",
				ExporterMetadata: map[string]interface{}{"openfeature": true, "provider": "go"},
			},
			events: []model.ExportableEvent{
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
			},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":1}`))),
				}
			},
			wantHeaders: func() http.Header {
				headers := http.Header{}
				headers.Set(consts.ContentTypeHeader, consts.ApplicationJson)
				return headers
			}(),
			wantReqBody: "{\"events\":[{\"kind\":\"feature\",\"contextKind\":\"user\",\"userKey\":\"ABCD\",\"creationDate\":1722266324,\"key\":\"random-key\",\"variation\":\"variationA\",\"value\":\"YO\",\"default\":false,\"version\":\"\",\"source\":\"SERVER\"},{\"kind\":\"feature\",\"contextKind\":\"user\",\"userKey\":\"EFGH\",\"creationDate\":1722266324,\"key\":\"random-key\",\"variation\":\"variationA\",\"value\":\"YO\",\"default\":false,\"version\":\"\",\"source\":\"SERVER\"}],\"meta\":{\"openfeature\":true,\"provider\":\"go\"}}",
		},
		{
			name:    "Valid api call with API Key",
			wantErr: assert.NoError,
			options: gofeatureflag.ProviderOptions{
				Endpoint:         "http://localhost:1031",
				APIKey:           "my-key",
				ExporterMetadata: map[string]interface{}{"openfeature": true, "provider": "go"},
			},
			events: []model.ExportableEvent{
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
			},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":1}`))),
				}
			},
			wantHeaders: func() http.Header {
				headers := http.Header{}
				headers.Set(consts.ContentTypeHeader, consts.ApplicationJson)
				headers.Set(consts.AuthorizationHeader, consts.BearerPrefix+"my-key")
				return headers
			}(),
			wantReqBody: "{\"events\":[{\"kind\":\"feature\",\"contextKind\":\"user\",\"userKey\":\"ABCD\",\"creationDate\":1722266324,\"key\":\"random-key\",\"variation\":\"variationA\",\"value\":\"YO\",\"default\":false,\"version\":\"\",\"source\":\"SERVER\"},{\"kind\":\"feature\",\"contextKind\":\"user\",\"userKey\":\"EFGH\",\"creationDate\":1722266324,\"key\":\"random-key\",\"variation\":\"variationA\",\"value\":\"YO\",\"default\":false,\"version\":\"\",\"source\":\"SERVER\"}],\"meta\":{\"openfeature\":true,\"provider\":\"go\"}}",
		},
		{
			name:    "Request failed",
			wantErr: assert.Error,
			options: gofeatureflag.ProviderOptions{
				Endpoint: "http://localhost:1031",
			},
			events: []model.ExportableEvent{},
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
			options: gofeatureflag.ProviderOptions{
				Endpoint: "http://localhost:1031",
			},
			events: []model.ExportableEvent{},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
				}
			},
		},
		{
			name:    "Valid api call with TrackingEvent",
			wantErr: assert.NoError,
			options: gofeatureflag.ProviderOptions{
				Endpoint:         "http://localhost:1031",
				APIKey:           "",
				ExporterMetadata: map[string]interface{}{"openfeature": true, "provider": "go"},
			},
			events: []model.ExportableEvent{
				model.TrackingEvent{
					Kind:         "tracking",
					ContextKind:  "anonymousUser",
					UserKey:      "ABCD",
					CreationDate: 1722266324,
					Key:          "random-key",
					EvaluationContext: map[string]interface{}{
						"anonymous":    true,
						"targetingKey": "ABCD",
					},
					TrackingDetails: map[string]interface{}{
						"event": "123",
					},
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
			},
			roundtripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"ingestedContentCount":1}`))),
				}
			},
			wantHeaders: func() http.Header {
				headers := http.Header{}
				headers.Set(consts.ContentTypeHeader, consts.ApplicationJson)
				return headers
			}(),
			wantReqBody: "{\"events\":[{\"kind\":\"tracking\",\"contextKind\":\"anonymousUser\",\"userKey\":\"ABCD\",\"creationDate\":1722266324,\"key\":\"random-key\",\"evaluationContext\":{\"anonymous\":true,\"targetingKey\":\"ABCD\"},\"trackingEventDetails\":{\"event\":\"123\"}},{\"kind\":\"feature\",\"contextKind\":\"user\",\"userKey\":\"EFGH\",\"creationDate\":1722266324,\"key\":\"random-key\",\"variation\":\"variationA\",\"value\":\"YO\",\"default\":false,\"version\":\"\",\"source\":\"SERVER\"}],\"meta\":{\"openfeature\":true,\"provider\":\"go\"}}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mrt := mock.RoundTripper{RoundTripFunc: tt.roundtripFunc, Err: tt.roundtripErr}
			client := &http.Client{Transport: &mrt}

			options := tt.options
			options.HTTPClient = client
			g := api.NewGoffAPI(options)
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
