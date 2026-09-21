package com.guardianforge;

/** The kinds of interventions the supervisor can issue. Open string constants, like Go. */
public final class InterventionType {
    private InterventionType() {
    }

    /** Pause the agent. */
    public static final String PAUSE = "PAUSE";

    /** Revoke a tool. */
    public static final String REVOKE_TOOL = "REVOKE_TOOL";

    /** Inject a constraint. */
    public static final String INJECT_CONSTRAINT = "INJECT_CONSTRAINT";

    /** Force a replan. */
    public static final String FORCE_REPLAN = "FORCE_REPLAN";

    /** Notify a human. */
    public static final String NOTIFY = "NOTIFY";

    /** Escalate to a human. */
    public static final String ESCALATE = "ESCALATE";
}
