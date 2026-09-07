package policy

import (
	"testing"

	"github.com/parag-labs/guardianforge/go/internal/models"
)

func ev(agent string, t models.EventType, tool, action string) models.AgentEvent {
	return models.AgentEvent{AgentID: agent, Type: t, Tool: tool, Action: action}
}

func TestEqualityRuleFires(t *testing.T) {
	p := models.Policy{
		PolicyID: "p1", Mode: models.ModeHard,
		Rules: []models.Rule{{RuleID: "r1", Field: "tool", Op: models.OpEquals, Value: "delete_database", Severity: models.SeverityCritical}},
	}
	e := New([]models.Policy{p})
	v, mode, sev := e.Evaluate(ev("a1", models.EventToolCall, "delete_database", ""))
	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(v))
	}
	if mode != models.ModeHard || sev != models.SeverityCritical {
		t.Fatalf("unexpected mode/sev: %s %s", mode, sev)
	}
}

func TestNoViolationForCleanEvent(t *testing.T) {
	p := models.Policy{PolicyID: "p1", Mode: models.ModeHard,
		Rules: []models.Rule{{RuleID: "r1", Field: "tool", Op: models.OpEquals, Value: "delete_database", Severity: models.SeverityCritical}}}
	e := New([]models.Policy{p})
	v, mode, _ := e.Evaluate(ev("a1", models.EventToolCall, "read_file", ""))
	if len(v) != 0 || mode != models.ModeObserve {
		t.Fatalf("clean event should produce no violation, got %d mode %s", len(v), mode)
	}
}

func TestRegexpRule(t *testing.T) {
	p := models.Policy{PolicyID: "p1", Mode: models.ModeSoft,
		Rules: []models.Rule{{RuleID: "r1", Field: "action", Op: models.OpMatches, Value: `(?i)privilege|escalat`, Severity: models.SeverityHigh}}}
	e := New([]models.Policy{p})
	v, _, _ := e.Evaluate(ev("a1", models.EventDecision, "", "requesting privilege escalation"))
	if len(v) != 1 {
		t.Fatalf("regexp rule should fire, got %d", len(v))
	}
}

func TestCountOverDetectsRepetition(t *testing.T) {
	p := models.Policy{PolicyID: "p1", Mode: models.ModeHard,
		Rules: []models.Rule{{RuleID: "loop", Field: "tool", Op: models.OpCountOver, Value: "search", Threshold: 4, WindowSize: 10, Severity: models.SeverityMedium}}}
	e := New([]models.Policy{p})
	var last []models.Violation
	for i := 0; i < 6; i++ {
		last, _, _ = e.Evaluate(ev("a1", models.EventToolCall, "search", ""))
	}
	if len(last) == 0 {
		t.Fatal("count_over should fire after the threshold is exceeded")
	}
	// A different agent with few calls should not fire.
	v, _, _ := e.Evaluate(ev("a2", models.EventToolCall, "search", ""))
	if len(v) != 0 {
		t.Fatal("count is per-agent; a2 should not fire")
	}
}

func TestStrictestModeWins(t *testing.T) {
	observe := models.Policy{PolicyID: "obs", Mode: models.ModeObserve,
		Rules: []models.Rule{{RuleID: "r1", Field: "type", Op: models.OpEquals, Value: "TOOL_CALL", Severity: models.SeverityLow}}}
	hard := models.Policy{PolicyID: "hard", Mode: models.ModeHard,
		Rules: []models.Rule{{RuleID: "r2", Field: "tool", Op: models.OpEquals, Value: "rm", Severity: models.SeverityHigh}}}
	e := New([]models.Policy{observe, hard})
	_, mode, sev := e.Evaluate(ev("a1", models.EventToolCall, "rm", ""))
	if mode != models.ModeHard {
		t.Fatalf("strictest mode should win, got %s", mode)
	}
	if sev != models.SeverityHigh {
		t.Fatalf("highest severity should win, got %s", sev)
	}
}

func TestMetadataField(t *testing.T) {
	p := models.Policy{PolicyID: "p1", Mode: models.ModeSoft,
		Rules: []models.Rule{{RuleID: "r1", Field: "metadata.pii", Op: models.OpEquals, Value: "true", Severity: models.SeverityHigh}}}
	e := New([]models.Policy{p})
	event := ev("a1", models.EventToolResult, "read", "")
	event.Metadata = map[string]string{"pii": "true"}
	v, _, _ := e.Evaluate(event)
	if len(v) != 1 {
		t.Fatal("metadata rule should fire")
	}
}
