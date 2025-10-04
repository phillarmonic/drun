package parser

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func (p *Parser) parseVariableStatement() *ast.VariableStatement {
	stmt := &ast.VariableStatement{
		Token: p.curToken,
	}

	switch p.curToken.Type {
	case lexer.LET:
		return p.parseLetStatement(stmt)
	case lexer.SET:
		return p.parseSetVariableStatement(stmt)
	case lexer.TRANSFORM:
		return p.parseTransformStatement(stmt)
	case lexer.CAPTURE:
		return p.parseCaptureVariableStatement(stmt)
	default:
		p.addError(fmt.Sprintf("unexpected variable operation token: %s", p.curToken.Type))
		return nil
	}
}

// parseLetStatement parses "let variable = value" or "let variable be expression" statements
func (p *Parser) parseLetStatement(stmt *ast.VariableStatement) *ast.VariableStatement {
	stmt.Operation = "let"

	// Check if next token is $variable (old syntax) or identifier (new syntax)
	switch p.peekToken.Type {
	case lexer.VARIABLE:
		// Old syntax: "let $variable = value" or "let $variable as type to value"
		if !p.expectPeekVariableName() {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		// Check for optional "as type" syntax
		if p.peekToken.Type == lexer.AS {
			p.nextToken() // consume AS
			// Parse type (list, string, number, etc.)
			if p.peekToken.Type == lexer.LIST || p.isTypeToken(p.peekToken.Type) {
				p.nextToken() // consume type
				// Type information is not currently stored in VariableStatement
				// but we accept it for syntax compatibility
			} else {
				p.addError("expected type after 'as'")
				return nil
			}

			// Expect "to" after type
			if !p.expectPeek(lexer.TO) {
				return nil
			}

			// Parse the value
			p.nextToken()
			stmt.Value = p.parseExpression()
			return stmt
		}

		if !p.expectPeek(lexer.EQUALS) {
			return nil
		}

		// Parse the value (could be string, number, or expression)
		p.nextToken()

		stmt.Value = p.parseExpression()
		return stmt
	case lexer.IDENT:
		// New syntax: "let variable be expression"
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer.BE) {
			return nil
		}

		// Parse the expression after "be"
		p.nextToken()
		stmt.Value = p.parseExpression()
		return stmt
	default:
		p.addError("expected variable name or identifier after 'let'")
		return nil
	}
}

// parseSetVariableStatement parses "set variable to value" statements
// Supports: set $variable to value
//
//	set $variable as list to ["value1", "value2"]
func (p *Parser) parseSetVariableStatement(stmt *ast.VariableStatement) *ast.VariableStatement {
	stmt.Operation = "set"

	if !p.expectPeekVariableName() {
		return nil
	}
	stmt.Variable = p.curToken.Literal

	// Check for optional "as type" syntax
	if p.peekToken.Type == lexer.AS {
		p.nextToken() // consume AS
		// Parse type (list, string, number, etc.)
		if p.peekToken.Type == lexer.LIST || p.isTypeToken(p.peekToken.Type) {
			p.nextToken() // consume type
			// Type information is not currently stored in VariableStatement
			// but we accept it for syntax compatibility
		} else {
			p.addError("expected type after 'as'")
			return nil
		}
	}

	if !p.expectPeek(lexer.TO) {
		return nil
	}

	// Parse the value
	p.nextToken()

	stmt.Value = p.parseExpression()
	return stmt
}

// parseTransformStatement parses "transform variable with function args" statements
func (p *Parser) parseTransformStatement(stmt *ast.VariableStatement) *ast.VariableStatement {
	stmt.Operation = "transform"

	if !p.expectPeekVariableName() {
		return nil
	}
	stmt.Variable = p.curToken.Literal

	if !p.expectPeek(lexer.WITH) {
		return nil
	}

	// Parse the function name (can be IDENT or reserved keywords like CONCAT, UPPERCASE, etc.)
	if !p.expectPeekFunctionName() {
		return nil
	}
	stmt.Function = p.curToken.Literal

	// Parse optional arguments
	for p.peekToken.Type != lexer.NEWLINE && p.peekToken.Type != lexer.DEDENT && p.peekToken.Type != lexer.EOF && p.peekToken.Type != lexer.COMMENT && p.peekToken.Type != lexer.MULTILINE_COMMENT {
		// Check if the next token looks like an argument
		if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.IDENT || p.peekToken.Type == lexer.NUMBER {
			p.nextToken()
			argValue := p.curToken.Literal
			// For identifiers, mark them for interpolation
			if p.curToken.Type == lexer.IDENT {
				argValue = "{" + p.curToken.Literal + "}"
			}
			stmt.Arguments = append(stmt.Arguments, argValue)
		} else {
			// Stop parsing arguments if we hit an unexpected token
			break
		}
	}

	return stmt
}

// parseCaptureVariableStatement parses "capture variable from expression" and "capture from shell command as $variable" statements
func (p *Parser) parseCaptureVariableStatement(stmt *ast.VariableStatement) *ast.VariableStatement {
	stmt.Operation = "capture"

	// Check if next token is "from" (shell syntax) or identifier (expression syntax)
	switch p.peekToken.Type {
	case lexer.FROM:
		// Shell syntax: "capture from shell "command" as $variable" or "capture from shell as $variable:"
		if !p.expectPeek(lexer.FROM) {
			return nil
		}

		if !p.expectPeek(lexer.SHELL) {
			return nil
		}

		// Check if this is multiline syntax (as $var:) or single-line syntax ("command" as $var)
		if p.peekToken.Type == lexer.AS {
			// Multiline syntax: "capture from shell as $variable:"
			if !p.expectPeek(lexer.AS) {
				return nil
			}

			if !p.expectPeekVariableName() {
				return nil
			}
			stmt.Variable = p.curToken.Literal

			if !p.expectPeek(lexer.COLON) {
				return nil
			}

			// Parse multiline commands
			return p.parseMultilineShellCapture(stmt)
		} else {
			// Single-line syntax: "capture from shell "command" as $variable"
			if !p.expectPeek(lexer.STRING) {
				return nil
			}
			command := p.curToken.Literal

			if !p.expectPeek(lexer.AS) {
				return nil
			}

			if !p.expectPeekVariableName() {
				return nil
			}
			stmt.Variable = p.curToken.Literal

			// Mark this as a shell capture by setting a special operation
			stmt.Operation = "capture_shell"

			// Create a literal expression for the command
			stmt.Value = &ast.LiteralExpression{
				Token: p.curToken,
				Value: command,
			}
			return stmt
		}
	case lexer.IDENT:
		// Expression syntax: "capture variable from expression"
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer.FROM) {
			return nil
		}

		// Parse the expression after "from"
		p.nextToken()
		stmt.Value = p.parseExpression()
		return stmt
	default:
		p.addError("expected 'from shell' or variable name after 'capture'")
		return nil
	}
}

// parseMultilineShellCapture parses multiline shell capture commands
func (p *Parser) parseMultilineShellCapture(stmt *ast.VariableStatement) *ast.VariableStatement {
	// Mark this as a shell capture by setting a special operation
	stmt.Operation = "capture_shell"

	// Expect INDENT (skip any newlines first)
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}

	// Parse command tokens until DEDENT
	p.nextToken() // Move to first token inside the block

	// Read all commands in the block
	commands := p.readCommandTokens()

	// Join commands with newlines to create a single script
	script := strings.Join(commands, "\n")

	// Create a literal expression for the combined script
	stmt.Value = &ast.LiteralExpression{
		Token: p.curToken,
		Value: script,
	}

	return stmt
}
