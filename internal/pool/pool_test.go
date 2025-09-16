package pool

import (
	"strings"
	"sync"
	"testing"
)

func TestStringBuilderPool_GetAndPut(t *testing.T) {
	// Get a builder from the pool
	sb := GetStringBuilder()
	if sb == nil {
		t.Fatal("GetStringBuilder returned nil")
	}

	// Should be empty/reset
	if sb.Len() != 0 {
		t.Errorf("Expected empty builder, got length %d", sb.Len())
	}

	// Use the builder
	sb.WriteString("test content")
	if sb.String() != "test content" {
		t.Errorf("Expected 'test content', got '%s'", sb.String())
	}

	// Put it back
	PutStringBuilder(sb)

	// Get another one - should be clean
	sb2 := GetStringBuilder()
	if sb2.Len() != 0 {
		t.Errorf("Expected clean builder from pool, got length %d", sb2.Len())
	}
}

func TestStringBuilderPool_LargeBuilderNotPooled(t *testing.T) {
	sb := GetStringBuilder()

	// Create a large string to exceed the 64KB limit
	largeString := strings.Repeat("a", 65*1024) // 65KB
	sb.WriteString(largeString)

	if sb.Cap() < 64*1024 {
		t.Skip("Builder didn't grow large enough for test")
	}

	// Put it back - should not be pooled due to size
	PutStringBuilder(sb)

	// This test is hard to verify directly since we can't inspect the pool,
	// but we can at least ensure the function doesn't panic
}

func TestStringBuilderPool_Concurrent(t *testing.T) {
	const numGoroutines = 100
	const numOperations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				sb := GetStringBuilder()
				sb.WriteString("goroutine")
				sb.WriteString(strings.Repeat("x", id))

				if sb.Len() == 0 {
					t.Errorf("Builder should not be empty after writing")
				}

				PutStringBuilder(sb)
			}
		}(i)
	}

	wg.Wait()
}

func TestByteSlicePool_GetAndPut(t *testing.T) {
	// Get a byte slice from the pool
	b := GetByteSlice()
	if b == nil {
		t.Fatal("GetByteSlice returned nil")
	}

	// Should be empty
	if len(*b) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(*b))
	}

	// Should have some capacity
	if cap(*b) == 0 {
		t.Error("Expected slice with capacity, got 0")
	}

	// Use the slice
	*b = append(*b, []byte("test data")...)
	if string(*b) != "test data" {
		t.Errorf("Expected 'test data', got '%s'", string(*b))
	}

	// Put it back
	PutByteSlice(b)

	// Get another one - should be clean
	b2 := GetByteSlice()
	if len(*b2) != 0 {
		t.Errorf("Expected clean slice from pool, got length %d", len(*b2))
	}
}

func TestByteSlicePool_LargeSliceNotPooled(t *testing.T) {
	b := GetByteSlice()

	// Create a large slice to exceed the 64KB limit
	largeData := make([]byte, 65*1024) // 65KB
	*b = append(*b, largeData...)

	if cap(*b) < 64*1024 {
		t.Skip("Slice didn't grow large enough for test")
	}

	// Put it back - should not be pooled due to size
	PutByteSlice(b)

	// This test is hard to verify directly since we can't inspect the pool,
	// but we can at least ensure the function doesn't panic
}

func TestByteSlicePool_Concurrent(t *testing.T) {
	const numGoroutines = 100
	const numOperations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				b := GetByteSlice()
				*b = append(*b, []byte("data")...)
				*b = append(*b, byte(id))

				if len(*b) == 0 {
					t.Errorf("Slice should not be empty after writing")
				}

				PutByteSlice(b)
			}
		}(i)
	}

	wg.Wait()
}

func TestStringMapPool_GetAndPut(t *testing.T) {
	// Get a map from the pool
	m := GetStringMap()
	if m == nil {
		t.Fatal("GetStringMap returned nil")
	}

	// Should be empty
	if len(m) != 0 {
		t.Errorf("Expected empty map, got length %d", len(m))
	}

	// Use the map
	m["key1"] = "value1"
	m["key2"] = 42
	m["key3"] = true

	if len(m) != 3 {
		t.Errorf("Expected map with 3 entries, got %d", len(m))
	}

	if m["key1"] != "value1" {
		t.Errorf("Expected 'value1', got '%v'", m["key1"])
	}

	// Put it back
	PutStringMap(m)

	// Get another one - should be clean
	m2 := GetStringMap()
	if len(m2) != 0 {
		t.Errorf("Expected clean map from pool, got length %d", len(m2))
	}
}

func TestStringMapPool_LargeMapNotPooled(t *testing.T) {
	m := GetStringMap()

	// Create a large map to exceed the 256 entry limit
	for i := 0; i < 300; i++ {
		m[string(rune('a'+i%26))+string(rune('0'+i/26))] = i
	}

	if len(m) < 256 {
		t.Skip("Map didn't grow large enough for test")
	}

	// Put it back - should not be pooled due to size
	PutStringMap(m)

	// This test is hard to verify directly since we can't inspect the pool,
	// but we can at least ensure the function doesn't panic
}

func TestStringMapPool_Concurrent(t *testing.T) {
	const numGoroutines = 100
	const numOperations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				m := GetStringMap()
				m["goroutine"] = id
				m["operation"] = j
				m["data"] = "test"

				if len(m) != 3 {
					t.Errorf("Expected map with 3 entries, got %d", len(m))
				}

				PutStringMap(m)
			}
		}(i)
	}

	wg.Wait()
}

func TestStringMapPool_ClearingBehavior(t *testing.T) {
	// Get a map and populate it
	m1 := GetStringMap()
	m1["test"] = "value"
	m1["number"] = 123

	// Put it back
	PutStringMap(m1)

	// Get another map - should be empty
	m2 := GetStringMap()
	if len(m2) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(m2))
	}

	// Verify specific keys are not present
	if _, exists := m2["test"]; exists {
		t.Error("Expected 'test' key to be cleared from pooled map")
	}
	if _, exists := m2["number"]; exists {
		t.Error("Expected 'number' key to be cleared from pooled map")
	}
}

// Benchmark tests to verify pool effectiveness
func BenchmarkStringBuilder_WithPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb := GetStringBuilder()
		sb.WriteString("benchmark test string")
		sb.WriteString(" with more content")
		_ = sb.String()
		PutStringBuilder(sb)
	}
}

func BenchmarkStringBuilder_WithoutPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb := &strings.Builder{}
		sb.WriteString("benchmark test string")
		sb.WriteString(" with more content")
		_ = sb.String()
	}
}

func BenchmarkByteSlice_WithPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice := GetByteSlice()
		*slice = append(*slice, []byte("benchmark test data")...)
		*slice = append(*slice, []byte(" with more content")...)
		_ = string(*slice)
		PutByteSlice(slice)
	}
}

func BenchmarkByteSlice_WithoutPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice := make([]byte, 0, 1024)
		slice = append(slice, []byte("benchmark test data")...)
		slice = append(slice, []byte(" with more content")...)
		_ = string(slice)
	}
}

func BenchmarkStringMap_WithPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := GetStringMap()
		m["key1"] = "value1"
		m["key2"] = i
		m["key3"] = true
		_ = len(m)
		PutStringMap(m)
	}
}

func BenchmarkStringMap_WithoutPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := make(map[string]any, 16)
		m["key1"] = "value1"
		m["key2"] = i
		m["key3"] = true
		_ = len(m)
	}
}
