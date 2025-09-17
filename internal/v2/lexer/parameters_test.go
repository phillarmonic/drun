package lexer

import (
	"testing"
)

func TestLexer_ParameterKeywords(t *testing.T) {
	input := `requires name
given title defaults to "friend"
accepts items as list of strings
when environment is "production"`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{REQUIRES, "requires"},
		{IDENT, "name"},
		{GIVEN, "given"},
		{IDENT, "title"},
		{DEFAULTS, "defaults"},
		{TO, "to"},
		{STRING, "friend"},
		{ACCEPTS, "accepts"},
		{IDENT, "items"},
		{AS, "as"},
		{LIST, "list"},
		{OF, "of"},
		{IDENT, "strings"},
		{WHEN, "when"},
		{IDENT, "environment"},
		{IS, "is"},
		{STRING, "production"},
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

func TestLexer_ControlFlowKeywords(t *testing.T) {
	input := `if condition:
  step "inside if"
else:
  step "inside else"

for each item in items:
  info "processing {item}"`

	expectedKeywords := []TokenType{
		IF, IDENT, COLON, // if condition:
		INDENT, STEP, STRING, // step "inside if"
		DEDENT, ELSE, COLON, // else:
		INDENT, STEP, STRING, // step "inside else"
		DEDENT, FOR, EACH, IDENT, IN, IDENT, COLON, // for each item in items:
		INDENT, INFO, STRING, // info "processing {item}"
		DEDENT, EOF,
	}

	lexer := NewLexer(input)

	for i, expectedType := range expectedKeywords {
		tok := lexer.NextToken()

		if tok.Type != expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal: %q)",
				i, expectedType, tok.Type, tok.Literal)
		}
	}
}
