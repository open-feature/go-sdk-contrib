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
        "id": 11,
        "feature": {
            "id": 11,
            "name": "disabled_string_flag",
            "default_enabled": true,
            "type": "STANDARD",
            "project": 1
        },
        "feature_state_value": "some_value",
        "enabled": false,
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
        "id": 21,
        "feature": {
            "id": 21,
            "name": "disabled_int_flag",
            "default_enabled": true,
            "type": "STANDARD",
            "project": 1
        },
        "feature_state_value": 100,
        "enabled": false,
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
        "id": 31,
        "feature": {
            "id": 31,
            "name": "disabled_float_flag",
            "default_enabled": true,
            "type": "STANDARD",
            "project": 1
        },
        "feature_state_value": "100.1",
        "enabled": false,
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
    },
    {
        "id": 41,
        "feature": {
            "id": 41,
            "name": "disabled_bool_flag",
            "default_enabled": true,
            "type": "STANDARD",
            "project": 1
        },
        "feature_state_value": true,
        "enabled": false,
        "environment": 1,
        "identity": null,
        "feature_segment": null
    },
    {
        "id": 5,
        "feature": {
            "id": 4,
            "name": "json_flag",
            "default_enabled": true,
            "type": "STANDARD",
            "project": 1
        },
        "feature_state_value": "{\"key\": \"value\"}",
        "enabled": true,
        "environment": 1,
        "identity": null,
        "feature_segment": null
    },
    {
        "id": 51,
        "feature": {
            "id": 51,
            "name": "disabled_json_flag",
            "default_enabled": true,
            "type": "STANDARD",
            "project": 1
        },
        "feature_state_value": "{\"key\": \"value\"}",
        "enabled": false,
        "environment": 1,
        "identity": null,
        "feature_segment": null
    }
]`

const IdentityResponseJson = `{
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
        },
        {
            "id": 104,
            "feature": {
                "id": 5,
                "name": "json_flag",
                "initial_value": null,
                "default_enabled": false,
                "type": "STANDARD",
                "project": 1
            },
            "feature_state_value": "{\"key\": \"value_override\"}",
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
}`

const EnvironmentAPIKey = "API_KEY"

func TestIntEvaluation(t *testing.T) {
	identifier := "test_user"
	trait := flagsmith.Trait{TraitKey: "of_key", TraitValue: "of_value"}
	defaultValue := int64(2)
	expectedValue := int64(100)
	expectedValueIdentityOverride := int64(101)

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
			expectedValue:       expectedValue,
			expectederrorString: "",
			expectedErrorCode:   "",
			reason:              of.StaticReason,
		},
		{
			name:                "Should resolve with default value when flag is disabled",
			flagKey:             "disabled_int_flag",
			expectedValue:       defaultValue,
			expectederrorString: "",
			expectedErrorCode:   "",
			reason:              of.DisabledReason,
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
		expectedValue       float64
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
			name:                "Should resolve with default value when flag is disabled",
			flagKey:             "disabled_float_flag",
			expectedValue:       defaultValue,
			expectederrorString: "",
			expectedErrorCode:   "",
			reason:              of.DisabledReason,
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

func TestStringEvaluation(t *testing.T) {
	identifier := "test_user"
	trait := flagsmith.Trait{TraitKey: "of_key", TraitValue: "of_value"}
	defaultValue := "default_value"
	expectedFlagValue := "some_value"
	expectedValueIdentityOverride := "some_value_override"

	traits := []*flagsmith.Trait{&trait}
	tests := []struct {
		name                string
		flagKey             string
		expectedValue       string
		expectederrorString string
		reason              of.Reason
		expectedErrorCode   of.ErrorCode
		evalCtx             map[string]interface{}
	}{
		{
			name:                "Should resolve a valid flag with Static reason",
			flagKey:             "string_flag",
			expectedValue:       expectedFlagValue,
			expectederrorString: "",
			expectedErrorCode:   "",
			reason:              of.StaticReason,
		},
		{
			name:                "Should resolve with default value when flag is disabled",
			flagKey:             "disabled_string_flag",
			expectedValue:       defaultValue,
			expectederrorString: "",
			expectedErrorCode:   "",
			reason:              of.DisabledReason,
		},
		{
			name:                "Should error if flag is of incorrect type",
			flagKey:             "int_flag",
			expectedValue:       defaultValue,
			expectederrorString: "flagsmith: Value 100 is not a valid string",
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
			flagKey:             "string_flag",
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
			flagKey:             "string_flag",
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
			flagKey:             "string_flag",
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
			flagKey:             "string_flag",
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

			res := provider.StringEvaluation(context.Background(), test.flagKey, defaultValue, test.evalCtx)

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

func TestBooleanEvaluation(t *testing.T) {
	identifier := "test_user"
	trait := flagsmith.Trait{TraitKey: "of_key", TraitValue: "of_value"}
	defaultValue := false
	expectedFlagValue := true
	expectedValueIdentityOverride := true

	traits := []*flagsmith.Trait{&trait}
	tests := []struct {
		name                string
		flagKey             string
		expectedValue       bool
		expectederrorString string
		reason              of.Reason
		expectedErrorCode   of.ErrorCode
		evalCtx             map[string]interface{}
	}{
		{
			name:                "Should resolve a valid flag with Static reason",
			flagKey:             "bool_flag",
			expectedValue:       expectedFlagValue,
			expectederrorString: "",
			expectedErrorCode:   "",
			reason:              of.StaticReason,
		},
		{
			name:                "Should resolve with default value when flag is disabled",
			flagKey:             "disabled_bool_flag",
			expectedValue:       defaultValue,
			expectederrorString: "",
			expectedErrorCode:   "",
			reason:              of.DisabledReason,
		},
		{
			name:                "Should error if flag is of incorrect type",
			flagKey:             "int_flag",
			expectedValue:       defaultValue,
			expectederrorString: "flagsmith: Value 100 is not a valid boolean",
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
			flagKey:             "bool_flag",
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
			flagKey:             "bool_flag",
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
			flagKey:             "bool_flag",
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
			flagKey:             "bool_flag",
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

			res := provider.BooleanEvaluation(context.Background(), test.flagKey, defaultValue, test.evalCtx)

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

func TestObjectEvaluation(t *testing.T) {
	identifier := "test_user"
	trait := flagsmith.Trait{TraitKey: "of_key", TraitValue: "of_value"}

	defaultValue := map[string]interface{}{"key1": "value1"}
	expectedFlagValue := map[string]interface{}{"key": "value"}

	expectedValueIdentityOverride := map[string]interface{}{"key": "value_override"}

	traits := []*flagsmith.Trait{&trait}

	tests := []struct {
		name                string
		flagKey             string
		expectedValue       interface{}
		expectederrorString string
		reason              of.Reason
		expectedErrorCode   of.ErrorCode
		evalCtx             map[string]interface{}
	}{
		{
			name:                "Should resolve a valid flag with Static reason",
			flagKey:             "json_flag",
			expectedValue:       expectedFlagValue,
			expectederrorString: "",
			expectedErrorCode:   "",
			reason:              of.StaticReason,
		},
		{
			name:                "Should resolve with default value when flag is disabled",
			flagKey:             "disabled_json_flag",
			expectedValue:       defaultValue,
			expectederrorString: "",
			expectedErrorCode:   "",
			reason:              of.DisabledReason,
		},
		{
			name:                "Should error if flag is of incorrect type",
			flagKey:             "int_flag",
			expectedValue:       defaultValue,
			expectederrorString: "flagsmith: Value 100 is not a valid object",
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
			flagKey:             "json_flag",
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
			flagKey:             "json_flag",
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
			flagKey:             "json_flag",
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
			flagKey:             "json_flag",
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

			res := provider.ObjectEvaluation(context.Background(), test.flagKey, defaultValue, test.evalCtx)

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
