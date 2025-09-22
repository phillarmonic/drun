package lexer

import (
	"testing"
)

func TestLexer_TabIndentation(t *testing.T) {
	// Test with tab characters for indentation
	input := "version: 2.0\n\ntask \"test\":\n\tinfo \"level 1\"\n\t\tinfo \"level 2\"\n\tinfo \"back to level 1\""

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
		{INFO, "info"},
		{STRING, "level 1"},
		{INDENT, ""},
		{INFO, "info"},
		{STRING, "level 2"},
		{DEDENT, ""},
		{INFO, "info"},
		{STRING, "back to level 1"},
		{DEDENT, ""},
		{EOF, ""},
	}

	lexer := NewLexer(input)

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

func TestLexer_MixedIndentation(t *testing.T) {
	// Test with mixed spaces and tabs (should work but not recommended)
	input := "version: 2.0\n\ntask \"test\":\n    info \"4 spaces\"\n\tinfo \"1 tab (equivalent to 4 spaces)\""

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
		{INFO, "info"},
		{STRING, "4 spaces"},
		{INFO, "info"},
		{STRING, "1 tab (equivalent to 4 spaces)"},
		{DEDENT, ""},
		{EOF, ""},
	}

	lexer := NewLexer(input)

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
