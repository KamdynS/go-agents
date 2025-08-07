package observability

import (
	"time"
)

// Metrics defines the interface for collecting agent metrics
type Metrics interface {
	// IncrementRequests increments the request counter
	IncrementRequests(labels map[string]string)
	
	// RecordLatency records request latency
	RecordLatency(duration time.Duration, labels map[string]string)
	
	// IncrementTokensUsed increments token usage counter
	IncrementTokensUsed(tokens int, labels map[string]string)
	
	// RecordError increments error counter
	RecordError(errorType string, labels map[string]string)
	
	// SetActiveAgents sets the gauge for active agents
	SetActiveAgents(count int)
}

// NoOpMetrics is a no-operation implementation of Metrics
type NoOpMetrics struct{}

// IncrementRequests implements Metrics interface
func (n *NoOpMetrics) IncrementRequests(labels map[string]string) {}

// RecordLatency implements Metrics interface
func (n *NoOpMetrics) RecordLatency(duration time.Duration, labels map[string]string) {}

// IncrementTokensUsed implements Metrics interface
func (n *NoOpMetrics) IncrementTokensUsed(tokens int, labels map[string]string) {}

// RecordError implements Metrics interface
func (n *NoOpMetrics) RecordError(errorType string, labels map[string]string) {}

// SetActiveAgents implements Metrics interface
func (n *NoOpMetrics) SetActiveAgents(count int) {}

// DefaultMetrics is a simple in-memory metrics collector
type DefaultMetrics struct {
	requests     int64
	totalLatency time.Duration
	tokensUsed   int64
	errors       map[string]int64
	activeAgents int
}

// NewDefaultMetrics creates a new DefaultMetrics instance
func NewDefaultMetrics() *DefaultMetrics {
	return &DefaultMetrics{
		errors: make(map[string]int64),
	}
}

// IncrementRequests implements Metrics interface
func (m *DefaultMetrics) IncrementRequests(labels map[string]string) {
	m.requests++
}

// RecordLatency implements Metrics interface
func (m *DefaultMetrics) RecordLatency(duration time.Duration, labels map[string]string) {
	m.totalLatency += duration
}

// IncrementTokensUsed implements Metrics interface
func (m *DefaultMetrics) IncrementTokensUsed(tokens int, labels map[string]string) {
	m.tokensUsed += int64(tokens)
}

// RecordError implements Metrics interface
func (m *DefaultMetrics) RecordError(errorType string, labels map[string]string) {
	m.errors[errorType]++
}

// SetActiveAgents implements Metrics interface
func (m *DefaultMetrics) SetActiveAgents(count int) {
	m.activeAgents = count
}

// GetStats returns current statistics
func (m *DefaultMetrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"requests":       m.requests,
		"total_latency":  m.totalLatency.String(),
		"tokens_used":    m.tokensUsed,
		"errors":         m.errors,
		"active_agents":  m.activeAgents,
	}
}

// Ensure implementations satisfy the interface
var _ Metrics = (*NoOpMetrics)(nil)
var _ Metrics = (*DefaultMetrics)(nil)