import type { PermissionPackageApprovalRequest, PermissionPackageTemplate } from "../../permissionPackages";
import type { Agent, AgentKey, Capability } from "../../types";
import type { EnvCheckRow } from "./envChecks";
import { maskSecret } from "./secretMask.ts";
import { keyStatus } from "./userWorkbench.ts";
import { approvalCapabilityRows, approvalRiskSummary } from "./approvalReview.ts";
import type { Surface } from "../router.ts";

// Notification derivation (plan §8; phase 2 adds handoff-token lifecycle and
// live structural environment rows). Dedupe id is `${kind}:${resourceId}:${status}`;
// read state lives in localStorage and syncs across tabs via the storage event.

export type NotificationKind = "approval" | "env" | "token";

export interface NotificationItem {
  createdAt: string;
  dedupeId: string;
  hash: string;
  kind: NotificationKind;
  params: Record<string, string | number>;
  status: string;
  subKey: string;
  surface: Surface;
  titleKey: string;
}

export interface NotificationDeriveInput {
  agents: readonly Agent[];
  approvals: readonly PermissionPackageApprovalRequest[];
  capabilities: readonly Capability[];
  envRows: readonly EnvCheckRow[];
  keys?: readonly AgentKey[];
  now?: number;
  sessionActor: string | null;
  // Live counts for the two structural empty-state environment rows; when
  // provided they replace the snapshot-derived noTarget/noApp rows so the
  // notifications resolve the moment the condition clears (round 5, #40).
  structuralEnv?: { applicationCount: number; registeredTargetCount: number } | null;
  templates: readonly PermissionPackageTemplate[];
}

// User-side notifications only ever concern the session actor's own requests
// (requestedBy filter, plan §9); the admin side sees the pending queue and
// environment failures.
export function deriveNotifications(input: NotificationDeriveInput): NotificationItem[] {
  const agentName = (id: string) => input.agents.find((agent) => agent.id === id)?.name ?? id;
  const templateById = new Map(input.templates.map((template) => [template.id, template]));
  const items: NotificationItem[] = [];

  for (const request of input.approvals) {
    const mine = input.sessionActor !== null && request.requestedBy === input.sessionActor;
    if (request.status === "pending") {
      const rows = approvalCapabilityRows(
        request, templateById.get(request.templateId) ?? null, input.capabilities,
      );
      const summary = approvalRiskSummary(rows);
      items.push({
        createdAt: request.createdAt,
        dedupeId: `approval:${request.id}:pending`,
        hash: `#admin/approvals?id=${request.id}`,
        kind: "approval",
        params: {
          allow: rows.length - summary.deniedCount,
          caller: agentName(request.callerInstanceId),
          deny: summary.deniedCount,
          target: agentName(request.targetId),
        },
        status: "pending",
        subKey: "rd.nt.pending.sub",
        surface: "admin",
        titleKey: "rd.nt.pending.title",
      });
    } else if (mine && (request.status === "approved" || request.status === "rejected")) {
      const reviewer = request.reviewedBy ?? "";
      const comment = request.reviewComment ?? "";
      const at = request.resolvedAt ?? request.updatedAt;
      items.push({
        createdAt: at,
        dedupeId: `approval:${request.id}:${request.status}`,
        hash: `#user/apply?approval=${request.id}`,
        kind: "approval",
        params: request.status === "rejected" ? { comment, reviewer } : { reviewer },
        status: request.status,
        subKey: request.status === "rejected"
          ? (comment ? "rd.nt.rejected.subReason" : "rd.nt.rejected.sub")
          : "rd.nt.approved.sub",
        surface: "user",
        titleKey: request.status === "rejected" ? "rd.nt.rejected.title" : "rd.nt.approved.title",
      });
    }
  }

  for (const row of input.envRows) {
    if (row.status !== "error" && row.status !== "warning") continue;
    if (input.structuralEnv && structuralEnvSubKeys.has(row.subKey)) continue;
    items.push({
      createdAt: "",
      dedupeId: `env:${row.key}:${row.status}`,
      hash: "#admin/cockpit",
      kind: "env",
      // The check row's own localized line doubles as the notification body.
      params: row.subParams ?? {},
      status: row.status,
      subKey: row.subKey,
      surface: "admin",
      titleKey: "rd.nt.env.title",
    });
  }

  if (input.structuralEnv) {
    if (input.structuralEnv.registeredTargetCount === 0) {
      items.push(structuralEnvItem("mcp", "rd.envcheck.mcp.noTarget"));
    }
    if (input.structuralEnv.applicationCount === 0) {
      items.push(structuralEnvItem("corePath", "rd.envcheck.corePath.noApp"));
    }
  }

  for (const key of input.keys ?? []) {
    if (!key.createdForHandoffId) continue;
    const status = keyStatus(key, input.now ?? Date.now());
    if (status === "active") continue;
    items.push({
      createdAt: status === "revoked" ? (key.revokedAt ?? key.expiresAt) : key.expiresAt,
      dedupeId: `token:${key.id}:${status}`,
      hash: "#user/golive",
      kind: "token",
      params: { prefix: maskSecret(key.prefix) },
      status,
      subKey: "rd.nt.token.sub",
      surface: "user",
      titleKey: `rd.nt.token.${status}.title`,
    });
  }

  return items.sort((a, b) => b.createdAt.localeCompare(a.createdAt));
}

// The two snapshot rows structuralEnv replaces, keyed by their sub copy.
const structuralEnvSubKeys = new Set(["rd.envcheck.mcp.noTarget", "rd.envcheck.corePath.noApp"]);

// Same dedupe ids and copy as the snapshot-derived rows, so read state and
// wording carry over unchanged while the condition becomes live.
function structuralEnvItem(rowKey: "mcp" | "corePath", subKey: string): NotificationItem {
  return {
    createdAt: "",
    dedupeId: `env:${rowKey}:warning`,
    hash: "#admin/cockpit",
    kind: "env",
    params: {},
    status: "warning",
    subKey,
    surface: "admin",
    titleKey: "rd.nt.env.title",
  };
}

export const NOTIFICATION_STORAGE_KEY = "agent-harbor-notifications-v1";

export interface NotificationReadState {
  // Items resolved before the first run count as read on first sight.
  baselineAt: string;
  readIds: string[];
}

export function emptyReadState(now: string): NotificationReadState {
  return { baselineAt: now, readIds: [] };
}

export function parseReadState(raw: string | null, now: string): NotificationReadState {
  if (!raw) return emptyReadState(now);
  try {
    const parsed = JSON.parse(raw) as Partial<NotificationReadState>;
    if (typeof parsed.baselineAt !== "string" || !Array.isArray(parsed.readIds)) {
      return emptyReadState(now);
    }
    return { baselineAt: parsed.baselineAt, readIds: parsed.readIds.filter((id): id is string => typeof id === "string") };
  } catch {
    return emptyReadState(now);
  }
}

export function serializeReadState(state: NotificationReadState): string {
  return JSON.stringify(state);
}

// The effective read set: explicit reads plus the baseline rule — anything
// already resolved (not pending, not an env failure) that predates the first
// run never shows up as unread history.
export function effectiveReadIds(
  items: readonly NotificationItem[],
  state: NotificationReadState,
): Set<string> {
  const effective = new Set(state.readIds);
  for (const item of items) {
    if (item.status === "pending" || item.kind === "env") continue;
    if (item.createdAt < state.baselineAt) effective.add(item.dedupeId);
  }
  return effective;
}

export interface NotificationCounts {
  admin: number;
  user: number;
}

export function unreadCounts(items: readonly NotificationItem[], read: ReadonlySet<string>): NotificationCounts {
  const counts: NotificationCounts = { admin: 0, user: 0 };
  for (const item of items) {
    if (read.has(item.dedupeId)) continue;
    counts[item.surface] += 1;
  }
  return counts;
}
