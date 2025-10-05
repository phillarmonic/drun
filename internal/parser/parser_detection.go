package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

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
			case lexer.IS, lexer.ARE:
				p.nextToken() // consume IS or ARE
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
