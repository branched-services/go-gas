package estimator

import (
	"sync"

	"github.com/branched-services/go-gas/pkg/eth"
)

// History stores recent blocks in a fixed-size ring buffer.
// Safe for concurrent access from multiple goroutines.
//
// Write frequency is ~1 per 12 seconds (new block), so RWMutex
// provides optimal read performance without lock-free complexity.
type History struct {
	mu     sync.RWMutex
	blocks []*eth.Block
	size   int
	head   int // next write position
	count  int // number of stored blocks
}

// NewHistory creates a new History with the given capacity.
func NewHistory(size int) *History {
	if size < 1 {
		size = 20
	}
	return &History{
		blocks: make([]*eth.Block, size),
		size:   size,
	}
}

// Push adds a block to the history.
// If the buffer is full, the oldest block is overwritten.
func (h *History) Push(block *eth.Block) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.blocks[h.head] = block
	h.head = (h.head + 1) % h.size
	if h.count < h.size {
		h.count++
	}
}

// Latest returns the most recently added block, or nil if empty.
func (h *History) Latest() *eth.Block {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.count == 0 {
		return nil
	}

	idx := (h.head - 1 + h.size) % h.size
	return h.blocks[idx]
}

// Snapshot returns a copy of all stored blocks, newest first.
// The returned slice is owned by the caller and safe to modify.
func (h *History) Snapshot() []*eth.Block {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*eth.Block, h.count)
	for i := 0; i < h.count; i++ {
		// Walk backwards from head
		idx := (h.head - 1 - i + h.size) % h.size
		result[i] = h.blocks[idx]
	}
	return result
}

// Len returns the number of blocks currently stored.
func (h *History) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.count
}

// Cap returns the maximum capacity of the history.
func (h *History) Cap() int {
	return h.size
}

// Clear removes all blocks from the history.
func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range h.blocks {
		h.blocks[i] = nil
	}
	h.head = 0
	h.count = 0
}
