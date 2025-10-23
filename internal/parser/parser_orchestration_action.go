package parser

import (
	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

// parseOrchestrationActionStatement parses orchestration actions in task bodies
// Examples: orchestrate "group" start, orchestrate "group" stop
func (p *Parser) parseOrchestrationActionStatement() *ast.OrchestrationActionStatement {
	stmt := &ast.OrchestrationActionStatement{
		Token:   p.curToken, // ORCHESTRATE token
		Options: make(map[string]string),
	}

	// Expect group name as string
	if !p.expectPeek(lexer.STRING) {
		p.addError("expected orchestration group name as string after 'orchestrate'")
		return nil
	}
	stmt.GroupName = p.curToken.Literal

	// Parse action (start, stop, restart, health_check, status, logs, etc.)
	switch p.peekToken.Type {
	case lexer.START:
		p.nextToken()
		stmt.Action = "start"
	case lexer.STOP:
		p.nextToken()
		stmt.Action = "stop"
	case lexer.RESTART:
		p.nextToken()
		stmt.Action = "restart"
	case lexer.STATUS:
		p.nextToken()
		stmt.Action = "status"
	case lexer.BUILD:
		p.nextToken()
		stmt.Action = "build"
	case lexer.PULL:
		p.nextToken()
		stmt.Action = "pull"
	case lexer.DOWN:
		p.nextToken()
		stmt.Action = "down"
	case lexer.UPDATE:
		p.nextToken()
		stmt.Action = "update"
	case lexer.SCALE:
		p.nextToken()
		stmt.Action = "scale"
	case lexer.IDENT:
		// Allow identifier for additional actions like "health_check", "logs", "sync"
		p.nextToken()
		switch p.curToken.Literal {
		case "health_check", "health", "logs", "sync", "clone_repositories":
			stmt.Action = p.curToken.Literal
		default:
			p.addError("unknown orchestration action: " + p.curToken.Literal)
			return nil
		}
	default:
		p.addError("expected orchestration action (start, stop, restart, etc.)")
		return nil
	}

	// Parse optional modifiers and options
	for {
		switch p.peekToken.Type {
		case lexer.SERVICES:
			// orchestrate "group" start services ["service1", "service2"]
			p.nextToken() // consume SERVICES
			stmt.ServiceFilters = p.parseOrchestrationStringArray()
		case lexer.SERVICE:
			p.nextToken() // consume SERVICE
			if p.expectPeek(lexer.STRING) {
				stmt.ServiceFilters = append(stmt.ServiceFilters, p.curToken.Literal)
			}
		case lexer.WITH:
			// orchestrate "group" start with timeout "30s"
			p.nextToken() // consume WITH
			// Parse key-value pairs
			for {
				if p.peekToken.Type == lexer.IDENT {
					p.nextToken()
					key := p.curToken.Literal
					if p.peekToken.Type == lexer.STRING {
						p.nextToken()
						stmt.Options[key] = p.curToken.Literal
					}
				}
				if p.peekToken.Type == lexer.COMMA {
					p.nextToken()
					continue
				}
				break
			}
		case lexer.TIMEOUT:
			p.nextToken() // consume TIMEOUT
			if p.expectPeek(lexer.STRING) {
				stmt.Options["timeout"] = p.curToken.Literal
			}
		case lexer.WAIT:
			// orchestrate "group" start wait for "healthy"
			p.nextToken() // consume WAIT
			if p.peekToken.Type == lexer.FOR {
				p.nextToken() // consume FOR
				if p.expectPeek(lexer.STRING) {
					stmt.Options["wait_for"] = p.curToken.Literal
				}
			}
		default:
			// No more options
			return stmt
		}
	}
}
