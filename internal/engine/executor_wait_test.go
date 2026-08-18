package engine

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestEngine_WaitSleepsForDuration(t *testing.T) {
	program, err := ParseString(`version: 2.0

task "demo":
  wait 0.1 seconds
  success "done"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)

	start := time.Now()
	if err := engine.Execute(program, "demo"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Errorf("expected execution to wait at least 100ms, took %v", elapsed)
	}

	if !strings.Contains(out.String(), "Waiting 100ms") {
		t.Errorf("expected wait output, got: %q", out.String())
	}
}

func TestEngine_WaitInterpolatesVariable(t *testing.T) {
	program, err := ParseString(`version: 2.0

task "demo":
  let $pause_len = "0.05"
  wait {$pause_len} seconds
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)

	start := time.Now()
	if err := engine.Execute(program, "demo"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Errorf("expected execution to wait at least 50ms, took %v", elapsed)
	}
}

func TestEngine_WaitRejectsNonNumericValue(t *testing.T) {
	program, err := ParseString(`version: 2.0

task "demo":
  let $pause_len = "abc"
  wait {$pause_len} seconds
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)

	err = engine.Execute(program, "demo")
	if err == nil {
		t.Fatal("expected error for non-numeric wait duration, got nil")
	}
	if !strings.Contains(err.Error(), "invalid wait duration") {
		t.Errorf("expected invalid duration error, got: %v", err)
	}
}
