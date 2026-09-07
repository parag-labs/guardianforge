// Package agents holds the supervisor - the agent that turns an aggregated signal into a
// validated governance decision. The model proposes a decision; this package validates it
// against the deterministic policy mode before it is ever allowed to become an
// intervention. That validation is the trust boundary for the whole system.
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/parag-labs/guardianforge/go/internal/agents/llm"
	"github.com/parag-labs/guardianforge/go/internal/models"
)

// validTypes is the closed set of interventions the model may name.
var validTypes = map[models.InterventionType]bool{
	models.InterventionPause:            true,
	models.InterventionRevokeTool:       true,
	models.InterventionInjectConstraint: true,
	models.InterventionForceReplan:      true,
	models.InterventionNotify:           true,
	models.InterventionEscalate:         true,
}

// Validation errors are typed so tests can assert exactly how a decision was rejected.
type (
	// ErrMalformed means the model returned text that isn't the expected JSON.
	ErrMalformed struct{ Cause error }
	// ErrUnknownType means the model named an intervention type that doesn't exist.
	ErrUnknownType struct{ Type models.InterventionType }
	// ErrObserveViolated means the model tried to intervene under an OBSERVE policy.
	ErrObserveViolated struct{}
	// ErrMissingType means the model chose to intervene without naming a type.
	ErrMissingType struct{}
)

func (e ErrMalformed) Error() string       { return "malformed supervisor output: " + e.Cause.Error() }
func (e ErrUnknownType) Error() string     { return fmt.Sprintf("unknown intervention type %q", e.Type) }
func (e ErrObserveViolated) Error() string { return "model tried to intervene under an OBSERVE policy" }
func (e ErrMissingType) Error() string     { return "model chose to intervene without a type" }

// Supervisor synthesizes decisions via a model, then validates them.
type Supervisor struct {
	model llm.LLMClient
}

// NewSupervisor builds a supervisor over a model.
func NewSupervisor(model llm.LLMClient) *Supervisor { return &Supervisor{model: model} }

// Decide asks the model for a decision and validates it. On any validation failure it
// fails safe: OBSERVE-mode signals fall back to "no intervention", stricter modes fall
// back to escalation to a human rather than acting on output that couldn't be trusted.
func (s *Supervisor) Decide(ctx context.Context, signal models.Signal) (models.Decision, error) {
	raw, err := s.model.Synthesize(ctx, signal)
	if err != nil {
		return failSafe(signal), err
	}
	dec, verr := Validate(raw, signal)
	if verr != nil {
		return failSafe(signal), verr
	}
	return dec, nil
}

// Validate parses and hard-checks a raw model decision against the signal's policy mode.
func Validate(raw string, signal models.Signal) (models.Decision, error) {
	var d models.Decision
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return models.Decision{}, ErrMalformed{Cause: err}
	}
	if d.Intervene {
		if d.Type == "" {
			return models.Decision{}, ErrMissingType{}
		}
		if !validTypes[d.Type] {
			return models.Decision{}, ErrUnknownType{Type: d.Type}
		}
		// The core guardrail: an OBSERVE policy can never produce an intervention, no
		// matter what the model says.
		if signal.MaxMode == models.ModeObserve {
			return models.Decision{}, ErrObserveViolated{}
		}
	}
	return d, nil
}

// failSafe is the deterministic fallback when the model can't be trusted.
func failSafe(signal models.Signal) models.Decision {
	if signal.MaxMode == models.ModeObserve {
		return models.Decision{Intervene: false, Reason: "fail-safe: observe-only", Confidence: 0}
	}
	return models.Decision{Intervene: true, Type: models.InterventionEscalate, Escalate: true, Reason: "fail-safe: model output rejected; escalating to a human", Confidence: 0}
}
