using GuardianForge.Core;

namespace GuardianForge.Agents;

/// <summary>The result of processing one event.</summary>
public sealed class Outcome
{
    public Signal Signal { get; set; } = new();
    public Decision Decision { get; set; } = new();
    public Intervention? Intervention { get; set; }
    public bool Escalated { get; set; }
    public string? Rejected { get; set; }
}

/// <summary>
/// The governance engine: the event-driven core that runs every event through the
/// deterministic detectors, asks the supervisor for a validated decision, and applies the
/// resulting intervention while recording everything in the hash-chained audit log.
/// </summary>
public sealed class Engine
{
    public PolicyEvaluator Policy { get; }
    public AnomalyDetector Anomaly { get; } = new();
    public TrustScorer Trust { get; } = new();
    public Supervisor Supervisor { get; }
    public InterventionExecutor Executor { get; }
    public AuditLog Audit { get; } = new();
    public Fleet Fleet { get; }
    public Metrics Metrics { get; } = new();

    private readonly object _lock = new();
    private readonly List<Intervention> _hitl = new();
    private int _seq;
    private readonly Func<DateTimeOffset> _now;

    public Engine(IEnumerable<Policy> policies, Fleet fleet, Supervisor supervisor, Func<DateTimeOffset>? now = null)
    {
        Policy = new PolicyEvaluator(policies);
        Fleet = fleet;
        Supervisor = supervisor;
        Executor = new InterventionExecutor(fleet);
        _now = now ?? (() => DateTimeOffset.UtcNow);
    }

    /// <summary>Run one event through the full governance loop.</summary>
    public async Task<Outcome> ProcessAsync(AgentEvent ev, CancellationToken ct = default)
    {
        Metrics.Inc("events_total", 1);

        var (violations, mode, sev) = Policy.Evaluate(ev);
        var anomalies = Anomaly.Observe(ev);

        // A strong anomaly with no matching policy still warrants at least a soft response.
        if (mode == PolicyMode.Observe && anomalies.Any(a => a.Score >= 0.5))
            mode = PolicyMode.Soft;

        var trust = Trust.Update(ev.AgentId, violations, anomalies, _now());
        if (violations.Count > 0) Metrics.Inc("violations_total", violations.Count);
        if (anomalies.Count > 0) Metrics.Inc("anomalies_total", anomalies.Count);

        var signal = new Signal
        {
            Event = ev, Violations = violations, Anomalies = anomalies,
            Trust = trust, MaxMode = mode, MaxSeverity = sev,
        };

        var (decision, rejected) = await Supervisor.DecideAsync(signal, ct);
        var outcome = new Outcome { Signal = signal, Decision = decision, Rejected = rejected };
        if (rejected != null) Metrics.Inc("supervisor_rejections_total", 1);
        AuditDecision(signal, decision);

        if (decision.Intervene)
        {
            var iv = BuildIntervention(ev, decision);
            if (Executor.Execute(iv))
            {
                outcome.Intervention = iv;
                Metrics.Inc("interventions_total", 1);
                Audit.Append("executor", "intervention", Json.Serialize(iv), _now());
                if (decision.Escalate)
                {
                    EnqueueHitl(iv);
                    outcome.Escalated = true;
                    Metrics.Inc("escalations_total", 1);
                }
            }
        }
        return outcome;
    }

    private Intervention BuildIntervention(AgentEvent ev, Decision d)
    {
        string id;
        lock (_lock) { _seq++; id = $"intv-{_seq:D4}"; }
        var parameters = new Dictionary<string, string>();
        if (d.Type == InterventionType.RevokeTool) parameters["tool"] = ev.Tool;
        if (d.Type == InterventionType.InjectConstraint) parameters["constraint"] = d.Reason;
        return new Intervention
        {
            InterventionId = id, TargetAgentId = ev.AgentId, TargetSessionId = ev.SessionId,
            Type = d.Type ?? "", Reason = d.Reason, Parameters = parameters, IssuedBy = "supervisor", IssuedAt = _now(),
        };
    }

    private void AuditDecision(Signal signal, Decision decision)
    {
        var details = Json.Serialize(new
        {
            agent = signal.Event.AgentId, mode = signal.MaxMode, severity = signal.MaxSeverity,
            violations = signal.Violations.Count, anomalies = signal.Anomalies.Count,
            trust = signal.Trust, intervene = decision.Intervene, type = decision.Type,
        });
        Audit.Append("supervisor", "decision", details, _now());
    }

    private void EnqueueHitl(Intervention iv)
    {
        lock (_lock) _hitl.Add(iv);
    }

    public List<Intervention> PendingHitl()
    {
        lock (_lock) return new List<Intervention>(_hitl);
    }

    public bool ResolveHitl(string id, string decision)
    {
        lock (_lock)
        {
            var idx = _hitl.FindIndex(x => x.InterventionId == id);
            if (idx < 0) return false;
            _hitl.RemoveAt(idx);
            Audit.Append("human", "hitl_resolution", $"{id}: {decision}", _now());
            return true;
        }
    }
}
