package tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/report"
	"github.com/alexandrmotologa/agentloop/internal/types"
)

func TestComplianceReportGeneration(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-3 * time.Second)

	exec := &types.Execution{
		ID:              "exec_audit_001",
		WorkflowID:      "procurement_approval",
		Status:          types.StatusCompleted,
		CurrentNodeID:   "finalize",
		State:           map[string]any{"approval_status": "APPROVED"},
		HealingAttempts: 1,
		TotalTokens:     1450,
		TotalCostUSD:    0.0029,
		StartedAt:       startTime,
		UpdatedAt:       now,
	}

	wfSpec := &types.WorkflowSpec{
		Version: "1",
		Workflow: types.WorkflowMetadata{
			ID:   "procurement_approval",
			Name: "Enterprise Procurement",
		},
		Budget: &types.BudgetConfig{
			MaxTotalTokens: 2000,
			MaxCostUSD:     0.010,
		},
	}

	auditLogs := []types.StepAuditEntry{
		{
			ID:             1,
			ExecutionID:    exec.ID,
			NodeID:         "validate_po",
			NodeType:       types.NodeTypeTransform,
			Status:         types.NodeStatusSuccess,
			LatencyMs:      12,
			Timestamp:      startTime,
		},
		{
			ID:             2,
			ExecutionID:    exec.ID,
			NodeID:         "risk_analyzer",
			NodeType:       types.NodeTypeLLMAgent,
			Status:         types.NodeStatusHealed,
			LatencyMs:      450,
			Timestamp:      startTime.Add(1 * time.Second),
		},
	}

	healingLogs := []types.HealingLogEntry{
		{
			ID:            1,
			ExecutionID:   exec.ID,
			NodeID:        "risk_analyzer",
			AttemptNumber: 1,
			ErrorCategory: "SCHEMA_VALIDATION_ERROR",
			OriginalError: "missing required field 'risk_score'",
			MutatedPrompt: "Ensure 'risk_score' numeric field is included in JSON output",
			Status:        "SUCCESS",
			Timestamp:     startTime.Add(500 * time.Millisecond),
		},
	}

	rep := report.BuildComplianceReport(exec, wfSpec, auditLogs, healingLogs, nil)

	// 1. Check structured fields
	if !rep.Budget.WithinBudget {
		t.Errorf("Expected within budget = true")
	}
	if rep.Budget.TotalTokens != 1450 {
		t.Errorf("Expected 1450 tokens, got %d", rep.Budget.TotalTokens)
	}
	if len(rep.Nodes) != 2 {
		t.Errorf("Expected 2 node audits, got %d", len(rep.Nodes))
	}
	if len(rep.Healing) != 1 {
		t.Errorf("Expected 1 healing record, got %d", len(rep.Healing))
	}

	// 2. Generate and verify Markdown
	md := report.GenerateMarkdownReport(rep)
	if !strings.Contains(md, "# Compliance & Audit Report: `exec_audit_001`") {
		t.Errorf("Markdown title missing")
	}
	if !strings.Contains(md, "COMPLIANT") {
		t.Errorf("Markdown budget status missing")
	}
	if !strings.Contains(md, "SCHEMA_VALIDATION_ERROR") {
		t.Errorf("Markdown self-healing section missing error category")
	}

	// 3. Generate and verify JSON
	jsonBytes, err := report.GenerateJSONReport(rep)
	if err != nil {
		t.Fatalf("GenerateJSONReport failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("JSON parse failed: %v", err)
	}
	if parsed["summary"].(map[string]any)["execution_id"] != "exec_audit_001" {
		t.Errorf("Unexpected execution ID in JSON report")
	}

	fmt.Println("✓ Compliance report generation (Markdown & JSON) and budget validation verified")
}
