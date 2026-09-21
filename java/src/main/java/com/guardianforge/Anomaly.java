package com.guardianforge;

/** A flagged behavioural anomaly. */
public final class Anomaly {
    /** The agent the anomaly was seen on. */
    public String agentId = "";

    /** The kind of anomaly. */
    public String type = "";

    /** The confidence score in {@code [0, 1]}. */
    public double score;

    /** A human-readable detail. */
    public String detail = "";

    /** Create an empty anomaly. */
    public Anomaly() {
    }

    /**
     * Create a fully specified anomaly.
     *
     * @param agentId the agent identifier
     * @param type the anomaly type
     * @param score the confidence score
     * @param detail the detail message
     */
    public Anomaly(String agentId, String type, double score, String detail) {
        this.agentId = agentId;
        this.type = type;
        this.score = score;
        this.detail = detail;
    }
}
