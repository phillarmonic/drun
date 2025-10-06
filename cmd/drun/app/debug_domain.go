package app

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/debug"
	"github.com/phillarmonic/drun/internal/domain/parameter"
	"github.com/phillarmonic/drun/internal/domain/task"
)

// Domain: Domain Layer Debugging
// This file contains logic for debugging the domain layer (task registry, dependencies, params)

// debugDomainLayer initializes domain services and shows their state
func debugDomainLayer(program *ast.Program, currentFile string) error {
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
		domainTask := task.NewTask(astTask, "", currentFile)
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

	return nil
}
