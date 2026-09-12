package debug

import (
	"bytes"
	"context"
	"os"
	"runtime/pprof"
	"strings"
	"testing"
	"time"
)

func TestCaptureGoroutineLeakProfileText(t *testing.T) {
	var text bytes.Buffer
	if err := CaptureGoroutineLeakProfile(&text, 1); err != nil {
		t.Fatalf("capture text profile: %v", err)
	}
	if !strings.Contains(text.String(), "goroutineleak profile: total") {
		t.Fatalf("text profile does not look like the goroutineleak profile:\n%s", text.String())
	}
}

func TestCaptureGoroutineLeakProfileBinary(t *testing.T) {
	var binary bytes.Buffer
	if err := CaptureGoroutineLeakProfile(&binary, 0); err != nil {
		t.Fatalf("capture binary profile: %v", err)
	}
	if binary.Len() == 0 {
		t.Fatal("binary profile is empty")
	}
}

func TestCaptureGoroutineLeakProfileRejectsNilWriter(t *testing.T) {
	if err := CaptureGoroutineLeakProfile(nil, 1); err == nil {
		t.Fatal("expected an error for a nil writer")
	}
}

// TestWriteGoroutineLeakProfile proves the profile is capturable through drun's
// debug tooling without a crash, and that a labelled blocked goroutine shows up
// in it with its labels.
func TestWriteGoroutineLeakProfile(t *testing.T) {
	blocked := make(chan struct{})
	go func() {
		pprof.SetGoroutineLabels(pprof.WithLabels(context.Background(), pprof.Labels(
			"drun.component", "test-leaker",
			"drun.worker", "3",
		)))
		close(blocked)
		select {}
	}()
	<-blocked
	// The runtime only classifies a goroutine as leaked once it has settled.
	time.Sleep(200 * time.Millisecond)

	path, err := WriteGoroutineLeakProfile(t.TempDir())
	if err != nil {
		t.Fatalf("write profile: %v", err)
	}
	if !strings.HasSuffix(path, ".pprof") {
		t.Fatalf("profile path = %q, want a .pprof file", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat profile: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("profile file is empty")
	}

	var text bytes.Buffer
	if err := CaptureGoroutineLeakProfile(&text, 1); err != nil {
		t.Fatalf("capture text profile: %v", err)
	}
	for _, want := range []string{"drun.component", "test-leaker", "drun.worker"} {
		if !strings.Contains(text.String(), want) {
			t.Errorf("leak profile missing %q:\n%s", want, text.String())
		}
	}
}
