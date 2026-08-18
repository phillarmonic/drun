package parser

import (
	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// parseWaitStatement parses a fixed-duration wait:
//
//	wait 5 seconds
//	wait {retry_count} minutes
//	wait 1 hour
func (p *Parser) parseWaitStatement() *ast.WaitStatement {
	stmt := &ast.WaitStatement{Token: p.curToken}

	// Value: a number literal or a {variable} interpolation
	switch p.peekToken.Type {
	case lexer.NUMBER:
		p.nextToken()
		stmt.Value = p.curToken.Literal
	case lexer.LBRACE:
		p.nextToken() // consume {
		if p.peekToken.Type != lexer.VARIABLE && p.peekToken.Type != lexer.IDENT {
			p.addErrorWithHelpAtPeek("expected variable name in wait duration interpolation",
				"use wait {$variable} <unit>, e.g. wait {$retries} seconds")
			return nil
		}
		p.nextToken()
		stmt.Value = "{" + p.curToken.Literal + "}"
		if p.peekToken.Type != lexer.RBRACE {
			p.addErrorWithHelpAtPeek("expected } after variable in wait duration",
				"use wait {$variable} <unit>, e.g. wait {$retries} seconds")
			return nil
		}
		p.nextToken() // consume }
	default:
		p.addErrorWithHelpAtPeek("expected a number or {$variable} after wait",
			"use wait <number> <unit> or wait {$variable} <unit>, e.g. wait 5 seconds")
		return nil
	}

	// Unit: second/s, minute/s, hour/s
	switch p.peekToken.Type {
	case lexer.SECOND, lexer.SECONDS:
		stmt.Unit = "second"
	case lexer.MINUTE, lexer.MINUTES:
		stmt.Unit = "minute"
	case lexer.HOUR, lexer.HOURS:
		stmt.Unit = "hour"
	default:
		p.addErrorWithHelpAtPeek("expected a time unit after wait duration",
			"use second(s), minute(s), or hour(s), e.g. wait 5 seconds")
		return nil
	}
	p.nextToken() // consume unit

	return stmt
}
