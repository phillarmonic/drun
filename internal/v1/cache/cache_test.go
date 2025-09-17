package cache

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/phillarmonic/drun/internal/v1/model"
	"github.com/phillarmonic/drun/internal/v1/tmpl"
)

func TestNewManager(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)

	manager := NewManager(tempDir, engine, false)

	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}

	if manager.cacheDir != tempDir {
		t.Errorf("Expected cache dir %q, got %q", tempDir, manager.cacheDir)
	}

	if manager.disabled {
		t.Error("Expected manager to be enabled")
	}
}

func TestNewManager_Disabled(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)

	manager := NewManager(tempDir, engine, true)

	if !manager.disabled {
		t.Error("Expected manager to be disabled")
	}
}

func TestManager_IsValid_NoCacheKey(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, false)

	recipe := &model.Recipe{
		Help: "Test recipe",
		Run:  model.Step{Lines: []string{"echo test"}},
		// No CacheKey
	}

	ctx := &model.ExecutionContext{}

	valid, err := manager.IsValid(recipe, ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if valid {
		t.Error("Expected cache to be invalid when no cache key is set")
	}
}

func TestManager_IsValid_Disabled(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, true) // disabled

	recipe := &model.Recipe{
		Help:     "Test recipe",
		CacheKey: "test-key",
		Run:      model.Step{Lines: []string{"echo test"}},
	}

	ctx := &model.ExecutionContext{}

	valid, err := manager.IsValid(recipe, ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if valid {
		t.Error("Expected cache to be invalid when manager is disabled")
	}
}

func TestManager_IsValid_NoCache(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, false)

	recipe := &model.Recipe{
		Help:     "Test recipe",
		CacheKey: "test-{{ .version }}",
		Run:      model.Step{Lines: []string{"echo test"}},
	}

	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"version": "1.0.0",
		},
	}

	valid, err := manager.IsValid(recipe, ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if valid {
		t.Error("Expected cache to be invalid when no cache file exists")
	}
}

func TestManager_MarkComplete_AndIsValid(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, false)

	recipe := &model.Recipe{
		Help:     "Test recipe",
		CacheKey: "test-{{ .version }}",
		Run:      model.Step{Lines: []string{"echo test"}},
	}

	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"version": "1.0.0",
		},
	}

	// Initially should not be valid
	valid, err := manager.IsValid(recipe, ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if valid {
		t.Error("Expected cache to be invalid initially")
	}

	// Mark as complete
	err = manager.MarkComplete(recipe, ctx)
	if err != nil {
		t.Fatalf("Expected no error marking complete, got %v", err)
	}

	// Now should be valid
	valid, err = manager.IsValid(recipe, ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !valid {
		t.Error("Expected cache to be valid after marking complete")
	}
}

func TestManager_MarkComplete_Disabled(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, true) // disabled

	recipe := &model.Recipe{
		Help:     "Test recipe",
		CacheKey: "test-key",
		Run:      model.Step{Lines: []string{"echo test"}},
	}

	ctx := &model.ExecutionContext{}

	err := manager.MarkComplete(recipe, ctx)

	if err != nil {
		t.Fatalf("Expected no error when disabled, got %v", err)
	}

	// Should not create any cache files
	files, _ := os.ReadDir(tempDir)
	if len(files) > 0 {
		t.Error("Expected no cache files when manager is disabled")
	}
}

func TestManager_MarkComplete_NoCacheKey(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, false)

	recipe := &model.Recipe{
		Help: "Test recipe",
		Run:  model.Step{Lines: []string{"echo test"}},
		// No CacheKey
	}

	ctx := &model.ExecutionContext{}

	err := manager.MarkComplete(recipe, ctx)

	if err != nil {
		t.Fatalf("Expected no error when no cache key, got %v", err)
	}

	// Should not create any cache files
	files, _ := os.ReadDir(tempDir)
	if len(files) > 0 {
		t.Error("Expected no cache files when no cache key is set")
	}
}

func TestManager_Clear(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, false)

	// Create some cache files
	recipe := &model.Recipe{
		Help:     "Test recipe",
		CacheKey: "test-key-1",
		Run:      model.Step{Lines: []string{"echo test"}},
	}

	ctx := &model.ExecutionContext{}

	err := manager.MarkComplete(recipe, ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify cache file exists
	files, _ := os.ReadDir(tempDir)
	if len(files) == 0 {
		t.Fatal("Expected cache files to be created")
	}

	// Clear cache
	err = manager.Clear()
	if err != nil {
		t.Fatalf("Expected no error clearing cache, got %v", err)
	}

	// Verify cache directory is empty or doesn't exist
	if _, err := os.Stat(tempDir); err == nil {
		files, _ := os.ReadDir(tempDir)
		if len(files) > 0 {
			t.Error("Expected cache directory to be empty after clear")
		}
	}
}

func TestManager_Clear_Disabled(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, true) // disabled

	err := manager.Clear()

	if err != nil {
		t.Fatalf("Expected no error when disabled, got %v", err)
	}
}

func TestManager_GetStats(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, false)

	// Initially should have no entries
	stats, err := manager.GetStats()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 total entries, got %d", stats.TotalEntries)
	}

	if stats.ValidEntries != 0 {
		t.Errorf("Expected 0 valid entries, got %d", stats.ValidEntries)
	}

	// Add some cache entries
	recipe1 := &model.Recipe{
		Help:     "Test recipe 1",
		CacheKey: "test-key-1",
		Run:      model.Step{Lines: []string{"echo test1"}},
	}

	recipe2 := &model.Recipe{
		Help:     "Test recipe 2",
		CacheKey: "test-key-2",
		Run:      model.Step{Lines: []string{"echo test2"}},
	}

	ctx := &model.ExecutionContext{}

	err = manager.MarkComplete(recipe1, ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = manager.MarkComplete(recipe2, ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check stats
	stats, err = manager.GetStats()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 total entries, got %d", stats.TotalEntries)
	}

	if stats.ValidEntries != 2 {
		t.Errorf("Expected 2 valid entries, got %d", stats.ValidEntries)
	}
}

func TestManager_GetStats_Disabled(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, true) // disabled

	stats, err := manager.GetStats()

	if err != nil {
		t.Fatalf("Expected no error when disabled, got %v", err)
	}

	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 total entries when disabled, got %d", stats.TotalEntries)
	}
}

func TestManager_IsValid_ExpiredCache(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, false)

	recipe := &model.Recipe{
		Help:     "Test recipe",
		CacheKey: "test-key",
		Run:      model.Step{Lines: []string{"echo test"}},
	}

	ctx := &model.ExecutionContext{}

	// Mark as complete
	err := manager.MarkComplete(recipe, ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Manually modify the cache file timestamp to be old
	cacheFile := manager.getCacheFilePath("test-key")
	oldTime := time.Now().Add(-2 * time.Hour) // 2 hours ago
	err = os.Chtimes(cacheFile, oldTime, oldTime)
	if err != nil {
		t.Fatalf("Failed to modify cache file timestamp: %v", err)
	}

	// Should now be invalid due to age
	valid, err := manager.IsValid(recipe, ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if valid {
		t.Error("Expected cache to be invalid due to age")
	}
}

func TestManager_getCacheFilePath(t *testing.T) {
	tempDir := t.TempDir()
	engine := tmpl.NewEngine(nil, nil, nil)
	manager := NewManager(tempDir, engine, false)

	cacheKey := "test-cache-key"
	filePath := manager.getCacheFilePath(cacheKey)

	// Should be in the cache directory
	if !strings.HasPrefix(filePath, tempDir) {
		t.Errorf("Expected cache file to be in %q, got %q", tempDir, filePath)
	}

	// Should have .cache extension
	if !strings.HasSuffix(filePath, ".cache") {
		t.Errorf("Expected cache file to have .cache extension, got %q", filePath)
	}

	// Same key should produce same path
	filePath2 := manager.getCacheFilePath(cacheKey)
	if filePath != filePath2 {
		t.Errorf("Expected same cache key to produce same path")
	}

	// Different keys should produce different paths
	filePath3 := manager.getCacheFilePath("different-key")
	if filePath == filePath3 {
		t.Errorf("Expected different cache keys to produce different paths")
	}
}
