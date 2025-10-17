package model

type DataCollectorRequest struct {
	Events []map[string]any `json:"events"`
	Meta   map[string]any   `json:"meta"`
}
