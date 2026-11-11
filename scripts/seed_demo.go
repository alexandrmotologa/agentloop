package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/alexandrmotologa/agentloop/internal/types"
)

func main() {
	dbPath := "./demo_showcase.db"
	db, err := store.OpenDB(dbPath)
	if err != nil {
		fmt.Printf("Failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 1. Save all example workflows
	exampleFiles := []string{
		"examples/invoice_reconciliation.yaml",
		"examples/customer_kyc.yaml",
		"examples/incident_remediation.yaml",
	}

	for _, file := range exampleFiles {
		dag, err := engine.LoadWorkflowFile(file)
		if err != nil {
			fmt.Printf("Error loading %s: %v\n", file, err)
			continue
		}
		rawYAML, _ := os.ReadFile(file)
		_ = db.SaveWorkflow(dag.Spec, string(rawYAML))
		fmt.Printf("Saved workflow: %s\n", dag.Spec.Workflow.ID)
	}

	// 2. Create DLQ Item with frozen state
	frozenState := map[string]any{
		"invoice_pdf_path": "./invoices/INV-9402.pdf",
		"extracted_data": map[string]any{
			"vendor_name":   "Acme Industrial Solutions Inc.",
			"tax_id":        "US-94827104",
			"total_amount":  5420.00,
			"currency":      "USD",
			"line_items": []map[string]any{
				{"description": "Server Rack Unit A", "quantity": 2, "unit_price": 2200, "total": 4400},
				{"description": "Gigabit Network Switch", "quantity": 1, "unit_price": 1020, "total": 1020},
			},
		},
		"erp_purchase_order": map[string]any{
			"id":           "PO-2026-881",
			"vendor_name":  "Acme Industrial Solutions Inc.",
			"vendor_tax_id": "US-94827104",
			"total_amount": 4250.00,
			"currency":     "USD",
			"status":       "APPROVED",
		},
		"reconciliation_result": map[string]any{
			"is_approved":         false,
			"variance_percentage": 27.5,
			"discrepancies": []string{
				"Total amount 5420.00 exceeds approved PO total 4250.00 by 27.5%",
				"Unit price for Server Rack Unit A elevated from 1800.00 to 2200.00",
			},
			"summary": "Financial disbursement blocked: variance exceeds 2% threshold.",
		},
	}

	initialState := map[string]any{
		"invoice_pdf_path": "./invoices/INV-9402.pdf",
		"extracted_data": map[string]any{
			"vendor_name":  "Acme Industrial Solutions Inc.",
			"tax_id":       "US-94827104",
			"total_amount": 0,
		},
		"erp_purchase_order": map[string]any{},
	}

	now := time.Now()
	execDLQ := &types.Execution{
		ID:              "exec_dlq_9402",
		WorkflowID:      "vendor-invoice-reconciliation",
		Status:          types.StatusPausedDLQ,
		CurrentNodeID:   "pause_for_human_review",
		State:           frozenState,
		HealingAttempts: 2,
		TotalTokens:     2840,
		TotalCostUSD:    0.0142,
		StartedAt:       now.Add(-4 * time.Minute),
		UpdatedAt:       now.Add(-3 * time.Minute),
	}
	_ = db.CreateExecution(execDLQ)

	frozenJSON, _ := json.Marshal(map[string]any{"state": frozenState})
	initJSON, _ := json.Marshal(map[string]any{"state": initialState})

	// Initial snapshot
	_ = db.SaveSnapshot(&types.StateSnapshot{
		ExecutionID:  execDLQ.ID,
		NodeID:       "extract_invoice",
		StateJSON:    string(initJSON),
		StepSequence: 1,
		CreatedAt:    now.Add(-4 * time.Minute),
	})

	_ = db.InsertDLQItem(&types.DLQItem{
		ID:                "dlq_9402_inv",
		ExecutionID:       execDLQ.ID,
		FailedNodeID:      "pause_for_human_review",
		FailureReason:     "Disbursement Variance Exceeded Gate",
		ErrorDetails:      "Invoice total 5420.00 exceeds approved ERP PO total 4250.00 by 27.5%. Router routed to human_checkpoint.",
		SnapshotStateJSON: string(frozenJSON),
		CreatedAt:         now.Add(-3 * time.Minute),
	})

	_ = db.AppendAuditLog(&types.StepAuditEntry{
		ExecutionID:    execDLQ.ID,
		NodeID:         "extract_invoice",
		NodeType:       types.NodeTypeLLMAgent,
		InputPayload:   `{"path": "./invoices/INV-9402.pdf"}`,
		OutputPayload:  `{"vendor_name": "Acme Industrial Solutions Inc.", "total": 5420.00}`,
		TokensConsumed: 1250,
		LatencyMs:      12,
		Status:         types.NodeStatusSuccess,
		Timestamp:      now.Add(-4 * time.Minute),
	})

	_ = db.AppendAuditLog(&types.StepAuditEntry{
		ExecutionID:    execDLQ.ID,
		NodeID:         "fetch_erp_po",
		NodeType:       types.NodeTypeToolExecution,
		InputPayload:   `{"query": "SELECT * FROM purchase_orders WHERE vendor_tax_id = 'US-94827104'"}`,
		OutputPayload:  `{"id": "PO-2026-881", "total_amount": 4250.00}`,
		TokensConsumed: 0,
		LatencyMs:      2,
		Status:         types.NodeStatusSuccess,
		Timestamp:      now.Add(-3 * time.Minute),
	})

	_ = db.AppendAuditLog(&types.StepAuditEntry{
		ExecutionID:    execDLQ.ID,
		NodeID:         "reconcile_and_audit",
		NodeType:       types.NodeTypeLLMAgent,
		InputPayload:   `{"variance_check": true}`,
		OutputPayload:  `{"is_approved": false, "variance_percentage": 27.5}`,
		TokensConsumed: 1590,
		LatencyMs:      18,
		Status:         types.NodeStatusHealed,
		Timestamp:      now.Add(-3 * time.Minute),
	})

	// 3. Create Execution Paused at Breakpoint
	execDebug := &types.Execution{
		ID:              "exec_dbg_7721",
		WorkflowID:      "vendor-invoice-reconciliation",
		Status:          types.StatusBreakpoint,
		CurrentNodeID:   "reconcile_and_audit",
		Breakpoints:     []string{"reconcile_and_audit"},
		State:           frozenState,
		HealingAttempts: 0,
		TotalTokens:     890,
		TotalCostUSD:    0.00445,
		StartedAt:       now.Add(-1 * time.Minute),
		UpdatedAt:       now.Add(-30 * time.Second),
	}
	_ = db.CreateExecution(execDebug)

	fmt.Println("✓ Seeded realistic workflows, DLQ incident, and breakpoint debugging execution successfully!")
}
