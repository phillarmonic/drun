package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
)

func TestEngine_DownloadDryRun(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		taskName       string
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "basic download dry run",
			input: `version: 2.0
task "download_test":
	download "https://example.com/file.zip" to "file.zip"`,
			taskName: "download_test",
			expectedOutput: []string{
				"[DRY RUN] Would download https://example.com/file.zip to file.zip",
			},
		},
		{
			name: "download with overwrite",
			input: `version: 2.0
task "download_overwrite":
	download "https://example.com/data.json" to "data.json" allow overwrite`,
			taskName: "download_overwrite",
			expectedOutput: []string{
				"[DRY RUN] Would download https://example.com/data.json to data.json (overwrite allowed)",
			},
		},
		{
			name: "download with permissions",
			input: `version: 2.0
task "download_perms":
	download "https://example.com/script.sh" to "script.sh" allow permissions ["read","write","execute"] to ["user"] allow permissions ["read"] to ["group","others"]`,
			taskName: "download_perms",
			expectedOutput: []string{
				"[DRY RUN] Would download https://example.com/script.sh to script.sh",
				"with permissions:",
			},
		},
		{
			name: "download with extraction",
			input: `version: 2.0
task "download_extract":
	download "https://example.com/archive.zip" to "archive.zip" extract to "extracted/"`,
			taskName: "download_extract",
			expectedOutput: []string{
				"[DRY RUN] Would download https://example.com/archive.zip to archive.zip",
			},
		},
		{
			name: "download extract and remove",
			input: `version: 2.0
task "download_extract_remove":
	download "https://example.com/release.tar.gz" to "release.tar.gz" extract to "release/" remove archive`,
			taskName: "download_extract_remove",
			expectedOutput: []string{
				"[DRY RUN] Would download https://example.com/release.tar.gz to release.tar.gz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)

			l := lexer.NewLexer(tt.input)
			p := parser.NewParser(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			err := engine.Execute(program, tt.taskName)
			if err != nil {
				t.Fatalf("Execution error: %v", err)
			}

			outputStr := output.String()

			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, outputStr)
				}
			}

			for _, notExpected := range tt.notExpected {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output NOT to contain %q, got:\n%s", notExpected, outputStr)
				}
			}
		})
	}
}

func TestEngine_DownloadVariableInterpolation(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		taskName       string
		expectedOutput []string
	}{
		{
			name: "download with version variable",
			input: `version: 2.0
task "download_version":
	requires $version
	download "https://example.com/releases/{$version}/app.zip" to "app-{$version}.zip"`,
			taskName: "download_version",
			expectedOutput: []string{
				"Would download https://example.com/releases/1.0.0/app.zip to app-1.0.0.zip",
			},
		},
		{
			name: "download with multiple variables",
			input: `version: 2.0
task "download_multi_vars":
	let $platform = "linux"
	let $arch = "amd64"
	download "https://example.com/releases/{$platform}-{$arch}/binary.tar.gz" to "{$platform}-{$arch}.tar.gz"`,
			taskName: "download_multi_vars",
			expectedOutput: []string{
				"Would download https://example.com/releases/linux-amd64/binary.tar.gz to linux-amd64.tar.gz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)

			l := lexer.NewLexer(tt.input)
			p := parser.NewParser(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			// For version test, set the variable
			var err error
			if strings.Contains(tt.input, "requires $version") {
				err = engine.ExecuteWithParams(program, tt.taskName, map[string]string{"version": "1.0.0"})
			} else {
				err = engine.Execute(program, tt.taskName)
			}
			if err != nil {
				t.Fatalf("Execution error: %v", err)
			}

			outputStr := output.String()

			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, outputStr)
				}
			}
		})
	}
}

func TestEngine_DownloadInLoop(t *testing.T) {
	input := `version: 2.0

task "download_multiple":
	for each $file in ["users","posts","comments"]:
		download "https://api.example.com/{$file}.json" to "{$file}.json"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	err := engine.Execute(program, "download_multiple")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	expectedFiles := []string{"users.json", "posts.json", "comments.json"}
	for _, file := range expectedFiles {
		if !strings.Contains(outputStr, file) {
			t.Errorf("Expected output to contain download for %q, got:\n%s", file, outputStr)
		}
	}
}

func TestEngine_DownloadParallelLoop(t *testing.T) {
	input := `version: 2.0

task "parallel_downloads":
	for each $resource in ["users","posts","comments"] in parallel:
		download "https://api.example.com/{$resource}.json" to "{$resource}.json"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	err := engine.Execute(program, "parallel_downloads")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// In dry-run mode, check that parallel execution is mentioned
	if !strings.Contains(outputStr, "Would execute 3 items in parallel") {
		t.Errorf("Expected parallel execution mention, got:\n%s", outputStr)
	}

	// Check that the resources are mentioned
	expectedResources := []string{"users", "posts", "comments"}
	for _, resource := range expectedResources {
		if !strings.Contains(outputStr, resource) {
			t.Errorf("Expected output to contain resource %q, got:\n%s", resource, outputStr)
		}
	}
}

func TestEngine_DownloadConditional(t *testing.T) {
	input := `version: 2.0

task "conditional_download":
	let $env = "production"
	when $env is "production":
		download "https://example.com/prod-config.json" to "config.json"
	otherwise:
		download "https://example.com/dev-config.json" to "config.json"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	err := engine.Execute(program, "conditional_download")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	if !strings.Contains(outputStr, "prod-config.json") {
		t.Errorf("Expected production config download, got:\n%s", outputStr)
	}

	if strings.Contains(outputStr, "dev-config.json") {
		t.Errorf("Should not download dev config, got:\n%s", outputStr)
	}
}

func TestEngine_DownloadWithTryCatch(t *testing.T) {
	input := `version: 2.0

task "safe_download":
	try:
		download "https://example.com/file.zip" to "file.zip"
		success "Downloaded successfully"
	catch:
		warn "Download failed, using fallback"
		download "https://fallback.example.com/file.zip" to "file.zip"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	err := engine.Execute(program, "safe_download")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	if !strings.Contains(outputStr, "https://example.com/file.zip") {
		t.Errorf("Expected primary download attempt, got:\n%s", outputStr)
	}
}

func TestEngine_DownloadPermissionMatrix(t *testing.T) {
	input := `version: 2.0

task "test_perms":
	download "https://example.com/binary" to "binary" allow permissions ["read","execute"] to ["user","group","others"] allow permissions ["write"] to ["user"]`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	err := engine.Execute(program, "test_perms")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check that permissions are mentioned in dry run output
	if !strings.Contains(outputStr, "with permissions:") {
		t.Errorf("Expected permissions in output, got:\n%s", outputStr)
	}
}

func TestEngine_DownloadExtraction(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		taskName       string
		expectedOutput []string
	}{
		{
			name: "extract without remove",
			input: `version: 2.0
task "extract_keep":
	download "https://example.com/archive.zip" to "archive.zip" extract to "extracted/"`,
			taskName: "extract_keep",
			expectedOutput: []string{
				"Would download https://example.com/archive.zip to archive.zip",
			},
		},
		{
			name: "extract with remove",
			input: `version: 2.0
task "extract_remove":
	download "https://example.com/release.tar.gz" to "release.tar.gz" extract to "release/" remove archive`,
			taskName: "extract_remove",
			expectedOutput: []string{
				"Would download https://example.com/release.tar.gz to release.tar.gz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)

			l := lexer.NewLexer(tt.input)
			p := parser.NewParser(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			err := engine.Execute(program, tt.taskName)
			if err != nil {
				t.Fatalf("Execution error: %v", err)
			}

			outputStr := output.String()

			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, outputStr)
				}
			}
		})
	}
}

func TestEngine_DownloadAuthAndHeaders(t *testing.T) {
	input := `version: 2.0

task "auth_download":
	download "https://api.example.com/private/data.json" to "data.json" with auth bearer "secret-token-123" with header "Accept: application/json" timeout "60s"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	err := engine.Execute(program, "auth_download")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	if !strings.Contains(outputStr, "https://api.example.com/private/data.json") {
		t.Errorf("Expected URL in output, got:\n%s", outputStr)
	}

	if !strings.Contains(outputStr, "data.json") {
		t.Errorf("Expected path in output, got:\n%s", outputStr)
	}
}
