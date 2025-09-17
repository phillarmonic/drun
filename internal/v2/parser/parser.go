package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/v2/ast"
	"github.com/phillarmonic/drun/internal/v2/lexer"
)

// Parser parses drun v2 source code into an AST
type Parser struct {
	lexer *lexer.Lexer

	curToken  lexer.Token
	peekToken lexer.Token

	errors []string
}

// New creates a new parser instance
func NewParser(l *lexer.Lexer) *Parser {
	p := &Parser{
		lexer:  l,
		errors: []string{},
	}

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// nextToken advances both curToken and peekToken
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// ParseProgram parses the entire program
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}

	// Skip any leading comments
	p.skipComments()

	// Parse version statement (required)
	if p.curToken.Type == lexer.VERSION {
		program.Version = p.parseVersionStatement()
	} else {
		p.addError(fmt.Sprintf("expected version statement, got %s", p.curToken.Type))
		return nil
	}

	// Skip comments between version and tasks
	p.skipComments()

	// Parse task statements
	for p.curToken.Type != lexer.EOF {
		if p.curToken.Type == lexer.TASK {
			task := p.parseTaskStatement()
			if task != nil {
				program.Tasks = append(program.Tasks, task)
			}
		} else if p.curToken.Type == lexer.COMMENT {
			p.nextToken() // Skip comments
		} else if p.curToken.Type == lexer.DEDENT {
			// Skip stray lexer.DEDENT tokens (they should be consumed by task parsing)
			p.nextToken()
		} else {
			p.addError(fmt.Sprintf("unexpected token: %s", p.curToken.Type))
			p.nextToken()
		}
	}

	return program
}

// parseVersionStatement parses a version statement
func (p *Parser) parseVersionStatement() *ast.VersionStatement {
	stmt := &ast.VersionStatement{Token: p.curToken}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	if !p.expectPeek(lexer.NUMBER) {
		return nil
	}

	stmt.Value = p.curToken.Literal
	p.nextToken()

	return stmt
}

// parseTaskStatement parses a task statement
func (p *Parser) parseTaskStatement() *ast.TaskStatement {
	stmt := &ast.TaskStatement{Token: p.curToken}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}

	stmt.Name = p.curToken.Literal

	// Check for optional "means" clause
	if p.peekToken.Type == lexer.MEANS {
		p.nextToken() // consume lexer.MEANS

		if !p.expectPeek(lexer.STRING) {
			return nil
		}

		stmt.Description = p.curToken.Literal
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Expect lexer.INDENT to start task body
	if !p.expectPeek(lexer.INDENT) {
		return nil
	}

	// Parse task body (parameters and statements)
	for {
		// Check if we've reached the end of the task body
		if p.peekToken.Type == lexer.DEDENT || p.peekToken.Type == lexer.EOF {
			break
		}

		p.nextToken() // Move to the next token

		if p.isParameterToken(p.curToken.Type) {
			param := p.parseParameterStatement()
			if param != nil {
				stmt.Parameters = append(stmt.Parameters, *param)
			}
		} else if p.isActionToken(p.curToken.Type) {
			action := p.parseActionStatement()
			if action != nil {
				stmt.Body = append(stmt.Body, action)
			}
		} else if p.curToken.Type == lexer.COMMENT {
			// Skip comments in task body
			continue
		} else {
			p.addError(fmt.Sprintf("unexpected token in task body: %s (peek: %s)", p.curToken.Type, p.peekToken.Type))
			break // Stop parsing on unexpected token
		}
	}

	// Consume lexer.DEDENT
	if p.peekToken.Type == lexer.DEDENT {
		p.nextToken() // Move to lexer.DEDENT
		p.nextToken() // Move past lexer.DEDENT
	}

	return stmt
}

// parseActionStatement parses an action statement (info, step, success, etc.)
func (p *Parser) parseActionStatement() *ast.ActionStatement {
	stmt := &ast.ActionStatement{
		Token:  p.curToken,
		Action: p.curToken.Literal,
	}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}

	stmt.Message = p.curToken.Literal

	return stmt
}

// parseParameterStatement parses parameter declarations (requires, given, accepts)
func (p *Parser) parseParameterStatement() *ast.ParameterStatement {
	stmt := &ast.ParameterStatement{
		Token: p.curToken,
		Type:  p.curToken.Literal,
	}

	// Parse parameter name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Handle different parameter types
	switch stmt.Type {
	case "requires":
		stmt.Required = true
		// Check for constraints: requires env from ["dev", "staging"]
		if p.peekToken.Type == lexer.FROM {
			p.nextToken() // consume FROM
			if p.peekToken.Type == lexer.LBRACKET {
				p.nextToken() // consume LBRACKET
				stmt.Constraints = p.parseStringList()
			}
		}

	case "given":
		stmt.Required = false
		// Expect: given name defaults to "value"
		if !p.expectPeek(lexer.DEFAULTS) {
			return nil
		}
		if !p.expectPeek(lexer.TO) {
			return nil
		}
		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.DefaultValue = p.curToken.Literal

	case "accepts":
		stmt.Required = false
		// Handle: accepts items as list of strings
		if p.peekToken.Type == lexer.AS {
			p.nextToken() // consume AS
			if p.peekToken.Type == lexer.LIST {
				p.nextToken() // consume LIST
				stmt.DataType = "list"
				if p.peekToken.Type == lexer.OF {
					p.nextToken() // consume OF
					if p.peekToken.Type == lexer.IDENT {
						p.nextToken() // consume type
						stmt.DataType = "list of " + p.curToken.Literal
					}
				}
			}
		}
	}

	return stmt
}

// parseStringList parses a list of strings like ["dev", "staging", "production"]
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

// isParameterToken checks if a token type represents a parameter declaration
func (p *Parser) isParameterToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.REQUIRES, lexer.GIVEN, lexer.ACCEPTS:
		return true
	default:
		return false
	}
}

// isActionToken checks if a token type represents an action
func (p *Parser) isActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.INFO, lexer.STEP, lexer.WARN, lexer.ERROR, lexer.SUCCESS, lexer.FAIL:
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

// peekError adds an error for unexpected peek token
func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

// addError adds an error message
func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)
}

// skipComments skips over comment tokens
func (p *Parser) skipComments() {
	for p.curToken.Type == lexer.COMMENT {
		p.nextToken()
	}
}

// Errors returns any parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}
