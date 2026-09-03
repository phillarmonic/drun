package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

// runTaskWithParams parses a full script and runs taskName with the given
// parameters, returning the captured output.
func runTaskWithParams(t *testing.T, input, taskName string, params map[string]string) (string, error) {
	t.Helper()

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err := engine.ExecuteWithParams(program, taskName, params)
	return output.String(), err
}

// Regression: `accepts $items as list of strings` parses (parser_parameter.go
// builds DataType "list of strings") but validateDataType only knew bare
// "list", so binding a value failed at runtime with
// "unknown data type: list of strings". A list of strings must bind cleanly.
func TestListOfStringsParameterBinds(t *testing.T) {
	input := `version: 2.0

task "report":
	accepts $items as list of strings
	info "Binding accepted"
`

	output, err := runTaskWithParams(t, input, "report", map[string]string{"items": "alpha,beta"})
	if err != nil {
		t.Fatalf("Execution error: %v\noutput:\n%s", err, output)
	}

	assertOutputContains(t, output, "Binding accepted")
}

// A scalar (non-list) value must be rejected for a list-of-strings parameter.
func TestListOfStringsRejectsScalarValue(t *testing.T) {
	input := `version: 2.0

task "report":
	accepts $items as list of strings
	info "unreachable"
`

	_, err := runTaskWithParams(t, input, "report", map[string]string{"items": "alpha"})
	if err == nil {
		t.Fatal("Expected validation error for scalar value, got nil")
	}
	if !strings.Contains(err.Error(), "must be a list") {
		t.Fatalf("Expected 'must be a list' error, got: %v", err)
	}
}

// A list-of-numbers parameter must accept a list of numbers.
func TestListOfNumbersParameterBinds(t *testing.T) {
	input := `version: 2.0

task "check":
	accepts $ports as list of numbers
	info "Ports accepted"
`

	output, err := runTaskWithParams(t, input, "check", map[string]string{"ports": "80,443,8080"})
	if err != nil {
		t.Fatalf("Execution error: %v\noutput:\n%s", err, output)
	}

	assertOutputContains(t, output, "Ports accepted")
}

// A list-of-numbers parameter must reject a list whose elements are not
// numbers, naming the offending element.
func TestListOfNumbersRejectsBadElements(t *testing.T) {
	input := `version: 2.0

task "check":
	accepts $ports as list of numbers
	info "unreachable"
`

	_, err := runTaskWithParams(t, input, "check", map[string]string{"ports": "80,https"})
	if err == nil {
		t.Fatal("Expected validation error for non-numeric element, got nil")
	}
	if !strings.Contains(err.Error(), "must be a number") {
		t.Fatalf("Expected element validation error, got: %v", err)
	}
}
