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
	PROJECT // project
	SET     // set
	INCLUDE // include
	BEFORE  // before
	AFTER   // after
	ANY     // any
	DEPENDS // depends
	ON      // on
	THEN    // then
	AND     // and

	// Docker keywords
	DOCKER    // docker
	IMAGE     // image
	CONTAINER // container
	COMPOSE   // compose
	BUILD     // build
	PUSH      // push
	PULL      // pull
	TAG       // tag
	REMOVE    // remove
	START     // start
	STOP      // stop
	UP        // up
	DOWN      // down

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

	// Built-in functions/conditions
	FILE   // file
	EXISTS // exists

	// Shell operations
	RUN     // run
	EXEC    // exec
	SHELL   // shell
	CAPTURE // capture
	OUTPUT  // output

	// Type keywords
	STRING_TYPE  // string
	NUMBER_TYPE  // number
	BOOLEAN_TYPE // boolean
	LIST_TYPE    // list

	// File operations
	CREATE // create
	COPY   // copy
	MOVE   // move
	DELETE // delete
	READ   // read
	WRITE  // write
	APPEND // append
	DIR    // dir

	// Error handling
	TRY     // try
	CATCH   // catch
	FINALLY // finally
	THROW   // throw
	RETHROW // rethrow
	IGNORE  // ignore

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
	case PROJECT:
		return "PROJECT"
	case SET:
		return "SET"
	case INCLUDE:
		return "INCLUDE"
	case BEFORE:
		return "BEFORE"
	case AFTER:
		return "AFTER"
	case ANY:
		return "ANY"
	case DEPENDS:
		return "DEPENDS"
	case ON:
		return "ON"
	case THEN:
		return "THEN"
	case AND:
		return "AND"
	case DOCKER:
		return "DOCKER"
	case IMAGE:
		return "IMAGE"
	case CONTAINER:
		return "CONTAINER"
	case COMPOSE:
		return "COMPOSE"
	case BUILD:
		return "BUILD"
	case PUSH:
		return "PUSH"
	case PULL:
		return "PULL"
	case TAG:
		return "TAG"
	case REMOVE:
		return "REMOVE"
	case START:
		return "START"
	case STOP:
		return "STOP"
	case UP:
		return "UP"
	case DOWN:
		return "DOWN"
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
	case FILE:
		return "FILE"
	case EXISTS:
		return "EXISTS"
	case RUN:
		return "RUN"
	case EXEC:
		return "EXEC"
	case SHELL:
		return "SHELL"
	case CAPTURE:
		return "CAPTURE"
	case OUTPUT:
		return "OUTPUT"
	case STRING_TYPE:
		return "STRING_TYPE"
	case NUMBER_TYPE:
		return "NUMBER_TYPE"
	case BOOLEAN_TYPE:
		return "BOOLEAN_TYPE"
	case LIST_TYPE:
		return "LIST_TYPE"
	case CREATE:
		return "CREATE"
	case COPY:
		return "COPY"
	case MOVE:
		return "MOVE"
	case DELETE:
		return "DELETE"
	case READ:
		return "READ"
	case WRITE:
		return "WRITE"
	case APPEND:
		return "APPEND"
	case DIR:
		return "DIR"
	case TRY:
		return "TRY"
	case CATCH:
		return "CATCH"
	case FINALLY:
		return "FINALLY"
	case THROW:
		return "THROW"
	case RETHROW:
		return "RETHROW"
	case IGNORE:
		return "IGNORE"
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
	"version":   VERSION,
	"task":      TASK,
	"means":     MEANS,
	"project":   PROJECT,
	"set":       SET,
	"include":   INCLUDE,
	"before":    BEFORE,
	"after":     AFTER,
	"any":       ANY,
	"depends":   DEPENDS,
	"on":        ON,
	"then":      THEN,
	"and":       AND,
	"docker":    DOCKER,
	"image":     IMAGE,
	"container": CONTAINER,
	"compose":   COMPOSE,
	"build":     BUILD,
	"push":      PUSH,
	"pull":      PULL,
	"tag":       TAG,
	"remove":    REMOVE,
	"start":     START,
	"stop":      STOP,
	"up":        UP,
	"down":      DOWN,
	"info":      INFO,
	"step":      STEP,
	"warn":      WARN,
	"error":     ERROR,
	"success":   SUCCESS,
	"fail":      FAIL,
	"requires":  REQUIRES,
	"given":     GIVEN,
	"accepts":   ACCEPTS,
	"defaults":  DEFAULTS,
	"to":        TO,
	"from":      FROM,
	"as":        AS,
	"list":      LIST,
	"of":        OF,
	"when":      WHEN,
	"if":        IF,
	"else":      ELSE,
	"for":       FOR,
	"each":      EACH,
	"in":        IN,
	"parallel":  PARALLEL,
	"is":        IS,
	"file":      FILE,
	"exists":    EXISTS,
	"run":       RUN,
	"exec":      EXEC,
	"shell":     SHELL,
	"capture":   CAPTURE,
	"output":    OUTPUT,
	"string":    STRING_TYPE,
	"number":    NUMBER_TYPE,
	"boolean":   BOOLEAN_TYPE,
	"create":    CREATE,
	"copy":      COPY,
	"move":      MOVE,
	"delete":    DELETE,
	"read":      READ,
	"write":     WRITE,
	"append":    APPEND,
	"dir":       DIR,
	"try":       TRY,
	"catch":     CATCH,
	"finally":   FINALLY,
	"throw":     THROW,
	"rethrow":   RETHROW,
	"ignore":    IGNORE,
	"true":      BOOLEAN,
	"false":     BOOLEAN,
}

// LookupIdent checks if an identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
