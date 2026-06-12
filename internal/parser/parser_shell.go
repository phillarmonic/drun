package parser

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// parseShellStatement parses a shell command statement (run, exec, shell, capture)
func (p *Parser) parseShellStatement() *ast.ShellStatement {
	stmt := &ast.ShellStatement{
		Token:  p.curToken,
		Action: p.curToken.Literal,
	}

	// Optional service scoping for run commands
	if stmt.Action == "run" && p.peekToken.Type == lexer.IN {
		p.nextToken() // consume IN
		if p.peekToken.Type == lexer.SERVICE || (p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "service") {
			p.nextToken() // consume SERVICE keyword
		} else {
			p.addError("expected 'service' keyword after 'in'")
			return nil
		}

		name, isLiteral, ok := p.parseServiceReference()
		if !ok {
			return nil
		}
		stmt.ServiceScoped = true
		stmt.ServiceName = name
		stmt.ServiceNameIsLiteral = isLiteral
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
	if p.peekToken.Type != lexer.STRING {
		p.addErrorWithHelpAtPeek(
			fmt.Sprintf("expected command string after '%s', got %s instead", stmt.Action, p.peekToken.Type),
			fmt.Sprintf("Shell commands require a quoted string. Example: %s \"your command here\"", stmt.Action),
		)
		return nil
	}
	p.nextToken() // consume STRING

	stmt.Command = p.curToken.Literal
	if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "attached" {
		if stmt.Action != "run" {
			p.addError("attached modifier is only supported for run statements")
			return nil
		}
		p.nextToken() // consume attached
		stmt.Attached = true
	}

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
			fmt.Fprintf(&currentLine, "\"%s\"", p.curToken.Literal)
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

// parseServiceReference parses a service name reference, allowing literals or variables
func (p *Parser) parseServiceReference() (string, bool, bool) {
	switch p.peekToken.Type {
	case lexer.STRING:
		p.nextToken()
		return p.curToken.Literal, true, true
	case lexer.VARIABLE, lexer.IDENT:
		p.nextToken()
		return p.curToken.Literal, false, true
	default:
		p.addError("expected service name (string or variable)")
		return "", false, false
	}
}
