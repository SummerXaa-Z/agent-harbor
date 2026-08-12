package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const (
	verifiedTaskResultMemoryKind       = "verified_task_result"
	verifiedTaskResultMemoryPolicyV1   = 1
	maxTaskResultSummaryRetention      = 30 * 24 * time.Hour
	defaultTaskResultSummaryNextAction = "review_memory_scope"
)

type verifiedTaskResultWriteGate struct {
	result              domain.TaskResultSummaryGateResult
	source              domain.TraceEvent
	dataScopes          []domain.DataScope
	summary             string
	candidateMemoryKind string
	auditScope          store.ManagementScope
}

type verifiedTaskResultSourceAccess struct {
	capability domain.Capability
	dataScopes []domain.DataScope
}

func (s *Server) preflightVerifiedTaskResultSummary(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateVerifiedTaskResultSummaryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	gate, err := s.evaluateVerifiedTaskResultSummaryWrite(r.Context(), r, req)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.appendVerifiedTaskResultDecisionAudit(r, gate.auditScope, taskResultSummaryWritePreflightAuditAction(gate.result), "", gate.candidateMemoryKind, gate.result, nil); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gate.result)
}

func (s *Server) createVerifiedTaskResultSummary(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateVerifiedTaskResultSummaryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	gate, err := s.evaluateVerifiedTaskResultSummaryWrite(r.Context(), r, req)
	if err != nil {
		writeError(w, err)
		return
	}
	if gate.result.Decision != domain.TaskResultSummaryGateAllowed {
		if err := s.appendVerifiedTaskResultDecisionAudit(r, gate.auditScope, "memory_write_denied", "", gate.candidateMemoryKind, gate.result, nil); err != nil {
			writeError(w, err)
			return
		}
		// A denied memory gate is an expected policy result, not an HTTP
		// transport failure. Returning the compact result prevents the caller
		// from learning anything about a rejected source or summary.
		writeJSON(w, http.StatusOK, gate.result)
		return
	}

	now := s.now()
	memory := domain.VerifiedTaskResultSummary{
		ID:               security.NewID("mem"),
		TenantID:         gate.source.TenantID,
		WorkspaceID:      gate.source.WorkspaceID,
		CallerInstanceID: gate.source.CallerInstanceID,
		SubjectID:        gate.source.SubjectID,
		TargetID:         gate.source.TargetID,
		CapabilityID:     gate.source.CapabilityID,
		SourceTraceID:    gate.source.ID,
		DataScopes:       domain.CloneDataScopes(gate.dataScopes),
		Summary:          gate.summary,
		PayloadDigest:    verifiedTaskResultSummaryDigest(gate.summary),
		Verification:     domain.TaskResultSummaryVerificationHumanReviewedRedacted,
		VerifiedBy:       managementActor(r),
		VerifiedAt:       now,
		CreatedAt:        now,
		ExpiresAt:        req.ExpiresAt.UTC(),
	}
	created, err := s.repo.CreateVerifiedTaskResultSummaryWithAudit(r.Context(), memory, func(created domain.VerifiedTaskResultSummary) domain.AuditEvent {
		return s.verifiedTaskResultDecisionAuditEvent(r, gate.auditScope, "memory_written", created.ID, gate.candidateMemoryKind, taskResultGateAllowed(), map[string]any{
			"sourceTraceId":   created.SourceTraceID,
			"sourceCreatedAt": gate.source.CreatedAt,
			"payloadDigest":   created.PayloadDigest,
			"expiresAt":       created.ExpiresAt,
			"verification":    created.Verification,
		})
	})
	if errors.Is(err, store.ErrVerifiedTaskResultSummarySourceAlreadyUsed) {
		result := taskResultGateDenied("memory_source_already_recorded", "review_existing_memory")
		if auditErr := s.appendVerifiedTaskResultDecisionAudit(r, gate.auditScope, "memory_write_denied", "", gate.candidateMemoryKind, result, nil); auditErr != nil {
			writeError(w, auditErr)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if errors.Is(err, store.ErrVerifiedTaskResultSummaryAlreadyExists) {
		result := taskResultGateDenied("memory_write_conflict", "retry_memory_write")
		if auditErr := s.appendVerifiedTaskResultDecisionAudit(r, gate.auditScope, "memory_write_denied", "", gate.candidateMemoryKind, result, nil); auditErr != nil {
			writeError(w, auditErr)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if errors.Is(err, store.ErrVerifiedTaskResultSummarySourceNotFound) {
		result := taskResultGateDenied("memory_source_stale", "refresh_memory_source")
		if auditErr := s.appendVerifiedTaskResultDecisionAudit(r, gate.auditScope, "memory_write_denied", "", gate.candidateMemoryKind, result, nil); auditErr != nil {
			writeError(w, auditErr)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) getVerifiedTaskResultSummary(w http.ResponseWriter, r *http.Request) {
	readScope, err := s.verifiedTaskResultReadScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	memory, found, err := s.repo.GetVerifiedTaskResultSummary(r.Context(), chi.URLParam(r, "id"), store.VerifiedTaskResultSummaryScope{
		TenantID:         readScope.TenantID,
		WorkspaceID:      readScope.WorkspaceID,
		CallerInstanceID: readScope.CallerInstanceID,
		SubjectID:        readScope.SubjectID,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !found {
		result := taskResultGateDenied("memory_not_available", defaultTaskResultSummaryNextAction)
		if err := s.appendVerifiedTaskResultDecisionAudit(r, readScope.ManagementScope, "memory_read_denied", "", verifiedTaskResultMemoryKind, result, nil); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, domain.TaskResultSummaryReadResponse{TaskResultSummaryGateResult: result})
		return
	}

	result := s.evaluateVerifiedTaskResultSummaryRead(r.Context(), &memory, readScope, s.now())
	resourceID := ""
	if result.Decision == domain.TaskResultSummaryGateAllowed {
		resourceID = memory.ID
	}
	if err := s.appendVerifiedTaskResultDecisionAudit(r, readScope.ManagementScope, taskResultSummaryReadAuditAction(result), resourceID, verifiedTaskResultMemoryKind, result, taskResultSummaryReadAuditExtra(memory, result)); err != nil {
		writeError(w, err)
		return
	}
	response := domain.TaskResultSummaryReadResponse{TaskResultSummaryGateResult: result}
	if result.Decision == domain.TaskResultSummaryGateAllowed {
		response.Memory = &memory
	}
	writeJSON(w, http.StatusOK, response)
}

type verifiedTaskResultReadScope struct {
	store.ManagementScope
	CallerInstanceID string
	SubjectID        string
}

func (s *Server) verifiedTaskResultReadScopeFromRequest(r *http.Request) (verifiedTaskResultReadScope, error) {
	scope := verifiedTaskResultReadScope{
		ManagementScope: store.ManagementScope{
			TenantID:    strings.TrimSpace(r.URL.Query().Get("tenantId")),
			WorkspaceID: strings.TrimSpace(r.URL.Query().Get("workspaceId")),
		},
		CallerInstanceID: strings.TrimSpace(r.URL.Query().Get("callerInstanceId")),
		SubjectID:        strings.TrimSpace(r.URL.Query().Get("subjectId")),
	}
	if scope.TenantID == "" || scope.WorkspaceID == "" || scope.CallerInstanceID == "" || scope.SubjectID == "" {
		return verifiedTaskResultReadScope{}, domain.BadRequest("VALIDATION_FAILED", "tenantId, workspaceId, callerInstanceId, and subjectId are required")
	}
	if err := s.requireRequestedScopeAllowed(r, scope.ManagementScope); err != nil {
		return verifiedTaskResultReadScope{}, err
	}
	return scope, nil
}

func (s *Server) evaluateVerifiedTaskResultSummaryWrite(ctx context.Context, r *http.Request, req domain.CreateVerifiedTaskResultSummaryRequest) (verifiedTaskResultWriteGate, error) {
	gate := verifiedTaskResultWriteGate{
		auditScope:          verifiedTaskResultRequestAuditScope(r),
		candidateMemoryKind: normalizedTaskResultSummaryMemoryKind(req.MemoryKind),
	}
	now := s.now()
	req.MemoryKind = strings.TrimSpace(req.MemoryKind)
	req.SourceTraceID = strings.TrimSpace(req.SourceTraceID)
	if req.MemoryKind != "" && req.MemoryKind != verifiedTaskResultMemoryKind {
		gate.result = taskResultGateApprovalRequired()
		return gate, nil
	}
	if req.SourceTraceID == "" {
		gate.result = taskResultGateDenied("memory_source_stale", "refresh_memory_source")
		return gate, nil
	}
	if req.Verification != domain.TaskResultSummaryVerificationHumanReviewedRedacted {
		gate.result = taskResultGateDenied("memory_verification_required", "verify_task_result")
		return gate, nil
	}
	if !req.ExpiresAt.After(now) || req.ExpiresAt.After(now.Add(maxTaskResultSummaryRetention)) {
		gate.result = taskResultGateDenied("memory_expiry_invalid", "set_memory_expiry")
		return gate, nil
	}
	source, found, err := s.repo.GetTrace(ctx, req.SourceTraceID)
	if err != nil {
		return verifiedTaskResultWriteGate{}, err
	}
	if !found || !verifiedTaskResultSourceTraceEligible(source) {
		gate.result = taskResultGateDenied("memory_source_stale", "refresh_memory_source")
		return gate, nil
	}
	sourceScope := store.ManagementScope{TenantID: source.TenantID, WorkspaceID: source.WorkspaceID}
	gate.auditScope = sourceScope
	if err := s.requireRequestedScopeAllowed(r, sourceScope); err != nil {
		var appErr domain.AppError
		if errors.As(err, &appErr) && appErr.Status == http.StatusForbidden {
			// Do not reveal whether a caller supplied a valid trace outside its
			// management scope. The caller gets the same refresh instruction as
			// it would for an unavailable source.
			gate.result = taskResultGateDenied("memory_source_stale", "refresh_memory_source")
			return gate, nil
		}
		return verifiedTaskResultWriteGate{}, err
	}
	if source.CreatedAt.After(now) || req.ExpiresAt.After(source.CreatedAt.Add(maxTaskResultSummaryRetention)) {
		gate.result = taskResultGateDenied("memory_expiry_invalid", "set_memory_expiry")
		return gate, nil
	}
	gate.auditScope = sourceScope

	access, accessOK, err := s.currentVerifiedTaskResultSourceAccess(ctx, source, now)
	if err != nil {
		return verifiedTaskResultWriteGate{}, err
	}
	if !accessOK {
		gate.result = taskResultGateDenied("memory_source_stale", "refresh_memory_source")
		return gate, nil
	}
	if verifiedTaskResultSourceRequiresApproval(access.capability) {
		gate.result = taskResultGateApprovalRequired()
		return gate, nil
	}
	_, found, err = s.repo.FindVerifiedTaskResultSummaryBySource(ctx, source.ID)
	if err != nil {
		return verifiedTaskResultWriteGate{}, err
	}
	if found {
		gate.result = taskResultGateDenied("memory_source_already_recorded", "review_existing_memory")
		return gate, nil
	}
	gate.result = taskResultGateAllowed()
	gate.source = source
	gate.dataScopes = access.dataScopes
	gate.summary = verifiedTaskResultSummaryFromSource(source)
	gate.auditScope = sourceScope
	return gate, nil
}

func (s *Server) evaluateVerifiedTaskResultSummaryRead(ctx context.Context, memory *domain.VerifiedTaskResultSummary, scope verifiedTaskResultReadScope, now time.Time) domain.TaskResultSummaryGateResult {
	if memory.TenantID != scope.TenantID || memory.WorkspaceID != scope.WorkspaceID || memory.CallerInstanceID != scope.CallerInstanceID || memory.SubjectID != scope.SubjectID {
		return taskResultGateDenied("memory_tenant_scope_mismatch", defaultTaskResultSummaryNextAction)
	}
	if !memory.ExpiresAt.After(now) {
		return taskResultGateDenied("memory_expired", "refresh_memory_source")
	}
	if memory.Verification != domain.TaskResultSummaryVerificationHumanReviewedRedacted || verifiedTaskResultSummaryDigest(memory.Summary) != memory.PayloadDigest {
		return taskResultGateDenied("memory_source_stale", "refresh_memory_source")
	}
	source, found, err := s.repo.GetTrace(ctx, memory.SourceTraceID)
	if err != nil || !found || !verifiedTaskResultMemoryMatchesSource(*memory, source) {
		return taskResultGateDenied("memory_source_stale", "refresh_memory_source")
	}
	access, accessOK, err := s.currentVerifiedTaskResultSourceAccess(ctx, source, now)
	if err != nil || !accessOK {
		return taskResultGateDenied("memory_source_stale", "refresh_memory_source")
	}
	if verifiedTaskResultSourceRequiresApproval(access.capability) {
		return taskResultGateApprovalRequired()
	}
	effectiveMemoryScopes, scopeOK := domain.EffectiveDataScopes(access.dataScopes, memory.DataScopes)
	if !scopeOK || len(effectiveMemoryScopes) == 0 {
		return taskResultGateDenied("memory_source_stale", "refresh_memory_source")
	}
	memory.DataScopes = effectiveMemoryScopes
	return taskResultGateAllowed()
}

func verifiedTaskResultSourceTraceEligible(source domain.TraceEvent) bool {
	return !source.CreatedAt.IsZero() &&
		source.Decision == domain.TraceDecisionAllowed &&
		source.UpstreamAttempts > 0 &&
		source.UpstreamStatus > 0 &&
		strings.TrimSpace(source.UpstreamError) == "" &&
		strings.TrimSpace(source.TenantID) != "" &&
		strings.TrimSpace(source.WorkspaceID) != "" &&
		strings.TrimSpace(source.CallerInstanceID) != "" &&
		strings.TrimSpace(source.SubjectID) != "" &&
		strings.TrimSpace(source.TargetID) != "" &&
		strings.TrimSpace(source.CapabilityID) != "" &&
		source.CapabilityVersion > 0 &&
		strings.TrimSpace(source.CapabilityFingerprint) != "" &&
		strings.TrimSpace(source.EntitlementID) != "" &&
		strings.TrimSpace(source.WorkspaceAssignmentID) != "" &&
		strings.TrimSpace(source.InstanceAssignmentID) != "" &&
		len(source.DataScopes) > 0
}

func verifiedTaskResultMemoryMatchesSource(memory domain.VerifiedTaskResultSummary, source domain.TraceEvent) bool {
	if !verifiedTaskResultSourceTraceEligible(source) ||
		memory.SourceTraceID != source.ID ||
		memory.TenantID != source.TenantID ||
		memory.WorkspaceID != source.WorkspaceID ||
		memory.CallerInstanceID != source.CallerInstanceID ||
		memory.SubjectID != source.SubjectID ||
		memory.TargetID != source.TargetID ||
		memory.CapabilityID != source.CapabilityID ||
		memory.ExpiresAt.After(source.CreatedAt.Add(maxTaskResultSummaryRetention)) ||
		len(memory.DataScopes) == 0 {
		return false
	}
	_, scopeOK := domain.EffectiveDataScopes(source.DataScopes, memory.DataScopes)
	return scopeOK
}

func (s *Server) currentVerifiedTaskResultSourceAccess(ctx context.Context, source domain.TraceEvent, now time.Time) (verifiedTaskResultSourceAccess, bool, error) {
	caller, found, err := s.repo.GetAgent(ctx, source.CallerInstanceID)
	if err != nil {
		return verifiedTaskResultSourceAccess{}, false, err
	}
	if !found || caller.Status != domain.AgentStatusActive || caller.TenantID != source.TenantID || caller.WorkspaceID != source.WorkspaceID {
		return verifiedTaskResultSourceAccess{}, false, nil
	}
	target, found, err := s.repo.GetAgent(ctx, source.TargetID)
	if err != nil {
		return verifiedTaskResultSourceAccess{}, false, err
	}
	if !found || target.Status != domain.AgentStatusActive || target.TenantID != source.TenantID || target.WorkspaceID != source.WorkspaceID {
		return verifiedTaskResultSourceAccess{}, false, nil
	}
	capability, found, err := s.repo.GetCapability(ctx, source.CapabilityID)
	if err != nil {
		return verifiedTaskResultSourceAccess{}, false, err
	}
	if !found || capability.TargetID != source.TargetID || capability.Version != source.CapabilityVersion || domain.CapabilityFingerprint(capability) != source.CapabilityFingerprint {
		return verifiedTaskResultSourceAccess{}, false, nil
	}
	liveDecision, err := s.repo.EvaluateCapabilityAccess(ctx, store.CapabilityAccessRequest{
		TenantID:         source.TenantID,
		WorkspaceID:      source.WorkspaceID,
		CallerInstanceID: source.CallerInstanceID,
		SubjectID:        source.SubjectID,
		TargetID:         source.TargetID,
		CapabilityID:     source.CapabilityID,
		Now:              now,
	})
	if err != nil {
		return verifiedTaskResultSourceAccess{}, false, err
	}
	if !liveDecision.Allowed ||
		liveDecision.EntitlementID != source.EntitlementID ||
		liveDecision.WorkspaceAssignmentID != source.WorkspaceAssignmentID ||
		liveDecision.InstanceAssignmentID != source.InstanceAssignmentID {
		return verifiedTaskResultSourceAccess{}, false, nil
	}
	// Re-read the capability after evaluating the entitlement chain. This
	// narrows the request-in-flight window and ensures the risk gate below uses
	// the same immutable witness as the source trace.
	capability, found, err = s.repo.GetCapability(ctx, source.CapabilityID)
	if err != nil {
		return verifiedTaskResultSourceAccess{}, false, err
	}
	if !found || capability.TargetID != source.TargetID || capability.Version != source.CapabilityVersion || domain.CapabilityFingerprint(capability) != source.CapabilityFingerprint {
		return verifiedTaskResultSourceAccess{}, false, nil
	}
	dataScopes, scopeOK := domain.EffectiveDataScopes(liveDecision.DataScopes, source.DataScopes)
	if !scopeOK || len(dataScopes) == 0 {
		return verifiedTaskResultSourceAccess{}, false, nil
	}
	return verifiedTaskResultSourceAccess{capability: capability, dataScopes: dataScopes}, true, nil
}

func verifiedTaskResultSourceRequiresApproval(capability domain.Capability) bool {
	return capability.RiskLevel == domain.CapabilityRiskHigh ||
		capability.RiskLevel == domain.CapabilityRiskCritical ||
		capability.Action == domain.CapabilityActionExport ||
		capability.Action == domain.CapabilityActionDelete ||
		capability.Action == domain.CapabilityActionAdmin
}

func verifiedTaskResultSummaryFromSource(source domain.TraceEvent) string {
	if source.UpstreamStatus >= http.StatusOK && source.UpstreamStatus < http.StatusMultipleChoices {
		return "Verified task result recorded from a successful allowed execution."
	}
	return "Verified task result recorded from an allowed execution with a non-success upstream response."
}

func verifiedTaskResultSummaryDigest(summary string) string {
	digest := sha256.Sum256([]byte(summary))
	return hex.EncodeToString(digest[:])
}

func normalizedTaskResultSummaryMemoryKind(value string) string {
	if strings.TrimSpace(value) == "" || strings.TrimSpace(value) == verifiedTaskResultMemoryKind {
		return verifiedTaskResultMemoryKind
	}
	return "unsupported"
}

func taskResultGateAllowed() domain.TaskResultSummaryGateResult {
	return domain.TaskResultSummaryGateResult{
		Decision:       domain.TaskResultSummaryGateAllowed,
		ReasonCode:     "memory_allowed",
		NextActionCode: "create_or_read_memory",
	}
}

func taskResultGateDenied(reasonCode string, nextActionCode string) domain.TaskResultSummaryGateResult {
	return domain.TaskResultSummaryGateResult{
		Decision:       domain.TaskResultSummaryGateDenied,
		ReasonCode:     reasonCode,
		NextActionCode: nextActionCode,
	}
}

func taskResultGateApprovalRequired() domain.TaskResultSummaryGateResult {
	return domain.TaskResultSummaryGateResult{
		Decision:       domain.TaskResultSummaryGateApprovalRequired,
		ReasonCode:     "memory_approval_required",
		NextActionCode: "request_memory_approval",
	}
}

func verifiedTaskResultRequestAuditScope(r *http.Request) store.ManagementScope {
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return store.ManagementScope{}
	}
	return store.ManagementScope{TenantID: principal.TenantID, WorkspaceID: principal.WorkspaceID}
}

func (s *Server) appendVerifiedTaskResultDecisionAudit(r *http.Request, scope store.ManagementScope, action string, resourceID string, candidateMemoryKind string, result domain.TaskResultSummaryGateResult, extra map[string]any) error {
	_, err := s.repo.AppendAuditEvent(r.Context(), s.verifiedTaskResultDecisionAuditEvent(r, scope, action, resourceID, candidateMemoryKind, result, extra))
	return err
}

func (s *Server) verifiedTaskResultDecisionAuditEvent(r *http.Request, scope store.ManagementScope, action string, resourceID string, candidateMemoryKind string, result domain.TaskResultSummaryGateResult, extra map[string]any) domain.AuditEvent {
	metadata := map[string]any{
		"memoryKind":     normalizedTaskResultSummaryMemoryKind(candidateMemoryKind),
		"decision":       result.Decision,
		"reasonCode":     result.ReasonCode,
		"nextActionCode": result.NextActionCode,
		"policyVersion":  verifiedTaskResultMemoryPolicyV1,
	}
	for key, value := range extra {
		metadata[key] = value
	}
	summary := "Verified task result memory decision"
	if result.Decision == domain.TaskResultSummaryGateDenied || result.Decision == domain.TaskResultSummaryGateApprovalRequired {
		summary = "Verified task result memory denied"
	}
	return s.managementAuditEvent(r, scope.TenantID, scope.WorkspaceID, action, "verified_task_result_summary", resourceID, summary, metadata)
}

func taskResultSummaryReadAuditAction(result domain.TaskResultSummaryGateResult) string {
	if result.Decision == domain.TaskResultSummaryGateAllowed {
		return "memory_read_allowed"
	}
	return "memory_read_denied"
}

func taskResultSummaryWritePreflightAuditAction(result domain.TaskResultSummaryGateResult) string {
	if result.Decision == domain.TaskResultSummaryGateAllowed {
		return "memory_write_preflight"
	}
	return "memory_write_denied"
}

func taskResultSummaryReadAuditExtra(memory domain.VerifiedTaskResultSummary, result domain.TaskResultSummaryGateResult) map[string]any {
	if result.Decision != domain.TaskResultSummaryGateAllowed {
		return nil
	}
	return map[string]any{
		"sourceTraceId": memory.SourceTraceID,
		"payloadDigest": memory.PayloadDigest,
		"expiresAt":     memory.ExpiresAt,
	}
}
