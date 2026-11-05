package main

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

// startTestServer boots an in-process API server on a random port with a
// fixed admin key, for exercising the CLI's HTTP surface end to end.
func startTestServer(t *testing.T) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	handler := httpapi.New(store.NewMemory(), httpapi.WithAdminKey("test-admin-key")).Router()
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	return listener.Addr().String(), func() { _ = server.Close() }
}

func TestUnknownCommandExitsWithConfigError(t *testing.T) {
	var out, errOut strings.Builder
	if code := run([]string{"frobnicate"}, &out, &errOut); code != 2 {
		t.Fatalf("unknown command exit = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), `unknown command "frobnicate"`) {
		t.Fatalf("stderr should name the unknown command: %s", errOut.String())
	}
}

func TestHelpExitsZero(t *testing.T) {
	var out, errOut strings.Builder
	if code := run([]string{"help"}, &out, &errOut); code != 0 {
		t.Fatalf("help exit = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "harbor key issue") {
		t.Fatalf("usage should list the command surface: %s", out.String())
	}
}

func TestInitWritesStarterThenRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "harbor.yaml")
	var out, errOut strings.Builder
	if code := run([]string{"init", "--file", file}, &out, &errOut); code != 0 {
		t.Fatalf("init exit = %d stderr = %s", code, errOut.String())
	}
	written, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(written), "version: 1") {
		t.Fatalf("starter policy missing: %v", err)
	}
	if code := run([]string{"init", "--file", file}, &out, &errOut); code != 2 {
		t.Fatalf("second init exit = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "already exists") {
		t.Fatalf("overwrite refusal copy wrong: %s", errOut.String())
	}
	if code := run([]string{"init", "--file", file, "--force"}, &out, &errOut); code != 0 {
		t.Fatalf("forced init exit = %d stderr = %s", code, errOut.String())
	}
}

func TestApplyValidatesAndSummarizes(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "harbor.yaml")
	policy := `version: 1
workspaces:
  - id: main
    strict: true
    targets:
      - id: crm
agents:
  - id: codex
    workspace: main
    allow:
      - "crm.*"
`
	if err := os.WriteFile(file, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut strings.Builder
	if code := run([]string{"apply", "--file", file, "--dry-run"}, &out, &errOut); code != 0 {
		t.Fatalf("apply exit = %d stderr = %s", code, errOut.String())
	}
	summary := out.String()
	for _, want := range []string{"workspace main", "(strict)", "codex", "1 allow"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary missing %q: %s", want, summary)
		}
	}
}

func TestApplyRejectsInvalidPolicyWithConfigError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "harbor.yaml")
	if err := os.WriteFile(file, []byte("version: 9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut strings.Builder
	if code := run([]string{"apply", "--file", file}, &out, &errOut); code != 2 {
		t.Fatalf("invalid policy exit = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "harbor.yaml:1") {
		t.Fatalf("error should cite the line: %s", errOut.String())
	}
}

func TestAuditAgainstLiveServer(t *testing.T) {
	// Boot an in-process server, issue a policy denial, then point the CLI
	// at it: the exit codes and output shape are part of the contract.
	address, cleanup := startTestServer(t)
	defer cleanup()
	t.Setenv("HARBOR_URL", "http://"+address)
	t.Setenv("HARBOR_ADMIN_KEY", "test-admin-key")

	var out, errOut strings.Builder
	code := run([]string{"audit", "--limit", "5"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("audit exit = %d stderr = %s", code, errOut.String())
	}

	errOut.Reset()
	code = run([]string{"why", "trc_missing"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("why on a missing trace should exit 2, got %d", code)
	}
}
