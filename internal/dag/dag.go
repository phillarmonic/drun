package dag

import (
	"fmt"
	"sort"

	"github.com/phillarmonic/drun/internal/model"
)

// Builder builds execution plans from recipe dependencies
type Builder struct {
	spec *model.Spec
}

// NewBuilder creates a new DAG builder
func NewBuilder(spec *model.Spec) *Builder {
	return &Builder{spec: spec}
}

// Build builds an execution plan for the given target recipe
func (b *Builder) Build(target string, ctx *model.ExecutionContext) (*model.ExecutionPlan, error) {
	// Check if target recipe exists
	if _, exists := b.spec.Recipes[target]; !exists {
		return nil, fmt.Errorf("recipe '%s' not found", target)
	}

	// Build dependency graph
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var nodes []model.PlanNode
	var edges [][2]int
	nodeIndex := make(map[string]int)

	if err := b.buildGraph(target, ctx, visited, visiting, &nodes, &edges, nodeIndex); err != nil {
		return nil, err
	}

	// Topological sort to determine execution order
	sorted, err := b.topologicalSort(nodes, edges)
	if err != nil {
		return nil, err
	}

	// Reorder nodes according to topological sort
	var sortedNodes []model.PlanNode
	for _, idx := range sorted {
		sortedNodes = append(sortedNodes, nodes[idx])
	}

	return &model.ExecutionPlan{
		Nodes: sortedNodes,
		Edges: edges,
	}, nil
}

// buildGraph recursively builds the dependency graph
func (b *Builder) buildGraph(
	recipeName string,
	ctx *model.ExecutionContext,
	visited, visiting map[string]bool,
	nodes *[]model.PlanNode,
	edges *[][2]int,
	nodeIndex map[string]int,
) error {
	if visiting[recipeName] {
		return fmt.Errorf("circular dependency detected involving recipe '%s'", recipeName)
	}

	if visited[recipeName] {
		return nil
	}

	visiting[recipeName] = true

	recipe, exists := b.spec.Recipes[recipeName]
	if !exists {
		return fmt.Errorf("recipe '%s' not found", recipeName)
	}

	// Process dependencies first
	for _, dep := range recipe.Deps {
		if err := b.buildGraph(dep, ctx, visited, visiting, nodes, edges, nodeIndex); err != nil {
			return err
		}

		// Add edge from dependency to current recipe
		depIndex, depExists := nodeIndex[dep]
		if !depExists {
			return fmt.Errorf("dependency '%s' not found in node index", dep)
		}

		currentIndex := len(*nodes) // Current recipe will be at this index
		*edges = append(*edges, [2]int{depIndex, currentIndex})
	}

	// Create recipe-specific context by merging recipe env vars
	recipeCtx := &model.ExecutionContext{
		Vars:        make(map[string]any),
		Env:         make(map[string]string),
		Flags:       ctx.Flags,
		Positionals: ctx.Positionals,
		OS:          ctx.OS,
		Arch:        ctx.Arch,
		Hostname:    ctx.Hostname,
	}

	// Copy base context
	for k, v := range ctx.Vars {
		recipeCtx.Vars[k] = v
	}
	for k, v := range ctx.Env {
		recipeCtx.Env[k] = v
	}

	// Add recipe-specific environment variables
	for k, v := range recipe.Env {
		recipeCtx.Env[k] = v
	}

	// Add current recipe as a node
	node := model.PlanNode{
		ID:        recipeName,
		Recipe:    &recipe,
		Context:   recipeCtx,
		Step:      recipe.Run,
		DependsOn: recipe.Deps,
	}

	*nodes = append(*nodes, node)
	nodeIndex[recipeName] = len(*nodes) - 1

	visiting[recipeName] = false
	visited[recipeName] = true

	return nil
}

// topologicalSort performs topological sorting using Kahn's algorithm
func (b *Builder) topologicalSort(nodes []model.PlanNode, edges [][2]int) ([]int, error) {
	n := len(nodes)
	if n == 0 {
		return []int{}, nil
	}

	// Build adjacency list and in-degree count
	adj := make([][]int, n)
	inDegree := make([]int, n)

	for _, edge := range edges {
		from, to := edge[0], edge[1]
		if from >= n || to >= n || from < 0 || to < 0 {
			return nil, fmt.Errorf("invalid edge: [%d, %d]", from, to)
		}
		adj[from] = append(adj[from], to)
		inDegree[to]++
	}

	// Find all nodes with no incoming edges
	var queue []int
	for i := 0; i < n; i++ {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	var result []int

	for len(queue) > 0 {
		// Remove a node with no incoming edges
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each neighbor of the current node
		for _, neighbor := range adj[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycles
	if len(result) != n {
		return nil, fmt.Errorf("circular dependency detected in recipe graph")
	}

	return result, nil
}

// GetParallelGroups returns groups of nodes that can be executed in parallel
func (b *Builder) GetParallelGroups(plan *model.ExecutionPlan) [][]int {
	n := len(plan.Nodes)
	if n == 0 {
		return [][]int{}
	}

	// Build adjacency list for dependencies
	adj := make([][]int, n)
	for _, edge := range plan.Edges {
		from, to := edge[0], edge[1]
		adj[from] = append(adj[from], to)
	}

	// Calculate the level of each node (longest path from a root)
	levels := make([]int, n)
	visited := make([]bool, n)

	var dfs func(int) int
	dfs = func(node int) int {
		if visited[node] {
			return levels[node]
		}
		visited[node] = true

		maxLevel := 0
		for _, dep := range adj[node] {
			level := dfs(dep)
			if level > maxLevel {
				maxLevel = level
			}
		}
		levels[node] = maxLevel + 1
		return levels[node]
	}

	// Calculate levels for all nodes
	for i := 0; i < n; i++ {
		if !visited[i] {
			dfs(i)
		}
	}

	// Group nodes by level
	levelGroups := make(map[int][]int)
	for i, level := range levels {
		levelGroups[level] = append(levelGroups[level], i)
	}

	// Convert to sorted slice of groups
	var sortedLevels []int
	for level := range levelGroups {
		sortedLevels = append(sortedLevels, level)
	}
	sort.Ints(sortedLevels)

	var groups [][]int
	for _, level := range sortedLevels {
		groups = append(groups, levelGroups[level])
	}

	return groups
}
