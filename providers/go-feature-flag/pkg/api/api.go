package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
)

const (
	contentTypeHeader = "Content-Type"
	apiKeyHeader      = "X-API-Key"
	ifNoneMatchHeader = "If-None-Match"
	applicationJSON   = "application/json"
)

// GoFeatureFlagAPI is a low-level HTTP client for the GO Feature Flag relay proxy.
type GoFeatureFlagAPI struct {
	// Endpoint is the base URL of the relay proxy (e.g. "http://localhost:1031").
	Endpoint string

	// HTTPClient is the HTTP client used for requests.
	// When nil, http.DefaultClient is used.
	HTTPClient *http.Client

	// APIKey is an optional bearer token for relay proxy authentication.
	// When set it is sent as "Authorization: Bearer <APIKey>".
	APIKey string

	// Headers holds additional HTTP headers to include in every request.
	Headers map[string]string

	// ExporterMetadata is metadata to be sent with every data collection request.
	ExporterMetadata map[string]any
}

// NewGoFeatureFlagAPI creates a new GoFeatureFlagAPI with the given endpoint and HTTP client.
// Pass nil for httpClient to use http.DefaultClient.
func NewGoFeatureFlagAPI(endpoint string, ExporterMetadata map[string]any, httpClient *http.Client) *GoFeatureFlagAPI {
	return &GoFeatureFlagAPI{
		Endpoint:   endpoint,
		HTTPClient: httpClient,
	}
}

// GetConfiguration calls POST /v1/flag/configuration on the relay proxy and returns the
// parsed flag configuration.
//
// flags is an optional list of flag keys to retrieve. Pass nil or an empty slice to retrieve
// all flags.
//
// Pass a non-empty etag string (previously obtained from FlagConfigResponse.Etag) to enable
// conditional requests: when the configuration has not changed the relay proxy responds with
// HTTP 304 and this method returns (nil, ErrNotModified).
func (a *GoFeatureFlagAPI) GetConfiguration(ctx context.Context, flags []string, etag string) (*FlagConfigResponse, error) {
	u, err := url.Parse(a.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetConfiguration: invalid endpoint %q: %w", a.Endpoint, err)
	}
	u.Path = path.Join(u.Path, "v1", "flag", "configuration")

	body, err := json.Marshal(FlagConfigurationRequest{Flags: flags})
	if err != nil {
		return nil, fmt.Errorf("GetConfiguration: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("GetConfiguration: create request: %w", err)
	}
	a.setHeaders(httpReq, etag)

	httpClient := a.getHttpClient()
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("GetConfiguration: HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		return parseConfigurationSuccessResponse(resp)
	case http.StatusNotModified:
		return nil, ErrNotModified
	default:
		rawBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GetConfiguration: request failed with status %s: %s", resp.Status, rawBody)
	}
}

// CollectData sends a collection of events to the GO Feature Flag relay proxy's data collector endpoint.
// Each element of events must be model.FeatureEvent or model.TrackingEvent.
// It serializes the provided events and associated exporter metadata into a JSON payload,
// sends an HTTP POST request, and checks for a successful response.
// Returns an error if marshalling, request creation, the HTTP call, or the status code indicate a failure.
func (a *GoFeatureFlagAPI) CollectData(events []model.CollectableEvent) error {
	u, _ := url.Parse(a.Endpoint)
	u.Path = path.Join(u.Path, "v1", "data", "collector")

	reqBody := model.DataCollectorRequest{
		Events: events,
		Meta:   a.ExporterMetadata,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	a.setHeaders(req, "")

	response, err := a.getHttpClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status: %v", response.Status)
	}
	return nil
}

// setHeaders sets the standard headers on req. It always sets Content-Type, conditionally
// adds the API key and If-None-Match headers, and then applies any caller-supplied headers.
func (a *GoFeatureFlagAPI) setHeaders(req *http.Request, etag string) {
	req.Header.Set(contentTypeHeader, applicationJSON)
	if a.APIKey != "" {
		req.Header.Set(apiKeyHeader, a.APIKey)
	}
	if etag != "" {
		req.Header.Set(ifNoneMatchHeader, etag)
	}
	for k, v := range a.Headers {
		req.Header.Set(k, v)
	}
}

// parseConfigurationSuccessResponse reads and unmarshals a successful (200) relay proxy response.
func parseConfigurationSuccessResponse(resp *http.Response) (*FlagConfigResponse, error) {
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GetConfiguration: read response body: %w", err)
	}

	var result FlagConfigResponse
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return nil, fmt.Errorf("GetConfiguration: unmarshal response: %w", err)
	}
	result.Etag = resp.Header.Get("ETag")
	lastUpdatedHeader := resp.Header.Get("Last-Updated")
	if lastUpdatedHeader != "" {
		if t, err := time.Parse(time.RFC3339, lastUpdatedHeader); err == nil {
			result.LastUpdated = &t
		}
	}
	return &result, nil
}

// getHttpClient returns the HTTP Client to use for the request.
func (a *GoFeatureFlagAPI) getHttpClient() *http.Client {
	client := a.HTTPClient
	if client == nil {
		client = DefaultHTTPClient()
	}
	return client
}
