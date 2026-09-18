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

type selfProfileResponse struct {
	Caller struct {
		InstanceID  string `json:"instanceId"`
		Name        string `json:"name"`
		TenantID    string `json:"tenantId"`
		WorkspaceID string `json:"workspaceId"`
		Status      string `json:"status"`
	} `json:"caller"`
	SubjectID string `json:"subjectId"`
	Key       struct {
		Kind      string    `json:"kind"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
		ExpiresAt time.Time `json:"expiresAt"`
	} `json:"key"`
	Handoff *struct {
		HandoffID            string   `json:"handoffId"`
		ApplicationID        string   `json:"applicationId"`
		TemplateID           string   `json:"templateId"`
		TargetID             string   `json:"targetId"`
		SubjectSelector      string   `json:"subjectSelector"`
		AllowedCapabilityIDs []string `json:"allowedCapabilityIds"`
	} `json:"handoff"`
	Targets []struct {
		TargetID     string `json:"targetId"`
		TargetName   string `json:"targetName"`
		ChannelType  string `json:"channelType"`
		Capabilities []struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		} `json:"capabilities"`
	} `json:"targets"`
	GeneratedAt time.Time `json:"generatedAt"`
}

func TestSelfAccessProfileShowsCallerBoundary(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := createDirectAgent(t, repo, "Support Assistant", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	// The governed target is owned by the parent tenant while the entitlement
	// chain lives on the caller's tenant — the ancestor-owned target shape the
	// self profile must resolve through entitlements, not target ownership.
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, map[string]any{"endpoint": "http://127.0.0.1:1/mcp"})
	searchCustomer := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	updateTicket := createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	createDirectAgent(t, repo, "Finance MCP", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, map[string]any{"endpoint": "http://127.0.0.1:1/mcp"})

	input := map[string]any{
		"callerInstanceId":      caller.ID,
		"region":                "us-east",
		"requestText":           "Allow only ticket updates for this tenant.",
		"requestedCapabilityId": updateTicket.ID,
		"subjectSelector":       "user:support-*",
		"targetId":              target.ID,
		"templateId":            "support-ticket-triage",
		"tenantId":              "tenant-east",
		"workspaceId":           "ws-support",
	}
	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, "POST", "/api/v1/permission-packages/approval-requests", input, ""))
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, "POST", "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{"reviewer": "security"}, ""))
	applyInput := map[string]any{
		"approvalRequestId":     approved.ID,
		"callerInstanceId":      caller.ID,
		"region":                input["region"],
		"requestText":           input["requestText"],
		"requestedCapabilityId": input["requestedCapabilityId"],
		"subjectSelector":       input["subjectSelector"],
		"targetId":              target.ID,
		"templateId":            input["templateId"],
		"tenantId":              input["tenantId"],
		"workspaceId":           input["workspaceId"],
	}
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, "POST", "/api/v1/permission-packages:apply", applyInput, ""))
	if applied.Application == nil || applied.Application.ID == "" {
		t.Fatalf("expected applied permission package application, got %#v", applied)
	}
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionDenied, caller, target, searchCustomer, "search_customer", "user:support-001", now.Add(time.Minute))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, updateTicket, "update_ticket", "user:support-001", now.Add(2*time.Minute))
	ready := decodeData[accessHandoffResponse](t, request(t, router, "GET", permissionPackageAccessHandoffPath(input, "", "user:support-001"), nil, ""))
	if ready.Status != "ready" || ready.TokenEligibility.Eligible != true {
		t.Fatalf("expected ready handoff before self profile checks, got %#v", ready)
	}
	created := decodeData[accessHandoffTokenCreateResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/access-handoff/tokens", accessHandoffTokenRequest(input, ready.ID, "user:support-001"), ""))
	if created.Key == "" {
		t.Fatalf("expected access handoff token, got %#v", created)
	}
	regularKey := createDirectTestAgentKey(t, repo, caller.ID, now)

	handoffProfile := decodeData[selfProfileResponse](t, requestWithRunIDAndSubject(t, router, http.MethodGet, "/api/v1/self/access-profile", nil, created.Key, "", "user:support-001"))
	if handoffProfile.Key.Kind != "access_handoff" || handoffProfile.SubjectID != "user:support-001" {
		t.Fatalf("expected access handoff key kind and subject echo, got %#v", handoffProfile)
	}
	if handoffProfile.Handoff == nil || handoffProfile.Handoff.ApplicationID != applied.Application.ID ||
		handoffProfile.Handoff.TargetID != target.ID || handoffProfile.Handoff.SubjectSelector != "user:support-*" {
		t.Fatalf("expected handoff binding block, got %#v", handoffProfile.Handoff)
	}
	if len(handoffProfile.Handoff.AllowedCapabilityIDs) != 1 || handoffProfile.Handoff.AllowedCapabilityIDs[0] != updateTicket.ID {
		t.Fatalf("expected handoff block to list exactly the requested capability, got %#v", handoffProfile.Handoff.AllowedCapabilityIDs)
	}
	if handoffProfile.Key.ExpiresAt.IsZero() || !handoffProfile.Key.ExpiresAt.After(now) {
		t.Fatalf("expected key expiry to be visible for renewal planning, got %#v", handoffProfile.Key)
	}
	if len(handoffProfile.Targets) != 1 || handoffProfile.Targets[0].TargetID != target.ID {
		t.Fatalf("expected only the bound target in handoff profile, got %#v", handoffProfile.Targets)
	}
	if len(handoffProfile.Targets[0].Capabilities) != 1 ||
		handoffProfile.Targets[0].Capabilities[0].ID != updateTicket.ID ||
		handoffProfile.Targets[0].Capabilities[0].Key != "update_ticket" {
		t.Fatalf("expected exactly the allowed capability in handoff profile, got %#v", handoffProfile.Targets[0].Capabilities)
	}

	wrongSubject := requestWithRunIDAndSubject(t, router, http.MethodGet, "/api/v1/self/access-profile", nil, created.Key, "", "user:sales-001")
	if wrongSubject.Code != http.StatusForbidden || !strings.Contains(wrongSubject.Body.String(), "access handoff token does not allow this subject") {
		t.Fatalf("expected subject selector to stay fail-closed, got status=%d body=%s", wrongSubject.Code, wrongSubject.Body.String())
	}
	missingSubject := request(t, router, http.MethodGet, "/api/v1/self/access-profile", nil, created.Key)
	if missingSubject.Code != http.StatusForbidden || !strings.Contains(missingSubject.Body.String(), "access handoff token does not allow this subject") {
		t.Fatalf("expected missing subject to stay fail-closed, got status=%d body=%s", missingSubject.Code, missingSubject.Body.String())
	}

	regularProfile := decodeData[selfProfileResponse](t, requestWithRunIDAndSubject(t, router, http.MethodGet, "/api/v1/self/access-profile", nil, regularKey, "", "user:support-001"))
	if regularProfile.Key.Kind != "agent" || regularProfile.Handoff != nil {
		t.Fatalf("expected ordinary key profile without handoff block, got %#v", regularProfile)
	}
	if len(regularProfile.Targets) != 1 || regularProfile.Targets[0].TargetID != target.ID {
		t.Fatalf("expected the ancestor-owned governed target via entitlement enumeration (decoy workspace target must stay invisible), got %#v", regularProfile.Targets)
	}
	if len(regularProfile.Targets[0].Capabilities) != 1 || regularProfile.Targets[0].Capabilities[0].Key != "update_ticket" {
		t.Fatalf("expected the granted capability only for ordinary key, got %#v", regularProfile.Targets[0].Capabilities)
	}
}

func TestAgentKeyAuthenticationErrorsAreDiagnosable(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	agent := createDirectAgent(t, repo, "Diagnostics Assistant", "tenant-root", "default", "local", domain.AgentStatusActive, nil)

	validKey := createDirectTestAgentKey(t, repo, agent.ID, now)
	if recorder := request(t, router, http.MethodGet, "/api/v1/self/access-profile", nil, validKey); recorder.Code != http.StatusOK {
		t.Fatalf("expected valid key to reach the self profile, got status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	revokedPlaintext, revokedPrefix := security.NewAgentKey()
	revoked, err := repo.CreateAgentKey(t.Context(), domain.AgentKey{
		ID:        security.NewID("key"),
		AgentID:   agent.ID,
		Name:      "revoked-test-key",
		Hash:      security.HashSecret(revokedPlaintext),
		Prefix:    revokedPrefix,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create revoked key: %v", err)
	}
	if _, ok, err := repo.RevokeAgentKey(t.Context(), revoked.ID, now.Add(time.Minute)); err != nil || !ok {
		t.Fatalf("revoke key: ok=%v err=%v", ok, err)
	}
	assertUnauthorizedReason(t, router, revokedPlaintext, "bearer token has been revoked")

	expiredPlaintext, expiredPrefix := security.NewAgentKey()
	if _, err := repo.CreateAgentKey(t.Context(), domain.AgentKey{
		ID:        security.NewID("key"),
		AgentID:   agent.ID,
		Name:      "expired-test-key",
		Hash:      security.HashSecret(expiredPlaintext),
		Prefix:    expiredPrefix,
		CreatedAt: now.Add(-time.Hour),
		ExpiresAt: now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("create expired key: %v", err)
	}
	assertUnauthorizedReason(t, router, expiredPlaintext, "bearer token has expired")

	assertUnauthorizedReason(t, router, "ah-nonexistent-token", "invalid or expired bearer token")

	missing := request(t, router, http.MethodGet, "/api/v1/self/access-profile", nil, "")
	if missing.Code != http.StatusUnauthorized || !strings.Contains(missing.Body.String(), "missing bearer token") {
		t.Fatalf("expected missing bearer rejection, got status=%d body=%s", missing.Code, missing.Body.String())
	}
}

func assertUnauthorizedReason(t *testing.T, router http.Handler, bearer string, reason string) {
	t.Helper()
	recorder := request(t, router, http.MethodGet, "/api/v1/self/access-profile", nil, bearer)
	if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), reason) {
		t.Fatalf("expected 401 with reason %q, got status=%d body=%s", reason, recorder.Code, recorder.Body.String())
	}
}
