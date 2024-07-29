package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/provider_v2/model"
	"net/http"
	"net/url"
	"path"
)

type GoFeatureFlagApiOptions struct {
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
	// DataFlushInterval (optional) interval time we use to call the relay proxy to collect data.
	// The parameter is used only if the cache is enabled, otherwise the collection of the data is done directly
	// when calling the evaluation API.
	// default: 1 minute
}

type GoFeatureFlagAPI struct {
	options GoFeatureFlagApiOptions
}

func NewGoFeatureFlagAPI(options GoFeatureFlagApiOptions) *GoFeatureFlagAPI {
	return &GoFeatureFlagAPI{options: options}
}

func (g *GoFeatureFlagAPI) CollectData(events []model.FeatureEvent) error {
	u, _ := url.Parse(g.options.Endpoint)
	u.Path = path.Join(u.Path, "v1", "/")
	u.Path = path.Join(u.Path, "data", "/")
	u.Path = path.Join(u.Path, "collector", "/")
	reqBody := model.DataCollectorRequest{
		Events: events,
		Meta:   map[string]string{"provider": "go", "openfeature": "true"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	if g.options.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+g.options.APIKey)
	}

	// Select the HTTP Client
	client := g.options.HTTPClient
	if client == nil {
		client = DefaultHTTPClient()
	}

	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status: %v", response.Status)
	}
	return nil
}
