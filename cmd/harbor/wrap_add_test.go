package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SummerXaa-Z/agent-harbor/internal/policy"
)

const wrapClientConfig = `{
  "otherSetting": true,
  "mcpServers": {
    "node_repl": {
      "command": "/usr/local/bin/node_repl",
      "args": ["--verbose", "--port", "1"],
      "env": {"TOKEN": "secret-value"}
    },
    "unmanaged": {"command": "keep-me"}
  }
}`

type wrapFixture struct {
	home    string
	config  string
	policy  string
	state   string
	wrapBin string
}

func setupWrapFixture(t *testing.T) *wrapFixture {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(home, ".claude.json")
	if err := os.WriteFile(config, []byte(wrapClientConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	wrapBin := filepath.Join(dir, "harbor-wrap")
	if err := os.WriteFile(wrapBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &wrapFixture{
		home:    home,
		config:  config,
		policy:  filepath.Join(dir, "harbor.yaml"),
		state:   filepath.Join(dir, "state.json"),
		wrapBin: wrapBin,
	}
}

// wrap runs `harbor wrap <sub> <positionals...>` with the fixture's flags
// inserted after the subcommand (flags must precede positionals - stdlib
// flag semantics).
func (f *wrapFixture) wrap(t *testing.T, sub string, positionals ...string) {
	t.Helper()
	flags := []string{"--state", f.state}
	if sub == "add" {
		flags = append(flags, "--wrap-bin", f.wrapBin, "--policy", f.policy)
	}
	args := append(append([]string{"wrap", sub}, flags...), positionals...)
	var out, errOut strings.Builder
	if code := run(args, &out, &errOut); code != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
}

func (f *wrapFixture) entry(t *testing.T, name string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(f.config)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	entry, _ := document["mcpServers"].(map[string]any)[name].(map[string]any)
	return entry
}

func TestWrapAddRewritesClientEntryAndGeneratesPolicy(t *testing.T) {
	f := setupWrapFixture(t)
	f.wrap(t, "add", "claude-code", "node_repl")

	entry := f.entry(t, "node_repl")
	if entry["command"] != f.wrapBin {
		t.Fatalf("command must be the wrapper: %+v", entry)
	}
	args := stringArgs(entry["args"])
	want := []string{"--policy", f.policy, "--agent", "claude-code", "--target", "node_repl", "--", "/usr/local/bin/node_repl", "--verbose", "--port", "1"}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Fatalf("wrapped args wrong: %v", args)
	}
	env, _ := entry["env"].(map[string]any)
	if env["TOKEN"] != "secret-value" {
		t.Fatalf("env must survive the rewrite: %+v", entry)
	}
	if f.entry(t, "unmanaged")["command"] != "keep-me" {
		t.Fatal("unmanaged servers must not be touched")
	}
	if _, err := os.Stat(f.config + ".harbor.bak"); err != nil {
		t.Fatal("the original config must be backed up")
	}

	parsed, err := policy.Parse(strings.NewReader(mustReadFile(t, f.policy)))
	if err != nil {
		t.Fatalf("generated policy invalid: %v", err)
	}
	if parsed.Workspaces[0].Targets[0].ID != "node_repl" || parsed.Agents[0].ID != "claude-code" {
		t.Fatalf("policy must declare the wrapped target and the client agent: %+v", parsed)
	}

	// Idempotent: a second add is a no-op that keeps the config intact.
	f.wrap(t, "add", "claude-code", "node_repl")
	if f.entry(t, "node_repl")["command"] != f.wrapBin {
		t.Fatal("second add must not double-wrap")
	}
}

func TestWrapRemoveRestoresTheOriginalEntry(t *testing.T) {
	f := setupWrapFixture(t)
	f.wrap(t, "add", "claude-code", "node_repl")
	f.wrap(t, "remove", "claude-code", "node_repl")

	entry := f.entry(t, "node_repl")
	if entry["command"] != "/usr/local/bin/node_repl" {
		t.Fatalf("original command must be restored: %+v", entry)
	}
	if strings.Join(stringArgs(entry["args"]), " ") != "--verbose --port 1" {
		t.Fatalf("original args must be restored: %+v", entry)
	}

	var out, errOut strings.Builder
	code := run([]string{"wrap", "remove", "--state", f.state, "claude-code", "node_repl"}, &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "not wrapped") {
		t.Fatalf("removing an unwrapped server should fail clearly: %d %s", code, errOut.String())
	}
}

func TestWrapAddExtendsAnExistingPolicyAndRefusesOddLayouts(t *testing.T) {
	f := setupWrapFixture(t)
	f.wrap(t, "add", "claude-code", "node_repl") // generates the policy

	// A second server extends the same file.
	f.entry(t, "unmanaged") // touch nothing
	configRaw := mustReadFile(t, f.config)
	_ = configRaw
	f.wrap(t, "add", "claude-code", "unmanaged")
	parsed, err := policy.Parse(strings.NewReader(mustReadFile(t, f.policy)))
	if err != nil {
		t.Fatalf("extended policy invalid: %v", err)
	}
	if len(parsed.Workspaces[0].Targets) != 2 || len(parsed.Agents) != 1 {
		t.Fatalf("second target appended, agent not duplicated: %+v", parsed)
	}

	// A policy the textual anchors cannot handle fails loudly and leaves
	// the file untouched - exercised on a server that is present but not
	// yet wrapped (already-wrapped names short-circuit earlier).
	odd := "version: 1\nworkspaces:\n  - id: other\n    strict: true\n    targets: []\n"
	if err := os.WriteFile(f.policy, []byte(odd), 0o644); err != nil {
		t.Fatal(err)
	}
	withThird := strings.Replace(wrapClientConfig, `"unmanaged": {"command": "keep-me"}`,
		"\"unmanaged\": {\"command\": \"keep-me\"},\n    \"late_add\": {\"command\": \"/bin/late\"}", 1)
	if err := os.WriteFile(f.config, []byte(withThird), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut strings.Builder
	code := run([]string{"wrap", "add", "--wrap-bin", f.wrapBin, "--policy", f.policy, "--state", f.state, "claude-code", "late_add"}, &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "manually") {
		t.Fatalf("odd layouts must fall back to manual instructions: %d %s", code, errOut.String())
	}
	if mustReadFile(t, f.policy) != odd {
		t.Fatal("the policy file must be untouched when extension fails")
	}
}
