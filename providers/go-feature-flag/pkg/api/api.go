package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/consts"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/gofferror"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
)

type GoffAPIOptions struct {
	// Endpoint contains the DNS of your GO Feature Flag relay proxy (ex: http://localhost:1031)
	Endpoint string

	// HTTPClient (optional) is the HTTP Client we will use to contact GO Feature Flag.
	// By default, we are using a custom HTTPClient with a timeout configure to 10000 milliseconds.
	HTTPClient *http.Client

	// APIKey  (optional) If the relay proxy is configured to authenticate the requests, you should provide
	// an API Key to the provider. Please ask the administrator of the relay proxy to provide an API Key.
	// (This feature is available only if you are using GO Feature Flag relay proxy v1.7.0 or above)
	// Default: null
	APIKey string

	// DataCollectorEndpoint (optional) endpoint to send the data collector events to.
	// If not set, we will use the same endpoint as the provider endpoint.
	DataCollectorEndpoint string

	// ExporterMetadata (optional) is the metadata we send to the GO Feature Flag relay proxy when we report the
	// evaluation data usage.
	ExporterMetadata map[string]any
}

// GoffAPI is the API layer to access GO Feature Flag relay proxy.
type GoffAPI struct {
	options GoffAPIOptions
}

// NewGoffAPI creates a new GoffAPI instance.
func NewGoffAPI(options GoffAPIOptions) GoffAPI {
	return GoffAPI{options: options}
}

// CollectData collects data from the GO Feature Flag relay proxy.
// Events must be a slice of ExportableEvent.
func (g *GoffAPI) CollectData(events []model.ExportableEvent) error {
	endpoint := g.options.DataCollectorEndpoint
	if endpoint == "" {
		endpoint = g.options.Endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, "v1", "data", "collector")

	eventsMap := make([]map[string]any, len(events))
	for i, event := range events {
		var err error
		eventsMap[i], err = event.ToMap()
		if err != nil {
			return fmt.Errorf("error converting event to map: %w", err)
		}
	}

	reqBody := model.DataCollectorRequest{
		Events: eventsMap,
		Meta:   g.options.ExporterMetadata,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	fmt.Println("jsonData", string(jsonData))

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set(consts.ContentTypeHeader, consts.ApplicationJson)
	if g.options.APIKey != "" {
		req.Header.Set(consts.AuthorizationHeader, consts.BearerPrefix+g.options.APIKey)
	}

	response, err := g.getHttpClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status: %v", response.Status)
	}
	return nil
}

// RetrieveFlagConfiguration retrieves the flag configuration from the GO Feature Flag API.
// etag: If provided, we call the API with "If-None-Match" header.
// flags: List of flags to retrieve, if not set or empty, we will retrieve all available flags.
// Returns a FlagConfigResponse with the flag configuration or an error.
func (g *GoffAPI) RetrieveFlagConfiguration(etag string, flags []string) (*model.FlagConfigResponse, error) {
	u, err := url.Parse(g.options.Endpoint)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "v1", "flag", "configuration")
	reqBody := model.FlagConfigRequest{Flags: flags}

	bodyStr, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(bodyStr))
	if err != nil {
		return nil, err
	}
	req.Header.Set(consts.ContentTypeHeader, consts.ApplicationJson)
	if etag != "" {
		req.Header.Set(consts.IfNoneMatchHeader, etag)
	}
	if g.options.APIKey != "" {
		req.Header.Set(consts.AuthorizationHeader, consts.BearerPrefix+g.options.APIKey)
	}

	response, err := g.getHttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	switch response.StatusCode {
	case http.StatusOK:
		return g.handleFlagConfigurationSuccess(response)
	case http.StatusNotModified:
		// Configuration has not changed
		lastUpdated, err := time.Parse(time.RFC3339, response.Header.Get(consts.LastModifiedHeader))
		if err != nil {
			// default to zero time if parsing fails
			lastUpdated = time.Time{}
		}
		return &model.FlagConfigResponse{Etag: response.Header.Get(consts.ETagHeader), LastUpdated: lastUpdated}, nil
	case http.StatusNotFound:
		return nil, gofferror.NewFlagConfigurationEndpointNotFoundError()
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, gofferror.NewUnauthorizedError(
			"Impossible to retrieve flag configuration: authentication/authorization error")
	case http.StatusBadRequest:
		body, _ := io.ReadAll(response.Body)
		return nil, gofferror.NewImpossibleToRetrieveConfigurationError(
			fmt.Sprintf("retrieve flag configuration error: Bad request: %s", string(body)))
	default:
		body, _ := io.ReadAll(response.Body)
		return nil, gofferror.NewImpossibleToRetrieveConfigurationError(
			fmt.Sprintf("retrieve flag configuration error: unexpected http code %d: %s",
				response.StatusCode, string(body)))
	}
}

// handleFlagConfigurationSuccess handles a successful response from the flag configuration endpoint.
func (g *GoffAPI) handleFlagConfigurationSuccess(response *http.Response) (*model.FlagConfigResponse, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, gofferror.NewImpossibleToRetrieveConfigurationError(
			fmt.Sprintf("error reading response body: %v", err))
	}

	var flagConfig model.FlagConfigResponse
	if err := json.Unmarshal(body, &flagConfig); err != nil {
		return nil, gofferror.NewImpossibleToRetrieveConfigurationError(
			fmt.Sprintf("error unmarshaling response: %v", err))
	}

	flagConfig.Etag = response.Header.Get(consts.ETagHeader)
	lastUpdated, err := time.Parse(time.RFC3339, response.Header.Get(consts.LastModifiedHeader))
	if err != nil {
		lastUpdated = time.Time{}
	}
	flagConfig.LastUpdated = lastUpdated
	return &flagConfig, nil
}

// getHttpClient returns the HTTP Client to use for the request.
func (g *GoffAPI) getHttpClient() *http.Client {
	client := g.options.HTTPClient
	if client == nil {
		client = consts.DefaultHTTPClient
	}
	return client
}
