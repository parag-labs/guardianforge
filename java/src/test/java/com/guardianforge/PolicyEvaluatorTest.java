package com.guardianforge;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.ArrayList;
import java.util.List;
import org.junit.jupiter.api.Test;

class PolicyEvaluatorTest {
    private static AgentEvent ev(String agent, String type, String tool, String action) {
        AgentEvent e = new AgentEvent();
        e.agentId = agent;
        e.type = type;
        e.tool = tool;
        e.action = action;
        return e;
    }

    /** Build a single-rule policy to keep the tests compact. */
    private static Policy one(String pid, String mode, Rule r) {
        return new Policy(pid, mode, new ArrayList<>(List.of(r)));
    }

    private static Rule rule(String id, String field, String op, String value, String severity) {
        Rule r = new Rule();
        r.ruleId = id;
        r.field = field;
        r.op = op;
        r.value = value;
        r.severity = severity;
        return r;
    }

    private static Rule countRule(String id, String value, int threshold, int windowSize) {
        Rule r = rule(id, "tool", Op.COUNT_OVER, value, Severity.MEDIUM);
        r.threshold = threshold;
        r.windowSize = windowSize;
        return r;
    }

    @Test
    void equalityRuleFires() {
        PolicyEvaluator e = new PolicyEvaluator(List.of(one("p1", PolicyMode.HARD,
                rule("r1", "tool", Op.EQUALS, "delete_database", Severity.CRITICAL))));
        PolicyResult r = e.evaluate(ev("a1", EventType.TOOL_CALL, "delete_database", ""));
        assertEquals(1, r.violations.size());
        assertEquals(PolicyMode.HARD, r.mode);
        assertEquals(Severity.CRITICAL, r.severity);
        assertEquals("rule r1 matched on tool", r.violations.get(0).reason);
        assertEquals("a1", r.violations.get(0).agentId);
    }

    @Test
    void noViolationForCleanEvent() {
        PolicyEvaluator e = new PolicyEvaluator(List.of(one("p1", PolicyMode.HARD,
                rule("r1", "tool", Op.EQUALS, "delete_database", Severity.CRITICAL))));
        PolicyResult r = e.evaluate(ev("a1", EventType.TOOL_CALL, "read_file", ""));
        assertTrue(r.violations.isEmpty());
        assertEquals(PolicyMode.OBSERVE, r.mode);
        assertEquals(Severity.INFO, r.severity);
    }

    @Test
    void notEqualsRule() {
        PolicyEvaluator e = new PolicyEvaluator(List.of(
                one("p1", PolicyMode.SOFT, rule("r1", "tool", Op.NOT_EQUALS, "safe", Severity.LOW))));
        assertEquals(1, e.evaluate(ev("a1", EventType.TOOL_CALL, "danger", "")).violations.size());
        assertTrue(e.evaluate(ev("a1", EventType.TOOL_CALL, "safe", "")).violations.isEmpty());
    }

    @Test
    void containsRule() {
        PolicyEvaluator e = new PolicyEvaluator(List.of(one("p1", PolicyMode.SOFT,
                rule("r1", "action", Op.CONTAINS, "drop", Severity.MEDIUM))));
        assertEquals(1, e.evaluate(ev("a1", EventType.DECISION, "", "drop table")).violations.size());
        assertTrue(e.evaluate(ev("a1", EventType.DECISION, "", "select rows")).violations.isEmpty());
    }

    @Test
    void regexpRuleWithInlineFlag() {
        PolicyEvaluator e = new PolicyEvaluator(List.of(one("p1", PolicyMode.SOFT,
                rule("r1", "action", Op.MATCHES, "(?i)privilege|escalat", Severity.HIGH))));
        PolicyResult r = e.evaluate(ev("a1", EventType.DECISION, "", "requesting privilege escalation"));
        assertEquals(1, r.violations.size());
    }

    @Test
    void inlineFlagIsCaseInsensitive() {
        PolicyEvaluator e = new PolicyEvaluator(List.of(one("p1", PolicyMode.SOFT,
                rule("r1", "action", Op.MATCHES, "(?i)privilege", Severity.HIGH))));
        assertEquals(1, e.evaluate(ev("a1", EventType.DECISION, "", "PRIVILEGE")).violations.size());
    }

    @Test
    void invalidRegexpNeverFires() {
        PolicyEvaluator e = new PolicyEvaluator(List.of(
                one("p1", PolicyMode.SOFT, rule("r1", "action", Op.MATCHES, "(unclosed", Severity.HIGH))));
        assertTrue(e.evaluate(ev("a1", EventType.DECISION, "", "(unclosed")).violations.isEmpty());
    }

    @Test
    void countOverDetectsRepetition() {
        PolicyEvaluator e =
                new PolicyEvaluator(List.of(one("p1", PolicyMode.HARD, countRule("loop", "search", 4, 10))));
        PolicyResult last = null;
        for (int i = 0; i < 6; i++) {
            last = e.evaluate(ev("a1", EventType.TOOL_CALL, "search", ""));
        }
        assertTrue(last.violations.size() > 0);
        // the count is per-agent, so a fresh agent does not fire
        assertTrue(e.evaluate(ev("a2", EventType.TOOL_CALL, "search", "")).violations.isEmpty());
    }

    @Test
    void countOverThresholdIsStrict() {
        PolicyEvaluator e =
                new PolicyEvaluator(List.of(one("p1", PolicyMode.HARD, countRule("loop", "search", 4, 10))));
        List<Integer> firesAt = new ArrayList<>();
        for (int i = 0; i < 6; i++) {
            if (!e.evaluate(ev("a1", EventType.TOOL_CALL, "search", "")).violations.isEmpty()) {
                firesAt.add(i + 1);
            }
        }
        assertEquals(5, firesAt.get(0));
    }

    @Test
    void countOverDefaultWindowUsesFullHistory() {
        PolicyEvaluator e =
                new PolicyEvaluator(List.of(one("p1", PolicyMode.HARD, countRule("loop", "search", 2, 0))));
        PolicyResult last = null;
        for (int i = 0; i < 3; i++) {
            last = e.evaluate(ev("a1", EventType.TOOL_CALL, "search", ""));
        }
        assertTrue(last.violations.size() > 0);
    }

    @Test
    void strictestModeAndHighestSeverityWin() {
        Policy observe = one("obs", PolicyMode.OBSERVE,
                rule("r1", "type", Op.EQUALS, "TOOL_CALL", Severity.LOW));
        Policy hard = one("hard", PolicyMode.HARD, rule("r2", "tool", Op.EQUALS, "rm", Severity.HIGH));
        PolicyResult r = new PolicyEvaluator(List.of(observe, hard))
                .evaluate(ev("a1", EventType.TOOL_CALL, "rm", ""));
        assertEquals(2, r.violations.size());
        assertEquals(PolicyMode.HARD, r.mode);
        assertEquals(Severity.HIGH, r.severity);
    }

    @Test
    void metadataField() {
        PolicyEvaluator e = new PolicyEvaluator(List.of(one("p1", PolicyMode.SOFT,
                rule("r1", "metadata.pii", Op.EQUALS, "true", Severity.HIGH))));
        AgentEvent event = ev("a1", EventType.TOOL_RESULT, "read", "");
        event.metadata.put("pii", "true");
        assertEquals(1, e.evaluate(event).violations.size());
    }

    @Test
    void missingMetadataFieldIsEmpty() {
        PolicyEvaluator e = new PolicyEvaluator(List.of(one("p1", PolicyMode.SOFT,
                rule("r1", "metadata.pii", Op.EQUALS, "", Severity.HIGH))));
        assertEquals(1, e.evaluate(ev("a1", EventType.TOOL_RESULT, "read", "")).violations.size());
    }

    @Test
    void unknownFieldIsEmpty() {
        PolicyEvaluator e = new PolicyEvaluator(
                List.of(one("p1", PolicyMode.SOFT, rule("r1", "nope", Op.EQUALS, "x", Severity.HIGH))));
        assertTrue(e.evaluate(ev("a1", EventType.TOOL_CALL, "t", "a")).violations.isEmpty());
    }

    @Test
    void unknownOperatorNeverFires() {
        PolicyEvaluator e = new PolicyEvaluator(
                List.of(one("p1", PolicyMode.SOFT, rule("r1", "tool", "nonsense", "rm", Severity.HIGH))));
        assertTrue(e.evaluate(ev("a1", EventType.TOOL_CALL, "rm", "")).violations.isEmpty());
    }

    @Test
    void setPoliciesReplacesAndRecompiles() {
        PolicyEvaluator e = new PolicyEvaluator(List.of());
        assertTrue(e.evaluate(ev("a1", EventType.TOOL_CALL, "rm", "")).violations.isEmpty());
        e.setPolicies(List.of(
                one("p1", PolicyMode.HARD, rule("r1", "tool", Op.EQUALS, "rm", Severity.HIGH))));
        assertEquals(1, e.evaluate(ev("a1", EventType.TOOL_CALL, "rm", "")).violations.size());
        assertEquals(1, e.policies().size());
    }
}
