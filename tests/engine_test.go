package tests

import (
	"context"
	"testing"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/dlq"
	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/alexandrmotologa/agentloop/internal/types"
)

func TestEngineEndToEndExecutionAndOverhead(t *testing.T) {
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	dlqMgr := dlq.NewManager(db)
	executor := engine.NewExecutor(db, dlqMgr)

	spec := &types.WorkflowSpec{
		Workflow: types.WorkflowMetadata{
			ID:                 "perf-test-workflow",
			Name:               "High-Throughput Performance Test",
			MaxHealingAttempts: 3,
		},
		State: map[string]any{
			"counter": 100,
		},
		Nodes: []types.NodeSpec{
			{
				ID:   "step_1",
				Type: types.NodeTypeTransform,
				Params: map[string]any{
					"expr": "{ count: state.counter + 50 }",
				},
				SaveTo: "step_1_res",
			},
			{
				ID:   "step_2",
				Type: types.NodeTypeTransform,
				Params: map[string]any{
					"expr": "{ doubled: state.step_1_res.count * 2 }",
				},
				SaveTo: "step_2_res",
			},
			{
				ID:   "step_3",
				Type: types.NodeTypeTransform,
				Params: map[string]any{
					"expr": "{ final: state.step_2_res.doubled + 10 }",
				},
				SaveTo: "final_res",
			},
		},
	}

	start := time.Now()
	exec, err := executor.StartSpec(context.Background(), spec, nil)
	totalDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if exec.Status != types.StatusCompleted {
		t.Fatalf("Expected COMPLETED status, got %s", exec.Status)
	}

	t.Logf("Total 3-node workflow execution time: %v", totalDuration)
	// Target is sub-25ms overhead per node transition
	if totalDuration > 100*time.Millisecond {
		t.Errorf("Execution overhead took longer than expected: %v", totalDuration)
	}

	// Verify state mutations
	finalRes, ok := exec.State["final_res"].(map[string]any)
	if !ok {
		t.Fatalf("Expected final_res in state, got %v", exec.State)
	}

	// (100 + 50) * 2 + 10 = 310
	expected := 310.0
	actual, ok := finalRes["final"].(float64)
	if !ok {
		if intVal, ok := finalRes["final"].(int); ok {
			actual = float64(intVal)
		} else {
			t.Fatalf("Unexpected type for final: %v", finalRes["final"])
		}
	}
	if actual != expected {
		t.Errorf("Expected result %.0f, got %.0f", expected, actual)
	}
}

func TestEngineStateRewind(t *testing.T) {
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	dlqMgr := dlq.NewManager(db)
	executor := engine.NewExecutor(db, dlqMgr)

	spec := &types.WorkflowSpec{
		Workflow: types.WorkflowMetadata{
			ID:   "rewind-workflow",
			Name: "Rewind Test Workflow",
		},
		State: map[string]any{
			"val": 10,
		},
		Nodes: []types.NodeSpec{
			{
				ID:     "node_1",
				Type:   types.NodeTypeTransform,
				Params: map[string]any{"expr": "{ val: 20 }"},
				SaveTo: "res_1",
				Next:   "node_2",
			},
			{
				ID:     "node_2",
				Type:   types.NodeTypeTransform,
				Params: map[string]any{"expr": "{ val: 30 }"},
				SaveTo: "res_2",
			},
		},
	}

	_ = db.SaveWorkflow(spec, "")

	exec, err := executor.StartSpec(context.Background(), spec, nil)
	if err != nil {
		t.Fatalf("StartSpec failed: %v", err)
	}

	// Check snapshots saved
	snapshots, err := db.GetSnapshots(exec.ID)
	if err != nil {
		t.Fatalf("GetSnapshots failed: %v", err)
	}
	if len(snapshots) < 2 {
		t.Fatalf("Expected at least 2 state snapshots, got %d", len(snapshots))
	}

	// Rewind back to node_1
	rewoundExec, err := executor.Rewind(context.Background(), exec.ID, "node_1")
	if err != nil {
		t.Fatalf("Rewind failed: %v", err)
	}

	if rewoundExec.Status != types.StatusCompleted {
		t.Errorf("Expected COMPLETED status after replay, got %s", rewoundExec.Status)
	}
}
