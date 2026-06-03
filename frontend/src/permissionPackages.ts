import type {
  Capability,
  CapabilityAction,
  CapabilityRisk,
  CapabilitySensitivity,
  DataScope,
} from "./types";

export type PermissionPackageDecision = "allow" | "deny";

export interface PermissionPackageTemplate {
  id: string;
  name: string;
  summary: string;
  allowedActions: CapabilityAction[];
  blockedActions: CapabilityAction[];
  blockedRisks: CapabilityRisk[];
  blockedSensitivities: CapabilitySensitivity[];
  defaultDataDomain: string;
  guardrails: PermissionPackageGuardrail[];
}

export interface PermissionPackageGuardrail {
  capabilityKey: string;
  expectedDecision: PermissionPackageDecision;
  reason: string;
  reasonKey: string;
}

export interface PermissionPackageDraftInput {
  callerInstanceId: string;
  region: string;
  requestText: string;
  subjectSelector?: string;
  targetId: string;
  templateId: string;
  tenantId: string;
  workspaceId: string;
}

export interface PermissionPackageDraft {
  id: string;
  input: PermissionPackageDraftInput;
  template: PermissionPackageTemplate;
  allowedCapabilities: Capability[];
  blockedCapabilities: Capability[];
  dataScopes: DataScope[];
  readiness: PermissionPackageReadiness;
  simulationRows: PermissionPackageSimulationRow[];
}

export interface PermissionPackageReadiness {
  canApply: boolean;
  missingFields: string[];
  warnings: string[];
}

export interface PermissionPackageSimulationRow {
  id: string;
  capabilityId?: string;
  capabilityKey: string;
  expectedDecision: PermissionPackageDecision;
  reason: string;
  reasonKey?: string;
  reasonValues?: Record<string, string>;
}

export interface PermissionPackageInventory {
  capabilities: Capability[];
}

export const permissionPackageTemplates: PermissionPackageTemplate[] = [
  {
    id: "sales-readonly",
    name: "Sales read-only",
    summary: "Allow CRM reads for a scoped sales tenant while blocking exports, deletes, admin actions, and restricted data.",
    allowedActions: ["read"],
    blockedActions: ["export", "delete", "admin"],
    blockedRisks: ["high", "critical"],
    blockedSensitivities: ["restricted"],
    defaultDataDomain: "crm",
    guardrails: [
      {
        capabilityKey: "cross-region-data",
        expectedDecision: "deny",
        reason: "Data scope keeps the package inside the selected region.",
        reasonKey: "permissionSimulation.guardrailRegion",
      },
      {
        capabilityKey: "sensitive-finance-fields",
        expectedDecision: "deny",
        reason: "Sales read-only does not include finance field access.",
        reasonKey: "permissionSimulation.guardrailFinance",
      },
    ],
  },
  {
    id: "support-ticket-triage",
    name: "Support ticket triage",
    summary: "Allow ticket reads and bounded updates while blocking exports, deletes, and admin operations.",
    allowedActions: ["read", "write"],
    blockedActions: ["export", "delete", "admin"],
    blockedRisks: ["critical"],
    blockedSensitivities: ["restricted"],
    defaultDataDomain: "support",
    guardrails: [
      {
        capabilityKey: "delete-ticket",
        expectedDecision: "deny",
        reason: "Support triage does not include destructive operations.",
        reasonKey: "permissionSimulation.guardrailDeleteTicket",
      },
    ],
  },
  {
    id: "analytics-sandbox",
    name: "Analytics sandbox",
    summary: "Allow read and execute capabilities for sandbox analysis while blocking writes, exports, and production admin actions.",
    allowedActions: ["read", "execute"],
    blockedActions: ["write", "export", "delete", "admin"],
    blockedRisks: ["high", "critical"],
    blockedSensitivities: ["restricted"],
    defaultDataDomain: "analytics",
    guardrails: [
      {
        capabilityKey: "production-write",
        expectedDecision: "deny",
        reason: "Analytics sandbox cannot write production data.",
        reasonKey: "permissionSimulation.guardrailProductionWrite",
      },
    ],
  },
  {
    id: "audit-readonly",
    name: "Audit read-only",
    summary: "Allow low-risk reads for audit review while blocking mutations, exports, and restricted data.",
    allowedActions: ["read"],
    blockedActions: ["write", "export", "delete", "admin"],
    blockedRisks: ["high", "critical"],
    blockedSensitivities: ["restricted"],
    defaultDataDomain: "audit",
    guardrails: [
      {
        capabilityKey: "audit-export",
        expectedDecision: "deny",
        reason: "Audit read-only requires a separate export approval.",
        reasonKey: "permissionSimulation.guardrailAuditExport",
      },
    ],
  },
];

export function createPermissionPackageDraft(
  input: PermissionPackageDraftInput,
  inventory: PermissionPackageInventory,
): PermissionPackageDraft {
  const template = permissionPackageTemplates.find((item) => item.id === input.templateId) ?? permissionPackageTemplates[0];
  const targetCapabilities = inventory.capabilities.filter((capability) => capability.targetId === input.targetId);
  const blockedCapabilities = targetCapabilities.filter((capability) => isBlockedByTemplate(capability, template));
  const allowedCapabilities = targetCapabilities.filter(
    (capability) => template.allowedActions.includes(capability.action) && !isBlockedByTemplate(capability, template),
  );
  const dataScopes = buildDraftDataScopes(input, template, allowedCapabilities);
  const readiness = buildReadiness(input, allowedCapabilities);
  return {
    allowedCapabilities,
    blockedCapabilities,
    dataScopes,
    id: draftId(input),
    input,
    readiness,
    simulationRows: buildSimulationRows(allowedCapabilities, blockedCapabilities, template),
    template,
  };
}

function isBlockedByTemplate(capability: Capability, template: PermissionPackageTemplate): boolean {
  return (
    template.blockedActions.includes(capability.action) ||
    template.blockedRisks.includes(capability.riskLevel) ||
    template.blockedSensitivities.includes(capability.sensitivity)
  );
}

function buildDraftDataScopes(
  input: PermissionPackageDraftInput,
  template: PermissionPackageTemplate,
  allowedCapabilities: Capability[],
): DataScope[] {
  const dataDomain =
    allowedCapabilities.flatMap((capability) => capability.dataDomains ?? [])[0] ||
    allowedCapabilities.flatMap((capability) => capability.dataScopes ?? []).find((scope) => scope.dataDomain)?.dataDomain ||
    template.defaultDataDomain;
  return [
    {
      dataDomain,
      region: input.region.trim() || undefined,
      tenantFilter: input.tenantId.trim() ? `tenant_id = '${input.tenantId.trim()}'` : undefined,
    },
  ];
}

function buildReadiness(
  input: PermissionPackageDraftInput,
  allowedCapabilities: Capability[],
): PermissionPackageReadiness {
  const missingFields = [
    ["tenantId", input.tenantId],
    ["workspaceId", input.workspaceId],
    ["callerInstanceId", input.callerInstanceId],
    ["targetId", input.targetId],
  ]
    .filter(([, value]) => !value.trim())
    .map(([field]) => field);
  const warnings = allowedCapabilities.length === 0 ? ["No matching allowed capabilities for the selected target."] : [];
  return {
    canApply: missingFields.length === 0 && warnings.length === 0,
    missingFields,
    warnings,
  };
}

function buildSimulationRows(
  allowedCapabilities: Capability[],
  blockedCapabilities: Capability[],
  template: PermissionPackageTemplate,
): PermissionPackageSimulationRow[] {
  return [
    ...allowedCapabilities.map((capability) => ({
      capabilityId: capability.id,
      capabilityKey: capability.key,
      expectedDecision: "allow" as const,
      id: `allow:${capability.id}`,
      reason: `${template.name} allows ${capability.action} capability ${capability.key}.`,
      reasonKey: "permissionSimulation.allowCapability",
      reasonValues: {
        action: capability.action,
        capability: capability.key,
        packageId: template.id,
      },
    })),
    ...blockedCapabilities.map((capability) => ({
      capabilityId: capability.id,
      capabilityKey: capability.key,
      expectedDecision: "deny" as const,
      id: `deny:${capability.id}`,
      reason: `${template.name} blocks ${capability.action} capability ${capability.key}.`,
      reasonKey: "permissionSimulation.blockCapability",
      reasonValues: {
        action: capability.action,
        capability: capability.key,
        packageId: template.id,
      },
    })),
    ...template.guardrails.map((guardrail) => ({
      ...guardrail,
      id: `guardrail:${guardrail.capabilityKey}`,
    })),
  ];
}

function draftId(input: PermissionPackageDraftInput): string {
  return [
    "pkgdraft",
    input.templateId,
    input.tenantId,
    input.workspaceId,
    input.callerInstanceId,
    input.targetId,
  ]
    .map((part) => part.trim().replace(/[^a-zA-Z0-9]+/g, "-").replace(/^-+|-+$/g, "").toLowerCase())
    .filter(Boolean)
    .join("-");
}
