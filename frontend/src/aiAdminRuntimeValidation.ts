import type { Capability } from "./types";
import type { PermissionPackageDraftInput } from "./permissionPackages";
import { subjectIdExampleFromSelector } from "./permissionPackages.ts";

export type AiAdminRuntimeValidationBlocker =
  | "requires_live_api"
  | "requires_application"
  | "requires_allowed_capability"
  | "requires_blocked_capability"
  | "requires_subject";

export interface AiAdminRuntimeValidationPlan {
  allowedCapabilityKey: string;
  blockedCapabilityKey: string;
  callerInstanceId: string;
  runId: string;
  subjectId: string;
  targetId: string;
}

export interface AiAdminRuntimeValidationReadiness {
  blockers: AiAdminRuntimeValidationBlocker[];
  plan: AiAdminRuntimeValidationPlan | null;
}

export function buildAiAdminRuntimeValidationReadiness({
  allowedCapabilities,
  blockedCapabilities,
  form,
  hasApplication,
  liveDataAvailable,
  runId
}: {
  allowedCapabilities: Capability[];
  blockedCapabilities: Capability[];
  form: PermissionPackageDraftInput;
  hasApplication: boolean;
  liveDataAvailable: boolean;
  runId: string;
}): AiAdminRuntimeValidationReadiness {
  const blockers: AiAdminRuntimeValidationBlocker[] = [];
  if (!liveDataAvailable) blockers.push("requires_live_api");
  if (!hasApplication) blockers.push("requires_application");
  const targetAllowed = allowedCapabilities.filter((capability) => capability.targetId === form.targetId);
  const targetBlocked = blockedCapabilities.filter((capability) => capability.targetId === form.targetId);
  const allowed = targetAllowed.find((capability) => capability.action === "read") ?? targetAllowed[0];
  const blocked = targetBlocked.find((capability) => capability.action === "export") ?? targetBlocked[0];
  if (!allowed) blockers.push("requires_allowed_capability");
  if (!blocked) blockers.push("requires_blocked_capability");
  const subjectId = subjectIdExampleFromSelector(form.subjectSelector) ?? "";
  if (!subjectId) blockers.push("requires_subject");
  const plan = blockers.length === 0 && allowed && blocked
    ? {
        allowedCapabilityKey: allowed.key,
        blockedCapabilityKey: blocked.key,
        callerInstanceId: form.callerInstanceId,
        runId,
        subjectId,
        targetId: form.targetId
      }
    : null;
  return { blockers, plan };
}

export function aiAdminRuntimeValidationBlockerMessageKey(
  blocker: AiAdminRuntimeValidationBlocker
): string {
  switch (blocker) {
    case "requires_live_api":
      return "message.fallbackDataModeActionBlocked";
    case "requires_application":
      return "message.aiAdminRuntimeValidationRequiresApplication";
    case "requires_allowed_capability":
      return "message.aiAdminRuntimeValidationNoAllowedCapability";
    case "requires_blocked_capability":
      return "message.aiAdminRuntimeValidationNoBlockedCapability";
    case "requires_subject":
      return "message.validationSubjectSelectorRequired";
  }
}

export function countUnclassifiedTargetCapabilities(
  capabilities: Capability[],
  targetId: string
): number {
  if (!targetId) return 0;
  return capabilities.filter(
    (capability) => capability.targetId === targetId && (capability.dataDomains?.length ?? 0) === 0
  ).length;
}
