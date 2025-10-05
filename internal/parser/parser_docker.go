package parser

import (
	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func (p *Parser) parseDockerStatement() *ast.DockerStatement {
	stmt := &ast.DockerStatement{
		Token:   p.curToken,
		Options: make(map[string]string),
	}

	// Parse operation (build, push, pull, run, etc.)
	switch p.peekToken.Type {
	case lexer.BUILD, lexer.PUSH, lexer.PULL, lexer.TAG, lexer.REMOVE, lexer.START, lexer.STOP, lexer.RUN:
		p.nextToken()
		stmt.Operation = p.curToken.Literal
	case lexer.COMPOSE:
		p.nextToken()
		stmt.Operation = "compose"
		stmt.Resource = "compose"

		// Parse compose command (up, down, build, etc.)
		if p.peekToken.Type == lexer.UP || p.peekToken.Type == lexer.DOWN || p.peekToken.Type == lexer.BUILD {
			p.nextToken()
			stmt.Options["command"] = p.curToken.Literal
		}
		return stmt
	case lexer.SCALE:
		// Handle "docker compose scale service "name" to 3"
		p.nextToken() // consume SCALE
		stmt.Operation = "scale"

		if p.peekToken.Type == lexer.COMPOSE {
			p.nextToken() // consume COMPOSE
			stmt.Resource = "compose"

			if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "service" {
				p.nextToken() // consume IDENT (service)
				stmt.Options["resource"] = "service"

				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Name = p.curToken.Literal
				}

				if p.peekToken.Type == lexer.TO {
					p.nextToken() // consume TO
					if p.peekToken.Type == lexer.NUMBER {
						p.nextToken()
						stmt.Options["replicas"] = p.curToken.Literal
					}
				}
			}
		}
		return stmt
	case lexer.IDENT:
		p.nextToken()
		stmt.Operation = p.curToken.Literal
	default:
		return nil
	}

	// Parse resource type (image, container)
	switch p.peekToken.Type {
	case lexer.IMAGE, lexer.CONTAINER:
		p.nextToken()
		stmt.Resource = p.curToken.Literal
	case lexer.IDENT:
		p.nextToken()
		stmt.Resource = p.curToken.Literal
	default:
		return nil
	}

	// Parse name (optional for some operations)
	if p.peekToken.Type == lexer.STRING {
		p.nextToken()
		stmt.Name = p.curToken.Literal
	}

	// Parse additional options (from, to, as, on, etc.)
	for p.peekToken.Type == lexer.FROM || p.peekToken.Type == lexer.TO || p.peekToken.Type == lexer.AS || p.peekToken.Type == lexer.ON || p.peekToken.Type == lexer.PORT || p.peekToken.Type == lexer.IDENT {
		p.nextToken()
		optionKey := p.curToken.Literal

		if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.NUMBER {
			p.nextToken()
			stmt.Options[optionKey] = p.curToken.Literal
		} else if optionKey == "on" && p.peekToken.Type == lexer.PORT {
			p.nextToken() // consume PORT
			if p.peekToken.Type == lexer.NUMBER {
				p.nextToken()
				stmt.Options["port"] = p.curToken.Literal
			}
		}
	}

	return stmt
}
