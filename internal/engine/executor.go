package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/healing"
	"github.com/alexandrmotologa/agentloop/internal/llm"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/alexandrmotologa/agentloop/internal/tools"
	"github.com/google/uuid"
)

// DLQFreezer defines the interface for freezing an execution to the Dead-Letter Queue.
type DLQFreezer interface {
	FreezeToDLQ(ctx context.Context, exec *Execution, failedNodeID string, reason string, errDetails string, notifications NotificationConfig) (*DLQItem, error)
	ResolveItem(id string, resolvedBy string, action string, instruction string) error
	GetItem(id string) (*DLQItem, error)
}

// Executor orchestrates workflow state transitions, sentinels, self-healing, and DLQ persistence.
type Executor struct {
	DB         *store.DB
	Tools      *tools.Registry
	LLMClients map[string]llm.Client
	DefaultLLM llm.Client
	DLQ        DLQFreezer
	Broker     *EventBroker
}

// NewExecutor creates a new workflow engine coordinator.
func NewExecutor(db *store.DB, dlq DLQFreezer) *Executor {
	var defaultLLM llm.Client = llm.NewMockClient()
	if os.Getenv("OPENAI_API_KEY") != "" {
		defaultLLM = llm.NewOpenAIClient("", "")
	}

	return &Executor{
		DB:         db,
		Tools:      tools.NewRegistry(),
		LLMClients: make(map[string]llm.Client),
		DefaultLLM: defaultLLM,
		DLQ:        dlq,
		Broker:     NewEventBroker(),
	}
}

// getLLMClient selects the appropriate provider client based on workflow model config.
func (e *Executor) getLLMClient(modelKey string, models map[string]ModelConfig) llm.Client {
	if cfg, ok := models[modelKey]; ok {
		switch cfg.Provider {
		case "openai":
			apiKey := ""
			if cfg.APIKeyEnv != "" {
				apiKey = os.Getenv(cfg.APIKeyEnv)
			}
			if apiKey != "" || os.Getenv("OPENAI_API_KEY") != "" {
				return llm.NewOpenAIClient(apiKey, cfg.BaseURL)
			}
		case "anthropic":
			apiKey := ""
			if cfg.APIKeyEnv != "" {
				apiKey = os.Getenv(cfg.APIKeyEnv)
			}
			if apiKey != "" || os.Getenv("ANTHROPIC_API_KEY") != "" {
				return llm.NewAnthropicClient(apiKey, cfg.BaseURL)
			}
		case "ollama":
			return llm.NewOllamaClient(cfg.BaseURL)
		case "mock":
			return llm.NewMockClient()
		}
	}

	return e.DefaultLLM
}

// Start creates and executes a workflow from a stored workflow specification.
func (e *Executor) Start(ctx context.Context, workflowID string, initialInput map[string]any) (*Execution, error) {
	spec, _, err := e.DB.GetWorkflow(workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow %s: %w", workflowID, err)
	}

	return e.StartSpec(ctx, spec, initialInput)
}

// StartSpec creates and executes a workflow from an in-memory WorkflowSpec.
func (e *Executor) StartSpec(ctx context.Context, spec *WorkflowSpec, initialInput map[string]any) (*Execution, error) {
	dag, err := BuildDAG(spec)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow specification: %w", err)
	}

	// Ensure workflow specification is recorded in the store for foreign key integrity
	if _, _, err := e.DB.GetWorkflow(spec.Workflow.ID); err != nil {
		_ = e.DB.SaveWorkflow(spec, "")
	}

	execID := "exec_" + uuid.New().String()[:8]
	state := NewState(spec.State, initialInput)

	exec := &Execution{
		ID:              execID,
		WorkflowID:      spec.Workflow.ID,
		Status:          StatusRunning,
		CurrentNodeID:   dag.Roots[0],
		State:           state.ToMap(),
		HealingAttempts: 0,
		StartedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := e.DB.CreateExecution(exec); err != nil {
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	e.Broker.Publish(ExecutionEvent{
		ExecutionID: exec.ID,
		WorkflowID:  exec.WorkflowID,
		Type:        EventWorkflowStarted,
		Status:      string(StatusRunning),
		Message:     fmt.Sprintf("Workflow execution %s started", exec.ID),
	})

	return e.runFromNode(ctx, dag, exec, state, dag.Roots[0], "")
}

// Resume continues an execution that was paused in the Dead-Letter Queue.
func (e *Executor) Resume(ctx context.Context, dlqID string, operatorInstruction string, statePatch map[string]any, retryCurrentNode bool) (*Execution, error) {
	dlqItem, err := e.DLQ.GetItem(dlqID)
	if err != nil {
		return nil, fmt.Errorf("dlq item not found: %w", err)
	}

	exec, err := e.DB.GetExecution(dlqItem.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("execution not found for dlq item: %w", err)
	}

	spec, _, err := e.DB.GetWorkflow(exec.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow spec not found for execution %s: %w", exec.WorkflowID, err)
	}

	dag, err := BuildDAG(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to build dag: %w", err)
	}

	state := NewState(exec.State)
	if len(statePatch) > 0 {
		state.ApplyPatch(statePatch)
		exec.State = state.ToMap()
	}

	action := "RESUME"
	if retryCurrentNode {
		action = "RETRY_NODE"
	}
	_ = e.DLQ.ResolveItem(dlqID, "operator", action, operatorInstruction)

	exec.Status = StatusRunning
	exec.HealingAttempts = 0
	_ = e.DB.UpdateExecution(exec)

	e.Broker.Publish(ExecutionEvent{
		ExecutionID: exec.ID,
		WorkflowID:  exec.WorkflowID,
		Type:        EventExecutionResumed,
		Status:      string(StatusRunning),
		NodeID:      dlqItem.FailedNodeID,
		Message:     fmt.Sprintf("Resumed by operator. Instruction: %s", operatorInstruction),
	})

	startNodeID := dlqItem.FailedNodeID
	if !retryCurrentNode {
		// Advance to next node
		nextNodes := dag.Outgoing[dlqItem.FailedNodeID]
		if len(nextNodes) > 0 {
			startNodeID = nextNodes[0]
		} else {
			// No further nodes, mark complete
			now := time.Now()
			exec.Status = StatusCompleted
			exec.CompletedAt = &now
			_ = e.DB.UpdateExecution(exec)
			return exec, nil
		}
	}

	return e.runFromNode(ctx, dag, exec, state, startNodeID, operatorInstruction)
}

// Rewind rolls back an execution's state to a previous node's snapshot and resumes execution.
func (e *Executor) Rewind(ctx context.Context, executionID string, targetNodeID string) (*Execution, error) {
	snapshots, err := e.DB.GetSnapshots(executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch snapshots: %w", err)
	}

	var targetSnap *StateSnapshot
	for i := len(snapshots) - 1; i >= 0; i-- {
		if snapshots[i].NodeID == targetNodeID {
			targetSnap = &snapshots[i]
			break
		}
	}

	if targetSnap == nil {
		return nil, fmt.Errorf("no state snapshot found for node %s in execution %s", targetNodeID, executionID)
	}

	exec, err := e.DB.GetExecution(executionID)
	if err != nil {
		return nil, fmt.Errorf("execution not found: %w", err)
	}

	spec, _, err := e.DB.GetWorkflow(exec.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow spec not found: %w", err)
	}

	dag, err := BuildDAG(spec)
	if err != nil {
		return nil, fmt.Errorf("dag build error: %w", err)
	}

	// Restore state from snapshot
	state := NewState()
	var envelope struct {
		State map[string]any `json:"state"`
	}
	if err := json.Unmarshal([]byte(targetSnap.StateJSON), &envelope); err == nil && len(envelope.State) > 0 {
		state = NewState(envelope.State)
	} else {
		_ = state.FromJSON(targetSnap.StateJSON)
	}

	exec.State = state.ToMap()
	exec.Status = StatusRunning
	exec.CurrentNodeID = targetNodeID
	exec.HealingAttempts = 0
	_ = e.DB.UpdateExecution(exec)

	return e.runFromNode(ctx, dag, exec, state, targetNodeID, "Rewound to previous checkpoint")
}

// runFromNode processes nodes sequentially starting from the given node ID.
func (e *Executor) runFromNode(
	ctx context.Context,
	dag *DAG,
	exec *Execution,
	state *State,
	startNodeID string,
	operatorInstruction string,
) (*Execution, error) {
	currentNodeID := startNodeID
	maxHealing := dag.Spec.Workflow.MaxHealingAttempts
	if maxHealing <= 0 {
		maxHealing = 3
	}

	stepSequence := 0

	for currentNodeID != "" {
		node, exists := dag.GetNode(currentNodeID)
		if !exists {
			return exec, fmt.Errorf("node not found in DAG: %s", currentNodeID)
		}

		exec.CurrentNodeID = node.ID
		exec.State = state.ToMap()
		_ = e.DB.UpdateExecution(exec)

		e.Broker.Publish(ExecutionEvent{
			ExecutionID: exec.ID,
			WorkflowID:  exec.WorkflowID,
			Type:        EventNodeStart,
			NodeID:      node.ID,
			Status:      string(NodeStatusRunning),
			Message:     fmt.Sprintf("Executing node %s (%s)", node.ID, node.Type),
		})

		stepSequence++
		startTime := time.Now()

		// Save state snapshot prior to node mutation
		stateJSON, _ := state.ToJSON()
		_ = e.DB.SaveSnapshot(&StateSnapshot{
			ExecutionID:  exec.ID,
			NodeID:       node.ID,
			StateJSON:    stateJSON,
			StepSequence: stepSequence,
		})

		var nextNodeID string
		var nodeErr error
		var outputPayload string
		var tokensConsumed int
		var nodeStatus = NodeStatusSuccess

		switch node.Type {
		case NodeTypeLLMAgent:
			outputPayload, tokensConsumed, nodeStatus, nodeErr = e.executeLLMNode(
				ctx, dag, exec, state, node, maxHealing, operatorInstruction,
			)
			operatorInstruction = "" // consume instruction

		case NodeTypeToolExecution:
			outputPayload, nodeStatus, nodeErr = e.executeToolNode(ctx, dag, exec, state, node)

		case NodeTypeConditionalRouter:
			outputPayload, nextNodeID, nodeErr = e.executeRouterNode(ctx, state, node)

		case NodeTypeHumanCheckpoint:
			// Explicit human pause checkpoint: freeze to DLQ immediately
			_, _ = e.DLQ.FreezeToDLQ(ctx, exec, node.ID, "Manual Human Review Required", "Node is configured as an explicit human_checkpoint gate", dag.Spec.Notifications)
			e.Broker.Publish(ExecutionEvent{
				ExecutionID: exec.ID,
				WorkflowID:  exec.WorkflowID,
				Type:        EventDLQPaused,
				NodeID:      node.ID,
				Status:      string(StatusPausedDLQ),
				Message:     fmt.Sprintf("Paused at human checkpoint gate: %s", node.ID),
			})
			return exec, nil

		case NodeTypeTransform:
			outputPayload, nodeErr = e.executeTransformNode(ctx, state, node)

		default:
			nodeErr = fmt.Errorf("unsupported node type: %s", node.Type)
		}

		latencyMs := time.Since(startTime).Milliseconds()

		// Record audit log
		inputJSON, _ := json.Marshal(node.Input)
		_ = e.DB.AppendAuditLog(&StepAuditEntry{
			ExecutionID:    exec.ID,
			NodeID:         node.ID,
			NodeType:       node.Type,
			InputPayload:   string(inputJSON),
			OutputPayload:  outputPayload,
			TokensConsumed: tokensConsumed,
			LatencyMs:      latencyMs,
			Status:         nodeStatus,
			Timestamp:      time.Now(),
		})

		// If execution was paused to DLQ, halt traversal immediately
		if exec.Status == StatusPausedDLQ {
			return exec, nil
		}

		if nodeErr != nil {

			// Execution terminal failure
			exec.Status = StatusFailed
			now := time.Now()
			exec.CompletedAt = &now
			_ = e.DB.UpdateExecution(exec)

			e.Broker.Publish(ExecutionEvent{
				ExecutionID: exec.ID,
				WorkflowID:  exec.WorkflowID,
				Type:        EventWorkflowFailed,
				NodeID:      node.ID,
				Status:      string(StatusFailed),
				Message:     nodeErr.Error(),
			})
			return exec, nodeErr
		}

		e.Broker.Publish(ExecutionEvent{
			ExecutionID: exec.ID,
			WorkflowID:  exec.WorkflowID,
			Type:        EventNodeComplete,
			NodeID:      node.ID,
			Status:      string(nodeStatus),
			Message:     fmt.Sprintf("Node %s completed in %dms", node.ID, latencyMs),
		})

		// Determine next node
		if nextNodeID != "" {
			currentNodeID = nextNodeID
		} else if node.Next != "" {
			currentNodeID = node.Next
		} else {
			outgoing := dag.Outgoing[node.ID]
			if len(outgoing) > 0 {
				currentNodeID = outgoing[0]
			} else {
				currentNodeID = "" // Reached terminal node
			}
		}
	}

	// Workflow successfully completed
	now := time.Now()
	exec.Status = StatusCompleted
	exec.CompletedAt = &now
	exec.State = state.ToMap()
	_ = e.DB.UpdateExecution(exec)

	e.Broker.Publish(ExecutionEvent{
		ExecutionID: exec.ID,
		WorkflowID:  exec.WorkflowID,
		Type:        EventWorkflowCompleted,
		Status:      string(StatusCompleted),
		Message:     fmt.Sprintf("Workflow execution %s completed successfully", exec.ID),
	})

	return exec, nil
}

// executeLLMNode handles prompting, structured validation, confidence sentinels, and self-healing.
func (e *Executor) executeLLMNode(
	ctx context.Context,
	dag *DAG,
	exec *Execution,
	state *State,
	node *NodeSpec,
	maxHealing int,
	operatorInstruction string,
) (outputStr string, tokens int, status NodeStatus, err error) {
	client := e.getLLMClient("primary", dag.Spec.Models)
	healingClient := e.getLLMClient("healing", dag.Spec.Models)

	interpolatedPrompt, err := Interpolate(node.Prompt, state)
	if err != nil {
		interpolatedPrompt = node.Prompt
	}

	if operatorInstruction != "" {
		interpolatedPrompt = fmt.Sprintf("%s\n\n[Operator Directive]: %s", interpolatedPrompt, operatorInstruction)
	}

	currentPrompt := interpolatedPrompt
	attempt := 0
	totalTokens := 0
	var lastRawOutput string

	for {
		attempt++
		currentClient := client
		if attempt > 1 {
			currentClient = healingClient
		}

		genResp, genErr := currentClient.Generate(ctx, llm.GenerateRequest{
			Model:        "primary",
			Prompt:       currentPrompt,
			Temperature:  0.1,
			OutputSchema: node.OutputSchema,
		})

		if genResp != nil {
			totalTokens += genResp.TokensConsumed
			lastRawOutput = genResp.Content
		}

		var parsedData map[string]any
		var valErrors []string
		var parseErr error

		if genResp != nil && node.OutputSchema != nil {
			parsedData, valErrors, parseErr = llm.ParseAndValidate(genResp.Content, node.OutputSchema)
		} else if genResp != nil {
			parsedData = genResp.StructuredOutput
		}

		confidence := 1.0
		if genResp != nil {
			confidence = llm.ComputeConfidence(genResp.Content, parsedData, valErrors)
		}

		// Sentinel checks
		isSchemaFailed := len(valErrors) > 0 || parseErr != nil
		isConfidenceFailed := node.ConfidenceThreshold > 0 && confidence < node.ConfidenceThreshold
		isSuccess := genErr == nil && !isSchemaFailed && !isConfidenceFailed

		if isSuccess {
			// Success! Save to state
			if node.SaveTo != "" && parsedData != nil {
				state.Set(node.SaveTo, parsedData)
			} else if node.SaveTo != "" && genResp != nil {
				state.Set(node.SaveTo, genResp.Content)
			}

			finalStatus := NodeStatusSuccess
			if attempt > 1 {
				finalStatus = NodeStatusHealed
			}

			outBytes, _ := json.Marshal(parsedData)
			if len(outBytes) == 0 && genResp != nil {
				outBytes = []byte(genResp.Content)
			}
			return string(outBytes), totalTokens, finalStatus, nil
		}

		// Self-healing or DLQ pause required
		errCategory := healing.Classify(genErr, valErrors, confidence, node.ConfidenceThreshold)
		errDetails := ""
		if genErr != nil {
			errDetails = genErr.Error()
		} else if len(valErrors) > 0 {
			errDetails = fmt.Sprintf("Schema validation failures: %v", valErrors)
		} else if isConfidenceFailed {
			errDetails = fmt.Sprintf("Confidence score %.2f is below threshold %.2f", confidence, node.ConfidenceThreshold)
		}

		// Check low confidence sentinel rule
		if isConfidenceFailed && node.OnLowConfidence != nil && node.OnLowConfidence.Action == "pause_to_dlq" {
			_, _ = e.DLQ.FreezeToDLQ(ctx, exec, node.ID, "Low Confidence Threshold Reached", errDetails, dag.Spec.Notifications)
			return lastRawOutput, totalTokens, NodeStatusPaused, nil
		}

		// Check if healing attempts exceeded
		if attempt > maxHealing {
			// Log exhausted attempt
			_ = e.DB.AppendHealingLog(&HealingLogEntry{
				ExecutionID:   exec.ID,
				NodeID:        node.ID,
				AttemptNumber: attempt,
				ErrorCategory: string(errCategory),
				OriginalError: errDetails,
				MutatedPrompt: currentPrompt,
				Status:        "EXHAUSTED",
			})

			// Freeze state to Dead-Letter Queue
			_, _ = e.DLQ.FreezeToDLQ(ctx, exec, node.ID, "Healing Attempts Exhausted", errDetails, dag.Spec.Notifications)
			return lastRawOutput, totalTokens, NodeStatusFailed, nil
		}

		// Synthesize surgical repair prompt
		customHint := ""
		if node.OnFailure != nil {
			customHint = node.OnFailure.PromptRepair
		}

		mutatedPrompt := healing.SynthesizeRepairPrompt(
			interpolatedPrompt,
			lastRawOutput,
			errCategory,
			errDetails,
			valErrors,
			node.OutputSchema,
			customHint,
		)

		// Record healing attempt in database
		_ = e.DB.AppendHealingLog(&HealingLogEntry{
			ExecutionID:   exec.ID,
			NodeID:        node.ID,
			AttemptNumber: attempt,
			ErrorCategory: string(errCategory),
			OriginalError: errDetails,
			MutatedPrompt: mutatedPrompt,
			Status:        "FAILED",
		})

		e.Broker.Publish(ExecutionEvent{
			ExecutionID: exec.ID,
			WorkflowID:  exec.WorkflowID,
			Type:        EventSelfHealing,
			NodeID:      node.ID,
			Status:      string(NodeStatusRunning),
			Message:     fmt.Sprintf("Self-healing attempt %d/%d for node %s (%s)", attempt, maxHealing, node.ID, errCategory),
		})

		// Backoff briefly between self-healing attempts
		backoffDuration := time.Duration(math.Pow(2, float64(attempt-1))*100) * time.Millisecond
		time.Sleep(backoffDuration)

		currentPrompt = mutatedPrompt
	}
}

// executeToolNode runs a deterministic tool with retry and fallback capabilities.
func (e *Executor) executeToolNode(
	ctx context.Context,
	dag *DAG,
	exec *Execution,
	state *State,
	node *NodeSpec,
) (outputStr string, status NodeStatus, err error) {
	interpolatedParams, err := InterpolateValue(node.Params, state)
	if err != nil {
		interpolatedParams = node.Params
	}

	paramsMap, ok := interpolatedParams.(map[string]any)
	if !ok {
		paramsMap = node.Params
	}

	maxAttempts := 1
	if node.Retry != nil && node.Retry.MaxAttempts > 1 {
		maxAttempts = node.Retry.MaxAttempts
	}

	var lastErr error
	var result any

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, lastErr = e.Tools.Execute(ctx, node.Tool, paramsMap)
		if lastErr == nil {
			break
		}

		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}
	}

	// If primary tool failed and fallback tool is configured
	if lastErr != nil && node.OnFailure != nil && node.OnFailure.FallbackTool != "" {
		fallbackTool := node.OnFailure.FallbackTool
		result, lastErr = e.Tools.Execute(ctx, fallbackTool, paramsMap)
	}

	if lastErr != nil {
		// Tool failed: freeze to DLQ
		_, _ = e.DLQ.FreezeToDLQ(ctx, exec, node.ID, "Tool Execution Failed", lastErr.Error(), dag.Spec.Notifications)
		return "", NodeStatusFailed, lastErr
	}

	if node.SaveTo != "" && result != nil {
		state.Set(node.SaveTo, result)
	}

	bytes, _ := json.Marshal(result)
	return string(bytes), NodeStatusSuccess, nil
}

// executeRouterNode evaluates branch conditions to select the next node in the DAG.
func (e *Executor) executeRouterNode(
	ctx context.Context,
	state *State,
	node *NodeSpec,
) (outputStr string, nextNodeID string, err error) {
	for _, cond := range node.Conditions {
		matches, err := EvaluateCondition(cond.Condition, state)
		if err == nil && matches {
			result := map[string]any{
				"selected_condition": cond.Condition,
				"next_node":          cond.Next,
			}
			bytes, _ := json.Marshal(result)
			return string(bytes), cond.Next, nil
		}
	}

	return "", "", fmt.Errorf("no router condition evaluated to true for node %s", node.ID)
}

// executeTransformNode runs expression-based data transformations.
func (e *Executor) executeTransformNode(
	ctx context.Context,
	state *State,
	node *NodeSpec,
) (outputStr string, err error) {
	paramsCopy := make(map[string]any, len(node.Params)+1)
	for k, v := range node.Params {
		paramsCopy[k] = v
	}
	paramsCopy["state"] = state.ToMap()

	result, err := tools.ExecuteJSONTransform(ctx, paramsCopy)
	if err != nil {
		return "", err
	}

	if node.SaveTo != "" && result != nil {
		state.Set(node.SaveTo, result)
	}

	bytes, _ := json.Marshal(result)
	return string(bytes), nil
}
