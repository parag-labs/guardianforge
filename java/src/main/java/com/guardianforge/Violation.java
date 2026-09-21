package com.guardianforge;

/** A recorded policy breach. */
public final class Violation {
    /** The policy that was breached. */
    public String policyId = "";

    /** The rule that matched. */
    public String ruleId = "";

    /** The agent responsible. */
    public String agentId = "";

    /** The severity of the breach. */
    public String severity = "";

    /** A human-readable explanation. */
    public String reason = "";

    /** Create an empty violation. */
    public Violation() {
    }

    /**
     * Create a fully specified violation.
     *
     * @param policyId the policy identifier
     * @param ruleId the rule identifier
     * @param agentId the agent identifier
     * @param severity the severity
     * @param reason the explanation
     */
    public Violation(String policyId, String ruleId, String agentId, String severity,
            String reason) {
        this.policyId = policyId;
        this.ruleId = ruleId;
        this.agentId = agentId;
        this.severity = severity;
        this.reason = reason;
    }
}
