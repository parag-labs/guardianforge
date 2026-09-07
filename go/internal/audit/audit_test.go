package audit

import (
	"testing"
	"time"
)

func fixed() time.Time { return time.Unix(0, 0).UTC() }

func TestChainVerifies(t *testing.T) {
	l := New()
	for i := 0; i < 50; i++ {
		l.Append("supervisor", "decision", "ok", fixed())
	}
	if !l.Verify() {
		t.Fatal("a clean chain should verify")
	}
	if l.Len() != 50 {
		t.Fatalf("want 50 entries, got %d", l.Len())
	}
}

func TestEditingAnEntryBreaksTheChain(t *testing.T) {
	l := New()
	for i := 0; i < 10; i++ {
		l.Append("policy", "violation", "detail", fixed())
	}
	// Tamper with a past entry's action directly in the backing slice.
	l.entriesRef()[4].Action = "tampered"
	if l.Verify() {
		t.Fatal("an edited entry must break verification")
	}
}

func TestReorderingBreaksTheChain(t *testing.T) {
	l := New()
	for i := 0; i < 10; i++ {
		l.Append("a", "act", "d", fixed())
	}
	ref := l.entriesRef()
	ref[3], ref[6] = ref[6], ref[3]
	if l.Verify() {
		t.Fatal("reordering entries must break verification")
	}
}

func TestEachEntryChainsToThePrevious(t *testing.T) {
	l := New()
	a := l.Append("x", "1", "d1", fixed())
	b := l.Append("x", "2", "d2", fixed())
	if b.PreviousHash != a.CurrentHash {
		t.Fatal("entry should chain to the previous hash")
	}
	if a.PreviousHash != "" {
		t.Fatal("first entry should have an empty previous hash")
	}
}
