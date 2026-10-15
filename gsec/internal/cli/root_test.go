package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

type runnerCall struct {
	args  []string
	stdin string
}

type fakeRunner struct {
	responses []string
	calls     []runnerCall
}

func (r *fakeRunner) Run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	call := runnerCall{args: append([]string(nil), args...)}
	if stdin != nil {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		call.stdin = string(data)
	}
	r.calls = append(r.calls, call)
	if len(r.responses) > 0 {
		_, _ = io.WriteString(stdout, r.responses[0])
		r.responses = r.responses[1:]
	}
	return nil
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr, &fakeRunner{}); code != 0 {
		t.Fatalf("expected success, got %d: %s", code, stderr.String())
	}
	for _, command := range []string{"view", "put", "versions"} {
		if !strings.Contains(stdout.String(), command) {
			t.Errorf("help output does not contain %q: %s", command, stdout.String())
		}
	}
}

func TestViewLatestAndSpecificVersion(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "latest", args: []string{"view", "my-secret"}, want: "latest"},
		{name: "specific", args: []string{"view", "my-secret", "--version", "7"}, want: "7"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := &fakeRunner{responses: []string{"secret contents"}}
			var stdout, stderr bytes.Buffer
			if code := run(test.args, &stdout, &stderr, runner); code != 0 {
				t.Fatalf("run failed: %s", stderr.String())
			}
			if stdout.String() != "secret contents" {
				t.Fatalf("unexpected output: %q", stdout.String())
			}
			want := []string{"secrets", "versions", "access", test.want, "--secret", "my-secret"}
			if strings.Join(runner.calls[0].args, " ") != strings.Join(want, " ") {
				t.Fatalf("unexpected gcloud args: %v", runner.calls[0].args)
			}
		})
	}
}

func TestPutUpdatesJSONAndAddsVersion(t *testing.T) {
	runner := &fakeRunner{responses: []string{`{"existing":1,"name":"old"}`}}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"put", "my-secret", "name", "new value"}, &stdout, &stderr, runner); code != 0 {
		t.Fatalf("run failed: %s", stderr.String())
	}
	if len(runner.calls) != 2 {
		t.Fatalf("expected access and add calls, got %d", len(runner.calls))
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(runner.calls[1].stdin), &payload); err != nil {
		t.Fatalf("invalid JSON sent to gcloud: %v", err)
	}
	if payload["existing"] != float64(1) || payload["name"] != "new value" {
		t.Fatalf("unexpected updated payload: %#v", payload)
	}
	if !strings.Contains(strings.Join(runner.calls[1].args, " "), "versions add my-secret --data-file=-") {
		t.Fatalf("unexpected add args: %v", runner.calls[1].args)
	}
}

func TestPutRejectsNonObjectJSON(t *testing.T) {
	runner := &fakeRunner{responses: []string{`[]`}}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"put", "my-secret", "key", "value"}, &stdout, &stderr, runner); code == 0 {
		t.Fatal("expected invalid payload to fail")
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected not to add a version, got %d calls", len(runner.calls))
	}
}

func TestVersionsListsActiveVersionsInCreationOrder(t *testing.T) {
	runner := &fakeRunner{responses: []string{`[
		{"name":"projects/p/secrets/s/versions/2","createTime":"2026-01-02T00:00:00Z","state":"ENABLED"},
		{"name":"projects/p/secrets/s/versions/1","createTime":"2026-01-01T00:00:00Z","state":"ENABLED"},
		{"name":"projects/p/secrets/s/versions/3","createTime":"2026-01-03T00:00:00Z","state":"DISABLED"}
	]`}}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"versions", "my-secret"}, &stdout, &stderr, runner); code != 0 {
		t.Fatalf("run failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "1\t2026-01-01T00:00:00Z\n2\t2026-01-02T00:00:00Z\n") || strings.Contains(stdout.String(), "\n3\t") {
		t.Fatalf("unexpected versions output: %q", stdout.String())
	}
}

func TestDeleteOldestDestroysEarliestActiveVersion(t *testing.T) {
	runner := &fakeRunner{responses: []string{`[
		{"name":"projects/p/secrets/s/versions/2","createTime":"2026-01-02T00:00:00Z","state":"ENABLED"},
		{"name":"projects/p/secrets/s/versions/1","createTime":"2026-01-01T00:00:00Z","state":"ENABLED"}
	]`}}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"versions", "delete-oldest", "my-secret"}, &stdout, &stderr, runner); code != 0 {
		t.Fatalf("run failed: %s", stderr.String())
	}
	want := []string{"secrets", "versions", "destroy", "1", "--secret", "my-secret"}
	if strings.Join(runner.calls[1].args, " ") != strings.Join(want, " ") {
		t.Fatalf("unexpected destroy args: %v", runner.calls[1].args)
	}
}
