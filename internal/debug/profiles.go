package debug

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)

// GoroutineLeakProfile is the name of the runtime profile that reports
// goroutines which are blocked forever and can never make progress. Go 1.27
// graduates it from an experiment to a standard profile, so it is always
// available here.
const GoroutineLeakProfile = "goroutineleak"

// ErrGoroutineLeakProfileUnavailable reports that the running Go runtime does
// not expose the goroutineleak profile.
var ErrGoroutineLeakProfileUnavailable = errors.New("the goroutineleak profile is unavailable on this Go runtime")

// CaptureGoroutineLeakProfile writes the runtime goroutineleak profile to w.
// A verbosity of zero writes the binary pprof format that "go tool pprof"
// reads; one or more writes the text form, which lists each blocked goroutine
// together with the pprof labels Go 1.27 records for it.
func CaptureGoroutineLeakProfile(w io.Writer, verbosity int) error {
	if w == nil {
		return errors.New("no writer for the goroutineleak profile")
	}

	profile := pprof.Lookup(GoroutineLeakProfile)
	if profile == nil {
		return ErrGoroutineLeakProfileUnavailable
	}

	if err := profile.WriteTo(w, verbosity); err != nil {
		return fmt.Errorf("writing the goroutineleak profile: %w", err)
	}
	return nil
}

// WriteGoroutineLeakProfile writes the binary goroutineleak profile to a
// timestamped file in dir and returns its path, ready for "go tool pprof".
// Nothing is written until a caller asks for it, so the profile costs nothing
// while debug capture is off.
func WriteGoroutineLeakProfile(dir string) (string, error) {
	path := filepath.Join(dir, fmt.Sprintf("drun-goroutineleak-%s.pprof", time.Now().Format("20060102-150405")))

	// #nosec G304 -- the profile path is a timestamped file in a directory the caller fixed.
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("creating the goroutineleak profile: %w", err)
	}
	defer func() { _ = file.Close() }()

	if err := CaptureGoroutineLeakProfile(file, 0); err != nil {
		return "", err
	}
	return path, nil
}
