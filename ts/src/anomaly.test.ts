import { describe, expect, it } from "vitest";
import { Detector } from "./anomaly";
import {
  ANOMALY_CYCLIC,
  ANOMALY_LOOP,
  ANOMALY_RATE_SPIKE,
  EVENT_TOOL_CALL,
  agentEvent,
  type AgentEvent,
  type Anomaly,
} from "./models";

function call(agent: string, tool: string): AgentEvent {
  return agentEvent({ agentId: agent, type: EVENT_TOOL_CALL, tool });
}

function types(anomalies: Anomaly[]): Set<string> {
  return new Set(anomalies.map((a) => a.type));
}

describe("Detector", () => {
  it("detects a loop", () => {
    const d = new Detector();
    let last: Anomaly[] = [];
    for (let i = 0; i < 6; i++) last = d.observe(call("a1", "search"));
    const loops = last.filter((a) => a.type === ANOMALY_LOOP);
    expect(loops).toHaveLength(1);
    expect(loops[0].score).toBe(0.5);
    expect(loops[0].detail).toBe("6 consecutive identical actions (TOOL_CALL:search:)");
    expect(loops[0].agentId).toBe("a1");
  });

  it("fires the loop detector at exactly five identical actions", () => {
    const d = new Detector();
    let last: Anomaly[] = [];
    for (let i = 0; i < 4; i++) last = d.observe(call("a1", "search"));
    expect(types(last).has(ANOMALY_LOOP)).toBe(false);
    last = d.observe(call("a1", "search")); // 5th identical
    expect(types(last).has(ANOMALY_LOOP)).toBe(true);
  });

  it("does not flag varied behaviour as a loop", () => {
    const d = new Detector();
    let last: Anomaly[] = [];
    for (const tool of ["a", "b", "c", "d", "e"]) last = d.observe(call("a1", tool));
    expect(types(last).has(ANOMALY_LOOP)).toBe(false);
  });

  it("detects a cyclic A-B-A-B pattern", () => {
    const d = new Detector();
    let last: Anomaly[] = [];
    for (let i = 0; i < 6; i++) last = d.observe(call("a1", i % 2 === 0 ? "x" : "y"));
    const cyclic = last.filter((a) => a.type === ANOMALY_CYCLIC);
    expect(cyclic).toHaveLength(1);
    expect(cyclic[0].score).toBe(0.7);
    expect(cyclic[0].detail).toBe('cyclic pattern between "TOOL_CALL:x:" and "TOOL_CALL:y:"');
  });

  it("detects a rate spike", () => {
    const d = new Detector();
    let last: Anomaly[] = [];
    for (let i = 0; i < 11; i++) {
      last = d.observe(call("a1", "t" + String.fromCharCode(97 + (i % 4))));
    }
    const spikes = last.filter((a) => a.type === ANOMALY_RATE_SPIKE);
    expect(spikes).toHaveLength(1);
    // round(11/12) = round(0.9166..) = 0.92
    expect(spikes[0].score).toBe(0.92);
    expect(spikes[0].detail).toBe("11 events within the observation window");
  });

  it("fires the rate detector at exactly ten events", () => {
    const d = new Detector();
    let last: Anomaly[] = [];
    for (let i = 0; i < 9; i++) {
      last = d.observe(call("a1", "t" + String.fromCharCode(97 + (i % 4))));
    }
    expect(types(last).has(ANOMALY_RATE_SPIKE)).toBe(false);
    last = d.observe(call("a1", "t" + String.fromCharCode(97 + (9 % 4)))); // 10th
    expect(types(last).has(ANOMALY_RATE_SPIKE)).toBe(true);
  });

  it("keeps per-agent history isolated", () => {
    const d = new Detector();
    for (let i = 0; i < 6; i++) d.observe(call("noisy", "search"));
    expect(d.observe(call("quiet", "read"))).toEqual([]);
  });

  it("trims history to the observation window", () => {
    const d = new Detector();
    for (let i = 0; i < 20; i++) d.observe(call("a1", "tool" + i));
    const last = d.observe(call("a1", "steady"));
    expect(types(last).has(ANOMALY_LOOP)).toBe(false);
  });

  it("returns nothing for a single first event", () => {
    const d = new Detector();
    expect(d.observe(call("a1", "only"))).toEqual([]);
  });

  it("clamps the loop score at 1.0", () => {
    const d = new Detector();
    let last: Anomaly[] = [];
    // 20 identical events; history trims to the 12-event window so count maxes at 12
    for (let i = 0; i < 20; i++) last = d.observe(call("a1", "same"));
    const loops = last.filter((a) => a.type === ANOMALY_LOOP);
    expect(loops).toHaveLength(1);
    expect(loops[0].score).toBeLessThanOrEqual(1.0);
  });

  it("does not report a cyclic pattern when both halves are identical", () => {
    const d = new Detector();
    let last: Anomaly[] = [];
    for (let i = 0; i < 6; i++) last = d.observe(call("a1", "same"));
    expect(types(last).has(ANOMALY_CYCLIC)).toBe(false);
  });
});
