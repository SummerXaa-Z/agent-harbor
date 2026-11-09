package httpapi_test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/policy"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

// policyFixture builds a strict harbor.yaml over dynamic ids. Line map:
// workspace at 3, allow rule at 11, deny rule at 13.
func policyFixture(callerID, targetID string) string {
	return fmt.Sprintf(`version: 1
workspaces:
  - id: main
    strict: true
    targets:
      - id: %s
agents:
  - id: %s
    workspace: main
    allow:
      - "%s.search*"
    deny:
      - capability: "%s.*delete*"
        reason: prod database is read-only
`, targetID, callerID, targetID, targetID)
}

func newRouterWithRepoAndPolicy(repo store.Repository, parsed *policy.Policy) http.Handler {
	return httpapi.New(repo,
		httpapi.WithUnauthenticatedAdminAllowed(true),
		httpapi.WithPolicy(parsed),
	).Router()
}

func setupPolicyGateFixture(t *testing.T) (http.Handler, store.Repository, string, string, string) {
	t.Helper()
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	caller := createDirectAgent(t, repo, "Policy Caller", "tenant-a", "ws-main", "local", domain.AgentStatusActive, nil)
	target := createDirectAgent(t, repo, "Policy Target", "tenant-a", "ws-main", "mcp", domain.AgentStatusActive, nil)
	search := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	bulk := createDirectCapabilityWithAction(t, repo, target.ID, "bulk_delete_customer", domain.CapabilityActionDelete, domain.CapabilityRiskCritical, domain.CapabilitySensitivityInternal, now)
	for _, capability := range []domain.Capability{search, bulk} {
		approved := request(t, router, http.MethodPatch, "/api/v1/capabilities/"+capability.ID, map[string]any{"discoveryStatus": "approved"}, "")
		if approved.Code != http.StatusOK {
			t.Fatalf("approve capability: status=%d body=%s", approved.Code, approved.Body.String())
		}
	}
	key := createDirectTestAgentKey(t, repo, caller.ID, now)
	parsed, err := policy.Parse(strings.NewReader(policyFixture(caller.ID, target.ID)))
	if err != nil {
		t.Fatalf("parse policy: %v", err)
	}
	return newRouterWithRepoAndPolicy(repo, parsed), repo, key, target.ID, caller.ID
}

func TestPolicyGateAllowsWithinAllowRuleAndRecordsPolicyFields(t *testing.T) {
	router, repo, key, targetID, _ := setupPolicyGateFixture(t)
	rec := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+targetID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "policy-allow",
		"method":  "tools/call",
		"params":  map[string]any{"name": "search_customer", "arguments": map[string]any{"query": "A-1"}},
	}, key, "policy-allow", "user:support-001")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected allowed call, got status=%d body=%s", rec.Code, rec.Body.String())
	}
	traces, err := repo.ListTraces(t.Context(), store.TraceFilter{RunID: "policy-allow"})
	if err != nil || len(traces) != 1 {
		t.Fatalf("expected one trace: rows=%d err=%v", len(traces), err)
	}
	trace := traces[0]
	if trace.Decision != domain.TraceDecisionAllowed || trace.RuleRef != "harbor.yaml:11" {
		t.Fatalf("policy allow trace wrong: %+v", trace)
	}
	if trace.ArgsDigest == "" || !strings.HasPrefix(trace.ArgsDigest, "sha256:") {
		t.Fatalf("args digest missing: %+v", trace)
	}
	if len(trace.RiskFlags) != 0 {
		t.Fatalf("read tool should carry no risk flags: %+v", trace.RiskFlags)
	}
}

func TestPolicyGateDenialExplainsItself(t *testing.T) {
	router, repo, key, targetID, _ := setupPolicyGateFixture(t)
	rec := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+targetID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "policy-deny",
		"method":  "tools/call",
		"params":  map[string]any{"name": "bulk_delete_customer", "arguments": map[string]any{}},
	}, key, "policy-deny", "user:support-001")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected denial, got status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Do not retry", "prod database is read-only", "audit trc_"} {
		if !strings.Contains(body, want) {
			t.Fatalf("denial copy missing %q: %s", want, body)
		}
	}
	if !strings.Contains(body, `"policy":"harbor.yaml:13"`) {
		t.Fatalf("denial details must cite the deny rule line: %s", body)
	}
	traces, err := repo.ListTraces(t.Context(), store.TraceFilter{RunID: "policy-deny"})
	if err != nil || len(traces) != 1 {
		t.Fatalf("expected one trace: rows=%d err=%v", len(traces), err)
	}
	trace := traces[0]
	if trace.Decision != domain.TraceDecisionDenied || trace.RuleRef != "harbor.yaml:13" {
		t.Fatalf("policy deny trace wrong: %+v", trace)
	}
	// The name pattern and the declared delete action collapse into one
	// deduplicated flag - this pins the merge path in capabilityRiskFlags.
	if len(trace.RiskFlags) != 1 || trace.RiskFlags[0] != "delete" {
		t.Fatalf("expected the merged delete flag: %+v", trace.RiskFlags)
	}
}

func TestPolicyGateStrictModeFiltersToolsListVisibility(t *testing.T) {
	router, _, key, targetID, _ := setupPolicyGateFixture(t)
	rec := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+targetID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "policy-list",
		"method":  "tools/list",
		"params":  map[string]any{},
	}, key, "policy-list", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("tools/list failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "search_customer") {
		t.Fatalf("allowed tool must be visible: %s", body)
	}
	if strings.Contains(body, "bulk_delete_customer") {
		t.Fatalf("strict mode must hide denied tools: %s", body)
	}
}

func TestLegacyChainModeUnchangedWithoutPolicy(t *testing.T) {
	// The same fixture with no policy loaded must keep the grant-chain
	// behavior: nothing is granted, so both tools are denied.
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	caller := createDirectAgent(t, repo, "Legacy Caller", "tenant-a", "ws-main", "local", domain.AgentStatusActive, nil)
	target := createDirectAgent(t, repo, "Legacy Target", "tenant-a", "ws-main", "mcp", domain.AgentStatusActive, nil)
	capability := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	approved := request(t, router, http.MethodPatch, "/api/v1/capabilities/"+capability.ID, map[string]any{"discoveryStatus": "approved"}, "")
	if approved.Code != http.StatusOK {
		t.Fatalf("approve capability: status=%d", approved.Code)
	}
	key := createDirectTestAgentKey(t, repo, caller.ID, now)
	rec := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "legacy-call",
		"method":  "tools/call",
		"params":  map[string]any{"name": "search_customer", "arguments": map[string]any{}},
	}, key, "legacy-call", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("legacy chain mode must deny ungranted calls, got status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "Do not retry") {
		t.Fatalf("legacy denials must keep the plain copy: %s", rec.Body.String())
	}
}

// TestPolicyGateIsTheSingleDataPlaneEntry pins the architecture constraint:
// tools/list and tools/call both route through policyDecide, and the legacy
// chain evaluation only ever runs behind an explicit nil-policy branch.
func TestPolicyGateIsTheSingleDataPlaneEntry(t *testing.T) {
	source, err := os.ReadFile("data_plane.go")
	if err != nil {
		t.Fatalf("read data_plane.go: %v", err)
	}
	if got := strings.Count(string(source), "s.policyDecide("); got != 2 {
		t.Fatalf("policyDecide must be called from exactly tools/call and tools/list (single filter), found %d call sites", got)
	}
	if got := strings.Count(string(source), "s.policy != nil"); got != 2 {
		t.Fatalf("legacy chain branches must stay explicitly guarded, found %d nil-policy checks", got)
	}
	if strings.Contains(string(source), "policy.Policy{") {
		t.Fatal("data_plane.go must not construct policies; it consumes the loaded one")
	}
}
