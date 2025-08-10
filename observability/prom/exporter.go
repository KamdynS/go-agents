package prom

import (
	"net/http"
	"strconv"
	"time"

	"github.com/KamdynS/go-agents/observability"
)

// Exporter implements observability.Metrics and exposes a Prometheus text endpoint
// without external dependencies. It aggregates counters and simple latency sums.
type Exporter struct {
	requests map[string]float64
	latency  map[string]float64
	tokens   map[string]float64
	errors   map[string]float64
	active   float64
}

// New creates a new in-process exporter.
func New() *Exporter {
	return &Exporter{
		requests: make(map[string]float64),
		latency:  make(map[string]float64),
		tokens:   make(map[string]float64),
		errors:   make(map[string]float64),
	}
}

// Handler returns an HTTP handler for a simple /metrics endpoint.
func Handler(e *Exporter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		// Requests
		for k, v := range e.requests {
			_, _ = w.Write([]byte("goagents_requests_total{label=\"" + k + "\"} " + formatFloat(v) + "\n"))
		}
		// Latency sums (seconds)
		for k, v := range e.latency {
			_, _ = w.Write([]byte("goagents_request_latency_seconds_sum{label=\"" + k + "\"} " + formatFloat(v) + "\n"))
		}
		// Tokens
		for k, v := range e.tokens {
			_, _ = w.Write([]byte("goagents_tokens_total{label=\"" + k + "\"} " + formatFloat(v) + "\n"))
		}
		// Errors
		for k, v := range e.errors {
			_, _ = w.Write([]byte("goagents_errors_total{label=\"" + k + "\"} " + formatFloat(v) + "\n"))
		}
		// Active agents
		_, _ = w.Write([]byte("goagents_active_agents " + formatFloat(e.active) + "\n"))
	})
}

func formatFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func (e *Exporter) IncrementRequests(labels map[string]string) { e.requests[labelKey(labels)]++ }
func (e *Exporter) RecordLatency(d time.Duration, labels map[string]string) {
	e.latency[labelKey(labels)] += d.Seconds()
}
func (e *Exporter) IncrementTokensUsed(tokens int, labels map[string]string) {
	e.tokens[labelKey(labels)] += float64(tokens)
}
func (e *Exporter) RecordError(errorType string, labels map[string]string) {
	key := errorType
	if len(labels) > 0 {
		key = key + "|" + labelKey(labels)
	}
	e.errors[key]++
}
func (e *Exporter) SetActiveAgents(count int) { e.active = float64(count) }

func labelKey(labels map[string]string) string {
	if v, ok := labels["route"]; ok {
		return v + "|" + labels["method"] + "|" + labels["status_code"]
	}
	if v, ok := labels["direction"]; ok {
		return v + "|" + labels["model"]
	}
	return "generic"
}

// Ensure interface compliance
var _ observability.Metrics = (*Exporter)(nil)
