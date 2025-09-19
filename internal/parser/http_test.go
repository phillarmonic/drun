package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_HTTPGetRequest(t *testing.T) {
	input := `version: 2.0

task "api_test":
  get "https://api.example.com/users"
  
  success "API request completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("program should have 1 task. got=%d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if len(task.Body) != 2 {
		t.Fatalf("task should have 2 statements. got=%d", len(task.Body))
	}

	// Check HTTP statement
	httpStmt, ok := task.Body[0].(*ast.HTTPStatement)
	if !ok {
		t.Fatalf("first statement should be HTTPStatement. got=%T", task.Body[0])
	}

	if httpStmt.Method != "GET" {
		t.Errorf("http method not 'GET'. got=%q", httpStmt.Method)
	}

	if httpStmt.URL != "https://api.example.com/users" {
		t.Errorf("http URL not 'https://api.example.com/users'. got=%q", httpStmt.URL)
	}
}

func TestParser_HTTPPostWithBody(t *testing.T) {
	input := `version: 2.0

task "create_user":
  post "https://api.example.com/users" with body "name=John&email=john@example.com"
  
  success "User created!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	httpStmt, ok := task.Body[0].(*ast.HTTPStatement)
	if !ok {
		t.Fatalf("first statement should be HTTPStatement. got=%T", task.Body[0])
	}

	if httpStmt.Method != "POST" {
		t.Errorf("http method not 'POST'. got=%q", httpStmt.Method)
	}

	if httpStmt.URL != "https://api.example.com/users" {
		t.Errorf("http URL not 'https://api.example.com/users'. got=%q", httpStmt.URL)
	}

	expectedBody := `name=John&email=john@example.com`
	if httpStmt.Body != expectedBody {
		t.Errorf("http body not '%s'. got=%q", expectedBody, httpStmt.Body)
	}
}

func TestParser_HTTPWithHeaders(t *testing.T) {
	input := `version: 2.0

task "api_with_headers":
  get "https://api.example.com/data" with header "Authorization: Bearer token123"
  
  success "Request with headers completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	httpStmt, ok := task.Body[0].(*ast.HTTPStatement)
	if !ok {
		t.Fatalf("first statement should be HTTPStatement. got=%T", task.Body[0])
	}

	if httpStmt.Method != "GET" {
		t.Errorf("http method not 'GET'. got=%q", httpStmt.Method)
	}

	if httpStmt.Headers["Authorization"] != "Bearer token123" {
		t.Errorf("http Authorization header not 'Bearer token123'. got=%q", httpStmt.Headers["Authorization"])
	}
}

func TestParser_HTTPWithAuth(t *testing.T) {
	input := `version: 2.0

task "authenticated_request":
  get "https://api.example.com/secure" with auth bearer "my-secret-token"
  
  success "Authenticated request completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	httpStmt, ok := task.Body[0].(*ast.HTTPStatement)
	if !ok {
		t.Fatalf("first statement should be HTTPStatement. got=%T", task.Body[0])
	}

	if httpStmt.Method != "GET" {
		t.Errorf("http method not 'GET'. got=%q", httpStmt.Method)
	}

	if httpStmt.Auth["bearer"] != "my-secret-token" {
		t.Errorf("http bearer auth not 'my-secret-token'. got=%q", httpStmt.Auth["bearer"])
	}
}

func TestParser_HTTPWithContentType(t *testing.T) {
	input := `version: 2.0

task "json_request":
  post "https://api.example.com/data" content type json with body "key=value"
  
  success "JSON request completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	httpStmt, ok := task.Body[0].(*ast.HTTPStatement)
	if !ok {
		t.Fatalf("first statement should be HTTPStatement. got=%T", task.Body[0])
	}

	if httpStmt.Method != "POST" {
		t.Errorf("http method not 'POST'. got=%q", httpStmt.Method)
	}

	if httpStmt.Headers["Content-Type"] != "application/json" {
		t.Errorf("http Content-Type header not 'application/json'. got=%q", httpStmt.Headers["Content-Type"])
	}

	if httpStmt.Body != `key=value` {
		t.Errorf("http body not 'key=value'. got=%q", httpStmt.Body)
	}
}

func TestParser_HTTPWithAccept(t *testing.T) {
	input := `version: 2.0

task "accept_json":
  get "https://api.example.com/data" accept json
  
  success "Request with Accept header completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	httpStmt, ok := task.Body[0].(*ast.HTTPStatement)
	if !ok {
		t.Fatalf("first statement should be HTTPStatement. got=%T", task.Body[0])
	}

	if httpStmt.Method != "GET" {
		t.Errorf("http method not 'GET'. got=%q", httpStmt.Method)
	}

	if httpStmt.Headers["Accept"] != "application/json" {
		t.Errorf("http Accept header not 'application/json'. got=%q", httpStmt.Headers["Accept"])
	}
}

func TestParser_HTTPWithTimeout(t *testing.T) {
	input := `version: 2.0

task "timeout_request":
  get "https://api.example.com/slow" timeout "30s"
  
  success "Request with timeout completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	httpStmt, ok := task.Body[0].(*ast.HTTPStatement)
	if !ok {
		t.Fatalf("first statement should be HTTPStatement. got=%T", task.Body[0])
	}

	if httpStmt.Method != "GET" {
		t.Errorf("http method not 'GET'. got=%q", httpStmt.Method)
	}

	if httpStmt.Options["timeout"] != "30s" {
		t.Errorf("http timeout option not '30s'. got=%q", httpStmt.Options["timeout"])
	}
}

func TestParser_HTTPMultipleMethods(t *testing.T) {
	input := `version: 2.0

task "api_workflow":
  get "https://api.example.com/users"
  post "https://api.example.com/users" with body "name=John"
  put "https://api.example.com/users/1" with body "name=Jane"
  delete "https://api.example.com/users/1"
  
  success "API workflow completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	if len(task.Body) != 5 {
		t.Fatalf("task should have 5 statements. got=%d", len(task.Body))
	}

	// Check all four HTTP statements
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for i, expectedMethod := range methods {
		httpStmt, ok := task.Body[i].(*ast.HTTPStatement)
		if !ok {
			t.Fatalf("statement %d should be HTTPStatement. got=%T", i, task.Body[i])
		}

		if httpStmt.Method != expectedMethod {
			t.Errorf("http method %d not '%s'. got=%q", i, expectedMethod, httpStmt.Method)
		}
	}
}
