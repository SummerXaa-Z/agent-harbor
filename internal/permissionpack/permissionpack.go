package permissionpack

import (
	"fmt"
	"strings"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

var templates = []domain.PermissionPackageTemplate{
	{
		ID:                   "sales-readonly",
		Version:              1,
		Name:                 "Sales read-only",
		Summary:              "Allow CRM reads for a scoped sales tenant while blocking exports, deletes, admin actions, and restricted data.",
		AllowedActions:       []domain.CapabilityAction{domain.CapabilityActionRead},
		BlockedActions:       []domain.CapabilityAction{domain.CapabilityActionExport, domain.CapabilityActionDelete, domain.CapabilityActionAdmin},
		BlockedRisks:         []domain.CapabilityRisk{domain.CapabilityRiskHigh, domain.CapabilityRiskCritical},
		BlockedSensitivities: []domain.CapabilitySensitivity{domain.CapabilitySensitivityRestricted},
		DefaultDataDomain:    "crm",
		Guardrails: []domain.PermissionPackageSimulationRow{
			{
				ID:               "guardrail:cross-region-data",
				CapabilityKey:    "cross-region-data",
				ExpectedDecision: domain.PermissionPackageDecisionDeny,
				Reason:           "Data scope keeps the package inside the selected region.",
				ReasonKey:        "permissionSimulation.guardrailRegion",
			},
			{
				ID:               "guardrail:sensitive-finance-fields",
				CapabilityKey:    "sensitive-finance-fields",
				ExpectedDecision: domain.PermissionPackageDecisionDeny,
				Reason:           "Sales read-only does not include finance field access.",
				ReasonKey:        "permissionSimulation.guardrailFinance",
			},
		},
	},
	{
		ID:                   "support-ticket-triage",
		Version:              1,
		Name:                 "Support ticket triage",
		Summary:              "Allow ticket reads and bounded updates while blocking exports, deletes, and admin operations.",
		AllowedActions:       []domain.CapabilityAction{domain.CapabilityActionRead, domain.CapabilityActionWrite},
		BlockedActions:       []domain.CapabilityAction{domain.CapabilityActionExport, domain.CapabilityActionDelete, domain.CapabilityActionAdmin},
		BlockedRisks:         []domain.CapabilityRisk{domain.CapabilityRiskCritical},
		BlockedSensitivities: []domain.CapabilitySensitivity{domain.CapabilitySensitivityRestricted},
		DefaultDataDomain:    "support",
		Guardrails: []domain.PermissionPackageSimulationRow{
			{
				ID:               "guardrail:delete-ticket",
				CapabilityKey:    "delete-ticket",
				ExpectedDecision: domain.PermissionPackageDecisionDeny,
				Reason:           "Support triage does not include destructive operations.",
				ReasonKey:        "permissionSimulation.guardrailDeleteTicket",
			},
		},
	},
	{
		ID:                   "analytics-sandbox",
		Version:              1,
		Name:                 "Analytics sandbox",
		Summary:              "Allow read and execute capabilities for sandbox analysis while blocking writes, exports, and production admin actions.",
		AllowedActions:       []domain.CapabilityAction{domain.CapabilityActionRead, domain.CapabilityActionExecute},
		BlockedActions:       []domain.CapabilityAction{domain.CapabilityActionWrite, domain.CapabilityActionExport, domain.CapabilityActionDelete, domain.CapabilityActionAdmin},
		BlockedRisks:         []domain.CapabilityRisk{domain.CapabilityRiskHigh, domain.CapabilityRiskCritical},
		BlockedSensitivities: []domain.CapabilitySensitivity{domain.CapabilitySensitivityRestricted},
		DefaultDataDomain:    "analytics",
		Guardrails: []domain.PermissionPackageSimulationRow{
			{
				ID:               "guardrail:production-write",
				CapabilityKey:    "production-write",
				ExpectedDecision: domain.PermissionPackageDecisionDeny,
				Reason:           "Analytics sandbox cannot write production data.",
				ReasonKey:        "permissionSimulation.guardrailProductionWrite",
			},
		},
	},
	{
		ID:                   "audit-readonly",
		Version:              1,
		Name:                 "Audit read-only",
		Summary:              "Allow low-risk reads for audit review while blocking mutations, exports, and restricted data.",
		AllowedActions:       []domain.CapabilityAction{domain.CapabilityActionRead},
		BlockedActions:       []domain.CapabilityAction{domain.CapabilityActionWrite, domain.CapabilityActionExport, domain.CapabilityActionDelete, domain.CapabilityActionAdmin},
		BlockedRisks:         []domain.CapabilityRisk{domain.CapabilityRiskHigh, domain.CapabilityRiskCritical},
		BlockedSensitivities: []domain.CapabilitySensitivity{domain.CapabilitySensitivityRestricted},
		DefaultDataDomain:    "audit",
		Guardrails: []domain.PermissionPackageSimulationRow{
			{
				ID:               "guardrail:audit-export",
				CapabilityKey:    "audit-export",
				ExpectedDecision: domain.PermissionPackageDecisionDeny,
				Reason:           "Audit read-only requires a separate export approval.",
				ReasonKey:        "permissionSimulation.guardrailAuditExport",
			},
		},
	},
}

func Templates() []domain.PermissionPackageTemplate {
	return cloneTemplates(templates)
}

func BuildDraft(input domain.PermissionPackageDraftRequest, capabilities []domain.Capability) (domain.PermissionPackageDraft, error) {
	input = normalizeInput(input)
	template, ok := templateByID(input.TemplateID)
	if !ok {
		return domain.PermissionPackageDraft{}, domain.BadRequest("VALIDATION_FAILED", fmt.Sprintf("permission package template %q not found", input.TemplateID))
	}
	targetCapabilities := make([]domain.Capability, 0, len(capabilities))
	for _, capability := range capabilities {
		if capability.TargetID == input.TargetID {
			targetCapabilities = append(targetCapabilities, capability)
		}
	}
	allowedCapabilities := make([]domain.Capability, 0, len(targetCapabilities))
	blockedCapabilities := make([]domain.Capability, 0, len(targetCapabilities))
	for _, capability := range targetCapabilities {
		if isBlockedByTemplate(capability, template) {
			blockedCapabilities = append(blockedCapabilities, capability)
			continue
		}
		if containsAction(template.AllowedActions, capability.Action) {
			allowedCapabilities = append(allowedCapabilities, capability)
		}
	}
	dataScopes := buildDataScopes(input, template, allowedCapabilities)
	readiness := buildReadiness(input, allowedCapabilities, dataScopes)
	return domain.PermissionPackageDraft{
		ID:                  draftID(input),
		Input:               input,
		Template:            template,
		AllowedCapabilities: cloneCapabilities(allowedCapabilities),
		BlockedCapabilities: cloneCapabilities(blockedCapabilities),
		DataScopes:          dataScopes,
		Readiness:           readiness,
		SimulationRows:      buildSimulationRows(allowedCapabilities, blockedCapabilities, template),
	}, nil
}

func normalizeInput(input domain.PermissionPackageDraftRequest) domain.PermissionPackageDraftRequest {
	input.CallerInstanceID = strings.TrimSpace(input.CallerInstanceID)
	input.Region = strings.TrimSpace(input.Region)
	input.RequestText = strings.TrimSpace(input.RequestText)
	input.SubjectSelector = strings.TrimSpace(input.SubjectSelector)
	input.TargetID = strings.TrimSpace(input.TargetID)
	input.TemplateID = strings.TrimSpace(input.TemplateID)
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.WorkspaceID = strings.TrimSpace(input.WorkspaceID)
	return input
}

func templateByID(id string) (domain.PermissionPackageTemplate, bool) {
	for _, template := range templates {
		if template.ID == id {
			return cloneTemplate(template), true
		}
	}
	return domain.PermissionPackageTemplate{}, false
}

func isBlockedByTemplate(capability domain.Capability, template domain.PermissionPackageTemplate) bool {
	return containsAction(template.BlockedActions, capability.Action) ||
		containsRisk(template.BlockedRisks, capability.RiskLevel) ||
		containsSensitivity(template.BlockedSensitivities, capability.Sensitivity)
}

func buildDataScopes(input domain.PermissionPackageDraftRequest, template domain.PermissionPackageTemplate, allowedCapabilities []domain.Capability) []domain.DataScope {
	dataDomain := ""
	for _, capability := range allowedCapabilities {
		for _, domainName := range capability.DataDomains {
			if strings.TrimSpace(domainName) != "" {
				dataDomain = strings.TrimSpace(domainName)
				break
			}
		}
		if dataDomain != "" {
			break
		}
		for _, scope := range capability.DataScopes {
			if strings.TrimSpace(scope.DataDomain) != "" {
				dataDomain = strings.TrimSpace(scope.DataDomain)
				break
			}
		}
		if dataDomain != "" {
			break
		}
	}
	if dataDomain == "" {
		dataDomain = template.DefaultDataDomain
	}
	scope := domain.DataScope{DataDomain: dataDomain}
	if input.Region != "" {
		scope.Region = input.Region
	}
	if input.TenantID != "" {
		scope.TenantFilter = fmt.Sprintf("tenant_id = '%s'", strings.ReplaceAll(input.TenantID, "'", "''"))
	}
	return []domain.DataScope{scope}
}

func buildReadiness(input domain.PermissionPackageDraftRequest, allowedCapabilities []domain.Capability, dataScopes []domain.DataScope) domain.PermissionPackageReadiness {
	missingFields := []string{}
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "tenantId", value: input.TenantID},
		{name: "workspaceId", value: input.WorkspaceID},
		{name: "callerInstanceId", value: input.CallerInstanceID},
		{name: "targetId", value: input.TargetID},
		{name: "templateId", value: input.TemplateID},
	} {
		if field.value == "" {
			missingFields = append(missingFields, field.name)
		}
	}
	warnings := []string{}
	if len(allowedCapabilities) == 0 {
		warnings = append(warnings, "No matching allowed capabilities for the selected target.")
	}
	for _, capability := range allowedCapabilities {
		if _, ok := domain.EffectiveDataScopes(capability.DataScopes, dataScopes); !ok {
			warnings = append(warnings, fmt.Sprintf("Permission package data scopes exceed capability boundary for %s.", capability.Key))
		}
	}
	return domain.PermissionPackageReadiness{
		CanApply:      len(missingFields) == 0 && len(warnings) == 0,
		MissingFields: missingFields,
		Warnings:      warnings,
	}
}

func buildSimulationRows(allowedCapabilities []domain.Capability, blockedCapabilities []domain.Capability, template domain.PermissionPackageTemplate) []domain.PermissionPackageSimulationRow {
	rows := make([]domain.PermissionPackageSimulationRow, 0, len(allowedCapabilities)+len(blockedCapabilities)+len(template.Guardrails))
	for _, capability := range allowedCapabilities {
		rows = append(rows, domain.PermissionPackageSimulationRow{
			ID:               "allow:" + capability.ID,
			CapabilityID:     capability.ID,
			CapabilityKey:    capability.Key,
			ExpectedDecision: domain.PermissionPackageDecisionAllow,
			Reason:           fmt.Sprintf("%s allows %s capability %s.", template.Name, capability.Action, capability.Key),
			ReasonKey:        "permissionSimulation.allowCapability",
			ReasonValues: map[string]string{
				"action":     string(capability.Action),
				"capability": capability.Key,
				"packageId":  template.ID,
			},
		})
	}
	for _, capability := range blockedCapabilities {
		rows = append(rows, domain.PermissionPackageSimulationRow{
			ID:               "deny:" + capability.ID,
			CapabilityID:     capability.ID,
			CapabilityKey:    capability.Key,
			ExpectedDecision: domain.PermissionPackageDecisionDeny,
			Reason:           fmt.Sprintf("%s blocks %s capability %s.", template.Name, capability.Action, capability.Key),
			ReasonKey:        "permissionSimulation.blockCapability",
			ReasonValues: map[string]string{
				"action":     string(capability.Action),
				"capability": capability.Key,
				"packageId":  template.ID,
			},
		})
	}
	for _, guardrail := range template.Guardrails {
		rows = append(rows, guardrail)
	}
	return rows
}

func draftID(input domain.PermissionPackageDraftRequest) string {
	parts := []string{"pkgdraft", input.TemplateID, input.TenantID, input.WorkspaceID, input.CallerInstanceID, input.TargetID}
	safe := make([]string, 0, len(parts))
	for _, part := range parts {
		sanitized := sanitizeIDPart(part)
		if sanitized != "" {
			safe = append(safe, sanitized)
		}
	}
	return strings.Join(safe, "-")
}

func sanitizeIDPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastHyphen := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen && out.Len() > 0 {
			out.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func containsAction(values []domain.CapabilityAction, target domain.CapabilityAction) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsRisk(values []domain.CapabilityRisk, target domain.CapabilityRisk) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsSensitivity(values []domain.CapabilitySensitivity, target domain.CapabilitySensitivity) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func cloneTemplates(values []domain.PermissionPackageTemplate) []domain.PermissionPackageTemplate {
	out := make([]domain.PermissionPackageTemplate, 0, len(values))
	for _, value := range values {
		out = append(out, cloneTemplate(value))
	}
	return out
}

func cloneTemplate(value domain.PermissionPackageTemplate) domain.PermissionPackageTemplate {
	value.AllowedActions = append([]domain.CapabilityAction(nil), value.AllowedActions...)
	value.BlockedActions = append([]domain.CapabilityAction(nil), value.BlockedActions...)
	value.BlockedRisks = append([]domain.CapabilityRisk(nil), value.BlockedRisks...)
	value.BlockedSensitivities = append([]domain.CapabilitySensitivity(nil), value.BlockedSensitivities...)
	value.Guardrails = append([]domain.PermissionPackageSimulationRow(nil), value.Guardrails...)
	return value
}

func cloneCapabilities(values []domain.Capability) []domain.Capability {
	out := make([]domain.Capability, 0, len(values))
	for _, value := range values {
		value.NativeScopes = append([]string(nil), value.NativeScopes...)
		value.DataDomains = append([]string(nil), value.DataDomains...)
		value.DataScopes = append([]domain.DataScope(nil), value.DataScopes...)
		out = append(out, value)
	}
	return out
}
