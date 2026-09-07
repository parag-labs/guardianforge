package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/parag-labs/guardianforge/go/internal/agents"
	"github.com/parag-labs/guardianforge/go/internal/agents/llm"
	"github.com/parag-labs/guardianforge/go/internal/fleet"
	"github.com/parag-labs/guardianforge/go/internal/models"
)

func fixed() time.Time { return time.Unix(0, 0).UTC() }

func engine() (*Engine, *fleet.Fleet) {
	f := fleet.New()
	sup := agents.NewSupervisor(llm.MockLLM{})
	return New(fleet.DefaultPolicies(), f, sup, fixed), f
}

func runScenario(t *testing.T, id string) (*Engine, *fleet.Fleet, *models.Intervention) {
	t.Helper()
	sc, ok := fleet.ScenarioByID(id)
	if !ok {
		t.Fatalf("scenario %s not found", id)
	}
	eng, f := engine()
	var last *models.Intervention
	for _, ev := range sc.Events {
		out := eng.Process(context.Background(), ev)
		if out.Intervention != nil {
			last = out.Intervention
		}
	}
	return eng, f, last
}

func TestDangerousToolIsRevoked(t *testing.T) {
	_, f, iv := runScenario(t, "dangerous_tool")
	if iv == nil || iv.Type != models.InterventionRevokeTool {
		t.Fatalf("expected a REVOKE_TOOL intervention, got %+v", iv)
	}
	if !f.IsToolRevoked("reaper", "delete_database") {
		t.Fatal("the destructive tool should be revoked on the fleet")
	}
}

func TestPrivilegeEscalationIsEscalated(t *testing.T) {
	eng, _, iv := runScenario(t, "privilege_escalation")
	if iv == nil || iv.Type != models.InterventionEscalate {
		t.Fatalf("expected an ESCALATE intervention, got %+v", iv)
	}
	if len(eng.PendingHITL()) == 0 {
		t.Fatal("an escalation should be queued for a human")
	}
}

func TestLoopIsInterrupted(t *testing.T) {
	_, f, iv := runScenario(t, "infinite_loop")
	if iv == nil {
		t.Fatal("a tool loop should trigger an intervention")
	}
	if !f.IsToolRevoked("looping-agent", "search") && !f.IsPaused("looping-agent") {
		t.Fatal("the looping agent should be constrained (tool revoked or paused)")
	}
}

func TestHealthyAgentIsLeftAlone(t *testing.T) {
	eng, f, iv := runScenario(t, "healthy_agent")
	if iv != nil {
		t.Fatalf("a healthy agent must not be intervened on, got %+v", iv)
	}
	if f.IsPaused("good-agent") {
		t.Fatal("a healthy agent must not be paused")
	}
	if eng.Metrics.Get("interventions_total") != 0 {
		t.Fatal("no interventions should have been recorded for a healthy agent")
	}
}

func TestPIILeakInjectsConstraint(t *testing.T) {
	_, f, iv := runScenario(t, "data_leak")
	if iv == nil || iv.Type != models.InterventionInjectConstraint {
		t.Fatalf("expected INJECT_CONSTRAINT, got %+v", iv)
	}
	if len(f.Snapshot("leaker").Constraints) == 0 {
		t.Fatal("a constraint should have been injected on the leaking agent")
	}
}

func TestAuditChainStaysIntact(t *testing.T) {
	eng, _, _ := runScenario(t, "dangerous_tool")
	if !eng.Audit.Verify() {
		t.Fatal("the audit chain must verify after processing")
	}
	if eng.Audit.Len() == 0 {
		t.Fatal("processing should have written audit entries")
	}
}

func TestRejectedSupervisorOutputFailsSafe(t *testing.T) {
	// A supervisor that hallucinates an intervention type must fail safe to escalation,
	// and must not, e.g., pause the wrong agent.
	f := fleet.New()
	bad := agents.NewSupervisor(llm.ScriptedLLM{Raw: `{"intervene":true,"type":"HACK","reason":"x","confidence":0.9}`})
	eng := New(fleet.DefaultPolicies(), f, bad, fixed)
	ev := models.AgentEvent{AgentID: "x", Type: models.EventToolCall, Tool: "delete_database"}
	out := eng.Process(context.Background(), ev)
	if out.Rejected == "" {
		t.Fatal("expected the hallucinated type to be rejected")
	}
	// Fail-safe escalation is still a valid, known intervention type.
	if out.Intervention == nil || out.Intervention.Type != models.InterventionEscalate {
		t.Fatalf("rejected output should fail safe to ESCALATE, got %+v", out.Intervention)
	}
}
