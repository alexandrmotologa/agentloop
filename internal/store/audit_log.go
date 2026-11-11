package store

import (
	"fmt"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/types"
)

func (db *DB) AppendAuditLog(entry *types.StepAuditEntry) error {
	query := `
	INSERT INTO step_audit_log (
		execution_id, node_id, node_type, input_payload, output_payload,
		tokens_consumed, latency_ms, status, timestamp
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	if entry.Timestamp.IsZero() {
		entry.Timestamp = now
	}

	res, err := db.Exec(query,
		entry.ExecutionID,
		entry.NodeID,
		string(entry.NodeType),
		entry.InputPayload,
		entry.OutputPayload,
		entry.TokensConsumed,
		entry.LatencyMs,
		string(entry.Status),
		entry.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to append step audit log: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		entry.ID = id
	}
	return nil
}

func (db *DB) GetAuditLogs(executionID string) ([]types.StepAuditEntry, error) {
	query := `
	SELECT id, execution_id, node_id, node_type, input_payload, output_payload,
	       tokens_consumed, latency_ms, status, timestamp
	FROM step_audit_log
	WHERE execution_id = ?
	ORDER BY id ASC
	`
	rows, err := db.Query(query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	var entries []types.StepAuditEntry
	for rows.Next() {
		var e types.StepAuditEntry
		var nodeTypeStr, statusStr string

		if err := rows.Scan(
			&e.ID,
			&e.ExecutionID,
			&e.NodeID,
			&nodeTypeStr,
			&e.InputPayload,
			&e.OutputPayload,
			&e.TokensConsumed,
			&e.LatencyMs,
			&statusStr,
			&e.Timestamp,
		); err != nil {
			return nil, err
		}

		e.NodeType = types.NodeType(nodeTypeStr)
		e.Status = types.NodeStatus(statusStr)
		entries = append(entries, e)
	}

	return entries, nil
}
