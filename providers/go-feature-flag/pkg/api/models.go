package api

import (
	"errors"
	"time"

	"github.com/thomaspoignant/go-feature-flag/modules/core/flag"
)

// ErrNotModified is returned by GetConfiguration when the server responds with
// HTTP 304 (the flag configuration has not changed since the last request).
var ErrNotModified = errors.New("configuration not modified")

// FlagConfigurationRequest is the optional request body sent to POST /v1/flag/configuration.
// Leave Flags empty (or pass nil) to retrieve the full flag configuration.
type FlagConfigurationRequest struct {
	// Flags is the subset of flag keys to retrieve. An empty slice retrieves all flags.
	Flags []string `json:"flags,omitempty"`
}

// FlagConfigResponse is the parsed response from POST /v1/flag/configuration.
// It mirrors the JS SDK FlagConfigResponse type and the relay proxy
// controller.FlagConfigurationResponse schema.
type FlagConfigResponse struct {
	// Flags maps each flag key to its InternalFlag definition.
	Flags map[string]flag.InternalFlag `json:"flags"`

	// EvaluationContextEnrichment holds additional attributes that should be
	// merged into every evaluation context before flag resolution.
	EvaluationContextEnrichment map[string]any `json:"evaluationContextEnrichment,omitempty"`

	// Etag is the ETag returned in the HTTP response header.
	// Callers can pass it back via If-None-Match on the next request to receive
	// HTTP 304 (and ErrNotModified) when nothing has changed.
	Etag string `json:"-"`

	// LastUpdated is the time when the flag configuration was last changed.
	// It is populated from the Last-Updated HTTP response header, not the body.
	LastUpdated *time.Time `json:"-"`

	// ErrorCode optionally contains an error code returned in the response body.
	ErrorCode string `json:"errorCode,omitempty"`

	// ErrorDetails optionally contains a human-readable error description returned
	// in the response body.
	ErrorDetails string `json:"errorDetails,omitempty"`
}
