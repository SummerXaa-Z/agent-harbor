package policy

import (
	"strings"
	"testing"
)

// fixture line map (asserted below): workspace main at 3, allow "crm.*" at
// 13, deny "crm.*delete*" at 15.
const fixture = `version: 1
workspaces:
  - id: main
    strict: false
    targets:
      - id: crm
        endpoint: https://crm.internal/mcp
    domains: [sales]
agents:
  - id: codex
    workspace: main
    allow:
      - "crm.*"
    deny:
      - capability: "crm.*delete*"
        reason: prod database is read-only
    subjects: ["user:support-*"]
`

func parseFixture(t *testing.T) *Policy {
	t.Helper()
	policy, err := Parse(strings.NewReader(fixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return policy
}

func TestParseFixtureStructure(t *testing.T) {
	policy := parseFixture(t)
	if policy.Version != 1 || len(policy.Workspaces) != 1 || len(policy.Agents) != 1 {
		t.Fatalf("unexpected structure: %+v", policy)
	}
	workspace := policy.Workspaces[0]
	if workspace.ID != "main" || workspace.Strict || len(workspace.Targets) != 1 || workspace.Targets[0].ID != "crm" {
		t.Fatalf("unexpected workspace: %+v", workspace)
	}
	agent := policy.Agents[0]
	if agent.ID != "codex" || agent.Workspace != "main" || len(agent.Allow) != 1 || len(agent.Deny) != 1 {
		t.Fatalf("unexpected agent: %+v", agent)
	}
	if agent.Deny[0].Reason != "prod database is read-only" {
		t.Fatalf("deny reason not parsed: %+v", agent.Deny[0])
	}
}

func TestRuleRefCarriesLineNumbers(t *testing.T) {
	policy := parseFixture(t)
	allowed := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: "lookup", Action: ActionInvoke})
	if !allowed.Allowed || allowed.RuleRef != "harbor.yaml:13" {
		t.Fatalf("allow decision should cite the allow rule line: %+v", allowed)
	}
	denied := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: "bulkDelete", Action: ActionInvoke})
	if denied.Allowed || denied.RuleRef != "harbor.yaml:15" {
		t.Fatalf("deny decision should cite the deny rule line: %+v", denied)
	}
}

func TestSemanticsDenyAlwaysWins(t *testing.T) {
	policy := parseFixture(t)
	decision := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: "lookupAndDelete", Action: ActionInvoke})
	if decision.Allowed {
		t.Fatal("deny must win over the crm.* allow")
	}
	if decision.Evidence[len(decision.Evidence)-1].Check != "deny" {
		t.Fatalf("evidence should end at the deny step: %+v", decision.Evidence)
	}
}

func TestSemanticsOpenModeDefaultsToAllow(t *testing.T) {
	// With allow narrowed to "crm.lookup*", an unrelated tool falls through
	// to the open-mode default rather than an allow match.
	narrow := strings.Replace(fixture, `- "crm.*"`, `- "crm.lookup*"`, 1)
	policy, err := Parse(strings.NewReader(narrow))
	if err != nil {
		t.Fatalf("parse narrowed fixture: %v", err)
	}
	decision := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: "anythingNew", Action: ActionInvoke})
	if !decision.Allowed || decision.Reason != "default-open" {
		t.Fatalf("open mode should default to allow: %+v", decision)
	}
}

func TestSemanticsStrictModeDefaultsToDeny(t *testing.T) {
	strict := strings.Replace(fixture, "strict: false", "strict: true", 1)
	strict = strings.Replace(strict, `- "crm.*"`, `- "crm.lookup*"`, 1)
	policy, err := Parse(strings.NewReader(strict))
	if err != nil {
		t.Fatalf("parse strict fixture: %v", err)
	}
	decision := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: "anythingNew", Action: ActionInvoke})
	if decision.Allowed {
		t.Fatal("strict mode must deny unmatched capabilities")
	}
	if decision.RuleRef != "harbor.yaml:3" {
		t.Fatalf("strict default should cite the workspace line: %+v", decision)
	}
	// An explicit allow still passes under strict.
	allowed := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: "lookup", Action: ActionInvoke})
	if !allowed.Allowed {
		t.Fatal("strict mode must honor explicit allows")
	}
}

func TestSemanticsBoundaryIsModeIndependent(t *testing.T) {
	for _, strict := range []bool{false, true} {
		source := strings.Replace(fixture, "strict: false", "strict: "+boolString(strict), 1)
		current, err := Parse(strings.NewReader(source))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		decision := current.Decide(Request{Agent: "codex", Target: "billing", Tool: "read", Action: ActionInvoke})
		if decision.Allowed {
			t.Fatalf("cross-workspace target must be denied regardless of mode (strict=%v)", strict)
		}
		if decision.Evidence[len(decision.Evidence)-1].Check != "boundary" {
			t.Fatalf("evidence should end at the boundary step: %+v", decision.Evidence)
		}
	}
}

func TestSemanticsVisibilitySharesTheEvaluationPath(t *testing.T) {
	strict := strings.Replace(fixture, "strict: false", "strict: true", 1)
	policy, _ := Parse(strings.NewReader(strict))
	for _, tool := range []string{"lookup", "bulkDelete", "unknownTool"} {
		load := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: tool, Action: ActionLoad})
		invoke := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: tool, Action: ActionInvoke})
		if load.Allowed != invoke.Allowed || load.Reason != invoke.Reason {
			t.Fatalf("load and invoke drifted for %s: %+v vs %+v", tool, load, invoke)
		}
	}
}

func TestUnknownAgentAndActionFailClosed(t *testing.T) {
	policy := parseFixture(t)
	if decision := policy.Decide(Request{Agent: "ghost", Target: "crm", Tool: "lookup", Action: ActionInvoke}); decision.Allowed {
		t.Fatal("unknown agent must be denied")
	}
	if decision := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: "lookup", Action: "grant"}); decision.Allowed {
		t.Fatal("unknown action must be denied")
	}
}

func TestSubjectSelectors(t *testing.T) {
	policy := parseFixture(t)
	if decision := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: "lookup", Action: ActionInvoke, Subject: "user:support-amy"}); !decision.Allowed {
		t.Fatalf("matching subject should pass: %+v", decision)
	}
	if decision := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: "lookup", Action: ActionInvoke, Subject: "user:finance-bo"}); decision.Allowed {
		t.Fatal("non-matching subject must be denied")
	}
	// An empty subject (the agent acting as itself) skips the selector.
	if decision := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: "lookup", Action: ActionInvoke}); !decision.Allowed {
		t.Fatal("agent acting as itself should not need a subject match")
	}
}

func TestWholeServerGlobCoversNestedNames(t *testing.T) {
	policy := parseFixture(t)
	if decision := policy.Decide(Request{Agent: "codex", Target: "crm", Tool: "deep.nested.tool", Action: ActionInvoke}); !decision.Allowed {
		t.Fatalf("crm.* must cover dotted tool names: %+v", decision)
	}
}

func TestGlobEdges(t *testing.T) {
	cases := []struct {
		pattern string
		value   string
		match   bool
	}{
		{"crm.*", "crm.lookup", true},
		{"crm.*", "crm.a.b", true},
		{"crm.*", "crmX.lookup", false},
		{"crm.lookup", "crm.lookup", true},
		{"crm.lookup", "crm.lookupx", false},
		{"*", "anything.at.all", true},
		{"*delete*", "crm.bulkDelete", true},
		{"*.delete*", "crm.bulkDelete", false},
		{"crm.*delete*", "crm.lookupAndDelete", true},
		{"crm.lookup", "crm.LOOKUP", true},
	}
	for _, test := range cases {
		compiled, err := compileGlob(test.pattern)
		if err != nil {
			t.Fatalf("compile %q: %v", test.pattern, err)
		}
		if got := compiled.match(test.value); got != test.match {
			t.Fatalf("glob %q vs %q = %v, want %v", test.pattern, test.value, got, test.match)
		}
	}
}

func TestParseRejectsUnknownFieldsWithLine(t *testing.T) {
	broken := strings.Replace(fixture, "    domains: [sales]", "    domainz: [sales]", 1)
	_, err := Parse(strings.NewReader(broken))
	if err == nil || !strings.Contains(err.Error(), "harbor.yaml:8") || !strings.Contains(err.Error(), `unknown field "domainz"`) {
		t.Fatalf("unknown field must be rejected with its line: %v", err)
	}
}

func TestParseRejectsBadStructure(t *testing.T) {
	cases := map[string]func(string) string{
		"unsupported version": func(s string) string { return strings.Replace(s, "version: 1", "version: 2", 1) },
		"missing version":     func(s string) string { return strings.Replace(s, "version: 1\n", "", 1) },
		"duplicate workspace": func(s string) string {
			return strings.Replace(s, "agents:", "  - id: main\n    targets: []\nagents:", 1)
		},
		"duplicate agent":       func(s string) string { return s + "  - id: codex\n    workspace: main\n" },
		"unknown workspace ref": func(s string) string { return strings.Replace(s, "workspace: main", "workspace: ghost", 1) },
		"empty deny pattern":    func(s string) string { return strings.Replace(s, `capability: "crm.*delete*"`, `capability: ""`, 1) },
		"missing workspace id":  func(s string) string { return strings.Replace(s, "  - id: main", "  - targets: []", 1) },
	}
	for name, mutate := range cases {
		_, err := Parse(strings.NewReader(mutate(fixture)))
		if err == nil {
			t.Fatalf("%s: expected an error", name)
		}
	}
}

func TestNameRiskFlags(t *testing.T) {
	if flags := NameRiskFlags("crm.bulkDelete"); len(flags) != 1 || flags[0] != "delete" {
		t.Fatalf("bulkDelete should flag delete: %v", flags)
	}
	if flags := NameRiskFlags("crm.ExportSecrets"); len(flags) != 2 || flags[0] != "export" || flags[1] != "secret" {
		t.Fatalf("ExportSecrets should flag export and secret: %v", flags)
	}
	if flags := NameRiskFlags("crm.lookupCustomer"); len(flags) != 0 {
		t.Fatalf("lookup should carry no flags: %v", flags)
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
