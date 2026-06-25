package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngineExecuteSelectsPlatformVariant(t *testing.T) {
	program, err := ParseString(`version: 2.0

@platform("linux")
task "shell":
  info "linux shell"

@platform("mac")
task "shell":
  info "mac shell"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)
	if err := engine.Execute(program, "shell"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	if platform := currentPlatformLabel(); platform == "mac" {
		if !strings.Contains(output, "mac shell") {
			t.Fatalf("expected mac output, got %q", output)
		}
	} else {
		if !strings.Contains(output, "linux shell") {
			t.Fatalf("expected linux output, got %q", output)
		}
	}
}

func TestEngineTaskCallSelectsPlatformVariant(t *testing.T) {
	program, err := ParseString(`version: 2.0

task "entry":
  call task "shell"

@platform("linux")
task "shell":
  info "linux shell"

@platform("mac")
task "shell":
  info "mac shell"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)
	if err := engine.Execute(program, "entry"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	if platform := currentPlatformLabel(); platform == "mac" {
		if !strings.Contains(output, "mac shell") {
			t.Fatalf("expected mac output, got %q", output)
		}
	} else {
		if !strings.Contains(output, "linux shell") {
			t.Fatalf("expected linux output, got %q", output)
		}
	}
}

func currentPlatformLabel() string {
	return currentPlatformForTest
}
