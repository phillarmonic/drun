package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_SequentialLoop(t *testing.T) {
	input := `version: 2.0

task "sequential loop":
  accepts items as list
  
  for each item in items:
    info "Processing: {item}"
    step "Working on {item}"
  
  success "Sequential loop completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"items": "file1.txt,file2.txt,file3.txt",
	}

	err = engine.ExecuteWithParams(program, "sequential loop", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔄 Executing 3 items sequentially",
		"📋 Processing item 1/3: file1.txt",
		"ℹ️  Processing: file1.txt",
		"🚀 Working on file1.txt",
		"📋 Processing item 2/3: file2.txt",
		"ℹ️  Processing: file2.txt",
		"🚀 Working on file2.txt",
		"📋 Processing item 3/3: file3.txt",
		"ℹ️  Processing: file3.txt",
		"🚀 Working on file3.txt",
		"✅ Sequential loop completed: 3 items processed",
		"✅ Sequential loop completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ParallelLoop(t *testing.T) {
	input := `version: 2.0

task "parallel loop":
  accepts files as list
  
  for each filename in files in parallel:
    info "Processing file: {filename}"
    step "Working on {filename}"
  
  success "Parallel loop completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"files": "doc1.pdf,doc2.pdf,doc3.pdf,doc4.pdf",
	}

	err = engine.ExecuteWithParams(program, "parallel loop", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔄 Starting parallel execution: 4 items, 4 workers",
		"✅ Worker completed item",
		"🏁 Parallel execution completed: 4/4 items processed",
		"✅ Parallel loop completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Should contain processing messages for all files (parallel execution may interleave output)
	files := []string{"doc1.pdf", "doc2.pdf", "doc3.pdf", "doc4.pdf"}
	for _, filename := range files {
		// Check that the file was processed (either in info or worker completion message)
		if !strings.Contains(outputStr, filename) {
			t.Errorf("Expected output to contain processing for file %s", filename)
		}
	}
}

func TestEngine_ParallelLoopDryRun(t *testing.T) {
	input := `version: 2.0

task "parallel dry run":
  accepts items as list
  
  for each item in items in parallel:
    info "Executing: {item}"
  
  success "Parallel dry run completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	params := map[string]string{
		"items": "build,test,deploy",
	}

	err = engine.ExecuteWithParams(program, "parallel dry run", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"[DRY RUN] Would execute 3 items in parallel",
		"max workers: 5",
		"Worker 1: item = build",
		"Worker 2: item = test",
		"Worker 3: item = deploy",
		"All 3 parallel executions would complete",
		"[DRY RUN] success: Parallel dry run completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ParallelLoopWithFileOperations(t *testing.T) {
	input := `version: 2.0

task "parallel file ops":
  accepts filenames as list
  
  for each filename in filenames in parallel:
    create file "{filename}"
    write "Content for {filename}" to file "{filename}"
    read file "{filename}" as content
    info "File {filename} contains: {content}"
    delete file "{filename}"
  
  success "Parallel file operations completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"filenames": "test1.txt,test2.txt,test3.txt",
	}

	err = engine.ExecuteWithParams(program, "parallel file ops", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔄 Starting parallel execution: 3 items, 3 workers",
		"🏁 Parallel execution completed: 3/3 items processed",
		"✅ Parallel file operations completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Should contain file operations for all files (parallel execution may interleave output)
	files := []string{"test1.txt", "test2.txt", "test3.txt"}
	for _, filename := range files {
		// Check that the file was processed (should appear in output)
		if !strings.Contains(outputStr, filename) {
			t.Errorf("Expected output to contain processing for file %s", filename)
		}
		// Check that content was read
		expectedContent := "Content for " + filename
		if !strings.Contains(outputStr, expectedContent) {
			t.Errorf("Expected output to contain content for %s", filename)
		}
	}
}

func TestEngine_EmptyLoop(t *testing.T) {
	input := `version: 2.0

task "empty loop":
  accepts items as list
  
  for each item in items:
    info "Processing: {item}"
  
  success "Empty loop completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"items": "", // empty list
	}

	err = engine.ExecuteWithParams(program, "empty loop", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"ℹ️  No items to process in loop",
		"✅ Empty loop completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_LoopWithConditionals(t *testing.T) {
	input := `version: 2.0

task "conditional loop":
  accepts environments as list
  given deploy_prod as boolean defaults to "false"
  
  for each env in environments:
    info "Processing environment: {env}"
    
    when env is "production":
      when deploy_prod is "true":
        info "✅ Deploying to production"
      when deploy_prod is "false":
        warn "⚠️  Skipping production deployment"
    
    when env is "staging":
      info "🧪 Deploying to staging"
    
    when env is "dev":
      info "🔧 Deploying to development"
  
  success "Conditional loop completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"environments": "dev,staging,production",
		"deploy_prod":  "false",
	}

	err = engine.ExecuteWithParams(program, "conditional loop", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔄 Executing 3 items sequentially",
		"ℹ️  Processing environment: dev",
		"ℹ️  🔧 Deploying to development",
		"ℹ️  Processing environment: staging",
		"ℹ️  🧪 Deploying to staging",
		"ℹ️  Processing environment: production",
		"⚠️  ⚠️  Skipping production deployment",
		"✅ Sequential loop completed: 3 items processed",
		"✅ Conditional loop completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}
