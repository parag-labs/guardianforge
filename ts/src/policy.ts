/**
 * The deterministic policy evaluator.
 *
 * Ported from the Go `internal/policy` package. It checks structured rules against
 * events - equality, contains, regexp, and windowed counts - and emits violations,
 * returning the strictest mode and severity among the policies that fired. This runs
 * first, before any model: the fast deterministic path decides the clear cases.
 */

import {
  MODE_OBSERVE,
  OP_CONTAINS,
  OP_COUNT_OVER,
  OP_EQUALS,
  OP_MATCHES,
  OP_NOT_EQUALS,
  SEVERITY_INFO,
  higherSeverity,
  stricterMode,
  violation,
  type AgentEvent,
  type Policy,
  type Rule,
  type Violation,
} from "./models";

const MAX_HISTORY = 256;

/** The result of evaluating one event: violations plus the strictest mode/severity. */
export interface EvaluationResult {
  violations: Violation[];
  mode: string;
  severity: string;
}

function fieldValue(ev: AgentEvent, field: string): string {
  if (field === "type") return ev.type;
  if (field === "tool") return ev.tool;
  if (field === "action") return ev.action;
  if (field.startsWith("metadata.")) return ev.metadata[field.slice("metadata.".length)] ?? "";
  return "";
}

/**
 * Compile a Python/Go regexp source for JavaScript.
 *
 * Python's `re` and Go's RE2 accept leading inline flags such as `(?i)`, `(?s)`, and
 * `(?im)`. JavaScript's `RegExp` rejects that syntax outright, so a pattern like
 * `(?i)privilege` would fail to compile and the rule would silently never fire -
 * diverging from the other ports. Strip a leading inline-flag group and map it onto
 * native RegExp flags instead.
 */
function compilePattern(source: string): RegExp {
  const m = /^\(\?([imsux]+)\)/.exec(source);
  if (m === null) return new RegExp(source);
  const flags = m[1]
    .split("")
    .filter((f) => f === "i" || f === "m" || f === "s")
    .join("");
  return new RegExp(source.slice(m[0].length), flags);
}

/** Holds the policy set and per-agent sliding history for count-based rules. */
export class Evaluator {
  private policyList: Policy[];
  private readonly history = new Map<string, string[]>();
  private regexps = new Map<string, RegExp>();

  constructor(policies: Policy[]) {
    this.policyList = [...policies];
    this.compile();
  }

  private compile(): void {
    this.regexps = new Map();
    for (const p of this.policyList) {
      for (const r of p.rules) {
        if (r.op === OP_MATCHES) {
          try {
            this.regexps.set(r.ruleId, compilePattern(r.value));
          } catch {
            // Go silently skips a rule whose pattern will not compile.
          }
        }
      }
    }
  }

  /** Return a copy of the current policy set. */
  policies(): Policy[] {
    return [...this.policyList];
  }

  /** Replace the policy set (used by the admin API). */
  setPolicies(policies: Policy[]): void {
    this.policyList = [...policies];
    this.compile();
  }

  /** Check an event against every policy; return violations, strictest mode, severity. */
  evaluate(ev: AgentEvent): EvaluationResult {
    const fp = `${ev.type}:${ev.tool}:${ev.action}`;
    let hist = [...(this.history.get(ev.agentId) ?? []), fp];
    if (hist.length > MAX_HISTORY) hist = hist.slice(hist.length - MAX_HISTORY);
    this.history.set(ev.agentId, hist);

    const violations: Violation[] = [];
    let mode = MODE_OBSERVE;
    let severity = SEVERITY_INFO;
    for (const p of this.policyList) {
      for (const r of p.rules) {
        if (this.matches(ev, r, hist)) {
          violations.push(
            violation({
              policyId: p.policyId,
              ruleId: r.ruleId,
              agentId: ev.agentId,
              severity: r.severity,
              reason: `rule ${r.ruleId} matched on ${r.field}`,
            }),
          );
          mode = stricterMode(mode, p.mode);
          severity = higherSeverity(severity, r.severity);
        }
      }
    }
    return { violations, mode, severity };
  }

  private matches(ev: AgentEvent, r: Rule, hist: string[]): boolean {
    if (r.op === OP_COUNT_OVER) {
      const window = r.windowSize > 0 ? r.windowSize : hist.length;
      const start = hist.length > window ? hist.length - window : 0;
      let count = 0;
      for (const h of hist.slice(start)) {
        if (h.includes(r.value)) count += 1;
      }
      return count > r.threshold;
    }

    const val = fieldValue(ev, r.field);
    if (r.op === OP_EQUALS) return val === r.value;
    if (r.op === OP_NOT_EQUALS) return val !== r.value;
    if (r.op === OP_CONTAINS) return val.includes(r.value);
    if (r.op === OP_MATCHES) {
      const rx = this.regexps.get(r.ruleId);
      return rx !== undefined ? rx.test(val) : false;
    }
    return false;
  }
}
