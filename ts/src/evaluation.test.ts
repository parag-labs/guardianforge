import { describe, expect, it } from "vitest";
import { Report, evalCase } from "./evaluation";

describe("Report.detectionAccuracy", () => {
  it("is zero when there are no scenarios", () => {
    expect(new Report({ scenarios: 0 }).detectionAccuracy()).toBe(0.0);
  });

  it("is the correct fraction", () => {
    expect(new Report({ scenarios: 4, detectCorrect: 3 }).detectionAccuracy()).toBe(0.75);
    expect(new Report({ scenarios: 5, detectCorrect: 5 }).detectionAccuracy()).toBe(1.0);
  });
});

describe("defaults", () => {
  it("defaults a Report to an intact, empty scorecard", () => {
    const r = new Report();
    expect(r.auditIntact).toBe(true);
    expect(r.cases).toEqual([]);
    expect(r.scenarios).toBe(0);
  });

  it("defaults a Case to a non-intervened, incorrect outcome", () => {
    const c = evalCase();
    expect(c.intervened).toBe(false);
    expect(c.correct).toBe(false);
    expect(c.type).toBe("");
  });
});

describe("Report.format", () => {
  it("renders a perfect scorecard", () => {
    const r = new Report({
      scenarios: 5,
      detectCorrect: 5,
      typeCorrect: 5,
      falsePositives: 0,
      auditIntact: true,
    });
    expect(r.format()).toBe(
      "GuardianForge Governance Evaluation\n\n" +
        "Scenarios:                 5\n" +
        "Detection accuracy:        100%\n" +
        "Intervention correctness:  100%\n" +
        "False positives:           0\n" +
        "Audit chain intact:        yes\n",
    );
  });

  it("renders a broken audit chain and false positives", () => {
    const r = new Report({
      scenarios: 4,
      detectCorrect: 3,
      typeCorrect: 3,
      falsePositives: 2,
      auditIntact: false,
    });
    expect(r.format()).toBe(
      "GuardianForge Governance Evaluation\n\n" +
        "Scenarios:                 4\n" +
        "Detection accuracy:        75%\n" +
        "Intervention correctness:  75%\n" +
        "False positives:           2\n" +
        "Audit chain intact:        NO\n",
    );
  });

  it("rounds a .5 percentage to even, down", () => {
    // 1/8 = 12.5% -> round half to even -> 12 (NOT 13). This is the Go %.0f contract.
    const out = new Report({ scenarios: 8, detectCorrect: 1, typeCorrect: 1 }).format();
    expect(out).toContain("Detection accuracy:        12%\n");
    expect(out).toContain("Intervention correctness:  12%\n");
  });

  it("rounds a .5 percentage to even, up", () => {
    // 3/8 = 37.5% -> nearest even -> 38.
    const out = new Report({ scenarios: 8, detectCorrect: 3, typeCorrect: 3 }).format();
    expect(out).toContain("Detection accuracy:        38%\n");
  });

  it("renders zero scenarios without dividing by zero", () => {
    const out = new Report({ scenarios: 0 }).format();
    expect(out).toContain("Scenarios:                 0\n");
    expect(out).toContain("Detection accuracy:        0%\n");
    expect(out).toContain("Intervention correctness:  0%\n");
  });
});
