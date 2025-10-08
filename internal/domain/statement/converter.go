package statement

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
)

// FromAST converts an AST statement to a domain statement
func FromAST(astStmt ast.Statement) (Statement, error) {
	switch s := astStmt.(type) {
	case *ast.ActionStatement:
		return &Action{
			ActionType:      s.Action,
			Message:         s.Message,
			LineBreakBefore: s.LineBreakBefore,
			LineBreakAfter:  s.LineBreakAfter,
		}, nil

	case *ast.ShellStatement:
		return &Shell{
			Action:       s.Action,
			Command:      s.Command,
			Commands:     s.Commands,
			CaptureVar:   s.CaptureVar,
			StreamOutput: s.StreamOutput,
			IsMultiline:  s.IsMultiline,
		}, nil

	case *ast.VariableStatement:
		var valueStr string
		if s.Value != nil {
			valueStr = s.Value.String()
		}
		return &Variable{
			Operation: s.Operation,
			Name:      s.Variable,
			Value:     valueStr,
			Function:  s.Function,
			Arguments: s.Arguments,
		}, nil

	case *ast.ConditionalStatement:
		body, err := FromASTList(s.Body)
		if err != nil {
			return nil, fmt.Errorf("converting conditional body: %w", err)
		}
		elseBody, err := FromASTList(s.ElseBody)
		if err != nil {
			return nil, fmt.Errorf("converting conditional else body: %w", err)
		}
		return &Conditional{
			ConditionType: s.Type,
			Condition:     s.Condition,
			Body:          body,
			ElseBody:      elseBody,
		}, nil

	case *ast.LoopStatement:
		body, err := FromASTList(s.Body)
		if err != nil {
			return nil, fmt.Errorf("converting loop body: %w", err)
		}
		var filter *Filter
		if s.Filter != nil {
			filter = &Filter{
				Variable: s.Filter.Variable,
				Operator: s.Filter.Operator,
				Value:    s.Filter.Value,
			}
		}
		return &Loop{
			LoopType:   s.Type,
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

	case *ast.TryStatement:
		tryBody, err := FromASTList(s.TryBody)
		if err != nil {
			return nil, fmt.Errorf("converting try body: %w", err)
		}

		var catchClauses []CatchClause
		for _, astCatch := range s.CatchClauses {
			catchBody, err := FromASTList(astCatch.Body)
			if err != nil {
				return nil, fmt.Errorf("converting catch body: %w", err)
			}
			catchClauses = append(catchClauses, CatchClause{
				ErrorType: astCatch.ErrorType,
				ErrorVar:  astCatch.ErrorVar,
				Body:      catchBody,
			})
		}

		finallyBody, err := FromASTList(s.FinallyBody)
		if err != nil {
			return nil, fmt.Errorf("converting finally body: %w", err)
		}

		return &Try{
			TryBody:      tryBody,
			CatchClauses: catchClauses,
			FinallyBody:  finallyBody,
		}, nil

	case *ast.ThrowStatement:
		return &Throw{
			Action:  s.Action,
			Message: s.Message,
		}, nil

	case *ast.BreakStatement:
		return &Break{
			Condition: s.Condition,
		}, nil

	case *ast.ContinueStatement:
		return &Continue{
			Condition: s.Condition,
		}, nil

	case *ast.TaskCallStatement:
		return &TaskCall{
			TaskName:   s.TaskName,
			Parameters: s.Parameters,
		}, nil

	case *ast.DockerStatement:
		return &Docker{
			Operation: s.Operation,
			Resource:  s.Resource,
			Name:      s.Name,
			Options:   s.Options,
		}, nil

	case *ast.GitStatement:
		return &Git{
			Operation: s.Operation,
			Resource:  s.Resource,
			Name:      s.Name,
			Options:   s.Options,
		}, nil

	case *ast.HTTPStatement:
		return &HTTP{
			Method:  s.Method,
			URL:     s.URL,
			Headers: s.Headers,
			Body:    s.Body,
			Auth:    s.Auth,
			Options: s.Options,
		}, nil

	case *ast.DownloadStatement:
		var permSpecs []PermissionSpec
		for _, astPerm := range s.AllowPermissions {
			permSpecs = append(permSpecs, PermissionSpec{
				Permissions: astPerm.Permissions,
				Targets:     astPerm.Targets,
			})
		}
		return &Download{
			URL:              s.URL,
			Path:             s.Path,
			AllowOverwrite:   s.AllowOverwrite,
			AllowPermissions: permSpecs,
			ExtractTo:        s.ExtractTo,
			RemoveArchive:    s.RemoveArchive,
			Headers:          s.Headers,
			Auth:             s.Auth,
			Options:          s.Options,
		}, nil

	case *ast.NetworkStatement:
		return &Network{
			Action:    s.Action,
			Target:    s.Target,
			Port:      s.Port,
			Options:   s.Options,
			Condition: s.Condition,
		}, nil

	case *ast.FileStatement:
		return &File{
			Action:     s.Action,
			Target:     s.Target,
			Source:     s.Source,
			Content:    s.Content,
			IsDir:      s.IsDir,
			CaptureVar: s.CaptureVar,
		}, nil

	case *ast.DetectionStatement:
		body, err := FromASTList(s.Body)
		if err != nil {
			return nil, fmt.Errorf("converting detection body: %w", err)
		}
		elseBody, err := FromASTList(s.ElseBody)
		if err != nil {
			return nil, fmt.Errorf("converting detection else body: %w", err)
		}
		return &Detection{
			DetectionType: s.Type,
			Target:        s.Target,
			Alternatives:  s.Alternatives,
			Condition:     s.Condition,
			Value:         s.Value,
			CaptureVar:    s.CaptureVar,
			Body:          body,
			ElseBody:      elseBody,
		}, nil

	case *ast.UseSnippetStatement:
		return &UseSnippet{
			SnippetName: s.SnippetName,
		}, nil

	case *ast.ParameterStatement:
		// Parameters are handled during task setup, not execution
		// Return nil to skip them in the body
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown AST statement type: %T", astStmt)
	}
}

// FromASTList converts a list of AST statements to domain statements
func FromASTList(astStmts []ast.Statement) ([]Statement, error) {
	var result []Statement
	for _, astStmt := range astStmts {
		domainStmt, err := FromAST(astStmt)
		if err != nil {
			return nil, err
		}
		// Skip nil statements (e.g., parameter declarations)
		if domainStmt != nil {
			result = append(result, domainStmt)
		}
	}
	return result, nil
}

// ToAST converts a domain statement back to an AST statement
// Note: This is still needed for nested structures (loops, conditionals, etc.)
// until executors are fully refactored to work with domain statements.
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
		for _, perm := range s.AllowPermissions {
			astPerms = append(astPerms, ast.PermissionSpec{
				Permissions: perm.Permissions,
				Targets:     perm.Targets,
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
	case *TaskCall:
		return &ast.TaskCallStatement{
			TaskName:   s.TaskName,
			Parameters: s.Parameters,
		}, nil
	default:
		return nil, fmt.Errorf("unknown domain statement type: %T", domainStmt)
	}
}

// ToASTList converts a list of domain statements to a list of AST statements
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
