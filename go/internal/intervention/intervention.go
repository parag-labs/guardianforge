// Package intervention applies a decided intervention to the monitored fleet. It is the
// executor at the end of the governance loop: given a validated Intervention, it changes
// the fleet's state (pause, revoke, constrain) or routes to a human (escalate/notify).
package intervention

import (
	"fmt"

	"github.com/parag-labs/guardianforge/go/internal/fleet"
	"github.com/parag-labs/guardianforge/go/internal/models"
)

// ErrUnknownType is returned for an intervention type the executor doesn't handle.
type ErrUnknownType struct{ Type models.InterventionType }

func (e ErrUnknownType) Error() string { return fmt.Sprintf("unknown intervention type %q", e.Type) }

// Executor applies interventions to a fleet.
type Executor struct {
	fleet *fleet.Fleet
}

// New builds an executor over a fleet.
func New(f *fleet.Fleet) *Executor { return &Executor{fleet: f} }

// Execute applies an intervention. PAUSE/REVOKE_TOOL/INJECT_CONSTRAINT change fleet
// state; NOTIFY/ESCALATE/FORCE_REPLAN are routed (their effect is the audit record and,
// for ESCALATE, the HITL queue) and don't mutate the fleet here.
func (e *Executor) Execute(i models.Intervention) error {
	switch i.Type {
	case models.InterventionPause:
		e.fleet.Pause(i.TargetAgentID)
	case models.InterventionRevokeTool:
		e.fleet.RevokeTool(i.TargetAgentID, i.Parameters["tool"])
	case models.InterventionInjectConstraint:
		c := i.Parameters["constraint"]
		if c == "" {
			c = i.Reason
		}
		e.fleet.InjectConstraint(i.TargetAgentID, c)
	case models.InterventionNotify, models.InterventionEscalate, models.InterventionForceReplan:
		// No fleet mutation - handled by the audit trail and (for escalate) the HITL queue.
	default:
		return ErrUnknownType{Type: i.Type}
	}
	return nil
}
