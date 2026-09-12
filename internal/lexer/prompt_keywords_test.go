package lexer

import (
	"testing"
)

// TestLexer_ConfirmPromptKeywords verifies that "confirm" and "prompt"
// resolve to their dedicated keyword token types.
func TestLexer_ConfirmPromptKeywords(t *testing.T) {
	tests := []struct {
		word string
		want TokenType
	}{
		{word: "confirm", want: CONFIRM},
		{word: "prompt", want: PROMPT},
	}

	for _, tt := range tests {
		if got := LookupIdent(tt.word); got != tt.want {
			t.Errorf("LookupIdent(%q) = %s, want %s", tt.word, got, tt.want)
		}
	}
}

// TestLexer_ConfirmPromptStatements checks the token streams produced by
// confirm/prompt statements inside a task body, covering every planned
// surface form: bare, with `as $var`, with a quoted string default, and
// with a bare `no` default (which must reuse the existing NO keyword
// token). DEFAULTS/TO/AS/STRING/VARIABLE/NO are all pre-existing tokens.
func TestLexer_ConfirmPromptStatements(t *testing.T) {
	input := `version: 2.0
task "deploy":
  confirm "Deploy to production?" as $deploy
  confirm "Delete build cache?" defaults to "no"
  prompt "Which environment?" as $environment
  prompt "Release notes?" defaults to "n/a" as $notes
  confirm "Retry the step?" defaults to no`

	lexer := NewLexer(input)

	expectedTokens := []struct {
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: VERSION, expectedLiteral: "version"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: NUMBER, expectedLiteral: "2.0"},
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "deploy"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: CONFIRM, expectedLiteral: "confirm"},
		{expectedType: STRING, expectedLiteral: "Deploy to production?"},
		{expectedType: AS, expectedLiteral: "as"},
		{expectedType: VARIABLE, expectedLiteral: "$deploy"},
		{expectedType: CONFIRM, expectedLiteral: "confirm"},
		{expectedType: STRING, expectedLiteral: "Delete build cache?"},
		{expectedType: DEFAULTS, expectedLiteral: "defaults"},
		{expectedType: TO, expectedLiteral: "to"},
		{expectedType: STRING, expectedLiteral: "no"},
		{expectedType: PROMPT, expectedLiteral: "prompt"},
		{expectedType: STRING, expectedLiteral: "Which environment?"},
		{expectedType: AS, expectedLiteral: "as"},
		{expectedType: VARIABLE, expectedLiteral: "$environment"},
		{expectedType: PROMPT, expectedLiteral: "prompt"},
		{expectedType: STRING, expectedLiteral: "Release notes?"},
		{expectedType: DEFAULTS, expectedLiteral: "defaults"},
		{expectedType: TO, expectedLiteral: "to"},
		{expectedType: STRING, expectedLiteral: "n/a"},
		{expectedType: AS, expectedLiteral: "as"},
		{expectedType: VARIABLE, expectedLiteral: "$notes"},
		{expectedType: CONFIRM, expectedLiteral: "confirm"},
		{expectedType: STRING, expectedLiteral: "Retry the step?"},
		{expectedType: DEFAULTS, expectedLiteral: "defaults"},
		{expectedType: TO, expectedLiteral: "to"},
		{expectedType: NO, expectedLiteral: "no"},
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
