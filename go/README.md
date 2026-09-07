# GuardianForge — Go

The complete Go implementation of GuardianForge. It is a fully independent, runnable system;
see the [repository README](../README.md) for the project overview and the governance loop.

## Run it

```bash
go test ./...                       # unit + integration + AI tests (no API key)
go run ./cmd/guardianforge -eval    # the governance scorecard (fails on regression)
go run ./cmd/guardianforge          # start the admin API on :8080
```

Point at a live model for supervisor synthesis (optional — the default is a deterministic
mock, no key required):

```bash
LLM_BASE_URL=https://api.openai.com/v1 LLM_API_KEY=sk-... LLM_MODEL=gpt-4o-mini go run ./cmd/guardianforge
```

## API

| Method & path | Purpose |
|---------------|---------|
| `POST /events` | Ingest one agent event; returns the governance outcome |
| `POST /scenarios/{id}` | Run a synthetic failure scenario end to end |
| `GET /scenarios` | List the built-in scenarios |
| `GET /policies` · `PUT /policies` | Inspect / replace the policy set |
| `GET /audit` | The hash-chained audit trail + intact status |
| `GET /hitl` · `POST /hitl/{id}/resolve` | The human-in-the-loop escalation queue |
| `GET /metrics` | Prometheus-text metrics |
| `GET /healthz` | Health check |

## Layout

```
go/
├── cmd/guardianforge/   the admin API server (and `-eval` scorecard)
├── internal/
│   ├── models/          typed governance contracts
│   ├── policy/          deterministic structured-rule evaluator
│   ├── anomaly/         loop / rate-spike / cyclic detectors
│   ├── trust/           per-agent trust scoring
│   ├── audit/           SHA-256 hash-chained audit log
│   ├── agents/          the supervisor + decision validation
│   │   └── llm/         provider-agnostic model: mock, scripted, OpenAI-compatible
│   ├── intervention/    the intervention executor
│   ├── fleet/           the synthetic monitored fleet + failure scenarios
│   ├── runtime/         the governance engine that wires it together
│   ├── api/             net/http admin API
│   ├── eval/            the governance scorecard
│   └── obs/             Prometheus-text metrics
└── Dockerfile
```

## Design notes

The deterministic detectors (policy, anomaly, trust) run on every event before the
supervisor is consulted; the supervisor's decision is validated against the policy mode
before it can act, and a rejected decision fails safe to escalation. The audit chain is
verified after every scenario in the tests. See the shared [DESIGN.md](../DESIGN.md).
