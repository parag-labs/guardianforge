package com.guardianforge;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Instant;
import java.util.List;
import org.junit.jupiter.api.Test;

class TrustScorerTest {
    private static final Instant FIXED = Instant.EPOCH;

    private static Violation violation(String severity) {
        Violation v = new Violation();
        v.severity = severity;
        return v;
    }

    private static Anomaly anomaly(String type, double score) {
        Anomaly a = new Anomaly();
        a.type = type;
        a.score = score;
        return a;
    }

    @Test
    void unseenAgentStartsAtBaseline() {
        TrustScorer s = new TrustScorer();
        assertEquals(TrustScorer.BASELINE, s.get("a1"), 1e-9);
        assertEquals(0.8, s.get("a1"), 1e-9);
    }

    @Test
    void scoreRecordForUnseenAgent() {
        TrustScore sc = new TrustScorer().score("a1");
        assertEquals("a1", sc.agentId);
        assertEquals(TrustScorer.BASELINE, sc.score, 1e-9);
        assertTrue(sc.factors.isEmpty());
    }

    @Test
    void violationsDecayTrust() {
        TrustScorer s = new TrustScorer();
        double before = s.get("a1");
        double after = s.update("a1", List.of(violation(Severity.CRITICAL)), null, FIXED);
        assertTrue(after < before);
        assertEquals(0.5, after, 1e-9); // 0.8 - 0.30
    }

    @Test
    void severityPenaltiesAreExact() {
        double[][] cases = {{0.30}, {0.15}, {0.08}, {0.03}};
        String[] sevs = {Severity.CRITICAL, Severity.HIGH, Severity.MEDIUM, Severity.LOW};
        for (int i = 0; i < sevs.length; i++) {
            TrustScorer s = new TrustScorer();
            double after = s.update("a1", List.of(violation(sevs[i])), null, FIXED);
            assertEquals(0.8 - cases[i][0], after, 1e-9);
        }
    }

    @Test
    void unknownSeverityCarriesNoPenalty() {
        TrustScorer s = new TrustScorer();
        assertEquals(0.8, s.update("a1", List.of(violation("bogus")), null, FIXED), 1e-9);
    }

    @Test
    void cleanBehaviourRecoversTrust() {
        TrustScorer s = new TrustScorer();
        s.update("a1", List.of(violation(Severity.HIGH)), null, FIXED);
        double low = s.get("a1");
        double after = s.update("a1", null, null, FIXED);
        assertTrue(after > low);
        assertEquals(0.66, after, 1e-9); // 0.65 + 0.01
    }

    @Test
    void cleanBehaviourRecordsFactor() {
        TrustScorer s = new TrustScorer();
        s.update("a1", null, null, FIXED);
        TrustScore sc = s.score("a1");
        assertEquals(1, sc.factors.size());
        assertEquals("clean_behaviour", sc.factors.get(0).name);
        assertEquals(0.01, sc.factors.get(0).delta, 1e-9);
        assertEquals(FIXED, sc.updatedAt);
    }

    @Test
    void anomalyPenaltyScaledByScore() {
        TrustScorer s = new TrustScorer();
        double after = s.update("a1", null, List.of(anomaly(AnomalyType.LOOP, 0.5)), FIXED);
        assertEquals(0.775, after, 1e-9); // 0.8 - 0.05*0.5
        assertEquals("anomaly:loop", s.score("a1").factors.get(0).name);
    }

    @Test
    void repeatedCriticalTriggersIsolation() {
        TrustScorer s = new TrustScorer();
        for (int i = 0; i < 3; i++) {
            s.update("bad", List.of(violation(Severity.CRITICAL)), null, FIXED);
        }
        assertTrue(s.shouldIsolate("bad"));
        assertEquals(0.0, s.get("bad"), 1e-9); // 0.8 -> 0.5 -> 0.2 -> clamped 0.0
        assertFalse(s.shouldIsolate("good"));
    }

    @Test
    void isolationThresholdBoundary() {
        TrustScorer s = new TrustScorer();
        s.update("a1", List.of(violation(Severity.HIGH)), null, FIXED); // 0.65
        s.update("a1", List.of(violation(Severity.HIGH)), null, FIXED); // 0.50
        assertFalse(s.shouldIsolate("a1"));
        s.update("a1", List.of(violation(Severity.HIGH)), null, FIXED); // 0.35
        assertFalse(s.shouldIsolate("a1"));
        s.update("a1", List.of(violation(Severity.HIGH)), null, FIXED); // 0.20
        assertTrue(s.shouldIsolate("a1"));
    }

    @Test
    void trustStaysInRange() {
        TrustScorer s = new TrustScorer();
        for (int i = 0; i < 20; i++) {
            s.update("a1", List.of(violation(Severity.CRITICAL)), null, FIXED);
        }
        assertTrue(s.get("a1") >= 0.0);
        for (int i = 0; i < 200; i++) {
            s.update("a1", null, null, FIXED);
        }
        assertTrue(s.get("a1") <= 1.0);
    }

    @Test
    void multipleViolationsClampAfterEach() {
        TrustScorer s = new TrustScorer();
        double after = s.update("a1",
                List.of(violation(Severity.CRITICAL), violation(Severity.CRITICAL)), null, FIXED);
        assertEquals(0.2, after, 1e-9); // 0.8 -> 0.5 -> 0.2
        assertEquals(2, s.score("a1").factors.size());
    }

    @Test
    void factorsReplacedEachUpdate() {
        TrustScorer s = new TrustScorer();
        s.update("a1", List.of(violation(Severity.LOW)), null, FIXED);
        s.update("a1", null, null, FIXED);
        TrustScore sc = s.score("a1");
        assertEquals(1, sc.factors.size());
        assertEquals("clean_behaviour", sc.factors.get(0).name);
    }

    @Test
    void emptyListsCountAsCleanBehaviour() {
        TrustScorer s = new TrustScorer();
        assertEquals(0.81, s.update("a1", List.of(), List.of(), FIXED), 1e-9);
    }

    @Test
    void agentsAreIndependent() {
        TrustScorer s = new TrustScorer();
        s.update("a1", List.of(violation(Severity.CRITICAL)), null, FIXED);
        assertEquals(TrustScorer.BASELINE, s.get("a2"), 1e-9);
    }
}
