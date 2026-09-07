namespace GuardianForge.Core;

/// <summary>
/// The statistical/pattern anomaly layer: per-agent detection of tool loops, rate spikes,
/// and cyclic A-B-A-B patterns. Deterministic and cheap; runs on every event before any
/// model is consulted.
/// </summary>
public sealed class AnomalyDetector
{
    private readonly object _lock = new();
    private readonly Dictionary<string, List<string>> _history = new();
    private const int Window = 12;
    private const int LoopThreshold = 5;
    private const int RateThreshold = 10;

    private static string Fp(AgentEvent ev) => $"{ev.Type}:{ev.Tool}:{ev.Action}";

    /// <summary>Record an event and return any anomalies detected for the agent.</summary>
    public List<Anomaly> Observe(AgentEvent ev)
    {
        lock (_lock)
        {
            if (!_history.TryGetValue(ev.AgentId, out var h))
            {
                h = new List<string>();
                _history[ev.AgentId] = h;
            }
            h.Add(Fp(ev));
            if (h.Count > Window) h.RemoveRange(0, h.Count - Window);

            var outList = new List<Anomaly>();
            if (DetectLoop(ev.AgentId, h) is { } loop) outList.Add(loop);
            if (DetectRate(ev.AgentId, h) is { } rate) outList.Add(rate);
            if (DetectCyclic(ev.AgentId, h) is { } cyc) outList.Add(cyc);
            return outList;
        }
    }

    private static Anomaly? DetectLoop(string agent, List<string> h)
    {
        if (h.Count == 0) return null;
        var last = h[^1];
        var count = 0;
        for (var i = h.Count - 1; i >= 0 && h[i] == last; i--) count++;
        if (count >= LoopThreshold)
        {
            var score = Round(Math.Min(1.0, (double)count / Window));
            return new Anomaly { AgentId = agent, Type = AnomalyType.Loop, Score = score, Detail = $"{count} consecutive identical actions ({last})" };
        }
        return null;
    }

    private static Anomaly? DetectRate(string agent, List<string> h)
    {
        if (h.Count >= RateThreshold)
        {
            var score = Round(Math.Min(1.0, (double)h.Count / Window));
            return new Anomaly { AgentId = agent, Type = AnomalyType.RateSpike, Score = score, Detail = $"{h.Count} events within the observation window" };
        }
        return null;
    }

    private static Anomaly? DetectCyclic(string agent, List<string> h)
    {
        if (h.Count < 6) return null;
        var tail = h.GetRange(h.Count - 6, 6);
        var a = tail[0];
        var b = tail[1];
        if (a == b) return null;
        for (var i = 0; i < tail.Count; i++)
        {
            var want = i % 2 == 0 ? a : b;
            if (tail[i] != want) return null;
        }
        return new Anomaly { AgentId = agent, Type = AnomalyType.Cyclic, Score = 0.7, Detail = $"cyclic pattern between \"{a}\" and \"{b}\"" };
    }

    private static double Round(double f) => Math.Round(f, 2);
}
