package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_DownloadBasic(t *testing.T) {
	input := `version: 2.0

task "download_file":
  download "https://example.com/file.zip" to "downloads/file.zip"
  
  success "File downloaded!"`

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

	// Check Download statement
	downloadStmt, ok := task.Body[0].(*ast.DownloadStatement)
	if !ok {
		t.Fatalf("first statement should be DownloadStatement. got=%T", task.Body[0])
	}

	if downloadStmt.URL != "https://example.com/file.zip" {
		t.Errorf("download URL not 'https://example.com/file.zip'. got=%q", downloadStmt.URL)
	}

	if downloadStmt.Path != "downloads/file.zip" {
		t.Errorf("download path not 'downloads/file.zip'. got=%q", downloadStmt.Path)
	}
}

func TestParser_DownloadWithOverwrite(t *testing.T) {
	input := `version: 2.0

task "download_overwrite":
  download "https://example.com/data.json" to "data.json" allow overwrite
  
  success "File downloaded with overwrite!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	downloadStmt, ok := task.Body[0].(*ast.DownloadStatement)
	if !ok {
		t.Fatalf("first statement should be DownloadStatement. got=%T", task.Body[0])
	}

	if !downloadStmt.AllowOverwrite {
		t.Errorf("AllowOverwrite should be true. got=%v", downloadStmt.AllowOverwrite)
	}
}

func TestParser_DownloadWithPermissions(t *testing.T) {
	input := `version: 2.0

task "download_with_perms":
  download "https://example.com/script.sh" to "script.sh" allow permissions ["read","write","execute"] to ["user"] allow permissions ["read"] to ["group","others"]
  
  success "File downloaded with permissions!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	downloadStmt, ok := task.Body[0].(*ast.DownloadStatement)
	if !ok {
		t.Fatalf("first statement should be DownloadStatement. got=%T", task.Body[0])
	}

	if len(downloadStmt.AllowPermissions) != 2 {
		t.Fatalf("should have 2 permission specs. got=%d", len(downloadStmt.AllowPermissions))
	}

	// Check first permission spec
	perm1 := downloadStmt.AllowPermissions[0]
	if len(perm1.Permissions) != 3 {
		t.Errorf("first permission spec should have 3 permissions. got=%d", len(perm1.Permissions))
	}
	if perm1.Permissions[0] != "read" || perm1.Permissions[1] != "write" || perm1.Permissions[2] != "execute" {
		t.Errorf("first permission spec permissions incorrect. got=%v", perm1.Permissions)
	}
	if len(perm1.Targets) != 1 || perm1.Targets[0] != "user" {
		t.Errorf("first permission spec targets incorrect. got=%v", perm1.Targets)
	}

	// Check second permission spec
	perm2 := downloadStmt.AllowPermissions[1]
	if len(perm2.Permissions) != 1 || perm2.Permissions[0] != "read" {
		t.Errorf("second permission spec permissions incorrect. got=%v", perm2.Permissions)
	}
	if len(perm2.Targets) != 2 || perm2.Targets[0] != "group" || perm2.Targets[1] != "others" {
		t.Errorf("second permission spec targets incorrect. got=%v", perm2.Targets)
	}
}

func TestParser_DownloadWithAuth(t *testing.T) {
	input := `version: 2.0

task "download_with_auth":
  download "https://example.com/private.zip" to "private.zip" with auth bearer "my-token-123"
  
  success "File downloaded with auth!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	downloadStmt, ok := task.Body[0].(*ast.DownloadStatement)
	if !ok {
		t.Fatalf("first statement should be DownloadStatement. got=%T", task.Body[0])
	}

	if downloadStmt.Auth["bearer"] != "my-token-123" {
		t.Errorf("bearer auth not 'my-token-123'. got=%q", downloadStmt.Auth["bearer"])
	}
}

func TestParser_DownloadWithHeaders(t *testing.T) {
	input := `version: 2.0

task "download_with_headers":
  download "https://example.com/file.json" to "file.json" with header "Accept: application/json" with header "User-Agent: drun/2.0"
  
  success "File downloaded with headers!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	downloadStmt, ok := task.Body[0].(*ast.DownloadStatement)
	if !ok {
		t.Fatalf("first statement should be DownloadStatement. got=%T", task.Body[0])
	}

	if downloadStmt.Headers["Accept"] != "application/json" {
		t.Errorf("Accept header not 'application/json'. got=%q", downloadStmt.Headers["Accept"])
	}

	if downloadStmt.Headers["User-Agent"] != "drun/2.0" {
		t.Errorf("User-Agent header not 'drun/2.0'. got=%q", downloadStmt.Headers["User-Agent"])
	}
}

func TestParser_DownloadWithTimeout(t *testing.T) {
	input := `version: 2.0

task "download_with_timeout":
  download "https://example.com/large.zip" to "large.zip" timeout "120s"
  
  success "File downloaded with timeout!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	downloadStmt, ok := task.Body[0].(*ast.DownloadStatement)
	if !ok {
		t.Fatalf("first statement should be DownloadStatement. got=%T", task.Body[0])
	}

	if downloadStmt.Options["timeout"] != "120s" {
		t.Errorf("timeout option not '120s'. got=%q", downloadStmt.Options["timeout"])
	}
}

func TestParser_DownloadExtractTo(t *testing.T) {
	input := `version: 2.0

task "download_and_extract":
  download "https://example.com/archive.zip" to "archive.zip" extract to "extracted/"
  
  success "Archive downloaded and extracted!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	downloadStmt, ok := task.Body[0].(*ast.DownloadStatement)
	if !ok {
		t.Fatalf("first statement should be DownloadStatement. got=%T", task.Body[0])
	}

	if downloadStmt.Path != "archive.zip" {
		t.Errorf("download path not 'archive.zip'. got=%q", downloadStmt.Path)
	}

	if downloadStmt.ExtractTo != "extracted/" {
		t.Errorf("extract to not 'extracted/'. got=%q", downloadStmt.ExtractTo)
	}

	if downloadStmt.RemoveArchive {
		t.Errorf("RemoveArchive should be false. got=%v", downloadStmt.RemoveArchive)
	}
}

func TestParser_DownloadExtractRemoveArchive(t *testing.T) {
	input := `version: 2.0

task "download_extract_remove":
  download "https://example.com/release.tar.gz" to "release.tar.gz" extract to "release/" remove archive
  
  success "Archive downloaded, extracted, and removed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	downloadStmt, ok := task.Body[0].(*ast.DownloadStatement)
	if !ok {
		t.Fatalf("first statement should be DownloadStatement. got=%T", task.Body[0])
	}

	if downloadStmt.ExtractTo != "release/" {
		t.Errorf("extract to not 'release/'. got=%q", downloadStmt.ExtractTo)
	}

	if !downloadStmt.RemoveArchive {
		t.Errorf("RemoveArchive should be true. got=%v", downloadStmt.RemoveArchive)
	}
}

func TestParser_DownloadCompleteFeatures(t *testing.T) {
	input := `version: 2.0

task "download_complete":
  download "https://api.example.com/releases/v1.0/app.tar.gz" to ".downloads/app.tar.gz" extract to "app/" remove archive allow overwrite timeout "60s" with auth bearer "token123" with header "Accept: application/octet-stream" allow permissions ["read","execute"] to ["user","group","others"]
  
  success "Complete download with all features!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	downloadStmt, ok := task.Body[0].(*ast.DownloadStatement)
	if !ok {
		t.Fatalf("first statement should be DownloadStatement. got=%T", task.Body[0])
	}

	// Check URL
	if downloadStmt.URL != "https://api.example.com/releases/v1.0/app.tar.gz" {
		t.Errorf("URL incorrect. got=%q", downloadStmt.URL)
	}

	// Check path
	if downloadStmt.Path != ".downloads/app.tar.gz" {
		t.Errorf("path incorrect. got=%q", downloadStmt.Path)
	}

	// Check extraction
	if downloadStmt.ExtractTo != "app/" {
		t.Errorf("extract to incorrect. got=%q", downloadStmt.ExtractTo)
	}

	if !downloadStmt.RemoveArchive {
		t.Errorf("RemoveArchive should be true. got=%v", downloadStmt.RemoveArchive)
	}

	// Check overwrite
	if !downloadStmt.AllowOverwrite {
		t.Errorf("AllowOverwrite should be true. got=%v", downloadStmt.AllowOverwrite)
	}

	// Check timeout
	if downloadStmt.Options["timeout"] != "60s" {
		t.Errorf("timeout option incorrect. got=%q", downloadStmt.Options["timeout"])
	}

	// Check auth
	if downloadStmt.Auth["bearer"] != "token123" {
		t.Errorf("bearer auth incorrect. got=%q", downloadStmt.Auth["bearer"])
	}

	// Check headers
	if downloadStmt.Headers["Accept"] != "application/octet-stream" {
		t.Errorf("Accept header incorrect. got=%q", downloadStmt.Headers["Accept"])
	}

	// Check permissions
	if len(downloadStmt.AllowPermissions) != 1 {
		t.Fatalf("should have 1 permission spec. got=%d", len(downloadStmt.AllowPermissions))
	}

	perm := downloadStmt.AllowPermissions[0]
	if len(perm.Permissions) != 2 || perm.Permissions[0] != "read" || perm.Permissions[1] != "execute" {
		t.Errorf("permissions incorrect. got=%v", perm.Permissions)
	}
	if len(perm.Targets) != 3 || perm.Targets[0] != "user" || perm.Targets[1] != "group" || perm.Targets[2] != "others" {
		t.Errorf("permission targets incorrect. got=%v", perm.Targets)
	}
}

func TestParser_DownloadWithVariableInterpolation(t *testing.T) {
	input := `version: 2.0

task "download_version":
  requires $version
  
  download "https://example.com/releases/{$version}/app.zip" to "downloads/app-{$version}.zip"
  
  success "Downloaded version {$version}!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	downloadStmt, ok := task.Body[0].(*ast.DownloadStatement)
	if !ok {
		t.Fatalf("first statement should be DownloadStatement. got=%T", task.Body[0])
	}

	if downloadStmt.URL != "https://example.com/releases/{$version}/app.zip" {
		t.Errorf("URL with variable incorrect. got=%q", downloadStmt.URL)
	}

	if downloadStmt.Path != "downloads/app-{$version}.zip" {
		t.Errorf("path with variable incorrect. got=%q", downloadStmt.Path)
	}
}

func TestParser_MultipleDownloads(t *testing.T) {
	input := `version: 2.0

task "download_multiple":
  download "https://example.com/file1.zip" to "file1.zip"
  download "https://example.com/file2.tar.gz" to "file2.tar.gz" extract to "file2/"
  download "https://example.com/file3.json" to "file3.json" allow overwrite
  
  success "All files downloaded!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	if len(task.Body) != 4 {
		t.Fatalf("task should have 4 statements. got=%d", len(task.Body))
	}

	// Check all three download statements
	for i := 0; i < 3; i++ {
		_, ok := task.Body[i].(*ast.DownloadStatement)
		if !ok {
			t.Fatalf("statement %d should be DownloadStatement. got=%T", i, task.Body[i])
		}
	}

	// Verify specific features
	stmt1 := task.Body[0].(*ast.DownloadStatement)
	if stmt1.URL != "https://example.com/file1.zip" {
		t.Errorf("first download URL incorrect. got=%q", stmt1.URL)
	}

	stmt2 := task.Body[1].(*ast.DownloadStatement)
	if stmt2.ExtractTo != "file2/" {
		t.Errorf("second download should have extract. got=%q", stmt2.ExtractTo)
	}

	stmt3 := task.Body[2].(*ast.DownloadStatement)
	if !stmt3.AllowOverwrite {
		t.Errorf("third download should allow overwrite. got=%v", stmt3.AllowOverwrite)
	}
}

