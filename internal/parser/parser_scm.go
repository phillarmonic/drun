package parser

import (
	"fmt"
	"net/url"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/scm"
)

func (p *Parser) parseSCMRegistryStatement() *ast.SCMRegistryStatement {
	stmt := &ast.SCMRegistryStatement{Token: p.curToken, Technologies: map[string]*ast.SCMTechnology{}}
	if !p.expectPeek(lexer.COLON) || !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}
	p.nextToken()
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		if p.isSCMTrivia() {
			p.nextToken()
			continue
		}
		name := p.curToken.Literal
		if name != "git" {
			p.addError(fmt.Sprintf("unsupported SCM technology %q (currently supported: git)", name))
			return nil
		}
		tech := p.parseSCMTechnology(name)
		if tech == nil {
			return nil
		}
		stmt.Technologies[name] = tech
		if p.curToken.Type == lexer.DEDENT {
			p.nextToken()
		}
	}
	return stmt
}

func (p *Parser) parseSCMTechnology(name string) *ast.SCMTechnology {
	tech := &ast.SCMTechnology{Name: name, Providers: map[string]*ast.SCMProvider{}}
	if !p.expectPeek(lexer.COLON) || !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}
	p.nextToken()
	aliases := map[string]string{}
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		if p.isSCMTrivia() {
			p.nextToken()
			continue
		}
		providerName := p.curToken.Literal
		if providerName != "github" && providerName != "gitlab" && providerName != "generic" {
			p.addError(fmt.Sprintf("unsupported git provider %q (supported: github, gitlab, generic)", providerName))
			return nil
		}
		provider := p.parseSCMProvider(providerName, aliases)
		if provider == nil {
			return nil
		}
		tech.Providers[providerName] = provider
		if p.curToken.Type == lexer.DEDENT {
			p.nextToken()
		}
	}
	return tech
}

func (p *Parser) parseSCMProvider(name string, aliases map[string]string) *ast.SCMProvider {
	provider := &ast.SCMProvider{Name: name, Sources: map[string]*ast.SCMSource{}}
	if !p.expectPeek(lexer.COLON) || !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}
	p.nextToken()
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		if p.isSCMTrivia() {
			p.nextToken()
			continue
		}
		alias := p.curToken.Literal
		if previous, exists := aliases[alias]; exists {
			p.addError(fmt.Sprintf("duplicate git SCM alias %q (already declared under %s)", alias, previous))
			return nil
		}
		source := p.parseSCMSource(name, alias)
		if source == nil {
			return nil
		}
		aliases[alias] = name
		provider.Sources[alias] = source
		if p.curToken.Type == lexer.DEDENT {
			p.nextToken()
		}
	}
	return provider
}

func (p *Parser) parseSCMSource(provider, alias string) *ast.SCMSource {
	source := &ast.SCMSource{Alias: alias, Provider: provider, Access: map[string]*ast.SCMAccessProfile{}}
	if !p.expectPeek(lexer.COLON) || !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}
	p.nextToken()
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		if p.isSCMTrivia() {
			p.nextToken()
			continue
		}
		field := p.curToken.Literal
		if field == "version" && p.peekToken.Literal == "tags" {
			contract := p.parseVersionTagContract()
			if contract == nil {
				return nil
			}
			source.VersionTags = contract
			continue
		}
		switch field {
		case "default", "metadata":
			if !p.expectPeek(lexer.COLON) {
				return nil
			}
			p.nextToken()
			if field == "default" {
				source.Default = p.curToken.Literal
			} else {
				source.Metadata = p.curToken.Literal
			}
			p.nextToken()
		case "https", "ssh", "cli", "remote", "filesystem":
			profile := p.parseSCMAccessProfile(field)
			if profile == nil {
				return nil
			}
			source.Access[field] = profile
		default:
			p.addError(fmt.Sprintf("unknown SCM source property %q", field))
			return nil
		}
	}
	if err := validateSCMSource(source); err != nil {
		p.addError(err.Error())
		return nil
	}
	return source
}

func (p *Parser) parseSCMAccessProfile(method string) *ast.SCMAccessProfile {
	profile := &ast.SCMAccessProfile{Method: method}
	if !p.expectPeek(lexer.COLON) {
		return nil
	}
	if p.peekToken.Type == lexer.STRING {
		p.nextToken()
		value := p.curToken.Literal
		setSCMProfileValue(profile, method, value)
		p.nextToken()
		return profile
	}
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}
	p.nextToken()
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		if p.isSCMTrivia() {
			p.nextToken()
			continue
		}
		key := p.curToken.Literal
		if !p.expectPeek(lexer.COLON) {
			return nil
		}
		p.nextToken()
		value := p.curToken.Literal
		switch key {
		case "url":
			profile.URL = value
		case "repository":
			profile.Repository = value
		case "host":
			profile.Host = value
		case "authentication":
			profile.Authentication = value
		case "key":
			profile.Key = value
		case "path":
			profile.Path = value
		default:
			p.addError(fmt.Sprintf("unknown %s SCM access property %q", method, key))
			return nil
		}
		p.nextToken()
	}
	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}
	return profile
}

func (p *Parser) parseVersionTagContract() *ast.VersionTagContract {
	contract := &ast.VersionTagContract{}
	p.nextToken() // tags
	if !p.expectPeek(lexer.COLON) {
		return nil
	}
	if p.peekToken.Type == lexer.STRING {
		p.nextToken()
		contract.Formats = []string{p.curToken.Literal}
		p.nextToken()
		return contract
	}
	if p.peekToken.Type != lexer.NEWLINE && p.peekToken.Type != lexer.INDENT {
		p.nextToken()
		contract.Preset = p.curToken.Literal
		p.nextToken()
		return contract
	}
	if !p.expectPeekSkipNewlines(lexer.INDENT) {
		return nil
	}
	p.nextToken()
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		if p.isSCMTrivia() {
			p.nextToken()
			continue
		}
		field := p.curToken.Literal
		if !p.expectPeek(lexer.COLON) {
			return nil
		}
		if field == "formats" {
			if !p.expectPeekSkipNewlines(lexer.INDENT) {
				return nil
			}
			p.nextToken()
			for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
				if p.curToken.Type == lexer.STRING {
					contract.Formats = append(contract.Formats, p.curToken.Literal)
				}
				p.nextToken()
			}
			if p.curToken.Type == lexer.DEDENT {
				p.nextToken()
			}
			continue
		}
		p.nextToken()
		switch field {
		case "format":
			contract.Formats = []string{p.curToken.Literal}
		case "pattern":
			contract.Pattern = p.curToken.Literal
		default:
			p.addError(fmt.Sprintf("unknown version tags property %q", field))
			return nil
		}
		p.nextToken()
	}
	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}
	return contract
}

func setSCMProfileValue(profile *ast.SCMAccessProfile, method, value string) {
	switch method {
	case "cli":
		profile.Repository = value
	case "filesystem":
		profile.Path = value
	default:
		profile.URL = value
	}
}

func validateSCMSource(source *ast.SCMSource) error {
	if len(source.Access) == 0 {
		return fmt.Errorf("SCM source %q must declare at least one access method", source.Alias)
	}
	allowed := map[string]map[string]bool{
		"github": {"https": true, "ssh": true, "cli": true}, "gitlab": {"https": true, "ssh": true, "cli": true},
		"generic": {"https": true, "ssh": true, "remote": true, "filesystem": true},
	}
	for method := range source.Access {
		if !allowed[source.Provider][method] {
			return fmt.Errorf("%s SCM source %q does not support %s access", source.Provider, source.Alias, method)
		}
		if err := validateSCMAccessProfile(source.Alias, source.Access[method]); err != nil {
			return err
		}
	}
	if len(source.Access) > 1 && source.Default == "" {
		return fmt.Errorf("SCM source %q declares multiple access methods and requires default", source.Alias)
	}
	if source.Default == "" {
		for method := range source.Access {
			source.Default = method
		}
	}
	if _, ok := source.Access[source.Default]; !ok {
		return fmt.Errorf("SCM source %q default %q is not declared", source.Alias, source.Default)
	}
	if source.Metadata != "" && source.Metadata != "refs" && source.Metadata != "fetch" {
		return fmt.Errorf("SCM source %q metadata must be refs or fetch", source.Alias)
	}
	if source.VersionTags != nil {
		if _, err := scm.NewGitVersionTagContract(
			source.VersionTags.Preset, source.VersionTags.Formats, source.VersionTags.Pattern, nil,
		); err != nil {
			return fmt.Errorf("SCM source %q: %w", source.Alias, err)
		}
	}
	return nil
}

func validateSCMAccessProfile(alias string, profile *ast.SCMAccessProfile) error {
	switch profile.Method {
	case "https", "remote":
		if profile.URL == "" {
			return fmt.Errorf("SCM source %q %s access requires a URL", alias, profile.Method)
		}
		if profile.Key != "" || profile.Repository != "" || profile.Path != "" {
			return fmt.Errorf("SCM source %q %s access contains properties for another access method", alias, profile.Method)
		}
		if profile.Method == "https" {
			parsed, err := url.Parse(profile.URL)
			if err != nil || parsed.Scheme != "https" {
				return fmt.Errorf("SCM source %q https access requires an https URL", alias)
			}
			if parsed.User != nil {
				return fmt.Errorf("SCM source %q HTTPS credentials must remain ambient and cannot appear in the URL", alias)
			}
		}
	case "ssh":
		if profile.URL == "" {
			return fmt.Errorf("SCM source %q ssh access requires a URL", alias)
		}
		if profile.Repository != "" || profile.Path != "" || profile.Authentication != "" {
			return fmt.Errorf("SCM source %q ssh access contains properties for another access method", alias)
		}
	case "cli":
		if profile.Repository == "" {
			return fmt.Errorf("SCM source %q cli access requires repository", alias)
		}
		if profile.URL != "" || profile.Key != "" || profile.Path != "" || profile.Authentication != "" {
			return fmt.Errorf("SCM source %q cli access contains properties for another access method", alias)
		}
	case "filesystem":
		if profile.Path == "" {
			return fmt.Errorf("SCM source %q filesystem access requires a path", alias)
		}
		if profile.URL != "" || profile.Key != "" || profile.Repository != "" || profile.Authentication != "" || profile.Host != "" {
			return fmt.Errorf("SCM source %q filesystem access contains properties for another access method", alias)
		}
	}
	if profile.Authentication != "" && profile.Authentication != "ambient" {
		return fmt.Errorf("SCM source %q authentication must be ambient; credentials cannot be declared", alias)
	}
	return nil
}

func (p *Parser) isSCMTrivia() bool {
	return p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.COMMENT || p.curToken.Type == lexer.MULTILINE_COMMENT
}
