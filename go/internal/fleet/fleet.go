// Package fleet is the synthetic monitored agent fleet - the stand-in for the real agent
// systems GuardianForge governs. Interventions act on this state (pausing an agent,
// revoking a tool, injecting a constraint), and the scenario generators emit event
// streams that deliberately misbehave so the governance loop has something to catch.
package fleet

import (
	"sync"
)

// AgentState is one monitored agent's live governance state.
type AgentState struct {
	AgentID      string          `json:"agent_id"`
	Paused       bool            `json:"paused"`
	RevokedTools map[string]bool `json:"revoked_tools"`
	Constraints  []string        `json:"constraints"`
}

// Fleet is a concurrency-safe set of monitored agents.
type Fleet struct {
	mu     sync.Mutex
	agents map[string]*AgentState
}

// New builds an empty fleet.
func New() *Fleet { return &Fleet{agents: map[string]*AgentState{}} }

func (f *Fleet) state(id string) *AgentState {
	s, ok := f.agents[id]
	if !ok {
		s = &AgentState{AgentID: id, RevokedTools: map[string]bool{}}
		f.agents[id] = s
	}
	return s
}

// Pause halts an agent.
func (f *Fleet) Pause(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state(id).Paused = true
}

// RevokeTool removes a tool from an agent.
func (f *Fleet) RevokeTool(id, tool string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state(id).RevokedTools[tool] = true
}

// InjectConstraint attaches a constraint to an agent.
func (f *Fleet) InjectConstraint(id, constraint string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.state(id)
	s.Constraints = append(s.Constraints, constraint)
}

// IsPaused reports whether an agent is paused.
func (f *Fleet) IsPaused(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state(id).Paused
}

// IsToolRevoked reports whether a tool is revoked for an agent.
func (f *Fleet) IsToolRevoked(id, tool string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state(id).RevokedTools[tool]
}

// Snapshot returns a copy of an agent's state.
func (f *Fleet) Snapshot(id string) AgentState {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.state(id)
	revoked := make(map[string]bool, len(s.RevokedTools))
	for k, v := range s.RevokedTools {
		revoked[k] = v
	}
	return AgentState{AgentID: s.AgentID, Paused: s.Paused, RevokedTools: revoked, Constraints: append([]string(nil), s.Constraints...)}
}
