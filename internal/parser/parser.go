package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/errors"
	lexer "github.com/phillarmonic/drun/internal/lexer"
)

// Parser parses drun v2 source code into an AST
type Parser struct {
	lexer *lexer.Lexer

	curToken  lexer.Token
	peekToken lexer.Token

	errors    []string // Legacy error list for backward compatibility
	errorList *errors.ParseErrorList
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

// NewParserWithSource creates a new parser instance with source information for better error reporting
func NewParserWithSource(l *lexer.Lexer, filename, source string) *Parser {
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

	// Parse task and template statements
	for p.curToken.Type != lexer.EOF {
		switch p.curToken.Type {
		case lexer.TEMPLATE:
			// Check if this is "template task"
			if p.peekToken.Type == lexer.TASK {
				template := p.parseTaskTemplateStatement()
				if template != nil {
					program.Templates = append(program.Templates, template)
				}
			} else {
				p.addError(fmt.Sprintf("unexpected token after template: %s", p.peekToken.Type))
				p.nextToken()
			}
		case lexer.TASK:
			task := p.parseTaskStatement()
			if task != nil {
				program.Tasks = append(program.Tasks, task)
			}
		case lexer.COMMENT, lexer.MULTILINE_COMMENT:
			p.nextToken() // Skip comments
		case lexer.NEWLINE:
			p.nextToken() // Skip newlines
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
