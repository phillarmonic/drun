package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

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
