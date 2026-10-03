package httpapi

import (
	"github.com/SummerXaa-Z/agent-harbor/internal/contracts"
	"net/http"
)

var systemCapabilities = []string{
	"capability_data_domain_classification_v1",
	"permission_package_requested_capability_v1",
	"permission_package_approval_requests",
	"permission_package_approval_withdraw",
	"permission_package_apply_preflight",
	"permission_package_applications",
	"permission_package_application_health",
	"permission_package_application_impact",
	"permission_package_production_readiness",
	"permission_package_access_handoff_v1",
	"permission_package_access_handoff_tokens_v1",
	"permission_package_consumed_approval_recovery",
	"management_mcp_tools_metadata_v4",
	"metrics_daily_v1",
	"target_probe_v1",
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type systemInfoResponse struct {
	Name                     string                                 `json:"name"`
	APIVersion               string                                 `json:"apiVersion"`
	AuthRequired             bool                                   `json:"authRequired"`
	Capabilities             []string                               `json:"capabilities"`
	ManagementMcpToolCatalog systemInfoManagementMcpToolCatalogInfo `json:"managementMcpToolCatalog"`
}

type systemInfoManagementMcpToolCatalogInfo struct {
	MetadataVersion             int      `json:"metadataVersion"`
	RequiredMetadata            []string `json:"requiredMetadata"`
	CatalogDigest               string   `json:"catalogDigest"`
	ToolCount                   int      `json:"toolCount"`
	ConfirmationRequiredTools   int      `json:"confirmationRequiredTools"`
	ToolsWithConfirmationSchema int      `json:"toolsWithConfirmationSchema"`
}

func (s *Server) systemInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, systemInfoResponse{
		Name:                     "AgentHarbor",
		APIVersion:               systemAPIVersion,
		AuthRequired:             !s.developmentAdminBypassActive(),
		Capabilities:             append([]string(nil), systemCapabilities...),
		ManagementMcpToolCatalog: systemInfoManagementMcpToolCatalogSummary(),
	})
}

func systemInfoManagementMcpToolCatalogSummary() systemInfoManagementMcpToolCatalogInfo {
	catalog := managementMCPToolsCatalogResult()
	summary := systemInfoManagementMcpToolCatalogInfo{
		MetadataVersion: catalog.MetadataVersion,
		RequiredMetadata: []string{
			"safety",
			"access",
			"lifecycle",
			"execution",
		},
		CatalogDigest: catalog.CatalogDigest,
		ToolCount:     len(catalog.Tools),
	}
	for _, tool := range catalog.Tools {
		if !managementMCPToolRequiresConfirmation(tool.Name) {
			continue
		}
		summary.ConfirmationRequiredTools++
		if managementMCPToolHasWriteConfirmationInputSchema(tool) {
			summary.ToolsWithConfirmationSchema++
		}
	}
	return summary
}

func managementMCPToolHasWriteConfirmationInputSchema(tool managementMCPTool) bool {
	if !jsonSchemaRequiredIncludesField(tool.InputSchema, "confirmation") {
		return false
	}
	properties, ok := jsonSchemaObjectProperty(tool.InputSchema, "properties")
	if !ok {
		return false
	}
	confirmation, ok := jsonSchemaObjectProperty(properties, "confirmation")
	if !ok || confirmation["type"] != "object" {
		return false
	}
	if !jsonSchemaRequiredIncludesField(confirmation, "confirmed") || !jsonSchemaRequiredIncludesField(confirmation, "reason") {
		return false
	}
	confirmationProperties, ok := jsonSchemaObjectProperty(confirmation, "properties")
	if !ok {
		return false
	}
	confirmed, ok := jsonSchemaObjectProperty(confirmationProperties, "confirmed")
	if !ok || confirmed["type"] != "boolean" {
		return false
	}
	reason, ok := jsonSchemaObjectProperty(confirmationProperties, "reason")
	if !ok || reason["type"] != "string" {
		return false
	}
	return jsonSchemaNumberValueEquals(reason, "minLength", 1) &&
		jsonSchemaNumberValueEquals(reason, "maxLength", maxManagementMCPWriteConfirmationReasonRunes)
}

func jsonSchemaObjectProperty(schema map[string]any, key string) (map[string]any, bool) {
	value, ok := schema[key].(map[string]any)
	return value, ok
}

func jsonSchemaRequiredIncludesField(schema map[string]any, field string) bool {
	required, ok := schema["required"]
	if !ok {
		return false
	}
	switch typed := required.(type) {
	case []string:
		for _, value := range typed {
			if value == field {
				return true
			}
		}
	case []any:
		for _, value := range typed {
			if value == field {
				return true
			}
		}
	}
	return false
}

func jsonSchemaNumberValueEquals(schema map[string]any, key string, expected int) bool {
	switch value := schema[key].(type) {
	case int:
		return value == expected
	case int64:
		return value == int64(expected)
	case float64:
		return value == float64(expected)
	default:
		return false
	}
}

func (s *Server) listProviderContracts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.Providers())
}

func (s *Server) listChannelContracts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.Channels())
}
