namespace GuardianForge.Core;

/// <summary>A tiny concurrency-safe counter registry exposing Prometheus text format.</summary>
public sealed class Metrics
{
    private readonly object _lock = new();
    private readonly Dictionary<string, double> _counters = new();

    public void Inc(string name, double delta)
    {
        lock (_lock) _counters[name] = _counters.GetValueOrDefault(name) + delta;
    }

    public double Get(string name)
    {
        lock (_lock) return _counters.GetValueOrDefault(name);
    }

    public string Expose()
    {
        lock (_lock)
        {
            var sb = new System.Text.StringBuilder();
            foreach (var name in _counters.Keys.OrderBy(k => k, StringComparer.Ordinal))
                sb.Append($"# TYPE {name} counter\n{name} {_counters[name]}\n");
            return sb.ToString();
        }
    }
}
