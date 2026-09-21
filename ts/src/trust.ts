/**
 * Per-agent trust scoring.
 *
 * Ported from the Go `internal/trust` package. Trust starts at a baseline and decays on
 * violations and anomalies (weighted by severity), recovering slowly on clean behaviour.
 * Low trust is itself a governance signal. The update is deterministic.
 */

import {
  EPOCH,
  SEVERITY_CRITICAL,
  SEVERITY_HIGH,
  SEVERITY_LOW,
  SEVERITY_MEDIUM,
  type Anomaly,
  type TrustFactor,
  type TrustScore,
  type Violation,
} from "./models";

export const BASELINE = 0.8;

function clamp(f: number): number {
  if (f > 1) return 1.0;
  if (f < 0) return 0.0;
  return f;
}

function severityPenalty(sev: string): number {
  if (sev === SEVERITY_CRITICAL) return 0.3;
  if (sev === SEVERITY_HIGH) return 0.15;
  if (sev === SEVERITY_MEDIUM) return 0.08;
  if (sev === SEVERITY_LOW) return 0.03;
  return 0.0;
}

/** Maintains per-agent trust. */
export class Scorer {
  private readonly scores = new Map<string, TrustScore>();

  /** Return an agent's current trust (`BASELINE` if unseen). */
  get(agentId: string): number {
    const sc = this.scores.get(agentId);
    return sc !== undefined ? sc.score : BASELINE;
  }

  /** Return the full trust record for an agent. */
  score(agentId: string): TrustScore {
    const sc = this.scores.get(agentId);
    if (sc !== undefined) return sc;
    return { agentId, score: BASELINE, factors: [], updatedAt: EPOCH };
  }

  /** Adjust trust from one event's violations/anomalies; clean events recover. */
  update(
    agentId: string,
    violations: Violation[] | null,
    anomalies: Anomaly[] | null,
    now: Date,
  ): number {
    const vs = violations ?? [];
    const as = anomalies ?? [];
    let sc = this.scores.get(agentId);
    if (sc === undefined) {
      sc = { agentId, score: BASELINE, factors: [], updatedAt: EPOCH };
    }

    const factors: TrustFactor[] = [];
    if (vs.length === 0 && as.length === 0) {
      const delta = 0.01;
      sc.score = clamp(sc.score + delta);
      factors.push({ name: "clean_behaviour", delta, reason: "no violation or anomaly" });
    } else {
      for (const v of vs) {
        const p = severityPenalty(v.severity);
        sc.score = clamp(sc.score - p);
        factors.push({ name: "violation", delta: -p, reason: v.reason });
      }
      for (const a of as) {
        const p = 0.05 * a.score;
        sc.score = clamp(sc.score - p);
        factors.push({ name: `anomaly:${a.type}`, delta: -p, reason: a.detail });
      }
    }

    sc.agentId = agentId;
    sc.factors = factors;
    sc.updatedAt = now;
    this.scores.set(agentId, sc);
    return sc.score;
  }

  /** Report whether an agent's trust has decayed below the isolation floor. */
  shouldIsolate(agentId: string): boolean {
    return this.get(agentId) < 0.3;
  }
}
