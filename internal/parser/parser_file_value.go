package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func (p *Parser) isFileValueStatementStart() bool {
	if p.curToken.Type != lexer.GET && p.curToken.Type != lexer.CHECK && p.curToken.Type != lexer.UPDATE {
		return false
	}
	return fileValueFormat(p.peekToken)
}

func fileValueFormat(token lexer.Token) bool {
	switch token.Type {
	case lexer.PROPERTY, lexer.JSON, lexer.MATCH:
		return true
	case lexer.IDENT:
		return token.Literal == "yaml" || token.Literal == "toml"
	default:
		return false
	}
}

func (p *Parser) parseFileValueStatement() *ast.FileValueStatement {
	stmt := &ast.FileValueStatement{Token: p.curToken, Operation: p.curToken.Literal}
	p.nextToken()
	if !fileValueFormat(p.curToken) {
		p.addError(fmt.Sprintf("expected file value format after %q", stmt.Operation))
		return nil
	}
	stmt.Format = p.curToken.Literal
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Selector = p.curToken.Literal

	switch stmt.Operation {
	case "get":
		if !p.expectPeek(lexer.FROM) || !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Target = p.curToken.Literal
		if !p.expectPeek(lexer.AS) || !p.expectPeekVariableName() {
			return nil
		}
		stmt.CaptureVar = p.getVariableName()
	case "check":
		if !p.expectPeek(lexer.IN) || !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Target = p.curToken.Literal
		p.nextToken()
		switch p.curToken.Literal {
		case "equals":
			stmt.Comparison = "equals"
		case "differs":
			stmt.Comparison = "differs"
			if p.peekToken.Type != lexer.FROM {
				p.addError("expected 'from' after 'differs'")
				return nil
			}
			p.nextToken()
		default:
			p.addError("expected 'equals' or 'differs from' in file value check")
			return nil
		}
		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Expected = p.curToken.Literal
	case "update":
		if !p.expectPeek(lexer.IN) || !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Target = p.curToken.Literal
		if !p.expectPeek(lexer.TO) || !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Value = p.curToken.Literal
		if !p.expectPeek(lexer.OR) {
			return nil
		}
		p.nextToken()
		switch p.curToken.Type {
		case lexer.FAIL:
			stmt.MissingPolicy = "fail"
		case lexer.ADD:
			stmt.MissingPolicy = "add"
			if p.peekToken.Type == lexer.AS {
				p.nextToken()
				p.nextToken()
				switch p.curToken.Type {
				case lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN_TYPE:
					stmt.ValueType = p.curToken.Literal
				default:
					p.addError("expected string, number, or boolean after 'or add as'")
					return nil
				}
			}
		default:
			p.addError("expected 'fail' or 'add' after 'or'")
			return nil
		}
		if stmt.Format == "match" && stmt.MissingPolicy == "add" {
			p.addError("regex match updates do not support 'or add'")
			return nil
		}
		if stmt.Format != "property" && stmt.MissingPolicy == "add" && stmt.ValueType == "" {
			p.addError("structured additions require 'as string', 'as number', or 'as boolean'")
			return nil
		}
	}
	return stmt
}
