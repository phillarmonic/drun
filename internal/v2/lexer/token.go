package lexer

import "fmt"

// TokenType represents the type of a token
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF

	// Literals
	STRING  // "hello world"
	NUMBER  // 2.0, 42
	BOOLEAN // true, false

	// Keywords
	VERSION // version
	TASK    // task
	MEANS   // means

	// Action keywords (built-in actions)
	INFO    // info
	STEP    // step
	WARN    // warn
	ERROR   // error
	SUCCESS // success
	FAIL    // fail

	// Parameter keywords
	REQUIRES // requires
	GIVEN    // given
	ACCEPTS  // accepts
	DEFAULTS // defaults
	TO       // to
	FROM     // from
	AS       // as
	LIST     // list
	OF       // of

	// Control flow keywords
	WHEN     // when
	IF       // if
	ELSE     // else
	FOR      // for
	EACH     // each
	IN       // in
	PARALLEL // parallel
	IS       // is

	// Identifiers and operators
	IDENT  // user-defined identifiers
	ASSIGN // :

	// Punctuation
	COLON    // :
	COMMA    // ,
	LPAREN   // (
	RPAREN   // )
	LBRACE   // {
	RBRACE   // }
	LBRACKET // [
	RBRACKET // ]

	// Indentation (Python-style)
	INDENT
	DEDENT
	NEWLINE

	// Comments
	COMMENT // # comment
)

// Token represents a single token
type Token struct {
	Type     TokenType
	Literal  string
	Line     int
	Column   int
	Position int
}

// String returns a string representation of the token type
func (t TokenType) String() string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case STRING:
		return "STRING"
	case NUMBER:
		return "NUMBER"
	case BOOLEAN:
		return "BOOLEAN"
	case VERSION:
		return "VERSION"
	case TASK:
		return "TASK"
	case MEANS:
		return "MEANS"
	case INFO:
		return "INFO"
	case STEP:
		return "STEP"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case SUCCESS:
		return "SUCCESS"
	case FAIL:
		return "FAIL"
	case REQUIRES:
		return "REQUIRES"
	case GIVEN:
		return "GIVEN"
	case ACCEPTS:
		return "ACCEPTS"
	case DEFAULTS:
		return "DEFAULTS"
	case TO:
		return "TO"
	case FROM:
		return "FROM"
	case AS:
		return "AS"
	case LIST:
		return "LIST"
	case OF:
		return "OF"
	case WHEN:
		return "WHEN"
	case IF:
		return "IF"
	case ELSE:
		return "ELSE"
	case FOR:
		return "FOR"
	case EACH:
		return "EACH"
	case IN:
		return "IN"
	case PARALLEL:
		return "PARALLEL"
	case IS:
		return "IS"
	case IDENT:
		return "IDENT"
	case ASSIGN:
		return "ASSIGN"
	case COLON:
		return "COLON"
	case COMMA:
		return "COMMA"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case LBRACKET:
		return "LBRACKET"
	case RBRACKET:
		return "RBRACKET"
	case INDENT:
		return "INDENT"
	case DEDENT:
		return "DEDENT"
	case NEWLINE:
		return "NEWLINE"
	case COMMENT:
		return "COMMENT"
	default:
		return "UNKNOWN"
	}
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("Token{Type: %s, Literal: %q, Line: %d, Column: %d}",
		t.Type, t.Literal, t.Line, t.Column)
}

// Keywords maps string literals to their token types
var keywords = map[string]TokenType{
	"version":  VERSION,
	"task":     TASK,
	"means":    MEANS,
	"info":     INFO,
	"step":     STEP,
	"warn":     WARN,
	"error":    ERROR,
	"success":  SUCCESS,
	"fail":     FAIL,
	"requires": REQUIRES,
	"given":    GIVEN,
	"accepts":  ACCEPTS,
	"defaults": DEFAULTS,
	"to":       TO,
	"from":     FROM,
	"as":       AS,
	"list":     LIST,
	"of":       OF,
	"when":     WHEN,
	"if":       IF,
	"else":     ELSE,
	"for":      FOR,
	"each":     EACH,
	"in":       IN,
	"parallel": PARALLEL,
	"is":       IS,
	"true":     BOOLEAN,
	"false":    BOOLEAN,
}

// LookupIdent checks if an identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
