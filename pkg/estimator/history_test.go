package estimator

import (
	"testing"

	"github.com/branched-services/go-gas/pkg/eth"
)

func TestHistory(t *testing.T) {
	h := NewHistory(3)

	// Helper
	makeBlock := func(n uint64) *eth.Block {
		return &eth.Block{Number: n}
	}

	// Push 1
	h.Push(makeBlock(1))
	if h.Len() != 1 {
		t.Errorf("Len = %d, want 1", h.Len())
	}
	if h.Latest().Number != 1 {
		t.Errorf("Latest = %d, want 1", h.Latest().Number)
	}

	// Push 2, 3
	h.Push(makeBlock(2))
	h.Push(makeBlock(3))
	if h.Len() != 3 {
		t.Errorf("Len = %d, want 3", h.Len())
	}
	if h.Latest().Number != 3 {
		t.Errorf("Latest = %d, want 3", h.Latest().Number)
	}

	// Snapshot (newest first)
	snap := h.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("Snapshot len = %d, want 3", len(snap))
	}
	if snap[0].Number != 3 {
		t.Errorf("snap[0] = %d, want 3", snap[0].Number)
	}
	if snap[2].Number != 1 {
		t.Errorf("snap[2] = %d, want 1", snap[2].Number)
	}

	// Push 4 (overwrite 1)
	h.Push(makeBlock(4))
	if h.Len() != 3 {
		t.Errorf("Len = %d, want 3", h.Len())
	}
	if h.Latest().Number != 4 {
		t.Errorf("Latest = %d, want 4", h.Latest().Number)
	}

	snap = h.Snapshot()
	if snap[0].Number != 4 {
		t.Errorf("snap[0] = %d, want 4", snap[0].Number)
	}
	if snap[1].Number != 3 {
		t.Errorf("snap[1] = %d, want 3", snap[1].Number)
	}
	if snap[2].Number != 2 {
		t.Errorf("snap[2] = %d, want 2", snap[2].Number)
	}
}
