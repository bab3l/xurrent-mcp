// Package otel provides optional OpenTelemetry tracing and metrics for the xurrent-mcp server.
//
// Build with: go build -tags=otel ./cmd/xurrent-mcp
//
// When the otel build tag is not used, all functions are no-ops with zero overhead.
//
// Environment:
//
//	OTEL_EXPORTER_OTLP_ENDPOINT — OTLP endpoint (e.g., http://localhost:4318)
//	OTEL_SERVICE_NAME — service name (default: xurrent-mcp)
package otel

// This file is compiled only when the "otel" build tag is NOT used.

// InitTracing is a no-op when OTEL is disabled.
func InitTracing() (func(), error) {
	return func() {}, nil
}

// InitMetrics is a no-op when OTEL is disabled.
func InitMetrics() (func(), error) {
	return func() {}, nil
}
