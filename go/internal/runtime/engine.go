// Package runtime is the governance engine: the event-driven core that runs every event
// through the deterministic detectors, asks the supervisor for a validated decision, and
// applies the resulting intervention while recording everything in the hash-chained audit
// log. This is where "deterministic first, model only for synthesis" is wired together.
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/parag-labs/guardianforge/go/internal/agents"
	"github.com/parag-labs/guardianforge/go/internal/anomaly"
	"github.com/parag-labs/guardianforge/go/internal/audit"
	"github.com/parag-labs/guardianforge/go/internal/fleet"
	"github.com/parag-labs/guardianforge/go/internal/intervention"
	"github.com/parag-labs/guardianforge/go/internal/models"
	"github.com/parag-labs/guardianforge/go/internal/obs"
	"github.com/parag-labs/guardianforge/go/internal/policy"
	"github.com/parag-labs/guardianforge/go/internal/trust"
)

// Outcome is the result of processing one event.
type Outcome struct {
	Signal       models.Signal        `json:"signal"`
	Decision     models.Decision      `json:"decision"`
	Intervention *models.Intervention `json:"intervention,omitempty"`
	Escalated    bool                 `json:"escalated"`
	Rejected     string               `json:"rejected,omitempty"`
}

// Engine wires the governance components together.
type Engine struct {
	Policy     *policy.Evaluator
	Anomaly    *anomaly.Detector
	Trust      *trust.Scorer
	Supervisor *agents.Supervisor
	Executor   *intervention.Executor
	Audit      *audit.Log
	Fleet      *fleet.Fleet
	Metrics    *obs.Metrics

	mu   sync.Mutex
	hitl []models.Intervention // pending human-in-the-loop escalations
	seq  int
	now  func() time.Time
}

// New builds an engine from a policy set, a fleet, and a supervisor.
func New(policies []models.Policy, f *fleet.Fleet, sup *agents.Supervisor, now func() time.Time) *Engine {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Engine{
		Policy:     policy.New(policies),
		Anomaly:    anomaly.New(),
		Trust:      trust.New(),
		Supervisor: sup,
		Executor:   intervention.New(f),
		Audit:      audit.New(),
		Fleet:      f,
		Metrics:    obs.New(),
		now:        now,
	}
}

// Process runs one event through the full governance loop and returns the outcome.
func (e *Engine) Process(ctx context.Context, ev models.AgentEvent) Outcome {
	e.Metrics.Inc("events_total", 1)

	violations, mode, sev := e.Policy.Evaluate(ev)
	anomalies := e.Anomaly.Observe(ev)

	// Anomalies are actionable even without a matching policy: a strong anomaly warrants
	// at least a soft response, so it never sits silently under an observe-only posture.
	if mode == models.ModeObserve && anyStrongAnomaly(anomalies) {
		mode = models.ModeSoft
	}

	trustScore := e.Trust.Update(ev.AgentID, violations, anomalies, e.now())
	if len(violations) > 0 {
		e.Metrics.Inc("violations_total", float64(len(violations)))
	}
	if len(anomalies) > 0 {
		e.Metrics.Inc("anomalies_total", float64(len(anomalies)))
	}

	signal := models.Signal{
		Event: ev, Violations: violations, Anomalies: anomalies,
		Trust: trustScore, MaxMode: mode, MaxSeverity: sev,
	}

	decision, err := e.Supervisor.Decide(ctx, signal)
	out := Outcome{Signal: signal, Decision: decision}
	if err != nil {
		out.Rejected = err.Error()
		e.Metrics.Inc("supervisor_rejections_total", 1)
	}
	e.audit("supervisor", "decision", signal, decision)

	if decision.Intervene {
		iv := e.buildIntervention(ev, decision)
		if execErr := e.Executor.Execute(iv); execErr == nil {
			out.Intervention = &iv
			e.Metrics.Inc("interventions_total", 1)
			e.auditIntervention(iv)
			if decision.Escalate {
				e.enqueueHITL(iv)
				out.Escalated = true
				e.Metrics.Inc("escalations_total", 1)
			}
		}
	}
	return out
}

func anyStrongAnomaly(as []models.Anomaly) bool {
	for _, a := range as {
		if a.Score >= 0.5 {
			return true
		}
	}
	return false
}

func (e *Engine) buildIntervention(ev models.AgentEvent, d models.Decision) models.Intervention {
	e.mu.Lock()
	e.seq++
	id := fmt.Sprintf("intv-%04d", e.seq)
	e.mu.Unlock()
	params := map[string]string{}
	if d.Type == models.InterventionRevokeTool {
		params["tool"] = ev.Tool
	}
	if d.Type == models.InterventionInjectConstraint {
		params["constraint"] = d.Reason
	}
	return models.Intervention{
		InterventionID: id, TargetAgentID: ev.AgentID, TargetSessionID: ev.SessionID,
		Type: d.Type, Reason: d.Reason, Parameters: params, IssuedBy: "supervisor", IssuedAt: e.now(),
	}
}

func (e *Engine) audit(actor, action string, signal models.Signal, decision models.Decision) {
	details, _ := json.Marshal(map[string]any{
		"agent": signal.Event.AgentID, "mode": signal.MaxMode, "severity": signal.MaxSeverity,
		"violations": len(signal.Violations), "anomalies": len(signal.Anomalies),
		"trust": signal.Trust, "intervene": decision.Intervene, "type": decision.Type,
	})
	e.Audit.Append(actor, action, string(details), e.now())
}

func (e *Engine) auditIntervention(iv models.Intervention) {
	details, _ := json.Marshal(iv)
	e.Audit.Append("executor", "intervention", string(details), e.now())
}

func (e *Engine) enqueueHITL(iv models.Intervention) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.hitl = append(e.hitl, iv)
}

// PendingHITL returns the queued human-in-the-loop escalations.
func (e *Engine) PendingHITL() []models.Intervention {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]models.Intervention(nil), e.hitl...)
}

// ResolveHITL removes an escalation from the queue after a human decision, recording it.
func (e *Engine) ResolveHITL(id, decision string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, iv := range e.hitl {
		if iv.InterventionID == id {
			e.hitl = append(e.hitl[:i], e.hitl[i+1:]...)
			e.Audit.Append("human", "hitl_resolution", fmt.Sprintf("%s: %s", id, decision), e.now())
			return true
		}
	}
	return false
}
