namespace GuardianForge.Core;

/// <summary>
/// Maintains a running trust score per agent. Trust starts at a baseline, decays on
/// violations and anomalies (weighted by severity), and recovers slowly on clean
/// behaviour. Deterministic. Low trust is itself a governance signal.
/// </summary>
public sealed class TrustScorer
{
    public const double Baseline = 0.8;

    private readonly object _lock = new();
    private readonly Dictionary<string, TrustScore> _scores = new();

    public double Get(string agentId)
    {
        lock (_lock)
            return _scores.TryGetValue(agentId, out var s) ? s.Score : Baseline;
    }

    public TrustScore Score(string agentId)
    {
        lock (_lock)
            return _scores.TryGetValue(agentId, out var s) ? s : new TrustScore { AgentId = agentId, Score = Baseline };
    }

    private static double SeverityPenalty(string sev) => sev switch
    {
        Core.Severity.Critical => 0.30,
        Core.Severity.High => 0.15,
        Core.Severity.Medium => 0.08,
        Core.Severity.Low => 0.03,
        _ => 0.0,
    };

    /// <summary>Update trust from one event's violations/anomalies; clean events recover.</summary>
    public double Update(string agentId, List<Violation> violations, List<Anomaly> anomalies, DateTimeOffset now)
    {
        lock (_lock)
        {
            if (!_scores.TryGetValue(agentId, out var sc))
                sc = new TrustScore { AgentId = agentId, Score = Baseline };

            var factors = new List<TrustFactor>();
            if (violations.Count == 0 && anomalies.Count == 0)
            {
                const double delta = 0.01;
                sc.Score = Clamp(sc.Score + delta);
                factors.Add(new TrustFactor { Name = "clean_behaviour", Delta = delta, Reason = "no violation or anomaly" });
            }
            else
            {
                foreach (var v in violations)
                {
                    var p = SeverityPenalty(v.Severity);
                    sc.Score = Clamp(sc.Score - p);
                    factors.Add(new TrustFactor { Name = "violation", Delta = -p, Reason = v.Reason });
                }
                foreach (var a in anomalies)
                {
                    var p = 0.05 * a.Score;
                    sc.Score = Clamp(sc.Score - p);
                    factors.Add(new TrustFactor { Name = $"anomaly:{a.Type}", Delta = -p, Reason = a.Detail });
                }
            }
            sc.AgentId = agentId;
            sc.Factors = factors;
            sc.UpdatedAt = now.ToUniversalTime();
            _scores[agentId] = sc;
            return sc.Score;
        }
    }

    public bool ShouldIsolate(string agentId) => Get(agentId) < 0.3;

    private static double Clamp(double f) => Math.Max(0.0, Math.Min(1.0, f));
}
