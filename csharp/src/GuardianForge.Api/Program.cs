using GuardianForge.Agents;
using GuardianForge.Core;

// GuardianForge-C# admin API. Read-and-control only - it never bypasses the deterministic
// governance loop or the audit log. Uses the deterministic mock supervisor by default.

var builder = WebApplication.CreateBuilder(args);
var engine = new Engine(Scenarios.DefaultPolicies(), new Fleet(), new Supervisor(new MockLlm()));
builder.Services.AddSingleton(engine);

var app = builder.Build();

static async Task<T?> ReadJson<T>(HttpRequest req)
{
    using var reader = new StreamReader(req.Body);
    var body = await reader.ReadToEndAsync();
    return string.IsNullOrWhiteSpace(body) ? default : Json.Deserialize<T>(body);
}

static IResult JsonResult(object value, int status = 200)
    => Results.Text(Json.Serialize(value), "application/json", statusCode: status);

app.MapGet("/healthz", () => JsonResult(new { status = "ok" }));

app.MapGet("/metrics", (Engine e) => Results.Text(e.Metrics.Expose(), "text/plain; version=0.0.4"));

app.MapPost("/events", async (HttpRequest req, Engine e) =>
{
    var ev = await ReadJson<AgentEvent>(req);
    if (ev is null) return JsonResult(new { error = "invalid event JSON" }, 400);
    if (ev.Timestamp == default) ev.Timestamp = DateTimeOffset.UtcNow;
    return JsonResult(await e.ProcessAsync(ev));
});

app.MapGet("/scenarios", () => JsonResult(Scenarios.All().Select(s => new { id = s.Id, title = s.Title })));

app.MapPost("/scenarios/{id}", async (string id, Engine e) =>
{
    var sc = Scenarios.ById(id);
    if (sc is null) return JsonResult(new { error = "unknown scenario" }, 404);
    var outcomes = new List<Outcome>();
    foreach (var ev in sc.Events) outcomes.Add(await e.ProcessAsync(ev));
    return JsonResult(new { scenario = id, outcomes, audit_intact = e.Audit.Verify() });
});

app.MapGet("/policies", (Engine e) => JsonResult(e.Policy.Policies()));

app.MapPut("/policies", async (HttpRequest req, Engine e) =>
{
    var policies = await ReadJson<List<Policy>>(req);
    if (policies is null) return JsonResult(new { error = "invalid policies JSON" }, 400);
    e.Policy.SetPolicies(policies);
    return JsonResult(new { count = policies.Count });
});

app.MapGet("/audit", (Engine e) => JsonResult(new { entries = e.Audit.Entries(), intact = e.Audit.Verify() }));

app.MapGet("/hitl", (Engine e) => JsonResult(e.PendingHitl()));

app.MapPost("/hitl/{id}/resolve", async (string id, HttpRequest req, Engine e) =>
{
    var body = await ReadJson<Dictionary<string, string>>(req) ?? new();
    var decision = body.GetValueOrDefault("decision", "acknowledged");
    return e.ResolveHitl(id, decision)
        ? JsonResult(new { resolved = id })
        : JsonResult(new { error = "no such pending escalation" }, 404);
});

app.Run();

/// <summary>Exposed so the integration test can host the API in-process.</summary>
public partial class Program { }
