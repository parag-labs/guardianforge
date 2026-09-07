// Package models holds GuardianForge's typed governance contracts: the events flowing
// from monitored agents, the policies evaluated against them, the interventions issued,
// trust scores, and the hash-chained audit entries. Deterministic components and the AI
// layer share these types so nothing is passed around untyped.
package models

import "time"

// EventType classifies an event emitted by a monitored agent.
type EventType string

const (
	EventToolCall   EventType = "TOOL_CALL"
	EventToolResult EventType = "TOOL_RESULT"
	EventMessage    EventType = "MESSAGE"
	EventDecision   EventType = "DECISION"
	EventError      EventType = "ERROR"
	EventHeartbeat  EventType = "HEARTBEAT"
)

// AgentEvent is one observation from a monitored agent fleet.
type AgentEvent struct {
	EventID   string    `json:"event_id"`
	AgentID   string    `json:"agent_id"`
	SessionID string    `json:"session_id"`
	FleetID   string    `json:"fleet_id"`
	Timestamp time.Time `json:"timestamp"`
	Type      EventType `json:"type"`
	// Tool is set for TOOL_CALL / TOOL_RESULT events.
	Tool string `json:"tool,omitempty"`
	// Action is a free-form action name (e.g. a decision or message intent).
	Action   string            `json:"action,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Severity ranks how serious a signal is.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// PolicyMode controls how strongly a policy's violations are enforced.
type PolicyMode string

const (
	// ModeObserve records violations but never intervenes.
	ModeObserve PolicyMode = "OBSERVE"
	// ModeSoft issues non-disruptive interventions (notify, inject constraint).
	ModeSoft PolicyMode = "SOFT"
	// ModeHard issues disruptive interventions (pause, revoke tool).
	ModeHard PolicyMode = "HARD"
	// ModeEscalate routes to a human.
	ModeEscalate PolicyMode = "ESCALATE"
)

// PolicyScope narrows which agents a policy applies to.
type PolicyScope string

const (
	ScopeGlobal    PolicyScope = "GLOBAL"
	ScopeFleet     PolicyScope = "FLEET"
	ScopeAgentRole PolicyScope = "AGENT_ROLE"
	ScopeSession   PolicyScope = "SESSION"
)

// Op is a comparison operator used by a structured rule condition.
type Op string

const (
	OpEquals    Op = "eq"
	OpNotEquals Op = "ne"
	OpContains  Op = "contains"
	OpMatches   Op = "matches" // regexp
	OpGreater   Op = "gt"
	OpCountOver Op = "count_over" // count of matching events over threshold
)

// Rule is one structured condition on an event (deterministic, not natural language).
// Keeping rules structured is what lets the policy evaluator be exact and fast; the LLM
// is reserved for genuinely ambiguous interpretation, never for the core check.
type Rule struct {
	RuleID     string   `json:"rule_id"`
	Field      string   `json:"field"` // event field: type, tool, action, metadata.<k>
	Op         Op       `json:"op"`
	Value      string   `json:"value"`
	Threshold  int      `json:"threshold,omitempty"` // for count_over
	WindowSize int      `json:"window_size,omitempty"`
	Severity   Severity `json:"severity"`
}

// Policy is a named set of rules with an enforcement mode.
type Policy struct {
	PolicyID string            `json:"policy_id"`
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Scope    PolicyScope       `json:"scope"`
	Rules    []Rule            `json:"rules"`
	Mode     PolicyMode        `json:"mode"`
	Labels   map[string]string `json:"labels,omitempty"`
}

// Violation records that an event breached a rule.
type Violation struct {
	PolicyID string   `json:"policy_id"`
	RuleID   string   `json:"rule_id"`
	AgentID  string   `json:"agent_id"`
	Severity Severity `json:"severity"`
	Reason   string   `json:"reason"`
}

// AnomalyType classifies a detected behavioral anomaly.
type AnomalyType string

const (
	AnomalyLoop      AnomalyType = "loop"       // repeated identical tool calls
	AnomalyRateSpike AnomalyType = "rate_spike" // burst of events
	AnomalyCyclic    AnomalyType = "cyclic"     // repeating A->B->A->B pattern
)

// Anomaly is a scored behavioral anomaly for an agent.
type Anomaly struct {
	AgentID string      `json:"agent_id"`
	Type    AnomalyType `json:"type"`
	Score   float64     `json:"score"` // 0..1
	Detail  string      `json:"detail"`
}

// InterventionType is the corrective action GuardianForge can take.
type InterventionType string

const (
	InterventionPause            InterventionType = "PAUSE"
	InterventionRevokeTool       InterventionType = "REVOKE_TOOL"
	InterventionInjectConstraint InterventionType = "INJECT_CONSTRAINT"
	InterventionForceReplan      InterventionType = "FORCE_REPLAN"
	InterventionNotify           InterventionType = "NOTIFY"
	InterventionEscalate         InterventionType = "ESCALATE"
)

// Intervention is a corrective action targeted at an agent/session.
type Intervention struct {
	InterventionID  string            `json:"intervention_id"`
	TargetAgentID   string            `json:"target_agent_id"`
	TargetSessionID string            `json:"target_session_id"`
	Type            InterventionType  `json:"type"`
	Reason          string            `json:"reason"`
	Parameters      map[string]string `json:"parameters,omitempty"`
	IssuedBy        string            `json:"issued_by"`
	IssuedAt        time.Time         `json:"issued_at"`
}

// TrustFactor is one contributor to an agent's trust score.
type TrustFactor struct {
	Name   string  `json:"name"`
	Delta  float64 `json:"delta"`
	Reason string  `json:"reason"`
}

// TrustScore is an agent's running trust value.
type TrustScore struct {
	AgentID   string        `json:"agent_id"`
	Score     float64       `json:"score"` // 0..1
	Factors   []TrustFactor `json:"factors,omitempty"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// AuditEntry is one hash-chained record in the tamper-evident audit log.
type AuditEntry struct {
	EntryID      string    `json:"entry_id"`
	PreviousHash string    `json:"previous_hash"`
	CurrentHash  string    `json:"current_hash"`
	Timestamp    time.Time `json:"timestamp"`
	Actor        string    `json:"actor"`
	Action       string    `json:"action"`
	Details      string    `json:"details"`
}

// Signal is the aggregated input handed to the supervisor: everything the deterministic
// layer found about an event.
type Signal struct {
	Event       AgentEvent  `json:"event"`
	Violations  []Violation `json:"violations"`
	Anomalies   []Anomaly   `json:"anomalies"`
	Trust       float64     `json:"trust"`
	MaxMode     PolicyMode  `json:"max_mode"`
	MaxSeverity Severity    `json:"max_severity"`
}

// Decision is the supervisor's validated output: intervene or observe, with rationale.
type Decision struct {
	Intervene  bool             `json:"intervene"`
	Type       InterventionType `json:"type,omitempty"`
	Reason     string           `json:"reason"`
	Escalate   bool             `json:"escalate"`
	Confidence float64          `json:"confidence"`
}
