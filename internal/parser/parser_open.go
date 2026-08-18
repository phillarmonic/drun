package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// parseOpenStatement parses:
//
//	open url "https://example.com"
//	open url "{$base_url}/docs"
//	open url "docs/index.html"
func (p *Parser) parseOpenStatement() *ast.OpenStatement {
	stmt := &ast.OpenStatement{Token: p.curToken}

	// Noun: only "url" (lexer.URL keyword token) is supported for now.
	switch p.peekToken.Type {
	case lexer.URL:
		p.nextToken()
		stmt.Noun = "url"
	case lexer.STRING:
		p.addErrorWithHelpAtPeek("expected a noun after 'open'",
			"use open url \"<target>\", e.g. open url \"https://example.com\"")
		return nil
	default:
		p.addErrorWithHelpAtPeek(fmt.Sprintf("unknown noun %q for 'open'", p.peekToken.Literal),
			"only \"url\" is currently supported: open url \"<target>\"")
		return nil
	}

	// Target: a quoted URL or file path (interpolation happens at execution time)
	if p.peekToken.Type != lexer.STRING {
		p.addErrorWithHelpAtPeek("expected a URL or file path string after 'open url'",
			"use open url \"<target>\", e.g. open url \"https://example.com\" or open url \"./report.html\"")
		return nil
	}
	p.nextToken()
	stmt.URL = p.curToken.Literal

	return stmt
}
