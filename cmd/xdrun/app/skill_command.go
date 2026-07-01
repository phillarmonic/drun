package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const drunBasicsSkillName = "drun-basics"

type installSkillResult struct {
	Created []string
	Updated []string
	Skipped []string
}

type skillTemplateFile struct {
	path        string
	content     string
	managedName string
}

// createSkillCommand creates the cmd:skill subcommand for installing project AI guidance.
func (a *App) createSkillCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cmd:skill",
		Short: "Install project AI guidance for drun",
		Long: `Install project-level guidance files that teach AI coding assistants
how to work with drun specs and the xdrun CLI.

Supported skill bundles:
  drun-basics  Install cross-agent drun/xdrun basics for a repository
  basics       Alias for drun-basics

Note: The 'cmd:' prefix is reserved for built-in commands to avoid conflicts with user tasks.`,
	}

	cmd.AddCommand(createSkillInstallCommand())

	return cmd
}

func createSkillInstallCommand() *cobra.Command {
	var (
		targetDir string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "install <skill-name>",
		Short: "Install a project skill bundle into a repository",
		Long: `Install a named project skill bundle into the target repository.

The drun-basics bundle creates AI guidance files for common assistants so
repositories using drun can teach agents the expected xdrun workflow.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillName, err := normalizeProjectSkillName(args[0])
			if err != nil {
				return err
			}

			if targetDir == "" {
				targetDir = "."
			}

			result, err := InstallProjectSkill(skillName, targetDir, force)
			if err != nil {
				return err
			}

			fmt.Printf("✅  Installed '%s' into %s\n", skillName, targetDir)
			for _, path := range result.Created {
				fmt.Printf("  created  %s\n", path)
			}
			for _, path := range result.Updated {
				fmt.Printf("  updated  %s\n", path)
			}
			for _, path := range result.Skipped {
				fmt.Printf("  skipped  %s (already exists)\n", path)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&targetDir, "target", ".", "Target repository root")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite managed files if they already exist")

	return cmd
}

// InstallProjectSkill installs a named AI guidance bundle into a repository.
func InstallProjectSkill(skillName, targetDir string, force bool) (installSkillResult, error) {
	skillName, err := normalizeProjectSkillName(skillName)
	if err != nil {
		return installSkillResult{}, err
	}

	info, err := os.Stat(targetDir)
	if err != nil {
		return installSkillResult{}, fmt.Errorf("failed to access target directory '%s': %w", targetDir, err)
	}
	if !info.IsDir() {
		return installSkillResult{}, fmt.Errorf("target '%s' is not a directory", targetDir)
	}

	projectName := inferProjectNameFromPath(targetDir)
	files := projectSkillTemplates(skillName, projectName)

	var result installSkillResult
	for _, file := range files {
		relPath := filepath.ToSlash(file.path)
		state, err := writeSkillFile(filepath.Join(targetDir, file.path), file.content, file.managedName, force)
		if err != nil {
			return installSkillResult{}, err
		}

		switch state {
		case "created":
			result.Created = append(result.Created, relPath)
		case "updated":
			result.Updated = append(result.Updated, relPath)
		case "skipped":
			result.Skipped = append(result.Skipped, relPath)
		}
	}

	sort.Strings(result.Created)
	sort.Strings(result.Updated)
	sort.Strings(result.Skipped)

	return result, nil
}

func normalizeProjectSkillName(name string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "drun-basics", "basics":
		return drunBasicsSkillName, nil
	default:
		return "", fmt.Errorf("unknown skill bundle %q\nSupported bundles: drun-basics", name)
	}
}

func inferProjectNameFromPath(path string) string {
	cleaned := filepath.Clean(path)
	name := strings.TrimSpace(filepath.Base(cleaned))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "my-app"
	}
	return name
}

func writeSkillFile(path, content, managedName string, force bool) (string, error) {
	if managedName != "" {
		return writeManagedBlockFile(path, content, managedName)
	}

	existed := false
	if existing, err := os.ReadFile(path); err == nil {
		existed = true
		if string(existing) == content {
			return "skipped", nil
		}
		if !force {
			return "skipped", nil
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read existing file '%s': %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return "", fmt.Errorf("failed to create directory for '%s': %w", path, err)
	}
	// #nosec G306 -- installer intentionally writes generated guidance files under the user-selected repository root.
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return "", fmt.Errorf("failed to write '%s': %w", path, err)
	}

	if existed {
		return "updated", nil
	}

	return "created", nil
}

func writeManagedBlockFile(path, content, managedName string) (string, error) {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read existing file '%s': %w", path, err)
	}

	startMarker := managedBlockStart(managedName)
	endMarker := managedBlockEnd(managedName)
	block := strings.Join([]string{startMarker, content, endMarker, ""}, "\n")

	var next string
	state := "created"
	if os.IsNotExist(err) {
		next = block
	} else {
		state = "updated"
		current := string(existing)
		updated, changed := upsertManagedBlock(current, block, startMarker, endMarker)
		if !changed {
			return "skipped", nil
		}
		next = updated
	}

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return "", fmt.Errorf("failed to create directory for '%s': %w", path, err)
	}
	// #nosec G304,G306,G703 -- installer intentionally updates a drun-managed block within repository guidance files under the user-selected root.
	if err := os.WriteFile(path, []byte(next), 0600); err != nil {
		return "", fmt.Errorf("failed to write '%s': %w", path, err)
	}

	return state, nil
}

func managedBlockStart(name string) string {
	return fmt.Sprintf("<!-- drun:skill:%s:start -->", name)
}

func managedBlockEnd(name string) string {
	return fmt.Sprintf("<!-- drun:skill:%s:end -->", name)
}

func upsertManagedBlock(current, block, startMarker, endMarker string) (string, bool) {
	start := strings.Index(current, startMarker)
	end := strings.Index(current, endMarker)
	if start >= 0 && end > start {
		end += len(endMarker)
		if end < len(current) && current[end] == '\n' {
			end++
		}
		next := current[:start] + block + current[end:]
		return next, next != current
	}

	trimmed := strings.TrimRight(current, "\n")
	if trimmed == "" {
		return block, true
	}

	next := trimmed + "\n\n" + block
	return next, next != current
}

func projectSkillTemplates(skillName, projectName string) []skillTemplateFile {
	if skillName != drunBasicsSkillName {
		return nil
	}

	guide := buildDrunBasicsGuide(projectName)

	return []skillTemplateFile{
		{
			path:    ".drun/ai/drun-basics.md",
			content: guide,
		},
		{
			path: ".codex/skills/drun-basics/SKILL.md",
			content: strings.Join([]string{
				"---",
				"name: drun-basics",
				`description: "Use when working in a repository that uses drun or xdrun for automation. Teaches the agent where specs live, how to run tasks, how to pass parameters, and how to verify tool requirements."`,
				"---",
				"",
				"# drun-basics",
				"",
				"Use this skill when the task mentions drun, xdrun, `.drun/spec.drun`, task automation, or repository workflows implemented in drun.",
				"",
				"Read `.drun/ai/drun-basics.md` completely before making changes or proposing commands.",
				"",
				"Apply the workflow in that guide exactly:",
				"- inspect the existing drun file before editing",
				"- keep task parameters in `key=value` form",
				"- prefer `xdrun --list` and focused task runs for verification",
				"- use `@platform(...)` for platform-specific tasks instead of burying OS differences inside one task",
				"- when a task family mixes platform variants with one unannotated task, drun resolves exact platform match first and uses the unannotated task as the fallback",
				"- use `requires tools:` for hard tool requirements",
				"",
			}, "\n"),
		},
		{
			path: ".cursor/rules/drun-basics.mdc",
			content: strings.Join([]string{
				"---",
				"description: Work on repositories that use drun/xdrun automation.",
				"globs:",
				`  - "**/*.drun"`,
				`  - "**/.drun/**"`,
				"alwaysApply: false",
				"---",
				"",
				"When the repository uses drun or xdrun, read `.drun/ai/drun-basics.md` before changing automation files.",
				"",
				"Important reminders:",
				"- `xdrun` executes drun specs",
				"- prefer `xdrun --list` to inspect available tasks",
				"- task parameters use `key=value` syntax, not CLI flags",
				"- CLI behavior still uses `--flags`",
				"- prefer `@platform(\"linux\")`, `@platform(\"mac\")`, and `@platform(\"windows\")` for platform-specific declarations",
				"- one unannotated task may coexist with platform-tagged variants as the fallback when no platform-specific variant matches",
				"- declare hard dependencies with `requires tools:`",
				"",
			}, "\n"),
		},
		{
			path: "AGENTS.md",
			content: strings.Join([]string{
				"When tasks mention drun, xdrun, or `.drun/spec.drun`, read `.drun/ai/drun-basics.md` first and follow it.",
			}, "\n"),
			managedName: drunBasicsSkillName,
		},
		{
			path: "CLAUDE.md",
			content: strings.Join([]string{
				"Before editing drun tasks or suggesting xdrun commands, read `.drun/ai/drun-basics.md` and follow its workflow.",
			}, "\n"),
			managedName: drunBasicsSkillName,
		},
		{
			path: ".github/copilot-instructions.md",
			content: strings.Join([]string{
				"Before editing drun files or suggesting task commands:",
				"- read `.drun/ai/drun-basics.md`",
				"- inspect the existing spec before changing it",
				"- use `xdrun --list` to discover tasks",
				"- keep task parameters in `key=value` form",
				"- use `@platform(...)` to separate platform-specific tasks clearly",
				"- remember that an unannotated task in the same family acts as the fallback after platform-specific variants are checked",
			}, "\n"),
			managedName: drunBasicsSkillName,
		},
	}
}

func buildDrunBasicsGuide(projectName string) string {
	return fmt.Sprintf(strings.Join([]string{
		"# drun Basics for AI Agents",
		"",
		"This repository uses drun for automation.",
		"The CLI binary is `xdrun`.",
		"",
		"## What to Know First",
		"",
		"- Main drun spec location: `.drun/spec.drun`",
		"- Default initialization command: `xdrun --init`",
		"- List available tasks: `xdrun --list`",
		"- Run a task: `xdrun <task>`",
		"- Pass task parameters as `key=value`, for example `xdrun deploy environment=production`",
		"- Keep CLI behavior flags separate, for example `xdrun deploy environment=production --dry-run`",
		"- Official upstream repository for clarification and broader docs: https://github.com/phillarmonic/drun",
		"",
		"## Recommended Workflow",
		"",
		"1. Read the existing drun file before making changes.",
		"2. If there is no spec yet, initialize one with `xdrun --init`.",
		"3. Use `xdrun --list` to inspect task names instead of guessing.",
		"4. For platform-specific workflows, prefer separate declarations with `@platform(...)` instead of mixing OS branches into one task when the behavior is substantially different.",
		"5. Use canonical platform names in new specs: `linux`, `mac`, `windows`. Legacy `darwin` still parses, but prefer `mac` in new code and examples.",
		"6. If a task family includes both platform-tagged variants and one unannotated task, drun resolves the exact platform variant first and uses the unannotated task as the fallback.",
		"7. When adding hard dependencies, declare them with `requires tools:`.",
		"8. Prefer small, readable tasks that explain intent with `means`, `info`, and `step`.",
		"9. For AI-driven CI or noisy checks, prefer tasks declared with `mode \"ci\"` so successful shell stdout/stderr stays buffered and only failure output is emitted.",
		"10. After editing a spec, run the narrowest relevant `xdrun` command to verify behavior.",
		"",
		"## Tool Checks",
		"",
		"Prefer declarative requirements when a task depends on a binary or minimum version:",
		"",
		"```drun",
		`project "%s" version "1.0":`,
		`  requires tools:`,
		`    go >= "1.21"`,
		`    docker`,
		"```",
		"",
		"Task-level checks are also valid:",
		"",
		"```drun",
		`task "test" means "Run the test suite":`,
		`  requires tools:`,
		`    go`,
		`  run "go test ./..."`,
		"```",
		"",
		"## Writing Good drun Specs",
		"",
		"- Keep the file readable at a glance.",
		"- Prefer task names that match user intent.",
		"- If two platforms need the same user-facing workflow name, use duplicate task names with disjoint `@platform(...)` annotations so `xdrun <task>` resolves the correct variant automatically.",
		"- A task family may also include one unannotated task as a fallback when no platform-specific variant matches.",
		"- Use `given $name defaults to ...` for optional parameters.",
		"- Use `call task ...` instead of duplicating steps across tasks.",
		"- Use `mode \"ci\"` for noisy validation tasks when you want to save output tokens during successful runs.",
		"- Keep shell commands explicit inside `run \"...\"`.",
		"",
		"## Lifecycle Basics",
		"",
		"- Bootstrap a repository with `xdrun --init`.",
		"- Evolve `.drun/spec.drun` as the source of truth for project automation.",
		"- Use `xdrun --list` as the quickest way to discover available workflows.",
		"- For CI-style tasks, `mode \"ci\"` buffers normal shell stdout/stderr and only prints that buffered output when a command fails, which reduces noisy output and saves tokens for AI runs.",
		"- Use targeted runs such as `xdrun test` or `xdrun build --dry-run` to validate changes.",
		"- If the local repository guidance is incomplete, check the official upstream repo for clarification: https://github.com/phillarmonic/drun",
		"",
		"## Example Starter Spec",
		"",
		"```drun",
		"version: 2.0",
		"",
		`project "%s" version "1.0":`,
		`  requires tools:`,
		`    go`,
		"",
		`task "default" means "Show available automation":`,
		`  info "Run xdrun --list to inspect tasks"`,
		"",
		`task "test" means "Run tests":`,
		`  run "go test ./..."`,
		"",
		`@platform("linux", "mac")`,
		`task "shell" means "Open a Unix shell":`,
		`  run "bash" attached`,
		"",
		`@platform("windows")`,
		`task "shell" means "Open PowerShell":`,
		`  run "pwsh.exe" attached`,
		"",
		`task "ci" mode "ci" means "Run noisy checks with buffered output":`,
		`  run "go test ./..."`,
		"```",
		"",
		"## Git Policy and Hooks",
		"",
		"Projects can define git conventions in the project body using the `git policy:` block.",
		"When a git policy is defined, use `xdrun cmd:hook install` to install drun-managed git hooks (like commit-msg, pre-push) that enforce these conventions.",
		"",
		"```drun",
		`project "%s" version "1.0":`,
		`  git policy:`,
		`    default branches: "master", "main"`,
		`    branch naming: "{type}/{identifier}-{description}"`,
		`    types: "feat", "fix", "chore"`,
		`    commit messages: "{identifier}: {message}"`,
		`    extract identifier from branch`,
		`    enforce signed commits`,
		"```",
	}, "\n"), projectName, projectName, projectName)
}
