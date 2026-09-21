package com.guardianforge;

import java.time.Instant;
import java.util.HashMap;
import java.util.Map;

/** A corrective action targeted at an agent or session. */
public final class Intervention {
    /** The intervention identifier. */
    public String interventionId = "";

    /** The agent it targets. */
    public String targetAgentId = "";

    /** The session it targets. */
    public String targetSessionId = "";

    /** The kind of intervention. */
    public String type = "";

    /** A human-readable reason. */
    public String reason = "";

    /** Free-form parameters. */
    public Map<String, String> parameters = new HashMap<>();

    /** Who issued it. */
    public String issuedBy = "";

    /** When it was issued. */
    public Instant issuedAt;
}
