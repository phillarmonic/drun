package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_GitCloneRepository(t *testing.T) {
	input := `version: 2.0

task "clone_repo":
  git clone repository "https://github.com/user/repo.git" to "local-dir"
  success "Repository cloned successfully!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "clone_repo", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔗 Running Git: git clone https://github.com/user/repo.git local-dir",
		"✅ Repository cloned successfully!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_GitInitRepository(t *testing.T) {
	input := `version: 2.0

task "init_repo":
  git init repository in "project-dir"
  success "Repository initialized!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "init_repo", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔗 Running Git: git init project-dir",
		"✅ Repository initialized!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_GitAddFiles(t *testing.T) {
	input := `version: 2.0

task "stage_changes":
  git add files "src/*.go"
  success "Files staged!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "stage_changes", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔗 Running Git: git add src/*.go",
		"✅ Files staged!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_GitCommitChanges(t *testing.T) {
	input := `version: 2.0

task "commit_work":
  git commit changes with message "Add new feature"
  success "Changes committed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "commit_work", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔗 Running Git: git commit -m \"Add new feature\"",
		"✅ Changes committed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_GitCommitAllChanges(t *testing.T) {
	input := `version: 2.0

task "commit_all":
  git commit all changes with message "Update documentation"
  success "All changes committed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "commit_all", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔗 Running Git: git commit -a -m \"Update documentation\"",
		"✅ All changes committed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_GitPushToRemote(t *testing.T) {
	input := `version: 2.0

task "push_changes":
  git push to remote "origin" branch "main"
  success "Changes pushed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "push_changes", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔗 Running Git: git push origin main",
		"✅ Changes pushed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_GitStatus(t *testing.T) {
	input := `version: 2.0

task "check_status":
  git status
  success "Status checked!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "check_status", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔗 Running Git: git status",
		"✅ Status checked!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_GitShowCurrentBranch(t *testing.T) {
	input := `version: 2.0

task "current_branch":
  git show current branch
  success "Current branch shown!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "current_branch", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔗 Running Git: git branch --show-current",
		"✅ Current branch shown!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_GitWithVariableInterpolation(t *testing.T) {
	input := `version: 2.0

task "git_workflow":
  requires repo_url from ["https://github.com/user/repo1.git", "https://github.com/user/repo2.git"]
  requires commit_msg from ["Initial commit", "Update code", "Fix bugs"]
  
  git clone repository "{repo_url}" to "local-repo"
  git add files "."
  git commit changes with message "{commit_msg}"
  success "Git workflow completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"repo_url":   "https://github.com/user/repo1.git",
		"commit_msg": "Initial commit",
	}

	err = engine.ExecuteWithParams(program, "git_workflow", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔗 Running Git: git clone https://github.com/user/repo1.git local-repo",
		"🔗 Running Git: git add .",
		"🔗 Running Git: git commit -m \"Initial commit\"",
		"✅ Git workflow completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_GitDryRun(t *testing.T) {
	input := `version: 2.0

task "git_operations":
  git add files "src/*.go"
  git commit changes with message "Update Go code"
  git push to remote "origin" branch "main"
  success "Git operations completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.ExecuteWithParams(program, "git_operations", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"[DRY RUN] Would execute Git command: git add src/*.go",
		"[DRY RUN] Would execute Git command: git commit -m \"Update Go code\"",
		"[DRY RUN] Would execute Git command: git push origin main",
		"[DRY RUN] success: Git operations completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_GitWithDependencies(t *testing.T) {
	input := `version: 2.0

task "stage_files":
  git add files "."
  success "Files staged!"

task "commit_and_push":
  depends on stage_files
  
  git commit changes with message "Automated commit"
  git push to remote "origin" branch "main"
  success "Changes committed and pushed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "commit_and_push", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	// Check that staging runs before commit and push
	stageIdx := strings.Index(outputStr, "git add .")
	commitIdx := strings.Index(outputStr, "git commit")

	if stageIdx == -1 {
		t.Errorf("Expected git add command to run")
	}
	if commitIdx == -1 {
		t.Errorf("Expected git commit command to run")
	}
	if stageIdx >= commitIdx {
		t.Errorf("Git add should run before git commit")
	}

	expectedParts := []string{
		"🔗 Running Git: git add .",
		"✅ Files staged!",
		"🔗 Running Git: git commit -m \"Automated commit\"",
		"🔗 Running Git: git push origin main",
		"✅ Changes committed and pushed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}
