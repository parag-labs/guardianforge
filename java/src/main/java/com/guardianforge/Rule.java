package com.guardianforge;

/** A single condition inside a policy. */
public final class Rule {
    /** The rule identifier. */
    public String ruleId = "";

    /** The event field this rule inspects. */
    public String field = "";

    /** The comparison operator. */
    public String op = "";

    /** The value to compare against. */
    public String value = "";

    /** The count threshold (for {@code count_over}). */
    public int threshold;

    /** The observation window size (for {@code count_over}). */
    public int windowSize;

    /** The severity to record when this rule matches. */
    public String severity = "";

    /** Create an empty rule. */
    public Rule() {
    }

    /**
     * Create a fully specified rule.
     *
     * @param ruleId the rule identifier
     * @param field the event field
     * @param op the operator
     * @param value the comparison value
     * @param threshold the count threshold
     * @param windowSize the window size
     * @param severity the severity to record
     */
    public Rule(String ruleId, String field, String op, String value, int threshold,
            int windowSize, String severity) {
        this.ruleId = ruleId;
        this.field = field;
        this.op = op;
        this.value = value;
        this.threshold = threshold;
        this.windowSize = windowSize;
        this.severity = severity;
    }
}
