package com.guardianforge;

import java.time.Instant;

/** One hash-chained record in the tamper-evident audit log. */
public final class AuditEntry {
    /** The entry identifier. */
    public String entryId = "";

    /** The hash of the previous entry. */
    public String previousHash = "";

    /** The hash of this entry. */
    public String currentHash = "";

    /** When the entry was recorded. */
    public Instant timestamp;

    /** The actor responsible. */
    public String actor = "";

    /** The action recorded. */
    public String action = "";

    /** Free-form details. */
    public String details = "";
}
