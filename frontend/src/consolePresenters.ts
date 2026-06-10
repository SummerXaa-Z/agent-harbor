import type { MetricTone } from "./consoleMetrics";
import type {
  AccessDecisionExplainResult,
  Agent,
  Capability,
  DataScope
} from "./types";

export type Tone = MetricTone;
export type Translator = (key: string, fallback?: string) => string;

export function translatedValue(t: Translator, value?: string) {
  return value ? t(`value.${value}`, value) : "";
}

export function policyEffectLabel(effect: "allow" | "deny", t: Translator) {
  return effect === "allow" ? t("status.policyAllow") : t("status.policyDeny");
}

export function accessDecisionOutcomeTone(outcome: AccessDecisionExplainResult["outcome"]): Tone {
  return outcome === "allowed" ? "success" : "danger";
}

export function accessDecisionOutcomeLabel(outcome: AccessDecisionExplainResult["outcome"], t: Translator) {
  return outcome === "allowed" ? t("text.decisionAllowed") : t("text.decisionDenied");
}

export function accessDecisionEvidenceTone(status: string): Tone {
  if (status === "matched") return "success";
  if (status === "blocking" || status === "missing" || status === "mismatch") return "danger";
  if (status === "not_approved" || status === "inactive") return "warning";
  return "neutral";
}

export function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false
  });
}

export function agentNameMap(agents: Agent[]) {
  return agents.reduce<Record<string, string>>((acc, agent) => {
    acc[agent.id] = agent.name;
    return acc;
  }, {});
}

export function readableIdentifierLabel(value: string) {
  const normalized = value.trim().replace(/^ws[-_]/, "").replace(/^workspace[-_]/, "");
  if (!normalized) return value;
  return normalized
    .split(/[-_]+/)
    .filter(Boolean)
    .map((part) => {
      const upper = part.toUpperCase();
      if (["AI", "API", "CRM", "MCP", "UI"].includes(upper)) return upper;
      return `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`;
    })
    .join(" ");
}

export function capabilityDisplayName(capability: Pick<Capability, "displayName" | "key">, t: Translator) {
  const fallback = capability.displayName && capability.displayName !== capability.key
    ? capability.displayName
    : readableIdentifierLabel(capability.key);
  return t(`capability.${capability.key}.name`, fallback);
}

export function capabilityKeyDisplayName(capabilityKey: string, t: Translator) {
  return t(`capability.${capabilityKey}.name`, readableIdentifierLabel(capabilityKey));
}

export function capabilitySummaryText(
  capability: Pick<Capability, "dataScopes" | "description" | "key">,
  t: Translator
) {
  return dataScopeText(capability.dataScopes, t)
    || t(`capability.${capability.key}.summary`, capability.description || readableIdentifierLabel(capability.key));
}

export function dataScopeValueLabels(t: Translator): Record<string, string> {
  const values = [
    "analytics",
    "audit",
    "contracts",
    "contract_packages",
    "crm",
    "customers",
    "confidential",
    "eu-west",
    "finance",
    "internal",
    "invoices",
    "packages",
    "restricted",
    "support",
    "tickets",
    "us-east",
    "us-west",
    "华东"
  ];
  return values.reduce<Record<string, string>>((labels, value) => {
    labels[value] = t(`dataScope.${value}`, translatedValue(t, value) || readableIdentifierLabel(value));
    return labels;
  }, {});
}

export function permissionEntityDisplayName(value: string, t: Translator) {
  const normalized = value.trim();
  if (!normalized) return value;
  const demoNames: Record<string, string> = {
    "Core Journey Caller": t("demo.coreJourneyCaller"),
    "Core Journey MCP Target": t("demo.coreJourneyTarget"),
    "Core Journey Project": t("demo.coreJourneyProject"),
    "Core Journey Root": t("demo.coreJourneyRoot"),
    "Core Journey Team": t("demo.coreJourneyTeam"),
    "default": t("text.defaultTenantName"),
    "default-bu": t("demo.permissionRequestApprovalTeam"),
    "default-bu-team": t("demo.permissionRequestApprovalProject"),
    "Default Business Unit": t("demo.permissionRequestApprovalTeam"),
    "Default Tenant": t("text.defaultTenantName"),
    "Default Workspace Team": t("demo.permissionRequestApprovalProject"),
    "Operations Console Caller": t("demo.permissionRequestApprovalCaller"),
    "Permission Package Approval": t("demo.permissionRequestApproval"),
    "Permission Package Approval Root": t("demo.permissionRequestApprovalRoot"),
    "Permission Package Approval Team": t("demo.permissionRequestApprovalTeam"),
    "Permission Package Approval Project": t("demo.permissionRequestApprovalProject"),
    "Permission Package Approval Caller": t("demo.permissionRequestApprovalCaller"),
    "Permission Package Approval MCP Target": t("demo.permissionRequestApprovalTarget"),
    "Permission Request Approval": t("demo.permissionRequestApproval"),
    "Permission Request Approval Root": t("demo.permissionRequestApprovalRoot"),
    "Permission Request Approval Team": t("demo.permissionRequestApprovalTeam"),
    "Permission Request Approval Project": t("demo.permissionRequestApprovalProject"),
    "Permission Request Approval Caller": t("demo.permissionRequestApprovalCaller"),
    "Permission Request Approval MCP Target": t("demo.permissionRequestApprovalTarget"),
    "Security Reviewer": t("accessSubject.securityReviewer.name")
  };
  if (demoNames[normalized]) return demoNames[normalized];
  if (normalized.includes("Permission Package Approval")) {
    return normalized.replaceAll("Permission Package Approval", t("demo.permissionRequestApproval"));
  }
  return normalized;
}

export function capabilityDiscoveryStatusLabel(status: Capability["discoveryStatus"], t: Translator) {
  if (status === "approved") return t("status.capabilityApproved");
  if (status === "deprecated") return t("status.capabilityDeprecated");
  if (status === "removed") return t("status.capabilityRemoved");
  return t("status.capabilityPendingReview");
}

export function dataScopeText(scopes?: DataScope[], t?: Translator) {
  if (!scopes || scopes.length === 0) return "";
  const labels = scopes
    .map((scope) =>
      [scope.dataDomain, scope.dataset, scope.schema, scope.table, scope.field, scope.classification, scope.region]
        .map((part) => (part && t ? t(`dataScope.${part}`, readableIdentifierLabel(part)) : part))
        .filter(Boolean)
        .join("/")
    )
    .filter(Boolean);
  if (labels.length === 0) return "";
  return labels.length > 2 ? `${labels.slice(0, 2).join(", ")} +${labels.length - 2}` : labels.join(", ");
}

export function riskTone(risk: Capability["riskLevel"]): Tone {
  if (risk === "critical" || risk === "high") return "danger";
  if (risk === "medium") return "warning";
  return "success";
}

export function capabilityStatusTone(status: Capability["discoveryStatus"]): Tone {
  if (status === "approved") return "success";
  if (status === "pending_review") return "warning";
  if (status === "removed") return "danger";
  return "neutral";
}
