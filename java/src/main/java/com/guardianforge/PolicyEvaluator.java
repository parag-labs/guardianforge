package com.guardianforge;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.regex.Pattern;
import java.util.regex.PatternSyntaxException;

/**
 * The deterministic policy evaluator.
 *
 * <p>Mirrors the Go {@code policy} package: it checks structured rules against events -
 * equality, inequality, contains, regexp, and windowed counts - and emits violations. It
 * runs before any LLM; only genuinely ambiguous cases are escalated to a model.
 */
public final class PolicyEvaluator {
    private List<Policy> policies;
    private final Map<String, List<String>> history = new HashMap<>();
    private final Map<String, Pattern> regexps = new HashMap<>();

    /**
     * Build an evaluator over a policy set, compiling {@code matches} rule regexes.
     *
     * @param policies the policy set
     */
    public PolicyEvaluator(List<Policy> policies) {
        this.policies = policies;
        compile();
    }

    private void compile() {
        regexps.clear();
        for (Policy p : policies) {
            for (Rule r : p.rules) {
                if (Op.MATCHES.equals(r.op)) {
                    try {
                        regexps.put(r.ruleId, Pattern.compile(r.value));
                    } catch (PatternSyntaxException e) {
                        // Skip rules whose regex fails to compile, like the Go reference.
                    }
                }
            }
        }
    }

    /**
     * Return a copy of the current policy set.
     *
     * @return the policies
     */
    public List<Policy> policies() {
        return new ArrayList<>(policies);
    }

    /**
     * Replace the policy set and recompile its regexes.
     *
     * @param p the new policy set
     */
    public void setPolicies(List<Policy> p) {
        this.policies = p;
        compile();
    }

    /**
     * Evaluate an event against every policy.
     *
     * @param ev the event to check
     * @return the violations plus the strictest mode and severity that fired
     */
    public PolicyResult evaluate(AgentEvent ev) {
        String fp = ev.type + ":" + ev.tool + ":" + ev.action;
        List<String> hist = history.getOrDefault(ev.agentId, new ArrayList<>());
        hist = new ArrayList<>(hist);
        hist.add(fp);
        if (hist.size() > 256) {
            hist = new ArrayList<>(hist.subList(hist.size() - 256, hist.size()));
        }
        history.put(ev.agentId, hist);

        List<Violation> violations = new ArrayList<>();
        String mode = PolicyMode.OBSERVE;
        String sev = Severity.INFO;
        for (Policy p : policies) {
            for (Rule r : p.rules) {
                if (matches(ev, r, hist)) {
                    violations.add(new Violation(p.policyId, r.ruleId, ev.agentId, r.severity,
                            "rule " + r.ruleId + " matched on " + r.field));
                    mode = PolicyMode.stricter(mode, p.mode);
                    sev = Severity.higher(sev, r.severity);
                }
            }
        }
        return new PolicyResult(violations, mode, sev);
    }

    private boolean matches(AgentEvent ev, Rule r, List<String> hist) {
        if (Op.COUNT_OVER.equals(r.op)) {
            int window = r.windowSize;
            if (window <= 0) {
                window = hist.size();
            }
            int start = 0;
            if (hist.size() > window) {
                start = hist.size() - window;
            }
            int count = 0;
            for (String h : hist.subList(start, hist.size())) {
                if (h.contains(r.value)) {
                    count++;
                }
            }
            return count > r.threshold;
        }

        String val = fieldValue(ev, r.field);
        switch (r.op) {
            case Op.EQUALS:
                return val.equals(r.value);
            case Op.NOT_EQUALS:
                return !val.equals(r.value);
            case Op.CONTAINS:
                return val.contains(r.value);
            case Op.MATCHES:
                Pattern rx = regexps.get(r.ruleId);
                return rx != null && rx.matcher(val).find();
            default:
                return false;
        }
    }

    private static String fieldValue(AgentEvent ev, String field) {
        if ("type".equals(field)) {
            return ev.type;
        }
        if ("tool".equals(field)) {
            return ev.tool;
        }
        if ("action".equals(field)) {
            return ev.action;
        }
        if (field.startsWith("metadata.")) {
            return ev.metadata.getOrDefault(field.substring("metadata.".length()), "");
        }
        return "";
    }
}
