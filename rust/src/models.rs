//! Typed governance contracts, ported from the Go `internal/models` package.
//!
//! The enum-like domains are modelled as open string constants exactly like the Go
//! reference's `type X string` declarations, so an out-of-band value stays representable
//! and each evaluator's default branch remains reachable.

use std::collections::HashMap;

// --- EventType -------------------------------------------------------------------------

/// An agent invoked a tool.
pub const EVENT_TOOL_CALL: &str = "TOOL_CALL";
/// A tool returned a result.
pub const EVENT_TOOL_RESULT: &str = "TOOL_RESULT";
/// A free-form message.
pub const EVENT_MESSAGE: &str = "MESSAGE";
/// A decision was taken.
pub const EVENT_DECISION: &str = "DECISION";
/// An error was raised.
pub const EVENT_ERROR: &str = "ERROR";
/// A periodic liveness signal.
pub const EVENT_HEARTBEAT: &str = "HEARTBEAT";

// --- Severity --------------------------------------------------------------------------

/// Informational severity.
pub const SEVERITY_INFO: &str = "info";
/// Low severity.
pub const SEVERITY_LOW: &str = "low";
/// Medium severity.
pub const SEVERITY_MEDIUM: &str = "medium";
/// High severity.
pub const SEVERITY_HIGH: &str = "high";
/// Critical severity.
pub const SEVERITY_CRITICAL: &str = "critical";

/// Rank a severity for comparison; unknown values rank lowest (0), matching Go.
pub fn severity_rank(sev: &str) -> i32 {
    match sev {
        SEVERITY_INFO => 0,
        SEVERITY_LOW => 1,
        SEVERITY_MEDIUM => 2,
        SEVERITY_HIGH => 3,
        SEVERITY_CRITICAL => 4,
        _ => 0,
    }
}

/// Return whichever severity ranks higher (`b` wins only if strictly higher).
pub fn higher_severity(a: &str, b: &str) -> String {
    if severity_rank(b) > severity_rank(a) {
        b.to_string()
    } else {
        a.to_string()
    }
}

// --- PolicyMode ------------------------------------------------------------------------

/// Observe only; never intervene.
pub const MODE_OBSERVE: &str = "OBSERVE";
/// Soft enforcement.
pub const MODE_SOFT: &str = "SOFT";
/// Hard enforcement.
pub const MODE_HARD: &str = "HARD";
/// Escalate to a human.
pub const MODE_ESCALATE: &str = "ESCALATE";

/// Rank an enforcement mode; unknown values rank lowest (0), matching Go.
pub fn mode_rank(mode: &str) -> i32 {
    match mode {
        MODE_OBSERVE => 0,
        MODE_SOFT => 1,
        MODE_HARD => 2,
        MODE_ESCALATE => 3,
        _ => 0,
    }
}

/// Return whichever mode is stricter (`b` wins only if strictly stricter).
pub fn stricter_mode(a: &str, b: &str) -> String {
    if mode_rank(b) > mode_rank(a) {
        b.to_string()
    } else {
        a.to_string()
    }
}

// --- PolicyScope -----------------------------------------------------------------------

/// Applies to every agent.
pub const SCOPE_GLOBAL: &str = "GLOBAL";
/// Applies to a fleet.
pub const SCOPE_FLEET: &str = "FLEET";
/// Applies to a role.
pub const SCOPE_AGENT_ROLE: &str = "AGENT_ROLE";
/// Applies to a session.
pub const SCOPE_SESSION: &str = "SESSION";

// --- Op --------------------------------------------------------------------------------

/// Equality comparison.
pub const OP_EQUALS: &str = "eq";
/// Inequality comparison.
pub const OP_NOT_EQUALS: &str = "ne";
/// Substring containment.
pub const OP_CONTAINS: &str = "contains";
/// Regular-expression match.
pub const OP_MATCHES: &str = "matches";
/// Greater-than comparison (declared in the reference but not wired into matching).
pub const OP_GREATER: &str = "gt";
/// Windowed occurrence count.
pub const OP_COUNT_OVER: &str = "count_over";

// --- AnomalyType -----------------------------------------------------------------------

/// The same action repeated back to back.
pub const ANOMALY_LOOP: &str = "loop";
/// Too many events in the window.
pub const ANOMALY_RATE_SPIKE: &str = "rate_spike";
/// An A-B-A-B oscillation.
pub const ANOMALY_CYCLIC: &str = "cyclic";

// --- InterventionType ------------------------------------------------------------------

/// Pause the agent.
pub const INTERVENTION_PAUSE: &str = "PAUSE";
/// Revoke a tool.
pub const INTERVENTION_REVOKE_TOOL: &str = "REVOKE_TOOL";
/// Inject a constraint.
pub const INTERVENTION_INJECT_CONSTRAINT: &str = "INJECT_CONSTRAINT";
/// Force a replan.
pub const INTERVENTION_FORCE_REPLAN: &str = "FORCE_REPLAN";
/// Notify a human.
pub const INTERVENTION_NOTIFY: &str = "NOTIFY";
/// Escalate to a human.
pub const INTERVENTION_ESCALATE: &str = "ESCALATE";

/// One observation emitted by a monitored agent.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct AgentEvent {
    /// The unique event identifier.
    pub event_id: String,
    /// The agent that emitted it.
    pub agent_id: String,
    /// The session it belongs to.
    pub session_id: String,
    /// The fleet it belongs to.
    pub fleet_id: String,
    /// When it occurred (unix nanoseconds; never inspected by the core).
    pub timestamp: i64,
    /// The event type.
    pub r#type: String,
    /// The tool involved, if any.
    pub tool: String,
    /// The action taken, if any.
    pub action: String,
    /// Free-form metadata.
    pub metadata: HashMap<String, String>,
}

/// One structured condition on an event.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Rule {
    /// The rule identifier.
    pub rule_id: String,
    /// The event field the rule inspects.
    pub field: String,
    /// The comparison operator.
    pub op: String,
    /// The comparison value.
    pub value: String,
    /// The count threshold (for `count_over`).
    pub threshold: i64,
    /// The sliding window size (for `count_over`).
    pub window_size: usize,
    /// The severity emitted when the rule fires.
    pub severity: String,
}

/// A named set of rules with an enforcement mode.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Policy {
    /// The policy identifier.
    pub policy_id: String,
    /// A human-readable name.
    pub name: String,
    /// The policy version.
    pub version: String,
    /// The scope the policy applies to.
    pub scope: String,
    /// The rules that make up the policy.
    pub rules: Vec<Rule>,
    /// The enforcement mode.
    pub mode: String,
    /// Free-form labels.
    pub labels: HashMap<String, String>,
}

/// A record that an event breached a rule.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Violation {
    /// The policy that fired.
    pub policy_id: String,
    /// The rule that matched.
    pub rule_id: String,
    /// The offending agent.
    pub agent_id: String,
    /// The severity of the breach.
    pub severity: String,
    /// A human-readable explanation.
    pub reason: String,
}

/// A scored behavioural anomaly for an agent.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Anomaly {
    /// The agent the anomaly concerns.
    pub agent_id: String,
    /// The anomaly type.
    pub r#type: String,
    /// The score in `[0, 1]`.
    pub score: f64,
    /// A human-readable detail.
    pub detail: String,
}

/// A corrective action targeted at an agent/session.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Intervention {
    /// The intervention identifier.
    pub intervention_id: String,
    /// The targeted agent.
    pub target_agent_id: String,
    /// The targeted session.
    pub target_session_id: String,
    /// The intervention type.
    pub r#type: String,
    /// Why it was issued.
    pub reason: String,
    /// Free-form parameters.
    pub parameters: HashMap<String, String>,
    /// Who issued it.
    pub issued_by: String,
    /// When it was issued (unix nanoseconds).
    pub issued_at: i64,
}

/// One contributor to an agent's trust score.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct TrustFactor {
    /// The factor name.
    pub name: String,
    /// The delta applied to the score.
    pub delta: f64,
    /// The reason for the delta.
    pub reason: String,
}

/// An agent's running trust value.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct TrustScore {
    /// The agent the score concerns.
    pub agent_id: String,
    /// The current score in `[0, 1]`.
    pub score: f64,
    /// The factors from the most recent update.
    pub factors: Vec<TrustFactor>,
    /// When it was last updated (unix nanoseconds).
    pub updated_at: i64,
}

/// One hash-chained record in the tamper-evident audit log.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct AuditEntry {
    /// The entry identifier.
    pub entry_id: String,
    /// The previous entry's hash.
    pub previous_hash: String,
    /// This entry's hash.
    pub current_hash: String,
    /// When it was recorded (unix nanoseconds).
    pub timestamp: i64,
    /// Who acted.
    pub actor: String,
    /// The action recorded.
    pub action: String,
    /// Free-form details.
    pub details: String,
}

/// The aggregated input handed to the supervisor.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Signal {
    /// The triggering event.
    pub event: AgentEvent,
    /// The violations found.
    pub violations: Vec<Violation>,
    /// The anomalies found.
    pub anomalies: Vec<Anomaly>,
    /// The agent's trust.
    pub trust: f64,
    /// The strictest mode among fired policies.
    pub max_mode: String,
    /// The highest severity among violations.
    pub max_severity: String,
}

/// The supervisor's validated output.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Decision {
    /// Whether to intervene.
    pub intervene: bool,
    /// The intervention type, if any.
    pub r#type: String,
    /// The rationale.
    pub reason: String,
    /// Whether to escalate to a human.
    pub escalate: bool,
    /// The confidence in `[0, 1]`.
    pub confidence: f64,
}
