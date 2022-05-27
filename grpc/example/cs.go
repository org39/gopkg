package main

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type CustomSampler struct {
	BaseSampler sdktrace.Sampler
}

func (s CustomSampler) Description() string {
	return "GRPCHealthCheckSampler"
}

func (s CustomSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if p.Kind == trace.SpanKindServer && p.Name == "grpc.health.v1.Health/Check" {
		return sdktrace.SamplingResult{
			Decision:   sdktrace.Drop,
			Tracestate: trace.SpanContextFromContext(p.ParentContext).TraceState(),
		}
	}

	return s.BaseSampler.ShouldSample(p)
}
