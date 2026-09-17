package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestRunNoArgsShowsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("expected root help output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "create      Create a Docker Compose override file") {
		t.Fatalf("expected create command in help output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "view        View the image tag of the first service") {
		t.Fatalf("expected view command in help output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "delete      Delete a Docker Compose override file") {
		t.Fatalf("expected delete command in help output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "ghcr.io/its-the-vibe/app:latest") {
		t.Fatalf("expected usage example in help output, got %q", stdout.String())
	}
}

func TestRunCreateNoArgsShowsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"create"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "docker-override create <img>") {
		t.Fatalf("expected create help output, got %q", stdout.String())
	}
}

func TestRunCreateMissingInputFile(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"create", "ghcr.io/example/app:latest", "missing.yml"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Error: input file \"missing.yml\" not found") {
		t.Fatalf("expected missing input error, got %q", stderr.String())
	}
}

func TestRunCreateUsesDefaultPaths(t *testing.T) {
	tempDir := t.TempDir()
	writeComposeFile(t, filepath.Join(tempDir, defaultComposePath), "version: '3.9'\nservices:\n  web:\n    build: .\n  worker:\n    command: run\nnetworks:\n  default: {}\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"create", "ghcr.io/example/app:latest"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Successfully generated \"docker-compose.override.yml\" with image: ghcr.io/example/app:latest") {
		t.Fatalf("expected success output, got %q", stdout.String())
	}

	output := readComposeFile(t, filepath.Join(tempDir, defaultOverridePath))
	if output["version"] != "3.9" {
		t.Fatalf("expected version to be preserved, got %#v", output["version"])
	}
	assertAllServicesUseImage(t, output, "ghcr.io/example/app:latest", []string{"web", "worker"})
	if _, ok := output["networks"]; !ok {
		t.Fatalf("expected non-service top-level keys to be preserved, got %#v", output)
	}
}

func TestRunCreateCustomPaths(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "compose.yml")
	outputPath := filepath.Join(tempDir, "compose.override.yml")
	writeComposeFile(t, inputPath, "services:\n  api:\n    image: old\n  jobs:\n    environment:\n      FOO: bar\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"create", "ghcr.io/example/custom:1.2.3", inputPath, outputPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	output := readComposeFile(t, outputPath)
	assertAllServicesUseImage(t, output, "ghcr.io/example/custom:1.2.3", []string{"api", "jobs"})
}

func TestRunViewDefaultsToCurrentAndFallsBackToBase(t *testing.T) {
	tempDir := t.TempDir()
	writeComposeFile(t, filepath.Join(tempDir, defaultComposePath), "services:\n  web:\n    image: ghcr.io/example/base:1.0.0\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"view"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "ghcr.io/example/base:1.0.0" {
		t.Fatalf("expected base image output, got %q", stdout.String())
	}
}

func TestRunViewDefaultsToCurrentAndUsesOverrideWhenPresent(t *testing.T) {
	tempDir := t.TempDir()
	writeComposeFile(t, filepath.Join(tempDir, defaultComposePath), "services:\n  web:\n    image: ghcr.io/example/base:1.0.0\n")
	writeComposeFile(t, filepath.Join(tempDir, defaultOverridePath), "services:\n  web:\n    image: ghcr.io/example/override:2.0.0\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"view"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "ghcr.io/example/override:2.0.0" {
		t.Fatalf("expected override image output, got %q", stdout.String())
	}
}

func TestRunViewBaseAndOverrideFlags(t *testing.T) {
	tempDir := t.TempDir()
	writeComposeFile(t, filepath.Join(tempDir, defaultComposePath), "services:\n  web:\n    image: ghcr.io/example/base:1.0.0\n")
	writeComposeFile(t, filepath.Join(tempDir, defaultOverridePath), "services:\n  web:\n    image: ghcr.io/example/override:2.0.0\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"view", "--base"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "ghcr.io/example/base:1.0.0" {
		t.Fatalf("expected base image output, got %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	exitCode = Run([]string{"view", "--override"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "ghcr.io/example/override:2.0.0" {
		t.Fatalf("expected override image output, got %q", stdout.String())
	}
}

func TestRunViewMutuallyExclusiveFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"view", "--base", "--override"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "flags --base, --override, and --current are mutually exclusive") {
		t.Fatalf("expected mutual-exclusion error, got %q", stderr.String())
	}
}

func TestRunViewUsesDeterministicServiceSelection(t *testing.T) {
	tempDir := t.TempDir()
	writeComposeFile(t, filepath.Join(tempDir, defaultComposePath), "services:\n  web:\n    image: ghcr.io/example/web:2.0.0\n  api:\n    image: ghcr.io/example/api:1.0.0\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"view", "--base"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "ghcr.io/example/api:1.0.0" {
		t.Fatalf("expected deterministic first image output, got %q", stdout.String())
	}
}

func TestRunViewRejectsNonScalarImageValue(t *testing.T) {
	tempDir := t.TempDir()
	writeComposeFile(t, filepath.Join(tempDir, defaultComposePath), "services:\n  web:\n    image:\n      repository: ghcr.io/example/web\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"view", "--base"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "image field in first service of \"docker-compose.yml\" must be a scalar value") {
		t.Fatalf("expected non-scalar image error, got %q", stderr.String())
	}
}

func TestRunDeleteRemovesOverrideFile(t *testing.T) {
	tempDir := t.TempDir()
	overridePath := filepath.Join(tempDir, defaultOverridePath)
	writeComposeFile(t, overridePath, "services:\n  web:\n    image: ghcr.io/example/override:2.0.0\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"delete"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "Deleted \"docker-compose.override.yml\"" {
		t.Fatalf("expected delete success output, got %q", stdout.String())
	}
	if _, err := os.Stat(overridePath); !os.IsNotExist(err) {
		t.Fatalf("expected override file to be deleted, stat err: %v", err)
	}
}

func TestRunDeleteHandlesMissingOverrideGracefully(t *testing.T) {
	tempDir := t.TempDir()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"delete"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "No override file found at \"docker-compose.override.yml\"" {
		t.Fatalf("expected missing-file output, got %q", stdout.String())
	}
}

func TestRunDeleteSupportsCustomPath(t *testing.T) {
	tempDir := t.TempDir()
	customPath := filepath.Join(tempDir, "custom.override.yml")
	writeComposeFile(t, customPath, "services:\n  web:\n    image: ghcr.io/example/override:2.0.0\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"delete", customPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "Deleted \""+customPath+"\"" {
		t.Fatalf("expected delete success output, got %q", stdout.String())
	}
	if _, err := os.Stat(customPath); !os.IsNotExist(err) {
		t.Fatalf("expected custom override file to be deleted, stat err: %v", err)
	}
}

func writeComposeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
}

func readComposeFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read compose file: %v", err)
	}

	var compose map[string]any
	if err := yaml.Unmarshal(data, &compose); err != nil {
		t.Fatalf("unmarshal compose file: %v", err)
	}
	return compose
}

func assertAllServicesUseImage(t *testing.T, compose map[string]any, image string, names []string) {
	t.Helper()
	services, ok := compose["services"].(map[string]any)
	if !ok {
		t.Fatalf("expected services map, got %#v", compose["services"])
	}
	for _, name := range names {
		service, ok := services[name].(map[string]any)
		if !ok {
			t.Fatalf("expected service %q to be a map, got %#v", name, services[name])
		}
		if len(service) != 1 {
			t.Fatalf("expected service %q to contain only image override, got %#v", name, service)
		}
		if service["image"] != image {
			t.Fatalf("expected service %q image %q, got %#v", name, image, service["image"])
		}
	}
}

func TestReplaceImageTag(t *testing.T) {
	tests := []struct {
		name      string
		baseImage string
		tag       string
		want      string
		wantErr   bool
	}{
		{
			name:      "Docker Hub official image",
			baseImage: "postgres:15",
			tag:       "16-alpine",
			want:      "postgres:16-alpine",
		},
		{
			name:      "Docker Hub user repo",
			baseImage: "myorg/webapp:v1.0",
			tag:       "v2.0",
			want:      "myorg/webapp:v2.0",
		},
		{
			name:      "GHCR registry",
			baseImage: "ghcr.io/its-the-vibe/querylab:latest",
			tag:       "feature",
			want:      "ghcr.io/its-the-vibe/querylab:feature",
		},
		{
			name:      "Private registry with port",
			baseImage: "localhost:5000/app:dev",
			tag:       "prod",
			want:      "localhost:5000/app:prod",
		},
		{
			name:      "Private registry with port and multi-level namespace",
			baseImage: "registry.corp.net:8443/team/service/worker:1.2.3",
			tag:       "2.0.0",
			want:      "registry.corp.net:8443/team/service/worker:2.0.0",
		},
		{
			name:      "Tag with leading colon",
			baseImage: "ghcr.io/its-the-vibe/app:old",
			tag:       ":new",
			want:      "ghcr.io/its-the-vibe/app:new",
		},
		{
			name:      "Image with tag and digest",
			baseImage: "ghcr.io/its-the-vibe/app:latest@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			tag:       "feature",
			want:      "ghcr.io/its-the-vibe/app:feature",
		},
		{
			name:      "Image without tag fails",
			baseImage: "postgres",
			tag:       "16",
			wantErr:   true,
		},
		{
			name:      "Image with registry port but without tag fails",
			baseImage: "localhost:5000/app",
			tag:       "dev",
			wantErr:   true,
		},
		{
			name:      "Image with digest only fails",
			baseImage: "ghcr.io/its-the-vibe/app@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			tag:       "feature",
			wantErr:   true,
		},
		{
			name:      "Empty tag fails",
			baseImage: "postgres:15",
			tag:       "",
			wantErr:   true,
		},
		{
			name:      "Whitespace tag fails",
			baseImage: "postgres:15",
			tag:       "   ",
			wantErr:   true,
		},
		{
			name:      "Invalid image without repo name",
			baseImage: ":latest",
			tag:       "feature",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceImageTag(tt.baseImage, tt.tag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("replaceImageTag(%q, %q) error = %v, wantErr %v", tt.baseImage, tt.tag, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("replaceImageTag(%q, %q) = %q, want %q", tt.baseImage, tt.tag, got, tt.want)
			}
		})
	}
}

func TestRunCreateWithOverrideTagDefaultPaths(t *testing.T) {
	tempDir := t.TempDir()
	writeComposeFile(t, filepath.Join(tempDir, defaultComposePath), "version: '3.9'\nservices:\n  web:\n    image: ghcr.io/its-the-vibe/querylab:latest\n  worker:\n    image: old:1.0.0\nnetworks:\n  default: {}\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"create", "--override-tag", "feature"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Successfully generated \"docker-compose.override.yml\" with image: ghcr.io/its-the-vibe/querylab:feature") {
		t.Fatalf("expected success output, got %q", stdout.String())
	}

	output := readComposeFile(t, filepath.Join(tempDir, defaultOverridePath))
	if output["version"] != "3.9" {
		t.Fatalf("expected version to be preserved, got %#v", output["version"])
	}
	assertAllServicesUseImage(t, output, "ghcr.io/its-the-vibe/querylab:feature", []string{"web", "worker"})
	if _, ok := output["networks"]; !ok {
		t.Fatalf("expected non-service top-level keys to be preserved, got %#v", output)
	}
}

func TestRunCreateWithOverrideTagCustomPaths(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "custom.yml")
	outputPath := filepath.Join(tempDir, "custom.override.yml")
	writeComposeFile(t, inputPath, "services:\n  api:\n    image: localhost:5000/myteam/myapp:v1\n  worker:\n    image: placeholder:old\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"create", "--override-tag", "v2", inputPath, outputPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Successfully generated \""+outputPath+"\" with image: localhost:5000/myteam/myapp:v2") {
		t.Fatalf("expected success output, got %q", stdout.String())
	}

	output := readComposeFile(t, outputPath)
	assertAllServicesUseImage(t, output, "localhost:5000/myteam/myapp:v2", []string{"api", "worker"})
}

func TestRunCreateWithOverrideTagCustomInputDefaultOutput(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "custom.yml")
	writeComposeFile(t, inputPath, "services:\n  api:\n    image: redis:6.2\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"create", "--override-tag", "7.0-alpine", inputPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	output := readComposeFile(t, filepath.Join(tempDir, defaultOverridePath))
	assertAllServicesUseImage(t, output, "redis:7.0-alpine", []string{"api"})
}

func TestRunCreateWithOverrideTagMissingInputFile(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"create", "--override-tag", "feature", "nonexistent.yml"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Error: input file \"nonexistent.yml\" not found") {
		t.Fatalf("expected missing input error, got %q", stderr.String())
	}
}

func TestRunCreateWithOverrideTagBaseImageMissingTag(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "compose.yml")
	writeComposeFile(t, inputPath, "services:\n  api:\n    image: postgres\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"create", "--override-tag", "16", inputPath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "does not contain a tag to replace") {
		t.Fatalf("expected missing tag error, got %q", stderr.String())
	}
}

func TestRunCreateWithOverrideTagEmptyTag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"create", "--override-tag", ""}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "override tag cannot be empty") {
		t.Fatalf("expected empty tag error, got %q", stderr.String())
	}
}

func TestRunCreateWithOverrideTagTooManyArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"create", "--override-tag", "feature", "arg1", "arg2", "arg3"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "accepts between 0 and 2 arg(s), received 3") {
		t.Fatalf("expected arg count error, got %q", stderr.String())
	}
}
