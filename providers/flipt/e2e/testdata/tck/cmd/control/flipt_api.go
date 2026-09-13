package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// targetUUID is the canonical targeting key that resolves targeting-key-flag
// to "hit" (see segmentsFor).
const targetUUID = "5c3d8535-f81a-4478-a6d3-afaa4d51199e"

// flag type and rollout enums, as the resource API payloads spell them.
const (
	flagTypeVariant      = "VARIANT_FLAG_TYPE"
	flagTypeBoolean      = "BOOLEAN_FLAG_TYPE"
	rolloutTypeThreshold = "THRESHOLD_ROLLOUT_TYPE"
	segmentOperatorAll   = "AND_SEGMENT_OPERATOR"
	matchTypeAll         = "ALL_MATCH_TYPE"
	comparisonString     = "STRING_COMPARISON_TYPE"
)

// variant describes a Flipt variant. Attachment is an arbitrary JSON value
// (string, number, boolean or object); see the FlagResourcePayload schema.
type variant struct {
	Key    string          `json:"key"`
	Name   string          `json:"name,omitempty"`
	Attach json.RawMessage `json:"attachment,omitempty"`
}

func objVariant(key, value string) variant {
	return variant{Key: key, Name: key, Attach: json.RawMessage(value)}
}

// plainVariant is a variant with no attachment. Scalar flag values in Flipt
// are carried by the variant key itself (the REST schema rejects non-object
// attachments), so the canonical scalar flags are seeded with their values as
// variant keys.
func plainVariant(key string) variant {
	return variant{Key: key, Name: key}
}

// threshold models a threshold rollout for a boolean flag.
type threshold struct {
	Percentage float64 `json:"percentage"`
	Value      bool    `json:"value"`
}

// rollout models a boolean flag rollout.
type rollout struct {
	Type      string     `json:"type"`
	Threshold *threshold `json:"threshold,omitempty"`
}

// distribution models a rule's variant distribution.
type distribution struct {
	Variant string  `json:"variant"`
	Rollout float64 `json:"rollout"`
}

// rule models a variant flag targeting rule.
type rule struct {
	SegmentOperator string         `json:"segmentOperator"`
	Segments        []string       `json:"segments"`
	Distributions   []distribution `json:"distributions"`
}

// constraint models a single segment constraint.
type constraint struct {
	Type     string `json:"type"`
	Property string `json:"property"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// flagPayload is the flipt.core.Flag resource payload.
type flagPayload struct {
	AtType         string    `json:"@type"`
	Key            string    `json:"key"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	Enabled        bool      `json:"enabled"`
	Variants       []variant `json:"variants,omitempty"`
	Rules          []rule    `json:"rules,omitempty"`
	Rollouts       []rollout `json:"rollouts,omitempty"`
	DefaultVariant string    `json:"defaultVariant,omitempty"`
}

// segmentPayload is the flipt.core.Segment resource payload.
type segmentPayload struct {
	AtType      string       `json:"@type"`
	Key         string       `json:"key"`
	Name        string       `json:"name"`
	MatchType   string       `json:"matchType"`
	Constraints []constraint `json:"constraints"`
}

// resourceRequest wraps a resource payload for the management API.
type resourceRequest struct {
	NamespaceKey string          `json:"namespaceKey"`
	Key          string          `json:"key"`
	Payload      json.RawMessage `json:"payload"`
}

// withRule applies the targeting rule to targeting-key-flag.
func withRule(fp *flagPayload, r rule) {
	fp.Rules = []rule{r}
}

// segmentsFor returns the segment resources the canonical set needs. The
// targeting rule lives on targeting-key-flag and is expressed as a segment
// whose constraint matches the canonical targeting key.
func segmentsFor() []segmentPayload {
	return []segmentPayload{{
		AtType:    "flipt.core.Segment",
		Key:       "targetingsg",
		Name:      "targetingsg",
		MatchType: matchTypeAll,
		Constraints: []constraint{{
			Type:     comparisonString,
			Property: "targetingKey",
			Operator: "eq",
			Value:    targetUUID,
		}},
	}}
}

// fliptAPI talks to the embedded Flipt management/evaluation API.
type fliptAPI struct {
	base   string
	client *http.Client
}

func newFliptAPI() *fliptAPI {
	return &fliptAPI{
		base:   fliptBaseURL,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// apiError carries the HTTP status of a failed management-API call so callers
// can branch on it (404 means a resource still has to be created).
type apiError struct {
	status int
	msg    string
}

func (e *apiError) Error() string { return e.msg }

func apiErr(key, op string, status int) error {
	return &apiError{status: status, msg: fmt.Sprintf("resource %q %s: %s", key, op, http.StatusText(status))}
}

// createResource posts a resource to the management API. A 409 (already
// exists) counts as success, which keeps a fresh store from failing when a
// resource was already seeded.
func (f *fliptAPI) createResource(ctx context.Context, req resourceRequest) error {
	httpReq, err := f.request(ctx, http.MethodPost, req)
	if err != nil {
		return err
	}
	status, err := f.do(httpReq)
	if err != nil {
		return err
	}
	if status == http.StatusConflict {
		return nil
	}
	if status < 200 || status >= 300 {
		return apiErr(req.Key, "create", status)
	}
	return nil
}

// updateResource puts a resource. The management API's PUT applies in place,
// which is what /reset and /change rely on: an update is served with no
// restart.
func (f *fliptAPI) updateResource(ctx context.Context, req resourceRequest) error {
	httpReq, err := f.request(ctx, http.MethodPut, req)
	if err != nil {
		return err
	}
	status, err := f.do(httpReq)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return apiErr(req.Key, "update", status)
	}
	return nil
}

// putOrCreate upserts a resource, which is what /reset relies on to restore
// the baseline without restarting the backend.
func (f *fliptAPI) putOrCreate(ctx context.Context, req resourceRequest) error {
	err := f.updateResource(ctx, req)
	if err == nil {
		return nil
	}
	var aerr *apiError
	if !errors.As(err, &aerr) || aerr.status != http.StatusNotFound {
		return err
	}
	return f.createResource(ctx, req)
}

func (f *fliptAPI) request(ctx context.Context, method string, req resourceRequest) (*http.Request, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, f.base+resourcePath(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq, nil
}

// do performs one request and reports its status, draining the body so the
// connection can be reused.
func (f *fliptAPI) do(req *http.Request) (int, error) {
	resp, err := f.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func resourcePath() string {
	return fmt.Sprintf("/api/v2/environments/%s/namespaces/%s/resources", environmentKey, namespaceKey)
}

// awaitUp polls flipt's /health endpoint until it answers 200, which is how a
// fresh /start knows evaluation is reachable before it seeds.
func (f *fliptAPI) awaitUp(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if !time.Now().Before(deadline) {
			return fmt.Errorf("flipt did not come up within %s", timeout)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.base+"/health", nil)
		if err != nil {
			return err
		}
		status, err := f.do(req)
		if err == nil && status == http.StatusOK {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}
