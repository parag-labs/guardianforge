package com.guardianforge;

/** One scenario's outcome in the governance scorecard. */
public final class Case {
    /** The scenario identifier. */
    public String id = "";

    /** Whether the supervisor intervened. */
    public boolean intervened;

    /** The intervention type produced, if any. */
    public String type = "";

    /** Whether the outcome matched the expected truth. */
    public boolean correct;

    /** Create an empty case. */
    public Case() {
    }

    /**
     * Create a fully specified case.
     *
     * @param id the scenario identifier
     * @param intervened whether the supervisor intervened
     * @param type the intervention type produced
     * @param correct whether the outcome matched truth
     */
    public Case(String id, boolean intervened, String type, boolean correct) {
        this.id = id;
        this.intervened = intervened;
        this.type = type;
        this.correct = correct;
    }
}
