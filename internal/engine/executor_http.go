package engine

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/domain/statement"
)

// Domain: HTTP Operations Execution
// This file contains executors for:
// - HTTP requests (GET, POST, PUT, DELETE, PATCH)
// - API interactions

// executeHTTP executes HTTP operations
func (e *Engine) executeHTTP(httpStmt *statement.HTTP, ctx *ExecutionContext) error {
	// Interpolate variables in HTTP statement
	method := httpStmt.Method
	url := e.interpolateVariables(httpStmt.URL, ctx)
	body := e.interpolateVariables(httpStmt.Body, ctx)

	// Interpolate headers
	headers := make(map[string]string, len(httpStmt.Headers))
	for key, value := range httpStmt.Headers {
		headers[key] = e.interpolateVariables(value, ctx)
	}

	// Interpolate auth
	auth := make(map[string]string, len(httpStmt.Auth))
	for key, value := range httpStmt.Auth {
		auth[key] = e.interpolateVariables(value, ctx)
	}

	// Interpolate options
	options := make(map[string]string, len(httpStmt.Options))
	for key, value := range httpStmt.Options {
		options[key] = e.interpolateVariables(value, ctx)
	}

	if e.dryRun {
		return e.buildHTTPCommand(method, url, body, headers, auth, options, true)
	}

	// Show what we're about to do with appropriate emoji
	switch method {
	case "GET":
		_, _ = fmt.Fprintf(e.output, "📥 GET request to: %s\n", url)
	case "POST":
		_, _ = fmt.Fprintf(e.output, "📤 POST request to: %s\n", url)
	case "PUT":
		_, _ = fmt.Fprintf(e.output, "🔄 PUT request to: %s\n", url)
	case "PATCH":
		_, _ = fmt.Fprintf(e.output, "🔧 PATCH request to: %s\n", url)
	case "DELETE":
		_, _ = fmt.Fprintf(e.output, "🗑️  DELETE request to: %s\n", url)
	case "HEAD":
		_, _ = fmt.Fprintf(e.output, "🔍 HEAD request to: %s\n", url)
	default:
		_, _ = fmt.Fprintf(e.output, "🌐 %s request to: %s\n", method, url)
	}

	// Handle special HTTP operations
	if downloadPath, exists := options["download"]; exists {
		_, _ = fmt.Fprintf(e.output, "💾 Downloading to: %s\n", downloadPath)
	}

	if uploadPath, exists := options["upload"]; exists {
		_, _ = fmt.Fprintf(e.output, "📤 Uploading from: %s\n", uploadPath)
	}

	// Build and execute the actual HTTP request
	return e.buildHTTPCommand(method, url, body, headers, auth, options, false)
}
