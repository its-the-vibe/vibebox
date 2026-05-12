package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractJSONPayload(t *testing.T) {
	in := "```json\n{\"statement_year\":2026,\"transactions\":[]}\n```"
	got := extractJSONPayload(in)
	want := `{"statement_year":2026,"transactions":[]}`
	if got != want {
		t.Fatalf("extractJSONPayload() = %q, want %q", got, want)
	}
}

func TestNormalizeDate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		year int
		out  string
	}{
		{name: "already ISO", in: "2026-03-10", out: "2026-03-10"},
		{name: "slash format with year", in: "10/03/2026", out: "2026-03-10"},
		{name: "without year", in: "10 Mar", year: 2026, out: "2026-03-10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeDate(tt.in, tt.year)
			if err != nil {
				t.Fatalf("normalizeDate() error = %v", err)
			}
			if got != tt.out {
				t.Fatalf("normalizeDate() = %q, want %q", got, tt.out)
			}
		})
	}
}

func TestNormalizeAmount(t *testing.T) {
	got := normalizeAmount("£1,234.50")
	if got != "1234.50" {
		t.Fatalf("normalizeAmount() = %q, want %q", got, "1234.50")
	}
}

func TestWriteTSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.tsv")
	txns := []transaction{{
		Date:        "2026-03-10",
		Description: "MONTHLY FEE",
		MoneyIn:     "",
		MoneyOut:    "3.00",
		Balance:     "737.26",
	}}

	if err := writeTSV(path, txns); err != nil {
		t.Fatalf("writeTSV() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	want := "Date|Description|Money In|Money Out|Balance\n2026-03-10|MONTHLY FEE||3.00|737.26\n"
	if string(data) != want {
		t.Fatalf("TSV contents = %q, want %q", string(data), want)
	}
}

func TestInferYearFromText(t *testing.T) {
	got := inferYearFromText("Statement period Mar 2026, txns in 2026 and 2025")
	if got != 2026 {
		t.Fatalf("inferYearFromText() = %d, want 2026", got)
	}
}
