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
    return tx(t, "message.apiContractIncompatible", {
      capabilities: result.missingCapabilities?.join(", ") || result.message
    });
  }
  return `${label}: ${result.message}`;
}
