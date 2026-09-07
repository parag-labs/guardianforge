using System.Text.RegularExpressions;

namespace GuardianForge.Core;

/// <summary>
/// The deterministic policy evaluator: it checks structured rules against events and emits
/// violations, returning the strictest mode and severity that fired. It runs first, before
/// any model - the fast deterministic path decides the clear cases.
/// </summary>
public sealed class PolicyEvaluator
{
    private readonly object _lock = new();
    private List<Policy> _policies;
    private readonly Dictionary<string, List<string>> _history = new();
    private Dictionary<string, Regex> _regexps = new();

    public PolicyEvaluator(IEnumerable<Policy> policies)
    {
        _policies = policies.ToList();
        Compile();
    }

    private void Compile()
    {
        _regexps = new Dictionary<string, Regex>();
        foreach (var p in _policies)
            foreach (var r in p.Rules)
                if (r.Op == Core.Op.Matches)
                    _regexps[r.RuleId] = new Regex(r.Value);
    }

    public List<Policy> Policies() { lock (_lock) return new List<Policy>(_policies); }

    public void SetPolicies(IEnumerable<Policy> policies)
    {
        lock (_lock)
        {
            _policies = policies.ToList();
            Compile();
        }
    }

    /// <summary>Evaluate an event, returning violations plus the strictest mode/severity.</summary>
    public (List<Violation> Violations, string Mode, string Severity) Evaluate(AgentEvent ev)
    {
        lock (_lock)
        {
            var fp = $"{ev.Type}:{ev.Tool}:{ev.Action}";
            if (!_history.TryGetValue(ev.AgentId, out var hist))
            {
                hist = new List<string>();
                _history[ev.AgentId] = hist;
            }
            hist.Add(fp);
            if (hist.Count > 256) hist.RemoveRange(0, hist.Count - 256);

            var violations = new List<Violation>();
            var mode = PolicyMode.Observe;
            var sev = Core.Severity.Info;
            foreach (var p in _policies)
            {
                foreach (var r in p.Rules)
                {
                    if (Matches(ev, r, hist))
                    {
                        violations.Add(new Violation
                        {
                            PolicyId = p.PolicyId, RuleId = r.RuleId, AgentId = ev.AgentId,
                            Severity = r.Severity, Reason = $"rule {r.RuleId} matched on {r.Field}",
                        });
                        if (PolicyMode.Rank(p.Mode) > PolicyMode.Rank(mode)) mode = p.Mode;
                        if (Core.Severity.Rank(r.Severity) > Core.Severity.Rank(sev)) sev = r.Severity;
                    }
                }
            }
            return (violations, mode, sev);
        }
    }

    private bool Matches(AgentEvent ev, Rule r, List<string> hist)
    {
        if (r.Op == Core.Op.CountOver)
        {
            var window = r.WindowSize > 0 ? r.WindowSize : hist.Count;
            var start = Math.Max(0, hist.Count - window);
            var count = 0;
            for (var i = start; i < hist.Count; i++)
                if (hist[i].Contains(r.Value, StringComparison.Ordinal)) count++;
            return count > r.Threshold;
        }

        var val = FieldValue(ev, r.Field);
        return r.Op switch
        {
            Core.Op.Equals => val == r.Value,
            Core.Op.NotEquals => val != r.Value,
            Core.Op.Contains => val.Contains(r.Value, StringComparison.Ordinal),
            Core.Op.Matches => _regexps.TryGetValue(r.RuleId, out var rx) && rx.IsMatch(val),
            _ => false,
        };
    }

    private static string FieldValue(AgentEvent ev, string field)
    {
        if (field == "type") return ev.Type;
        if (field == "tool") return ev.Tool;
        if (field == "action") return ev.Action;
        if (field.StartsWith("metadata.", StringComparison.Ordinal))
            return ev.Metadata.TryGetValue(field["metadata.".Length..], out var v) ? v : "";
        return "";
    }
}
