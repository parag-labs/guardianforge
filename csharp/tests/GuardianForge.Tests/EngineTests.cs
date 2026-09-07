using GuardianForge.Agents;
using GuardianForge.Core;
using Xunit;

namespace GuardianForge.Tests;

public class EngineTests
{
    private static readonly Func<DateTimeOffset> Now = () => DateTimeOffset.FromUnixTimeSeconds(0);

    private static async Task<(Engine, Fleet, Intervention?)> RunScenario(string id)
    {
        var sc = Scenarios.ById(id)!;
        var fleet = new Fleet();
        var eng = new Engine(Scenarios.DefaultPolicies(), fleet, new Supervisor(new MockLlm()), Now);
        Intervention? last = null;
        foreach (var ev in sc.Events)
        {
            var o = await eng.ProcessAsync(ev);
            if (o.Intervention != null) last = o.Intervention;
        }
        return (eng, fleet, last);
    }

    [Fact]
    public async Task DangerousToolIsRevoked()
    {
        var (_, fleet, iv) = await RunScenario("dangerous_tool");
        Assert.NotNull(iv);
        Assert.Equal(InterventionType.RevokeTool, iv!.Type);
        Assert.True(fleet.IsToolRevoked("reaper", "delete_database"));
    }

    [Fact]
    public async Task PrivilegeEscalationIsEscalated()
    {
        var (eng, _, iv) = await RunScenario("privilege_escalation");
        Assert.NotNull(iv);
        Assert.Equal(InterventionType.Escalate, iv!.Type);
        Assert.NotEmpty(eng.PendingHitl());
    }

    [Fact]
    public async Task LoopIsInterrupted()
    {
        var (_, fleet, iv) = await RunScenario("infinite_loop");
        Assert.NotNull(iv);
        Assert.True(fleet.IsToolRevoked("looping-agent", "search") || fleet.IsPaused("looping-agent"));
    }

    [Fact]
    public async Task HealthyAgentIsLeftAlone()
    {
        var (eng, fleet, iv) = await RunScenario("healthy_agent");
        Assert.Null(iv);
        Assert.False(fleet.IsPaused("good-agent"));
        Assert.Equal(0, eng.Metrics.Get("interventions_total"));
    }

    [Fact]
    public async Task PiiLeakInjectsConstraint()
    {
        var (_, fleet, iv) = await RunScenario("data_leak");
        Assert.NotNull(iv);
        Assert.Equal(InterventionType.InjectConstraint, iv!.Type);
        Assert.NotEmpty(fleet.Snapshot("leaker").Constraints);
    }

    [Fact]
    public async Task AuditChainStaysIntact()
    {
        var (eng, _, _) = await RunScenario("dangerous_tool");
        Assert.True(eng.Audit.Verify());
        Assert.True(eng.Audit.Count > 0);
    }

    [Fact]
    public async Task RejectedSupervisorOutputFailsSafe()
    {
        var fleet = new Fleet();
        var bad = new Supervisor(new ScriptedLlm { Raw = "{\"intervene\":true,\"type\":\"HACK\",\"reason\":\"x\",\"confidence\":0.9}" });
        var eng = new Engine(Scenarios.DefaultPolicies(), fleet, bad, Now);
        var o = await eng.ProcessAsync(new AgentEvent { AgentId = "x", Type = EventType.ToolCall, Tool = "delete_database" });
        Assert.NotNull(o.Rejected);
        Assert.NotNull(o.Intervention);
        Assert.Equal(InterventionType.Escalate, o.Intervention!.Type);
    }

    [Fact]
    public async Task EvaluationMeetsQualityBar()
    {
        var r = await Evaluation.RunAsync();
        Assert.Equal(5, r.Scenarios);
        Assert.Equal(1.0, r.DetectionAccuracy);
        Assert.Equal(r.Scenarios, r.TypeCorrect);
        Assert.Equal(0, r.FalsePositives);
        Assert.True(r.AuditIntact);
    }
}
