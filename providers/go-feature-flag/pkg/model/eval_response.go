package model

type EvalResponse[T JsonType] struct {
	TrackEvents   bool   `json:"trackEvents"`
	VariationType string `json:"variationType"`
	Failed        bool   `json:"failed"`
	Version       string `json:"version"`
	Reason        string `json:"reason"`
	ErrorCode     string `json:"errorCode"`
	Value         T      `json:"value"`
}
