package com.guardianforge;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;

/** An agent's current trust standing. */
public final class TrustScore {
    /** The agent the score belongs to. */
    public String agentId = "";

    /** The score in {@code [0, 1]}. */
    public double score;

    /** The factors that produced the most recent update. */
    public List<TrustFactor> factors = new ArrayList<>();

    /** When the score was last updated. */
    public Instant updatedAt;

    /** Create an empty trust score. */
    public TrustScore() {
    }

    /**
     * Create a trust score for an agent.
     *
     * @param agentId the agent identifier
     * @param score the initial score
     */
    public TrustScore(String agentId, double score) {
        this.agentId = agentId;
        this.score = score;
    }
}
