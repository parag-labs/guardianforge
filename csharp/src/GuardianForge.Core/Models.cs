namespace GuardianForge.Core;

// GuardianForge's governance contracts. The "enum" domains are modeled as string
// constants (mirroring the Go version's string types) so the wire format is plain strings
// and validation of model output is explicit rather than a deserialization side effect.

/// <summary>Event types emitted by monitored agents.</summary>
public static class EventType
{
    public const string ToolCall = "TOOL_CALL";
    public const string ToolResult = "TOOL_RESULT";
    public const string Message = "MESSAGE";
    public const string Decision = "DECISION";
    public const string Error = "ERROR";
    public const string Heartbeat = "HEARTBEAT";
}

/// <summary>Signal severities, ordered by <see cref="Severity.Rank"/>.</summary>
public static class Severity
{
    public const string Info = "info";
    public const string Low = "low";
    public const string Medium = "medium";
    public const string High = "high";
    public const string Critical = "critical";

    public static int Rank(string s) => s switch
    {
        Info => 0, Low => 1, Medium => 2, High => 3, Critical => 4, _ => 0,
    };

    public static bool AtLeastHigh(string s) => s == High || s == Critical;
}

/// <summary>Policy enforcement modes, ordered by <see cref="PolicyMode.Rank"/>.</summary>
public static class PolicyMode
{
    public const string Observe = "OBSERVE";
    public const string Soft = "SOFT";
    public const string Hard = "HARD";
    public const string Escalate = "ESCALATE";

    public static int Rank(string m) => m switch
    {
        Observe => 0, Soft => 1, Hard => 2, Escalate => 3, _ => 0,
    };
}

/// <summary>Rule comparison operators.</summary>
public static class Op
{
    public const string Equals = "eq";
    public const string NotEquals = "ne";
    public const string Contains = "contains";
    public const string Matches = "matches";
    public const string CountOver = "count_over";
}

/// <summary>Anomaly kinds.</summary>
public static class AnomalyType
{
    public const string Loop = "loop";
    public const string RateSpike = "rate_spike";
    public const string Cyclic = "cyclic";
}

/// <summary>Intervention kinds. This is the closed set the supervisor may name.</summary>
public static class InterventionType
{
    public const string Pause = "PAUSE";
    public const string RevokeTool = "REVOKE_TOOL";
    public const string InjectConstraint = "INJECT_CONSTRAINT";
    public const string ForceReplan = "FORCE_REPLAN";
    public const string Notify = "NOTIFY";
    public const string Escalate = "ESCALATE";

    public static readonly HashSet<string> All = new()
    {
        Pause, RevokeTool, InjectConstraint, ForceReplan, Notify, Escalate,
    };
}

/// <summary>One observation from a monitored agent.</summary>
public sealed class AgentEvent
{
    public string EventId { get; set; } = "";
    public string AgentId { get; set; } = "";
    public string SessionId { get; set; } = "";
    public string FleetId { get; set; } = "";
    public DateTimeOffset Timestamp { get; set; }
    public string Type { get; set; } = "";
    public string Tool { get; set; } = "";
    public string Action { get; set; } = "";
    public Dictionary<string, string> Metadata { get; set; } = new();
}

/// <summary>One structured condition on an event.</summary>
public sealed class Rule
{
    public string RuleId { get; set; } = "";
    public string Field { get; set; } = "";
    public string Op { get; set; } = "";
    public string Value { get; set; } = "";
    public int Threshold { get; set; }
    public int WindowSize { get; set; }
    public string Severity { get; set; } = Core.Severity.Info;
}

/// <summary>A named set of rules with an enforcement mode.</summary>
public sealed class Policy
{
    public string PolicyId { get; set; } = "";
    public string Name { get; set; } = "";
    public string Version { get; set; } = "";
    public string Scope { get; set; } = "GLOBAL";
    public List<Rule> Rules { get; set; } = new();
    public string Mode { get; set; } = PolicyMode.Observe;
}

/// <summary>An event that breached a rule.</summary>
public sealed class Violation
{
    public string PolicyId { get; set; } = "";
    public string RuleId { get; set; } = "";
    public string AgentId { get; set; } = "";
    public string Severity { get; set; } = Core.Severity.Info;
    public string Reason { get; set; } = "";
}

/// <summary>A scored behavioural anomaly.</summary>
public sealed class Anomaly
{
    public string AgentId { get; set; } = "";
    public string Type { get; set; } = "";
    public double Score { get; set; }
    public string Detail { get; set; } = "";
}

/// <summary>A corrective action targeted at an agent.</summary>
public sealed class Intervention
{
    public string InterventionId { get; set; } = "";
    public string TargetAgentId { get; set; } = "";
    public string TargetSessionId { get; set; } = "";
    public string Type { get; set; } = "";
    public string Reason { get; set; } = "";
    public Dictionary<string, string> Parameters { get; set; } = new();
    public string IssuedBy { get; set; } = "";
    public DateTimeOffset IssuedAt { get; set; }
}

/// <summary>One contributor to a trust score.</summary>
public sealed class TrustFactor
{
    public string Name { get; set; } = "";
    public double Delta { get; set; }
    public string Reason { get; set; } = "";
}

/// <summary>An agent's running trust value.</summary>
public sealed class TrustScore
{
    public string AgentId { get; set; } = "";
    public double Score { get; set; }
    public List<TrustFactor> Factors { get; set; } = new();
    public DateTimeOffset UpdatedAt { get; set; }
}

/// <summary>One hash-chained audit record.</summary>
public sealed class AuditEntry
{
    public string EntryId { get; set; } = "";
    public string PreviousHash { get; set; } = "";
    public string CurrentHash { get; set; } = "";
    public DateTimeOffset Timestamp { get; set; }
    public string Actor { get; set; } = "";
    public string Action { get; set; } = "";
    public string Details { get; set; } = "";
}

/// <summary>The aggregated input handed to the supervisor.</summary>
public sealed class Signal
{
    public AgentEvent Event { get; set; } = new();
    public List<Violation> Violations { get; set; } = new();
    public List<Anomaly> Anomalies { get; set; } = new();
    public double Trust { get; set; }
    public string MaxMode { get; set; } = PolicyMode.Observe;
    public string MaxSeverity { get; set; } = Severity.Info;
}

/// <summary>The supervisor's decision: intervene or observe, with rationale.</summary>
public sealed class Decision
{
    public bool Intervene { get; set; }
    public string? Type { get; set; }
    public string Reason { get; set; } = "";
    public bool Escalate { get; set; }
    public double Confidence { get; set; }
}
