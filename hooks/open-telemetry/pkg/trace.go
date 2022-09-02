package otel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type tracerClientInterface interface {
	tracer() trace.Tracer
}

type tracerClient struct{}

func (t *tracerClient) tracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer(traceName)
}
