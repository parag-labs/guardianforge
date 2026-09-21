"""GuardianForge's typed governance contracts.

These mirror the Go ``internal/models`` package: the events flowing from monitored agents,
the policies evaluated against them, the interventions issued, trust scores, and the
hash-chained audit entries. The "enum" domains are modelled as plain string constants
(matching the Go string types) so the wire format is strings and validation is explicit.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone

# --- EventType -------------------------------------------------------------------------

EVENT_TOOL_CALL = "TOOL_CALL"
EVENT_TOOL_RESULT = "TOOL_RESULT"
EVENT_MESSAGE = "MESSAGE"
EVENT_DECISION = "DECISION"
EVENT_ERROR = "ERROR"
EVENT_HEARTBEAT = "HEARTBEAT"

# --- Severity --------------------------------------------------------------------------

SEVERITY_INFO = "info"
SEVERITY_LOW = "low"
SEVERITY_MEDIUM = "medium"
SEVERITY_HIGH = "high"
SEVERITY_CRITICAL = "critical"

_SEVERITY_RANK = {
    SEVERITY_INFO: 0,
    SEVERITY_LOW: 1,
    SEVERITY_MEDIUM: 2,
    SEVERITY_HIGH: 3,
    SEVERITY_CRITICAL: 4,
}


def severity_rank(sev: str) -> int:
    """Order severities; unknown values rank lowest (0), matching the Go default."""
    return _SEVERITY_RANK.get(sev, 0)


def higher_severity(a: str, b: str) -> str:
    """Return whichever severity ranks higher (``b`` wins only if strictly higher)."""
    return b if severity_rank(b) > severity_rank(a) else a


# --- PolicyMode ------------------------------------------------------------------------

MODE_OBSERVE = "OBSERVE"
MODE_SOFT = "SOFT"
MODE_HARD = "HARD"
MODE_ESCALATE = "ESCALATE"

_MODE_RANK = {
    MODE_OBSERVE: 0,
    MODE_SOFT: 1,
    MODE_HARD: 2,
    MODE_ESCALATE: 3,
}


def mode_rank(mode: str) -> int:
    """Order enforcement strictness; unknown values rank lowest (0)."""
    return _MODE_RANK.get(mode, 0)


def stricter_mode(a: str, b: str) -> str:
    """Return whichever mode is stricter (``b`` wins only if strictly stricter)."""
    return b if mode_rank(b) > mode_rank(a) else a


# --- PolicyScope -----------------------------------------------------------------------

SCOPE_GLOBAL = "GLOBAL"
SCOPE_FLEET = "FLEET"
SCOPE_AGENT_ROLE = "AGENT_ROLE"
SCOPE_SESSION = "SESSION"

# --- Op --------------------------------------------------------------------------------

OP_EQUALS = "eq"
OP_NOT_EQUALS = "ne"
OP_CONTAINS = "contains"
OP_MATCHES = "matches"
OP_GREATER = "gt"
OP_COUNT_OVER = "count_over"

# --- AnomalyType -----------------------------------------------------------------------

ANOMALY_LOOP = "loop"
ANOMALY_RATE_SPIKE = "rate_spike"
ANOMALY_CYCLIC = "cyclic"

# --- InterventionType ------------------------------------------------------------------

INTERVENTION_PAUSE = "PAUSE"
INTERVENTION_REVOKE_TOOL = "REVOKE_TOOL"
INTERVENTION_INJECT_CONSTRAINT = "INJECT_CONSTRAINT"
INTERVENTION_FORCE_REPLAN = "FORCE_REPLAN"
INTERVENTION_NOTIFY = "NOTIFY"
INTERVENTION_ESCALATE = "ESCALATE"

_EPOCH = datetime.fromtimestamp(0, tz=timezone.utc)


@dataclass
class AgentEvent:
    """One observation emitted by a monitored agent."""

    event_id: str = ""
    agent_id: str = ""
    session_id: str = ""
    fleet_id: str = ""
    timestamp: datetime = _EPOCH
    type: str = ""
    tool: str = ""
    action: str = ""
    metadata: dict[str, str] = field(default_factory=dict)


@dataclass
class Rule:
    """One structured condition on an event (deterministic, not natural language)."""

    rule_id: str = ""
    field: str = ""
    op: str = ""
    value: str = ""
    threshold: int = 0
    window_size: int = 0
    severity: str = SEVERITY_INFO


@dataclass
class Policy:
    """A named set of rules with an enforcement mode."""

    policy_id: str = ""
    name: str = ""
    version: str = ""
    scope: str = SCOPE_GLOBAL
    rules: list[Rule] = field(default_factory=list)
    mode: str = MODE_OBSERVE
    labels: dict[str, str] = field(default_factory=dict)


@dataclass
class Violation:
    """A record that an event breached a rule."""

    policy_id: str = ""
    rule_id: str = ""
    agent_id: str = ""
    severity: str = SEVERITY_INFO
    reason: str = ""


@dataclass
class Anomaly:
    """A scored behavioural anomaly for an agent."""

    agent_id: str = ""
    type: str = ""
    score: float = 0.0
    detail: str = ""


@dataclass
class Intervention:
    """A corrective action targeted at an agent/session."""

    intervention_id: str = ""
    target_agent_id: str = ""
    target_session_id: str = ""
    type: str = ""
    reason: str = ""
    parameters: dict[str, str] = field(default_factory=dict)
    issued_by: str = ""
    issued_at: datetime = _EPOCH


@dataclass
class TrustFactor:
    """One contributor to an agent's trust score."""

    name: str = ""
    delta: float = 0.0
    reason: str = ""


@dataclass
class TrustScore:
    """An agent's running trust value."""

    agent_id: str = ""
    score: float = 0.0
    factors: list[TrustFactor] = field(default_factory=list)
    updated_at: datetime = _EPOCH


@dataclass
class AuditEntry:
    """One hash-chained record in the tamper-evident audit log."""

    entry_id: str = ""
    previous_hash: str = ""
    current_hash: str = ""
    timestamp: datetime = _EPOCH
    actor: str = ""
    action: str = ""
    details: str = ""


@dataclass
class Signal:
    """The aggregated input handed to the supervisor."""

    event: AgentEvent = field(default_factory=AgentEvent)
    violations: list[Violation] = field(default_factory=list)
    anomalies: list[Anomaly] = field(default_factory=list)
    trust: float = 0.0
    max_mode: str = MODE_OBSERVE
    max_severity: str = SEVERITY_INFO


@dataclass
class Decision:
    """The supervisor's validated output: intervene or observe, with rationale."""

    intervene: bool = False
    type: str = ""
    reason: str = ""
    escalate: bool = False
    confidence: float = 0.0
