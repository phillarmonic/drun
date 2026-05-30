package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func parseStringForWorkdirTest(t *testing.T, input string) *ast.Program {
	t.Helper()
	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)
	if program == nil {
		t.Fatal("ParseProgram() returned nil")
	}
	return program
}

// TestUseWorkdirParsing verifies that `use workdir "path"` is correctly parsed
// into a ChangeWorkdirStatement in the AST.
func TestUseWorkdirParsing(t *testing.T) {
	input := `version: 2.0

task "build-frontend" means "Builds the dev frontend":
    use workdir "frontend"
    run "npm run build:dev"
`

	program := parseStringForWorkdirTest(t, input)

	if len(program.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if task.Name != "build-frontend" {
		t.Errorf("Expected task name 'build-frontend', got %q", task.Name)
	}

	// The task body should contain: ChangeWorkdirStatement, ShellStatement
	if len(task.Body) != 2 {
		t.Fatalf("Expected 2 body statements, got %d", len(task.Body))
	}

	// First statement: use workdir
	workdirStmt, ok := task.Body[0].(*ast.ChangeWorkdirStatement)
	if !ok {
		t.Fatalf("Expected first body statement to be *ast.ChangeWorkdirStatement, got %T", task.Body[0])
	}
	if workdirStmt.Path != "frontend" {
		t.Errorf("Expected workdir path 'frontend', got %q", workdirStmt.Path)
	}

	// Second statement: run command
	shellStmt, ok := task.Body[1].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected second body statement to be *ast.ShellStatement, got %T", task.Body[1])
	}
	if shellStmt.Command != "npm run build:dev" {
		t.Errorf("Expected shell command 'npm run build:dev', got %q", shellStmt.Command)
	}
}

// TestUseWorkdirWithVariable verifies that variable interpolation syntax is preserved in the AST path.
func TestUseWorkdirWithVariable(t *testing.T) {
	input := `version: 2.0

task "build":
    let $subdir = "frontend"
    use workdir "{$subdir}"
    run "npm run build"
`

	program := parseStringForWorkdirTest(t, input)

	task := program.Tasks[0]
	// body: variable(let), workdir, shell
	if len(task.Body) != 3 {
		t.Fatalf("Expected 3 body statements, got %d", len(task.Body))
	}

	workdirStmt, ok := task.Body[1].(*ast.ChangeWorkdirStatement)
	if !ok {
		t.Fatalf("Expected second body statement to be *ast.ChangeWorkdirStatement, got %T", task.Body[1])
	}
	if workdirStmt.Path != "{$subdir}" {
		t.Errorf("Expected path '{$subdir}', got %q", workdirStmt.Path)
	}
}

// TestUseWorkdirMultiple verifies that multiple `use workdir` statements in one task parse correctly.
func TestUseWorkdirMultiple(t *testing.T) {
	input := `version: 2.0

task "multi":
    use workdir "first"
    run "pwd"
    use workdir "second"
    run "pwd"
`

	program := parseStringForWorkdirTest(t, input)

	task := program.Tasks[0]
	if len(task.Body) != 4 {
		t.Fatalf("Expected 4 body statements, got %d", len(task.Body))
	}

	for _, idx := range []int{0, 2} {
		if _, ok := task.Body[idx].(*ast.ChangeWorkdirStatement); !ok {
			t.Errorf("Expected statement %d to be *ast.ChangeWorkdirStatement, got %T", idx, task.Body[idx])
		}
	}
}

// TestUseWorkdirDoesNotBreakSnippet verifies that `use snippet` still works
// correctly alongside `use workdir`.
func TestUseWorkdirDoesNotBreakSnippet(t *testing.T) {
	input := `version: 2.0

project "test":
    snippet "greet":
        info "Hello!"

task "mixed":
    use workdir "frontend"
    use snippet "greet"
    run "npm run build"
`

	program := parseStringForWorkdirTest(t, input)

	task := program.Tasks[0]
	if len(task.Body) != 3 {
		t.Fatalf("Expected 3 body statements, got %d", len(task.Body))
	}

	if _, ok := task.Body[0].(*ast.ChangeWorkdirStatement); !ok {
		t.Errorf("Expected first statement to be ChangeWorkdirStatement, got %T", task.Body[0])
	}
	if _, ok := task.Body[1].(*ast.UseSnippetStatement); !ok {
		t.Errorf("Expected second statement to be UseSnippetStatement, got %T", task.Body[1])
	}
}

// TestUseWorkdirASTString verifies the String() representation of ChangeWorkdirStatement.
func TestUseWorkdirASTString(t *testing.T) {
	stmt := &ast.ChangeWorkdirStatement{
		Path: "frontend",
	}
	expected := `use workdir "frontend"`
	if got := stmt.String(); got != expected {
		t.Errorf("Expected String() = %q, got %q", expected, got)
	}
}
