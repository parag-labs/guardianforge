package com.guardianforge;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/** A named collection of rules with an enforcement mode. */
public final class Policy {
    /** The policy identifier. */
    public String policyId = "";

    /** A human-readable name. */
    public String name = "";

    /** The policy version. */
    public String version = "";

    /** The scope this policy applies to. */
    public String scope = "";

    /** The rules that make up this policy. */
    public List<Rule> rules = new ArrayList<>();

    /** The enforcement mode. */
    public String mode = "";

    /** Free-form labels. */
    public Map<String, String> labels = new HashMap<>();

    /** Create an empty policy. */
    public Policy() {
    }

    /**
     * Create a policy with an id, mode, and rules.
     *
     * @param policyId the policy identifier
     * @param mode the enforcement mode
     * @param rules the rules
     */
    public Policy(String policyId, String mode, List<Rule> rules) {
        this.policyId = policyId;
        this.mode = mode;
        this.rules = rules;
    }
}
