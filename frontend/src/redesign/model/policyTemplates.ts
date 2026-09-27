import type { PermissionPackageTemplate } from "../../permissionPackages";

// Access-policy page view-model. Templates are a static read-only catalog
// (plan difference #7): no edit, no clone — the only action is starting an
// application derived from one.

export interface TemplateRuleRow {
  capabilityKey: string;
  decision: "allow" | "deny";
  reason: string;
  reasonKey: string;
}

export function templateRuleRows(template: PermissionPackageTemplate): TemplateRuleRow[] {
  return template.guardrails.map((guardrail) => ({
    capabilityKey: guardrail.capabilityKey,
    decision: guardrail.expectedDecision,
    reason: guardrail.reason,
    reasonKey: guardrail.reasonKey
  }));
}

// Resolves the ?template= deep link; unknown ids fall back to the first
// template so the detail card never points at nothing.
export function normalizeTemplateId(
  value: string | undefined,
  templates: readonly PermissionPackageTemplate[],
): string | null {
  const fallback = templates.length > 0 ? templates[0].id : null;
  if (!value) return fallback;
  return templates.some((template) => template.id === value) ? value : fallback;
}
