package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_HTTPGetRequest(t *testing.T) {
	input := `version: 2.0

task "api_test":
  get "https://api.example.com/users"
  success "API request completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "api_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🌐 Making HTTP request: curl -X GET https://api.example.com/users",
		"✅ API request completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_HTTPPostWithBody(t *testing.T) {
	input := `version: 2.0

task "create_user":
  post "https://api.example.com/users" with body "name=John&email=john@example.com"
  success "User created!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "create_user", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🌐 Making HTTP request: curl -X POST -d name=John&email=john@example.com https://api.example.com/users",
		"✅ User created!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_HTTPWithHeaders(t *testing.T) {
	input := `version: 2.0

task "api_with_headers":
  get "https://api.example.com/data" with header "Authorization: Bearer token123"
  success "Request with headers completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "api_with_headers", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🌐 Making HTTP request: curl -X GET -H \"Authorization: Bearer token123\" https://api.example.com/data",
		"✅ Request with headers completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_HTTPWithAuth(t *testing.T) {
	input := `version: 2.0

task "authenticated_request":
  get "https://api.example.com/secure" with auth bearer "my-secret-token"
  success "Authenticated request completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "authenticated_request", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🌐 Making HTTP request: curl -X GET -H \"Authorization: Bearer my-secret-token\" https://api.example.com/secure",
		"✅ Authenticated request completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_HTTPWithContentType(t *testing.T) {
	input := `version: 2.0

task "json_request":
  post "https://api.example.com/data" content type json with body "key=value"
  success "JSON request completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "json_request", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🌐 Making HTTP request: curl -X POST -H \"Content-Type: application/json\" -d key=value https://api.example.com/data",
		"✅ JSON request completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_HTTPWithAccept(t *testing.T) {
	input := `version: 2.0

task "accept_json":
  get "https://api.example.com/data" accept json
  success "Request with Accept header completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "accept_json", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🌐 Making HTTP request: curl -X GET -H \"Accept: application/json\" https://api.example.com/data",
		"✅ Request with Accept header completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_HTTPWithTimeout(t *testing.T) {
	input := `version: 2.0

task "timeout_request":
  get "https://api.example.com/slow" timeout "30s"
  success "Request with timeout completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "timeout_request", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🌐 Making HTTP request: curl -X GET --max-time 30s https://api.example.com/slow",
		"✅ Request with timeout completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_HTTPWithVariableInterpolation(t *testing.T) {
	input := `version: 2.0

task "api_with_params":
  requires api_url from ["https://api.example.com", "https://test.api.com"]
  requires auth_token from ["token123", "token456", "token789"]
  
  get "{api_url}/users" with auth bearer "{auth_token}"
  success "API request with parameters completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"api_url":    "https://api.example.com",
		"auth_token": "token123",
	}

	err = engine.ExecuteWithParams(program, "api_with_params", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🌐 Making HTTP request: curl -X GET -H \"Authorization: Bearer token123\" https://api.example.com/users",
		"✅ API request with parameters completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_HTTPDryRun(t *testing.T) {
	input := `version: 2.0

task "api_operations":
  get "https://api.example.com/users"
  post "https://api.example.com/users" with body "name=John"
  put "https://api.example.com/users/1" with body "name=Jane"
  delete "https://api.example.com/users/1"
  success "API operations completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.ExecuteWithParams(program, "api_operations", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"[DRY RUN] Would execute HTTP request: curl -X GET https://api.example.com/users",
		"[DRY RUN] Would execute HTTP request: curl -X POST -d name=John https://api.example.com/users",
		"[DRY RUN] Would execute HTTP request: curl -X PUT -d name=Jane https://api.example.com/users/1",
		"[DRY RUN] Would execute HTTP request: curl -X DELETE https://api.example.com/users/1",
		"[DRY RUN] success: API operations completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_HTTPMultipleMethods(t *testing.T) {
	input := `version: 2.0

task "api_workflow":
  get "https://api.example.com/users"
  post "https://api.example.com/users" with body "name=John"
  put "https://api.example.com/users/1" with body "name=Jane"
  delete "https://api.example.com/users/1"
  success "API workflow completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "api_workflow", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	// Check that all HTTP methods are executed
	httpMethods := []string{
		"curl -X GET https://api.example.com/users",
		"curl -X POST -d name=John https://api.example.com/users",
		"curl -X PUT -d name=Jane https://api.example.com/users/1",
		"curl -X DELETE https://api.example.com/users/1",
	}

	for _, method := range httpMethods {
		if !strings.Contains(outputStr, method) {
			t.Errorf("Expected output to contain HTTP method %q, got %q", method, outputStr)
		}
	}

	if !strings.Contains(outputStr, "✅ API workflow completed!") {
		t.Errorf("Expected success message in output")
	}
}
