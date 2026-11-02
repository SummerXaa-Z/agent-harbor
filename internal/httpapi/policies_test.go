package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestRoutePolicyCRUDAndAudit(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-policy",
		"name":        "Policy Caller",
		"workspaceId": "ws-policy",
		"channelType": "local",
		"status":      "active",
	})
	target := createDirectAgent(t, repo, "Policy Target", "tenant-policy", "ws-policy", "mcp", domain.AgentStatusActive, nil)
	crossScopeTarget := createDirectAgent(t, repo, "Cross Scope Target", "tenant-other", "ws-other", "mcp", domain.AgentStatusActive, nil)

	crossScopeResp := request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Cross scope allow",
		"callerAgentId": caller.ID,
		"targetAgentId": crossScopeTarget.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/list",
		"effect":        "allow",
	}, "")
	if crossScopeResp.Code != http.StatusBadRequest {
		t.Fatalf("cross-scope route policy should be rejected, got %d body=%s", crossScopeResp.Code, crossScopeResp.Body.String())
	}
	badRetryAttempts := request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Bad retry attempts",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/list",
		"effect":        "allow",
		"retry": map[string]any{
			"maxAttempts": 5,
		},
	}, "")
	if badRetryAttempts.Code != http.StatusBadRequest {
		t.Fatalf("route policy retry maxAttempts should be rejected, got %d body=%s", badRetryAttempts.Code, badRetryAttempts.Body.String())
	}

	createResp := request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          " Allow tool list ",
		"callerAgentId": " " + caller.ID + " ",
		"targetAgentId": " " + target.ID + " ",
		"routeType":     " mcp ",
		"routeKey":      " tools/list ",
		"effect":        "allow",
		"priority":      25,
	}, "")
	if createResp.Code != http.StatusCreated {
		t.Fatalf("route policy create failed: status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	created := decodeData[routePolicyResponse](t, createResp)
	if created.TenantID != "tenant-policy" || created.WorkspaceID != "ws-policy" ||
		created.Name != "Allow tool list" || created.RouteType != "mcp" || created.RouteKey != "tools/list" ||
		created.Effect != "allow" || created.Status != "enabled" || created.Priority != 25 {
		t.Fatalf("unexpected created route policy: %#v", created)
	}
	duplicate := request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Duplicate allow tool list",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/list",
		"effect":        "allow",
		"priority":      25,
	}, "")
	if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), "DUPLICATE_RESOURCE_MUTATION") {
		t.Fatalf("duplicate route policy should be a conflict, got %d body=%s", duplicate.Code, duplicate.Body.String())
	}

	list := decodeData[[]routePolicyResponse](t, request(t, router, http.MethodGet, "/api/v1/route-policies?tenantId=tenant-policy&workspaceId=ws-policy", nil, ""))
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("expected scoped route policy list, got %#v", list)
	}

	updated := decodeData[routePolicyResponse](t, request(t, router, http.MethodPatch, "/api/v1/route-policies/"+created.ID, map[string]any{
		"effect":   "deny",
		"name":     "Deny tool list",
		"priority": 40,
		"status":   "enabled",
	}, ""))
	if updated.Effect != "deny" || updated.Name != "Deny tool list" || updated.Priority != 40 || updated.Status != "enabled" {
		t.Fatalf("unexpected updated route policy: %#v", updated)
	}
	badRetryStatus := request(t, router, http.MethodPatch, "/api/v1/route-policies/"+created.ID, map[string]any{
		"retry": map[string]any{
			"statusCodes": []any{429},
		},
	}, "")
	if badRetryStatus.Code != http.StatusBadRequest {
		t.Fatalf("route policy retry statusCodes should be rejected, got %d body=%s", badRetryStatus.Code, badRetryStatus.Body.String())
	}

	disabled := decodeData[routePolicyResponse](t, request(t, router, http.MethodDelete, "/api/v1/route-policies/"+created.ID, nil, ""))
	if disabled.Status != "disabled" {
		t.Fatalf("delete should disable route policy, got %#v", disabled)
	}

	events := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?resourceId="+created.ID, nil, ""))
	if got := auditActions(events); strings.Join(got, ",") != "route_policy.created,route_policy.updated,route_policy.disabled" {
		t.Fatalf("unexpected route policy audit actions: %#v events=%#v", got, events)
	}
}

func TestDirectCrossScopeRoutePolicyIsIgnoredByDataPlane(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-a",
		"name":        "Cross Scope Caller",
		"workspaceId": "ws-a",
		"channelType": "local",
		"status":      "active",
	})
	target := createDirectAgent(t, repo, "Cross Scope Target", "tenant-b", "ws-b", "mcp", domain.AgentStatusActive, nil)
	key := decodeData[keyResponse](t, request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": caller.ID,
	}, ""))

	now := time.Now().UTC()
	if _, err := repo.CreateRoutePolicy(t.Context(), domain.RoutePolicy{
		ID:          security.NewID("rpl"),
		TenantID:    caller.TenantID,
		WorkspaceID: caller.WorkspaceID,
		Name:        "Direct cross scope allow",
		CallerID:    caller.ID,
		TargetID:    target.ID,
		RouteType:   "mcp",
		RouteKey:    "tools/call",
		Effect:      domain.RoutePolicyEffectAllow,
		Status:      domain.RoutePolicyStatusEnabled,
		Priority:    100,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("create direct cross-scope route policy: %v", err)
	}

	policies := decodeData[[]routePolicyResponse](t, request(t, router, http.MethodGet, "/api/v1/route-policies?tenantId=tenant-a&workspaceId=ws-a", nil, ""))
	if len(policies) != 0 {
		t.Fatalf("direct cross-scope route policy should not be visible in scoped management list, got %#v", policies)
	}

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("direct cross-scope route policy should not allow data-plane call, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRoutePolicyCreatePreservesExplicitZeroPriority(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller, key := createLocalCallerWithKey(t, router, "Zero Priority Caller")
	target := createDirectAgent(t, repo, "Zero Priority Target", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)

	deny := decodeData[routePolicyResponse](t, request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Medium deny",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
		"effect":        "deny",
		"priority":      50,
	}, ""))
	allow := decodeData[routePolicyResponse](t, request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Lowest allow",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
		"effect":        "allow",
		"priority":      0,
	}, ""))
	if deny.Priority != 50 || allow.Priority != 0 {
		t.Fatalf("expected explicit priorities to be preserved, deny=%#v allow=%#v", deny, allow)
	}

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("priority 50 deny should beat explicit priority 0 allow, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRoutePoliciesDriveDataPlaneBeforeLegacyGrants(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller, key := createLocalCallerWithKey(t, router, "Route Policy Caller")
	target := createDirectAgent(t, repo, "Route Policy Target", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	legacyAllowed := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-route-policy")
	if legacyAllowed.Code != http.StatusOK {
		t.Fatalf("legacy grant should allow before policies, got %d body=%s", legacyAllowed.Code, legacyAllowed.Body.String())
	}

	denyPolicy := decodeData[routePolicyResponse](t, request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Deny calls",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
		"effect":        "deny",
		"priority":      100,
	}, ""))
	denied := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-route-policy")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("route policy deny should override legacy grant, got %d body=%s", denied.Code, denied.Body.String())
	}

	allowPolicy := decodeData[routePolicyResponse](t, request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Allow calls",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
		"effect":        "allow",
		"priority":      200,
	}, ""))
	allowed := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-route-policy")
	if allowed.Code != http.StatusOK {
		t.Fatalf("higher priority allow policy should win, got %d body=%s", allowed.Code, allowed.Body.String())
	}

	request(t, router, http.MethodDelete, "/api/v1/route-policies/"+allowPolicy.ID, nil, "")
	deniedAgain := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-route-policy")
	if deniedAgain.Code != http.StatusForbidden {
		t.Fatalf("disabled allow should reveal deny policy %s, got %d", denyPolicy.ID, deniedAgain.Code)
	}

	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-route-policy", nil, ""))
	reasons := make([]string, 0, len(traces))
	for _, trace := range traces {
		reasons = append(reasons, trace.Reason)
	}
	if !strings.Contains(strings.Join(reasons, ","), "route policy denied") || !strings.Contains(strings.Join(reasons, ","), "route policy allowed") {
		t.Fatalf("expected route policy trace reasons, got %#v", traces)
	}
}

func TestRevokeAccessGrantDeniesLaterCalls(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller := createAgent(t, router, map[string]any{
		"name":        "Revocable Caller",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	})
	target := createDirectAgent(t, repo, "Revocable Target", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	key := decodeData[keyResponse](t, request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": caller.ID,
	}, ""))
	grant := decodeData[grantResponse](t, request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, ""))

	allowed := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected allowed before grant revoke, got %d", allowed.Code)
	}

	revoked := request(t, router, http.MethodDelete, "/api/v1/access-grants/"+grant.ID, nil, "")
	if revoked.Code != http.StatusOK {
		t.Fatalf("expected revoke grant success, got %d body=%s", revoked.Code, revoked.Body.String())
	}
	denied := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("revoked grant should deny later call, got %d", denied.Code)
	}
}
