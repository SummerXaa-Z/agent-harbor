import type { Translator } from "./consolePresenters";

export const systemCapabilityLabelKeyByName: Record<string, string> = {
  capability_data_domain_classification_v1: "systemCapability.capabilityDataDomainClassification",
  management_mcp_tools_metadata_v4: "systemCapability.managementMcpToolsMetadataV4",
  permission_package_applications: "systemCapability.permissionPackageApplications",
  permission_package_application_health: "systemCapability.permissionPackageApplicationHealth",
  permission_package_application_impact: "systemCapability.permissionPackageApplicationImpact",
  permission_package_access_handoff_v1: "systemCapability.permissionPackageAccessHandoff",
  permission_package_access_handoff_tokens_v1: "systemCapability.permissionPackageAccessHandoffTokens",
  permission_package_apply_preflight: "systemCapability.permissionPackageApplyPreflight",
  permission_package_approval_requests: "systemCapability.permissionPackageApprovalRequests",
  permission_package_approval_withdraw: "systemCapability.permissionPackageApprovalWithdraw",
  permission_package_consumed_approval_recovery: "systemCapability.permissionPackageConsumedApprovalRecovery",
  permission_package_production_readiness: "systemCapability.permissionPackageProductionReadiness",
  permission_package_requested_capability_v1: "systemCapability.permissionPackageRequestedCapability"
};

export function systemCapabilityLabelKeys(capabilities: string[] | undefined): string[] {
  if (!Array.isArray(capabilities) || capabilities.length === 0) return [];
  const keys = capabilities.map((capability) => systemCapabilityLabelKeyByName[capability]);
  return keys.every(isDefinedString) ? keys : [];
}

export function systemCapabilityLabels(capabilities: string[] | undefined, t: Translator): string[] {
  return systemCapabilityLabelKeys(capabilities).map((key) => t(key));
}

function isDefinedString(value: string | undefined): value is string {
  return Boolean(value);
}
