package errors

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/lexer"
)

// ParseError represents a parsing error with position information
type ParseError struct {
	Message  string
	Token    lexer.Token
	Filename string
	Source   string // The original source code
}

// Error implements the error interface
func (e *ParseError) Error() string {
	return e.Message
}

// FormatError formats a parse error with file location and visual indicator
func (e *ParseError) FormatError() string {
	var result strings.Builder

	// Write the main error message with file:line:column
	result.WriteString(fmt.Sprintf("\033[31mError\033[0m: %s\n", e.Message))
	result.WriteString(fmt.Sprintf("  \033[36m--> %s:%d:%d\033[0m\n", e.Filename, e.Token.Line, e.Token.Column))

	// Get the source line
	lines := strings.Split(e.Source, "\n")
	if e.Token.Line > 0 && e.Token.Line <= len(lines) {
		sourceLine := lines[e.Token.Line-1]
		lineNumStr := fmt.Sprintf("%d", e.Token.Line)

		// Show the line with line number
		result.WriteString(fmt.Sprintf("   \033[34m%s\033[0m | %s\n", lineNumStr, sourceLine))

		// Show the caret pointing to the error position
		spaces := strings.Repeat(" ", len(lineNumStr)) + " | " + strings.Repeat(" ", e.Token.Column-1)
		result.WriteString(fmt.Sprintf("   %s\033[31m^\033[0m\n", spaces))

		// Add helpful suggestions for common errors
		suggestion := e.getSuggestion()
		if suggestion != "" {
			result.WriteString(fmt.Sprintf("   \033[33mHelp:\033[0m %s\n", suggestion))
		}
	}

	return result.String()
}

// getSuggestion returns a helpful suggestion for common parsing errors
func (e *ParseError) getSuggestion() string {
	msg := strings.ToLower(e.Message)

	if strings.Contains(msg, "expected next token to be colon") {
		return "Add a ':' at the end of the line"
	}

	if strings.Contains(msg, "expected version statement") {
		return "Start your file with 'version: 2.0'"
	}

	if strings.Contains(msg, "unexpected token") && strings.Contains(msg, "string") {
		return "Check for missing colons or incorrect syntax"
	}

	if strings.Contains(msg, "expected 'file'") {
		return "Use 'create file \"path\"' or 'create directory \"path\"'"
	}

	return ""
}

// ParseErrorList represents a collection of parse errors
type ParseErrorList struct {
	Errors   []*ParseError
	Filename string
	Source   string
}

// Error implements the error interface
func (el *ParseErrorList) Error() string {
	if len(el.Errors) == 0 {
		return "no errors"
	}

	if len(el.Errors) == 1 {
		return el.Errors[0].Error()
	}

	var messages []string
	for _, err := range el.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// FormatErrors formats all parse errors with visual indicators
func (el *ParseErrorList) FormatErrors() string {
	if len(el.Errors) == 0 {
		return ""
	}

	var result strings.Builder

	// Limit the number of errors shown to avoid overwhelming the user
	maxErrors := 3
	errorsToShow := el.Errors
	if len(errorsToShow) > maxErrors {
		errorsToShow = errorsToShow[:maxErrors]
	}

	// Header
	if len(el.Errors) == 1 {
		result.WriteString("Parse error:\n\n")
	} else if len(el.Errors) <= maxErrors {
		result.WriteString(fmt.Sprintf("Parse errors (%d):\n\n", len(el.Errors)))
	} else {
		result.WriteString(fmt.Sprintf("Parse errors (showing first %d of %d):\n\n", maxErrors, len(el.Errors)))
	}

	// Format each error
	for i, err := range errorsToShow {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(err.FormatError())
	}

	// Show helpful hint if there were more errors
	if len(el.Errors) > maxErrors {
		result.WriteString(fmt.Sprintf("\n\033[33mNote:\033[0m %d additional errors not shown. Fix the above errors first.\n", len(el.Errors)-maxErrors))
	}

	return result.String()
}

// NewParseError creates a new parse error
func NewParseError(message string, token lexer.Token, filename, source string) *ParseError {
	return &ParseError{
		Message:  message,
		Token:    token,
		Filename: filename,
		Source:   source,
	}
}

// NewParseErrorList creates a new parse error list
func NewParseErrorList(filename, source string) *ParseErrorList {
	return &ParseErrorList{
		Errors:   make([]*ParseError, 0),
		Filename: filename,
		Source:   source,
	}
}

// Add adds a parse error to the list
func (el *ParseErrorList) Add(message string, token lexer.Token) {
	err := NewParseError(message, token, el.Filename, el.Source)
	el.Errors = append(el.Errors, err)
}

// HasErrors returns true if there are any errors
func (el *ParseErrorList) HasErrors() bool {
	return len(el.Errors) > 0
}

// ParameterValidationError represents a parameter validation error that should not show usage
type ParameterValidationError struct {
	Message string
}

// Error implements the error interface
func (e *ParameterValidationError) Error() string {
	return e.Message
}

// NewParameterValidationError creates a new parameter validation error
func NewParameterValidationError(message string) *ParameterValidationError {
	return &ParameterValidationError{
		Message: message,
	}
}
