package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

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
	case lexer.VARIABLE:
		// This could be "for $i in range" or "for $variable in iterable"
		return p.parseForVariableStatement(stmt)
	default:
		p.addError(fmt.Sprintf("unexpected token after for: %s (variables must start with $)", p.peekToken.Type))
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
		// Regular "for each $variable in $iterable"
		if !p.expectPeek(lexer.VARIABLE) {
			p.addError("expected variable (with $ prefix) after 'each'")
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer.IN) {
			return nil
		}

		// Accept VARIABLE and array literals for iterable
		switch p.peekToken.Type {
		case lexer.VARIABLE:
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
			p.addError(fmt.Sprintf("expected variable (with $ prefix) or array literal for iterable, got %s", p.peekToken.Type))
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

// parseForVariableStatement parses "for $variable in range" or "for $variable in iterable"
func (p *Parser) parseForVariableStatement(stmt *ast.LoopStatement) *ast.LoopStatement {
	// Accept only VARIABLE tokens (must have $ prefix)
	switch p.peekToken.Type {
	case lexer.VARIABLE:
		p.nextToken()
		stmt.Variable = p.curToken.Literal
	default:
		p.addError(fmt.Sprintf("expected variable (with $ prefix), got %s instead", p.peekToken.Type))
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
		// Regular "for $variable in iterable"
		stmt.Type = "each"
		// Accept VARIABLE and array literals for iterable
		switch p.peekToken.Type {
		case lexer.VARIABLE:
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
			p.addError(fmt.Sprintf("expected variable (with $ prefix) or array literal for iterable, got %s", p.peekToken.Type))
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

	if !p.expectPeekIdentifierLike() {
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
				body = append(body, useSnippet)
			} else {
				p.addError(fmt.Sprintf("expected 'snippet' after 'use', got %s", p.peekToken.Type))
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
