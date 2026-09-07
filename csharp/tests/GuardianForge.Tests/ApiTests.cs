using System.Net;
using Microsoft.AspNetCore.Mvc.Testing;
using Xunit;

namespace GuardianForge.Tests;

public class ApiTests
{
    // A fresh factory per test so the singleton engine's state never leaks between tests
    // (the policies PUT test mutates it).
    private static HttpClient NewClient() => new WebApplicationFactory<Program>().CreateClient();

    [Fact]
    public async Task IngestEventReturnsOutcome()
    {
        var client = NewClient();
        var resp = await client.PostAsync("/events",
            new StringContent("{\"agent_id\":\"reaper\",\"type\":\"TOOL_CALL\",\"tool\":\"delete_database\"}"));
        resp.EnsureSuccessStatusCode();
        Assert.Contains("REVOKE_TOOL", await resp.Content.ReadAsStringAsync());
    }

    [Fact]
    public async Task RunScenarioAndAudit()
    {
        var client = NewClient();
        var resp = await client.PostAsync("/scenarios/privilege_escalation", new StringContent(""));
        resp.EnsureSuccessStatusCode();
        Assert.Contains("ESCALATE", await resp.Content.ReadAsStringAsync());

        var audit = await client.GetStringAsync("/audit");
        Assert.Contains("\"intact\":true", audit);
    }

    [Fact]
    public async Task PoliciesRoundTrip()
    {
        var client = NewClient();
        var get = await client.GetStringAsync("/policies");
        Assert.Contains("destructive-tools", get);

        var put = await client.PutAsync("/policies", new StringContent("[]"));
        put.EnsureSuccessStatusCode();
        Assert.Contains("\"count\":0", await put.Content.ReadAsStringAsync());
    }

    [Fact]
    public async Task Healthz()
    {
        var client = NewClient();
        var resp = await client.GetAsync("/healthz");
        Assert.Equal(HttpStatusCode.OK, resp.StatusCode);
    }
}
