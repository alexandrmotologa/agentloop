package dlq

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/engine"
)

// SnapshotEnvelope contains full metadata and state frozen at a checkpoint.
type SnapshotEnvelope struct {
	ExecutionID string         `json:"execution_id"`
	WorkflowID  string         `json:"workflow_id"`
	NodeID      string         `json:"node_id"`
	State       map[string]any `json:"state"`
	CapturedAt  time.Time      `json:"captured_at"`
	Reason      string         `json:"reason"`
	Error       string         `json:"error,omitempty"`
}

// CreateSnapshotEnvelope serializes the state and context into a JSON snapshot.
func CreateSnapshotEnvelope(exec *engine.Execution, nodeID string, reason string, errDetails string) (string, error) {
	envelope := SnapshotEnvelope{
		ExecutionID: exec.ID,
		WorkflowID:  exec.WorkflowID,
		NodeID:      nodeID,
		State:       exec.State,
		CapturedAt:  time.Now(),
		Reason:      reason,
		Error:       errDetails,
	}

	bytes, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("failed to marshal snapshot envelope: %w", err)
	}

	return string(bytes), nil
}
