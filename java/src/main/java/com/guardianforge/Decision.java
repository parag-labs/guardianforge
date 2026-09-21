package com.guardianforge;

/** The supervisor's validated output: intervene or observe, with rationale. */
public final class Decision {
    /** Whether to intervene. */
    public boolean intervene;

    /** The intervention type, if any. */
    public String type = "";

    /** A human-readable reason. */
    public String reason = "";

    /** Whether to escalate to a human. */
    public boolean escalate;

    /** The confidence in {@code [0, 1]}. */
    public double confidence;
}
