package dag

import (
	"fmt"
	"sort"

	"github.com/phillarmonic/drun/internal/v1/model"
)

// Builder builds execution plans from recipe dependencies
type Builder struct {
	spec          *model.Spec
	planCache     map[string]*model.ExecutionPlan // Cache plans by target recipe
	estimatedSize int                             // Cached estimation for pre-allocation
}

// NewBuilder creates a new DAG builder
func NewBuilder(spec *model.Spec) *Builder {
	estimatedSize := len(spec.Recipes)
	if estimatedSize > 100 {
		estimatedSize = 100 // Cap estimation for very large specs
	}

	return &Builder{
		spec:          spec,
		planCache:     make(map[string]*model.ExecutionPlan),
		estimatedSize: estimatedSize,
	}
}

// Build builds an execution plan for the given target recipe
func (b *Builder) Build(target string, ctx *model.ExecutionContext) (*model.ExecutionPlan, error) {
	// Check if target recipe exists
	if _, exists := b.spec.Recipes[target]; !exists {
		return nil, fmt.Errorf("recipe '%s' not found", target)
	}

	// Build dependency graph with pre-allocated slices using cached estimation
	visited := make(map[string]bool, b.estimatedSize)
	visiting := make(map[string]bool, b.estimatedSize)
	nodes := make([]model.PlanNode, 0, b.estimatedSize)
	edges := make([][2]int, 0, b.estimatedSize*2) // Estimate 2 edges per node
	nodeIndex := make(map[string]int, b.estimatedSize)

	if err := b.buildGraph(target, ctx, visited, visiting, &nodes, &edges, nodeIndex); err != nil {
		return nil, err
	}

	// Topological sort to determine execution order
	sorted, err := b.topologicalSort(nodes, edges)
	if err != nil {
		return nil, err
	}

	// Reorder nodes according to topological sort
	sortedNodes := make([]model.PlanNode, len(sorted))
	for i, idx := range sorted {
		sortedNodes[i] = nodes[idx]
	}

	// Compute execution levels for parallel execution
	levels := b.computeExecutionLevels(sortedNodes, edges)

	// Debug output (can be enabled for troubleshooting)
	// fmt.Printf("DEBUG: DAG has %d nodes, %d edges\n", len(sortedNodes), len(edges))

	return &model.ExecutionPlan{
		Nodes:  sortedNodes,
		Edges:  edges,
		Levels: levels,
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
	}

	// Create recipe-specific context by merging recipe env vars
	// Pre-allocate maps with estimated sizes
	varsSize := len(ctx.Vars) + len(recipe.Env)
	envSize := len(ctx.Env) + len(recipe.Env)

	recipeCtx := &model.ExecutionContext{
		Vars:        make(map[string]any, varsSize),
		Env:         make(map[string]string, envSize),
		Secrets:     ctx.Secrets,     // Share reference (read-only)
		Flags:       ctx.Flags,       // Share reference (read-only)
		Positionals: ctx.Positionals, // Share reference (read-only)
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

	// Handle matrix expansion
	if len(recipe.Matrix) > 0 {
		// Expand matrix into multiple nodes
		matrixCombinations := b.generateMatrixCombinations(recipe.Matrix)

		for i, combination := range matrixCombinations {
			// Create matrix-specific context
			matrixCtx := &model.ExecutionContext{
				Vars:        make(map[string]any, len(recipeCtx.Vars)+len(combination)),
				Env:         make(map[string]string, len(recipeCtx.Env)),
				Secrets:     recipeCtx.Secrets, // Share secrets across matrix
				Flags:       recipeCtx.Flags,
				Positionals: recipeCtx.Positionals,
				OS:          recipeCtx.OS,
				Arch:        recipeCtx.Arch,
				Hostname:    recipeCtx.Hostname,
			}

			// Copy base context
			for k, v := range recipeCtx.Vars {
				matrixCtx.Vars[k] = v
			}
			for k, v := range recipeCtx.Env {
				matrixCtx.Env[k] = v
			}

			// Add matrix variables with matrix_ prefix
			for k, v := range combination {
				matrixCtx.Vars["matrix_"+k] = v
			}

			// Create unique node ID for this matrix combination
			nodeID := fmt.Sprintf("%s[%d]", recipeName, i)

			node := model.PlanNode{
				ID:        nodeID,
				Recipe:    &recipe,
				Context:   matrixCtx,
				Step:      recipe.Run,
				DependsOn: recipe.Deps,
			}

			*nodes = append(*nodes, node)
			currentIndex := len(*nodes) - 1

			// Map both the matrix node ID and the base recipe name to this index
			nodeIndex[nodeID] = currentIndex
			if i == 0 {
				// First matrix node also represents the base recipe
				nodeIndex[recipeName] = currentIndex
			}

			// Add edges from dependencies to this matrix node
			for _, dep := range recipe.Deps {
				depIndex, depExists := nodeIndex[dep]
				if !depExists {
					return fmt.Errorf("dependency '%s' not found for matrix recipe '%s'", dep, recipeName)
				}
				*edges = append(*edges, [2]int{depIndex, currentIndex})
			}
		}
	} else {
		// Regular single node
		node := model.PlanNode{
			ID:        recipeName,
			Recipe:    &recipe,
			Context:   recipeCtx,
			Step:      recipe.Run,
			DependsOn: recipe.Deps,
		}

		*nodes = append(*nodes, node)
		currentIndex := len(*nodes) - 1
		nodeIndex[recipeName] = currentIndex

		// Add edges from dependencies to current recipe
		for _, dep := range recipe.Deps {
			depIndex, depExists := nodeIndex[dep]
			if !depExists {
				return fmt.Errorf("dependency '%s' not found", dep)
			}
			*edges = append(*edges, [2]int{depIndex, currentIndex})
		}
	}

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

	// Build adjacency list and in-degree count with pre-allocation
	adj := make([][]int, n)
	inDegree := make([]int, n)

	// Pre-allocate adjacency lists based on estimated out-degree
	avgOutDegree := len(edges)
	if n > 0 {
		avgOutDegree = len(edges) / n
	}
	if avgOutDegree == 0 {
		avgOutDegree = 1 // Minimum allocation
	}
	for i := range adj {
		adj[i] = make([]int, 0, avgOutDegree)
	}

	for _, edge := range edges {
		from, to := edge[0], edge[1]
		if from >= n || to >= n || from < 0 || to < 0 {
			return nil, fmt.Errorf("invalid edge: [%d, %d]", from, to)
		}
		adj[from] = append(adj[from], to)
		inDegree[to]++
	}

	// Find all nodes with no incoming edges
	queue := make([]int, 0, n) // Pre-allocate queue
	for i := 0; i < n; i++ {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	result := make([]int, 0, n) // Pre-allocate result

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

// computeExecutionLevels computes which nodes can be executed in parallel
func (b *Builder) computeExecutionLevels(nodes []model.PlanNode, edges [][2]int) [][]int {
	nodeCount := len(nodes)
	if nodeCount == 0 {
		return [][]int{}
	}

	// Build adjacency list and in-degree count with pre-allocation
	adjList := make([][]int, nodeCount)
	inDegree := make([]int, nodeCount)

	// Pre-allocate adjacency lists
	avgOutDegree := len(edges)
	if nodeCount > 0 {
		avgOutDegree = len(edges) / nodeCount
	}
	if avgOutDegree == 0 {
		avgOutDegree = 1
	}
	for i := range adjList {
		adjList[i] = make([]int, 0, avgOutDegree)
	}

	for _, edge := range edges {
		from, to := edge[0], edge[1]
		adjList[from] = append(adjList[from], to)
		inDegree[to]++
	}

	// Pre-allocate levels slice with estimated size
	levels := make([][]int, 0, nodeCount/2+1) // Estimate levels
	remaining := make([]bool, nodeCount)
	for i := range remaining {
		remaining[i] = true
	}

	// Process nodes level by level using topological sort
	for {
		currentLevel := make([]int, 0, nodeCount) // Pre-allocate with max capacity

		// Find all nodes with no remaining dependencies
		for i := 0; i < nodeCount; i++ {
			if remaining[i] && inDegree[i] == 0 {
				currentLevel = append(currentLevel, i)
			}
		}

		if len(currentLevel) == 0 {
			break // No more nodes to process
		}

		// All nodes in this level can run in parallel
		levels = append(levels, currentLevel)
		for _, nodeIdx := range currentLevel {
			remaining[nodeIdx] = false
			for _, neighbor := range adjList[nodeIdx] {
				inDegree[neighbor]--
			}
		}
	}

	return levels
}

// generateMatrixCombinations generates all combinations of matrix variables
func (b *Builder) generateMatrixCombinations(matrix map[string][]any) []map[string]any {
	if len(matrix) == 0 {
		return []map[string]any{}
	}

	// Get keys and values
	keys := make([]string, 0, len(matrix))
	values := make([][]any, 0, len(matrix))

	for k, v := range matrix {
		keys = append(keys, k)
		values = append(values, v)
	}

	// Calculate total combinations
	totalCombinations := 1
	for _, vals := range values {
		totalCombinations *= len(vals)
	}

	// Generate all combinations
	combinations := make([]map[string]any, 0, totalCombinations)

	// Use recursive approach to generate combinations
	var generate func(int, map[string]any)
	generate = func(keyIndex int, current map[string]any) {
		if keyIndex == len(keys) {
			// Make a copy of the current combination
			combination := make(map[string]any, len(current))
			for k, v := range current {
				combination[k] = v
			}
			combinations = append(combinations, combination)
			return
		}

		key := keys[keyIndex]
		for _, value := range values[keyIndex] {
			current[key] = value
			generate(keyIndex+1, current)
		}
	}

	generate(0, make(map[string]any))
	return combinations
}
