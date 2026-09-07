package eval

import "testing"

func TestGovernanceMeetsQualityBar(t *testing.T) {
	r := Run()
	if r.Scenarios != 5 {
		t.Fatalf("want 5 scenarios, got %d", r.Scenarios)
	}
	if r.DetectionAccuracy() < 1.0 {
		t.Errorf("detection accuracy below 100%%: %.0f%% (%+v)", 100*r.DetectionAccuracy(), r.Cases)
	}
	if r.TypeCorrect != r.Scenarios {
		t.Errorf("every scenario should get the right response, got %d/%d (%+v)", r.TypeCorrect, r.Scenarios, r.Cases)
	}
	if r.FalsePositives != 0 {
		t.Errorf("healthy agents must not be intervened on, got %d false positives", r.FalsePositives)
	}
	if !r.AuditIntact {
		t.Error("the audit chain must stay intact across the evaluation")
	}
}
