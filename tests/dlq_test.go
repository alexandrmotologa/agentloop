package tests

import (
	"context"
	"testing"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/dlq"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/alexandrmotologa/agentloop/internal/types"
)

func TestDLQFreezingAndResolution(t *testing.T) {
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	mgr := dlq.NewManager(db)

	// Create test workflow & execution
	wfSpec := &types.WorkflowSpec{
		Workflow: types.WorkflowMetadata{ID: "test-dlq", Name: "Test DLQ"},
		Nodes:    []types.NodeSpec{{ID: "faulty_node", Type: types.NodeTypeLLMAgent}},
	}
	_ = db.SaveWorkflow(wfSpec, "version: '1'")

	exec := &types.Execution{
		ID:            "exec_dlq_001",
		WorkflowID:    "test-dlq",
		Status:        types.StatusRunning,
		CurrentNodeID: "faulty_node",
		State: map[string]any{
			"counter": 42,
			"status":  "CRASHED",
		},
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution failed: %v", err)
	}

	// Freeze to DLQ
	item, err := mgr.FreezeToDLQ(
		context.Background(),
		exec,
		"faulty_node",
		"Max Healing Exhausted",
		"Schema validation failed 3 times",
		types.NotificationConfig{},
	)
	if err != nil {
		t.Fatalf("FreezeToDLQ failed: %v", err)
	}

	if item.ID == "" {
		t.Fatalf("Expected non-empty DLQ ID")
	}

	// Verify execution state was updated in DB
	savedExec, err := db.GetExecution(exec.ID)
	if err != nil {
		t.Fatalf("GetExecution failed: %v", err)
	}
	if savedExec.Status != types.StatusPausedDLQ {
		t.Errorf("Expected execution status %s, got %s", types.StatusPausedDLQ, savedExec.Status)
	}

	// List pending DLQ items
	pending, err := mgr.ListItems("PENDING")
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("Expected 1 pending DLQ item, got %d", len(pending))
	}

	// Resolve DLQ item
	err = mgr.ResolveItem(item.ID, "lead_engineer", "RESUME", "Approved manual override")
	if err != nil {
		t.Fatalf("ResolveItem failed: %v", err)
	}

	// Check resolved list
	resolved, err := mgr.ListItems("RESOLVED")
	if err != nil {
		t.Fatalf("ListItems RESOLVED failed: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("Expected 1 resolved DLQ item, got %d", len(resolved))
	}
	if resolved[0].ResolvedBy != "lead_engineer" {
		t.Errorf("Expected resolved by 'lead_engineer', got '%s'", resolved[0].ResolvedBy)
	}
}
