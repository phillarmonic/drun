package parser

import (
	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// isChangelogStatementStart reports whether the current position starts a
// changelog promotion statement: promote changelog ...
func (p *Parser) isChangelogStatementStart() bool {
	return p.curToken.Type == lexer.PROMOTE && p.peekToken.Type == lexer.CHANGELOG
}

// parseChangelogStatement parses a Keep a Changelog promotion:
//
//	promote changelog "CHANGELOG.md" to version "1.5.0"
//	promote changelog "CHANGELOG.md" to version "{$release_version}" on "2026-09-01"
func (p *Parser) parseChangelogStatement() *ast.ChangelogStatement {
	stmt := &ast.ChangelogStatement{Token: p.curToken}

	p.nextToken() // consume 'changelog'
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Path = p.curToken.Literal

	if !p.expectPeek(lexer.TO) || !p.expectPeek(lexer.VERSION) {
		return nil
	}
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Version = p.curToken.Literal

	if p.peekToken.Type == lexer.ON {
		p.nextToken() // consume 'on'
		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Date = p.curToken.Literal
	}

	return stmt
}
