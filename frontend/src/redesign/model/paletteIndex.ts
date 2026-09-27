import type { PermissionPackageApprovalRequest } from "../../permissionPackages";
import type { Agent } from "../../types";
import { agentEndpoint } from "./registryCatalog.ts";
import {
  adminViews,
  userViews,
  viewLabelKey,
  type RedesignView,
  type Surface,
} from "../router.ts";

// Command-palette index (plan P5). Everything is plain data: label keys for
// pages/actions (translated at render time), free-form labels for recent
// resources (runtime data such as agent names and approval ids).

export type PaletteGroupId = "recent" | "actions" | "pages";

interface PaletteBase {
  group: PaletteGroupId;
  hash: string;
  id: string;
  surface: Surface;
}

export type PaletteItem =
  | (PaletteBase & { group: "pages"; labelKey: string; view: RedesignView })
  | (PaletteBase & { group: "actions"; labelKey: string })
  | (PaletteBase & { group: "recent"; label: string; sub: string });

interface PaletteActionSpec {
  id: string;
  labelKey: string;
  surface: Surface;
}

// High-frequency entries that navigate to the page owning the action; every
// one of these is a real link (no zombie buttons).
const actionSpecs: readonly PaletteActionSpec[] = [
  { id: "apply", labelKey: "rd.palette.action.apply", surface: "user" },
  { id: "ask", labelKey: "rd.palette.action.ask", surface: "user" },
  { id: "golive", labelKey: "rd.palette.action.golive", surface: "user" },
  { id: "new-tenant", labelKey: "rd.palette.action.newTenant", surface: "admin" },
  { id: "register", labelKey: "rd.palette.action.register", surface: "admin" },
  { id: "grant-chain", labelKey: "rd.palette.action.grantChain", surface: "admin" },
  { id: "new-route", labelKey: "rd.palette.action.newRoute", surface: "admin" },
  { id: "env-check", labelKey: "rd.palette.action.envCheck", surface: "admin" },
];

const actionHash: Record<string, string> = {
  apply: "#user/apply",
  ask: "#user/ask",
  golive: "#user/golive",
  "new-tenant": "#admin/tenants",
  register: "#admin/registry",
  "grant-chain": "#admin/capabilities",
  "new-route": "#admin/routes",
  "env-check": "#admin/cockpit",
};

// Every page of both surfaces, then the action entries. Order defines the
// palette's default listing.
export function paletteItems(): PaletteItem[] {
  const pages: PaletteItem[] = [...userViews, ...adminViews].map((view) => ({
    group: "pages",
    hash: viewHashForPalette(view),
    id: `page:${view}`,
    labelKey: viewLabelKey(view),
    surface: (userViews as readonly string[]).includes(view) ? "user" : "admin",
    view,
  }));
  const actions: PaletteItem[] = actionSpecs.map((spec) => ({
    group: "actions",
    hash: actionHash[spec.id],
    id: `action:${spec.id}`,
    labelKey: spec.labelKey,
    surface: spec.surface,
  }));
  return [...actions, ...pages];
}

function viewHashForPalette(view: RedesignView): string {
  return (userViews as readonly string[]).includes(view) ? `#user/${view}` : `#admin/${view}`;
}

export const recentResourceLimit = 5;

// Recent resources: freshest registered agents and newest approval requests,
// deep-linking to their pages (the registry and approvals views both accept
// the corresponding query parameter).
export function recentItems(input: {
  agents: readonly Agent[];
  approvals: readonly PermissionPackageApprovalRequest[];
}): PaletteItem[] {
  const agents = [...input.agents]
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
    .slice(0, recentResourceLimit)
    .map((agent) => ({
      group: "recent" as const,
      hash: `#admin/registry?agent=${agent.id}`,
      id: `recent:agent:${agent.id}`,
      label: agent.name,
      sub: agent.channelType === "local" ? agent.id : agentEndpoint(agent) || agent.id,
      surface: "admin" as const,
    }));
  const approvals = [...input.approvals]
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
    .slice(0, recentResourceLimit)
    .map((request) => ({
      group: "recent" as const,
      hash: `#admin/approvals?id=${request.id}`,
      id: `recent:approval:${request.id}`,
      label: request.templateId,
      sub: request.id,
      surface: "admin" as const,
    }));
  return interleave(agents, approvals).slice(0, recentResourceLimit);
}

// Keeps the newest of both kinds near the top instead of a full agent block.
function interleave<T>(a: readonly T[], b: readonly T[]): T[] {
  const out: T[] = [];
  const max = Math.max(a.length, b.length);
  for (let i = 0; i < max; i += 1) {
    if (i < a.length) out.push(a[i]);
    if (i < b.length) out.push(b[i]);
  }
  return out;
}

// Case-insensitive substring match over the translated label (and the sub
// line for recent items); an empty query keeps everything.
export function filterPaletteItems(
  items: readonly PaletteItem[],
  query: string,
  resolveLabel: (item: PaletteItem) => string,
): PaletteItem[] {
  const needle = query.trim().toLowerCase();
  if (!needle) return [...items];
  return items.filter((item) => {
    const haystack = `${resolveLabel(item)} ${item.group === "recent" ? item.sub : ""}`.toLowerCase();
    return haystack.includes(needle);
  });
}
