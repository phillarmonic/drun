package lexer

// No imports needed for basic lexer

// Lexer tokenizes drun v2 source code
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number
	column       int  // current column number

	// Indentation tracking for Python-style blocks
	indentStack    []int // stack of indentation levels
	atLineStart    bool  // true if we're at the start of a line
	pendingDedents int   // number of DEDENT tokens to emit
}

// New creates a new lexer instance
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:       input,
		line:        1,
		column:      0,
		indentStack: []int{0}, // start with zero indentation
		atLineStart: true,
	}
	l.readChar()
	return l
}

// readChar reads the next character and advances position
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII NUL represents EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++

	if l.ch == '\n' {
		l.line++
		l.column = 0
		l.atLineStart = true
	} else {
		l.column++
	}
}

// peekChar returns the next character without advancing position
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken scans and returns the next token
func (l *Lexer) NextToken() Token {
	var tok Token

	// Handle pending DEDENT tokens first
	if l.pendingDedents > 0 {
		l.pendingDedents--
		return Token{
			Type:     DEDENT,
			Literal:  "",
			Line:     l.line,
			Column:   l.column,
			Position: l.position,
		}
	}

	// Handle indentation at line start
	if l.atLineStart {
		return l.handleIndentation()
	}

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column
	tok.Position = l.position

	switch l.ch {
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
	case ':':
		tok.Type = COLON
		tok.Literal = string(l.ch)
	case ',':
		tok.Type = COMMA
		tok.Literal = string(l.ch)
	case '(':
		tok.Type = LPAREN
		tok.Literal = string(l.ch)
	case ')':
		tok.Type = RPAREN
		tok.Literal = string(l.ch)
	case '{':
		tok.Type = LBRACE
		tok.Literal = string(l.ch)
	case '}':
		tok.Type = RBRACE
		tok.Literal = string(l.ch)
	case '[':
		tok.Type = LBRACKET
		tok.Literal = string(l.ch)
	case ']':
		tok.Type = RBRACKET
		tok.Literal = string(l.ch)
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = GTE
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = GT
			tok.Literal = string(l.ch)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = LTE
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = LT
			tok.Literal = string(l.ch)
		}
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = EQ
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = NE
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
		}
	case '#':
		tok.Type = COMMENT
		tok.Literal = l.readComment()
		return tok // Don't call readChar() again
	case '\n':
		tok.Type = NEWLINE
		tok.Literal = string(l.ch)
		l.atLineStart = true
	case 0:
		tok.Literal = ""
		tok.Type = EOF
		// Emit any remaining DEDENT tokens for end of file
		if len(l.indentStack) > 1 {
			l.pendingDedents = len(l.indentStack) - 1
			l.indentStack = []int{0}
			if l.pendingDedents > 0 {
				l.pendingDedents--
				tok.Type = DEDENT
			}
		}
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			return tok // Don't call readChar() again
		} else if isDigit(l.ch) {
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
			return tok // Don't call readChar() again
		} else {
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
		}
	}

	l.readChar()
	return tok
}

// handleIndentation processes indentation at the start of a line
func (l *Lexer) handleIndentation() Token {
	l.atLineStart = false

	// Skip empty lines (just continue to next token)
	if l.ch == '\n' {
		l.readChar()
		l.atLineStart = true
		return l.NextToken()
	}

	// Handle comment lines - don't process indentation for comments
	if l.ch == '#' {
		return l.NextToken()
	}

	// Count indentation (spaces only for now)
	indent := 0
	pos := l.position
	for pos < len(l.input) && l.input[pos] == ' ' {
		indent++
		pos++
	}

	// Skip the indentation characters
	for i := 0; i < indent; i++ {
		l.readChar()
	}

	currentIndent := l.indentStack[len(l.indentStack)-1]

	if indent > currentIndent {
		// Increased indentation - INDENT token
		l.indentStack = append(l.indentStack, indent)
		return Token{
			Type:     INDENT,
			Literal:  "",
			Line:     l.line,
			Column:   l.column - indent,
			Position: l.position - indent,
		}
	} else if indent < currentIndent {
		// Decreased indentation - DEDENT token(s)
		dedentCount := 0
		for len(l.indentStack) > 1 && l.indentStack[len(l.indentStack)-1] > indent {
			l.indentStack = l.indentStack[:len(l.indentStack)-1]
			dedentCount++
		}

		// Check if indentation matches a previous level
		if l.indentStack[len(l.indentStack)-1] != indent {
			// Indentation error - doesn't match any previous level
			return Token{
				Type:     ILLEGAL,
				Literal:  "indentation error",
				Line:     l.line,
				Column:   l.column - indent,
				Position: l.position - indent,
			}
		}

		// Emit first DEDENT, queue the rest
		if dedentCount > 1 {
			l.pendingDedents = dedentCount - 1
		}

		return Token{
			Type:     DEDENT,
			Literal:  "",
			Line:     l.line,
			Column:   l.column - indent,
			Position: l.position - indent,
		}
	}

	// Same indentation level - continue with normal tokenization
	return l.NextToken()
}

// readString reads a string literal
func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		// TODO: Handle escape sequences
	}
	return l.input[position:l.position]
}

// readComment reads a comment until end of line (but doesn't consume the newline)
func (l *Lexer) readComment() string {
	position := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readIdentifier reads an identifier or keyword
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number (integer or float)
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}

	// Handle decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar() // consume '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position]
}

// skipWhitespace skips whitespace characters (except newlines and indentation)
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

// isLetter checks if character is a letter
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

// isDigit checks if character is a digit
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// AllTokens returns all tokens from the input (useful for testing)
func (l *Lexer) AllTokens() []Token {
	var tokens []Token

	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	return tokens
}
