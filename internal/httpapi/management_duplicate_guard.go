package httpapi

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const duplicateResourceMutationCode = "DUPLICATE_RESOURCE_MUTATION"
const duplicateAgentKeyWindow = 2 * time.Minute

func duplicateResourceMutation(message string) domain.AppError {
	return domain.Conflict(duplicateResourceMutationCode, message)
}

func (s *Server) rejectDuplicateAgentCreate(ctx context.Context, agent domain.Agent) error {
	rows, err := s.repo.ListAgents(ctx, store.AgentFilter{ManagementScope: store.ManagementScope{
		TenantID:    agent.TenantID,
		WorkspaceID: agent.WorkspaceID,
	}})
	if err != nil {
		return err
	}
	for _, existing := range rows {
		if existing.Status == domain.AgentStatusDisabled {
			continue
		}
		if existing.WorkspaceID == agent.WorkspaceID &&
			strings.EqualFold(strings.TrimSpace(existing.Name), strings.TrimSpace(agent.Name)) &&
			strings.TrimSpace(existing.ChannelType) == strings.TrimSpace(agent.ChannelType) {
			return duplicateResourceMutation("an agent with this name and channel already exists in this tenant workspace")
		}
	}
	return nil
}

func (s *Server) rejectRecentDuplicateAgentKey(ctx context.Context, agent domain.Agent, name string, now time.Time) error {
	rows, err := s.repo.ListAgentKeys(ctx, store.ManagementScope{
		TenantID:    agent.TenantID,
		WorkspaceID: agent.WorkspaceID,
	})
	if err != nil {
		return err
	}
	windowStart := now.Add(-duplicateAgentKeyWindow)
	for _, existing := range rows {
		if existing.AgentID != agent.ID || strings.TrimSpace(existing.Name) != strings.TrimSpace(name) {
			continue
		}
		if !existing.RevokedAt.IsZero() || !existing.ExpiresAt.After(now) {
			continue
		}
		if !existing.CreatedAt.Before(windowStart) {
			return duplicateResourceMutation("an active agent key with this name was just created")
		}
	}
	return nil
}

func sameCredentials(left map[string]string, right map[string]string) bool {
	if left == nil {
		left = map[string]string{}
	}
	if right == nil {
		right = map[string]string{}
	}
	return reflect.DeepEqual(left, right)
}

func (s *Server) rejectDuplicateRoutePolicy(ctx context.Context, policy domain.RoutePolicy) error {
	rows, err := s.repo.ListRoutePolicies(ctx, store.ManagementScope{
		TenantID:    policy.TenantID,
		WorkspaceID: policy.WorkspaceID,
	})
	if err != nil {
		return err
	}
	for _, existing := range rows {
		if existing.Status == domain.RoutePolicyStatusDisabled {
			continue
		}
		if existing.CallerID == policy.CallerID &&
			existing.TargetID == policy.TargetID &&
			existing.RouteType == policy.RouteType &&
			existing.RouteKey == policy.RouteKey &&
			existing.Effect == policy.Effect &&
			existing.Status == policy.Status &&
			existing.Priority == policy.Priority &&
			sameRoutePolicyRetry(existing.Retry, policy.Retry) {
			return duplicateResourceMutation("a matching route policy already exists")
		}
	}
	return nil
}

func sameRoutePolicyRetry(left *domain.RoutePolicyRetry, right *domain.RoutePolicyRetry) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.MaxAttempts == right.MaxAttempts &&
		left.BackoffMs == right.BackoffMs &&
		reflect.DeepEqual(left.StatusCodes, right.StatusCodes)
}
