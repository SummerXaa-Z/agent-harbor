import { subjectIdExampleFromSelector } from "../../permissionPackages.ts";
import type {
  PermissionPackageApplication,
  PermissionPackageApprovalRequest,
  PermissionPackageDraftInput,
  PermissionPackageProductionReadinessFilter,
} from "../../permissionPackages.ts";
import type { Agent } from "../../types.ts";
import type { RouteParams } from "../router.ts";

export const ACCESS_CONTEXT_STORAGE_KEY = "agent-harbor-access-context-v1";

// The caller → target request the user is working on. Region and request text
// are part of the approval snapshot, so they travel with the scope.
export interface AccessContext {
  callerInstanceId: string;
  region: string;
  requestedCapabilityId: string;
  requestText: string;
  subjectSelector: string;
  targetId: string;
  templateId: string;
  tenantId: string;
  workspaceId: string;
}

export type AccessContextSource = "route" | "stored" | "application" | "none";

const contextFields = [
  "callerInstanceId",
  "region",
  "requestedCapabilityId",
  "requestText",
  "subjectSelector",
  "targetId",
  "templateId",
  "tenantId",
  "workspaceId",
] as const satisfies readonly (keyof AccessContext)[];

const scopeFields = [
  "tenantId",
  "workspaceId",
  "callerInstanceId",
  "targetId",
  "templateId",
  "requestedCapabilityId",
  "subjectSelector",
] as const satisfies readonly (keyof AccessContext)[];

export const emptyAccessContext: AccessContext = {
  callerInstanceId: "",
  region: "",
  requestedCapabilityId: "",
  requestText: "",
  subjectSelector: "",
  targetId: "",
  templateId: "",
  tenantId: "",
  workspaceId: "",
};

export function normalizeAccessContext(value: Partial<AccessContext>): AccessContext {
  const next = { ...emptyAccessContext };
  for (const field of contextFields) {
    const raw = value[field];
    next[field] = typeof raw === "string" ? raw.trim() : "";
  }
  return next;
}

export function parseStoredAccessContext(raw: string | null | undefined): AccessContext | null {
  if (!raw) return null;
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return null;
    const context = normalizeAccessContext(parsed as Partial<AccessContext>);
    return context.callerInstanceId && context.targetId ? context : null;
  } catch {
    return null;
  }
}

export function serializeAccessContext(context: AccessContext): string {
  return JSON.stringify(normalizeAccessContext(context));
}

export function accessContextFromApplication(application: PermissionPackageApplication): AccessContext {
  return normalizeAccessContext({
    callerInstanceId: application.callerInstanceId,
    region: application.region,
    requestedCapabilityId: application.requestedCapabilityId,
    requestText: application.requestText,
    subjectSelector: application.subjectSelector,
    targetId: application.targetId,
    templateId: application.templateId,
    tenantId: application.tenantId,
    workspaceId: application.workspaceId,
  });
}

export function accessContextFromApprovalRequest(request: PermissionPackageApprovalRequest): AccessContext {
  return normalizeAccessContext({
    callerInstanceId: request.callerInstanceId,
    region: request.region,
    requestedCapabilityId: request.requestedCapabilityId,
    requestText: request.requestText,
    subjectSelector: request.subjectSelector,
    targetId: request.targetId,
    templateId: request.templateId,
    tenantId: request.tenantId,
    workspaceId: request.workspaceId,
  });
}

export function latestApplication(applications: readonly PermissionPackageApplication[]): PermissionPackageApplication | null {
  let latest: PermissionPackageApplication | null = null;
  for (const application of applications) {
    if (!latest || Date.parse(application.appliedAt) > Date.parse(latest.appliedAt)) latest = application;
  }
  return latest;
}

export function accessContextKey(context: AccessContext): string {
  return scopeFields.map((field) => context[field]).join("|");
}

export function sameAccessScope(a: AccessContext, b: AccessContext): boolean {
  return accessContextKey(a) === accessContextKey(b);
}

// Recent applications, newest first and one per scope, for the context switcher.
// Holder-bound sessions pass their owned agents so a stale cached row can never
// resurface an out-of-scope context; the server already narrows the list.
export function accessContextOptions(
  applications: readonly PermissionPackageApplication[],
  limit = 6,
  holderAgentIds?: readonly string[],
): { application: PermissionPackageApplication; context: AccessContext; key: string }[] {
  const seen = new Set<string>();
  const options: { application: PermissionPackageApplication; context: AccessContext; key: string }[] = [];
  const owned = holderAgentIds?.length ? new Set(holderAgentIds) : null;
  const scoped = owned ? applications.filter((application) => owned.has(application.callerInstanceId)) : applications;
  const sorted = [...scoped].sort((a, b) => Date.parse(b.appliedAt) - Date.parse(a.appliedAt));
  for (const application of sorted) {
    const context = accessContextFromApplication(application);
    const key = accessContextKey(context);
    if (seen.has(key)) continue;
    seen.add(key);
    options.push({ application, context, key });
    if (options.length >= limit) break;
  }
  return options;
}

// Deep links name agents only; tenant and workspace come from the caller.
export function accessContextFromRouteParams(params: RouteParams, agents: readonly Agent[]): Partial<AccessContext> | null {
  const callerInstanceId = params.caller?.trim() ?? "";
  const targetId = params.target?.trim() ?? "";
  if (!callerInstanceId && !targetId && !params.template) return null;
  const caller = agents.find((agent) => agent.id === callerInstanceId);
  const partial: Partial<AccessContext> = {};
  if (callerInstanceId) partial.callerInstanceId = callerInstanceId;
  if (caller) {
    partial.tenantId = caller.tenantId;
    partial.workspaceId = caller.workspaceId;
  }
  if (targetId) partial.targetId = targetId;
  if (params.template) partial.templateId = params.template.trim();
  if (params.capability) partial.requestedCapabilityId = params.capability.trim();
  if (params.subject) partial.subjectSelector = params.subject.trim();
  return partial;
}

export interface ResolvedAccessContext {
  context: AccessContext;
  source: AccessContextSource;
}

// Precedence: deep link, then the context the user last worked on, then the
// most recent application. A deep link to a different request drops the
// stored request text and region, which belong to the old snapshot.
export function resolveAccessContext({
  agents,
  applications,
  routeParams,
  stored,
}: {
  agents: readonly Agent[];
  applications: readonly PermissionPackageApplication[];
  routeParams: RouteParams;
  stored: AccessContext | null;
}): ResolvedAccessContext {
  const latest = latestApplication(applications);
  const base = validContext(stored, agents)
    ? { context: stored as AccessContext, source: "stored" as const }
    : latest
      ? { context: accessContextFromApplication(latest), source: "application" as const }
      : { context: emptyAccessContext, source: "none" as const };
  const fromRoute = accessContextFromRouteParams(routeParams, agents);
  if (!fromRoute) return base;
  const merged = normalizeAccessContext({ ...base.context, ...fromRoute });
  const sameRequest = merged.callerInstanceId === base.context.callerInstanceId
    && merged.targetId === base.context.targetId
    && merged.templateId === base.context.templateId;
  if (!sameRequest) {
    merged.region = "";
    merged.requestText = "";
    if (!fromRoute.requestedCapabilityId) merged.requestedCapabilityId = "";
    if (!fromRoute.subjectSelector) merged.subjectSelector = "";
  }
  return { context: merged, source: "route" };
}

function validContext(context: AccessContext | null, agents: readonly Agent[]): boolean {
  if (!context) return false;
  if (agents.length === 0) return true;
  return agents.some((agent) => agent.id === context.callerInstanceId)
    && agents.some((agent) => agent.id === context.targetId);
}

export function accessContextRouteParams(context: AccessContext): RouteParams {
  const params: RouteParams = {};
  if (context.templateId) params.template = context.templateId;
  if (context.callerInstanceId) params.caller = context.callerInstanceId;
  if (context.targetId) params.target = context.targetId;
  if (context.requestedCapabilityId) params.capability = context.requestedCapabilityId;
  if (context.subjectSelector) params.subject = context.subjectSelector;
  return params;
}

export function accessContextDraftInput(context: AccessContext): PermissionPackageDraftInput {
  return {
    callerInstanceId: context.callerInstanceId,
    region: context.region,
    requestText: context.requestText,
    requestedCapabilityId: context.requestedCapabilityId || undefined,
    subjectSelector: context.subjectSelector,
    targetId: context.targetId,
    templateId: context.templateId,
    tenantId: context.tenantId,
    workspaceId: context.workspaceId,
  };
}

export function accessContextFromDraftInput(input: PermissionPackageDraftInput): AccessContext {
  return normalizeAccessContext({ ...input, requestedCapabilityId: input.requestedCapabilityId ?? "" });
}

export function readinessFilterFromContext(
  context: AccessContext,
  approvalRequestId = "",
): PermissionPackageProductionReadinessFilter {
  return {
    approvalRequestId: approvalRequestId || undefined,
    callerInstanceId: context.callerInstanceId,
    region: context.region || undefined,
    requestText: context.requestText || undefined,
    requestedCapabilityId: context.requestedCapabilityId || undefined,
    subjectId: subjectIdExampleFromSelector(context.subjectSelector),
    subjectSelector: context.subjectSelector || undefined,
    targetId: context.targetId,
    templateId: context.templateId,
    tenantId: context.tenantId,
    workspaceId: context.workspaceId,
  };
}

export function accessContextComplete(context: AccessContext): boolean {
  return Boolean(context.tenantId && context.workspaceId && context.callerInstanceId && context.targetId && context.templateId);
}
