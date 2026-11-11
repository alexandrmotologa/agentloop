package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/types"
)

func (db *DB) CreateExecution(exec *types.Execution) error {
	stateJSON, err := json.Marshal(exec.State)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	query := `
	INSERT INTO executions (
		id, workflow_id, status, current_node_id, state_json, healing_attempts,
		total_tokens, total_cost_usd, started_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	if exec.StartedAt.IsZero() {
		exec.StartedAt = now
	}
	exec.UpdatedAt = now

	_, err = db.Exec(query,
		exec.ID,
		exec.WorkflowID,
		string(exec.Status),
		exec.CurrentNodeID,
		string(stateJSON),
		exec.HealingAttempts,
		exec.TotalTokens,
		exec.TotalCostUSD,
		exec.StartedAt,
		exec.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert execution: %w", err)
	}
	return nil
}

func (db *DB) UpdateExecution(exec *types.Execution) error {
	stateJSON, err := json.Marshal(exec.State)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	query := `
	UPDATE executions SET
		status = ?,
		current_node_id = ?,
		state_json = ?,
		healing_attempts = ?,
		total_tokens = ?,
		total_cost_usd = ?,
		updated_at = ?,
		completed_at = ?
	WHERE id = ?
	`
	exec.UpdatedAt = time.Now()

	_, err = db.Exec(query,
		string(exec.Status),
		exec.CurrentNodeID,
		string(stateJSON),
		exec.HealingAttempts,
		exec.TotalTokens,
		exec.TotalCostUSD,
		exec.UpdatedAt,
		exec.CompletedAt,
		exec.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}
	return nil
}

func (db *DB) GetExecution(id string) (*types.Execution, error) {
	query := `
	SELECT id, workflow_id, status, current_node_id, state_json, healing_attempts,
	       total_tokens, total_cost_usd, started_at, updated_at, completed_at
	FROM executions
	WHERE id = ?
	`
	var exec types.Execution
	var statusStr, stateJSON string
	var completedAt sql.NullTime

	err := db.QueryRow(query, id).Scan(
		&exec.ID,
		&exec.WorkflowID,
		&statusStr,
		&exec.CurrentNodeID,
		&stateJSON,
		&exec.HealingAttempts,
		&exec.TotalTokens,
		&exec.TotalCostUSD,
		&exec.StartedAt,
		&exec.UpdatedAt,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	exec.Status = types.ExecutionStatus(statusStr)
	if completedAt.Valid {
		exec.CompletedAt = &completedAt.Time
	}

	var stateMap map[string]any
	if err := json.Unmarshal([]byte(stateJSON), &stateMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execution state: %w", err)
	}
	exec.State = stateMap

	return &exec, nil
}

func (db *DB) ListExecutions(limit int, status string) ([]types.Execution, error) {
	if limit <= 0 {
		limit = 50
	}

	var rows *sql.Rows
	var err error

	if status != "" {
		query := `
		SELECT id, workflow_id, status, current_node_id, state_json, healing_attempts,
		       total_tokens, total_cost_usd, started_at, updated_at, completed_at
		FROM executions
		WHERE status = ?
		ORDER BY started_at DESC
		LIMIT ?
		`
		rows, err = db.Query(query, status, limit)
	} else {
		query := `
		SELECT id, workflow_id, status, current_node_id, state_json, healing_attempts,
		       total_tokens, total_cost_usd, started_at, updated_at, completed_at
		FROM executions
		ORDER BY started_at DESC
		LIMIT ?
		`
		rows, err = db.Query(query, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var list []types.Execution
	for rows.Next() {
		var exec types.Execution
		var statusStr, stateJSON string
		var completedAt sql.NullTime

		if err := rows.Scan(
			&exec.ID,
			&exec.WorkflowID,
			&statusStr,
			&exec.CurrentNodeID,
			&stateJSON,
			&exec.HealingAttempts,
			&exec.TotalTokens,
			&exec.TotalCostUSD,
			&exec.StartedAt,
			&exec.UpdatedAt,
			&completedAt,
		); err != nil {
			return nil, err
		}

		exec.Status = types.ExecutionStatus(statusStr)
		if completedAt.Valid {
			exec.CompletedAt = &completedAt.Time
		}

		var stateMap map[string]any
		if err := json.Unmarshal([]byte(stateJSON), &stateMap); err == nil {
			exec.State = stateMap
		}

		list = append(list, exec)
	}

	return list, nil
}

func (db *DB) SaveSnapshot(snap *types.StateSnapshot) error {
	query := `
	INSERT INTO state_snapshots (execution_id, node_id, state_json, step_sequence, created_at)
	VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err := db.Exec(query, snap.ExecutionID, snap.NodeID, snap.StateJSON, snap.StepSequence)
	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}
	return nil
}

func (db *DB) GetSnapshots(executionID string) ([]types.StateSnapshot, error) {
	query := `
	SELECT id, execution_id, node_id, state_json, step_sequence, created_at
	FROM state_snapshots
	WHERE execution_id = ?
	ORDER BY step_sequence ASC
	`
	rows, err := db.Query(query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []types.StateSnapshot
	for rows.Next() {
		var s types.StateSnapshot
		if err := rows.Scan(&s.ID, &s.ExecutionID, &s.NodeID, &s.StateJSON, &s.StepSequence, &s.CreatedAt); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, nil
}

// CheckAndSetIdempotency verifies if a webhook or trigger idempotency key has already been processed.
func (db *DB) CheckAndSetIdempotency(key string, workflowID string, executionID string) (bool, string, error) {
	var existingExecID string
	err := db.QueryRow("SELECT execution_id FROM idempotency_keys WHERE key = ?", key).Scan(&existingExecID)
	if err == nil {
		// Key already exists!
		return true, existingExecID, nil
	}
	if err != sql.ErrNoRows {
		return false, "", err
	}

	// Insert new key
	_, err = db.Exec("INSERT INTO idempotency_keys (key, workflow_id, execution_id) VALUES (?, ?, ?)", key, workflowID, executionID)
	if err != nil {
		return false, "", err
	}

	return false, "", nil
}
