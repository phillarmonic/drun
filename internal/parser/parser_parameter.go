package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

// parseParameterStatement parses parameter declarations (requires, given, accepts)
func (p *Parser) parseParameterStatement() *ast.ParameterStatement {
	stmt := &ast.ParameterStatement{
		Token:    p.curToken,
		Type:     p.curToken.Literal,
		DataType: "string", // default type
	}

	// Parse parameter name (accept both $variable and bare identifier)
	if p.peekToken.Type != lexer.VARIABLE && p.peekToken.Type != lexer.IDENT {
		p.addError(fmt.Sprintf("expected parameter name, got %s instead", p.peekToken.Type))
		return nil
	}
	p.nextToken()
	// Store parameter name without the $ prefix if present
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
			if p.peekToken.Type != lexer.TO {
				p.addErrorWithHelpAtPeek(
					fmt.Sprintf("expected 'to' after 'defaults', got %s instead", p.peekToken.Type),
					"Use 'defaults to' for default values. Example: requires $env defaults to \"dev\"",
				)
				return nil
			}
			p.nextToken() // consume TO

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

		// Check for constraints BEFORE defaults: given $env from ["dev", "staging"] defaults to "dev"
		if p.peekToken.Type == lexer.FROM {
			p.nextToken() // consume FROM
			if p.peekToken.Type == lexer.LBRACKET {
				p.nextToken() // consume LBRACKET
				stmt.Constraints = p.parseStringList()
			}
		}

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

		// Check for constraints after default value (legacy syntax): given name defaults to "value" from ["list"]
		if p.peekToken.Type == lexer.FROM {
			p.nextToken() // consume FROM
			if p.peekToken.Type == lexer.LBRACKET {
				p.nextToken() // consume LBRACKET
				stmt.Constraints = p.parseStringList()
			}
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
