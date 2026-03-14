package config

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Timezone != "America/Sao_Paulo" {
		t.Errorf("Timezone = %q, want America/Sao_Paulo", cfg.Timezone)
	}
	if cfg.DebounceMs != 3000 {
		t.Errorf("DebounceMs = %d, want 3000", cfg.DebounceMs)
	}
	if cfg.DebounceMaxMs != 15000 {
		t.Errorf("DebounceMaxMs = %d, want 15000", cfg.DebounceMaxMs)
	}
	if cfg.SessionTimeoutMin != 240 {
		t.Errorf("SessionTimeoutMin = %d, want 240", cfg.SessionTimeoutMin)
	}
	if cfg.ContextBudget != 4000 {
		t.Errorf("ContextBudget = %d, want 4000", cfg.ContextBudget)
	}
	if cfg.MCPServerEnabled != false {
		t.Error("MCPServerEnabled should default to false")
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg, err := LoadConfig(db)
	if err != nil {
		t.Fatal(err)
	}

	cfg.BaseURL = "https://custom.api/v1"
	cfg.APIKey = "sk-test-key-12345"
	cfg.MaxHistory = 50
	cfg.DebounceMs = 5000
	cfg.DebounceMaxMs = 20000
	cfg.SessionTimeoutMin = 120
	cfg.ContextBudget = 8000
	cfg.Debug = true
	cfg.Timezone = "UTC"

	if err := SaveConfig(db, cfg); err != nil {
		t.Fatal("SaveConfig:", err)
	}

	loaded, err := LoadConfig(db)
	if err != nil {
		t.Fatal("LoadConfig:", err)
	}

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"BaseURL", loaded.BaseURL, "https://custom.api/v1"},
		{"APIKey", loaded.APIKey, "sk-test-key-12345"},
		{"MaxHistory", loaded.MaxHistory, 50},
		{"DebounceMs", loaded.DebounceMs, 5000},
		{"DebounceMaxMs", loaded.DebounceMaxMs, 20000},
		{"SessionTimeoutMin", loaded.SessionTimeoutMin, 120},
		{"ContextBudget", loaded.ContextBudget, 8000},
		{"Debug", loaded.Debug, true},
		{"Timezone", loaded.Timezone, "UTC"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestPartialConfig(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if err := createConfigTable(db); err != nil {
		t.Fatal(err)
	}
	db.Exec("INSERT INTO config (key, value) VALUES ('debug', 'true')")

	cfg, err := LoadConfig(db)
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.Debug {
		t.Error("Debug should be true")
	}

	if cfg.Timezone != "America/Sao_Paulo" {
		t.Errorf("Timezone = %q, want default America/Sao_Paulo", cfg.Timezone)
	}
}

func TestApiKeyNotCleared(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg, err := LoadConfig(db)
	if err != nil {
		t.Fatal(err)
	}

	cfg.APIKey = "sk-original-key"
	if err := SaveConfig(db, cfg); err != nil {
		t.Fatal(err)
	}

	cfg.APIKey = ""
	if err := SaveConfig(db, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig(db)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.APIKey != "sk-original-key" {
		t.Errorf("APIKey = %q, want sk-original-key (should not be cleared)", loaded.APIKey)
	}
}
