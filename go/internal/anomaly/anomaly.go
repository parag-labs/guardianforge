// Package anomaly is the statistical/pattern anomaly layer. It watches each agent's recent
// behaviour and flags loops (the same tool hammered over and over), rate spikes, and
// cyclic A-B-A-B patterns. It is deterministic and cheap - it runs on every event before
// any LLM is consulted.
package anomaly

import (
	"fmt"
	"sync"

	"github.com/parag-labs/guardianforge/go/internal/models"
)

// Detector keeps a bounded per-agent history and scores anomalies.
type Detector struct {
	mu      sync.Mutex
	history map[string][]string
	window  int
	// loopThreshold: how many identical calls in the window trips a loop.
	loopThreshold int
	// rateThreshold: how many events in the window trips a rate spike.
	rateThreshold int
}

// New builds a detector with sensible defaults.
func New() *Detector {
	return &Detector{
		history:       map[string][]string{},
		window:        12,
		loopThreshold: 5,
		rateThreshold: 10,
	}
}

// fingerprint of an event for pattern matching.
func fp(ev models.AgentEvent) string {
	return string(ev.Type) + ":" + ev.Tool + ":" + ev.Action
}

// Observe records an event and returns any anomalies detected for the agent.
func (d *Detector) Observe(ev models.AgentEvent) []models.Anomaly {
	d.mu.Lock()
	defer d.mu.Unlock()

	h := append(d.history[ev.AgentID], fp(ev))
	if len(h) > d.window {
		h = h[len(h)-d.window:]
	}
	d.history[ev.AgentID] = h

	var out []models.Anomaly
	if a, ok := d.detectLoop(ev.AgentID, h); ok {
		out = append(out, a)
	}
	if a, ok := d.detectRate(ev.AgentID, h); ok {
		out = append(out, a)
	}
	if a, ok := d.detectCyclic(ev.AgentID, h); ok {
		out = append(out, a)
	}
	return out
}

// detectLoop flags many identical fingerprints among the most recent events.
func (d *Detector) detectLoop(agent string, h []string) (models.Anomaly, bool) {
	if len(h) == 0 {
		return models.Anomaly{}, false
	}
	last := h[len(h)-1]
	count := 0
	for i := len(h) - 1; i >= 0 && h[i] == last; i-- {
		count++
	}
	if count >= d.loopThreshold {
		score := clamp(float64(count) / float64(d.window))
		return models.Anomaly{
			AgentID: agent, Type: models.AnomalyLoop, Score: round(score),
			Detail: fmt.Sprintf("%d consecutive identical actions (%s)", count, last),
		}, true
	}
	return models.Anomaly{}, false
}

// detectRate flags a burst: the window is (nearly) full of events for one agent.
func (d *Detector) detectRate(agent string, h []string) (models.Anomaly, bool) {
	if len(h) >= d.rateThreshold {
		score := clamp(float64(len(h)) / float64(d.window))
		return models.Anomaly{
			AgentID: agent, Type: models.AnomalyRateSpike, Score: round(score),
			Detail: fmt.Sprintf("%d events within the observation window", len(h)),
		}, true
	}
	return models.Anomaly{}, false
}

// detectCyclic flags an alternating A-B-A-B pattern over the last several events.
func (d *Detector) detectCyclic(agent string, h []string) (models.Anomaly, bool) {
	if len(h) < 6 {
		return models.Anomaly{}, false
	}
	tail := h[len(h)-6:]
	a, b := tail[0], tail[1]
	if a == b {
		return models.Anomaly{}, false
	}
	for i, x := range tail {
		want := a
		if i%2 == 1 {
			want = b
		}
		if x != want {
			return models.Anomaly{}, false
		}
	}
	return models.Anomaly{
		AgentID: agent, Type: models.AnomalyCyclic, Score: 0.7,
		Detail: fmt.Sprintf("cyclic pattern between %q and %q", a, b),
	}, true
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

func round(f float64) float64 { return float64(int(f*100+0.5)) / 100 }
