package httpapi_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestVerifiedTaskResultSummaryAllowsVerifiedInScopeRead(t *testing.T) {
	now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	repo := store.NewMemory()
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)

	preflight := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries:preflight", verifiedTaskResultSummaryRequest(fixture.trace.ID, now.Add(24*time.Hour)), "", "platform-key")
	if preflight.Code != http.StatusOK {
		t.Fatalf("preflight failed: status=%d body=%s", preflight.Code, preflight.Body.String())
	}
	preflightResult := decodeData[domain.TaskResultSummaryGateResult](t, preflight)
	if preflightResult.Decision != domain.TaskResultSummaryGateAllowed || preflightResult.ReasonCode != "memory_allowed" {
		t.Fatalf("unexpected preflight result: %#v", preflightResult)
	}

	createdResponse := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries", verifiedTaskResultSummaryRequest(fixture.trace.ID, now.Add(24*time.Hour)), "", "platform-key")
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create summary failed: status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
	}
	created := decodeData[domain.VerifiedTaskResultSummary](t, createdResponse)
	if created.SourceTraceID != fixture.trace.ID || created.TenantID != fixture.tenantID || created.WorkspaceID != fixture.workspaceID || created.CallerInstanceID != fixture.caller.ID || created.SubjectID != fixture.subjectID {
		t.Fatalf("created summary did not derive the full source scope: %#v", created)
	}
	if created.PayloadDigest == "" || created.Summary != "Verified task result recorded from a successful allowed execution." || created.Verification != domain.TaskResultSummaryVerificationHumanReviewedRedacted || created.VerifiedBy != "platform" {
		t.Fatalf("created summary is missing verification evidence: %#v", created)
	}

	read := requestWithAdmin(t, router, http.MethodGet, verifiedTaskResultSummaryReadPath(created.ID, fixture), nil, "", "platform-key")
	if read.Code != http.StatusOK {
		t.Fatalf("read summary failed: status=%d body=%s", read.Code, read.Body.String())
	}
	readResult := decodeData[domain.TaskResultSummaryReadResponse](t, read)
	if readResult.Decision != domain.TaskResultSummaryGateAllowed || readResult.Memory == nil || readResult.Memory.ID != created.ID || readResult.Memory.Summary != created.Summary {
		t.Fatalf("expected an in-scope verified summary, got %#v", readResult)
	}

	assertTaskResultMemoryAudit(t, repo, "memory_write_preflight", fixture.tenantID, fixture.workspaceID, "memory_allowed")
	assertTaskResultMemoryAudit(t, repo, "memory_written", fixture.tenantID, fixture.workspaceID, "memory_allowed")
	assertTaskResultMemoryAudit(t, repo, "memory_read_allowed", fixture.tenantID, fixture.workspaceID, "memory_allowed")
}

func TestVerifiedTaskResultSummaryRejectsCrossTenantReadWithoutLeaks(t *testing.T) {
	now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	repo := store.NewMemory()
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{
			{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
			{Actor: "sg-admin", Key: "sg-key", Role: "tenant_admin", TenantID: "singapore", WorkspaceID: "support-sg"},
		}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	sg := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)
	my := seedVerifiedTaskResultFixture(t, repo, "malaysia", "support-my", "country=MY", now)

	createdResponse := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries", verifiedTaskResultSummaryRequest(my.trace.ID, now.Add(24*time.Hour)), "", "platform-key")
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create MY summary failed: status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
	}
	myMemory := decodeData[domain.VerifiedTaskResultSummary](t, createdResponse)

	read := requestWithAdmin(t, router, http.MethodGet, verifiedTaskResultSummaryReadPath(myMemory.ID, sg), nil, "", "sg-key")
	if read.Code != http.StatusOK {
		t.Fatalf("cross-tenant read should return a safe gate decision, got status=%d body=%s", read.Code, read.Body.String())
	}
	readResult := decodeData[domain.TaskResultSummaryReadResponse](t, read)
	if readResult.Decision != domain.TaskResultSummaryGateDenied || readResult.ReasonCode != "memory_not_available" || readResult.NextActionCode != "review_memory_scope" || readResult.Memory != nil {
		t.Fatalf("unexpected cross-tenant gate result: %#v", readResult)
	}
	assertResponseDoesNotContain(t, read.Body.String(), myMemory.ID, myMemory.Summary, myMemory.SourceTraceID, myMemory.PayloadDigest, "\"memory\"")
	assertTaskResultMemoryAudit(t, repo, "memory_read_denied", sg.tenantID, sg.workspaceID, "memory_not_available")

	writePreflight := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries:preflight", verifiedTaskResultSummaryRequest(my.trace.ID, now.Add(24*time.Hour)), "", "sg-key")
	if writePreflight.Code != http.StatusOK {
		t.Fatalf("cross-tenant preflight should return a safe gate decision, got status=%d body=%s", writePreflight.Code, writePreflight.Body.String())
	}
	writeResult := decodeData[domain.TaskResultSummaryGateResult](t, writePreflight)
	if writeResult.Decision != domain.TaskResultSummaryGateDenied || writeResult.ReasonCode != "memory_source_stale" || writeResult.NextActionCode != "refresh_memory_source" {
		t.Fatalf("cross-tenant preflight must hide source existence, got %#v", writeResult)
	}
	assertResponseDoesNotContain(t, writePreflight.Body.String(), my.trace.ID, myMemory.ID, myMemory.Summary, myMemory.SourceTraceID, myMemory.PayloadDigest)
}

func TestVerifiedTaskResultSummaryRejectsExpiredReadWithoutRenewing(t *testing.T) {
	now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	currentNow := now
	repo := store.NewMemory()
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
		httpapi.WithClock(func() time.Time { return currentNow }),
	).Router()
	fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)

	createdResponse := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries", verifiedTaskResultSummaryRequest(fixture.trace.ID, now.Add(time.Hour)), "", "platform-key")
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create summary failed: status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
	}
	created := decodeData[domain.VerifiedTaskResultSummary](t, createdResponse)
	currentNow = now.Add(2 * time.Hour)

	read := requestWithAdmin(t, router, http.MethodGet, verifiedTaskResultSummaryReadPath(created.ID, fixture), nil, "", "platform-key")
	if read.Code != http.StatusOK {
		t.Fatalf("expired read should return a safe gate decision, got status=%d body=%s", read.Code, read.Body.String())
	}
	readResult := decodeData[domain.TaskResultSummaryReadResponse](t, read)
	if readResult.Decision != domain.TaskResultSummaryGateDenied || readResult.ReasonCode != "memory_expired" || readResult.NextActionCode != "refresh_memory_source" || readResult.Memory != nil {
		t.Fatalf("unexpected expired gate result: %#v", readResult)
	}
	assertResponseDoesNotContain(t, read.Body.String(), created.ID, created.Summary, created.SourceTraceID, created.PayloadDigest, "\"memory\"")
	stored, found, err := repo.GetVerifiedTaskResultSummary(t.Context(), created.ID, store.VerifiedTaskResultSummaryScope{
		TenantID:         fixture.tenantID,
		WorkspaceID:      fixture.workspaceID,
		CallerInstanceID: fixture.caller.ID,
		SubjectID:        fixture.subjectID,
	})
	if err != nil || !found {
		t.Fatalf("get stored expired summary: found=%v err=%v", found, err)
	}
	if !stored.ExpiresAt.Equal(created.ExpiresAt) {
		t.Fatalf("expired read must not renew or rewrite expiry: got=%s want=%s", stored.ExpiresAt, created.ExpiresAt)
	}
	assertTaskResultMemoryAudit(t, repo, "memory_read_denied", fixture.tenantID, fixture.workspaceID, "memory_expired")
}

func TestVerifiedTaskResultSummaryRechecksLiveAuthorization(t *testing.T) {
	now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	repo := store.NewMemory()
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)
	createdResponse := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries", verifiedTaskResultSummaryRequest(fixture.trace.ID, now.Add(24*time.Hour)), "", "platform-key")
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create summary failed: status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
	}
	created := decodeData[domain.VerifiedTaskResultSummary](t, createdResponse)
	capability := fixture.capability
	capability.DiscoveryStatus = domain.CapabilityDiscoveryDeprecated
	capability.UpdatedAt = now.Add(time.Minute)
	if _, ok, err := repo.UpdateCapability(t.Context(), capability); err != nil || !ok {
		t.Fatalf("deprecate source capability: ok=%v err=%v", ok, err)
	}

	read := requestWithAdmin(t, router, http.MethodGet, verifiedTaskResultSummaryReadPath(created.ID, fixture), nil, "", "platform-key")
	if read.Code != http.StatusOK {
		t.Fatalf("stale authorization read should return a safe gate decision, got status=%d body=%s", read.Code, read.Body.String())
	}
	readResult := decodeData[domain.TaskResultSummaryReadResponse](t, read)
	if readResult.Decision != domain.TaskResultSummaryGateDenied || readResult.ReasonCode != "memory_source_stale" || readResult.Memory != nil {
		t.Fatalf("read must recheck live authorization instead of trusting memory: %#v", readResult)
	}
	assertResponseDoesNotContain(t, read.Body.String(), created.ID, created.Summary, created.SourceTraceID, created.PayloadDigest, "\"memory\"")
	assertTaskResultMemoryAudit(t, repo, "memory_read_denied", fixture.tenantID, fixture.workspaceID, "memory_source_stale")
}

func TestVerifiedTaskResultSummaryRechecksLiveAgentStatus(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		disabledID func(verifiedTaskResultFixture) string
	}{
		{name: "caller", disabledID: func(fixture verifiedTaskResultFixture) string { return fixture.caller.ID }},
		{name: "target", disabledID: func(fixture verifiedTaskResultFixture) string { return fixture.target.ID }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
			repo := store.NewMemory()
			router := httpapi.New(
				repo,
				httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
				httpapi.WithClock(func() time.Time { return now }),
			).Router()
			fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)
			createdResponse := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries", verifiedTaskResultSummaryRequest(fixture.trace.ID, now.Add(24*time.Hour)), "", "platform-key")
			if createdResponse.Code != http.StatusCreated {
				t.Fatalf("create summary failed: status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
			}
			created := decodeData[domain.VerifiedTaskResultSummary](t, createdResponse)
			if _, ok, err := repo.DisableAgent(t.Context(), testCase.disabledID(fixture), now.Add(time.Minute)); err != nil || !ok {
				t.Fatalf("disable %s agent: ok=%v err=%v", testCase.name, ok, err)
			}

			read := requestWithAdmin(t, router, http.MethodGet, verifiedTaskResultSummaryReadPath(created.ID, fixture), nil, "", "platform-key")
			if read.Code != http.StatusOK {
				t.Fatalf("disabled %s read should return a safe gate decision, got status=%d body=%s", testCase.name, read.Code, read.Body.String())
			}
			readResult := decodeData[domain.TaskResultSummaryReadResponse](t, read)
			if readResult.Decision != domain.TaskResultSummaryGateDenied || readResult.ReasonCode != "memory_source_stale" || readResult.Memory != nil {
				t.Fatalf("read must recheck %s active status: %#v", testCase.name, readResult)
			}
			assertResponseDoesNotContain(t, read.Body.String(), created.ID, created.Summary, created.SourceTraceID, created.PayloadDigest, "\"memory\"")
		})
	}
}

func TestVerifiedTaskResultSummaryRequiresApprovalForHighRiskWrite(t *testing.T) {
	now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	repo := store.NewMemory()
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)
	highRiskCapability := fixture.capability
	highRiskCapability.Action = domain.CapabilityActionExport
	highRiskCapability.RiskLevel = domain.CapabilityRiskHigh
	highRiskCapability.UpdatedAt = now.Add(time.Minute)
	highRiskCapability, ok, err := repo.UpdateCapability(t.Context(), highRiskCapability)
	if err != nil || !ok {
		t.Fatalf("mark source capability high risk: ok=%v err=%v", ok, err)
	}
	highRiskTrace := fixture.trace
	highRiskTrace.ID = security.NewID("trc")
	highRiskTrace.CapabilityVersion = highRiskCapability.Version
	highRiskTrace.CapabilityFingerprint = domain.CapabilityFingerprint(highRiskCapability)
	highRiskTrace.CreatedAt = now
	if _, err := repo.AppendTrace(t.Context(), highRiskTrace); err != nil {
		t.Fatalf("append high-risk trace: %v", err)
	}

	preflight := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries:preflight", verifiedTaskResultSummaryRequest(highRiskTrace.ID, now.Add(24*time.Hour)), "", "platform-key")
	if preflight.Code != http.StatusOK {
		t.Fatalf("high-risk preflight failed: status=%d body=%s", preflight.Code, preflight.Body.String())
	}
	result := decodeData[domain.TaskResultSummaryGateResult](t, preflight)
	if result.Decision != domain.TaskResultSummaryGateApprovalRequired || result.ReasonCode != "memory_approval_required" || result.NextActionCode != "request_memory_approval" {
		t.Fatalf("high-risk write should require approval, got %#v", result)
	}
	assertResponseDoesNotContain(t, preflight.Body.String(), fixture.trace.ID, "sourceTraceId", "payloadDigest", "memoryId")
	directWrite := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries", verifiedTaskResultSummaryRequest(highRiskTrace.ID, now.Add(24*time.Hour)), "", "platform-key")
	if directWrite.Code != http.StatusOK {
		t.Fatalf("high-risk direct write should return a safe gate decision, got status=%d body=%s", directWrite.Code, directWrite.Body.String())
	}
	directResult := decodeData[domain.TaskResultSummaryGateResult](t, directWrite)
	if directResult.Decision != domain.TaskResultSummaryGateApprovalRequired || directResult.ReasonCode != "memory_approval_required" {
		t.Fatalf("high-risk direct write should require approval, got %#v", directResult)
	}
	stored, err := repo.ListVerifiedTaskResultSummaries(t.Context(), verifiedTaskResultSummaryStoreFilter(fixture))
	if err != nil || len(stored) != 0 {
		t.Fatalf("high-risk preflight must not persist a summary: rows=%#v err=%v", stored, err)
	}
	assertTaskResultMemoryAudit(t, repo, "memory_write_denied", fixture.tenantID, fixture.workspaceID, "memory_approval_required")
}

func TestVerifiedTaskResultSummaryRejectsChangedCapabilityFingerprint(t *testing.T) {
	now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	repo := store.NewMemory()
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)
	changed := fixture.capability
	changed.RiskLevel = domain.CapabilityRiskHigh
	changed.DataScopes = []domain.DataScope{{DataDomain: "support", Dataset: "tickets", TenantFilter: "country=SG", Field: "title"}}
	changed.UpdatedAt = now.Add(time.Minute)
	if _, ok, err := repo.UpdateCapability(t.Context(), changed); err != nil || !ok {
		t.Fatalf("change source capability policy: ok=%v err=%v", ok, err)
	}

	response := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries:preflight", verifiedTaskResultSummaryRequest(fixture.trace.ID, now.Add(24*time.Hour)), "", "platform-key")
	if response.Code != http.StatusOK {
		t.Fatalf("changed-capability preflight should return a safe gate decision, got status=%d body=%s", response.Code, response.Body.String())
	}
	result := decodeData[domain.TaskResultSummaryGateResult](t, response)
	if result.Decision != domain.TaskResultSummaryGateDenied || result.ReasonCode != "memory_source_stale" || result.NextActionCode != "refresh_memory_source" {
		t.Fatalf("changed capability fingerprint must invalidate the old trace, got %#v", result)
	}
	assertResponseDoesNotContain(t, response.Body.String(), fixture.trace.ID, "sourceTraceId", "payloadDigest", "memoryId")
	assertTaskResultMemoryAudit(t, repo, "memory_write_denied", fixture.tenantID, fixture.workspaceID, "memory_source_stale")
}

func TestVerifiedTaskResultSummaryBoundsExpiryToSourceTrace(t *testing.T) {
	now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	repo := store.NewMemory()
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)
	oldTrace := fixture.trace
	oldTrace.ID = security.NewID("trc")
	oldTrace.CreatedAt = now.Add(-30 * 24 * time.Hour)
	if _, err := repo.AppendTrace(t.Context(), oldTrace); err != nil {
		t.Fatalf("append old source trace: %v", err)
	}

	response := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries:preflight", verifiedTaskResultSummaryRequest(oldTrace.ID, now.Add(time.Hour)), "", "platform-key")
	if response.Code != http.StatusOK {
		t.Fatalf("old-source preflight should return a safe gate decision, got status=%d body=%s", response.Code, response.Body.String())
	}
	result := decodeData[domain.TaskResultSummaryGateResult](t, response)
	if result.Decision != domain.TaskResultSummaryGateDenied || result.ReasonCode != "memory_expiry_invalid" || result.NextActionCode != "set_memory_expiry" {
		t.Fatalf("expiry must be bounded by source trace time, got %#v", result)
	}
	assertResponseDoesNotContain(t, response.Body.String(), oldTrace.ID, "sourceTraceId", "payloadDigest", "memoryId")
	assertTaskResultMemoryAudit(t, repo, "memory_write_denied", fixture.tenantID, fixture.workspaceID, "memory_expiry_invalid")
}

func TestVerifiedTaskResultSummaryAuditsUnsupportedKindWithoutRawInput(t *testing.T) {
	now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	repo := store.NewMemory()
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)
	const unsupportedKind = "authorization_decision"
	response := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries", map[string]any{
		"memoryKind":    unsupportedKind,
		"sourceTraceId": fixture.trace.ID,
		"verification":  string(domain.TaskResultSummaryVerificationHumanReviewedRedacted),
		"expiresAt":     now.Add(24 * time.Hour).Format(time.RFC3339Nano),
	}, "", "platform-key")
	if response.Code != http.StatusOK {
		t.Fatalf("unsupported kind should return a safe gate decision, got status=%d body=%s", response.Code, response.Body.String())
	}
	result := decodeData[domain.TaskResultSummaryGateResult](t, response)
	if result.Decision != domain.TaskResultSummaryGateApprovalRequired || result.ReasonCode != "memory_approval_required" {
		t.Fatalf("unsupported kind should require approval, got %#v", result)
	}
	assertResponseDoesNotContain(t, response.Body.String(), unsupportedKind, fixture.trace.ID)
	events, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{Action: "memory_write_denied", ResourceType: "verified_task_result_summary"})
	if err != nil {
		t.Fatalf("list unsupported-kind audit: %v", err)
	}
	for _, event := range events {
		if event.Metadata["reasonCode"] == "memory_approval_required" {
			if event.Metadata["memoryKind"] != "unsupported" || containsAuditString(event.Metadata, unsupportedKind) {
				t.Fatalf("unsupported candidate kind must be normalized in audit: %#v", event.Metadata)
			}
			return
		}
	}
	t.Fatalf("missing unsupported-kind audit: %#v", events)
}

func TestVerifiedTaskResultSummaryRejectsClientSuppliedFreeText(t *testing.T) {
	now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	repo := store.NewMemory()
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)

	response := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries", map[string]any{
		"sourceTraceId": fixture.trace.ID,
		"summary":       "The subject can export contracts.",
		"verification":  string(domain.TaskResultSummaryVerificationHumanReviewedRedacted),
		"expiresAt":     now.Add(24 * time.Hour).Format(time.RFC3339Nano),
	}, "", "platform-key")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("client-supplied free text must be rejected, got status=%d body=%s", response.Code, response.Body.String())
	}
	assertResponseDoesNotContain(t, response.Body.String(), "export contracts", fixture.trace.ID)
	stored, err := repo.ListVerifiedTaskResultSummaries(t.Context(), verifiedTaskResultSummaryStoreFilter(fixture))
	if err != nil || len(stored) != 0 {
		t.Fatalf("client-supplied free text must not persist a summary: rows=%#v err=%v", stored, err)
	}
}

func TestVerifiedTaskResultSummaryRejectsDuplicateSourceWithoutLeaks(t *testing.T) {
	now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	repo := store.NewMemory()
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "platform", Key: "platform-key", Role: "platform_admin"}}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	fixture := seedVerifiedTaskResultFixture(t, repo, "singapore", "support-sg", "country=SG", now)
	requestBody := verifiedTaskResultSummaryRequest(fixture.trace.ID, now.Add(24*time.Hour))
	createdResponse := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries", requestBody, "", "platform-key")
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create initial summary: status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
	}
	created := decodeData[domain.VerifiedTaskResultSummary](t, createdResponse)

	preflight := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries:preflight", requestBody, "", "platform-key")
	if preflight.Code != http.StatusOK {
		t.Fatalf("duplicate preflight should return a safe gate decision, got status=%d body=%s", preflight.Code, preflight.Body.String())
	}
	preflightResult := decodeData[domain.TaskResultSummaryGateResult](t, preflight)
	if preflightResult.Decision != domain.TaskResultSummaryGateDenied || preflightResult.ReasonCode != "memory_source_already_recorded" || preflightResult.NextActionCode != "review_existing_memory" {
		t.Fatalf("unexpected duplicate preflight result: %#v", preflightResult)
	}
	assertResponseDoesNotContain(t, preflight.Body.String(), created.ID, created.SourceTraceID, created.PayloadDigest, "\"memory\"")

	directWrite := requestWithAdmin(t, router, http.MethodPost, "/api/v1/task-result-summaries", requestBody, "", "platform-key")
	if directWrite.Code != http.StatusOK {
		t.Fatalf("duplicate write should return a safe gate decision, got status=%d body=%s", directWrite.Code, directWrite.Body.String())
	}
	directResult := decodeData[domain.TaskResultSummaryGateResult](t, directWrite)
	if directResult.Decision != domain.TaskResultSummaryGateDenied || directResult.ReasonCode != "memory_source_already_recorded" {
		t.Fatalf("unexpected duplicate direct write result: %#v", directResult)
	}
	storeFilter := verifiedTaskResultSummaryStoreFilter(fixture)
	storeFilter.SourceTraceID = fixture.trace.ID
	stored, err := repo.ListVerifiedTaskResultSummaries(t.Context(), storeFilter)
	if err != nil || len(stored) != 1 || stored[0].ID != created.ID {
		t.Fatalf("duplicate source must not create another summary: rows=%#v err=%v", stored, err)
	}
	assertTaskResultMemoryAudit(t, repo, "memory_write_denied", fixture.tenantID, fixture.workspaceID, "memory_source_already_recorded")
}

type verifiedTaskResultFixture struct {
	tenantID    string
	workspaceID string
	caller      domain.Agent
	target      domain.Agent
	capability  domain.Capability
	subjectID   string
	trace       domain.TraceEvent
	dataScopes  []domain.DataScope
}

func seedVerifiedTaskResultFixture(t *testing.T, repo store.Repository, tenantID string, workspaceID string, tenantFilter string, now time.Time) verifiedTaskResultFixture {
	t.Helper()
	createDirectTenant(t, repo, tenantID, "", tenantID, now)
	caller := createDirectAgent(t, repo, "Task result caller "+tenantID, tenantID, workspaceID, "local", domain.AgentStatusActive, nil)
	target := createDirectAgent(t, repo, "Task result target "+tenantID, tenantID, workspaceID, "mcp", domain.AgentStatusActive, nil)
	dataScopes := []domain.DataScope{{DataDomain: "support", Dataset: "tickets", TenantFilter: tenantFilter}}
	capability := createDirectCapability(t, repo, target.ID, "lookup_ticket_"+tenantID, dataScopes, now)
	entitlement := createDirectTenantEntitlement(t, repo, tenantID, target.ID, capability.ID, dataScopes, now)
	workspaceAssignment := createDirectWorkspaceAssignment(t, repo, entitlement.ID, tenantID, workspaceID, dataScopes, now)
	instanceAssignment := createDirectInstanceAssignment(t, repo, workspaceAssignment.ID, tenantID, workspaceID, caller.ID, dataScopes, now)
	subjectID := "user:support-001"
	trace := domain.TraceEvent{
		ID:                    security.NewID("trc"),
		RunID:                 "verified-task-result-" + tenantID,
		CallerID:              caller.ID,
		TargetID:              target.ID,
		RouteType:             "mcp",
		RouteKey:              "tools/call",
		TenantID:              tenantID,
		WorkspaceID:           workspaceID,
		CallerInstanceID:      caller.ID,
		SubjectID:             subjectID,
		CapabilityID:          capability.ID,
		CapabilityVersion:     capability.Version,
		CapabilityFingerprint: domain.CapabilityFingerprint(capability),
		EntitlementID:         entitlement.ID,
		WorkspaceAssignmentID: workspaceAssignment.ID,
		InstanceAssignmentID:  instanceAssignment.ID,
		DataScopes:            dataScopes,
		Decision:              domain.TraceDecisionAllowed,
		UpstreamAttempts:      1,
		UpstreamStatus:        http.StatusOK,
		CreatedAt:             now.Add(-time.Minute),
	}
	if _, err := repo.AppendTrace(t.Context(), trace); err != nil {
		t.Fatalf("append fixture trace: %v", err)
	}
	return verifiedTaskResultFixture{
		tenantID:    tenantID,
		workspaceID: workspaceID,
		caller:      caller,
		target:      target,
		capability:  capability,
		subjectID:   subjectID,
		trace:       trace,
		dataScopes:  dataScopes,
	}
}

func verifiedTaskResultSummaryRequest(sourceTraceID string, expiresAt time.Time) map[string]any {
	return map[string]any{
		"sourceTraceId": sourceTraceID,
		"verification":  string(domain.TaskResultSummaryVerificationHumanReviewedRedacted),
		"expiresAt":     expiresAt.Format(time.RFC3339Nano),
	}
}

func verifiedTaskResultSummaryReadPath(memoryID string, fixture verifiedTaskResultFixture) string {
	query := url.Values{}
	query.Set("tenantId", fixture.tenantID)
	query.Set("workspaceId", fixture.workspaceID)
	query.Set("callerInstanceId", fixture.caller.ID)
	query.Set("subjectId", fixture.subjectID)
	return "/api/v1/task-result-summaries/" + url.PathEscape(memoryID) + "?" + query.Encode()
}

func verifiedTaskResultSummaryStoreFilter(fixture verifiedTaskResultFixture) store.VerifiedTaskResultSummaryFilter {
	return store.VerifiedTaskResultSummaryFilter{VerifiedTaskResultSummaryScope: store.VerifiedTaskResultSummaryScope{
		TenantID:         fixture.tenantID,
		WorkspaceID:      fixture.workspaceID,
		CallerInstanceID: fixture.caller.ID,
		SubjectID:        fixture.subjectID,
	}}
}

func assertTaskResultMemoryAudit(t *testing.T, repo store.Repository, action string, tenantID string, workspaceID string, reasonCode string) {
	t.Helper()
	events, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{
		ManagementScope: store.ManagementScope{TenantID: tenantID, WorkspaceID: workspaceID},
		Action:          action,
		ResourceType:    "verified_task_result_summary",
	})
	if err != nil {
		t.Fatalf("list task-result memory audit events: %v", err)
	}
	for _, event := range events {
		if got, _ := event.Metadata["reasonCode"].(string); got == reasonCode {
			if event.Metadata["summary"] != nil || event.Metadata["payload"] != nil {
				t.Fatalf("memory audit must not retain summary or payload: %#v", event.Metadata)
			}
			return
		}
	}
	t.Fatalf("missing audit action=%q reasonCode=%q events=%#v", action, reasonCode, events)
}

func containsAuditString(metadata map[string]any, value string) bool {
	for _, candidate := range metadata {
		if text, ok := candidate.(string); ok && text == value {
			return true
		}
	}
	return false
}
