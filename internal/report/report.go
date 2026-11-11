package report

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/types"
)

// ComplianceReport represents the structured data for an execution audit.
type ComplianceReport struct {
	GeneratedAt time.Time          `json:"generated_at"`
	Summary     ExecutionSummary   `json:"summary"`
	Budget      BudgetAudit        `json:"budget"`
	Nodes       []NodeAuditRecord  `json:"nodes"`
	Healing     []HealingRecord    `json:"healing_events"`
	HITL        *HITLRecord        `json:"human_in_the_loop,omitempty"`
}

type ExecutionSummary struct {
	ExecutionID string    `json:"execution_id"`
	WorkflowID  string    `json:"workflow_id"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	DurationMs  int64     `json:"duration_ms"`
}

type BudgetAudit struct {
	TotalTokens   int     `json:"total_tokens"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
	MaxTokens     int     `json:"max_tokens_limit,omitempty"`
	MaxCostUSD    float64 `json:"max_cost_limit_usd,omitempty"`
	WithinBudget  bool    `json:"within_budget"`
}

type NodeAuditRecord struct {
	NodeID     string `json:"node_id"`
	NodeType   string `json:"node_type"`
	Status     string `json:"status"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

type HealingRecord struct {
	ID            string    `json:"id"`
	NodeID        string    `json:"node_id"`
	ErrorCategory string    `json:"error_category"`
	Strategy      string    `json:"strategy"`
	AttemptNumber int       `json:"attempt_number"`
	Resolved      bool      `json:"resolved"`
	CreatedAt     time.Time `json:"created_at"`
}

type HITLRecord struct {
	DLQID         string     `json:"dlq_id"`
	FailedNodeID  string     `json:"failed_node_id"`
	Reason        string     `json:"reason"`
	Status        string     `json:"status"`
	FrozenAt      time.Time  `json:"frozen_at"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy    string     `json:"resolved_by,omitempty"`
	Action        string     `json:"action,omitempty"`
	Instruction   string     `json:"instruction,omitempty"`
}

// BuildComplianceReport compiles audit records into the structured format.
func BuildComplianceReport(
	exec *types.Execution,
	wfSpec *types.WorkflowSpec,
	auditLogs []types.StepAuditEntry,
	healingLogs []types.HealingLogEntry,
	dlqItem *types.DLQItem,
) *ComplianceReport {
	durationMs := int64(0)
	if !exec.UpdatedAt.IsZero() && !exec.StartedAt.IsZero() {
		durationMs = exec.UpdatedAt.Sub(exec.StartedAt).Milliseconds()
	}

	withinBudget := true
	var maxTokens int
	var maxCost float64
	if wfSpec != nil && wfSpec.Budget != nil {
		maxTokens = wfSpec.Budget.MaxTotalTokens
		maxCost = wfSpec.Budget.MaxCostUSD
		if maxTokens > 0 && exec.TotalTokens > maxTokens {
			withinBudget = false
		}
		if maxCost > 0 && exec.TotalCostUSD > maxCost {
			withinBudget = false
		}
	}

	nodes := make([]NodeAuditRecord, 0, len(auditLogs))
	for _, l := range auditLogs {
		nodes = append(nodes, NodeAuditRecord{
			NodeID:     l.NodeID,
			NodeType:   string(l.NodeType),
			Status:     string(l.Status),
			DurationMs: l.LatencyMs,
			Error:      "",
		})
	}

	healingRecords := make([]HealingRecord, 0, len(healingLogs))
	for _, h := range healingLogs {
		healingRecords = append(healingRecords, HealingRecord{
			ID:            fmt.Sprintf("%d", h.ID),
			NodeID:        h.NodeID,
			ErrorCategory: h.ErrorCategory,
			Strategy:      "prompt_mutation",
			AttemptNumber: h.AttemptNumber,
			Resolved:      h.Status == "SUCCESS",
			CreatedAt:     h.Timestamp,
		})
	}

	var hitl *HITLRecord
	if dlqItem != nil {
		status := "FROZEN"
		if dlqItem.ResolvedAt != nil {
			status = "RESOLVED (" + dlqItem.ResolutionAction + ")"
		}

		hitl = &HITLRecord{
			DLQID:        dlqItem.ID,
			FailedNodeID: dlqItem.FailedNodeID,
			Reason:       dlqItem.FailureReason,
			Status:       status,
			FrozenAt:     dlqItem.CreatedAt,
			ResolvedAt:   dlqItem.ResolvedAt,
			ResolvedBy:   dlqItem.ResolvedBy,
			Action:       dlqItem.ResolutionAction,
			Instruction:  dlqItem.OperatorInstruction,
		}
	}

	return &ComplianceReport{
		GeneratedAt: time.Now().UTC(),
		Summary: ExecutionSummary{
			ExecutionID: exec.ID,
			WorkflowID:  exec.WorkflowID,
			Status:      string(exec.Status),
			StartedAt:   exec.StartedAt,
			CompletedAt: exec.UpdatedAt,
			DurationMs:  durationMs,
		},
		Budget: BudgetAudit{
			TotalTokens:  exec.TotalTokens,
			TotalCostUSD: exec.TotalCostUSD,
			MaxTokens:    maxTokens,
			MaxCostUSD:   maxCost,
			WithinBudget: withinBudget,
		},
		Nodes:   nodes,
		Healing: healingRecords,
		HITL:    hitl,
	}
}

// GenerateJSONReport serializes the compliance audit into formatted JSON.
func GenerateJSONReport(rep *ComplianceReport) ([]byte, error) {
	return json.MarshalIndent(rep, "", "  ")
}

// GenerateMarkdownReport renders a clean, professional GitHub-flavored Markdown report.
func GenerateMarkdownReport(rep *ComplianceReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Compliance & Audit Report: `%s`\n\n", rep.Summary.ExecutionID))
	sb.WriteString(fmt.Sprintf("**Workflow**: `%s`  \n", rep.Summary.WorkflowID))
	sb.WriteString(fmt.Sprintf("**Execution Status**: `%s`  \n", rep.Summary.Status))
	sb.WriteString(fmt.Sprintf("**Generated At**: %s  \n\n", rep.GeneratedAt.Format(time.RFC3339)))

	sb.WriteString("## 1. Executive Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|---|---|\n")
	sb.WriteString(fmt.Sprintf("| Execution ID | `%s` |\n", rep.Summary.ExecutionID))
	sb.WriteString(fmt.Sprintf("| Workflow ID | `%s` |\n", rep.Summary.WorkflowID))
	sb.WriteString(fmt.Sprintf("| Final Status | **%s** |\n", rep.Summary.Status))
	sb.WriteString(fmt.Sprintf("| Started At | %s |\n", rep.Summary.StartedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("| Completed At | %s |\n", rep.Summary.CompletedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("| Total Duration | %d ms (%.2fs) |\n", rep.Summary.DurationMs, float64(rep.Summary.DurationMs)/1000.0))
	sb.WriteString("\n")

	sb.WriteString("## 2. Resource & Budget Sentinel Audit\n\n")
	sb.WriteString("| Resource Metric | Consumption | Limit / Constraint | Compliance Status |\n")
	sb.WriteString("|---|---|---|---|\n")
	
	tokensLimit := "Unlimited"
	if rep.Budget.MaxTokens > 0 {
		tokensLimit = fmt.Sprintf("%d tokens", rep.Budget.MaxTokens)
	}
	costLimit := "Unlimited"
	if rep.Budget.MaxCostUSD > 0 {
		costLimit = fmt.Sprintf("$%.4f USD", rep.Budget.MaxCostUSD)
	}
	
	budgetStatus := "COMPLIANT"
	if !rep.Budget.WithinBudget {
		budgetStatus = "EXCEEDED"
	}

	sb.WriteString(fmt.Sprintf("| Tokens Consumed | %d | %s | %s |\n", rep.Budget.TotalTokens, tokensLimit, budgetStatus))
	sb.WriteString(fmt.Sprintf("| LLM Estimated Cost | $%.6f USD | %s | %s |\n", rep.Budget.TotalCostUSD, costLimit, budgetStatus))
	sb.WriteString("\n")

	sb.WriteString("## 3. Node Execution Trace & Lineage\n\n")
	if len(rep.Nodes) == 0 {
		sb.WriteString("No node audit logs recorded for this execution.\n\n")
	} else {
		sb.WriteString("| Sequence | Node ID | Node Type | Status | Latency (ms) | Error / Notes |\n")
		sb.WriteString("|---|---|---|---|---|---|\n")
		for idx, n := range rep.Nodes {
			errStr := "-"
			if n.Error != "" {
				errStr = fmt.Sprintf("`%s`", strings.ReplaceAll(n.Error, "\n", " "))
			}
			sb.WriteString(fmt.Sprintf("| %d | `%s` | `%s` | %s | %d ms | %s |\n",
				idx+1, n.NodeID, n.NodeType, n.Status, n.DurationMs, errStr))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## 4. Autonomous Self-Healing Audit Trail\n\n")
	if len(rep.Healing) == 0 {
		sb.WriteString("Zero self-healing interventions required. All workflow steps succeeded deterministically.\n\n")
	} else {
		sb.WriteString("| Attempt | Node ID | Error Category | Repair Strategy | Resolved | Timestamp |\n")
		sb.WriteString("|---|---|---|---|---|---|\n")
		for _, h := range rep.Healing {
			res := "Escalated"
			if h.Resolved {
				res = "Resolved"
			}
			sb.WriteString(fmt.Sprintf("| #%d | `%s` | `%s` | `%s` | %s | %s |\n",
				h.AttemptNumber, h.NodeID, h.ErrorCategory, h.Strategy, res, h.CreatedAt.Format(time.RFC3339)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## 5. Human-in-the-Loop (HITL) Interventions\n\n")
	if rep.HITL == nil {
		sb.WriteString("No Dead-Letter Queue escalations or manual human overrides were required.\n\n")
	} else {
		sb.WriteString("| Field | Details |\n")
		sb.WriteString("|---|---|\n")
		sb.WriteString(fmt.Sprintf("| DLQ Item ID | `%s` |\n", rep.HITL.DLQID))
		sb.WriteString(fmt.Sprintf("| Failed Node | `%s` |\n", rep.HITL.FailedNodeID))
		sb.WriteString(fmt.Sprintf("| Escalation Reason | %s |\n", rep.HITL.Reason))
		sb.WriteString(fmt.Sprintf("| DLQ Status | **%s** |\n", rep.HITL.Status))
		sb.WriteString(fmt.Sprintf("| Frozen At | %s |\n", rep.HITL.FrozenAt.Format(time.RFC3339)))
		if rep.HITL.ResolvedAt != nil {
			sb.WriteString(fmt.Sprintf("| Resolved At | %s |\n", rep.HITL.ResolvedAt.Format(time.RFC3339)))
			sb.WriteString(fmt.Sprintf("| Resolved By | `%s` |\n", rep.HITL.ResolvedBy))
			sb.WriteString(fmt.Sprintf("| Operator Action | `%s` |\n", rep.HITL.Action))
			sb.WriteString(fmt.Sprintf("| Human Instructions | %s |\n", rep.HITL.Instruction))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("*Generated autonomously by `agentloop` enterprise workflow engine compliance auditor.*\n")

	return sb.String()
}
