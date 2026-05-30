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
	ListAgents(context.Context, AgentFilter) ([]domain.Agent, error)
	GetAgent(context.Context, string) (domain.Agent, bool, error)
	UpdateAgent(context.Context, domain.Agent) (domain.Agent, bool, error)
	DisableAgent(context.Context, string, time.Time) (domain.Agent, bool, error)
	CreateAgentKey(context.Context, domain.AgentKey) (domain.AgentKey, error)
	ListAgentKeys(context.Context, ManagementScope) ([]domain.AgentKey, error)
	RevokeAgentKey(context.Context, string, time.Time) (domain.AgentKey, bool, error)
	FindAgentByKeyHash(context.Context, string, time.Time) (domain.Agent, bool, error)
	CreateAccessGrant(context.Context, domain.AccessGrant) (domain.AccessGrant, error)
	ListAccessGrants(context.Context, ManagementScope) ([]domain.AccessGrant, error)
	RevokeAccessGrant(context.Context, string, time.Time) (domain.AccessGrant, bool, error)
	HasGrant(context.Context, string, string, string, string, time.Time) bool
	AppendTrace(context.Context, domain.TraceEvent) (domain.TraceEvent, error)
	ListTraces(context.Context, TraceFilter) ([]domain.TraceEvent, error)
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

type Memory struct {
	mu     sync.RWMutex
	agents map[string]domain.Agent
	keys   map[string]domain.AgentKey
	grants map[string]domain.AccessGrant
	traces []domain.TraceEvent
}

func NewMemory() *Memory {
	return &Memory{
		agents: make(map[string]domain.Agent),
		keys:   make(map[string]domain.AgentKey),
		grants: make(map[string]domain.AccessGrant),
	}
}

func (m *Memory) CreateAgent(_ context.Context, agent domain.Agent) (domain.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents[agent.ID] = agent
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
	if _, ok := m.agents[agent.ID]; !ok {
		return domain.Agent{}, false, nil
	}
	m.agents[agent.ID] = agent
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

func (m *Memory) CreateAgentKey(_ context.Context, key domain.AgentKey) (domain.AgentKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[key.ID] = key
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

func (m *Memory) HasGrant(_ context.Context, callerID string, targetID string, routeType string, routeKey string, now time.Time) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
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

func agentMatchesScope(agent domain.Agent, scope ManagementScope) bool {
	if scope.TenantID != "" && agent.TenantID != scope.TenantID {
		return false
	}
	if scope.WorkspaceID != "" && agent.WorkspaceID != scope.WorkspaceID {
		return false
	}
	return true
}
