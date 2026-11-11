package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/types"
)

func (db *DB) InsertDLQItem(item *types.DLQItem) error {
	query := `
	INSERT INTO dlq_items (
		id, execution_id, failed_node_id, failure_reason, error_details, snapshot_state_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}

	_, err := db.Exec(query,
		item.ID,
		item.ExecutionID,
		item.FailedNodeID,
		item.FailureReason,
		item.ErrorDetails,
		item.SnapshotStateJSON,
		item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert dlq item: %w", err)
	}
	return nil
}

func (db *DB) GetDLQItem(id string) (*types.DLQItem, error) {
	query := `
	SELECT id, execution_id, failed_node_id, failure_reason, error_details, snapshot_state_json,
	       resolved_by, resolution_action, operator_instruction, created_at, resolved_at
	FROM dlq_items
	WHERE id = ?
	`
	var item types.DLQItem
	var resolvedBy, resolutionAction, operatorInstruction sql.NullString
	var resolvedAt sql.NullTime

	err := db.QueryRow(query, id).Scan(
		&item.ID,
		&item.ExecutionID,
		&item.FailedNodeID,
		&item.FailureReason,
		&item.ErrorDetails,
		&item.SnapshotStateJSON,
		&resolvedBy,
		&resolutionAction,
		&operatorInstruction,
		&item.CreatedAt,
		&resolvedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("dlq item not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dlq item: %w", err)
	}

	if resolvedBy.Valid {
		item.ResolvedBy = resolvedBy.String
	}
	if resolutionAction.Valid {
		item.ResolutionAction = resolutionAction.String
	}
	if operatorInstruction.Valid {
		item.OperatorInstruction = operatorInstruction.String
	}
	if resolvedAt.Valid {
		item.ResolvedAt = &resolvedAt.Time
	}

	return &item, nil
}

func (db *DB) GetDLQItemByExecutionID(executionID string) (*types.DLQItem, error) {
	query := `
	SELECT id, execution_id, failed_node_id, failure_reason, error_details, snapshot_state_json,
	       resolved_by, resolution_action, operator_instruction, created_at, resolved_at
	FROM dlq_items
	WHERE execution_id = ?
	ORDER BY created_at DESC
	LIMIT 1
	`
	var item types.DLQItem
	var resolvedBy, resolutionAction, operatorInstruction sql.NullString
	var resolvedAt sql.NullTime

	err := db.QueryRow(query, executionID).Scan(
		&item.ID,
		&item.ExecutionID,
		&item.FailedNodeID,
		&item.FailureReason,
		&item.ErrorDetails,
		&item.SnapshotStateJSON,
		&resolvedBy,
		&resolutionAction,
		&operatorInstruction,
		&item.CreatedAt,
		&resolvedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no dlq item found for execution: %s", executionID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dlq item for execution: %w", err)
	}

	if resolvedBy.Valid {
		item.ResolvedBy = resolvedBy.String
	}
	if resolutionAction.Valid {
		item.ResolutionAction = resolutionAction.String
	}
	if operatorInstruction.Valid {
		item.OperatorInstruction = operatorInstruction.String
	}
	if resolvedAt.Valid {
		item.ResolvedAt = &resolvedAt.Time
	}

	return &item, nil
}

func (db *DB) ListDLQItems(status string) ([]types.DLQItem, error) {
	var query string
	var rows *sql.Rows
	var err error

	if status == "PENDING" {
		query = `
		SELECT id, execution_id, failed_node_id, failure_reason, error_details, snapshot_state_json,
		       resolved_by, resolution_action, operator_instruction, created_at, resolved_at
		FROM dlq_items
		WHERE resolved_at IS NULL
		ORDER BY created_at DESC
		`
		rows, err = db.Query(query)
	} else if status == "RESOLVED" {
		query = `
		SELECT id, execution_id, failed_node_id, failure_reason, error_details, snapshot_state_json,
		       resolved_by, resolution_action, operator_instruction, created_at, resolved_at
		FROM dlq_items
		WHERE resolved_at IS NOT NULL
		ORDER BY resolved_at DESC
		`
		rows, err = db.Query(query)
	} else {
		query = `
		SELECT id, execution_id, failed_node_id, failure_reason, error_details, snapshot_state_json,
		       resolved_by, resolution_action, operator_instruction, created_at, resolved_at
		FROM dlq_items
		ORDER BY created_at DESC
		`
		rows, err = db.Query(query)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list dlq items: %w", err)
	}
	defer rows.Close()

	var list []types.DLQItem
	for rows.Next() {
		var item types.DLQItem
		var resolvedBy, resolutionAction, operatorInstruction sql.NullString
		var resolvedAt sql.NullTime

		if err := rows.Scan(
			&item.ID,
			&item.ExecutionID,
			&item.FailedNodeID,
			&item.FailureReason,
			&item.ErrorDetails,
			&item.SnapshotStateJSON,
			&resolvedBy,
			&resolutionAction,
			&operatorInstruction,
			&item.CreatedAt,
			&resolvedAt,
		); err != nil {
			return nil, err
		}

		if resolvedBy.Valid {
			item.ResolvedBy = resolvedBy.String
		}
		if resolutionAction.Valid {
			item.ResolutionAction = resolutionAction.String
		}
		if operatorInstruction.Valid {
			item.OperatorInstruction = operatorInstruction.String
		}
		if resolvedAt.Valid {
			item.ResolvedAt = &resolvedAt.Time
		}

		list = append(list, item)
	}

	return list, nil
}

func (db *DB) ResolveDLQItem(id string, resolvedBy string, action string, instruction string) error {
	query := `
	UPDATE dlq_items SET
		resolved_by = ?,
		resolution_action = ?,
		operator_instruction = ?,
		resolved_at = CURRENT_TIMESTAMP
	WHERE id = ?
	`
	res, err := db.Exec(query, resolvedBy, action, instruction, id)
	if err != nil {
		return fmt.Errorf("failed to resolve dlq item: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("dlq item not found: %s", id)
	}
	return nil
}
