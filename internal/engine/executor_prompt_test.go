package engine

import (
	"bytes"
	"strings"
	"testing"
)

// promptProgram builds a single-task program whose body contains the given
// statements and executes it with opts, returning the captured output.
func promptProgram(t *testing.T, body string, opts ...Option) (*bytes.Buffer, error) {
	t.Helper()
	program, err := ParseString("version: 2.0\n\ntask \"demo\":\n" + body)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngineWithOptions(append([]Option{WithOutput(&out), WithInput(strings.NewReader(""))}, opts...)...)
	return &out, engine.Execute(program, "demo")
}

func TestEngine_AskConfirmScriptedReader(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		defaultAnswer bool
		want          bool
	}{
		{"yes", "yes\n", false, true},
		{"yes uppercase", "YES\n", false, true},
		{"y short form", "y\n", false, true},
		{"true spelling", "true\n", false, true},
		{"no", "no\n", true, false},
		{"no mixed case", "No\n", false, false},
		{"n short form", "n\n", true, false},
		{"false spelling", "false\n", true, false},
		{"empty line falls back to true default", "\n", true, true},
		{"empty line falls back to false default", "\n", false, false},
		{"end of input falls back to default", "", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			engine := NewEngineWithOptions(WithOutput(&out), WithInput(strings.NewReader(tt.input)))

			got, err := engine.askConfirm("Continue?", tt.defaultAnswer)
			if err != nil {
				t.Fatalf("askConfirm() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("askConfirm() = %v, want %v", got, tt.want)
			}
			if strings.Contains(out.String(), "Please answer yes or no") {
				t.Errorf("unexpected retry message for valid input %q: %q", tt.input, out.String())
			}
		})
	}
}

func TestEngine_AskConfirmInvalidAnswerReasks(t *testing.T) {
	var out bytes.Buffer
	engine := NewEngineWithOptions(WithOutput(&out), WithInput(strings.NewReader("maybe\n")))

	// The first answer is invalid; the reader is then exhausted, which is
	// treated like an empty line, so the shown default is returned.
	got, err := engine.askConfirm("Continue?", true)
	if err != nil {
		t.Fatalf("askConfirm() error = %v", err)
	}
	if !got {
		t.Errorf("askConfirm() = %v, want true (fallback to default)", got)
	}
	if !strings.Contains(out.String(), "Please answer yes or no") {
		t.Errorf("expected re-ask message after invalid answer, got: %q", out.String())
	}
	if !strings.Contains(out.String(), "❓") {
		t.Errorf("expected prompt marker in output, got: %q", out.String())
	}
}

func TestEngine_ConfirmNonInteractiveUsesDeclaredDefault(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantLine string
	}{
		{"default no stored as false", `  confirm "Continue?" defaults to "no" as $migrate
  info "migrate={$migrate}"`, "migrate=false"},
		{"default yes stored as true", `  confirm "Continue?" defaults to yes as $migrate
  info "migrate={$migrate}"`, "migrate=true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := promptProgram(t, tt.body)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !strings.Contains(out.String(), tt.wantLine) {
				t.Errorf("expected %q in output, got: %q", tt.wantLine, out.String())
			}
		})
	}
}

func TestEngine_ConfirmNonInteractiveNoDefaultErrors(t *testing.T) {
	_, err := promptProgram(t, "  confirm \"Deploy?\" as $deploy")
	if err == nil {
		t.Fatal("expected error for confirm without default in non-interactive context, got nil")
	}
	if !strings.Contains(err.Error(), "no interactive terminal and no default answer") {
		t.Errorf("expected clear non-interactive error, got: %v", err)
	}
}

func TestEngine_ConfirmBareAcceptedContinues(t *testing.T) {
	out, err := promptProgram(t, `  confirm "Continue?" defaults to "yes"
  info "after gate"`)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "after gate") {
		t.Errorf("expected task to continue after accepted gate, got: %q", out.String())
	}
	if strings.Contains(out.String(), "Confirmation declined") {
		t.Errorf("unexpected decline notice for accepted gate, got: %q", out.String())
	}
}

func TestEngine_ConfirmBareDeclinedStopsGracefully(t *testing.T) {
	out, err := promptProgram(t, `  confirm "Continue?" defaults to "no"
  info "after gate"`)
	if err != nil {
		t.Fatalf("Execute() should stop gracefully (nil error), got: %v", err)
	}
	if !strings.Contains(out.String(), "Confirmation declined") {
		t.Errorf("expected decline notice, got: %q", out.String())
	}
	if strings.Contains(out.String(), "after gate") {
		t.Errorf("expected statements after the declined gate not to run, got: %q", out.String())
	}
}

func TestEngine_ConfirmAssumeYesOverridesDeclaredDefault(t *testing.T) {
	out, err := promptProgram(t, `  confirm "Continue?" defaults to "no" as $migrate
  info "migrate={$migrate}"`, WithAssumeYes(true))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "migrate=true") {
		t.Errorf("expected --yes assumption to win over declared default, got: %q", out.String())
	}
}

func TestEngine_ConfirmAssumeNoStoresFalseAsVar(t *testing.T) {
	out, err := promptProgram(t, `  confirm "Continue?" defaults to "yes" as $migrate
  info "migrate={$migrate}"`, WithAssumeYes(false))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "migrate=false") {
		t.Errorf("expected --no assumption to store false, got: %q", out.String())
	}
}

func TestEngine_ConfirmAssumeNoDeclinesBareGate(t *testing.T) {
	out, err := promptProgram(t, `  confirm "Continue?" defaults to "yes"
  info "after gate"`, WithAssumeYes(false))
	if err != nil {
		t.Fatalf("Execute() should stop gracefully (nil error), got: %v", err)
	}
	if !strings.Contains(out.String(), "Confirmation declined") {
		t.Errorf("expected decline notice under --no, got: %q", out.String())
	}
	if strings.Contains(out.String(), "after gate") {
		t.Errorf("expected task to stop at declined bare gate, got: %q", out.String())
	}
}

func TestEngine_PromptNonInteractiveUsesDeclaredDefault(t *testing.T) {
	out, err := promptProgram(t, `  prompt "Which environment?" defaults to "dev" as $env
  info "env={$env}"`)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "env=dev") {
		t.Errorf("expected declared prompt default in output, got: %q", out.String())
	}
}

func TestEngine_PromptNonInteractiveNoDefaultErrors(t *testing.T) {
	_, err := promptProgram(t, `  prompt "Release notes?" as $notes`)
	if err == nil {
		t.Fatal("expected error for prompt without default in non-interactive context, got nil")
	}
	if !strings.Contains(err.Error(), "no interactive terminal and no default answer") {
		t.Errorf("expected clear non-interactive error, got: %v", err)
	}
}

func TestEngine_PromptMissingResultVariableErrors(t *testing.T) {
	_, err := promptProgram(t, `  prompt "Which environment?" defaults to "dev"`)
	if err == nil {
		t.Fatal("expected error for prompt without result variable, got nil")
	}
	if !strings.Contains(err.Error(), "result variable is required") {
		t.Errorf("expected result-variable error, got: %v", err)
	}
}

func TestEngine_ConfirmDryRunPrintsWithoutBlocking(t *testing.T) {
	out, err := promptProgram(t, `  confirm "Deploy to production?" as $deploy
  info "deploy={$deploy}"`, WithDryRun(true))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "[DRY RUN] would ask: \"Deploy to production?\"") {
		t.Errorf("expected dry-run question report, got: %q", out.String())
	}
	// Dry-run confirmations without a declared default resolve to yes.
	if !strings.Contains(out.String(), "deploy=true") {
		t.Errorf("expected dry-run confirm to resolve to true, got: %q", out.String())
	}
}

func TestEngine_ConfirmDryRunUsesDeclaredDefault(t *testing.T) {
	out, err := promptProgram(t, `  confirm "Deploy?" defaults to "no" as $deploy
  info "deploy={$deploy}"`, WithDryRun(true))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "deploy=false") {
		t.Errorf("expected dry-run confirm to resolve to its declared default, got: %q", out.String())
	}
}

func TestEngine_ConfirmDryRunBareGateDoesNotStop(t *testing.T) {
	out, err := promptProgram(t, `  confirm "Continue?"
  info "after gate"`, WithDryRun(true))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if strings.Contains(out.String(), "Confirmation declined") {
		t.Errorf("dry-run bare confirm should resolve to yes and not decline, got: %q", out.String())
	}
	if !strings.Contains(out.String(), "after gate") {
		t.Errorf("expected task to continue in dry run, got: %q", out.String())
	}
}

func TestEngine_PromptDryRunPrintsWithoutBlocking(t *testing.T) {
	out, err := promptProgram(t, `  prompt "Which environment?" defaults to "dev" as $env
  info "env={$env}"`, WithDryRun(true))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "[DRY RUN] would ask: \"Which environment?\"") {
		t.Errorf("expected dry-run question report, got: %q", out.String())
	}
	if !strings.Contains(out.String(), "env=dev") {
		t.Errorf("expected dry-run prompt to resolve to its declared default, got: %q", out.String())
	}
}
