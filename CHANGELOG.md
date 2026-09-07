# Changelog

All notable changes to GuardianForge are documented here.

## [0.1.0] - 2026-09-07

First working version — the governance MVP end to end, implemented in Go, fully tested with
the deterministic mock supervisor. The C# implementation mirrors the same design.

### Added (Go)
- **Deterministic policy evaluator** — structured rules (equality, contains, regexp,
  windowed counts) with four enforcement modes (observe / soft / hard / escalate).
- **Anomaly workers** — per-agent loop, rate-spike, and cyclic-pattern detection, scored.
- **Trust scorer** — running per-agent trust that decays on violations/anomalies, recovers
  on clean behaviour, and drives isolation at the floor.
- **Hash-chained audit log** — SHA-256 chained, tamper-evident, with `Verify`.
- **Supervisor + validation** — the one model step, with a validator that rejects unknown
  intervention types and any intervention under an observe-only policy, failing safe.
- **Provider-agnostic model layer** — deterministic mock, scriptable stub, OpenAI-compatible.
- **Intervention executor + synthetic fleet** — pause, revoke tool, inject constraint, plus
  a human-in-the-loop escalation queue.
- **Governance engine** wiring the event-driven core, and a **net/http admin API** (events,
  scenarios, policies, audit, HITL, metrics, health).
- **Evaluation harness** (`-eval`) over 5 failure scenarios; wired into CI. Current
  scorecard: 100% detection, 100% intervention correctness, 0 false positives, audit intact.
- Docs, `DESIGN.md`, Dockerfile, docker-compose, and a GitHub Actions CI pipeline (fmt, vet,
  `-race` tests, build, evaluate).
