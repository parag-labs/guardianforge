/**
 * GuardianForge's typed governance contracts.
 *
 * These mirror the Go `internal/models` package: the events flowing from monitored
 * agents, the policies evaluated against them, the interventions issued, trust scores,
 * and the hash-chained audit entries. The "enum" domains are modelled as plain string
 * constants (matching the Go string types) so the wire format is strings and validation
 * is explicit.
 */

// --- EventType -------------------------------------------------------------------------

export const EVENT_TOOL_CALL = "TOOL_CALL";
export const EVENT_TOOL_RESULT = "TOOL_RESULT";
export const EVENT_MESSAGE = "MESSAGE";
export const EVENT_DECISION = "DECISION";
export const EVENT_ERROR = "ERROR";
export const EVENT_HEARTBEAT = "HEARTBEAT";

// --- Severity --------------------------------------------------------------------------

export const SEVERITY_INFO = "info";
export const SEVERITY_LOW = "low";
export const SEVERITY_MEDIUM = "medium";
export const SEVERITY_HIGH = "high";
export const SEVERITY_CRITICAL = "critical";

const SEVERITY_RANK: Record<string, number> = {
  [SEVERITY_INFO]: 0,
  [SEVERITY_LOW]: 1,
  [SEVERITY_MEDIUM]: 2,
  [SEVERITY_HIGH]: 3,
  [SEVERITY_CRITICAL]: 4,
};

/** Order severities; unknown values rank lowest (0), matching the Go default. */
export function severityRank(sev: string): number {
  return SEVERITY_RANK[sev] ?? 0;
}

/** Return whichever severity ranks higher (`b` wins only if strictly higher). */
export function higherSeverity(a: string, b: string): string {
  return severityRank(b) > severityRank(a) ? b : a;
}

// --- PolicyMode ------------------------------------------------------------------------

export const MODE_OBSERVE = "OBSERVE";
export const MODE_SOFT = "SOFT";
export const MODE_HARD = "HARD";
export const MODE_ESCALATE = "ESCALATE";

const MODE_RANK: Record<string, number> = {
  [MODE_OBSERVE]: 0,
  [MODE_SOFT]: 1,
  [MODE_HARD]: 2,
  [MODE_ESCALATE]: 3,
};

/** Order enforcement strictness; unknown values rank lowest (0). */
export function modeRank(mode: string): number {
  return MODE_RANK[mode] ?? 0;
}

/** Return whichever mode is stricter (`b` wins only if strictly stricter). */
export function stricterMode(a: string, b: string): string {
  return modeRank(b) > modeRank(a) ? b : a;
}

// --- PolicyScope -----------------------------------------------------------------------

export const SCOPE_GLOBAL = "GLOBAL";
export const SCOPE_FLEET = "FLEET";
export const SCOPE_AGENT_ROLE = "AGENT_ROLE";
export const SCOPE_SESSION = "SESSION";

// --- Op --------------------------------------------------------------------------------

export const OP_EQUALS = "eq";
export const OP_NOT_EQUALS = "ne";
export const OP_CONTAINS = "contains";
export const OP_MATCHES = "matches";
export const OP_GREATER = "gt";
export const OP_COUNT_OVER = "count_over";

// --- AnomalyType -----------------------------------------------------------------------

export const ANOMALY_LOOP = "loop";
export const ANOMALY_RATE_SPIKE = "rate_spike";
export const ANOMALY_CYCLIC = "cyclic";

// --- InterventionType ------------------------------------------------------------------

export const INTERVENTION_PAUSE = "PAUSE";
export const INTERVENTION_REVOKE_TOOL = "REVOKE_TOOL";
export const INTERVENTION_INJECT_CONSTRAINT = "INJECT_CONSTRAINT";
export const INTERVENTION_FORCE_REPLAN = "FORCE_REPLAN";
export const INTERVENTION_NOTIFY = "NOTIFY";
export const INTERVENTION_ESCALATE = "ESCALATE";

/** Unix epoch, the zero value for every timestamp field. */
export const EPOCH = new Date(0);

/** One observation emitted by a monitored agent. */
export interface AgentEvent {
  eventId: string;
  agentId: string;
  sessionId: string;
  fleetId: string;
  timestamp: Date;
  type: string;
  tool: string;
  action: string;
  metadata: Record<string, string>;
}

/** Build an AgentEvent, defaulting every unset field to its zero value. */
export function agentEvent(init: Partial<AgentEvent> = {}): AgentEvent {
  return {
    eventId: "",
    agentId: "",
    sessionId: "",
    fleetId: "",
    timestamp: EPOCH,
    type: "",
    tool: "",
    action: "",
    metadata: {},
    ...init,
  };
}

/** One structured condition on an event (deterministic, not natural language). */
export interface Rule {
  ruleId: string;
  field: string;
  op: string;
  value: string;
  threshold: number;
  windowSize: number;
  severity: string;
}

/** Build a Rule with zero-value defaults. */
export function rule(init: Partial<Rule> = {}): Rule {
  return {
    ruleId: "",
    field: "",
    op: "",
    value: "",
    threshold: 0,
    windowSize: 0,
    severity: SEVERITY_INFO,
    ...init,
  };
}

/** A named set of rules with an enforcement mode. */
export interface Policy {
  policyId: string;
  name: string;
  version: string;
  scope: string;
  rules: Rule[];
  mode: string;
  labels: Record<string, string>;
}

/** Build a Policy with zero-value defaults. */
export function policy(init: Partial<Policy> = {}): Policy {
  return {
    policyId: "",
    name: "",
    version: "",
    scope: SCOPE_GLOBAL,
    rules: [],
    mode: MODE_OBSERVE,
    labels: {},
    ...init,
  };
}

/** A record that an event breached a rule. */
export interface Violation {
  policyId: string;
  ruleId: string;
  agentId: string;
  severity: string;
  reason: string;
}

/** Build a Violation with zero-value defaults. */
export function violation(init: Partial<Violation> = {}): Violation {
  return { policyId: "", ruleId: "", agentId: "", severity: SEVERITY_INFO, reason: "", ...init };
}

/** A scored behavioural anomaly for an agent. */
export interface Anomaly {
  agentId: string;
  type: string;
  score: number;
  detail: string;
}

/** Build an Anomaly with zero-value defaults. */
export function anomaly(init: Partial<Anomaly> = {}): Anomaly {
  return { agentId: "", type: "", score: 0, detail: "", ...init };
}

/** A corrective action targeted at an agent/session. */
export interface Intervention {
  interventionId: string;
  targetAgentId: string;
  targetSessionId: string;
  type: string;
  reason: string;
  parameters: Record<string, string>;
  issuedBy: string;
  issuedAt: Date;
}

/** One contributor to an agent's trust score. */
export interface TrustFactor {
  name: string;
  delta: number;
  reason: string;
}

/** An agent's running trust value. */
export interface TrustScore {
  agentId: string;
  score: number;
  factors: TrustFactor[];
  updatedAt: Date;
}

/** One hash-chained record in the tamper-evident audit log. */
export interface AuditEntry {
  entryId: string;
  previousHash: string;
  currentHash: string;
  timestamp: Date;
  actor: string;
  action: string;
  details: string;
}

/** The aggregated input handed to the supervisor. */
export interface Signal {
  event: AgentEvent;
  violations: Violation[];
  anomalies: Anomaly[];
  trust: number;
  maxMode: string;
  maxSeverity: string;
}

/** The supervisor's validated output: intervene or observe, with rationale. */
export interface Decision {
  intervene: boolean;
  type: string;
  reason: string;
  escalate: boolean;
  confidence: number;
}
