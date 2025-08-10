package prom

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExporterMetricsAndHandler(t *testing.T) {
	e := New()
	e.IncrementRequests(map[string]string{"route": "/chat", "method": "POST", "status_code": "200"})
	e.RecordLatency(3*time.Millisecond, map[string]string{"route": "/chat", "method": "POST", "status_code": "200"})
	e.IncrementTokensUsed(7, map[string]string{"direction": "input", "model": "gpt"})
	e.RecordError("tool_error", map[string]string{"route": "/chat", "method": "POST", "status_code": "500"})
	e.SetActiveAgents(2)

	rr := httptest.NewRecorder()
	Handler(e).ServeHTTP(rr, httptest.NewRequest("GET", "/metrics", nil))
	body := rr.Body.String()
	if !strings.Contains(body, "goagents_requests_total") || !strings.Contains(body, "goagents_active_agents") {
		t.Fatalf("unexpected metrics body: %s", body)
	}
}
