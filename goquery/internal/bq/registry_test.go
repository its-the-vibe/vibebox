package bq

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLookupQueryFromFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "queries.json")
	sqlPath := filepath.Join(tempDir, "monthly-balance-extremes.sql")
	if err := os.WriteFile(sqlPath, []byte("SELECT 1"), 0o644); err != nil {
		t.Fatalf("write sql file: %v", err)
	}
	content := `{"monthly-balance-extremes":{"file":"monthly-balance-extremes.sql"}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	registry, err := LoadQueryRegistry(path)
	if err != nil {
		t.Fatalf("LoadQueryRegistry returned error: %v", err)
	}

	query, err := LookupQuery(registry, "monthly-balance-extremes")
	if err != nil {
		t.Fatalf("LookupQuery returned error: %v", err)
	}
	if query != "SELECT 1" {
		t.Fatalf("unexpected query: %q", query)
	}
}

func TestLookupQueryUnknownName(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "queries.json")
	content := `{"monthly-balance-extremes":{"sql":"SELECT 1"}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	registry, err := LoadQueryRegistry(path)
	if err != nil {
		t.Fatalf("LoadQueryRegistry returned error: %v", err)
	}

	_, err = LookupQuery(registry, "invalid-name")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `unknown query "invalid-name"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadQueryRegistryMissingQueryDefinition(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "queries.json")
	content := `{"monthly-balance-extremes":{}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadQueryRegistry(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `has no sql or file value`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
