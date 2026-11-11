package store

import (
	"fmt"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/types"
)

func (db *DB) AppendHealingLog(entry *types.HealingLogEntry) error {
	query := `
	INSERT INTO healing_logs (
		execution_id, node_id, attempt_number, error_category, original_error,
		mutated_prompt, status, timestamp
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	if entry.Timestamp.IsZero() {
		entry.Timestamp = now
	}

	res, err := db.Exec(query,
		entry.ExecutionID,
		entry.NodeID,
		entry.AttemptNumber,
		entry.ErrorCategory,
		entry.OriginalError,
		entry.MutatedPrompt,
		entry.Status,
		entry.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to append healing log: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		entry.ID = id
	}
	return nil
}

func (db *DB) GetHealingLogs(executionID string) ([]types.HealingLogEntry, error) {
	query := `
	SELECT id, execution_id, node_id, attempt_number, error_category, original_error,
	       mutated_prompt, status, timestamp
	FROM healing_logs
	WHERE execution_id = ?
	ORDER BY id ASC
	`
	rows, err := db.Query(query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get healing logs: %w", err)
	}
	defer rows.Close()

	var entries []types.HealingLogEntry
	for rows.Next() {
		var e types.HealingLogEntry
		if err := rows.Scan(
			&e.ID,
			&e.ExecutionID,
			&e.NodeID,
			&e.AttemptNumber,
			&e.ErrorCategory,
			&e.OriginalError,
			&e.MutatedPrompt,
			&e.Status,
			&e.Timestamp,
		); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	return entries, nil
}
