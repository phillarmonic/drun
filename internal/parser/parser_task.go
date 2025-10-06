package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func (p *Parser) parseTaskStatement() *ast.TaskStatement {
	stmt := &ast.TaskStatement{Token: p.curToken}

	// Expect task name as a quoted string
	if p.peekToken.Type != lexer.STRING {
		// Provide helpful error message for unquoted task names
		p.addErrorWithHelpAtPeek(
			fmt.Sprintf("expected task name as quoted string, got %s instead", p.peekToken.Type),
			"Task names must be quoted. Use: task \""+p.peekToken.Literal+"\" instead of: task "+p.peekToken.Literal,
		)
		return nil
	}
	p.nextToken()

	stmt.Name = p.curToken.Literal

	// Check for optional "means" clause
	if p.peekToken.Type == lexer.MEANS {
		p.nextToken() // consume lexer.MEANS

		if !p.expectPeek(lexer.STRING) {
			return nil
		}

		stmt.Description = p.curToken.Literal
	}

	// Expect colon at end of task declaration
	if p.peekToken.Type != lexer.COLON {
		// Special error message pointing to end of current line, not next line
		p.addErrorWithHelp(
			fmt.Sprintf("expected ':' at end of task declaration, got %s on next line instead", p.peekToken.Type),
			"Add a ':' at the end of the task declaration line (after the task name or description)",
		)
		return nil
	}
	p.nextToken() // consume COLON

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
		} else if p.curToken.Type == lexer.USE {
			// Check for USE snippet
			if p.peekToken.Type == lexer.SNIPPET {
				p.nextToken() // consume SNIPPET

				if !p.expectPeek(lexer.STRING) {
					continue
				}

				useSnippet := &ast.UseSnippetStatement{
					Token:       p.curToken,
					SnippetName: p.curToken.Literal,
				}
				stmt.Body = append(stmt.Body, useSnippet)
			} else {
				p.addError(fmt.Sprintf("expected 'snippet' after 'use', got %s", p.peekToken.Type))
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

// parseTaskOrTemplateInstance determines if this is a regular task or a task from template
// parseTaskTemplateStatement parses a template task definition
// Syntax: template task "name": <parameters and body>
func (p *Parser) parseTaskTemplateStatement() *ast.TaskTemplateStatement {
	stmt := &ast.TaskTemplateStatement{Token: p.curToken}

	// Expect "task"
	if !p.expectPeek(lexer.TASK) {
		return nil
	}

	// Expect template name (string)
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Check for optional "means" clause
	if p.peekToken.Type == lexer.MEANS {
		p.nextToken() // consume MEANS

		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Description = p.curToken.Literal
	}

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Expect INDENT to start template body
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}

	// Parse template body (parameters and statements)
	for p.peekToken.Type != lexer.DEDENT && p.peekToken.Type != lexer.EOF {
		p.nextToken()

		// Skip newlines
		for p.curToken.Type == lexer.NEWLINE && p.peekToken.Type != lexer.DEDENT && p.peekToken.Type != lexer.EOF {
			p.nextToken()
		}

		if p.curToken.Type == lexer.DEDENT || p.curToken.Type == lexer.EOF {
			break
		}

		if p.isParameterToken(p.curToken.Type) {
			param := p.parseParameterStatement()
			if param != nil {
				stmt.Parameters = append(stmt.Parameters, *param)
			}
		} else {
			// Parse regular statements (delegate to existing statement parsing)
			// For now, we'll just collect the body statements
			bodyStmt := p.parseStatementInTaskBody()
			if bodyStmt != nil {
				stmt.Body = append(stmt.Body, bodyStmt)
			}
		}
	}

	// Consume DEDENT
	if p.peekToken.Type == lexer.DEDENT {
		p.nextToken() // Move to DEDENT
		p.nextToken() // Move past DEDENT
	}

	return stmt
}

// parseStatementInTaskBody is a helper that parses statements within a task or template body
func (p *Parser) parseStatementInTaskBody() ast.Statement {
	// Check for USE snippet
	if p.curToken.Type == lexer.USE {
		if p.peekToken.Type == lexer.SNIPPET {
			p.nextToken() // consume SNIPPET

			if !p.expectPeek(lexer.STRING) {
				return nil
			}

			return &ast.UseSnippetStatement{
				Token:       p.curToken,
				SnippetName: p.curToken.Literal,
			}
		}
	}

	// Handle control flow and statement keywords
	switch p.curToken.Type {
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.FOR:
		return p.parseForStatement()
	case lexer.WHEN:
		return p.parseWhenStatement()
	case lexer.CALL:
		return p.parseTaskCallStatement()
	case lexer.SET, lexer.LET:
		return p.parseVariableStatement()
	case lexer.TRY:
		return p.parseErrorHandlingStatement()
	}

	// Delegate to existing statement parsing logic
	if p.isActionToken(p.curToken.Type) {
		return p.parseActionStatement()
	} else if p.isShellActionToken(p.curToken.Type) {
		return p.parseShellStatement()
	}

	return nil
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
