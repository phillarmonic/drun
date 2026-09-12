package lexer

import (
	"testing"
)

func TestLexer_TabIndentation(t *testing.T) {
	// Test with tab characters for indentation
	input := "version: 2.0\n\ntask \"test\":\n\tinfo \"level 1\"\n\t\tinfo \"level 2\"\n\tinfo \"back to level 1\""

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
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "level 1"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "level 2"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "back to level 1"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
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
		{expectedType: STRING, expectedLiteral: "4 spaces"},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "1 tab (equivalent to 4 spaces)"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
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

func TestLexer_IndentedCommentDoesNotEstablishBlockIndentation(t *testing.T) {
	input := "task \"test\":\n\t  \t# deliberately irregular comment indentation\n    info \"level 1\""

	expectedTokens := []struct {
		expectedLiteral string
		expectedType    TokenType
	}{
		{expectedType: TASK, expectedLiteral: "task"},
		{expectedType: STRING, expectedLiteral: "test"},
		{expectedType: COLON, expectedLiteral: ":"},
		{expectedType: COMMENT, expectedLiteral: "# deliberately irregular comment indentation"},
		{expectedType: INDENT, expectedLiteral: ""},
		{expectedType: INFO, expectedLiteral: "info"},
		{expectedType: STRING, expectedLiteral: "level 1"},
		{expectedType: DEDENT, expectedLiteral: ""},
		{expectedType: EOF, expectedLiteral: ""},
	}

	lexer := NewLexer(input)

	for i, tt := range expectedTokens {
		tok := lexer.NextToken()
		if tok.Type != tt.expectedType || tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - expected=(%q, %q), got=(%q, %q)",
				i, tt.expectedType, tt.expectedLiteral, tok.Type, tok.Literal)
		}
	}
}

func TestLexer_IndentedNonCodeLinesDoNotChangeExistingBlockIndentation(t *testing.T) {
	input := "task \"test\":\n    info \"before\"\n\t\t  # irregular comment indentation\n\t \t  \n    info \"after\""

	expectedTypes := []TokenType{
		TASK, STRING, COLON,
		INDENT, INFO, STRING,
		COMMENT,
		INFO, STRING,
		DEDENT, EOF,
	}

	lexer := NewLexer(input)
	for i, expectedType := range expectedTypes {
		tok := lexer.NextToken()
		if tok.Type != expectedType {
			t.Fatalf("tests[%d] - expected=%q, got=%q (literal: %q)",
				i, expectedType, tok.Type, tok.Literal)
		}
	}
}
