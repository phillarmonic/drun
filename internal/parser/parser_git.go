package parser

import (
	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

// parseGitStatement parses Git operations
func (p *Parser) parseGitStatement() *ast.GitStatement {
	stmt := &ast.GitStatement{
		Token:   p.curToken,
		Options: make(map[string]string),
	}

	// Parse Git operation
	switch p.peekToken.Type {
	case lexer.CREATE:
		// git create branch "name"
		// git create tag "v1.0.0"
		p.nextToken() // consume CREATE
		stmt.Operation = p.curToken.Literal

		switch p.peekToken.Type {
		case lexer.BRANCH:
			p.nextToken() // consume BRANCH
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		case lexer.TAG:
			p.nextToken() // consume TAG
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer.CHECKOUT:
		// git checkout branch "name"
		p.nextToken() // consume CHECKOUT
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.BRANCH {
			p.nextToken() // consume BRANCH
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

	case lexer.MERGE:
		// git merge branch "name"
		p.nextToken() // consume MERGE
		stmt.Operation = p.curToken.Literal

		if p.peekToken.Type == lexer.BRANCH {
			p.nextToken() // consume BRANCH
			stmt.Resource = p.curToken.Literal

			if p.peekToken.Type == lexer.STRING {
				p.nextToken()
				stmt.Name = p.curToken.Literal
			}
		}

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
			switch p.peekToken.Type {
			case lexer.STRING:
				p.nextToken()
				stmt.Options[optionKey] = p.curToken.Literal
			case lexer.REMOTE, lexer.BRANCH, lexer.MESSAGE:
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
