package flagsmith

import (
	"context"
	"fmt"
	"github.com/Flagsmith/flagsmith-go-client/v2"
	flagsmithClient "github.com/Flagsmith/flagsmith-go-client/v2"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

var targetingKey = "123"
var flagKey = "flag_key"
var flag_one = flagsmithClient.Flag{
	Enabled:     true,
	Value:       true,
	FeatureID:   1,
	FeatureName: "flag_one",
}

// TODO: format the json
const FlagsJson = `
[
{
	"id": 1,
	"feature": {
		"id": 1,
		"name": "string_flag",
		"default_enabled": true,
		"type": "STANDARD",
		"project": 1
	},
	"feature_state_value": "some_value",
	"enabled": true,
	"environment": 1,
	"identity": null,
	"feature_segment": null
},
{
	"id": 2,
	"feature": {
		"id": 2,
		"name": "int_flag",
		"default_enabled": true,
		"type": "STANDARD",
		"project": 1
	},
	"feature_state_value": 100,
	"enabled": true,
	"environment": 1,
	"identity": null,
	"feature_segment": null
},

{
	"id": 3,
	"feature": {
		"id": 3,
		"name": "float_flag",
		"default_enabled": true,
		"type": "STANDARD",
		"project": 1
	},
	"feature_state_value": "100.1",
	"enabled": true,
	"environment": 1,
	"identity": null,
	"feature_segment": null
},

{
	"id": 4,
	"feature": {
		"id": 4,
		"name": "bool_flag",
		"default_enabled": true,
		"type": "STANDARD",
		"project": 1
	},
	"feature_state_value": true,
	"enabled": true,
	"environment": 1,
	"identity": null,
	"feature_segment": null
}

]
`
const IdentityResponseJson = `
{
	"flags": [
		{
			"id": 100,
			"feature": {
				"id": 1,
				"name": "string_flag",
				"initial_value": null,
				"default_enabled": false,
				"type": "STANDARD",
				"project": 1
			},
			"feature_state_value": "some_value_override",
			"enabled": true,
			"environment": 1,
			"identity": null,
			"feature_segment": null
		},
		{
			"id": 101,
			"feature": {
				"id": 2,
				"name": "int_flag",
				"initial_value": null,
				"default_enabled": false,
				"type": "STANDARD",
				"project": 1
			},
			"feature_state_value": 101,
			"enabled": true,
			"environment": 1,
			"identity": null,
			"feature_segment": null
		},
		{
			"id": 102,
			"feature": {
				"id": 3,
				"name": "float_flag",
				"initial_value": null,
				"default_enabled": false,
				"type": "STANDARD",
				"project": 1
			},
			"feature_state_value": "101.1",
			"enabled": true,
			"environment": 1,
			"identity": null,
			"feature_segment": null
		},
		{
			"id": 103,
			"feature": {
				"id": 4,
				"name": "bool_flag",
				"initial_value": null,
				"default_enabled": false,
				"type": "STANDARD",
				"project": 1
			},
			"feature_state_value": true,
			"enabled": true,
			"environment": 1,
			"identity": null,
			"feature_segment": null
		}
	],
	"traits": [
		{
			"trait_key": "foo",
			"trait_value": "bar"
		}
	]
}


`

const EnvironmentAPIKey = "API_KEY"

func TestIntEvaluation(t *testing.T) {
	identifier := "test_user"
	trait := flagsmith.Trait{TraitKey: "of_key", TraitValue: "of_value"}
	defaultValue := int64(2)

	traits := []*flagsmith.Trait{&trait}
	tests := []struct {
		name                string
		flagKey             string
		expectedValue       int64
		expectederrorString string
		reason              of.Reason
		expectedErrorCode   of.ErrorCode
		evalCtx             map[string]interface{}
	}{
		{
			name:                "Should resolve a valid flag with Static reason",
			flagKey:             "int_flag",
			expectedValue:       100,
			expectederrorString: "",
			expectedErrorCode:   "",
			reason:              of.StaticReason,
		},
		{
			name:                "Should error if flag is of incorrect type",
			flagKey:             "string_flag",
			expectedValue:       defaultValue,
			expectederrorString: "flagsmith: Value some_value is not a valid int",
			reason:              of.ErrorReason,
			expectedErrorCode:   of.TypeMismatchCode,
		},
		{
			name:                "Should error if flag does not exists",
			flagKey:             "flag_that_does_not_exists",
			expectedValue:       defaultValue,
			expectederrorString: "flagsmith: No feature found with name flag_that_does_not_exists",
			reason:              of.ErrorReason,
			expectedErrorCode:   of.FlagNotFoundCode,
		},
		{
			name:                "Should resolve a valid flag with identifier and no traits",
			flagKey:             "int_flag",
			expectedValue:       101,
			expectederrorString: "",
			reason:              of.TargetingMatchReason,
			expectedErrorCode:   of.FlagNotFoundCode,
			evalCtx: map[string]interface{}{
				of.TargetingKey: identifier,
			},
		},
		{
			name:                "Should error if identifier is not a string",
			flagKey:             "int_flag",
			expectedValue:       defaultValue,
			expectederrorString: "flagsmith: targeting key is not a string",
			reason:              of.ErrorReason,
			expectedErrorCode:   of.InvalidContextCode,
			evalCtx: map[string]interface{}{
				of.TargetingKey: map[string]interface{}{},
			},
		},
		{
			name:                "Should error if provided traits are not valid",
			flagKey:             "int_flag",
			expectedValue:       defaultValue,
			expectederrorString: "flagsmith: invalid traits: expected type []*flagsmithClient.Trait, got map[string]interface {}",
			reason:              of.ErrorReason,
			expectedErrorCode:   of.InvalidContextCode,
			evalCtx: map[string]interface{}{
				of.TargetingKey: identifier,
				"traits":        map[string]interface{}{},
			},
		},

		{
			name:                "Should resolve if provided traits are valid",
			flagKey:             "int_flag",
			expectedValue:       101,
			expectederrorString: "",
			reason:              of.TargetingMatchReason,
			expectedErrorCode:   of.InvalidContextCode,
			evalCtx: map[string]interface{}{
				of.TargetingKey: identifier,
				"traits":        traits,
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flags/", func(rw http.ResponseWriter, req *http.Request) {

		assert.Equal(t, EnvironmentAPIKey, req.Header.Get("X-Environment-Key"))

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, err := io.WriteString(rw, FlagsJson)

		assert.NoError(t, err)

	})
	expectedRequestBodyWithoutTraits := fmt.Sprintf(`{"identifier":"%s"}`, identifier)
	expectedRequestBodyWithTraits := fmt.Sprintf(`{"identifier":"%s","traits":[{"trait_key":"of_key","trait_value":"of_value"}]}`, identifier)
	mux.HandleFunc("/api/v1/identities/", func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, EnvironmentAPIKey, req.Header.Get("X-Environment-Key"))

		rawBody, err := io.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Contains(t, []string{expectedRequestBodyWithoutTraits, expectedRequestBodyWithTraits}, string(rawBody))
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, err = io.WriteString(rw, IdentityResponseJson)

		assert.NoError(t, err)

	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := flagsmithClient.NewClient(EnvironmentAPIKey,
		flagsmith.WithBaseURL(server.URL+"/api/v1/"))

	provider := NewProvider(client)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := provider.IntEvaluation(context.Background(), test.flagKey, defaultValue, test.evalCtx)
			assert.Equal(t, test.expectedValue, res.Value)
			assert.Equal(t, test.reason, res.ProviderResolutionDetail.Reason)
			if test.expectederrorString != "" {
				resolutionDetails := res.ResolutionDetail()
				assert.Equal(t, test.expectedErrorCode, resolutionDetails.ErrorCode)
				assert.Equal(t, test.expectederrorString, resolutionDetails.ErrorMessage)
			}

		})
	}

}




func TestFloatEvaluation(t *testing.T) {
	identifier := "test_user"
	trait := flagsmith.Trait{TraitKey: "of_key", TraitValue: "of_value"}
	defaultValue := float64(2.1)
	expectedFlagValue := float64(100.1)
	expectedValueIdentityOverride := float64(101.1)

	traits := []*flagsmith.Trait{&trait}
	tests := []struct {
		name                string
		flagKey             string
		expectedValue      float64
		expectederrorString string
		reason              of.Reason
		expectedErrorCode   of.ErrorCode
		evalCtx             map[string]interface{}
	}{
		{
			name:                "Should resolve a valid flag with Static reason",
			flagKey:             "float_flag",
			expectedValue:       expectedFlagValue,
			expectederrorString: "",
			expectedErrorCode:   "",
			reason:              of.StaticReason,
		},
		{
			name:                "Should error if flag is of incorrect type",
			flagKey:             "string_flag",
			expectedValue:       defaultValue,
			expectederrorString: "flagsmith: Value some_value is not a valid float",
			reason:              of.ErrorReason,
			expectedErrorCode:   of.TypeMismatchCode,
		},
		{
			name:                "Should error if flag does not exists",
			flagKey:             "flag_that_does_not_exists",
			expectedValue:       defaultValue,
			expectederrorString: "flagsmith: No feature found with name flag_that_does_not_exists",
			reason:              of.ErrorReason,
			expectedErrorCode:   of.FlagNotFoundCode,
		},
		{
			name:                "Should resolve a valid flag with identifier and no traits",
			flagKey:             "float_flag",
			expectedValue:       expectedValueIdentityOverride,
			expectederrorString: "",
			reason:              of.TargetingMatchReason,
			expectedErrorCode:   of.FlagNotFoundCode,
			evalCtx: map[string]interface{}{
				of.TargetingKey: identifier,
			},
		},
		{
			name:                "Should error if identifier is not a string",
			flagKey:             "float_flag",
			expectedValue:       defaultValue,
			expectederrorString: "flagsmith: targeting key is not a string",
			reason:              of.ErrorReason,
			expectedErrorCode:   of.InvalidContextCode,
			evalCtx: map[string]interface{}{
				of.TargetingKey: map[string]interface{}{},
			},
		},
		{
			name:                "Should error if provided traits are not valid",
			flagKey:             "float_flag",
			expectedValue:       defaultValue,
			expectederrorString: "flagsmith: invalid traits: expected type []*flagsmithClient.Trait, got map[string]interface {}",
			reason:              of.ErrorReason,
			expectedErrorCode:   of.InvalidContextCode,
			evalCtx: map[string]interface{}{
				of.TargetingKey: identifier,
				"traits":        map[string]interface{}{},
			},
		},

		{
			name:                "Should resolve if provided traits are valid",
			flagKey:             "float_flag",
			expectedValue:       expectedValueIdentityOverride,
			expectederrorString: "",
			reason:              of.TargetingMatchReason,
			expectedErrorCode:   of.InvalidContextCode,
			evalCtx: map[string]interface{}{
				of.TargetingKey: identifier,
				"traits":        traits,
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flags/", func(rw http.ResponseWriter, req *http.Request) {

		assert.Equal(t, EnvironmentAPIKey, req.Header.Get("X-Environment-Key"))

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, err := io.WriteString(rw, FlagsJson)

		assert.NoError(t, err)

	})
	expectedRequestBodyWithoutTraits := fmt.Sprintf(`{"identifier":"%s"}`, identifier)
	expectedRequestBodyWithTraits := fmt.Sprintf(`{"identifier":"%s","traits":[{"trait_key":"of_key","trait_value":"of_value"}]}`, identifier)
	mux.HandleFunc("/api/v1/identities/", func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, EnvironmentAPIKey, req.Header.Get("X-Environment-Key"))

		rawBody, err := io.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Contains(t, []string{expectedRequestBodyWithoutTraits, expectedRequestBodyWithTraits}, string(rawBody))
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, err = io.WriteString(rw, IdentityResponseJson)

		assert.NoError(t, err)

	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := flagsmithClient.NewClient(EnvironmentAPIKey,
		flagsmith.WithBaseURL(server.URL+"/api/v1/"))

	provider := NewProvider(client)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := provider.FloatEvaluation(context.Background(), test.flagKey, defaultValue, test.evalCtx)
			assert.Equal(t, test.expectedValue, res.Value)
			assert.Equal(t, test.reason, res.ProviderResolutionDetail.Reason)
			if test.expectederrorString != "" {
				resolutionDetails := res.ResolutionDetail()
				assert.Equal(t, test.expectedErrorCode, resolutionDetails.ErrorCode)
				assert.Equal(t, test.expectederrorString, resolutionDetails.ErrorMessage)
			}

		})
	}

}
