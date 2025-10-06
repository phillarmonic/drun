package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

// Action and task call parsing methods

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

	// Check for optional "add line break before/after" for step actions
	if stmt.Action == "step" {
		for {
			if p.peekToken.Type == lexer.ADD {
				p.nextToken() // consume ADD
				if p.peekToken.Type == lexer.LINE {
					p.nextToken() // consume LINE
					if p.peekToken.Type == lexer.BREAK {
						p.nextToken() // consume BREAK
						switch p.peekToken.Type {
						case lexer.BEFORE:
							p.nextToken() // consume BEFORE
							stmt.LineBreakBefore = true
						case lexer.AFTER:
							p.nextToken() // consume AFTER
							stmt.LineBreakAfter = true
						}
					}
				}
			} else {
				break
			}
		}
	}

	return stmt
}

// parseTaskCallStatement parses a task call statement (call task "name" with param="value")
func (p *Parser) parseTaskCallStatement() *ast.TaskCallStatement {
	stmt := &ast.TaskCallStatement{
		Token:      p.curToken,
		Parameters: make(map[string]string),
	}

	// Expect "task" after "call"
	if p.peekToken.Type != lexer.TASK {
		p.addErrorWithHelpAtPeek(
			fmt.Sprintf("expected 'task' after 'call', got %s instead", p.peekToken.Type),
			"Use 'call task' to call another task. Example: call task \"build\" with env=\"production\"",
		)
		return nil
	}
	p.nextToken() // consume TASK

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
		// Stop when we hit tokens that indicate end of parameters
		for (p.peekToken.Type == lexer.IDENT || p.isKeywordToken(p.peekToken.Type)) &&
			p.peekToken.Type != lexer.NEWLINE && p.peekToken.Type != lexer.COMMENT &&
			p.peekToken.Type != lexer.DEDENT && p.peekToken.Type != lexer.EOF {

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
