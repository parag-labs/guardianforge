package com.guardianforge;

import java.util.ArrayList;
import java.util.List;

/** The outcome of evaluating one event: the violations plus the strictest mode and severity. */
public final class PolicyResult {
    /** The violations that fired. */
    public List<Violation> violations = new ArrayList<>();

    /** The strictest enforcement mode among the policies that fired. */
    public String mode = PolicyMode.OBSERVE;

    /** The highest severity among the rules that fired. */
    public String severity = Severity.INFO;

    /** Create an empty result (no violations, OBSERVE/info). */
    public PolicyResult() {
    }

    /**
     * Create a fully specified result.
     *
     * @param violations the violations
     * @param mode the strictest mode
     * @param severity the highest severity
     */
    public PolicyResult(List<Violation> violations, String mode, String severity) {
        this.violations = violations;
        this.mode = mode;
        this.severity = severity;
    }
}
