import { subjectIdExampleFromSelector } from "../../permissionPackages.ts";
import type { Capability } from "../../types.ts";
import type { AccessContext } from "./accessContext.ts";

// The go-live readiness check requires one allowed and one denied gateway run
// record before a short-lived token can be issued. This journey produces that
// evidence with a temporary agent key that is revoked as soon as the calls
// finish, so the user never needs a pre-existing token to bootstrap (eval
// round 4, finding #25). Ported from the legacy console's runtime validation
// with the denied call made optional: templates that block nothing skip the
// denied evidence, matching the backend's "not applicable" semantics.

export type RuntimeValidationBlocker =
  | "requiresLiveApi"
  | "requiresApplication"
  | "requiresAllowedCapability"
  | "requiresSubject";

export interface RuntimeValidationPlan {
  allowedCapabilityKey: string;
  // Null when the draft blocks no capability; the denied call is skipped.
  blockedCapabilityKey: string | null;
  callerInstanceId: string;
  runId: string;
  subjectId: string;
  targetId: string;
}

export interface RuntimeValidationReadiness {
  blockers: RuntimeValidationBlocker[];
  plan: RuntimeValidationPlan | null;
}

export function buildRuntimeValidationReadiness({
  allowedCapabilities,
  blockedCapabilities,
  context,
  hasApplication,
  liveDataAvailable,
  runId,
}: {
  allowedCapabilities: readonly Capability[];
  blockedCapabilities: readonly Capability[];
  context: Pick<AccessContext, "callerInstanceId" | "subjectSelector" | "targetId">;
  hasApplication: boolean;
  liveDataAvailable: boolean;
  runId: string;
}): RuntimeValidationReadiness {
  const blockers: RuntimeValidationBlocker[] = [];
  if (!liveDataAvailable) blockers.push("requiresLiveApi");
  if (!hasApplication) blockers.push("requiresApplication");
  const targetAllowed = allowedCapabilities.filter((capability) => capability.targetId === context.targetId);
  const targetBlocked = blockedCapabilities.filter((capability) => capability.targetId === context.targetId);
  // Read actions make the least-noise allowed probe; exports make the most
  // representative denied one (the same preference the legacy journey used).
  const allowed = targetAllowed.find((capability) => capability.action === "read") ?? targetAllowed[0];
  const blocked = targetBlocked.find((capability) => capability.action === "export") ?? targetBlocked[0];
  if (!allowed) blockers.push("requiresAllowedCapability");
  const subjectId = subjectIdExampleFromSelector(context.subjectSelector) ?? "";
  if (!subjectId) blockers.push("requiresSubject");
  const plan = blockers.length === 0 && allowed
    ? {
        allowedCapabilityKey: allowed.key,
        blockedCapabilityKey: blocked ? blocked.key : null,
        callerInstanceId: context.callerInstanceId,
        runId,
        subjectId,
        targetId: context.targetId,
      }
    : null;
  return { blockers, plan };
}

export function mcpToolsListPayload() {
  return {
    id: "tools-list",
    jsonrpc: "2.0" as const,
    method: "tools/list",
  };
}

export function mcpToolCallPayload(toolName: string) {
  return {
    id: `call-${toolName}`,
    jsonrpc: "2.0" as const,
    method: "tools/call",
    params: {
      arguments: {
        query: "acme",
        status: "triaged",
        ticketId: "T-1000",
      },
      name: toolName,
    },
  };
}

// How a finished validation sequence landed. A denied probe that is not
// rejected by the gateway (round 6, finding #41 — the caller holds a grant
// wider than the package) is a diagnosed condition, not an abort: the
// allowed probe still runs so its evidence lands, and the outcome carries
// the observed status so the UI can name the capability and the fix.
export type RuntimeValidationRunOutcome =
  | { kind: "toolListFailed"; status: number }
  | { allowedStatus: number; blockedCapabilityKey: string; deniedStatus: number; kind: "denyUnexpected" }
  | { allowedStatus: number; kind: "allowedFailed"; status: number }
  | { allowedStatus: number; deniedStatus: number | null; kind: "completed" };

export function classifyRuntimeValidationRun(input: {
  allowedOk: boolean;
  allowedStatus: number;
  blockedCapabilityKey: string | null;
  deniedStatus: number | null;
  toolListOk: boolean;
  toolListStatus: number;
}): RuntimeValidationRunOutcome {
  if (!input.toolListOk) return { kind: "toolListFailed", status: input.toolListStatus };
  // deniedStatus !== 403 with a blocked capability planned means the gateway
  // let the blocked call through; null means no denied probe was planned.
  const denyUnexpected = input.blockedCapabilityKey !== null && input.deniedStatus !== 403;
  if (!input.allowedOk) return { allowedStatus: input.allowedStatus, kind: "allowedFailed", status: input.allowedStatus };
  if (denyUnexpected && input.blockedCapabilityKey !== null) {
    return {
      allowedStatus: input.allowedStatus,
      blockedCapabilityKey: input.blockedCapabilityKey,
      deniedStatus: input.deniedStatus ?? 0,
      kind: "denyUnexpected",
    };
  }
  return { allowedStatus: input.allowedStatus, deniedStatus: input.deniedStatus === 403 ? 403 : null, kind: "completed" };
}
