package app

import (
	"fmt"
	"os"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/debug"
	"github.com/phillarmonic/drun/internal/domain/parameter"
	"github.com/phillarmonic/drun/internal/domain/statement"
	"github.com/phillarmonic/drun/internal/domain/task"
	"github.com/phillarmonic/drun/internal/engine/planner"
)

// Domain: Domain Layer Debugging
// This file contains logic for debugging the domain layer (task registry, dependencies, params)

// DebugOptions contains options for debugging
type DebugOptions struct {
	ShowPlan       bool
	ExportGraphviz string
	ExportMermaid  string
	ExportJSON     string
}

// debugDomainLayer initializes domain services and shows their state
func debugDomainLayer(program *ast.Program, currentFile string, opts DebugOptions) error {
	if program == nil {
		fmt.Println("=== DOMAIN LAYER DEBUG ===")
		fmt.Println("Program is nil - cannot debug domain layer")
		fmt.Println()
		return nil
	}

	// Initialize domain services (same as engine does)
	taskReg := task.NewRegistry()
	paramValidator := parameter.NewValidator()
	depResolver := task.NewDependencyResolver(taskReg)

	// Register all tasks
	for _, astTask := range program.Tasks {
		domainTask, err := task.NewTask(astTask, "", currentFile)
		if err != nil {
			return fmt.Errorf("converting task %s: %w", astTask.Name, err)
		}
		if err := taskReg.Register(domainTask); err != nil {
			return fmt.Errorf("task registration failed: %v", err)
		}
	}

	// Prepare debug info
	debugInfo := debug.DomainDebugInfo{
		TaskRegistry:       taskReg,
		DependencyResolver: depResolver,
		ParameterValidator: paramValidator,
	}

	// Show domain layer information
	debug.DebugDomain(debugInfo)

	// Show dependency resolution for each task
	fmt.Println("🔍 Dependency Resolution Analysis:")
	fmt.Println()
	tasks := taskReg.List()
	if len(tasks) == 0 {
		fmt.Println("  No tasks registered")
	} else {
		for _, domainTask := range tasks {
			fullName := domainTask.FullName()

			fmt.Printf("  Task: %s\n", fullName)
			if domainTask.HasDependencies() {
				// Try to resolve dependencies
				resolved, err := depResolver.Resolve(fullName)
				if err != nil {
					fmt.Printf("    ❌ Resolution failed: %v\n", err)
				} else {
					fmt.Printf("    ✅ Execution order (%d tasks):\n", len(resolved))
					for i, dep := range resolved {
						marker := "→"
						if i == len(resolved)-1 {
							marker = "🎯"
						}
						fmt.Printf("       %s %s\n", marker, dep.FullName())
					}

					// Check for parallel opportunities
					groups, err := depResolver.GetParallelGroups(domainTask)
					if err == nil && len(groups) > 1 {
						fmt.Printf("    🚀 Parallel execution opportunities: %d groups\n", len(groups))
						for i, group := range groups {
							if len(group) > 1 {
								fmt.Printf("       Group %d (parallel): %d tasks\n", i+1, len(group))
								for _, parallelTask := range group {
									fmt.Printf("         • %s\n", parallelTask.Name)
								}
							}
						}
					}
				}
			} else {
				fmt.Println("    No dependencies")
			}
			fmt.Println()
		}
	}

	// If plan visualization is requested, generate execution plans for each task
	if opts.ShowPlan || opts.ExportGraphviz != "" || opts.ExportMermaid != "" || opts.ExportJSON != "" {
		fmt.Println("📋 Execution Plan Visualization:")
		fmt.Println()

		plannerInstance := planner.NewPlanner(taskReg, depResolver)

		// Generate plans for tasks with dependencies or for all if explicitly requested
		tasksToVisualize := tasks
		if len(tasksToVisualize) == 0 {
			fmt.Println("  No tasks to visualize")
		}

		for _, domainTask := range tasksToVisualize {
			if !domainTask.HasDependencies() && len(tasksToVisualize) > 1 {
				continue // Skip simple tasks unless it's the only one
			}

			fullName := domainTask.FullName()
			fmt.Printf("  Creating execution plan for: %s\n", fullName)

			// Create project context for planning
			var projectCtx *planner.ProjectContext
			if program.Project != nil {
				projectCtx = &planner.ProjectContext{
					Name:    program.Project.Name,
					Version: program.Project.Version,
				}
				// Convert lifecycle hooks if present
				for _, setting := range program.Project.Settings {
					if hook, ok := setting.(*ast.LifecycleHook); ok {
						hookBody, err := statement.FromASTList(hook.Body)
						if err != nil {
							continue
						}
						switch hook.Type {
						case "setup":
							projectCtx.SetupHooks = append(projectCtx.SetupHooks, hookBody...)
						case "teardown":
							projectCtx.TeardownHooks = append(projectCtx.TeardownHooks, hookBody...)
						case "before":
							projectCtx.BeforeHooks = append(projectCtx.BeforeHooks, hookBody...)
						case "after":
							projectCtx.AfterHooks = append(projectCtx.AfterHooks, hookBody...)
						}
					}
				}
			}

			// Generate execution plan
			plan, err := plannerInstance.Plan(fullName, program, projectCtx)
			if err != nil {
				fmt.Printf("    ❌ Failed to create plan: %v\n", err)
				continue
			}

			// Show plan summary
			if opts.ShowPlan {
				debug.DebugExecutionPlan(plan)
			}

			// Convert plan to debug format
			planInfo := convertPlanToDebugInfo(plan)

			// Export formats
			if opts.ExportGraphviz != "" {
				dot := debug.ExportExecutionPlanGraphviz(planInfo)
				filename := fmt.Sprintf("%s-%s.dot", opts.ExportGraphviz, fullName)
				if err := os.WriteFile(filename, []byte(dot), 0644); err != nil {
					fmt.Printf("    ❌ Failed to write Graphviz file: %v\n", err)
				} else {
					fmt.Printf("    ✅ Graphviz exported to: %s\n", filename)
					fmt.Printf("       Render with: dot -Tpng %s -o %s.png\n", filename, filename)
				}
			}

			if opts.ExportMermaid != "" {
				mermaid := debug.ExportExecutionPlanMermaid(planInfo)
				filename := fmt.Sprintf("%s-%s.mmd", opts.ExportMermaid, fullName)
				if err := os.WriteFile(filename, []byte(mermaid), 0644); err != nil {
					fmt.Printf("    ❌ Failed to write Mermaid file: %v\n", err)
				} else {
					fmt.Printf("    ✅ Mermaid exported to: %s\n", filename)
				}
			}

			if opts.ExportJSON != "" {
				jsonStr, err := debug.ExportExecutionPlanJSON(planInfo)
				if err != nil {
					fmt.Printf("    ❌ Failed to export JSON: %v\n", err)
				} else {
					filename := fmt.Sprintf("%s-%s.json", opts.ExportJSON, fullName)
					if err := os.WriteFile(filename, []byte(jsonStr), 0644); err != nil {
						fmt.Printf("    ❌ Failed to write JSON file: %v\n", err)
					} else {
						fmt.Printf("    ✅ JSON exported to: %s\n", filename)
					}
				}
			}

			fmt.Println()
		}
	}

	return nil
}

// convertPlanToDebugInfo converts an execution plan to debug info format
func convertPlanToDebugInfo(plan *planner.ExecutionPlan) debug.ExecutionPlanInfo {
	planInfo := debug.ExecutionPlanInfo{
		TargetTask:     plan.TargetTask,
		ExecutionOrder: plan.ExecutionOrder,
		Tasks:          make(map[string]debug.TaskInfo),
		ProjectName:    plan.ProjectName,
		ProjectVersion: plan.ProjectVersion,
		Namespaces:     plan.GetNamespaces(),
		TaskCount:      len(plan.Tasks),
	}

	// Convert hook info
	if plan.Hooks != nil {
		planInfo.Hooks = &debug.HookInfo{
			SetupCount:    len(plan.Hooks.SetupHooks),
			TeardownCount: len(plan.Hooks.TeardownHooks),
			BeforeCount:   len(plan.Hooks.BeforeHooks),
			AfterCount:    len(plan.Hooks.AfterHooks),
		}
	}

	// Convert task info
	for name, taskPlan := range plan.Tasks {
		params := make([]debug.ParameterInfo, len(taskPlan.Parameters))
		for i, p := range taskPlan.Parameters {
			params[i] = debug.ParameterInfo{
				Name:       p.Name,
				Type:       p.Type,
				Required:   p.Required,
				HasDefault: p.HasDefault,
				DataType:   p.DataType,
			}
		}

		// Extract dependencies from task
		deps := make([]string, 0)
		// Find this task in execution order and mark all previous tasks as potential dependencies
		for i, orderName := range plan.ExecutionOrder {
			if orderName == name && i > 0 {
				// Previous tasks in execution order are dependencies
				deps = append(deps, plan.ExecutionOrder[:i]...)
				break
			}
		}

		planInfo.Tasks[name] = debug.TaskInfo{
			Name:           taskPlan.Name,
			Description:    taskPlan.Description,
			Namespace:      taskPlan.Namespace,
			Source:         taskPlan.Source,
			Parameters:     params,
			Dependencies:   deps,
			StatementCount: len(taskPlan.Body),
		}
	}

	return planInfo
}
