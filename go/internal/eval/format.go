package eval

import "fmt"

// sprintf renders the scorecard in a stable, readable form.
func sprintf(r Report) string {
	intact := "yes"
	if !r.AuditIntact {
		intact = "NO"
	}
	typeAcc := 0.0
	if r.Scenarios > 0 {
		typeAcc = 100.0 * float64(r.TypeCorrect) / float64(r.Scenarios)
	}
	return fmt.Sprintf(
		"GuardianForge Governance Evaluation\n\n"+
			"Scenarios:                 %d\n"+
			"Detection accuracy:        %.0f%%\n"+
			"Intervention correctness:  %.0f%%\n"+
			"False positives:           %d\n"+
			"Audit chain intact:        %s\n",
		r.Scenarios,
		100.0*r.DetectionAccuracy(),
		typeAcc,
		r.FalsePositives,
		intact,
	)
}
