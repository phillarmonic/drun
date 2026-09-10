package lexer

import (
	"testing"
)

func TestLexer_GitEnsureKeyword(t *testing.T) {
	if token := LookupIdent("ensure"); token != ENSURE {
		t.Fatalf("LookupIdent(ensure) = %s, want ENSURE", token)
	}
}

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
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: COMMENT, expectedLiteral: "# Hello World - Your First drun v2 Task"},
		{expectedType: COMMENT, expectedLiteral: "# This demonstrates the most basic semantic syntax"},
		{expectedType: VERSION, expectedLiteral: "version"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: NUMBER, expectedLiteral: "2.0"},
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "hello"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "Hello from drun v2! 👋"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "hello world"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: STEP, expectedLiteral: "step"},
		{expectedType: STRING, expectedLiteral: "Starting hello world example"},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "Welcome to the semantic task runner!"},
		{expectedType: SUCCESS, expectedLiteral: "success"},
		{expectedType: STRING, expectedLiteral: "Hello world completed successfully!"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
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
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: VERSION, expectedLiteral: "version"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: NUMBER, expectedLiteral: "2.0"},
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "test"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "message"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
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
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: VERSION, expectedLiteral: "version"},
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: MEANS, expectedLiteral: "means"},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STEP, expectedLiteral: "step"},
		{expectedType: WARN, expectedLiteral: "warn"},
		{expectedType: ERROR, expectedLiteral: "error"},
		{expectedType: SUCCESS, expectedLiteral: "success"},
		{expectedType: FAIL, expectedLiteral: "fail"},
		{expectedType: BOOLEAN, expectedLiteral: "true"},
		{expectedType: BOOLEAN, expectedLiteral: "false"},
		{expectedType: EOF, expectedLiteral: ""},
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
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: TASK, expectedLiteral: "task keyword"},
		{expectedType: STRING, expectedLiteral: "task name"},
		{expectedType: COLON, expectedLiteral: "colon"},
		{expectedType: INDENT, expectedLiteral: "indent to level 1"},
		{expectedType: INFO, expectedLiteral: "info at level 1"},
		{expectedType: STRING, expectedLiteral: "message"},
		{expectedType: INDENT, expectedLiteral: "indent to level 2"},
		{expectedType: INFO, expectedLiteral: "info at level 2"},
		{expectedType: STRING, expectedLiteral: "message"},
		{expectedType: DEDENT, expectedLiteral: "dedent to level 1"},
		{expectedType: INFO, expectedLiteral: "info back at level 1"},
		{expectedType: STRING, expectedLiteral: "message"},
		{expectedType: DEDENT, expectedLiteral: "dedent to level 0"},
		{expectedType: INFO, expectedLiteral: "info at level 0"},
		{expectedType: STRING, expectedLiteral: "message"},
		{expectedType: EOF, expectedLiteral: "end of file"},
	}

	lexer := NewLexer(input)

	for i, tt := range tests {
		tok := lexer.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] (%s) - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tt.expectedType, tok.Type)
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
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: VERSION, expectedLiteral: "version"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: NUMBER, expectedLiteral: "2.0"},
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "test"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: GIVEN, expectedLiteral: "given"},
		{expectedType: VARIABLE, expectedLiteral: "$features"},
		{expectedType: AS, expectedLiteral: "as"},
		{expectedType: LIST, expectedLiteral: "list"},
		{expectedType: DEFAULTS, expectedLiteral: "defaults"},
		{expectedType: TO, expectedLiteral: "to"},
		{expectedType: EMPTY, expectedLiteral: "empty"}, // Test that empty is tokenized as EMPTY
		{expectedType: GIVEN, expectedLiteral: "given"},
		{expectedType: VARIABLE, expectedLiteral: "$name"},
		{expectedType: DEFAULTS, expectedLiteral: "defaults"},
		{expectedType: TO, expectedLiteral: "to"},
		{expectedType: STRING, expectedLiteral: ""},
		{expectedType: IF, expectedLiteral: "if"},
		{expectedType: VARIABLE, expectedLiteral: "$features"},
		{expectedType: IS, expectedLiteral: "is"},
		{expectedType: EMPTY, expectedLiteral: "empty"}, // Test that empty is tokenized as EMPTY in conditions
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "Features is empty"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: IF, expectedLiteral: "if"},
		{expectedType: VARIABLE, expectedLiteral: "$features"},
		{expectedType: IS, expectedLiteral: "is"},
		{expectedType: NOT, expectedLiteral: "not"},
		{expectedType: EMPTY, expectedLiteral: "empty"}, // Test that empty is tokenized as EMPTY in "is not empty"
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "Features: {$features}"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
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
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: MULTILINE_COMMENT, expectedLiteral: "/*\n    Drun Lifecycle Hooks Example\n    This example demonstrates the new tool-level lifecycle hooks\n    that run once at drun startup and shutdown\n*/"},
		{expectedType: VERSION, expectedLiteral: "version"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: NUMBER, expectedLiteral: "2.0"},
		{expectedType: MULTILINE_COMMENT, expectedLiteral: "/* Another multiline comment */"},
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "test"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "Hello World"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
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
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: MULTILINE_COMMENT, expectedLiteral: "/*\n    This comment is not terminated\n    \nversion: 2.0"},
		{expectedType: EOF, expectedLiteral: ""},
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

func TestLexer_EscapedQuotes(t *testing.T) {
	input := `task "test with \"escaped\" quotes":
  info "This has \"quotes\" inside"
  run "echo \"Hello World\""`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "test with \"escaped\" quotes"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "This has \"quotes\" inside"},
		{expectedType: RUN, expectedLiteral: "run"},
		{expectedType: STRING, expectedLiteral: "echo \"Hello World\""},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
	}

	for i, tt := range expectedTokens {
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

func TestLexer_MultilineStrings(t *testing.T) {
	input := `task "test":
  run "line 1
line 2
line 3"`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "test"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: RUN, expectedLiteral: "run"},
		{expectedType: STRING, expectedLiteral: "line 1\nline 2\nline 3"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
	}

	for i, tt := range expectedTokens {
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

func TestLexer_MultilineStringsWithEscapedQuotes(t *testing.T) {
	input := `task "test":
  run "echo \"Hello
World\"
Done"`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "test"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: RUN, expectedLiteral: "run"},
		{expectedType: STRING, expectedLiteral: "echo \"Hello\nWorld\"\nDone"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
	}

	for i, tt := range expectedTokens {
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

func TestLexer_MultilineStringsWithLineContinuation(t *testing.T) {
	input := `task "test":
  run "line 1 \
line 2 \
line 3"`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "test"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: RUN, expectedLiteral: "run"},
		{expectedType: STRING, expectedLiteral: "line 1 line 2 line 3"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
	}

	for i, tt := range expectedTokens {
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

func TestLexer_MultilineStringsComplex(t *testing.T) {
	input := `task "judge" means "Evaluate code quality":
  step "Generating a code report"
  run "rm -f coverage.xml
docker compose exec -e APP_ENV=test -e XDEBUG_MODE=coverage -u=www-data php vendor/bin/phpunit --coverage-clover ./coverage.xml
docker compose exec -e APP_ENV=test -e XDEBUG_MODE=coverage -u=www-data php bin/console tests:probe-coverage coverage.xml"`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "judge"},
		{expectedType: MEANS, expectedLiteral: "means"},
		{expectedType: STRING, expectedLiteral: "Evaluate code quality"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: STEP, expectedLiteral: "step"},
		{expectedType: STRING, expectedLiteral: "Generating a code report"},
		{expectedType: RUN, expectedLiteral: "run"},
		{expectedType: STRING, expectedLiteral: "rm -f coverage.xml\ndocker compose exec -e APP_ENV=test -e XDEBUG_MODE=coverage -u=www-data php vendor/bin/phpunit --coverage-clover ./coverage.xml\ndocker compose exec -e APP_ENV=test -e XDEBUG_MODE=coverage -u=www-data php bin/console tests:probe-coverage coverage.xml"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
	}

	for i, tt := range expectedTokens {
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

func TestLexer_MultilineStringsWithInterpolation(t *testing.T) {
	input := `task "test":
  let $env = "production"
  run "echo {$env}
is ready
to deploy"`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "test"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: LET, expectedLiteral: "let"},
		{expectedType: VARIABLE, expectedLiteral: "$env"},
		{expectedType: EQUALS, expectedLiteral: "="},
		{expectedType: STRING, expectedLiteral: "production"},
		{expectedType: RUN, expectedLiteral: "run"},
		{expectedType: STRING, expectedLiteral: "echo {$env}\nis ready\nto deploy"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
	}

	for i, tt := range expectedTokens {
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

func TestLexer_MultilineStringsMixedFeatures(t *testing.T) {
	input := `task "test":
  run "echo \"Starting...\"
Line 2 with \
continuation \
here
Final line"`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "test"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: RUN, expectedLiteral: "run"},
		{expectedType: STRING, expectedLiteral: "echo \"Starting...\"\nLine 2 with continuation here\nFinal line"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
	}

	for i, tt := range expectedTokens {
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
