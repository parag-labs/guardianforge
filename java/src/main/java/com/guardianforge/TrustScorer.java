package com.guardianforge;

import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Maintains a running trust score per agent.
 *
 * <p>Mirrors the Go {@code trust} package: trust starts at a baseline, decays on violations
 * and anomalies weighted by severity, and recovers slowly on clean behaviour. Low trust is
 * itself a governance signal. The update is deterministic.
 */
public final class TrustScorer {
    /** The starting trust for a new agent. */
    public static final double BASELINE = 0.8;

    private final Map<String, TrustScore> scores = new HashMap<>();

    /** Build an empty scorer. */
    public TrustScorer() {
    }

    /**
     * Return an agent's current trust, or the baseline if unseen.
     *
     * @param agentId the agent identifier
     * @return the current score
     */
    public double get(String agentId) {
        TrustScore sc = scores.get(agentId);
        return sc != null ? sc.score : BASELINE;
    }

    /**
     * Return the full trust record for an agent (a baseline record if unseen).
     *
     * @param agentId the agent identifier
     * @return the trust record
     */
    public TrustScore score(String agentId) {
        TrustScore sc = scores.get(agentId);
        return sc != null ? sc : new TrustScore(agentId, BASELINE);
    }

    private static double severityPenalty(String sev) {
        switch (sev) {
            case Severity.CRITICAL:
                return 0.30;
            case Severity.HIGH:
                return 0.15;
            case Severity.MEDIUM:
                return 0.08;
            case Severity.LOW:
                return 0.03;
            default:
                return 0.0;
        }
    }

    /**
     * Adjust an agent's trust from one event's violations and anomalies, recovering slightly
     * on a clean event. Clamping is applied after each step, matching the reference.
     *
     * <p>A {@code null} list is treated as empty, mirroring the Python reference's
     * {@code violations or []} so a caller can pass null for "nothing happened".
     *
     * @param agentId the agent identifier
     * @param violations the violations on this event, or null for none
     * @param anomalies the anomalies on this event, or null for none
     * @param now the update timestamp
     * @return the new score
     */
    public double update(String agentId, List<Violation> violations, List<Anomaly> anomalies,
            Instant now) {
        List<Violation> vs = violations != null ? violations : List.of();
        List<Anomaly> as = anomalies != null ? anomalies : List.of();
        TrustScore existing = scores.get(agentId);
        double score = existing != null ? existing.score : BASELINE;
        List<TrustFactor> factors = new ArrayList<>();

        if (vs.isEmpty() && as.isEmpty()) {
            double delta = 0.01;
            score = clamp(score + delta);
            factors.add(new TrustFactor("clean_behaviour", delta, "no violation or anomaly"));
        } else {
            for (Violation v : vs) {
                double p = severityPenalty(v.severity);
                score = clamp(score - p);
                factors.add(new TrustFactor("violation", -p, v.reason));
            }
            for (Anomaly a : as) {
                double p = 0.05 * a.score;
                score = clamp(score - p);
                factors.add(new TrustFactor("anomaly:" + a.type, -p, a.detail));
            }
        }

        TrustScore updated = new TrustScore(agentId, score);
        updated.factors = factors;
        updated.updatedAt = now;
        scores.put(agentId, updated);
        return score;
    }

    /**
     * Report whether an agent's trust has decayed below the isolation floor of 0.3.
     *
     * @param agentId the agent identifier
     * @return whether the agent should be isolated
     */
    public boolean shouldIsolate(String agentId) {
        return get(agentId) < 0.3;
    }

    private static double clamp(double f) {
        if (f > 1) {
            return 1;
        }
        if (f < 0) {
            return 0;
        }
        return f;
    }
}
