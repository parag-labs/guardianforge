package com.guardianforge;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;
import java.util.stream.Collectors;
import org.junit.jupiter.api.Test;

class AnomalyDetectorTest {
    private static AgentEvent call(String agent, String tool) {
        AgentEvent ev = new AgentEvent();
        ev.agentId = agent;
        ev.type = EventType.TOOL_CALL;
        ev.tool = tool;
        return ev;
    }

    private static List<String> types(List<Anomaly> anomalies) {
        return anomalies.stream().map(a -> a.type).collect(Collectors.toList());
    }

    @Test
    void detectsLoop() {
        AnomalyDetector d = new AnomalyDetector();
        List<Anomaly> last = List.of();
        for (int i = 0; i < 6; i++) {
            last = d.observe(call("a1", "search"));
        }
        List<Anomaly> loops =
                last.stream().filter(a -> a.type.equals(AnomalyType.LOOP)).collect(Collectors.toList());
        assertEquals(1, loops.size());
        assertEquals(0.5, loops.get(0).score, 1e-9);
        assertEquals("6 consecutive identical actions (TOOL_CALL:search:)", loops.get(0).detail);
        assertEquals("a1", loops.get(0).agentId);
    }

    @Test
    void loopThresholdIsExactlyFive() {
        AnomalyDetector d = new AnomalyDetector();
        List<Anomaly> last = List.of();
        for (int i = 0; i < 4; i++) {
            last = d.observe(call("a1", "search"));
        }
        assertFalse(types(last).contains(AnomalyType.LOOP));
        last = d.observe(call("a1", "search")); // 5th identical
        assertTrue(types(last).contains(AnomalyType.LOOP));
    }

    @Test
    void noLoopForVariedBehaviour() {
        AnomalyDetector d = new AnomalyDetector();
        List<Anomaly> last = List.of();
        for (String tool : List.of("a", "b", "c", "d", "e")) {
            last = d.observe(call("a1", tool));
        }
        assertFalse(types(last).contains(AnomalyType.LOOP));
    }

    @Test
    void detectsCyclicPattern() {
        AnomalyDetector d = new AnomalyDetector();
        List<Anomaly> last = List.of();
        for (int i = 0; i < 6; i++) {
            last = d.observe(call("a1", i % 2 == 0 ? "x" : "y"));
        }
        List<Anomaly> cyclic =
                last.stream().filter(a -> a.type.equals(AnomalyType.CYCLIC)).collect(Collectors.toList());
        assertEquals(1, cyclic.size());
        assertEquals(0.7, cyclic.get(0).score, 1e-9);
        assertEquals("cyclic pattern between \"TOOL_CALL:x:\" and \"TOOL_CALL:y:\"",
                cyclic.get(0).detail);
    }

    @Test
    void detectsRateSpike() {
        AnomalyDetector d = new AnomalyDetector();
        List<Anomaly> last = List.of();
        for (int i = 0; i < 11; i++) {
            last = d.observe(call("a1", "t" + (char) ('a' + i % 4)));
        }
        List<Anomaly> spikes = last.stream()
                .filter(a -> a.type.equals(AnomalyType.RATE_SPIKE))
                .collect(Collectors.toList());
        assertEquals(1, spikes.size());
        // round(11/12) = round(0.9166..) = 0.92
        assertEquals(0.92, spikes.get(0).score, 1e-9);
        assertEquals("11 events within the observation window", spikes.get(0).detail);
    }

    @Test
    void rateSpikeThresholdIsExactlyTen() {
        AnomalyDetector d = new AnomalyDetector();
        List<Anomaly> last = List.of();
        for (int i = 0; i < 9; i++) {
            last = d.observe(call("a1", "t" + (char) ('a' + i % 4)));
        }
        assertFalse(types(last).contains(AnomalyType.RATE_SPIKE));
        last = d.observe(call("a1", "t" + (char) ('a' + 9 % 4))); // 10th
        assertTrue(types(last).contains(AnomalyType.RATE_SPIKE));
    }

    @Test
    void perAgentIsolation() {
        AnomalyDetector d = new AnomalyDetector();
        for (int i = 0; i < 6; i++) {
            d.observe(call("noisy", "search"));
        }
        assertTrue(d.observe(call("quiet", "read")).isEmpty());
    }

    @Test
    void windowTrimsHistory() {
        AnomalyDetector d = new AnomalyDetector();
        for (int i = 0; i < 20; i++) {
            d.observe(call("a1", "tool" + i));
        }
        assertFalse(types(d.observe(call("a1", "steady"))).contains(AnomalyType.LOOP));
    }

    @Test
    void emptyHistoryDoesNotCrash() {
        assertTrue(new AnomalyDetector().observe(call("a1", "only")).isEmpty());
    }

    @Test
    void loopScoreIsClamped() {
        AnomalyDetector d = new AnomalyDetector();
        List<Anomaly> last = List.of();
        for (int i = 0; i < 20; i++) {
            last = d.observe(call("a1", "same"));
        }
        List<Anomaly> loops =
                last.stream().filter(a -> a.type.equals(AnomalyType.LOOP)).collect(Collectors.toList());
        assertEquals(1, loops.size());
        assertTrue(loops.get(0).score <= 1.0);
    }

    @Test
    void noCyclicWhenBothHalvesIdentical() {
        AnomalyDetector d = new AnomalyDetector();
        List<Anomaly> last = List.of();
        for (int i = 0; i < 6; i++) {
            last = d.observe(call("a1", "same"));
        }
        assertFalse(types(last).contains(AnomalyType.CYCLIC));
    }
}
