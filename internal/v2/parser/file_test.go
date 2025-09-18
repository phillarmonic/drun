package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/v2/ast"
	"github.com/phillarmonic/drun/internal/v2/lexer"
)

func TestParser_FileOperations(t *testing.T) {
	input := `version: 2.0

task "file operations":
  create file "test.txt"
  create dir "testdir"
  copy "source.txt" to "dest.txt"
  move "old.txt" to "new.txt"
  delete file "unwanted.txt"
  delete dir "olddir"
  read file "config.json" as config
  write "Hello World" to file "greeting.txt"
  append "New line" to file "log.txt"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if len(task.Body) != 9 {
		t.Fatalf("Expected 9 statements, got %d", len(task.Body))
	}

	// Test each file operation
	expectedOps := []struct {
		action     string
		target     string
		source     string
		content    string
		isDir      bool
		captureVar string
	}{
		{"create", "test.txt", "", "", false, ""},
		{"create", "testdir", "", "", true, ""},
		{"copy", "dest.txt", "source.txt", "", false, ""},
		{"move", "new.txt", "old.txt", "", false, ""},
		{"delete", "unwanted.txt", "", "", false, ""},
		{"delete", "olddir", "", "", true, ""},
		{"read", "config.json", "", "", false, "config"},
		{"write", "greeting.txt", "", "Hello World", false, ""},
		{"append", "log.txt", "", "New line", false, ""},
	}

	for i, expected := range expectedOps {
		stmt := task.Body[i]
		fileStmt, ok := stmt.(*ast.FileStatement)
		if !ok {
			t.Errorf("Statement %d: expected FileStatement, got %T", i, stmt)
			continue
		}

		if fileStmt.Action != expected.action {
			t.Errorf("Statement %d: expected action %s, got %s", i, expected.action, fileStmt.Action)
		}

		if fileStmt.Target != expected.target {
			t.Errorf("Statement %d: expected target %s, got %s", i, expected.target, fileStmt.Target)
		}

		if fileStmt.Source != expected.source {
			t.Errorf("Statement %d: expected source %s, got %s", i, expected.source, fileStmt.Source)
		}

		if fileStmt.Content != expected.content {
			t.Errorf("Statement %d: expected content %s, got %s", i, expected.content, fileStmt.Content)
		}

		if fileStmt.IsDir != expected.isDir {
			t.Errorf("Statement %d: expected isDir %t, got %t", i, expected.isDir, fileStmt.IsDir)
		}

		if fileStmt.CaptureVar != expected.captureVar {
			t.Errorf("Statement %d: expected captureVar %s, got %s", i, expected.captureVar, fileStmt.CaptureVar)
		}
	}
}

func TestParser_CreateFileOperation(t *testing.T) {
	input := `version: 2.0

task "create test":
  create file "test.txt"
  create dir "testdir"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	task := program.Tasks[0]
	if len(task.Body) != 2 {
		t.Fatalf("Expected 2 statements, got %d", len(task.Body))
	}

	// Test create file
	fileStmt := task.Body[0].(*ast.FileStatement)
	if fileStmt.Action != "create" {
		t.Errorf("Expected action 'create', got %s", fileStmt.Action)
	}
	if fileStmt.Target != "test.txt" {
		t.Errorf("Expected target 'test.txt', got %s", fileStmt.Target)
	}
	if fileStmt.IsDir {
		t.Errorf("Expected IsDir false for file creation")
	}

	// Test create dir
	dirStmt := task.Body[1].(*ast.FileStatement)
	if dirStmt.Action != "create" {
		t.Errorf("Expected action 'create', got %s", dirStmt.Action)
	}
	if dirStmt.Target != "testdir" {
		t.Errorf("Expected target 'testdir', got %s", dirStmt.Target)
	}
	if !dirStmt.IsDir {
		t.Errorf("Expected IsDir true for directory creation")
	}
}

func TestParser_CopyMoveOperations(t *testing.T) {
	input := `version: 2.0

task "copy move test":
  copy "source.txt" to "dest.txt"
  move "old.txt" to "new.txt"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	task := program.Tasks[0]
	if len(task.Body) != 2 {
		t.Fatalf("Expected 2 statements, got %d", len(task.Body))
	}

	// Test copy
	copyStmt := task.Body[0].(*ast.FileStatement)
	if copyStmt.Action != "copy" {
		t.Errorf("Expected action 'copy', got %s", copyStmt.Action)
	}
	if copyStmt.Source != "source.txt" {
		t.Errorf("Expected source 'source.txt', got %s", copyStmt.Source)
	}
	if copyStmt.Target != "dest.txt" {
		t.Errorf("Expected target 'dest.txt', got %s", copyStmt.Target)
	}

	// Test move
	moveStmt := task.Body[1].(*ast.FileStatement)
	if moveStmt.Action != "move" {
		t.Errorf("Expected action 'move', got %s", moveStmt.Action)
	}
	if moveStmt.Source != "old.txt" {
		t.Errorf("Expected source 'old.txt', got %s", moveStmt.Source)
	}
	if moveStmt.Target != "new.txt" {
		t.Errorf("Expected target 'new.txt', got %s", moveStmt.Target)
	}
}

func TestParser_ReadWriteOperations(t *testing.T) {
	input := `version: 2.0

task "read write test":
  read file "config.json" as config
  write "Hello World" to file "greeting.txt"
  append "New line" to file "log.txt"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	task := program.Tasks[0]
	if len(task.Body) != 3 {
		t.Fatalf("Expected 3 statements, got %d", len(task.Body))
	}

	// Test read with capture
	readStmt := task.Body[0].(*ast.FileStatement)
	if readStmt.Action != "read" {
		t.Errorf("Expected action 'read', got %s", readStmt.Action)
	}
	if readStmt.Target != "config.json" {
		t.Errorf("Expected target 'config.json', got %s", readStmt.Target)
	}
	if readStmt.CaptureVar != "config" {
		t.Errorf("Expected captureVar 'config', got %s", readStmt.CaptureVar)
	}

	// Test write
	writeStmt := task.Body[1].(*ast.FileStatement)
	if writeStmt.Action != "write" {
		t.Errorf("Expected action 'write', got %s", writeStmt.Action)
	}
	if writeStmt.Content != "Hello World" {
		t.Errorf("Expected content 'Hello World', got %s", writeStmt.Content)
	}
	if writeStmt.Target != "greeting.txt" {
		t.Errorf("Expected target 'greeting.txt', got %s", writeStmt.Target)
	}

	// Test append
	appendStmt := task.Body[2].(*ast.FileStatement)
	if appendStmt.Action != "append" {
		t.Errorf("Expected action 'append', got %s", appendStmt.Action)
	}
	if appendStmt.Content != "New line" {
		t.Errorf("Expected content 'New line', got %s", appendStmt.Content)
	}
	if appendStmt.Target != "log.txt" {
		t.Errorf("Expected target 'log.txt', got %s", appendStmt.Target)
	}
}
