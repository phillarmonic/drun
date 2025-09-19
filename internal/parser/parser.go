package parser

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/errors"
	lexer2 "github.com/phillarmonic/drun/internal/lexer"
)

// Parser parses drun v2 source code into an AST
type Parser struct {
	lexer *lexer2.Lexer

	curToken  lexer2.Token
	peekToken lexer2.Token

	errors    []string // Legacy error list for backward compatibility
	errorList *errors.ParseErrorList
}

// New creates a new parser instance
func NewParser(l *lexer2.Lexer) *Parser {
	p := &Parser{
		lexer:  l,
		errors: []string{},
	}

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// NewParserWithSource creates a new parser instance with source information for better error reporting
func NewParserWithSource(l *lexer2.Lexer, filename, source string) *Parser {
	p := &Parser{
		lexer:     l,
		errors:    []string{},
		errorList: errors.NewParseErrorList(filename, source),
	}

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// nextToken advances both curToken and peekToken
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// ParseProgram parses the entire program
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}

	// Skip any leading comments
	p.skipComments()

	// Parse version statement (required)
	if p.curToken.Type == lexer2.VERSION {
		program.Version = p.parseVersionStatement()
	} else {
		p.addError(fmt.Sprintf("expected version statement, got %s", p.curToken.Type))
		return nil
	}

	// Skip comments between version and project/tasks
	p.skipComments()

	// Parse optional project statement
	if p.curToken.Type == lexer2.PROJECT {
		program.Project = p.parseProjectStatement()
		p.skipComments()
	}

	// Parse task statements
	for p.curToken.Type != lexer2.EOF {
		switch p.curToken.Type {
		case lexer2.TASK:
			task := p.parseTaskStatement()
			if task != nil {
				program.Tasks = append(program.Tasks, task)
			}
		case lexer2.COMMENT:
			p.nextToken() // Skip comments
		case lexer2.DEDENT:
			// Skip stray lexer.DEDENT tokens (they should be consumed by task parsing)
			p.nextToken()
		default:
			p.addError(fmt.Sprintf("unexpected token: %s", p.curToken.Type))
			p.nextToken()
		}
	}

	return program
}

// parseVersionStatement parses a version statement
func (p *Parser) parseVersionStatement() *ast.VersionStatement {
	stmt := &ast.VersionStatement{Token: p.curToken}

	if !p.expectPeek(lexer2.COLON) {
		return nil
	}

	if !p.expectPeek(lexer2.NUMBER) {
		return nil
	}

	stmt.Value = p.curToken.Literal
	p.nextToken()

	return stmt
}

// parseProjectStatement parses a project statement
func (p *Parser) parseProjectStatement() *ast.ProjectStatement {
	stmt := &ast.ProjectStatement{Token: p.curToken}

	// Expect project name (string)
	if !p.expectPeek(lexer2.STRING) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Check for optional version
	if p.peekToken.Type == lexer2.VERSION {
		p.nextToken() // consume VERSION token
		if !p.expectPeek(lexer2.STRING) {
			return nil
		}
		stmt.Version = p.curToken.Literal
	}

	// Expect colon
	if !p.expectPeek(lexer2.COLON) {
		return nil
	}

	// Parse project settings (optional)
	if p.peekToken.Type == lexer2.INDENT {
		p.nextToken() // consume INDENT
		p.nextToken() // move to first token inside the block

		for p.curToken.Type != lexer2.DEDENT && p.curToken.Type != lexer2.EOF {
			switch p.curToken.Type {
			case lexer2.SET:
				setting := p.parseSetStatement()
				if setting != nil {
					stmt.Settings = append(stmt.Settings, setting)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer2.INCLUDE:
				setting := p.parseIncludeStatement()
				if setting != nil {
					stmt.Settings = append(stmt.Settings, setting)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer2.BEFORE, lexer2.AFTER:
				hook := p.parseLifecycleHook()
				if hook != nil {
					stmt.Settings = append(stmt.Settings, hook)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer2.SHELL:
				shellConfig := p.parseShellConfigStatement()
				if shellConfig != nil {
					stmt.Settings = append(stmt.Settings, shellConfig)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer2.COMMENT:
				p.nextToken() // Skip comments
			default:
				p.addError(fmt.Sprintf("unexpected token in project body: %s", p.curToken.Type))
				p.nextToken()
			}
		}

		if p.curToken.Type == lexer2.DEDENT {
			p.nextToken() // consume DEDENT
		}
	} else {
		// No INDENT found, advance to next token for proper parsing flow
		p.nextToken()
	}

	return stmt
}

// parseSetStatement parses a set statement (set key to value)
func (p *Parser) parseSetStatement() *ast.SetStatement {
	stmt := &ast.SetStatement{Token: p.curToken}

	// Expect identifier (key) - allow Git, HTTP, Docker, and File keywords as set keys
	switch p.peekToken.Type {
	case lexer2.IDENT, lexer2.MESSAGE, lexer2.BRANCH, lexer2.REMOTE, lexer2.STATUS, lexer2.LOG, lexer2.COMMIT, lexer2.ADD, lexer2.PUSH, lexer2.PULL,
		lexer2.GET, lexer2.POST, lexer2.PUT, lexer2.DELETE, lexer2.PATCH, lexer2.HEAD, lexer2.OPTIONS, lexer2.HTTP, lexer2.HTTPS, lexer2.URL, lexer2.API, lexer2.JSON, lexer2.XML,
		lexer2.TIMEOUT, lexer2.RETRY, lexer2.AUTH, lexer2.BEARER, lexer2.BASIC, lexer2.TOKEN, lexer2.HEADER, lexer2.BODY, lexer2.DATA,
		lexer2.SCALE, lexer2.PORT, lexer2.REGISTRY, lexer2.CHECKOUT, lexer2.BACKUP, lexer2.CHECK, lexer2.SIZE, lexer2.DIRECTORY:
		p.nextToken()
	default:
		p.addError(fmt.Sprintf("expected set key, got %s instead", p.peekToken.Type))
		return nil
	}
	stmt.Key = p.curToken.Literal

	// Expect "to"
	if !p.expectPeek(lexer2.TO) {
		return nil
	}

	// Expect value (string)
	if !p.expectPeek(lexer2.STRING) {
		return nil
	}
	stmt.Value = p.curToken.Literal

	p.nextToken()
	return stmt
}

// parseIncludeStatement parses an include statement
func (p *Parser) parseIncludeStatement() *ast.IncludeStatement {
	stmt := &ast.IncludeStatement{Token: p.curToken}

	// Expect path (string)
	if !p.expectPeek(lexer2.STRING) {
		return nil
	}
	stmt.Path = p.curToken.Literal

	p.nextToken()
	return stmt
}

// parseShellConfigStatement parses shell configuration for different platforms
func (p *Parser) parseShellConfigStatement() *ast.ShellConfigStatement {
	stmt := &ast.ShellConfigStatement{
		Token:     p.curToken,
		Platforms: make(map[string]*ast.PlatformShellConfig),
	}

	// Expect "config"
	if !p.expectPeek(lexer2.CONFIG) {
		return nil
	}

	// Expect colon
	if !p.expectPeek(lexer2.COLON) {
		return nil
	}

	// Expect indented block with platform configurations
	if !p.expectPeek(lexer2.INDENT) {
		return nil
	}

	p.nextToken() // move to first token inside the block

	for p.curToken.Type != lexer2.DEDENT && p.curToken.Type != lexer2.EOF {
		switch p.curToken.Type {
		case lexer2.IDENT:
			platform := p.curToken.Literal

			// Expect colon after platform name
			if !p.expectPeek(lexer2.COLON) {
				return nil
			}

			// Parse platform configuration
			config := p.parsePlatformShellConfig()
			if config != nil {
				stmt.Platforms[platform] = config
			}
		case lexer2.COMMENT:
			p.nextToken() // Skip comments
		default:
			p.addError(fmt.Sprintf("unexpected token in shell config: %s", p.curToken.Type))
			p.nextToken()
		}
	}

	p.nextToken() // consume DEDENT
	return stmt
}

// parsePlatformShellConfig parses configuration for a specific platform
func (p *Parser) parsePlatformShellConfig() *ast.PlatformShellConfig {
	config := &ast.PlatformShellConfig{
		Environment: make(map[string]string),
	}

	// Expect indented block
	if !p.expectPeek(lexer2.INDENT) {
		return nil
	}

	p.nextToken() // move to first token inside the block

	for p.curToken.Type != lexer2.DEDENT && p.curToken.Type != lexer2.EOF {
		switch p.curToken.Type {
		case lexer2.IDENT, lexer2.ENVIRONMENT:
			key := p.curToken.Literal

			// Expect colon
			if !p.expectPeek(lexer2.COLON) {
				return nil
			}

			switch key {
			case "executable":
				// Expect string value
				if !p.expectPeek(lexer2.STRING) {
					return nil
				}
				config.Executable = p.curToken.Literal
				p.nextToken()

			case "args":
				// Parse array of strings
				config.Args = p.parseStringArray()

			case "environment":
				// Parse key-value pairs
				envVars := p.parseKeyValuePairs()
				for k, v := range envVars {
					config.Environment[k] = v
				}

			default:
				p.addError(fmt.Sprintf("unknown shell config key: %s", key))
				p.nextToken()
			}
		case lexer2.COMMENT:
			p.nextToken() // Skip comments
		default:
			p.addError(fmt.Sprintf("unexpected token in platform config: %s", p.curToken.Type))
			p.nextToken()
		}
	}

	p.nextToken() // consume DEDENT
	return config
}

// parseStringArray parses an array of strings in YAML-like format
func (p *Parser) parseStringArray() []string {
	var result []string

	// Expect indented block
	if !p.expectPeek(lexer2.INDENT) {
		return result
	}

	p.nextToken() // move to first token inside the block

	for p.curToken.Type != lexer2.DEDENT && p.curToken.Type != lexer2.EOF {
		if p.curToken.Type == lexer2.MINUS {
			// Expect string after dash
			if !p.expectPeek(lexer2.STRING) {
				p.nextToken()
				continue
			}
			result = append(result, p.curToken.Literal)
			p.nextToken()
		} else if p.curToken.Type == lexer2.COMMENT {
			p.nextToken() // Skip comments
		} else {
			p.addError(fmt.Sprintf("expected array item (- \"value\"), got %s", p.curToken.Type))
			p.nextToken()
		}
	}

	p.nextToken() // consume DEDENT
	return result
}

// parseKeyValuePairs parses key-value pairs in YAML-like format
func (p *Parser) parseKeyValuePairs() map[string]string {
	result := make(map[string]string)

	// Expect indented block
	if !p.expectPeek(lexer2.INDENT) {
		return result
	}

	p.nextToken() // move to first token inside the block

	for p.curToken.Type != lexer2.DEDENT && p.curToken.Type != lexer2.EOF {
		if p.curToken.Type == lexer2.IDENT {
			key := p.curToken.Literal

			// Expect colon
			if !p.expectPeek(lexer2.COLON) {
				p.nextToken()
				continue
			}

			// Expect string value
			if !p.expectPeek(lexer2.STRING) {
				p.nextToken()
				continue
			}

			result[key] = p.curToken.Literal
			p.nextToken()
		} else if p.curToken.Type == lexer2.COMMENT {
			p.nextToken() // Skip comments
		} else {
			p.addError(fmt.Sprintf("expected key-value pair (key: \"value\"), got %s", p.curToken.Type))
			p.nextToken()
		}
	}

	p.nextToken() // consume DEDENT
	return result
}

// parseLifecycleHook parses before/after hooks
func (p *Parser) parseLifecycleHook() *ast.LifecycleHook {
	hook := &ast.LifecycleHook{Token: p.curToken}
	hook.Type = p.curToken.Literal // "before" or "after"

	// Expect "any"
	if !p.expectPeek(lexer2.ANY) {
		return nil
	}
	hook.Scope = p.curToken.Literal

	// Expect "task"
	if !p.expectPeek(lexer2.TASK) {
		return nil
	}

	// Expect colon
	if !p.expectPeek(lexer2.COLON) {
		return nil
	}

	// Parse hook body - expect INDENT and parse statements
	if p.peekToken.Type == lexer2.INDENT {
		p.nextToken() // consume INDENT
		p.nextToken() // move to first statement

		// Parse statements until DEDENT
		for p.curToken.Type != lexer2.DEDENT && p.curToken.Type != lexer2.EOF {
			if p.isActionToken(p.curToken.Type) {
				if p.isShellActionToken(p.curToken.Type) {
					shell := p.parseShellStatement()
					if shell != nil {
						hook.Body = append(hook.Body, shell)
					}
				} else {
					action := p.parseActionStatement()
					if action != nil {
						hook.Body = append(hook.Body, action)
					}
				}
				p.nextToken() // advance to next token after parsing
			} else if p.curToken.Type == lexer2.COMMENT {
				p.nextToken() // Skip comments
				continue
			} else {
				p.addError(fmt.Sprintf("unexpected token in lifecycle hook body: %s", p.curToken.Type))
				p.nextToken()
			}
		}

		if p.curToken.Type == lexer2.DEDENT {
			p.nextToken() // consume DEDENT for hook body
		}
	}

	return hook
}

// parseTaskStatement parses a task statement
func (p *Parser) parseTaskStatement() *ast.TaskStatement {
	stmt := &ast.TaskStatement{Token: p.curToken}

	if !p.expectPeek(lexer2.STRING) {
		return nil
	}

	stmt.Name = p.curToken.Literal

	// Check for optional "means" clause
	if p.peekToken.Type == lexer2.MEANS {
		p.nextToken() // consume lexer.MEANS

		if !p.expectPeek(lexer2.STRING) {
			return nil
		}

		stmt.Description = p.curToken.Literal
	}

	if !p.expectPeek(lexer2.COLON) {
		return nil
	}

	// Expect lexer.INDENT to start task body
	if !p.expectPeek(lexer2.INDENT) {
		return nil
	}

	// Parse task body (parameters and statements)
	for p.peekToken.Type != lexer2.DEDENT && p.peekToken.Type != lexer2.EOF {
		p.nextToken() // Move to the next token

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
			if p.curToken.Type == lexer2.RUN {
				// Look ahead to determine if this is shell or docker command
				if p.peekToken.Type == lexer2.STRING || p.peekToken.Type == lexer2.COLON {
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
			if p.curToken.Type == lexer2.CREATE {
				// Look ahead to determine if this is git or file operation
				if p.peekToken.Type == lexer2.BRANCH || p.peekToken.Type == lexer2.TAG {
					git := p.parseGitStatement()
					if git != nil {
						stmt.Body = append(stmt.Body, git)
					}
				} else if p.peekToken.Type == lexer2.DIRECTORY || p.peekToken.Type == lexer2.DIR || (p.peekToken.Type == lexer2.IDENT && p.peekToken.Literal == "file") {
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
				file := p.parseFileStatement()
				if file != nil {
					stmt.Body = append(stmt.Body, file)
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
		} else if p.curToken.Type == lexer2.COMMENT {
			// Skip comments in task body
			continue
		} else if p.curToken.Type == lexer2.NEWLINE {
			// Skip newlines in task body
			continue
		} else {
			p.addError(fmt.Sprintf("unexpected token in task body: %s (peek: %s)", p.curToken.Type, p.peekToken.Type))
			break // Stop parsing on unexpected token
		}
	}

	// Consume lexer.DEDENT
	if p.peekToken.Type == lexer2.DEDENT {
		p.nextToken() // Move to lexer.DEDENT
		p.nextToken() // Move past lexer.DEDENT
	}

	return stmt
}

// parseActionStatement parses an action statement (info, step, success, etc.)
func (p *Parser) parseActionStatement() *ast.ActionStatement {
	stmt := &ast.ActionStatement{
		Token:  p.curToken,
		Action: p.curToken.Literal,
	}

	if !p.expectPeek(lexer2.STRING) {
		return nil
	}

	stmt.Message = p.curToken.Literal

	return stmt
}

// parseShellStatement parses a shell command statement (run, exec, shell, capture)
func (p *Parser) parseShellStatement() *ast.ShellStatement {
	stmt := &ast.ShellStatement{
		Token:  p.curToken,
		Action: p.curToken.Literal,
	}

	// Check if this is multiline syntax (action followed by colon or capture with "as")
	if p.peekToken.Type == lexer2.COLON {
		return p.parseMultilineShellStatement(stmt)
	}

	// Special case for "capture as $var:" syntax
	if stmt.Action == "capture" && p.peekToken.Type == lexer2.AS {
		return p.parseMultilineShellStatement(stmt)
	}

	// Handle single-line syntax
	if stmt.Action == "capture" {
		return p.parseCaptureStatement(stmt)
	}

	// Regular shell command with string
	if !p.expectPeek(lexer2.STRING) {
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
		if !p.expectPeek(lexer2.AS) {
			return nil
		}
		if !p.expectPeekVariableName() {
			return nil
		}
		stmt.CaptureVar = p.getVariableName()
	}

	// Expect colon
	if !p.expectPeek(lexer2.COLON) {
		return nil
	}

	// Expect INDENT
	if !p.expectPeek(lexer2.INDENT) {
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
	if !p.expectPeek(lexer2.STRING) {
		return nil
	}

	stmt.Command = p.curToken.Literal

	// Check for capture syntax: capture "command" as variable_name
	if p.peekToken.Type == lexer2.AS {
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
	var currentCommand []string

	for p.curToken.Type != lexer2.DEDENT && p.curToken.Type != lexer2.EOF {
		if p.curToken.Type == lexer2.COMMENT {
			// Skip comments
			p.nextToken()
			continue
		}

		// If we encounter an IDENT token and we already have tokens in currentCommand,
		// it might be a new command, but we need to be smart about it
		if p.curToken.Type == lexer2.IDENT && len(currentCommand) > 0 {
			// Check if the previous token suggests this IDENT is part of the same command
			// For example, if previous token was ILLEGAL (like "-"), this IDENT is likely part of the same command
			prevTokenWasFlag := len(currentCommand) > 0 &&
				(strings.HasPrefix(currentCommand[len(currentCommand)-1], "-") ||
					currentCommand[len(currentCommand)-1] == "-")

			if !prevTokenWasFlag {
				// This looks like a new command, save the previous one
				commands = append(commands, strings.Join(currentCommand, " "))
				currentCommand = []string{}
			}
		}

		// Handle different token types appropriately for shell commands
		switch p.curToken.Type {
		case lexer2.STRING:
			// For STRING tokens, add quotes back to preserve shell semantics
			currentCommand = append(currentCommand, fmt.Sprintf("\"%s\"", p.curToken.Literal))
		default:
			// For other tokens, add them as-is
			currentCommand = append(currentCommand, p.curToken.Literal)
		}

		p.nextToken()
	}

	// Add the last command if there is one
	if len(currentCommand) > 0 {
		commands = append(commands, strings.Join(currentCommand, " "))
	}

	return commands
}

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
	default:
		p.addError(fmt.Sprintf("unknown file operation: %s", stmt.Action))
		return nil
	}
}

// parseCreateStatement parses "create file/dir" statements
func (p *Parser) parseCreateStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: create file "path" or create dir "path" or create directory "path"
	switch p.peekToken.Type {
	case lexer2.IDENT:
		p.nextToken() // consume IDENT
		if p.curToken.Literal == "file" {
			stmt.IsDir = false
		} else {
			p.addError("expected 'file', 'dir', or 'directory' after 'create'")
			return nil
		}
	case lexer2.DIR:
		p.nextToken() // consume DIR
		stmt.IsDir = true
	case lexer2.DIRECTORY:
		p.nextToken() // consume DIRECTORY
		stmt.IsDir = true
	default:
		p.addError("expected 'file', 'dir', or 'directory' after 'create'")
		return nil
	}

	if !p.expectPeek(lexer2.STRING) {
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

	if !p.expectPeek(lexer2.TO) {
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
	case lexer2.STRING:
		return p.curToken.Literal
	case lexer2.LBRACE:
		// Parse {$variable} syntax
		if !p.expectPeek(lexer2.VARIABLE) {
			return ""
		}
		variable := p.curToken.Literal
		if !p.expectPeek(lexer2.RBRACE) {
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
	if !p.expectPeek(lexer2.STRING) {
		return nil
	}
	stmt.Source = p.curToken.Literal

	if !p.expectPeek(lexer2.TO) {
		return nil
	}

	if !p.expectPeek(lexer2.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	return stmt
}

// parseDeleteStatement parses "delete" statements
func (p *Parser) parseDeleteStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: delete file "path" or delete dir "path"
	switch p.peekToken.Type {
	case lexer2.IDENT:
		p.nextToken() // consume IDENT
		if p.curToken.Literal == "file" {
			stmt.IsDir = false
		} else {
			p.addError("expected 'file' or 'dir' after 'delete'")
			return nil
		}
	case lexer2.DIR:
		p.nextToken() // consume DIR
		stmt.IsDir = true
	default:
		p.addError("expected 'file' or 'dir' after 'delete'")
		return nil
	}

	if !p.expectPeek(lexer2.STRING) {
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

	if !p.expectPeek(lexer2.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	// Check for capture syntax: read file "path" as variable
	if p.peekToken.Type == lexer2.AS {
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
	if !p.expectPeek(lexer2.STRING) {
		return nil
	}
	stmt.Content = p.curToken.Literal

	if !p.expectPeek(lexer2.TO) {
		return nil
	}

	if !p.expectPeekFileKeyword() {
		return nil
	}

	if !p.expectPeek(lexer2.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	return stmt
}

// parseAppendStatement parses "append" statements
func (p *Parser) parseAppendStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: append "content" to file "path"
	if !p.expectPeek(lexer2.STRING) {
		return nil
	}
	stmt.Content = p.curToken.Literal

	if !p.expectPeek(lexer2.TO) {
		return nil
	}

	if !p.expectPeekFileKeyword() {
		return nil
	}

	if !p.expectPeek(lexer2.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	return stmt
}

// parseBackupStatement parses "backup" statements
func (p *Parser) parseBackupStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: backup "source" as "backup-name"
	if !p.expectPeek(lexer2.STRING) {
		return nil
	}
	stmt.Source = p.curToken.Literal

	if p.peekToken.Type == lexer2.AS {
		p.nextToken() // consume AS
		if !p.expectPeek(lexer2.STRING) {
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
	// Expect: get size of file "path"
	switch p.peekToken.Type {
	case lexer2.IF:
		p.nextToken() // consume IF
		if !p.expectPeekFileKeyword() {
			return nil
		}
		if !p.expectPeek(lexer2.STRING) {
			return nil
		}
		stmt.Target = p.curToken.Literal

		if p.peekToken.Type == lexer2.EXISTS {
			p.nextToken() // consume EXISTS
			stmt.Action = "check_exists"
		}
	case lexer2.SIZE:
		p.nextToken() // consume SIZE
		if p.peekToken.Type == lexer2.OF {
			p.nextToken() // consume OF
			if !p.expectPeekFileKeyword() {
				return nil
			}
			if !p.expectPeek(lexer2.STRING) {
				return nil
			}
			stmt.Target = p.curToken.Literal
			stmt.Action = "get_size"
		}
	}

	return stmt
}

// parseErrorHandlingStatement parses try/catch/finally statements
func (p *Parser) parseErrorHandlingStatement() *ast.TryStatement {
	if p.curToken.Type != lexer2.TRY {
		p.addError("expected 'try' keyword")
		return nil
	}

	stmt := &ast.TryStatement{
		Token: p.curToken,
	}

	if !p.expectPeek(lexer2.COLON) {
		return nil
	}

	// Parse try body (parseControlFlowBody handles INDENT internally)
	stmt.TryBody = p.parseControlFlowBody()

	// Parse catch clauses
	for p.peekToken.Type == lexer2.CATCH {
		p.nextToken() // consume CATCH
		catchClause := p.parseCatchClause()
		if catchClause != nil {
			stmt.CatchClauses = append(stmt.CatchClauses, *catchClause)
		}
	}

	// Parse optional finally clause
	if p.peekToken.Type == lexer2.FINALLY {
		p.nextToken() // consume FINALLY
		if !p.expectPeek(lexer2.COLON) {
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
	case lexer2.IDENT:
		p.nextToken() // consume error type
		clause.ErrorType = p.curToken.Literal

		// Check for "as variable" clause
		if p.peekToken.Type == lexer2.AS {
			p.nextToken() // consume AS
			if !p.expectPeekVariableName() {
				return nil
			}
			clause.ErrorVar = p.curToken.Literal
		}
	case lexer2.AS:
		p.nextToken() // consume AS
		if !p.expectPeekVariableName() {
			return nil
		}
		clause.ErrorVar = p.curToken.Literal
	}

	if !p.expectPeek(lexer2.COLON) {
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
		if !p.expectPeek(lexer2.STRING) {
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

// parseParameterStatement parses parameter declarations (requires, given, accepts)
func (p *Parser) parseParameterStatement() *ast.ParameterStatement {
	stmt := &ast.ParameterStatement{
		Token:    p.curToken,
		Type:     p.curToken.Literal,
		DataType: "string", // default type
	}

	// Parse parameter name (expect $variable syntax)
	if p.peekToken.Type != lexer2.VARIABLE {
		p.addError(fmt.Sprintf("expected parameter name ($variable), got %s instead", p.peekToken.Type))
		return nil
	}
	p.nextToken()
	// Store parameter name without the $ prefix
	if strings.HasPrefix(p.curToken.Literal, "$") {
		stmt.Name = p.curToken.Literal[1:] // Remove the $ prefix
	} else {
		stmt.Name = p.curToken.Literal
	}

	// Check for type declaration: "as type"
	if p.peekToken.Type == lexer2.AS {
		p.nextToken() // consume AS
		if p.isTypeToken(p.peekToken.Type) {
			p.nextToken() // consume type token
			stmt.DataType = p.curToken.Literal
		} else if p.peekToken.Type == lexer2.LIST {
			p.nextToken() // consume LIST
			stmt.DataType = "list"
			if p.peekToken.Type == lexer2.OF {
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
		if p.peekToken.Type == lexer2.FROM {
			p.nextToken() // consume FROM
			if p.peekToken.Type == lexer2.LBRACKET {
				p.nextToken() // consume LBRACKET
				stmt.Constraints = p.parseStringList()
			}
		}

	case "given":
		stmt.Required = false
		// Expect: given name defaults to "value"
		if !p.expectPeek(lexer2.DEFAULTS) {
			return nil
		}
		if !p.expectPeek(lexer2.TO) {
			return nil
		}

		// Parse default value - can be string, number, boolean, or built-in function
		switch p.peekToken.Type {
		case lexer2.STRING, lexer2.NUMBER, lexer2.BOOLEAN:
			p.nextToken()
			stmt.DefaultValue = p.curToken.Literal
		case lexer2.CURRENT:
			// Handle "current git commit" built-in function
			p.nextToken() // consume CURRENT
			if p.peekToken.Type == lexer2.GIT {
				p.nextToken() // consume GIT
				if p.peekToken.Type == lexer2.COMMIT {
					p.nextToken() // consume COMMIT
					stmt.DefaultValue = "current git commit"
				}
			}
		default:
			p.addError(fmt.Sprintf("expected default value (string, number, boolean, or built-in function), got %s", p.peekToken.Type))
			return nil
		}

	case "accepts":
		stmt.Required = false
		// accepts can have constraints too
		if p.peekToken.Type == lexer2.FROM {
			p.nextToken() // consume FROM
			if p.peekToken.Type == lexer2.LBRACKET {
				p.nextToken() // consume LBRACKET
				stmt.Constraints = p.parseStringList()
			}
		}
	}

	return stmt
}

// parseDependencyStatement parses a dependency declaration
func (p *Parser) parseDependencyStatement() *ast.DependencyGroup {
	group := &ast.DependencyGroup{
		Token:        p.curToken,
		Dependencies: []ast.DependencyItem{},
		Sequential:   false, // default to parallel
	}

	// Expect "on"
	if !p.expectPeek(lexer2.ON) {
		return nil
	}

	// Parse dependency list
	for {
		// Expect task name (identifier or Docker keyword)
		switch p.peekToken.Type {
		case lexer2.IDENT:
			p.nextToken()
		case lexer2.BUILD, lexer2.PUSH, lexer2.PULL, lexer2.TAG, lexer2.REMOVE, lexer2.START, lexer2.STOP, lexer2.RUN,
			lexer2.CLONE, lexer2.INIT, lexer2.BRANCH, lexer2.SWITCH, lexer2.MERGE, lexer2.ADD, lexer2.COMMIT, lexer2.FETCH, lexer2.STATUS, lexer2.LOG, lexer2.SHOW,
			lexer2.GET, lexer2.POST, lexer2.PUT, lexer2.DELETE, lexer2.PATCH, lexer2.HEAD, lexer2.OPTIONS, lexer2.HTTP, lexer2.HTTPS:
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
		if p.peekToken.Type == lexer2.IN {
			p.nextToken() // consume IN
			if p.peekToken.Type == lexer2.PARALLEL {
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
		case lexer2.AND:
			p.nextToken() // consume AND
			group.Sequential = true
		case lexer2.COMMA:
			p.nextToken() // consume COMMA
			// Keep Sequential as false (parallel)
		case lexer2.THEN:
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

// parseDockerStatement parses Docker operations
func (p *Parser) parseDockerStatement() *ast.DockerStatement {
	stmt := &ast.DockerStatement{
		Token:   p.curToken,
		Options: make(map[string]string),
	}

	// Parse operation (build, push, pull, run, etc.)
	switch p.peekToken.Type {
	case lexer2.BUILD, lexer2.PUSH, lexer2.PULL, lexer2.TAG, lexer2.REMOVE, lexer2.START, lexer2.STOP, lexer2.RUN:
		p.nextToken()
		stmt.Operation = p.curToken.Literal
	case lexer2.COMPOSE:
		p.nextToken()
		stmt.Operation = "compose"
		stmt.Resource = "compose"

		// Parse compose command (up, down, build, etc.)
		if p.peekToken.Type == lexer2.UP || p.peekToken.Type == lexer2.DOWN || p.peekToken.Type == lexer2.BUILD {
			p.nextToken()
			stmt.Options["command"] = p.curToken.Literal
		}
		return stmt
	case lexer2.SCALE:
		// Handle "docker compose scale service "name" to 3"
		p.nextToken() // consume SCALE
		stmt.Operation = "scale"

		if p.peekToken.Type == lexer2.COMPOSE {
			p.nextToken() // consume COMPOSE
			stmt.Resource = "compose"

			if p.peekToken.Type == lexer2.IDENT && p.peekToken.Literal == "service" {
				p.nextToken() // consume IDENT (service)
				stmt.Options["resource"] = "service"

				if p.peekToken.Type == lexer2.STRING {
					p.nextToken()
					stmt.Name = p.curToken.Literal
				}

				if p.peekToken.Type == lexer2.TO {
					p.nextToken() // consume TO
					if p.peekToken.Type == lexer2.NUMBER {
						p.nextToken()
						stmt.Options["replicas"] = p.curToken.Literal
					}
				}
			}
		}
		return stmt
	case lexer2.IDENT:
		p.nextToken()
		stmt.Operation = p.curToken.Literal
	default:
		return nil
	}

	// Parse resource type (image, container)
	switch p.peekToken.Type {
	case lexer2.IMAGE, lexer2.CONTAINER:
		p.nextToken()
		stmt.Resource = p.curToken.Literal
	case lexer2.IDENT:
		p.nextToken()
		stmt.Resource = p.curToken.Literal
	default:
		return nil
	}

	// Parse name (optional for some operations)
	if p.peekToken.Type == lexer2.STRING {
		p.nextToken()
		stmt.Name = p.curToken.Literal
	}

	// Parse additional options (from, to, as, on, etc.)
	for p.peekToken.Type == lexer2.FROM || p.peekToken.Type == lexer2.TO || p.peekToken.Type == lexer2.AS || p.peekToken.Type == lexer2.ON || p.peekToken.Type == lexer2.PORT || p.peekToken.Type == lexer2.IDENT {
		p.nextToken()
		optionKey := p.curToken.Literal

		if p.peekToken.Type == lexer2.STRING || p.peekToken.Type == lexer2.NUMBER {
			p.nextToken()
			stmt.Options[optionKey] = p.curToken.Literal
		} else if optionKey == "on" && p.peekToken.Type == lexer2.PORT {
			p.nextToken() // consume PORT
			if p.peekToken.Type == lexer2.NUMBER {
				p.nextToken()
				stmt.Options["port"] = p.curToken.Literal
			}
		}
	}

	return stmt
}

// parseGitStatement parses Git operations
func (p *Parser) parseGitStatement() *ast.GitStatement {
	stmt := &ast.GitStatement{
		Token:   p.curToken,
		Options: make(map[string]string),
	}

	// Parse Git operation
	switch p.peekToken.Type {
	case lexer2.CREATE:
		// git create branch "name"
		// git create tag "v1.0.0"
		p.nextToken() // consume CREATE
		stmt.Operation = p.curToken.Literal

		switch p.peekToken.Type {
		case lexer2.BRANCH:
			p.nextToken() // consume BRANCH
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		case lexer2.TAG:
			p.nextToken() // consume TAG
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer2.CHECKOUT:
		// git checkout branch "name"
		p.nextToken() // consume CHECKOUT
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer2.BRANCH {
			p.nextToken() // consume BRANCH
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer2.MERGE:
		// git merge branch "name"
		p.nextToken() // consume MERGE
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer2.BRANCH {
			p.nextToken() // consume BRANCH
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer2.CLONE:
		// git clone repository "url" to "dir"
		p.nextToken() // consume CLONE
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer2.REPOSITORY {
			p.nextToken() // consume REPOSITORY
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer2.INIT:
		// git init repository in "dir"
		p.nextToken() // consume INIT
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer2.REPOSITORY {
			p.nextToken() // consume REPOSITORY
			stmt.Resource = p.curToken.Literal
		}

	case lexer2.ADD:
		// git add files "pattern"
		p.nextToken() // consume ADD
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer2.FILES {
			p.nextToken() // consume FILES
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer2.COMMIT:
		// git commit changes with message "msg"
		// git commit all changes with message "msg"
		p.nextToken() // consume COMMIT
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer2.ALL {
			p.nextToken() // consume ALL
			stmt.Options["all"] = "true"
		}

		if p.peekToken.Type == lexer2.CHANGES {
			p.nextToken() // consume CHANGES
			stmt.Resource = p.curToken.Literal
		}

		// Parse "with message 'text'"
		if p.peekToken.Type == lexer2.WITH {
			p.nextToken() // consume WITH
			if p.peekToken.Type == lexer2.MESSAGE {
				p.nextToken() // consume MESSAGE
				if p.peekToken.Type == lexer2.STRING {
					p.nextToken()
					stmt.Options["message"] = p.curToken.Literal
				}
			}
		}

	case lexer2.PUSH:
		// git push to remote "origin" branch "main"
		// git push tag "v1.0.0" to remote "origin"
		p.nextToken() // consume PUSH
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer2.TAG {
			p.nextToken() // consume TAG
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

		// Handle "to remote 'origin' branch 'main'" - this will be handled in options parsing

	case lexer2.PULL:
		// git pull from remote "origin" branch "main"
		p.nextToken() // consume PULL
		stmt.Operation = p.curToken.Literal

	case lexer2.FETCH:
		// git fetch from remote "origin"
		p.nextToken() // consume FETCH
		stmt.Operation = p.curToken.Literal

	case lexer2.BRANCH:
		// git create branch "name"
		// git switch to branch "name"
		// git delete branch "name"
		// git merge branch "name" into "target"
		p.nextToken() // consume BRANCH
		stmt.Resource = p.curToken.Literal

		// Look for operation before branch
		if stmt.Token.Literal == "git" {
			// This should be handled by looking at previous tokens
			// For now, assume it's a create operation
			stmt.Operation = "create"
		}

	case lexer2.STATUS:
		// git status
		p.nextToken() // consume STATUS
		stmt.Operation = p.curToken.Literal

	case lexer2.LOG:
		// git log --oneline
		p.nextToken() // consume LOG
		stmt.Operation = p.curToken.Literal

	case lexer2.SHOW:
		// git show current branch
		// git show current commit
		p.nextToken() // consume SHOW
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer2.CURRENT {
			p.nextToken() // consume CURRENT
			stmt.Options["current"] = "true"

			if p.peekToken.Type == lexer2.BRANCH || p.peekToken.Type == lexer2.COMMIT {
				p.nextToken()
				stmt.Resource = p.curToken.Literal
			}
		}

	default:
		// Handle operations that come before git (create, switch, delete, merge)
		if p.peekToken.Type == lexer2.IDENT {
			p.nextToken()
			stmt.Operation = p.curToken.Literal
		} else {
			return nil
		}
	}

	// Parse additional options (to, from, with, into, in, etc.)
	for p.peekToken.Type == lexer2.TO || p.peekToken.Type == lexer2.FROM || p.peekToken.Type == lexer2.WITH ||
		p.peekToken.Type == lexer2.INTO || p.peekToken.Type == lexer2.IN || p.peekToken.Type == lexer2.REMOTE ||
		p.peekToken.Type == lexer2.BRANCH || p.peekToken.Type == lexer2.MESSAGE || p.peekToken.Type == lexer2.IDENT {
		p.nextToken()

		switch p.curToken.Type {
		case lexer2.TO, lexer2.FROM, lexer2.WITH, lexer2.INTO, lexer2.IN:
			optionKey := p.curToken.Literal
			switch p.peekToken.Type {
			case lexer2.STRING:
				p.nextToken()
				stmt.Options[optionKey] = p.curToken.Literal
			case lexer2.REMOTE, lexer2.BRANCH, lexer2.MESSAGE:
				p.nextToken()
				keywordType := p.curToken.Literal
				if p.peekToken.Type == lexer2.STRING {
					p.nextToken()
					stmt.Options[keywordType] = p.curToken.Literal
				}
			}
		case lexer2.REMOTE, lexer2.BRANCH, lexer2.MESSAGE:
			keywordType := p.curToken.Literal
			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Options[keywordType] = p.curToken.Literal
			}
		case lexer2.IDENT:
			optionKey := p.curToken.Literal
			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Options[optionKey] = p.curToken.Literal
			}
		}
	}

	return stmt
}

// parseHTTPStatement parses HTTP operations
func (p *Parser) parseHTTPStatement() *ast.HTTPStatement {
	stmt := &ast.HTTPStatement{
		Token:   p.curToken,
		Headers: make(map[string]string),
		Auth:    make(map[string]string),
		Options: make(map[string]string),
	}

	// Determine HTTP method
	switch p.curToken.Type {
	case lexer2.GET:
		stmt.Method = "GET"
	case lexer2.POST:
		stmt.Method = "POST"
	case lexer2.PUT:
		stmt.Method = "PUT"
	case lexer2.DELETE:
		stmt.Method = "DELETE"
	case lexer2.PATCH:
		stmt.Method = "PATCH"
	case lexer2.HEAD:
		stmt.Method = "HEAD"
	case lexer2.OPTIONS:
		stmt.Method = "OPTIONS"
	case lexer2.HTTP, lexer2.HTTPS:
		// For "http request" or "https request" syntax
		if p.peekToken.Type == lexer2.REQUEST {
			p.nextToken()       // consume REQUEST
			stmt.Method = "GET" // default to GET
		} else {
			return nil
		}
	default:
		return nil
	}

	// Parse URL/endpoint
	switch p.peekToken.Type {
	case lexer2.REQUEST:
		p.nextToken() // consume REQUEST
		if p.peekToken.Type == lexer2.TO {
			p.nextToken() // consume TO
			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.URL = p.curToken.Literal
			}
		}
	case lexer2.TO, lexer2.STRING:
		if p.peekToken.Type == lexer2.TO {
			p.nextToken() // consume TO
		}
		if p.peekToken.Type == lexer2.STRING {
			p.nextToken()
			stmt.URL = p.curToken.Literal
		}
	}

	// Parse additional options (headers, body, auth, etc.)
	for p.peekToken.Type == lexer2.WITH || p.peekToken.Type == lexer2.HEADER || p.peekToken.Type == lexer2.HEADERS ||
		p.peekToken.Type == lexer2.BODY || p.peekToken.Type == lexer2.DATA || p.peekToken.Type == lexer2.AUTH ||
		p.peekToken.Type == lexer2.BEARER || p.peekToken.Type == lexer2.BASIC || p.peekToken.Type == lexer2.TOKEN ||
		p.peekToken.Type == lexer2.TIMEOUT || p.peekToken.Type == lexer2.RETRY || p.peekToken.Type == lexer2.ACCEPT ||
		p.peekToken.Type == lexer2.CONTENT || p.peekToken.Type == lexer2.TYPE {

		p.nextToken()

		switch p.curToken.Type {
		case lexer2.WITH:
			// Parse "with header", "with body", "with auth", etc.
			switch p.peekToken.Type {
			case lexer2.HEADER:
				p.nextToken() // consume HEADER
				if p.peekToken.Type == lexer2.STRING {
					p.nextToken()
					headerValue := p.curToken.Literal
					// Parse "key: value" format
					if colonIdx := strings.Index(headerValue, ":"); colonIdx != -1 {
						key := strings.TrimSpace(headerValue[:colonIdx])
						value := strings.TrimSpace(headerValue[colonIdx+1:])
						stmt.Headers[key] = value
					}
				}
			case lexer2.BODY, lexer2.DATA:
				p.nextToken() // consume BODY/DATA
				if p.peekToken.Type == lexer2.STRING {
					p.nextToken()
					stmt.Body = p.curToken.Literal
				}
			case lexer2.AUTH:
				p.nextToken() // consume AUTH
				if p.peekToken.Type == lexer2.BEARER || p.peekToken.Type == lexer2.BASIC {
					p.nextToken()
					authType := p.curToken.Literal
					if p.peekToken.Type == lexer2.STRING {
						p.nextToken()
						stmt.Auth[authType] = p.curToken.Literal
					}
				}
			case lexer2.TOKEN:
				p.nextToken() // consume TOKEN
				if p.peekToken.Type == lexer2.STRING {
					p.nextToken()
					stmt.Auth["bearer"] = p.curToken.Literal
				}
			}

		case lexer2.HEADER, lexer2.HEADERS:
			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				headerValue := p.curToken.Literal
				// Parse "key: value" format
				if colonIdx := strings.Index(headerValue, ":"); colonIdx != -1 {
					key := strings.TrimSpace(headerValue[:colonIdx])
					value := strings.TrimSpace(headerValue[colonIdx+1:])
					stmt.Headers[key] = value
				}
			}

		case lexer2.BODY, lexer2.DATA:
			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Body = p.curToken.Literal
			}

		case lexer2.AUTH:
			if p.peekToken.Type == lexer2.BEARER || p.peekToken.Type == lexer2.BASIC {
				p.nextToken()
				authType := p.curToken.Literal
				if p.peekToken.Type == lexer2.STRING {
					p.nextToken()
					stmt.Auth[authType] = p.curToken.Literal
				}
			}

		case lexer2.BEARER, lexer2.BASIC:
			authType := p.curToken.Literal
			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Auth[authType] = p.curToken.Literal
			}

		case lexer2.TOKEN:
			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Auth["bearer"] = p.curToken.Literal
			}

		case lexer2.TIMEOUT, lexer2.RETRY:
			optionKey := p.curToken.Literal
			if p.peekToken.Type == lexer2.STRING {
				p.nextToken()
				stmt.Options[optionKey] = p.curToken.Literal
			}

		case lexer2.ACCEPT:
			switch p.peekToken.Type {
			case lexer2.JSON, lexer2.XML:
				p.nextToken()
				stmt.Headers["Accept"] = "application/" + p.curToken.Literal
			case lexer2.STRING:
				p.nextToken()
				stmt.Headers["Accept"] = p.curToken.Literal
			}

		case lexer2.CONTENT:
			if p.peekToken.Type == lexer2.TYPE {
				p.nextToken() // consume TYPE
				switch p.peekToken.Type {
				case lexer2.JSON, lexer2.XML:
					p.nextToken()
					stmt.Headers["Content-Type"] = "application/" + p.curToken.Literal
				case lexer2.STRING:
					p.nextToken()
					stmt.Headers["Content-Type"] = p.curToken.Literal
				}
			}
		}
	}

	return stmt
}

// parseDetectionStatement parses smart detection operations
func (p *Parser) parseDetectionStatement() *ast.DetectionStatement {
	stmt := &ast.DetectionStatement{
		Token: p.curToken,
	}

	switch p.curToken.Type {
	case lexer2.DETECT:
		// detect project type
		// detect docker
		// detect node version
		// detect available "docker compose" or "docker-compose" as $compose_cmd
		stmt.Type = "detect"

		if p.peekToken.Type == lexer2.AVAILABLE {
			// detect available "tool1" or "tool2" as $var
			p.nextToken() // consume AVAILABLE
			stmt.Type = "detect_available"
			stmt.Condition = "available"

			// Parse first tool (required)
			if p.peekToken.Type == lexer2.STRING || p.isToolToken(p.peekToken.Type) {
				p.nextToken()
				stmt.Target = p.curToken.Literal
			} else {
				p.errors = append(p.errors, fmt.Sprintf("expected tool name after 'detect available', got %s", p.peekToken.Type))
				return stmt
			}

			// Parse alternatives (optional "or" clauses)
			for p.peekToken.Type == lexer2.OR {
				p.nextToken() // consume OR
				if p.peekToken.Type == lexer2.STRING || p.isToolToken(p.peekToken.Type) {
					p.nextToken()
					stmt.Alternatives = append(stmt.Alternatives, p.curToken.Literal)
				} else {
					p.errors = append(p.errors, fmt.Sprintf("expected tool name after 'or', got %s", p.peekToken.Type))
					return stmt
				}
			}

			// Parse capture variable (optional "as $var")
			if p.peekToken.Type == lexer2.AS {
				p.nextToken() // consume AS
				if p.peekToken.Type == lexer2.VARIABLE {
					p.nextToken()
					stmt.CaptureVar = p.getVariableName()
				} else {
					p.errors = append(p.errors, fmt.Sprintf("expected variable name after 'as', got %s", p.peekToken.Type))
					return stmt
				}
			}

		} else if p.peekToken.Type == lexer2.PROJECT {
			p.nextToken() // consume PROJECT
			stmt.Target = "project"
			if p.peekToken.Type == lexer2.TYPE {
				p.nextToken() // consume TYPE
				stmt.Condition = "type"
			}
		} else if p.isToolToken(p.peekToken.Type) {
			p.nextToken()
			stmt.Target = p.curToken.Literal

			if p.peekToken.Type == lexer2.VERSION {
				p.nextToken() // consume VERSION
				stmt.Condition = "version"
			}
		}

	case lexer2.IF:
		// if docker is available:
		// if "docker buildx" is available:
		// if node version >= "16":
		stmt.Type = "if_available"

		if p.isToolToken(p.peekToken.Type) || p.peekToken.Type == lexer2.STRING {
			p.nextToken()
			stmt.Target = p.curToken.Literal

			switch p.peekToken.Type {
			case lexer2.IS:
				p.nextToken() // consume IS
				if p.peekToken.Type == lexer2.AVAILABLE {
					p.nextToken() // consume AVAILABLE
					stmt.Condition = "available"
				}
			case lexer2.VERSION:
				p.nextToken() // consume VERSION
				stmt.Type = "if_version"

				// Parse comparison operator
				if p.peekToken.Type == lexer2.GTE || p.peekToken.Type == lexer2.GT ||
					p.peekToken.Type == lexer2.LTE || p.peekToken.Type == lexer2.LT ||
					p.peekToken.Type == lexer2.EQ || p.peekToken.Type == lexer2.NE {
					p.nextToken()
					stmt.Condition = p.curToken.Literal

					if p.peekToken.Type == lexer2.STRING {
						p.nextToken()
						stmt.Value = p.curToken.Literal
					}
				}
			}
		}

	case lexer2.WHEN:
		// when in ci environment:
		// when in production environment:
		stmt.Type = "when_environment"

		if p.peekToken.Type == lexer2.IN {
			p.nextToken() // consume IN

			if p.isEnvironmentToken(p.peekToken.Type) {
				p.nextToken()
				stmt.Target = p.curToken.Literal

				if p.peekToken.Type == lexer2.ENVIRONMENT {
					p.nextToken() // consume ENVIRONMENT
					stmt.Condition = "environment"
				}
			}
		}
	}

	// Parse body if there's a colon
	if p.peekToken.Type == lexer2.COLON {
		p.nextToken() // consume COLON
		stmt.Body = p.parseControlFlowBody()

		// Check for else clause (similar to parseIfStatement)
		if p.peekToken.Type == lexer2.ELSE {
			p.nextToken() // consume ELSE
			if !p.expectPeek(lexer2.COLON) {
				return stmt
			}
			stmt.ElseBody = p.parseControlFlowBody()
		}
	}

	return stmt
}

// isToolToken checks if a token represents a tool name
func (p *Parser) isToolToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.DOCKER, lexer2.GIT, lexer2.NODE, lexer2.NPM, lexer2.YARN, lexer2.PYTHON, lexer2.PIP,
		lexer2.GO, lexer2.GOLANG, lexer2.JAVA, lexer2.RUBY, lexer2.PHP, lexer2.RUST, lexer2.KUBECTL, lexer2.HELM,
		lexer2.TERRAFORM, lexer2.AWS, lexer2.GCP, lexer2.AZURE:
		return true
	default:
		return false
	}
}

// isEnvironmentToken checks if a token represents an environment name
func (p *Parser) isEnvironmentToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.CI, lexer2.LOCAL, lexer2.PRODUCTION, lexer2.STAGING, lexer2.DEVELOPMENT:
		return true
	default:
		return false
	}
}

// isDetectionContext checks if the current context suggests a detection statement
func (p *Parser) isDetectionContext() bool {
	switch p.curToken.Type {
	case lexer2.DETECT:
		return true
	case lexer2.IF:
		// Check if this is "if <tool> is available" or "if <tool> version ..."
		return p.isToolToken(p.peekToken.Type) || p.peekToken.Type == lexer2.STRING
	case lexer2.WHEN:
		// Check if this is "when in <environment> environment"
		return p.peekToken.Type == lexer2.IN
	default:
		return false
	}
}

// parseStringList parses a list of strings like ["dev", "staging", "production"]
func (p *Parser) parseStringList() []string {
	var items []string

	for p.peekToken.Type != lexer2.RBRACKET && p.peekToken.Type != lexer2.EOF {
		if !p.expectPeek(lexer2.STRING) {
			break
		}
		items = append(items, p.curToken.Literal)

		// Check for comma
		if p.peekToken.Type == lexer2.COMMA {
			p.nextToken() // consume comma
		}
	}

	// Consume RBRACKET
	if p.peekToken.Type == lexer2.RBRACKET {
		p.nextToken()
	}

	return items
}

// isDependencyToken checks if a token type represents a dependency declaration
func (p *Parser) isDependencyToken(tokenType lexer2.TokenType) bool {
	return tokenType == lexer2.DEPENDS
}

// isDockerToken checks if a token type represents a Docker statement
func (p *Parser) isDockerToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.DOCKER, lexer2.BUILD, lexer2.TAG, lexer2.PUSH, lexer2.PULL, lexer2.RUN, lexer2.STOP, lexer2.START, lexer2.SCALE:
		return true
	default:
		return false
	}
}

// isGitToken checks if a token type represents a Git statement
func (p *Parser) isGitToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.GIT, lexer2.CREATE, lexer2.CHECKOUT, lexer2.MERGE:
		return true
	default:
		return false
	}
}

// isHTTPToken checks if a token type represents an HTTP statement
func (p *Parser) isHTTPToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.HTTP, lexer2.HTTPS, lexer2.GET, lexer2.POST, lexer2.PUT, lexer2.DELETE, lexer2.PATCH, lexer2.HEAD, lexer2.OPTIONS:
		return true
	default:
		return false
	}
}

// isDetectionToken checks if a token type represents a detection statement
func (p *Parser) isDetectionToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.DETECT, lexer2.IF, lexer2.WHEN:
		return true
	default:
		return false
	}
}

// isParameterToken checks if a token type represents a parameter declaration
func (p *Parser) isParameterToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.REQUIRES, lexer2.GIVEN, lexer2.ACCEPTS:
		return true
	default:
		return false
	}
}

// isControlFlowToken checks if a token type represents a control flow statement
func (p *Parser) isControlFlowToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.WHEN, lexer2.IF, lexer2.FOR:
		return true
	default:
		return false
	}
}

// isActionToken checks if a token type represents an action
func (p *Parser) isActionToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.INFO, lexer2.STEP, lexer2.WARN, lexer2.ERROR, lexer2.SUCCESS, lexer2.FAIL,
		lexer2.RUN, lexer2.EXEC, lexer2.SHELL, lexer2.CAPTURE,
		lexer2.CREATE, lexer2.COPY, lexer2.MOVE, lexer2.DELETE, lexer2.READ, lexer2.WRITE, lexer2.APPEND, lexer2.BACKUP, lexer2.CHECK:
		return true
	default:
		return false
	}
}

// isShellActionToken checks if a token type represents a shell action
func (p *Parser) isShellActionToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.RUN, lexer2.EXEC, lexer2.SHELL, lexer2.CAPTURE:
		return true
	default:
		return false
	}
}

// isTypeToken checks if a token type represents a data type
func (p *Parser) isTypeToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.STRING_TYPE, lexer2.NUMBER_TYPE, lexer2.BOOLEAN_TYPE, lexer2.LIST_TYPE, lexer2.IDENT:
		return true
	default:
		return false
	}
}

// isFileActionToken checks if a token type represents a file action
func (p *Parser) isFileActionToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.COPY, lexer2.MOVE, lexer2.DELETE, lexer2.READ, lexer2.WRITE, lexer2.APPEND, lexer2.BACKUP, lexer2.CHECK:
		return true
	default:
		return false
	}
}

// isErrorHandlingToken checks if a token type represents error handling
func (p *Parser) isErrorHandlingToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.TRY:
		return true
	default:
		return false
	}
}

// isThrowActionToken checks if a token type represents a throw action
func (p *Parser) isThrowActionToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.THROW, lexer2.RETHROW, lexer2.IGNORE:
		return true
	default:
		return false
	}
}

// expectPeek checks the peek token type and advances if it matches
func (p *Parser) expectPeek(t lexer2.TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// peekError adds an error for unexpected peek token
func (p *Parser) peekError(t lexer2.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)

	// Also add to new error system if available
	if p.errorList != nil {
		p.errorList.Add(msg, p.peekToken)
	}
}

// addError adds an error message
func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)

	// Also add to new error system if available
	if p.errorList != nil {
		p.errorList.Add(msg, p.curToken)
	}
}

// skipComments skips over comment tokens
func (p *Parser) skipComments() {
	for p.curToken.Type == lexer2.COMMENT {
		p.nextToken()
	}
}

// Errors returns any parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

// ErrorList returns the enhanced error list with position information
func (p *Parser) ErrorList() *errors.ParseErrorList {
	return p.errorList
}

// parseControlFlowStatement parses control flow statements (when, if, for)
func (p *Parser) parseControlFlowStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer2.WHEN:
		return p.parseWhenStatement()
	case lexer2.IF:
		return p.parseIfStatement()
	case lexer2.FOR:
		return p.parseForStatement()
	default:
		p.addError(fmt.Sprintf("unexpected control flow token: %s", p.curToken.Type))
		return nil
	}
}

// parseWhenStatement parses when statements: when condition:
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

	if !p.expectPeek(lexer2.COLON) {
		return nil
	}

	// Parse body
	stmt.Body = p.parseControlFlowBody()

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

	if !p.expectPeek(lexer2.COLON) {
		return nil
	}

	// Parse if body
	stmt.Body = p.parseControlFlowBody()

	// Check for else clause
	if p.peekToken.Type == lexer2.ELSE {
		p.nextToken() // consume ELSE
		if !p.expectPeek(lexer2.COLON) {
			return nil
		}
		stmt.ElseBody = p.parseControlFlowBody()
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
	case lexer2.EACH:
		return p.parseForEachStatement(stmt)
	case lexer2.IDENT:
		// This could be "for i in range" or "for variable in iterable"
		return p.parseForVariableStatement(stmt)
	default:
		p.addError(fmt.Sprintf("unexpected token after for: %s", p.peekToken.Type))
		return nil
	}
}

// parseForEachStatement parses "for each" loops
func (p *Parser) parseForEachStatement(stmt *ast.LoopStatement) *ast.LoopStatement {
	stmt.Type = "each"

	if !p.expectPeek(lexer2.EACH) {
		return nil
	}

	// Check for special each types: "line" or "match"
	switch p.peekToken.Type {
	case lexer2.LINE:
		p.nextToken() // consume LINE
		stmt.Type = "line"

		if !p.expectPeek(lexer2.IDENT) {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer2.IN) {
			return nil
		}

		if !p.expectPeekFileKeyword() {
			return nil
		}

		if !p.expectPeek(lexer2.STRING) {
			return nil
		}
		stmt.Iterable = p.curToken.Literal

	case lexer2.MATCH:
		p.nextToken() // consume MATCH
		stmt.Type = "match"

		if !p.expectPeek(lexer2.IDENT) {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer2.IN) {
			return nil
		}

		if !p.expectPeek(lexer2.PATTERN) {
			return nil
		}

		if !p.expectPeek(lexer2.STRING) {
			return nil
		}
		stmt.Iterable = p.curToken.Literal

	default:
		// Regular "for each variable in iterable"
		if !p.expectPeek(lexer2.IDENT) {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer2.IN) {
			return nil
		}

		if !p.expectPeek(lexer2.IDENT) {
			return nil
		}
		stmt.Iterable = p.curToken.Literal
	}

	// Check for filter: "where variable operator value"
	if p.peekToken.Type == lexer2.WHERE {
		stmt.Filter = p.parseFilterExpression()
	}

	// Check for "in parallel"
	if p.peekToken.Type == lexer2.IN && p.peekToken.Literal == "in" {
		p.nextToken() // consume IN
		if p.peekToken.Type == lexer2.PARALLEL {
			p.nextToken() // consume PARALLEL
			stmt.Parallel = true
		}
	}

	if !p.expectPeek(lexer2.COLON) {
		return nil
	}

	// Parse body
	stmt.Body = p.parseControlFlowBody()

	return stmt
}

// parseForVariableStatement parses "for variable in range" or "for variable in iterable"
func (p *Parser) parseForVariableStatement(stmt *ast.LoopStatement) *ast.LoopStatement {
	if !p.expectPeek(lexer2.IDENT) {
		return nil
	}
	stmt.Variable = p.curToken.Literal

	if !p.expectPeek(lexer2.IN) {
		return nil
	}

	// Check if this is a range loop
	if p.peekToken.Type == lexer2.RANGE {
		p.nextToken() // consume RANGE
		stmt.Type = "range"

		// Parse range: start to end [step step_value]
		if !p.expectPeek(lexer2.NUMBER) && !p.expectPeek(lexer2.IDENT) {
			return nil
		}
		stmt.RangeStart = p.curToken.Literal

		if !p.expectPeek(lexer2.TO) {
			return nil
		}

		if !p.expectPeek(lexer2.NUMBER) && !p.expectPeek(lexer2.IDENT) {
			return nil
		}
		stmt.RangeEnd = p.curToken.Literal

		// Optional step
		if p.peekToken.Type == lexer2.STEP {
			p.nextToken() // consume STEP
			if !p.expectPeek(lexer2.NUMBER) && !p.expectPeek(lexer2.IDENT) {
				return nil
			}
			stmt.RangeStep = p.curToken.Literal
		}

	} else {
		// Regular "for variable in iterable"
		stmt.Type = "each"
		if !p.expectPeek(lexer2.IDENT) {
			return nil
		}
		stmt.Iterable = p.curToken.Literal
	}

	// Check for filter: "where variable operator value"
	if p.peekToken.Type == lexer2.WHERE {
		stmt.Filter = p.parseFilterExpression()
	}

	// Check for "in parallel"
	if p.peekToken.Type == lexer2.IN && p.peekToken.Literal == "in" {
		p.nextToken() // consume IN
		if p.peekToken.Type == lexer2.PARALLEL {
			p.nextToken() // consume PARALLEL
			stmt.Parallel = true
		}
	}

	if !p.expectPeek(lexer2.COLON) {
		return nil
	}

	// Parse body
	stmt.Body = p.parseControlFlowBody()

	return stmt
}

// parseFilterExpression parses filter conditions like "where item contains 'test'"
func (p *Parser) parseFilterExpression() *ast.FilterExpression {
	if !p.expectPeek(lexer2.WHERE) {
		return nil
	}

	filter := &ast.FilterExpression{}

	if !p.expectPeek(lexer2.IDENT) {
		return nil
	}
	filter.Variable = p.curToken.Literal

	// Parse operator
	p.nextToken()

	switch p.curToken.Type {
	case lexer2.CONTAINS:
		filter.Operator = p.curToken.Literal
	case lexer2.STARTS:
		filter.Operator = p.curToken.Literal
		// Check for "starts with"
		if p.peekToken.Type == lexer2.WITH {
			p.nextToken() // consume WITH
			filter.Operator = "starts with"
		}
	case lexer2.ENDS:
		filter.Operator = p.curToken.Literal
		// Check for "ends with"
		if p.peekToken.Type == lexer2.WITH {
			p.nextToken() // consume WITH
			filter.Operator = "ends with"
		}
	case lexer2.MATCHES:
		filter.Operator = p.curToken.Literal
	case lexer2.EQ, lexer2.NE, lexer2.GT, lexer2.GTE, lexer2.LT, lexer2.LTE:
		filter.Operator = p.curToken.Literal
	default:
		p.addError(fmt.Sprintf("unexpected filter operator: %s", p.curToken.Type))
		return nil
	}

	// Parse value
	if !p.expectPeek(lexer2.STRING) && !p.expectPeek(lexer2.IDENT) && !p.expectPeek(lexer2.NUMBER) {
		return nil
	}
	filter.Value = p.curToken.Literal

	return filter
}

// isBreakContinueToken checks if a token represents break or continue
func (p *Parser) isBreakContinueToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.BREAK, lexer2.CONTINUE:
		return true
	default:
		return false
	}
}

// parseBreakContinueStatement parses break and continue statements
func (p *Parser) parseBreakContinueStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer2.BREAK:
		return p.parseBreakStatement()
	case lexer2.CONTINUE:
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
	if p.peekToken.Type == lexer2.WHEN {
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
	if p.peekToken.Type == lexer2.IF {
		p.nextToken() // consume IF
		stmt.Condition = p.parseSimpleCondition()
	}

	return stmt
}

// isVariableOperationToken checks if a token represents variable operations
func (p *Parser) isVariableOperationToken(tokenType lexer2.TokenType) bool {
	switch tokenType {
	case lexer2.LET, lexer2.SET, lexer2.TRANSFORM:
		return true
	default:
		return false
	}
}

// parseVariableStatement parses variable operation statements
func (p *Parser) parseVariableStatement() *ast.VariableStatement {
	stmt := &ast.VariableStatement{
		Token: p.curToken,
	}

	switch p.curToken.Type {
	case lexer2.LET:
		return p.parseLetStatement(stmt)
	case lexer2.SET:
		return p.parseSetVariableStatement(stmt)
	case lexer2.TRANSFORM:
		return p.parseTransformStatement(stmt)
	default:
		p.addError(fmt.Sprintf("unexpected variable operation token: %s", p.curToken.Type))
		return nil
	}
}

// parseLetStatement parses "let variable = value" statements
func (p *Parser) parseLetStatement(stmt *ast.VariableStatement) *ast.VariableStatement {
	stmt.Operation = "let"

	if !p.expectPeekVariableName() {
		return nil
	}
	stmt.Variable = p.curToken.Literal

	if !p.expectPeek(lexer2.EQUALS) {
		return nil
	}

	// Parse the value (could be string, number, or expression)
	p.nextToken()

	stmt.Value = p.parseVariableValue()
	return stmt
}

// parseSetVariableStatement parses "set variable to value" statements
func (p *Parser) parseSetVariableStatement(stmt *ast.VariableStatement) *ast.VariableStatement {
	stmt.Operation = "set"

	if !p.expectPeekVariableName() {
		return nil
	}
	stmt.Variable = p.curToken.Literal

	if !p.expectPeek(lexer2.TO) {
		return nil
	}

	// Parse the value
	p.nextToken()

	stmt.Value = p.parseVariableValue()
	return stmt
}

// parseTransformStatement parses "transform variable with function args" statements
func (p *Parser) parseTransformStatement(stmt *ast.VariableStatement) *ast.VariableStatement {
	stmt.Operation = "transform"

	if !p.expectPeekVariableName() {
		return nil
	}
	stmt.Variable = p.curToken.Literal

	if !p.expectPeek(lexer2.WITH) {
		return nil
	}

	// Parse the function name (can be IDENT or reserved keywords like CONCAT, UPPERCASE, etc.)
	if !p.expectPeekFunctionName() {
		return nil
	}
	stmt.Function = p.curToken.Literal

	// Parse optional arguments
	for p.peekToken.Type != lexer2.NEWLINE && p.peekToken.Type != lexer2.DEDENT && p.peekToken.Type != lexer2.EOF && p.peekToken.Type != lexer2.COMMENT {
		// Check if the next token looks like an argument
		if p.peekToken.Type == lexer2.STRING || p.peekToken.Type == lexer2.IDENT || p.peekToken.Type == lexer2.NUMBER {
			p.nextToken()
			argValue := p.curToken.Literal
			// For identifiers, mark them for interpolation
			if p.curToken.Type == lexer2.IDENT {
				argValue = "{" + p.curToken.Literal + "}"
			}
			stmt.Arguments = append(stmt.Arguments, argValue)
		} else {
			// Stop parsing arguments if we hit an unexpected token
			break
		}
	}

	return stmt
}

// parseVariableValue parses variable values (strings, numbers, expressions)
func (p *Parser) parseVariableValue() string {
	switch p.curToken.Type {
	case lexer2.STRING:
		return p.curToken.Literal
	case lexer2.NUMBER:
		return p.curToken.Literal
	case lexer2.IDENT:
		// For identifiers, we need to mark them for interpolation
		return "{" + p.curToken.Literal + "}"
	default:
		// For complex expressions, just return the literal for now
		return p.curToken.Literal
	}
}

// expectPeekVariableName checks for variable names using $variable syntax
func (p *Parser) expectPeekVariableName() bool {
	if p.peekToken.Type != lexer2.VARIABLE {
		p.addError(fmt.Sprintf("expected variable name ($variable), got %s instead", p.peekToken.Type))
		return false
	}

	p.nextToken()
	return true
}

// expectPeekFileKeyword checks for the "file" keyword (as IDENT)
func (p *Parser) expectPeekFileKeyword() bool {
	if p.peekToken.Type != lexer2.IDENT || p.peekToken.Literal != "file" {
		p.addError(fmt.Sprintf("expected 'file', got %s instead", p.peekToken.Type))
		return false
	}

	p.nextToken()
	return true
}

// getVariableName returns the variable name without the $ prefix
func (p *Parser) getVariableName() string {
	if p.curToken.Type == lexer2.VARIABLE && len(p.curToken.Literal) > 1 {
		return p.curToken.Literal[1:] // Remove the $ prefix
	}
	return p.curToken.Literal
}

// expectPeekFunctionName checks for function names (can be IDENT or reserved keywords)
func (p *Parser) expectPeekFunctionName() bool {
	// Function names can be regular identifiers or reserved keywords used as function names
	validFunctionTokens := map[lexer2.TokenType]bool{
		lexer2.IDENT:     true,
		lexer2.CONCAT:    true,
		lexer2.SPLIT:     true,
		lexer2.REPLACE:   true,
		lexer2.TRIM:      true,
		lexer2.UPPERCASE: true,
		lexer2.LOWERCASE: true,
		lexer2.PREPEND:   true,
		lexer2.JOIN:      true,
		lexer2.SLICE:     true,
		lexer2.LENGTH:    true,
		lexer2.KEYS:      true,
		lexer2.VALUES:    true,
		lexer2.SUBTRACT:  true,
		lexer2.MULTIPLY:  true,
		lexer2.DIVIDE:    true,
		lexer2.MODULO:    true,
	}

	if !validFunctionTokens[p.peekToken.Type] {
		p.addError(fmt.Sprintf("expected function name, got %s instead", p.peekToken.Type))
		return false
	}
	p.nextToken()
	return true
}

// parseConditionExpression parses condition expressions like "environment is production"
func (p *Parser) parseConditionExpression() string {
	var parts []string

	// Read tokens until we hit a colon
	for p.peekToken.Type != lexer2.COLON && p.peekToken.Type != lexer2.EOF {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)
	}

	return strings.Join(parts, " ")
}

// parseSimpleCondition parses simple conditions for break/continue statements
func (p *Parser) parseSimpleCondition() string {
	var parts []string

	// Parse a simple expression: variable operator value
	// This should be something like "item == 'stop'" or "count > 10"

	// Get the variable
	if p.peekToken.Type == lexer2.IDENT {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)
	}

	// Get the operator
	if p.peekToken.Type == lexer2.EQ || p.peekToken.Type == lexer2.NE ||
		p.peekToken.Type == lexer2.GT || p.peekToken.Type == lexer2.GTE ||
		p.peekToken.Type == lexer2.LT || p.peekToken.Type == lexer2.LTE ||
		p.peekToken.Type == lexer2.CONTAINS || p.peekToken.Type == lexer2.STARTS ||
		p.peekToken.Type == lexer2.ENDS || p.peekToken.Type == lexer2.MATCHES {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)

		// Handle "starts with" and "ends with"
		if (p.curToken.Type == lexer2.STARTS || p.curToken.Type == lexer2.ENDS) && p.peekToken.Type == lexer2.WITH {
			p.nextToken() // consume WITH
			parts = append(parts, p.curToken.Literal)
		}
	}

	// Get the value
	if p.peekToken.Type == lexer2.STRING || p.peekToken.Type == lexer2.NUMBER || p.peekToken.Type == lexer2.IDENT {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)
	}

	return strings.Join(parts, " ")
}

// parseControlFlowBody parses the body of control flow statements
func (p *Parser) parseControlFlowBody() []ast.Statement {
	var body []ast.Statement

	// Expect INDENT
	if !p.expectPeek(lexer2.INDENT) {
		return body
	}

	// Parse statements until DEDENT
	for p.peekToken.Type != lexer2.DEDENT && p.peekToken.Type != lexer2.EOF {
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
			if p.curToken.Type == lexer2.RUN {
				// Look ahead to determine if this is shell or docker command
				if p.peekToken.Type == lexer2.STRING || p.peekToken.Type == lexer2.COLON {
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
				file := p.parseFileStatement()
				if file != nil {
					body = append(body, file)
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
		} else if p.curToken.Type == lexer2.COMMENT {
			// Skip comments
			continue
		} else if p.curToken.Type == lexer2.NEWLINE {
			// Skip newlines
			continue
		} else {
			p.addError(fmt.Sprintf("unexpected token in control flow body: %s", p.curToken.Type))
			break
		}
	}

	// Consume DEDENT
	if p.peekToken.Type == lexer2.DEDENT {
		p.nextToken()
	}

	return body
}
