package parser

import (
	"fmt"
	"strconv"
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

	// Parse task statements
	for p.curToken.Type != lexer.EOF {
		switch p.curToken.Type {
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

	// Expect path (string)
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Path = p.curToken.Literal

	p.nextToken()
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
func (p *Parser) parseTaskStatement() *ast.TaskStatement {
	stmt := &ast.TaskStatement{Token: p.curToken}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}

	stmt.Name = p.curToken.Literal

	// Check for optional "means" clause
	if p.peekToken.Type == lexer.MEANS {
		p.nextToken() // consume lexer.MEANS

		if !p.expectPeek(lexer.STRING) {
			return nil
		}

		stmt.Description = p.curToken.Literal
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Expect lexer.INDENT to start task body (skip any newlines first)
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}

	// Parse task body (parameters and statements)
	for p.peekToken.Type != lexer.DEDENT && p.peekToken.Type != lexer.EOF {
		p.nextToken() // Move to the next token

		// Skip any NEWLINE tokens that might appear in the task body, but stop if we hit DEDENT
		for p.curToken.Type == lexer.NEWLINE && p.peekToken.Type != lexer.DEDENT && p.peekToken.Type != lexer.EOF {
			p.nextToken()
		}

		// If we've reached DEDENT or EOF after skipping newlines, break out of the loop
		if p.curToken.Type == lexer.DEDENT || p.curToken.Type == lexer.EOF {
			break
		}

		if p.isDependencyToken(p.curToken.Type) {
			dep := p.parseDependencyStatement()
			if dep != nil {
				stmt.Dependencies = append(stmt.Dependencies, *dep)
			}
		} else if p.isParameterToken(p.curToken.Type) {
			param := p.parseParameterStatement()
			if param != nil {
				stmt.Parameters = append(stmt.Parameters, *param)
			}
		} else if p.isDetectionToken(p.curToken.Type) && p.isDetectionContext() {
			detection := p.parseDetectionStatement()
			if detection != nil {
				stmt.Body = append(stmt.Body, detection)
			}
		} else if p.isControlFlowToken(p.curToken.Type) {
			controlFlow := p.parseControlFlowStatement()
			if controlFlow != nil {
				stmt.Body = append(stmt.Body, controlFlow)
			}
		} else if p.isErrorHandlingToken(p.curToken.Type) {
			errorHandling := p.parseErrorHandlingStatement()
			if errorHandling != nil {
				stmt.Body = append(stmt.Body, errorHandling)
			}
		} else if p.isThrowActionToken(p.curToken.Type) {
			throw := p.parseThrowStatement()
			if throw != nil {
				stmt.Body = append(stmt.Body, throw)
			}
		} else if p.isDockerToken(p.curToken.Type) {
			// Special handling for RUN token - check context
			if p.curToken.Type == lexer.RUN {
				// Look ahead to determine if this is shell or docker command
				if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.COLON {
					// This is "run 'command'" or "run:" - shell command
					shell := p.parseShellStatement()
					if shell != nil {
						stmt.Body = append(stmt.Body, shell)
					}
				} else {
					// This is "docker run container" - docker command
					docker := p.parseDockerStatement()
					if docker != nil {
						stmt.Body = append(stmt.Body, docker)
					}
				}
			} else {
				docker := p.parseDockerStatement()
				if docker != nil {
					stmt.Body = append(stmt.Body, docker)
				}
			}
		} else if p.isGitToken(p.curToken.Type) {
			// Special handling for CREATE token - check context
			if p.curToken.Type == lexer.CREATE {
				// Look ahead to determine if this is git or file operation
				if p.peekToken.Type == lexer.BRANCH || p.peekToken.Type == lexer.TAG {
					git := p.parseGitStatement()
					if git != nil {
						stmt.Body = append(stmt.Body, git)
					}
				} else if p.peekToken.Type == lexer.DIRECTORY || p.peekToken.Type == lexer.DIR || (p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "file") {
					file := p.parseFileStatement()
					if file != nil {
						stmt.Body = append(stmt.Body, file)
					}
				} else {
					p.addError("ambiguous 'create' statement - specify 'branch', 'tag', 'file', 'dir', or 'directory'")
				}
			} else {
				git := p.parseGitStatement()
				if git != nil {
					stmt.Body = append(stmt.Body, git)
				}
			}
		} else if p.isHTTPToken(p.curToken.Type) {
			http := p.parseHTTPStatement()
			if http != nil {
				stmt.Body = append(stmt.Body, http)
			}
		} else if p.isNetworkToken(p.curToken.Type) {
			network := p.parseNetworkStatement()
			if network != nil {
				stmt.Body = append(stmt.Body, network)
			}
		} else if p.isBreakContinueToken(p.curToken.Type) {
			breakContinue := p.parseBreakContinueStatement()
			if breakContinue != nil {
				stmt.Body = append(stmt.Body, breakContinue)
			}
		} else if p.isVariableOperationToken(p.curToken.Type) {
			variable := p.parseVariableStatement()
			if variable != nil {
				stmt.Body = append(stmt.Body, variable)
			}
		} else if p.isActionToken(p.curToken.Type) {
			if p.isShellActionToken(p.curToken.Type) {
				shell := p.parseShellStatement()
				if shell != nil {
					stmt.Body = append(stmt.Body, shell)
				}
			} else if p.isFileActionToken(p.curToken.Type) {
				// Special handling for CHECK token
				if p.curToken.Type == lexer.CHECK {
					switch p.peekToken.Type {
					case lexer.HEALTH:
						// Definitely a network health check
						network := p.parseNetworkStatement()
						if network != nil {
							stmt.Body = append(stmt.Body, network)
						}
					case lexer.IF:
						// This is "check if X" - determine if it's a port check
						if p.isPortCheckPattern() {
							// This is "check if port" - network operation
							network := p.parseNetworkStatement()
							if network != nil {
								stmt.Body = append(stmt.Body, network)
							}
						} else {
							// This is "check if file" or other - file operation
							file := p.parseFileStatement()
							if file != nil {
								stmt.Body = append(stmt.Body, file)
							}
						}
					default:
						// Other check operations (check size, etc.) - file operations
						file := p.parseFileStatement()
						if file != nil {
							stmt.Body = append(stmt.Body, file)
						}
					}
				} else {
					// Regular file operation
					file := p.parseFileStatement()
					if file != nil {
						stmt.Body = append(stmt.Body, file)
					}
				}
			} else if p.isThrowActionToken(p.curToken.Type) {
				throw := p.parseThrowStatement()
				if throw != nil {
					stmt.Body = append(stmt.Body, throw)
				}
			} else {
				action := p.parseActionStatement()
				if action != nil {
					stmt.Body = append(stmt.Body, action)
				}
			}
		} else if p.isCallToken(p.curToken.Type) {
			call := p.parseTaskCallStatement()
			if call != nil {
				stmt.Body = append(stmt.Body, call)
			}
		} else if p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT {
			// Skip comments in task body
			continue
		} else if p.curToken.Type == lexer.NEWLINE {
			// Skip newlines in task body
			continue
		} else {
			p.addError(fmt.Sprintf("unexpected token in task body: %s (peek: %s) at line %d, column %d", p.curToken.Type, p.peekToken.Type, p.curToken.Line, p.curToken.Column))
			break // Stop parsing on unexpected token
		}
	}

	// Consume lexer.DEDENT
	if p.peekToken.Type == lexer.DEDENT {
		p.nextToken() // Move to lexer.DEDENT
		p.nextToken() // Move past lexer.DEDENT
	}

	return stmt
}

// parseActionStatement parses an action statement (info, step, success, etc.)
func (p *Parser) parseActionStatement() *ast.ActionStatement {
	stmt := &ast.ActionStatement{
		Token:  p.curToken,
		Action: p.curToken.Literal,
	}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}

	stmt.Message = p.curToken.Literal

	return stmt
}

// parseTaskCallStatement parses a task call statement (call task "name" with param="value")
func (p *Parser) parseTaskCallStatement() *ast.TaskCallStatement {
	stmt := &ast.TaskCallStatement{
		Token:      p.curToken,
		Parameters: make(map[string]string),
	}

	// Expect "task" after "call"
	if !p.expectPeek(lexer.TASK) {
		p.addError("expected 'task' after 'call'")
		return nil
	}

	// Expect task name as string, identifier, or valid keyword
	// Note: Unquoted names must be single tokens (no hyphens unless quoted)
	if p.peekToken.Type != lexer.STRING && p.peekToken.Type != lexer.IDENT && !p.isValidTaskNameToken(p.peekToken.Type) {
		p.addError("expected task name as string or identifier (use quotes for names with hyphens or spaces)")
		return nil
	}

	p.nextToken() // consume the task name token
	stmt.TaskName = p.curToken.Literal

	// Check for optional "with" parameters
	if p.peekToken.Type == lexer.WITH {
		p.nextToken() // consume "with"

		// Parse parameters - continue while we see parameter names (IDENT or keywords)
		// We allow both IDENT and keywords as parameter names
		for p.peekToken.Type == lexer.IDENT || p.isKeywordToken(p.peekToken.Type) {
			p.nextToken() // consume parameter name
			paramName := p.curToken.Literal

			// Expect "="
			if !p.expectPeek(lexer.EQUALS) {
				p.addError("expected '=' after parameter name")
				return nil
			}

			// Expect parameter value as string
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected parameter value as string")
				return nil
			}

			stmt.Parameters[paramName] = p.curToken.Literal
		}
	}

	return stmt
}

// parseShellStatement parses a shell command statement (run, exec, shell, capture)
func (p *Parser) parseShellStatement() *ast.ShellStatement {
	stmt := &ast.ShellStatement{
		Token:  p.curToken,
		Action: p.curToken.Literal,
	}

	// Check if this is multiline syntax (action followed by colon or capture with "as")
	if p.peekToken.Type == lexer.COLON {
		return p.parseMultilineShellStatement(stmt)
	}

	// Special case for "capture as $var:" syntax
	if stmt.Action == "capture" && p.peekToken.Type == lexer.AS {
		return p.parseMultilineShellStatement(stmt)
	}

	// Handle single-line syntax
	if stmt.Action == "capture" {
		return p.parseCaptureStatement(stmt)
	}

	// Regular shell command with string
	if !p.expectPeek(lexer.STRING) {
		return nil
	}

	stmt.Command = p.curToken.Literal

	// Set streaming behavior based on action type
	switch stmt.Action {
	case "run", "exec":
		stmt.StreamOutput = true
	case "shell":
		stmt.StreamOutput = true
	case "capture":
		stmt.StreamOutput = false
	}

	return stmt
}

// parseMultilineShellStatement parses multiline shell commands (run:, exec:, shell:, capture as $var:)
func (p *Parser) parseMultilineShellStatement(stmt *ast.ShellStatement) *ast.ShellStatement {
	// Handle capture with "as variable" syntax
	if stmt.Action == "capture" {
		if !p.expectPeek(lexer.AS) {
			return nil
		}
		if !p.expectPeekVariableName() {
			return nil
		}
		stmt.CaptureVar = p.getVariableName()
	}

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Expect INDENT (skip any newlines first)
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}

	// Parse command tokens until DEDENT
	p.nextToken() // Move to first token inside the block

	// Read all commands in the block
	commands := p.readCommandTokens()

	// Don't consume DEDENT here - let the task parsing loop handle it

	stmt.Commands = commands
	stmt.IsMultiline = true

	// Set streaming behavior based on action type
	switch stmt.Action {
	case "run", "exec":
		stmt.StreamOutput = true
	case "shell":
		stmt.StreamOutput = true
	case "capture":
		stmt.StreamOutput = false
	}

	return stmt
}

// parseCaptureStatement parses single-line capture statements
func (p *Parser) parseCaptureStatement(stmt *ast.ShellStatement) *ast.ShellStatement {
	if !p.expectPeek(lexer.STRING) {
		return nil
	}

	stmt.Command = p.curToken.Literal

	// Check for capture syntax: capture "command" as variable_name
	if p.peekToken.Type == lexer.AS {
		p.nextToken() // consume AS
		if !p.expectPeekVariableName() {
			return nil
		}
		stmt.CaptureVar = p.getVariableName()
	}

	stmt.StreamOutput = false
	return stmt
}

// readCommandTokens reads tokens and groups them into individual commands
func (p *Parser) readCommandTokens() []string {
	var commands []string
	var currentLine strings.Builder
	currentLineNum := p.curToken.Line

	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		if p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT {
			// Skip comments but they might indicate a line break
			p.nextToken()
			continue
		}

		// Check if we're on a new line
		if p.curToken.Line != currentLineNum && currentLine.Len() > 0 {
			// Save the current line and start a new one
			commands = append(commands, strings.TrimSpace(currentLine.String()))
			currentLine.Reset()
		}
		currentLineNum = p.curToken.Line

		// Add the current token to the line
		if p.curToken.Type == lexer.STRING {
			currentLine.WriteString(fmt.Sprintf("\"%s\"", p.curToken.Literal))
		} else {
			currentLine.WriteString(p.curToken.Literal)
		}

		// Check if we need to add a space before the next token
		nextToken := p.peekToken
		if nextToken.Type != lexer.DEDENT && nextToken.Type != lexer.EOF && nextToken.Type != lexer.COMMENT && nextToken.Type != lexer.MULTILINE_COMMENT {
			// Add space if the next token is on the same line
			if nextToken.Line == p.curToken.Line {
				currentLine.WriteString(" ")
			}
		}

		p.nextToken()
	}

	// Add the last line if there is one
	if currentLine.Len() > 0 {
		commands = append(commands, strings.TrimSpace(currentLine.String()))
	}

	// Filter out empty commands
	var filteredCommands []string
	for _, cmd := range commands {
		if strings.TrimSpace(cmd) != "" {
			filteredCommands = append(filteredCommands, strings.TrimSpace(cmd))
		}
	}

	return filteredCommands
}

// parseFileStatement parses file operation statements (create, copy, move, delete, read, write, append)
func (p *Parser) parseFileStatement() *ast.FileStatement {
	stmt := &ast.FileStatement{
		Token:  p.curToken,
		Action: p.curToken.Literal,
	}

	switch stmt.Action {
	case "create":
		return p.parseCreateStatement(stmt)
	case "copy":
		return p.parseCopyStatement(stmt)
	case "move":
		return p.parseMoveStatement(stmt)
	case "delete":
		return p.parseDeleteStatement(stmt)
	case "read":
		return p.parseReadStatement(stmt)
	case "write":
		return p.parseWriteStatement(stmt)
	case "append":
		return p.parseAppendStatement(stmt)
	case "backup":
		return p.parseBackupStatement(stmt)
	case "check":
		return p.parseCheckStatement(stmt)
	default:
		p.addError(fmt.Sprintf("unknown file operation: %s", stmt.Action))
		return nil
	}
}

// parseCreateStatement parses "create file/dir" statements
func (p *Parser) parseCreateStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: create file "path" or create dir "path" or create directory "path"
	switch p.peekToken.Type {
	case lexer.IDENT:
		p.nextToken() // consume IDENT
		if p.curToken.Literal == "file" {
			stmt.IsDir = false
		} else {
			p.addError("expected 'file', 'dir', or 'directory' after 'create'")
			return nil
		}
	case lexer.DIR:
		p.nextToken() // consume DIR
		stmt.IsDir = true
	case lexer.DIRECTORY:
		p.nextToken() // consume DIRECTORY
		stmt.IsDir = true
	default:
		p.addError("expected 'file', 'dir', or 'directory' after 'create'")
		return nil
	}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	return stmt
}

// parseCopyStatement parses "copy" statements
func (p *Parser) parseCopyStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: copy "source" to "target" or copy {variable} to "target"
	source := p.parseFilePathOrVariable()
	if source == "" {
		return nil
	}
	stmt.Source = source

	if !p.expectPeek(lexer.TO) {
		return nil
	}

	target := p.parseFilePathOrVariable()
	if target == "" {
		return nil
	}
	stmt.Target = target

	return stmt
}

// parseFilePathOrVariable parses either a string literal or a variable interpolation {var}
func (p *Parser) parseFilePathOrVariable() string {
	p.nextToken()

	switch p.curToken.Type {
	case lexer.STRING:
		return p.curToken.Literal
	case lexer.LBRACE:
		// Parse {$variable} syntax
		if !p.expectPeek(lexer.VARIABLE) {
			return ""
		}
		variable := p.curToken.Literal
		if !p.expectPeek(lexer.RBRACE) {
			return ""
		}
		return "{" + variable + "}"
	default:
		p.addError(fmt.Sprintf("expected file path (string or {$variable}), got %s", p.curToken.Type))
		return ""
	}
}

// parseMoveStatement parses "move" statements
func (p *Parser) parseMoveStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: move "source" to "target"
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Source = p.curToken.Literal

	if !p.expectPeek(lexer.TO) {
		return nil
	}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	return stmt
}

// parseDeleteStatement parses "delete" statements
func (p *Parser) parseDeleteStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: delete file "path" or delete dir "path"
	switch p.peekToken.Type {
	case lexer.IDENT:
		p.nextToken() // consume IDENT
		if p.curToken.Literal == "file" {
			stmt.IsDir = false
		} else {
			p.addError("expected 'file' or 'dir' after 'delete'")
			return nil
		}
	case lexer.DIR:
		p.nextToken() // consume DIR
		stmt.IsDir = true
	default:
		p.addError("expected 'file' or 'dir' after 'delete'")
		return nil
	}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	return stmt
}

// parseReadStatement parses "read" statements
func (p *Parser) parseReadStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: read file "path" [as variable]
	if !p.expectPeekFileKeyword() {
		return nil
	}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	// Check for capture syntax: read file "path" as variable
	if p.peekToken.Type == lexer.AS {
		p.nextToken() // consume AS
		if !p.expectPeekVariableName() {
			return nil
		}
		stmt.CaptureVar = p.getVariableName()
	}

	return stmt
}

// parseWriteStatement parses "write" statements
func (p *Parser) parseWriteStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: write "content" to file "path"
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Content = p.curToken.Literal

	if !p.expectPeek(lexer.TO) {
		return nil
	}

	if !p.expectPeekFileKeyword() {
		return nil
	}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	return stmt
}

// parseAppendStatement parses "append" statements
func (p *Parser) parseAppendStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: append "content" to file "path"
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Content = p.curToken.Literal

	if !p.expectPeek(lexer.TO) {
		return nil
	}

	if !p.expectPeekFileKeyword() {
		return nil
	}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	return stmt
}

// parseBackupStatement parses "backup" statements
func (p *Parser) parseBackupStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: backup "source" as "backup-name"
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Source = p.curToken.Literal

	if p.peekToken.Type == lexer.AS {
		p.nextToken() // consume AS
		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Target = p.curToken.Literal
	} else {
		// Generate default backup name with timestamp
		stmt.Target = "" // Will be generated in execution
	}

	return stmt
}

// parseCheckStatement parses "check" statements
func (p *Parser) parseCheckStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: check if file "path" exists
	// Expect: check size of file "path"
	switch p.peekToken.Type {
	case lexer.IF:
		p.nextToken() // consume IF
		if !p.expectPeekFileKeyword() {
			return nil
		}
		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Target = p.curToken.Literal

		if p.peekToken.Type == lexer.EXISTS {
			p.nextToken() // consume EXISTS
			stmt.Action = "check_exists"
		}
	case lexer.SIZE:
		p.nextToken() // consume SIZE
		if p.peekToken.Type == lexer.OF {
			p.nextToken() // consume OF
			if !p.expectPeekFileKeyword() {
				return nil
			}
			if !p.expectPeek(lexer.STRING) {
				return nil
			}
			stmt.Target = p.curToken.Literal
			stmt.Action = "get_size"
		}
	}

	return stmt
}

// parseErrorHandlingStatement parses try/catch/finally statements
func (p *Parser) parseErrorHandlingStatement() *ast.TryStatement {
	if p.curToken.Type != lexer.TRY {
		p.addError("expected 'try' keyword")
		return nil
	}

	stmt := &ast.TryStatement{
		Token: p.curToken,
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse try body (parseControlFlowBody handles INDENT internally)
	stmt.TryBody = p.parseControlFlowBody()

	// Parse catch clauses
	for p.peekToken.Type == lexer.CATCH {
		p.nextToken() // consume CATCH
		catchClause := p.parseCatchClause()
		if catchClause != nil {
			stmt.CatchClauses = append(stmt.CatchClauses, *catchClause)
		}
	}

	// Parse optional finally clause
	if p.peekToken.Type == lexer.FINALLY {
		p.nextToken() // consume FINALLY
		if !p.expectPeek(lexer.COLON) {
			return nil
		}
		stmt.FinallyBody = p.parseControlFlowBody()
	}

	return stmt
}

// parseCatchClause parses a catch clause
func (p *Parser) parseCatchClause() *ast.CatchClause {
	clause := &ast.CatchClause{
		Token: p.curToken,
	}

	// Check for specific error type or "as" clause
	switch p.peekToken.Type {
	case lexer.IDENT:
		p.nextToken() // consume error type
		clause.ErrorType = p.curToken.Literal

		// Check for "as variable" clause
		if p.peekToken.Type == lexer.AS {
			p.nextToken() // consume AS
			if !p.expectPeekVariableName() {
				return nil
			}
			clause.ErrorVar = p.curToken.Literal
		}
	case lexer.AS:
		p.nextToken() // consume AS
		if !p.expectPeekVariableName() {
			return nil
		}
		clause.ErrorVar = p.curToken.Literal
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse catch body (parseControlFlowBody handles INDENT internally)
	clause.Body = p.parseControlFlowBody()

	return clause
}

// parseThrowStatement parses throw, rethrow, and ignore statements
func (p *Parser) parseThrowStatement() *ast.ThrowStatement {
	stmt := &ast.ThrowStatement{
		Token:  p.curToken,
		Action: p.curToken.Literal,
	}

	switch stmt.Action {
	case "throw":
		// Expect: throw "message"
		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Message = p.curToken.Literal
	case "rethrow":
		// No additional parameters needed
	case "ignore":
		// No additional parameters needed
	default:
		p.addError(fmt.Sprintf("unknown throw action: %s", stmt.Action))
		return nil
	}

	return stmt
}

// parseParameterStatement parses parameter declarations (requires, given, accepts)
func (p *Parser) parseParameterStatement() *ast.ParameterStatement {
	stmt := &ast.ParameterStatement{
		Token:    p.curToken,
		Type:     p.curToken.Literal,
		DataType: "string", // default type
	}

	// Parse parameter name (expect $variable syntax)
	if p.peekToken.Type != lexer.VARIABLE {
		p.addError(fmt.Sprintf("expected parameter name ($variable), got %s instead", p.peekToken.Type))
		return nil
	}
	p.nextToken()
	// Store parameter name without the $ prefix
	if strings.HasPrefix(p.curToken.Literal, "$") {
		stmt.Name = p.curToken.Literal[1:] // Remove the $ prefix
	} else {
		stmt.Name = p.curToken.Literal
	}

	// Check for type declaration: "as type"
	if p.peekToken.Type == lexer.AS {
		p.nextToken() // consume AS
		if p.isTypeToken(p.peekToken.Type) {
			p.nextToken() // consume type token
			stmt.DataType = p.curToken.Literal

			// Check for advanced constraints after type
			p.parseAdvancedConstraints(stmt)
		} else if p.peekToken.Type == lexer.LIST {
			p.nextToken() // consume LIST
			stmt.DataType = "list"
			stmt.Variadic = true // list parameters are variadic by default
			if p.peekToken.Type == lexer.OF {
				p.nextToken() // consume OF
				if p.isTypeToken(p.peekToken.Type) {
					p.nextToken() // consume element type
					stmt.DataType = "list of " + p.curToken.Literal
				}
			}
		} else {
			p.addError("expected type after 'as'")
			return nil
		}
	}

	// Handle different parameter types
	switch stmt.Type {
	case "requires":
		stmt.Required = true
		// Check for constraints: requires env from ["dev", "staging"]
		if p.peekToken.Type == lexer.FROM {
			p.nextToken() // consume FROM
			if p.peekToken.Type == lexer.LBRACKET {
				p.nextToken() // consume LBRACKET
				stmt.Constraints = p.parseStringList()
			}
		}

		// Check for optional default value: requires env from ["dev", "staging"] defaults to "dev"
		if p.peekToken.Type == lexer.DEFAULTS {
			p.nextToken() // consume DEFAULTS
			if !p.expectPeek(lexer.TO) {
				return nil
			}

			// Parse default value - can be string, number, boolean, empty, or built-in function
			switch p.peekToken.Type {
			case lexer.STRING, lexer.NUMBER, lexer.BOOLEAN:
				p.nextToken()
				stmt.DefaultValue = p.curToken.Literal
				stmt.HasDefault = true
			case lexer.EMPTY:
				// Handle "empty" keyword - treat as empty string
				p.nextToken()
				stmt.DefaultValue = ""
				stmt.HasDefault = true
			case lexer.LBRACE:
				// Handle "{builtin function}" syntax
				p.nextToken() // consume LBRACE
				var funcParts []string

				// Read tokens until RBRACE
				for p.peekToken.Type != lexer.RBRACE && p.peekToken.Type != lexer.EOF {
					p.nextToken()
					funcParts = append(funcParts, p.curToken.Literal)
				}

				if p.peekToken.Type != lexer.RBRACE {
					p.addError("expected '}' to close builtin function call")
					return nil
				}
				p.nextToken() // consume RBRACE

				// Join the function parts and store as the default value
				stmt.DefaultValue = "{" + strings.Join(funcParts, " ") + "}"
				stmt.HasDefault = true
			default:
				p.addError(fmt.Sprintf("expected default value (string, number, boolean, empty, or built-in function), got %s", p.peekToken.Type))
				return nil
			}

			// Validate that the default value is in the constraints list (if constraints exist)
			if len(stmt.Constraints) > 0 {
				// Remove quotes from default value for comparison (if it's a string literal)
				defaultVal := stmt.DefaultValue
				if len(defaultVal) >= 2 && defaultVal[0] == '"' && defaultVal[len(defaultVal)-1] == '"' {
					defaultVal = defaultVal[1 : len(defaultVal)-1]
				}

				found := false
				for _, constraint := range stmt.Constraints {
					if constraint == defaultVal {
						found = true
						break
					}
				}

				if !found {
					p.addError(fmt.Sprintf("default value '%s' must be one of the allowed values: [%s]",
						defaultVal, strings.Join(stmt.Constraints, ", ")))
					return nil
				}
			}
		}

	case "given":
		stmt.Required = false
		// Expect: given name defaults to "value"
		if !p.expectPeek(lexer.DEFAULTS) {
			return nil
		}
		if !p.expectPeek(lexer.TO) {
			return nil
		}

		// Parse default value - can be string, number, boolean, empty, or built-in function
		switch p.peekToken.Type {
		case lexer.STRING, lexer.NUMBER, lexer.BOOLEAN:
			p.nextToken()
			stmt.DefaultValue = p.curToken.Literal
			stmt.HasDefault = true
		case lexer.EMPTY:
			// Handle "empty" keyword - treat as empty string
			p.nextToken()
			stmt.DefaultValue = ""
			stmt.HasDefault = true
		case lexer.LBRACE:
			// Handle "{builtin function}" syntax
			p.nextToken() // consume LBRACE
			var funcParts []string

			// Read tokens until RBRACE
			for p.peekToken.Type != lexer.RBRACE && p.peekToken.Type != lexer.EOF {
				p.nextToken()
				funcParts = append(funcParts, p.curToken.Literal)
			}

			if p.peekToken.Type != lexer.RBRACE {
				p.addError("expected '}' to close builtin function call")
				return nil
			}
			p.nextToken() // consume RBRACE

			// Join the function parts and store as the default value
			stmt.DefaultValue = "{" + strings.Join(funcParts, " ") + "}"
			stmt.HasDefault = true
		case lexer.CURRENT:
			// Handle legacy "current git commit" built-in function (for backward compatibility)
			p.nextToken() // consume CURRENT
			if p.peekToken.Type == lexer.GIT {
				p.nextToken() // consume GIT
				if p.peekToken.Type == lexer.COMMIT {
					p.nextToken() // consume COMMIT
					stmt.DefaultValue = "current git commit"
					stmt.HasDefault = true
				}
			}
		default:
			p.addError(fmt.Sprintf("expected default value (string, number, boolean, empty, or built-in function), got %s", p.peekToken.Type))
			return nil
		}

		// Check for constraints after default value: given name defaults to "value" from ["list"]
		if p.peekToken.Type == lexer.FROM {
			p.nextToken() // consume FROM
			if p.peekToken.Type == lexer.LBRACKET {
				p.nextToken() // consume LBRACKET
				stmt.Constraints = p.parseStringList()
			}
		}

	case "accepts":
		stmt.Required = false
		// accepts can have constraints too
		if p.peekToken.Type == lexer.FROM {
			p.nextToken() // consume FROM
			if p.peekToken.Type == lexer.LBRACKET {
				p.nextToken() // consume LBRACKET
				stmt.Constraints = p.parseStringList()
			}
		}
	}

	return stmt
}

// parseAdvancedConstraints parses advanced parameter constraints
func (p *Parser) parseAdvancedConstraints(stmt *ast.ParameterStatement) {
	for {
		switch p.peekToken.Type {
		case lexer.BETWEEN:
			p.parseRangeConstraint(stmt)
		case lexer.MATCHING:
			p.parsePatternConstraint(stmt)
		default:
			return // No more constraints
		}
	}
}

// parseRangeConstraint parses "between min and max" constraints
func (p *Parser) parseRangeConstraint(stmt *ast.ParameterStatement) {
	p.nextToken() // consume BETWEEN

	// Expect a number for minimum value
	if !p.expectPeek(lexer.NUMBER) {
		return
	}

	minVal, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.addError(fmt.Sprintf("invalid minimum value: %s", p.curToken.Literal))
		return
	}
	stmt.MinValue = &minVal

	// Expect AND
	if !p.expectPeek(lexer.AND) {
		return
	}

	// Expect a number for maximum value
	if !p.expectPeek(lexer.NUMBER) {
		return
	}

	maxVal, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.addError(fmt.Sprintf("invalid maximum value: %s", p.curToken.Literal))
		return
	}
	stmt.MaxValue = &maxVal
}

// parsePatternConstraint parses "matching pattern", "matching email format", or "matching macro" constraints
func (p *Parser) parsePatternConstraint(stmt *ast.ParameterStatement) {
	p.nextToken() // consume MATCHING

	switch p.peekToken.Type {
	case lexer.PATTERN:
		p.nextToken() // consume PATTERN
		if !p.expectPeek(lexer.STRING) {
			return
		}
		stmt.Pattern = p.curToken.Literal

	case lexer.EMAIL:
		p.nextToken() // consume EMAIL
		if p.peekToken.Type == lexer.FORMAT {
			p.nextToken() // consume FORMAT
		}
		stmt.EmailFormat = true

	case lexer.IDENT:
		// Check if it's a pattern macro (e.g., "matching semver")
		p.nextToken() // consume IDENT
		stmt.PatternMacro = p.curToken.Literal

	default:
		// Check if it's a keyword token that can be used as a pattern macro
		if macroName := p.getPatternMacroName(p.peekToken.Type); macroName != "" {
			p.nextToken() // consume the token
			stmt.PatternMacro = macroName
		} else {
			p.addError("expected 'pattern', 'email', or pattern macro name after 'matching'")
		}
	}
}

// getPatternMacroName returns the pattern macro name for keyword tokens that can be used as macros
func (p *Parser) getPatternMacroName(tokenType lexer.TokenType) string {
	switch tokenType {
	case lexer.URL:
		return "url"
	default:
		return ""
	}
}

// parseDependencyStatement parses a dependency declaration
func (p *Parser) parseDependencyStatement() *ast.DependencyGroup {
	group := &ast.DependencyGroup{
		Token:        p.curToken,
		Dependencies: []ast.DependencyItem{},
		Sequential:   false, // default to parallel
	}

	// Expect "on"
	if !p.expectPeek(lexer.ON) {
		return nil
	}

	// Parse dependency list
	for {
		// Expect task name (identifier or Docker keyword)
		switch p.peekToken.Type {
		case lexer.IDENT:
			p.nextToken()
		case lexer.BUILD, lexer.PUSH, lexer.PULL, lexer.TAG, lexer.REMOVE, lexer.START, lexer.STOP, lexer.RUN,
			lexer.CLONE, lexer.INIT, lexer.BRANCH, lexer.SWITCH, lexer.MERGE, lexer.ADD, lexer.COMMIT, lexer.FETCH, lexer.STATUS, lexer.LOG, lexer.SHOW,
			lexer.GET, lexer.POST, lexer.PUT, lexer.DELETE, lexer.PATCH, lexer.HEAD, lexer.OPTIONS, lexer.HTTP, lexer.HTTPS, lexer.TEST:
			p.nextToken()
		default:
			p.addError(fmt.Sprintf("expected task name, got %s instead", p.peekToken.Type))
			return nil
		}

		dep := ast.DependencyItem{
			Name:     p.curToken.Literal,
			Parallel: false, // default
		}

		// Check for "in parallel" modifier
		if p.peekToken.Type == lexer.IN {
			p.nextToken() // consume IN
			if p.peekToken.Type == lexer.PARALLEL {
				p.nextToken() // consume PARALLEL
				dep.Parallel = true
			} else {
				// Put back the IN token by not advancing
				p.addError("expected 'parallel' after 'in'")
				return nil
			}
		}

		group.Dependencies = append(group.Dependencies, dep)

		// Check what comes next
		switch p.peekToken.Type {
		case lexer.AND:
			p.nextToken() // consume AND
			group.Sequential = true
		case lexer.COMMA:
			p.nextToken() // consume COMMA
			// Keep Sequential as false (parallel)
		case lexer.THEN:
			// Handle "then" - this creates a new dependency group
			// For now, we'll treat it as sequential
			p.nextToken() // consume THEN
			group.Sequential = true
		default:
			// End of dependency list
			return group
		}
	}
}

// parseDockerStatement parses Docker operations
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

// parseGitStatement parses Git operations
func (p *Parser) parseGitStatement() *ast.GitStatement {
	stmt := &ast.GitStatement{
		Token:   p.curToken,
		Options: make(map[string]string),
	}

	// Parse Git operation
	switch p.peekToken.Type {
	case lexer.CREATE:
		// git create branch "name"
		// git create tag "v1.0.0"
		p.nextToken() // consume CREATE
		stmt.Operation = p.curToken.Literal

		switch p.peekToken.Type {
		case lexer.BRANCH:
			p.nextToken() // consume BRANCH
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		case lexer.TAG:
			p.nextToken() // consume TAG
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer.CHECKOUT:
		// git checkout branch "name"
		p.nextToken() // consume CHECKOUT
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.BRANCH {
			p.nextToken() // consume BRANCH
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer.MERGE:
		// git merge branch "name"
		p.nextToken() // consume MERGE
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.BRANCH {
			p.nextToken() // consume BRANCH
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer.CLONE:
		// git clone repository "url" to "dir"
		p.nextToken() // consume CLONE
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.REPOSITORY {
			p.nextToken() // consume REPOSITORY
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer.INIT:
		// git init repository in "dir"
		p.nextToken() // consume INIT
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.REPOSITORY {
			p.nextToken() // consume REPOSITORY
			stmt.Resource = p.curToken.Literal
		}

	case lexer.ADD:
		// git add files "pattern"
		p.nextToken() // consume ADD
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.FILES {
			p.nextToken() // consume FILES
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer.COMMIT:
		// git commit changes with message "msg"
		// git commit all changes with message "msg"
		p.nextToken() // consume COMMIT
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.ALL {
			p.nextToken() // consume ALL
			stmt.Options["all"] = "true"
		}

		if p.peekToken.Type == lexer.CHANGES {
			p.nextToken() // consume CHANGES
			stmt.Resource = p.curToken.Literal
		}

		// Parse "with message 'text'"
		if p.peekToken.Type == lexer.WITH {
			p.nextToken() // consume WITH
			if p.peekToken.Type == lexer.MESSAGE {
				p.nextToken() // consume MESSAGE
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Options["message"] = p.curToken.Literal
				}
			}
		}

	case lexer.PUSH:
		// git push to remote "origin" branch "main"
		// git push tag "v1.0.0" to remote "origin"
		p.nextToken() // consume PUSH
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.TAG {
			p.nextToken() // consume TAG
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

		// Handle "to remote 'origin' branch 'main'" - this will be handled in options parsing

	case lexer.PULL:
		// git pull from remote "origin" branch "main"
		p.nextToken() // consume PULL
		stmt.Operation = p.curToken.Literal

	case lexer.FETCH:
		// git fetch from remote "origin"
		p.nextToken() // consume FETCH
		stmt.Operation = p.curToken.Literal

	case lexer.BRANCH:
		// git create branch "name"
		// git switch to branch "name"
		// git delete branch "name"
		// git merge branch "name" into "target"
		p.nextToken() // consume BRANCH
		stmt.Resource = p.curToken.Literal

		// Look for operation before branch
		if stmt.Token.Literal == "git" {
			// This should be handled by looking at previous tokens
			// For now, assume it's a create operation
			stmt.Operation = "create"
		}

	case lexer.STATUS:
		// git status
		p.nextToken() // consume STATUS
		stmt.Operation = p.curToken.Literal

	case lexer.LOG:
		// git log --oneline
		p.nextToken() // consume LOG
		stmt.Operation = p.curToken.Literal

	case lexer.SHOW:
		// git show current branch
		// git show current commit
		p.nextToken() // consume SHOW
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.CURRENT {
			p.nextToken() // consume CURRENT
			stmt.Options["current"] = "true"

			if p.peekToken.Type == lexer.BRANCH || p.peekToken.Type == lexer.COMMIT {
				p.nextToken()
				stmt.Resource = p.curToken.Literal
			}
		}

	default:
		// Handle operations that come before git (create, switch, delete, merge)
		if p.peekToken.Type == lexer.IDENT {
			p.nextToken()
			stmt.Operation = p.curToken.Literal
		} else {
			return nil
		}
	}

	// Parse additional options (to, from, with, into, in, etc.)
	for p.peekToken.Type == lexer.TO || p.peekToken.Type == lexer.FROM || p.peekToken.Type == lexer.WITH ||
		p.peekToken.Type == lexer.INTO || p.peekToken.Type == lexer.IN || p.peekToken.Type == lexer.REMOTE ||
		p.peekToken.Type == lexer.BRANCH || p.peekToken.Type == lexer.MESSAGE || p.peekToken.Type == lexer.IDENT {
		p.nextToken()

		switch p.curToken.Type {
		case lexer.TO, lexer.FROM, lexer.WITH, lexer.INTO, lexer.IN:
			optionKey := p.curToken.Literal
			switch p.peekToken.Type {
			case lexer.STRING:
				p.nextToken()
				stmt.Options[optionKey] = p.curToken.Literal
			case lexer.REMOTE, lexer.BRANCH, lexer.MESSAGE:
				p.nextToken()
				keywordType := p.curToken.Literal
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Options[keywordType] = p.curToken.Literal
				}
			}
		case lexer.REMOTE, lexer.BRANCH, lexer.MESSAGE:
			keywordType := p.curToken.Literal
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Options[keywordType] = p.curToken.Literal
			}
		case lexer.IDENT:
			optionKey := p.curToken.Literal
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Options[optionKey] = p.curToken.Literal
			}
		}
	}

	return stmt
}

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

// parseDetectionStatement parses smart detection operations
func (p *Parser) parseDetectionStatement() *ast.DetectionStatement {
	stmt := &ast.DetectionStatement{
		Token: p.curToken,
	}

	switch p.curToken.Type {
	case lexer.DETECT:
		// detect project type
		// detect docker
		// detect node version
		// detect available "docker compose" or "docker-compose" as $compose_cmd
		stmt.Type = "detect"

		if p.peekToken.Type == lexer.AVAILABLE {
			// detect available "tool1" or "tool2" as $var
			p.nextToken() // consume AVAILABLE
			stmt.Type = "detect_available"
			stmt.Condition = "available"

			// Parse first tool (required)
			if p.peekToken.Type == lexer.STRING || p.isToolToken(p.peekToken.Type) {
				p.nextToken()
				stmt.Target = p.curToken.Literal
			} else {
				p.errors = append(p.errors, fmt.Sprintf("expected tool name after 'detect available', got %s", p.peekToken.Type))
				return stmt
			}

			// Parse alternatives (optional "or" clauses)
			for p.peekToken.Type == lexer.OR {
				p.nextToken() // consume OR
				if p.peekToken.Type == lexer.STRING || p.isToolToken(p.peekToken.Type) {
					p.nextToken()
					stmt.Alternatives = append(stmt.Alternatives, p.curToken.Literal)
				} else {
					p.errors = append(p.errors, fmt.Sprintf("expected tool name after 'or', got %s", p.peekToken.Type))
					return stmt
				}
			}

			// Parse capture variable (optional "as $var")
			if p.peekToken.Type == lexer.AS {
				p.nextToken() // consume AS
				if p.peekToken.Type == lexer.VARIABLE {
					p.nextToken()
					stmt.CaptureVar = p.getVariableName()
				} else {
					p.errors = append(p.errors, fmt.Sprintf("expected variable name after 'as', got %s", p.peekToken.Type))
					return stmt
				}
			}

		} else if p.peekToken.Type == lexer.PROJECT {
			p.nextToken() // consume PROJECT
			stmt.Target = "project"
			if p.peekToken.Type == lexer.TYPE {
				p.nextToken() // consume TYPE
				stmt.Condition = "type"
			}
		} else if p.isToolToken(p.peekToken.Type) {
			p.nextToken()
			stmt.Target = p.curToken.Literal

			if p.peekToken.Type == lexer.VERSION {
				p.nextToken() // consume VERSION
				stmt.Condition = "version"
			}
		}

	case lexer.IF:
		// if docker is available:
		// if "docker buildx" is available:
		// if docker,"docker-compose" is not available:
		// if node version >= "16":
		stmt.Type = "if_available"

		if p.isToolToken(p.peekToken.Type) || p.peekToken.Type == lexer.STRING {
			p.nextToken()
			stmt.Target = p.curToken.Literal

			// Parse additional tools separated by commas
			for p.peekToken.Type == lexer.COMMA {
				p.nextToken() // consume COMMA
				if p.peekToken.Type == lexer.STRING || p.isToolToken(p.peekToken.Type) {
					p.nextToken()
					stmt.Alternatives = append(stmt.Alternatives, p.curToken.Literal)
				} else {
					p.errors = append(p.errors, fmt.Sprintf("expected tool name after comma, got %s", p.peekToken.Type))
					return stmt
				}
			}

			switch p.peekToken.Type {
			case lexer.IS:
				p.nextToken() // consume IS
				switch p.peekToken.Type {
				case lexer.AVAILABLE:
					p.nextToken() // consume AVAILABLE
					stmt.Condition = "available"
				case lexer.NOT:
					p.nextToken() // consume NOT
					if p.peekToken.Type == lexer.AVAILABLE {
						p.nextToken() // consume AVAILABLE
						stmt.Condition = "not_available"
					}
				}
			case lexer.VERSION:
				p.nextToken() // consume VERSION
				stmt.Type = "if_version"

				// Parse comparison operator
				if p.peekToken.Type == lexer.GTE || p.peekToken.Type == lexer.GT ||
					p.peekToken.Type == lexer.LTE || p.peekToken.Type == lexer.LT ||
					p.peekToken.Type == lexer.EQ || p.peekToken.Type == lexer.NE {
					p.nextToken()
					stmt.Condition = p.curToken.Literal

					if p.peekToken.Type == lexer.STRING {
						p.nextToken()
						stmt.Value = p.curToken.Literal
					}
				}
			}
		}

	case lexer.WHEN:
		// when in ci environment:
		// when in production environment:
		stmt.Type = "when_environment"

		if p.peekToken.Type == lexer.IN {
			p.nextToken() // consume IN

			if p.isEnvironmentToken(p.peekToken.Type) {
				p.nextToken()
				stmt.Target = p.curToken.Literal

				if p.peekToken.Type == lexer.ENVIRONMENT {
					p.nextToken() // consume ENVIRONMENT
					stmt.Condition = "environment"
				}
			}
		}
	}

	// Parse body if there's a colon
	if p.peekToken.Type == lexer.COLON {
		p.nextToken() // consume COLON
		stmt.Body = p.parseControlFlowBody()

		// Check for else clause (similar to parseIfStatement)
		if p.peekToken.Type == lexer.ELSE {
			p.nextToken() // consume ELSE
			if !p.expectPeek(lexer.COLON) {
				return stmt
			}
			stmt.ElseBody = p.parseControlFlowBody()
		}
	}

	return stmt
}

// isToolToken checks if a token represents a tool name
func (p *Parser) isToolToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.DOCKER, lexer.GIT, lexer.NODE, lexer.NPM, lexer.YARN, lexer.PNPM, lexer.BUN,
		lexer.PYTHON, lexer.PIP, lexer.GO, lexer.GOLANG, lexer.CARGO,
		lexer.JAVA, lexer.MAVEN, lexer.GRADLE, lexer.RUBY, lexer.GEM,
		lexer.PHP, lexer.COMPOSER, lexer.RUST, lexer.MAKE,
		lexer.KUBECTL, lexer.HELM, lexer.TERRAFORM, lexer.AWS, lexer.GCP, lexer.AZURE:
		return true
	default:
		return false
	}
}

// isEnvironmentToken checks if a token represents an environment name
func (p *Parser) isEnvironmentToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.CI, lexer.LOCAL, lexer.PRODUCTION, lexer.STAGING, lexer.DEVELOPMENT:
		return true
	default:
		return false
	}
}

// isDetectionContext checks if the current context suggests a detection statement
func (p *Parser) isDetectionContext() bool {
	switch p.curToken.Type {
	case lexer.DETECT:
		return true
	case lexer.IF:
		// Check if this is "if <tool> is available" or "if <tool> version ..."
		return p.isToolToken(p.peekToken.Type) || p.peekToken.Type == lexer.STRING
	case lexer.WHEN:
		// Check if this is "when in <environment> environment"
		return p.peekToken.Type == lexer.IN
	default:
		return false
	}
}

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
	case lexer.REQUIRES, lexer.GIVEN, lexer.ACCEPTS:
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
		lexer.DETECT, lexer.GIVEN, lexer.REQUIRES, lexer.DEFAULTS, lexer.BREAK, lexer.CONTINUE:
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
func (p *Parser) parseControlFlowStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.WHEN:
		return p.parseWhenStatement()
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.FOR:
		return p.parseForStatement()
	default:
		p.addError(fmt.Sprintf("unexpected control flow token: %s", p.curToken.Type))
		return nil
	}
}

// parseWhenStatement parses when statements: when condition: ... otherwise: ...
func (p *Parser) parseWhenStatement() *ast.ConditionalStatement {
	stmt := &ast.ConditionalStatement{
		Token: p.curToken,
		Type:  "when",
	}

	// Parse condition (everything until colon)
	condition := p.parseConditionExpression()
	if condition == "" {
		return nil
	}
	stmt.Condition = condition

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse body
	stmt.Body = p.parseControlFlowBody()

	// Check for otherwise clause
	if p.peekToken.Type == lexer.OTHERWISE {
		p.nextToken() // consume OTHERWISE
		if !p.expectPeek(lexer.COLON) {
			return nil
		}
		stmt.ElseBody = p.parseControlFlowBody()
	}

	return stmt
}

// parseIfStatement parses if/else statements
func (p *Parser) parseIfStatement() *ast.ConditionalStatement {
	stmt := &ast.ConditionalStatement{
		Token: p.curToken,
		Type:  "if",
	}

	// Parse condition
	condition := p.parseConditionExpression()
	if condition == "" {
		return nil
	}
	stmt.Condition = condition

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse if body
	stmt.Body = p.parseControlFlowBody()

	// Check for else clause
	if p.peekToken.Type == lexer.ELSE {
		p.nextToken() // consume ELSE

		// Check if this is "else if" (else followed by if)
		if p.peekToken.Type == lexer.IF {
			// This is an "else if" - parse it as a nested if statement
			// Set current token to IF so parseIfStatement works correctly
			p.nextToken() // consume IF, now curToken is IF
			elseIfStmt := p.parseIfStatement()
			if elseIfStmt != nil {
				stmt.ElseBody = []ast.Statement{elseIfStmt}
			}
		} else {
			// This is a regular "else" - expect colon and parse body
			if !p.expectPeek(lexer.COLON) {
				return nil
			}
			stmt.ElseBody = p.parseControlFlowBody()
		}
	}

	return stmt
}

// parseForStatement parses for loops (each, range, line, match)
func (p *Parser) parseForStatement() *ast.LoopStatement {
	stmt := &ast.LoopStatement{
		Token: p.curToken,
	}

	// Check what type of for loop this is
	switch p.peekToken.Type {
	case lexer.EACH:
		return p.parseForEachStatement(stmt)
	case lexer.IDENT, lexer.VARIABLE:
		// This could be "for i in range" or "for variable in iterable"
		return p.parseForVariableStatement(stmt)
	default:
		p.addError(fmt.Sprintf("unexpected token after for: %s", p.peekToken.Type))
		return nil
	}
}

// parseForEachStatement parses "for each" loops
func (p *Parser) parseForEachStatement(stmt *ast.LoopStatement) *ast.LoopStatement {
	stmt.Type = "each"

	if !p.expectPeek(lexer.EACH) {
		return nil
	}

	// Check for special each types: "line" or "match"
	switch p.peekToken.Type {
	case lexer.LINE:
		p.nextToken() // consume LINE
		stmt.Type = "line"

		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer.IN) {
			return nil
		}

		if !p.expectPeekFileKeyword() {
			return nil
		}

		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Iterable = p.curToken.Literal

	case lexer.MATCH:
		p.nextToken() // consume MATCH
		stmt.Type = "match"

		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer.IN) {
			return nil
		}

		if !p.expectPeek(lexer.PATTERN) {
			return nil
		}

		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Iterable = p.curToken.Literal

	default:
		// Regular "for each variable in iterable"
		if !p.expectPeekIdentifierLike() {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer.IN) {
			return nil
		}

		// Accept IDENT, VARIABLE, and array literals for iterable
		switch p.peekToken.Type {
		case lexer.IDENT, lexer.VARIABLE:
			p.nextToken()
			stmt.Iterable = p.curToken.Literal
		case lexer.LBRACKET:
			// Parse array literal as iterable
			p.nextToken()
			arrayExpr := p.parseArrayLiteral()
			if arrayExpr != nil {
				stmt.Iterable = arrayExpr.String()
			} else {
				return nil
			}
		default:
			p.addError(fmt.Sprintf("expected identifier, variable, or array literal for iterable, got %s", p.peekToken.Type))
			return nil
		}
	}

	// Check for filter: "where variable operator value"
	if p.peekToken.Type == lexer.WHERE {
		stmt.Filter = p.parseFilterExpression()
	}

	// Check for "in parallel"
	if p.peekToken.Type == lexer.IN && p.peekToken.Literal == "in" {
		p.nextToken() // consume IN
		if p.peekToken.Type == lexer.PARALLEL {
			p.nextToken() // consume PARALLEL
			stmt.Parallel = true
		}
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse body
	stmt.Body = p.parseControlFlowBody()

	return stmt
}

// parseForVariableStatement parses "for variable in range" or "for variable in iterable"
func (p *Parser) parseForVariableStatement(stmt *ast.LoopStatement) *ast.LoopStatement {
	// Accept both IDENT and VARIABLE tokens
	switch p.peekToken.Type {
	case lexer.IDENT, lexer.VARIABLE:
		p.nextToken()
		stmt.Variable = p.curToken.Literal
	default:
		p.addError(fmt.Sprintf("expected identifier or variable, got %s instead", p.peekToken.Type))
		return nil
	}

	if !p.expectPeek(lexer.IN) {
		return nil
	}

	// Check if this is a range loop
	if p.peekToken.Type == lexer.RANGE {
		p.nextToken() // consume RANGE
		stmt.Type = "range"

		// Parse range: start to end [step step_value]
		if !p.expectPeek(lexer.NUMBER) && !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.RangeStart = p.curToken.Literal

		if !p.expectPeek(lexer.TO) {
			return nil
		}

		if !p.expectPeek(lexer.NUMBER) && !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.RangeEnd = p.curToken.Literal

		// Optional step
		if p.peekToken.Type == lexer.STEP {
			p.nextToken() // consume STEP
			if !p.expectPeek(lexer.NUMBER) && !p.expectPeek(lexer.IDENT) {
				return nil
			}
			stmt.RangeStep = p.curToken.Literal
		}

	} else {
		// Regular "for variable in iterable"
		stmt.Type = "each"
		// Accept IDENT, VARIABLE, and array literals for iterable
		switch p.peekToken.Type {
		case lexer.IDENT, lexer.VARIABLE:
			p.nextToken()
			stmt.Iterable = p.curToken.Literal
		case lexer.LBRACKET:
			// Parse array literal as iterable
			p.nextToken()
			arrayExpr := p.parseArrayLiteral()
			if arrayExpr != nil {
				stmt.Iterable = arrayExpr.String()
			} else {
				return nil
			}
		default:
			p.addError(fmt.Sprintf("expected identifier, variable, or array literal for iterable, got %s", p.peekToken.Type))
			return nil
		}
	}

	// Check for filter: "where variable operator value"
	if p.peekToken.Type == lexer.WHERE {
		stmt.Filter = p.parseFilterExpression()
	}

	// Check for "in parallel"
	if p.peekToken.Type == lexer.IN && p.peekToken.Literal == "in" {
		p.nextToken() // consume IN
		if p.peekToken.Type == lexer.PARALLEL {
			p.nextToken() // consume PARALLEL
			stmt.Parallel = true
		}
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse body
	stmt.Body = p.parseControlFlowBody()

	return stmt
}

// parseFilterExpression parses filter conditions like "where item contains 'test'"
func (p *Parser) parseFilterExpression() *ast.FilterExpression {
	if !p.expectPeek(lexer.WHERE) {
		return nil
	}

	filter := &ast.FilterExpression{}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	filter.Variable = p.curToken.Literal

	// Parse operator
	p.nextToken()

	switch p.curToken.Type {
	case lexer.CONTAINS:
		filter.Operator = p.curToken.Literal
	case lexer.STARTS:
		filter.Operator = p.curToken.Literal
		// Check for "starts with"
		if p.peekToken.Type == lexer.WITH {
			p.nextToken() // consume WITH
			filter.Operator = "starts with"
		}
	case lexer.ENDS:
		filter.Operator = p.curToken.Literal
		// Check for "ends with"
		if p.peekToken.Type == lexer.WITH {
			p.nextToken() // consume WITH
			filter.Operator = "ends with"
		}
	case lexer.MATCHES:
		filter.Operator = p.curToken.Literal
	case lexer.EQ, lexer.NE, lexer.GT, lexer.GTE, lexer.LT, lexer.LTE:
		filter.Operator = p.curToken.Literal
	default:
		p.addError(fmt.Sprintf("unexpected filter operator: %s", p.curToken.Type))
		return nil
	}

	// Parse value
	if !p.expectPeek(lexer.STRING) && !p.expectPeek(lexer.IDENT) && !p.expectPeek(lexer.NUMBER) {
		return nil
	}
	filter.Value = p.curToken.Literal

	return filter
}

// isBreakContinueToken checks if a token represents break or continue
func (p *Parser) isBreakContinueToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.BREAK, lexer.CONTINUE:
		return true
	default:
		return false
	}
}

// parseBreakContinueStatement parses break and continue statements
func (p *Parser) parseBreakContinueStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.BREAK:
		return p.parseBreakStatement()
	case lexer.CONTINUE:
		return p.parseContinueStatement()
	default:
		p.addError(fmt.Sprintf("unexpected break/continue token: %s", p.curToken.Type))
		return nil
	}
}

// parseBreakStatement parses break statements
func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	stmt := &ast.BreakStatement{
		Token: p.curToken,
	}

	// Check for conditional break: "break when condition"
	if p.peekToken.Type == lexer.WHEN {
		p.nextToken() // consume WHEN
		stmt.Condition = p.parseSimpleCondition()
	}

	return stmt
}

// parseContinueStatement parses continue statements
func (p *Parser) parseContinueStatement() *ast.ContinueStatement {
	stmt := &ast.ContinueStatement{
		Token: p.curToken,
	}

	// Check for conditional continue: "continue if condition"
	if p.peekToken.Type == lexer.IF {
		p.nextToken() // consume IF
		stmt.Condition = p.parseSimpleCondition()
	}

	return stmt
}

// isVariableOperationToken checks if a token represents variable operations
func (p *Parser) isVariableOperationToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.LET, lexer.SET, lexer.TRANSFORM, lexer.CAPTURE:
		return true
	default:
		return false
	}
}

// parseVariableStatement parses variable operation statements
func (p *Parser) parseVariableStatement() *ast.VariableStatement {
	stmt := &ast.VariableStatement{
		Token: p.curToken,
	}

	switch p.curToken.Type {
	case lexer.LET:
		return p.parseLetStatement(stmt)
	case lexer.SET:
		return p.parseSetVariableStatement(stmt)
	case lexer.TRANSFORM:
		return p.parseTransformStatement(stmt)
	case lexer.CAPTURE:
		return p.parseCaptureVariableStatement(stmt)
	default:
		p.addError(fmt.Sprintf("unexpected variable operation token: %s", p.curToken.Type))
		return nil
	}
}

// parseLetStatement parses "let variable = value" or "let variable be expression" statements
func (p *Parser) parseLetStatement(stmt *ast.VariableStatement) *ast.VariableStatement {
	stmt.Operation = "let"

	// Check if next token is $variable (old syntax) or identifier (new syntax)
	switch p.peekToken.Type {
	case lexer.VARIABLE:
		// Old syntax: "let $variable = value"
		if !p.expectPeekVariableName() {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer.EQUALS) {
			return nil
		}

		// Parse the value (could be string, number, or expression)
		p.nextToken()

		stmt.Value = p.parseExpression()
		return stmt
	case lexer.IDENT:
		// New syntax: "let variable be expression"
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer.BE) {
			return nil
		}

		// Parse the expression after "be"
		p.nextToken()
		stmt.Value = p.parseExpression()
		return stmt
	default:
		p.addError("expected variable name or identifier after 'let'")
		return nil
	}
}

// parseSetVariableStatement parses "set variable to value" statements
func (p *Parser) parseSetVariableStatement(stmt *ast.VariableStatement) *ast.VariableStatement {
	stmt.Operation = "set"

	if !p.expectPeekVariableName() {
		return nil
	}
	stmt.Variable = p.curToken.Literal

	if !p.expectPeek(lexer.TO) {
		return nil
	}

	// Parse the value
	p.nextToken()

	stmt.Value = p.parseExpression()
	return stmt
}

// parseTransformStatement parses "transform variable with function args" statements
func (p *Parser) parseTransformStatement(stmt *ast.VariableStatement) *ast.VariableStatement {
	stmt.Operation = "transform"

	if !p.expectPeekVariableName() {
		return nil
	}
	stmt.Variable = p.curToken.Literal

	if !p.expectPeek(lexer.WITH) {
		return nil
	}

	// Parse the function name (can be IDENT or reserved keywords like CONCAT, UPPERCASE, etc.)
	if !p.expectPeekFunctionName() {
		return nil
	}
	stmt.Function = p.curToken.Literal

	// Parse optional arguments
	for p.peekToken.Type != lexer.NEWLINE && p.peekToken.Type != lexer.DEDENT && p.peekToken.Type != lexer.EOF && p.peekToken.Type != lexer.COMMENT && p.peekToken.Type != lexer.MULTILINE_COMMENT {
		// Check if the next token looks like an argument
		if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.IDENT || p.peekToken.Type == lexer.NUMBER {
			p.nextToken()
			argValue := p.curToken.Literal
			// For identifiers, mark them for interpolation
			if p.curToken.Type == lexer.IDENT {
				argValue = "{" + p.curToken.Literal + "}"
			}
			stmt.Arguments = append(stmt.Arguments, argValue)
		} else {
			// Stop parsing arguments if we hit an unexpected token
			break
		}
	}

	return stmt
}

// parseCaptureVariableStatement parses "capture variable from expression" and "capture from shell command as $variable" statements
func (p *Parser) parseCaptureVariableStatement(stmt *ast.VariableStatement) *ast.VariableStatement {
	stmt.Operation = "capture"

	// Check if next token is "from" (shell syntax) or identifier (expression syntax)
	switch p.peekToken.Type {
	case lexer.FROM:
		// Shell syntax: "capture from shell "command" as $variable" or "capture from shell as $variable:"
		if !p.expectPeek(lexer.FROM) {
			return nil
		}

		if !p.expectPeek(lexer.SHELL) {
			return nil
		}

		// Check if this is multiline syntax (as $var:) or single-line syntax ("command" as $var)
		if p.peekToken.Type == lexer.AS {
			// Multiline syntax: "capture from shell as $variable:"
			if !p.expectPeek(lexer.AS) {
				return nil
			}

			if !p.expectPeekVariableName() {
				return nil
			}
			stmt.Variable = p.curToken.Literal

			if !p.expectPeek(lexer.COLON) {
				return nil
			}

			// Parse multiline commands
			return p.parseMultilineShellCapture(stmt)
		} else {
			// Single-line syntax: "capture from shell "command" as $variable"
			if !p.expectPeek(lexer.STRING) {
				return nil
			}
			command := p.curToken.Literal

			if !p.expectPeek(lexer.AS) {
				return nil
			}

			if !p.expectPeekVariableName() {
				return nil
			}
			stmt.Variable = p.curToken.Literal

			// Mark this as a shell capture by setting a special operation
			stmt.Operation = "capture_shell"

			// Create a literal expression for the command
			stmt.Value = &ast.LiteralExpression{
				Token: p.curToken,
				Value: command,
			}
			return stmt
		}
	case lexer.IDENT:
		// Expression syntax: "capture variable from expression"
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer.FROM) {
			return nil
		}

		// Parse the expression after "from"
		p.nextToken()
		stmt.Value = p.parseExpression()
		return stmt
	default:
		p.addError("expected 'from shell' or variable name after 'capture'")
		return nil
	}
}

// parseMultilineShellCapture parses multiline shell capture commands
func (p *Parser) parseMultilineShellCapture(stmt *ast.VariableStatement) *ast.VariableStatement {
	// Mark this as a shell capture by setting a special operation
	stmt.Operation = "capture_shell"

	// Expect INDENT (skip any newlines first)
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}

	// Parse command tokens until DEDENT
	p.nextToken() // Move to first token inside the block

	// Read all commands in the block
	commands := p.readCommandTokens()

	// Join commands with newlines to create a single script
	script := strings.Join(commands, "\n")

	// Create a literal expression for the combined script
	stmt.Value = &ast.LiteralExpression{
		Token: p.curToken,
		Value: script,
	}

	return stmt
}

// parseExpression parses expressions with proper precedence
func (p *Parser) parseExpression() ast.Expression {
	return p.parseInfixExpression()
}

// parseInfixExpression parses binary expressions with operator precedence
func (p *Parser) parseInfixExpression() ast.Expression {
	left := p.parsePrimaryExpression()
	if left == nil {
		return nil
	}

	// Check for binary operators
	for p.peekToken.Type == lexer.PLUS || p.peekToken.Type == lexer.MINUS ||
		p.peekToken.Type == lexer.STAR || p.peekToken.Type == lexer.SLASH ||
		p.peekToken.Type == lexer.EQUALS ||
		p.peekToken.Type == lexer.GT || p.peekToken.Type == lexer.LT ||
		p.peekToken.Type == lexer.GTE || p.peekToken.Type == lexer.LTE ||
		p.peekToken.Type == lexer.EQ || p.peekToken.Type == lexer.NE {

		operator := p.peekToken.Literal
		p.nextToken() // consume operator
		p.nextToken() // move to right operand

		right := p.parsePrimaryExpression()
		if right == nil {
			return nil
		}

		left = &ast.BinaryExpression{
			Token:    p.curToken,
			Left:     left,
			Operator: operator,
			Right:    right,
		}
	}

	return left
}

// parsePrimaryExpression parses primary expressions (literals, identifiers, function calls)
func (p *Parser) parsePrimaryExpression() ast.Expression {
	switch p.curToken.Type {
	case lexer.STRING:
		return &ast.LiteralExpression{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
	case lexer.NUMBER:
		return &ast.LiteralExpression{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
	case lexer.BOOLEAN:
		return &ast.LiteralExpression{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
	case lexer.LBRACE:
		// Parse {expression} - could be single identifier or multi-word expression
		p.nextToken() // consume LBRACE

		var parts []string

		// Read tokens until RBRACE
		for p.curToken.Type != lexer.RBRACE && p.curToken.Type != lexer.EOF {
			parts = append(parts, p.curToken.Literal)
			p.nextToken()
		}

		if p.curToken.Type != lexer.RBRACE {
			p.addError("expected '}' to close brace expression")
			return nil
		}

		// Join the parts to create the expression and preserve braces for interpolation
		expression := "{" + strings.Join(parts, " ") + "}"

		return &ast.LiteralExpression{
			Token: p.curToken,
			Value: expression,
		}
	case lexer.IDENT:
		// Handle function calls like "now" or simple identifiers
		if p.peekToken.Type == lexer.LPAREN {
			// Function call with parentheses
			return p.parseFunctionCall()
		} else {
			// Simple function call or identifier
			return &ast.FunctionCallExpression{
				Token:     p.curToken,
				Function:  p.curToken.Literal,
				Arguments: []ast.Expression{},
			}
		}
	case lexer.VARIABLE:
		// Handle $variable references
		return &ast.IdentifierExpression{
			Token: p.curToken,
			Value: p.curToken.Literal, // This includes the $ prefix
		}
	case lexer.LBRACKET:
		// Parse array literal ["item1", "item2", "item3"]
		return p.parseArrayLiteral()
	default:
		p.addError(fmt.Sprintf("unexpected token in expression: %s", p.curToken.Type))
		return nil
	}
}

// parseFunctionCall parses function calls with arguments
func (p *Parser) parseFunctionCall() ast.Expression {
	function := p.curToken.Literal

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	var args []ast.Expression
	if p.peekToken.Type != lexer.RPAREN {
		p.nextToken()
		args = append(args, p.parseExpression())

		for p.peekToken.Type == lexer.COMMA {
			p.nextToken() // consume comma
			p.nextToken() // move to next argument
			args = append(args, p.parseExpression())
		}
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return &ast.FunctionCallExpression{
		Token:     p.curToken,
		Function:  function,
		Arguments: args,
	}
}

// parseArrayLiteral parses array literals like ["item1", "item2", "item3"]
func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{
		Token:    p.curToken, // LBRACKET
		Elements: []ast.Expression{},
	}

	// Handle empty array []
	if p.peekToken.Type == lexer.RBRACKET {
		p.nextToken() // consume RBRACKET
		return array
	}

	// Parse first element
	p.nextToken()
	array.Elements = append(array.Elements, p.parseExpression())

	// Parse remaining elements separated by commas
	for p.peekToken.Type == lexer.COMMA {
		p.nextToken() // consume comma
		p.nextToken() // move to next element
		array.Elements = append(array.Elements, p.parseExpression())
	}

	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	return array
}

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
	if p.peekToken.Type == lexer.IDENT {
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
func (p *Parser) parseControlFlowBody() []ast.Statement {
	var body []ast.Statement

	// Expect INDENT
	if !p.expectPeek(lexer.INDENT) {
		return body
	}

	// Parse statements until DEDENT
	for p.peekToken.Type != lexer.DEDENT && p.peekToken.Type != lexer.EOF {
		p.nextToken()

		if p.isDetectionToken(p.curToken.Type) && p.isDetectionContext() {
			detection := p.parseDetectionStatement()
			if detection != nil {
				body = append(body, detection)
			}
		} else if p.isThrowActionToken(p.curToken.Type) {
			throw := p.parseThrowStatement()
			if throw != nil {
				body = append(body, throw)
			}
		} else if p.isDockerToken(p.curToken.Type) {
			// Special handling for RUN token - check context
			if p.curToken.Type == lexer.RUN {
				// Look ahead to determine if this is shell or docker command
				if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.COLON {
					// This is "run 'command'" or "run:" - shell command
					shell := p.parseShellStatement()
					if shell != nil {
						body = append(body, shell)
					}
				} else {
					// This is "docker run container" - docker command
					docker := p.parseDockerStatement()
					if docker != nil {
						body = append(body, docker)
					}
				}
			} else {
				docker := p.parseDockerStatement()
				if docker != nil {
					body = append(body, docker)
				}
			}
		} else if p.isGitToken(p.curToken.Type) {
			git := p.parseGitStatement()
			if git != nil {
				body = append(body, git)
			}
		} else if p.isHTTPToken(p.curToken.Type) {
			http := p.parseHTTPStatement()
			if http != nil {
				body = append(body, http)
			}
		} else if p.isNetworkToken(p.curToken.Type) {
			network := p.parseNetworkStatement()
			if network != nil {
				body = append(body, network)
			}
		} else if p.isBreakContinueToken(p.curToken.Type) {
			breakContinue := p.parseBreakContinueStatement()
			if breakContinue != nil {
				body = append(body, breakContinue)
			}
		} else if p.isVariableOperationToken(p.curToken.Type) {
			variable := p.parseVariableStatement()
			if variable != nil {
				body = append(body, variable)
			}
		} else if p.isActionToken(p.curToken.Type) {
			if p.isShellActionToken(p.curToken.Type) {
				shell := p.parseShellStatement()
				if shell != nil {
					body = append(body, shell)
				}
			} else if p.isFileActionToken(p.curToken.Type) {
				// Special handling for CHECK token
				if p.curToken.Type == lexer.CHECK {
					switch p.peekToken.Type {
					case lexer.HEALTH:
						// Definitely a network health check
						network := p.parseNetworkStatement()
						if network != nil {
							body = append(body, network)
						}
					case lexer.IF:
						// This is "check if X" - determine if it's a port check
						if p.isPortCheckPattern() {
							// This is "check if port" - network operation
							network := p.parseNetworkStatement()
							if network != nil {
								body = append(body, network)
							}
						} else {
							// This is "check if file" or other - file operation
							file := p.parseFileStatement()
							if file != nil {
								body = append(body, file)
							}
						}
					default:
						// Other check operations (check size, etc.) - file operations
						file := p.parseFileStatement()
						if file != nil {
							body = append(body, file)
						}
					}
				} else {
					// Regular file operation
					file := p.parseFileStatement()
					if file != nil {
						body = append(body, file)
					}
				}
			} else if p.isThrowActionToken(p.curToken.Type) {
				throw := p.parseThrowStatement()
				if throw != nil {
					body = append(body, throw)
				}
			} else {
				action := p.parseActionStatement()
				if action != nil {
					body = append(body, action)
				}
			}
		} else if p.isControlFlowToken(p.curToken.Type) {
			controlFlow := p.parseControlFlowStatement()
			if controlFlow != nil {
				body = append(body, controlFlow)
			}
		} else if p.isErrorHandlingToken(p.curToken.Type) {
			errorHandling := p.parseErrorHandlingStatement()
			if errorHandling != nil {
				body = append(body, errorHandling)
			}
		} else if p.isCallToken(p.curToken.Type) {
			call := p.parseTaskCallStatement()
			if call != nil {
				body = append(body, call)
			}
		} else if p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT {
			// Skip comments
			continue
		} else if p.curToken.Type == lexer.NEWLINE {
			// Skip newlines
			continue
		} else {
			p.addError(fmt.Sprintf("unexpected token in control flow body: %s", p.curToken.Type))
			break
		}
	}

	// Consume DEDENT
	if p.peekToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return body
}
