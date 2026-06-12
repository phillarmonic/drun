package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// parseRequiresToolsStatement parses a "requires tools:" block.
// The current token is REQUIRES when this is called.
//
// Syntax:
//
//	requires tools:
//	    golangci-lint >= "2.12"
//	    gosec >= "2.27" <= "3.0"
//	    docker
func (p *Parser) parseRequiresToolsStatement() *ast.RequiresToolsStatement {
	stmt := &ast.RequiresToolsStatement{Token: p.curToken}

	// Current token is REQUIRES, peek should be TOOLS
	if !p.expectPeek(lexer.TOOLS) {
		return nil
	}

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Expect indented block (skip any newlines first)
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}

	// Move to first token inside the block
	p.nextToken()

	// Parse tool requirements until DEDENT
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		// Skip newlines and comments
		if p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT {
			p.nextToken()
			continue
		}

		// Parse a tool requirement line
		toolReq := p.parseToolRequirement()
		if toolReq != nil {
			stmt.Tools = append(stmt.Tools, *toolReq)
		} else {
			// On error, advance to avoid infinite loop
			p.nextToken()
		}
	}

	// Do not advance past DEDENT here. The task parser expects to be left on the last token of the statement.
	// The project parser will manually advance past DEDENT.

	if len(stmt.Tools) == 0 {
		p.addError("requires tools: block must contain at least one tool requirement")
		return nil
	}

	return stmt
}

// parseToolRequirement parses a single tool requirement line.
// Examples:
//
//	gosec
//	gosec >= "2.27"
//	gosec >= "2.27" <= "3.0"
//	golangci-lint >= "2.12"
func (p *Parser) parseToolRequirement() *ast.ToolRequirement {
	// Tool name can be an IDENT or a keyword token that's also a tool name
	// (e.g., GO, DOCKER, GIT, NPM, etc.)
	toolName, ok := p.parseToolName()
	if !ok {
		return nil
	}

	req := &ast.ToolRequirement{
		Name: toolName,
	}

	// Parse zero or more version constraints: operator version
	for p.curToken.Type == lexer.GTE || p.curToken.Type == lexer.GT ||
		p.curToken.Type == lexer.LTE || p.curToken.Type == lexer.LT {

		operator := p.curToken.Literal

		// Expect version string or number
		p.nextToken()
		var version string
		switch p.curToken.Type {
		case lexer.STRING:
			version = p.curToken.Literal
		case lexer.NUMBER:
			version = p.curToken.Literal
		default:
			p.addError(fmt.Sprintf("expected version string or number after '%s', got %s instead", operator, p.curToken.Type))
			return nil
		}

		req.Constraints = append(req.Constraints, ast.VersionConstraint{
			Operator: operator,
			Version:  version,
		})

		// Advance to see if there's another constraint
		p.nextToken()
	}

	return req
}

// parseToolName parses a tool name, handling dashed names like "golangci-lint".
// Returns the tool name and true on success.
func (p *Parser) parseToolName() (string, bool) {
	// Accept IDENT or keyword tokens that represent tool names
	if !p.isToolNameToken(p.curToken.Type) {
		p.addError(fmt.Sprintf("expected tool name, got %s instead", p.curToken.Type))
		return "", false
	}

	name := p.curToken.Literal

	// Handle dashed names like "golangci-lint"
	for p.peekToken.Type == lexer.MINUS {
		p.nextToken() // consume MINUS

		// Next token must be another name part
		p.nextToken()
		if !p.isToolNameToken(p.curToken.Type) {
			p.addError("expected identifier after '-' in tool name")
			return "", false
		}
		name += "-" + p.curToken.Literal
	}

	// Advance past the tool name to the next token (operator or next line)
	p.nextToken()

	return name, true
}

// isToolNameToken checks if a token type can be used as a tool name (or part of one).
func (p *Parser) isToolNameToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.IDENT:
		return true
	// Well-known tool keywords
	case lexer.DOCKER, lexer.GIT, lexer.GO, lexer.GOLANG,
		lexer.NODE, lexer.NPM, lexer.YARN, lexer.PNPM, lexer.BUN,
		lexer.PYTHON, lexer.PIP,
		lexer.CARGO, lexer.RUST,
		lexer.JAVA, lexer.MAVEN, lexer.GRADLE,
		lexer.RUBY, lexer.GEM,
		lexer.PHP, lexer.COMPOSER,
		lexer.KUBECTL, lexer.HELM, lexer.TERRAFORM,
		lexer.AWS, lexer.GCP, lexer.AZURE,
		lexer.MAKE, lexer.TOOL:
		return true
	default:
		return false
	}
}
