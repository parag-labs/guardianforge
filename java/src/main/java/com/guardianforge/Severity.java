package com.guardianforge;

/** Alert/violation severities with a total order. Open string constants, like Go. */
public final class Severity {
    private Severity() {
    }

    /** Informational. */
    public static final String INFO = "info";

    /** Low. */
    public static final String LOW = "low";

    /** Medium. */
    public static final String MEDIUM = "medium";

    /** High. */
    public static final String HIGH = "high";

    /** Critical. */
    public static final String CRITICAL = "critical";

    /**
     * Rank a severity for comparison; unknown values rank lowest (0), matching Go.
     *
     * @param sev the severity string
     * @return the rank in {@code [0, 4]}
     */
    public static int rank(String sev) {
        switch (sev) {
            case INFO:
                return 0;
            case LOW:
                return 1;
            case MEDIUM:
                return 2;
            case HIGH:
                return 3;
            case CRITICAL:
                return 4;
            default:
                return 0;
        }
    }

    /**
     * Return whichever severity ranks higher ({@code b} wins only if strictly higher).
     *
     * @param a the first severity
     * @param b the second severity
     * @return the higher-ranked severity
     */
    public static String higher(String a, String b) {
        return rank(b) > rank(a) ? b : a;
    }
}
