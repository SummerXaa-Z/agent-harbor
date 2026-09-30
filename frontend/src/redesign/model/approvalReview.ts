import type {
  Capability,
  CapabilityAction,
  CapabilityRisk,
} from "../../types.ts";
import type {
  PermissionPackageApprovalEffectiveStatus,
  PermissionPackageApprovalRequest,
  PermissionPackageTemplate,
} from "../../permissionPackages.ts";
import { permissionPackageApprovalEffectiveStatus } from "../../permissionPackages.ts";

export type ApprovalTab = PermissionPackageApprovalEffectiveStatus | "all";

export interface ApprovalCapabilityRow {
  // Capability from the registry when the key matches, else null (the row
  // still renders from the template guardrail).
  capability: Capability | null;
  key: string;
  allowed: boolean;
  isRequested: boolean;
}

// Allow/deny rows for one request. Guardrails come first — the template fixes
// which capabilities the package denies by design (least privilege) — then the
// request snapshot's allowed keys fill in what the guardrails don't cover,
// because the built-in templates carry deny-only guardrails while the concrete
// allow list lives only in the snapshot.
export function approvalCapabilityRows(
  request: PermissionPackageApprovalRequest,
  template: PermissionPackageTemplate | null,
  capabilities: readonly Capability[],
): ApprovalCapabilityRow[] {
  const byKey = new Map(capabilities.map((capability) => [capability.key, capability] as const));
  const requestedKey = capabilityKeyById(capabilities, request.requestedCapabilityId);
  const row = (key: string, allowed: boolean): ApprovalCapabilityRow => ({
    allowed,
    capability: byKey.get(key) ?? null,
    isRequested: key === requestedKey,
    key,
  });
  if (template) {
    const rows = template.guardrails.map((guardrail) => row(guardrail.capabilityKey, guardrail.expectedDecision === "allow"));
    const covered = new Set(rows.map((covered) => covered.key));
    for (const key of request.allowedCapabilityKeys) {
      if (covered.has(key)) continue;
      covered.add(key);
      rows.push(row(key, true));
    }
    return rows;
  }
  // Without the template catalog (sample data or older backend) every key in
  // the request snapshot is an allowed one; nothing is invented as denied.
  return request.allowedCapabilityKeys.map((key) => row(key, true));
}

function capabilityKeyById(capabilities: readonly Capability[], id?: string): string | null {
  if (!id) return null;
  return capabilities.find((capability) => capability.id === id)?.key ?? null;
}

export interface ApprovalListEntry {
  request: PermissionPackageApprovalRequest;
  status: PermissionPackageApprovalEffectiveStatus;
}

// Newest first, with the effective status resolved once for filtering, chips
// and sorting. API-created requests arrive through the same endpoint.
export function approvalList(requests: readonly PermissionPackageApprovalRequest[]): ApprovalListEntry[] {
  return requests
    .map((request) => ({ request, status: permissionPackageApprovalEffectiveStatus(request) }))
    .sort((a, b) => Date.parse(b.request.createdAt) - Date.parse(a.request.createdAt));
}

export function approvalTabCounts(entries: readonly ApprovalListEntry[]): Record<ApprovalTab, number> {
  const counts: Record<ApprovalTab, number> = { all: entries.length, approved: 0, expired: 0, pending: 0, rejected: 0, withdrawn: 0 };
  for (const entry of entries) counts[entry.status] += 1;
  return counts;
}

export function filterApprovalList(
  entries: readonly ApprovalListEntry[],
  tab: ApprovalTab,
): ApprovalListEntry[] {
  return tab === "all" ? [...entries] : entries.filter((entry) => entry.status === tab);
}

export interface ApprovalRiskSummary {
  deniedCount: number;
  highRiskCount: number;
}

export function approvalRiskSummary(rows: readonly ApprovalCapabilityRow[]): ApprovalRiskSummary {
  return {
    deniedCount: rows.filter((row) => !row.allowed).length,
    highRiskCount: rows.filter((row) => row.capability && isHighRisk(row.capability.riskLevel)).length,
  };
}

function isHighRisk(risk: CapabilityRisk | undefined): boolean {
  return risk === "high" || risk === "critical";
}

// Least-privilege notice text key: denied guardrails are by design, and high
// risk capabilities are why the request needs a reviewer at all.
export function approvalNoticeKey(summary: ApprovalRiskSummary): "leastPrivilege" | "highRisk" | "matchesTemplate" {
  if (summary.deniedCount > 0) return "leastPrivilege";
  if (summary.highRiskCount > 0) return "highRisk";
  return "matchesTemplate";
}

export function capabilityRiskLevel(capability: Capability | null): CapabilityRisk | null {
  return capability?.riskLevel ?? null;
}

export function capabilityAction(capability: Capability | null): CapabilityAction | null {
  return capability?.action ?? null;
}
