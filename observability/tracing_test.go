package observability

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestDefaultTracerAndHTTPHelpers(t *testing.T) {
	oldT := TracerImpl
	TracerImpl = NewDefaultTracer()
	t.Cleanup(func() { TracerImpl = oldT })

	span, ctx := TracerImpl.StartSpan(context.Background(), "op")
	span.SetAttribute(AttrHTTPMethod, "GET")
	span.SetStatus(StatusCodeOk, "")
	span.AddEvent("evt", map[string]interface{}{"k": "v"})
	span.End()

	// Context helpers
	id := GenerateRequestID()
	ctx = WithRequestID(ctx, id)
	if have, ok := RequestIDFromContext(ctx); !ok || have == "" {
		t.Fatalf("request id missing")
	}

	// HTTP inject/extract
	req := httptest.NewRequest("GET", "/", nil)
	ctx2 := ExtractHTTPContext(ctx, req)
	rw := httptest.NewRecorder()
	InjectHTTPHeaders(rw, ctx2)
	if rw.Header().Get("X-Request-ID") == "" {
		t.Fatalf("missing header")
	}
}
