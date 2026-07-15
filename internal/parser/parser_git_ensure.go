package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/scm"
)

func (p *Parser) parseGitEnsureVersionStatement() *ast.GitEnsureVersionStatement {
	stmt := &ast.GitEnsureVersionStatement{Token: p.curToken}
	if !p.expectPeekLiteral("ensure") {
		return nil
	}
	p.nextToken()
	if p.curToken.Type != lexer.VARIABLE && p.curToken.Type != lexer.STRING {
		p.addError("git ensure candidate must be a variable or string")
		return nil
	}
	stmt.Candidate = p.curToken.Literal
	stmt.CandidateIsVariable = p.curToken.Type == lexer.VARIABLE
	for _, literal := range []string{"is", "newer", "than", "latest", "version", "from"} {
		if !p.expectPeekLiteral(literal) {
			return nil
		}
	}
	p.nextToken()
	if p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.EOF || p.curToken.Literal == "" {
		p.addError("git ensure requires a source alias")
		return nil
	}
	stmt.Source = p.curToken.Literal

	phase := 0 // using, matching tags, as
	for p.peekToken.Type != lexer.EOF {
		if p.peekToken.Type != lexer.INDENT && p.peekToken.Type != lexer.DEDENT &&
			p.peekToken.Line > p.curToken.Line && p.peekToken.Column <= stmt.Token.Column {
			return validateGitEnsureVersion(p, stmt)
		}
		p.nextToken()
		if p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT {
			continue
		}
		if p.curToken.Type == lexer.NEWLINE {
			if p.peekToken.Type == lexer.INDENT {
				p.nextToken()
				if !isGitEnsureModifier(p.peekToken.Literal) {
					return validateGitEnsureVersion(p, stmt)
				}
				continue
			}
			if p.peekToken.Type == lexer.DEDENT {
				p.nextToken()
			}
			return validateGitEnsureVersion(p, stmt)
		}
		if p.curToken.Type == lexer.INDENT {
			continue
		}
		if p.curToken.Type == lexer.DEDENT {
			return validateGitEnsureVersion(p, stmt)
		}
		switch p.curToken.Literal {
		case "using":
			if phase > 0 || stmt.AccessMethod != "" {
				p.addError("git ensure 'using' must appear once before matching tags and capture")
				return nil
			}
			p.nextToken()
			stmt.AccessMethod = p.curToken.Literal
			phase = 1
		case "matching":
			if phase > 1 || stmt.TagPreset != "" || stmt.TagFormat != "" || stmt.TagPattern != "" {
				p.addError("git ensure 'matching tags' must appear once before capture")
				return nil
			}
			if !p.expectPeekLiteral("tags") {
				return nil
			}
			if p.peekToken.Literal == "pattern" {
				p.nextToken()
				p.nextToken()
				stmt.TagPattern = p.curToken.Literal
			} else {
				p.nextToken()
				if p.curToken.Type == lexer.STRING {
					stmt.TagFormat = p.curToken.Literal
				} else {
					stmt.TagPreset = p.curToken.Literal
				}
			}
			phase = 2
		case "as":
			if phase > 2 || stmt.CaptureVar != "" {
				p.addError("git ensure capture must appear once and last")
				return nil
			}
			if !p.expectPeekVariableName() {
				return nil
			}
			stmt.CaptureVar = p.getVariableName()
			switch p.peekToken.Type {
			case lexer.NEWLINE:
				p.nextToken()
				if p.peekToken.Type == lexer.DEDENT {
					p.nextToken()
				}
			case lexer.DEDENT:
				p.nextToken()
			}
			return validateGitEnsureVersion(p, stmt)
		default:
			p.addError(fmt.Sprintf("unexpected git ensure modifier %q", p.curToken.Literal))
			return nil
		}
	}
	return validateGitEnsureVersion(p, stmt)
}

func isGitEnsureModifier(literal string) bool {
	return literal == "using" || literal == "matching" || literal == "as"
}

func validateGitEnsureVersion(p *Parser, stmt *ast.GitEnsureVersionStatement) *ast.GitEnsureVersionStatement {
	if stmt.AccessMethod != "" {
		switch stmt.AccessMethod {
		case "https", "ssh", "cli", "remote", "filesystem":
		default:
			p.addError(fmt.Sprintf("unknown git access method %q", stmt.AccessMethod))
			return nil
		}
	}
	if stmt.TagPreset != "" || stmt.TagFormat != "" || stmt.TagPattern != "" {
		formats := []string(nil)
		if stmt.TagFormat != "" {
			formats = []string{stmt.TagFormat}
		}
		if _, err := scm.NewGitVersionTagContract(stmt.TagPreset, formats, stmt.TagPattern, nil); err != nil {
			p.addError(fmt.Sprintf("invalid inline version tag contract: %v", err))
			return nil
		}
	}
	return stmt
}
