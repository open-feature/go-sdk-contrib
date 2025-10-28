package model_test

import (
	"testing"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackingEvent_ToMap(t *testing.T) {
	tests := []struct {
		name          string
		trackingEvent model.TrackingEvent
		wantFields    map[string]any
		wantErr       bool
	}{
		{
			name: "complete tracking event with value and attributes",
			trackingEvent: model.TrackingEvent{
				Kind:         "tracking",
				ContextKind:  "user",
				UserKey:      "user-123",
				CreationDate: 1680246000011,
				Key:          "click-event",
				EvaluationContext: map[string]any{
					"targetingKey": "user-123",
					"email":        "user@example.com",
				},
				TrackingDetails: openfeature.NewTrackingEventDetails(100.5).
					Add("action", "button-click").
					Add("page", "home"),
			},
			wantFields: map[string]any{
				"kind":         "tracking",
				"contextKind":  "user",
				"userKey":      "user-123",
				"creationDate": float64(1680246000011),
				"key":          "click-event",
				"evaluationContext": map[string]any{
					"targetingKey": "user-123",
					"email":        "user@example.com",
				},
				"trackingEventDetails": map[string]any{
					"value": 100.5,
					"attributes": map[string]any{
						"action": "button-click",
						"page":   "home",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "tracking event with zero value",
			trackingEvent: model.TrackingEvent{
				Kind:         "tracking",
				ContextKind:  "anonymousUser",
				UserKey:      "anon-456",
				CreationDate: 1680246000012,
				Key:          "page-view",
				EvaluationContext: map[string]any{
					"targetingKey": "anon-456",
				},
				TrackingDetails: openfeature.NewTrackingEventDetails(0),
			},
			wantFields: map[string]any{
				"kind":         "tracking",
				"contextKind":  "anonymousUser",
				"userKey":      "anon-456",
				"creationDate": float64(1680246000012),
				"key":          "page-view",
				"evaluationContext": map[string]any{
					"targetingKey": "anon-456",
				},
				"trackingEventDetails": map[string]any{
					"value":      float64(0),
					"attributes": map[string]any{},
				},
			},
			wantErr: false,
		},
		{
			name: "tracking event with negative value",
			trackingEvent: model.TrackingEvent{
				Kind:         "tracking",
				ContextKind:  "user",
				UserKey:      "user-789",
				CreationDate: 1680246000013,
				Key:          "refund",
				EvaluationContext: map[string]any{
					"targetingKey": "user-789",
				},
				TrackingDetails: openfeature.NewTrackingEventDetails(-50.25).
					Add("reason", "defective"),
			},
			wantFields: map[string]any{
				"kind":         "tracking",
				"contextKind":  "user",
				"userKey":      "user-789",
				"creationDate": float64(1680246000013),
				"key":          "refund",
				"evaluationContext": map[string]any{
					"targetingKey": "user-789",
				},
				"trackingEventDetails": map[string]any{
					"value": -50.25,
					"attributes": map[string]any{
						"reason": "defective",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "tracking event with complex attributes",
			trackingEvent: model.TrackingEvent{
				Kind:         "tracking",
				ContextKind:  "user",
				UserKey:      "user-complex",
				CreationDate: 1680246000014,
				Key:          "complex-event",
				EvaluationContext: map[string]any{
					"targetingKey": "user-complex",
					"nested": map[string]any{
						"level1": "value1",
					},
				},
				TrackingDetails: openfeature.NewTrackingEventDetails(999.99).
					Add("stringAttr", "test").
					Add("numberAttr", 42).
					Add("boolAttr", true).
					Add("mapAttr", map[string]string{"key": "value"}).
					Add("arrayAttr", []string{"item1", "item2"}),
			},
			wantFields: map[string]any{
				"kind":         "tracking",
				"contextKind":  "user",
				"userKey":      "user-complex",
				"creationDate": float64(1680246000014),
				"key":          "complex-event",
				"trackingEventDetails": map[string]any{
					"value": 999.99,
					"attributes": map[string]any{
						"stringAttr": "test",
						"numberAttr": 42,
						"boolAttr":   true,
						"mapAttr":    map[string]string{"key": "value"},
						"arrayAttr":  []string{"item1", "item2"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.trackingEvent.ToMap()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// Check all expected fields
			for key, expectedValue := range tt.wantFields {
				actualValue, exists := result[key]
				assert.True(t, exists, "Expected field %s to exist in result", key)
				assert.Equal(t, expectedValue, actualValue, "Field %s has unexpected value", key)
			}

			// Specifically validate trackingEventDetails structure
			trackingDetails, ok := result["trackingEventDetails"].(map[string]any)
			require.True(t, ok, "trackingEventDetails should be a map")

			// Verify value exists
			_, hasValue := trackingDetails["value"]
			assert.True(t, hasValue, "trackingEventDetails should have 'value' field")

			// Verify attributes exists
			_, hasAttributes := trackingDetails["attributes"]
			assert.True(t, hasAttributes, "trackingEventDetails should have 'attributes' field")
		})
	}
}

func TestNewTrackingEvent(t *testing.T) {
	tests := []struct {
		name                string
		ctx                 openfeature.EvaluationContext
		trackingEventName   string
		trackingDetails     openfeature.TrackingEventDetails
		expectedContextKind string
	}{
		{
			name: "user context",
			ctx: openfeature.NewEvaluationContext(
				"user-123",
				map[string]any{
					"email": "user@example.com",
					"name":  "John Doe",
				},
			),
			trackingEventName:   "purchase",
			trackingDetails:     openfeature.NewTrackingEventDetails(99.99).Add("item", "widget"),
			expectedContextKind: "user",
		},
		{
			name: "anonymous user context",
			ctx: openfeature.NewEvaluationContext(
				"anon-456",
				map[string]any{
					"anonymous": true,
				},
			),
			trackingEventName:   "page-view",
			trackingDetails:     openfeature.NewTrackingEventDetails(1),
			expectedContextKind: "anonymousUser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := model.NewTrackingEvent(tt.ctx, tt.trackingEventName, tt.trackingDetails)

			assert.Equal(t, "tracking", event.Kind)
			assert.Equal(t, tt.expectedContextKind, event.ContextKind)
			assert.Equal(t, tt.ctx.TargetingKey(), event.UserKey)
			assert.Equal(t, tt.trackingEventName, event.Key)
			assert.NotZero(t, event.CreationDate)

			// Verify evaluation context includes targeting key
			assert.Equal(t, tt.ctx.TargetingKey(), event.EvaluationContext[openfeature.TargetingKey])

			// Verify tracking details are preserved
			resultMap, err := event.ToMap()
			require.NoError(t, err)

			trackingDetails, ok := resultMap["trackingEventDetails"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tt.trackingDetails.Value(), trackingDetails["value"])
		})
	}
}
