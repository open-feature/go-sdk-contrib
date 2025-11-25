package ofrephandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/open-feature/go-sdk/openfeature"
)

const prefix = "/ofrep/v1/evaluate/flags/"

// Handler implements an HTTP handler that wraps an OpenFeature provider
// and exposes it as an OFREP-compliant HTTP service.
type Handler struct {
	provider openfeature.FeatureProvider
	config   *Configuration
}

// New creates a new OFREP HTTP handler that wraps the given provider.
func New(provider openfeature.FeatureProvider, opts ...Option) *Handler {
	config := &Configuration{
		requirePathPrefix: true,
		requirePOST:       true,
		pathValueName:     "key",
	}

	for _, opt := range opts {
		opt(config)
	}

	return &Handler{
		provider: provider,
		config:   config,
	}
}

// EvaluationRequest represents the OFREP evaluation request
type EvaluationRequest struct {
	Context any `json:"context,omitempty"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.config.requirePathPrefix && !strings.HasPrefix(r.URL.Path, prefix) {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Endpoint not found")
		return
	}

	if h.config.requirePOST && r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	var request EvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		h.writeError(w, http.StatusBadRequest, "PARSE_ERROR", fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	evalCtx := openfeature.FlattenedContext{}
	if ctx, ok := request.Context.(map[string]any); ok {
		evalCtx = openfeature.FlattenedContext(ctx)
	}

	flagKey := r.PathValue(h.config.pathValueName)
	if objectResult := h.provider.ObjectEvaluation(r.Context(), flagKey, nil, evalCtx); objectResult.Reason != openfeature.ErrorReason {
		if objectResult.Reason == openfeature.DefaultReason {
			h.writeFlagNotFoundError(w, flagKey)
			return
		}
		h.writeSuccess(w, flagKey, objectResult)
		return
	}
	h.writeFlagNotFoundError(w, flagKey)
}

type EvaluationResponse struct {
	Key      string         `json:"key"`
	Value    any            `json:"value"`
	Reason   string         `json:"reason"`
	Variant  string         `json:"variant,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (h *Handler) writeSuccess(w http.ResponseWriter, flagKey string, objectResult openfeature.InterfaceResolutionDetail) {
	w.WriteHeader(http.StatusOK)
	response := EvaluationResponse{
		Key:      flagKey,
		Value:    objectResult.Value,
		Reason:   string(objectResult.Reason),
		Variant:  objectResult.Variant,
		Metadata: objectResult.FlagMetadata,
	}
	json.NewEncoder(w).Encode(response)
}

type ErrorResponse struct {
	ErrorCode    string `json:"errorCode"`
	ErrorDetails string `json:"errorDetails"`
}

func (h *Handler) writeError(w http.ResponseWriter, statusCode int, errorCode, errorDetails string) {
	w.WriteHeader(statusCode)
	errorResponse := ErrorResponse{
		ErrorCode:    errorCode,
		ErrorDetails: errorDetails,
	}
	json.NewEncoder(w).Encode(errorResponse)
}

func (h *Handler) writeFlagNotFoundError(w http.ResponseWriter, flagKey string) {
	h.writeError(w, http.StatusNotFound, string(openfeature.FlagNotFoundCode), fmt.Sprintf("Flag '%s' not found", flagKey))
}
