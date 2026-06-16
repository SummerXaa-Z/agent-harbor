import type { Translator } from "./consolePresenters"

const knownProductionReadinessActions: Record<string, string> = {
  "Apply the approved permission package before production readiness.": "productionNext.applyApproved",
  "Inspect the latest permission package application scope before go-live.": "productionNext.inspectScope",
  "Production readiness evidence is complete.": "productionNext.complete",
  "Production readiness is complete.": "productionNext.complete",
  "Resolve apply preflight blockers before claiming production readiness.": "productionNext.resolvePreflight",
  "Resolve impact review blockers before production readiness.": "productionNext.resolveImpact",
  "Review application health and drift blockers before production readiness.": "productionNext.reviewHealth",
  "Run a denied MCP call that proves blocked tools stay blocked.": "productionNext.runDenied",
  "Run an allowed MCP call with the production subject before go-live.": "productionNext.runAllowed",
  "Verify permission package applied audit evidence before production readiness.": "productionNext.verifyAudit",
  "Verify the permission package applied audit record before production readiness.": "productionNext.verifyAudit",
  "Verify tenant entitlement, workspace assignment, and caller assignment evidence.": "productionNext.verifyGrantChain",
  "Verify tenant entitlement, workspace assignment, and caller assignment records.": "productionNext.verifyGrantChain"
}

export function permissionProductionReadinessNextAction(action: string, t: Translator) {
  const key = knownProductionReadinessActions[action]
  return key ? t(key) : sanitizeProductionReadinessAction(action)
}

export function sanitizeProductionReadinessAction(action: string) {
  return action
    .replace(/\bevidence\b/gi, (match) => match[0] === match[0].toUpperCase() ? "Records" : "records")
    .replaceAll("证据", "记录")
}
