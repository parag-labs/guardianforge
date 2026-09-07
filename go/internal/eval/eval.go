// Package eval scores GuardianForge against the fixed failure scenarios: does it detect
// and correctly respond to each injected failure, never intervene on a healthy agent, and
// keep the audit chain intact throughout? This is the governance scorecard, wired into CI.
package eval

import (
	"context"
	"time"

	"github.com/parag-labs/guardianforge/go/internal/agents"
	"github.com/parag-labs/guardianforge/go/internal/agents/llm"
	"github.com/parag-labs/guardianforge/go/internal/fleet"
	"github.com/parag-labs/guardianforge/go/internal/models"
	"github.com/parag-labs/guardianforge/go/internal/runtime"
)

// Report is the aggregate scorecard.
type Report struct {
	Scenarios      int
	DetectCorrect  int
	TypeCorrect    int
	FalsePositives int
	AuditIntact    bool
	Cases          []Case
}

// Case is one scenario's outcome.
type Case struct {
	ID         string
	Intervened bool
	Type       models.InterventionType
	Correct    bool
}

// DetectionAccuracy is the fraction of scenarios where intervene/observe matched truth.
func (r Report) DetectionAccuracy() float64 {
	if r.Scenarios == 0 {
		return 0
	}
	return float64(r.DetectCorrect) / float64(r.Scenarios)
}

// Run executes every scenario with the deterministic mock supervisor.
func Run() Report {
	now := func() time.Time { return time.Unix(0, 0).UTC() }
	rep := Report{AuditIntact: true}

	for _, sc := range fleet.Scenarios() {
		rep.Scenarios++
		f := fleet.New()
		sup := agents.NewSupervisor(llm.MockLLM{})
		eng := runtime.New(fleet.DefaultPolicies(), f, sup, now)

		var lastIntervention *models.Intervention
		for _, ev := range sc.Events {
			out := eng.Process(context.Background(), ev)
			if out.Intervention != nil {
				lastIntervention = out.Intervention
			}
		}

		intervened := lastIntervention != nil
		detectOK := intervened == sc.ExpectIntervene
		if detectOK {
			rep.DetectCorrect++
		}
		if intervened && !sc.ExpectIntervene {
			rep.FalsePositives++
		}

		var gotType models.InterventionType
		typeOK := true
		if sc.ExpectIntervene {
			if intervened {
				gotType = lastIntervention.Type
				typeOK = gotType == sc.ExpectType
			} else {
				typeOK = false
			}
			if typeOK {
				rep.TypeCorrect++
			}
		} else {
			// Healthy agent: "type correct" means it produced no intervention.
			if !intervened {
				rep.TypeCorrect++
			} else {
				typeOK = false
			}
		}

		if !eng.Audit.Verify() {
			rep.AuditIntact = false
		}

		rep.Cases = append(rep.Cases, Case{ID: sc.ID, Intervened: intervened, Type: gotType, Correct: detectOK && typeOK})
	}
	return rep
}

// Format renders the scorecard.
func (r Report) Format() string {
	return sprintf(r)
}
