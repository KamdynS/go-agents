package observability

import (
	"testing"
	"time"
)

func TestNoOpMetrics(t *testing.T) {
	var m Metrics = &NoOpMetrics{}
	m.IncrementRequests(nil)
	m.RecordLatency(time.Millisecond, nil)
	m.IncrementTokensUsed(10, nil)
	m.RecordError("x", nil)
	m.SetActiveAgents(1)
}

func TestDefaultMetrics(t *testing.T) {
	m := NewDefaultMetrics()
	m.IncrementRequests(map[string]string{"route": "/x"})
	m.RecordLatency(2*time.Millisecond, nil)
	m.IncrementTokensUsed(5, nil)
	m.RecordError("boom", nil)
	m.SetActiveAgents(3)
	s := m.GetStats()
	if s["requests"].(int64) != 1 {
		t.Fatalf("requests wrong: %+v", s)
	}
	if s["active_agents"].(int) != 3 {
		t.Fatalf("active wrong: %+v", s)
	}
}
