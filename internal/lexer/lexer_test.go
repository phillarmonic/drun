package lexer

import (
	"testing"
)

func TestLexer_HelloWorld(t *testing.T) {
	input := `# Hello World - Your First drun v2 Task
# This demonstrates the most basic semantic syntax

version: 2.0

task "hello":
  info "Hello from drun v2! 👋"

task "hello world":
  step "Starting hello world example"
  info "Welcome to the semantic task runner!"
  success "Hello world completed successfully!"`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{COMMENT, "# Hello World - Your First drun v2 Task"},
		{COMMENT, "# This demonstrates the most basic semantic syntax"},
		{VERSION, "version"},
		{COLON, ":"},
		{NUMBER, "2.0"},
		{TASK, "task"},
		{STRING, "hello"},
		{COLON, ":"},
		{INDENT, ""},
		{INFO, "info"},
		{STRING, "Hello from drun v2! 👋"},
		{DEDENT, ""},
		{TASK, "task"},
		{STRING, "hello world"},
		{COLON, ":"},
		{INDENT, ""},
		{STEP, "step"},
		{STRING, "Starting hello world example"},
		{INFO, "info"},
		{STRING, "Welcome to the semantic task runner!"},
		{SUCCESS, "success"},
		{STRING, "Hello world completed successfully!"},
		{DEDENT, ""},
		{EOF, ""},
	}

	for i, expected := range expectedTokens {
		tok := lexer.NextToken()

		if tok.Type != expected.expectedType {
			t.Fatalf("test[%d] - tokentype wrong. expected=%q, got=%q (literal: %q)",
				i, expected.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != expected.expectedLiteral {
			t.Fatalf("test[%d] - literal wrong. expected=%q, got=%q",
				i, expected.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_BasicTokens(t *testing.T) {
	input := `version: 2.0
task "test":
  info "message"`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{VERSION, "version"},
		{COLON, ":"},
		{NUMBER, "2.0"},
		{TASK, "task"},
		{STRING, "test"},
		{COLON, ":"},
		{INDENT, ""},
		{INFO, "info"},
		{STRING, "message"},
		{DEDENT, ""},
		{EOF, ""},
	}

	lexer := NewLexer(input)

	for i, tt := range tests {
		tok := lexer.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_Keywords(t *testing.T) {
	input := `version task means info step warn error success fail true false`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{VERSION, "version"},
		{TASK, "task"},
		{MEANS, "means"},
		{INFO, "info"},
		{STEP, "step"},
		{WARN, "warn"},
		{ERROR, "error"},
		{SUCCESS, "success"},
		{FAIL, "fail"},
		{BOOLEAN, "true"},
		{BOOLEAN, "false"},
		{EOF, ""},
	}

	lexer := NewLexer(input)

	for i, tt := range tests {
		tok := lexer.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_Indentation(t *testing.T) {
	input := `task "test":
  info "level 1"
    info "level 2"
  info "back to level 1"
info "level 0"`

	tests := []struct {
		expectedType TokenType
		description  string
	}{
		{TASK, "task keyword"},
		{STRING, "task name"},
		{COLON, "colon"},
		{INDENT, "indent to level 1"},
		{INFO, "info at level 1"},
		{STRING, "message"},
		{INDENT, "indent to level 2"},
		{INFO, "info at level 2"},
		{STRING, "message"},
		{DEDENT, "dedent to level 1"},
		{INFO, "info back at level 1"},
		{STRING, "message"},
		{DEDENT, "dedent to level 0"},
		{INFO, "info at level 0"},
		{STRING, "message"},
		{EOF, "end of file"},
	}

	lexer := NewLexer(input)

	for i, tt := range tests {
		tok := lexer.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] (%s) - tokentype wrong. expected=%q, got=%q",
				i, tt.description, tt.expectedType, tok.Type)
		}
	}
}

func TestLexer_EmptyKeyword(t *testing.T) {
	input := `version: 2.0

task "test":
  given $features as list defaults to empty
  given $name defaults to ""
  
  if $features is empty:
    info "Features is empty"
    
  if $features is not empty:
    info "Features: {$features}"`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{VERSION, "version"},
		{COLON, ":"},
		{NUMBER, "2.0"},
		{TASK, "task"},
		{STRING, "test"},
		{COLON, ":"},
		{INDENT, ""},
		{GIVEN, "given"},
		{VARIABLE, "$features"},
		{AS, "as"},
		{LIST, "list"},
		{DEFAULTS, "defaults"},
		{TO, "to"},
		{EMPTY, "empty"}, // Test that empty is tokenized as EMPTY
		{GIVEN, "given"},
		{VARIABLE, "$name"},
		{DEFAULTS, "defaults"},
		{TO, "to"},
		{STRING, ""},
		{IF, "if"},
		{VARIABLE, "$features"},
		{IS, "is"},
		{EMPTY, "empty"}, // Test that empty is tokenized as EMPTY in conditions
		{COLON, ":"},
		{INDENT, ""},
		{INFO, "info"},
		{STRING, "Features is empty"},
		{DEDENT, ""},
		{IF, "if"},
		{VARIABLE, "$features"},
		{IS, "is"},
		{IDENT, "not"},
		{EMPTY, "empty"}, // Test that empty is tokenized as EMPTY in "is not empty"
		{COLON, ":"},
		{INDENT, ""},
		{INFO, "info"},
		{STRING, "Features: {$features}"},
		{DEDENT, ""},
		{DEDENT, ""},
		{EOF, ""},
	}

	for i, tt := range expectedTokens {
		tok := lexer.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal: %q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_MultilineComments(t *testing.T) {
	input := `/*
    Drun Lifecycle Hooks Example
    This example demonstrates the new tool-level lifecycle hooks
    that run once at drun startup and shutdown
*/

version: 2.0

/* Another multiline comment */
task "test":
	info "Hello World"`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{MULTILINE_COMMENT, "/*\n    Drun Lifecycle Hooks Example\n    This example demonstrates the new tool-level lifecycle hooks\n    that run once at drun startup and shutdown\n*/"},
		{VERSION, "version"},
		{COLON, ":"},
		{NUMBER, "2.0"},
		{MULTILINE_COMMENT, "/* Another multiline comment */"},
		{TASK, "task"},
		{STRING, "test"},
		{COLON, ":"},
		{INDENT, ""},
		{INFO, "info"},
		{STRING, "Hello World"},
		{DEDENT, ""},
		{EOF, ""},
	}

	for i, tt := range expectedTokens {
		tok := lexer.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal: %q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_UnterminatedMultilineComment(t *testing.T) {
	input := `/*
    This comment is not terminated
    
version: 2.0`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{MULTILINE_COMMENT, "/*\n    This comment is not terminated\n    \nversion: 2.0"},
		{EOF, ""},
	}

	for i, tt := range expectedTokens {
		tok := lexer.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal: %q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}
