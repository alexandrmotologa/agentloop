package tests

import (
	"fmt"
	"testing"

	"github.com/alexandrmotologa/agentloop/internal/cron"
	"github.com/alexandrmotologa/agentloop/internal/dlq"
	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/alexandrmotologa/agentloop/internal/types"
)

func TestCronScheduler(t *testing.T) {
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	dlqMgr := dlq.NewManager(db)
	executor := engine.NewExecutor(db, dlqMgr)
	scheduler := cron.NewScheduler(db, executor)

	// Save workflow with hourly schedule
	wfSpec := &types.WorkflowSpec{
		Version: "1",
		Workflow: types.WorkflowMetadata{
			ID:       "hourly-reconciliation",
			Name:     "Hourly Reconciliation",
			Schedule: "@hourly",
		},
		Nodes: []types.NodeSpec{
			{
				ID:   "check",
				Type: types.NodeTypeTransform,
				Params: map[string]any{
					"expr": "1 + 1",
				},
			},
		},
	}

	if err := db.SaveWorkflow(wfSpec, ""); err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	if err := scheduler.Start(); err != nil {
		t.Fatalf("Scheduler Start failed: %v", err)
	}
	defer scheduler.Stop()

	jobs := scheduler.ListJobs()
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 scheduled job, got %d", len(jobs))
	}

	if jobs[0].WorkflowID != "hourly-reconciliation" {
		t.Errorf("Expected workflow ID 'hourly-reconciliation', got %s", jobs[0].WorkflowID)
	}

	// Add second workflow and reload
	wf2 := &types.WorkflowSpec{
		Version: "1",
		Workflow: types.WorkflowMetadata{
			ID:       "daily-cleanup",
			Name:     "Daily Cleanup",
			Schedule: "@daily",
		},
		Nodes: []types.NodeSpec{
			{
				ID:   "clean",
				Type: types.NodeTypeTransform,
				Params: map[string]any{
					"expr": "'done'",
				},
			},
		},
	}
	if err := db.SaveWorkflow(wf2, ""); err != nil {
		t.Fatalf("SaveWorkflow 2 failed: %v", err)
	}

	if err := scheduler.Reload(); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	jobs = scheduler.ListJobs()
	if len(jobs) != 2 {
		t.Fatalf("Expected 2 scheduled jobs after reload, got %d", len(jobs))
	}

	fmt.Println("✓ Cron scheduler registration, job listing, and dynamic reload verified")
}
