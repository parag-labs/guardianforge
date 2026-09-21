/**
 * Deterministic statistical/pattern anomaly detection.
 *
 * Ported from the Go `internal/anomaly` package. It watches each agent's recent
 * behaviour and flags loops (the same tool hammered over and over), rate spikes, and
 * cyclic A-B-A-B patterns. It is deterministic and cheap, and runs on every event
 * before any model is consulted.
 */

import {
  ANOMALY_CYCLIC,
  ANOMALY_LOOP,
  ANOMALY_RATE_SPIKE,
  anomaly,
  type AgentEvent,
  type Anomaly,
} from "./models";

const WINDOW = 12;
const LOOP_THRESHOLD = 5;
const RATE_THRESHOLD = 10;

function clamp(f: number): number {
  if (f > 1) return 1.0;
  if (f < 0) return 0.0;
  return f;
}

/** Reproduce Go's `float64(int(f*100+0.5)) / 100` (round half up for f >= 0). */
function roundScore(f: number): number {
  return Math.trunc(f * 100 + 0.5) / 100;
}

/** Fingerprint of an event for pattern matching. */
function fingerprint(ev: AgentEvent): string {
  return `${ev.type}:${ev.tool}:${ev.action}`;
}

/** Keeps a bounded per-agent history and scores anomalies. */
export class Detector {
  private readonly history = new Map<string, string[]>();
  private readonly window = WINDOW;
  private readonly loopThreshold = LOOP_THRESHOLD;
  private readonly rateThreshold = RATE_THRESHOLD;

  /** Record an event and return any anomalies detected for the agent. */
  observe(ev: AgentEvent): Anomaly[] {
    let h = [...(this.history.get(ev.agentId) ?? []), fingerprint(ev)];
    if (h.length > this.window) h = h.slice(h.length - this.window);
    this.history.set(ev.agentId, h);

    const out: Anomaly[] = [];
    const loop = this.detectLoop(ev.agentId, h);
    if (loop !== null) out.push(loop);
    const rate = this.detectRate(ev.agentId, h);
    if (rate !== null) out.push(rate);
    const cyclic = this.detectCyclic(ev.agentId, h);
    if (cyclic !== null) out.push(cyclic);
    return out;
  }

  private detectLoop(agent: string, h: string[]): Anomaly | null {
    if (h.length === 0) return null;
    const last = h[h.length - 1];
    let count = 0;
    let i = h.length - 1;
    while (i >= 0 && h[i] === last) {
      count += 1;
      i -= 1;
    }
    if (count >= this.loopThreshold) {
      return anomaly({
        agentId: agent,
        type: ANOMALY_LOOP,
        score: roundScore(clamp(count / this.window)),
        detail: `${count} consecutive identical actions (${last})`,
      });
    }
    return null;
  }

  private detectRate(agent: string, h: string[]): Anomaly | null {
    if (h.length >= this.rateThreshold) {
      return anomaly({
        agentId: agent,
        type: ANOMALY_RATE_SPIKE,
        score: roundScore(clamp(h.length / this.window)),
        detail: `${h.length} events within the observation window`,
      });
    }
    return null;
  }

  private detectCyclic(agent: string, h: string[]): Anomaly | null {
    if (h.length < 6) return null;
    const tail = h.slice(h.length - 6);
    const a = tail[0];
    const b = tail[1];
    if (a === b) return null;
    for (let i = 0; i < tail.length; i++) {
      const want = i % 2 === 0 ? a : b;
      if (tail[i] !== want) return null;
    }
    return anomaly({
      agentId: agent,
      type: ANOMALY_CYCLIC,
      score: 0.7,
      detail: `cyclic pattern between "${a}" and "${b}"`,
    });
  }
}
