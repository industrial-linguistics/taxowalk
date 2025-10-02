package history

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

type ClassificationRecord struct {
	ID               int64
	Timestamp        time.Time
	ProductDesc      string
	Category         string
	CategoryID       string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

func Open(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS classifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		product_description TEXT NOT NULL,
		category_name TEXT,
		category_id TEXT,
		prompt_tokens INTEGER DEFAULT 0,
		completion_tokens INTEGER DEFAULT 0,
		total_tokens INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON classifications(timestamp);
	`
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	return nil
}

func (d *DB) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *DB) RecordClassification(productDesc, categoryName, categoryID string, promptTokens, completionTokens, totalTokens int) error {
	_, err := d.db.Exec(`
		INSERT INTO classifications (product_description, category_name, category_id, prompt_tokens, completion_tokens, total_tokens)
		VALUES (?, ?, ?, ?, ?, ?)`,
		productDesc, categoryName, categoryID, promptTokens, completionTokens, totalTokens,
	)
	if err != nil {
		return fmt.Errorf("failed to record classification: %w", err)
	}
	return nil
}

func (d *DB) GetTotalTokens() (int64, error) {
	var total int64
	err := d.db.QueryRow("SELECT COALESCE(SUM(total_tokens), 0) FROM classifications").Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total tokens: %w", err)
	}
	return total, nil
}

func (d *DB) GetTokensLast24Hours() (int64, error) {
	cutoff := time.Now().Add(-24 * time.Hour)
	var total int64
	err := d.db.QueryRow(`
		SELECT COALESCE(SUM(total_tokens), 0)
		FROM classifications
		WHERE timestamp >= ?`,
		cutoff,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get tokens for last 24 hours: %w", err)
	}
	return total, nil
}

func (d *DB) GetAllRecords() ([]ClassificationRecord, error) {
	rows, err := d.db.Query(`
		SELECT id, timestamp, product_description,
		       COALESCE(category_name, ''), COALESCE(category_id, ''),
		       prompt_tokens, completion_tokens, total_tokens
		FROM classifications
		ORDER BY timestamp DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var records []ClassificationRecord
	for rows.Next() {
		var r ClassificationRecord
		err := rows.Scan(&r.ID, &r.Timestamp, &r.ProductDesc, &r.Category, &r.CategoryID,
			&r.PromptTokens, &r.CompletionTokens, &r.TotalTokens)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
