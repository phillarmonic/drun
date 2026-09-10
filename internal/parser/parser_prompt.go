package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// parseConfirmStatement parses an interactive y/n confirmation gate:
//
//	confirm "Deploy to production?"
//	confirm "Delete build cache?" defaults to "no"
//	confirm "Run database migrations?" as $migrate
//	confirm "Continue?" defaults to "yes" as $continue
//
// The question string is required. The optional clauses must appear in the
// documented order: `defaults to <value>` first, then `as $var`.
func (p *Parser) parseConfirmStatement() *ast.ConfirmStatement {
	stmt := &ast.ConfirmStatement{Token: p.curToken}

	// Question: confirm "<question>"
	if p.peekToken.Type != lexer.STRING {
		p.addErrorWithHelpAtPeek(
			"expected a question string after 'confirm'",
			"use confirm \"<question>\", e.g. confirm \"Deploy to production?\"",
		)
		return nil
	}
	p.nextToken()
	stmt.Question = p.curToken.Literal

	defaultValue, hasDefault, resultVar, ok := p.parsePromptAnswerOptions()
	if !ok {
		return nil
	}
	stmt.DefaultValue = defaultValue
	stmt.HasDefault = hasDefault
	stmt.ResultVar = resultVar

	return stmt
}

// parsePromptStatement parses a free-form interactive text prompt:
//
//	prompt "Which environment?" as $environment
//	prompt "Release notes?" defaults to "n/a" as $notes
//
// The question string is required. The optional clauses must appear in the
// documented order: `defaults to <value>` first, then `as $var`.
func (p *Parser) parsePromptStatement() *ast.PromptStatement {
	stmt := &ast.PromptStatement{Token: p.curToken}

	// Question: prompt "<question>"
	if p.peekToken.Type != lexer.STRING {
		p.addErrorWithHelpAtPeek(
			"expected a question string after 'prompt'",
			"use prompt \"<question>\" as $var, e.g. prompt \"Which environment?\" as $environment",
		)
		return nil
	}
	p.nextToken()
	stmt.Question = p.curToken.Literal

	defaultValue, hasDefault, resultVar, ok := p.parsePromptAnswerOptions()
	if !ok {
		return nil
	}
	stmt.DefaultValue = defaultValue
	stmt.HasDefault = hasDefault
	stmt.ResultVar = resultVar

	return stmt
}

// parsePromptAnswerOptions parses the optional clauses shared by confirm and
// prompt statements, in the documented order:
//
//	[defaults to <value>] [as $var]
//
// The current token must be the consumed question string. Both clauses are
// optional; on success ok is true and the returned defaultValue/hasDefault/
// resultVar carry the parsed values (resultVar has no '$' prefix).
func (p *Parser) parsePromptAnswerOptions() (defaultValue string, hasDefault bool, resultVar string, ok bool) {
	// Optional "defaults to <value>"
	if p.peekToken.Type == lexer.DEFAULTS {
		p.nextToken() // consume DEFAULTS
		if p.peekToken.Type != lexer.TO {
			p.addErrorWithHelpAtPeek(
				fmt.Sprintf("expected 'to' after 'defaults', got %s instead", p.peekToken.Type),
				"use 'defaults to' to declare a default answer, e.g. confirm \"Continue?\" defaults to \"yes\"",
			)
			return "", false, "", false
		}
		p.nextToken() // consume TO

		// Default answer: a quoted/plain value (string, number, boolean) or a
		// bare yes/no. Bare "no" is the lexer.NO keyword; bare "yes" lexes as
		// an IDENT because "yes" is not a reserved keyword.
		switch p.peekToken.Type {
		case lexer.STRING, lexer.NUMBER, lexer.BOOLEAN, lexer.NO:
			p.nextToken()
			defaultValue = p.curToken.Literal
			hasDefault = true
		case lexer.IDENT:
			if p.peekToken.Literal == "yes" {
				p.nextToken()
				defaultValue = p.curToken.Literal
				hasDefault = true
			} else {
				p.addErrorWithHelpAtPeek(
					fmt.Sprintf("expected a default answer after 'defaults to', got %q instead", p.peekToken.Literal),
					"use a quoted value or a bare yes/no, e.g. defaults to \"no\"",
				)
				return "", false, "", false
			}
		default:
			p.addErrorWithHelpAtPeek(
				fmt.Sprintf("expected a default answer after 'defaults to', got %s instead", p.peekToken.Type),
				"use a quoted value or a bare yes/no, e.g. defaults to \"no\"",
			)
			return "", false, "", false
		}
	}

	// Optional "as $var"
	if p.peekToken.Type == lexer.AS {
		p.nextToken() // consume AS
		if !p.expectPeekVariableName() {
			return "", false, "", false
		}
		resultVar = p.getVariableName()
	}

	return defaultValue, hasDefault, resultVar, true
}
