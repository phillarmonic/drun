package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// CISummaryFormat selects how a ci-mode run renders its per-step summary.
type CISummaryFormat string

const (
	// CISummaryText renders one `STEP=<slug> STATUS=<status>` line per step,
	// followed by a final `CI_STATUS=<status>` line. This is the default.
	CISummaryText CISummaryFormat = "text"
	// CISummaryJSON renders the same information as a single JSON object on
	// one line, instead of key=value lines.
	CISummaryJSON CISummaryFormat = "json"
	// CISummaryOff disables the summary entirely, leaving only the step banners
	// and any buffered failure output.
	CISummaryOff CISummaryFormat = "off"
)

// Verdict values used by the summary.
const (
	ciStatusPassed = "passed"
	ciStatusFailed = "failed"
)

// NormalizeCISummaryFormat validates a user-supplied ci summary format. The
// empty string selects the default (text).
func NormalizeCISummaryFormat(format string) (CISummaryFormat, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", string(CISummaryText):
		return CISummaryText, nil
	case string(CISummaryJSON):
		return CISummaryJSON, nil
	case string(CISummaryOff):
		return CISummaryOff, nil
	default:
		return "", fmt.Errorf("unsupported ci summary format %q (supported: text, json, off)", format)
	}
}

// normalizeCISummaryFormatOrDefault resolves a summary format, falling back to
// the default when the value is unrecognized.
func normalizeCISummaryFormatOrDefault(format string) CISummaryFormat {
	normalized, err := NormalizeCISummaryFormat(format)
	if err != nil {
		return CISummaryText
	}
	return normalized
}

// beginCIRun arms the per-step summary when the invoked task runs in ci mode.
// It is called once per Execute invocation; a run that is not a ci run leaves
// the ledger nil so no summary is printed.
func (e *Engine) beginCIRun(targetMode string) {
	e.ciLedger = nil
	if e.ciSummary == CISummaryOff || !strings.EqualFold(targetMode, "ci") {
		return
	}
	e.ciLedger = newCIStepLedger(e.ciSummary)
}

// recordCIStep records a step action when the active run is a ci run and the
// current task is still executing in ci mode.
func (e *Engine) recordCIStep(label string, ctx *ExecutionContext) {
	if e.ciLedger == nil || !isCIMode(ctx) {
		return
	}
	e.ciLedger.begin(label)
}

// finishCIRun flushes the in-flight step with the run's final verdict, renders
// the summary, and disarms the ledger.
func (e *Engine) finishCIRun(taskName, status string) {
	if e.ciLedger == nil {
		return
	}
	e.ciLedger.finish(status, time.Now())
	e.ciLedger.render(e.output, taskName, status)
	e.ciLedger = nil
}

// ciStep is one recorded step of a ci-mode run.
type ciStep struct {
	Step       string `json:"step"`
	Label      string `json:"label,omitempty"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
}

// ciSummaryJSON is the machine-readable shape of a ci summary.
type ciSummaryJSON struct {
	Task       string   `json:"task"`
	Status     string   `json:"status"`
	Steps      []ciStep `json:"steps"`
	DurationMS int64    `json:"duration_ms"`
}

// ciStepLedger accumulates the steps of a ci-mode run so the engine can print a
// compact, stable verdict for each one once the run finishes.
//
// A step has no explicit end: execution reaching the next step (or the end of
// the run) is what makes it pass. The ledger therefore marks the previous step
// passed when a new one begins, and marks the pending step with the run's final
// status when the run ends.
type ciStepLedger struct {
	started      time.Time
	currentStart time.Time
	format       CISummaryFormat
	currentLabel string
	steps        []ciStep
	mu           sync.Mutex
}

func newCIStepLedger(format CISummaryFormat) *ciStepLedger {
	return &ciStepLedger{format: format, started: time.Now()}
}

// begin records the start of a step. Any step already in flight is flushed as
// passed because execution advanced past it.
func (l *ciStepLedger) begin(label string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.flushLocked(now, ciStatusPassed)

	l.currentLabel = label
	l.currentStart = now
}

// finish flushes the in-flight step with the run's final verdict.
func (l *ciStepLedger) finish(status string, at time.Time) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	l.flushLocked(at, status)
}

func (l *ciStepLedger) flushLocked(at time.Time, status string) {
	if l.currentLabel == "" {
		return
	}

	slug := ciStepSlug(l.currentLabel)
	step := ciStep{
		Step:       slug,
		Status:     status,
		DurationMS: max(at.Sub(l.currentStart).Milliseconds(), 0),
	}
	if slug != l.currentLabel {
		step.Label = l.currentLabel
	}
	l.steps = append(l.steps, step)
	l.currentLabel = ""
}

func (l *ciStepLedger) snapshot() []ciStep {
	l.mu.Lock()
	defer l.mu.Unlock()

	return append([]ciStep(nil), l.steps...)
}

// render writes the summary. It is a no-op when the format is off or no step
// was recorded, so a ci task without steps keeps its previous output.
func (l *ciStepLedger) render(w io.Writer, task, status string) {
	if l == nil || w == nil || l.format == CISummaryOff {
		return
	}

	steps := l.snapshot()
	if len(steps) == 0 {
		return
	}

	totalMS := max(time.Since(l.started).Milliseconds(), 0)

	if l.format == CISummaryJSON {
		data, err := json.Marshal(ciSummaryJSON{
			Task:       task,
			Status:     status,
			DurationMS: totalMS,
			Steps:      steps,
		})
		if err != nil {
			return
		}
		_, _ = fmt.Fprintf(w, "%s\n", data)
		return
	}

	passed, failed := 0, 0
	for _, step := range steps {
		if step.Status == ciStatusPassed {
			passed++
		} else {
			failed++
		}
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "CI STEP SUMMARY")
	for _, step := range steps {
		_, _ = fmt.Fprintf(w, "STEP=%s STATUS=%s DURATION_MS=%d\n", step.Step, step.Status, step.DurationMS)
	}
	_, _ = fmt.Fprintf(w, "CI_STATUS=%s STEPS=%d PASSED=%d FAILED=%d DURATION_MS=%d\n",
		status, len(steps), passed, failed, totalMS)
}

// ciStepSlug turns a human step label into a stable, grep-friendly token.
func ciStepSlug(label string) string {
	var b strings.Builder

	lastDash := true // suppresses leading dashes
	for _, r := range strings.ToLower(label) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "step"
	}
	return slug
}

// isCIMode reports whether the current task executes in ci mode.
func isCIMode(ctx *ExecutionContext) bool {
	return ctx != nil && strings.EqualFold(ctx.CurrentTaskMode, "ci")
}
