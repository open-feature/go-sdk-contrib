package harness_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	harness "github.com/harness/ff-golang-server-sdk/client"
	"github.com/harness/ff-golang-server-sdk/rest"
	"github.com/harness/ff-golang-server-sdk/test_helpers"
	"github.com/jarcoal/httpmock"
	harnessProvider "github.com/open-feature/go-sdk-contrib/providers/harness/pkg"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/stretchr/testify/require"
)

// based on Harness test: https://github.com/harness/ff-golang-server-sdk/blob/main/client/client_test.go

const (
	ValidSDKKey   = "27bed8d2-2610-462b-90eb-d80fd594b623"
	EmptySDKKey   = ""
	InvaliDSDKKey = "an invalid flagIdentifier"
	URL           = "http://localhost/api/1.0"

	//nolint
	ValidAuthToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwcm9qZWN0IjoiMTA0MjM5NzYtODQ1MS00NmZjLTg2NzctYmNiZDM3MTA3M2JhIiwiZW52aXJvbm1lbnQiOiI3ZWQxMDI1ZC1hOWIxLTQxMjktYTg4Zi1lMjdlZjM2MDk4MmQiLCJwcm9qZWN0SWRlbnRpZmllciI6IiIsImVudmlyb25tZW50SWRlbnRpZmllciI6IlByZVByb2R1Y3Rpb24iLCJhY2NvdW50SUQiOiIiLCJvcmdhbml6YXRpb24iOiIwMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDAiLCJjbHVzdGVySWRlbnRpZmllciI6IjEifQ.E4O_u42HkR0q4AwTTViFTCnNa89Kwftks7Gh-GvQfuE"
)

// responderQueue is a type that manages a queue of responders
type responderQueue struct {
	responders []httpmock.Responder
	index      int
}

// newResponderQueue creates a new instance of responderQueue with the provided responders
func newResponderQueue(responders []httpmock.Responder) *responderQueue {
	return &responderQueue{
		responders: responders,
		index:      0,
	}
}

// getNextResponder is a method that returns the next responder in the queue
func (q *responderQueue) getNextResponder(req *http.Request) (*http.Response, error) {
	if q.index >= len(q.responders) {
		// Stop running tests as the input is invalid at this stage.
		log.Fatal("Not enough responders provided to the test function being executed")
	}
	responder := q.responders[q.index]
	q.index++
	return responder(req)
}

// TestMain runs before the other tests
func TestMain(m *testing.M) {
	// httpMock overwrites the http.DefaultClient
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	os.Exit(m.Run())
}

func registerResponders(authResponder httpmock.Responder, targetSegmentsResponder httpmock.Responder, featureConfigsResponder httpmock.Responder) {
	httpmock.RegisterResponder("POST", "http://localhost/api/1.0/client/auth", authResponder)
	httpmock.RegisterResponder("GET", "http://localhost/api/1.0/client/env/7ed1025d-a9b1-4129-a88f-e27ef360982d/target-segments", targetSegmentsResponder)
	httpmock.RegisterResponder("GET", "http://localhost/api/1.0/client/env/7ed1025d-a9b1-4129-a88f-e27ef360982d/feature-configs", featureConfigsResponder)
}

// Same as registerResponders except the auth response can be different per call
func registerMultipleResponseResponders(authResponder []httpmock.Responder, targetSegmentsResponder httpmock.Responder, featureConfigsResponder httpmock.Responder) {
	authQueue := newResponderQueue(authResponder)
	httpmock.RegisterResponder("POST", "http://localhost/api/1.0/client/auth", authQueue.getNextResponder)

	// These responders don't need different responses per call
	httpmock.RegisterResponder("GET", "http://localhost/api/1.0/client/env/7ed1025d-a9b1-4129-a88f-e27ef360982d/target-segments", targetSegmentsResponder)
	httpmock.RegisterResponder("GET", "http://localhost/api/1.0/client/env/7ed1025d-a9b1-4129-a88f-e27ef360982d/feature-configs", featureConfigsResponder)
}

func TestBooleanEvaluation(t *testing.T) {
	authSuccessResponse := AuthResponse(200, ValidAuthToken)
	registerResponders(authSuccessResponse, TargetSegmentsResponse, FeatureConfigsResponse)

	providerConfig := harnessProvider.ProviderConfig{
		Options: []harness.ConfigOption{
			harness.WithWaitForInitialized(true),
			harness.WithURL(URL),
			harness.WithStreamEnabled(false),
			harness.WithHTTPClient(http.DefaultClient),
			harness.WithStoreEnabled(false),
		},
		SdkKey: ValidSDKKey,
	}

	provider, err := harnessProvider.NewProvider(providerConfig)
	if err != nil {
		t.Fail()
	}
	err = provider.Init(of.EvaluationContext{})
	if err != nil {
		t.Fail()
	}

	ctx := context.Background()

	target := map[string]interface{}{
		of.TargetingKey: "john",
		"Firstname":     "John",
		"Lastname":      "Doe",
		"Email":         "john@doe.com",
	}

	resolution := provider.BooleanEvaluation(ctx, "TestTrueOn", false, target)
	if resolution.Value != true {
		t.Fatalf("Expected one of the variant payloads")
	}

	t.Run("evalCtx empty", func(t *testing.T) {
		resolution := provider.BooleanEvaluation(ctx, "MadeUpIDontExist", false, nil)
		require.Equal(t, false, resolution.Value)
	})

	of.SetProvider(provider)
	ofClient := of.NewClient("my-app")

	evalCtx := of.NewEvaluationContext(
		"john",
		map[string]interface{}{
			"Firstname": "John",
			"Lastname":  "Doe",
			"Email":     "john@doe.com",
		},
	)
	enabled, err := ofClient.BooleanValue(context.Background(), "TestTrueOn", false, evalCtx)
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}

}

func TestStringEvaluation(t *testing.T) {
	authSuccessResponse := AuthResponse(200, ValidAuthToken)
	registerResponders(authSuccessResponse, TargetSegmentsResponse, FeatureConfigsResponse)

	providerConfig := harnessProvider.ProviderConfig{
		Options: []harness.ConfigOption{
			harness.WithWaitForInitialized(true),
			harness.WithURL(URL),
			harness.WithStreamEnabled(false),
			harness.WithHTTPClient(http.DefaultClient),
			harness.WithStoreEnabled(false),
		},
		SdkKey: ValidSDKKey,
	}

	provider, err := harnessProvider.NewProvider(providerConfig)
	if err != nil {
		t.Fail()
	}
	err = provider.Init(of.EvaluationContext{})
	if err != nil {
		t.Fail()
	}

	ctx := context.Background()

	target := map[string]interface{}{
		"Identifier": "john",
		"Firstname":  "John",
		"Lastname":   "Doe",
		"Email":      "john@doe.com",
	}

	resolution := provider.StringEvaluation(ctx, "TestStringAOn", "foo", target)
	if resolution.Value != "A" {
		t.Fatalf("Expected one of the variant payloads")
	}

	resolution = provider.StringEvaluation(ctx, "TestStringAOff", "foo", target)
	if resolution.Value != "B" {
		t.Fatalf("Expected one of the variant payloads")
	}

	resolution = provider.StringEvaluation(ctx, "TestStringAOnWithPreReqFalse", "foo", target)
	if resolution.Value != "B" {
		t.Fatalf("Expected one of the variant payloads")
	}

	resolution = provider.StringEvaluation(ctx, "TestStringAOnWithPreReqTrue", "foo", target)
	if resolution.Value != "A" {
		t.Fatalf("Expected one of the variant payloads")
	}

}

var AuthResponse = func(statusCode int, authToken string) func(req *http.Request) (*http.Response, error) {

	return func(req *http.Request) (*http.Response, error) {
		// Return the appropriate error based on the provided status code
		return httpmock.NewJsonResponse(statusCode, rest.AuthenticationResponse{
			AuthToken: authToken})
	}
}

var AuthResponseDetailed = func(statusCode int, status string, bodyString string) func(req *http.Request) (*http.Response, error) {

	return func(req *http.Request) (*http.Response, error) {
		response := &http.Response{
			StatusCode: statusCode,
			Status:     status,
			Body:       io.NopCloser(bytes.NewReader([]byte(bodyString))),
			Header:     make(http.Header),
		}

		response.Header.Add("Content-Type", "application/json")

		return response, nil
	}
}

var TargetSegmentsResponse = func(req *http.Request) (*http.Response, error) {
	var AllSegmentsResponse []rest.Segment

	err := json.Unmarshal([]byte(`[
		{
			"environment": "PreProduction",
			"excluded": [],
			"identifier": "Beta_Users",
			"included": [
				{
					"identifier": "john",
					"name": "John",
				},
				{
					"identifier": "paul",
					"name": "Paul",
				}
			],
			"name": "Beta Users"
		}
	]`), &AllSegmentsResponse)
	if err != nil {
		return test_helpers.JsonError(err)
	}
	return httpmock.NewJsonResponse(200, AllSegmentsResponse)
}

var FeatureConfigsResponse = func(req *http.Request) (*http.Response, error) {
	var FeatureConfigResponse []rest.FeatureConfig
	FeatureConfigResponse = append(FeatureConfigResponse, test_helpers.MakeBoolFeatureConfigs("TestTrueOn", "true", "false", "on")...)
	FeatureConfigResponse = append(FeatureConfigResponse, test_helpers.MakeBoolFeatureConfigs("TestTrueOff", "true", "false", "off")...)

	FeatureConfigResponse = append(FeatureConfigResponse, test_helpers.MakeBoolFeatureConfigs("TestFalseOn", "false", "true", "on")...)
	FeatureConfigResponse = append(FeatureConfigResponse, test_helpers.MakeBoolFeatureConfigs("TestFalseOff", "false", "true", "off")...)

	FeatureConfigResponse = append(FeatureConfigResponse, test_helpers.MakeBoolFeatureConfigs("TestTrueOnWithPreReqFalse", "true", "false", "on", test_helpers.MakeBoolPreRequisite("PreReq1", "false"))...)
	FeatureConfigResponse = append(FeatureConfigResponse, test_helpers.MakeBoolFeatureConfigs("TestTrueOnWithPreReqTrue", "true", "false", "on", test_helpers.MakeBoolPreRequisite("PreReq1", "true"))...)

	FeatureConfigResponse = append(FeatureConfigResponse, test_helpers.MakeStringFeatureConfigs("TestStringAOn", "Alpha", "Bravo", "on")...)
	FeatureConfigResponse = append(FeatureConfigResponse, test_helpers.MakeStringFeatureConfigs("TestStringAOff", "Alpha", "Bravo", "off")...)

	FeatureConfigResponse = append(FeatureConfigResponse, test_helpers.MakeStringFeatureConfigs("TestStringAOnWithPreReqFalse", "Alpha", "Bravo", "on", test_helpers.MakeBoolPreRequisite("PreReq1", "false"))...)
	FeatureConfigResponse = append(FeatureConfigResponse, test_helpers.MakeStringFeatureConfigs("TestStringAOnWithPreReqTrue", "Alpha", "Bravo", "on", test_helpers.MakeBoolPreRequisite("PreReq1", "true"))...)

	return httpmock.NewJsonResponse(200, FeatureConfigResponse)
}
