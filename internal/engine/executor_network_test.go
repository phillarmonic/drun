package engine

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

// startTestListener opens a TCP listener on 127.0.0.1 with an ephemeral port
// and returns the port number. The listener is closed when the test ends.
func startTestListener(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start test listener: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	// Accept connections in the background so probes complete
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	return fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
}

// unusedTestPort returns a port that was free at call time (listener closed
// immediately), so a dial against it is refused.
func unusedTestPort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to allocate test port: %v", err)
	}
	port := fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	_ = listener.Close()
	return port
}

func TestPortCheckEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		port        string
		expected    string
		expectError bool
	}{
		{name: "host and port", target: "localhost", port: "5432", expected: "localhost:5432"},
		{name: "ipv4 and port", target: "127.0.0.1", port: "8080", expected: "127.0.0.1:8080"},
		{name: "target already has port", target: "db.internal:5432", port: "", expected: "db.internal:5432"},
		{name: "empty host", target: "", port: "5432", expectError: true},
		{name: "no port anywhere", target: "localhost", port: "", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint, err := portCheckEndpoint(tt.target, tt.port)
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error, got endpoint %q", endpoint)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if endpoint != tt.expected {
				t.Errorf("Expected endpoint %q, got %q", tt.expected, endpoint)
			}
		})
	}
}

func TestParsePortCheckTimeout(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		expected    time.Duration
		expectError bool
	}{
		{name: "empty uses default", raw: "", expected: defaultPortCheckTimeout},
		{name: "go duration", raw: "2s", expected: 2 * time.Second},
		{name: "milliseconds", raw: "500ms", expected: 500 * time.Millisecond},
		{name: "bare seconds", raw: "10", expected: 10 * time.Second},
		{name: "invalid", raw: "soon", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout, err := parsePortCheckTimeout(tt.raw)
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error, got timeout %v", timeout)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if timeout != tt.expected {
				t.Errorf("Expected timeout %v, got %v", tt.expected, timeout)
			}
		})
	}
}

func TestPortCheckAction(t *testing.T) {
	openPort := startTestListener(t)
	closedPort := unusedTestPort(t)

	tests := []struct {
		name        string
		port        string
		expectError bool
	}{
		{name: "open port succeeds", port: openPort, expectError: false},
		{name: "closed port fails", port: closedPort, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := fmt.Sprintf(`version: 2.0

task "probe":
  test connection to "127.0.0.1" on port %s timeout "1s"`, tt.port)

			l := lexer.NewLexer(input)
			p := parser.NewParser(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			var output bytes.Buffer
			e := NewEngine(&output)
			err := e.ExecuteWithParams(program, "probe", nil)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected closed port to fail, but succeeded. Output:\n%s", output.String())
				}
				if !strings.Contains(output.String(), "Port check failed") {
					t.Errorf("Expected failure output, got:\n%s", output.String())
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected open port to succeed, got: %v", err)
			}
			if !strings.Contains(output.String(), "Port is open") {
				t.Errorf("Expected success output, got:\n%s", output.String())
			}
		})
	}
}

func TestPortCheckActionDryRun(t *testing.T) {
	closedPort := unusedTestPort(t)

	input := fmt.Sprintf(`version: 2.0

task "probe":
  test connection to "127.0.0.1" on port %s timeout "1s"`, closedPort)

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var output bytes.Buffer
	e := NewEngine(&output)
	e.SetDryRun(true)

	// Dry run must not dial: a closed port should not fail the task
	if err := e.ExecuteWithParams(program, "probe", nil); err != nil {
		t.Fatalf("Dry run should not fail on closed port, got: %v", err)
	}
	if !strings.Contains(output.String(), "[DRY RUN] Would probe TCP port") {
		t.Errorf("Expected dry-run output, got:\n%s", output.String())
	}
}

func TestPortCondition(t *testing.T) {
	openPort := startTestListener(t)
	closedPort := unusedTestPort(t)

	tests := []struct {
		name           string
		condition      string
		expectedOutput []string
		notExpected    []string
	}{
		{
			name:           "open port takes if branch",
			condition:      fmt.Sprintf("port %s is open on \"127.0.0.1\"", openPort),
			expectedOutput: []string{"branch: taken"},
			notExpected:    []string{"branch: skipped"},
		},
		{
			name:           "closed port takes else branch",
			condition:      fmt.Sprintf("port %s is open on \"127.0.0.1\"", closedPort),
			expectedOutput: []string{"branch: skipped"},
			notExpected:    []string{"branch: taken"},
		},
		{
			name:           "not open is true for closed port",
			condition:      fmt.Sprintf("port %s is not open on \"127.0.0.1\"", closedPort),
			expectedOutput: []string{"branch: taken"},
			notExpected:    []string{"branch: skipped"},
		},
		{
			name:           "not open is false for open port",
			condition:      fmt.Sprintf("port %s is not open on \"127.0.0.1\"", openPort),
			expectedOutput: []string{"branch: skipped"},
			notExpected:    []string{"branch: taken"},
		},
		{
			name:           "with timeout option",
			condition:      fmt.Sprintf("port %s is open on \"127.0.0.1\" with timeout \"1s\"", openPort),
			expectedOutput: []string{"branch: taken"},
			notExpected:    []string{"branch: skipped"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := fmt.Sprintf(`version: 2.0

task "probe":
  if %s:
    info "branch: taken"
  else:
    info "branch: skipped"`, tt.condition)

			l := lexer.NewLexer(input)
			p := parser.NewParser(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			var output bytes.Buffer
			e := NewEngine(&output)
			if err := e.ExecuteWithParams(program, "probe", nil); err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			outputStr := output.String()
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, outputStr)
				}
			}
			for _, notExpected := range tt.notExpected {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output to NOT contain %q, but got:\n%s", notExpected, outputStr)
				}
			}
		})
	}
}

func TestPortConditionInterpolation(t *testing.T) {
	openPort := startTestListener(t)

	input := fmt.Sprintf(`version: 2.0

task "probe":
  given $db_port defaults to "%s"
  given $db_host defaults to "127.0.0.1"
  if port {$db_port} is open on "{$db_host}":
    info "database is up"
  else:
    info "database is down"`, openPort)

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var output bytes.Buffer
	e := NewEngine(&output)
	if err := e.ExecuteWithParams(program, "probe", nil); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if !strings.Contains(output.String(), "database is up") {
		t.Errorf("Expected interpolated port condition to be true, got:\n%s", output.String())
	}
}

func TestPortConditionDryRun(t *testing.T) {
	openPort := startTestListener(t)

	input := fmt.Sprintf(`version: 2.0

task "probe":
  if port %s is open on "127.0.0.1":
    info "branch: taken"
  else:
    info "branch: skipped"`, openPort)

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var output bytes.Buffer
	e := NewEngine(&output)
	e.SetDryRun(true)

	// Dry run must not dial: even an open port evaluates to closed
	if err := e.ExecuteWithParams(program, "probe", nil); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}
	if !strings.Contains(output.String(), "branch: skipped") {
		t.Errorf("Expected port condition to be false in dry run, got:\n%s", output.String())
	}
}
