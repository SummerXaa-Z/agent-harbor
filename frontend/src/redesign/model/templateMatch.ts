import { templateAllowsCapability } from "../../askJourney.ts";
import type { PermissionPackageTemplate } from "../../permissionPackages.ts";
import type { Capability } from "../../types.ts";
import { capabilityNeedsDataDomain } from "./capabilityGovernance.ts";

export type TemplateMatchLevel = "recommended" | "partial" | "none";

export interface TemplateMatch {
  allowedCount: number;
  blockedCount: number;
  level: TemplateMatchLevel;
  // Capabilities the template's action/risk rules would allow but that stay
  // blocked only because they have no data domain yet (governance gap, not a
  // least-privilege decision).
  missingDomainBlockedCount: number;
  // Set only when a specific capability was requested.
  requestedCovered: boolean | null;
  template: PermissionPackageTemplate;
}

const levelRank: Record<TemplateMatchLevel, number> = { recommended: 0, partial: 1, none: 2 };

// Uses the same per-capability rule as the ask handoff's template choice, so
// the template "request access" pre-selects is always the recommended one.
// With a requested capability the draft narrows to that capability alone,
// which is why every other capability counts as blocked.
export function templateMatch(
  template: PermissionPackageTemplate,
  targetCapabilities: readonly Capability[],
  requestedCapabilityId = "",
): TemplateMatch {
  const requestedId = requestedCapabilityId.trim();
  const allowed = targetCapabilities.filter((capability) => (
    (!requestedId || capability.id === requestedId) && templateAllowsCapability(capability, template)
  ));
  const allowedCount = allowed.length;
  const blockedCount = targetCapabilities.length - allowedCount;
  const missingDomainBlockedCount = targetCapabilities.filter((capability) => (
    (!requestedId || capability.id === requestedId)
      && capabilityNeedsDataDomain(capability)
      && templateAllowsCapabilityIgnoringDomain(capability, template)
  )).length;
  if (requestedId) {
    const covered = allowedCount === 1;
    return { allowedCount, blockedCount, level: covered ? "recommended" : "none", missingDomainBlockedCount, requestedCovered: covered, template };
  }
  const level: TemplateMatchLevel = allowedCount === 0 ? "none" : blockedCount === 0 ? "recommended" : "partial";
  return { allowedCount, blockedCount, level, missingDomainBlockedCount, requestedCovered: null, template };
}

export function rankTemplates(
  templates: readonly PermissionPackageTemplate[],
  targetCapabilities: readonly Capability[],
  requestedCapabilityId = "",
): TemplateMatch[] {
  return templates
    .map((template, index) => ({ index, match: templateMatch(template, targetCapabilities, requestedCapabilityId) }))
    .sort((a, b) => (
      levelRank[a.match.level] - levelRank[b.match.level]
      || b.match.allowedCount - a.match.allowedCount
      || a.index - b.index
    ))
    .map(({ match }) => match);
}

export function recommendedTemplateId(
  templates: readonly PermissionPackageTemplate[],
  targetCapabilities: readonly Capability[],
  requestedCapabilityId = "",
): string {
  const best = rankTemplates(templates, targetCapabilities, requestedCapabilityId)[0];
  return best && best.level !== "none" ? best.template.id : "";
}

// Same match as templateAllowsCapability minus the data-domain clauses, so a
// capability whose only gap is a missing domain still counts as one the
// template would have allowed.
export function templateAllowsCapabilityIgnoringDomain(
  capability: Capability,
  template: PermissionPackageTemplate,
) {
  return (
    template.allowedActions.includes(capability.action) &&
    !template.blockedActions.includes(capability.action) &&
    !template.blockedRisks.includes(capability.riskLevel) &&
    !template.blockedSensitivities.includes(capability.sensitivity) &&
    !template.guardrails.some((guardrail) => (
      guardrail.capabilityKey === capability.key && guardrail.expectedDecision === "deny"
    ))
  );
}
