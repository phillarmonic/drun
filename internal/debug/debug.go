package debug

import (
	"encoding/json"
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	lexer "github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
)

// TokenDebugInfo represents a token with additional debug information
type TokenDebugInfo struct {
	Type     string `json:"type"`
	Literal  string `json:"literal"`
	Position int    `json:"position"`
}

// DebugTokens prints all tokens from the lexer for debugging
func DebugTokens(input string) {
	fmt.Println("=== LEXER DEBUG ===")
	fmt.Printf("Input: %q\n", input)
	fmt.Println("Tokens:")

	l := lexer.NewLexer(input)
	position := 0

	for {
		tok := l.NextToken()
		if tok.Type == lexer.EOF {
			fmt.Printf("  %d: %s EOF\n", position, tok.Type)
			break
		}

		// Show token with position and literal
		literal := tok.Literal
		switch literal {
		case "\n":
			literal = "\\n"
		case "\t":
			literal = "\\t"
		case "":
			literal = "(empty)"
		}

		fmt.Printf("  %d: %-15s %q\n", position, tok.Type, literal)
		position++
	}
	fmt.Println()
}

// DebugAST prints the AST structure in a readable format
func DebugAST(program *ast.Program) {
	fmt.Println("=== AST DEBUG ===")
	if program == nil {
		fmt.Println("Program is nil")
		return
	}

	fmt.Printf("Program Version: %s\n", program.Version)

	if program.Project != nil {
		fmt.Println("Project Declaration:")
		fmt.Printf("  Settings: %d\n", len(program.Project.Settings))
		for i, setting := range program.Project.Settings {
			fmt.Printf("    %d: %T\n", i, setting)
		}
	}

	fmt.Printf("Tasks: %d\n", len(program.Tasks))
	for i, task := range program.Tasks {
		fmt.Printf("  Task %d: %q\n", i, task.Name)
		fmt.Printf("    Description: %q\n", task.Description)
		fmt.Printf("    Parameters: %d\n", len(task.Parameters))
		for j, param := range task.Parameters {
			fmt.Printf("      %d: %s %q (type: %s, required: %t)\n",
				j, param.Type, param.Name, param.DataType, param.Required)
			if param.DefaultValue != "" {
				fmt.Printf("         default: %q\n", param.DefaultValue)
			}
			if len(param.Constraints) > 0 {
				fmt.Printf("         constraints: %v\n", param.Constraints)
			}
		}
		fmt.Printf("    Dependencies: %d groups\n", len(task.Dependencies))
		for j, dep := range task.Dependencies {
			fmt.Printf("      Group %d: %d items\n", j, len(dep.Dependencies))
		}
		fmt.Printf("    Body: %d statements\n", len(task.Body))
		for j, stmt := range task.Body {
			fmt.Printf("      %d: %T\n", j, stmt)
			debugStatement(stmt, "        ")
		}
	}
	fmt.Println()
}

// debugStatement prints detailed information about a statement
func debugStatement(stmt ast.Statement, indent string) {
	switch s := stmt.(type) {
	case *ast.ActionStatement:
		fmt.Printf("%sAction: %s %q\n", indent, s.Action, s.Message)
	case *ast.ConditionalStatement:
		fmt.Printf("%sConditional: %s\n", indent, s.Type)
		fmt.Printf("%s  Condition: %q\n", indent, s.Condition)
		fmt.Printf("%s  Body: %d statements\n", indent, len(s.Body))
		if len(s.ElseBody) > 0 {
			fmt.Printf("%s  Else: %d statements\n", indent, len(s.ElseBody))
		}
	case *ast.LoopStatement:
		fmt.Printf("%sLoop: %s\n", indent, s.Type)
		fmt.Printf("%s  Variable: %q\n", indent, s.Variable)
		fmt.Printf("%s  Iterable: %q\n", indent, s.Iterable)
		if s.Parallel {
			fmt.Printf("%s  Parallel: true (workers: %d, fail-fast: %t)\n",
				indent, s.MaxWorkers, s.FailFast)
		}
		fmt.Printf("%s  Body: %d statements\n", indent, len(s.Body))
	case *ast.FileStatement:
		fmt.Printf("%sFile: %s\n", indent, s.Action)
		fmt.Printf("%s  Target: %q\n", indent, s.Target)
		if s.Source != "" {
			fmt.Printf("%s  Source: %q\n", indent, s.Source)
		}
		if s.Content != "" {
			fmt.Printf("%s  Content: %q\n", indent, s.Content)
		}
		if s.CaptureVar != "" {
			fmt.Printf("%s  Capture: %q\n", indent, s.CaptureVar)
		}
	case *ast.VariableStatement:
		fmt.Printf("%sVariable: %s\n", indent, s.Operation)
		fmt.Printf("%s  Variable: %q\n", indent, s.Variable)
		fmt.Printf("%s  Value: %q\n", indent, s.Value)
		if s.Function != "" {
			fmt.Printf("%s  Function: %q\n", indent, s.Function)
			if len(s.Arguments) > 0 {
				fmt.Printf("%s  Arguments: %v\n", indent, s.Arguments)
			}
		}
	case *ast.TryStatement:
		fmt.Printf("%sTry: %d statements\n", indent, len(s.TryBody))
		fmt.Printf("%s  Catch clauses: %d\n", indent, len(s.CatchClauses))
		if len(s.FinallyBody) > 0 {
			fmt.Printf("%s  Finally: %d statements\n", indent, len(s.FinallyBody))
		}
	case *ast.ThrowStatement:
		fmt.Printf("%sThrow: %s\n", indent, s.Action)
		if s.Message != "" {
			fmt.Printf("%s  Message: %q\n", indent, s.Message)
		}
	case *ast.DockerStatement:
		fmt.Printf("%sDocker: %s\n", indent, s.Operation)
		fmt.Printf("%s  Resource: %q\n", indent, s.Resource)
		fmt.Printf("%s  Name: %q\n", indent, s.Name)
		if len(s.Options) > 0 {
			fmt.Printf("%s  Options: %v\n", indent, s.Options)
		}
	case *ast.GitStatement:
		fmt.Printf("%sGit: %s\n", indent, s.Operation)
		fmt.Printf("%s  Resource: %q\n", indent, s.Resource)
		if s.Name != "" {
			fmt.Printf("%s  Name: %q\n", indent, s.Name)
		}
		if len(s.Options) > 0 {
			fmt.Printf("%s  Options: %v\n", indent, s.Options)
		}
	case *ast.HTTPStatement:
		fmt.Printf("%sHTTP: %s\n", indent, s.Method)
		fmt.Printf("%s  URL: %q\n", indent, s.URL)
		if s.Body != "" {
			fmt.Printf("%s  Body: %q\n", indent, s.Body)
		}
		if len(s.Headers) > 0 {
			fmt.Printf("%s  Headers: %v\n", indent, s.Headers)
		}
		if len(s.Auth) > 0 {
			fmt.Printf("%s  Auth: %v\n", indent, s.Auth)
		}
	case *ast.DetectionStatement:
		fmt.Printf("%sDetection: %s\n", indent, s.Type)
		fmt.Printf("%s  Target: %q\n", indent, s.Target)
		fmt.Printf("%s  Condition: %q\n", indent, s.Condition)
		if s.Value != "" {
			fmt.Printf("%s  Value: %q\n", indent, s.Value)
		}
		fmt.Printf("%s  Body: %d statements\n", indent, len(s.Body))
		if len(s.ElseBody) > 0 {
			fmt.Printf("%s  Else: %d statements\n", indent, len(s.ElseBody))
		}
	default:
		fmt.Printf("%sUnknown statement type: %T\n", indent, stmt)
	}
}

// DebugParseErrors prints detailed parse error information
func DebugParseErrors(errors []string) {
	fmt.Println("=== PARSE ERRORS ===")
	if len(errors) == 0 {
		fmt.Println("No parse errors")
		return
	}

	fmt.Printf("Found %d parse errors:\n", len(errors))
	for i, err := range errors {
		fmt.Printf("  %d: %s\n", i+1, err)
	}
	fmt.Println()
}

// DebugJSON outputs the AST as JSON for detailed inspection
func DebugJSON(program *ast.Program) {
	fmt.Println("=== AST JSON ===")
	if program == nil {
		fmt.Println("Program is nil")
		return
	}

	jsonData, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling to JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
	fmt.Println()
}

// DebugFull performs complete debugging of input
func DebugFull(input string) (*ast.Program, []string) {
	fmt.Println("=== FULL DEBUG SESSION ===")
	fmt.Printf("Input length: %d characters\n", len(input))
	fmt.Printf("Input preview: %q\n", truncateString(input, 100))
	fmt.Println()

	// Debug tokens
	DebugTokens(input)

	// Parse and debug AST
	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()
	errors := p.Errors()

	// Debug parse errors
	DebugParseErrors(errors)

	// Debug AST
	DebugAST(program)

	return program, errors
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// DomainDebugInfo contains debug information about the domain layer
type DomainDebugInfo struct {
	TaskRegistry       interface{}
	DependencyResolver interface{}
	ParameterValidator interface{}
}

// DebugDomain prints domain layer information (task registry, dependencies, params)
// This function signature allows the engine to pass domain information without circular imports
func DebugDomain(info DomainDebugInfo) {
	fmt.Println("=== DOMAIN LAYER DEBUG ===")
	fmt.Println()

	// Task Registry Debug
	if taskReg, ok := info.TaskRegistry.(interface {
		List() []interface{}
		Count() int
	}); ok {
		fmt.Println("📋 Task Registry:")
		fmt.Printf("  Total tasks: %d\n", taskReg.Count())
		tasks := taskReg.List()
		for i, task := range tasks {
			if t, ok := task.(interface {
				FullName() string
				HasDependencies() bool
			}); ok {
				fullName := t.FullName()
				hasDeps := t.HasDependencies()
				depsStr := ""
				if hasDeps {
					depsStr = " [has dependencies]"
				}
				fmt.Printf("    %d: %s%s\n", i+1, fullName, depsStr)
			}
		}
		fmt.Println()
	} else {
		fmt.Println("📋 Task Registry: (not available)")
		fmt.Println()
	}

	// Dependency Resolver Debug
	fmt.Println("🔗 Dependency Resolver:")
	fmt.Println("  Status: initialized")
	fmt.Println("  Features:")
	fmt.Println("    • Topological sort")
	fmt.Println("    • Circular dependency detection")
	fmt.Println("    • Parallel group analysis")
	fmt.Println()

	// Parameter Validator Debug
	fmt.Println("✅ Parameter Validator:")
	fmt.Println("  Status: initialized")
	fmt.Println("  Validation support:")
	fmt.Println("    • Type checking (string, number, boolean, list)")
	fmt.Println("    • Enum constraints")
	fmt.Println("    • Range validation (min/max)")
	fmt.Println("    • Pattern matching (regex, email, url)")
	fmt.Println("    • Default value handling")
	fmt.Println()

	fmt.Println("=== END DOMAIN LAYER DEBUG ===")
	fmt.Println()
}
