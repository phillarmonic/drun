package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

// parseServiceStatement parses a service declaration
// service "service_name" in "path/to/service" means "Description":
func (p *Parser) parseServiceStatement() *ast.ServiceStatement {
	stmt := &ast.ServiceStatement{
		Token: p.curToken,
	}

	// Expect "service"
	if !p.expectPeek(lexer.STRING) {
		p.addError("expected service name string")
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Expect "in"
	if !p.expectPeek(lexer.IN) {
		p.addError("expected 'in' keyword after service name")
		return nil
	}

	// Expect path string
	if !p.expectPeek(lexer.STRING) {
		p.addError("expected service path string")
		return nil
	}
	stmt.Path = p.curToken.Literal

	// Optional "means" clause
	if p.peekToken.Type == lexer.MEANS {
		p.nextToken()
		if !p.expectPeek(lexer.STRING) {
			p.addError("expected description string after 'means'")
			return nil
		}
		stmt.Description = p.curToken.Literal
	}

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' after service declaration")
		return nil
	}

	// Expect indent (optional for empty services)
	switch p.peekToken.Type {
	case lexer.INDENT:
		p.nextToken() // consume INDENT
	case lexer.EOF:
		// Empty service body - return immediately
		return stmt
	default:
		p.addError("expected indent after service declaration")
		return nil
	}

	stmt.Environment = make(map[string]string)

serviceBody:
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		// Skip newlines
		for p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}

		switch p.curToken.Type {
		case lexer.DEPENDS:
			// Parse dependencies
			stmt.Dependencies = p.parseServiceDependencies()
		case lexer.REPOSITORY:
			// Parse repository config
			stmt.Repository = p.parseRepositoryConfig()
		case lexer.HEALTH:
			// Parse health check config
			stmt.HealthCheck = p.parseHealthCheckConfig()
		case lexer.BUILD:
			// Parse build config
			stmt.Build = p.parseBuildConfig()
		case lexer.COMPOSE:
			// Parse compose config - handle both "compose:" and "compose file" syntax
			if p.peekToken.Type == lexer.FILE {
				// Handle "compose file" syntax
				p.nextToken() // consume "file"
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected compose file string")
					return nil
				}
				stmt.Compose = &ast.ComposeConfig{
					File: p.curToken.Literal,
				}
				p.nextToken()
			} else {
				// Handle "compose:" syntax
				stmt.Compose = p.parseComposeConfig()
			}
		case lexer.ENVIRONMENT:
			// Parse environment variables
			stmt.Environment = p.parseEnvironmentMap()
		case lexer.ENV_FILE:
			// Parse env_file config
			stmt.EnvFile = p.parseEnvFileConfig()
		case lexer.DOCKER:
			// Parse docker networks config
			if p.peekToken.Type == lexer.NETWORKS {
				p.nextToken() // consume "networks"
				stmt.Networks = p.parseDockerNetworksConfig()
			} else {
				p.addError("expected 'networks' after 'docker'")
			}
		case lexer.PRE:
			// Parse pre_task
			if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "task" {
				p.nextToken() // consume "task"
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected task name string after 'pre'")
				return nil
			}
			stmt.PreTask = p.curToken.Literal
			p.nextToken()
		case lexer.IDENT:
			// Handle "post_task" using IDENT
			if p.curToken.Literal == "post_task" || p.curToken.Literal == "post" {
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected task name string after 'post'")
					return nil
				}
				stmt.PostTask = p.curToken.Literal
				p.nextToken()
			} else {
				p.nextToken()
			}
		case lexer.INDENT:
			// Skip INDENT tokens in service body
			p.nextToken()
		case lexer.DEDENT:
			// End of service body
			break serviceBody
		default:
			p.addError(fmt.Sprintf("unexpected token in service body: %s", p.curToken.Type))
			p.nextToken()
		}
	}

	// Consume the DEDENT
	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return stmt
}

// parseServiceDependencies parses service dependencies
// depends on ["service1", "service2"]
func (p *Parser) parseServiceDependencies() []string {
	if !p.expectPeek(lexer.ON) {
		p.addError("expected 'on' after 'depends'")
		return nil
	}

	if !p.expectPeek(lexer.LBRACKET) {
		p.addError("expected '[' for dependencies list")
		return nil
	}

	dependencies := []string{}
	p.nextToken()

	for p.curToken.Type != lexer.RBRACKET && p.curToken.Type != lexer.EOF {
		if p.curToken.Type == lexer.STRING {
			dependencies = append(dependencies, p.curToken.Literal)
		} else {
			p.addError(fmt.Sprintf("expected string in dependencies list, got %s", p.curToken.Type))
		}

		p.nextToken()

		if p.curToken.Type == lexer.COMMA {
			p.nextToken()
		}
	}

	if p.curToken.Type != lexer.RBRACKET {
		p.addError("expected ']' to close dependencies list")
		return nil
	}

	p.nextToken()
	return dependencies
}

// parseRepositoryConfig parses repository configuration
func (p *Parser) parseRepositoryConfig() *ast.RepositoryConfig {
	config := &ast.RepositoryConfig{
		Clone: true, // default to true
	}

	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' after 'repository'")
		return nil
	}

	if !p.expectPeek(lexer.INDENT) {
		p.addError("expected indent after 'repository:'")
		return nil
	}

repoBody:
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		for p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}

		switch p.curToken.Type {
		case lexer.URL:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected URL string")
				return nil
			}
			config.URL = p.curToken.Literal
			p.nextToken()
		case lexer.BRANCH:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected branch string")
				return nil
			}
			config.Branch = p.curToken.Literal
			p.nextToken()
		case lexer.TAG:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected tag string")
				return nil
			}
			config.Tag = p.curToken.Literal
			p.nextToken()
		case lexer.SSH:
			if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "key" {
				p.nextToken() // consume 'key'
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected SSH key path string")
				return nil
			}
			config.SSHKey = p.curToken.Literal
			p.nextToken()
		case lexer.CLONE:
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for clone")
				return nil
			}
			config.Clone = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.UPDATE:
			if p.peekToken.Type == lexer.ON {
				p.nextToken() // consume 'on'
				if p.peekToken.Type == lexer.START {
					p.nextToken() // consume 'start'
				}
			}
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for update_on_start")
				return nil
			}
			config.UpdateOnStart = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.DEDENT:
			break repoBody
		default:
			p.nextToken()
		}
	}

	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return config
}

// parseHealthCheckConfig parses health check configuration
func (p *Parser) parseHealthCheckConfig() *ast.HealthCheckConfig {
	config := &ast.HealthCheckConfig{}

	// Expect "health check:"
	if !p.expectPeek(lexer.CHECK) {
		p.addError("expected 'check' after 'health'")
		return nil
	}

	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' after 'health check'")
		return nil
	}

	if !p.expectPeek(lexer.INDENT) {
		p.addError("expected indent after 'health check:'")
		return nil
	}

	config.Headers = make(map[string]string)

healthBody:
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		for p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}

		switch p.curToken.Type {
		case lexer.TYPE:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected type string")
				return nil
			}
			config.Type = p.curToken.Literal
			p.nextToken()
		case lexer.ENDPOINT:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected endpoint string")
				return nil
			}
			config.Endpoint = p.curToken.Literal
			p.nextToken()
		case lexer.DOMAIN:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected domain string")
				return nil
			}
			config.Domain = p.curToken.Literal
			p.nextToken()
		case lexer.CONTAINER:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected container string")
				return nil
			}
			config.Container = p.curToken.Literal
			p.nextToken()
		case lexer.COMMAND:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected command string")
				return nil
			}
			config.Command = p.curToken.Literal
			p.nextToken()
		case lexer.TIMEOUT:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected timeout string")
				return nil
			}
			config.Timeout = p.curToken.Literal
			p.nextToken()
		case lexer.INTERVAL:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected interval string")
				return nil
			}
			config.Interval = p.curToken.Literal
			p.nextToken()
		case lexer.RETRIES:
			if !p.expectPeek(lexer.NUMBER) {
				p.addError("expected number for retries")
				return nil
			}
			retries, _ := strconv.Atoi(p.curToken.Literal)
			config.Retries = retries
			p.nextToken()
		case lexer.CONDITION:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected condition string")
				return nil
			}
			config.Condition = p.curToken.Literal
			p.nextToken()
		case lexer.RECORD:
			if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "type" {
				p.nextToken() // consume 'type'
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected record type string")
				return nil
			}
			config.RecordType = p.curToken.Literal
			p.nextToken()
		case lexer.EXPECTED:
			p.nextToken()
			if p.curToken.Type == lexer.IP || (p.curToken.Type == lexer.IDENT && p.curToken.Literal == "ip") {
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected expected_ip string")
					return nil
				}
				config.ExpectedIP = p.curToken.Literal
				p.nextToken()
			}
		case lexer.HEADERS:
			config.Headers = p.parseHeadersMap()
		case lexer.WORKING:
			if p.peekToken.Type == lexer.DIRECTORY {
				p.nextToken() // consume 'directory'
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected working directory string")
				return nil
			}
			config.WorkingDir = p.curToken.Literal
			p.nextToken()
		case lexer.START:
			if p.peekToken.Type == lexer.PERIOD {
				p.nextToken() // consume 'period'
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected start period string")
				return nil
			}
			config.StartPeriod = p.curToken.Literal
			p.nextToken()
		case lexer.DEDENT:
			break healthBody
		default:
			p.nextToken()
		}
	}

	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return config
}

// parseBuildConfig parses build configuration
func (p *Parser) parseBuildConfig() *ast.BuildConfig {
	config := &ast.BuildConfig{}

	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' after 'build'")
		return nil
	}

	if !p.expectPeek(lexer.INDENT) {
		p.addError("expected indent after 'build:'")
		return nil
	}

buildBody:
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		for p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}

		switch p.curToken.Type {
		case lexer.REQUIRED:
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for required")
				return nil
			}
			config.Required = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.COMMAND:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected command string")
				return nil
			}
			config.Command = p.curToken.Literal
			p.nextToken()
		case lexer.MAKEFILE:
			p.nextToken()
			// Could be: "makefile path" or "makefile_timeout duration"
			if p.curToken.Type == lexer.STRING {
				// This is the makefile path
				config.Makefile = p.curToken.Literal
				p.nextToken()
			} else if p.curToken.Type == lexer.IDENT && p.curToken.Literal == "timeout" {
				// This is makefile_timeout
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected makefile timeout string")
					return nil
				}
				config.MakefileTimeout = p.curToken.Literal
				p.nextToken()
			}
		case lexer.MAKE:
			p.nextToken()
			if p.curToken.Type == lexer.TARGET || (p.curToken.Type == lexer.IDENT && p.curToken.Literal == "target") {
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected make target string")
					return nil
				}
				config.MakeTarget = p.curToken.Literal
				p.nextToken()
			} else if p.curToken.Type == lexer.ARGS || (p.curToken.Type == lexer.IDENT && p.curToken.Literal == "args") {
				config.MakeArgs = p.parseOrchestrationStringArray()
			}
		case lexer.WORKING:
			if p.peekToken.Type == lexer.DIRECTORY {
				p.nextToken() // consume 'directory'
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected working directory string")
				return nil
			}
			config.WorkingDirectory = p.curToken.Literal
			p.nextToken()
		case lexer.PARALLEL:
			if p.peekToken.Type == lexer.JOBS {
				p.nextToken() // consume 'jobs'
			}
			if !p.expectPeek(lexer.NUMBER) {
				p.addError("expected number for parallel jobs")
				return nil
			}
			jobs, _ := strconv.Atoi(p.curToken.Literal)
			config.ParallelJobs = jobs
			p.nextToken()
		case lexer.VERBOSE:
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for verbose")
				return nil
			}
			config.Verbose = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.ALLOCATE_TTY:
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for allocate_tty")
				return nil
			}
			config.AllocateTTY = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.RETRY:
			if p.peekToken.Type == lexer.ON {
				p.nextToken() // consume 'on'
				if p.peekToken.Type == lexer.FAILURE {
					p.nextToken() // consume 'failure'
				}
			}
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for retry_on_failure")
				return nil
			}
			config.RetryOnFailure = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.MAX:
			if p.peekToken.Type == lexer.RETRIES {
				p.nextToken() // consume 'retries'
			}
			if !p.expectPeek(lexer.NUMBER) {
				p.addError("expected number for max_retries")
				return nil
			}
			retries, _ := strconv.Atoi(p.curToken.Literal)
			config.MaxRetries = retries
			p.nextToken()
		case lexer.FALLBACK:
			if p.peekToken.Type == lexer.COMMAND {
				p.nextToken() // consume 'command'
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected fallback command string")
				return nil
			}
			config.FallbackCommand = p.curToken.Literal
			p.nextToken()
		case lexer.DEDENT:
			break buildBody
		default:
			p.nextToken()
		}
	}

	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return config
}

// parseComposeConfig parses compose configuration
func (p *Parser) parseComposeConfig() *ast.ComposeConfig {
	config := &ast.ComposeConfig{}

	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' after 'compose'")
		return nil
	}

	if !p.expectPeek(lexer.INDENT) {
		p.addError("expected indent after 'compose:'")
		return nil
	}

composeBody:
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		for p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}

		switch p.curToken.Type {
		case lexer.FILE:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected file string")
				return nil
			}
			config.File = p.curToken.Literal
			p.nextToken()
		case lexer.PROJECT:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected project string")
				return nil
			}
			config.Project = p.curToken.Literal
			p.nextToken()
		case lexer.OPTIONS:
			config.Options = p.parseComposeOptions()
		case lexer.DEDENT:
			break composeBody
		default:
			p.nextToken()
		}
	}

	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return config
}

// parseComposeOptions parses compose options
func (p *Parser) parseComposeOptions() *ast.ComposeOptions {
	options := &ast.ComposeOptions{}

	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' after 'options'")
		return nil
	}

	if !p.expectPeek(lexer.INDENT) {
		p.addError("expected indent after 'options:'")
		return nil
	}

optionsBody:
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		for p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}

		switch p.curToken.Type {
		case lexer.FORCE:
			if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "recreate" {
				p.nextToken() // consume 'recreate'
			}
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for force_recreate")
				return nil
			}
			options.ForceRecreate = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.NO:
			if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "deps" {
				p.nextToken() // consume 'deps'
			}
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for no_deps")
				return nil
			}
			options.NoDeps = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.BUILD:
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for build")
				return nil
			}
			options.Build = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.PULL:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected pull policy string")
				return nil
			}
			options.Pull = p.curToken.Literal
			p.nextToken()
		case lexer.TIMEOUT:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected timeout string")
				return nil
			}
			options.Timeout = p.curToken.Literal
			p.nextToken()
		case lexer.SCALE:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected scale string")
				return nil
			}
			options.Scale = p.curToken.Literal
			p.nextToken()
		case lexer.WAIT:
			// Check if next is timeout
			if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "timeout" {
				p.nextToken() // consume 'timeout'
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected wait timeout string")
					return nil
				}
				options.WaitTimeout = p.curToken.Literal
				p.nextToken()
			} else {
				if !p.expectPeek(lexer.BOOLEAN) {
					p.addError("expected boolean for wait")
					return nil
				}
				options.Wait = p.curToken.Literal == "true"
				p.nextToken()
			}
		case lexer.DEDENT:
			break optionsBody
		default:
			p.nextToken()
		}
	}

	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return options
}

// parseEnvFileConfig parses env_file configuration
func (p *Parser) parseEnvFileConfig() *ast.EnvFileConfig {
	config := &ast.EnvFileConfig{}

	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' after 'env_file'")
		return nil
	}

	if !p.expectPeek(lexer.INDENT) {
		p.addError("expected indent after 'env_file:'")
		return nil
	}

envFileBody:
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		for p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}

		switch p.curToken.Type {
		case lexer.REQUIRED:
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for required")
				return nil
			}
			config.Required = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.TASK:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected task name string")
				return nil
			}
			config.Task = p.curToken.Literal
			p.nextToken()
		case lexer.DEDENT:
			break envFileBody
		default:
			p.nextToken()
		}
	}

	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return config
}

// parseOrchestrateStatement parses an orchestration declaration
// orchestrate "group_name" means "Description":
func (p *Parser) parseOrchestrateStatement() *ast.OrchestrateStatement {
	stmt := &ast.OrchestrateStatement{
		Token: p.curToken,
	}

	// Expect "orchestrate"
	if !p.expectPeek(lexer.STRING) {
		p.addError("expected orchestration name string")
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Optional "means" clause
	if p.peekToken.Type == lexer.MEANS {
		p.nextToken()
		if !p.expectPeek(lexer.STRING) {
			p.addError("expected description string after 'means'")
			return nil
		}
		stmt.Description = p.curToken.Literal
	}

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' after orchestrate declaration")
		return nil
	}

	// Expect indent
	if !p.expectPeek(lexer.INDENT) {
		p.addError("expected indent after orchestrate declaration")
		return nil
	}

	// Move past the INDENT token
	p.nextToken()

	// Parse orchestration body
	stmt.Scale = make(map[string]int)

orchestrateBody:
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		for p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}

		switch p.curToken.Type {
		case lexer.SERVICES:
			stmt.Services = p.parseOrchestrationStringArray()
		case lexer.STRATEGY:
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected strategy string")
				return nil
			}
			stmt.Strategy = p.curToken.Literal
			p.nextToken()
		case lexer.CIRCUIT:
			if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "breaker" {
				p.nextToken() // consume 'breaker'
			}
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for circuit_breaker")
				return nil
			}
			stmt.CircuitBreaker = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.STOP:
			if p.peekToken.Type == lexer.ON {
				p.nextToken() // consume 'on'
				if p.peekToken.Type == lexer.FAILURE {
					p.nextToken() // consume 'failure'
				}
			}
			if !p.expectPeek(lexer.BOOLEAN) {
				p.addError("expected boolean for stop_on_failure")
				return nil
			}
			stmt.StopOnFailure = p.curToken.Literal == "true"
			p.nextToken()
		case lexer.HEALTH:
			if p.peekToken.Type == lexer.CHECK {
				p.nextToken() // consume 'check'
				if p.peekToken.Type == lexer.INTERVAL {
					p.nextToken() // consume 'interval'
				}
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected health check interval string")
				return nil
			}
			stmt.HealthCheckInterval = p.curToken.Literal
			p.nextToken()
		case lexer.STARTUP:
			if p.peekToken.Type == lexer.TIMEOUT {
				p.nextToken() // consume 'timeout'
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected startup timeout string")
				return nil
			}
			stmt.StartupTimeout = p.curToken.Literal
			p.nextToken()
		case lexer.SHUTDOWN:
			if p.peekToken.Type == lexer.TIMEOUT {
				p.nextToken() // consume 'timeout'
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected shutdown timeout string")
				return nil
			}
			stmt.ShutdownTimeout = p.curToken.Literal
			p.nextToken()
		case lexer.PRE:
			if p.peekToken.Type == lexer.IDENT || p.peekToken.Type == lexer.TASK {
				p.nextToken()
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected task name string after 'pre'")
				return nil
			}
			stmt.PreTask = p.curToken.Literal
			p.nextToken()
		case lexer.POST:
			if p.peekToken.Type == lexer.IDENT || p.peekToken.Type == lexer.TASK {
				p.nextToken()
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected task name string after 'post'")
				return nil
			}
			stmt.PostTask = p.curToken.Literal
			p.nextToken()
		case lexer.FAILURE:
			if p.peekToken.Type == lexer.THRESHOLD {
				p.nextToken() // consume 'threshold'
			}
			if !p.expectPeek(lexer.NUMBER) {
				p.addError("expected number for failure_threshold")
				return nil
			}
			threshold, _ := strconv.Atoi(p.curToken.Literal)
			stmt.FailureThreshold = threshold
			p.nextToken()
		case lexer.RECOVERY:
			if p.peekToken.Type == lexer.TIMEOUT {
				p.nextToken() // consume 'timeout'
			}
			if !p.expectPeek(lexer.STRING) {
				p.addError("expected recovery timeout string")
				return nil
			}
			stmt.RecoveryTimeout = p.curToken.Literal
			p.nextToken()
		case lexer.MAKEFILE:
			if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "order" {
				p.nextToken() // consume 'order'
				stmt.MakefileOrder = p.parseOrchestrationStringArray()
			} else if p.peekToken.Type == lexer.TIMEOUT {
				p.nextToken() // consume 'timeout'
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected makefile timeout string")
					return nil
				}
				stmt.MakefileTimeout = p.curToken.Literal
				p.nextToken()
			}
		case lexer.CLONE:
			if p.peekToken.Type == lexer.IDENT && p.peekToken.Literal == "order" {
				p.nextToken() // consume 'order'
				stmt.CloneOrder = p.parseOrchestrationStringArray()
			} else if p.peekToken.Type == lexer.TIMEOUT {
				p.nextToken() // consume 'timeout'
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected clone timeout string")
					return nil
				}
				stmt.CloneTimeout = p.curToken.Literal
				p.nextToken()
			}
		case lexer.DEDENT:
			break orchestrateBody
		case lexer.IDENT:
			// Handle orchestration options like stop_on_failure, circuit_breaker
			switch p.curToken.Literal {
			case "stop_on_failure":
				p.nextToken() // consume the identifier
				if p.curToken.Type != lexer.BOOLEAN {
					p.addError(fmt.Sprintf("expected boolean for stop_on_failure, got %s", p.curToken.Type))
					return nil
				}
				stmt.StopOnFailure = p.curToken.Literal == "true"
				p.nextToken()
			case "circuit_breaker":
				p.nextToken() // consume the identifier
				if p.curToken.Type != lexer.BOOLEAN {
					p.addError(fmt.Sprintf("expected boolean for circuit_breaker, got %s", p.curToken.Type))
					return nil
				}
				stmt.CircuitBreaker = p.curToken.Literal == "true"
				p.nextToken()
			case "health_check_interval":
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected health check interval string")
					return nil
				}
				stmt.HealthCheckInterval = p.curToken.Literal
				p.nextToken()
			case "startup_timeout":
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected startup timeout string")
					return nil
				}
				stmt.StartupTimeout = p.curToken.Literal
				p.nextToken()
			case "shutdown_timeout":
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected shutdown timeout string")
					return nil
				}
				stmt.ShutdownTimeout = p.curToken.Literal
				p.nextToken()
			case "pre_task":
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected task name string after 'pre_task'")
					return nil
				}
				stmt.PreTask = p.curToken.Literal
				p.nextToken()
			case "post_task":
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected task name string after 'post_task'")
					return nil
				}
				stmt.PostTask = p.curToken.Literal
				p.nextToken()
			case "makefile_timeout":
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected makefile timeout string")
					return nil
				}
				stmt.MakefileTimeout = p.curToken.Literal
				p.nextToken()
			case "clone_timeout":
				if !p.expectPeek(lexer.STRING) {
					p.addError("expected clone timeout string")
					return nil
				}
				stmt.CloneTimeout = p.curToken.Literal
				p.nextToken()
			case "makefile_order":
				stmt.MakefileOrder = p.parseOrchestrationStringArray()
			case "clone_order":
				stmt.CloneOrder = p.parseOrchestrationStringArray()
			default:
				p.addError(fmt.Sprintf("unexpected identifier in orchestration body: %s", p.curToken.Literal))
				p.nextToken()
			}
		case lexer.COMMENT, lexer.MULTILINE_COMMENT:
			p.nextToken() // Skip comments
		default:
			p.addError(fmt.Sprintf("unexpected token in orchestration body: %s", p.curToken.Type))
			p.nextToken() // Advance to avoid infinite loop
		}
	}

	// Consume the DEDENT
	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return stmt
}

// parseOrchestrationStringArray parses an array of strings ["string1", "string2"]
func (p *Parser) parseOrchestrationStringArray() []string {
	result := []string{}

	if !p.expectPeek(lexer.LBRACKET) {
		p.addError("expected '[' for array")
		return nil
	}

	p.nextToken()

	for p.curToken.Type != lexer.RBRACKET && p.curToken.Type != lexer.EOF {
		if p.curToken.Type == lexer.STRING {
			result = append(result, p.curToken.Literal)
		} else {
			p.addError(fmt.Sprintf("expected string in array, got %s", p.curToken.Type))
			// Advance to avoid infinite loop
			p.nextToken()
			continue
		}

		p.nextToken()

		if p.curToken.Type == lexer.COMMA {
			p.nextToken()
		}
	}

	if p.curToken.Type != lexer.RBRACKET {
		p.addError("expected ']' to close array")
		return nil
	}

	p.nextToken()
	return result
}

// parseEnvironmentMap parses a map of environment variables
func (p *Parser) parseEnvironmentMap() map[string]string {
	envMap := make(map[string]string)

	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' after 'environment'")
		return nil
	}

	if !p.expectPeek(lexer.INDENT) {
		p.addError("expected indent after 'environment:'")
		return nil
	}

envMapBody:
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		for p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}

		if p.curToken.Type == lexer.IDENT || p.curToken.Type == lexer.STRING {
			key := p.curToken.Literal

			// Strip quotes if it's a STRING token
			if p.curToken.Type == lexer.STRING {
				key = strings.Trim(key, "\"")
			}

			if !p.expectPeek(lexer.STRING) {
				p.addError(fmt.Sprintf("expected value string for environment variable %s", key))
				return nil
			}
			envMap[key] = p.curToken.Literal
			p.nextToken()
		} else if p.curToken.Type == lexer.DEDENT {
			break envMapBody
		} else {
			p.nextToken()
		}
	}

	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return envMap
}

// parseHeadersMap parses a map of HTTP headers
func (p *Parser) parseHeadersMap() map[string]string {
	headersMap := make(map[string]string)

	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' after 'headers'")
		return nil
	}

	if !p.expectPeek(lexer.INDENT) {
		p.addError("expected indent after 'headers:'")
		return nil
	}

	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		for p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}

		if p.curToken.Type == lexer.IDENT || p.curToken.Type == lexer.STRING {
			key := p.curToken.Literal

			// Strip quotes if it's a STRING token
			if p.curToken.Type == lexer.STRING {
				key = strings.Trim(key, "\"")
			}

			if !p.expectPeek(lexer.STRING) {
				p.addError(fmt.Sprintf("expected value string for header %s", key))
				return nil
			}
			headersMap[key] = p.curToken.Literal
			p.nextToken()
		} else if p.curToken.Type == lexer.DEDENT {
			break
		} else {
			p.nextToken()
		}
	}

	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return headersMap
}

// parseDockerNetworksConfig parses docker networks configuration
func (p *Parser) parseDockerNetworksConfig() map[string]*ast.DockerNetworkConfig {
	networks := make(map[string]*ast.DockerNetworkConfig)

	// Expect colon after "networks"
	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' after 'networks'")
		return networks
	}
	p.nextToken() // consume ':'

	// Skip initial INDENT if present
	if p.curToken.Type == lexer.INDENT {
		p.nextToken()
	}

	// Parse network configurations
	for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
		if p.curToken.Type == lexer.STRING {
			networkName := p.curToken.Literal
			networkConfig := &ast.DockerNetworkConfig{
				Token: p.curToken,
				Name:  networkName,
			}

			p.nextToken() // consume network name

			// Expect colon after network name
			if p.curToken.Type != lexer.COLON {
				p.addError(fmt.Sprintf("expected ':' after network name '%s', got %s", networkName, p.curToken.Type))
				return networks
			}
			p.nextToken() // consume ':'

			// Skip INDENT if present
			if p.curToken.Type == lexer.INDENT {
				p.nextToken()
			}

			// Parse network properties (direct format without dashes)
		networkProps:
			for p.curToken.Type != lexer.DEDENT && p.curToken.Type != lexer.EOF {
				// Skip newlines
				for p.curToken.Type == lexer.NEWLINE {
					p.nextToken()
				}

				// Parse the property
				switch p.curToken.Type {
				case lexer.EXTERNAL:
					p.nextToken() // consume 'external'
					// Expect colon and boolean
					if p.curToken.Type == lexer.COLON {
						p.nextToken()
						if p.curToken.Type == lexer.BOOLEAN {
							networkConfig.External = p.curToken.Literal == "true"
							p.nextToken()
						}
					}
				case lexer.REQUIRED:
					p.nextToken() // consume 'required'
					// Expect colon and boolean
					if p.curToken.Type == lexer.COLON {
						p.nextToken()
						if p.curToken.Type == lexer.BOOLEAN {
							networkConfig.Required = p.curToken.Literal == "true"
							p.nextToken()
						}
					}
				case lexer.AUTOPROVISION:
					p.nextToken() // consume 'autoprovision'
					// Expect colon and boolean
					if p.curToken.Type == lexer.COLON {
						p.nextToken()
						if p.curToken.Type == lexer.BOOLEAN {
							networkConfig.AutoProvision = p.curToken.Literal == "true"
							p.nextToken()
						}
					}
				case lexer.DRIVER:
					p.nextToken() // consume 'driver'
					// Expect colon and string
					if p.curToken.Type == lexer.COLON {
						p.nextToken()
						if p.curToken.Type == lexer.STRING {
							networkConfig.Driver = p.curToken.Literal
							p.nextToken()
						}
					}
				case lexer.DEDENT:
					// End of network config
					break networkProps
				default:
					p.addError(fmt.Sprintf("unexpected token in network config: %s", p.curToken.Type))
					p.nextToken()
				}
			}

			// Skip DEDENT if present
			if p.curToken.Type == lexer.DEDENT {
				p.nextToken()
			}

			networks[networkName] = networkConfig
		} else {
			p.nextToken()
		}
	}

	// Skip final DEDENT if present
	if p.curToken.Type == lexer.DEDENT {
		p.nextToken()
	}

	return networks
}
