package httpapi_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const taskResultMemoryIDHeaderForTest = "X-AgentHarbor-Task-Result-Memory-Id"

func TestMCPToolCallExplicitlyAttachesVerifiedTaskResultContext(t *testing.T) {
	now := time.Date(2026, time.August, 12, 9, 0, 0, 0, time.UTC)
	repo := store.NewMemory()
	var capturedContext string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContext = r.Header.Get("X-AgentHarbor-Context")
		if got := r.Header.Get(taskResultMemoryIDHeaderForTest); got != "" {
			t.Fatalf("memory selector header must not reach upstream, got %q", got)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)
	fixture.target.ChannelConfig = map[string]any{"endpoint": upstream.URL}
	if _, found, err := repo.UpdateAgent(t.Context(), fixture.target); err != nil || !found {
		t.Fatalf("update runtime target: found=%v err=%v", found, err)
	}
	key := createRuntimeTaskResultAgentKey(t, repo, fixture.caller, now)
	memory := createRuntimeTaskResultMemory(t, router, fixture, now.Add(24*time.Hour))

	response := requestRuntimeTaskResultToolCall(t, router, fixture, key, memory.ID)
	if response.Code != http.StatusAccepted {
		t.Fatalf("runtime call failed: status=%d body=%s", response.Code, response.Body.String())
	}
	decoded, err := base64.RawURLEncoding.DecodeString(capturedContext)
	if err != nil {
		t.Fatalf("decode Agent Harbor context: %v", err)
	}
	var payload struct {
		TaskResultContext *struct {
			Kind       string             `json:"kind"`
			Summary    string             `json:"summary"`
			DataScopes []domain.DataScope `json:"dataScopes"`
		} `json:"taskResultContext"`
	}
	if err := json.Unmarshal(decoded, &payload); err != nil {
		t.Fatalf("unmarshal Agent Harbor context: %v", err)
	}
	if payload.TaskResultContext == nil {
		t.Fatalf("expected explicit task result context, got %s", decoded)
	}
	if payload.TaskResultContext.Kind != "verified_task_result" || payload.TaskResultContext.Summary != memory.Summary || !reflect.DeepEqual(payload.TaskResultContext.DataScopes, fixture.dataScopes) {
		t.Fatalf("unexpected task result context: %#v", payload.TaskResultContext)
	}
	for _, secret := range []string{memory.ID, memory.SourceTraceID, memory.PayloadDigest} {
		if secret != "" && strings.Contains(string(decoded), secret) {
			t.Fatalf("controlled context leaked memory metadata %q: %s", secret, decoded)
		}
	}
	assertTaskResultMemoryAudit(t, repo, "memory_runtime_context_allowed", fixture.tenantID, fixture.workspaceID, "memory_allowed")
}

func TestMCPToolCallRejectsUnavailableTaskResultContextWithoutCallingUpstream(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		currentNow := time.Date(2026, time.August, 12, 9, 0, 0, 0, time.UTC)
		repo := store.NewMemory()
		upstreamCalls := 0
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upstreamCalls++
			w.WriteHeader(http.StatusAccepted)
		}))
		defer upstream.Close()
		router := httpapi.New(
			repo,
			httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
			httpapi.WithClock(func() time.Time { return currentNow }),
		).Router()
		fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", currentNow)
		fixture.target.ChannelConfig = map[string]any{"endpoint": upstream.URL}
		if _, found, err := repo.UpdateAgent(t.Context(), fixture.target); err != nil || !found {
			t.Fatalf("update runtime target: found=%v err=%v", found, err)
		}
		key := createRuntimeTaskResultAgentKey(t, repo, fixture.caller, currentNow)
		memory := createRuntimeTaskResultMemory(t, router, fixture, currentNow.Add(time.Hour))
		currentNow = currentNow.Add(2 * time.Hour)

		response := requestRuntimeTaskResultToolCall(t, router, fixture, key, memory.ID)
		assertRuntimeTaskResultUnavailable(t, response, memory)
		if upstreamCalls != 0 {
			t.Fatalf("expired memory must stop the upstream call, got %d calls", upstreamCalls)
		}
		assertTaskResultMemoryAudit(t, repo, "memory_runtime_context_denied", fixture.tenantID, fixture.workspaceID, "memory_expired")
		assertRuntimeTaskResultDeniedTrace(t, repo, fixture)
	})

	t.Run("cross tenant", func(t *testing.T) {
		now := time.Date(2026, time.August, 12, 9, 0, 0, 0, time.UTC)
		repo := store.NewMemory()
		upstreamCalls := 0
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upstreamCalls++
			w.WriteHeader(http.StatusAccepted)
		}))
		defer upstream.Close()
		router := httpapi.New(
			repo,
			httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
			httpapi.WithClock(func() time.Time { return now }),
		).Router()
		sg := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)
		my := seedVerifiedTaskResultFixture(t, repo, "malaysia", "support-my", "country=MY", now)
		sg.target.ChannelConfig = map[string]any{"endpoint": upstream.URL}
		if _, found, err := repo.UpdateAgent(t.Context(), sg.target); err != nil || !found {
			t.Fatalf("update runtime target: found=%v err=%v", found, err)
		}
		key := createRuntimeTaskResultAgentKey(t, repo, sg.caller, now)
		memory := createRuntimeTaskResultMemory(t, router, my, now.Add(24*time.Hour))

		response := requestRuntimeTaskResultToolCall(t, router, sg, key, memory.ID)
		assertRuntimeTaskResultUnavailable(t, response, memory)
		if upstreamCalls != 0 {
			t.Fatalf("cross-tenant memory must stop the upstream call, got %d calls", upstreamCalls)
		}
		assertTaskResultMemoryAudit(t, repo, "memory_runtime_context_denied", sg.tenantID, sg.workspaceID, "memory_not_available")
	})
}

func TestMCPToolCallNeverUsesMemoryToOverrideBaseAuthorization(t *testing.T) {
	now := time.Date(2026, time.August, 12, 9, 0, 0, 0, time.UTC)
	repo := store.NewMemory()
	upstreamCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		w.WriteHeader(http.StatusAccepted)
	}))
	defer upstream.Close()
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)
	fixture.target.ChannelConfig = map[string]any{"endpoint": upstream.URL}
	if _, found, err := repo.UpdateAgent(t.Context(), fixture.target); err != nil || !found {
		t.Fatalf("update runtime target: found=%v err=%v", found, err)
	}
	key := createRuntimeTaskResultAgentKey(t, repo, fixture.caller, now)
	memory := createRuntimeTaskResultMemory(t, router, fixture, now.Add(24*time.Hour))
	fixture.capability.DiscoveryStatus = domain.CapabilityDiscoveryDeprecated
	fixture.capability.UpdatedAt = now.Add(time.Minute)
	if _, found, err := repo.UpdateCapability(t.Context(), fixture.capability); err != nil || !found {
		t.Fatalf("deprecate runtime capability: found=%v err=%v", found, err)
	}

	response := requestRuntimeTaskResultToolCall(t, router, fixture, key, memory.ID)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"error":"PERMISSION_DENIED"`) {
		t.Fatalf("base authorization should deny first: status=%d body=%s", response.Code, response.Body.String())
	}
	if upstreamCalls != 0 {
		t.Fatalf("denied base authorization must stop the upstream call, got %d calls", upstreamCalls)
	}
	events, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{
		ManagementScope: store.ManagementScope{TenantID: fixture.tenantID, WorkspaceID: fixture.workspaceID},
		ResourceType:    "verified_task_result_summary",
	})
	if err != nil {
		t.Fatalf("list runtime memory audits: %v", err)
	}
	for _, event := range events {
		if strings.HasPrefix(event.Action, "memory_runtime_context_") {
			t.Fatalf("memory resolver ran before base authorization: %#v", event)
		}
	}
}

func createRuntimeTaskResultAgentKey(t *testing.T, repo store.Repository, caller domain.Agent, now time.Time) string {
	t.Helper()
	plaintext, prefix := security.NewAgentKey()
	if _, err := repo.CreateAgentKey(t.Context(), domain.AgentKey{
		ID:        security.NewID("key"),
		AgentID:   caller.ID,
		Name:      "runtime task result context",
		Hash:      security.HashSecret(plaintext),
		Prefix:    prefix,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("create runtime agent key: %v", err)
	}
	return plaintext
}

func createRuntimeTaskResultMemory(t *testing.T, router http.Handler, fixture verifiedTaskResultFixture, expiresAt time.Time) domain.VerifiedTaskResultSummary {
	t.Helper()
	response := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries", verifiedTaskResultSummaryRequest(fixture.trace.ID, expiresAt), "", "platform-key")
	if response.Code != http.StatusCreated {
		t.Fatalf("create runtime memory: status=%d body=%s", response.Code, response.Body.String())
	}
	return decodeData[domain.VerifiedTaskResultSummary](t, response)
}

func requestRuntimeTaskResultToolCall(t *testing.T, router http.Handler, fixture verifiedTaskResultFixture, key string, memoryID string) *httptest.ResponseRecorder {
	t.Helper()
	recorder, request := buildRequest(t, http.MethodPost, "/api/v1/mcp/agents/"+fixture.target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "runtime-memory-call",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      fixture.capability.Key,
			"arguments": map[string]any{"ticketId": "ticket-001"},
		},
	}, key, "runtime-memory-run", "")
	request.Header.Set("X-AgentHarbor-Subject-Id", fixture.subjectID)
	request.Header.Set(taskResultMemoryIDHeaderForTest, memoryID)
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertRuntimeTaskResultUnavailable(t *testing.T, response *httptest.ResponseRecorder, memory domain.VerifiedTaskResultSummary) {
	t.Helper()
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"error":"MEMORY_CONTEXT_NOT_AVAILABLE"`) {
		t.Fatalf("expected generic runtime memory denial: status=%d body=%s", response.Code, response.Body.String())
	}
	for _, secret := range []string{memory.ID, memory.Summary, memory.SourceTraceID, memory.PayloadDigest} {
		if secret != "" && strings.Contains(response.Body.String(), secret) {
			t.Fatalf("runtime memory denial leaked %q: %s", secret, response.Body.String())
		}
	}
}

func assertRuntimeTaskResultDeniedTrace(t *testing.T, repo store.Repository, fixture verifiedTaskResultFixture) {
	t.Helper()
	traces, err := repo.ListTraces(t.Context(), store.TraceFilter{
		ManagementScope: store.ManagementScope{TenantID: fixture.tenantID, WorkspaceID: fixture.workspaceID},
		TargetID:        fixture.target.ID,
		Decision:        domain.TraceDecisionDenied,
		Limit:           20,
	})
	if err != nil {
		t.Fatalf("list runtime memory denial traces: %v", err)
	}
	for _, trace := range traces {
		if trace.CapabilityID == fixture.capability.ID && trace.Reason == "requested task result context is not available" && trace.UpstreamAttempts == 0 {
			return
		}
	}
	t.Fatalf("missing denied runtime trace: %#v", traces)
}
