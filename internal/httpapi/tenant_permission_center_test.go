package httpapi_test

import (
	"bytes"
	"context"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestTenantPermissionCenterSummarizesTenantForPlatformAdmin(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})
	tenantID, workspaceID, _, target, capability := seedTenantPermissionCenterFixture(t, repo)
	now := time.Now().UTC()
	_, err := repo.CreateAdminIdentityWithAudit(context.Background(), domain.AdminIdentity{
		ID:          "adm_center_tenant",
		Actor:       "tenant-admin-center",
		DisplayName: "Tenant Center Admin",
		Role:        domain.AdminIdentityRoleTenantAdmin,
		KeyHash:     security.HashSecret("tenant-admin-key"),
		KeyPrefix:   "ahadm_center",
		Status:      domain.AdminIdentityStatusActive,
		Source:      domain.AdminIdentitySourceManaged,
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, func(identity domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: "audit_center_admin", Action: "admin_identity.created", ResourceType: "admin_identity", ResourceID: identity.ID, Actor: "platform", CreatedAt: now}
	})
	if err != nil {
		t.Fatalf("seed admin identity: %v", err)
	}

	center := decodeData[tenantPermissionCenterResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/"+tenantID+"/permission-center", nil, "", "platform-key"))
	if center.Tenant.ID != tenantID {
		t.Fatalf("unexpected tenant center tenant: %#v", center.Tenant)
	}
	if center.OperatorBoundary.Actor != "platform" || !center.OperatorBoundary.CanManageAdministrators {
		t.Fatalf("platform boundary should manage administrators, got %#v", center.OperatorBoundary)
	}
	if len(center.Administrators) != 1 || center.Administrators[0].Actor != "tenant-admin-center" {
		t.Fatalf("expected tenant administrator summary, got %#v", center.Administrators)
	}
	if len(center.Workspaces) != 1 || center.Workspaces[0].WorkspaceID != workspaceID || center.Workspaces[0].CallerCount != 1 || center.Workspaces[0].TargetCount != 1 {
		t.Fatalf("unexpected workspace summary: %#v", center.Workspaces)
	}
	if len(center.Capabilities) != 1 || center.Capabilities[0].CapabilityID != capability.ID || center.Capabilities[0].TargetID != target.ID || center.Capabilities[0].Effect != "allow" {
		t.Fatalf("unexpected capability summary: %#v", center.Capabilities)
	}
	if len(center.PermissionPacks) == 0 || center.PermissionPacks[0].Status == "" {
		t.Fatalf("expected permission package projection, got %#v", center.PermissionPacks)
	}
	if got := tenantCenterActionCodes(center.NextActions); !reflect.DeepEqual(got, []string{"manage_administrators", "open_access_profile", "start_permission_change"}) {
		t.Fatalf("unexpected next actions: %#v", got)
	}
	raw := mustJSON(t, center)
	if bytes.Contains(raw, []byte("tenant-admin-key")) || bytes.Contains(raw, []byte("keyHash")) {
		t.Fatalf("tenant center must not expose admin key material: %s", raw)
	}
}

func TestTenantPermissionCenterHonorsScopedAdminBoundary(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-child-center", WorkspaceID: "ws-support-center"},
	})
	tenantID, workspaceID, _, _, _ := seedTenantPermissionCenterFixture(t, repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-west-center", "tenant-root-center", "Finance", now)

	scopedResponse := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/"+tenantID+"/permission-center", nil, "", "east-key")
	center := decodeData[tenantPermissionCenterResponse](t, scopedResponse)
	if center.OperatorBoundary.Actor != "east-admin" || center.OperatorBoundary.CanManageAdministrators {
		t.Fatalf("tenant admin boundary should be read-only for admin management, got %#v", center.OperatorBoundary)
	}
	if center.Administrators == nil || bytes.Contains(scopedResponse.Body.Bytes(), []byte(`"administrators":null`)) {
		t.Fatalf("tenant admin boundary should return an empty administrators array, body=%s", scopedResponse.Body.String())
	}
	if center.OperatorBoundary.TenantID != tenantID || center.OperatorBoundary.WorkspaceID != workspaceID {
		t.Fatalf("unexpected scoped boundary: %#v", center.OperatorBoundary)
	}
	if got := tenantCenterActionCodes(center.NextActions); slices.Contains(got, "manage_administrators") {
		t.Fatalf("tenant admin should not get manage_administrators action: %#v", got)
	}

	widen := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/tenant-west-center/permission-center", nil, "", "east-key")
	if widen.Code != http.StatusForbidden {
		t.Fatalf("tenant admin should not fetch outside tenant center, got %d body=%s", widen.Code, widen.Body.String())
	}
}

func TestTenantPermissionCenterFiltersCapabilitiesToScopedWorkspace(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-child-center", WorkspaceID: "ws-support-center"},
	})
	tenantID, _, _, _, supportCapability := seedTenantPermissionCenterFixture(t, repo)
	now := time.Now().UTC()
	financeTarget := createDirectAgent(t, repo, "Finance Export Service", tenantID, "ws-finance-center", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	financeEntitlement := createDirectTenantEntitlement(t, repo, tenantID, financeTarget.ID, financeCapability.ID, []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)
	createDirectWorkspaceAssignment(t, repo, financeEntitlement.ID, tenantID, "ws-finance-center", []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)

	center := decodeData[tenantPermissionCenterResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/"+tenantID+"/permission-center", nil, "", "support-key"))
	if len(center.Capabilities) != 1 {
		t.Fatalf("scoped workspace should see exactly one capability, got %#v", center.Capabilities)
	}
	if center.Capabilities[0].CapabilityID != supportCapability.ID {
		t.Fatalf("scoped workspace leaked another capability: %#v", center.Capabilities)
	}
	raw := mustJSON(t, center)
	if bytes.Contains(raw, []byte(financeCapability.ID)) || bytes.Contains(raw, []byte("finance")) || bytes.Contains(raw, []byte("ws-finance-center")) {
		t.Fatalf("scoped workspace should not expose finance capability or data scope: %s", raw)
	}
}

func TestTenantPermissionCenterRedactsOutOfScopeHydratedObjects(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-child-center", WorkspaceID: "ws-support-center"},
	})
	tenantID, _, _, _, _ := seedTenantPermissionCenterFixture(t, repo)
	now := time.Now().UTC()
	financeTarget := createDirectAgent(t, repo, "Finance Export Service", tenantID, "ws-finance-center", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	financeEntitlement := createDirectTenantEntitlement(t, repo, tenantID, financeTarget.ID, financeCapability.ID, []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)
	createDirectWorkspaceAssignment(t, repo, financeEntitlement.ID, tenantID, "ws-support-center", []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)

	resp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/"+tenantID+"/permission-center", nil, "", "support-key")
	if resp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read tenant permission center, got %d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if strings.Contains(body, "Finance Export Service") || strings.Contains(body, "export_invoices") {
		t.Fatalf("tenant permission center leaked hydrated out-of-scope object details: %s", body)
	}
}

func TestTenantPermissionCenterRequiresRegisteredTenant(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})
	resp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/unregistered/permission-center", nil, "", "platform-key")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("permission center should require registered tenant, got %d body=%s", resp.Code, resp.Body.String())
	}
}
