package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/types"
	"gopkg.in/yaml.v3"
)

type WorkflowSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	NodeCount int       `json:"node_count"`
	Schedule  string    `json:"schedule,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (db *DB) SaveWorkflow(spec *types.WorkflowSpec, rawYAML string) error {
	if rawYAML == "" && spec != nil {
		bytes, err := yaml.Marshal(spec)
		if err == nil {
			rawYAML = string(bytes)
		}
	}

	query := `
	INSERT INTO workflows (id, name, spec_yaml, updated_at)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(id) DO UPDATE SET
		name = excluded.name,
		spec_yaml = excluded.spec_yaml,
		updated_at = CURRENT_TIMESTAMP;
	`
	_, err := db.Exec(query, spec.Workflow.ID, spec.Workflow.Name, rawYAML)
	if err != nil {
		return fmt.Errorf("failed to save workflow: %w", err)
	}
	return nil
}

func (db *DB) GetWorkflow(id string) (*types.WorkflowSpec, string, error) {
	query := `SELECT spec_yaml FROM workflows WHERE id = ?`
	var rawYAML string
	err := db.QueryRow(query, id).Scan(&rawYAML)
	if err == sql.ErrNoRows {
		return nil, "", fmt.Errorf("workflow not found: %s", id)
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to get workflow: %w", err)
	}

	var spec types.WorkflowSpec
	if err := yaml.Unmarshal([]byte(rawYAML), &spec); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal workflow spec: %w", err)
	}

	return &spec, rawYAML, nil
}

func (db *DB) ListWorkflows() ([]WorkflowSummary, error) {
	query := `SELECT id, name, spec_yaml, created_at, updated_at FROM workflows ORDER BY updated_at DESC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var list []WorkflowSummary
	for rows.Next() {
		var s WorkflowSummary
		var rawYAML string
		if err := rows.Scan(&s.ID, &s.Name, &rawYAML, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		var spec types.WorkflowSpec
		if err := yaml.Unmarshal([]byte(rawYAML), &spec); err == nil {
			s.NodeCount = len(spec.Nodes)
			s.Schedule = spec.Workflow.Schedule
		}
		list = append(list, s)
	}

	return list, nil
}

func (db *DB) DeleteWorkflow(id string) error {
	query := `DELETE FROM workflows WHERE id = ?`
	res, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("workflow not found: %s", id)
	}
	return nil
}
