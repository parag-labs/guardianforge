// Package audit is the tamper-evident, append-only audit log. Every governance decision
// and intervention is chained by SHA-256, so any after-the-fact edit to a past entry
// breaks the chain and is caught by Verify. This is the immutable attributable trail the
// whole system is accountable to.
package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/parag-labs/guardianforge/go/internal/models"
)

// Log is a concurrency-safe hash-chained audit log.
type Log struct {
	mu      sync.Mutex
	entries []models.AuditEntry
	seq     int
}

// New builds an empty log.
func New() *Log { return &Log{} }

// hash computes the chained hash for an entry from its fields and the previous hash.
func hash(prev, ts, actor, action, details string) string {
	sum := sha256.Sum256([]byte(prev + "|" + ts + "|" + actor + "|" + action + "|" + details))
	return hex.EncodeToString(sum[:])
}

// Append records an entry, chaining it to the previous one, and returns it.
func (l *Log) Append(actor, action, details string, now time.Time) models.AuditEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	prev := ""
	if len(l.entries) > 0 {
		prev = l.entries[len(l.entries)-1].CurrentHash
	}
	l.seq++
	ts := now.UTC().Format(time.RFC3339Nano)
	e := models.AuditEntry{
		EntryID:      fmt.Sprintf("audit-%d", l.seq),
		PreviousHash: prev,
		Timestamp:    now.UTC(),
		Actor:        actor,
		Action:       action,
		Details:      details,
	}
	e.CurrentHash = hash(prev, ts, actor, action, details)
	l.entries = append(l.entries, e)
	return e
}

// Entries returns a copy of the log.
func (l *Log) Entries() []models.AuditEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]models.AuditEntry(nil), l.entries...)
}

// Len returns the number of entries.
func (l *Log) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// Verify recomputes the chain and reports whether it is intact. Any edit to a past
// entry's fields, or any reordering, makes a recomputed hash disagree and returns false.
func (l *Log) Verify() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	prev := ""
	for _, e := range l.entries {
		ts := e.Timestamp.UTC().Format(time.RFC3339Nano)
		want := hash(prev, ts, e.Actor, e.Action, e.Details)
		if want != e.CurrentHash || e.PreviousHash != prev {
			return false
		}
		prev = e.CurrentHash
	}
	return true
}

// entriesRef exposes the backing slice for white-box tamper tests in this package.
func (l *Log) entriesRef() []models.AuditEntry { return l.entries }
