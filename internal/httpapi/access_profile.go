package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const (
	defaultAccessProfileTraceLimit = 20
	maxAccessProfileTraceLimit     = 100

	accessProfileScopeValid   = "valid"
	accessProfileScopeInvalid = "invalid"
)

type accessProfileQuery struct {
	WorkspaceID      string
	TargetID         string
	CapabilityID     string
	CallerInstanceID string
	TraceLimit       int
}

type tenantAccessProfileResponse struct {
	Tenant       domain.Tenant              `json:"tenant"`
	ScopeTenants []domain.Tenant            `json:"scopeTenants"`
	Summary      tenantAccessProfileSummary `json:"summary"`
	Grants       []tenantAccessProfileGrant `json:"grants"`
	RecentTraces []domain.TraceEvent        `json:"recentTraces"`
	GeneratedAt  time.Time                  `json:"generatedAt"`
}

type tenantAccessProfileSummary struct {
	TenantCount              int `json:"tenantCount"`
	GrantCount               int `json:"grantCount"`
	TargetCount              int `json:"targetCount"`
	CapabilityCount          int `json:"capabilityCount"`
	WorkspaceAssignmentCount int `json:"workspaceAssignmentCount"`
	InstanceAssignmentCount  int `json:"instanceAssignmentCount"`
	RecentAllowedTraceCount  int `json:"recentAllowedTraceCount"`
	RecentDeniedTraceCount   int `json:"recentDeniedTraceCount"`
}

type tenantAccessProfileGrant struct {
	TenantEntitlement         domain.TenantEntitlement       `json:"tenantEntitlement"`
	Target                    *domain.Agent                  `json:"target,omitempty"`
	Capability                *domain.Capability             `json:"capability,omitempty"`
	EffectiveTenantDataScopes []domain.DataScope             `json:"effectiveTenantDataScopes,omitempty"`
	ScopeStatus               string                         `json:"scopeStatus"`
	ScopeReason               string                         `json:"scopeReason,omitempty"`
	WorkspaceAssignments      []tenantAccessProfileWorkspace `json:"workspaceAssignments"`
}

type tenantAccessProfileWorkspace struct {
	WorkspaceAssignment          domain.WorkspaceAssignment    `json:"workspaceAssignment"`
	EffectiveWorkspaceDataScopes []domain.DataScope            `json:"effectiveWorkspaceDataScopes,omitempty"`
	ScopeStatus                  string                        `json:"scopeStatus"`
	ScopeReason                  string                        `json:"scopeReason,omitempty"`
	InstanceAssignments          []tenantAccessProfileInstance `json:"instanceAssignments"`
}

type tenantAccessProfileInstance struct {
	InstanceAssignment          domain.InstanceAssignment `json:"instanceAssignment"`
	CallerInstance              *domain.Agent             `json:"callerInstance,omitempty"`
	EffectiveInstanceDataScopes []domain.DataScope        `json:"effectiveInstanceDataScopes,omitempty"`
	ScopeStatus                 string                    `json:"scopeStatus"`
	ScopeReason                 string                    `json:"scopeReason,omitempty"`
}

func (s *Server) getTenantAccessProfile(w http.ResponseWriter, r *http.Request) {
	query, err := accessProfileQueryFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	profile, err := s.buildTenantAccessProfileForRequest(r, strings.TrimSpace(chi.URLParam(r, "id")), query)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func accessProfileQueryFromRequest(r *http.Request) (accessProfileQuery, error) {
	values := r.URL.Query()
	query := accessProfileQuery{
		WorkspaceID:      strings.TrimSpace(values.Get("workspaceId")),
		TargetID:         strings.TrimSpace(values.Get("targetId")),
		CapabilityID:     strings.TrimSpace(values.Get("capabilityId")),
		CallerInstanceID: strings.TrimSpace(values.Get("callerInstanceId")),
		TraceLimit:       defaultAccessProfileTraceLimit,
	}
	if raw := strings.TrimSpace(values.Get("traceLimit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 0 || limit > maxAccessProfileTraceLimit {
			return accessProfileQuery{}, domain.BadRequest("VALIDATION_FAILED", "traceLimit must be between 0 and 100")
		}
		query.TraceLimit = limit
	}
	return query, nil
}

func (s *Server) buildTenantAccessProfileForRequest(r *http.Request, tenantID string, query accessProfileQuery) (tenantAccessProfileResponse, error) {
	tenantID = strings.TrimSpace(tenantID)
	requested := store.ManagementScope{
		TenantID:    tenantID,
		WorkspaceID: strings.TrimSpace(query.WorkspaceID),
	}
	effective, err := s.effectiveManagementScopeForRequest(r, requested)
	if err != nil {
		return tenantAccessProfileResponse{}, err
	}
	if tenantID != "" && effective.TenantID != tenantID {
		return tenantAccessProfileResponse{}, domain.PermissionDenied("resource tenant is outside authenticated admin scope")
	}
	query.WorkspaceID = effective.WorkspaceID
	return s.buildTenantAccessProfile(r.Context(), tenantID, query)
}

func (s *Server) buildTenantAccessProfile(ctx context.Context, tenantID string, query accessProfileQuery) (tenantAccessProfileResponse, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return tenantAccessProfileResponse{}, domain.BadRequest("VALIDATION_FAILED", "tenant id is required")
	}
	generatedAt := s.now()
	tenant, scopeTenants, err := s.accessProfileTenants(ctx, tenantID, generatedAt)
	if err != nil {
		return tenantAccessProfileResponse{}, err
	}
	scope := store.ManagementScope{TenantID: tenantID}
	entitlements, err := s.repo.ListTenantEntitlements(ctx, store.EntitlementFilter{
		ManagementScope: scope,
		TargetID:        query.TargetID,
		CapabilityID:    query.CapabilityID,
	})
	if err != nil {
		return tenantAccessProfileResponse{}, err
	}
	workspaceAssignments, err := s.repo.ListWorkspaceAssignments(ctx, store.AssignmentFilter{
		ManagementScope: store.ManagementScope{TenantID: tenantID, WorkspaceID: query.WorkspaceID},
	})
	if err != nil {
		return tenantAccessProfileResponse{}, err
	}
	instanceAssignments, err := s.repo.ListInstanceAssignments(ctx, store.InstanceAssignmentFilter{
		ManagementScope:  store.ManagementScope{TenantID: tenantID, WorkspaceID: query.WorkspaceID},
		CallerInstanceID: query.CallerInstanceID,
		CapabilityID:     query.CapabilityID,
	})
	if err != nil {
		return tenantAccessProfileResponse{}, err
	}
	recentTraces, err := s.accessProfileRecentTraces(ctx, tenantID, query)
	if err != nil {
		return tenantAccessProfileResponse{}, err
	}

	workspaceByEntitlement := map[string][]domain.WorkspaceAssignment{}
	for _, assignment := range workspaceAssignments {
		workspaceByEntitlement[assignment.TenantEntitlementID] = append(workspaceByEntitlement[assignment.TenantEntitlementID], assignment)
	}
	instanceByWorkspace := map[string][]domain.InstanceAssignment{}
	for _, assignment := range instanceAssignments {
		instanceByWorkspace[assignment.WorkspaceAssignmentID] = append(instanceByWorkspace[assignment.WorkspaceAssignmentID], assignment)
	}

	agentCache := map[string]*domain.Agent{}
	capabilityCache := map[string]*domain.Capability{}
	grants := make([]tenantAccessProfileGrant, 0, len(entitlements))
	for _, entitlement := range entitlements {
		grant, err := s.accessProfileGrant(ctx, entitlement, workspaceByEntitlement[entitlement.ID], instanceByWorkspace, agentCache, capabilityCache, query)
		if err != nil {
			return tenantAccessProfileResponse{}, err
		}
		if shouldSkipAccessProfileGrant(grant, query) {
			continue
		}
		grants = append(grants, grant)
	}

	profile := tenantAccessProfileResponse{
		Tenant:       tenant,
		ScopeTenants: scopeTenants,
		Grants:       grants,
		RecentTraces: recentTraces,
		GeneratedAt:  generatedAt,
	}
	profile.Summary = summarizeTenantAccessProfile(profile)
	return profile, nil
}

func (s *Server) accessProfileTenants(ctx context.Context, tenantID string, generatedAt time.Time) (domain.Tenant, []domain.Tenant, error) {
	tenant, ok, err := s.repo.GetTenant(ctx, tenantID)
	if err != nil {
		return domain.Tenant{}, nil, err
	}
	if !ok {
		flat := domain.Tenant{
			ID:        tenantID,
			Level:     0,
			Name:      tenantID,
			Status:    domain.TenantStatusActive,
			CreatedAt: generatedAt,
			UpdatedAt: generatedAt,
		}
		return flat, []domain.Tenant{flat}, nil
	}
	scopeTenants, err := s.repo.ListTenants(ctx, store.TenantFilter{TenantID: tenantID})
	if err != nil {
		return domain.Tenant{}, nil, err
	}
	if len(scopeTenants) == 0 {
		scopeTenants = []domain.Tenant{tenant}
	}
	return tenant, scopeTenants, nil
}

func (s *Server) accessProfileGrant(
	ctx context.Context,
	entitlement domain.TenantEntitlement,
	workspaceAssignments []domain.WorkspaceAssignment,
	instanceByWorkspace map[string][]domain.InstanceAssignment,
	agentCache map[string]*domain.Agent,
	capabilityCache map[string]*domain.Capability,
	query accessProfileQuery,
) (tenantAccessProfileGrant, error) {
	target, err := s.accessProfileAgent(ctx, entitlement.TargetID, agentCache)
	if err != nil {
		return tenantAccessProfileGrant{}, err
	}
	capability, err := s.accessProfileCapability(ctx, entitlement.CapabilityID, capabilityCache)
	if err != nil {
		return tenantAccessProfileGrant{}, err
	}
	if target != nil {
		allowedTenant, err := s.tenantCanReceiveTargetEntitlement(ctx, target.TenantID, entitlement.TenantID)
		if err != nil {
			return tenantAccessProfileGrant{}, err
		}
		if !allowedTenant || (query.WorkspaceID != "" && target.WorkspaceID != query.WorkspaceID) {
			target = nil
			capability = nil
		}
	}
	grant := tenantAccessProfileGrant{
		TenantEntitlement:    entitlement,
		Target:               target,
		Capability:           capability,
		ScopeStatus:          accessProfileScopeValid,
		WorkspaceAssignments: []tenantAccessProfileWorkspace{},
	}
	if target == nil {
		markAccessProfileGrantInvalid(&grant, "target not found")
	}
	if capability == nil {
		markAccessProfileGrantInvalid(&grant, "capability not found")
	} else if capability.TargetID != entitlement.TargetID {
		markAccessProfileGrantInvalid(&grant, "capability target mismatch")
	} else {
		effective, ok := domain.EffectiveDataScopes(capability.DataScopes, entitlement.DataScopes)
		if !ok {
			markAccessProfileGrantInvalid(&grant, "tenant entitlement dataScopes exceed capability dataScopes")
		} else {
			grant.EffectiveTenantDataScopes = effective
		}
	}
	for _, assignment := range workspaceAssignments {
		workspace := s.accessProfileWorkspace(ctx, grant, assignment, instanceByWorkspace[assignment.ID], agentCache, query)
		if query.CallerInstanceID != "" && len(workspace.InstanceAssignments) == 0 {
			continue
		}
		grant.WorkspaceAssignments = append(grant.WorkspaceAssignments, workspace)
	}
	return grant, nil
}

func (s *Server) accessProfileWorkspace(
	ctx context.Context,
	grant tenantAccessProfileGrant,
	assignment domain.WorkspaceAssignment,
	instanceAssignments []domain.InstanceAssignment,
	agentCache map[string]*domain.Agent,
	query accessProfileQuery,
) tenantAccessProfileWorkspace {
	workspace := tenantAccessProfileWorkspace{
		WorkspaceAssignment: assignment,
		ScopeStatus:         accessProfileScopeValid,
		InstanceAssignments: []tenantAccessProfileInstance{},
	}
	if grant.ScopeStatus != accessProfileScopeValid {
		markAccessProfileWorkspaceInvalid(&workspace, "tenant entitlement scope is invalid")
	} else if effective, ok := domain.EffectiveDataScopes(grant.EffectiveTenantDataScopes, assignment.DataScopes); ok {
		workspace.EffectiveWorkspaceDataScopes = effective
	} else {
		markAccessProfileWorkspaceInvalid(&workspace, "workspace assignment dataScopes exceed tenant entitlement dataScopes")
	}
	for _, assignment := range instanceAssignments {
		instance, err := s.accessProfileInstance(ctx, workspace, assignment, agentCache)
		if err != nil {
			markAccessProfileInstanceInvalid(&instance, "caller instance lookup failed")
		}
		workspace.InstanceAssignments = append(workspace.InstanceAssignments, instance)
	}
	if query.CallerInstanceID != "" && len(workspace.InstanceAssignments) == 0 {
		return workspace
	}
	return workspace
}

func (s *Server) accessProfileInstance(
	ctx context.Context,
	workspace tenantAccessProfileWorkspace,
	assignment domain.InstanceAssignment,
	agentCache map[string]*domain.Agent,
) (tenantAccessProfileInstance, error) {
	caller, err := s.accessProfileAgent(ctx, assignment.CallerInstanceID, agentCache)
	if err != nil {
		return tenantAccessProfileInstance{InstanceAssignment: assignment, ScopeStatus: accessProfileScopeInvalid, ScopeReason: "caller instance lookup failed"}, err
	}
	instance := tenantAccessProfileInstance{
		InstanceAssignment: assignment,
		CallerInstance:     caller,
		ScopeStatus:        accessProfileScopeValid,
	}
	if caller == nil {
		markAccessProfileInstanceInvalid(&instance, "caller instance not found")
	} else if caller.TenantID != assignment.TenantID || caller.WorkspaceID != assignment.WorkspaceID {
		markAccessProfileInstanceInvalid(&instance, "caller instance tenant or workspace mismatch")
	}
	if workspace.ScopeStatus != accessProfileScopeValid {
		markAccessProfileInstanceInvalid(&instance, "workspace assignment scope is invalid")
	} else if effective, ok := domain.EffectiveDataScopes(workspace.EffectiveWorkspaceDataScopes, assignment.DataScopes); ok {
		instance.EffectiveInstanceDataScopes = effective
	} else {
		markAccessProfileInstanceInvalid(&instance, "instance assignment dataScopes exceed workspace assignment dataScopes")
	}
	return instance, nil
}

func (s *Server) accessProfileAgent(ctx context.Context, id string, cache map[string]*domain.Agent) (*domain.Agent, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil
	}
	if agent, ok := cache[id]; ok {
		return agent, nil
	}
	agent, ok, err := s.repo.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		cache[id] = nil
		return nil, nil
	}
	cache[id] = &agent
	return &agent, nil
}

func (s *Server) accessProfileCapability(ctx context.Context, id string, cache map[string]*domain.Capability) (*domain.Capability, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil
	}
	if capability, ok := cache[id]; ok {
		return capability, nil
	}
	capability, ok, err := s.repo.GetCapability(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		cache[id] = nil
		return nil, nil
	}
	cache[id] = &capability
	return &capability, nil
}

func (s *Server) accessProfileRecentTraces(ctx context.Context, tenantID string, query accessProfileQuery) ([]domain.TraceEvent, error) {
	if query.TraceLimit == 0 {
		return []domain.TraceEvent{}, nil
	}
	traces, err := s.repo.ListTraces(ctx, store.TraceFilter{
		ManagementScope: store.ManagementScope{TenantID: tenantID, WorkspaceID: query.WorkspaceID},
		CallerID:        query.CallerInstanceID,
		TargetID:        query.TargetID,
	})
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.TraceEvent, 0, len(traces))
	for _, trace := range traces {
		if query.CapabilityID != "" && trace.CapabilityID != query.CapabilityID {
			continue
		}
		filtered = append(filtered, trace)
	}
	start := len(filtered) - query.TraceLimit
	if start < 0 {
		start = 0
	}
	recent := append([]domain.TraceEvent(nil), filtered[start:]...)
	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}
	return recent, nil
}

func shouldSkipAccessProfileGrant(grant tenantAccessProfileGrant, query accessProfileQuery) bool {
	if query.WorkspaceID == "" && query.CallerInstanceID == "" {
		return false
	}
	return len(grant.WorkspaceAssignments) == 0
}

func summarizeTenantAccessProfile(profile tenantAccessProfileResponse) tenantAccessProfileSummary {
	targetIDs := map[string]struct{}{}
	capabilityIDs := map[string]struct{}{}
	workspaceAssignments := 0
	instanceAssignments := 0
	for _, grant := range profile.Grants {
		if grant.Target != nil {
			targetIDs[grant.Target.ID] = struct{}{}
		}
		if grant.Capability != nil {
			capabilityIDs[grant.Capability.ID] = struct{}{}
		}
		workspaceAssignments += len(grant.WorkspaceAssignments)
		for _, workspace := range grant.WorkspaceAssignments {
			instanceAssignments += len(workspace.InstanceAssignments)
		}
	}
	summary := tenantAccessProfileSummary{
		TenantCount:              len(profile.ScopeTenants),
		GrantCount:               len(profile.Grants),
		TargetCount:              len(targetIDs),
		CapabilityCount:          len(capabilityIDs),
		WorkspaceAssignmentCount: workspaceAssignments,
		InstanceAssignmentCount:  instanceAssignments,
	}
	for _, trace := range profile.RecentTraces {
		switch trace.Decision {
		case domain.TraceDecisionAllowed:
			summary.RecentAllowedTraceCount++
		case domain.TraceDecisionDenied:
			summary.RecentDeniedTraceCount++
		}
	}
	return summary
}

func markAccessProfileGrantInvalid(grant *tenantAccessProfileGrant, reason string) {
	grant.ScopeStatus = accessProfileScopeInvalid
	if grant.ScopeReason == "" {
		grant.ScopeReason = reason
	}
}

func markAccessProfileWorkspaceInvalid(workspace *tenantAccessProfileWorkspace, reason string) {
	workspace.ScopeStatus = accessProfileScopeInvalid
	if workspace.ScopeReason == "" {
		workspace.ScopeReason = reason
	}
}

func markAccessProfileInstanceInvalid(instance *tenantAccessProfileInstance, reason string) {
	instance.ScopeStatus = accessProfileScopeInvalid
	if instance.ScopeReason == "" {
		instance.ScopeReason = reason
	}
}
