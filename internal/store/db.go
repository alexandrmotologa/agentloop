package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection.
type DB struct {
	*sql.DB
}

// OpenDB opens or creates an SQLite database in WAL mode and runs migrations.
func OpenDB(path string) (*DB, error) {
	if path != ":memory:" {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory %s: %w", dir, err)
		}
	}

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	// Optimize connection pool for SQLite
	db.SetMaxOpenConns(1)

	sdb := &DB{DB: db}
	if err := sdb.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return sdb, nil
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS workflows (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		spec_yaml TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS executions (
		id TEXT PRIMARY KEY,
		workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
		status TEXT NOT NULL,
		current_node_id TEXT,
		state_json TEXT NOT NULL,
		healing_attempts INTEGER DEFAULT 0,
		total_tokens INTEGER DEFAULT 0,
		total_cost_usd REAL DEFAULT 0.0,
		started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS state_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
		node_id TEXT NOT NULL,
		state_json TEXT NOT NULL,
		step_sequence INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS dlq_items (
		id TEXT PRIMARY KEY,
		execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
		failed_node_id TEXT NOT NULL,
		failure_reason TEXT NOT NULL,
		error_details TEXT,
		snapshot_state_json TEXT NOT NULL,
		resolved_by TEXT,
		resolution_action TEXT,
		operator_instruction TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		resolved_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS step_audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
		node_id TEXT NOT NULL,
		node_type TEXT NOT NULL,
		input_payload TEXT,
		output_payload TEXT,
		tokens_consumed INTEGER DEFAULT 0,
		latency_ms INTEGER NOT NULL,
		status TEXT NOT NULL,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS healing_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
		node_id TEXT NOT NULL,
		attempt_number INTEGER NOT NULL,
		error_category TEXT NOT NULL,
		original_error TEXT NOT NULL,
		mutated_prompt TEXT NOT NULL,
		status TEXT NOT NULL,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS idempotency_keys (
		key TEXT PRIMARY KEY,
		workflow_id TEXT NOT NULL,
		execution_id TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_executions_workflow ON executions(workflow_id);
	CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
	CREATE INDEX IF NOT EXISTS idx_dlq_execution ON dlq_items(execution_id);
	CREATE INDEX IF NOT EXISTS idx_audit_execution ON step_audit_log(execution_id);
	CREATE INDEX IF NOT EXISTS idx_healing_execution ON healing_logs(execution_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Dynamic column migrations if table already existed without new columns
	_, _ = db.Exec("ALTER TABLE executions ADD COLUMN total_tokens INTEGER DEFAULT 0;")
	_, _ = db.Exec("ALTER TABLE executions ADD COLUMN total_cost_usd REAL DEFAULT 0.0;")

	return nil
}
