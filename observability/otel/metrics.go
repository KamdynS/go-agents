//go:build adapters_otel
// +build adapters_otel

package otel

import (
	"time"

	"github.com/KamdynS/go-agents/observability"
)

// MetricsAdapter provides a no-op shim that can later be wired to OTel metrics (alpha).
type MetricsAdapter struct{}

func (m *MetricsAdapter) IncrementRequests(labels map[string]string)                     {}
func (m *MetricsAdapter) RecordLatency(duration time.Duration, labels map[string]string) {}
func (m *MetricsAdapter) IncrementTokensUsed(tokens int, labels map[string]string)       {}
func (m *MetricsAdapter) RecordError(errorType string, labels map[string]string)         {}
func (m *MetricsAdapter) SetActiveAgents(count int)                                      {}

var _ observability.Metrics = (*MetricsAdapter)(nil)
