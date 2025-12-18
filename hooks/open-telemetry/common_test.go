package otel

import (
	"go.openfeature.dev/openfeature/v2"
	"go.opentelemetry.io/otel/attribute"
)

// test commons

var (
	scopeKey         = "scope"
	scopeValue       = "7c34165e-fbef-11ed-be56-0242ac120002"
	scopeDescription = DimensionDescription{
		Key:  scopeKey,
		Type: String,
	}
)

var (
	stageKey         = "stage"
	stageValue       = 1
	stageDescription = DimensionDescription{
		Key:  stageKey,
		Type: Int,
	}
)

var (
	scoreKey         = "score"
	scoreValue       = 4.5
	scoreDescription = DimensionDescription{
		Key:  scoreKey,
		Type: Float,
	}
)

var (
	cachedKey         = "cached"
	cacheValue        = false
	cachedDescription = DimensionDescription{
		Key:  cachedKey,
		Type: Bool,
	}
)

var evalMetadata = map[string]any{
	scopeKey:  scopeValue,
	stageKey:  stageValue,
	scoreKey:  scoreValue,
	cachedKey: cacheValue,
}

var extractionCallback = func(metadata openfeature.FlagMetadata) []attribute.KeyValue {
	attribs := []attribute.KeyValue{}

	scope, err := metadata.GetString(scopeKey)
	if err != nil {
		panic(err)
	}

	attribs = append(attribs, attribute.String(scopeKey, scope))

	stage, err := metadata.GetInt(stageKey)
	if err != nil {
		panic(err)
	}

	attribs = append(attribs, attribute.Int64(stageKey, stage))

	score, err := metadata.GetFloat(scoreKey)
	if err != nil {
		panic(err)
	}

	attribs = append(attribs, attribute.Float64(scoreKey, score))

	cached, err := metadata.GetBool(cachedKey)
	if err != nil {
		panic(err)
	}

	attribs = append(attribs, attribute.Bool(cachedKey, cached))
	return attribs
}
