package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeProjectSkillName(t *testing.T) {
	t.Parallel()

	got, err := normalizeProjectSkillName("basics")
	if err != nil {
		t.Fatalf("normalizeProjectSkillName() error = %v", err)
	}
	if got != drunBasicsSkillName {
		t.Fatalf("normalizeProjectSkillName() = %q, want %q", got, drunBasicsSkillName)
	}
}

func TestInstallProjectSkillCreatesExpectedFiles(t *testing.T) {
	t.Parallel()

	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "sample-app")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	result, err := InstallProjectSkill("drun-basics", projectDir, false)
	if err != nil {
		t.Fatalf("InstallProjectSkill() error = %v", err)
	}

	if len(result.Created) != 6 {
		t.Fatalf("created files = %d, want 6 (%v)", len(result.Created), result.Created)
	}
	if len(result.Updated) != 0 {
		t.Fatalf("updated files = %v, want none", result.Updated)
	}
	if len(result.Skipped) != 0 {
		t.Fatalf("skipped files = %v, want none", result.Skipped)
	}

	files := []string{
		".drun/ai/drun-basics.md",
		".codex/skills/drun-basics/SKILL.md",
		".cursor/rules/drun-basics.mdc",
		".github/copilot-instructions.md",
		"AGENTS.md",
		"CLAUDE.md",
	}

	for _, relativePath := range files {
		fullPath := filepath.Join(projectDir, relativePath)
		if _, err := os.Stat(fullPath); err != nil {
			t.Fatalf("expected %s to exist: %v", relativePath, err)
		}
	}

	guideContent, err := os.ReadFile(filepath.Join(projectDir, ".drun/ai/drun-basics.md"))
	if err != nil {
		t.Fatalf("ReadFile() guide error = %v", err)
	}
	guide := string(guideContent)
	if !strings.Contains(guide, `project "sample-app" version "1.0":`) {
		t.Fatalf("guide did not embed target directory name:\n%s", guide)
	}
	if !strings.Contains(guide, "`xdrun --list`") {
		t.Fatalf("guide should mention xdrun --list:\n%s", guide)
	}
}

func TestInstallProjectSkillSkipsExistingFilesWithoutForce(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	agentsPath := filepath.Join(projectDir, "AGENTS.md")
	original := "# Existing agent instructions\n\nKeep this line.\n"
	if err := os.WriteFile(agentsPath, []byte(original), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := InstallProjectSkill("drun-basics", projectDir, false)
	if err != nil {
		t.Fatalf("InstallProjectSkill() error = %v", err)
	}

	if !containsPath(result.Updated, "AGENTS.md") {
		t.Fatalf("expected AGENTS.md to be updated, got %v", result.Updated)
	}

	content, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(content)
	if !strings.Contains(got, "Keep this line.") {
		t.Fatalf("AGENTS.md lost existing content:\n%s", got)
	}
	if !strings.Contains(got, managedBlockStart(drunBasicsSkillName)) {
		t.Fatalf("AGENTS.md did not include managed start marker:\n%s", got)
	}
	if !strings.Contains(got, ".drun/ai/drun-basics.md") {
		t.Fatalf("AGENTS.md did not include installed guidance:\n%s", got)
	}
}

func TestInstallProjectSkillUpdatesExistingManagedBlock(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	agentsPath := filepath.Join(projectDir, "AGENTS.md")
	original := strings.Join([]string{
		"# Existing",
		"",
		managedBlockStart(drunBasicsSkillName),
		"Old drun guidance",
		managedBlockEnd(drunBasicsSkillName),
		"",
		"Keep this too.",
	}, "\n")
	if err := os.WriteFile(agentsPath, []byte(original), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := InstallProjectSkill("drun-basics", projectDir, false)
	if err != nil {
		t.Fatalf("InstallProjectSkill() error = %v", err)
	}

	if !containsPath(result.Updated, "AGENTS.md") {
		t.Fatalf("expected AGENTS.md to be updated, got %v", result.Updated)
	}

	content, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(content)
	if strings.Contains(got, "Old drun guidance") {
		t.Fatalf("AGENTS.md still contains stale managed content:\n%s", got)
	}
	if !strings.Contains(got, ".drun/ai/drun-basics.md") {
		t.Fatalf("AGENTS.md did not contain refreshed guidance:\n%s", got)
	}
	if !strings.Contains(got, "Keep this too.") {
		t.Fatalf("AGENTS.md lost surrounding content:\n%s", got)
	}
}

func TestUpsertManagedBlockAppendsWhenMissing(t *testing.T) {
	t.Parallel()

	current := "# Header\n"
	block := strings.Join([]string{
		managedBlockStart(drunBasicsSkillName),
		"hello",
		managedBlockEnd(drunBasicsSkillName),
		"",
	}, "\n")

	got, changed := upsertManagedBlock(current, block, managedBlockStart(drunBasicsSkillName), managedBlockEnd(drunBasicsSkillName))
	if !changed {
		t.Fatal("expected upsertManagedBlock to report change")
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("expected managed block to be appended:\n%s", got)
	}
}

func containsPath(paths []string, needle string) bool {
	for _, path := range paths {
		if path == needle {
			return true
		}
	}
	return false
}
