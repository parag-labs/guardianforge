package com.guardianforge;

import java.util.ArrayList;
import java.util.List;

/** The aggregated deterministic findings handed to the supervisor for one event. */
public final class Signal {
    /** The event under consideration. */
    public AgentEvent event = new AgentEvent();

    /** The violations found. */
    public List<Violation> violations = new ArrayList<>();

    /** The anomalies found. */
    public List<Anomaly> anomalies = new ArrayList<>();

    /** The agent's current trust. */
    public double trust;

    /** The strictest policy mode that fired. */
    public String maxMode = "";

    /** The highest severity that fired. */
    public String maxSeverity = "";
}
