package statement

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
)

// ToAST converts a domain statement back to an AST statement
// This is a temporary bridge to allow the domain layer to be decoupled
// while the execution layer still uses AST statements.
// This will be removed in Phase 4 when executors are updated to work with domain statements directly.
func ToAST(domainStmt Statement) (ast.Statement, error) {
	switch s := domainStmt.(type) {
	case *Action:
		return &ast.ActionStatement{
			Action:          s.ActionType,
			Message:         s.Message,
			LineBreakBefore: s.LineBreakBefore,
			LineBreakAfter:  s.LineBreakAfter,
		}, nil

	case *Shell:
		return &ast.ShellStatement{
			Action:       s.Action,
			Command:      s.Command,
			Commands:     s.Commands,
			CaptureVar:   s.CaptureVar,
			StreamOutput: s.StreamOutput,
			IsMultiline:  s.IsMultiline,
		}, nil

	case *Variable:
		// For variable, we need to parse the value string back to an expression
		// For now, we'll use a simple LiteralExpression
		var value ast.Expression
		if s.Value != "" {
			value = &ast.LiteralExpression{Value: s.Value}
		}
		return &ast.VariableStatement{
			Operation: s.Operation,
			Variable:  s.Name,
			Value:     value,
			Function:  s.Function,
			Arguments: s.Arguments,
		}, nil

	case *Conditional:
		body, err := ToASTList(s.Body)
		if err != nil {
			return nil, fmt.Errorf("converting conditional body: %w", err)
		}
		elseBody, err := ToASTList(s.ElseBody)
		if err != nil {
			return nil, fmt.Errorf("converting conditional else body: %w", err)
		}
		return &ast.ConditionalStatement{
			Type:      s.ConditionType,
			Condition: s.Condition,
			Body:      body,
			ElseBody:  elseBody,
		}, nil

	case *Loop:
		body, err := ToASTList(s.Body)
		if err != nil {
			return nil, fmt.Errorf("converting loop body: %w", err)
		}
		var filter *ast.FilterExpression
		if s.Filter != nil {
			filter = &ast.FilterExpression{
				Variable: s.Filter.Variable,
				Operator: s.Filter.Operator,
				Value:    s.Filter.Value,
			}
		}
		return &ast.LoopStatement{
			Type:       s.LoopType,
			Variable:   s.Variable,
			Iterable:   s.Iterable,
			RangeStart: s.RangeStart,
			RangeEnd:   s.RangeEnd,
			RangeStep:  s.RangeStep,
			Filter:     filter,
			Parallel:   s.Parallel,
			MaxWorkers: s.MaxWorkers,
			FailFast:   s.FailFast,
			Body:       body,
		}, nil

	case *Try:
		tryBody, err := ToASTList(s.TryBody)
		if err != nil {
			return nil, fmt.Errorf("converting try body: %w", err)
		}

		var catchClauses []ast.CatchClause
		for _, domainCatch := range s.CatchClauses {
			catchBody, err := ToASTList(domainCatch.Body)
			if err != nil {
				return nil, fmt.Errorf("converting catch body: %w", err)
			}
			catchClauses = append(catchClauses, ast.CatchClause{
				ErrorType: domainCatch.ErrorType,
				ErrorVar:  domainCatch.ErrorVar,
				Body:      catchBody,
			})
		}

		finallyBody, err := ToASTList(s.FinallyBody)
		if err != nil {
			return nil, fmt.Errorf("converting finally body: %w", err)
		}

		return &ast.TryStatement{
			TryBody:      tryBody,
			CatchClauses: catchClauses,
			FinallyBody:  finallyBody,
		}, nil

	case *Throw:
		return &ast.ThrowStatement{
			Action:  s.Action,
			Message: s.Message,
		}, nil

	case *Break:
		return &ast.BreakStatement{
			Condition: s.Condition,
		}, nil

	case *Continue:
		return &ast.ContinueStatement{
			Condition: s.Condition,
		}, nil

	case *TaskCall:
		return &ast.TaskCallStatement{
			TaskName:   s.TaskName,
			Parameters: s.Parameters,
		}, nil

	case *Docker:
		return &ast.DockerStatement{
			Operation: s.Operation,
			Resource:  s.Resource,
			Name:      s.Name,
			Options:   s.Options,
		}, nil

	case *Git:
		return &ast.GitStatement{
			Operation: s.Operation,
			Resource:  s.Resource,
			Name:      s.Name,
			Options:   s.Options,
		}, nil

	case *HTTP:
		return &ast.HTTPStatement{
			Method:  s.Method,
			URL:     s.URL,
			Headers: s.Headers,
			Body:    s.Body,
			Auth:    s.Auth,
			Options: s.Options,
		}, nil

	case *Download:
		var astPerms []ast.PermissionSpec
		for _, domainPerm := range s.AllowPermissions {
			astPerms = append(astPerms, ast.PermissionSpec{
				Permissions: domainPerm.Permissions,
				Targets:     domainPerm.Targets,
			})
		}
		return &ast.DownloadStatement{
			URL:              s.URL,
			Path:             s.Path,
			AllowOverwrite:   s.AllowOverwrite,
			AllowPermissions: astPerms,
			ExtractTo:        s.ExtractTo,
			RemoveArchive:    s.RemoveArchive,
			Headers:          s.Headers,
			Auth:             s.Auth,
			Options:          s.Options,
		}, nil

	case *Network:
		return &ast.NetworkStatement{
			Action:    s.Action,
			Target:    s.Target,
			Port:      s.Port,
			Options:   s.Options,
			Condition: s.Condition,
		}, nil

	case *File:
		return &ast.FileStatement{
			Action:     s.Action,
			Target:     s.Target,
			Source:     s.Source,
			Content:    s.Content,
			IsDir:      s.IsDir,
			CaptureVar: s.CaptureVar,
		}, nil

	case *Detection:
		body, err := ToASTList(s.Body)
		if err != nil {
			return nil, fmt.Errorf("converting detection body: %w", err)
		}
		elseBody, err := ToASTList(s.ElseBody)
		if err != nil {
			return nil, fmt.Errorf("converting detection else body: %w", err)
		}
		return &ast.DetectionStatement{
			Type:         s.DetectionType,
			Target:       s.Target,
			Alternatives: s.Alternatives,
			Condition:    s.Condition,
			Value:        s.Value,
			CaptureVar:   s.CaptureVar,
			Body:         body,
			ElseBody:     elseBody,
		}, nil

	case *UseSnippet:
		return &ast.UseSnippetStatement{
			SnippetName: s.SnippetName,
		}, nil

	default:
		return nil, fmt.Errorf("unknown domain statement type: %T", domainStmt)
	}
}

// ToASTList converts a list of domain statements to AST statements
func ToASTList(domainStmts []Statement) ([]ast.Statement, error) {
	var result []ast.Statement
	for _, domainStmt := range domainStmts {
		astStmt, err := ToAST(domainStmt)
		if err != nil {
			return nil, err
		}
		result = append(result, astStmt)
	}
	return result, nil
}
