package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SummerXaa-Z/agent-harbor/internal/clients"
	"github.com/SummerXaa-Z/agent-harbor/internal/policy"
)

const importFixture = `{
  "mcpServers": {
    "crm": {"type": "http", "url": "https://crm.internal/mcp"},
    "filesystem": {"command": "npx", "args": ["-y", "filesystem-mcp"], "env": {"SECRET": "s3cr3t"}}
  }
}`

func TestScanSourcesClassifiesAndDedupes(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "servers.json")
	if err := os.WriteFile(source, []byte(importFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, used, err := scanSources(source)
	if err != nil || len(used) != 1 {
		t.Fatalf("scan: used=%v err=%v", used, err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected two entries: %+v", entries)
	}
	if endpoint, ok := entryEndpoint(entries[0]); entries[0].Name != "crm" || endpoint != "https://crm.internal/mcp" || !ok {
		t.Fatalf("http entry wrong: %+v", entries[0])
	}
	if _, ok := entryEndpoint(entries[1]); ok {
		t.Fatalf("filesystem is stdio, not http: %+v", entries[1])
	}
	if entryEnv(entries[1])["SECRET"] != "s3cr3t" {
		t.Fatalf("env not read: %+v", entries[1])
	}
}

func TestGeneratedPolicyValidatesAsBaseline(t *testing.T) {
	state := importState{
		Targets: map[string]string{"crm": "agt_crm", "fs": "agt_fs"},
		Clients: map[string]string{"claude-code": "agt_caller_cc"},
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "harbor.yaml")
	if err := writeGeneratedPolicy(file, state); err != nil {
		t.Fatalf("generate: %v", err)
	}
	parsed, err := policy.Parse(strings.NewReader(mustReadFile(t, file)))
	if err != nil {
		t.Fatalf("generated policy must validate: %v", err)
	}
	if parsed.Workspaces[0].Strict {
		t.Fatal("import-as-baseline must generate open mode")
	}
	if len(parsed.Workspaces[0].Targets) != 2 {
		t.Fatalf("both targets must be inside the circle: %+v", parsed.Workspaces[0].Targets)
	}
	if len(parsed.Agents) != 1 || parsed.Agents[0].ID != "agt_caller_cc" {
		t.Fatalf("caller agent missing: %+v", parsed.Agents)
	}
}

func TestConnectEditsConfigWithBackup(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "claude.json")
	original := `{"otherSetting": true, "mcpServers": {"crm": {"command": "old"}, "unmanaged": {"command": "keep"}}}`
	if err := os.WriteFile(config, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := clients.ClientSpec{Key: "claude-code", Format: clients.FormatMCPJSON, Writable: true}
	state := importState{Targets: map[string]string{"crm": "agt_crm"}}
	client := &cliClient{baseURL: "http://127.0.0.1:19090"}
	if err := connectEditConfig(config, spec, state, client, "ahx_test"); err != nil {
		t.Fatalf("edit: %v", err)
	}
	backup, err := os.ReadFile(config + ".harbor.bak")
	if err != nil || string(backup) != original {
		t.Fatalf("backup must hold the original: %v", err)
	}
	var document map[string]any
	raw := mustReadFile(t, config)
	if err := json.Unmarshal([]byte(raw), &document); err != nil {
		t.Fatal(err)
	}
	if document["otherSetting"] != true {
		t.Fatalf("unrelated settings must survive: %s", raw)
	}
	servers := document["mcpServers"].(map[string]any)
	unmanaged := servers["unmanaged"].(map[string]any)
	if unmanaged["command"] != "keep" {
		t.Fatalf("unmanaged servers must not be touched: %s", raw)
	}
	wired := servers["crm"].(map[string]any)
	if wired["type"] != "http" || !strings.Contains(wired["url"].(string), "/api/v1/mcp/agents/agt_crm/rpc") {
		t.Fatalf("crm must point at the gateway: %s", raw)
	}
	headers := wired["headers"].(map[string]any)
	if headers["Authorization"] != "Bearer ahx_test" {
		t.Fatalf("agent key header missing: %s", raw)
	}
}

func TestSnippetForCodexIsPrintOnly(t *testing.T) {
	state := importState{Targets: map[string]string{"crm": "agt_crm"}}
	spec := clients.ClientSpec{Key: "codex", Format: clients.FormatCodexTOML, Writable: false}
	snippet := snippetFor(spec, state, &cliClient{baseURL: "http://h:1"}, "ahx_k")
	for _, want := range []string{"[mcp_servers.crm]", `url = "http://h:1/api/v1/mcp/agents/agt_crm/rpc"`, "Authorization"} {
		if !strings.Contains(snippet, want) {
			t.Fatalf("codex snippet missing %q: %s", want, snippet)
		}
	}
}

// The end-to-end adoption funnel against a live in-process server: import
// creates targets and callers, emits a valid baseline policy plus state,
// and connect --print renders the snippet from that state.
func TestImportAndConnectEndToEnd(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	dir := t.TempDir()
	source := filepath.Join(dir, "servers.json")
	if err := os.WriteFile(source, []byte(importFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	address, cleanup := startTestServer(t)
	defer cleanup()
	t.Setenv("HARBOR_URL", "http://"+address)
	t.Setenv("HARBOR_ADMIN_KEY", "test-admin-key")

	var out, errOut strings.Builder
	outputYAML := filepath.Join(dir, "harbor.yaml")
	stateFile := filepath.Join(dir, "state.json")
	code := run([]string{"import", "--from", source, "--output", outputYAML, "--state", stateFile}, &out, &errOut)
	if code != 0 {
		t.Fatalf("import exit = %d stderr = %s stdout = %s", code, errOut.String(), out.String())
	}
	console := out.String()
	for _, want := range []string{"1 HTTP target", "stdio: filesystem", "harbor connect claude-code"} {
		if !strings.Contains(console, want) {
			t.Fatalf("import output missing %q: %s", want, console)
		}
	}
	state := loadStateOrFatal(t, stateFile)
	if len(state.Targets) != 1 || state.Targets["crm"] == "" {
		t.Fatalf("crm should be the one imported target: %+v", state.Targets)
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"connect", "--state", stateFile, "--print", "--agent-key", "ahx_e2e", "claude-code"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("connect exit = %d stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "/api/v1/mcp/agents/"+state.Targets["crm"]+"/rpc") {
		t.Fatalf("connect snippet should carry the harbor rpc url: %s", out.String())
	}

	// The wiring must hold end to end: the policy file validates and a
	// boundary check against the live target would pass.
	parsed, err := policy.Parse(strings.NewReader(mustReadFile(t, outputYAML)))
	if err != nil {
		t.Fatalf("generated policy invalid: %v", err)
	}
	if parsed.Workspaces[0].Targets[0].ID == "" {
		t.Fatal("target ids must be materialized in the policy")
	}
}

func loadStateOrFatal(t *testing.T, path string) importState {
	t.Helper()
	state, err := loadState(path)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	return state
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
