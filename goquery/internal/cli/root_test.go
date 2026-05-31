package cli

import (
	"bytes"
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

	exitCode := Run([]string{"query", "invalid-name"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Error: unknown query \"invalid-name\"") {
		t.Fatalf("expected unknown query error, got %q", stderr.String())
	}
}
