package store

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

type Repository interface {
	CreateTenant(context.Context, domain.Tenant) (domain.Tenant, error)
	CreateTenantWithAudit(context.Context, domain.Tenant, TenantAuditBuilder) (domain.Tenant, error)
	ListTenants(context.Context, TenantFilter) ([]domain.Tenant, error)
	GetTenant(context.Context, string) (domain.Tenant, bool, error)
	CreateAgent(context.Context, domain.Agent) (domain.Agent, error)
	CreateAgentWithAudit(context.Context, domain.Agent, AgentAuditBuilder) (domain.Agent, error)
	ListAgents(context.Context, AgentFilter) ([]domain.Agent, error)
	GetAgent(context.Context, string) (domain.Agent, bool, error)
	UpdateAgent(context.Context, domain.Agent) (domain.Agent, bool, error)
	UpdateAgentWithAudit(context.Context, domain.Agent, AgentAuditBuilder) (domain.Agent, bool, error)
	RotateAgentCredentials(context.Context, string, map[string]string, time.Time) (domain.Agent, bool, error)
	RotateAgentCredentialsWithAudit(context.Context, string, map[string]string, time.Time, AgentAuditBuilder) (domain.Agent, bool, error)
	DisableAgent(context.Context, string, time.Time) (domain.Agent, bool, error)
	DisableAgentWithAudit(context.Context, string, time.Time, AgentAuditBuilder) (domain.Agent, bool, error)
	CreateAgentKey(context.Context, domain.AgentKey) (domain.AgentKey, error)
	CreateAgentKeyWithAudit(context.Context, domain.AgentKey, AgentKeyAuditBuilder) (domain.AgentKey, error)
	ListAgentKeys(context.Context, ManagementScope) ([]domain.AgentKey, error)
	RevokeAgentKey(context.Context, string, time.Time) (domain.AgentKey, bool, error)
	RevokeAgentKeyWithAudit(context.Context, string, time.Time, AgentKeyAuditBuilder) (domain.AgentKey, bool, error)
	FindAgentByKeyHash(context.Context, string, time.Time) (domain.Agent, bool, error)
	CreateAccessGrant(context.Context, domain.AccessGrant) (domain.AccessGrant, error)
	CreateAccessGrantWithAudit(context.Context, domain.AccessGrant, AccessGrantAuditBuilder) (domain.AccessGrant, error)
	ListAccessGrants(context.Context, ManagementScope) ([]domain.AccessGrant, error)
	RevokeAccessGrant(context.Context, string, time.Time) (domain.AccessGrant, bool, error)
	RevokeAccessGrantWithAudit(context.Context, string, time.Time, AccessGrantAuditBuilder) (domain.AccessGrant, bool, error)
	HasGrant(context.Context, string, string, string, string, time.Time) bool
	UpsertCapability(context.Context, domain.Capability) (domain.Capability, error)
	ListCapabilities(context.Context, CapabilityFilter) ([]domain.Capability, error)
	GetCapability(context.Context, string) (domain.Capability, bool, error)
	UpdateCapability(context.Context, domain.Capability) (domain.Capability, bool, error)
	CreateTenantEntitlement(context.Context, domain.TenantEntitlement) (domain.TenantEntitlement, error)
	ListTenantEntitlements(context.Context, EntitlementFilter) ([]domain.TenantEntitlement, error)
	CreateWorkspaceAssignment(context.Context, domain.WorkspaceAssignment) (domain.WorkspaceAssignment, error)
	ListWorkspaceAssignments(context.Context, AssignmentFilter) ([]domain.WorkspaceAssignment, error)
	CreateInstanceAssignment(context.Context, domain.InstanceAssignment) (domain.InstanceAssignment, error)
	ListInstanceAssignments(context.Context, InstanceAssignmentFilter) ([]domain.InstanceAssignment, error)
	CreatePermissionPackageApplication(context.Context, domain.PermissionPackageApplication) (domain.PermissionPackageApplication, error)
	ListPermissionPackageApplications(context.Context, PermissionPackageApplicationFilter) ([]domain.PermissionPackageApplication, error)
	CreatePermissionPackageApprovalRequest(context.Context, domain.PermissionPackageApprovalRequest) (domain.PermissionPackageApprovalRequest, error)
	ListPermissionPackageApprovalRequests(context.Context, PermissionPackageApprovalRequestFilter) ([]domain.PermissionPackageApprovalRequest, error)
	GetPermissionPackageApprovalRequest(context.Context, string) (domain.PermissionPackageApprovalRequest, bool, error)
	UpdatePermissionPackageApprovalRequest(context.Context, domain.PermissionPackageApprovalRequest) (domain.PermissionPackageApprovalRequest, bool, error)
	EvaluateCapabilityAccess(context.Context, CapabilityAccessRequest) (domain.CapabilityAccessDecision, error)
	CreateRoutePolicy(context.Context, domain.RoutePolicy) (domain.RoutePolicy, error)
	CreateRoutePolicyWithAudit(context.Context, domain.RoutePolicy, RoutePolicyAuditBuilder) (domain.RoutePolicy, error)
	ListRoutePolicies(context.Context, ManagementScope) ([]domain.RoutePolicy, error)
	GetRoutePolicy(context.Context, string) (domain.RoutePolicy, bool, error)
	UpdateRoutePolicy(context.Context, domain.RoutePolicy) (domain.RoutePolicy, bool, error)
	UpdateRoutePolicyWithAudit(context.Context, domain.RoutePolicy, RoutePolicyAuditBuilder) (domain.RoutePolicy, bool, error)
	DisableRoutePolicy(context.Context, string, time.Time) (domain.RoutePolicy, bool, error)
	DisableRoutePolicyWithAudit(context.Context, string, time.Time, RoutePolicyAuditBuilder) (domain.RoutePolicy, bool, error)
	EvaluateRouteAccess(context.Context, string, string, string, string, time.Time) (domain.RouteAccessDecision, error)
	AppendTrace(context.Context, domain.TraceEvent) (domain.TraceEvent, error)
	ListTraces(context.Context, TraceFilter) ([]domain.TraceEvent, error)
	AppendAuditEvent(context.Context, domain.AuditEvent) (domain.AuditEvent, error)
	ListAuditEvents(context.Context, AuditEventFilter) ([]domain.AuditEvent, error)
}

type AgentAuditBuilder func(domain.Agent) domain.AuditEvent
type AgentKeyAuditBuilder func(domain.AgentKey) domain.AuditEvent
type AccessGrantAuditBuilder func(domain.AccessGrant) domain.AuditEvent
type RoutePolicyAuditBuilder func(domain.RoutePolicy) domain.AuditEvent
type TenantAuditBuilder func(domain.Tenant) domain.AuditEvent

type ManagementScope struct {
	TenantID    string
	WorkspaceID string
}

type TenantFilter struct {
	TenantID       string
	ParentTenantID string
}

type AgentFilter struct {
	ManagementScope
}

type TraceFilter struct {
	ManagementScope
	RunID    string
	Decision domain.TraceDecision
	CallerID string
	TargetID string
}

type AuditEventFilter struct {
	ManagementScope
	Action       string
	ResourceType string
	ResourceID   string
	Limit        int
}

type CapabilityFilter struct {
	ManagementScope
	TargetID string
	Status   domain.CapabilityDiscoveryStatus
}

type EntitlementFilter struct {
	ManagementScope
	TargetID     string
	CapabilityID string
}

type AssignmentFilter struct {
	ManagementScope
	EntitlementID string
}

type InstanceAssignmentFilter struct {
	ManagementScope
	CallerInstanceID string
	CapabilityID     string
}

type PermissionPackageApplicationFilter struct {
	ManagementScope
	TemplateID       string
	TargetID         string
	CallerInstanceID string
	Limit            int
}

type PermissionPackageApprovalRequestFilter struct {
	ManagementScope
	TemplateID       string
	TargetID         string
	CallerInstanceID string
	Status           domain.PermissionPackageApprovalStatus
	Limit            int
}

type CapabilityAccessRequest struct {
	TenantID         string
	WorkspaceID      string
	CallerInstanceID string
	SubjectID        string
	TargetID         string
	CapabilityID     string
	Now              time.Time
}

type Memory struct {
	mu                   sync.RWMutex
	tenants              map[string]domain.Tenant
	agents               map[string]domain.Agent
	keys                 map[string]domain.AgentKey
	grants               map[string]domain.AccessGrant
	capabilities         map[string]domain.Capability
	entitlements         map[string]domain.TenantEntitlement
	workspaceAssignments map[string]domain.WorkspaceAssignment
	instanceAssignments  map[string]domain.InstanceAssignment
	packageApplications  map[string]domain.PermissionPackageApplication
	packageApprovals     map[string]domain.PermissionPackageApprovalRequest
	policies             map[string]domain.RoutePolicy
	traces               []domain.TraceEvent
	audits               []domain.AuditEvent
}

func NewMemory() *Memory {
	return &Memory{
		tenants:              make(map[string]domain.Tenant),
		agents:               make(map[string]domain.Agent),
		keys:                 make(map[string]domain.AgentKey),
		grants:               make(map[string]domain.AccessGrant),
		capabilities:         make(map[string]domain.Capability),
		entitlements:         make(map[string]domain.TenantEntitlement),
		workspaceAssignments: make(map[string]domain.WorkspaceAssignment),
		instanceAssignments:  make(map[string]domain.InstanceAssignment),
		packageApplications:  make(map[string]domain.PermissionPackageApplication),
		packageApprovals:     make(map[string]domain.PermissionPackageApprovalRequest),
		policies:             make(map[string]domain.RoutePolicy),
	}
}

func (m *Memory) CreateTenant(_ context.Context, tenant domain.Tenant) (domain.Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenants[tenant.ID] = tenant
	return tenant, nil
}

func (m *Memory) CreateTenantWithAudit(_ context.Context, tenant domain.Tenant, build TenantAuditBuilder) (domain.Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenants[tenant.ID] = tenant
	m.audits = append(m.audits, build(tenant))
	return tenant, nil
}

func (m *Memory) ListTenants(_ context.Context, filter TenantFilter) ([]domain.Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(filter.TenantID)
	parentTenantID := strings.TrimSpace(filter.ParentTenantID)
	rows := make([]domain.Tenant, 0, len(m.tenants))
	for _, tenant := range m.tenants {
		if tenantIDs != nil {
			if _, ok := tenantIDs[tenant.ID]; !ok {
				continue
			}
		}
		if parentTenantID != "" && tenant.ParentTenantID != parentTenantID {
			continue
		}
		rows = append(rows, tenant)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Level != rows[j].Level {
			return rows[i].Level < rows[j].Level
		}
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	return rows, nil
}

func (m *Memory) GetTenant(_ context.Context, id string) (domain.Tenant, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenant, ok := m.tenants[id]
	return tenant, ok, nil
}

func (m *Memory) tenantIDsForScopeLocked(tenantID string) map[string]struct{} {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil
	}
	if _, ok := m.tenants[tenantID]; !ok {
		return map[string]struct{}{tenantID: {}}
	}
	ids := map[string]struct{}{tenantID: {}}
	for changed := true; changed; {
		changed = false
		for id, tenant := range m.tenants {
			if _, exists := ids[id]; exists {
				continue
			}
			if _, parentIncluded := ids[tenant.ParentTenantID]; parentIncluded {
				ids[id] = struct{}{}
				changed = true
			}
		}
	}
	return ids
}

func (m *Memory) CreateAgent(_ context.Context, agent domain.Agent) (domain.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents[agent.ID] = agent
	return agent, nil
}

func (m *Memory) CreateAgentWithAudit(_ context.Context, agent domain.Agent, build AgentAuditBuilder) (domain.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents[agent.ID] = agent
	m.audits = append(m.audits, build(agent))
	return agent, nil
}

func (m *Memory) ListAgents(_ context.Context, filter AgentFilter) ([]domain.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenantIDs := m.tenantIDsForScopeLocked(filter.TenantID)
	rows := make([]domain.Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		if agentMatchesScope(agent, filter.ManagementScope, tenantIDs) {
			rows = append(rows, agent)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	return rows, nil
}

func (m *Memory) GetAgent(_ context.Context, id string) (domain.Agent, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	agent, ok := m.agents[id]
	return agent, ok, nil
}

func (m *Memory) UpdateAgent(_ context.Context, agent domain.Agent) (domain.Agent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.agents[agent.ID]
	if !ok {
		return domain.Agent{}, false, nil
	}
	agent.Credentials = existing.Credentials
	agent.CredentialVersion = existing.CredentialVersion
	m.agents[agent.ID] = agent
	return agent, true, nil
}

func (m *Memory) UpdateAgentWithAudit(_ context.Context, agent domain.Agent, build AgentAuditBuilder) (domain.Agent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.agents[agent.ID]
	if !ok {
		return domain.Agent{}, false, nil
	}
	agent.Credentials = existing.Credentials
	agent.CredentialVersion = existing.CredentialVersion
	m.agents[agent.ID] = agent
	m.audits = append(m.audits, build(agent))
	return agent, true, nil
}

func (m *Memory) RotateAgentCredentials(_ context.Context, id string, credentials map[string]string, now time.Time) (domain.Agent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[id]
	if !ok {
		return domain.Agent{}, false, nil
	}
	agent.Credentials = credentials
	agent.CredentialVersion++
	agent.UpdatedAt = now
	m.agents[id] = agent
	return agent, true, nil
}

func (m *Memory) RotateAgentCredentialsWithAudit(_ context.Context, id string, credentials map[string]string, now time.Time, build AgentAuditBuilder) (domain.Agent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[id]
	if !ok {
		return domain.Agent{}, false, nil
	}
	agent.Credentials = credentials
	agent.CredentialVersion++
	agent.UpdatedAt = now
	m.agents[id] = agent
	m.audits = append(m.audits, build(agent))
	return agent, true, nil
}

func (m *Memory) DisableAgent(_ context.Context, id string, now time.Time) (domain.Agent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[id]
	if !ok {
		return domain.Agent{}, false, nil
	}
	agent.Status = domain.AgentStatusDisabled
	agent.UpdatedAt = now
	m.agents[id] = agent
	return agent, true, nil
}

func (m *Memory) DisableAgentWithAudit(_ context.Context, id string, now time.Time, build AgentAuditBuilder) (domain.Agent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[id]
	if !ok {
		return domain.Agent{}, false, nil
	}
	agent.Status = domain.AgentStatusDisabled
	agent.UpdatedAt = now
	m.agents[id] = agent
	m.audits = append(m.audits, build(agent))
	return agent, true, nil
}

func (m *Memory) CreateAgentKey(_ context.Context, key domain.AgentKey) (domain.AgentKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[key.ID] = key
	return key, nil
}

func (m *Memory) CreateAgentKeyWithAudit(_ context.Context, key domain.AgentKey, build AgentKeyAuditBuilder) (domain.AgentKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[key.ID] = key
	m.audits = append(m.audits, build(key))
	return key, nil
}

func (m *Memory) ListAgentKeys(_ context.Context, scope ManagementScope) ([]domain.AgentKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(scope.TenantID)
	rows := make([]domain.AgentKey, 0, len(m.keys))
	for _, key := range m.keys {
		agent, ok := m.agents[key.AgentID]
		if !ok || !agentMatchesScope(agent, scope, tenantIDs) {
			continue
		}
		rows = append(rows, key)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	return rows, nil
}

func (m *Memory) RevokeAgentKey(_ context.Context, id string, now time.Time) (domain.AgentKey, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.keys[id]
	if !ok {
		return domain.AgentKey{}, false, nil
	}
	key.RevokedAt = now
	m.keys[id] = key
	return key, true, nil
}

func (m *Memory) RevokeAgentKeyWithAudit(_ context.Context, id string, now time.Time, build AgentKeyAuditBuilder) (domain.AgentKey, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.keys[id]
	if !ok {
		return domain.AgentKey{}, false, nil
	}
	key.RevokedAt = now
	m.keys[id] = key
	m.audits = append(m.audits, build(key))
	return key, true, nil
}

func (m *Memory) FindAgentByKeyHash(_ context.Context, hash string, now time.Time) (domain.Agent, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, key := range m.keys {
		if key.Hash != hash {
			continue
		}
		if !key.RevokedAt.IsZero() || now.After(key.ExpiresAt) {
			return domain.Agent{}, false, nil
		}
		agent, ok := m.agents[key.AgentID]
		if !ok || agent.Status != domain.AgentStatusActive {
			return domain.Agent{}, false, nil
		}
		return agent, ok, nil
	}
	return domain.Agent{}, false, nil
}

func (m *Memory) CreateAccessGrant(_ context.Context, grant domain.AccessGrant) (domain.AccessGrant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.grants[grant.ID] = grant
	return grant, nil
}

func (m *Memory) CreateAccessGrantWithAudit(_ context.Context, grant domain.AccessGrant, build AccessGrantAuditBuilder) (domain.AccessGrant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.grants[grant.ID] = grant
	m.audits = append(m.audits, build(grant))
	return grant, nil
}

func (m *Memory) ListAccessGrants(_ context.Context, scope ManagementScope) ([]domain.AccessGrant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(scope.TenantID)
	rows := make([]domain.AccessGrant, 0, len(m.grants))
	for _, grant := range m.grants {
		if !m.grantMatchesScope(grant, scope, tenantIDs) {
			continue
		}
		rows = append(rows, grant)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	return rows, nil
}

func (m *Memory) RevokeAccessGrant(_ context.Context, id string, now time.Time) (domain.AccessGrant, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	grant, ok := m.grants[id]
	if !ok {
		return domain.AccessGrant{}, false, nil
	}
	grant.RevokedAt = now
	m.grants[id] = grant
	return grant, true, nil
}

func (m *Memory) RevokeAccessGrantWithAudit(_ context.Context, id string, now time.Time, build AccessGrantAuditBuilder) (domain.AccessGrant, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	grant, ok := m.grants[id]
	if !ok {
		return domain.AccessGrant{}, false, nil
	}
	grant.RevokedAt = now
	m.grants[id] = grant
	m.audits = append(m.audits, build(grant))
	return grant, true, nil
}

func (m *Memory) HasGrant(_ context.Context, callerID string, targetID string, routeType string, routeKey string, now time.Time) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hasGrantLocked(callerID, targetID, routeType, routeKey, now)
}

func (m *Memory) UpsertCapability(_ context.Context, capability domain.Capability) (domain.Capability, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	capability = cloneCapability(capability)
	for id, existing := range m.capabilities {
		if existing.TargetID == capability.TargetID && existing.Type == capability.Type && existing.Key == capability.Key && id != capability.ID {
			delete(m.capabilities, id)
			break
		}
	}
	m.capabilities[capability.ID] = capability
	return cloneCapability(capability), nil
}

func (m *Memory) ListCapabilities(_ context.Context, filter CapabilityFilter) ([]domain.Capability, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(filter.TenantID)
	rows := make([]domain.Capability, 0, len(m.capabilities))
	for _, capability := range m.capabilities {
		if !m.capabilityMatchesFilter(capability, filter, tenantIDs) {
			continue
		}
		rows = append(rows, cloneCapability(capability))
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].UpdatedAt.Equal(rows[j].UpdatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].UpdatedAt.Before(rows[j].UpdatedAt)
	})
	return rows, nil
}

func (m *Memory) GetCapability(_ context.Context, id string) (domain.Capability, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	capability, ok := m.capabilities[id]
	return cloneCapability(capability), ok, nil
}

func (m *Memory) UpdateCapability(_ context.Context, capability domain.Capability) (domain.Capability, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.capabilities[capability.ID]
	if !ok {
		return domain.Capability{}, false, nil
	}
	capability.TargetID = existing.TargetID
	capability.Type = existing.Type
	capability.Key = existing.Key
	capability.DiscoveredAt = existing.DiscoveredAt
	capability = cloneCapability(capability)
	m.capabilities[capability.ID] = capability
	return cloneCapability(capability), true, nil
}

func (m *Memory) CreateTenantEntitlement(_ context.Context, entitlement domain.TenantEntitlement) (domain.TenantEntitlement, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entitlement = cloneTenantEntitlement(entitlement)
	m.entitlements[entitlement.ID] = entitlement
	return cloneTenantEntitlement(entitlement), nil
}

func (m *Memory) ListTenantEntitlements(_ context.Context, filter EntitlementFilter) ([]domain.TenantEntitlement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(filter.TenantID)
	rows := make([]domain.TenantEntitlement, 0, len(m.entitlements))
	for _, entitlement := range m.entitlements {
		if !tenantEntitlementMatchesFilter(entitlement, filter, tenantIDs) {
			continue
		}
		rows = append(rows, cloneTenantEntitlement(entitlement))
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	return rows, nil
}

func (m *Memory) CreateWorkspaceAssignment(_ context.Context, assignment domain.WorkspaceAssignment) (domain.WorkspaceAssignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	assignment = cloneWorkspaceAssignment(assignment)
	m.workspaceAssignments[assignment.ID] = assignment
	return cloneWorkspaceAssignment(assignment), nil
}

func (m *Memory) ListWorkspaceAssignments(_ context.Context, filter AssignmentFilter) ([]domain.WorkspaceAssignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(filter.TenantID)
	rows := make([]domain.WorkspaceAssignment, 0, len(m.workspaceAssignments))
	for _, assignment := range m.workspaceAssignments {
		if !workspaceAssignmentMatchesFilter(assignment, filter, tenantIDs) {
			continue
		}
		rows = append(rows, cloneWorkspaceAssignment(assignment))
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	return rows, nil
}

func (m *Memory) CreateInstanceAssignment(_ context.Context, assignment domain.InstanceAssignment) (domain.InstanceAssignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	assignment = cloneInstanceAssignment(assignment)
	m.instanceAssignments[assignment.ID] = assignment
	return cloneInstanceAssignment(assignment), nil
}

func (m *Memory) ListInstanceAssignments(_ context.Context, filter InstanceAssignmentFilter) ([]domain.InstanceAssignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(filter.TenantID)
	rows := make([]domain.InstanceAssignment, 0, len(m.instanceAssignments))
	for _, assignment := range m.instanceAssignments {
		if !m.instanceAssignmentMatchesFilter(assignment, filter, tenantIDs) {
			continue
		}
		rows = append(rows, cloneInstanceAssignment(assignment))
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	return rows, nil
}

func (m *Memory) CreatePermissionPackageApplication(_ context.Context, application domain.PermissionPackageApplication) (domain.PermissionPackageApplication, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	application = clonePermissionPackageApplication(application)
	m.packageApplications[application.ID] = application
	return clonePermissionPackageApplication(application), nil
}

func (m *Memory) ListPermissionPackageApplications(_ context.Context, filter PermissionPackageApplicationFilter) ([]domain.PermissionPackageApplication, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(filter.TenantID)
	rows := make([]domain.PermissionPackageApplication, 0, len(m.packageApplications))
	for _, application := range m.packageApplications {
		if !permissionPackageApplicationMatchesFilter(application, filter, tenantIDs) {
			continue
		}
		rows = append(rows, clonePermissionPackageApplication(application))
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].AppliedAt.Equal(rows[j].AppliedAt) {
			return rows[i].ID > rows[j].ID
		}
		return rows[i].AppliedAt.After(rows[j].AppliedAt)
	})
	if filter.Limit > 0 && len(rows) > filter.Limit {
		rows = rows[:filter.Limit]
	}
	return rows, nil
}

func (m *Memory) CreatePermissionPackageApprovalRequest(_ context.Context, request domain.PermissionPackageApprovalRequest) (domain.PermissionPackageApprovalRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	request = clonePermissionPackageApprovalRequest(request)
	m.packageApprovals[request.ID] = request
	return clonePermissionPackageApprovalRequest(request), nil
}

func (m *Memory) ListPermissionPackageApprovalRequests(_ context.Context, filter PermissionPackageApprovalRequestFilter) ([]domain.PermissionPackageApprovalRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(filter.TenantID)
	rows := make([]domain.PermissionPackageApprovalRequest, 0, len(m.packageApprovals))
	for _, request := range m.packageApprovals {
		if !permissionPackageApprovalRequestMatchesFilter(request, filter, tenantIDs) {
			continue
		}
		rows = append(rows, clonePermissionPackageApprovalRequest(request))
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID > rows[j].ID
		}
		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})
	if filter.Limit > 0 && len(rows) > filter.Limit {
		rows = rows[:filter.Limit]
	}
	return rows, nil
}

func (m *Memory) GetPermissionPackageApprovalRequest(_ context.Context, id string) (domain.PermissionPackageApprovalRequest, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	request, ok := m.packageApprovals[id]
	if !ok {
		return domain.PermissionPackageApprovalRequest{}, false, nil
	}
	return clonePermissionPackageApprovalRequest(request), true, nil
}

func (m *Memory) UpdatePermissionPackageApprovalRequest(_ context.Context, request domain.PermissionPackageApprovalRequest) (domain.PermissionPackageApprovalRequest, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.packageApprovals[request.ID]; !ok {
		return domain.PermissionPackageApprovalRequest{}, false, nil
	}
	request = clonePermissionPackageApprovalRequest(request)
	m.packageApprovals[request.ID] = request
	return clonePermissionPackageApprovalRequest(request), true, nil
}

func (m *Memory) EvaluateCapabilityAccess(_ context.Context, req CapabilityAccessRequest) (domain.CapabilityAccessDecision, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	capability, ok := m.capabilities[req.CapabilityID]
	if !ok || capability.TargetID != req.TargetID {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "capability", Reason: "capability is not registered for target"}, nil
	}
	if capability.DiscoveryStatus != domain.CapabilityDiscoveryApproved {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "capability", CapabilityID: capability.ID, Reason: "capability is not approved"}, nil
	}
	entitlement, ok := m.matchTenantEntitlementLocked(req, capability.ID)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "tenant_entitlement", CapabilityID: capability.ID, Reason: "tenant has no entitlement for capability"}, nil
	}
	if entitlement.Effect == domain.PolicyEffectDeny {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "tenant_entitlement", CapabilityID: capability.ID, EntitlementID: entitlement.ID, Reason: "tenant entitlement denies capability"}, nil
	}
	entitlementScopes, ok := domain.EffectiveDataScopes(capability.DataScopes, entitlement.DataScopes)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "tenant_entitlement", CapabilityID: capability.ID, EntitlementID: entitlement.ID, Reason: "tenant entitlement data scopes exceed capability boundary"}, nil
	}
	workspaceAssignment, ok := m.matchWorkspaceAssignmentLocked(req, entitlement.ID)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "workspace_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, Reason: "workspace has no assignment for capability"}, nil
	}
	if workspaceAssignment.Effect == domain.PolicyEffectDeny {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "workspace_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, Reason: "workspace assignment denies capability"}, nil
	}
	workspaceScopes, ok := domain.EffectiveDataScopes(entitlementScopes, workspaceAssignment.DataScopes)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "workspace_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, Reason: "workspace assignment data scopes exceed tenant entitlement boundary"}, nil
	}
	instanceAssignment, ok := m.matchInstanceAssignmentLocked(req, workspaceAssignment.ID)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "instance_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, Reason: "caller instance has no assignment for capability"}, nil
	}
	if instanceAssignment.Effect == domain.PolicyEffectDeny {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "instance_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, InstanceAssignmentID: instanceAssignment.ID, Reason: "caller instance assignment denies capability"}, nil
	}
	instanceScopes, ok := domain.EffectiveDataScopes(workspaceScopes, instanceAssignment.DataScopes)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "instance_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, InstanceAssignmentID: instanceAssignment.ID, Reason: "caller instance assignment data scopes exceed workspace assignment boundary"}, nil
	}
	return domain.CapabilityAccessDecision{
		Allowed:               true,
		Source:                "capability_governance",
		CapabilityID:          capability.ID,
		EntitlementID:         entitlement.ID,
		WorkspaceAssignmentID: workspaceAssignment.ID,
		InstanceAssignmentID:  instanceAssignment.ID,
		Reason:                "capability assignment matched",
		DataScopes:            instanceScopes,
	}, nil
}

func (m *Memory) CreateRoutePolicy(_ context.Context, policy domain.RoutePolicy) (domain.RoutePolicy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	policy.Retry = cloneRoutePolicyRetry(policy.Retry)
	m.policies[policy.ID] = policy
	return policy, nil
}

func (m *Memory) CreateRoutePolicyWithAudit(_ context.Context, policy domain.RoutePolicy, build RoutePolicyAuditBuilder) (domain.RoutePolicy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[policy.ID] = policy
	m.audits = append(m.audits, build(policy))
	return policy, nil
}

func (m *Memory) ListRoutePolicies(_ context.Context, scope ManagementScope) ([]domain.RoutePolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(scope.TenantID)
	rows := make([]domain.RoutePolicy, 0, len(m.policies))
	for _, policy := range m.policies {
		if !routePolicyMatchesScope(policy, scope, tenantIDs) {
			continue
		}
		rows = append(rows, policy)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	return rows, nil
}

func (m *Memory) GetRoutePolicy(_ context.Context, id string) (domain.RoutePolicy, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	policy, ok := m.policies[id]
	return policy, ok, nil
}

func (m *Memory) UpdateRoutePolicy(_ context.Context, policy domain.RoutePolicy) (domain.RoutePolicy, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.policies[policy.ID]
	if !ok {
		return domain.RoutePolicy{}, false, nil
	}
	policy.Retry = cloneRoutePolicyRetry(policy.Retry)
	policy.TenantID = existing.TenantID
	policy.WorkspaceID = existing.WorkspaceID
	policy.CallerID = existing.CallerID
	policy.TargetID = existing.TargetID
	policy.CreatedAt = existing.CreatedAt
	m.policies[policy.ID] = policy
	return policy, true, nil
}

func (m *Memory) UpdateRoutePolicyWithAudit(_ context.Context, policy domain.RoutePolicy, build RoutePolicyAuditBuilder) (domain.RoutePolicy, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.policies[policy.ID]
	if !ok {
		return domain.RoutePolicy{}, false, nil
	}
	policy.TenantID = existing.TenantID
	policy.WorkspaceID = existing.WorkspaceID
	policy.CallerID = existing.CallerID
	policy.TargetID = existing.TargetID
	policy.CreatedAt = existing.CreatedAt
	m.policies[policy.ID] = policy
	m.audits = append(m.audits, build(policy))
	return policy, true, nil
}

func (m *Memory) DisableRoutePolicy(_ context.Context, id string, now time.Time) (domain.RoutePolicy, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	policy, ok := m.policies[id]
	if !ok {
		return domain.RoutePolicy{}, false, nil
	}
	policy.Status = domain.RoutePolicyStatusDisabled
	policy.UpdatedAt = now
	m.policies[id] = policy
	return policy, true, nil
}

func (m *Memory) DisableRoutePolicyWithAudit(_ context.Context, id string, now time.Time, build RoutePolicyAuditBuilder) (domain.RoutePolicy, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	policy, ok := m.policies[id]
	if !ok {
		return domain.RoutePolicy{}, false, nil
	}
	policy.Status = domain.RoutePolicyStatusDisabled
	policy.UpdatedAt = now
	m.policies[id] = policy
	m.audits = append(m.audits, build(policy))
	return policy, true, nil
}

func (m *Memory) EvaluateRouteAccess(_ context.Context, callerID string, targetID string, routeType string, routeKey string, now time.Time) (domain.RouteAccessDecision, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if policy, ok := m.topMatchingRoutePolicyLocked(callerID, targetID, routeType, routeKey); ok {
		return routePolicyDecision(policy), nil
	}
	if m.hasGrantLocked(callerID, targetID, routeType, routeKey, now) {
		return domain.RouteAccessDecision{
			Allowed: true,
			Source:  "access_grant",
			Reason:  "access grant matched",
		}, nil
	}
	return domain.RouteAccessDecision{
		Allowed: false,
		Source:  "none",
		Reason:  "caller has no route policy or access grant for target route",
	}, nil
}

func (m *Memory) hasGrantLocked(callerID string, targetID string, routeType string, routeKey string, now time.Time) bool {
	for _, grant := range m.grants {
		if grant.CallerID != callerID || grant.TargetID != targetID {
			continue
		}
		if !grant.RevokedAt.IsZero() || (!grant.ExpiresAt.IsZero() && now.After(grant.ExpiresAt)) {
			continue
		}
		if grant.RouteType == "" || grant.RouteType == routeType {
			if grant.RouteKey == "" || grant.RouteKey == routeKey {
				return true
			}
		}
	}
	return false
}

func (m *Memory) topMatchingRoutePolicyLocked(callerID string, targetID string, routeType string, routeKey string) (domain.RoutePolicy, bool) {
	var best domain.RoutePolicy
	found := false
	caller, callerOK := m.agents[callerID]
	target, targetOK := m.agents[targetID]
	if !callerOK || !targetOK {
		return domain.RoutePolicy{}, false
	}
	for _, policy := range m.policies {
		if !routePolicyMatches(policy, caller, target, routeType, routeKey) {
			continue
		}
		if !found || routePolicyPrecedes(policy, best) {
			best = policy
			found = true
		}
	}
	return best, found
}

func (m *Memory) AppendTrace(_ context.Context, event domain.TraceEvent) (domain.TraceEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.traces = append(m.traces, event)
	return event, nil
}

func (m *Memory) ListTraces(_ context.Context, filter TraceFilter) ([]domain.TraceEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(filter.TenantID)
	rows := make([]domain.TraceEvent, 0, len(m.traces))
	for _, trace := range m.traces {
		if filter.RunID != "" && trace.RunID != filter.RunID {
			continue
		}
		if filter.Decision != "" && trace.Decision != filter.Decision {
			continue
		}
		if filter.CallerID != "" && trace.CallerID != filter.CallerID {
			continue
		}
		if filter.TargetID != "" && trace.TargetID != filter.TargetID {
			continue
		}
		if !m.traceMatchesScope(trace, filter.ManagementScope, tenantIDs) {
			continue
		}
		rows = append(rows, trace)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	return rows, nil
}

func (m *Memory) AppendAuditEvent(_ context.Context, event domain.AuditEvent) (domain.AuditEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audits = append(m.audits, event)
	return event, nil
}

func (m *Memory) ListAuditEvents(_ context.Context, filter AuditEventFilter) ([]domain.AuditEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantIDs := m.tenantIDsForScopeLocked(filter.TenantID)
	rows := make([]domain.AuditEvent, 0, len(m.audits))
	for _, event := range m.audits {
		if filter.Action != "" && event.Action != filter.Action {
			continue
		}
		if filter.ResourceType != "" && event.ResourceType != filter.ResourceType {
			continue
		}
		if filter.ResourceID != "" && event.ResourceID != filter.ResourceID {
			continue
		}
		if !auditEventMatchesScope(event, filter.ManagementScope, tenantIDs) {
			continue
		}
		rows = append(rows, event)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	if filter.Limit > 0 && len(rows) > filter.Limit {
		rows = rows[:filter.Limit]
	}
	return rows, nil
}

func (m *Memory) grantMatchesScope(grant domain.AccessGrant, scope ManagementScope, tenantIDs map[string]struct{}) bool {
	if scope.TenantID == "" && scope.WorkspaceID == "" {
		return true
	}
	caller, callerOK := m.agents[grant.CallerID]
	target, targetOK := m.agents[grant.TargetID]
	return (callerOK && agentMatchesScope(caller, scope, tenantIDs)) || (targetOK && agentMatchesScope(target, scope, tenantIDs))
}

func (m *Memory) traceMatchesScope(trace domain.TraceEvent, scope ManagementScope, tenantIDs map[string]struct{}) bool {
	if scope.TenantID == "" && scope.WorkspaceID == "" {
		return true
	}
	if trace.TenantID != "" || trace.WorkspaceID != "" {
		if !tenantIDMatchesScope(trace.TenantID, scope.TenantID, tenantIDs) {
			return false
		}
		if scope.WorkspaceID != "" && trace.WorkspaceID != scope.WorkspaceID {
			return false
		}
		return true
	}
	caller, callerOK := m.agents[trace.CallerID]
	target, targetOK := m.agents[trace.TargetID]
	return (callerOK && agentMatchesScope(caller, scope, tenantIDs)) || (targetOK && agentMatchesScope(target, scope, tenantIDs))
}

func (m *Memory) capabilityMatchesFilter(capability domain.Capability, filter CapabilityFilter, tenantIDs map[string]struct{}) bool {
	if filter.TargetID != "" && capability.TargetID != filter.TargetID {
		return false
	}
	if filter.Status != "" && capability.DiscoveryStatus != filter.Status {
		return false
	}
	if filter.TenantID == "" && filter.WorkspaceID == "" {
		return true
	}
	target, ok := m.agents[capability.TargetID]
	return ok && agentMatchesScope(target, filter.ManagementScope, tenantIDs)
}

func tenantEntitlementMatchesFilter(entitlement domain.TenantEntitlement, filter EntitlementFilter, tenantIDs map[string]struct{}) bool {
	if !tenantIDMatchesScope(entitlement.TenantID, filter.TenantID, tenantIDs) {
		return false
	}
	if filter.TargetID != "" && entitlement.TargetID != filter.TargetID {
		return false
	}
	if filter.CapabilityID != "" && entitlement.CapabilityID != filter.CapabilityID {
		return false
	}
	return true
}

func workspaceAssignmentMatchesFilter(assignment domain.WorkspaceAssignment, filter AssignmentFilter, tenantIDs map[string]struct{}) bool {
	if !tenantIDMatchesScope(assignment.TenantID, filter.TenantID, tenantIDs) {
		return false
	}
	if filter.WorkspaceID != "" && assignment.WorkspaceID != filter.WorkspaceID {
		return false
	}
	if filter.EntitlementID != "" && assignment.TenantEntitlementID != filter.EntitlementID {
		return false
	}
	return true
}

func (m *Memory) instanceAssignmentMatchesFilter(assignment domain.InstanceAssignment, filter InstanceAssignmentFilter, tenantIDs map[string]struct{}) bool {
	if !tenantIDMatchesScope(assignment.TenantID, filter.TenantID, tenantIDs) {
		return false
	}
	if filter.WorkspaceID != "" && assignment.WorkspaceID != filter.WorkspaceID {
		return false
	}
	if filter.CallerInstanceID != "" && assignment.CallerInstanceID != filter.CallerInstanceID {
		return false
	}
	if filter.CapabilityID == "" {
		return true
	}
	workspaceAssignment, ok := m.workspaceAssignments[assignment.WorkspaceAssignmentID]
	if !ok {
		return false
	}
	entitlement, ok := m.entitlements[workspaceAssignment.TenantEntitlementID]
	return ok && entitlement.CapabilityID == filter.CapabilityID
}

func permissionPackageApplicationMatchesFilter(application domain.PermissionPackageApplication, filter PermissionPackageApplicationFilter, tenantIDs map[string]struct{}) bool {
	if !tenantIDMatchesScope(application.TenantID, filter.TenantID, tenantIDs) {
		return false
	}
	if filter.WorkspaceID != "" && application.WorkspaceID != filter.WorkspaceID {
		return false
	}
	if filter.TemplateID != "" && application.TemplateID != filter.TemplateID {
		return false
	}
	if filter.TargetID != "" && application.TargetID != filter.TargetID {
		return false
	}
	if filter.CallerInstanceID != "" && application.CallerInstanceID != filter.CallerInstanceID {
		return false
	}
	return true
}

func permissionPackageApprovalRequestMatchesFilter(request domain.PermissionPackageApprovalRequest, filter PermissionPackageApprovalRequestFilter, tenantIDs map[string]struct{}) bool {
	if !tenantIDMatchesScope(request.TenantID, filter.TenantID, tenantIDs) {
		return false
	}
	if filter.WorkspaceID != "" && request.WorkspaceID != filter.WorkspaceID {
		return false
	}
	if filter.TemplateID != "" && request.TemplateID != filter.TemplateID {
		return false
	}
	if filter.TargetID != "" && request.TargetID != filter.TargetID {
		return false
	}
	if filter.CallerInstanceID != "" && request.CallerInstanceID != filter.CallerInstanceID {
		return false
	}
	if filter.Status != "" && request.Status != filter.Status {
		return false
	}
	return true
}

func (m *Memory) matchTenantEntitlementLocked(req CapabilityAccessRequest, capabilityID string) (domain.TenantEntitlement, bool) {
	var best domain.TenantEntitlement
	found := false
	for _, entitlement := range m.entitlements {
		if entitlement.TenantID != req.TenantID || entitlement.TargetID != req.TargetID || entitlement.CapabilityID != capabilityID {
			continue
		}
		if entitlement.Status != domain.PolicyStatusEnabled {
			continue
		}
		if !found || tenantEntitlementPrecedes(entitlement, best) {
			best = entitlement
			found = true
		}
	}
	return cloneTenantEntitlement(best), found
}

func (m *Memory) matchWorkspaceAssignmentLocked(req CapabilityAccessRequest, entitlementID string) (domain.WorkspaceAssignment, bool) {
	var best domain.WorkspaceAssignment
	found := false
	for _, assignment := range m.workspaceAssignments {
		if assignment.TenantEntitlementID != entitlementID || assignment.TenantID != req.TenantID || assignment.WorkspaceID != req.WorkspaceID {
			continue
		}
		if assignment.Status != domain.PolicyStatusEnabled {
			continue
		}
		if !found || assignmentPrecedes(assignment.Effect, assignment.CreatedAt, assignment.ID, best.Effect, best.CreatedAt, best.ID) {
			best = assignment
			found = true
		}
	}
	return cloneWorkspaceAssignment(best), found
}

func (m *Memory) matchInstanceAssignmentLocked(req CapabilityAccessRequest, workspaceAssignmentID string) (domain.InstanceAssignment, bool) {
	var best domain.InstanceAssignment
	found := false
	for _, assignment := range m.instanceAssignments {
		if assignment.WorkspaceAssignmentID != workspaceAssignmentID || assignment.TenantID != req.TenantID || assignment.WorkspaceID != req.WorkspaceID || assignment.CallerInstanceID != req.CallerInstanceID {
			continue
		}
		if assignment.Status != domain.PolicyStatusEnabled {
			continue
		}
		if !subjectSelectorMatches(assignment.SubjectSelector, req.SubjectID) {
			continue
		}
		if !found || assignmentPrecedes(assignment.Effect, assignment.CreatedAt, assignment.ID, best.Effect, best.CreatedAt, best.ID) {
			best = assignment
			found = true
		}
	}
	return cloneInstanceAssignment(best), found
}

func tenantEntitlementPrecedes(left domain.TenantEntitlement, right domain.TenantEntitlement) bool {
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	if left.Effect != right.Effect {
		return left.Effect == domain.PolicyEffectDeny
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		return left.CreatedAt.Before(right.CreatedAt)
	}
	return left.ID < right.ID
}

func policyEffectPrecedes(left domain.PolicyEffect, right domain.PolicyEffect) bool {
	return left == domain.PolicyEffectDeny && right != domain.PolicyEffectDeny
}

func assignmentPrecedes(leftEffect domain.PolicyEffect, leftCreatedAt time.Time, leftID string, rightEffect domain.PolicyEffect, rightCreatedAt time.Time, rightID string) bool {
	if leftEffect != rightEffect {
		return leftEffect == domain.PolicyEffectDeny
	}
	if !leftCreatedAt.Equal(rightCreatedAt) {
		return leftCreatedAt.Before(rightCreatedAt)
	}
	return leftID < rightID
}

func subjectSelectorMatches(selector string, subjectID string) bool {
	selector = strings.TrimSpace(selector)
	subjectID = strings.TrimSpace(subjectID)
	if selector == "" {
		return true
	}
	if selector == subjectID {
		return true
	}
	if strings.HasSuffix(selector, "*") {
		return strings.HasPrefix(subjectID, strings.TrimSuffix(selector, "*"))
	}
	return false
}

func routePolicyMatchesScope(policy domain.RoutePolicy, scope ManagementScope, tenantIDs map[string]struct{}) bool {
	if !tenantIDMatchesScope(policy.TenantID, scope.TenantID, tenantIDs) {
		return false
	}
	if scope.WorkspaceID != "" && policy.WorkspaceID != scope.WorkspaceID {
		return false
	}
	return true
}

func routePolicyMatches(policy domain.RoutePolicy, caller domain.Agent, target domain.Agent, routeType string, routeKey string) bool {
	if policy.CallerID != caller.ID || policy.TargetID != target.ID {
		return false
	}
	if policy.TenantID != caller.TenantID || policy.WorkspaceID != caller.WorkspaceID {
		return false
	}
	if target.TenantID != caller.TenantID || target.WorkspaceID != caller.WorkspaceID {
		return false
	}
	if policy.Status != domain.RoutePolicyStatusEnabled {
		return false
	}
	if policy.RouteType != "" && policy.RouteType != routeType {
		return false
	}
	if policy.RouteKey != "" && policy.RouteKey != routeKey {
		return false
	}
	return true
}

func routePolicyPrecedes(left domain.RoutePolicy, right domain.RoutePolicy) bool {
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	if left.Effect != right.Effect {
		return left.Effect == domain.RoutePolicyEffectDeny
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		return left.CreatedAt.Before(right.CreatedAt)
	}
	return left.ID < right.ID
}

func routePolicyDecision(policy domain.RoutePolicy) domain.RouteAccessDecision {
	allowed := policy.Effect == domain.RoutePolicyEffectAllow
	reason := "route policy allowed"
	if !allowed {
		reason = "route policy denied"
	}
	return domain.RouteAccessDecision{
		Allowed:  allowed,
		Source:   "route_policy",
		PolicyID: policy.ID,
		Reason:   reason,
		Retry:    routePolicyRetryForDecision(policy),
	}
}

func routePolicyRetryForDecision(policy domain.RoutePolicy) *domain.RoutePolicyRetry {
	if policy.Effect != domain.RoutePolicyEffectAllow {
		return nil
	}
	return cloneRoutePolicyRetry(policy.Retry)
}

func cloneRoutePolicyRetry(retry *domain.RoutePolicyRetry) *domain.RoutePolicyRetry {
	if retry == nil {
		return nil
	}
	copied := *retry
	if retry.StatusCodes != nil {
		copied.StatusCodes = append([]int(nil), retry.StatusCodes...)
	}
	return &copied
}

func agentMatchesScope(agent domain.Agent, scope ManagementScope, tenantIDs map[string]struct{}) bool {
	if !tenantIDMatchesScope(agent.TenantID, scope.TenantID, tenantIDs) {
		return false
	}
	if scope.WorkspaceID != "" && agent.WorkspaceID != scope.WorkspaceID {
		return false
	}
	return true
}

func auditEventMatchesScope(event domain.AuditEvent, scope ManagementScope, tenantIDs map[string]struct{}) bool {
	if !tenantIDMatchesScope(event.TenantID, scope.TenantID, tenantIDs) {
		return false
	}
	if scope.WorkspaceID != "" && event.WorkspaceID != scope.WorkspaceID {
		return false
	}
	return true
}

func tenantIDMatchesScope(actual string, scopeTenantID string, tenantIDs map[string]struct{}) bool {
	scopeTenantID = strings.TrimSpace(scopeTenantID)
	if scopeTenantID == "" {
		return true
	}
	if tenantIDs == nil {
		return actual == scopeTenantID
	}
	_, ok := tenantIDs[actual]
	return ok
}

func cloneCapability(capability domain.Capability) domain.Capability {
	capability.InputSchema = cloneMap(capability.InputSchema)
	capability.OutputSchema = cloneMap(capability.OutputSchema)
	capability.NativeScopes = cloneStrings(capability.NativeScopes)
	capability.DataDomains = cloneStrings(capability.DataDomains)
	capability.DataScopes = cloneDataScopes(capability.DataScopes)
	return capability
}

func cloneTenantEntitlement(entitlement domain.TenantEntitlement) domain.TenantEntitlement {
	entitlement.DataScopes = cloneDataScopes(entitlement.DataScopes)
	return entitlement
}

func cloneWorkspaceAssignment(assignment domain.WorkspaceAssignment) domain.WorkspaceAssignment {
	assignment.DataScopes = cloneDataScopes(assignment.DataScopes)
	return assignment
}

func cloneInstanceAssignment(assignment domain.InstanceAssignment) domain.InstanceAssignment {
	assignment.DataScopes = cloneDataScopes(assignment.DataScopes)
	return assignment
}

func clonePermissionPackageApplication(application domain.PermissionPackageApplication) domain.PermissionPackageApplication {
	application.DataScopes = cloneDataScopes(application.DataScopes)
	application.AllowedCapabilityIDs = cloneStrings(application.AllowedCapabilityIDs)
	application.AllowedCapabilityKeys = cloneStrings(application.AllowedCapabilityKeys)
	application.TenantEntitlementIDs = cloneStrings(application.TenantEntitlementIDs)
	application.WorkspaceAssignmentIDs = cloneStrings(application.WorkspaceAssignmentIDs)
	application.InstanceAssignmentIDs = cloneStrings(application.InstanceAssignmentIDs)
	return application
}

func clonePermissionPackageApprovalRequest(request domain.PermissionPackageApprovalRequest) domain.PermissionPackageApprovalRequest {
	request.DataScopes = cloneDataScopes(request.DataScopes)
	request.AllowedCapabilityIDs = cloneStrings(request.AllowedCapabilityIDs)
	request.AllowedCapabilityKeys = cloneStrings(request.AllowedCapabilityKeys)
	request.PolicyGate = clonePermissionPackagePolicyGate(request.PolicyGate)
	return request
}

func clonePermissionPackagePolicyGate(gate domain.PermissionPackagePolicyGate) domain.PermissionPackagePolicyGate {
	gate.Reasons = clonePermissionPackagePolicyReasons(gate.Reasons)
	gate.NextActions = cloneStrings(gate.NextActions)
	return gate
}

func clonePermissionPackagePolicyReasons(reasons []domain.PermissionPackagePolicyReason) []domain.PermissionPackagePolicyReason {
	if reasons == nil {
		return nil
	}
	out := make([]domain.PermissionPackagePolicyReason, 0, len(reasons))
	for _, reason := range reasons {
		reason.ReasonValues = cloneStringMap(reason.ReasonValues)
		out = append(out, reason)
	}
	return out
}

func cloneStringMap(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	copied := make(map[string]string, len(value))
	for key, item := range value {
		copied[key] = item
	}
	return copied
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	copied := make(map[string]any, len(value))
	for key, item := range value {
		copied[key] = item
	}
	return copied
}

func cloneStrings(value []string) []string {
	if value == nil {
		return nil
	}
	return append([]string(nil), value...)
}

func cloneDataScopes(value []domain.DataScope) []domain.DataScope {
	if value == nil {
		return nil
	}
	return append([]domain.DataScope(nil), value...)
}
