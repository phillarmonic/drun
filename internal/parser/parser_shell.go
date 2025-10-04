package parser

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

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
