package types

import (
	"time"
)

// ExecutionStatus represents the lifecycle state of a workflow execution.
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "PENDING"
	StatusRunning   ExecutionStatus = "RUNNING"
	StatusCompleted ExecutionStatus = "COMPLETED"
	StatusFailed    ExecutionStatus = "FAILED"
	StatusPausedDLQ ExecutionStatus = "PAUSED_DLQ"
)

// NodeStatus represents the state of an individual node in an execution.
type NodeStatus string

const (
	NodeStatusPending NodeStatus = "PENDING"
	NodeStatusRunning NodeStatus = "RUNNING"
	NodeStatusSuccess NodeStatus = "SUCCESS"
	NodeStatusHealed  NodeStatus = "HEALED"
	NodeStatusFailed  NodeStatus = "FAILED"
	NodeStatusPaused  NodeStatus = "PAUSED"
)

// NodeType represents the kind of operation performed by a node.
type NodeType string

const (
	NodeTypeLLMAgent          NodeType = "llm_agent"
	NodeTypeToolExecution     NodeType = "tool_execution"
	NodeTypeConditionalRouter NodeType = "conditional_router"
	NodeTypeHumanCheckpoint   NodeType = "human_checkpoint"
	NodeTypeTransform         NodeType = "transform"
)

// WorkflowSpec defines the declarative structure of a workflow parsed from YAML or JSON.
type WorkflowSpec struct {
	Version       string                 `json:"version" yaml:"version"`
	Workflow      WorkflowMetadata       `json:"workflow" yaml:"workflow"`
	Models        map[string]ModelConfig `json:"models" yaml:"models"`
	State         map[string]any         `json:"state" yaml:"state"`
	Nodes         []NodeSpec             `json:"nodes" yaml:"nodes"`
	Notifications NotificationConfig     `json:"notifications" yaml:"notifications"`
}

// WorkflowMetadata contains identification and global constraints.
type WorkflowMetadata struct {
	ID                 string `json:"id" yaml:"id"`
	Name               string `json:"name" yaml:"name"`
	Description        string `json:"description" yaml:"description"`
	Timeout            string `json:"timeout" yaml:"timeout"`
	MaxHealingAttempts int    `json:"max_healing_attempts" yaml:"max_healing_attempts"`
}

// ModelConfig configures an LLM provider and model parameters.
type ModelConfig struct {
	Provider    string  `json:"provider" yaml:"provider"` // openai, anthropic, ollama, mock
	Model       string  `json:"model" yaml:"model"`
	Temperature float64 `json:"temperature" yaml:"temperature"`
	MaxTokens   int     `json:"max_tokens" yaml:"max_tokens"`
	BaseURL     string  `json:"base_url" yaml:"base_url"`
	APIKeyEnv   string  `json:"api_key_env" yaml:"api_key_env"`
}

// NodeSpec defines a single execution step within the workflow DAG.
type NodeSpec struct {
	ID                  string             `json:"id" yaml:"id"`
	Type                NodeType           `json:"type" yaml:"type"`
	Description         string             `json:"description" yaml:"description"`
	DependsOn           []string           `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Next                string             `json:"next,omitempty" yaml:"next,omitempty"`
	Prompt              string             `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	Input               map[string]any     `json:"input,omitempty" yaml:"input,omitempty"`
	OutputSchema        map[string]any     `json:"output_schema,omitempty" yaml:"output_schema,omitempty"`
	ConfidenceThreshold float64            `json:"confidence_threshold,omitempty" yaml:"confidence_threshold,omitempty"`
	OnLowConfidence     *LowConfidenceRule `json:"on_low_confidence,omitempty" yaml:"on_low_confidence,omitempty"`
	OnFailure           *FailureRule       `json:"on_failure,omitempty" yaml:"on_failure,omitempty"`
	Tool                string             `json:"tool,omitempty" yaml:"tool,omitempty"`
	Params              map[string]any     `json:"params,omitempty" yaml:"params,omitempty"`
	Retry               *RetryConfig       `json:"retry,omitempty" yaml:"retry,omitempty"`
	Conditions          []ConditionRule    `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	SaveTo              string             `json:"save_to,omitempty" yaml:"save_to,omitempty"`
}

// LowConfidenceRule dictates sentinel behavior when output confidence falls below threshold.
type LowConfidenceRule struct {
	Action string `json:"action" yaml:"action"` // pause_to_dlq, self_heal, continue
	Reason string `json:"reason" yaml:"reason"`
}

// FailureRule dictates how node execution failures are handled.
type FailureRule struct {
	Strategy     string `json:"strategy" yaml:"strategy"` // self_heal, retry, pause_to_dlq, abort
	PromptRepair string `json:"prompt_repair,omitempty" yaml:"prompt_repair,omitempty"`
	FallbackTool string `json:"fallback_tool,omitempty" yaml:"fallback_tool,omitempty"`
}

// RetryConfig specifies backoff behavior for transient failures.
type RetryConfig struct {
	MaxAttempts     int    `json:"max_attempts" yaml:"max_attempts"`
	Backoff         string `json:"backoff" yaml:"backoff"` // exponential, linear, fixed
	InitialInterval string `json:"initial_interval,omitempty" yaml:"initial_interval,omitempty"`
}

// ConditionRule defines branch evaluation for conditional router nodes.
type ConditionRule struct {
	Condition string `json:"condition" yaml:"condition"`
	Next      string `json:"next" yaml:"next"`
}

// NotificationConfig sets alert destinations.
type NotificationConfig struct {
	Webhook *WebhookAlertConfig `json:"webhook,omitempty" yaml:"webhook,omitempty"`
	Slack   *SlackAlertConfig   `json:"slack,omitempty" yaml:"slack,omitempty"`
}

type WebhookAlertConfig struct {
	URL string `json:"url" yaml:"url"`
}

type SlackAlertConfig struct {
	WebhookURL string `json:"webhook_url" yaml:"webhook_url"`
	Channel    string `json:"channel" yaml:"channel"`
}

// Execution tracks an instance of a running workflow.
type Execution struct {
	ID              string          `json:"id"`
	WorkflowID      string          `json:"workflow_id"`
	Status          ExecutionStatus `json:"status"`
	CurrentNodeID   string          `json:"current_node_id"`
	State           map[string]any  `json:"state"`
	HealingAttempts int             `json:"healing_attempts"`
	StartedAt       time.Time       `json:"started_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
}

// DLQItem represents a suspended execution in the Dead-Letter Queue.
type DLQItem struct {
	ID                  string     `json:"id"`
	ExecutionID         string     `json:"execution_id"`
	FailedNodeID        string     `json:"failed_node_id"`
	FailureReason       string     `json:"failure_reason"`
	ErrorDetails        string     `json:"error_details"`
	SnapshotStateJSON   string     `json:"snapshot_state_json"`
	ResolvedBy          string     `json:"resolved_by,omitempty"`
	ResolutionAction    string     `json:"resolution_action,omitempty"`
	OperatorInstruction string     `json:"operator_instruction,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	ResolvedAt          *time.Time `json:"resolved_at,omitempty"`
}

// StepAuditEntry records an immutable execution step.
type StepAuditEntry struct {
	ID             int64      `json:"id"`
	ExecutionID    string     `json:"execution_id"`
	NodeID         string     `json:"node_id"`
	NodeType       NodeType   `json:"node_type"`
	InputPayload   string     `json:"input_payload"`
	OutputPayload  string     `json:"output_payload"`
	TokensConsumed int        `json:"tokens_consumed"`
	LatencyMs      int64      `json:"latency_ms"`
	Status         NodeStatus `json:"status"`
	Timestamp      time.Time  `json:"timestamp"`
}

// HealingLogEntry records a single reflection and prompt repair attempt.
type HealingLogEntry struct {
	ID             int64     `json:"id"`
	ExecutionID    string    `json:"execution_id"`
	NodeID         string    `json:"node_id"`
	AttemptNumber  int       `json:"attempt_number"`
	ErrorCategory  string    `json:"error_category"`
	OriginalError  string    `json:"original_error"`
	MutatedPrompt  string    `json:"mutated_prompt"`
	Status         string    `json:"status"` // SUCCESS, FAILED, EXHAUSTED
	Timestamp      time.Time `json:"timestamp"`
}

// StateSnapshot captures full execution state at a specific node transition.
type StateSnapshot struct {
	ID           int64     `json:"id"`
	ExecutionID  string    `json:"execution_id"`
	NodeID       string    `json:"node_id"`
	StateJSON    string    `json:"state_json"`
	StepSequence int       `json:"step_sequence"`
	CreatedAt    time.Time `json:"created_at"`
}
