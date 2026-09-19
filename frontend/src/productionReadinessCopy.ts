import type { Translator } from "./consolePresenters";
import type { PermissionPackageProductionNextActionCode } from "./permissionPackages";

// Keyed by the backend nextActionCode contract (the readiness API returns codes
// alongside the English messages), never by matching the message text itself.
export const productionReadinessNextActionKeys: Partial<
  Record<PermissionPackageProductionNextActionCode, string>
> = {
  apply_permission_package: "productionNext.applyApproved",
  export_acceptance_report: "productionNext.complete",
  reapply_permission_package: "productionNext.reapplyPermissionPackage",
  resolve_impact_blockers: "productionNext.resolveImpact",
  resolve_preflight_blockers: "productionNext.resolvePreflight",
  review_application_health: "productionNext.reviewHealth",
  review_application_scope: "productionNext.inspectScope",
  review_subject_scope: "productionNext.reviewSubjectScope",
  run_allowed_runtime_call: "productionNext.runAllowed",
  run_denied_runtime_call: "productionNext.runDenied",
  verify_access_profile: "productionNext.verifyGrantChain",
  verify_applied_audit: "productionNext.verifyAudit"
};

export function permissionProductionReadinessNextAction(
  code: string,
  fallbackMessage: string,
  t: Translator
) {
  const key = productionReadinessNextActionKeys[code as PermissionPackageProductionNextActionCode];
  return key ? t(key) : sanitizeProductionReadinessAction(fallbackMessage);
}

export function sanitizeProductionReadinessAction(action: string) {
  return action
    .replace(/\bevidence\b/gi, (match) => match[0] === match[0].toUpperCase() ? "Records" : "records")
    .replaceAll("证据", "记录")
}
