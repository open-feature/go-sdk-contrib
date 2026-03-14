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

type GoFeatureFlagAPIOptions struct {
	Endpoint             string
	DataCollectorBaseURL string
	HTTPClient           *http.Client
	APIKey               string
	Headers              map[string]string
	ExporterMetadata     map[string]any
}

// GoFeatureFlagAPI is a low-level HTTP client for the GO Feature Flag relay proxy.
type GoFeatureFlagAPI struct {
	// Endpoint is the base URL of the relay proxy (e.g. "http://localhost:1031").
	endpoint string

	// HTTPClient is the HTTP client used for requests.
	// When nil, http.DefaultClient is used.
	httpClient *http.Client

	// APIKey is an optional API key for relay proxy authentication.
	// When set it is sent as "X-API-Key: <APIKey>".
	apiKey string

	// Headers holds additional HTTP headers to include in every request.
	headers map[string]string

	// ExporterMetadata is metadata to be sent with every data collection request.
	exporterMetadata map[string]any

	// DataCollectorBaseURL (optional) overrides the base URL used for data collection.
	// When empty, Endpoint is used.
	dataCollectorBaseURL string
}

// NewGoFeatureFlagAPI creates a new GoFeatureFlagAPI with the given endpoint and HTTP client.
// Pass nil for httpClient to use http.DefaultClient.
func NewGoFeatureFlagAPI(options GoFeatureFlagAPIOptions) *GoFeatureFlagAPI {
	return &GoFeatureFlagAPI{
		endpoint:             options.Endpoint,
		httpClient:           options.HTTPClient,
		apiKey:               options.APIKey,
		headers:              options.Headers,
		exporterMetadata:     options.ExporterMetadata,
		dataCollectorBaseURL: options.DataCollectorBaseURL,
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
	u, err := url.Parse(a.endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetConfiguration: invalid endpoint %q: %w", a.endpoint, err)
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
		rawBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("GetConfiguration: request failed with status %s, and could not read body: %w", resp.Status, readErr)
		}
		return nil, fmt.Errorf("GetConfiguration: request failed with status %s: %s", resp.Status, rawBody)
	}
}

// CollectData sends a collection of events to the GO Feature Flag relay proxy's data collector endpoint.
// Each element of events must be model.FeatureEvent or model.TrackingEvent.
// It serializes the provided events and associated exporter metadata into a JSON payload,
// sends an HTTP POST request, and checks for a successful response.
// Returns an error if marshalling, request creation, the HTTP call, or the status code indicate a failure.
func (a *GoFeatureFlagAPI) CollectData(events []model.CollectableEvent) error {
	effectiveEndpoint := a.endpoint
	if a.dataCollectorBaseURL != "" {
		effectiveEndpoint = a.dataCollectorBaseURL
	}
	u, err := url.Parse(effectiveEndpoint)
	if err != nil {
		return fmt.Errorf("CollectData: invalid endpoint %q: %w", effectiveEndpoint, err)
	}
	u.Path = path.Join(u.Path, "v1", "data", "collector")

	reqBody := model.DataCollectorRequest{
		Events: events,
		Meta:   a.exporterMetadata,
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
	if a.apiKey != "" {
		req.Header.Set(apiKeyHeader, a.apiKey)
	}
	if etag != "" {
		req.Header.Set(ifNoneMatchHeader, etag)
	}
	for k, v := range a.headers {
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
	client := a.httpClient
	if client == nil {
		client = DefaultHTTPClient()
	}
	return client
}
