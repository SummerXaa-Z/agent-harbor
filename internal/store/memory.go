package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

type Repository interface {
	CreateAgent(context.Context, domain.Agent) (domain.Agent, error)
	CreateAgentWithAudit(context.Context, domain.Agent, domain.AuditEvent) (domain.Agent, error)
	ListAgents(context.Context, AgentFilter) ([]domain.Agent, error)
	GetAgent(context.Context, string) (domain.Agent, bool, error)
	UpdateAgent(context.Context, domain.Agent) (domain.Agent, bool, error)
	UpdateAgentWithAudit(context.Context, domain.Agent, domain.AuditEvent) (domain.Agent, bool, error)
	RotateAgentCredentials(context.Context, string, map[string]string, time.Time) (domain.Agent, bool, error)
	RotateAgentCredentialsWithAudit(context.Context, string, map[string]string, time.Time, domain.AuditEvent) (domain.Agent, bool, error)
	DisableAgent(context.Context, string, time.Time) (domain.Agent, bool, error)
	DisableAgentWithAudit(context.Context, string, time.Time, domain.AuditEvent) (domain.Agent, bool, error)
	CreateAgentKey(context.Context, domain.AgentKey) (domain.AgentKey, error)
	CreateAgentKeyWithAudit(context.Context, domain.AgentKey, domain.AuditEvent) (domain.AgentKey, error)
	ListAgentKeys(context.Context, ManagementScope) ([]domain.AgentKey, error)
	RevokeAgentKey(context.Context, string, time.Time) (domain.AgentKey, bool, error)
	RevokeAgentKeyWithAudit(context.Context, string, time.Time, domain.AuditEvent) (domain.AgentKey, bool, error)
	FindAgentByKeyHash(context.Context, string, time.Time) (domain.Agent, bool, error)
	CreateAccessGrant(context.Context, domain.AccessGrant) (domain.AccessGrant, error)
	CreateAccessGrantWithAudit(context.Context, domain.AccessGrant, domain.AuditEvent) (domain.AccessGrant, error)
	ListAccessGrants(context.Context, ManagementScope) ([]domain.AccessGrant, error)
	RevokeAccessGrant(context.Context, string, time.Time) (domain.AccessGrant, bool, error)
	RevokeAccessGrantWithAudit(context.Context, string, time.Time, domain.AuditEvent) (domain.AccessGrant, bool, error)
	HasGrant(context.Context, string, string, string, string, time.Time) bool
	CreateRoutePolicy(context.Context, domain.RoutePolicy) (domain.RoutePolicy, error)
	CreateRoutePolicyWithAudit(context.Context, domain.RoutePolicy, domain.AuditEvent) (domain.RoutePolicy, error)
	ListRoutePolicies(context.Context, ManagementScope) ([]domain.RoutePolicy, error)
	GetRoutePolicy(context.Context, string) (domain.RoutePolicy, bool, error)
	UpdateRoutePolicy(context.Context, domain.RoutePolicy) (domain.RoutePolicy, bool, error)
	UpdateRoutePolicyWithAudit(context.Context, domain.RoutePolicy, domain.AuditEvent) (domain.RoutePolicy, bool, error)
	DisableRoutePolicy(context.Context, string, time.Time) (domain.RoutePolicy, bool, error)
	DisableRoutePolicyWithAudit(context.Context, string, time.Time, domain.AuditEvent) (domain.RoutePolicy, bool, error)
	EvaluateRouteAccess(context.Context, string, string, string, string, time.Time) (domain.RouteAccessDecision, error)
	AppendTrace(context.Context, domain.TraceEvent) (domain.TraceEvent, error)
	ListTraces(context.Context, TraceFilter) ([]domain.TraceEvent, error)
	AppendAuditEvent(context.Context, domain.AuditEvent) (domain.AuditEvent, error)
	ListAuditEvents(context.Context, AuditEventFilter) ([]domain.AuditEvent, error)
}

type ManagementScope struct {
	TenantID    string
	WorkspaceID string
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

type Memory struct {
	mu       sync.RWMutex
	agents   map[string]domain.Agent
	keys     map[string]domain.AgentKey
	grants   map[string]domain.AccessGrant
	policies map[string]domain.RoutePolicy
	traces   []domain.TraceEvent
	audits   []domain.AuditEvent
}

func NewMemory() *Memory {
	return &Memory{
		agents:   make(map[string]domain.Agent),
		keys:     make(map[string]domain.AgentKey),
		grants:   make(map[string]domain.AccessGrant),
		policies: make(map[string]domain.RoutePolicy),
	}
}

func (m *Memory) CreateAgent(_ context.Context, agent domain.Agent) (domain.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents[agent.ID] = agent
	return agent, nil
}

func (m *Memory) CreateAgentWithAudit(_ context.Context, agent domain.Agent, audit domain.AuditEvent) (domain.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents[agent.ID] = agent
	m.audits = append(m.audits, audit)
	return agent, nil
}

func (m *Memory) ListAgents(_ context.Context, filter AgentFilter) ([]domain.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rows := make([]domain.Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		if agentMatchesScope(agent, filter.ManagementScope) {
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

func (m *Memory) UpdateAgentWithAudit(_ context.Context, agent domain.Agent, audit domain.AuditEvent) (domain.Agent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.agents[agent.ID]
	if !ok {
		return domain.Agent{}, false, nil
	}
	agent.Credentials = existing.Credentials
	agent.CredentialVersion = existing.CredentialVersion
	m.agents[agent.ID] = agent
	m.audits = append(m.audits, audit)
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

func (m *Memory) RotateAgentCredentialsWithAudit(_ context.Context, id string, credentials map[string]string, now time.Time, audit domain.AuditEvent) (domain.Agent, bool, error) {
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
	m.audits = append(m.audits, audit)
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

func (m *Memory) DisableAgentWithAudit(_ context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.Agent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[id]
	if !ok {
		return domain.Agent{}, false, nil
	}
	agent.Status = domain.AgentStatusDisabled
	agent.UpdatedAt = now
	m.agents[id] = agent
	m.audits = append(m.audits, audit)
	return agent, true, nil
}

func (m *Memory) CreateAgentKey(_ context.Context, key domain.AgentKey) (domain.AgentKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[key.ID] = key
	return key, nil
}

func (m *Memory) CreateAgentKeyWithAudit(_ context.Context, key domain.AgentKey, audit domain.AuditEvent) (domain.AgentKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[key.ID] = key
	m.audits = append(m.audits, audit)
	return key, nil
}

func (m *Memory) ListAgentKeys(_ context.Context, scope ManagementScope) ([]domain.AgentKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rows := make([]domain.AgentKey, 0, len(m.keys))
	for _, key := range m.keys {
		agent, ok := m.agents[key.AgentID]
		if !ok || !agentMatchesScope(agent, scope) {
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

func (m *Memory) RevokeAgentKeyWithAudit(_ context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.AgentKey, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.keys[id]
	if !ok {
		return domain.AgentKey{}, false, nil
	}
	key.RevokedAt = now
	m.keys[id] = key
	m.audits = append(m.audits, audit)
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

func (m *Memory) CreateAccessGrantWithAudit(_ context.Context, grant domain.AccessGrant, audit domain.AuditEvent) (domain.AccessGrant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.grants[grant.ID] = grant
	m.audits = append(m.audits, audit)
	return grant, nil
}

func (m *Memory) ListAccessGrants(_ context.Context, scope ManagementScope) ([]domain.AccessGrant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rows := make([]domain.AccessGrant, 0, len(m.grants))
	for _, grant := range m.grants {
		if !m.grantMatchesScope(grant, scope) {
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

func (m *Memory) RevokeAccessGrantWithAudit(_ context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.AccessGrant, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	grant, ok := m.grants[id]
	if !ok {
		return domain.AccessGrant{}, false, nil
	}
	grant.RevokedAt = now
	m.grants[id] = grant
	m.audits = append(m.audits, audit)
	return grant, true, nil
}

func (m *Memory) HasGrant(_ context.Context, callerID string, targetID string, routeType string, routeKey string, now time.Time) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hasGrantLocked(callerID, targetID, routeType, routeKey, now)
}

func (m *Memory) CreateRoutePolicy(_ context.Context, policy domain.RoutePolicy) (domain.RoutePolicy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[policy.ID] = policy
	return policy, nil
}

func (m *Memory) CreateRoutePolicyWithAudit(_ context.Context, policy domain.RoutePolicy, audit domain.AuditEvent) (domain.RoutePolicy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[policy.ID] = policy
	m.audits = append(m.audits, audit)
	return policy, nil
}

func (m *Memory) ListRoutePolicies(_ context.Context, scope ManagementScope) ([]domain.RoutePolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rows := make([]domain.RoutePolicy, 0, len(m.policies))
	for _, policy := range m.policies {
		if !routePolicyMatchesScope(policy, scope) {
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
	policy.TenantID = existing.TenantID
	policy.WorkspaceID = existing.WorkspaceID
	policy.CallerID = existing.CallerID
	policy.TargetID = existing.TargetID
	policy.CreatedAt = existing.CreatedAt
	m.policies[policy.ID] = policy
	return policy, true, nil
}

func (m *Memory) UpdateRoutePolicyWithAudit(_ context.Context, policy domain.RoutePolicy, audit domain.AuditEvent) (domain.RoutePolicy, bool, error) {
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
	m.audits = append(m.audits, audit)
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

func (m *Memory) DisableRoutePolicyWithAudit(_ context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.RoutePolicy, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	policy, ok := m.policies[id]
	if !ok {
		return domain.RoutePolicy{}, false, nil
	}
	policy.Status = domain.RoutePolicyStatusDisabled
	policy.UpdatedAt = now
	m.policies[id] = policy
	m.audits = append(m.audits, audit)
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
		if !m.traceMatchesScope(trace, filter.ManagementScope) {
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
		if !auditEventMatchesScope(event, filter.ManagementScope) {
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

func (m *Memory) grantMatchesScope(grant domain.AccessGrant, scope ManagementScope) bool {
	if scope.TenantID == "" && scope.WorkspaceID == "" {
		return true
	}
	caller, callerOK := m.agents[grant.CallerID]
	target, targetOK := m.agents[grant.TargetID]
	return (callerOK && agentMatchesScope(caller, scope)) || (targetOK && agentMatchesScope(target, scope))
}

func (m *Memory) traceMatchesScope(trace domain.TraceEvent, scope ManagementScope) bool {
	if scope.TenantID == "" && scope.WorkspaceID == "" {
		return true
	}
	caller, callerOK := m.agents[trace.CallerID]
	target, targetOK := m.agents[trace.TargetID]
	return (callerOK && agentMatchesScope(caller, scope)) || (targetOK && agentMatchesScope(target, scope))
}

func routePolicyMatchesScope(policy domain.RoutePolicy, scope ManagementScope) bool {
	if scope.TenantID != "" && policy.TenantID != scope.TenantID {
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
	}
}

func agentMatchesScope(agent domain.Agent, scope ManagementScope) bool {
	if scope.TenantID != "" && agent.TenantID != scope.TenantID {
		return false
	}
	if scope.WorkspaceID != "" && agent.WorkspaceID != scope.WorkspaceID {
		return false
	}
	return true
}

func auditEventMatchesScope(event domain.AuditEvent, scope ManagementScope) bool {
	if scope.TenantID != "" && event.TenantID != scope.TenantID {
		return false
	}
	if scope.WorkspaceID != "" && event.WorkspaceID != scope.WorkspaceID {
		return false
	}
	return true
}
