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
	content := `{"monthly-balance-extremes":"SELECT 1"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	query, err := LookupQuery(path, "monthly-balance-extremes")
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
	content := `{"monthly-balance-extremes":"SELECT 1"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LookupQuery(path, "invalid-name")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `unknown query "invalid-name"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
