import type { HealthCheckResult } from "./api";
import type { Translator } from "./consolePresenters";
import { systemCapabilityLabels } from "./systemCapabilityLabels.ts";
import { isManagementMcpToolCatalogContractIssue } from "./systemInfoContract.ts";

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
    const capabilityLabels = systemCapabilityLabels(result.missingCapabilities, t);
    if (capabilityLabels.length === 0) {
      return t("message.apiContractIncompatibleUnknown");
    }
    return tx(t, "message.apiContractIncompatible", {
      capabilities: capabilityLabels.join(", ")
    });
  }
  // Technical causes stay readable for engineers while the zh label keeps the
  // failure classified, matching the upstream-error message convention.
  const detail = result.message.trim();
  return detail ? `${label} (${detail})` : label;
}

function hasOnlyManagementMcpCatalogContractIssues(
  contractIssues: string[] | undefined,
  missingCapabilities: string[] | undefined
): boolean {
  const issues = Array.isArray(contractIssues) ? contractIssues : [];
  const capabilities = Array.isArray(missingCapabilities) ? missingCapabilities : [];
  return capabilities.length === 0 && issues.length > 0 && issues.every(isManagementMcpToolCatalogContractIssue);
}
