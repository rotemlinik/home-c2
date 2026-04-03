package db

import (
	"testing"
)

func TestOpenInMemory(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error: %v", err)
	}
	defer database.Close()

	// Verify tables were created
	tables := []string{"people", "categories", "tasks", "integrations", "sync_log", "whatsapp_session", "whatsapp_messages", "whatsapp_contacts"}
	for _, table := range tables {
		var name string
		err := database.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestMigrateIdempotent(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	defer database.Close()

	// Run migrate again — should not error (CREATE TABLE IF NOT EXISTS)
	if err := migrate(database); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestSeedCategories(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	expected := []string{"Housekeeping", "Car", "Health", "Packages", "Finance", "Social"}

	rows, err := database.Query(`SELECT name FROM categories ORDER BY name`)
	if err != nil {
		t.Fatalf("query categories: %v", err)
	}
	defer rows.Close()

	found := map[string]bool{}
	for rows.Next() {
		var name string
		rows.Scan(&name)
		found[name] = true
	}

	for _, e := range expected {
		if !found[e] {
			t.Errorf("seed category %q not found", e)
		}
	}
}

func TestSourceRefColumn(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	// Insert a task with source_ref
	_, err = database.Exec(`
		INSERT INTO tasks (title, type, status, source, source_ref)
		VALUES ('test', 'one-off', 'pending', 'gmail', 'ref-123')
	`)
	if err != nil {
		t.Fatalf("insert with source_ref: %v", err)
	}

	var ref string
	err = database.QueryRow(`SELECT source_ref FROM tasks WHERE source_ref = 'ref-123'`).Scan(&ref)
	if err != nil {
		t.Fatalf("query source_ref: %v", err)
	}
	if ref != "ref-123" {
		t.Errorf("source_ref = %q, want ref-123", ref)
	}
}
