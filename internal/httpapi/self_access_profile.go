package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const (
	selfAccessProfileKeyKindAgent         = "agent"
	selfAccessProfileKeyKindAccessHandoff = "access_handoff"
)

type selfAccessProfileResponse struct {
	Caller      selfAccessProfileCaller   `json:"caller"`
	SubjectID   string                    `json:"subjectId,omitempty"`
	Key         selfAccessProfileKey      `json:"key"`
	Handoff     *selfAccessProfileHandoff `json:"handoff,omitempty"`
	Targets     []selfAccessProfileTarget `json:"targets"`
	GeneratedAt time.Time                 `json:"generatedAt"`
}

type selfAccessProfileCaller struct {
	InstanceID  string `json:"instanceId"`
	Name        string `json:"name"`
	TenantID    string `json:"tenantId"`
	WorkspaceID string `json:"workspaceId"`
	Status      string `json:"status"`
}

type selfAccessProfileKey struct {
	Kind      string    `json:"kind"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type selfAccessProfileHandoff struct {
	HandoffID            string   `json:"handoffId"`
	ApplicationID        string   `json:"applicationId"`
	TemplateID           string   `json:"templateId"`
	TargetID             string   `json:"targetId"`
	SubjectSelector      string   `json:"subjectSelector,omitempty"`
	AllowedCapabilityIDs []string `json:"allowedCapabilityIds"`
}

type selfAccessProfileTarget struct {
	TargetID     string                        `json:"targetId"`
	TargetName   string                        `json:"targetName,omitempty"`
	ChannelType  string                        `json:"channelType,omitempty"`
	Capabilities []selfAccessProfileCapability `json:"capabilities"`
}

type selfAccessProfileCapability struct {
	ID          string             `json:"id"`
	Key         string             `json:"key"`
	DisplayName string             `json:"displayName,omitempty"`
	Type        string             `json:"type"`
	RiskLevel   string             `json:"riskLevel,omitempty"`
	DataDomains []string           `json:"dataDomains,omitempty"`
	DataScopes  []domain.DataScope `json:"dataScopes,omitempty"`
}

// getSelfAccessProfile answers "what can this credential actually call" for
// the calling agent key itself. Ordinary keys see every capability that
// evaluates as allowed for their tenant's entitlement chain — including
// targets owned by ancestor tenants, since entitlements reference targets
// regardless of the target's own tenant; access handoff keys are bounded by
// their application binding, mirroring the exact surface the governed data
// plane would serve. It is deliberately read-only: recovery of expired or
// revoked tokens still routes through an administrator until the 0.3.x My
// Access request_access loop lands.
func (s *Server) getSelfAccessProfile(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	key := agentKeyFromContext(r.Context())
	identity := identityFromRequest(r, caller)

	handoffApplication, err := s.accessHandoffApplicationForKey(r.Context(), key, identity.SubjectID)
	if err != nil {
		writeError(w, err)
		return
	}

	targetIDs := []string{}
	seenTargets := map[string]struct{}{}
	if handoffApplication != nil {
		targetIDs = append(targetIDs, handoffApplication.TargetID)
		seenTargets[handoffApplication.TargetID] = struct{}{}
	} else {
		entitlements, err := s.repo.ListTenantEntitlements(r.Context(), store.EntitlementFilter{
			ManagementScope: store.ManagementScope{TenantID: caller.TenantID},
		})
		if err != nil {
			writeError(w, err)
			return
		}
		for _, entitlement := range entitlements {
			if _, seen := seenTargets[entitlement.TargetID]; seen {
				continue
			}
			seenTargets[entitlement.TargetID] = struct{}{}
			targetIDs = append(targetIDs, entitlement.TargetID)
		}
	}
	now := s.now()
	profileTargets := []selfAccessProfileTarget{}
	for _, targetID := range targetIDs {
		target, ok, err := s.repo.GetAgent(r.Context(), targetID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !ok || target.Status != domain.AgentStatusActive {
			continue
		}
		capabilities, err := s.repo.ListCapabilities(r.Context(), store.CapabilityFilter{TargetID: target.ID})
		if err != nil {
			writeError(w, err)
			return
		}
		allowed := []selfAccessProfileCapability{}
		for _, capability := range capabilities {
			if handoffApplication != nil && !stringSliceContains(handoffApplication.AllowedCapabilityIDs, capability.ID) {
				continue
			}
			decision, err := s.repo.EvaluateCapabilityAccess(r.Context(), store.CapabilityAccessRequest{
				TenantID:         identity.TenantID,
				WorkspaceID:      identity.WorkspaceID,
				CallerInstanceID: identity.CallerInstanceID,
				SubjectID:        identity.SubjectID,
				TargetID:         target.ID,
				CapabilityID:     capability.ID,
				Now:              now,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			if !decision.Allowed {
				continue
			}
			allowed = append(allowed, selfAccessProfileCapability{
				ID:          capability.ID,
				Key:         capability.Key,
				DisplayName: capability.DisplayName,
				Type:        string(capability.Type),
				RiskLevel:   string(capability.RiskLevel),
				DataDomains: capability.DataDomains,
				DataScopes:  decision.DataScopes,
			})
		}
		if len(allowed) == 0 {
			continue
		}
		profileTargets = append(profileTargets, selfAccessProfileTarget{
			TargetID:     target.ID,
			TargetName:   target.Name,
			ChannelType:  string(target.ChannelType),
			Capabilities: allowed,
		})
	}

	response := selfAccessProfileResponse{
		Caller: selfAccessProfileCaller{
			InstanceID:  caller.ID,
			Name:        caller.Name,
			TenantID:    caller.TenantID,
			WorkspaceID: caller.WorkspaceID,
			Status:      string(caller.Status),
		},
		SubjectID: identity.SubjectID,
		Key: selfAccessProfileKey{
			Kind:      selfAccessProfileKeyKindAgent,
			Name:      key.Name,
			CreatedAt: key.CreatedAt,
			ExpiresAt: key.ExpiresAt,
		},
		Targets:     profileTargets,
		GeneratedAt: now,
	}
	if strings.TrimSpace(key.CreatedForHandoffID) != "" && handoffApplication != nil {
		response.Key.Kind = selfAccessProfileKeyKindAccessHandoff
		response.Handoff = &selfAccessProfileHandoff{
			HandoffID:            strings.TrimSpace(key.CreatedForHandoffID),
			ApplicationID:        handoffApplication.ID,
			TemplateID:           handoffApplication.TemplateID,
			TargetID:             handoffApplication.TargetID,
			SubjectSelector:      handoffApplication.SubjectSelector,
			AllowedCapabilityIDs: handoffApplication.AllowedCapabilityIDs,
		}
	}
	writeJSON(w, http.StatusOK, response)
}
