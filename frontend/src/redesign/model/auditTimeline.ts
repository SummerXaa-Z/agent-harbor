import type { AuditEvent, TraceEvent } from "../../types.ts";

export type AuditResultKind = "success" | "denied" | "failed";
export type AuditRowKind = "audit" | "trace";

export interface AuditTimelineRow {
  // Raw event for the detail modal (structured JSON).
  event: AuditEvent | TraceEvent;
  id: string;
  kind: AuditRowKind;
  // Audit action (`agent.created`) or `call` for traces; used as a filter key.
  operation: string;
  // Human label is resolved in the view through i18n (auditAction.* keys).
  result: AuditResultKind;
  // Resource identifier: audit resourceId, trace capabilityId.
  resource: string;
  // Audit resourceType (`agent`, `capability`) or `trace`.
  resourceType: string;
  subject: string;
  // Audit summary / trace decision reason, raw server text.
  summary: string | null;
  time: string;
}

export interface AuditTimelineFilter {
  result: AuditResultKind | "all";
  type: string | "all";
}

// Traces: allowed without an upstream error succeeded; denied is a policy
// decision, not a failure; an allowed call whose upstream errored failed.
export function traceResultKind(trace: TraceEvent): AuditResultKind {
  if (trace.upstreamError?.trim()) return "failed";
  return trace.decision === "allowed" ? "success" : "denied";
}

// Newest first. Both endpoints return ascending windows (P0), so the merge
// re-sorts instead of assuming an order.
export function auditTimelineRows(
  audits: readonly AuditEvent[],
  traces: readonly TraceEvent[],
): AuditTimelineRow[] {
  const rows: AuditTimelineRow[] = [
    ...audits.map((event): AuditTimelineRow => ({
      event,
      id: event.id,
      kind: "audit",
      operation: event.action,
      result: "success",
      resource: event.resourceId,
      resourceType: event.resourceType,
      subject: event.actor,
      summary: event.summary ?? null,
      time: event.createdAt,
    })),
    ...traces.map((event): AuditTimelineRow => ({
      event,
      id: event.id,
      kind: "trace",
      operation: "call",
      result: traceResultKind(event),
      resource: event.capabilityId ?? event.targetAgentId,
      resourceType: "trace",
      subject: event.subjectId ?? event.callerAgentId ?? "",
      summary: event.reason ?? null,
      time: event.createdAt,
    })),
  ];
  return rows.sort((a, b) => Date.parse(b.time) - Date.parse(a.time));
}

export function filterAuditTimeline(
  rows: readonly AuditTimelineRow[],
  filter: AuditTimelineFilter,
): AuditTimelineRow[] {
  return rows.filter(
    (row) =>
      (filter.result === "all" || row.result === filter.result)
      && (filter.type === "all" || row.resourceType === filter.type),
  );
}

// Distinct audit resource types in the data, in first-seen order; the filter
// select offers exactly what the window contains, never invented options.
export function auditResourceTypes(rows: readonly AuditTimelineRow[]): string[] {
  const types: string[] = [];
  for (const row of rows) {
    if (row.kind === "audit" && row.resourceType && !types.includes(row.resourceType)) types.push(row.resourceType);
  }
  return types;
}

export function todayAuditCount(rows: readonly AuditTimelineRow[], now: Date = new Date()): number {
  const start = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  return rows.filter((row) => {
    const time = Date.parse(row.time);
    return Number.isFinite(time) && time >= start;
  }).length;
}

export function auditRangeSince(range: "today" | "week" | "month", now: Date = new Date()): string {
  const start = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  if (range === "today") return start.toISOString();
  start.setDate(start.getDate() - (range === "week" ? 6 : 29));
  return start.toISOString();
}

// Resource-type label keys reuse the legacy auditResource.* mapping so both
// consoles name resources identically; only values the legacy console never
// had get their own rd.audit.type.* key.
export function auditResourceTypeLabelKey(resourceType: string): string | null {
  const known: Record<string, string> = {
    access_grant: "auditResource.access_grant",
    access_handoff_token: "rd.audit.type.access_handoff_token",
    admin_identity: "auditResource.admin_identity",
    agent: "auditResource.agent",
    agent_key: "auditResource.agent_key",
    capability: "auditResource.capability",
    instance_assignment: "auditResource.instance_assignment",
    permission_package: "auditResource.permission_package",
    permission_package_approval_request: "auditResource.permission_package_approval_request",
    route_policy: "auditResource.route_policy",
    tenant: "auditResource.tenant",
    tenant_entitlement: "auditResource.tenant_entitlement",
    workspace_assignment: "auditResource.workspace_assignment",
  };
  return known[resourceType] ?? null;
}
