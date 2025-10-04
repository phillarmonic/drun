package parser

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/errors"
	lexer "github.com/phillarmonic/drun/internal/lexer"
)

// Parser parses drun v2 source code into an AST
type Parser struct {
	lexer *lexer.Lexer

	curToken  lexer.Token
	peekToken lexer.Token

	errors    []string // Legacy error list for backward compatibility
	errorList *errors.ParseErrorList
}

// New creates a new parser instance
func NewParser(l *lexer.Lexer) *Parser {
	p := &Parser{
		lexer:  l,
		errors: []string{},
	}

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// NewParserWithSource creates a new parser instance with source information for better error reporting
func NewParserWithSource(l *lexer.Lexer, filename, source string) *Parser {
	p := &Parser{
		lexer:     l,
		errors:    []string{},
		errorList: errors.NewParseErrorList(filename, source),
	}

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// nextToken advances both curToken and peekToken
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// ParseProgram parses the entire program
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}

	// Skip any leading comments
	p.skipComments()

	// Parse version statement (required)
	if p.curToken.Type == lexer.VERSION {
		program.Version = p.parseVersionStatement()
	} else {
		p.addError(fmt.Sprintf("expected version statement, got %s", p.curToken.Type))
		return nil
	}

	// Skip comments between version and project/tasks
	p.skipComments()

	// Parse optional project statement
	if p.curToken.Type == lexer.PROJECT {
		program.Project = p.parseProjectStatement()
		p.skipComments()
	}

	// Parse task and template statements
	for p.curToken.Type != lexer.EOF {
		switch p.curToken.Type {
		case lexer.TEMPLATE:
			// Check if this is "template task"
			if p.peekToken.Type == lexer.TASK {
				template := p.parseTaskTemplateStatement()
				if template != nil {
					program.Templates = append(program.Templates, template)
				}
			} else {
				p.addError(fmt.Sprintf("unexpected token after template: %s", p.peekToken.Type))
				p.nextToken()
			}
		case lexer.TASK:
			task := p.parseTaskStatement()
			if task != nil {
				program.Tasks = append(program.Tasks, task)
			}
		case lexer.COMMENT, lexer.MULTILINE_COMMENT:
			p.nextToken() // Skip comments
		case lexer.NEWLINE:
			p.nextToken() // Skip newlines
		case lexer.DEDENT:
			// Skip stray lexer.DEDENT tokens (they should be consumed by task parsing)
			p.nextToken()
		default:
			p.addError(fmt.Sprintf("unexpected token: %s", p.curToken.Type))
			p.nextToken()
		}
	}

	return program
}

// parseVersionStatement parses a version statement
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

// parseTaskStatement parses a task statement

// Docker, Git, HTTP, Download, and Network parsing methods moved to separate files

// parseDetectionStatement parses smart detection operations

// parseNetworkStatement parses network operations (health checks, port testing, ping)

// parseStringList parses a list of strings like ["dev", "staging", "production"]
func (p *Parser) parseStringList() []string {
	var items []string

	for p.peekToken.Type != lexer.RBRACKET && p.peekToken.Type != lexer.EOF {
		if !p.expectPeek(lexer.STRING) {
			break
		}
		items = append(items, p.curToken.Literal)

		// Check for comma
		if p.peekToken.Type == lexer.COMMA {
			p.nextToken() // consume comma
		}
	}

	// Consume RBRACKET
	if p.peekToken.Type == lexer.RBRACKET {
		p.nextToken()
	}

	return items
}

// isDependencyToken checks if a token type represents a dependency declaration
func (p *Parser) isDependencyToken(tokenType lexer.TokenType) bool {
	return tokenType == lexer.DEPENDS
}

// isDockerToken checks if a token type represents a Docker statement
func (p *Parser) isDockerToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.DOCKER, lexer.BUILD, lexer.TAG, lexer.PUSH, lexer.PULL, lexer.RUN, lexer.STOP, lexer.START, lexer.SCALE:
		return true
	default:
		return false
	}
}

// isGitToken checks if a token type represents a Git statement
func (p *Parser) isGitToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.GIT, lexer.CREATE, lexer.CHECKOUT, lexer.MERGE:
		return true
	default:
		return false
	}
}

// isHTTPToken checks if a token type represents an HTTP statement
func (p *Parser) isHTTPToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.HTTP, lexer.HTTPS, lexer.GET, lexer.POST, lexer.PUT, lexer.DELETE, lexer.PATCH, lexer.HEAD, lexer.OPTIONS, lexer.DOWNLOAD:
		return true
	default:
		return false
	}
}

// isNetworkToken checks if a token type represents a network statement
func (p *Parser) isNetworkToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.WAIT, lexer.PING, lexer.TEST:
		return true
	default:
		return false
	}
}

// isDetectionToken checks if a token type represents a detection statement
func (p *Parser) isDetectionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.DETECT, lexer.IF, lexer.WHEN:
		return true
	default:
		return false
	}
}

// isParameterToken checks if a token type represents a parameter declaration
func (p *Parser) isParameterToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.REQUIRES, lexer.GIVEN, lexer.ACCEPTS, lexer.PARAMETER:
		return true
	default:
		return false
	}
}

// isControlFlowToken checks if a token type represents a control flow statement
func (p *Parser) isControlFlowToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.WHEN, lexer.IF, lexer.FOR:
		return true
	default:
		return false
	}
}

// isActionToken checks if a token type represents an action
func (p *Parser) isActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.INFO, lexer.STEP, lexer.WARN, lexer.ERROR, lexer.SUCCESS, lexer.FAIL, lexer.ECHO,
		lexer.RUN, lexer.EXEC, lexer.SHELL, lexer.CAPTURE,
		lexer.CREATE, lexer.COPY, lexer.MOVE, lexer.DELETE, lexer.READ, lexer.WRITE, lexer.APPEND, lexer.BACKUP, lexer.CHECK:
		return true
	default:
		return false
	}
}

// isCallToken checks if a token type represents a task call
func (p *Parser) isCallToken(tokenType lexer.TokenType) bool {
	return tokenType == lexer.CALL
}

// isValidTaskNameToken checks if a token type can be used as a task name without quotes
func (p *Parser) isValidTaskNameToken(tokenType lexer.TokenType) bool {
	// Allow common task-related keywords to be used as task names
	switch tokenType {
	case lexer.TEST, lexer.BUILD, lexer.CI,
		lexer.START, lexer.STOP,
		lexer.BACKUP, lexer.CHECK, lexer.VERIFY:
		return true
	default:
		return false
	}
}

// isKeywordToken checks if a token type is a keyword (can be used as a parameter name)
func (p *Parser) isKeywordToken(tokenType lexer.TokenType) bool {
	// Return false for basic tokens, structural keywords, and statement-starting keywords
	switch tokenType {
	case lexer.ILLEGAL, lexer.EOF, lexer.STRING, lexer.NUMBER, lexer.BOOLEAN, lexer.VARIABLE, lexer.IDENT:
		// Basic tokens
		return false
	case lexer.VERSION, lexer.TASK, lexer.PROJECT, lexer.DRUN,
		lexer.SETUP, lexer.TEARDOWN, lexer.BEFORE, lexer.AFTER,
		lexer.IF, lexer.ELSE, lexer.WHEN, lexer.OTHERWISE,
		lexer.FOR, lexer.IN, lexer.PARALLEL,
		lexer.WITH, lexer.TRY, lexer.CATCH, lexer.FINALLY,
		lexer.THROW, lexer.IGNORE, lexer.CALL,
		lexer.COLON, lexer.EQUALS, lexer.COMMA, lexer.LPAREN, lexer.RPAREN,
		lexer.LBRACE, lexer.RBRACE, lexer.LBRACKET, lexer.RBRACKET,
		lexer.NEWLINE, lexer.INDENT, lexer.DEDENT,
		lexer.INFO, lexer.STEP, lexer.WARN, lexer.ERROR, lexer.SUCCESS, lexer.FAIL, lexer.ECHO,
		lexer.RUN, lexer.EXEC, lexer.SHELL, lexer.CAPTURE,
		lexer.CREATE, lexer.COPY, lexer.MOVE, lexer.DELETE, lexer.READ, lexer.WRITE, lexer.APPEND, lexer.BACKUP, lexer.CHECK,
		lexer.DOCKER, lexer.GIT, lexer.HTTP, lexer.HTTPS, lexer.GET, lexer.POST, lexer.PUT, lexer.PATCH, lexer.HEAD, lexer.OPTIONS,
		lexer.DETECT, lexer.GIVEN, lexer.REQUIRES, lexer.DEFAULTS, lexer.BREAK, lexer.CONTINUE,
		lexer.USE, lexer.SNIPPET, lexer.TEMPLATE, lexer.PARAMETER, lexer.MIXIN, lexer.USES, lexer.INCLUDES:
		// Structural keywords, action keywords, and statement-starting keywords
		return false
	default:
		// Everything else (like ENVIRONMENT, TARGET, etc.) can be a parameter name
		return true
	}
}

// isShellActionToken checks if a token type represents a shell action
func (p *Parser) isShellActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.RUN, lexer.EXEC, lexer.SHELL, lexer.CAPTURE:
		return true
	default:
		return false
	}
}

// isTypeToken checks if a token type represents a data type
func (p *Parser) isTypeToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN_TYPE, lexer.LIST_TYPE, lexer.IDENT:
		return true
	default:
		return false
	}
}

// isFileActionToken checks if a token type represents a file action
func (p *Parser) isFileActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.COPY, lexer.MOVE, lexer.DELETE, lexer.READ, lexer.WRITE, lexer.APPEND, lexer.BACKUP, lexer.CHECK:
		return true
	default:
		return false
	}
}

// isErrorHandlingToken checks if a token type represents error handling
func (p *Parser) isErrorHandlingToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TRY:
		return true
	default:
		return false
	}
}

// isThrowActionToken checks if a token type represents a throw action
func (p *Parser) isThrowActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.THROW, lexer.RETHROW, lexer.IGNORE:
		return true
	default:
		return false
	}
}

// expectPeek checks the peek token type and advances if it matches
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// expectPeekSkipNewlines expects a token type but skips any NEWLINE tokens first
func (p *Parser) expectPeekSkipNewlines(t lexer.TokenType) bool {
	// Skip any NEWLINE tokens
	for p.peekToken.Type == lexer.NEWLINE {
		p.nextToken() // consume the NEWLINE
	}

	// Now check for the expected token
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// peekError adds an error for unexpected peek token
func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)

	// Also add to new error system if available
	if p.errorList != nil {
		p.errorList.Add(msg, p.peekToken)
	}
}

// addError adds an error message
func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)

	// Also add to new error system if available
	if p.errorList != nil {
		p.errorList.Add(msg, p.curToken)
	}
}

// skipComments skips over comment tokens and newlines
func (p *Parser) skipComments() {
	for p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT || p.curToken.Type == lexer.NEWLINE {
		p.nextToken()
	}
}

// Errors returns any parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

// ErrorList returns the enhanced error list with position information
func (p *Parser) ErrorList() *errors.ParseErrorList {
	return p.errorList
}

// parseControlFlowStatement parses control flow statements (when, if, for)

// isVariableOperationToken checks if a token represents variable operations
func (p *Parser) isVariableOperationToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.LET, lexer.SET, lexer.TRANSFORM, lexer.CAPTURE:
		return true
	default:
		return false
	}
}

// parseVariableStatement and related methods moved to parser_variable.go

// Expression parsing methods moved to parser_expression.go

// expectPeekVariableName checks for variable names using $variable syntax
func (p *Parser) expectPeekVariableName() bool {
	if p.peekToken.Type != lexer.VARIABLE {
		p.addError(fmt.Sprintf("expected variable name ($variable), got %s instead", p.peekToken.Type))
		return false
	}

	p.nextToken()
	return true
}

// expectPeekFileKeyword checks for the "file" keyword (as IDENT)
func (p *Parser) expectPeekFileKeyword() bool {
	if p.peekToken.Type != lexer.IDENT || p.peekToken.Literal != "file" {
		p.addError(fmt.Sprintf("expected 'file', got %s instead", p.peekToken.Type))
		return false
	}

	p.nextToken()
	return true
}

// getVariableName returns the variable name without the $ prefix
func (p *Parser) getVariableName() string {
	if p.curToken.Type == lexer.VARIABLE && len(p.curToken.Literal) > 1 {
		return p.curToken.Literal[1:] // Remove the $ prefix
	}
	return p.curToken.Literal
}

// expectPeekIdentifierLike checks for identifier-like tokens (IDENT, VARIABLE, or keywords that can be used as identifiers)
func (p *Parser) expectPeekIdentifierLike() bool {
	switch p.peekToken.Type {
	case lexer.IDENT, lexer.VARIABLE, lexer.SERVICE, lexer.ENVIRONMENT, lexer.HOST, lexer.PORT, lexer.VERSION, lexer.TOOL:
		p.nextToken()
		return true
	default:
		p.addError(fmt.Sprintf("expected identifier or variable, got %s instead", p.peekToken.Type))
		return false
	}
}

// isPortCheckPattern checks if the current "check if" is a port check without consuming tokens
func (p *Parser) isPortCheckPattern() bool {
	// We're currently at CHECK token, peek is IF
	// We need to check if the pattern is "check if port"

	// Use a simple string-based approach by examining the lexer's input
	// Get the current position and look for "port" after "if"

	// This is a simplified approach - look at the raw input around current position
	if p.lexer == nil {
		return false
	}

	// Create a temporary lexer from current position to peek ahead
	// We'll use a different approach: examine the input string directly

	// Get current token position and look ahead in the input
	currentPos := p.curToken.Position
	input := p.lexer.GetInput() // We need to add this method to lexer

	// Look for "if port" pattern starting from current position
	if currentPos >= 0 && currentPos < len(input) {
		// Find "if" after current position
		remaining := input[currentPos:]
		ifIndex := strings.Index(remaining, "if")
		if ifIndex >= 0 {
			afterIf := remaining[ifIndex+2:]
			// Skip whitespace and look for "port"
			afterIf = strings.TrimLeft(afterIf, " \t")
			return strings.HasPrefix(afterIf, "port")
		}
	}

	return false
}

// expectPeekFunctionName checks for function names (can be IDENT or reserved keywords)
func (p *Parser) expectPeekFunctionName() bool {
	// Function names can be regular identifiers or reserved keywords used as function names
	validFunctionTokens := map[lexer.TokenType]bool{
		lexer.IDENT:     true,
		lexer.CONCAT:    true,
		lexer.SPLIT:     true,
		lexer.REPLACE:   true,
		lexer.TRIM:      true,
		lexer.UPPERCASE: true,
		lexer.LOWERCASE: true,
		lexer.PREPEND:   true,
		lexer.JOIN:      true,
		lexer.SLICE:     true,
		lexer.LENGTH:    true,
		lexer.KEYS:      true,
		lexer.VALUES:    true,
		lexer.SUBTRACT:  true,
		lexer.MULTIPLY:  true,
		lexer.DIVIDE:    true,
		lexer.MODULO:    true,
	}

	if !validFunctionTokens[p.peekToken.Type] {
		p.addError(fmt.Sprintf("expected function name, got %s instead", p.peekToken.Type))
		return false
	}
	p.nextToken()
	return true
}

// parseConditionExpression parses condition expressions like "environment is production"
func (p *Parser) parseConditionExpression() string {
	var parts []string

	// Read tokens until we hit a colon
	for p.peekToken.Type != lexer.COLON && p.peekToken.Type != lexer.EOF {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)
	}

	return strings.Join(parts, " ")
}

// parseSimpleCondition parses simple conditions for break/continue statements
func (p *Parser) parseSimpleCondition() string {
	var parts []string

	// Parse a simple expression: variable operator value
	// This should be something like "item == 'stop'" or "count > 10"

	// Get the variable
	if p.peekToken.Type == lexer.IDENT || p.peekToken.Type == lexer.VARIABLE {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)
	}

	// Get the operator
	if p.peekToken.Type == lexer.EQ || p.peekToken.Type == lexer.NE ||
		p.peekToken.Type == lexer.GT || p.peekToken.Type == lexer.GTE ||
		p.peekToken.Type == lexer.LT || p.peekToken.Type == lexer.LTE ||
		p.peekToken.Type == lexer.CONTAINS || p.peekToken.Type == lexer.STARTS ||
		p.peekToken.Type == lexer.ENDS || p.peekToken.Type == lexer.MATCHES {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)

		// Handle "starts with" and "ends with"
		if (p.curToken.Type == lexer.STARTS || p.curToken.Type == lexer.ENDS) && p.peekToken.Type == lexer.WITH {
			p.nextToken() // consume WITH
			parts = append(parts, p.curToken.Literal)
		}
	}

	// Get the value
	if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.NUMBER || p.peekToken.Type == lexer.IDENT {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)
	}

	return strings.Join(parts, " ")
}

// parseControlFlowBody parses the body of control flow statements
