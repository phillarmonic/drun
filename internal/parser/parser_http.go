package parser

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

// parseHTTPStatement parses HTTP operations
func (p *Parser) parseHTTPStatement() ast.Statement {
	// Handle DOWNLOAD as a separate statement type
	if p.curToken.Type == lexer.DOWNLOAD {
		return p.parseDownloadStatement()
	}

	stmt := &ast.HTTPStatement{
		Token:   p.curToken,
		Headers: make(map[string]string),
		Auth:    make(map[string]string),
		Options: make(map[string]string),
	}

	// Determine HTTP method
	switch p.curToken.Type {
	case lexer.GET:
		stmt.Method = "GET"
	case lexer.POST:
		stmt.Method = "POST"
	case lexer.PUT:
		stmt.Method = "PUT"
	case lexer.DELETE:
		stmt.Method = "DELETE"
	case lexer.PATCH:
		stmt.Method = "PATCH"
	case lexer.HEAD:
		stmt.Method = "HEAD"
	case lexer.OPTIONS:
		stmt.Method = "OPTIONS"
	case lexer.HTTP, lexer.HTTPS:
		// For "http request" or "https request" syntax
		if p.peekToken.Type == lexer.REQUEST {
			p.nextToken()       // consume REQUEST
			stmt.Method = "GET" // default to GET
		} else {
			return nil
		}
	default:
		return nil
	}

	// Parse URL/endpoint
	switch p.peekToken.Type {
	case lexer.REQUEST:
		p.nextToken() // consume REQUEST
		if p.peekToken.Type == lexer.TO {
			p.nextToken() // consume TO
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.URL = p.curToken.Literal
			}
		}
	case lexer.TO, lexer.STRING:
		if p.peekToken.Type == lexer.TO {
			p.nextToken() // consume TO
		}
		if p.peekToken.Type == lexer.STRING {
			p.nextToken()
			stmt.URL = p.curToken.Literal
		}
	}

	// Parse additional options (headers, body, auth, etc.)
	for p.peekToken.Type == lexer.WITH || p.peekToken.Type == lexer.HEADER || p.peekToken.Type == lexer.HEADERS ||
		p.peekToken.Type == lexer.BODY || p.peekToken.Type == lexer.DATA || p.peekToken.Type == lexer.AUTH ||
		p.peekToken.Type == lexer.BEARER || p.peekToken.Type == lexer.BASIC || p.peekToken.Type == lexer.TOKEN ||
		p.peekToken.Type == lexer.TIMEOUT || p.peekToken.Type == lexer.RETRY || p.peekToken.Type == lexer.ACCEPT ||
		p.peekToken.Type == lexer.CONTENT || p.peekToken.Type == lexer.TYPE {

		p.nextToken()

		switch p.curToken.Type {
		case lexer.WITH:
			// Parse "with header", "with body", "with auth", etc.
			switch p.peekToken.Type {
			case lexer.HEADER:
				p.nextToken() // consume HEADER
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					headerValue := p.curToken.Literal
					// Parse "key: value" format
					if colonIdx := strings.Index(headerValue, ":"); colonIdx != -1 {
						key := strings.TrimSpace(headerValue[:colonIdx])
						value := strings.TrimSpace(headerValue[colonIdx+1:])
						stmt.Headers[key] = value
					}
				}
			case lexer.BODY, lexer.DATA:
				p.nextToken() // consume BODY/DATA
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Body = p.curToken.Literal
				}
			case lexer.AUTH:
				p.nextToken() // consume AUTH
				if p.peekToken.Type == lexer.BEARER || p.peekToken.Type == lexer.BASIC {
					p.nextToken()
					authType := p.curToken.Literal
					if p.peekToken.Type == lexer.STRING {
						p.nextToken()
						stmt.Auth[authType] = p.curToken.Literal
					}
				}
			case lexer.TOKEN:
				p.nextToken() // consume TOKEN
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Auth["bearer"] = p.curToken.Literal
				}
			}

		case lexer.HEADER, lexer.HEADERS:
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				headerValue := p.curToken.Literal
				// Parse "key: value" format
				if colonIdx := strings.Index(headerValue, ":"); colonIdx != -1 {
					key := strings.TrimSpace(headerValue[:colonIdx])
					value := strings.TrimSpace(headerValue[colonIdx+1:])
					stmt.Headers[key] = value
				}
			}

		case lexer.BODY, lexer.DATA:
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Body = p.curToken.Literal
			}

		case lexer.AUTH:
			if p.peekToken.Type == lexer.BEARER || p.peekToken.Type == lexer.BASIC {
				p.nextToken()
				authType := p.curToken.Literal
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Auth[authType] = p.curToken.Literal
				}
			}

		case lexer.BEARER, lexer.BASIC:
			authType := p.curToken.Literal
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Auth[authType] = p.curToken.Literal
			}

		case lexer.TOKEN:
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Auth["bearer"] = p.curToken.Literal
			}

		case lexer.TIMEOUT, lexer.RETRY:
			optionKey := p.curToken.Literal
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Options[optionKey] = p.curToken.Literal
			}

		case lexer.ACCEPT:
			switch p.peekToken.Type {
			case lexer.JSON, lexer.XML:
				p.nextToken()
				stmt.Headers["Accept"] = "application/" + p.curToken.Literal
			case lexer.STRING:
				p.nextToken()
				stmt.Headers["Accept"] = p.curToken.Literal
			}

		case lexer.CONTENT:
			if p.peekToken.Type == lexer.TYPE {
				p.nextToken() // consume TYPE
				switch p.peekToken.Type {
				case lexer.JSON, lexer.XML:
					p.nextToken()
					stmt.Headers["Content-Type"] = "application/" + p.curToken.Literal
				case lexer.STRING:
					p.nextToken()
					stmt.Headers["Content-Type"] = p.curToken.Literal
				}
			}
		}
	}

	return stmt
}

// parseDownloadStatement parses download operations
// Syntax: download "url" to "path" [allow overwrite] [with header "..."] [timeout "..."]
func (p *Parser) parseDownloadStatement() *ast.DownloadStatement {
	stmt := &ast.DownloadStatement{
		Token:   p.curToken,
		Headers: make(map[string]string),
		Auth:    make(map[string]string),
		Options: make(map[string]string),
	}

	// Parse URL
	if p.peekToken.Type == lexer.STRING {
		p.nextToken()
		stmt.URL = p.curToken.Literal
	} else {
		p.addError(fmt.Sprintf("expected URL string after 'download', got %s", p.peekToken.Type))
		return nil
	}

	// Parse "to path" (required) then optional "extract to"
	if p.peekToken.Type == lexer.TO {
		p.nextToken() // consume TO
		if p.peekToken.Type == lexer.STRING {
			p.nextToken()
			stmt.Path = p.curToken.Literal
		} else {
			p.addError(fmt.Sprintf("expected path string after 'to', got %s", p.peekToken.Type))
			return nil
		}
	} else {
		p.addError(fmt.Sprintf("expected 'to' after URL, got %s", p.peekToken.Type))
		return nil
	}

	// Check for optional "extract to"
	if p.peekToken.Type == lexer.EXTRACT {
		p.nextToken() // consume EXTRACT
		if p.peekToken.Type == lexer.TO {
			p.nextToken() // consume TO
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.ExtractTo = p.curToken.Literal
			} else {
				p.addError(fmt.Sprintf("expected directory path after 'extract to', got %s", p.peekToken.Type))
				return nil
			}
		} else {
			p.addError(fmt.Sprintf("expected 'to' after 'extract', got %s", p.peekToken.Type))
			return nil
		}
	}

	// Parse optional modifiers (allow overwrite, allow permissions, headers, auth, timeout, etc.)
	for {
		switch p.peekToken.Type {
		case lexer.ALLOW:
			p.nextToken() // consume ALLOW

			// Handle "overwrite"
			if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "overwrite" {
				p.nextToken() // consume "overwrite"
				stmt.AllowOverwrite = true
			} else if p.peekToken.Type == lexer.PERMISSIONS {
				// Handle "permissions ["read","write"] to ["user","group","others"]"
				p.nextToken() // consume PERMISSIONS

				permSpec := ast.PermissionSpec{
					Permissions: []string{},
					Targets:     []string{},
				}

				// Parse permissions array
				if p.peekToken.Type == lexer.LBRACKET {
					p.nextToken() // consume [

					for {
						if p.peekToken.Type == lexer.STRING {
							p.nextToken()
							perm := p.curToken.Literal
							// Validate permission type
							if perm == "read" || perm == "write" || perm == "execute" {
								permSpec.Permissions = append(permSpec.Permissions, perm)
							} else {
								p.addError(fmt.Sprintf("invalid permission: %s (must be read, write, or execute)", perm))
							}
						}

						if p.peekToken.Type == lexer.COMMA {
							p.nextToken() // consume comma
							continue
						} else if p.peekToken.Type == lexer.RBRACKET {
							p.nextToken() // consume ]
							break
						} else {
							break
						}
					}
				}

				// Parse "to" keyword
				if p.peekToken.Type == lexer.TO {
					p.nextToken() // consume TO

					// Parse targets array
					if p.peekToken.Type == lexer.LBRACKET {
						p.nextToken() // consume [

						for {
							if p.peekToken.Type == lexer.STRING {
								p.nextToken()
								target := p.curToken.Literal
								// Validate target type
								if target == "user" || target == "group" || target == "others" {
									permSpec.Targets = append(permSpec.Targets, target)
								} else {
									p.addError(fmt.Sprintf("invalid permission target: %s (must be user, group, or others)", target))
								}
							}

							if p.peekToken.Type == lexer.COMMA {
								p.nextToken() // consume comma
								continue
							} else if p.peekToken.Type == lexer.RBRACKET {
								p.nextToken() // consume ]
								break
							} else {
								break
							}
						}
					}
				}

				// Add permission spec to statement
				stmt.AllowPermissions = append(stmt.AllowPermissions, permSpec)
			}

		case lexer.WITH:
			p.nextToken() // consume WITH
			switch p.peekToken.Type {
			case lexer.HEADER:
				p.nextToken() // consume HEADER
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					headerValue := p.curToken.Literal
					// Parse "key: value" format
					if colonIdx := strings.Index(headerValue, ":"); colonIdx != -1 {
						key := strings.TrimSpace(headerValue[:colonIdx])
						value := strings.TrimSpace(headerValue[colonIdx+1:])
						stmt.Headers[key] = value
					}
				}

			case lexer.AUTH:
				p.nextToken() // consume AUTH
				if p.peekToken.Type == lexer.BEARER || p.peekToken.Type == lexer.BASIC {
					p.nextToken()
					authType := p.curToken.Literal
					if p.peekToken.Type == lexer.STRING {
						p.nextToken()
						stmt.Auth[authType] = p.curToken.Literal
					}
				}
			}

		case lexer.TIMEOUT, lexer.RETRY:
			p.nextToken() // consume TIMEOUT/RETRY
			optionKey := p.curToken.Literal
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Options[optionKey] = p.curToken.Literal
			}

		case lexer.REMOVE:
			p.nextToken() // consume REMOVE
			if p.peekToken.Type == lexer.ARCHIVE {
				p.nextToken() // consume ARCHIVE
				stmt.RemoveArchive = true
			}

		default:
			// No more options
			return stmt
		}
	}
}
