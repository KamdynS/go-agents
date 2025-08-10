//go:build adapters_otel
// +build adapters_otel

package otel

import (
	"context"
	"fmt"

	"github.com/KamdynS/go-agents/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Tracer implements observability.Tracer using OpenTelemetry.
type Tracer struct{ tracer trace.Tracer }

func NewTracer(serviceName string, _ interface{}) *Tracer {
	return &Tracer{tracer: otel.Tracer(serviceName)}
}

func (t *Tracer) StartSpan(ctx context.Context, name string) (observability.Span, context.Context) {
	ctx, span := t.tracer.Start(ctx, name)
	return &spanWrapper{span: span, ctx: ctx}, ctx
}

func (t *Tracer) SpanFromContext(ctx context.Context) observability.Span {
	// OTel retrieves span from context implicitly; we wrap a no-op child for interface compatibility
	_, span := t.tracer.Start(ctx, "child")
	return &spanWrapper{span: span, ctx: ctx}
}

type spanWrapper struct {
	span trace.Span
	ctx  context.Context
}

func (s *spanWrapper) SetAttribute(key string, value interface{}) {
	s.span.SetAttributes(attribute.String(key, toString(value)))
}
func (s *spanWrapper) SetStatus(code observability.StatusCode, message string) {
	// Map to OTel status via event; keep simple for v0
	s.span.AddEvent("status", trace.WithAttributes(attribute.Int("code", int(code)), attribute.String("message", message)))
}
func (s *spanWrapper) AddEvent(name string, attrs map[string]interface{}) {
	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		kvs = append(kvs, attribute.String(k, toString(v)))
	}
	s.span.AddEvent(name, trace.WithAttributes(kvs...))
}
func (s *spanWrapper) End()                     { s.span.End() }
func (s *spanWrapper) Context() context.Context { return s.ctx }

func toString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Ensure interface compliance
var _ observability.Tracer = (*Tracer)(nil)
var _ observability.Span = (*spanWrapper)(nil)
