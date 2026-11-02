package httpapi_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestMCPCapabilityDiscoveryAndAssignmentManagement(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"capability-discovery","result":{"tools":[{"name":"search_customer","description":"Search customers","inputSchema":{"type":"object"}},{"name":"export_contracts","description":"Export contracts","inputSchema":{"type":"object"}}]}}`))
	}))
	defer upstream.Close()

	now := time.Now().UTC()
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-a", WorkspaceID: "ws-sales", Name: "Capability Caller", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Capability MCP", "tenant-a", "ws-sales", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL,
	})

	capabilities := decodeData[[]capabilityResponse](t, request(t, router, http.MethodPost, "/api/v1/targets/"+target.ID+"/capabilities:refresh", nil, ""))
	if len(capabilities) != 2 {
		t.Fatalf("expected two discovered capabilities, got %#v", capabilities)
	}
	search := capabilityByKey(t, capabilities, "search_customer")
	if search.DiscoveryStatus != "pending_review" || search.Action != "read" {
		t.Fatalf("unexpected search capability defaults: %#v", search)
	}
	export := capabilityByKey(t, capabilities, "export_contracts")
	if export.DiscoveryStatus != "pending_review" || export.Action != "export" || export.RiskLevel != "high" {
		t.Fatalf("unexpected export capability defaults: %#v", export)
	}

	approved := decodeData[capabilityResponse](t, request(t, router, http.MethodPatch, "/api/v1/capabilities/"+search.ID, map[string]any{
		"dataDomains":     []string{" crm ", "crm"},
		"discoveryStatus": "approved",
	}, ""))
	if approved.DiscoveryStatus != "approved" || !reflect.DeepEqual(approved.DataDomains, []string{"crm"}) {
		t.Fatalf("capability should be approved, got %#v", approved)
	}
	refreshed := decodeData[[]capabilityResponse](t, request(t, router, http.MethodPost, "/api/v1/targets/"+target.ID+"/capabilities:refresh", nil, ""))
	refreshedSearch := capabilityByKey(t, refreshed, "search_customer")
	if refreshedSearch.DiscoveryStatus != "approved" || !reflect.DeepEqual(refreshedSearch.DataDomains, []string{"crm"}) {
		t.Fatalf("capability classification should survive discovery refresh, got %#v", refreshedSearch)
	}
	entitlement := decodeData[tenantEntitlementResponse](t, request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-a",
		"targetId":     target.ID,
		"capabilityId": search.ID,
		"effect":       "allow",
		"status":       "enabled",
	}, ""))
	if entitlement.TenantID != "tenant-a" || entitlement.CapabilityID != search.ID {
		t.Fatalf("unexpected entitlement: %#v", entitlement)
	}
	workspaceAssignment := decodeData[workspaceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/workspace-assignments", map[string]any{
		"tenantEntitlementId": entitlement.ID,
		"workspaceId":         "ws-sales",
		"effect":              "allow",
		"status":              "enabled",
	}, ""))
	if workspaceAssignment.WorkspaceID != "ws-sales" {
		t.Fatalf("unexpected workspace assignment: %#v", workspaceAssignment)
	}
	for _, subjectSelector := range []string{"", " ", "*"} {
		resp := request(t, router, http.MethodPost, "/api/v1/instance-assignments", map[string]any{
			"workspaceAssignmentId": workspaceAssignment.ID,
			"callerInstanceId":      caller.ID,
			"subjectSelector":       subjectSelector,
			"effect":                "allow",
			"status":                "enabled",
		}, "")
		if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Body.String(), "subjectSelector") {
			t.Fatalf("subjectSelector %q should be rejected, status=%d body=%s", subjectSelector, resp.Code, resp.Body.String())
		}
	}
	instanceAssignment := decodeData[instanceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/instance-assignments", map[string]any{
		"workspaceAssignmentId": workspaceAssignment.ID,
		"callerInstanceId":      caller.ID,
		"subjectSelector":       "user:sales-*",
		"effect":                "allow",
		"status":                "enabled",
	}, ""))
	if instanceAssignment.CallerInstanceID != caller.ID {
		t.Fatalf("unexpected instance assignment: %#v", instanceAssignment)
	}
}
