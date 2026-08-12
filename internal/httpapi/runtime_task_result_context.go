package httpapi

import (
	"net/http"
	"strings"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const (
	agentHarborTaskResultMemoryIDHeader = "X-AgentHarbor-Task-Result-Memory-Id"
	maxTaskResultMemoryIDLength         = 128
)

func requestedTaskResultMemoryID(r *http.Request) (string, bool, bool) {
	values, present := r.Header[http.CanonicalHeaderKey(agentHarborTaskResultMemoryIDHeader)]
	if !present {
		return "", false, true
	}
	if len(values) != 1 {
		return "", true, false
	}
	memoryID := strings.TrimSpace(values[0])
	if memoryID == "" || len(memoryID) > maxTaskResultMemoryIDLength || strings.Contains(memoryID, ",") {
		return "", true, false
	}
	return memoryID, true, true
}

func (s *Server) resolveVerifiedTaskResultRuntimeContext(
	r *http.Request,
	caller domain.Agent,
	identity runtimeIdentity,
	target domain.Agent,
	capability domain.Capability,
	decision domain.CapabilityAccessDecision,
) (*agentHarborTaskResultContext, error) {
	memoryID, requested, valid := requestedTaskResultMemoryID(r)
	if !requested {
		return nil, nil
	}
	if !valid {
		result := taskResultGateDenied("memory_request_invalid", defaultTaskResultSummaryNextAction)
		if err := s.appendVerifiedTaskResultRuntimeAudit(r, caller.ID, identity, target.ID, capability.ID, "", result); err != nil {
			return nil, err
		}
		return nil, taskResultRuntimeContextUnavailable()
	}
	if strings.TrimSpace(identity.TenantID) == "" ||
		strings.TrimSpace(identity.WorkspaceID) == "" ||
		strings.TrimSpace(identity.CallerInstanceID) == "" ||
		strings.TrimSpace(identity.SubjectID) == "" {
		result := taskResultGateDenied("memory_runtime_scope_missing", defaultTaskResultSummaryNextAction)
		if err := s.appendVerifiedTaskResultRuntimeAudit(r, caller.ID, identity, target.ID, capability.ID, "", result); err != nil {
			return nil, err
		}
		return nil, taskResultRuntimeContextUnavailable()
	}

	memory, found, err := s.repo.GetVerifiedTaskResultSummary(r.Context(), memoryID, store.VerifiedTaskResultSummaryScope{
		TenantID:         identity.TenantID,
		WorkspaceID:      identity.WorkspaceID,
		CallerInstanceID: identity.CallerInstanceID,
		SubjectID:        identity.SubjectID,
	})
	if err != nil {
		return nil, err
	}
	if !found {
		result := taskResultGateDenied("memory_not_available", defaultTaskResultSummaryNextAction)
		if err := s.appendVerifiedTaskResultRuntimeAudit(r, caller.ID, identity, target.ID, capability.ID, "", result); err != nil {
			return nil, err
		}
		return nil, taskResultRuntimeContextUnavailable()
	}

	readScope := verifiedTaskResultReadScope{
		ManagementScope:  store.ManagementScope{TenantID: identity.TenantID, WorkspaceID: identity.WorkspaceID},
		CallerInstanceID: identity.CallerInstanceID,
		SubjectID:        identity.SubjectID,
	}
	result := s.evaluateVerifiedTaskResultSummaryRead(r.Context(), &memory, readScope, s.now())
	if result.Decision == domain.TaskResultSummaryGateAllowed && (memory.TargetID != target.ID || memory.CapabilityID != capability.ID) {
		result = taskResultGateDenied("memory_runtime_target_mismatch", defaultTaskResultSummaryNextAction)
	}
	if result.Decision == domain.TaskResultSummaryGateAllowed {
		effectiveScopes, ok := domain.EffectiveDataScopes(decision.DataScopes, memory.DataScopes)
		if !ok || len(effectiveScopes) == 0 {
			result = taskResultGateDenied("memory_source_stale", "refresh_memory_source")
		} else {
			memory.DataScopes = effectiveScopes
		}
	}
	if result.Decision != domain.TaskResultSummaryGateAllowed {
		if err := s.appendVerifiedTaskResultRuntimeAudit(r, caller.ID, identity, target.ID, capability.ID, "", result); err != nil {
			return nil, err
		}
		return nil, taskResultRuntimeContextUnavailable()
	}
	if err := s.appendVerifiedTaskResultRuntimeAudit(r, caller.ID, identity, target.ID, capability.ID, memory.ID, result); err != nil {
		return nil, err
	}
	return &agentHarborTaskResultContext{
		Kind:       verifiedTaskResultMemoryKind,
		Summary:    memory.Summary,
		DataScopes: domain.CloneDataScopes(memory.DataScopes),
	}, nil
}

func (s *Server) appendVerifiedTaskResultRuntimeAudit(
	r *http.Request,
	actor string,
	identity runtimeIdentity,
	targetID string,
	capabilityID string,
	resourceID string,
	result domain.TaskResultSummaryGateResult,
) error {
	action := "memory_runtime_context_denied"
	summary := "Verified task result runtime context denied"
	if result.Decision == domain.TaskResultSummaryGateAllowed {
		action = "memory_runtime_context_allowed"
		summary = "Verified task result runtime context attached"
	}
	_, err := s.repo.AppendAuditEvent(r.Context(), domain.AuditEvent{
		ID:           security.NewID("aud"),
		TenantID:     identity.TenantID,
		WorkspaceID:  identity.WorkspaceID,
		Actor:        actor,
		Action:       action,
		ResourceType: "verified_task_result_summary",
		ResourceID:   resourceID,
		Summary:      summary,
		Metadata: map[string]any{
			"decision":       result.Decision,
			"reasonCode":     result.ReasonCode,
			"nextActionCode": result.NextActionCode,
			"policyVersion":  verifiedTaskResultMemoryPolicyV1,
			"targetId":       targetID,
			"capabilityId":   capabilityID,
		},
		CreatedAt: s.now(),
	})
	return err
}

func taskResultRuntimeContextUnavailable() domain.AppError {
	return domain.AppError{
		Status:  http.StatusForbidden,
		Code:    "MEMORY_CONTEXT_NOT_AVAILABLE",
		Message: "requested task result context is not available",
	}
}
