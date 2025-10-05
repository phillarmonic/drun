package parser

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

// Expression parsing methods

// parseExpression parses expressions
func (p *Parser) parseExpression() ast.Expression {
	return p.parseInfixExpression()
}

// parseInfixExpression parses binary expressions with operator precedence
func (p *Parser) parseInfixExpression() ast.Expression {
	left := p.parsePrimaryExpression()
	if left == nil {
		return nil
	}

	// Check for binary operators
	for p.peekToken.Type == lexer.PLUS || p.peekToken.Type == lexer.MINUS ||
		p.peekToken.Type == lexer.STAR || p.peekToken.Type == lexer.SLASH ||
		p.peekToken.Type == lexer.EQUALS ||
		p.peekToken.Type == lexer.GT || p.peekToken.Type == lexer.LT ||
		p.peekToken.Type == lexer.GTE || p.peekToken.Type == lexer.LTE ||
		p.peekToken.Type == lexer.EQ || p.peekToken.Type == lexer.NE {

		operator := p.peekToken.Literal
		p.nextToken() // consume operator
		p.nextToken() // move to right operand

		right := p.parsePrimaryExpression()
		if right == nil {
			return nil
		}

		left = &ast.BinaryExpression{
			Token:    p.curToken,
			Left:     left,
			Operator: operator,
			Right:    right,
		}
	}

	return left
}

// parsePrimaryExpression parses primary expressions (literals, identifiers, function calls)
func (p *Parser) parsePrimaryExpression() ast.Expression {
	switch p.curToken.Type {
	case lexer.STRING:
		return &ast.LiteralExpression{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
	case lexer.NUMBER:
		return &ast.LiteralExpression{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
	case lexer.BOOLEAN:
		return &ast.LiteralExpression{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
	case lexer.LBRACE:
		// Parse {expression} - could be single identifier or multi-word expression
		p.nextToken() // consume LBRACE

		var parts []string

		// Read tokens until RBRACE
		for p.curToken.Type != lexer.RBRACE && p.curToken.Type != lexer.EOF {
			parts = append(parts, p.curToken.Literal)
			p.nextToken()
		}

		if p.curToken.Type != lexer.RBRACE {
			p.addError("expected '}' to close brace expression")
			return nil
		}

		// Join the parts to create the expression and preserve braces for interpolation
		expression := "{" + strings.Join(parts, " ") + "}"

		return &ast.LiteralExpression{
			Token: p.curToken,
			Value: expression,
		}
	case lexer.IDENT:
		// Handle function calls like "now" or simple identifiers
		if p.peekToken.Type == lexer.LPAREN {
			// Function call with parentheses
			return p.parseFunctionCall()
		} else {
			// Simple function call or identifier
			return &ast.FunctionCallExpression{
				Token:     p.curToken,
				Function:  p.curToken.Literal,
				Arguments: []ast.Expression{},
			}
		}
	case lexer.VARIABLE:
		// Handle $variable references
		return &ast.IdentifierExpression{
			Token: p.curToken,
			Value: p.curToken.Literal, // This includes the $ prefix
		}
	case lexer.LBRACKET:
		// Parse array literal ["item1", "item2", "item3"]
		return p.parseArrayLiteral()
	default:
		p.addError(fmt.Sprintf("unexpected token in expression: %s", p.curToken.Type))
		return nil
	}
}

// parseFunctionCall parses function calls with arguments
func (p *Parser) parseFunctionCall() ast.Expression {
	function := p.curToken.Literal

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	var args []ast.Expression
	if p.peekToken.Type != lexer.RPAREN {
		p.nextToken()
		args = append(args, p.parseExpression())

		for p.peekToken.Type == lexer.COMMA {
			p.nextToken() // consume comma
			p.nextToken() // move to next argument
			args = append(args, p.parseExpression())
		}
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return &ast.FunctionCallExpression{
		Token:     p.curToken,
		Function:  function,
		Arguments: args,
	}
}

// parseArrayLiteral parses array literals like ["item1", "item2", "item3"]
func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{
		Token:    p.curToken, // LBRACKET
		Elements: []ast.Expression{},
	}

	// Handle empty array []
	if p.peekToken.Type == lexer.RBRACKET {
		p.nextToken() // consume RBRACKET
		return array
	}

	// Parse first element
	p.nextToken()
	array.Elements = append(array.Elements, p.parseExpression())

	// Parse remaining elements separated by commas
	for p.peekToken.Type == lexer.COMMA {
		p.nextToken() // consume comma
		p.nextToken() // move to next element
		array.Elements = append(array.Elements, p.parseExpression())
	}

	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	return array
}
