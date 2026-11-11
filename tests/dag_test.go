package tests

import (
	"testing"

	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/types"
)

func TestDAGValidationAndTopologicalSort(t *testing.T) {
	spec := &types.WorkflowSpec{
		Workflow: types.WorkflowMetadata{
			ID:   "test-dag",
			Name: "Test DAG",
		},
		Nodes: []types.NodeSpec{
			{ID: "node_a", Type: types.NodeTypeLLMAgent, Next: "node_b"},
			{ID: "node_b", Type: types.NodeTypeToolExecution, Next: "node_c"},
			{ID: "node_c", Type: types.NodeTypeTransform},
		},
	}

	dag, err := engine.BuildDAG(spec)
	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	order, err := dag.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	expected := []string{"node_a", "node_b", "node_c"}
	if len(order) != len(expected) {
		t.Fatalf("Expected %d nodes, got %d", len(expected), len(order))
	}
	for i, id := range expected {
		if order[i] != id {
			t.Errorf("At index %d expected %s, got %s", i, id, order[i])
		}
	}
}

func TestDAGCycleDetection(t *testing.T) {
	spec := &types.WorkflowSpec{
		Workflow: types.WorkflowMetadata{
			ID:   "cycle-dag",
			Name: "Cycle DAG",
		},
		Nodes: []types.NodeSpec{
			{ID: "node_1", Type: types.NodeTypeLLMAgent, Next: "node_2"},
			{ID: "node_2", Type: types.NodeTypeToolExecution, Next: "node_3"},
			{ID: "node_3", Type: types.NodeTypeTransform, Next: "node_1"}, // Cycle!
		},
	}

	_, err := engine.BuildDAG(spec)
	if err == nil {
		t.Fatalf("Expected error for circular DAG dependency, got nil")
	}
	t.Logf("Correctly rejected cyclic DAG: %v", err)
}
