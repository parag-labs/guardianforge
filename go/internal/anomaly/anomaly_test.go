package anomaly

import (
	"testing"

	"github.com/parag-labs/guardianforge/go/internal/models"
)

func call(agent, tool string) models.AgentEvent {
	return models.AgentEvent{AgentID: agent, Type: models.EventToolCall, Tool: tool}
}

func TestDetectsLoop(t *testing.T) {
	d := New()
	var last []models.Anomaly
	for i := 0; i < 6; i++ {
		last = d.Observe(call("a1", "search"))
	}
	if !hasType(last, models.AnomalyLoop) {
		t.Fatalf("expected a loop anomaly, got %+v", last)
	}
}

func TestNoLoopForVariedBehaviour(t *testing.T) {
	d := New()
	tools := []string{"a", "b", "c", "d", "e"}
	var last []models.Anomaly
	for _, tl := range tools {
		last = d.Observe(call("a1", tl))
	}
	if hasType(last, models.AnomalyLoop) {
		t.Fatal("varied behaviour should not trip the loop detector")
	}
}

func TestDetectsCyclicPattern(t *testing.T) {
	d := New()
	var last []models.Anomaly
	for i := 0; i < 6; i++ {
		tool := "x"
		if i%2 == 1 {
			tool = "y"
		}
		last = d.Observe(call("a1", tool))
	}
	if !hasType(last, models.AnomalyCyclic) {
		t.Fatalf("expected a cyclic anomaly, got %+v", last)
	}
}

func TestDetectsRateSpike(t *testing.T) {
	d := New()
	var last []models.Anomaly
	// Alternate tools so the loop detector doesn't dominate, but still fill the window.
	for i := 0; i < 11; i++ {
		tool := "t" + string(rune('a'+i%4))
		last = d.Observe(call("a1", tool))
	}
	if !hasType(last, models.AnomalyRateSpike) {
		t.Fatalf("expected a rate spike, got %+v", last)
	}
}

func TestPerAgentIsolation(t *testing.T) {
	d := New()
	for i := 0; i < 6; i++ {
		d.Observe(call("noisy", "search"))
	}
	// A quiet agent should have no anomalies.
	got := d.Observe(call("quiet", "read"))
	if len(got) != 0 {
		t.Fatalf("quiet agent should have no anomalies, got %+v", got)
	}
}

func hasType(as []models.Anomaly, t models.AnomalyType) bool {
	for _, a := range as {
		if a.Type == t {
			return true
		}
	}
	return false
}
