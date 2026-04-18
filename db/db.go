package db

import (
	"database/sql"
	_ "embed"
	"strings"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, err
	}

	// Migrazione 1: drop tabella utenti con vecchio schema password-based.
	if legacy, err := hasColumn(db, "utenti", "password_hash"); err != nil {
		db.Close()
		return nil, err
	} else if legacy {
		if _, err := db.Exec("DROP TABLE IF EXISTS login_tokens; DROP TABLE utenti;"); err != nil {
			db.Close()
			return nil, err
		}
	}

	// Migrazione 2: drop lezioni/iscrizioni_lezione legacy (schema piatto con colonna titolo).
	if legacy, err := hasColumn(db, "lezioni", "titolo"); err != nil {
		db.Close()
		return nil, err
	} else if legacy {
		if _, err := db.Exec("DROP TABLE IF EXISTS iscrizioni_lezione; DROP TABLE IF EXISTS lezioni;"); err != nil {
			db.Close()
			return nil, err
		}
	}

	// Migrazione 3: aggiorna CHECK constraint utenti.ruolo per includere 'maestro'.
	if needs, err := needsRuoloMigration(db); err != nil {
		db.Close()
		return nil, err
	} else if needs {
		stmts := []string{
			`CREATE TABLE utenti_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				email TEXT UNIQUE NOT NULL,
				nome TEXT NOT NULL,
				ruolo TEXT NOT NULL DEFAULT 'staff' CHECK(ruolo IN ('admin', 'staff', 'maestro')),
				creato_il DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
			`INSERT OR IGNORE INTO utenti_new SELECT id, email, nome, ruolo, creato_il FROM utenti`,
			`DROP TABLE IF EXISTS login_tokens`,
			`DROP TABLE utenti`,
			`ALTER TABLE utenti_new RENAME TO utenti`,
		}
		if _, err := db.Exec("PRAGMA foreign_keys = OFF;"); err != nil {
			db.Close()
			return nil, err
		}
		for _, s := range stmts {
			if _, err := db.Exec(s); err != nil {
				db.Close()
				return nil, err
			}
		}
		if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
			db.Close()
			return nil, err
		}
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

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

// needsRuoloMigration controlla se la tabella utenti non include ancora 'maestro' nel CHECK.
func needsRuoloMigration(db *sql.DB) (bool, error) {
	var createSQL string
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='utenti'`).Scan(&createSQL)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !strings.Contains(createSQL, "'maestro'"), nil
}
