package parser

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/errors"
	"github.com/phillarmonic/drun/internal/lexer"
)

func (p *Parser) parseStringList() []string {
	var items []string

	for p.peekToken.Type != lexer.RBRACKET && p.peekToken.Type != lexer.EOF {
		if !p.expectPeek(lexer.STRING) {
			break
		}
		items = append(items, p.curToken.Literal)

		// Check for comma
		if p.peekToken.Type == lexer.COMMA {
			p.nextToken() // consume comma
		}
	}

	// Consume RBRACKET
	if p.peekToken.Type == lexer.RBRACKET {
		p.nextToken()
	}

	return items
}

// isDependencyToken checks if a token type represents a dependency declaration
func (p *Parser) isDependencyToken(tokenType lexer.TokenType) bool {
	return tokenType == lexer.DEPENDS
}

// isDockerToken checks if a token type represents a Docker statement
func (p *Parser) isDockerToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.DOCKER, lexer.BUILD, lexer.TAG, lexer.PUSH, lexer.PULL, lexer.RUN, lexer.STOP, lexer.START, lexer.SCALE:
		return true
	default:
		return false
	}
}

// isGitToken checks if a token type represents a Git statement
func (p *Parser) isGitToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.GIT, lexer.CREATE, lexer.CHECKOUT, lexer.MERGE:
		return true
	default:
		return false
	}
}

// isHTTPToken checks if a token type represents an HTTP statement
func (p *Parser) isHTTPToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.HTTP, lexer.HTTPS, lexer.GET, lexer.POST, lexer.PUT, lexer.DELETE, lexer.PATCH, lexer.HEAD, lexer.OPTIONS, lexer.DOWNLOAD:
		return true
	default:
		return false
	}
}

// isNetworkToken checks if a token type represents a network statement
func (p *Parser) isNetworkToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.WAIT, lexer.PING, lexer.TEST:
		return true
	default:
		return false
	}
}

// isDetectionToken checks if a token type represents a detection statement
func (p *Parser) isDetectionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.DETECT, lexer.IF, lexer.WHEN:
		return true
	default:
		return false
	}
}

// isParameterToken checks if a token type represents a parameter declaration
func (p *Parser) isParameterToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.REQUIRES, lexer.GIVEN, lexer.ACCEPTS, lexer.PARAMETER:
		return true
	default:
		return false
	}
}

// isControlFlowToken checks if a token type represents a control flow statement
func (p *Parser) isControlFlowToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.WHEN, lexer.IF, lexer.FOR:
		return true
	default:
		return false
	}
}

// isActionToken checks if a token type represents an action
func (p *Parser) isActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.INFO, lexer.STEP, lexer.WARN, lexer.ERROR, lexer.SUCCESS, lexer.FAIL, lexer.ECHO,
		lexer.RUN, lexer.EXEC, lexer.SHELL, lexer.CAPTURE,
		lexer.CREATE, lexer.COPY, lexer.MOVE, lexer.DELETE, lexer.READ, lexer.WRITE, lexer.APPEND, lexer.BACKUP, lexer.CHECK:
		return true
	default:
		return false
	}
}

// isCallToken checks if a token type represents a task call
func (p *Parser) isCallToken(tokenType lexer.TokenType) bool {
	return tokenType == lexer.CALL
}

// isValidTaskNameToken checks if a token type can be used as a task name without quotes
func (p *Parser) isValidTaskNameToken(tokenType lexer.TokenType) bool {
	// Allow common task-related keywords to be used as task names
	switch tokenType {
	case lexer.TEST, lexer.BUILD, lexer.CI,
		lexer.START, lexer.STOP,
		lexer.BACKUP, lexer.CHECK, lexer.VERIFY:
		return true
	default:
		return false
	}
}

// isKeywordToken checks if a token type is a keyword (can be used as a parameter name)
func (p *Parser) isKeywordToken(tokenType lexer.TokenType) bool {
	// Return false for basic tokens, structural keywords, and statement-starting keywords
	switch tokenType {
	case lexer.ILLEGAL, lexer.EOF, lexer.STRING, lexer.NUMBER, lexer.BOOLEAN, lexer.VARIABLE, lexer.IDENT:
		// Basic tokens
		return false
	case lexer.VERSION, lexer.TASK, lexer.PROJECT, lexer.DRUN,
		lexer.SETUP, lexer.TEARDOWN, lexer.BEFORE, lexer.AFTER,
		lexer.IF, lexer.ELSE, lexer.WHEN, lexer.OTHERWISE,
		lexer.FOR, lexer.IN, lexer.PARALLEL,
		lexer.WITH, lexer.TRY, lexer.CATCH, lexer.FINALLY,
		lexer.THROW, lexer.IGNORE, lexer.CALL,
		lexer.COLON, lexer.EQUALS, lexer.COMMA, lexer.LPAREN, lexer.RPAREN,
		lexer.LBRACE, lexer.RBRACE, lexer.LBRACKET, lexer.RBRACKET,
		lexer.NEWLINE, lexer.INDENT, lexer.DEDENT,
		lexer.INFO, lexer.STEP, lexer.WARN, lexer.ERROR, lexer.SUCCESS, lexer.FAIL, lexer.ECHO,
		lexer.RUN, lexer.EXEC, lexer.SHELL, lexer.CAPTURE,
		lexer.CREATE, lexer.COPY, lexer.MOVE, lexer.DELETE, lexer.READ, lexer.WRITE, lexer.APPEND, lexer.BACKUP, lexer.CHECK,
		lexer.DOCKER, lexer.GIT, lexer.HTTP, lexer.HTTPS, lexer.GET, lexer.POST, lexer.PUT, lexer.PATCH, lexer.HEAD, lexer.OPTIONS,
		lexer.DETECT, lexer.GIVEN, lexer.REQUIRES, lexer.DEFAULTS, lexer.BREAK, lexer.CONTINUE,
		lexer.USE, lexer.SNIPPET, lexer.TEMPLATE, lexer.PARAMETER, lexer.MIXIN, lexer.USES, lexer.INCLUDES:
		// Structural keywords, action keywords, and statement-starting keywords
		return false
	default:
		// Everything else (like ENVIRONMENT, TARGET, etc.) can be a parameter name
		return true
	}
}

// isShellActionToken checks if a token type represents a shell action
func (p *Parser) isShellActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.RUN, lexer.EXEC, lexer.SHELL, lexer.CAPTURE:
		return true
	default:
		return false
	}
}

// isTypeToken checks if a token type represents a data type
func (p *Parser) isTypeToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN_TYPE, lexer.LIST_TYPE, lexer.IDENT:
		return true
	default:
		return false
	}
}

// isFileActionToken checks if a token type represents a file action
func (p *Parser) isFileActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.COPY, lexer.MOVE, lexer.DELETE, lexer.READ, lexer.WRITE, lexer.APPEND, lexer.BACKUP, lexer.CHECK:
		return true
	default:
		return false
	}
}

// isErrorHandlingToken checks if a token type represents error handling
func (p *Parser) isErrorHandlingToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TRY:
		return true
	default:
		return false
	}
}

// isThrowActionToken checks if a token type represents a throw action
func (p *Parser) isThrowActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.THROW, lexer.RETHROW, lexer.IGNORE:
		return true
	default:
		return false
	}
}

// expectPeek checks the peek token type and advances if it matches
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// expectPeekSkipNewlines expects a token type but skips any NEWLINE tokens first
func (p *Parser) expectPeekSkipNewlines(t lexer.TokenType) bool {
	// Skip any NEWLINE tokens
	for p.peekToken.Type == lexer.NEWLINE {
		p.nextToken() // consume the NEWLINE
	}

	// Now check for the expected token
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// peekError adds an error for unexpected peek token
func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)

	// Also add to new error system if available
	if p.errorList != nil {
		p.errorList.Add(msg, p.peekToken)
	}
}

// addError adds an error message
func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)

	// Also add to new error system if available
	if p.errorList != nil {
		p.errorList.Add(msg, p.curToken)
	}
}

// skipComments skips over comment tokens and newlines
func (p *Parser) skipComments() {
	for p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT || p.curToken.Type == lexer.NEWLINE {
		p.nextToken()
	}
}

// Errors returns any parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

// ErrorList returns the enhanced error list with position information
func (p *Parser) ErrorList() *errors.ParseErrorList {
	return p.errorList
}

// parseControlFlowStatement parses control flow statements (when, if, for)

// isVariableOperationToken checks if a token represents variable operations
func (p *Parser) isVariableOperationToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.LET, lexer.SET, lexer.TRANSFORM, lexer.CAPTURE:
		return true
	default:
		return false
	}
}

// parseVariableStatement and related methods moved to parser_variable.go

// Expression parsing methods moved to parser_expression.go

// expectPeekVariableName checks for variable names using $variable syntax
func (p *Parser) expectPeekVariableName() bool {
	if p.peekToken.Type != lexer.VARIABLE {
		p.addError(fmt.Sprintf("expected variable name ($variable), got %s instead", p.peekToken.Type))
		return false
	}

	p.nextToken()

	// Check if variable name is reserved
	if !p.validateVariableName(p.curToken.Literal) {
		return false
	}

	return true
}

// validateVariableName checks if a variable name is reserved
func (p *Parser) validateVariableName(name string) bool {
	// Reserved variable names that cannot be user-defined
	reservedNames := []string{
		"$globals", // Used to access project settings
	}

	for _, reserved := range reservedNames {
		if name == reserved {
			p.addError(fmt.Sprintf("cannot use reserved variable name '%s'", name))
			return false
		}
	}

	return true
}

// expectPeekFileKeyword checks for the "file" keyword (as IDENT)
func (p *Parser) expectPeekFileKeyword() bool {
	if p.peekToken.Type != lexer.IDENT || p.peekToken.Literal != "file" {
		p.addError(fmt.Sprintf("expected 'file', got %s instead", p.peekToken.Type))
		return false
	}

	p.nextToken()
	return true
}

// getVariableName returns the variable name without the $ prefix
func (p *Parser) getVariableName() string {
	if p.curToken.Type == lexer.VARIABLE && len(p.curToken.Literal) > 1 {
		return p.curToken.Literal[1:] // Remove the $ prefix
	}
	return p.curToken.Literal
}

// expectPeekIdentifierLike checks for identifier-like tokens (IDENT, VARIABLE, or keywords that can be used as identifiers)
func (p *Parser) expectPeekIdentifierLike() bool {
	switch p.peekToken.Type {
	case lexer.IDENT, lexer.VARIABLE, lexer.SERVICE, lexer.ENVIRONMENT, lexer.HOST, lexer.PORT, lexer.VERSION, lexer.TOOL:
		p.nextToken()
		return true
	default:
		p.addError(fmt.Sprintf("expected identifier or variable, got %s instead", p.peekToken.Type))
		return false
	}
}

// isPortCheckPattern checks if the current "check if" is a port check without consuming tokens
func (p *Parser) isPortCheckPattern() bool {
	// We're currently at CHECK token, peek is IF
	// We need to check if the pattern is "check if port"

	// Use a simple string-based approach by examining the lexer's input
	// Get the current position and look for "port" after "if"

	// This is a simplified approach - look at the raw input around current position
	if p.lexer == nil {
		return false
	}

	// Create a temporary lexer from current position to peek ahead
	// We'll use a different approach: examine the input string directly

	// Get current token position and look ahead in the input
	currentPos := p.curToken.Position
	input := p.lexer.GetInput() // We need to add this method to lexer

	// Look for "if port" pattern starting from current position
	if currentPos >= 0 && currentPos < len(input) {
		// Find "if" after current position
		remaining := input[currentPos:]
		ifIndex := strings.Index(remaining, "if")
		if ifIndex >= 0 {
			afterIf := remaining[ifIndex+2:]
			// Skip whitespace and look for "port"
			afterIf = strings.TrimLeft(afterIf, " \t")
			return strings.HasPrefix(afterIf, "port")
		}
	}

	return false
}

// expectPeekFunctionName checks for function names (can be IDENT or reserved keywords)
func (p *Parser) expectPeekFunctionName() bool {
	// Function names can be regular identifiers or reserved keywords used as function names
	validFunctionTokens := map[lexer.TokenType]bool{
		lexer.IDENT:     true,
		lexer.CONCAT:    true,
		lexer.SPLIT:     true,
		lexer.REPLACE:   true,
		lexer.TRIM:      true,
		lexer.UPPERCASE: true,
		lexer.LOWERCASE: true,
		lexer.PREPEND:   true,
		lexer.JOIN:      true,
		lexer.SLICE:     true,
		lexer.LENGTH:    true,
		lexer.KEYS:      true,
		lexer.VALUES:    true,
		lexer.SUBTRACT:  true,
		lexer.MULTIPLY:  true,
		lexer.DIVIDE:    true,
		lexer.MODULO:    true,
	}

	if !validFunctionTokens[p.peekToken.Type] {
		p.addError(fmt.Sprintf("expected function name, got %s instead", p.peekToken.Type))
		return false
	}
	p.nextToken()
	return true
}

// parseConditionExpression parses condition expressions like "environment is production"
func (p *Parser) parseConditionExpression() string {
	var parts []string

	// Read tokens until we hit a colon
	for p.peekToken.Type != lexer.COLON && p.peekToken.Type != lexer.EOF {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)
	}

	return strings.Join(parts, " ")
}

// parseSimpleCondition parses simple conditions for break/continue statements
func (p *Parser) parseSimpleCondition() string {
	var parts []string

	// Parse a simple expression: variable operator value
	// This should be something like "item == 'stop'" or "count > 10"

	// Get the variable
	if p.peekToken.Type == lexer.IDENT || p.peekToken.Type == lexer.VARIABLE {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)
	}

	// Get the operator
	if p.peekToken.Type == lexer.EQ || p.peekToken.Type == lexer.NE ||
		p.peekToken.Type == lexer.GT || p.peekToken.Type == lexer.GTE ||
		p.peekToken.Type == lexer.LT || p.peekToken.Type == lexer.LTE ||
		p.peekToken.Type == lexer.CONTAINS || p.peekToken.Type == lexer.STARTS ||
		p.peekToken.Type == lexer.ENDS || p.peekToken.Type == lexer.MATCHES {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)

		// Handle "starts with" and "ends with"
		if (p.curToken.Type == lexer.STARTS || p.curToken.Type == lexer.ENDS) && p.peekToken.Type == lexer.WITH {
			p.nextToken() // consume WITH
			parts = append(parts, p.curToken.Literal)
		}
	}

	// Get the value
	if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.NUMBER || p.peekToken.Type == lexer.IDENT {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)
	}

	return strings.Join(parts, " ")
}
