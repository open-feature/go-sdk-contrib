package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/consts"
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

// getHttpClient returns the HTTP Client to use for the request.
func (g *GoffAPI) getHttpClient() *http.Client {
	client := g.options.HTTPClient
	if client == nil {
		client = consts.DefaultHTTPClient
	}
	return client
}
