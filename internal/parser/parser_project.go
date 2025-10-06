package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func (p *Parser) parseVersionStatement() *ast.VersionStatement {
	stmt := &ast.VersionStatement{Token: p.curToken}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	if !p.expectPeek(lexer.NUMBER) {
		return nil
	}

	stmt.Value = p.curToken.Literal
	p.nextToken()

	return stmt
}

// parseProjectStatement parses a project statement
func (p *Parser) parseProjectStatement() *ast.ProjectStatement {
	stmt := &ast.ProjectStatement{Token: p.curToken}

	// Expect project name (string)
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Check for optional version
	if p.peekToken.Type == lexer.VERSION {
		p.nextToken() // consume VERSION token
		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Version = p.curToken.Literal
	}

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse project settings (optional)
	if p.peekToken.Type == lexer.INDENT {
		p.nextToken() // consume INDENT
		p.nextToken() // move to first token inside the block

		for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
			switch p.curToken.Type {
			case lexer.SET:
				setting := p.parseSetStatement()
				if setting != nil {
					stmt.Settings = append(stmt.Settings, setting)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer.PARAMETER:
				setting := p.parseProjectParameterStatement()
				if setting != nil {
					stmt.Settings = append(stmt.Settings, setting)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer.SNIPPET:
				setting := p.parseSnippetStatement()
				if setting != nil {
					stmt.Settings = append(stmt.Settings, setting)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer.INCLUDE:
				setting := p.parseIncludeStatement()
				if setting != nil {
					stmt.Settings = append(stmt.Settings, setting)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer.BEFORE, lexer.AFTER, lexer.ON:
				hook := p.parseLifecycleHook()
				if hook != nil {
					stmt.Settings = append(stmt.Settings, hook)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer.SHELL:
				shellConfig := p.parseShellConfigStatement()
				if shellConfig != nil {
					stmt.Settings = append(stmt.Settings, shellConfig)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer.COMMENT, lexer.MULTILINE_COMMENT:
				p.nextToken() // Skip comments
			case lexer.NEWLINE:
				p.nextToken() // Skip newlines
			default:
				p.addError(fmt.Sprintf("unexpected token in project body: %s", p.curToken.Type))
				p.nextToken()
			}
		}

		if p.curToken.Type == lexer.DEDENT {
			p.nextToken() // consume DEDENT
		}
	} else {
		// No INDENT found, advance to next token for proper parsing flow
		p.nextToken()
	}

	return stmt
}

// parseSetStatement parses a set statement with two syntaxes:
// 1. set key to "value"
// 2. set key as list to ["value1", "value2", "value3"]
func (p *Parser) parseSetStatement() *ast.SetStatement {
	stmt := &ast.SetStatement{Token: p.curToken}

	// Expect identifier (key) - allow Git, HTTP, Docker, and File keywords as set keys
	switch p.peekToken.Type {
	case lexer.IDENT, lexer.MESSAGE, lexer.BRANCH, lexer.REMOTE, lexer.STATUS, lexer.LOG, lexer.COMMIT, lexer.ADD, lexer.PUSH, lexer.PULL,
		lexer.GET, lexer.POST, lexer.PUT, lexer.DELETE, lexer.PATCH, lexer.HEAD, lexer.OPTIONS, lexer.HTTP, lexer.HTTPS, lexer.URL, lexer.API, lexer.JSON, lexer.XML,
		lexer.TIMEOUT, lexer.RETRY, lexer.AUTH, lexer.BEARER, lexer.BASIC, lexer.TOKEN, lexer.HEADER, lexer.BODY, lexer.DATA,
		lexer.SCALE, lexer.PORT, lexer.REGISTRY, lexer.CHECKOUT, lexer.BACKUP, lexer.CHECK, lexer.SIZE, lexer.DIRECTORY:
		p.nextToken()
	default:
		p.addError(fmt.Sprintf("expected set key, got %s instead", p.peekToken.Type))
		return nil
	}
	stmt.Key = p.curToken.Literal

	// Check for optional "as list" syntax or direct "to"
	switch p.peekToken.Type {
	case lexer.AS:
		p.nextToken() // consume AS
		if !p.expectPeek(lexer.LIST) {
			return nil
		}
		// Now expect "to"
		if !p.expectPeek(lexer.TO) {
			return nil
		}
	case lexer.TO:
		p.nextToken() // consume TO
	default:
		p.addError(fmt.Sprintf("expected 'as list to' or 'to', got %s instead", p.peekToken.Type))
		return nil
	}

	// Parse expression (string literal or array literal)
	p.nextToken()
	stmt.Value = p.parseExpression()
	if stmt.Value == nil {
		return nil
	}

	p.nextToken() // advance to next token
	return stmt
}

// parseIncludeStatement parses an include statement
func (p *Parser) parseIncludeStatement() *ast.IncludeStatement {
	stmt := &ast.IncludeStatement{Token: p.curToken}

	p.nextToken() // move past 'include'

	// Check for "include from drunhub path"
	if p.curToken.Type == lexer.FROM {
		p.nextToken() // move past 'from'

		// Check for drunhub keyword
		if p.curToken.Type == lexer.DRUNHUB {
			p.nextToken() // move past 'drunhub'

			// Expect path (identifier like ops/docker or string)
			var drunhubPath string
			switch p.curToken.Type {
			case lexer.IDENT, lexer.STRING:
				drunhubPath = p.curToken.Literal
			default:
				p.addError(fmt.Sprintf("expected path after drunhub, got %s", p.curToken.Type))
				return nil
			}

			// Convert to drunhub protocol URL
			stmt.Path = "drunhub:" + drunhubPath

			p.nextToken()

			// Check for optional "as namespace"
			if p.curToken.Type == lexer.AS {
				p.nextToken() // move past 'as'

				if p.curToken.Type == lexer.IDENT {
					stmt.Namespace = p.curToken.Literal
					p.nextToken()
				} else {
					p.addError(fmt.Sprintf("expected namespace identifier after 'as', got %s", p.curToken.Type))
					return nil
				}
			}

			return stmt
		}

		// If not drunhub, it must be a regular FROM clause (for backward compatibility)
		// Back up and let the regular parsing handle it
		// This shouldn't happen in normal parsing flow, but handle it gracefully
		p.addError("expected 'drunhub' after 'from' or use 'include snippets/templates/tasks from path'")
		return nil
	}

	// Check for selective import: include snippets, templates from "path"
	// or basic import: include "path"
	if p.curToken.Type == lexer.SNIPPETS || p.curToken.Type == lexer.TEMPLATES || p.curToken.Type == lexer.TASKS {
		// Parse selectors
		for {
			switch p.curToken.Type {
			case lexer.SNIPPETS:
				stmt.Selectors = append(stmt.Selectors, "snippets")
			case lexer.TEMPLATES:
				stmt.Selectors = append(stmt.Selectors, "templates")
			case lexer.TASKS:
				stmt.Selectors = append(stmt.Selectors, "tasks")
			default:
				p.addError(fmt.Sprintf("unexpected token in include statement: %s", p.curToken.Type))
				return nil
			}

			p.nextToken()

			// Check for comma (more selectors) or FROM
			if p.curToken.Type == lexer.COMMA {
				p.nextToken() // skip comma
				continue
			}
			break
		}

		// Expect FROM keyword
		if p.curToken.Type != lexer.FROM {
			p.addError(fmt.Sprintf("expected FROM after selectors, got %s", p.curToken.Type))
			return nil
		}
		p.nextToken()
	}

	// Expect path (string)
	if p.curToken.Type != lexer.STRING {
		p.addError(fmt.Sprintf("expected string path, got %s", p.curToken.Type))
		return nil
	}
	stmt.Path = p.curToken.Literal

	p.nextToken()

	// Check for optional "as namespace"
	if p.curToken.Type == lexer.AS {
		p.nextToken() // move past 'as'

		if p.curToken.Type == lexer.IDENT {
			stmt.Namespace = p.curToken.Literal
			p.nextToken()
		} else {
			p.addError(fmt.Sprintf("expected namespace identifier after 'as', got %s", p.curToken.Type))
			return nil
		}
	}

	return stmt
}

// parseProjectParameterStatement parses a project-level parameter definition
// Syntax: parameter $name as type defaults to "value"
func (p *Parser) parseProjectParameterStatement() *ast.ProjectParameterStatement {
	stmt := &ast.ProjectParameterStatement{Token: p.curToken}

	// Expect variable (parameter name)
	if !p.expectPeek(lexer.VARIABLE) {
		return nil
	}

	// Strip the $ prefix from the variable name
	paramName := p.curToken.Literal
	if len(paramName) > 0 && paramName[0] == '$' {
		paramName = paramName[1:]
	}
	stmt.Name = paramName

	// Check for type constraint (as type)
	if p.peekToken.Type == lexer.AS {
		p.nextToken() // consume AS
		p.nextToken() // move to type

		switch p.curToken.Type {
		case lexer.STRING_TYPE:
			stmt.DataType = "string"
		case lexer.NUMBER_TYPE:
			stmt.DataType = "number"
		case lexer.BOOLEAN_TYPE:
			stmt.DataType = "boolean"
		case lexer.LIST:
			stmt.DataType = "list"
		case lexer.IDENT:
			stmt.DataType = p.curToken.Literal
		default:
			p.addError(fmt.Sprintf("expected type, got %s", p.curToken.Type))
			return nil
		}
	}

	// Check for value constraints (from [...])
	if p.peekToken.Type == lexer.FROM {
		p.nextToken() // consume FROM

		if !p.expectPeek(lexer.LBRACKET) {
			return nil
		}

		// Parse array elements
		for p.peekToken.Type != lexer.RBRACKET && p.peekToken.Type != lexer.EOF {
			p.nextToken()

			switch p.curToken.Type {
			case lexer.STRING, lexer.IDENT:
				stmt.Constraints = append(stmt.Constraints, p.curToken.Literal)
			}

			if p.peekToken.Type == lexer.COMMA {
				p.nextToken()
			}
		}

		if !p.expectPeek(lexer.RBRACKET) {
			return nil
		}
	}

	// Check for default value (defaults to "value")
	if p.peekToken.Type == lexer.DEFAULTS {
		p.nextToken() // consume DEFAULTS

		if !p.expectPeek(lexer.TO) {
			return nil
		}

		p.nextToken() // move to value

		switch p.curToken.Type {
		case lexer.STRING, lexer.BOOLEAN, lexer.NUMBER:
			stmt.DefaultValue = p.curToken.Literal
			stmt.HasDefault = true
		default:
			p.addError(fmt.Sprintf("expected default value, got %s", p.curToken.Type))
			return nil
		}
	}

	p.nextToken()
	return stmt
}

// parseSnippetStatement parses a snippet definition
// Syntax: snippet "name": <body>
func (p *Parser) parseSnippetStatement() *ast.SnippetStatement {
	stmt := &ast.SnippetStatement{Token: p.curToken}

	// Expect snippet name (string)
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse snippet body - expect INDENT and parse statements (similar to lifecycle hooks)
	if p.peekToken.Type == lexer.INDENT {
		p.nextToken() // consume INDENT

		// Parse statements until DEDENT
		for p.peekToken.Type != lexer.DEDENT && p.peekToken.Type != lexer.EOF {
			p.nextToken() // Move to the next token

			// Skip newlines and comments
			if p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT {
				continue
			}

			// Parse statement based on token type
			var bodyStmt ast.Statement

			if p.isActionToken(p.curToken.Type) {
				if p.isShellActionToken(p.curToken.Type) {
					bodyStmt = p.parseShellStatement()
				} else {
					bodyStmt = p.parseActionStatement()
				}
			} else if p.isVariableOperationToken(p.curToken.Type) {
				bodyStmt = p.parseVariableStatement()
			} else if p.isControlFlowToken(p.curToken.Type) {
				bodyStmt = p.parseControlFlowStatement()
			} else if p.curToken.Type == lexer.USE && p.peekToken.Type == lexer.SNIPPET {
				p.nextToken() // consume SNIPPET
				if p.expectPeek(lexer.STRING) {
					bodyStmt = &ast.UseSnippetStatement{
						Token:       p.curToken,
						SnippetName: p.curToken.Literal,
					}
				}
			} else if p.isCallToken(p.curToken.Type) {
				bodyStmt = p.parseTaskCallStatement()
			} else {
				p.addError(fmt.Sprintf("unexpected token in snippet body: %s", p.curToken.Type))
				break
			}

			if bodyStmt != nil {
				stmt.Body = append(stmt.Body, bodyStmt)
			}
		}

		// Consume DEDENT for snippet body and advance to next token for project parser
		if p.peekToken.Type == lexer.DEDENT {
			p.nextToken() // consume DEDENT for snippet body
			p.nextToken() // advance to next token for project parser to continue
		}
	}

	return stmt
}

// parseShellConfigStatement parses shell configuration for different platforms
func (p *Parser) parseShellConfigStatement() *ast.ShellConfigStatement {
	stmt := &ast.ShellConfigStatement{
		Token:     p.curToken,
		Platforms: make(map[string]*ast.PlatformShellConfig),
	}

	// Expect "config"
	if !p.expectPeek(lexer.CONFIG) {
		return nil
	}

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Expect indented block with platform configurations (skip any newlines first)
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}

	p.nextToken() // move to first token inside the block

	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		switch p.curToken.Type {
		case lexer.IDENT:
			platform := p.curToken.Literal

			// Expect colon after platform name
			if !p.expectPeek(lexer.COLON) {
				return nil
			}

			// Parse platform configuration
			config := p.parsePlatformShellConfig()
			if config != nil {
				stmt.Platforms[platform] = config
			}
		case lexer.COMMENT, lexer.MULTILINE_COMMENT:
			p.nextToken() // Skip comments
		default:
			p.addError(fmt.Sprintf("unexpected token in shell config: %s", p.curToken.Type))
			p.nextToken()
		}
	}

	p.nextToken() // consume DEDENT
	return stmt
}

// parsePlatformShellConfig parses configuration for a specific platform
func (p *Parser) parsePlatformShellConfig() *ast.PlatformShellConfig {
	config := &ast.PlatformShellConfig{
		Environment: make(map[string]string),
	}

	// Expect indented block (skip any newlines first)
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}

	p.nextToken() // move to first token inside the block

	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		switch p.curToken.Type {
		case lexer.IDENT, lexer.ENVIRONMENT:
			key := p.curToken.Literal

			// Expect colon
			if !p.expectPeek(lexer.COLON) {
				return nil
			}

			switch key {
			case "executable":
				// Expect string value
				if !p.expectPeek(lexer.STRING) {
					return nil
				}
				config.Executable = p.curToken.Literal
				p.nextToken()

			case "args":
				// Parse array of strings
				config.Args = p.parseStringArray()

			case "environment":
				// Parse key-value pairs
				envVars := p.parseKeyValuePairs()
				for k, v := range envVars {
					config.Environment[k] = v
				}

			default:
				p.addError(fmt.Sprintf("unknown shell config key: %s", key))
				p.nextToken()
			}
		case lexer.COMMENT, lexer.MULTILINE_COMMENT:
			p.nextToken() // Skip comments
		default:
			p.addError(fmt.Sprintf("unexpected token in platform config: %s", p.curToken.Type))
			p.nextToken()
		}
	}

	p.nextToken() // consume DEDENT
	return config
}

// parseStringArray parses an array of strings in YAML-like format
func (p *Parser) parseStringArray() []string {
	var result []string

	// Expect indented block (skip any newlines first)
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return result
	}

	p.nextToken() // move to first token inside the block

	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		if p.curToken.Type == lexer.MINUS {
			// Expect string after dash
			if !p.expectPeek(lexer.STRING) {
				p.nextToken()
				continue
			}
			result = append(result, p.curToken.Literal)
			p.nextToken()
		} else if p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT {
			p.nextToken() // Skip comments
		} else {
			p.addError(fmt.Sprintf("expected array item (- \"value\"), got %s", p.curToken.Type))
			p.nextToken()
		}
	}

	p.nextToken() // consume DEDENT
	return result
}

// parseKeyValuePairs parses key-value pairs in YAML-like format
func (p *Parser) parseKeyValuePairs() map[string]string {
	result := make(map[string]string)

	// Expect indented block (skip any newlines first)
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return result
	}

	p.nextToken() // move to first token inside the block

	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		if p.curToken.Type == lexer.IDENT {
			key := p.curToken.Literal

			// Expect colon
			if !p.expectPeek(lexer.COLON) {
				p.nextToken()
				continue
			}

			// Expect string value
			if !p.expectPeek(lexer.STRING) {
				p.nextToken()
				continue
			}

			result[key] = p.curToken.Literal
			p.nextToken()
		} else if p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT {
			p.nextToken() // Skip comments
		} else {
			p.addError(fmt.Sprintf("expected key-value pair (key: \"value\"), got %s", p.curToken.Type))
			p.nextToken()
		}
	}

	p.nextToken() // consume DEDENT
	return result
}

// parseLifecycleHook parses lifecycle hooks (both old and new syntax)
func (p *Parser) parseLifecycleHook() *ast.LifecycleHook {
	hook := &ast.LifecycleHook{Token: p.curToken}

	if p.curToken.Type == lexer.ON {
		// New syntax: "on drun setup:" or "on drun teardown:"

		// Expect "drun"
		if !p.expectPeek(lexer.DRUN) {
			return nil
		}
		hook.Scope = p.curToken.Literal

		// Expect "setup" or "teardown"
		p.nextToken()
		if p.curToken.Type != lexer.SETUP && p.curToken.Type != lexer.TEARDOWN {
			p.addError("expected 'setup' or 'teardown' after 'on drun'")
			return nil
		}
		hook.Type = p.curToken.Literal

		// Expect colon
		if !p.expectPeek(lexer.COLON) {
			return nil
		}
	} else {
		// Old syntax: "before any task:" or "after any task:"
		hook.Type = p.curToken.Literal // "before" or "after"

		// Expect "any"
		if !p.expectPeek(lexer.ANY) {
			return nil
		}
		hook.Scope = p.curToken.Literal

		// Expect "task"
		if !p.expectPeek(lexer.TASK) {
			return nil
		}

		// Expect colon
		if !p.expectPeek(lexer.COLON) {
			return nil
		}
	}

	// Parse hook body - expect INDENT and parse statements
	if p.peekToken.Type == lexer.INDENT {
		p.nextToken() // consume INDENT

		// Parse statements until DEDENT (using same pattern as parseControlFlowBody)
		for p.peekToken.Type != lexer.DEDENT && p.peekToken.Type != lexer.EOF {
			p.nextToken() // Move to the next token

			if p.isVariableOperationToken(p.curToken.Type) {
				variable := p.parseVariableStatement()
				if variable != nil {
					hook.Body = append(hook.Body, variable)
				}
			} else if p.isDetectionToken(p.curToken.Type) && p.isDetectionContext() {
				detection := p.parseDetectionStatement()
				if detection != nil {
					hook.Body = append(hook.Body, detection)
				}
			} else if p.isControlFlowToken(p.curToken.Type) {
				controlFlow := p.parseControlFlowStatement()
				if controlFlow != nil {
					hook.Body = append(hook.Body, controlFlow)
				}
			} else if p.isErrorHandlingToken(p.curToken.Type) {
				errorHandling := p.parseErrorHandlingStatement()
				if errorHandling != nil {
					hook.Body = append(hook.Body, errorHandling)
				}
			} else if p.isThrowActionToken(p.curToken.Type) {
				throw := p.parseThrowStatement()
				if throw != nil {
					hook.Body = append(hook.Body, throw)
				}
			} else if p.isDockerToken(p.curToken.Type) {
				// Special handling for RUN token - check context
				if p.curToken.Type == lexer.RUN {
					// Look ahead to determine if this is shell or docker command
					if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.COLON {
						// This is "run 'command'" or "run:" - shell command
						shell := p.parseShellStatement()
						if shell != nil {
							hook.Body = append(hook.Body, shell)
						}
					} else {
						// This is "docker run container" - docker command
						docker := p.parseDockerStatement()
						if docker != nil {
							hook.Body = append(hook.Body, docker)
						}
					}
				} else {
					docker := p.parseDockerStatement()
					if docker != nil {
						hook.Body = append(hook.Body, docker)
					}
				}
			} else if p.isGitToken(p.curToken.Type) {
				// Special handling for CREATE token - check context
				if p.curToken.Type == lexer.CREATE {
					// Look ahead to determine if this is git or file operation
					if p.peekToken.Type == lexer.BRANCH || p.peekToken.Type == lexer.TAG {
						git := p.parseGitStatement()
						if git != nil {
							hook.Body = append(hook.Body, git)
						}
					} else if p.peekToken.Type == lexer.DIRECTORY || p.peekToken.Type == lexer.DIR || (p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "file") {
						file := p.parseFileStatement()
						if file != nil {
							hook.Body = append(hook.Body, file)
						}
					} else {
						git := p.parseGitStatement()
						if git != nil {
							hook.Body = append(hook.Body, git)
						}
					}
				} else {
					git := p.parseGitStatement()
					if git != nil {
						hook.Body = append(hook.Body, git)
					}
				}
			} else if p.isHTTPToken(p.curToken.Type) {
				http := p.parseHTTPStatement()
				if http != nil {
					hook.Body = append(hook.Body, http)
				}
			} else if p.isNetworkToken(p.curToken.Type) {
				network := p.parseNetworkStatement()
				if network != nil {
					hook.Body = append(hook.Body, network)
				}
			} else if p.isFileActionToken(p.curToken.Type) {
				file := p.parseFileStatement()
				if file != nil {
					hook.Body = append(hook.Body, file)
				}
			} else if p.isActionToken(p.curToken.Type) {
				if p.isShellActionToken(p.curToken.Type) {
					shell := p.parseShellStatement()
					if shell != nil {
						hook.Body = append(hook.Body, shell)
					}
				} else {
					action := p.parseActionStatement()
					if action != nil {
						hook.Body = append(hook.Body, action)
					}
				}
			} else if p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT {
				// Skip comments
				continue
			} else if p.curToken.Type == lexer.NEWLINE {
				// Skip newlines
				continue
			} else {
				p.addError(fmt.Sprintf("unexpected token in lifecycle hook body: %s", p.curToken.Type))
				break // Stop parsing on unexpected token
			}
		}

		// Consume DEDENT for hook body and advance to next token for project parser
		if p.peekToken.Type == lexer.DEDENT {
			p.nextToken() // consume DEDENT for hook body
			p.nextToken() // advance to next token for project parser to continue
		}
	}

	return hook
}
