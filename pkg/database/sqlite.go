package database

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// NewSQLiteDB creates and initializes a SQLite database connection.
// If the database file doesn't exist or is empty, it initializes the schema.
func NewSQLiteDB(path string) (*sql.DB, error) {
	if path == "" {
		path = ":memory:"
	}

	// Use pragmas for improved concurrency and reliability
	dsn := path + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize schema (idempotent)
	if err := InitializeSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// InitializeSchema executes the embedded SQL schema in an idempotent way.
func InitializeSchema(db *sql.DB) error {
	statements := splitSQLStatements(schemaSQL)
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement %d: %w", i+1, err)
		}
	}
	return nil
}

// splitSQLStatements splits a multi-statement SQL string into individual statements.
// It handles comments and blank lines and returns complete statements ending with semicolons.
func splitSQLStatements(sqlText string) []string {
	var stmts []string
	var cur strings.Builder
	lines := strings.Split(sqlText, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		cur.WriteString(line)
		cur.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			stmts = append(stmts, cur.String())
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		rem := strings.TrimSpace(cur.String())
		if rem != "" {
			stmts = append(stmts, rem)
		}
	}
	return stmts
}
