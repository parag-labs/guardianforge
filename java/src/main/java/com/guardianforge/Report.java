package com.guardianforge;

import java.util.ArrayList;
import java.util.List;

/**
 * The governance scorecard (pure scoring and formatting).
 *
 * <p>Mirrors the pure parts of the Go {@code eval} package: the {@code Report} and
 * {@code Case} aggregates, the detection-accuracy ratio, and the stable text {@code format}.
 * The scenario driver ({@code Run}) is intentionally excluded - it wires the fleet, agents,
 * LLM, and runtime, which remain in Go.
 *
 * <p>Percentages round half-to-even to match Go's {@code %.0f} (strconv), which differs from
 * the existing C# mirror's away-from-zero rounding on exact-half values.
 */
public final class Report {
    /** The number of scenarios scored. */
    public int scenarios;

    /** The count of scenarios where intervene/observe matched truth. */
    public int detectCorrect;

    /** The count of scenarios with the correct intervention type. */
    public int typeCorrect;

    /** The count of false-positive interventions. */
    public int falsePositives;

    /** Whether the audit chain stayed intact (true by default, like the reference). */
    public boolean auditIntact = true;

    /** The per-scenario outcomes. */
    public List<Case> cases = new ArrayList<>();

    /** Create an empty report with an intact audit chain. */
    public Report() {
    }

    /**
     * Return the fraction of scenarios where intervene/observe matched truth.
     *
     * @return the detection accuracy in {@code [0, 1]} (0 when there are no scenarios)
     */
    public double detectionAccuracy() {
        if (scenarios == 0) {
            return 0.0;
        }
        return (double) detectCorrect / (double) scenarios;
    }

    /**
     * Render the scorecard in a stable, readable form.
     *
     * @return the formatted scorecard
     */
    public String format() {
        String intact = auditIntact ? "yes" : "NO";
        double typeAcc = 0.0;
        if (scenarios > 0) {
            typeAcc = 100.0 * (double) typeCorrect / (double) scenarios;
        }
        StringBuilder sb = new StringBuilder();
        sb.append("GuardianForge Governance Evaluation\n\n");
        sb.append(lj("Scenarios:")).append(scenarios).append("\n");
        sb.append(lj("Detection accuracy:")).append(percent(100.0 * detectionAccuracy()))
                .append("\n");
        sb.append(lj("Intervention correctness:")).append(percent(typeAcc)).append("\n");
        sb.append(lj("False positives:")).append(falsePositives).append("\n");
        sb.append(lj("Audit chain intact:")).append(intact).append("\n");
        return sb.toString();
    }

    private static String lj(String s) {
        StringBuilder b = new StringBuilder(s);
        while (b.length() < 27) {
            b.append(' ');
        }
        return b.toString();
    }

    private static String percent(double value) {
        return roundHalfEven(value) + "%";
    }

    /**
     * Round to the nearest integer with ties going to even, matching Go's {@code %.0f} for
     * non-negative inputs.
     *
     * @param x the value to round (assumed non-negative)
     * @return the rounded integer
     */
    static long roundHalfEven(double x) {
        long floor = (long) Math.floor(x);
        double diff = x - floor;
        if (diff < 0.5) {
            return floor;
        }
        if (diff > 0.5) {
            return floor + 1;
        }
        return (floor % 2 == 0) ? floor : floor + 1;
    }
}
