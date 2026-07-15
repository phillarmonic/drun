package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

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
	case "replace":
		return p.parseReplaceStatement(stmt)
	default:
		p.addError(fmt.Sprintf("unknown file operation: %s", stmt.Action))
		return nil
	}
}

// parseCreateStatement parses "create file/dir" statements
func (p *Parser) parseCreateStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: create file "path" or create dir "path" or create directory "path"
	switch p.peekToken.Type {
	case lexer.FILE:
		p.nextToken() // consume FILE
		stmt.IsDir = false
	case lexer.DIR, lexer.DIRECTORY:
		p.nextToken() // consume DIR/DIRECTORY
		stmt.IsDir = true
	case lexer.IDENT:
		p.nextToken() // consume IDENT
		switch p.curToken.Literal {
		case "file":
			stmt.IsDir = false
		case "dir", "directory":
			stmt.IsDir = true
		default:
			p.addError("expected 'file', 'dir', or 'directory' after 'create'")
			return nil
		}
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
	case lexer.FILE:
		p.nextToken() // consume FILE
		stmt.IsDir = false
	case lexer.DIR, lexer.DIRECTORY:
		p.nextToken() // consume DIR/DIRECTORY
		stmt.IsDir = true
	case lexer.IDENT:
		p.nextToken() // consume IDENT
		switch p.curToken.Literal {
		case "file":
			stmt.IsDir = false
		case "dir", "directory":
			stmt.IsDir = true
		default:
			p.addError("expected 'file', 'dir', or 'directory' after 'delete'")
			return nil
		}
	default:
		p.addError("expected 'file', 'dir', or 'directory' after 'delete'")
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

// parseReplaceStatement parses "replace in" statements with multiple replacements
func (p *Parser) parseReplaceStatement(stmt *ast.FileStatement) *ast.FileStatement {
	stmt.Action = "replace"
	stmt.Replacements = make(map[string]string)

	if !p.expectPeek(lexer.IN) {
		p.addError("expected 'in' after 'replace'")
		return nil
	}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}

	// Move to first token inside the block
	p.nextToken()

	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		// Non-code lines do not belong to the replacement entry list and may
		// appear immediately before the DEDENT produced by the next code line.
		for p.curToken.Type == lexer.NEWLINE ||
			p.curToken.Type == lexer.COMMENT ||
			p.curToken.Type == lexer.MULTILINE_COMMENT {
			p.nextToken()
		}

		if p.curToken.Type == lexer.DEDENT || p.curToken.Type == lexer.EOF {
			break
		}

		if p.curToken.Type != lexer.STRING {
			p.addError(fmt.Sprintf("expected replacement pattern string, got %s", p.curToken.Type))
			p.nextToken()
			continue
		}

		oldValue := p.curToken.Literal

		if !p.expectPeek(lexer.WITH) {
			return nil
		}

		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Replacements[oldValue] = p.curToken.Literal

		// Move to next potential entry
		p.nextToken()
	}

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
