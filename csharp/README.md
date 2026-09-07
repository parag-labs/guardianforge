# GuardianForge — C#

The complete .NET 10 implementation of GuardianForge. It is a fully independent, runnable
system that mirrors the [Go version](../go/) and the shared [design](../DESIGN.md).

## Run it

```bash
dotnet test                                              # all unit + integration tests
dotnet run --project src/GuardianForge.Demo -- --eval    # the governance scorecard
dotnet run --project src/GuardianForge.Api               # the admin API on :5000/:8080
```

Point at a live model for supervisor synthesis by implementing `ILlmClient` against an
OpenAI-compatible endpoint; the default `MockLlm` needs no API key.

## Projects

```
csharp/
├── GuardianForge.sln
├── src/
│   ├── GuardianForge.Core/          models, policy, anomaly, trust, audit, fleet, intervention
│   ├── GuardianForge.Agents/        the supervisor + model layer + the governance Engine
│   ├── GuardianForge.Api/           ASP.NET minimal-API admin surface
│   └── GuardianForge.Demo/          console: `--eval` scorecard + a scenario walkthrough
└── tests/
    └── GuardianForge.Tests/         xUnit: core, agents, engine, and API integration tests
```

(The spec's `GuardianForge.Infrastructure` — the fleet and intervention executor — is folded
into `GuardianForge.Core` to keep the project graph small.)

## API

Same surface as the Go version: `POST /events`, `POST /scenarios/{id}`, `GET /scenarios`,
`GET`/`PUT /policies`, `GET /audit`, `GET`/`POST /hitl`, `GET /metrics`, `GET /healthz`.

## Design notes

The deterministic detectors (policy, anomaly, trust) run on every event before the
supervisor is consulted; the supervisor's decision is validated against the policy mode
(`Supervisor.Validate`) before it can act, and a rejected decision fails safe to escalation.
The audit chain is SHA-256 hash-chained and verified after every scenario in the tests. See
the shared [DESIGN.md](../DESIGN.md).
