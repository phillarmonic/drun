package parser

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
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
		return p.parseDockerComposeStatement(stmt)
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

func (p *Parser) parseDockerComposeStatement(stmt *ast.DockerStatement) *ast.DockerStatement {
	// Optional: docker compose in service "<name>"
	if p.peekToken.Type == lexer.IN {
		p.nextToken() // consume IN
		if p.peekToken.Type == lexer.SERVICE || (p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "service") {
			p.nextToken()
		} else {
			p.addError("expected 'service' keyword after 'in'")
			return nil
		}

		name, isLiteral, ok := p.parseServiceReference()
		if !ok {
			return nil
		}
		stmt.ServiceScoped = true
		stmt.ServiceName = name
		stmt.ServiceNameIsLiteral = isLiteral
	}

	raw := p.collectInlineCommand()
	if raw != "" {
		stmt.Options["args"] = raw
		if _, exists := stmt.Options["command"]; !exists {
			fields := strings.Fields(raw)
			if len(fields) > 0 {
				stmt.Options["command"] = fields[0]
			}
		}
	}

	return stmt
}

func (p *Parser) collectInlineCommand() string {
	if p.peekToken.Type == lexer.NEWLINE || p.peekToken.Type == lexer.DEDENT || p.peekToken.Type == lexer.EOF {
		return ""
	}

	var builder strings.Builder
	lastWasMinus := false

collectLoop:
	for {
		switch p.peekToken.Type {
		case lexer.NEWLINE, lexer.DEDENT, lexer.EOF:
			break collectLoop
		case lexer.COMMENT, lexer.MULTILINE_COMMENT:
			p.nextToken()
			continue
		default:
			p.nextToken()
			tok := p.curToken
			if tok.Type == lexer.MINUS {
				if builder.Len() > 0 && !lastWasMinus {
					builder.WriteByte(' ')
				}
				builder.WriteByte('-')
				lastWasMinus = true
				continue
			}

			literal := tok.Literal
			if tok.Type == lexer.STRING {
				literal = fmt.Sprintf("\"%s\"", tok.Literal)
			}

			if builder.Len() > 0 && !lastWasMinus {
				builder.WriteByte(' ')
			}
			builder.WriteString(literal)
			lastWasMinus = false
		}
	}

	return strings.TrimSpace(builder.String())
}
