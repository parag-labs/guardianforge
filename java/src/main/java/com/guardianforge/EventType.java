package com.guardianforge;

/** The kinds of events an agent can emit. Open string constants, like the Go reference. */
public final class EventType {
    private EventType() {
    }

    /** An agent invoked a tool. */
    public static final String TOOL_CALL = "TOOL_CALL";

    /** A tool returned a result. */
    public static final String TOOL_RESULT = "TOOL_RESULT";

    /** A free-form message. */
    public static final String MESSAGE = "MESSAGE";

    /** A decision was taken. */
    public static final String DECISION = "DECISION";

    /** An error was raised. */
    public static final String ERROR = "ERROR";

    /** A periodic liveness signal. */
    public static final String HEARTBEAT = "HEARTBEAT";
}
