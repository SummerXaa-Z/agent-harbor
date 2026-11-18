package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SummerXaa-Z/agent-harbor/internal/policy"
)

func TestWrapPolicyRefusesCrossWorkspaceChanges(t *testing.T) {
	for _, target := range []string{"node_repl", "other"} {
		t.Run(target, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "policy.yaml")
			raw := "version: 1\nworkspaces:\n  - id: B\n    strict: false\n    targets:\n      - id: " + target + "\n  - id: A\n    strict: false\n    targets: []\nagents:\n  - id: claude-code\n    workspace: A\n  - id: other-agent\n    workspace: B\n"
			if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			if err := ensureWrapPolicy(path, "node_repl", "claude-code"); err == nil {
				t.Error("ambiguous multi-workspace addition must require manual configuration")
			}
			if got := mustReadFile(t, path); got != raw {
				t.Error("policy was changed")
			}
			parsed, err := policy.Parse(strings.NewReader(mustReadFile(t, path)))
			if err != nil {
				t.Fatal(err)
			}
			got := parsed.Decide(policy.Request{Agent: "other-agent", Target: "node_repl", Tool: "read", Action: policy.ActionInvoke}).Allowed
			if got != (target == "node_repl") {
				t.Error("unrelated workspace permission changed")
			}
		})
	}
}

func TestWrapPolicyAcceptsTargetInAgentsWorkspace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.yaml")
	raw := "version: 1\nworkspaces:\n  - id: B\n    targets: []\n  - id: A\n    strict: true\n    targets:\n      - id: node_repl\nagents:\n  - id: claude-code\n    workspace: A\n    deny:\n      - capability: node_repl.*\n        reason: blocked\n"
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ensureWrapPolicy(path, "node_repl", "claude-code"); err != nil {
		t.Fatal(err)
	}
	if mustReadFile(t, path) != raw {
		t.Fatal("existing policy must not change")
	}
}

func TestWrapPolicyQuotesIdentifiers(t *testing.T) {
	for _, name := range []string{"name # comment", "name\n      - id: injected", "true", "a: b"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "policy.yaml")
			if err := ensureWrapPolicy(path, name, "claude-code"); err != nil {
				t.Fatal(err)
			}
			parsed, err := policy.Parse(strings.NewReader(mustReadFile(t, path)))
			if err != nil {
				t.Fatal(err)
			}
			if len(parsed.Workspaces) != 1 || len(parsed.Workspaces[0].Targets) != 1 || parsed.Workspaces[0].Targets[0].ID != name {
				t.Fatalf("identifier changed policy structure: %+v", parsed.Workspaces)
			}
			if err := ensureWrapPolicy(path, "second # name", "claude-code"); err != nil {
				t.Fatal(err)
			}
			parsed, err = policy.Parse(strings.NewReader(mustReadFile(t, path)))
			if err != nil {
				t.Fatal(err)
			}
			if len(parsed.Workspaces[0].Targets) != 2 || parsed.Workspaces[0].Targets[0].ID != "second # name" {
				t.Fatalf("extension lost literal identifier: %+v", parsed.Workspaces)
			}
		})
	}
}
