package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type permissionPackageProductionReadinessQuery struct {
	TenantID              string
	WorkspaceID           string
	TemplateID            string
	TargetID              string
	CallerInstanceID      string
	RequestedCapabilityID string
	SubjectID             string
	Region                string
	RequestText           string
	SubjectSelector       string
	ApprovalRequestID     string
	TraceLimit            int
}

type permissionPackageWorkbenchPreviewResponse struct {
	Draft               domain.PermissionPackageDraft                 `json:"draft"`
	ApprovalRequest     *domain.PermissionPackageApprovalRequest      `json:"approvalRequest,omitempty"`
	LatestApplication   *domain.PermissionPackageApplication          `json:"latestApplication,omitempty"`
	ProductionReadiness *permissionPackageProductionReadinessResponse `json:"productionReadiness,omitempty"`
	Summary             permissionPackageWorkbenchSummary             `json:"summary"`
	GeneratedAt         time.Time                                     `json:"generatedAt"`
}

type permissionPackageWorkbenchSummary struct {
	Status                 string                           `json:"status"`
	PrimaryActionCode      string                           `json:"primaryActionCode"`
	NextActionCode         string                           `json:"nextActionCode,omitempty"`
	ApprovalRequired       bool                             `json:"approvalRequired"`
	CanApply               bool                             `json:"canApply"`
	Applied                bool                             `json:"applied"`
	RuntimeEvidenceReady   bool                             `json:"runtimeEvidenceReady"`
	ProductionReady        bool                             `json:"productionReady"`
	AllowedCapabilityCount int                              `json:"allowedCapabilityCount"`
	BlockedCapabilityCount int                              `json:"blockedCapabilityCount"`
	PlannedObjectCount     int                              `json:"plannedObjectCount"`
	ReadinessReadyCount    int                              `json:"readinessReadyCount"`
	ReadinessTotalCount    int                              `json:"readinessTotalCount"`
	BlockingCount          int                              `json:"blockingCount"`
	WarningCount           int                              `json:"warningCount"`
	Steps                  []permissionPackageWorkbenchStep `json:"steps"`
}

type permissionPackageWorkbenchStep struct {
	Key        string `json:"key"`
	Status     string `json:"status"`
	DetailCode string `json:"detailCode"`
	Count      int    `json:"count,omitempty"`
	Total      int    `json:"total,omitempty"`
}

type permissionPackageProductionReadinessResponse struct {
	Status            string                                          `json:"status"`
	Summary           permissionPackageProductionReadinessSummary     `json:"summary"`
	Checks            []permissionPackageProductionReadinessCheck     `json:"checks"`
	LatestApplication *domain.PermissionPackageApplication            `json:"latestApplication,omitempty"`
	Preflight         *domain.PermissionPackageApplyPreflightResponse `json:"preflight,omitempty"`
	ApplicationHealth *permissionPackageApplicationHealthRow          `json:"applicationHealth,omitempty"`
	ApplicationImpact *permissionPackageApplicationImpactResponse     `json:"applicationImpact,omitempty"`
	AccessProfile     *tenantAccessProfileResponse                    `json:"accessProfile,omitempty"`
	RuntimeEvidence   permissionPackageRuntimeEvidence                `json:"runtimeEvidence"`
	AuditEvidence     permissionPackageAuditEvidence                  `json:"auditEvidence"`
	NextActionCode    string                                          `json:"nextActionCode"`
	NextActions       []string                                        `json:"nextActions"`
	GeneratedAt       time.Time                                       `json:"generatedAt"`
}

type permissionPackageProductionReadinessSummary struct {
	ReadyCount         int  `json:"readyCount"`
	WarningCount       int  `json:"warningCount"`
	BlockingCount      int  `json:"blockingCount"`
	HasApplication     bool `json:"hasApplication"`
	HasAllowedTrace    bool `json:"hasAllowedTrace"`
	HasDeniedTrace     bool `json:"hasDeniedTrace"`
	HasAppliedAudit    bool `json:"hasAppliedAudit"`
	AccessProfileReady bool `json:"accessProfileReady"`
}

type permissionPackageProductionReadinessCheck struct {
	Code       string                                    `json:"code"`
	Severity   domain.PermissionPackagePreflightSeverity `json:"severity"`
	Message    string                                    `json:"message"`
	EvidenceID string                                    `json:"evidenceId,omitempty"`
}

type permissionPackageRuntimeEvidence struct {
	AllowedTrace *domain.TraceEvent `json:"allowedTrace,omitempty"`
	DeniedTrace  *domain.TraceEvent `json:"deniedTrace,omitempty"`
}

type permissionPackageAuditEvidence struct {
	AppliedEvent *domain.AuditEvent `json:"appliedEvent,omitempty"`
}

const (
	permissionPackageProductionEvidenceReportVersion         = "production-readiness-report/v1"
	permissionPackageProductionEvidenceReportDigestAlgorithm = "sha256-canonical-json-v1"
)

type permissionPackageProductionEvidenceReportResponse struct {
	ReportVersion         string                                      `json:"reportVersion"`
	ReportDigest          string                                      `json:"reportDigest,omitempty"`
	ReportDigestAlgorithm string                                      `json:"reportDigestAlgorithm"`
	GeneratedBy           string                                      `json:"generatedBy"`
	GeneratedAt           time.Time                                   `json:"generatedAt"`
	PlatformContract      permissionPackageProductionPlatformContract `json:"platformContract"`
	Scope                 permissionPackageProductionEvidenceScope    `json:"scope"`
	Status                string                                      `json:"status"`
	Summary               permissionPackageProductionReadinessSummary `json:"summary"`
	Checks                []permissionPackageProductionReadinessCheck `json:"checks"`
	Evidence              permissionPackageProductionEvidenceRefs     `json:"evidence"`
	NextActionCode        string                                      `json:"nextActionCode"`
	NextActions           []string                                    `json:"nextActions"`
	ReadinessGeneratedAt  time.Time                                   `json:"readinessGeneratedAt"`
}

type permissionPackageProductionPlatformContract struct {
	APIVersion               string                                                   `json:"apiVersion"`
	ManagementMcpToolCatalog permissionPackageProductionManagementMcpToolCatalogStamp `json:"managementMcpToolCatalog"`
}

type permissionPackageProductionManagementMcpToolCatalogStamp struct {
	MetadataVersion int    `json:"metadataVersion"`
	CatalogDigest   string `json:"catalogDigest"`
}

type permissionPackageProductionEvidenceScope struct {
	TenantID              string `json:"tenantId"`
	WorkspaceID           string `json:"workspaceId"`
	TemplateID            string `json:"templateId"`
	TargetID              string `json:"targetId"`
	CallerInstanceID      string `json:"callerInstanceId"`
	RequestedCapabilityID string `json:"requestedCapabilityId,omitempty"`
	SubjectID             string `json:"subjectId,omitempty"`
	Region                string `json:"region,omitempty"`
	SubjectSelector       string `json:"subjectSelector,omitempty"`
}

type permissionPackageProductionEvidenceRefs struct {
	Application       permissionPackageProductionApplicationEvidence `json:"application"`
	Runtime           permissionPackageProductionRuntimeEvidence     `json:"runtime"`
	Audit             permissionPackageProductionAuditEvidence       `json:"audit"`
	AccessProfile     permissionPackageProductionEvidenceState       `json:"accessProfile"`
	ApplicationHealth permissionPackageProductionEvidenceState       `json:"applicationHealth"`
	ApplicationImpact permissionPackageProductionEvidenceState       `json:"applicationImpact"`
}

type permissionPackageProductionApplicationEvidence struct {
	Present               bool               `json:"present"`
	ID                    string             `json:"id,omitempty"`
	DraftID               string             `json:"draftId,omitempty"`
	RequestedCapabilityID string             `json:"requestedCapabilityId,omitempty"`
	TemplateVersion       int                `json:"templateVersion,omitempty"`
	AppliedAt             *time.Time         `json:"appliedAt,omitempty"`
	AllowedCapabilityIDs  []string           `json:"allowedCapabilityIds,omitempty"`
	AllowedCapabilityKeys []string           `json:"allowedCapabilityKeys,omitempty"`
	DataScopes            []domain.DataScope `json:"dataScopes,omitempty"`
}

type permissionPackageProductionRuntimeEvidence struct {
	AllowedTraceID string `json:"allowedTraceId,omitempty"`
	DeniedTraceID  string `json:"deniedTraceId,omitempty"`
}

type permissionPackageProductionAuditEvidence struct {
	AppliedEventID string `json:"appliedEventId,omitempty"`
}

type permissionPackageProductionEvidenceState struct {
	Present bool   `json:"present"`
	Status  string `json:"status,omitempty"`
}

func (s *Server) previewPermissionPackageWorkbench(w http.ResponseWriter, r *http.Request) {
	var req domain.PermissionPackageApplyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageDraftScope(r, req.PermissionPackageDraftRequest); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.permissionPackageWorkbenchPreview(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getPermissionPackageProductionReadiness(w http.ResponseWriter, r *http.Request) {
	query, err := permissionPackageProductionReadinessQueryFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageQueryScope(r, query); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.permissionPackageProductionReadiness(r.Context(), query)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getPermissionPackageProductionEvidenceReport(w http.ResponseWriter, r *http.Request) {
	query, err := permissionPackageProductionReadinessQueryFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageQueryScope(r, query); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.permissionPackageProductionEvidenceReport(r.Context(), query, managementActor(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func permissionPackageProductionReadinessQueryFromRequest(r *http.Request) (permissionPackageProductionReadinessQuery, error) {
	values := r.URL.Query()
	query := permissionPackageProductionReadinessQuery{
		TenantID:              strings.TrimSpace(values.Get("tenantId")),
		WorkspaceID:           strings.TrimSpace(values.Get("workspaceId")),
		TemplateID:            strings.TrimSpace(values.Get("templateId")),
		TargetID:              strings.TrimSpace(values.Get("targetId")),
		CallerInstanceID:      strings.TrimSpace(values.Get("callerInstanceId")),
		RequestedCapabilityID: strings.TrimSpace(values.Get("requestedCapabilityId")),
		SubjectID:             strings.TrimSpace(values.Get("subjectId")),
		Region:                strings.TrimSpace(values.Get("region")),
		RequestText:           strings.TrimSpace(values.Get("requestText")),
		SubjectSelector:       strings.TrimSpace(values.Get("subjectSelector")),
		ApprovalRequestID:     strings.TrimSpace(values.Get("approvalRequestId")),
		TraceLimit:            defaultAccessProfileTraceLimit,
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "tenantId", value: query.TenantID},
		{name: "workspaceId", value: query.WorkspaceID},
		{name: "templateId", value: query.TemplateID},
		{name: "targetId", value: query.TargetID},
		{name: "callerInstanceId", value: query.CallerInstanceID},
	} {
		if required.value == "" {
			return permissionPackageProductionReadinessQuery{}, domain.BadRequest("VALIDATION_FAILED", required.name+" is required")
		}
	}
	if raw := strings.TrimSpace(values.Get("traceLimit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 0 || limit > maxAccessProfileTraceLimit {
			return permissionPackageProductionReadinessQuery{}, domain.BadRequest("VALIDATION_FAILED", "traceLimit must be between 0 and 100")
		}
		query.TraceLimit = limit
	}
	return query, nil
}

func (s *Server) permissionPackageWorkbenchPreview(ctx context.Context, req domain.PermissionPackageApplyRequest) (permissionPackageWorkbenchPreviewResponse, error) {
	req.ApprovalRequestID = strings.TrimSpace(req.ApprovalRequestID)
	draft, err := s.buildPermissionPackageDraft(ctx, req.PermissionPackageDraftRequest)
	if err != nil {
		return permissionPackageWorkbenchPreviewResponse{}, err
	}
	approval, err := s.permissionPackageWorkbenchApprovalRequest(ctx, req.ApprovalRequestID, draft)
	if err != nil {
		return permissionPackageWorkbenchPreviewResponse{}, err
	}
	if approval != nil && approval.Status == domain.PermissionPackageApprovalStatusApproved {
		req.ApprovalRequestID = approval.ID
	}

	var readiness *permissionPackageProductionReadinessResponse
	if permissionPackageWorkbenchShouldCheckReadiness(draft, approval) {
		next, err := s.permissionPackageProductionReadiness(ctx, permissionPackageProductionReadinessQueryFromApplyRequest(req))
		if err != nil {
			return permissionPackageWorkbenchPreviewResponse{}, err
		}
		readiness = &next
	}

	return permissionPackageWorkbenchPreviewResponse{
		Draft:               draft,
		ApprovalRequest:     approval,
		LatestApplication:   permissionPackageWorkbenchLatestApplication(readiness),
		ProductionReadiness: readiness,
		Summary:             permissionPackageWorkbenchSummaryFor(draft, approval, readiness),
		GeneratedAt:         s.now(),
	}, nil
}

func permissionPackageWorkbenchShouldCheckReadiness(draft domain.PermissionPackageDraft, approval *domain.PermissionPackageApprovalRequest) bool {
	if !permissionPackageWorkbenchCanCheckReadiness(draft) {
		return false
	}
	if approval == nil {
		return true
	}
	return approval.Status == domain.PermissionPackageApprovalStatusApproved
}

func permissionPackageProductionReadinessQueryFromApplyRequest(req domain.PermissionPackageApplyRequest) permissionPackageProductionReadinessQuery {
	input := trimPermissionPackageDraftRequest(req.PermissionPackageDraftRequest)
	return permissionPackageProductionReadinessQuery{
		TenantID:              input.TenantID,
		WorkspaceID:           input.WorkspaceID,
		TemplateID:            input.TemplateID,
		TargetID:              input.TargetID,
		CallerInstanceID:      input.CallerInstanceID,
		RequestedCapabilityID: input.RequestedCapabilityID,
		Region:                input.Region,
		RequestText:           input.RequestText,
		SubjectSelector:       input.SubjectSelector,
		ApprovalRequestID:     strings.TrimSpace(req.ApprovalRequestID),
		TraceLimit:            defaultAccessProfileTraceLimit,
	}
}

func permissionPackageWorkbenchCanCheckReadiness(draft domain.PermissionPackageDraft) bool {
	input := draft.Input
	return input.TenantID != "" &&
		input.WorkspaceID != "" &&
		input.TemplateID != "" &&
		input.TargetID != "" &&
		input.CallerInstanceID != ""
}

func (s *Server) permissionPackageWorkbenchApprovalRequest(ctx context.Context, approvalRequestID string, draft domain.PermissionPackageDraft) (*domain.PermissionPackageApprovalRequest, error) {
	if draft.PolicyGate.CanApplyDirectly || !draft.Readiness.CanApply {
		return nil, nil
	}
	now := s.now()
	if approvalRequestID != "" {
		approval, ok, err := s.repo.GetPermissionPackageApprovalRequest(ctx, approvalRequestID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, domain.NotFound("approval request not found")
		}
		if !permissionPackageApprovalRequestMatchesDraftSnapshot(approval, draft) {
			return nil, domain.BadRequest("VALIDATION_FAILED", "approval request does not match current permission request")
		}
		if permissionPackageApprovalRequestExpired(approval, now) {
			return nil, nil
		}
		return &approval, nil
	}
	rows, err := s.repo.ListPermissionPackageApprovalRequests(ctx, store.PermissionPackageApprovalRequestFilter{
		ManagementScope:            store.ManagementScope{TenantID: draft.Input.TenantID, WorkspaceID: draft.Input.WorkspaceID},
		TemplateID:                 draft.Template.ID,
		TargetID:                   draft.Input.TargetID,
		CallerInstanceID:           draft.Input.CallerInstanceID,
		RequestedCapabilityID:      draft.Input.RequestedCapabilityID,
		MatchRequestedCapabilityID: true,
		Limit:                      20,
	})
	if err != nil {
		return nil, err
	}
	sortPermissionPackageApprovalRequests(rows)
	var selected *domain.PermissionPackageApprovalRequest
	for _, row := range rows {
		if !permissionPackageApprovalRequestMatchesDraftSnapshot(row, draft) {
			continue
		}
		if (row.Status == domain.PermissionPackageApprovalStatusPending || row.Status == domain.PermissionPackageApprovalStatusApproved) &&
			permissionPackageApprovalRequestExpired(row, now) {
			continue
		}
		if row.Status == domain.PermissionPackageApprovalStatusApproved &&
			validatePermissionPackageApprovalForDraft(row, draft, now) != nil {
			continue
		}
		candidate := row
		if selected == nil || permissionPackageApprovalPreviewRank(candidate) > permissionPackageApprovalPreviewRank(*selected) {
			selected = &candidate
		}
	}
	return selected, nil
}

func permissionPackageApprovalPreviewRank(request domain.PermissionPackageApprovalRequest) int {
	switch request.Status {
	case domain.PermissionPackageApprovalStatusApproved:
		return 3
	case domain.PermissionPackageApprovalStatusPending:
		return 2
	case domain.PermissionPackageApprovalStatusRejected:
		return 1
	case domain.PermissionPackageApprovalStatusWithdrawn:
		return 1
	default:
		return 0
	}
}

func permissionPackageApprovalRequestMatchesDraftSnapshot(approval domain.PermissionPackageApprovalRequest, draft domain.PermissionPackageDraft) bool {
	allowedCapabilityIDs, allowedCapabilityKeys := permissionPackageCapabilityIDsAndKeys(draft.AllowedCapabilities)
	allowedCapabilityFingerprints := permissionPackageCapabilityFingerprints(draft.AllowedCapabilities)
	return approval.DraftID == draft.ID &&
		approval.TemplateID == draft.Template.ID &&
		approval.TemplateVersion == draft.Template.Version &&
		approval.PolicyVersion == draft.PolicyGate.PolicyVersion &&
		approval.TenantID == draft.Input.TenantID &&
		approval.WorkspaceID == draft.Input.WorkspaceID &&
		approval.TargetID == draft.Input.TargetID &&
		approval.CallerInstanceID == draft.Input.CallerInstanceID &&
		approval.RequestedCapabilityID == draft.Input.RequestedCapabilityID &&
		approval.SubjectSelector == draft.Input.SubjectSelector &&
		approval.RequestText == draft.Input.RequestText &&
		approval.Region == draft.Input.Region &&
		samePermissionPackageDataScopes(approval.DataScopes, draft.DataScopes) &&
		sameStringSet(approval.AllowedCapabilityIDs, allowedCapabilityIDs) &&
		sameStringSet(approval.AllowedCapabilityKeys, allowedCapabilityKeys) &&
		sameStringSet(approval.AllowedCapabilityFingerprints, allowedCapabilityFingerprints)
}

func permissionPackageWorkbenchLatestApplication(readiness *permissionPackageProductionReadinessResponse) *domain.PermissionPackageApplication {
	if readiness == nil {
		return nil
	}
	return readiness.LatestApplication
}

func permissionPackageWorkbenchSummaryFor(draft domain.PermissionPackageDraft, approval *domain.PermissionPackageApprovalRequest, readiness *permissionPackageProductionReadinessResponse) permissionPackageWorkbenchSummary {
	approvalRequired := !draft.PolicyGate.CanApplyDirectly
	approvalResolvedWithoutApproval := approvalRequired && approval != nil &&
		(approval.Status == domain.PermissionPackageApprovalStatusRejected || approval.Status == domain.PermissionPackageApprovalStatusWithdrawn)
	applied := readiness != nil && readiness.LatestApplication != nil
	approvalApproved := !approvalRequired || applied || approval != nil && approval.Status == domain.PermissionPackageApprovalStatusApproved
	runtimeEvidenceReady := permissionPackageProductionRuntimeEvidenceReady(readiness)
	productionReady := readiness != nil && readiness.Status == "ready"
	canApply := draft.Readiness.CanApply && approvalApproved && !approvalResolvedWithoutApproval
	summary := permissionPackageWorkbenchSummary{
		ApprovalRequired:       approvalRequired,
		CanApply:               canApply,
		Applied:                applied,
		RuntimeEvidenceReady:   runtimeEvidenceReady,
		ProductionReady:        productionReady,
		AllowedCapabilityCount: len(draft.AllowedCapabilities),
		BlockedCapabilityCount: len(draft.BlockedCapabilities),
	}
	if readiness != nil {
		summary.NextActionCode = readiness.NextActionCode
		summary.ReadinessReadyCount = readiness.Summary.ReadyCount
		summary.ReadinessTotalCount = len(readiness.Checks)
		summary.BlockingCount = readiness.Summary.BlockingCount
		summary.WarningCount = readiness.Summary.WarningCount
		if readiness.Preflight != nil {
			summary.PlannedObjectCount = readiness.Preflight.Summary.PlannedTenantEntitlementCount +
				readiness.Preflight.Summary.PlannedWorkspaceAssignmentCount +
				readiness.Preflight.Summary.PlannedInstanceAssignmentCount
		}
	}
	if summary.PlannedObjectCount == 0 {
		summary.PlannedObjectCount = len(draft.AllowedCapabilities) * 3
	}
	summary.Status, summary.PrimaryActionCode = permissionPackageWorkbenchStatusAndAction(draft, approval, applied, productionReady, canApply)
	summary.Steps = permissionPackageWorkbenchSteps(draft, approval, applied, runtimeEvidenceReady, productionReady, canApply)
	return summary
}

func permissionPackageWorkbenchStatusAndAction(draft domain.PermissionPackageDraft, approval *domain.PermissionPackageApprovalRequest, applied bool, productionReady bool, canApply bool) (string, string) {
	approvalRequired := !draft.PolicyGate.CanApplyDirectly
	if !draft.Readiness.CanApply {
		return "needs_input", "complete_request"
	}
	if productionReady {
		return "production_ready", "export_acceptance_report"
	}
	if applied {
		return "validating", "run_runtime_validation"
	}
	if approvalRequired {
		if approval == nil ||
			approval.Status == domain.PermissionPackageApprovalStatusRejected ||
			approval.Status == domain.PermissionPackageApprovalStatusWithdrawn {
			return "awaiting_approval", "create_approval_request"
		}
		if approval.Status == domain.PermissionPackageApprovalStatusPending {
			return "awaiting_approval", "review_approval_request"
		}
	}
	if !applied && canApply {
		return "ready_to_apply", "apply_permission_package"
	}
	return "blocked", "complete_request"
}

func permissionPackageWorkbenchSteps(draft domain.PermissionPackageDraft, approval *domain.PermissionPackageApprovalRequest, applied bool, runtimeEvidenceReady bool, productionReady bool, canApply bool) []permissionPackageWorkbenchStep {
	approvalRequired := !draft.PolicyGate.CanApplyDirectly
	approvalComplete := !approvalRequired || applied || approval != nil && approval.Status == domain.PermissionPackageApprovalStatusApproved
	requestStatus := "complete"
	requestDetail := "request_ready"
	if !draft.Readiness.CanApply {
		requestStatus = "current"
		requestDetail = "request_needs_input"
	}
	approvalStatus := "waiting"
	approvalDetail := "approval_waiting"
	if approvalComplete {
		approvalStatus = "complete"
		if approvalRequired {
			approvalDetail = "approval_approved"
		} else {
			approvalDetail = "approval_not_required"
		}
	} else if draft.Readiness.CanApply {
		approvalStatus = "current"
		approvalDetail = "approval_required"
		if approval != nil && approval.Status == domain.PermissionPackageApprovalStatusPending {
			approvalDetail = "approval_pending"
		}
		if approval != nil && approval.Status == domain.PermissionPackageApprovalStatusRejected {
			approvalStatus = "blocked"
			approvalDetail = "approval_rejected"
		}
		if approval != nil && approval.Status == domain.PermissionPackageApprovalStatusWithdrawn {
			approvalDetail = "approval_withdrawn"
		}
	}
	applyStatus := "waiting"
	applyDetail := "apply_waiting"
	if applied {
		applyStatus = "complete"
		applyDetail = "apply_done"
	} else if canApply {
		applyStatus = "current"
		applyDetail = "apply_ready"
	}
	validationStatus := "waiting"
	validationDetail := "validation_waiting"
	if runtimeEvidenceReady {
		validationStatus = "complete"
		validationDetail = "validation_ready"
	} else if applied {
		validationStatus = "current"
		validationDetail = "validation_needed"
	}
	acceptanceStatus := "waiting"
	acceptanceDetail := "acceptance_waiting"
	if productionReady {
		acceptanceStatus = "complete"
		acceptanceDetail = "acceptance_ready"
	} else if applied {
		acceptanceStatus = "current"
		acceptanceDetail = "acceptance_needed"
	}
	return []permissionPackageWorkbenchStep{
		{Key: "request", Status: requestStatus, DetailCode: requestDetail},
		{Key: "approval", Status: approvalStatus, DetailCode: approvalDetail},
		{Key: "apply", Status: applyStatus, DetailCode: applyDetail},
		{Key: "validation", Status: validationStatus, DetailCode: validationDetail},
		{Key: "acceptance", Status: acceptanceStatus, DetailCode: acceptanceDetail},
	}
}

func (s *Server) permissionPackageProductionReadiness(ctx context.Context, query permissionPackageProductionReadinessQuery) (permissionPackageProductionReadinessResponse, error) {
	result := permissionPackageProductionReadinessResponse{
		Checks:      []permissionPackageProductionReadinessCheck{},
		NextActions: []string{},
		GeneratedAt: s.now(),
	}
	applications, err := s.repo.ListPermissionPackageApplications(ctx, store.PermissionPackageApplicationFilter{
		ManagementScope:            store.ManagementScope{TenantID: query.TenantID, WorkspaceID: query.WorkspaceID},
		TemplateID:                 query.TemplateID,
		TargetID:                   query.TargetID,
		CallerInstanceID:           query.CallerInstanceID,
		RequestedCapabilityID:      query.RequestedCapabilityID,
		MatchRequestedCapabilityID: true,
		Limit:                      1,
	})
	if err != nil {
		return permissionPackageProductionReadinessResponse{}, err
	}
	if len(applications) > 0 {
		visibleApplications, err := s.visiblePermissionPackageApplications(ctx, applications, store.ManagementScope{TenantID: query.TenantID, WorkspaceID: query.WorkspaceID})
		if err != nil {
			return permissionPackageProductionReadinessResponse{}, err
		}
		if len(visibleApplications) == 0 {
			applications = nil
		} else {
			applications = permissionPackageApplicationsForRequestedCapability(visibleApplications, query.RequestedCapabilityID)
		}
	}
	legacyExactEquivalent := false
	if len(applications) == 0 && query.RequestedCapabilityID != "" {
		legacyApplications, err := s.repo.ListPermissionPackageApplications(ctx, store.PermissionPackageApplicationFilter{
			ManagementScope:            store.ManagementScope{TenantID: query.TenantID, WorkspaceID: query.WorkspaceID},
			TemplateID:                 query.TemplateID,
			TargetID:                   query.TargetID,
			CallerInstanceID:           query.CallerInstanceID,
			MatchRequestedCapabilityID: true,
		})
		if err != nil {
			return permissionPackageProductionReadinessResponse{}, err
		}
		visibleLegacyApplications, err := s.visiblePermissionPackageApplications(ctx, legacyApplications, store.ManagementScope{TenantID: query.TenantID, WorkspaceID: query.WorkspaceID})
		if err != nil {
			return permissionPackageProductionReadinessResponse{}, err
		}
		for _, application := range visibleLegacyApplications {
			if len(application.AllowedCapabilityIDs) != 1 || application.AllowedCapabilityIDs[0] != query.RequestedCapabilityID {
				continue
			}
			candidateQuery := permissionPackageProductionReadinessQueryWithApplicationDefaults(query, application)
			candidatePreflight, err := s.preflightPermissionPackageRequest(ctx, domain.PermissionPackageApplyRequest{
				PermissionPackageDraftRequest: domain.PermissionPackageDraftRequest{
					CallerInstanceID:      candidateQuery.CallerInstanceID,
					Region:                candidateQuery.Region,
					RequestText:           candidateQuery.RequestText,
					RequestedCapabilityID: candidateQuery.RequestedCapabilityID,
					SubjectSelector:       candidateQuery.SubjectSelector,
					TargetID:              candidateQuery.TargetID,
					TemplateID:            candidateQuery.TemplateID,
					TenantID:              candidateQuery.TenantID,
					WorkspaceID:           candidateQuery.WorkspaceID,
				},
				ApprovalRequestID: candidateQuery.ApprovalRequestID,
			})
			if err != nil {
				return permissionPackageProductionReadinessResponse{}, err
			}
			if permissionPackageProductionLegacyApplicationMatchesExactDraft(candidateQuery, application, candidatePreflight.Draft) {
				applications = []domain.PermissionPackageApplication{application}
				legacyExactEquivalent = true
				break
			}
		}
	}
	if len(applications) > 0 {
		latest := applications[0]
		result.LatestApplication = &latest
		query = permissionPackageProductionReadinessQueryWithApplicationDefaults(query, latest)
	}
	if query.SubjectID != "" {
		if subjectSelectorMatchesRuntime(query.SubjectSelector, query.SubjectID) {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("subject_scope_match", domain.PermissionPackagePreflightPassed, "Production subject is covered by the requested subject selector.", query.SubjectID))
		} else {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("subject_scope_match", domain.PermissionPackagePreflightBlocking, "Production subject is outside the requested subject selector.", query.SubjectID))
			permissionPackageProductionAddNextAction(&result, "review_subject_scope", "Use the application subject selector and a production subject that it covers.")
		}
	}

	preflight, err := s.preflightPermissionPackageRequest(ctx, domain.PermissionPackageApplyRequest{
		PermissionPackageDraftRequest: domain.PermissionPackageDraftRequest{
			CallerInstanceID:      query.CallerInstanceID,
			Region:                query.Region,
			RequestText:           query.RequestText,
			RequestedCapabilityID: query.RequestedCapabilityID,
			SubjectSelector:       query.SubjectSelector,
			TargetID:              query.TargetID,
			TemplateID:            query.TemplateID,
			TenantID:              query.TenantID,
			WorkspaceID:           query.WorkspaceID,
		},
		ApprovalRequestID: query.ApprovalRequestID,
	})
	if err != nil {
		return permissionPackageProductionReadinessResponse{}, err
	}
	result.Preflight = &preflight
	if permissionPackageProductionPreflightReady(preflight, result.LatestApplication) {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("preflight_ready", domain.PermissionPackagePreflightPassed, "Permission package draft and safety preflight are acceptable for production readiness.", ""))
	} else {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("preflight_ready", domain.PermissionPackagePreflightBlocking, "Permission package preflight still has blocking checks.", ""))
		permissionPackageProductionAddNextAction(&result, "resolve_preflight_blockers", "Resolve apply preflight blockers before claiming production readiness.")
	}

	if result.LatestApplication == nil {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_present", domain.PermissionPackagePreflightBlocking, "No permission package application exists for this tenant, workspace, template, target, and caller.", ""))
		permissionPackageProductionAddNextAction(&result, "apply_permission_package", "Apply the approved permission package before production readiness.")
	} else {
		result.Summary.HasApplication = true
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_present", domain.PermissionPackagePreflightPassed, "Permission package application record is present.", result.LatestApplication.ID))
		if permissionPackageProductionApplicationMatchesDraft(query, *result.LatestApplication, preflight.Draft) {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_scope_match", domain.PermissionPackagePreflightPassed, "Latest application matches the requested production scope.", result.LatestApplication.ID))
		} else if legacyExactEquivalent && permissionPackageProductionLegacyApplicationMatchesExactDraft(query, *result.LatestApplication, preflight.Draft) {
			result.Checks = append(result.Checks,
				permissionPackageProductionReadinessCheckFor("application_scope_match", domain.PermissionPackagePreflightPassed, "Legacy application has the same exact single-capability boundary as this request.", result.LatestApplication.ID),
				permissionPackageProductionReadinessCheckFor("legacy_exact_equivalent", domain.PermissionPackagePreflightInfo, "Legacy application provenance is package-level, but its effective boundary is exactly the requested capability.", result.LatestApplication.ID),
			)
		} else {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_scope_match", domain.PermissionPackagePreflightBlocking, "Latest application does not match the requested production scope.", result.LatestApplication.ID))
			permissionPackageProductionAddNextAction(&result, "review_application_scope", "Inspect the latest permission package application scope before go-live.")
		}
		capabilitiesCurrent, err := s.permissionPackageApplicationCapabilitiesCurrent(ctx, *result.LatestApplication)
		if err != nil {
			return permissionPackageProductionReadinessResponse{}, err
		}
		if capabilitiesCurrent {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_capabilities_current", domain.PermissionPackagePreflightPassed, "Applied capabilities have not changed since this permission package was applied.", result.LatestApplication.ID))
		} else {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_capabilities_current", domain.PermissionPackagePreflightBlocking, "One or more applied capabilities changed after this permission package was applied.", result.LatestApplication.ID))
			permissionPackageProductionAddNextAction(&result, "reapply_permission_package", "Review the changed capability contract and apply a fresh permission package approval.")
		}
		impact, err := s.permissionPackageApplicationImpact(ctx, *result.LatestApplication)
		if err != nil {
			return permissionPackageProductionReadinessResponse{}, err
		}
		result.ApplicationImpact = &impact
		healthStatus := permissionPackageApplicationHealthStatus(impact)
		health := permissionPackageApplicationHealthRow{
			Application:        *result.LatestApplication,
			Status:             healthStatus,
			BlockerCodes:       append([]string{}, impact.RollbackReview.BlockerCodes...),
			CreatedObjectCount: impact.Summary.CreatedObjectCount,
			ActiveObjectCount:  impact.Summary.ActiveObjectCount,
			MissingObjectCount: impact.Summary.MissingObjectCount,
			RollbackReady:      impact.Summary.RollbackReady,
		}
		result.ApplicationHealth = &health
		if healthStatus == "ready" {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_health_ready", domain.PermissionPackagePreflightPassed, "Latest permission package application is healthy.", result.LatestApplication.ID))
		} else {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_health_ready", domain.PermissionPackagePreflightBlocking, "Latest permission package application is not healthy.", result.LatestApplication.ID))
			permissionPackageProductionAddNextAction(&result, "review_application_health", "Review application health and drift blockers before production readiness.")
		}
		if impact.RollbackReview.Ready && impact.Summary.MissingObjectCount == 0 {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("impact_ready", domain.PermissionPackagePreflightPassed, "Application impact review shows active created grant objects.", result.LatestApplication.ID))
		} else {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("impact_ready", domain.PermissionPackagePreflightBlocking, "Application impact review has missing or inactive created objects.", result.LatestApplication.ID))
			permissionPackageProductionAddNextAction(&result, "resolve_impact_blockers", "Resolve impact review blockers before production readiness.")
		}
	}

	profile, err := s.buildTenantAccessProfile(ctx, query.TenantID, accessProfileQuery{
		WorkspaceID:      query.WorkspaceID,
		TargetID:         query.TargetID,
		CallerInstanceID: query.CallerInstanceID,
		TraceLimit:       query.TraceLimit,
	})
	if err != nil {
		return permissionPackageProductionReadinessResponse{}, err
	}
	result.AccessProfile = &profile
	accessEvidenceID := permissionPackageProductionAccessProfileEvidenceID(profile, query, result.LatestApplication)
	if accessEvidenceID != "" {
		result.Summary.AccessProfileReady = true
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("access_profile_chain_present", domain.PermissionPackagePreflightPassed, "Tenant access profile contains an effective target, workspace, and caller grant chain.", accessEvidenceID))
	} else {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("access_profile_chain_present", domain.PermissionPackagePreflightBlocking, "Tenant access profile does not contain an effective grant chain for this caller and target.", ""))
		permissionPackageProductionAddNextAction(&result, "verify_access_profile", "Verify tenant entitlement, workspace assignment, and caller assignment records.")
	}

	traces := []domain.TraceEvent{}
	if query.TraceLimit > 0 {
		traces, err = s.repo.ListTraces(ctx, store.TraceFilter{
			ManagementScope: store.ManagementScope{TenantID: query.TenantID, WorkspaceID: query.WorkspaceID},
			CallerID:        query.CallerInstanceID,
			TargetID:        query.TargetID,
			Limit:           query.TraceLimit,
		})
		if err != nil {
			return permissionPackageProductionReadinessResponse{}, err
		}
	}
	allowedEvidenceCapabilityIDs := permissionPackageCapabilityIDSet(preflight.Draft.AllowedCapabilities)
	blockedEvidenceCapabilityIDs := permissionPackageCapabilityIDSet(preflight.Draft.BlockedCapabilities)
	result.RuntimeEvidence.AllowedTrace = permissionPackageProductionLatestTrace(traces, domain.TraceDecisionAllowed, query.SubjectID, allowedEvidenceCapabilityIDs)
	result.RuntimeEvidence.DeniedTrace = permissionPackageProductionLatestTrace(traces, domain.TraceDecisionDenied, query.SubjectID, blockedEvidenceCapabilityIDs)
	if result.RuntimeEvidence.AllowedTrace != nil {
		result.Summary.HasAllowedTrace = true
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("runtime_allowed_trace_present", domain.PermissionPackagePreflightPassed, "Runtime allowed record is present for this caller and target.", result.RuntimeEvidence.AllowedTrace.ID))
	} else {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("runtime_allowed_trace_present", domain.PermissionPackagePreflightBlocking, "Runtime allowed record is missing for this caller and target.", ""))
		permissionPackageProductionAddNextAction(&result, "run_allowed_runtime_call", "Run an allowed MCP call with the production subject before go-live.")
	}
	if result.RuntimeEvidence.DeniedTrace != nil {
		result.Summary.HasDeniedTrace = true
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("runtime_denied_trace_present", domain.PermissionPackagePreflightPassed, "Runtime denied record is present for this caller and target.", result.RuntimeEvidence.DeniedTrace.ID))
	} else if len(blockedEvidenceCapabilityIDs) == 0 {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("runtime_denied_trace_not_applicable", domain.PermissionPackagePreflightPassed, "No blocked catalog capability exists for this exact request, so a denied runtime call is not applicable.", ""))
	} else {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("runtime_denied_trace_present", domain.PermissionPackagePreflightBlocking, "Runtime denied record is missing for this caller and target.", ""))
		permissionPackageProductionAddNextAction(&result, "run_denied_runtime_call", "Run a denied MCP call that proves blocked tools stay blocked.")
	}

	if result.LatestApplication != nil {
		events, err := s.repo.ListAuditEvents(ctx, store.AuditEventFilter{
			ManagementScope: store.ManagementScope{TenantID: query.TenantID, WorkspaceID: query.WorkspaceID},
			Action:          "permission_package.applied",
			ResourceID:      result.LatestApplication.ID,
			Limit:           1,
		})
		if err != nil {
			return permissionPackageProductionReadinessResponse{}, err
		}
		events, err = s.visibleAuditEvents(ctx, events, store.ManagementScope{TenantID: query.TenantID, WorkspaceID: query.WorkspaceID})
		if err != nil {
			return permissionPackageProductionReadinessResponse{}, err
		}
		if len(events) > 0 {
			appliedEvent := events[0]
			result.AuditEvidence.AppliedEvent = &appliedEvent
		}
	}
	if result.AuditEvidence.AppliedEvent != nil {
		result.Summary.HasAppliedAudit = true
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("applied_audit_event_present", domain.PermissionPackagePreflightPassed, "Applied audit event is present for this permission package application.", result.AuditEvidence.AppliedEvent.ID))
	} else {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("applied_audit_event_present", domain.PermissionPackagePreflightBlocking, "Applied audit event is missing for this permission package application.", ""))
		permissionPackageProductionAddNextAction(&result, "verify_applied_audit", "Verify the permission package applied audit record before production readiness.")
	}

	result.Summary = permissionPackageProductionReadinessSummaryFor(result)
	result.Status = permissionPackageProductionReadinessStatus(result.Summary)
	if result.Status == "ready" {
		permissionPackageProductionAddNextAction(&result, "export_acceptance_report", "Production readiness is complete.")
	}
	return result, nil
}

func permissionPackageProductionRuntimeEvidenceReady(readiness *permissionPackageProductionReadinessResponse) bool {
	if readiness == nil || !readiness.Summary.HasAllowedTrace {
		return false
	}
	if readiness.Summary.HasDeniedTrace {
		return true
	}
	for _, check := range readiness.Checks {
		if check.Code == "runtime_denied_trace_not_applicable" && check.Severity == domain.PermissionPackagePreflightPassed {
			return true
		}
	}
	return false
}

func (s *Server) permissionPackageProductionEvidenceReport(ctx context.Context, query permissionPackageProductionReadinessQuery, generatedBy string) (permissionPackageProductionEvidenceReportResponse, error) {
	readiness, err := s.permissionPackageProductionReadiness(ctx, query)
	if err != nil {
		return permissionPackageProductionEvidenceReportResponse{}, err
	}
	return permissionPackageProductionEvidenceReportFromReadiness(query, readiness, generatedBy), nil
}

func permissionPackageProductionEvidenceReportFromReadiness(query permissionPackageProductionReadinessQuery, readiness permissionPackageProductionReadinessResponse, generatedBy string) permissionPackageProductionEvidenceReportResponse {
	scope := permissionPackageProductionEvidenceScope{
		TenantID:              query.TenantID,
		WorkspaceID:           query.WorkspaceID,
		TemplateID:            query.TemplateID,
		TargetID:              query.TargetID,
		CallerInstanceID:      query.CallerInstanceID,
		RequestedCapabilityID: query.RequestedCapabilityID,
		SubjectID:             query.SubjectID,
		Region:                query.Region,
		SubjectSelector:       query.SubjectSelector,
	}
	evidence := permissionPackageProductionEvidenceRefs{
		AccessProfile:     permissionPackageProductionEvidenceState{Present: readiness.Summary.AccessProfileReady},
		ApplicationHealth: permissionPackageProductionEvidenceState{Present: readiness.ApplicationHealth != nil},
		ApplicationImpact: permissionPackageProductionEvidenceState{Present: readiness.ApplicationImpact != nil},
	}
	if readiness.ApplicationHealth != nil {
		evidence.ApplicationHealth.Status = readiness.ApplicationHealth.Status
	}
	if readiness.ApplicationImpact != nil && readiness.ApplicationImpact.Summary.RollbackReady {
		evidence.ApplicationImpact.Status = "ready"
	} else if readiness.ApplicationImpact != nil {
		evidence.ApplicationImpact.Status = "blocked"
	}
	if readiness.LatestApplication != nil {
		application := readiness.LatestApplication
		scope.Region = stringOrDefault(scope.Region, application.Region)
		scope.SubjectSelector = stringOrDefault(scope.SubjectSelector, application.SubjectSelector)
		appliedAt := application.AppliedAt
		evidence.Application = permissionPackageProductionApplicationEvidence{
			Present:               true,
			ID:                    application.ID,
			DraftID:               application.DraftID,
			RequestedCapabilityID: application.RequestedCapabilityID,
			TemplateVersion:       application.TemplateVersion,
			AppliedAt:             &appliedAt,
			AllowedCapabilityIDs:  append([]string(nil), application.AllowedCapabilityIDs...),
			AllowedCapabilityKeys: append([]string(nil), application.AllowedCapabilityKeys...),
			DataScopes:            domain.CloneDataScopes(application.DataScopes),
		}
	}
	if readiness.RuntimeEvidence.AllowedTrace != nil {
		evidence.Runtime.AllowedTraceID = readiness.RuntimeEvidence.AllowedTrace.ID
	}
	if readiness.RuntimeEvidence.DeniedTrace != nil {
		evidence.Runtime.DeniedTraceID = readiness.RuntimeEvidence.DeniedTrace.ID
	}
	if readiness.AuditEvidence.AppliedEvent != nil {
		evidence.Audit.AppliedEventID = readiness.AuditEvidence.AppliedEvent.ID
	}
	toolCatalog := systemInfoManagementMcpToolCatalogSummary()
	report := permissionPackageProductionEvidenceReportResponse{
		ReportVersion:         permissionPackageProductionEvidenceReportVersion,
		ReportDigestAlgorithm: permissionPackageProductionEvidenceReportDigestAlgorithm,
		GeneratedBy:           generatedBy,
		GeneratedAt:           readiness.GeneratedAt,
		PlatformContract: permissionPackageProductionPlatformContract{
			APIVersion: systemAPIVersion,
			ManagementMcpToolCatalog: permissionPackageProductionManagementMcpToolCatalogStamp{
				MetadataVersion: toolCatalog.MetadataVersion,
				CatalogDigest:   toolCatalog.CatalogDigest,
			},
		},
		Scope:                scope,
		Status:               readiness.Status,
		Summary:              readiness.Summary,
		Checks:               append([]permissionPackageProductionReadinessCheck(nil), readiness.Checks...),
		Evidence:             evidence,
		NextActionCode:       readiness.NextActionCode,
		NextActions:          append([]string(nil), readiness.NextActions...),
		ReadinessGeneratedAt: readiness.GeneratedAt,
	}
	report.ReportDigest = permissionPackageProductionEvidenceReportDigest(report)
	return report
}

func permissionPackageProductionEvidenceReportDigest(report permissionPackageProductionEvidenceReportResponse) string {
	report.ReportDigest = ""
	payload, err := json.Marshal(report)
	if err != nil {
		return ""
	}
	var canonical any
	if err := json.Unmarshal(payload, &canonical); err != nil {
		return ""
	}
	canonicalPayload, err := json.Marshal(canonical)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(canonicalPayload)
	return hex.EncodeToString(sum[:])
}

func stringOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func permissionPackageProductionReadinessQueryWithApplicationDefaults(query permissionPackageProductionReadinessQuery, application domain.PermissionPackageApplication) permissionPackageProductionReadinessQuery {
	if query.Region == "" {
		query.Region = application.Region
	}
	if query.RequestText == "" {
		query.RequestText = application.RequestText
	}
	if query.SubjectSelector == "" {
		query.SubjectSelector = application.SubjectSelector
	}
	return query
}

func permissionPackageProductionPreflightReady(preflight domain.PermissionPackageApplyPreflightResponse, latestApplication *domain.PermissionPackageApplication) bool {
	if preflight.Summary.CanApply {
		return true
	}
	if latestApplication == nil {
		return false
	}
	for _, check := range preflight.Checks {
		if check.Severity != domain.PermissionPackagePreflightBlocking {
			continue
		}
		if check.Code == "approval_request_missing" || check.Code == "approval_request_invalid" {
			continue
		}
		if check.Code == "application_already_applied" {
			continue
		}
		return false
	}
	return true
}

func permissionPackageProductionApplicationScopeMatches(query permissionPackageProductionReadinessQuery, application domain.PermissionPackageApplication) bool {
	return application.TenantID == query.TenantID &&
		application.WorkspaceID == query.WorkspaceID &&
		application.TemplateID == query.TemplateID &&
		application.TargetID == query.TargetID &&
		application.CallerInstanceID == query.CallerInstanceID &&
		application.RequestedCapabilityID == query.RequestedCapabilityID &&
		application.SubjectSelector == query.SubjectSelector &&
		application.Region == query.Region
}

func permissionPackageProductionApplicationMatchesDraft(query permissionPackageProductionReadinessQuery, application domain.PermissionPackageApplication, draft domain.PermissionPackageDraft) bool {
	allowedIDs, allowedKeys := permissionPackageCapabilityIDsAndKeys(draft.AllowedCapabilities)
	return permissionPackageProductionApplicationScopeMatches(query, application) &&
		application.RequestedCapabilityID == draft.Input.RequestedCapabilityID &&
		application.SubjectSelector == draft.Input.SubjectSelector &&
		application.Region == draft.Input.Region &&
		application.TemplateVersion == draft.Template.Version &&
		samePermissionPackageDataScopes(application.DataScopes, draft.DataScopes) &&
		sameStringSet(application.AllowedCapabilityIDs, allowedIDs) &&
		sameStringSet(application.AllowedCapabilityKeys, allowedKeys)
}

func permissionPackageProductionLegacyApplicationMatchesExactDraft(query permissionPackageProductionReadinessQuery, application domain.PermissionPackageApplication, draft domain.PermissionPackageDraft) bool {
	if query.RequestedCapabilityID == "" || application.RequestedCapabilityID != "" || draft.Input.RequestedCapabilityID != query.RequestedCapabilityID {
		return false
	}
	allowedIDs, allowedKeys := permissionPackageCapabilityIDsAndKeys(draft.AllowedCapabilities)
	return application.TenantID == query.TenantID &&
		application.WorkspaceID == query.WorkspaceID &&
		application.TemplateID == query.TemplateID &&
		application.TargetID == query.TargetID &&
		application.CallerInstanceID == query.CallerInstanceID &&
		application.SubjectSelector == query.SubjectSelector &&
		application.Region == query.Region &&
		application.TemplateVersion == draft.Template.Version &&
		samePermissionPackageDataScopes(application.DataScopes, draft.DataScopes) &&
		sameStringSet(application.AllowedCapabilityIDs, allowedIDs) &&
		sameStringSet(application.AllowedCapabilityKeys, allowedKeys)
}

func (s *Server) permissionPackageApplicationCapabilitiesCurrent(ctx context.Context, application domain.PermissionPackageApplication) (bool, error) {
	if application.AppliedAt.IsZero() || len(application.AllowedCapabilityIDs) == 0 {
		return false, nil
	}
	for _, capabilityID := range application.AllowedCapabilityIDs {
		capability, ok, err := s.repo.GetCapability(ctx, capabilityID)
		if err != nil {
			return false, err
		}
		if !ok || capability.TargetID != application.TargetID || capability.UpdatedAt.After(application.AppliedAt) {
			return false, nil
		}
	}
	return true, nil
}

func permissionPackageProductionAccessProfileEvidenceID(profile tenantAccessProfileResponse, query permissionPackageProductionReadinessQuery, application *domain.PermissionPackageApplication) string {
	allowedCapabilityIDs := map[string]struct{}{}
	if query.RequestedCapabilityID != "" {
		allowedCapabilityIDs[query.RequestedCapabilityID] = struct{}{}
	} else if application != nil {
		for _, capabilityID := range application.AllowedCapabilityIDs {
			allowedCapabilityIDs[capabilityID] = struct{}{}
		}
	}
	for _, grant := range profile.Grants {
		if grant.ScopeStatus != accessProfileScopeValid || grant.Target == nil || grant.Target.ID != query.TargetID || grant.Capability == nil {
			continue
		}
		if len(allowedCapabilityIDs) > 0 {
			if _, ok := allowedCapabilityIDs[grant.Capability.ID]; !ok {
				continue
			}
		}
		for _, workspace := range grant.WorkspaceAssignments {
			if workspace.ScopeStatus != accessProfileScopeValid || workspace.WorkspaceAssignment.WorkspaceID != query.WorkspaceID {
				continue
			}
			for _, instance := range workspace.InstanceAssignments {
				if instance.ScopeStatus == accessProfileScopeValid && instance.InstanceAssignment.CallerInstanceID == query.CallerInstanceID {
					return instance.InstanceAssignment.ID
				}
			}
		}
	}
	return ""
}

func permissionPackageCapabilityIDSet(capabilities []domain.Capability) map[string]struct{} {
	result := make(map[string]struct{}, len(capabilities))
	for _, capability := range capabilities {
		if capability.ID != "" {
			result[capability.ID] = struct{}{}
		}
	}
	return result
}

func permissionPackageProductionLatestTrace(traces []domain.TraceEvent, decision domain.TraceDecision, subjectID string, capabilityIDs map[string]struct{}) *domain.TraceEvent {
	if len(capabilityIDs) == 0 {
		return nil
	}
	for index := len(traces) - 1; index >= 0; index-- {
		trace := traces[index]
		if trace.Decision != decision {
			continue
		}
		if subjectID != "" && trace.SubjectID != subjectID {
			continue
		}
		if _, ok := capabilityIDs[trace.CapabilityID]; !ok {
			continue
		}
		return &trace
	}
	return nil
}

func permissionPackageProductionReadinessCheckFor(code string, severity domain.PermissionPackagePreflightSeverity, message string, evidenceID string) permissionPackageProductionReadinessCheck {
	return permissionPackageProductionReadinessCheck{
		Code:       code,
		Severity:   severity,
		Message:    message,
		EvidenceID: evidenceID,
	}
}

func permissionPackageProductionAddNextAction(result *permissionPackageProductionReadinessResponse, code string, message string) {
	if result.NextActionCode == "" {
		result.NextActionCode = code
	}
	result.NextActions = appendUniqueString(result.NextActions, message)
}

func permissionPackageProductionReadinessSummaryFor(result permissionPackageProductionReadinessResponse) permissionPackageProductionReadinessSummary {
	summary := result.Summary
	for _, check := range result.Checks {
		switch check.Severity {
		case domain.PermissionPackagePreflightPassed:
			summary.ReadyCount++
		case domain.PermissionPackagePreflightWarning:
			summary.WarningCount++
		case domain.PermissionPackagePreflightBlocking:
			summary.BlockingCount++
		}
	}
	return summary
}

func permissionPackageProductionReadinessStatus(summary permissionPackageProductionReadinessSummary) string {
	if summary.BlockingCount > 0 {
		return "blocked"
	}
	if summary.WarningCount > 0 {
		return "needs_review"
	}
	return "ready"
}
