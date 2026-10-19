package engine

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

// executeCI runs a program with the given engine options and returns the
// captured output and execution error.
func executeCI(t *testing.T, input, task string, opts ...Option) (string, error) {
	t.Helper()

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	options := append([]Option{WithOutput(&buf)}, opts...)
	err := NewEngineWithOptions(options...).Execute(program, task)
	return buf.String(), err
}

func TestCIModePrintsPerStepSummary(t *testing.T) {
	input := `
version: 2.0

task "ci" mode "ci":
  step "PHP unit tests"
  run "echo ok"
  step "Docs build"
  run "echo ok"
  success "done"
`

	out, err := executeCI(t, input, "ci")
	if err != nil {
		t.Fatalf("Execution failed: %v\n%s", err, out)
	}

	for _, want := range []string{
		"CI STEP SUMMARY",
		"STEP=php-unit-tests STATUS=passed",
		"STEP=docs-build STATUS=passed",
		"CI_STATUS=passed STEPS=2 PASSED=2 FAILED=0",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output:\n%s", want, out)
		}
	}
}

func TestCIModeSummaryMarksFailingStep(t *testing.T) {
	input := `
version: 2.0

task "ci" mode "ci":
  step "Lint"
  run "echo ok"
  step "Unit tests"
  run "exit 1"
  step "Never reached"
  run "echo ok"
`

	out, err := executeCI(t, input, "ci")
	if err == nil {
		t.Fatalf("expected failure, got success:\n%s", out)
	}

	for _, want := range []string{
		"STEP=lint STATUS=passed",
		"STEP=unit-tests STATUS=failed",
		"CI_STATUS=failed STEPS=2 PASSED=1 FAILED=1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output:\n%s", want, out)
		}
	}
	if strings.Contains(out, "never-reached") {
		t.Errorf("a step after the failure should not be recorded:\n%s", out)
	}
}

func TestCIModeSummaryRecordsCalledTaskSteps(t *testing.T) {
	input := `
version: 2.0

task "lint":
  step "Static analysis"
  run "echo ok"

task "ci" mode "ci":
  step "Lint"
  call task "lint"
  step "Tests"
  run "echo ok"
`

	out, err := executeCI(t, input, "ci")
	if err != nil {
		t.Fatalf("Execution failed: %v\n%s", err, out)
	}

	// Steps from a called task that inherits ci mode are part of the run.
	for _, want := range []string{
		"STEP=lint STATUS=passed",
		"STEP=static-analysis STATUS=passed",
		"STEP=tests STATUS=passed",
		"CI_STATUS=passed STEPS=3 PASSED=3 FAILED=0",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output:\n%s", want, out)
		}
	}
}

func TestCIModeSummaryJSON(t *testing.T) {
	input := `
version: 2.0

task "ci" mode "ci":
  step "PHP unit tests"
  run "echo ok"
  step "Docs build"
  run "echo ok"
  success "done"
`

	out, err := executeCI(t, input, "ci", WithCISummary("json"))
	if err != nil {
		t.Fatalf("Execution failed: %v\n%s", err, out)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	last := lines[len(lines)-1]

	var payload ciSummaryJSON
	if unmarshalErr := json.Unmarshal([]byte(last), &payload); unmarshalErr != nil {
		t.Fatalf("last line is not valid JSON summary (%v): %q\nfull output:\n%s", unmarshalErr, last, out)
	}

	if payload.Task != "ci" {
		t.Errorf("task = %q, want ci", payload.Task)
	}
	if payload.Status != ciStatusPassed {
		t.Errorf("status = %q, want passed", payload.Status)
	}
	if len(payload.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(payload.Steps))
	}
	if payload.Steps[0].Step != "php-unit-tests" || payload.Steps[0].Label != "PHP unit tests" {
		t.Errorf("first step = %+v, want slug and label", payload.Steps[0])
	}
	if payload.Steps[1].Step != "docs-build" || payload.Steps[1].Status != ciStatusPassed {
		t.Errorf("second step = %+v, want passed docs-build", payload.Steps[1])
	}
}

func TestCIModeSummaryCanBeDisabled(t *testing.T) {
	input := `
version: 2.0

task "ci" mode "ci":
  step "PHP unit tests"
  run "echo ok"
`

	out, err := executeCI(t, input, "ci", WithCISummary("off"))
	if err != nil {
		t.Fatalf("Execution failed: %v\n%s", err, out)
	}

	if strings.Contains(out, "CI STEP SUMMARY") || strings.Contains(out, "CI_STATUS=") {
		t.Errorf("expected no summary when disabled, got:\n%s", out)
	}
}

func TestNormalModeHasNoStepSummary(t *testing.T) {
	input := `
version: 2.0

task "build":
  step "Compile"
  run "echo ok"
`

	out, err := executeCI(t, input, "build")
	if err != nil {
		t.Fatalf("Execution failed: %v\n%s", err, out)
	}

	if strings.Contains(out, "CI STEP SUMMARY") || strings.Contains(out, "CI_STATUS=") {
		t.Errorf("normal mode should not print a ci summary, got:\n%s", out)
	}
}

func TestCIStepSlug(t *testing.T) {
	tests := map[string]string{
		"PHP unit tests":                     "php-unit-tests",
		"  Docs build  ":                     "docs-build",
		"test-php":                           "test-php",
		"Vulnerability checks (govulncheck)": "vulnerability-checks-govulncheck",
		"Already_underscored":                "already-underscored",
		"":                                   "step",
		"###":                                "step",
		"Rélease-check":                      "r-lease-check",
		"multi\nline\nstep":                  "multi-line-step",
	}

	for input, want := range tests {
		if got := ciStepSlug(input); got != want {
			t.Errorf("ciStepSlug(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeCISummaryFormat(t *testing.T) {
	valid := map[string]CISummaryFormat{
		"":     CISummaryText,
		"text": CISummaryText,
		"json": CISummaryJSON,
		"off":  CISummaryOff,
		"Off":  CISummaryOff,
	}
	for input, want := range valid {
		got, err := NormalizeCISummaryFormat(input)
		if err != nil {
			t.Errorf("NormalizeCISummaryFormat(%q) error = %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizeCISummaryFormat(%q) = %q, want %q", input, got, want)
		}
	}

	// Surrounding whitespace and case are normalized.
	if got, err := NormalizeCISummaryFormat(" JSON "); err != nil || got != CISummaryJSON {
		t.Errorf("NormalizeCISummaryFormat(whitespace) = %q, %v; want json", got, err)
	}

	if _, err := NormalizeCISummaryFormat("verbose"); err == nil {
		t.Error("expected an error for an unsupported format")
	}
}
