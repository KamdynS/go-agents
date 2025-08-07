package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

// Tracer defines the interface for distributed tracing
type Tracer interface {
	// StartSpan creates a new span with the given name
	StartSpan(ctx context.Context, name string) (Span, context.Context)

	// SpanFromContext extracts the span from context
	SpanFromContext(ctx context.Context) Span
}

// Span represents a tracing span
type Span interface {
	// SetAttribute sets an attribute on the span
	SetAttribute(key string, value interface{})

	// SetStatus sets the span status
	SetStatus(code StatusCode, message string)

	// AddEvent adds an event to the span
	AddEvent(name string, attributes map[string]interface{})

	// End finishes the span
	End()

	// Context returns the span context
	Context() context.Context
}

// StatusCode represents span status codes
type StatusCode int

const (
	StatusCodeUnset StatusCode = iota
	StatusCodeOk
	StatusCodeError
)

// Common attribute keys (align loosely with OTel HTTP and GenAI conventions)
const (
	AttrHTTPMethod   = "http.method"
	AttrHTTPRoute    = "http.route"
	AttrHTTPStatus   = "http.status_code"
	AttrRequestID    = "request.id"
	AttrProvider     = "genai.provider"
	AttrModel        = "genai.model"
	AttrFinishReason = "genai.finish_reason"
	AttrToolName     = "genai.tool.name"
	AttrTokensInput  = "genai.tokens.input"
	AttrTokensOutput = "genai.tokens.output"
)

// Global, swappable implementations (no-ops by default)
var (
	TracerImpl  Tracer  = &NoOpTracer{}
	MetricsImpl Metrics = &NoOpMetrics{}
)

// SetTracer swaps the global tracer implementation
func SetTracer(t Tracer) { TracerImpl = t }

// SetMetrics swaps the global metrics implementation
func SetMetrics(m Metrics) { MetricsImpl = m }

// NoOpTracer is a no-operation implementation of Tracer
type NoOpTracer struct{}

// StartSpan implements Tracer interface
func (t *NoOpTracer) StartSpan(ctx context.Context, name string) (Span, context.Context) {
	return &NoOpSpan{}, ctx
}

// SpanFromContext implements Tracer interface
func (t *NoOpTracer) SpanFromContext(ctx context.Context) Span {
	return &NoOpSpan{}
}

// NoOpSpan is a no-operation implementation of Span
type NoOpSpan struct{}

// SetAttribute implements Span interface
func (s *NoOpSpan) SetAttribute(key string, value interface{}) {}

// SetStatus implements Span interface
func (s *NoOpSpan) SetStatus(code StatusCode, message string) {}

// AddEvent implements Span interface
func (s *NoOpSpan) AddEvent(name string, attributes map[string]interface{}) {}

// End implements Span interface
func (s *NoOpSpan) End() {}

// Context implements Span interface
func (s *NoOpSpan) Context() context.Context {
	return context.Background()
}

// DefaultTracer is a simple in-memory tracer for development
type DefaultTracer struct {
	spans []SpanData
}

// SpanData holds information about a completed span
type SpanData struct {
	Name       string                 `json:"name"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	Status     StatusCode             `json:"status"`
	Message    string                 `json:"message"`
	Attributes map[string]interface{} `json:"attributes"`
	Events     []Event                `json:"events"`
}

// Event represents a span event
type Event struct {
	Name       string                 `json:"name"`
	Time       time.Time              `json:"time"`
	Attributes map[string]interface{} `json:"attributes"`
}

// NewDefaultTracer creates a new DefaultTracer instance
func NewDefaultTracer() *DefaultTracer {
	return &DefaultTracer{
		spans: make([]SpanData, 0),
	}
}

// StartSpan implements Tracer interface
func (t *DefaultTracer) StartSpan(ctx context.Context, name string) (Span, context.Context) {
	span := &DefaultSpan{
		tracer:     t,
		name:       name,
		startTime:  time.Now(),
		attributes: make(map[string]interface{}),
		events:     make([]Event, 0),
	}
	return span, context.WithValue(ctx, "span", span)
}

// SpanFromContext implements Tracer interface
func (t *DefaultTracer) SpanFromContext(ctx context.Context) Span {
	if span, ok := ctx.Value("span").(Span); ok {
		return span
	}
	return &NoOpSpan{}
}

// GetSpans returns all recorded spans
func (t *DefaultTracer) GetSpans() []SpanData {
	return t.spans
}

// DefaultSpan is a simple in-memory span implementation
type DefaultSpan struct {
	tracer     *DefaultTracer
	name       string
	startTime  time.Time
	endTime    time.Time
	status     StatusCode
	message    string
	attributes map[string]interface{}
	events     []Event
	ended      bool
}

// SetAttribute implements Span interface
func (s *DefaultSpan) SetAttribute(key string, value interface{}) {
	if s.ended {
		return
	}
	s.attributes[key] = value
}

// SetStatus implements Span interface
func (s *DefaultSpan) SetStatus(code StatusCode, message string) {
	if s.ended {
		return
	}
	s.status = code
	s.message = message
}

// AddEvent implements Span interface
func (s *DefaultSpan) AddEvent(name string, attributes map[string]interface{}) {
	if s.ended {
		return
	}
	event := Event{
		Name:       name,
		Time:       time.Now(),
		Attributes: attributes,
	}
	s.events = append(s.events, event)
}

// End implements Span interface
func (s *DefaultSpan) End() {
	if s.ended {
		return
	}
	s.ended = true
	s.endTime = time.Now()

	// Record the completed span
	spanData := SpanData{
		Name:       s.name,
		StartTime:  s.startTime,
		EndTime:    s.endTime,
		Duration:   s.endTime.Sub(s.startTime),
		Status:     s.status,
		Message:    s.message,
		Attributes: s.attributes,
		Events:     s.events,
	}
	s.tracer.spans = append(s.tracer.spans, spanData)
}

// Context implements Span interface
func (s *DefaultSpan) Context() context.Context {
	return context.WithValue(context.Background(), "span", s)
}

// Ensure implementations satisfy interfaces
var _ Tracer = (*NoOpTracer)(nil)
var _ Tracer = (*DefaultTracer)(nil)
var _ Span = (*NoOpSpan)(nil)
var _ Span = (*DefaultSpan)(nil)

// ----- Simple HTTP context propagation helpers -----

const (
	headerRequestID   = "X-Request-ID"
	headerTraceParent = "Traceparent" // reserved for future OTel wiring
)

type requestIDKey struct{}

// GenerateRequestID returns a random 16-byte hex string
func GenerateRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// WithRequestID stores a request id in the context
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFromContext retrieves a request id from context
func RequestIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(requestIDKey{})
	if v == nil {
		return "", false
	}
	id, ok := v.(string)
	return id, ok
}

// ExtractHTTPContext extracts basic propagation headers into context
func ExtractHTTPContext(ctx context.Context, r *http.Request) context.Context {
	id := r.Header.Get(headerRequestID)
	if id == "" {
		id = GenerateRequestID()
	}
	return WithRequestID(ctx, id)
}

// InjectHTTPHeaders writes propagation headers to the response
func InjectHTTPHeaders(w http.ResponseWriter, ctx context.Context) {
	if id, ok := RequestIDFromContext(ctx); ok {
		w.Header().Set(headerRequestID, id)
	}
}
