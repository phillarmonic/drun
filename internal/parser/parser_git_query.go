package parser

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/scm"
)

func (p *Parser) parseGitQueryStatement() *ast.GitQueryStatement {
	stmt := &ast.GitQueryStatement{Token: p.curToken, OrderBy: "version"}
	if !p.expectPeekLiteral("get") || !p.expectPeekLiteral("latest") {
		return nil
	}
	p.nextToken()
	if p.curToken.Literal != "tag" && p.curToken.Literal != "version" {
		p.addError("git get latest expects tag or version")
		return nil
	}
	stmt.Result = p.curToken.Literal
	if !p.expectPeekLiteral("from") {
		return nil
	}
	p.nextToken()
	stmt.Source = p.curToken.Literal

	for p.peekToken.Type != lexer.EOF {
		p.nextToken()
		if p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.INDENT || p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT {
			continue
		}
		if p.curToken.Type == lexer.DEDENT {
			break
		}
		switch p.curToken.Literal {
		case "using":
			p.nextToken()
			stmt.AccessMethod = p.curToken.Literal
		case "matching":
			p.nextToken()
			switch p.curToken.Literal {
			case "tags":
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
			case "version":
				p.nextToken()
				stmt.VersionMatcher = p.curToken.Literal
			default:
				p.addError("matching expects tags or version")
				return nil
			}
		case "in":
			if !p.expectPeekLiteral("series") {
				return nil
			}
			p.nextToken()
			stmt.Series = p.curToken.Literal
		case "ordered":
			if !p.expectPeekLiteral("by") {
				return nil
			}
			p.nextToken()
			stmt.OrderBy = p.curToken.Literal
		case "allow":
			if !p.expectPeekLiteral("fetch") {
				return nil
			}
			stmt.AllowFetch = true
		case "as":
			if !p.expectPeekVariableName() {
				return nil
			}
			stmt.CaptureVar = p.getVariableName()
			if err := validateGitQuery(stmt); err != nil {
				p.addError(err.Error())
				return nil
			}
			return stmt
		default:
			p.addError(fmt.Sprintf("unexpected git query modifier %q", p.curToken.Literal))
			return nil
		}
	}
	p.addError("git query must end with 'as $variable'")
	return nil
}

func validateGitQuery(stmt *ast.GitQueryStatement) error {
	if stmt.Source == "" {
		return fmt.Errorf("git query requires a source alias")
	}
	if stmt.AccessMethod != "" {
		switch stmt.AccessMethod {
		case "https", "ssh", "cli", "remote", "filesystem":
		default:
			return fmt.Errorf("unknown git access method %q", stmt.AccessMethod)
		}
	}
	tagForms := 0
	if stmt.TagPreset != "" {
		tagForms++
	}
	if stmt.TagFormat != "" {
		tagForms++
	}
	if stmt.TagPattern != "" {
		tagForms++
	}
	if tagForms > 1 {
		return fmt.Errorf("matching tags preset, format, and pattern forms are mutually exclusive")
	}
	if tagForms > 0 {
		formats := []string(nil)
		if stmt.TagFormat != "" {
			formats = []string{stmt.TagFormat}
		}
		if _, err := scm.NewGitVersionTagContract(stmt.TagPreset, formats, stmt.TagPattern, nil); err != nil {
			return fmt.Errorf("invalid inline version tag contract: %w", err)
		}
	}
	if stmt.Series != "" && stmt.VersionMatcher != "" {
		return fmt.Errorf("in series and matching version are mutually exclusive")
	}
	if stmt.Series != "" {
		if _, err := scm.SeriesConstraint(stmt.Series); err != nil {
			return err
		}
	}
	if stmt.VersionMatcher != "" {
		if _, err := scm.ParseVersionConstraint(stmt.VersionMatcher); err != nil {
			return err
		}
	}
	if stmt.OrderBy != "version" && stmt.OrderBy != "date" {
		return fmt.Errorf("git query ordering must be version or date, got %q", stmt.OrderBy)
	}
	return nil
}

func (p *Parser) expectPeekLiteral(literal string) bool {
	if p.peekToken.Literal == literal {
		p.nextToken()
		return true
	}
	p.addError(fmt.Sprintf("expected %q, got %q", literal, p.peekToken.Literal))
	return false
}
