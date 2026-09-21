import { describe, expect, it } from "vitest";
import { Evaluator } from "./policy";
import {
  EVENT_DECISION,
  EVENT_TOOL_CALL,
  EVENT_TOOL_RESULT,
  MODE_HARD,
  MODE_OBSERVE,
  MODE_SOFT,
  OP_CONTAINS,
  OP_COUNT_OVER,
  OP_EQUALS,
  OP_MATCHES,
  OP_NOT_EQUALS,
  SEVERITY_CRITICAL,
  SEVERITY_HIGH,
  SEVERITY_INFO,
  SEVERITY_LOW,
  SEVERITY_MEDIUM,
  agentEvent,
  policy,
  rule,
  type AgentEvent,
  type Policy,
  type Rule,
} from "./models";

function ev(agent: string, type: string, tool: string, action: string): AgentEvent {
  return agentEvent({ agentId: agent, type, tool, action });
}

/** Build a single-rule policy to keep the tests compact. */
function one(pid: string, mode: string, ruleInit: Partial<Rule>): Policy {
  return policy({ policyId: pid, mode, rules: [rule(ruleInit)] });
}

describe("Evaluator", () => {
  it("fires an equality rule", () => {
    const e = new Evaluator([
      one("p1", MODE_HARD, {
        ruleId: "r1",
        field: "tool",
        op: OP_EQUALS,
        value: "delete_database",
        severity: SEVERITY_CRITICAL,
      }),
    ]);
    const r = e.evaluate(ev("a1", EVENT_TOOL_CALL, "delete_database", ""));
    expect(r.violations).toHaveLength(1);
    expect(r.mode).toBe(MODE_HARD);
    expect(r.severity).toBe(SEVERITY_CRITICAL);
    expect(r.violations[0].reason).toBe("rule r1 matched on tool");
    expect(r.violations[0].agentId).toBe("a1");
  });

  it("reports nothing for a clean event", () => {
    const e = new Evaluator([
      one("p1", MODE_HARD, {
        ruleId: "r1",
        field: "tool",
        op: OP_EQUALS,
        value: "delete_database",
        severity: SEVERITY_CRITICAL,
      }),
    ]);
    const r = e.evaluate(ev("a1", EVENT_TOOL_CALL, "read_file", ""));
    expect(r.violations).toEqual([]);
    expect(r.mode).toBe(MODE_OBSERVE);
    expect(r.severity).toBe(SEVERITY_INFO);
  });

  it("handles a not-equals rule", () => {
    const e = new Evaluator([
      one("p1", MODE_SOFT, {
        ruleId: "r1",
        field: "tool",
        op: OP_NOT_EQUALS,
        value: "safe",
        severity: SEVERITY_LOW,
      }),
    ]);
    expect(e.evaluate(ev("a1", EVENT_TOOL_CALL, "danger", "")).violations).toHaveLength(1);
    expect(e.evaluate(ev("a1", EVENT_TOOL_CALL, "safe", "")).violations).toEqual([]);
  });

  it("handles a contains rule", () => {
    const e = new Evaluator([
      one("p1", MODE_SOFT, {
        ruleId: "r1",
        field: "action",
        op: OP_CONTAINS,
        value: "drop",
        severity: SEVERITY_MEDIUM,
      }),
    ]);
    expect(e.evaluate(ev("a1", EVENT_DECISION, "", "drop table")).violations).toHaveLength(1);
    expect(e.evaluate(ev("a1", EVENT_DECISION, "", "select rows")).violations).toEqual([]);
  });

  it("handles a regexp rule with a leading (?i) inline flag", () => {
    // Python/Go accept (?i) inline; JavaScript does not, so the port maps it to a flag.
    const e = new Evaluator([
      one("p1", MODE_SOFT, {
        ruleId: "r1",
        field: "action",
        op: OP_MATCHES,
        value: "(?i)privilege|escalat",
        severity: SEVERITY_HIGH,
      }),
    ]);
    const r = e.evaluate(ev("a1", EVENT_DECISION, "", "requesting privilege escalation"));
    expect(r.violations).toHaveLength(1);
  });

  it("applies (?i) case-insensitively", () => {
    const e = new Evaluator([
      one("p1", MODE_SOFT, {
        ruleId: "r1",
        field: "action",
        op: OP_MATCHES,
        value: "(?i)privilege",
        severity: SEVERITY_HIGH,
      }),
    ]);
    expect(e.evaluate(ev("a1", EVENT_DECISION, "", "PRIVILEGE")).violations).toHaveLength(1);
  });

  it("never fires a rule whose pattern does not compile", () => {
    const e = new Evaluator([
      one("p1", MODE_SOFT, {
        ruleId: "r1",
        field: "action",
        op: OP_MATCHES,
        value: "(unclosed",
        severity: SEVERITY_HIGH,
      }),
    ]);
    expect(e.evaluate(ev("a1", EVENT_DECISION, "", "(unclosed")).violations).toEqual([]);
  });

  it("detects repetition with count_over", () => {
    const e = new Evaluator([
      one("p1", MODE_HARD, {
        ruleId: "loop",
        field: "tool",
        op: OP_COUNT_OVER,
        value: "search",
        threshold: 4,
        windowSize: 10,
        severity: SEVERITY_MEDIUM,
      }),
    ]);
    let last = e.evaluate(ev("a1", EVENT_TOOL_CALL, "search", ""));
    for (let i = 1; i < 6; i++) last = e.evaluate(ev("a1", EVENT_TOOL_CALL, "search", ""));
    expect(last.violations.length).toBeGreaterThan(0);
    // count is per-agent, so a fresh agent does not fire
    expect(e.evaluate(ev("a2", EVENT_TOOL_CALL, "search", "")).violations).toEqual([]);
  });

  it("treats the count_over threshold as strictly greater", () => {
    const e = new Evaluator([
      one("p1", MODE_HARD, {
        ruleId: "loop",
        field: "tool",
        op: OP_COUNT_OVER,
        value: "search",
        threshold: 4,
        windowSize: 10,
        severity: SEVERITY_MEDIUM,
      }),
    ]);
    const firesAt: number[] = [];
    for (let i = 0; i < 6; i++) {
      const r = e.evaluate(ev("a1", EVENT_TOOL_CALL, "search", ""));
      if (r.violations.length > 0) firesAt.push(i + 1);
    }
    expect(firesAt[0]).toBe(5);
  });

  it("uses the full history when window_size is 0", () => {
    const e = new Evaluator([
      one("p1", MODE_HARD, {
        ruleId: "loop",
        field: "tool",
        op: OP_COUNT_OVER,
        value: "search",
        threshold: 2,
        windowSize: 0,
        severity: SEVERITY_MEDIUM,
      }),
    ]);
    let last = e.evaluate(ev("a1", EVENT_TOOL_CALL, "search", ""));
    for (let i = 1; i < 3; i++) last = e.evaluate(ev("a1", EVENT_TOOL_CALL, "search", ""));
    expect(last.violations.length).toBeGreaterThan(0);
  });

  it("returns the strictest mode and highest severity", () => {
    const observe = one("obs", MODE_OBSERVE, {
      ruleId: "r1",
      field: "type",
      op: OP_EQUALS,
      value: "TOOL_CALL",
      severity: SEVERITY_LOW,
    });
    const hard = one("hard", MODE_HARD, {
      ruleId: "r2",
      field: "tool",
      op: OP_EQUALS,
      value: "rm",
      severity: SEVERITY_HIGH,
    });
    const r = new Evaluator([observe, hard]).evaluate(ev("a1", EVENT_TOOL_CALL, "rm", ""));
    expect(r.violations).toHaveLength(2);
    expect(r.mode).toBe(MODE_HARD);
    expect(r.severity).toBe(SEVERITY_HIGH);
  });

  it("reads a metadata field", () => {
    const e = new Evaluator([
      one("p1", MODE_SOFT, {
        ruleId: "r1",
        field: "metadata.pii",
        op: OP_EQUALS,
        value: "true",
        severity: SEVERITY_HIGH,
      }),
    ]);
    const event = ev("a1", EVENT_TOOL_RESULT, "read", "");
    event.metadata = { pii: "true" };
    expect(e.evaluate(event).violations).toHaveLength(1);
  });

  it("reads an absent metadata field as the empty string", () => {
    const e = new Evaluator([
      one("p1", MODE_SOFT, {
        ruleId: "r1",
        field: "metadata.pii",
        op: OP_EQUALS,
        value: "",
        severity: SEVERITY_HIGH,
      }),
    ]);
    expect(e.evaluate(ev("a1", EVENT_TOOL_RESULT, "read", "")).violations).toHaveLength(1);
  });

  it("reads an unknown field as the empty string", () => {
    const e = new Evaluator([
      one("p1", MODE_SOFT, {
        ruleId: "r1",
        field: "nope",
        op: OP_EQUALS,
        value: "x",
        severity: SEVERITY_HIGH,
      }),
    ]);
    expect(e.evaluate(ev("a1", EVENT_TOOL_CALL, "t", "a")).violations).toEqual([]);
  });

  it("replaces and recompiles on setPolicies", () => {
    const e = new Evaluator([]);
    expect(e.evaluate(ev("a1", EVENT_TOOL_CALL, "rm", "")).violations).toEqual([]);
    e.setPolicies([
      one("p1", MODE_HARD, {
        ruleId: "r1",
        field: "tool",
        op: OP_EQUALS,
        value: "rm",
        severity: SEVERITY_HIGH,
      }),
    ]);
    expect(e.evaluate(ev("a1", EVENT_TOOL_CALL, "rm", "")).violations).toHaveLength(1);
    expect(e.policies()).toHaveLength(1);
  });

  it("returns a copy from policies() so callers cannot mutate the set", () => {
    const e = new Evaluator([
      one("p1", MODE_HARD, { ruleId: "r1", field: "tool", op: OP_EQUALS, value: "rm" }),
    ]);
    e.policies().pop();
    expect(e.policies()).toHaveLength(1);
  });

  it("ignores an unknown operator", () => {
    const e = new Evaluator([
      one("p1", MODE_SOFT, { ruleId: "r1", field: "tool", op: "nonsense", value: "rm" }),
    ]);
    expect(e.evaluate(ev("a1", EVENT_TOOL_CALL, "rm", "")).violations).toEqual([]);
  });
});
