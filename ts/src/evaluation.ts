/**
 * The governance scorecard (pure scoring + formatting).
 *
 * Ported from the pure parts of the Go `internal/eval` package: the `Report` and `Case`
 * aggregates, the detection-accuracy ratio, and the stable text `format`. The scenario
 * driver (`Run`) is intentionally excluded here - it wires the fleet, agents, LLM, and
 * runtime, which remain in Go.
 */

/** Round to the nearest integer, ties to even - matching Go's `%.0f` (x >= 0). */
function roundHalfEven(x: number): number {
  const floor = Math.floor(x);
  const diff = x - floor;
  if (diff < 0.5) return floor;
  if (diff > 0.5) return floor + 1;
  return floor % 2 === 0 ? floor : floor + 1;
}

function percent(value: number): string {
  return `${roundHalfEven(value)}%`;
}

/** One scenario's outcome. */
export interface Case {
  id: string;
  intervened: boolean;
  type: string;
  correct: boolean;
}

/** Build a Case with zero-value defaults. */
export function evalCase(init: Partial<Case> = {}): Case {
  return { id: "", intervened: false, type: "", correct: false, ...init };
}

/** The aggregate scorecard. */
export class Report {
  scenarios: number;
  detectCorrect: number;
  typeCorrect: number;
  falsePositives: number;
  auditIntact: boolean;
  cases: Case[];

  constructor(
    init: Partial<{
      scenarios: number;
      detectCorrect: number;
      typeCorrect: number;
      falsePositives: number;
      auditIntact: boolean;
      cases: Case[];
    }> = {},
  ) {
    this.scenarios = init.scenarios ?? 0;
    this.detectCorrect = init.detectCorrect ?? 0;
    this.typeCorrect = init.typeCorrect ?? 0;
    this.falsePositives = init.falsePositives ?? 0;
    this.auditIntact = init.auditIntact ?? true;
    this.cases = init.cases ?? [];
  }

  /** Fraction of scenarios where intervene/observe matched truth (0 if none). */
  detectionAccuracy(): number {
    if (this.scenarios === 0) return 0.0;
    return this.detectCorrect / this.scenarios;
  }

  /** Render the scorecard in a stable, readable form. */
  format(): string {
    const intact = this.auditIntact ? "yes" : "NO";
    let typeAcc = 0.0;
    if (this.scenarios > 0) typeAcc = (100.0 * this.typeCorrect) / this.scenarios;
    return (
      "GuardianForge Governance Evaluation\n\n" +
      `${"Scenarios:".padEnd(27)}${this.scenarios}\n` +
      `${"Detection accuracy:".padEnd(27)}${percent(100.0 * this.detectionAccuracy())}\n` +
      `${"Intervention correctness:".padEnd(27)}${percent(typeAcc)}\n` +
      `${"False positives:".padEnd(27)}${this.falsePositives}\n` +
      `${"Audit chain intact:".padEnd(27)}${intact}\n`
    );
  }
}
