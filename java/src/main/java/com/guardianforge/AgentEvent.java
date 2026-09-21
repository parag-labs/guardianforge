package com.guardianforge;

import java.time.Instant;
import java.util.HashMap;
import java.util.Map;

/** One observation emitted by a monitored agent. */
public final class AgentEvent {
    /** The unique event identifier. */
    public String eventId = "";

    /** The agent that emitted it. */
    public String agentId = "";

    /** The session it belongs to. */
    public String sessionId = "";

    /** The fleet it belongs to. */
    public String fleetId = "";

    /** When it occurred (never inspected by the core). */
    public Instant timestamp;

    /** The event type. */
    public String type = "";

    /** The tool involved, if any. */
    public String tool = "";

    /** The action taken, if any. */
    public String action = "";

    /** Free-form metadata. */
    public Map<String, String> metadata = new HashMap<>();
}
