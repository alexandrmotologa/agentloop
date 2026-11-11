package engine

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DAG represents a parsed, validated workflow graph.
type DAG struct {
	Spec     *WorkflowSpec
	Nodes    map[string]*NodeSpec
	Outgoing map[string][]string // from -> []to
	Incoming map[string][]string // to -> []from
	Roots    []string
}

// ParseWorkflowYAML parses YAML bytes into a DAG and validates it.
func ParseWorkflowYAML(data []byte) (*DAG, error) {
	var spec WorkflowSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse workflow yaml: %w", err)
	}

	return BuildDAG(&spec)
}

// LoadWorkflowFile reads a file from disk and parses it into a DAG.
func LoadWorkflowFile(path string) (*DAG, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file %s: %w", path, err)
	}
	return ParseWorkflowYAML(data)
}

// BuildDAG constructs and validates a DAG from a WorkflowSpec.
func BuildDAG(spec *WorkflowSpec) (*DAG, error) {
	if spec == nil {
		return nil, fmt.Errorf("workflow spec cannot be nil")
	}
	if spec.Workflow.ID == "" {
		return nil, fmt.Errorf("workflow id is required")
	}
	if len(spec.Nodes) == 0 {
		return nil, fmt.Errorf("workflow must contain at least one node")
	}

	dag := &DAG{
		Spec:     spec,
		Nodes:    make(map[string]*NodeSpec, len(spec.Nodes)),
		Outgoing: make(map[string][]string),
		Incoming: make(map[string][]string),
	}

	for i := range spec.Nodes {
		node := &spec.Nodes[i]
		if node.ID == "" {
			return nil, fmt.Errorf("node at index %d has empty id", i)
		}
		if _, exists := dag.Nodes[node.ID]; exists {
			return nil, fmt.Errorf("duplicate node id: %s", node.ID)
		}
		dag.Nodes[node.ID] = node
		dag.Outgoing[node.ID] = nil
		dag.Incoming[node.ID] = nil
	}

	// Build edges based on:
	// 1. Explicit depends_on (incoming edges)
	// 2. Explicit next (outgoing edge)
	// 3. Conditional router branches (outgoing edges)
	// 4. Default sequential ordering if neither depends_on nor next is specified
	for i := 0; i < len(spec.Nodes); i++ {
		curr := &spec.Nodes[i]

		// 1. depends_on
		for _, depID := range curr.DependsOn {
			if _, exists := dag.Nodes[depID]; !exists {
				return nil, fmt.Errorf("node %s depends on non-existent node: %s", curr.ID, depID)
			}
			dag.addEdge(depID, curr.ID)
		}

		// 2. next
		if curr.Next != "" {
			if _, exists := dag.Nodes[curr.Next]; !exists {
				return nil, fmt.Errorf("node %s specifies non-existent next node: %s", curr.ID, curr.Next)
			}
			dag.addEdge(curr.ID, curr.Next)
		}

		// 3. router conditions
		for _, cond := range curr.Conditions {
			if cond.Next != "" {
				if _, exists := dag.Nodes[cond.Next]; !exists {
					return nil, fmt.Errorf("router node %s specifies non-existent condition target: %s", curr.ID, cond.Next)
				}
				dag.addEdge(curr.ID, cond.Next)
			}
		}

		// 4. Default sequential connection if no routing configured and not preceded by a router
		hasRouting := len(curr.Conditions) > 0 || curr.Next != ""
		prevIsRouter := i > 0 && spec.Nodes[i-1].Type == NodeTypeConditionalRouter
		if !hasRouting && len(curr.DependsOn) == 0 && i < len(spec.Nodes)-1 && !prevIsRouter {
			nextSequential := spec.Nodes[i+1].ID
			dag.addEdge(curr.ID, nextSequential)
		}
	}

	// Compute Roots (nodes with 0 incoming dependencies)
	for id := range dag.Nodes {
		if len(dag.Incoming[id]) == 0 {
			dag.Roots = append(dag.Roots, id)
		}
	}

	if len(dag.Roots) == 0 && len(dag.Nodes) > 0 {
		return nil, fmt.Errorf("workflow contains a cycle: no root entry node found")
	}

	// Cycle detection using topological sort (Kahn's algorithm)
	if _, err := dag.TopologicalSort(); err != nil {
		return nil, err
	}

	return dag, nil
}

func (d *DAG) addEdge(from, to string) {
	for _, existing := range d.Outgoing[from] {
		if existing == to {
			return
		}
	}
	d.Outgoing[from] = append(d.Outgoing[from], to)
	d.Incoming[to] = append(d.Incoming[to], from)
}

// TopologicalSort returns node IDs in valid dependency execution order.
// Returns an error if a cycle exists in the workflow graph.
func (d *DAG) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int, len(d.Nodes))
	for id := range d.Nodes {
		inDegree[id] = len(d.Incoming[id])
	}

	queue := make([]string, 0, len(d.Roots))
	for _, root := range d.Roots {
		queue = append(queue, root)
	}

	var order []string
	visitedCount := 0

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)
		visitedCount++

		for _, neighbor := range d.Outgoing[curr] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if visitedCount != len(d.Nodes) {
		return nil, fmt.Errorf("workflow graph contains a cycle: visited %d of %d nodes", visitedCount, len(d.Nodes))
	}

	return order, nil
}

// GetNode returns a node by ID.
func (d *DAG) GetNode(id string) (*NodeSpec, bool) {
	node, exists := d.Nodes[id]
	return node, exists
}
