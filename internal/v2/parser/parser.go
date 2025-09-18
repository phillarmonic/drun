package parser

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/v2/ast"
	"github.com/phillarmonic/drun/internal/v2/lexer"
)

// Parser parses drun v2 source code into an AST
type Parser struct {
	lexer *lexer.Lexer

	curToken  lexer.Token
	peekToken lexer.Token

	errors []string
}

// New creates a new parser instance
func NewParser(l *lexer.Lexer) *Parser {
	p := &Parser{
		lexer:  l,
		errors: []string{},
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
	if p.curToken.Type == lexer.VERSION {
		program.Version = p.parseVersionStatement()
	} else {
		p.addError(fmt.Sprintf("expected version statement, got %s", p.curToken.Type))
		return nil
	}

	// Skip comments between version and project/tasks
	p.skipComments()

	// Parse optional project statement
	if p.curToken.Type == lexer.PROJECT {
		program.Project = p.parseProjectStatement()
		p.skipComments()
	}

	// Parse task statements
	for p.curToken.Type != lexer.EOF {
		switch p.curToken.Type {
		case lexer.TASK:
			task := p.parseTaskStatement()
			if task != nil {
				program.Tasks = append(program.Tasks, task)
			}
		case lexer.COMMENT:
			p.nextToken() // Skip comments
		case lexer.DEDENT:
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

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	if !p.expectPeek(lexer.NUMBER) {
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
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Check for optional version
	if p.peekToken.Type == lexer.VERSION {
		p.nextToken() // consume VERSION token
		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Version = p.curToken.Literal
	}

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse project settings (optional)
	if p.peekToken.Type == lexer.INDENT {
		p.nextToken() // consume INDENT
		p.nextToken() // move to first token inside the block

		for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
			switch p.curToken.Type {
			case lexer.SET:
				setting := p.parseSetStatement()
				if setting != nil {
					stmt.Settings = append(stmt.Settings, setting)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer.INCLUDE:
				setting := p.parseIncludeStatement()
				if setting != nil {
					stmt.Settings = append(stmt.Settings, setting)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer.BEFORE, lexer.AFTER:
				hook := p.parseLifecycleHook()
				if hook != nil {
					stmt.Settings = append(stmt.Settings, hook)
				} else {
					// If parsing failed, advance to avoid infinite loop
					p.nextToken()
				}
			case lexer.COMMENT:
				p.nextToken() // Skip comments
			default:
				p.addError(fmt.Sprintf("unexpected token in project body: %s", p.curToken.Type))
				p.nextToken()
			}
		}

		if p.curToken.Type == lexer.DEDENT {
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

	// Expect identifier (key) - allow Git and HTTP keywords as set keys
	switch p.peekToken.Type {
	case lexer.IDENT, lexer.MESSAGE, lexer.BRANCH, lexer.REMOTE, lexer.STATUS, lexer.LOG, lexer.COMMIT, lexer.ADD, lexer.PUSH, lexer.PULL,
		lexer.GET, lexer.POST, lexer.PUT, lexer.DELETE, lexer.PATCH, lexer.HEAD, lexer.OPTIONS, lexer.HTTP, lexer.HTTPS, lexer.URL, lexer.API, lexer.JSON, lexer.XML,
		lexer.TIMEOUT, lexer.RETRY, lexer.AUTH, lexer.BEARER, lexer.BASIC, lexer.TOKEN, lexer.HEADER, lexer.BODY, lexer.DATA:
		p.nextToken()
	default:
		p.addError(fmt.Sprintf("expected set key, got %s instead", p.peekToken.Type))
		return nil
	}
	stmt.Key = p.curToken.Literal

	// Expect "to"
	if !p.expectPeek(lexer.TO) {
		return nil
	}

	// Expect value (string)
	if !p.expectPeek(lexer.STRING) {
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
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Path = p.curToken.Literal

	p.nextToken()
	return stmt
}

// parseLifecycleHook parses before/after hooks
func (p *Parser) parseLifecycleHook() *ast.LifecycleHook {
	hook := &ast.LifecycleHook{Token: p.curToken}
	hook.Type = p.curToken.Literal // "before" or "after"

	// Expect "any"
	if !p.expectPeek(lexer.ANY) {
		return nil
	}
	hook.Scope = p.curToken.Literal

	// Expect "task"
	if !p.expectPeek(lexer.TASK) {
		return nil
	}

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse hook body - expect INDENT and parse statements
	if p.peekToken.Type == lexer.INDENT {
		p.nextToken() // consume INDENT
		p.nextToken() // move to first statement

		// Parse statements until DEDENT
		for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
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
			} else if p.curToken.Type == lexer.COMMENT {
				p.nextToken() // Skip comments
				continue
			} else {
				p.addError(fmt.Sprintf("unexpected token in lifecycle hook body: %s", p.curToken.Type))
				p.nextToken()
			}
		}

		if p.curToken.Type == lexer.DEDENT {
			p.nextToken() // consume DEDENT for hook body
		}
	}

	return hook
}

// parseTaskStatement parses a task statement
func (p *Parser) parseTaskStatement() *ast.TaskStatement {
	stmt := &ast.TaskStatement{Token: p.curToken}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}

	stmt.Name = p.curToken.Literal

	// Check for optional "means" clause
	if p.peekToken.Type == lexer.MEANS {
		p.nextToken() // consume lexer.MEANS

		if !p.expectPeek(lexer.STRING) {
			return nil
		}

		stmt.Description = p.curToken.Literal
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Expect lexer.INDENT to start task body
	if !p.expectPeek(lexer.INDENT) {
		return nil
	}

	// Parse task body (parameters and statements)
	for p.peekToken.Type != lexer.DEDENT && p.peekToken.Type != lexer.EOF {
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
			docker := p.parseDockerStatement()
			if docker != nil {
				stmt.Body = append(stmt.Body, docker)
			}
		} else if p.isGitToken(p.curToken.Type) {
			git := p.parseGitStatement()
			if git != nil {
				stmt.Body = append(stmt.Body, git)
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
		} else if p.curToken.Type == lexer.COMMENT {
			// Skip comments in task body
			continue
		} else {
			p.addError(fmt.Sprintf("unexpected token in task body: %s (peek: %s)", p.curToken.Type, p.peekToken.Type))
			break // Stop parsing on unexpected token
		}
	}

	// Consume lexer.DEDENT
	if p.peekToken.Type == lexer.DEDENT {
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

	if !p.expectPeek(lexer.STRING) {
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

	if !p.expectPeek(lexer.STRING) {
		return nil
	}

	stmt.Command = p.curToken.Literal

	// Check for capture syntax: capture "command" as variable_name
	if stmt.Action == "capture" && p.peekToken.Type == lexer.AS {
		p.nextToken() // consume AS
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.CaptureVar = p.curToken.Literal
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
	default:
		p.addError(fmt.Sprintf("unknown file operation: %s", stmt.Action))
		return nil
	}
}

// parseCreateStatement parses "create file/dir" statements
func (p *Parser) parseCreateStatement(stmt *ast.FileStatement) *ast.FileStatement {
	// Expect: create file "path" or create dir "path"
	switch p.peekToken.Type {
	case lexer.FILE:
		p.nextToken() // consume FILE
		stmt.IsDir = false
	case lexer.DIR:
		p.nextToken() // consume DIR
		stmt.IsDir = true
	default:
		p.addError("expected 'file' or 'dir' after 'create'")
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
	// Expect: copy "source" to "target"
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
	case lexer.DIR:
		p.nextToken() // consume DIR
		stmt.IsDir = true
	default:
		p.addError("expected 'file' or 'dir' after 'delete'")
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
	if !p.expectPeek(lexer.FILE) {
		return nil
	}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	// Check for capture syntax: read file "path" as variable
	if p.peekToken.Type == lexer.AS {
		p.nextToken() // consume AS
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.CaptureVar = p.curToken.Literal
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

	if !p.expectPeek(lexer.FILE) {
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

	if !p.expectPeek(lexer.FILE) {
		return nil
	}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Target = p.curToken.Literal

	return stmt
}

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
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			clause.ErrorVar = p.curToken.Literal
		}
	case lexer.AS:
		p.nextToken() // consume AS
		if !p.expectPeek(lexer.IDENT) {
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

// parseParameterStatement parses parameter declarations (requires, given, accepts)
func (p *Parser) parseParameterStatement() *ast.ParameterStatement {
	stmt := &ast.ParameterStatement{
		Token:    p.curToken,
		Type:     p.curToken.Literal,
		DataType: "string", // default type
	}

	// Parse parameter name (allow Git and HTTP keywords as parameter names)
	switch p.peekToken.Type {
	case lexer.IDENT, lexer.MESSAGE, lexer.BRANCH, lexer.REMOTE, lexer.STATUS, lexer.LOG, lexer.COMMIT, lexer.ADD, lexer.PUSH, lexer.PULL,
		lexer.GET, lexer.POST, lexer.PUT, lexer.DELETE, lexer.PATCH, lexer.HEAD, lexer.OPTIONS, lexer.HTTP, lexer.HTTPS, lexer.URL, lexer.API, lexer.JSON, lexer.XML,
		lexer.TIMEOUT, lexer.RETRY, lexer.AUTH, lexer.BEARER, lexer.BASIC, lexer.TOKEN, lexer.HEADER, lexer.BODY, lexer.DATA:
		p.nextToken()
	default:
		p.addError(fmt.Sprintf("expected parameter name, got %s instead", p.peekToken.Type))
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Check for type declaration: "as type"
	if p.peekToken.Type == lexer.AS {
		p.nextToken() // consume AS
		if p.isTypeToken(p.peekToken.Type) {
			p.nextToken() // consume type token
			stmt.DataType = p.curToken.Literal
		} else if p.peekToken.Type == lexer.LIST {
			p.nextToken() // consume LIST
			stmt.DataType = "list"
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

	case "given":
		stmt.Required = false
		// Expect: given name defaults to "value"
		if !p.expectPeek(lexer.DEFAULTS) {
			return nil
		}
		if !p.expectPeek(lexer.TO) {
			return nil
		}
		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.DefaultValue = p.curToken.Literal

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

// parseDependencyStatement parses a dependency declaration
func (p *Parser) parseDependencyStatement() *ast.DependencyGroup {
	group := &ast.DependencyGroup{
		Token:        p.curToken,
		Dependencies: []ast.DependencyItem{},
		Sequential:   false, // default to parallel
	}

	// Expect "on"
	if !p.expectPeek(lexer.ON) {
		return nil
	}

	// Parse dependency list
	for {
		// Expect task name (identifier or Docker keyword)
		switch p.peekToken.Type {
		case lexer.IDENT:
			p.nextToken()
		case lexer.BUILD, lexer.PUSH, lexer.PULL, lexer.TAG, lexer.REMOVE, lexer.START, lexer.STOP, lexer.RUN,
			lexer.CLONE, lexer.INIT, lexer.BRANCH, lexer.SWITCH, lexer.MERGE, lexer.ADD, lexer.COMMIT, lexer.FETCH, lexer.STATUS, lexer.LOG, lexer.SHOW,
			lexer.GET, lexer.POST, lexer.PUT, lexer.DELETE, lexer.PATCH, lexer.HEAD, lexer.OPTIONS, lexer.HTTP, lexer.HTTPS:
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
		if p.peekToken.Type == lexer.IN {
			p.nextToken() // consume IN
			if p.peekToken.Type == lexer.PARALLEL {
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
		case lexer.AND:
			p.nextToken() // consume AND
			group.Sequential = true
		case lexer.COMMA:
			p.nextToken() // consume COMMA
			// Keep Sequential as false (parallel)
		case lexer.THEN:
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
	case lexer.BUILD, lexer.PUSH, lexer.PULL, lexer.TAG, lexer.REMOVE, lexer.START, lexer.STOP, lexer.RUN:
		p.nextToken()
		stmt.Operation = p.curToken.Literal
	case lexer.COMPOSE:
		p.nextToken()
		stmt.Operation = "compose"
		stmt.Resource = "compose"

		// Parse compose command (up, down, build, etc.)
		if p.peekToken.Type == lexer.UP || p.peekToken.Type == lexer.DOWN || p.peekToken.Type == lexer.BUILD {
			p.nextToken()
			stmt.Options["command"] = p.curToken.Literal
		}
		return stmt
	case lexer.IDENT:
		p.nextToken()
		stmt.Operation = p.curToken.Literal
	default:
		return nil
	}

	// Parse resource type (image, container)
	switch p.peekToken.Type {
	case lexer.IMAGE, lexer.CONTAINER:
		p.nextToken()
		stmt.Resource = p.curToken.Literal
	case lexer.IDENT:
		p.nextToken()
		stmt.Resource = p.curToken.Literal
	default:
		return nil
	}

	// Parse name (optional for some operations)
	if p.peekToken.Type == lexer.STRING {
		p.nextToken()
		stmt.Name = p.curToken.Literal
	}

	// Parse additional options (from, to, as, etc.)
	for p.peekToken.Type == lexer.FROM || p.peekToken.Type == lexer.TO || p.peekToken.Type == lexer.AS || p.peekToken.Type == lexer.IDENT {
		p.nextToken()
		optionKey := p.curToken.Literal

		if p.peekToken.Type == lexer.STRING {
			p.nextToken()
			stmt.Options[optionKey] = p.curToken.Literal
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
	case lexer.CLONE:
		// git clone repository "url" to "dir"
		p.nextToken() // consume CLONE
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.REPOSITORY {
			p.nextToken() // consume REPOSITORY
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer.INIT:
		// git init repository in "dir"
		p.nextToken() // consume INIT
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.REPOSITORY {
			p.nextToken() // consume REPOSITORY
			stmt.Resource = p.curToken.Literal
		}

	case lexer.ADD:
		// git add files "pattern"
		p.nextToken() // consume ADD
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.FILES {
			p.nextToken() // consume FILES
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer.COMMIT:
		// git commit changes with message "msg"
		// git commit all changes with message "msg"
		p.nextToken() // consume COMMIT
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.ALL {
			p.nextToken() // consume ALL
			stmt.Options["all"] = "true"
		}

		if p.peekToken.Type == lexer.CHANGES {
			p.nextToken() // consume CHANGES
			stmt.Resource = p.curToken.Literal
		}

		// Parse "with message 'text'"
		if p.peekToken.Type == lexer.WITH {
			p.nextToken() // consume WITH
			if p.peekToken.Type == lexer.MESSAGE {
				p.nextToken() // consume MESSAGE
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Options["message"] = p.curToken.Literal
				}
			}
		}

	case lexer.PUSH:
		// git push to remote "origin" branch "main"
		// git push tag "v1.0.0" to remote "origin"
		p.nextToken() // consume PUSH
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.TAG {
			p.nextToken() // consume TAG
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

		// Handle "to remote 'origin' branch 'main'" - this will be handled in options parsing

	case lexer.PULL:
		// git pull from remote "origin" branch "main"
		p.nextToken() // consume PULL
		stmt.Operation = p.curToken.Literal

	case lexer.FETCH:
		// git fetch from remote "origin"
		p.nextToken() // consume FETCH
		stmt.Operation = p.curToken.Literal

	case lexer.BRANCH:
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

	case lexer.STATUS:
		// git status
		p.nextToken() // consume STATUS
		stmt.Operation = p.curToken.Literal

	case lexer.LOG:
		// git log --oneline
		p.nextToken() // consume LOG
		stmt.Operation = p.curToken.Literal

	case lexer.SHOW:
		// git show current branch
		// git show current commit
		p.nextToken() // consume SHOW
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.CURRENT {
			p.nextToken() // consume CURRENT
			stmt.Options["current"] = "true"

			if p.peekToken.Type == lexer.BRANCH || p.peekToken.Type == lexer.COMMIT {
				p.nextToken()
				stmt.Resource = p.curToken.Literal
			}
		}

	default:
		// Handle operations that come before git (create, switch, delete, merge)
		if p.peekToken.Type == lexer.IDENT {
			p.nextToken()
			stmt.Operation = p.curToken.Literal
		} else {
			return nil
		}
	}

	// Parse additional options (to, from, with, into, in, etc.)
	for p.peekToken.Type == lexer.TO || p.peekToken.Type == lexer.FROM || p.peekToken.Type == lexer.WITH ||
		p.peekToken.Type == lexer.INTO || p.peekToken.Type == lexer.IN || p.peekToken.Type == lexer.REMOTE ||
		p.peekToken.Type == lexer.BRANCH || p.peekToken.Type == lexer.MESSAGE || p.peekToken.Type == lexer.IDENT {
		p.nextToken()

		switch p.curToken.Type {
		case lexer.TO, lexer.FROM, lexer.WITH, lexer.INTO, lexer.IN:
			optionKey := p.curToken.Literal
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Options[optionKey] = p.curToken.Literal
			} else if p.peekToken.Type == lexer.REMOTE || p.peekToken.Type == lexer.BRANCH || p.peekToken.Type == lexer.MESSAGE {
				p.nextToken()
				keywordType := p.curToken.Literal
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Options[keywordType] = p.curToken.Literal
				}
			}
		case lexer.REMOTE, lexer.BRANCH, lexer.MESSAGE:
			keywordType := p.curToken.Literal
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Options[keywordType] = p.curToken.Literal
			}
		case lexer.IDENT:
			optionKey := p.curToken.Literal
			if p.peekToken.Type == lexer.STRING {
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
	case lexer.GET:
		stmt.Method = "GET"
	case lexer.POST:
		stmt.Method = "POST"
	case lexer.PUT:
		stmt.Method = "PUT"
	case lexer.DELETE:
		stmt.Method = "DELETE"
	case lexer.PATCH:
		stmt.Method = "PATCH"
	case lexer.HEAD:
		stmt.Method = "HEAD"
	case lexer.OPTIONS:
		stmt.Method = "OPTIONS"
	case lexer.HTTP, lexer.HTTPS:
		// For "http request" or "https request" syntax
		if p.peekToken.Type == lexer.REQUEST {
			p.nextToken()       // consume REQUEST
			stmt.Method = "GET" // default to GET
		} else {
			return nil
		}
	default:
		return nil
	}

	// Parse URL/endpoint
	if p.peekToken.Type == lexer.REQUEST {
		p.nextToken() // consume REQUEST
		if p.peekToken.Type == lexer.TO {
			p.nextToken() // consume TO
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.URL = p.curToken.Literal
			}
		}
	} else if p.peekToken.Type == lexer.TO || p.peekToken.Type == lexer.STRING {
		if p.peekToken.Type == lexer.TO {
			p.nextToken() // consume TO
		}
		if p.peekToken.Type == lexer.STRING {
			p.nextToken()
			stmt.URL = p.curToken.Literal
		}
	}

	// Parse additional options (headers, body, auth, etc.)
	for p.peekToken.Type == lexer.WITH || p.peekToken.Type == lexer.HEADER || p.peekToken.Type == lexer.HEADERS ||
		p.peekToken.Type == lexer.BODY || p.peekToken.Type == lexer.DATA || p.peekToken.Type == lexer.AUTH ||
		p.peekToken.Type == lexer.BEARER || p.peekToken.Type == lexer.BASIC || p.peekToken.Type == lexer.TOKEN ||
		p.peekToken.Type == lexer.TIMEOUT || p.peekToken.Type == lexer.RETRY || p.peekToken.Type == lexer.ACCEPT ||
		p.peekToken.Type == lexer.CONTENT || p.peekToken.Type == lexer.TYPE {

		p.nextToken()

		switch p.curToken.Type {
		case lexer.WITH:
			// Parse "with header", "with body", "with auth", etc.
			if p.peekToken.Type == lexer.HEADER {
				p.nextToken() // consume HEADER
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					headerValue := p.curToken.Literal
					// Parse "key: value" format
					if colonIdx := strings.Index(headerValue, ":"); colonIdx != -1 {
						key := strings.TrimSpace(headerValue[:colonIdx])
						value := strings.TrimSpace(headerValue[colonIdx+1:])
						stmt.Headers[key] = value
					}
				}
			} else if p.peekToken.Type == lexer.BODY || p.peekToken.Type == lexer.DATA {
				p.nextToken() // consume BODY/DATA
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Body = p.curToken.Literal
				}
			} else if p.peekToken.Type == lexer.AUTH {
				p.nextToken() // consume AUTH
				if p.peekToken.Type == lexer.BEARER || p.peekToken.Type == lexer.BASIC {
					p.nextToken()
					authType := p.curToken.Literal
					if p.peekToken.Type == lexer.STRING {
						p.nextToken()
						stmt.Auth[authType] = p.curToken.Literal
					}
				}
			} else if p.peekToken.Type == lexer.TOKEN {
				p.nextToken() // consume TOKEN
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Auth["bearer"] = p.curToken.Literal
				}
			}

		case lexer.HEADER, lexer.HEADERS:
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				headerValue := p.curToken.Literal
				// Parse "key: value" format
				if colonIdx := strings.Index(headerValue, ":"); colonIdx != -1 {
					key := strings.TrimSpace(headerValue[:colonIdx])
					value := strings.TrimSpace(headerValue[colonIdx+1:])
					stmt.Headers[key] = value
				}
			}

		case lexer.BODY, lexer.DATA:
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Body = p.curToken.Literal
			}

		case lexer.AUTH:
			if p.peekToken.Type == lexer.BEARER || p.peekToken.Type == lexer.BASIC {
				p.nextToken()
				authType := p.curToken.Literal
				if p.peekToken.Type == lexer.STRING {
					p.nextToken()
					stmt.Auth[authType] = p.curToken.Literal
				}
			}

		case lexer.BEARER, lexer.BASIC:
			authType := p.curToken.Literal
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Auth[authType] = p.curToken.Literal
			}

		case lexer.TOKEN:
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Auth["bearer"] = p.curToken.Literal
			}

		case lexer.TIMEOUT, lexer.RETRY:
			optionKey := p.curToken.Literal
			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Options[optionKey] = p.curToken.Literal
			}

		case lexer.ACCEPT:
			if p.peekToken.Type == lexer.JSON || p.peekToken.Type == lexer.XML {
				p.nextToken()
				stmt.Headers["Accept"] = "application/" + p.curToken.Literal
			} else if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Headers["Accept"] = p.curToken.Literal
			}

		case lexer.CONTENT:
			if p.peekToken.Type == lexer.TYPE {
				p.nextToken() // consume TYPE
				if p.peekToken.Type == lexer.JSON || p.peekToken.Type == lexer.XML {
					p.nextToken()
					stmt.Headers["Content-Type"] = "application/" + p.curToken.Literal
				} else if p.peekToken.Type == lexer.STRING {
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
	case lexer.DETECT:
		// detect project type
		// detect docker
		// detect node version
		stmt.Type = "detect"

		if p.peekToken.Type == lexer.PROJECT {
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
		// if node version >= "16":
		stmt.Type = "if_available"

		if p.isToolToken(p.peekToken.Type) {
			p.nextToken()
			stmt.Target = p.curToken.Literal

			if p.peekToken.Type == lexer.IS {
				p.nextToken() // consume IS
				if p.peekToken.Type == lexer.AVAILABLE {
					p.nextToken() // consume AVAILABLE
					stmt.Condition = "available"
				}
			} else if p.peekToken.Type == lexer.VERSION {
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
	case lexer.DOCKER, lexer.GIT, lexer.NODE, lexer.NPM, lexer.YARN, lexer.PYTHON, lexer.PIP,
		lexer.GO, lexer.GOLANG, lexer.JAVA, lexer.RUBY, lexer.PHP, lexer.RUST, lexer.KUBECTL, lexer.HELM,
		lexer.TERRAFORM, lexer.AWS, lexer.GCP, lexer.AZURE:
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
		return p.isToolToken(p.peekToken.Type)
	case lexer.WHEN:
		// Check if this is "when in <environment> environment"
		return p.peekToken.Type == lexer.IN
	default:
		return false
	}
}

// parseStringList parses a list of strings like ["dev", "staging", "production"]
func (p *Parser) parseStringList() []string {
	var items []string

	for p.peekToken.Type != lexer.RBRACKET && p.peekToken.Type != lexer.EOF {
		if !p.expectPeek(lexer.STRING) {
			break
		}
		items = append(items, p.curToken.Literal)

		// Check for comma
		if p.peekToken.Type == lexer.COMMA {
			p.nextToken() // consume comma
		}
	}

	// Consume RBRACKET
	if p.peekToken.Type == lexer.RBRACKET {
		p.nextToken()
	}

	return items
}

// isDependencyToken checks if a token type represents a dependency declaration
func (p *Parser) isDependencyToken(tokenType lexer.TokenType) bool {
	return tokenType == lexer.DEPENDS
}

// isDockerToken checks if a token type represents a Docker statement
func (p *Parser) isDockerToken(tokenType lexer.TokenType) bool {
	return tokenType == lexer.DOCKER
}

// isGitToken checks if a token type represents a Git statement
func (p *Parser) isGitToken(tokenType lexer.TokenType) bool {
	return tokenType == lexer.GIT
}

// isHTTPToken checks if a token type represents an HTTP statement
func (p *Parser) isHTTPToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.HTTP, lexer.HTTPS, lexer.GET, lexer.POST, lexer.PUT, lexer.DELETE, lexer.PATCH, lexer.HEAD, lexer.OPTIONS:
		return true
	default:
		return false
	}
}

// isDetectionToken checks if a token type represents a detection statement
func (p *Parser) isDetectionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.DETECT, lexer.IF, lexer.WHEN:
		return true
	default:
		return false
	}
}

// isParameterToken checks if a token type represents a parameter declaration
func (p *Parser) isParameterToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.REQUIRES, lexer.GIVEN, lexer.ACCEPTS:
		return true
	default:
		return false
	}
}

// isControlFlowToken checks if a token type represents a control flow statement
func (p *Parser) isControlFlowToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.WHEN, lexer.IF, lexer.FOR:
		return true
	default:
		return false
	}
}

// isActionToken checks if a token type represents an action
func (p *Parser) isActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.INFO, lexer.STEP, lexer.WARN, lexer.ERROR, lexer.SUCCESS, lexer.FAIL,
		lexer.RUN, lexer.EXEC, lexer.SHELL, lexer.CAPTURE,
		lexer.CREATE, lexer.COPY, lexer.MOVE, lexer.DELETE, lexer.READ, lexer.WRITE, lexer.APPEND:
		return true
	default:
		return false
	}
}

// isShellActionToken checks if a token type represents a shell action
func (p *Parser) isShellActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.RUN, lexer.EXEC, lexer.SHELL, lexer.CAPTURE:
		return true
	default:
		return false
	}
}

// isTypeToken checks if a token type represents a data type
func (p *Parser) isTypeToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN_TYPE, lexer.LIST_TYPE:
		return true
	default:
		return false
	}
}

// isFileActionToken checks if a token type represents a file action
func (p *Parser) isFileActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.CREATE, lexer.COPY, lexer.MOVE, lexer.DELETE, lexer.READ, lexer.WRITE, lexer.APPEND:
		return true
	default:
		return false
	}
}

// isErrorHandlingToken checks if a token type represents error handling
func (p *Parser) isErrorHandlingToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TRY:
		return true
	default:
		return false
	}
}

// isThrowActionToken checks if a token type represents a throw action
func (p *Parser) isThrowActionToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.THROW, lexer.RETHROW, lexer.IGNORE:
		return true
	default:
		return false
	}
}

// expectPeek checks the peek token type and advances if it matches
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// peekError adds an error for unexpected peek token
func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

// addError adds an error message
func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)
}

// skipComments skips over comment tokens
func (p *Parser) skipComments() {
	for p.curToken.Type == lexer.COMMENT {
		p.nextToken()
	}
}

// Errors returns any parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

// parseControlFlowStatement parses control flow statements (when, if, for)
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

	if !p.expectPeek(lexer.COLON) {
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

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse if body
	stmt.Body = p.parseControlFlowBody()

	// Check for else clause
	if p.peekToken.Type == lexer.ELSE {
		p.nextToken() // consume ELSE
		if !p.expectPeek(lexer.COLON) {
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
	case lexer.EACH:
		return p.parseForEachStatement(stmt)
	case lexer.IDENT:
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

	if !p.expectPeek(lexer.EACH) {
		return nil
	}

	// Check for special each types: "line" or "match"
	if p.peekToken.Type == lexer.LINE {
		p.nextToken() // consume LINE
		stmt.Type = "line"

		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer.IN) {
			return nil
		}

		if !p.expectPeek(lexer.FILE) {
			return nil
		}

		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		stmt.Iterable = p.curToken.Literal

	} else if p.peekToken.Type == lexer.MATCH {
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

	} else {
		// Regular "for each variable in iterable"
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Variable = p.curToken.Literal

		if !p.expectPeek(lexer.IN) {
			return nil
		}

		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Iterable = p.curToken.Literal
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

// parseForVariableStatement parses "for variable in range" or "for variable in iterable"
func (p *Parser) parseForVariableStatement(stmt *ast.LoopStatement) *ast.LoopStatement {
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Variable = p.curToken.Literal

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
		// Regular "for variable in iterable"
		stmt.Type = "each"
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Iterable = p.curToken.Literal
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

	if !p.expectPeek(lexer.IDENT) {
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

// parseConditionExpression parses condition expressions like "environment is production"
func (p *Parser) parseConditionExpression() string {
	var parts []string

	// Read tokens until we hit a colon
	for p.peekToken.Type != lexer.COLON && p.peekToken.Type != lexer.EOF {
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
	if p.peekToken.Type == lexer.IDENT {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)
	}

	// Get the operator
	if p.peekToken.Type == lexer.EQ || p.peekToken.Type == lexer.NE ||
		p.peekToken.Type == lexer.GT || p.peekToken.Type == lexer.GTE ||
		p.peekToken.Type == lexer.LT || p.peekToken.Type == lexer.LTE ||
		p.peekToken.Type == lexer.CONTAINS || p.peekToken.Type == lexer.STARTS ||
		p.peekToken.Type == lexer.ENDS || p.peekToken.Type == lexer.MATCHES {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)

		// Handle "starts with" and "ends with"
		if (p.curToken.Type == lexer.STARTS || p.curToken.Type == lexer.ENDS) && p.peekToken.Type == lexer.WITH {
			p.nextToken() // consume WITH
			parts = append(parts, p.curToken.Literal)
		}
	}

	// Get the value
	if p.peekToken.Type == lexer.STRING || p.peekToken.Type == lexer.NUMBER || p.peekToken.Type == lexer.IDENT {
		p.nextToken()
		parts = append(parts, p.curToken.Literal)
	}

	return strings.Join(parts, " ")
}

// parseControlFlowBody parses the body of control flow statements
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
			docker := p.parseDockerStatement()
			if docker != nil {
				body = append(body, docker)
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
		} else if p.curToken.Type == lexer.COMMENT {
			// Skip comments
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
