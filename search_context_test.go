package main

import "testing"

func TestMaxScanLineSize(t *testing.T) {
	if maxScanLineSize != 16*1024*1024 {
		t.Errorf("maxScanLineSize = %d, want %d", maxScanLineSize, 16*1024*1024)
	}
}

func TestNewScanState_defaultContextLines(t *testing.T) {
	if defaultContextLines != 2 {
		t.Errorf("defaultContextLines = %d, want 2", defaultContextLines)
	}
}

func TestNewScanState(t *testing.T) {
	st := newScanState(3)
	if st == nil {
		t.Fatal("newScanState returned nil")
	}
	if cap(st.prev) != 3 {
		t.Errorf("prev cap = %d, want 3", cap(st.prev))
	}
}

func TestScanState_lifecycle(t *testing.T) {
	// Full cycle: record results, advance lines, fill after-context, stop when limit reached.
	st := newScanState(2)

	// Advance 2 lines before the match (builds the "before" context).
	st.advance("line1", 2)
	st.advance("line2", 2)

	// Record a match at line 5.
	st.record(SearchResult{
		FilePath:      "file.go",
		LineNum:       5,
		Content:       "matched text",
		ContextBefore: st.before(),
	}, 2)

	if len(st.results) != 1 {
		t.Fatalf("results len = %d, want 1", len(st.results))
	}
	if len(st.pending) != 1 {
		t.Fatalf("pending len = %d, want 1", len(st.pending))
	}
	if len(st.results[0].ContextAfter) != 0 {
		t.Fatalf("contextAfter len = %d, want 0", len(st.results[0].ContextAfter))
	}
	if len(st.results[0].ContextBefore) != 2 {
		t.Fatalf("contextBefore len = %d, want 2", len(st.results[0].ContextBefore))
	}

	// Fill first trailing line.
	st.fillAfter("after1")
	if len(st.pending) != 1 {
		t.Fatalf("pending after fillAfter 1 = %d, want 1", len(st.pending))
	}

	// Fill second trailing line — satisfied, pending cleared.
	st.fillAfter("after2")
	if len(st.pending) != 0 {
		t.Fatalf("pending after fillAfter 2 = %d, want 0", len(st.pending))
	}
	if len(st.results[0].ContextAfter) != 2 {
		t.Fatalf("contextAfter after 2 fills = %d, want 2", len(st.results[0].ContextAfter))
	}

	// Limit=1: done since we have 1 result and no pending.
	if !st.done(1) {
		t.Error("done(1) = false, want true")
	}
	if st.done(2) {
		t.Error("done(2) = true, want false (limit not reached)")
	}
}

func TestScanState_beforeReturnsCopy(t *testing.T) {
	st := newScanState(3)
	st.advance("a", 3)
	st.advance("b", 3)
	st.advance("c", 3)

	got := st.before()
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("before() = %v, want [a b c]", got)
	}

	// Mutating the returned slice must not affect internal state.
	got[0] = "x"
	if len(st.prev) != 3 || st.prev[0] != "a" {
		t.Errorf("prev[0] = %q, want 'a' (before() returned alias)", st.prev[0])
	}
}

func TestScanState_advanceRolling(t *testing.T) {
	st := newScanState(2)
	st.advance("line1", 2)
	if len(st.prev) != 1 || st.prev[0] != "line1" {
		t.Errorf("after line1: prev = %v, want [line1]", st.prev)
	}
	st.advance("line2", 2)
	if len(st.prev) != 2 || st.prev[1] != "line2" {
		t.Errorf("after line2: prev = %v", st.prev)
	}
	// Rolling buffer — third advance drops first.
	st.advance("line3", 2)
	if len(st.prev) != 2 {
		t.Errorf("prev len = %d, want 2 (cap)", len(st.prev))
	}
	if st.prev[0] != "line2" || st.prev[1] != "line3" {
		t.Errorf("after line3: prev = %v, want [line2 line3]", st.prev)
	}
}

func TestScanState_record_zeroContext(t *testing.T) {
	st := newScanState(0)
	st.record(SearchResult{FilePath: "f.go", LineNum: 1, Content: "match"}, 0)
	// record always creates a pending entry; call fillAfter once
	// to clear it even with 0 trailing lines needed.
	st.fillAfter("")
	if len(st.pending) != 0 {
		t.Errorf("pending = %d, want 0 after fill", len(st.pending))
	}
	if len(st.results) != 1 {
		t.Errorf("results = %d, want 1", len(st.results))
	}
	if !st.done(1) {
		t.Error("done(1) = false, want true after fill")
	}
}

func TestScanState_doneWithPending(t *testing.T) {
	st := newScanState(2)
	st.record(SearchResult{FilePath: "f.go", LineNum: 1, Content: "match"}, 2)
	// Pending results prevent done even at limit.
	if st.done(1) {
		t.Error("done(1) = true but pending matches still need trailing context")
	}
}