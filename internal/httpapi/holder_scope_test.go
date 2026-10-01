package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpapi "github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

type holderScopeErrorEnvelope struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func decodeResponseBody(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response body %q: %v", rec.Body.String(), err)
	}
}

func createHolderAdminIdentity(t *testing.T, repo store.Repository, id string, actor string, key string, role domain.AdminIdentityRole, tenantID string, workspaceID string, ownedAgentIDs []string) domain.AdminIdentity {
	t.Helper()
	now := time.Now().UTC()
	identity, err := repo.CreateAdminIdentityWithAudit(context.Background(), domain.AdminIdentity{
		ID:            id,
		Actor:         actor,
		DisplayName:   actor,
		Role:          role,
		KeyHash:       security.HashSecret(key),
		KeyPrefix:     "ah_test",
		Status:        domain.AdminIdentityStatusActive,
		Source:        domain.AdminIdentitySourceManaged,
		TenantID:      tenantID,
		WorkspaceID:   workspaceID,
		OwnedAgentIDs: ownedAgentIDs,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, func(created domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: "audit_" + id, Action: "admin_identity.created", ResourceType: "admin_identity", ResourceID: created.ID, Actor: "platform", CreatedAt: now}
	})
	if err != nil {
		t.Fatalf("create holder admin identity %s: %v", actor, err)
	}
	return identity
}

func TestHolderScopeJourneyMatrix(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "bootstrap-tenant", Key: "bootstrap-tenant-key", Role: "tenant_admin", TenantID: "tenant-child-center", WorkspaceID: "ws-support-center"},
	})
	tenantID, workspaceID, caller, _, _ := seedTenantPermissionCenterFixture(t, repo)
	otherCaller := createDirectAgent(t, repo, "Other Caller", tenantID, workspaceID, "local", domain.AgentStatusActive, nil)

	createHolderAdminIdentity(t, repo, "adm_unbound", "unbound-admin", "unbound-key", domain.AdminIdentityRoleTenantAdmin, tenantID, workspaceID, nil)
	createHolderAdminIdentity(t, repo, "adm_bound", "bound-admin", "bound-key", domain.AdminIdentityRoleTenantAdmin, tenantID, workspaceID, []string{caller.ID})
	createHolderAdminIdentity(t, repo, "adm_reviewer", "bound-reviewer", "reviewer-key", domain.AdminIdentityRoleSecurityReviewer, tenantID, "", []string{caller.ID})

	profilePath := func(callerID string) string {
		return "/api/v1/tenants/" + tenantID + "/access-profile?workspaceId=" + workspaceID + "&callerInstanceId=" + callerID
	}
	draftPath := "/api/v1/permission-packages/drafts"

	cases := []struct {
		name       string
		key        string
		callerID   string
		wantStatus int
		wantCode   string
	}{
		{"platform admin sees any caller", "platform-key", otherCaller.ID, http.StatusOK, ""},
		{"bootstrap tenant admin keeps unbound fallback", "bootstrap-tenant-key", otherCaller.ID, http.StatusOK, ""},
		{"unbound managed tenant admin keeps fallback", "unbound-key", otherCaller.ID, http.StatusOK, ""},
		{"bound tenant admin sees owned caller", "bound-key", caller.ID, http.StatusOK, ""},
		{"bound tenant admin denied other caller", "bound-key", otherCaller.ID, http.StatusForbidden, "HOLDER_SCOPE_DENIED"},
		{"bound reviewer sees owned caller", "reviewer-key", caller.ID, http.StatusOK, ""},
		{"bound reviewer denied other caller", "reviewer-key", otherCaller.ID, http.StatusForbidden, "HOLDER_SCOPE_DENIED"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rec := requestWithAdmin(t, router, http.MethodGet, profilePath(testCase.callerID), nil, "", testCase.key)
			if rec.Code != testCase.wantStatus {
				t.Fatalf("access profile status = %d want %d body=%s", rec.Code, testCase.wantStatus, rec.Body.String())
			}
			if testCase.wantCode != "" {
				var envelope holderScopeErrorEnvelope
				decodeResponseBody(t, rec, &envelope)
				if envelope.Error != testCase.wantCode {
					t.Fatalf("error code = %q want %q body=%s", envelope.Error, testCase.wantCode, rec.Body.String())
				}
			}
		})
	}

	draftDenied := requestWithAdmin(t, router, http.MethodPost, draftPath, map[string]any{
		"tenantId":         tenantID,
		"workspaceId":      workspaceID,
		"callerInstanceId": otherCaller.ID,
	}, "", "bound-key")
	if draftDenied.Code != http.StatusForbidden {
		t.Fatalf("draft status = %d want %d body=%s", draftDenied.Code, http.StatusForbidden, draftDenied.Body.String())
	}
	var envelope holderScopeErrorEnvelope
	decodeResponseBody(t, draftDenied, &envelope)
	if envelope.Error != "HOLDER_SCOPE_DENIED" {
		t.Fatalf("draft error code = %q body=%s", envelope.Error, draftDenied.Body.String())
	}

	draftOwned := requestWithAdmin(t, router, http.MethodPost, draftPath, map[string]any{
		"tenantId":         tenantID,
		"workspaceId":      workspaceID,
		"callerInstanceId": caller.ID,
	}, "", "bound-key")
	if draftOwned.Code == http.StatusForbidden {
		t.Fatalf("owned caller draft must not be holder-denied, body=%s", draftOwned.Body.String())
	}
}

func TestHolderScopeRebindTakesEffectImmediately(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})
	tenantID, workspaceID, caller, _, _ := seedTenantPermissionCenterFixture(t, repo)
	otherCaller := createDirectAgent(t, repo, "Other Caller", tenantID, workspaceID, "local", domain.AgentStatusActive, nil)
	identity := createHolderAdminIdentity(t, repo, "adm_rebind", "rebind-admin", "rebind-key", domain.AdminIdentityRoleTenantAdmin, tenantID, workspaceID, []string{caller.ID})

	profilePath := func(callerID string) string {
		return "/api/v1/tenants/" + tenantID + "/access-profile?workspaceId=" + workspaceID + "&callerInstanceId=" + callerID
	}
	if rec := requestWithAdmin(t, router, http.MethodGet, profilePath(caller.ID), nil, "", "rebind-key"); rec.Code != http.StatusOK {
		t.Fatalf("owned caller should pass before rebind, got %d body=%s", rec.Code, rec.Body.String())
	}

	patched := requestWithAdmin(t, router, http.MethodPatch, "/api/v1/admin-identities/"+identity.ID, map[string]any{
		"ownedAgentIds": []string{otherCaller.ID},
	}, "", "platform-key")
	if patched.Code != http.StatusOK {
		t.Fatalf("rebind status = %d want 200 body=%s", patched.Code, patched.Body.String())
	}
	if rec := requestWithAdmin(t, router, http.MethodGet, profilePath(caller.ID), nil, "", "rebind-key"); rec.Code != http.StatusForbidden {
		t.Fatalf("previous owned caller should be denied immediately after rebind, got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := requestWithAdmin(t, router, http.MethodGet, profilePath(otherCaller.ID), nil, "", "rebind-key"); rec.Code != http.StatusOK {
		t.Fatalf("newly owned caller should pass immediately after rebind, got %d body=%s", rec.Code, rec.Body.String())
	}
}

type holderScopeListRow struct {
	ID               string `json:"id"`
	CallerInstanceID string `json:"callerInstanceId,omitempty"`
}

type holderScopeAgentKeyRow struct {
	ID      string `json:"id"`
	AgentID string `json:"agentId"`
}

func TestHolderScopeNarrowsLists(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})
	tenantID, workspaceID, _, _, _ := seedTenantPermissionCenterFixture(t, repo)
	now := time.Now().UTC()
	first := createDirectPermissionPackageApprovalRequest(t, repo, "ar_holder_first", tenantID, workspaceID, now)
	second := createDirectPermissionPackageApprovalRequest(t, repo, "ar_holder_second", tenantID, workspaceID, now)
	createHolderAdminIdentity(t, repo, "adm_list", "list-admin", "list-key", domain.AdminIdentityRoleTenantAdmin, tenantID, workspaceID, []string{"agt_caller_ar_holder_first"})

	approvals := decodeData[[]holderScopeListRow](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests", nil, "", "list-key"))
	if len(approvals) != 1 || approvals[0].ID != first.ID || approvals[0].CallerInstanceID != "agt_caller_ar_holder_first" {
		t.Fatalf("approval list should narrow to owned caller, got %#v", approvals)
	}

	if _, err := repo.CreatePermissionPackageApplication(context.Background(), domain.PermissionPackageApplication{
		ID: "ppa_holder_first", DraftID: "ppd_holder_first", TemplateID: "support-ticket-triage", TemplateVersion: 1,
		TenantID: tenantID, WorkspaceID: workspaceID, TargetID: first.TargetID, CallerInstanceID: "agt_caller_ar_holder_first",
		SubjectSelector: "user:support-*", AllowedCapabilityIDs: []string{}, AllowedCapabilityKeys: []string{},
		TenantEntitlementIDs: []string{}, WorkspaceAssignmentIDs: []string{}, InstanceAssignmentIDs: []string{},
		AppliedAt: now,
	}); err != nil {
		t.Fatalf("create first application: %v", err)
	}
	if _, err := repo.CreatePermissionPackageApplication(context.Background(), domain.PermissionPackageApplication{
		ID: "ppa_holder_second", DraftID: "ppd_holder_second", TemplateID: "support-ticket-triage", TemplateVersion: 1,
		TenantID: tenantID, WorkspaceID: workspaceID, TargetID: second.TargetID, CallerInstanceID: "agt_caller_ar_holder_second",
		SubjectSelector: "user:support-*", AllowedCapabilityIDs: []string{}, AllowedCapabilityKeys: []string{},
		TenantEntitlementIDs: []string{}, WorkspaceAssignmentIDs: []string{}, InstanceAssignmentIDs: []string{},
		AppliedAt: now,
	}); err != nil {
		t.Fatalf("create second application: %v", err)
	}
	applications := decodeData[[]holderScopeListRow](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/applications", nil, "", "list-key"))
	if len(applications) != 1 || applications[0].ID != "ppa_holder_first" {
		t.Fatalf("application list should narrow to owned caller, got %#v", applications)
	}

	if _, err := repo.CreateAgentKey(context.Background(), domain.AgentKey{ID: "key_holder_first", AgentID: "agt_caller_ar_holder_first", Name: "first", CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}); err != nil {
		t.Fatalf("create first agent key: %v", err)
	}
	if _, err := repo.CreateAgentKey(context.Background(), domain.AgentKey{ID: "key_holder_second", AgentID: "agt_caller_ar_holder_second", Name: "second", CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}); err != nil {
		t.Fatalf("create second agent key: %v", err)
	}
	keys := decodeData[[]holderScopeAgentKeyRow](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/api-keys", nil, "", "list-key"))
	if len(keys) != 1 || keys[0].ID != "key_holder_first" {
		t.Fatalf("agent key list should narrow to owned agent, got %#v", keys)
	}
}

func TestUpdateAdminIdentityOwnedAgents(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "tenant-op", Key: "tenant-op-key", Role: "tenant_admin", TenantID: "tenant-child-center", WorkspaceID: "ws-support-center"},
	})
	tenantID, workspaceID, caller, _, _ := seedTenantPermissionCenterFixture(t, repo)
	otherCaller := createDirectAgent(t, repo, "Other Caller", tenantID, workspaceID, "local", domain.AgentStatusActive, nil)
	createDirectTenant(t, repo, "tenant-holder-foreign", "", "Foreign", time.Now().UTC())
	foreignCaller := createDirectAgent(t, repo, "Foreign Caller", "tenant-holder-foreign", "ws-foreign", "local", domain.AgentStatusActive, nil)

	created := decodeData[createAdminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
		"actor":         "bound-operator",
		"role":          "tenant_admin",
		"tenantId":      tenantID,
		"workspaceId":   workspaceID,
		"ownedAgentIds": []string{caller.ID},
	}, "", "platform-key"))
	if len(created.Identity.OwnedAgentIDs) != 1 || created.Identity.OwnedAgentIDs[0] != caller.ID {
		t.Fatalf("created identity should carry owned agents, got %#v", created.Identity.OwnedAgentIDs)
	}

	scopedCreate := requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
		"actor":         "scoped-creator",
		"role":          "tenant_admin",
		"tenantId":      tenantID,
		"ownedAgentIds": []string{caller.ID},
	}, "", "tenant-op-key")
	if scopedCreate.Code != http.StatusForbidden {
		t.Fatalf("scoped admin must not create identities, got %d body=%s", scopedCreate.Code, scopedCreate.Body.String())
	}

	foreignBind := requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
		"actor":         "foreign-bound",
		"role":          "tenant_admin",
		"tenantId":      tenantID,
		"ownedAgentIds": []string{foreignCaller.ID},
	}, "", "platform-key")
	if foreignBind.Code != http.StatusBadRequest {
		t.Fatalf("binding outside the identity tenant should fail, got %d body=%s", foreignBind.Code, foreignBind.Body.String())
	}

	platformBind := decodeData[createAdminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
		"actor":         "platform-bound",
		"role":          "platform_admin",
		"ownedAgentIds": []string{foreignCaller.ID},
	}, "", "platform-key"))
	if len(platformBind.Identity.OwnedAgentIDs) != 1 || platformBind.Identity.OwnedAgentIDs[0] != foreignCaller.ID {
		t.Fatalf("platform identity may bind any agent, got %#v", platformBind.Identity.OwnedAgentIDs)
	}

	unknownAgent := requestWithAdmin(t, router, http.MethodPatch, "/api/v1/admin-identities/"+created.Identity.ID, map[string]any{
		"ownedAgentIds": []string{"agt_missing"},
	}, "", "platform-key")
	if unknownAgent.Code != http.StatusBadRequest {
		t.Fatalf("unknown agent should fail, got %d body=%s", unknownAgent.Code, unknownAgent.Body.String())
	}

	bootstrap := requestWithAdmin(t, router, http.MethodPatch, "/api/v1/admin-identities/bootstrap:platform", map[string]any{
		"ownedAgentIds": []string{caller.ID},
	}, "", "platform-key")
	if bootstrap.Code != http.StatusBadRequest {
		t.Fatalf("bootstrap identity should be read-only, got %d body=%s", bootstrap.Code, bootstrap.Body.String())
	}

	scopedPatch := requestWithAdmin(t, router, http.MethodPatch, "/api/v1/admin-identities/"+created.Identity.ID, map[string]any{
		"ownedAgentIds": []string{caller.ID},
	}, "", "tenant-op-key")
	if scopedPatch.Code != http.StatusForbidden {
		t.Fatalf("scoped admin must not patch identities, got %d body=%s", scopedPatch.Code, scopedPatch.Body.String())
	}

	patched := decodeData[adminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPatch, "/api/v1/admin-identities/"+created.Identity.ID, map[string]any{
		"ownedAgentIds": []string{caller.ID, otherCaller.ID, "  ", caller.ID},
	}, "", "platform-key"))
	if len(patched.OwnedAgentIDs) != 2 || patched.OwnedAgentIDs[0] != caller.ID || patched.OwnedAgentIDs[1] != otherCaller.ID {
		t.Fatalf("patch should trim, dedupe and sort owned agents, got %#v", patched.OwnedAgentIDs)
	}

	events, err := repo.ListAuditEvents(context.Background(), store.AuditEventFilter{ResourceType: "admin_identity", Limit: 50})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	var updatedEvent *domain.AuditEvent
	for i, event := range events {
		if event.Action == "admin_identity.updated" && event.ResourceID == created.Identity.ID {
			updatedEvent = &events[i]
			break
		}
	}
	if updatedEvent == nil {
		t.Fatalf("expected admin_identity.updated audit event, got %#v", events)
	}
	if added, ok := updatedEvent.Metadata["addedOwnedAgentIds"].([]string); !ok || len(added) != 1 || added[0] != otherCaller.ID {
		t.Fatalf("updated event should record added agents, got %#v", updatedEvent.Metadata["addedOwnedAgentIds"])
	}
	if removed, ok := updatedEvent.Metadata["removedOwnedAgentIds"].([]string); !ok || len(removed) != 0 {
		t.Fatalf("updated event should record removed agents, got %#v", updatedEvent.Metadata["removedOwnedAgentIds"])
	}

	cleared := decodeData[adminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPatch, "/api/v1/admin-identities/"+created.Identity.ID, map[string]any{
		"ownedAgentIds": []string{},
	}, "", "platform-key"))
	if len(cleared.OwnedAgentIDs) != 0 {
		t.Fatalf("clearing the binding should store an empty set, got %#v", cleared.OwnedAgentIDs)
	}
	if rec := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/"+tenantID+"/access-profile?workspaceId="+workspaceID+"&callerInstanceId="+otherCaller.ID, nil, "", created.Key); rec.Code != http.StatusOK {
		t.Fatalf("cleared binding should restore tenant-scope fallback, got %d body=%s", rec.Code, rec.Body.String())
	}
}

type holderScopeSessionResponse struct {
	Actor          string   `json:"actor"`
	Role           string   `json:"role"`
	RequiresLogin  bool     `json:"requiresLogin"`
	HolderAgentIDs []string `json:"holderAgentIds"`
}

func TestConsoleSessionReportsHolderAgentIds(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})
	tenantID, workspaceID, caller, _, _ := seedTenantPermissionCenterFixture(t, repo)
	createHolderAdminIdentity(t, repo, "adm_session", "session-admin", "session-key", domain.AdminIdentityRoleTenantAdmin, tenantID, workspaceID, []string{caller.ID})
	createHolderAdminIdentity(t, repo, "adm_session_unbound", "session-unbound", "session-unbound-key", domain.AdminIdentityRoleTenantAdmin, tenantID, workspaceID, nil)

	login := func(key string) holderScopeSessionResponse {
		t.Helper()
		return decodeData[holderScopeSessionResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": key}, "", ""))
	}

	bound := login("session-key")
	if bound.Actor != "session-admin" || len(bound.HolderAgentIDs) != 1 || bound.HolderAgentIDs[0] != caller.ID {
		t.Fatalf("bound login should report holder agents, got %#v", bound)
	}
	if unbound := login("session-unbound-key"); len(unbound.HolderAgentIDs) != 0 {
		t.Fatalf("unbound login should omit holder agents, got %#v", unbound)
	}
	if platform := login("platform-key"); len(platform.HolderAgentIDs) != 0 {
		t.Fatalf("platform login should omit holder agents, got %#v", platform)
	}

	demoRouter := newRouterWithRepo(repo)
	demo := decodeData[holderScopeSessionResponse](t, requestWithAdmin(t, demoRouter, http.MethodGet, "/api/v1/auth/session", nil, "", ""))
	if demo.RequiresLogin || demo.Actor != "local-dev" || len(demo.HolderAgentIDs) != 0 {
		t.Fatalf("demo session must stay unchanged, got %#v", demo)
	}
}
