package pool

import (
	"strings"
	"sync"
)

// StringBuilderPool provides a pool of strings.Builder objects to reduce allocations
var StringBuilderPool = sync.Pool{
	New: func() any {
		return &strings.Builder{}
	},
}

// GetStringBuilder gets a strings.Builder from the pool
func GetStringBuilder() *strings.Builder {
	sb := StringBuilderPool.Get().(*strings.Builder)
	sb.Reset() // Ensure it's clean
	return sb
}

// PutStringBuilder returns a strings.Builder to the pool
func PutStringBuilder(sb *strings.Builder) {
	// Only pool builders that aren't too large to avoid memory bloat
	if sb.Cap() < 64*1024 { // 64KB limit
		StringBuilderPool.Put(sb)
	}
}

// ByteSlicePool provides a pool of byte slice pointers to reduce allocations
var ByteSlicePool = sync.Pool{
	New: func() any {
		// Start with a reasonable size
		b := make([]byte, 0, 1024)
		return &b
	},
}

// GetByteSlice gets a byte slice from the pool
func GetByteSlice() *[]byte {
	return ByteSlicePool.Get().(*[]byte)
}

// PutByteSlice returns a byte slice to the pool
func PutByteSlice(b *[]byte) {
	// Only pool slices that aren't too large
	if cap(*b) < 64*1024 { // 64KB limit
		// Reset the slice
		*b = (*b)[:0]
		ByteSlicePool.Put(b)
	}
}

// MapPool provides a pool of string maps to reduce allocations
var MapPool = sync.Pool{
	New: func() any {
		return make(map[string]any, 16) // Start with reasonable capacity
	},
}

// GetStringMap gets a string map from the pool
func GetStringMap() map[string]any {
	m := MapPool.Get().(map[string]any)
	// Clear the map
	for k := range m {
		delete(m, k)
	}
	return m
}

// PutStringMap returns a string map to the pool
func PutStringMap(m map[string]any) {
	// Only pool maps that aren't too large
	if len(m) < 256 {
		MapPool.Put(m)
	}
}
