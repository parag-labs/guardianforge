package agents

import (
	"context"
	"testing"

	"github.com/parag-labs/guardianforge/go/internal/agents/llm"
	"github.com/parag-labs/guardianforge/go/internal/models"
)

func sig(mode models.PolicyMode, trust float64) models.Signal {
	return models.Signal{
		Event:   models.AgentEvent{AgentID: "a1", Type: models.EventToolCall, Tool: "rm"},
		MaxMode: mode, Trust: trust,
		Violations: []models.Violation{{Severity: models.SeverityHigh}},
	}
}

func TestSupervisorHardModeIntervenes(t *testing.T) {
	sup := NewSupervisor(llm.MockLLM{})
	d, err := sup.Decide(context.Background(), sig(models.ModeHard, 0.8))
	if err != nil {
		t.Fatal(err)
	}
	if !d.Intervene || d.Type != models.InterventionRevokeTool {
		t.Fatalf("hard mode on a tool-call violation should revoke the tool, got %+v", d)
	}
}

func TestSupervisorObserveDoesNotIntervene(t *testing.T) {
	sup := NewSupervisor(llm.MockLLM{})
	d, err := sup.Decide(context.Background(), sig(models.ModeObserve, 0.8))
	if err != nil {
		t.Fatal(err)
	}
	if d.Intervene {
		t.Fatalf("observe mode must not intervene, got %+v", d)
	}
}

func TestLowTrustIsolatesAndEscalates(t *testing.T) {
	sup := NewSupervisor(llm.MockLLM{})
	d, _ := sup.Decide(context.Background(), sig(models.ModeSoft, 0.1))
	if !d.Intervene || !d.Escalate {
		t.Fatalf("low trust should intervene and escalate, got %+v", d)
	}
}

func TestValidateRejectsMalformed(t *testing.T) {
	_, err := Validate("{not json", sig(models.ModeHard, 0.8))
	if _, ok := err.(ErrMalformed); !ok {
		t.Fatalf("want ErrMalformed, got %v", err)
	}
}

func TestValidateRejectsUnknownType(t *testing.T) {
	raw := `{"intervene":true,"type":"NUKE_FROM_ORBIT","reason":"x","confidence":0.9}`
	_, err := Validate(raw, sig(models.ModeHard, 0.8))
	if _, ok := err.(ErrUnknownType); !ok {
		t.Fatalf("want ErrUnknownType, got %v", err)
	}
}

func TestValidateRejectsInterveneUnderObserve(t *testing.T) {
	raw := `{"intervene":true,"type":"PAUSE","reason":"x","confidence":0.9}`
	_, err := Validate(raw, sig(models.ModeObserve, 0.8))
	if _, ok := err.(ErrObserveViolated); !ok {
		t.Fatalf("want ErrObserveViolated, got %v", err)
	}
}

func TestScriptedInvalidFailsSafeToEscalate(t *testing.T) {
	// A model that hallucinates an intervention type must fail safe.
	bad := llm.ScriptedLLM{Raw: `{"intervene":true,"type":"HACK","reason":"trust me","confidence":0.99}`}
	sup := NewSupervisor(bad)
	d, err := sup.Decide(context.Background(), sig(models.ModeHard, 0.8))
	if err == nil {
		t.Fatal("expected a validation error")
	}
	if !d.Intervene || d.Type != models.InterventionEscalate {
		t.Fatalf("stricter-mode fail-safe should escalate to a human, got %+v", d)
	}
}

func TestObserveFailSafeDoesNotIntervene(t *testing.T) {
	bad := llm.ScriptedLLM{Raw: `garbage`}
	sup := NewSupervisor(bad)
	d, _ := sup.Decide(context.Background(), sig(models.ModeObserve, 0.8))
	if d.Intervene {
		t.Fatalf("observe fail-safe must not intervene, got %+v", d)
	}
}
