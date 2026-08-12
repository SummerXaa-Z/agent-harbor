package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

const agentHarborContextHeader = "X-AgentHarbor-Context"

type agentHarborTaskResultContext struct {
	Kind       string             `json:"kind"`
	Summary    string             `json:"summary"`
	DataScopes []domain.DataScope `json:"dataScopes"`
}

type agentHarborContextPayload struct {
	SchemaVersion     string                        `json:"schemaVersion"`
	PlatformID        string                        `json:"platformId"`
	TenantID          string                        `json:"tenantId"`
	WorkspaceID       string                        `json:"workspaceId"`
	TargetID          string                        `json:"targetId"`
	CallerInstanceID  string                        `json:"callerInstanceId"`
	CallerSubject     string                        `json:"callerSubject,omitempty"`
	CapabilityID      string                        `json:"capabilityId"`
	CapabilityKey     string                        `json:"capabilityKey"`
	ToolName          string                        `json:"toolName"`
	DataScopes        []domain.DataScope            `json:"dataScopes"`
	TaskResultContext *agentHarborTaskResultContext `json:"taskResultContext,omitempty"`
}

func agentHarborContextHeaderValue(identity runtimeIdentity, targetID string, capability domain.Capability, decision domain.CapabilityAccessDecision, toolName string, taskResultContext *agentHarborTaskResultContext) (string, error) {
	dataScopes := domain.CloneDataScopes(decision.DataScopes)
	if dataScopes == nil {
		dataScopes = []domain.DataScope{}
	}
	payload := agentHarborContextPayload{
		SchemaVersion:     "2026-06-01",
		PlatformID:        identity.PlatformID,
		TenantID:          identity.TenantID,
		WorkspaceID:       identity.WorkspaceID,
		TargetID:          targetID,
		CallerInstanceID:  identity.CallerInstanceID,
		CallerSubject:     identity.SubjectID,
		CapabilityID:      capability.ID,
		CapabilityKey:     capability.Key,
		ToolName:          toolName,
		DataScopes:        dataScopes,
		TaskResultContext: taskResultContext,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}

func isReservedAgentHarborHeader(name string) bool {
	name = strings.TrimSpace(name)
	return strings.EqualFold(name, agentHarborContextHeader) ||
		strings.EqualFold(name, agentHarborTaskResultMemoryIDHeader)
}
