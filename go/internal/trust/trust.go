// Package trust maintains a running trust score per agent. Trust starts at a baseline and
// decays on violations and anomalies (weighted by severity), recovering slowly on clean
// behaviour. Low trust is itself a governance signal - a chronically misbehaving agent can
// be recommended for isolation. The update is deterministic.
package trust

import (
	"sync"
	"time"

	"github.com/parag-labs/guardianforge/go/internal/models"
)

// Baseline is the starting trust for a new agent.
const Baseline = 0.8

// Scorer maintains per-agent trust.
type Scorer struct {
	mu     sync.Mutex
	scores map[string]models.TrustScore
}

// New builds an empty scorer.
func New() *Scorer {
	return &Scorer{scores: map[string]models.TrustScore{}}
}

// Get returns an agent's current trust (Baseline if unseen).
func (s *Scorer) Get(agentID string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sc, ok := s.scores[agentID]; ok {
		return sc.Score
	}
	return Baseline
}

// Score returns the full trust record for an agent.
func (s *Scorer) Score(agentID string) models.TrustScore {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sc, ok := s.scores[agentID]; ok {
		return sc
	}
	return models.TrustScore{AgentID: agentID, Score: Baseline}
}

// severityPenalty is how much trust each violation severity removes.
func severityPenalty(sev models.Severity) float64 {
	switch sev {
	case models.SeverityCritical:
		return 0.30
	case models.SeverityHigh:
		return 0.15
	case models.SeverityMedium:
		return 0.08
	case models.SeverityLow:
		return 0.03
	default:
		return 0.0
	}
}

// Update adjusts an agent's trust from the violations and anomalies on one event, and a
// small recovery when the event was clean. Returns the new score.
func (s *Scorer) Update(agentID string, violations []models.Violation, anomalies []models.Anomaly, now time.Time) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	sc, ok := s.scores[agentID]
	if !ok {
		sc = models.TrustScore{AgentID: agentID, Score: Baseline}
	}
	var factors []models.TrustFactor

	if len(violations) == 0 && len(anomalies) == 0 {
		// Clean behaviour recovers trust slowly toward 1.0.
		delta := 0.01
		sc.Score = clamp(sc.Score + delta)
		factors = append(factors, models.TrustFactor{Name: "clean_behaviour", Delta: delta, Reason: "no violation or anomaly"})
	} else {
		for _, v := range violations {
			p := severityPenalty(v.Severity)
			sc.Score = clamp(sc.Score - p)
			factors = append(factors, models.TrustFactor{Name: "violation", Delta: -p, Reason: v.Reason})
		}
		for _, a := range anomalies {
			p := 0.05 * a.Score
			sc.Score = clamp(sc.Score - p)
			factors = append(factors, models.TrustFactor{Name: "anomaly:" + string(a.Type), Delta: -p, Reason: a.Detail})
		}
	}
	sc.Factors = factors
	sc.UpdatedAt = now.UTC()
	s.scores[agentID] = sc
	return sc.Score
}

// ShouldIsolate reports whether an agent's trust has decayed below the isolation floor.
func (s *Scorer) ShouldIsolate(agentID string) bool {
	return s.Get(agentID) < 0.3
}

func clamp(f float64) float64 {
	if f > 1 {
		return 1
	}
	if f < 0 {
		return 0
	}
	return f
}
