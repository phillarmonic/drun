package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

// parseSecretStatement parses secret management statements
// Syntax:
//   secret set "key" to "value" [in namespace "ns"]
//   secret get "key" [from namespace "ns"] [or "default"]
//   secret delete "key" [from namespace "ns"]
//   secret exists "key" [from namespace "ns"]
//   secret list [matching "pattern"] [from namespace "ns"]
func (p *Parser) parseSecretStatement() *ast.SecretStatement {
	stmt := &ast.SecretStatement{
		Token: p.curToken,
	}

	// Expect operation (set, get, delete, exists, list)
	if !p.expectPeek(lexer.IDENT) {
		p.addError("expected secret operation (set, get, delete, exists, list)")
		return nil
	}

	stmt.Operation = p.curToken.Literal

	switch stmt.Operation {
	case "set":
		return p.parseSecretSetStatement(stmt)
	case "get":
		return p.parseSecretGetStatement(stmt)
	case "delete":
		return p.parseSecretDeleteStatement(stmt)
	case "exists":
		return p.parseSecretExistsStatement(stmt)
	case "list":
		return p.parseSecretListStatement(stmt)
	default:
		p.addError(fmt.Sprintf("unknown secret operation: %s (expected: set, get, delete, exists, list)", stmt.Operation))
		return nil
	}
}

// parseSecretSetStatement parses: secret set "key" to "value" [in namespace "ns"]
func (p *Parser) parseSecretSetStatement(stmt *ast.SecretStatement) *ast.SecretStatement {
	// Expect key (string)
	if !p.expectPeek(lexer.STRING) {
		p.addError("expected secret key (string)")
		return nil
	}
	stmt.Key = p.curToken.Literal

	// Expect "to"
	if !p.expectPeek(lexer.TO) {
		p.addError("expected 'to' after secret key")
		return nil
	}

	// Parse value expression
	p.nextToken()
	stmt.Value = p.parseExpression()

	// Optional: in namespace "ns"
	if p.peekToken.Type == lexer.IN {
		p.nextToken() // consume IN
		if !p.expectPeek(lexer.NAMESPACE) {
			p.addError("expected 'namespace' after 'in'")
			return nil
		}
		if !p.expectPeek(lexer.STRING) {
			p.addError("expected namespace name (string)")
			return nil
		}
		stmt.Namespace = p.curToken.Literal
	}

	return stmt
}

// parseSecretGetStatement parses: secret get "key" [from namespace "ns"] [or "default"]
func (p *Parser) parseSecretGetStatement(stmt *ast.SecretStatement) *ast.SecretStatement {
	// Expect key (string)
	if !p.expectPeek(lexer.STRING) {
		p.addError("expected secret key (string)")
		return nil
	}
	stmt.Key = p.curToken.Literal

	// Optional: from namespace "ns"
	if p.peekToken.Type == lexer.FROM {
		p.nextToken() // consume FROM
		if !p.expectPeek(lexer.NAMESPACE) {
			p.addError("expected 'namespace' after 'from'")
			return nil
		}
		if !p.expectPeek(lexer.STRING) {
			p.addError("expected namespace name (string)")
			return nil
		}
		stmt.Namespace = p.curToken.Literal
	}

	// Optional: or "default_value"
	if p.peekToken.Type == lexer.OR {
		p.nextToken() // consume OR
		p.nextToken() // move to value
		stmt.Default = p.parseExpression()
	}

	return stmt
}

// parseSecretDeleteStatement parses: secret delete "key" [from namespace "ns"]
func (p *Parser) parseSecretDeleteStatement(stmt *ast.SecretStatement) *ast.SecretStatement {
	// Expect key (string)
	if !p.expectPeek(lexer.STRING) {
		p.addError("expected secret key (string)")
		return nil
	}
	stmt.Key = p.curToken.Literal

	// Optional: from namespace "ns"
	if p.peekToken.Type == lexer.FROM {
		p.nextToken() // consume FROM
		if !p.expectPeek(lexer.NAMESPACE) {
			p.addError("expected 'namespace' after 'from'")
			return nil
		}
		if !p.expectPeek(lexer.STRING) {
			p.addError("expected namespace name (string)")
			return nil
		}
		stmt.Namespace = p.curToken.Literal
	}

	return stmt
}

// parseSecretExistsStatement parses: secret exists "key" [from namespace "ns"]
func (p *Parser) parseSecretExistsStatement(stmt *ast.SecretStatement) *ast.SecretStatement {
	// Expect key (string)
	if !p.expectPeek(lexer.STRING) {
		p.addError("expected secret key (string)")
		return nil
	}
	stmt.Key = p.curToken.Literal

	// Optional: from namespace "ns"
	if p.peekToken.Type == lexer.FROM {
		p.nextToken() // consume FROM
		if !p.expectPeek(lexer.NAMESPACE) {
			p.addError("expected 'namespace' after 'from'")
			return nil
		}
		if !p.expectPeek(lexer.STRING) {
			p.addError("expected namespace name (string)")
			return nil
		}
		stmt.Namespace = p.curToken.Literal
	}

	return stmt
}

// parseSecretListStatement parses: secret list [matching "pattern"] [from namespace "ns"]
func (p *Parser) parseSecretListStatement(stmt *ast.SecretStatement) *ast.SecretStatement {
	// Optional: matching "pattern"
	if p.peekToken.Type == lexer.MATCHING {
		p.nextToken() // consume MATCHING
		if !p.expectPeek(lexer.STRING) {
			p.addError("expected pattern (string) after 'matching'")
			return nil
		}
		stmt.Pattern = p.curToken.Literal
	}

	// Optional: from namespace "ns"
	if p.peekToken.Type == lexer.FROM {
		p.nextToken() // consume FROM
		if !p.expectPeek(lexer.NAMESPACE) {
			p.addError("expected 'namespace' after 'from'")
			return nil
		}
		if !p.expectPeek(lexer.STRING) {
			p.addError("expected namespace name (string)")
			return nil
		}
		stmt.Namespace = p.curToken.Literal
	}

	return stmt
}

