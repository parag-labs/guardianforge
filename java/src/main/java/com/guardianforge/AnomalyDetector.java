package com.guardianforge;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * The deterministic pattern-anomaly layer.
 *
 * <p>Mirrors the Go {@code anomaly} package: it keeps a bounded per-agent history and flags
 * loops (the same action hammered over and over), rate spikes, and cyclic A-B-A-B patterns.
 * It runs on every event before any LLM is consulted.
 */
public final class AnomalyDetector {
    private final Map<String, List<String>> history = new HashMap<>();
    private final int window = 12;
    private final int loopThreshold = 5;
    private final int rateThreshold = 10;

    /** Build a detector with the reference defaults. */
    public AnomalyDetector() {
    }

    private static String fp(AgentEvent ev) {
        return ev.type + ":" + ev.tool + ":" + ev.action;
    }

    /**
     * Record an event and return any anomalies detected for its agent.
     *
     * @param ev the event to observe
     * @return the anomalies flagged (loop, then rate, then cyclic order preserved)
     */
    public List<Anomaly> observe(AgentEvent ev) {
        List<String> h = history.getOrDefault(ev.agentId, new ArrayList<>());
        h = new ArrayList<>(h);
        h.add(fp(ev));
        if (h.size() > window) {
            h = new ArrayList<>(h.subList(h.size() - window, h.size()));
        }
        history.put(ev.agentId, h);

        List<Anomaly> out = new ArrayList<>();
        Anomaly loop = detectLoop(ev.agentId, h);
        if (loop != null) {
            out.add(loop);
        }
        Anomaly rate = detectRate(ev.agentId, h);
        if (rate != null) {
            out.add(rate);
        }
        Anomaly cyclic = detectCyclic(ev.agentId, h);
        if (cyclic != null) {
            out.add(cyclic);
        }
        return out;
    }

    private Anomaly detectLoop(String agent, List<String> h) {
        if (h.isEmpty()) {
            return null;
        }
        String last = h.get(h.size() - 1);
        int count = 0;
        for (int i = h.size() - 1; i >= 0 && h.get(i).equals(last); i--) {
            count++;
        }
        if (count >= loopThreshold) {
            double score = clamp((double) count / (double) window);
            return new Anomaly(agent, AnomalyType.LOOP, round(score),
                    count + " consecutive identical actions (" + last + ")");
        }
        return null;
    }

    private Anomaly detectRate(String agent, List<String> h) {
        if (h.size() >= rateThreshold) {
            double score = clamp((double) h.size() / (double) window);
            return new Anomaly(agent, AnomalyType.RATE_SPIKE, round(score),
                    h.size() + " events within the observation window");
        }
        return null;
    }

    private Anomaly detectCyclic(String agent, List<String> h) {
        if (h.size() < 6) {
            return null;
        }
        List<String> tail = h.subList(h.size() - 6, h.size());
        String a = tail.get(0);
        String b = tail.get(1);
        if (a.equals(b)) {
            return null;
        }
        for (int i = 0; i < tail.size(); i++) {
            String want = (i % 2 == 1) ? b : a;
            if (!tail.get(i).equals(want)) {
                return null;
            }
        }
        return new Anomaly(agent, AnomalyType.CYCLIC, 0.7,
                "cyclic pattern between \"" + a + "\" and \"" + b + "\"");
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

    private static double round(double f) {
        return (double) ((int) (f * 100 + 0.5)) / 100;
    }
}
