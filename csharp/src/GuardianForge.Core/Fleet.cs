namespace GuardianForge.Core;

/// <summary>One monitored agent's live governance state.</summary>
public sealed class AgentState
{
    public string AgentId { get; set; } = "";
    public bool Paused { get; set; }
    public HashSet<string> RevokedTools { get; set; } = new();
    public List<string> Constraints { get; set; } = new();
}

/// <summary>
/// The synthetic monitored agent fleet - the stand-in for the real systems GuardianForge
/// governs. Interventions act on this state.
/// </summary>
public sealed class Fleet
{
    private readonly object _lock = new();
    private readonly Dictionary<string, AgentState> _agents = new();

    private AgentState State(string id)
    {
        if (!_agents.TryGetValue(id, out var s))
        {
            s = new AgentState { AgentId = id };
            _agents[id] = s;
        }
        return s;
    }

    public void Pause(string id) { lock (_lock) State(id).Paused = true; }
    public void RevokeTool(string id, string tool) { lock (_lock) State(id).RevokedTools.Add(tool); }
    public void InjectConstraint(string id, string c) { lock (_lock) State(id).Constraints.Add(c); }

    public bool IsPaused(string id) { lock (_lock) return State(id).Paused; }
    public bool IsToolRevoked(string id, string tool) { lock (_lock) return State(id).RevokedTools.Contains(tool); }

    public AgentState Snapshot(string id)
    {
        lock (_lock)
        {
            var s = State(id);
            return new AgentState
            {
                AgentId = s.AgentId, Paused = s.Paused,
                RevokedTools = new HashSet<string>(s.RevokedTools),
                Constraints = new List<string>(s.Constraints),
            };
        }
    }
}
