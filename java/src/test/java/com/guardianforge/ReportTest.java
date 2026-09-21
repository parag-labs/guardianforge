package com.guardianforge;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;

class ReportTest {
    private static Report report(int scenarios, int detectCorrect, int typeCorrect,
            int falsePositives, boolean auditIntact) {
        Report r = new Report();
        r.scenarios = scenarios;
        r.detectCorrect = detectCorrect;
        r.typeCorrect = typeCorrect;
        r.falsePositives = falsePositives;
        r.auditIntact = auditIntact;
        return r;
    }

    @Test
    void detectionAccuracyIsZeroWithoutScenarios() {
        assertEquals(0.0, report(0, 0, 0, 0, true).detectionAccuracy(), 1e-9);
    }

    @Test
    void detectionAccuracyIsAFraction() {
        assertEquals(0.75, report(4, 3, 0, 0, true).detectionAccuracy(), 1e-9);
        assertEquals(1.0, report(5, 5, 0, 0, true).detectionAccuracy(), 1e-9);
    }

    @Test
    void reportDefaults() {
        Report r = new Report();
        assertTrue(r.auditIntact);
        assertTrue(r.cases.isEmpty());
        assertEquals(0, r.scenarios);
    }

    @Test
    void caseDefaults() {
        Case c = new Case();
        assertFalse(c.intervened);
        assertFalse(c.correct);
        assertEquals("", c.type);
    }

    @Test
    void formatsAPerfectScorecard() {
        String expected = "GuardianForge Governance Evaluation\n\n"
                + "Scenarios:                 5\n"
                + "Detection accuracy:        100%\n"
                + "Intervention correctness:  100%\n"
                + "False positives:           0\n"
                + "Audit chain intact:        yes\n";
        assertEquals(expected, report(5, 5, 5, 0, true).format());
    }

    @Test
    void formatsABrokenAuditChain() {
        String expected = "GuardianForge Governance Evaluation\n\n"
                + "Scenarios:                 4\n"
                + "Detection accuracy:        75%\n"
                + "Intervention correctness:  75%\n"
                + "False positives:           2\n"
                + "Audit chain intact:        NO\n";
        assertEquals(expected, report(4, 3, 3, 2, false).format());
    }

    @Test
    void roundsHalfToEvenDown() {
        // 1/8 = 12.5% -> round half to even -> 12 (NOT 13). This is the Go %.0f contract.
        String out = report(8, 1, 1, 0, true).format();
        assertTrue(out.contains("Detection accuracy:        12%\n"));
        assertTrue(out.contains("Intervention correctness:  12%\n"));
    }

    @Test
    void roundsHalfToEvenUp() {
        // 3/8 = 37.5% -> nearest even -> 38.
        assertTrue(report(8, 3, 3, 0, true).format().contains("Detection accuracy:        38%\n"));
    }

    @Test
    void formatsZeroScenarios() {
        String out = report(0, 0, 0, 0, true).format();
        assertTrue(out.contains("Scenarios:                 0\n"));
        assertTrue(out.contains("Detection accuracy:        0%\n"));
        assertTrue(out.contains("Intervention correctness:  0%\n"));
    }
}
