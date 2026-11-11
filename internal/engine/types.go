package engine

import (
	"github.com/alexandrmotologa/agentloop/internal/types"
)

// Re-export types from internal/types for convenience
type (
	ExecutionStatus    = types.ExecutionStatus
	NodeStatus         = types.NodeStatus
	NodeType           = types.NodeType
	WorkflowSpec       = types.WorkflowSpec
	WorkflowMetadata   = types.WorkflowMetadata
	BudgetConfig       = types.BudgetConfig
	ModelConfig        = types.ModelConfig
	NodeSpec           = types.NodeSpec
	LowConfidenceRule  = types.LowConfidenceRule
	FailureRule        = types.FailureRule
	RetryConfig        = types.RetryConfig
	ConditionRule      = types.ConditionRule
	NotificationConfig = types.NotificationConfig
	WebhookAlertConfig = types.WebhookAlertConfig
	SlackAlertConfig   = types.SlackAlertConfig
	Execution          = types.Execution
	DLQItem            = types.DLQItem
	StepAuditEntry     = types.StepAuditEntry
	HealingLogEntry    = types.HealingLogEntry
	StateSnapshot      = types.StateSnapshot
)

const (
	StatusPending    = types.StatusPending
	StatusRunning    = types.StatusRunning
	StatusCompleted  = types.StatusCompleted
	StatusFailed     = types.StatusFailed
	StatusPausedDLQ  = types.StatusPausedDLQ
	StatusBreakpoint = types.StatusBreakpoint

	NodeStatusPending = types.NodeStatusPending
	NodeStatusRunning = types.NodeStatusRunning
	NodeStatusSuccess = types.NodeStatusSuccess
	NodeStatusHealed  = types.NodeStatusHealed
	NodeStatusFailed  = types.NodeStatusFailed
	NodeStatusPaused  = types.NodeStatusPaused
	NodeStatusHalted  = types.NodeStatusHalted

	NodeTypeLLMAgent          = types.NodeTypeLLMAgent
	NodeTypeToolExecution     = types.NodeTypeToolExecution
	NodeTypeConditionalRouter = types.NodeTypeConditionalRouter
	NodeTypeHumanCheckpoint   = types.NodeTypeHumanCheckpoint
	NodeTypeTransform         = types.NodeTypeTransform
)
