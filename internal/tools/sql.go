package tools

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	sharedDB     *sql.DB
	sharedDBOnce sync.Once
)

func getSharedDemoDB() (*sql.DB, error) {
	var err error
	sharedDBOnce.Do(func() {
		sharedDB, err = sql.Open("sqlite", ":memory:?_pragma=busy_timeout(5000)")
		if err == nil {
			seedDemoERP(sharedDB)
		}
	})
	return sharedDB, err
}

func seedDemoERP(db *sql.DB) {
	schema := `
	CREATE TABLE IF NOT EXISTS purchase_orders (
		id TEXT PRIMARY KEY,
		vendor_tax_id TEXT NOT NULL,
		vendor_name TEXT NOT NULL,
		total_amount REAL NOT NULL,
		currency TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	INSERT OR IGNORE INTO purchase_orders (id, vendor_tax_id, vendor_name, total_amount, currency, status)
	VALUES
		('PO-2026-881', 'US-94827104', 'Acme Industrial Solutions Inc.', 4250.00, 'USD', 'APPROVED'),
		('PO-2026-882', 'US-11223344', 'Global Tech Supplies LLC', 12500.00, 'USD', 'APPROVED'),
		('PO-2026-883', 'DE-99887766', 'Bavaria Sensor Dynamics GmbH', 3400.00, 'EUR', 'PENDING');
	`
	_, _ = db.Exec(schema)
}

// ExecuteSQLiteQuery executes a SQL statement against an SQLite database.
func ExecuteSQLiteQuery(ctx context.Context, params map[string]any) (any, error) {
	queryVal, ok := params["query"].(string)
	if !ok || queryVal == "" {
		return nil, fmt.Errorf("sqlite_query requires 'query' parameter")
	}

	var db *sql.DB
	var err error

	dbPath, _ := params["db_path"].(string)
	if dbPath != "" && dbPath != ":memory:" {
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open database at %s: %w", dbPath, err)
		}
		defer db.Close()
	} else {
		db, err = getSharedDemoDB()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-memory database: %w", err)
		}
	}

	var args []any
	if argsVal, exists := params["args"]; exists && argsVal != nil {
		switch a := argsVal.(type) {
		case []any:
			args = a
		case []string:
			for _, s := range a {
				args = append(args, s)
			}
		default:
			args = append(args, a)
		}
	}

	isSelect := strings.HasPrefix(strings.TrimSpace(strings.ToUpper(queryVal)), "SELECT")
	if isSelect {
		rows, err := db.QueryContext(ctx, queryVal, args...)
		if err != nil {
			return nil, fmt.Errorf("sql query execution error: %w", err)
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("failed to read columns: %w", err)
		}

		var results []map[string]any
		for rows.Next() {
			columns := make([]any, len(cols))
			columnPointers := make([]any, len(cols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			if err := rows.Scan(columnPointers...); err != nil {
				return nil, fmt.Errorf("failed to scan row: %w", err)
			}

			rowMap := make(map[string]any)
			for i, colName := range cols {
				val := columns[i]
				if b, ok := val.([]byte); ok {
					rowMap[colName] = string(b)
				} else {
					rowMap[colName] = val
				}
			}
			results = append(results, rowMap)
		}

		if len(results) == 1 {
			// Also return the single row as top-level map for easy template and expression access
			return results[0], nil
		}

		return results, nil
	}

	// Exec statement (INSERT/UPDATE/DELETE)
	res, err := db.ExecContext(ctx, queryVal, args...)
	if err != nil {
		return nil, fmt.Errorf("sql exec error: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	lastInsertID, _ := res.LastInsertId()

	return map[string]any{
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertID,
	}, nil
}
