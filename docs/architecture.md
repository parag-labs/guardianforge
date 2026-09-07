# Architecture

GuardianForge is a supervisory layer over a monitored agent fleet. Both language versions
follow the same logical architecture; this describes it in terms of the Go packages (the C#
projects mirror them).

## The pipeline

1. **Ingestion** — an `AgentEvent` arrives (via `POST /events` or a scenario runner).
2. **Policy evaluation** (`policy`) — structured rules are checked against the event,
   producing violations plus the strictest enforcement mode and severity that fired.
3. **Anomaly detection** (`anomaly`) — per-agent loop, rate-spike, and cyclic detectors
   score behavioural anomalies.
4. **Trust** (`trust`) — the agent's running trust score is updated from the violations and
   anomalies (and recovers on clean behaviour).
5. **Signal aggregation** (`runtime`) — everything above becomes a single `Signal`. A strong
   anomaly with no matching policy is raised to at least a soft posture so it is never
   silently ignored.
6. **Supervision** (`agents` + `agents/llm`) — the supervisor (the one model step) returns a
   decision, which is validated against the policy mode. A rejected decision fails safe.
7. **Intervention** (`intervention`) — an allowed decision becomes an `Intervention` applied
   to the `fleet` (pause, revoke tool, inject constraint) or routed to the HITL queue.
8. **Audit** (`audit`) — every decision and intervention is SHA-256 hash-chained.

## Event-driven core

The `runtime.Engine` is the event-driven core: `Process(event)` runs one event through the
whole pipeline synchronously and deterministically (given the mock supervisor). The admin
API and the evaluation harness both drive it the same way, so what CI verifies is exactly
what runs in the demo.

## Components

| Package (Go) | C# project | Role |
|--------------|-----------|------|
| `policy` | `GuardianForge.Core` | deterministic structured-rule evaluator |
| `anomaly` | `GuardianForge.Core` | loop / rate / cyclic detectors |
| `trust` | `GuardianForge.Core` | per-agent trust scoring |
| `audit` | `GuardianForge.Core` | hash-chained audit log |
| `agents` + `agents/llm` | `GuardianForge.Agents` | supervisor + model layer |
| `intervention`, `fleet` | `GuardianForge.Infrastructure` | executor + monitored fleet |
| `runtime` | `GuardianForge.Core` | the governance engine |
| `api` | `GuardianForge.Api` | the admin/control HTTP surface |
| `eval` | `GuardianForge.Demo` | the governance scorecard |
