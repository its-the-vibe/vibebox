package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestBuildExtractionPrompt(t *testing.T) {
	prompt := buildExtractionPrompt("santander")
	if !strings.Contains(prompt, "Use the attached PNG directly for text extraction") {
		t.Fatalf("buildExtractionPrompt() missing image extraction instruction")
	}
}

func TestBuildExtractionPromptSumUp(t *testing.T) {
	prompt := buildExtractionPrompt("sumup")
	if !strings.Contains(prompt, "Date, Reference, Type, Amount, Description") {
		t.Fatalf("buildExtractionPrompt(sumup) missing SumUp header detection instruction")
	}
	if !strings.Contains(prompt, "positive values to money_in; negative values to money_out") {
		t.Fatalf("buildExtractionPrompt(sumup) missing amount mapping instruction")
	}
	if !strings.Contains(prompt, "joined with \" - \" and omit empty fields") {
		t.Fatalf("buildExtractionPrompt(sumup) missing description mapping instruction")
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

func TestDefaultBackend(t *testing.T) {
	os.Unsetenv("STMTPNG2TSV_BACKEND")
	if got := defaultBackend(); got != "gemini" {
		t.Errorf("defaultBackend() = %q, want %q", got, "gemini")
	}

	os.Setenv("STMTPNG2TSV_BACKEND", "copilot")
	defer os.Unsetenv("STMTPNG2TSV_BACKEND")
	if got := defaultBackend(); got != "copilot" {
		t.Errorf("defaultBackend() = %q, want %q", got, "copilot")
	}
}

func TestDefaultModel(t *testing.T) {
	os.Unsetenv("STMTPNG2TSV_MODEL")
	if got := defaultModel("copilot"); got != "gpt-4.1" {
		t.Errorf("defaultModel(copilot) = %q, want %q", got, "gpt-4.1")
	}
	if got := defaultModel("gemini"); got != "gemini-flash-latest" {
		t.Errorf("defaultModel(gemini) = %q, want %q", got, "gemini-flash-latest")
	}

	os.Setenv("STMTPNG2TSV_MODEL", "custom-model")
	defer os.Unsetenv("STMTPNG2TSV_MODEL")
	if got := defaultModel("copilot"); got != "custom-model" {
		t.Errorf("defaultModel(copilot) = %q, want %q", got, "custom-model")
	}
}

func TestNormalizeBankFormat(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", want: "santander"},
		{in: "santander", want: "santander"},
		{in: "SUMUP", want: "sumup"},
		{in: "unknown", wantErr: true},
	}

	for _, tt := range tests {
		got, err := normalizeBankFormat(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("normalizeBankFormat(%q) expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("normalizeBankFormat(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("normalizeBankFormat(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestResolveInputPaths(t *testing.T) {
	t.Run("single flag input", func(t *testing.T) {
		got, err := resolveInputPaths("statement.png", nil)
		if err != nil {
			t.Fatalf("resolveInputPaths() error = %v", err)
		}
		if len(got) != 1 || got[0] != "statement.png" {
			t.Fatalf("resolveInputPaths() = %#v, want [statement.png]", got)
		}
	})

	t.Run("multiple positional inputs", func(t *testing.T) {
		got, err := resolveInputPaths("", []string{"page2.png", "page1.png"})
		if err != nil {
			t.Fatalf("resolveInputPaths() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("resolveInputPaths() len = %d, want 2", len(got))
		}
		if got[0] != "page2.png" || got[1] != "page1.png" {
			t.Fatalf("resolveInputPaths() = %#v, want [page2.png page1.png]", got)
		}
	})
}

func TestSortPathsLogical(t *testing.T) {
	paths := []string{"statement-page10.png", "statement-page2.png", "statement-page1.png"}
	sortPathsLogical(paths)
	want := []string{"statement-page1.png", "statement-page2.png", "statement-page10.png"}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("sortPathsLogical() = %#v, want %#v", paths, want)
		}
	}
}

func TestSortTransactionsByDate(t *testing.T) {
	txns := []transaction{
		{Date: "2026-03-10", Description: "B"},
		{Date: "2026-03-01", Description: "A"},
		{Date: "2026-03-10", Description: "C"},
	}
	sortTransactionsByDate(txns)
	if txns[0].Date != "2026-03-01" || txns[1].Description != "B" || txns[2].Description != "C" {
		t.Fatalf("sortTransactionsByDate() unexpected order: %#v", txns)
	}
}
