package hashedset

import (
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"github.com/cespare/xxhash/v2"
)

// HashedSet is a thread-safe set that uses 64-bit xxhash for efficient comparison
type HashedSet struct {
	mu     sync.RWMutex
	hashes map[uint64]string // hash -> original value for debugging
}

// New creates a new empty HashedSet
func New() *HashedSet {
	return &HashedSet{
		hashes: make(map[uint64]string),
	}
}

// NewFromSlice creates a HashedSet from a slice of strings
func NewFromSlice(values []string) *HashedSet {
	hs := New()
	for _, v := range values {
		hs.Add(v)
	}
	return hs
}

// Add adds a value to the set
func (h *HashedSet) Add(value string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	hash := xxhash.Sum64String(value)
	h.hashes[hash] = value
}

// Delete removes a value from the set
func (h *HashedSet) Delete(value string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	hash := xxhash.Sum64String(value)
	if _, exists := h.hashes[hash]; exists {
		delete(h.hashes, hash)
		return true
	}
	return false
}

// Has checks if a value exists in the set
func (h *HashedSet) Has(value string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hash := xxhash.Sum64String(value)
	_, exists := h.hashes[hash]
	return exists
}

// Size returns the number of elements in the set
func (h *HashedSet) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.hashes)
}

// Clear removes all elements from the set
func (h *HashedSet) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.hashes = make(map[uint64]string)
}

// Hash64String returns a deterministic 64-bit hash of all elements combined
// The hash is calculated by sorting all individual hashes and combining them
func (h *HashedSet) Hash64String() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.hashes) == 0 {
		return "0000000000000000"
	}

	// Collect and sort hashes for deterministic output
	sorted := make([]uint64, 0, len(h.hashes))
	for hash := range h.hashes {
		sorted = append(sorted, hash)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	// Combine all hashes using xxhash
	combined := xxhash.New()
	buf := make([]byte, 8)
	for _, hash := range sorted {
		binary.LittleEndian.PutUint64(buf, hash)
		combined.Write(buf)
	}

	return fmt.Sprintf("%016x", combined.Sum64())
}

// Values returns all values in the set (for debugging)
func (h *HashedSet) Values() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	values := make([]string, 0, len(h.hashes))
	for _, v := range h.hashes {
		values = append(values, v)
	}
	return values
}

// Clone creates a deep copy of the HashedSet
func (h *HashedSet) Clone() *HashedSet {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clone := &HashedSet{
		hashes: make(map[uint64]string, len(h.hashes)),
	}
	for k, v := range h.hashes {
		clone.hashes[k] = v
	}
	return clone
}
