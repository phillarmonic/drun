package engine

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"golang.org/x/term"
)

// Domain: Interactive Input Execution
// This file contains the executors for interactive input statements:
//
//	confirm "<question>" [defaults to yes|no] [as $var]
//	prompt  "<question>" [defaults to <value>] as $var
//
// Both interpolate {$var} in the question and the default value. In a
// non-interactive environment (no TTY, or CI) they never block: resolution
// falls back to the declared default, then to the global --yes/--no
// assumption, then to a clear error.

// errUserAborted is returned by a bare confirm gate whose answer was no.
// The task runner treats it as a graceful stop (it prints a notice and ends
// the run successfully with exit code 0) rather than as a task failure.
var errUserAborted = errors.New("confirmation declined")

// canPromptInteractively reports whether an interactive statement may block
// reading an answer from a real terminal: the run must not be a dry run and
// must not be executing inside a CI environment, and e.input must be an
// *os.File connected to a TTY.
func (e *Engine) canPromptInteractively() bool {
	if e.dryRun || isCIEnvironment() {
		return false
	}
	file, ok := e.input.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

// readInteractiveLine reads a single line from e.input, trimming the trailing
// newline. An end-of-input that yields no text behaves like an empty line so
// callers fall back to their default answer.
func (e *Engine) readInteractiveLine() (string, error) {
	line, err := bufio.NewReader(e.input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// parseYesNo parses a confirm answer (y/yes/n/no, or the "true"/"false" and
// 1/0 spellings) case-insensitively.
func parseYesNo(text string) (answer, ok bool) {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "y", "yes", "true", "1":
		return true, true
	case "n", "no", "false", "0":
		return false, true
	}
	return false, false
}

// executeConfirm executes:
//
//	confirm "<question>" [defaults to yes|no] [as $var]
//
// The bare form is a gate: declining (answering no) returns errUserAborted so
// the task stops gracefully. With `as $var` the answer is stored using drun's
// string-boolean convention ("true"/"false") and never aborts the task.
func (e *Engine) executeConfirm(stmt *statement.Confirm, ctx *ExecutionContext) error {
	question, err := e.interpolateVariablesWithError(stmt.Question, ctx)
	if err != nil {
		return fmt.Errorf("confirm: %w", err)
	}

	var defaultAnswer bool
	if stmt.HasDefault {
		defaultText, err := e.interpolateVariablesWithError(stmt.DefaultValue, ctx)
		if err != nil {
			return fmt.Errorf("confirm: %w", err)
		}
		parsed, ok := parseYesNo(defaultText)
		if !ok {
			return fmt.Errorf("confirm %q: invalid default answer %q — use yes or no", question, defaultText)
		}
		defaultAnswer = parsed
	}

	// Dry run never blocks: report what would be asked and resolve to the
	// declared default, then the --yes/--no assumption, then yes.
	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] would ask: %q\n", question)
		answer := defaultAnswer
		if !stmt.HasDefault {
			answer = true
			if e.assumeYes != nil {
				answer = *e.assumeYes
			}
		}
		return e.finishConfirm(stmt, ctx, answer)
	}

	// A global --yes/--no assumption short-circuits even on a TTY.
	if e.assumeYes != nil {
		return e.finishConfirm(stmt, ctx, *e.assumeYes)
	}

	if e.canPromptInteractively() {
		answer, err := e.askConfirm(question, defaultAnswer)
		if err != nil {
			return err
		}
		return e.finishConfirm(stmt, ctx, answer)
	}

	// Non-interactive (no TTY / CI): use the declared default, else fail with
	// a clear error — never block waiting for input.
	if stmt.HasDefault {
		return e.finishConfirm(stmt, ctx, defaultAnswer)
	}
	return fmt.Errorf("confirm %q: no interactive terminal and no default answer; "+
		"add `defaults to yes|no` or pass --yes/--no", question)
}

// askConfirm renders the y/n prompt (marking the default) and reads the
// answer from e.input, re-asking until y/yes/n/no or an empty line is given.
// An empty line resolves to the default answer shown in the prompt.
func (e *Engine) askConfirm(question string, defaultAnswer bool) (bool, error) {
	marker := "y/N"
	if defaultAnswer {
		marker = "Y/n"
	}
	_, _ = fmt.Fprintf(e.output, "❓ %s [%s] ", question, marker)

	for {
		line, err := e.readInteractiveLine()
		if err != nil {
			return false, err
		}
		if strings.TrimSpace(line) == "" {
			return defaultAnswer, nil
		}
		if answer, ok := parseYesNo(line); ok {
			return answer, nil
		}
		_, _ = fmt.Fprintf(e.output, "    Please answer yes or no [%s] ", marker)
	}
}

// finishConfirm stores the answer for `as $var` confirmations or returns
// errUserAborted when a bare gate is declined.
func (e *Engine) finishConfirm(stmt *statement.Confirm, ctx *ExecutionContext, answer bool) error {
	if stmt.ResultVar != "" {
		ctx.Variables[stmt.ResultVar] = strconv.FormatBool(answer)
		return nil
	}
	if !answer {
		return errUserAborted
	}
	return nil
}

// executePrompt executes:
//
//	prompt "<question>" [defaults to <value>] as $var
//
// The raw typed answer (or the default) is stored in the result variable.
func (e *Engine) executePrompt(stmt *statement.Prompt, ctx *ExecutionContext) error {
	if stmt.ResultVar == "" {
		return errors.New("prompt: a result variable is required — use `prompt \"<question>\" as $var`")
	}

	question, err := e.interpolateVariablesWithError(stmt.Question, ctx)
	if err != nil {
		return fmt.Errorf("prompt: %w", err)
	}

	defaultText := ""
	if stmt.HasDefault {
		defaultText, err = e.interpolateVariablesWithError(stmt.DefaultValue, ctx)
		if err != nil {
			return fmt.Errorf("prompt: %w", err)
		}
	}

	// Dry run never blocks: report what would be asked and resolve to the
	// declared default (an empty answer when none is declared).
	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] would ask: %q\n", question)
		return e.finishPrompt(stmt, ctx, defaultText)
	}

	// A global --yes/--no assumption short-circuits even on a TTY: accept the
	// declared default, or an empty answer when none is declared.
	if e.assumeYes != nil {
		return e.finishPrompt(stmt, ctx, defaultText)
	}

	if e.canPromptInteractively() {
		_, _ = fmt.Fprintf(e.output, "❓ %s", question)
		if stmt.HasDefault {
			_, _ = fmt.Fprintf(e.output, " [%s]", defaultText)
		}
		_, _ = fmt.Fprint(e.output, ": ")
		line, err := e.readInteractiveLine()
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" && stmt.HasDefault {
			line = defaultText
		}
		return e.finishPrompt(stmt, ctx, line)
	}

	// Non-interactive (no TTY / CI): use the declared default, else fail with
	// a clear error — never block waiting for input.
	if stmt.HasDefault {
		return e.finishPrompt(stmt, ctx, defaultText)
	}
	return fmt.Errorf("prompt %q: no interactive terminal and no default answer; "+
		"add `defaults to ...` or pass --yes/--no", question)
}

// finishPrompt stores the resolved answer into the result variable.
func (e *Engine) finishPrompt(stmt *statement.Prompt, ctx *ExecutionContext, answer string) error {
	ctx.Variables[stmt.ResultVar] = answer
	return nil
}
