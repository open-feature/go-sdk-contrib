package model

type DataCollectorRequest struct {
	Events []FeatureEvent `json:"events"`
	Meta   map[string]any `json:"meta"`
}
