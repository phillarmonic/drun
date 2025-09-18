package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/v2/ast"
	"github.com/phillarmonic/drun/internal/v2/lexer"
)

func TestParser_GitCloneRepository(t *testing.T) {
	input := `version: 2.0

task "clone_repo":
  git clone repository "https://github.com/user/repo.git" to "local-dir"
  
  success "Repository cloned successfully!"`

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

	// Check Git statement
	gitStmt, ok := task.Body[0].(*ast.GitStatement)
	if !ok {
		t.Fatalf("first statement should be GitStatement. got=%T", task.Body[0])
	}

	if gitStmt.Operation != "clone" {
		t.Errorf("git operation not 'clone'. got=%q", gitStmt.Operation)
	}

	if gitStmt.Resource != "repository" {
		t.Errorf("git resource not 'repository'. got=%q", gitStmt.Resource)
	}

	if gitStmt.Name != "https://github.com/user/repo.git" {
		t.Errorf("git repository URL not 'https://github.com/user/repo.git'. got=%q", gitStmt.Name)
	}

	if gitStmt.Options["to"] != "local-dir" {
		t.Errorf("git 'to' option not 'local-dir'. got=%q", gitStmt.Options["to"])
	}
}

func TestParser_GitInitRepository(t *testing.T) {
	input := `version: 2.0

task "init_repo":
  git init repository in "project-dir"
  
  success "Repository initialized!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	gitStmt, ok := task.Body[0].(*ast.GitStatement)
	if !ok {
		t.Fatalf("first statement should be GitStatement. got=%T", task.Body[0])
	}

	if gitStmt.Operation != "init" {
		t.Errorf("git operation not 'init'. got=%q", gitStmt.Operation)
	}

	if gitStmt.Resource != "repository" {
		t.Errorf("git resource not 'repository'. got=%q", gitStmt.Resource)
	}

	if gitStmt.Options["in"] != "project-dir" {
		t.Errorf("git 'in' option not 'project-dir'. got=%q", gitStmt.Options["in"])
	}
}

func TestParser_GitAddFiles(t *testing.T) {
	input := `version: 2.0

task "stage_changes":
  git add files "src/*.go"
  
  success "Files staged!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	gitStmt, ok := task.Body[0].(*ast.GitStatement)
	if !ok {
		t.Fatalf("first statement should be GitStatement. got=%T", task.Body[0])
	}

	if gitStmt.Operation != "add" {
		t.Errorf("git operation not 'add'. got=%q", gitStmt.Operation)
	}

	if gitStmt.Resource != "files" {
		t.Errorf("git resource not 'files'. got=%q", gitStmt.Resource)
	}

	if gitStmt.Name != "src/*.go" {
		t.Errorf("git file pattern not 'src/*.go'. got=%q", gitStmt.Name)
	}
}

func TestParser_GitCommitChanges(t *testing.T) {
	input := `version: 2.0

task "commit_work":
  git commit changes with message "Add new feature"
  
  success "Changes committed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	gitStmt, ok := task.Body[0].(*ast.GitStatement)
	if !ok {
		t.Fatalf("first statement should be GitStatement. got=%T", task.Body[0])
	}

	if gitStmt.Operation != "commit" {
		t.Errorf("git operation not 'commit'. got=%q", gitStmt.Operation)
	}

	if gitStmt.Resource != "changes" {
		t.Errorf("git resource not 'changes'. got=%q", gitStmt.Resource)
	}

	if gitStmt.Options["message"] != "Add new feature" {
		t.Errorf("git 'message' option not 'Add new feature'. got=%q", gitStmt.Options["message"])
	}
}

func TestParser_GitCommitAllChanges(t *testing.T) {
	input := `version: 2.0

task "commit_all":
  git commit all changes with message "Update documentation"
  
  success "All changes committed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	gitStmt, ok := task.Body[0].(*ast.GitStatement)
	if !ok {
		t.Fatalf("first statement should be GitStatement. got=%T", task.Body[0])
	}

	if gitStmt.Operation != "commit" {
		t.Errorf("git operation not 'commit'. got=%q", gitStmt.Operation)
	}

	if gitStmt.Resource != "changes" {
		t.Errorf("git resource not 'changes'. got=%q", gitStmt.Resource)
	}

	if gitStmt.Options["all"] != "true" {
		t.Errorf("git 'all' option not 'true'. got=%q", gitStmt.Options["all"])
	}

	if gitStmt.Options["message"] != "Update documentation" {
		t.Errorf("git 'message' option not 'Update documentation'. got=%q", gitStmt.Options["message"])
	}
}

func TestParser_GitPushToRemote(t *testing.T) {
	input := `version: 2.0

task "push_changes":
  git push to remote "origin" branch "main"
  
  success "Changes pushed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	gitStmt, ok := task.Body[0].(*ast.GitStatement)
	if !ok {
		t.Fatalf("first statement should be GitStatement. got=%T", task.Body[0])
	}

	if gitStmt.Operation != "push" {
		t.Errorf("git operation not 'push'. got=%q", gitStmt.Operation)
	}

	if gitStmt.Options["remote"] != "origin" {
		t.Errorf("git 'remote' option not 'origin'. got=%q", gitStmt.Options["remote"])
	}

	if gitStmt.Options["branch"] != "main" {
		t.Errorf("git 'branch' option not 'main'. got=%q", gitStmt.Options["branch"])
	}
}

func TestParser_GitStatus(t *testing.T) {
	input := `version: 2.0

task "check_status":
  git status
  
  success "Status checked!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	gitStmt, ok := task.Body[0].(*ast.GitStatement)
	if !ok {
		t.Fatalf("first statement should be GitStatement. got=%T", task.Body[0])
	}

	if gitStmt.Operation != "status" {
		t.Errorf("git operation not 'status'. got=%q", gitStmt.Operation)
	}
}

func TestParser_GitShowCurrentBranch(t *testing.T) {
	input := `version: 2.0

task "current_branch":
  git show current branch
  
  success "Current branch shown!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	gitStmt, ok := task.Body[0].(*ast.GitStatement)
	if !ok {
		t.Fatalf("first statement should be GitStatement. got=%T", task.Body[0])
	}

	if gitStmt.Operation != "show" {
		t.Errorf("git operation not 'show'. got=%q", gitStmt.Operation)
	}

	if gitStmt.Resource != "branch" {
		t.Errorf("git resource not 'branch'. got=%q", gitStmt.Resource)
	}

	if gitStmt.Options["current"] != "true" {
		t.Errorf("git 'current' option not 'true'. got=%q", gitStmt.Options["current"])
	}
}

func TestParser_GitMultipleOperations(t *testing.T) {
	input := `version: 2.0

task "git_workflow":
  git add files "."
  git commit changes with message "Update code"
  git push to remote "origin" branch "main"
  
  success "Git workflow completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	if len(task.Body) != 4 {
		t.Fatalf("task should have 4 statements. got=%d", len(task.Body))
	}

	// Check all three Git statements
	operations := []string{"add", "commit", "push"}
	for i, expectedOp := range operations {
		gitStmt, ok := task.Body[i].(*ast.GitStatement)
		if !ok {
			t.Fatalf("statement %d should be GitStatement. got=%T", i, task.Body[i])
		}

		if gitStmt.Operation != expectedOp {
			t.Errorf("git operation %d not '%s'. got=%q", i, expectedOp, gitStmt.Operation)
		}
	}
}
