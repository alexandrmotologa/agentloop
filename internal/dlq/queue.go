package dlq

import (
	"context"
	"fmt"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
	"github.com/google/uuid"
)

// Manager coordinates DLQ freezing, storage, and resumption.
type Manager struct {
	db         *store.DB
	dispatcher *Dispatcher
}

// NewManager creates a new DLQ manager.
func NewManager(db *store.DB) *Manager {
	return &Manager{
		db:         db,
		dispatcher: NewDispatcher(),
	}
}

// FreezeToDLQ suspends execution and persists state into DLQ and state snapshots.
func (m *Manager) FreezeToDLQ(
	ctx context.Context,
	exec *engine.Execution,
	failedNodeID string,
	reason string,
	errDetails string,
	notifications engine.NotificationConfig,
) (*engine.DLQItem, error) {
	snapshotJSON, err := CreateSnapshotEnvelope(exec, failedNodeID, reason, errDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	dlqID := "dlq_" + uuid.New().String()[:8]
	item := &engine.DLQItem{
		ID:                dlqID,
		ExecutionID:       exec.ID,
		FailedNodeID:      failedNodeID,
		FailureReason:     reason,
		ErrorDetails:      errDetails,
		SnapshotStateJSON: snapshotJSON,
		CreatedAt:         time.Now(),
	}

	if err := m.db.InsertDLQItem(item); err != nil {
		return nil, fmt.Errorf("failed to store dlq item: %w", err)
	}

	// Save snapshot for state history / time-travel
	snap := &engine.StateSnapshot{
		ExecutionID:  exec.ID,
		NodeID:       failedNodeID,
		StateJSON:    snapshotJSON,
		StepSequence: exec.HealingAttempts,
	}
	_ = m.db.SaveSnapshot(snap)

	// Update execution status
	exec.Status = engine.StatusPausedDLQ
	exec.CurrentNodeID = failedNodeID
	if err := m.db.UpdateExecution(exec); err != nil {
		return nil, fmt.Errorf("failed to update execution status to PAUSED_DLQ: %w", err)
	}

	// Dispatch alerts
	m.dispatcher.Dispatch(ctx, notifications, item)

	return item, nil
}

// GetItem retrieves a DLQ item by ID.
func (m *Manager) GetItem(id string) (*engine.DLQItem, error) {
	return m.db.GetDLQItem(id)
}

// ListItems retrieves DLQ items by status (e.g. "PENDING", "RESOLVED", "").
func (m *Manager) ListItems(status string) ([]engine.DLQItem, error) {
	return m.db.ListDLQItems(status)
}

// ResolveItem records operator resolution on the DLQ entry.
func (m *Manager) ResolveItem(id string, resolvedBy string, action string, instruction string) error {
	return m.db.ResolveDLQItem(id, resolvedBy, action, instruction)
}
