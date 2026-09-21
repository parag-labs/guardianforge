package com.guardianforge;

/** The kinds of anomalies the detector can flag. Open string constants, like Go. */
public final class AnomalyType {
    private AnomalyType() {
    }

    /** The same action repeated back to back. */
    public static final String LOOP = "loop";

    /** Too many events in the window. */
    public static final String RATE_SPIKE = "rate_spike";

    /** An A-B-A-B oscillation. */
    public static final String CYCLIC = "cyclic";
}
