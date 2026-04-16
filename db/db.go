package db

import (
	"database/sql"
	_ "embed"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// InitDB opens a SQLite database, configures it with WAL mode and foreign key constraints,
// and initializes the schema.
func InitDB(dbPath string) (*sql.DB, error) {
	// Open SQLite database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		db.Close()
		return nil, err
	}

	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, err
	}

	// One-time migration: drop legacy password-based utenti table so the new
	// email/magic-link schema below can be created fresh. Safe & idempotent:
	// runs only when the legacy password_hash column is still present.
	if legacy, err := hasColumn(db, "utenti", "password_hash"); err != nil {
		db.Close()
		return nil, err
	} else if legacy {
		if _, err := db.Exec("DROP TABLE IF EXISTS login_tokens; DROP TABLE utenti;"); err != nil {
			db.Close()
			return nil, err
		}
	}

	// Execute schema
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// hasColumn reports whether the given column exists on the given table.
// Returns false (no error) if the table itself doesn't exist yet.
func hasColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
