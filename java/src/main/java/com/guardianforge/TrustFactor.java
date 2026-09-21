package com.guardianforge;

/** One contribution to a trust score. */
public final class TrustFactor {
    /** The factor name. */
    public String name = "";

    /** The signed delta applied to the score. */
    public double delta;

    /** A human-readable reason. */
    public String reason = "";

    /** Create an empty factor. */
    public TrustFactor() {
    }

    /**
     * Create a fully specified factor.
     *
     * @param name the factor name
     * @param delta the signed delta
     * @param reason the reason
     */
    public TrustFactor(String name, double delta, String reason) {
        this.name = name;
        this.delta = delta;
        this.reason = reason;
    }
}
