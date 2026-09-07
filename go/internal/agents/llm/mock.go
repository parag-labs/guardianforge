package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/parag-labs/guardianforge/go/internal/models"
)

// MockLLM is the deterministic supervisor stand-in used in every test and the default
// demo. It synthesizes a Decision from the aggregated signal with fixed rules: low trust
// isolates, and otherwise the strictest policy mode dictates the response. It cannot
// invent an intervention the policy mode doesn't allow, which is exactly what the
// supervisor's validator also enforces on a real model.
type MockLLM struct{}

// Name identifies the provider.
func (MockLLM) Name() string { return "mock" }

// Synthesize maps a signal to a Decision deterministically.
func (MockLLM) Synthesize(_ context.Context, s models.Signal) (string, error) {
	d := decide(s)
	b, err := json.Marshal(d)
	return string(b), err
}

// decide is the deterministic policy-mode -> decision mapping (shared shape with what a
// real model is asked to produce).
func decide(s models.Signal) models.Decision {
	// A chronically untrusted agent is isolated and escalated regardless of mode.
	if s.Trust < 0.3 {
		return models.Decision{
			Intervene: true, Type: models.InterventionPause, Escalate: true,
			Reason: "agent trust below isolation floor; pausing and escalating", Confidence: 0.95,
		}
	}

	switch s.MaxMode {
	case models.ModeObserve:
		return models.Decision{Intervene: false, Reason: "observe-only policy; recording without intervention", Confidence: 0.9}
	case models.ModeSoft:
		t := models.InterventionNotify
		reason := "soft policy; notifying"
		// A strong soft signal - an anomaly, or a high/critical-severity violation like a
		// PII leak - warrants actively constraining the agent rather than just notifying.
		if len(s.Anomalies) > 0 || sevAtLeastHigh(s.MaxSeverity) {
			t = models.InterventionInjectConstraint
			reason = "soft policy; injecting a constraint to contain the risk"
		}
		return models.Decision{Intervene: true, Type: t, Reason: reason, Confidence: 0.85}
	case models.ModeHard:
		if s.Event.Type == models.EventToolCall && len(s.Violations) > 0 && s.Event.Tool != "" {
			return models.Decision{
				Intervene: true, Type: models.InterventionRevokeTool,
				Reason: fmt.Sprintf("hard policy; revoking tool %q", s.Event.Tool), Confidence: 0.9,
			}
		}
		return models.Decision{Intervene: true, Type: models.InterventionPause, Reason: "hard policy; pausing the agent", Confidence: 0.9}
	case models.ModeEscalate:
		return models.Decision{Intervene: true, Type: models.InterventionEscalate, Escalate: true, Reason: "policy requires human review", Confidence: 0.8}
	default:
		return models.Decision{Intervene: false, Reason: "no applicable policy", Confidence: 0.7}
	}
}

// sevAtLeastHigh reports whether a severity is high or critical.
func sevAtLeastHigh(s models.Severity) bool {
	return s == models.SeverityHigh || s == models.SeverityCritical
}

// Explain produces a deterministic narrative for an intervention.
func (MockLLM) Explain(_ context.Context, i models.Intervention, s models.Signal) (string, error) {
	return fmt.Sprintf(
		"Intervention %s (%s) was issued against agent %s because %s. At decision time the agent had %d policy violation(s), %d anomaly(ies), and a trust score of %.2f.",
		i.InterventionID, i.Type, i.TargetAgentID, i.Reason, len(s.Violations), len(s.Anomalies), s.Trust,
	), nil
}
