package parser

import (
	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

// parseNetworkStatement parses network operations (health checks, port testing, ping)
func (p *Parser) parseNetworkStatement() *ast.NetworkStatement {
	stmt := &ast.NetworkStatement{
		Token:   p.curToken,
		Options: make(map[string]string),
	}

	// Determine network action based on current token and context
	switch p.curToken.Type {
	case lexer.WAIT:
		// "wait for service at URL to be ready"
		stmt.Action = "wait_for_service"

		// Expect "for service at"
		if p.peekToken.Type == lexer.FOR {
			p.nextToken() // consume FOR
			if p.peekToken.Type == lexer.SERVICE {
				p.nextToken() // consume SERVICE
				if p.peekToken.Type == lexer.AT {
					p.nextToken() // consume AT
					if p.peekToken.Type == lexer.STRING {
						p.nextToken()
						stmt.Target = p.curToken.Literal

						// Expect "to be ready"
						if p.peekToken.Type == lexer.TO {
							p.nextToken() // consume TO
							if p.peekToken.Type == lexer.BE {
								p.nextToken() // consume BE
								if p.peekToken.Type == lexer.READY {
									p.nextToken() // consume READY
								}
							}
						}
					}
				}
			}
		}

	case lexer.PING:
		// "ping host hostname"
		stmt.Action = "ping"

		// Expect "host"
		if p.peekToken.Type == lexer.HOST {
			p.nextToken() // consume HOST
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Target = p.curToken.Literal
			}
		}

	case lexer.TEST:
		// "test connection to host on port X"
		stmt.Action = "port_check"

		// Expect "connection to"
		if p.peekToken.Type == lexer.CONNECTION {
			p.nextToken() // consume CONNECTION
			if p.peekToken.Type == lexer.TO {
				p.nextToken() // consume TO
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Target = p.curToken.Literal

					// Expect "on port X"
					if p.peekToken.Type == lexer.ON {
						p.nextToken() // consume ON
						if p.peekToken.Type == lexer.PORT {
							p.nextToken() // consume PORT
							if p.peekToken.Type == lexer.NUMBER {
								p.nextToken()
								stmt.Port = p.curToken.Literal
							}
						}
					}
				}
			}
		}

	case lexer.CHECK:
		// "check health of service at URL" or "check if port X is open on host"
		switch p.peekToken.Type {
		case lexer.HEALTH:
			p.nextToken() // consume HEALTH
			stmt.Action = "health_check"

			// Expect "of service at"
			if p.peekToken.Type == lexer.OF {
				p.nextToken() // consume OF
				if p.peekToken.Type == lexer.SERVICE {
					p.nextToken() // consume SERVICE
					if p.peekToken.Type == lexer.AT {
						p.nextToken() // consume AT
						if p.peekToken.Type == lexer.STRING {
							p.nextToken()
							stmt.Target = p.curToken.Literal
						}
					}
				}
			}
		case lexer.IF:
			p.nextToken() // consume IF
			if p.peekToken.Type == lexer.PORT {
				p.nextToken() // consume PORT
				stmt.Action = "port_check"

				// Expect port number
				if p.peekToken.Type == lexer.NUMBER {
					p.nextToken()
					stmt.Port = p.curToken.Literal

					// Expect "is open on"
					if p.peekToken.Type == lexer.IS {
						p.nextToken() // consume IS
						if p.peekToken.Type == lexer.OPEN {
							p.nextToken() // consume OPEN
							if p.peekToken.Type == lexer.ON {
								p.nextToken() // consume ON
								if p.peekToken.Type == lexer.STRING {
									p.nextToken()
									stmt.Target = p.curToken.Literal
								}
							}
						}
					}
				}
			}
		}
	}

	// Parse additional options (timeout, retry, expect, etc.)
	for p.peekToken.Type == lexer.TIMEOUT || p.peekToken.Type == lexer.RETRY ||
		p.peekToken.Type == lexer.EXPECT || p.peekToken.Type == lexer.WITH {
		p.nextToken()

		switch p.curToken.Type {
		case lexer.TIMEOUT:
			if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.NUMBER {
				p.nextToken()
				stmt.Options["timeout"] = p.curToken.Literal
			}
		case lexer.RETRY:
			if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.NUMBER {
				p.nextToken()
				stmt.Options["retry"] = p.curToken.Literal
			}
		case lexer.EXPECT:
			if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.NUMBER {
				p.nextToken()
				stmt.Condition = p.curToken.Literal
			}
		case lexer.WITH:
			// Handle "with" options
			if p.peekToken.Type == lexer.IDENT {
				p.nextToken()
				optionKey := p.curToken.Literal
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Options[optionKey] = p.curToken.Literal
				}
			}
		}
	}

	return stmt
}
