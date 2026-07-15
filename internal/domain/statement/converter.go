package statement

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/ast"
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
			Action:               s.Action,
			Command:              s.Command,
			Commands:             s.Commands,
			CaptureVar:           s.CaptureVar,
			Attached:             s.Attached,
			StreamOutput:         s.StreamOutput,
			IsMultiline:          s.IsMultiline,
			ServiceScoped:        s.ServiceScoped,
			ServiceName:          s.ServiceName,
			ServiceNameIsLiteral: s.ServiceNameIsLiteral,
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
			Operation:            s.Operation,
			Resource:             s.Resource,
			Name:                 s.Name,
			Options:              s.Options,
			ServiceScoped:        s.ServiceScoped,
			ServiceName:          s.ServiceName,
			ServiceNameIsLiteral: s.ServiceNameIsLiteral,
		}, nil

	case *ast.GitStatement:
		return &Git{
			Operation: s.Operation,
			Resource:  s.Resource,
			Name:      s.Name,
			Options:   s.Options,
		}, nil

	case *ast.GitQueryStatement:
		return &GitQuery{
			Result:         s.Result,
			Source:         s.Source,
			AccessMethod:   s.AccessMethod,
			TagPreset:      s.TagPreset,
			TagFormat:      s.TagFormat,
			TagPattern:     s.TagPattern,
			Series:         s.Series,
			VersionMatcher: s.VersionMatcher,
			OrderBy:        s.OrderBy,
			AllowFetch:     s.AllowFetch,
			CaptureVar:     s.CaptureVar,
		}, nil

	case *ast.GitEnsureVersionStatement:
		return &GitEnsureVersion{
			Candidate:           s.Candidate,
			CandidateIsVariable: s.CandidateIsVariable,
			Source:              s.Source,
			AccessMethod:        s.AccessMethod,
			TagPreset:           s.TagPreset,
			TagFormat:           s.TagFormat,
			TagPattern:          s.TagPattern,
			CaptureVar:          s.CaptureVar,
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
			Action:       s.Action,
			Target:       s.Target,
			Source:       s.Source,
			Content:      s.Content,
			IsDir:        s.IsDir,
			CaptureVar:   s.CaptureVar,
			Replacements: s.Replacements,
		}, nil

	case *ast.FileValueStatement:
		return &FileValue{
			Operation: s.Operation, Format: s.Format, Selector: s.Selector,
			Target: s.Target, CaptureVar: s.CaptureVar, Comparison: s.Comparison,
			Expected: s.Expected, Value: s.Value, MissingPolicy: s.MissingPolicy,
			ValueType: s.ValueType,
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
			VersionOp:     s.VersionOp,
			VersionValue:  s.VersionValue,
			CaptureVar:    s.CaptureVar,
			Body:          body,
			ElseBody:      elseBody,
		}, nil

	case *ast.UseSnippetStatement:
		return &UseSnippet{
			SnippetName: s.SnippetName,
		}, nil

	case *ast.TaskFromTemplateStatement:
		return &TaskFromTemplate{
			Name:         s.Name,
			TemplateName: s.TemplateName,
			Overrides:    s.Overrides,
		}, nil

	case *ast.ParameterStatement:
		// Parameters are handled during task setup, not execution
		// Return nil to skip them in the body
		return nil, nil

	case *ast.SecretStatement:
		var valueStr, defaultStr string
		if s.Value != nil {
			valueStr = s.Value.String()
		}
		if s.Default != nil {
			defaultStr = s.Default.String()
		}
		return &Secret{
			Operation: s.Operation,
			Key:       s.Key,
			Value:     valueStr,
			Namespace: s.Namespace,
			Pattern:   s.Pattern,
			Default:   defaultStr,
		}, nil

	case *ast.OrchestrationActionStatement:
		return &Orchestration{
			GroupName:      s.GroupName,
			Action:         s.Action,
			Options:        s.Options,
			ServiceFilters: s.ServiceFilters,
		}, nil

	case *ast.RequiresToolsStatement:
		var tools []ToolRequirement
		for _, astTool := range s.Tools {
			var constraints []VersionConstraint
			for _, astConstraint := range astTool.Constraints {
				constraints = append(constraints, VersionConstraint{
					Operator: astConstraint.Operator,
					Version:  astConstraint.Version,
				})
			}
			tools = append(tools, ToolRequirement{
				Name:          astTool.Name,
				Constraints:   constraints,
				AutoProvision: astTool.AutoProvision,
			})
		}
		return &RequiresTools{
			Tools: tools,
		}, nil

	case *ast.ChangeWorkdirStatement:
		return &ChangeWorkdir{
			Path: s.Path,
		}, nil

	case *ast.GitPolicyStatement:
		return &GitPolicy{
			DefaultBranches:      s.DefaultBranches,
			BranchPattern:        s.BranchPattern,
			BranchTypes:          s.BranchTypes,
			CommitPattern:        s.CommitPattern,
			ExtractIdentifier:    s.ExtractIdentifier,
			CommitMinLength:      s.CommitMinLength,
			CommitBans:           s.CommitBans,
			EnforceSignedCommits: s.EnforceSignedCommits,
		}, nil

	case *ast.GitValidateStatement:
		return &GitValidate{
			Target: s.Target,
			Value:  s.Value,
		}, nil

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
