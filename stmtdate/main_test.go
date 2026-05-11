package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParsePDFCreationDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{
			name:  "with timezone apostrophes",
			input: "D:20260410091139+00'00'",
			want:  time.Date(2026, time.April, 10, 9, 11, 39, 0, time.UTC),
		},
		{
			name:  "without prefix",
			input: "20260410091139+0000",
			want:  time.Date(2026, time.April, 10, 9, 11, 39, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePDFCreationDate(tt.input)
			if err != nil {
				t.Fatalf("parsePDFCreationDate() error = %v", err)
			}

			if !got.Equal(tt.want) {
				t.Fatalf("parsePDFCreationDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatementMonthFromCreationDate(t *testing.T) {
	got, err := statementMonthFromCreationDate("D:20260410091139+00'00'")
	if err != nil {
		t.Fatalf("statementMonthFromCreationDate() error = %v", err)
	}

	if got != "2026-03" {
		t.Fatalf("statementMonthFromCreationDate() = %q, want %q", got, "2026-03")
	}
}

func TestRenamedPath(t *testing.T) {
	got := renamedPath("/tmp/File.pdf", "2026-03")
	want := "/tmp/File-2026-03.pdf"
	if got != want {
		t.Fatalf("renamedPath() = %q, want %q", got, want)
	}
}

func TestRenameFileWithMonth(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "File.pdf")
	if err := os.WriteFile(oldPath, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	newPath, err := renameFileWithMonth(oldPath, "2026-03")
	if err != nil {
		t.Fatalf("renameFileWithMonth() error = %v", err)
	}

	if newPath != filepath.Join(dir, "File-2026-03.pdf") {
		t.Fatalf("renameFileWithMonth() path = %q", newPath)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old path to be gone, stat err = %v", err)
	}

	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected new path to exist, stat err = %v", err)
	}
}
