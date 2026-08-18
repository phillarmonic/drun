package app

import (
	"github.com/phillarmonic/drun/v2/internal/ast"
)

// ProgramUsesOpenURL reports whether any task in the program contains an
// open url statement at any nesting depth. This is used before execution to
// decide whether a trust prompt is needed.
func ProgramUsesOpenURL(program *ast.Program) bool {
	for _, task := range program.Tasks {
		if bodyContainsOpen(task.Body) {
			return true
		}
	}
	for _, tmpl := range program.Templates {
		if bodyContainsOpen(tmpl.Body) {
			return true
		}
	}
	return false
}

func bodyContainsOpen(stmts []ast.Statement) bool {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.OpenStatement:
			return true
		case *ast.ConditionalStatement:
			if bodyContainsOpen(s.Body) || bodyContainsOpen(s.ElseBody) {
				return true
			}
		case *ast.LoopStatement:
			if bodyContainsOpen(s.Body) {
				return true
			}
		case *ast.TryStatement:
			if bodyContainsOpen(s.TryBody) || bodyContainsOpen(s.FinallyBody) {
				return true
			}
			for _, c := range s.CatchClauses {
				if bodyContainsOpen(c.Body) {
					return true
				}
			}
		}
	}
	return false
}
