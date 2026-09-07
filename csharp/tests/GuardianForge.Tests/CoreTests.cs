using GuardianForge.Core;
using Xunit;

namespace GuardianForge.Tests;

public class CoreTests
{
    private static AgentEvent Ev(string agent, string type, string tool = "", string action = "")
        => new() { AgentId = agent, Type = type, Tool = tool, Action = action };

    private static readonly DateTimeOffset Fixed = DateTimeOffset.FromUnixTimeSeconds(0);

    // --- audit ---

    [Fact]
    public void AuditChainVerifies()
    {
        var log = new AuditLog();
        for (var i = 0; i < 50; i++) log.Append("supervisor", "decision", "ok", Fixed);
        Assert.True(log.Verify());
        Assert.Equal(50, log.Count);
    }

    [Fact]
    public void EditingAnEntryBreaksTheChain()
    {
        var log = new AuditLog();
        for (var i = 0; i < 10; i++) log.Append("a", "act", "d", Fixed);
        log.Backing[4].Action = "tampered";
        Assert.False(log.Verify());
    }

    [Fact]
    public void ReorderingBreaksTheChain()
    {
        var log = new AuditLog();
        for (var i = 0; i < 10; i++) log.Append("a", "act", "d", Fixed);
        (log.Backing[3], log.Backing[6]) = (log.Backing[6], log.Backing[3]);
        Assert.False(log.Verify());
    }

    // --- policy ---

    [Fact]
    public void EqualityRuleFires()
    {
        var p = new Policy { PolicyId = "p1", Mode = PolicyMode.Hard, Rules = { new Rule { RuleId = "r1", Field = "tool", Op = Op.Equals, Value = "delete_database", Severity = Severity.Critical } } };
        var e = new PolicyEvaluator(new[] { p });
        var (v, mode, sev) = e.Evaluate(Ev("a1", EventType.ToolCall, "delete_database"));
        Assert.Single(v);
        Assert.Equal(PolicyMode.Hard, mode);
        Assert.Equal(Severity.Critical, sev);
    }

    [Fact]
    public void CleanEventNoViolation()
    {
        var p = new Policy { PolicyId = "p1", Mode = PolicyMode.Hard, Rules = { new Rule { RuleId = "r1", Field = "tool", Op = Op.Equals, Value = "delete_database", Severity = Severity.Critical } } };
        var e = new PolicyEvaluator(new[] { p });
        var (v, mode, _) = e.Evaluate(Ev("a1", EventType.ToolCall, "read_file"));
        Assert.Empty(v);
        Assert.Equal(PolicyMode.Observe, mode);
    }

    [Fact]
    public void CountOverDetectsLoop()
    {
        var p = new Policy { PolicyId = "p1", Mode = PolicyMode.Hard, Rules = { new Rule { RuleId = "loop", Field = "tool", Op = Op.CountOver, Value = "search", Threshold = 4, WindowSize = 10, Severity = Severity.Medium } } };
        var e = new PolicyEvaluator(new[] { p });
        List<Violation> last = new();
        for (var i = 0; i < 6; i++) last = e.Evaluate(Ev("a1", EventType.ToolCall, "search")).Violations;
        Assert.NotEmpty(last);
        Assert.Empty(e.Evaluate(Ev("a2", EventType.ToolCall, "search")).Violations);
    }

    // --- anomaly ---

    [Fact]
    public void DetectsLoop()
    {
        var d = new AnomalyDetector();
        List<Anomaly> last = new();
        for (var i = 0; i < 6; i++) last = d.Observe(Ev("a1", EventType.ToolCall, "search"));
        Assert.Contains(last, a => a.Type == AnomalyType.Loop);
    }

    [Fact]
    public void DetectsCyclic()
    {
        var d = new AnomalyDetector();
        List<Anomaly> last = new();
        for (var i = 0; i < 6; i++) last = d.Observe(Ev("a1", EventType.ToolCall, i % 2 == 0 ? "x" : "y"));
        Assert.Contains(last, a => a.Type == AnomalyType.Cyclic);
    }

    // --- trust ---

    [Fact]
    public void ViolationsDecayTrustRecoveryRestores()
    {
        var s = new TrustScorer();
        var after = s.Update("a1", new() { new Violation { Severity = Severity.Critical } }, new(), Fixed);
        Assert.True(after < TrustScorer.Baseline);
        var recovered = s.Update("a1", new(), new(), Fixed);
        Assert.True(recovered > after);
    }

    [Fact]
    public void RepeatedCriticalIsolates()
    {
        var s = new TrustScorer();
        for (var i = 0; i < 3; i++) s.Update("bad", new() { new Violation { Severity = Severity.Critical } }, new(), Fixed);
        Assert.True(s.ShouldIsolate("bad"));
        Assert.False(s.ShouldIsolate("good"));
    }
}
