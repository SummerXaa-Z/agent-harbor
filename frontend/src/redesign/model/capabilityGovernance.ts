import type { Capability } from "../../types.ts";

// Domain extraction mirrors templateAllowsCapability (askJourney.ts): a
// capability only reaches template matching through dataDomains or the
// domains carried on dataScopes. Empty means "unclassified" — the capability
// then fails every template match and permission previews treat it as
// blocked, which is easy to mistake for a template decision.
export function capabilityDataDomains(capability: Capability): string[] {
  const domains = new Set<string>();
  for (const domain of capability.dataDomains ?? []) {
    const trimmed = domain.trim();
    if (trimmed) domains.add(trimmed);
  }
  for (const scope of capability.dataScopes ?? []) {
    const trimmed = (scope.dataDomain ?? "").trim();
    if (trimmed) domains.add(trimmed);
  }
  return [...domains];
}

export function capabilityNeedsDataDomain(capability: Capability): boolean {
  return capabilityDataDomains(capability).length === 0;
}

// Splits a preview's blocked capabilities into the two very different
// reasons: missing governance metadata (fixable by classifying the domain)
// versus blocked by the template's design (least privilege).
export function splitBlockedCapabilities(
  blocked: readonly Capability[],
): { blockedByDesign: Capability[]; missingDomain: Capability[] } {
  const blockedByDesign: Capability[] = [];
  const missingDomain: Capability[] = [];
  for (const capability of blocked) {
    if (capabilityNeedsDataDomain(capability)) missingDomain.push(capability);
    else blockedByDesign.push(capability);
  }
  return { blockedByDesign, missingDomain };
}

export function unclassifiedCapabilities(capabilities: readonly Capability[]): Capability[] {
  return capabilities.filter((capability) => capabilityNeedsDataDomain(capability));
}
