import { describe, expect, it } from "vitest";
import { BASELINE, Scorer } from "./trust";
import {
  ANOMALY_LOOP,
  SEVERITY_CRITICAL,
  SEVERITY_HIGH,
  SEVERITY_LOW,
  SEVERITY_MEDIUM,
  anomaly,
  violation,
} from "./models";

const fixed = new Date(0);

describe("Scorer", () => {
  it("starts an unseen agent at the baseline", () => {
    const s = new Scorer();
    expect(s.get("a1")).toBe(BASELINE);
    expect(s.get("a1")).toBe(0.8);
  });

  it("returns a baseline record for an unseen agent", () => {
    const sc = new Scorer().score("a1");
    expect(sc.agentId).toBe("a1");
    expect(sc.score).toBe(BASELINE);
    expect(sc.factors).toEqual([]);
  });

  it("decays trust on a violation", () => {
    const s = new Scorer();
    const before = s.get("a1");
    const after = s.update("a1", [violation({ severity: SEVERITY_CRITICAL })], null, fixed);
    expect(after).toBeLessThan(before);
    expect(after).toBe(0.5); // 0.8 - 0.30
  });

  it("applies the exact penalty per severity", () => {
    const cases: [string, number][] = [
      [SEVERITY_CRITICAL, 0.8 - 0.3],
      [SEVERITY_HIGH, 0.8 - 0.15],
      [SEVERITY_MEDIUM, 0.8 - 0.08],
      [SEVERITY_LOW, 0.8 - 0.03],
    ];
    for (const [sev, expected] of cases) {
      const s = new Scorer();
      const after = s.update("a1", [violation({ severity: sev })], null, fixed);
      expect(Math.abs(after - expected)).toBeLessThan(1e-9);
    }
  });

  it("gives no penalty for an unknown severity", () => {
    const s = new Scorer();
    const after = s.update("a1", [violation({ severity: "bogus" })], null, fixed);
    expect(after).toBe(0.8);
  });

  it("recovers trust on clean behaviour", () => {
    const s = new Scorer();
    s.update("a1", [violation({ severity: SEVERITY_HIGH })], null, fixed);
    const low = s.get("a1");
    const after = s.update("a1", null, null, fixed);
    expect(after).toBeGreaterThan(low);
    expect(Math.abs(after - 0.66)).toBeLessThan(1e-9); // 0.65 + 0.01
  });

  it("records a clean_behaviour factor", () => {
    const s = new Scorer();
    s.update("a1", null, null, fixed);
    const sc = s.score("a1");
    expect(sc.factors).toHaveLength(1);
    expect(sc.factors[0].name).toBe("clean_behaviour");
    expect(sc.factors[0].delta).toBe(0.01);
    expect(sc.updatedAt).toEqual(fixed);
  });

  it("scales the anomaly penalty by its score", () => {
    const s = new Scorer();
    const after = s.update("a1", null, [anomaly({ type: ANOMALY_LOOP, score: 0.5 })], fixed);
    expect(Math.abs(after - 0.775)).toBeLessThan(1e-9); // 0.8 - 0.05*0.5
    expect(s.score("a1").factors[0].name).toBe("anomaly:loop");
  });

  it("isolates an agent after repeated critical violations", () => {
    const s = new Scorer();
    for (let i = 0; i < 3; i++) {
      s.update("bad", [violation({ severity: SEVERITY_CRITICAL })], null, fixed);
    }
    expect(s.shouldIsolate("bad")).toBe(true);
    expect(s.get("bad")).toBe(0.0); // 0.8 -> 0.5 -> 0.2 -> clamped 0.0
    expect(s.shouldIsolate("good")).toBe(false);
  });

  it("uses a strict < 0.3 isolation boundary", () => {
    const s = new Scorer();
    s.update("a1", [violation({ severity: SEVERITY_HIGH })], null, fixed); // 0.65
    s.update("a1", [violation({ severity: SEVERITY_HIGH })], null, fixed); // 0.50
    expect(s.shouldIsolate("a1")).toBe(false);
    s.update("a1", [violation({ severity: SEVERITY_HIGH })], null, fixed); // 0.35
    expect(s.shouldIsolate("a1")).toBe(false);
    s.update("a1", [violation({ severity: SEVERITY_HIGH })], null, fixed); // 0.20
    expect(s.shouldIsolate("a1")).toBe(true);
  });

  it("keeps trust within [0, 1]", () => {
    const s = new Scorer();
    for (let i = 0; i < 20; i++) {
      s.update("a1", [violation({ severity: SEVERITY_CRITICAL })], null, fixed);
    }
    expect(s.get("a1")).toBeGreaterThanOrEqual(0.0);
    for (let i = 0; i < 200; i++) s.update("a1", null, null, fixed);
    expect(s.get("a1")).toBeLessThanOrEqual(1.0);
  });

  it("clamps after each violation in a batch", () => {
    const s = new Scorer();
    const after = s.update(
      "a1",
      [violation({ severity: SEVERITY_CRITICAL }), violation({ severity: SEVERITY_CRITICAL })],
      null,
      fixed,
    );
    expect(Math.abs(after - 0.2)).toBeLessThan(1e-9); // 0.8 -> 0.5 -> 0.2
    expect(s.score("a1").factors).toHaveLength(2);
  });

  it("replaces the factor list on each update", () => {
    const s = new Scorer();
    s.update("a1", [violation({ severity: SEVERITY_LOW })], null, fixed);
    s.update("a1", null, null, fixed);
    const sc = s.score("a1");
    expect(sc.factors).toHaveLength(1);
    expect(sc.factors[0].name).toBe("clean_behaviour");
  });

  it("treats empty arrays as clean behaviour", () => {
    const s = new Scorer();
    const after = s.update("a1", [], [], fixed);
    expect(Math.abs(after - 0.81)).toBeLessThan(1e-9);
  });

  it("keeps agents independent", () => {
    const s = new Scorer();
    s.update("a1", [violation({ severity: SEVERITY_CRITICAL })], null, fixed);
    expect(s.get("a2")).toBe(BASELINE);
  });
});
