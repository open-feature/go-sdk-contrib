package evaluate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/outbound"
	of "github.com/open-feature/go-sdk/openfeature"
)

// Outbound defines the contract for resolver's outbound communication, matching OFREP API.
type BulkOutbound interface {
	// Bulk flags resolving
	Bulk(ctx context.Context, paylod []byte) (*outbound.Resolution, error)
}

func NewBulkEvaluator(client BulkOutbound, evalCtx of.FlattenedContext) *BulkEvaluator {
	b := &BulkEvaluator{
		client:  client,
		evalCtx: evalCtx,
		values:  map[string]bulkEvaluationValue{},
	}
	b.resolver = b
	return b
}

type BulkEvaluator struct {
	Flags
	client  BulkOutbound
	evalCtx of.FlattenedContext
	values  map[string]bulkEvaluationValue
	mu      sync.RWMutex
}

func (b *BulkEvaluator) Fetch(ctx context.Context) error {
	payload, err := json.Marshal(requestFrom(b.evalCtx))
	if err != nil {
		return err
	}
	res, err := b.client.Bulk(ctx, payload)
	if err != nil {
		return err
	}

	switch res.Status {
	case http.StatusOK: // 200
		var data bulkEvaluationSuccess
		err := json.Unmarshal(res.Data, &data)
		if err != nil {
			return err
		}
		values := make(map[string]bulkEvaluationValue)
		for _, value := range data.Flags {
			values[value.Key] = value
		}
		b.setValues(values)
	case http.StatusNotModified: // 304
		// No changes
	case http.StatusBadRequest: // 400
		return parseError400(res.Data)
	case http.StatusUnauthorized, http.StatusForbidden: // 401, 403
		return of.NewGeneralResolutionError("authentication/authorization error")
	case http.StatusTooManyRequests: // 429
		after := parse429(res)
		if after == 0 {
			return of.NewGeneralResolutionError("rate limit exceeded")
		}
		return of.NewGeneralResolutionError(
			fmt.Sprintf("rate limit exceeded, try again after %f seconds", after.Seconds()))
	case http.StatusInternalServerError: // 500
		return parseError500(res.Data)
	default:
		return parseError500(res.Data)
	}

	return nil
}

func (b *BulkEvaluator) setValues(values map[string]bulkEvaluationValue) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.values = values
}

func (b *BulkEvaluator) resolveSingle(ctx context.Context, key string, evalCtx map[string]any) (*successDto, *of.ResolutionError) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if s, ok := b.values[key]; ok {
		if s.ErrorCode != "" {
			resErr := resolutionFromEvaluationError(s.evaluationError)
			return nil, &resErr
		}
		return &s.successDto, nil
	}
	resErr := of.NewFlagNotFoundResolutionError(fmt.Sprintf("flag for key '%s' does not exist", key))
	return nil, &resErr
}

type bulkEvaluationSuccess struct {
	Flags []bulkEvaluationValue
}

type bulkEvaluationValue struct {
	successDto
	evaluationError
}
