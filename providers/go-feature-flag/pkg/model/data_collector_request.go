package model

// CollectableEvent is implemented by FeatureEvent and TrackingEvent.
// Only these types may be sent to the data collector.
type CollectableEvent interface {
	collectableEvent() // unexported marker; only model types can implement
}

type DataCollectorRequest struct {
	Events []CollectableEvent `json:"events"`
	Meta   map[string]any     `json:"meta"`
}
