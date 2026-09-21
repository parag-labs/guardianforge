package com.guardianforge;

/** Rule comparison operators. Open string constants, like Go. */
public final class Op {
    private Op() {
    }

    /** Equality comparison. */
    public static final String EQUALS = "eq";

    /** Inequality comparison. */
    public static final String NOT_EQUALS = "ne";

    /** Substring containment. */
    public static final String CONTAINS = "contains";

    /** Regular-expression match. */
    public static final String MATCHES = "matches";

    /** Greater-than comparison (declared in the reference but not wired into matching). */
    public static final String GREATER = "gt";

    /** Windowed occurrence count. */
    public static final String COUNT_OVER = "count_over";
}
