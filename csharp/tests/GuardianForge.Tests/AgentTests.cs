using GuardianForge.Agents;
using GuardianForge.Core;
using Xunit;

namespace GuardianForge.Tests;

public class AgentTests
{
    private static readonly Func<DateTimeOffset> Now = () => DateTimeOffset.FromUnixTimeSeconds(0);

    private static Signal Sig(string mode, double trust) => new()
    {
        Event = new AgentEvent { AgentId = "a1", Type = EventType.ToolCall, Tool = "rm" },
        MaxMode = mode, Trust = trust,
        Violations = { new Violation { Severity = Severity.High } },
    };

    [Fact]
    public async Task HardModeRevokesTool()
    {
        var sup = new Supervisor(new MockLlm());
        var (d, _) = await sup.DecideAsync(Sig(PolicyMode.Hard, 0.8));
        Assert.True(d.Intervene);
        Assert.Equal(InterventionType.RevokeTool, d.Type);
    }

    [Fact]
    public async Task ObserveDoesNotIntervene()
    {
        var sup = new Supervisor(new MockLlm());
        var (d, _) = await sup.DecideAsync(Sig(PolicyMode.Observe, 0.8));
        Assert.False(d.Intervene);
    }

    [Fact]
    public async Task LowTrustEscalates()
    {
        var sup = new Supervisor(new MockLlm());
        var (d, _) = await sup.DecideAsync(Sig(PolicyMode.Soft, 0.1));
        Assert.True(d is { Intervene: true, Escalate: true });
    }

    [Fact]
    public void ValidateRejectsMalformed()
    {
        var r = Supervisor.Validate("{not json", Sig(PolicyMode.Hard, 0.8));
        Assert.Equal(RejectKind.Malformed, r.Reject);
    }

    [Fact]
    public void ValidateRejectsUnknownType()
    {
        var r = Supervisor.Validate("{\"intervene\":true,\"type\":\"NUKE\",\"reason\":\"x\",\"confidence\":0.9}", Sig(PolicyMode.Hard, 0.8));
        Assert.Equal(RejectKind.UnknownType, r.Reject);
    }

    [Fact]
    public void ValidateRejectsInterveneUnderObserve()
    {
        var r = Supervisor.Validate("{\"intervene\":true,\"type\":\"PAUSE\",\"reason\":\"x\",\"confidence\":0.9}", Sig(PolicyMode.Observe, 0.8));
        Assert.Equal(RejectKind.ObserveViolated, r.Reject);
    }

    [Fact]
    public async Task ScriptedHallucinationFailsSafeToEscalate()
    {
        var bad = new ScriptedLlm { Raw = "{\"intervene\":true,\"type\":\"HACK\",\"reason\":\"trust me\",\"confidence\":0.99}" };
        var sup = new Supervisor(bad);
        var (d, rejected) = await sup.DecideAsync(Sig(PolicyMode.Hard, 0.8));
        Assert.NotNull(rejected);
        Assert.True(d.Intervene);
        Assert.Equal(InterventionType.Escalate, d.Type);
    }
}
