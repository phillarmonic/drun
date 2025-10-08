package debug

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExecutionPlanInfo represents a serializable execution plan for debugging
type ExecutionPlanInfo struct {
	TargetTask     string              `json:"target_task"`
	ExecutionOrder []string            `json:"execution_order"`
	Tasks          map[string]TaskInfo `json:"tasks"`
	Hooks          *HookInfo           `json:"hooks,omitempty"`
	ProjectName    string              `json:"project_name,omitempty"`
	ProjectVersion string              `json:"project_version,omitempty"`
	Namespaces     []string            `json:"namespaces,omitempty"`
	TaskCount      int                 `json:"task_count"`
}

// TaskInfo represents a task in the execution plan
type TaskInfo struct {
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	Namespace    string         `json:"namespace,omitempty"`
	Source       string         `json:"source,omitempty"`
	Parameters   []ParameterInfo `json:"parameters,omitempty"`
	Dependencies []string       `json:"dependencies,omitempty"`
	StatementCount int          `json:"statement_count"`
}

// ParameterInfo represents parameter metadata
type ParameterInfo struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Required   bool     `json:"required"`
	HasDefault bool     `json:"has_default"`
	DataType   string   `json:"data_type,omitempty"`
}

// HookInfo represents lifecycle hooks in the plan
type HookInfo struct {
	SetupCount    int `json:"setup_count"`
	TeardownCount int `json:"teardown_count"`
	BeforeCount   int `json:"before_count"`
	AfterCount    int `json:"after_count"`
}

// DebugExecutionPlan prints detailed execution plan information
func DebugExecutionPlan(plan interface{}) {
	fmt.Println("=== EXECUTION PLAN DEBUG ===")
	fmt.Println()

	// Use type assertion to extract plan information
	if p, ok := plan.(interface {
		GetTargetTask() string
		GetExecutionOrder() []string
		GetTaskCount() int
		GetProjectName() string
		GetProjectVersion() string
		GetNamespaces() []string
	}); ok {
		fmt.Println("📊 Plan Overview:")
		fmt.Printf("  Target Task: %s\n", p.GetTargetTask())
		fmt.Printf("  Total Tasks: %d\n", p.GetTaskCount())
		fmt.Printf("  Project: %s", p.GetProjectName())
		if version := p.GetProjectVersion(); version != "" {
			fmt.Printf(" v%s", version)
		}
		fmt.Println()
		
		namespaces := p.GetNamespaces()
		if len(namespaces) > 0 {
			fmt.Printf("  Namespaces: %s\n", strings.Join(namespaces, ", "))
		}
		fmt.Println()

		fmt.Println("🔄 Execution Order:")
		order := p.GetExecutionOrder()
		for i, taskName := range order {
			marker := "  →"
			if i == len(order)-1 {
				marker = "  🎯"
			}
			fmt.Printf("%s %s\n", marker, taskName)
		}
		fmt.Println()
	}

	fmt.Println("=== END EXECUTION PLAN DEBUG ===")
	fmt.Println()
}

// ExportExecutionPlanJSON exports the execution plan as JSON
func ExportExecutionPlanJSON(planInfo ExecutionPlanInfo) (string, error) {
	jsonData, err := json.MarshalIndent(planInfo, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan to JSON: %w", err)
	}
	return string(jsonData), nil
}

// ExportExecutionPlanGraphviz exports the execution plan as Graphviz DOT format
func ExportExecutionPlanGraphviz(planInfo ExecutionPlanInfo) string {
	var b strings.Builder
	
	// Start digraph
	b.WriteString("digraph ExecutionPlan {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=box, style=rounded];\n")
	b.WriteString("  \n")
	
	// Add graph metadata
	if planInfo.ProjectName != "" {
		label := planInfo.ProjectName
		if planInfo.ProjectVersion != "" {
			label += " v" + planInfo.ProjectVersion
		}
		b.WriteString(fmt.Sprintf("  label=\"%s\";\n", escapeGraphviz(label)))
		b.WriteString("  labelloc=t;\n")
		b.WriteString("  fontsize=16;\n")
		b.WriteString("  \n")
	}
	
	// Node definitions with colors
	b.WriteString("  // Task nodes\n")
	for _, taskName := range planInfo.ExecutionOrder {
		taskInfo, exists := planInfo.Tasks[taskName]
		if !exists {
			continue
		}
		
		// Determine node color based on task type
		color := "lightblue"
		if taskName == planInfo.TargetTask {
			color = "lightgreen"
		} else if taskInfo.Namespace != "" {
			color = "lightyellow"
		}
		
		// Build label with description
		label := taskName
		if taskInfo.Description != "" {
			label += "\\n" + escapeGraphviz(taskInfo.Description)
		}
		if len(taskInfo.Parameters) > 0 {
			label += fmt.Sprintf("\\n(%d params)", len(taskInfo.Parameters))
		}
		
		b.WriteString(fmt.Sprintf("  \"%s\" [fillcolor=%s, style=\"rounded,filled\", label=\"%s\"];\n", 
			taskName, color, label))
	}
	b.WriteString("  \n")
	
	// Dependency edges
	b.WriteString("  // Dependencies\n")
	seen := make(map[string]bool)
	for i := 0; i < len(planInfo.ExecutionOrder)-1; i++ {
		from := planInfo.ExecutionOrder[i]
		to := planInfo.ExecutionOrder[i+1]
		
		// Check if this is a real dependency or just execution order
		taskInfo := planInfo.Tasks[to]
		isDep := false
		for _, dep := range taskInfo.Dependencies {
			if dep == from {
				isDep = true
				break
			}
		}
		
		edgeKey := from + "->" + to
		if !seen[edgeKey] {
			style := "solid"
			if !isDep {
				style = "dashed"
			}
			b.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [style=%s];\n", from, to, style))
			seen[edgeKey] = true
		}
	}
	b.WriteString("  \n")
	
	// Add legend
	b.WriteString("  // Legend\n")
	b.WriteString("  subgraph cluster_legend {\n")
	b.WriteString("    label=\"Legend\";\n")
	b.WriteString("    style=dashed;\n")
	b.WriteString("    \"Target Task\" [fillcolor=lightgreen, style=\"rounded,filled\"];\n")
	b.WriteString("    \"Regular Task\" [fillcolor=lightblue, style=\"rounded,filled\"];\n")
	b.WriteString("    \"Namespaced Task\" [fillcolor=lightyellow, style=\"rounded,filled\"];\n")
	b.WriteString("    \"Target Task\" -> \"Regular Task\" [label=\"dependency\", style=solid];\n")
	b.WriteString("    \"Regular Task\" -> \"Namespaced Task\" [label=\"execution order\", style=dashed];\n")
	b.WriteString("  }\n")
	
	b.WriteString("}\n")
	
	return b.String()
}

// ExportExecutionPlanMermaid exports the execution plan as Mermaid diagram
func ExportExecutionPlanMermaid(planInfo ExecutionPlanInfo) string {
	var b strings.Builder
	
	b.WriteString("```mermaid\n")
	b.WriteString("graph LR\n")
	
	// Add title
	if planInfo.ProjectName != "" {
		title := planInfo.ProjectName
		if planInfo.ProjectVersion != "" {
			title += " v" + planInfo.ProjectVersion
		}
		b.WriteString(fmt.Sprintf("  title[%s]\n", escapeMermaid(title)))
		b.WriteString("  style title fill:#f9f,stroke:#333,stroke-width:2px\n")
	}
	
	// Node definitions
	for _, taskName := range planInfo.ExecutionOrder {
		taskInfo, exists := planInfo.Tasks[taskName]
		if !exists {
			continue
		}
		
		// Create node ID (mermaid doesn't like dots)
		nodeID := strings.ReplaceAll(taskName, ".", "_")
		
		// Build label
		label := taskName
		if taskInfo.Description != "" {
			label += "<br/>" + escapeMermaid(taskInfo.Description)
		}
		
		// Determine node style
		if taskName == planInfo.TargetTask {
			b.WriteString(fmt.Sprintf("  %s[\"%s\"]\n", nodeID, label))
			b.WriteString(fmt.Sprintf("  style %s fill:#90EE90\n", nodeID))
		} else if taskInfo.Namespace != "" {
			b.WriteString(fmt.Sprintf("  %s[\"%s\"]\n", nodeID, label))
			b.WriteString(fmt.Sprintf("  style %s fill:#FFFFE0\n", nodeID))
		} else {
			b.WriteString(fmt.Sprintf("  %s[\"%s\"]\n", nodeID, label))
			b.WriteString(fmt.Sprintf("  style %s fill:#ADD8E6\n", nodeID))
		}
	}
	
	// Edges
	for i := 0; i < len(planInfo.ExecutionOrder)-1; i++ {
		fromID := strings.ReplaceAll(planInfo.ExecutionOrder[i], ".", "_")
		toID := strings.ReplaceAll(planInfo.ExecutionOrder[i+1], ".", "_")
		b.WriteString(fmt.Sprintf("  %s --> %s\n", fromID, toID))
	}
	
	b.WriteString("```\n")
	
	return b.String()
}

// escapeGraphviz escapes special characters for Graphviz DOT format
func escapeGraphviz(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// escapeMermaid escapes special characters for Mermaid format
func escapeMermaid(s string) string {
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
