package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/alexandrmotologa/agentloop/internal/dlq"
	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/alexandrmotologa/agentloop/internal/types"
)

func TestBreakpointDebugger(t *testing.T) {
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	dlqMgr := dlq.NewManager(db)
	executor := engine.NewExecutor(db, dlqMgr)

	// Workflow with 3 sequential transform steps, step 2 marked with breakpoint
	spec := &types.WorkflowSpec{
		Version: "1",
		Workflow: types.WorkflowMetadata{
			ID:   "debug-workflow",
			Name: "Debug Workflow",
		},
		Nodes: []types.NodeSpec{
			{
				ID:   "node_1",
				Type: types.NodeTypeTransform,
				Params: map[string]any{
					"expr": "{ step1: 'done' }",
				},
				SaveTo: "res1",
			},
			{
				ID:         "node_2",
				Type:       types.NodeTypeTransform,
				DependsOn:  []string{"node_1"},
				Breakpoint: true, // Breakpoint configured!
				Params: map[string]any{
					"expr": "{ step2: 'done' }",
				},
				SaveTo: "res2",
			},
			{
				ID:        "node_3",
				Type:      types.NodeTypeTransform,
				DependsOn: []string{"node_2"},
				Params: map[string]any{
					"expr": "{ step3: 'done' }",
				},
				SaveTo: "res3",
			},
		},
	}

	ctx := context.Background()

	// 1. Start execution -> must halt before node_2
	exec, err := executor.StartSpec(ctx, spec, nil)
	if err != nil {
		t.Fatalf("StartSpec failed: %v", err)
	}

	if exec.Status != types.StatusBreakpoint {
		t.Fatalf("Expected status %s, got %s", types.StatusBreakpoint, exec.Status)
	}
	if exec.CurrentNodeID != "node_2" {
		t.Errorf("Expected execution halted at node_2, got %s", exec.CurrentNodeID)
	}

	// Verify step 1 finished and is in state
	if exec.State["res1"] == nil {
		t.Errorf("Expected res1 to be populated in state")
	}
	if exec.State["res2"] != nil {
		t.Errorf("Expected res2 not yet executed")
	}

	// 2. Step forward once -> should execute node_2 and halt at node_3
	stepExec, err := executor.Step(ctx, exec.ID)
	if err != nil {
		t.Fatalf("Step failed: %v", err)
	}

	if stepExec.Status != types.StatusBreakpoint {
		t.Fatalf("Expected status %s after step, got %s", types.StatusBreakpoint, stepExec.Status)
	}
	if stepExec.CurrentNodeID != "node_3" {
		t.Errorf("Expected execution halted at node_3 after step, got %s", stepExec.CurrentNodeID)
	}
	if stepExec.State["res2"] == nil {
		t.Errorf("Expected res2 to be populated after step")
	}

	// 3. Continue execution -> should run remaining nodes and complete
	contExec, err := executor.Continue(ctx, exec.ID)
	if err != nil {
		t.Fatalf("Continue failed: %v", err)
	}

	if contExec.Status != types.StatusCompleted {
		t.Fatalf("Expected status %s after continue, got %s", types.StatusCompleted, contExec.Status)
	}
	if contExec.State["res3"] == nil {
		t.Errorf("Expected res3 to be populated after completion")
	}

	fmt.Println("✓ Breakpoint halt, single-step execution, and continue to completion verified")
}
