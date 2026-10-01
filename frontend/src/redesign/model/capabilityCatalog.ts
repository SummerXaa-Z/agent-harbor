import type {
  Capability,
  InstanceAssignment,
  TenantEntitlement,
  WorkspaceAssignment
} from "../../types";

// Capability governance view-model: per-target capability catalog, data-domain
// distribution for the donut, and the existing grant chains (entitlement →
// workspace assignment → instance assignment).

function capabilityDomain(capability: Capability): string {
  return capability.dataDomains?.[0] ?? capability.dataScopes?.find((scope) => scope.dataDomain)?.dataDomain ?? "";
}

export interface CapabilityDomainSegment {
  count: number;
  domain: string;
  fraction: number;
}

// Data-domain distribution over the given capabilities, largest first.
// Capabilities without a domain land in the unclassified bucket.
export function capabilityDomainSegments(capabilities: readonly Capability[]): CapabilityDomainSegment[] {
  const counts = new Map<string, number>();
  for (const capability of capabilities) {
    const domain = capabilityDomain(capability);
    counts.set(domain, (counts.get(domain) ?? 0) + 1);
  }
  const total = capabilities.length;
  return [...counts.entries()]
    .sort((a, b) =>
      b[1] - a[1]
      || (a[0] === "" ? 1 : b[0] === "" ? -1 : a[0].localeCompare(b[0]))
    )
    .map(([domain, count]) => ({ count, domain, fraction: total > 0 ? count / total : 0 }));
}

export interface CapabilityGrantRow {
  callerCount: number;
  capabilityId: string;
  effect: TenantEntitlement["effect"];
  entitlementId: string;
  status: TenantEntitlement["status"];
  subjectSelectors: string[];
  tenantId: string;
  workspaceCount: number;
}

export interface CapabilityGrantChainInput {
  entitlements: readonly TenantEntitlement[];
  instanceAssignments: readonly InstanceAssignment[];
  workspaceAssignments: readonly WorkspaceAssignment[];
}

// Collapses the three grant-chain layers into one row per entitlement, with
// workspace and caller counts plus the distinct subject selectors below.
export function capabilityGrantRows({
  entitlements,
  instanceAssignments,
  workspaceAssignments
}: CapabilityGrantChainInput): CapabilityGrantRow[] {
  return entitlements
    .map((entitlement) => {
      const workspaces = workspaceAssignments.filter((item) => item.tenantEntitlementId === entitlement.id);
      const instances = instanceAssignments.filter((item) =>
        workspaces.some((workspace) => workspace.id === item.workspaceAssignmentId)
      );
      return {
        callerCount: new Set(instances.map((item) => item.callerInstanceId)).size,
        capabilityId: entitlement.capabilityId,
        effect: entitlement.effect,
        entitlementId: entitlement.id,
        status: entitlement.status,
        subjectSelectors: [
          ...new Set(
            instances
              .map((item) => item.subjectSelector)
              .filter((selector): selector is string => Boolean(selector))
          )
        ],
        tenantId: entitlement.tenantId,
        workspaceCount: workspaces.length
      };
    })
    .sort((a, b) => a.capabilityId.localeCompare(b.capabilityId));
}

export const capabilityRiskLevels = ["low", "medium", "high", "critical"] as const;
export type CapabilityRiskLevel = (typeof capabilityRiskLevels)[number];

export function normalizeRiskLevel(value: string): CapabilityRiskLevel | null {
  return (capabilityRiskLevels as readonly string[]).includes(value) ? (value as CapabilityRiskLevel) : null;
}
