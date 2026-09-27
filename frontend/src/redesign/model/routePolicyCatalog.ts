import type { RoutePolicy } from "../../types";

// Route-policy view-model. The match column shows the real routing fields
// (caller, target, routeType, routeKey) — routes never match on subjects.

export interface RoutePolicyRow {
  callerName: string;
  policy: RoutePolicy;
  routeKey: string;
  routeType: string;
  targetName: string;
}

export function routePolicyRows(
  policies: readonly RoutePolicy[],
  agentName: (agentId: string) => string,
): RoutePolicyRow[] {
  return policies
    .map((policy): RoutePolicyRow => ({
      callerName: policy.callerAgentId ? agentName(policy.callerAgentId) : "",
      policy,
      routeKey: policy.routeKey ?? "",
      routeType: policy.routeType,
      targetName: policy.targetAgentId ? agentName(policy.targetAgentId) : policy.targetAgentId
    }))
    .sort((a, b) => b.policy.priority - a.policy.priority || a.policy.id.localeCompare(b.policy.id));
}

export interface PriorityParse {
  ok: boolean;
  value: number;
}

export function parsePriorityInput(value: string): PriorityParse {
  const trimmed = value.trim();
  if (!/^\d+$/.test(trimmed)) return { ok: false, value: 0 };
  return { ok: true, value: Number.parseInt(trimmed, 10) };
}

export const routeKeyPresets = ["initialize", "tools/list", "tools/call"] as const;

export function normalizeRouteStatus(value: string): RoutePolicy["status"] {
  return value === "disabled" ? "disabled" : "enabled";
}

// Route types observed in the data, with the two channel kinds as the
// baseline so an empty deployment still offers valid options.
export function routeTypeOptions(policies: readonly RoutePolicy[]): string[] {
  return [...new Set(["mcp", "openapi", ...policies.map((policy) => policy.routeType).filter(Boolean)])];
}
