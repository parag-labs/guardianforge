package trust

import (
	"testing"
	"time"

	"github.com/parag-labs/guardianforge/go/internal/models"
)

func fixed() time.Time { return time.Unix(0, 0).UTC() }

func TestUnseenAgentStartsAtBaseline(t *testing.T) {
	s := New()
	if s.Get("a1") != Baseline {
		t.Fatalf("want baseline %.2f, got %.2f", Baseline, s.Get("a1"))
	}
}

func TestViolationsDecayTrust(t *testing.T) {
	s := New()
	before := s.Get("a1")
	after := s.Update("a1", []models.Violation{{Severity: models.SeverityCritical}}, nil, fixed())
	if after >= before {
		t.Fatalf("a critical violation should lower trust: %.2f -> %.2f", before, after)
	}
}

func TestCleanBehaviourRecoversTrust(t *testing.T) {
	s := New()
	// Knock it down first.
	s.Update("a1", []models.Violation{{Severity: models.SeverityHigh}}, nil, fixed())
	low := s.Get("a1")
	// Then behave.
	after := s.Update("a1", nil, nil, fixed())
	if after <= low {
		t.Fatalf("clean behaviour should recover trust: %.2f -> %.2f", low, after)
	}
}

func TestRepeatedCriticalTriggersIsolation(t *testing.T) {
	s := New()
	for i := 0; i < 3; i++ {
		s.Update("bad", []models.Violation{{Severity: models.SeverityCritical}}, nil, fixed())
	}
	if !s.ShouldIsolate("bad") {
		t.Fatalf("agent with repeated critical violations should be isolated (trust %.2f)", s.Get("bad"))
	}
	if s.ShouldIsolate("good") {
		t.Fatal("an unseen agent should not be isolated")
	}
}

func TestTrustStaysInRange(t *testing.T) {
	s := New()
	for i := 0; i < 20; i++ {
		s.Update("a1", []models.Violation{{Severity: models.SeverityCritical}}, nil, fixed())
	}
	if s.Get("a1") < 0 {
		t.Fatal("trust must not go below 0")
	}
	for i := 0; i < 200; i++ {
		s.Update("a1", nil, nil, fixed())
	}
	if s.Get("a1") > 1 {
		t.Fatal("trust must not exceed 1")
	}
}
