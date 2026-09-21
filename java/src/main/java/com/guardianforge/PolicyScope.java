package com.guardianforge;

/** The scope a policy applies to. Open string constants, like Go. */
public final class PolicyScope {
    private PolicyScope() {
    }

    /** Applies to every agent. */
    public static final String GLOBAL = "GLOBAL";

    /** Applies to a fleet. */
    public static final String FLEET = "FLEET";

    /** Applies to a role. */
    public static final String AGENT_ROLE = "AGENT_ROLE";

    /** Applies to a session. */
    public static final String SESSION = "SESSION";
}
