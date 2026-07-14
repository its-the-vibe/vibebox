package cli

import (
	"bytes"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
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

func TestRunList(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	tempDir := t.TempDir()
	queryConfigPath := filepath.Join(tempDir, "queries.json")
	if err := os.WriteFile(queryConfigPath, []byte(`{
		"query-b":{"sql":"SELECT 2"},
		"query-a":{"sql":"SELECT 1"}
	}`), 0o644); err != nil {
		t.Fatalf("write query config: %v", err)
	}
	t.Setenv("GOQUERY_QUERIES_FILE", queryConfigPath)

	exitCode := Run([]string{"list"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	expected := "Available queries:\n  - query-a\n  - query-b\n"
	if stdout.String() != expected {
		t.Fatalf("expected output %q, got %q", expected, stdout.String())
	}
}

func TestRunListEmptyConfig(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	tempDir := t.TempDir()
	queryConfigPath := filepath.Join(tempDir, "queries.json")
	if err := os.WriteFile(queryConfigPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write query config: %v", err)
	}
	t.Setenv("GOQUERY_QUERIES_FILE", queryConfigPath)

	exitCode := Run([]string{"list"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	expected := "Available queries:\n"
	if stdout.String() != expected {
		t.Fatalf("expected output %q, got %q", expected, stdout.String())
	}
}

func TestRunListMissingConfig(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	t.Setenv("GOQUERY_QUERIES_FILE", "non-existent.json")

	exitCode := Run([]string{"list"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Error: failed to read query config") {
		t.Fatalf("expected config read error, got %q", stderr.String())
	}
}

func TestSchemaHeader(t *testing.T) {
	header := schemaHeader(bigquery.Schema{
		{Name: "year_month"},
		{Name: "max_balance"},
		{Name: "max_date"},
	})

	if header != "year_month | max_balance | max_date" {
		t.Fatalf("unexpected header: %q", header)
	}
}

func TestFormatByType(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		fieldType bigquery.FieldType
		want      string
	}{
		{
			name:      "numeric",
			value:     big.NewRat(12345, 100),
			fieldType: bigquery.NumericFieldType,
			want:      "123.45",
		},
		{
			name:      "timestamp",
			value:     time.Date(2026, 7, 14, 18, 30, 0, 0, time.UTC),
			fieldType: bigquery.TimestampFieldType,
			want:      "2026-07-14",
		},
		{
			name:      "date",
			value:     civil.Date{Year: 2026, Month: 7, Day: 14},
			fieldType: bigquery.DateFieldType,
			want:      "2026-07-14",
		},
		{
			name:      "string",
			value:     "account-a",
			fieldType: bigquery.StringFieldType,
			want:      "account-a",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatByType(tc.value, tc.fieldType)
			if got != tc.want {
				t.Fatalf("formatByType() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPrintRowUsesSchemaFormatting(t *testing.T) {
	var out bytes.Buffer
	row := []bigquery.Value{"2026-07", big.NewRat(12345, 100), civil.Date{Year: 2026, Month: 7, Day: 14}}
	schema := bigquery.Schema{
		{Name: "year_month", Type: bigquery.StringFieldType},
		{Name: "max_balance", Type: bigquery.NumericFieldType},
		{Name: "max_date", Type: bigquery.DateFieldType},
	}

	if err := printRow(&out, row, schema); err != nil {
		t.Fatalf("printRow() error = %v", err)
	}

	if out.String() != "2026-07 | 123.45 | 2026-07-14\n" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}
