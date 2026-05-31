package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunNoArgsShowsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("expected help usage output, got %q", stdout.String())
	}
}

func TestRunQueryNoArgsShowsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"query"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "goquery query <query-name>") {
		t.Fatalf("expected query help output, got %q", stdout.String())
	}
}

func TestRunQueryInvalidName(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	tempDir := t.TempDir()
	queryConfigPath := filepath.Join(tempDir, "queries.example.json")
	querySQLPath := filepath.Join(tempDir, "monthly-balance-extremes.example.sql")
	if err := os.WriteFile(querySQLPath, []byte("SELECT 1"), 0o644); err != nil {
		t.Fatalf("write query sql: %v", err)
	}
	if err := os.WriteFile(queryConfigPath, []byte(`{"monthly-balance-extremes":{"file":"monthly-balance-extremes.example.sql"}}`), 0o644); err != nil {
		t.Fatalf("write query config: %v", err)
	}
	t.Setenv("GOQUERY_QUERIES_FILE", queryConfigPath)

	exitCode := Run([]string{"query", "invalid-name"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Error: unknown query \"invalid-name\"") {
		t.Fatalf("expected unknown query error, got %q", stderr.String())
	}
}

func TestRunSchemaNoArgsShowsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"schema"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "goquery schema <dataset> <table>") {
		t.Fatalf("expected schema help output, got %q", stdout.String())
	}
}

func TestRunSchemaMissingProjectID(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	t.Setenv("GOOGLE_PROJECT_ID", "")

	exitCode := Run([]string{"schema", "my_dataset", "my_table"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Error: GOOGLE_PROJECT_ID is required") {
		t.Fatalf("expected missing project error, got %q", stderr.String())
	}
}

func TestRunSchemaInvalidDatasetName(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	t.Setenv("GOOGLE_PROJECT_ID", "my-project")

	exitCode := Run([]string{"schema", "invalid.dataset", "my_table"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Error: invalid dataset name \"invalid.dataset\"") {
		t.Fatalf("expected invalid dataset error, got %q", stderr.String())
	}
}

func TestRunSchemaInvalidTableName(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	t.Setenv("GOOGLE_PROJECT_ID", "my-project")

	exitCode := Run([]string{"schema", "my_dataset", "invalid.table"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Error: invalid table name \"invalid.table\"") {
		t.Fatalf("expected invalid table error, got %q", stderr.String())
	}
}
