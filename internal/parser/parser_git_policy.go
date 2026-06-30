package parser

import (
	"fmt"
	"strconv"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// parseGitPolicyStatement parses a "git policy:" block inside a project.
// The current token is GIT when this is called and peek is POLICY.
//
// Syntax:
//
//	git policy:
//	    default branches:
//	        master
//	        develop
//	    branch naming:
//	        pattern "{type}/{identifier}-{description}"
//	        types: feat, fix, hotfix, chore
//	    commit messages:
//	        pattern "{identifier}: {message}"
//	        extract identifier from branch
//	        min length 10
//	        ban "WIP"
//	    enforce signed commits
func (p *Parser) parseGitPolicyStatement() *ast.GitPolicyStatement {
	stmt := &ast.GitPolicyStatement{Token: p.curToken}

	// Current token is GIT, peek should be POLICY
	if !p.expectPeek(lexer.POLICY) {
		return nil
	}

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Expect indented block
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}

	// Move to first token inside the block
	p.nextToken()

	// Parse git policy properties until DEDENT
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		switch p.curToken.Type {
		case lexer.NEWLINE, lexer.COMMENT, lexer.MULTILINE_COMMENT:
			p.nextToken()
			continue

		case lexer.BRANCH:
			// "branch:"
			if !p.expectPeek(lexer.COLON) {
				p.nextToken()
				continue
			}

			// Expect indented block
			if !p.expectPeekSkipNewlines(lexer.INDENT) {
				p.nextToken()
				continue
			}
			p.nextToken() // move to first token in block

			for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
				switch p.curToken.Type {
				case lexer.NEWLINE, lexer.COMMENT, lexer.MULTILINE_COMMENT:
					p.nextToken()
					continue

				case lexer.DEFAULT_KW:
					// "default branches:"
					if p.peekToken.Type == lexer.BRANCHES {
						p.nextToken() // consume BRANCHES
						if !p.expectPeek(lexer.COLON) {
							p.nextToken()
							continue
						}
						stmt.DefaultBranches = append(stmt.DefaultBranches, p.parseCommaSeparatedStrings()...)
					} else {
						p.addError(fmt.Sprintf("expected 'branches' after 'default', got %s", p.peekToken.Type))
						p.nextToken()
					}

				case lexer.NAMING:
					if !p.expectPeek(lexer.COLON) {
						p.nextToken()
						continue
					}
					if p.expectPeek(lexer.STRING) {
						stmt.BranchPattern = p.curToken.Literal
						p.nextToken() // move past the string
					}

				case lexer.TYPES_KW:
					if !p.expectPeek(lexer.COLON) {
						p.nextToken()
						continue
					}
					stmt.BranchTypes = p.parseCommaSeparatedStrings()

				default:
					p.addError(fmt.Sprintf("unexpected token in branch block: %s (%q)", p.curToken.Type, p.curToken.Literal))
					p.nextToken()
				}
			}

			// Move past DEDENT of the branch block
			if p.curToken.Type == lexer.DEDENT {
				p.nextToken()
			}

		case lexer.COMMIT:
			// "commit:"
			if !p.expectPeek(lexer.COLON) {
				p.nextToken()
				continue
			}

			// Expect indented block
			if !p.expectPeekSkipNewlines(lexer.INDENT) {
				p.nextToken()
				continue
			}
			p.nextToken() // move to first token in block

			for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
				switch p.curToken.Type {
				case lexer.NEWLINE, lexer.COMMENT, lexer.MULTILINE_COMMENT:
					p.nextToken()
					continue

				case lexer.MESSAGES:
					if !p.expectPeek(lexer.COLON) {
						p.nextToken()
						continue
					}
					if p.expectPeek(lexer.STRING) {
						stmt.CommitPattern = p.curToken.Literal
						p.nextToken() // move past the string
					}

				case lexer.BAN:
					if !p.expectPeek(lexer.COLON) {
						p.nextToken()
						continue
					}
					stmt.CommitBans = p.parseCommaSeparatedStrings()

				case lexer.MIN:
					if p.peekToken.Type == lexer.LENGTH {
						p.nextToken() // consume LENGTH
						if !p.expectPeek(lexer.COLON) {
							p.nextToken()
							continue
						}
						if p.expectPeek(lexer.NUMBER) {
							n, err := strconv.Atoi(p.curToken.Literal)
							if err == nil {
								stmt.CommitMinLength = n
							} else {
								p.addError(fmt.Sprintf("invalid min length value: %q", p.curToken.Literal))
							}
							p.nextToken() // move past the number
						}
					} else {
						p.addError(fmt.Sprintf("expected 'length' after 'min', got %s", p.peekToken.Type))
						p.nextToken()
					}

				case lexer.EXTRACT:
					p.nextToken() // consume EXTRACT
					if p.curToken.Type == lexer.IDENT && p.curToken.Literal == "identifier" {
						p.nextToken()
					}
					if p.curToken.Type == lexer.FROM {
						p.nextToken()
					}
					if p.curToken.Type == lexer.BRANCH {
						p.nextToken()
					}
					stmt.ExtractIdentifier = true

				case lexer.ENFORCE:
					if p.peekToken.Type == lexer.SIGNED {
						p.nextToken() // consume SIGNED
						if p.peekToken.Type == lexer.COMMITS {
							p.nextToken() // consume COMMITS
						}
						stmt.EnforceSignedCommits = true
						p.nextToken() // advance past the clause
					} else {
						p.addError(fmt.Sprintf("expected 'signed' after 'enforce', got %s", p.peekToken.Type))
						p.nextToken()
					}

				default:
					p.addError(fmt.Sprintf("unexpected token in commit block: %s (%q)", p.curToken.Type, p.curToken.Literal))
					p.nextToken()
				}
			}

			// Move past DEDENT of the commit block
			if p.curToken.Type == lexer.DEDENT {
				p.nextToken()
			}

		default:
			p.addError(fmt.Sprintf("unexpected token in git policy body: %s (%q)", p.curToken.Type, p.curToken.Literal))
			p.nextToken()
		}
	}

	// Do not advance past DEDENT — the project parser handles that
	return stmt
}

// parseCommaSeparatedStrings parses a comma-separated list of strings on the same line.
// Example: "master", "develop"
func (p *Parser) parseCommaSeparatedStrings() []string {
	var items []string

	// Move past COLON
	p.nextToken()

	for p.curToken.Type == lexer.STRING || p.curToken.Type == lexer.COMMA {
		if p.curToken.Type == lexer.COMMA {
			p.nextToken()
			continue
		}

		if p.curToken.Type == lexer.STRING {
			items = append(items, p.curToken.Literal)
		}
		p.nextToken()
	}

	return items
}

// parseGitValidateStatement parses a "git validate ..." statement.
// The current token is GIT, peek is VALIDATE.
//
// Syntax:
//
//	git validate branch name
//	git validate commit message "explicit message"
//	git validate signed commits
//	git validate all
func (p *Parser) parseGitValidateStatement() *ast.GitValidateStatement {
	stmt := &ast.GitValidateStatement{Token: p.curToken}

	// Current token is GIT, peek should be VALIDATE
	p.nextToken() // consume VALIDATE

	// Determine the validation target
	switch p.peekToken.Type {
	case lexer.BRANCH:
		// git validate branch name
		p.nextToken() // consume BRANCH
		if p.peekToken.Type == lexer.NAME_KW {
			p.nextToken() // consume NAME
		}
		stmt.Target = "branch_name"

	case lexer.COMMIT:
		// git validate commit message "..."
		p.nextToken() // consume COMMIT
		if p.peekToken.Type == lexer.MESSAGE {
			p.nextToken() // consume MESSAGE
		}
		stmt.Target = "commit_message"
		// Optional explicit value
		if p.peekToken.Type == lexer.STRING {
			p.nextToken()
			stmt.Value = p.curToken.Literal
		}

	case lexer.SIGNED:
		// git validate signed commits
		p.nextToken() // consume SIGNED
		if p.peekToken.Type == lexer.COMMITS {
			p.nextToken() // consume COMMITS
		}
		stmt.Target = "signed_commits"

	case lexer.ALL:
		// git validate all
		p.nextToken() // consume ALL
		stmt.Target = "all"

	default:
		p.addError(fmt.Sprintf("expected branch, commit, signed, or all after 'git validate', got %s", p.peekToken.Type))
		return nil
	}

	return stmt
}
