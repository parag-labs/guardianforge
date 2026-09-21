package com.guardianforge;

/** Policy enforcement modes with a total order. Open string constants, like Go. */
public final class PolicyMode {
    private PolicyMode() {
    }

    /** Observe only; never intervene. */
    public static final String OBSERVE = "OBSERVE";

    /** Soft enforcement. */
    public static final String SOFT = "SOFT";

    /** Hard enforcement. */
    public static final String HARD = "HARD";

    /** Escalate to a human. */
    public static final String ESCALATE = "ESCALATE";

    /**
     * Rank an enforcement mode; unknown values rank lowest (0), matching Go.
     *
     * @param mode the mode string
     * @return the rank in {@code [0, 3]}
     */
    public static int rank(String mode) {
        switch (mode) {
            case OBSERVE:
                return 0;
            case SOFT:
                return 1;
            case HARD:
                return 2;
            case ESCALATE:
                return 3;
            default:
                return 0;
        }
    }

    /**
     * Return whichever mode is stricter ({@code b} wins only if strictly stricter).
     *
     * @param a the first mode
     * @param b the second mode
     * @return the stricter mode
     */
    public static String stricter(String a, String b) {
        return rank(b) > rank(a) ? b : a;
    }
}
