using GuardianForge.Core;

namespace GuardianForge.Agents;

/// <summary>The provider-agnostic model interface. The supervisor and audit explainer
/// reason through this; everything else depends only on it.</summary>
public interface ILlmClient
{
    string Name { get; }
    /// <summary>Synthesize a raw JSON <see cref="Decision"/> from a signal (returns text,
    /// so the supervisor owns parsing and validation).</summary>
    Task<string> SynthesizeAsync(Signal signal, CancellationToken ct = default);
    /// <summary>Produce a human-readable narrative for an intervention.</summary>
    Task<string> ExplainAsync(Intervention intervention, Signal signal, CancellationToken ct = default);
}

/// <summary>The deterministic supervisor stand-in used in every test and the default demo.
/// It maps a signal to a decision with fixed rules; it cannot invent an intervention the
/// policy mode doesn't allow, which the supervisor's validator also enforces.</summary>
public sealed class MockLlm : ILlmClient
{
    public string Name => "mock";

    public Task<string> SynthesizeAsync(Signal s, CancellationToken ct = default)
        => Task.FromResult(Json.Serialize(Decide(s)));

    public static Decision Decide(Signal s)
    {
        // A chronically untrusted agent is isolated and escalated regardless of mode.
        if (s.Trust < 0.3)
            return new Decision { Intervene = true, Type = InterventionType.Pause, Escalate = true, Reason = "agent trust below isolation floor; pausing and escalating", Confidence = 0.95 };

        switch (s.MaxMode)
        {
            case PolicyMode.Observe:
                return new Decision { Intervene = false, Reason = "observe-only policy; recording without intervention", Confidence = 0.9 };
            case PolicyMode.Soft:
            {
                var type = InterventionType.Notify;
                var reason = "soft policy; notifying";
                if (s.Anomalies.Count > 0 || Severity.AtLeastHigh(s.MaxSeverity))
                {
                    type = InterventionType.InjectConstraint;
                    reason = "soft policy; injecting a constraint to contain the risk";
                }
                return new Decision { Intervene = true, Type = type, Reason = reason, Confidence = 0.85 };
            }
            case PolicyMode.Hard:
                if (s.Event.Type == EventType.ToolCall && s.Violations.Count > 0 && !string.IsNullOrEmpty(s.Event.Tool))
                    return new Decision { Intervene = true, Type = InterventionType.RevokeTool, Reason = $"hard policy; revoking tool \"{s.Event.Tool}\"", Confidence = 0.9 };
                return new Decision { Intervene = true, Type = InterventionType.Pause, Reason = "hard policy; pausing the agent", Confidence = 0.9 };
            case PolicyMode.Escalate:
                return new Decision { Intervene = true, Type = InterventionType.Escalate, Escalate = true, Reason = "policy requires human review", Confidence = 0.8 };
            default:
                return new Decision { Intervene = false, Reason = "no applicable policy", Confidence = 0.7 };
        }
    }

    public Task<string> ExplainAsync(Intervention i, Signal s, CancellationToken ct = default)
        => Task.FromResult(
            $"Intervention {i.InterventionId} ({i.Type}) was issued against agent {i.TargetAgentId} because {i.Reason}. " +
            $"At decision time the agent had {s.Violations.Count} policy violation(s), {s.Anomalies.Count} anomaly(ies), and a trust score of {s.Trust:0.00}.");
}

/// <summary>Returns a fixed raw response (or throws) so the validator can be driven with
/// adversarial output.</summary>
public sealed class ScriptedLlm : ILlmClient
{
    public string Raw { get; init; } = "";
    public Exception? Error { get; init; }
    public string Narrative { get; init; } = "";

    public string Name => "scripted";

    public Task<string> SynthesizeAsync(Signal signal, CancellationToken ct = default)
        => Error != null ? Task.FromException<string>(Error) : Task.FromResult(Raw);

    public Task<string> ExplainAsync(Intervention intervention, Signal signal, CancellationToken ct = default)
        => Task.FromResult(Narrative);
}
