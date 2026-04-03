package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1) // sqlite doesn't support concurrent writes

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	// PRAGMAs must run outside of transactions
	db.Exec(`PRAGMA journal_mode=WAL`)
	db.Exec(`PRAGMA foreign_keys=ON`)

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS people (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS categories (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);

		CREATE TABLE IF NOT EXISTS tasks (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			title           TEXT NOT NULL,
			type            TEXT NOT NULL CHECK(type IN ('one-off', 'recurring')),
			recurrence_unit TEXT CHECK(recurrence_unit IN ('day', 'week', 'month')),
			recurrence_every INTEGER,
			due_date        DATE,
			assignee_id     INTEGER REFERENCES people(id),
			category_id     INTEGER REFERENCES categories(id),
			status          TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'done', 'snoozed')),
			completed_by    INTEGER REFERENCES people(id),
			completed_at    DATETIME,
			notes           TEXT DEFAULT '',
			source          TEXT NOT NULL DEFAULT 'manual' CHECK(source IN ('manual', 'gmail', 'imessage', 'whatsapp')),
			parent_id       INTEGER REFERENCES tasks(id),
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		INSERT OR IGNORE INTO categories (name) VALUES
			('Housekeeping'),
			('Car'),
			('Health'),
			('Packages'),
			('Finance'),
			('Social');

		CREATE TABLE IF NOT EXISTS integrations (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			source            TEXT NOT NULL UNIQUE,
			access_token      TEXT NOT NULL,
			refresh_token     TEXT NOT NULL,
			token_expiry      DATETIME NOT NULL,
			last_synced_at    DATETIME,
			last_message_date DATETIME,
			email             TEXT,
			created_at        DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS sync_log (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			source        TEXT NOT NULL,
			synced_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
			emails_read   INTEGER DEFAULT 0,
			tasks_created INTEGER DEFAULT 0,
			error         TEXT
		);

		CREATE TABLE IF NOT EXISTS whatsapp_session (
			id            INTEGER PRIMARY KEY CHECK(id = 1),
			status        TEXT NOT NULL DEFAULT 'disconnected',
			phone         TEXT,
			qr_data       TEXT,
			qr_expires_at DATETIME,
			connected_at  DATETIME,
			last_synced_at DATETIME,
			error         TEXT
		);

		CREATE TABLE IF NOT EXISTS whatsapp_messages (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			wa_message_id TEXT NOT NULL UNIQUE,
			sender_jid    TEXT NOT NULL,
			sender_name   TEXT,
			body          TEXT NOT NULL,
			received_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			processed     INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS whatsapp_contacts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			jid        TEXT NOT NULL UNIQUE,
			label      TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	// Add source_ref column to tasks if it doesn't exist (SQLite returns an error on duplicate column)
	tx.Exec(`ALTER TABLE tasks ADD COLUMN source_ref TEXT`) //nolint:errcheck — ignored if column already exists

	// Ensure unique constraint on source_ref for deduplication
	tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_source_ref ON tasks(source_ref) WHERE source_ref IS NOT NULL`)

	return tx.Commit()
}
