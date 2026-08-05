package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func TestParser_CreatePathAtConditionalBranchBoundary(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		isDir bool
	}{
		{name: "directory", line: `create directory "~/Library/Application Support/example/cache"`, isDir: true},
		{name: "folder", line: `create folder "~/Library/Application Support/example/cache"`, isDir: true},
		{name: "file", line: `create file "$HOME/example/cache/config.json"`, isDir: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `version: 2.0

@platform("mac")
task "install" means "Installs the plugin for macOS":
    step "Installing for MacOS"
    info "Will install to: ~/Library/Audio/Plug-Ins/VST3"
    if folder "~/Library/Audio/Plug-Ins/VST3" not exists:
        warning "The folder ~/Library/Audio/Plug-Ins/VST3 doesn't exist."
        ` + tt.line + `
    else:
        info "Folder exists."
    success "Install check complete."
`

			l := lexer.NewLexer(input)
			p := NewParser(l)
			program := p.ParseProgram()
			checkParserErrors(t, p)

			if len(program.Tasks) != 1 {
				t.Fatalf("expected 1 task, got %d", len(program.Tasks))
			}
			task := program.Tasks[0]
			if len(task.Body) != 4 {
				t.Fatalf("expected 4 task statements, got %d", len(task.Body))
			}
			conditional, ok := task.Body[2].(*ast.ConditionalStatement)
			if !ok {
				t.Fatalf("expected third statement to be *ast.ConditionalStatement, got %T", task.Body[2])
			}
			if len(conditional.Body) != 2 {
				t.Fatalf("expected 2 if-body statements, got %d", len(conditional.Body))
			}
			created, ok := conditional.Body[1].(*ast.FileStatement)
			if !ok {
				t.Fatalf("expected final if-body statement to be *ast.FileStatement, got %T", conditional.Body[1])
			}
			if created.Action != "create" || created.IsDir != tt.isDir {
				t.Fatalf("unexpected create statement: action=%q isDir=%t", created.Action, created.IsDir)
			}
			if len(conditional.ElseBody) != 1 {
				t.Fatalf("expected 1 else-body statement, got %d", len(conditional.ElseBody))
			}
		})
	}
}
