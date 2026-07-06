import type { HealthCheckResult } from "./api";
import type { Translator } from "./consolePresenters";

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}

export function healthCheckFailureDetail(t: Translator, label: string, result: HealthCheckResult) {
  if (result.code === "api_contract_unavailable") {
    return t("message.apiContractUnavailable");
  }
  if (result.code === "api_contract_incompatible") {
    if (hasOnlyManagementMcpCatalogContractIssues(result.contractIssues, result.missingCapabilities)) {
      return t("message.apiContractIncompatibleManagementCatalog");
    }
    return tx(t, "message.apiContractIncompatible", {
      capabilities: systemCapabilityLabels(result.missingCapabilities, t).join(", ") || result.message
    });
  }
  return `${label}: ${result.message}`;
}

function hasOnlyManagementMcpCatalogContractIssues(
  contractIssues: string[] | undefined,
  missingCapabilities: string[] | undefined
): boolean {
  const issues = Array.isArray(contractIssues) ? contractIssues : [];
  const capabilities = Array.isArray(missingCapabilities) ? missingCapabilities : [];
  return capabilities.length === 0 && issues.length > 0 && issues.every((issue) => (
    issue === "managementMcpToolCatalog.metadataVersion" ||
    issue.startsWith("managementMcpToolCatalog.requiredMetadata.")
  ));
}

function systemCapabilityLabels(capabilities: string[] | undefined, t: Translator): string[] {
  if (!Array.isArray(capabilities)) return [];
  return capabilities.map((capability) => {
    const key = systemCapabilityLabelKeyByName[capability];
    return key ? t(key) : capability;
  });
}

const systemCapabilityLabelKeyByName: Record<string, string> = {
  management_mcp_tools_metadata_v1: "systemCapability.managementMcpToolsMetadataV1",
  permission_package_applications: "systemCapability.permissionPackageApplications",
  permission_package_application_health: "systemCapability.permissionPackageApplicationHealth",
  permission_package_application_impact: "systemCapability.permissionPackageApplicationImpact",
  permission_package_apply_preflight: "systemCapability.permissionPackageApplyPreflight",
  permission_package_approval_requests: "systemCapability.permissionPackageApprovalRequests",
  permission_package_approval_withdraw: "systemCapability.permissionPackageApprovalWithdraw",
  permission_package_consumed_approval_recovery: "systemCapability.permissionPackageConsumedApprovalRecovery",
  permission_package_production_readiness: "systemCapability.permissionPackageProductionReadiness"
};
