package model

import (
	"time"

	"github.com/thomaspoignant/go-feature-flag/modules/core/flag"
)

// FlagConfigRequest is the request body for the flag configuration endpoint.
type FlagConfigRequest struct {
	// Flags is a list of flag keys to retrieve.
	Flags []string `json:"flags"`
}

// FlagConfigResponse is the response from the flag configuration endpoint.
type FlagConfigResponse struct {
	// Flags is a dictionary that contains the flag key and its corresponding Flag object.
	Flags map[string]flag.InternalFlag `json:"flags"`

	// EvaluationContextEnrichment is a dictionary that contains additional context for the evaluation of flags.
	EvaluationContextEnrichment map[string]interface{} `json:"evaluationContextEnrichment"`

	// Etag is a string that represents the entity tag of the flag configuration response.
	Etag string `json:"etag"`

	// LastUpdated is a nullable DateTime that represents the last time the flag configuration was updated.
	LastUpdated time.Time `json:"lastUpdated"`
}
