import { resolveJourneyMcpEndpoint } from "../../connectionDiagnostics.ts";
import type { PermissionPackageApplication } from "../../permissionPackages.ts";
import type { Agent, AuditEvent, Capability, TargetProbeResult, TraceEvent } from "../../types.ts";

export type EnvCheckKey = "api" | "mcp" | "corePath" | "health";
export type EnvCheckStatus = "ok" | "warning" | "error" | "unsupported";

export interface EnvCheckRow {
  // Technical cause for the fix-guidance modal (probe message, endpoint,
  // catalog issues). Empty when the row is healthy.
  detail: string;
  // i18n keys listing what is broken, one bullet each; empty when healthy.
  fixKeys: string[];
  key: EnvCheckKey;
  status: EnvCheckStatus;
  subKey: string;
  subParams?: Record<string, string | number>;
}

// Mirrors the legacy resolver: among registered MCP targets prefer active
// ones that carry approved capabilities, so one dead experimental target
// cannot flunk the check for a healthy stack.
export function preferredProbeTarget(
  agents: readonly Agent[],
  capabilities: readonly Capability[],
): { agent: Agent; endpoint: string } | null {
  const endpoint = resolveJourneyMcpEndpoint([...agents], [...capabilities], "");
  if (!endpoint) return null;
  const agent = agents.find((candidate) => agentEndpoint(candidate) === endpoint) ?? null;
  return agent ? { agent, endpoint } : null;
}

export function registeredMcpTargets(agents: readonly Agent[]): Agent[] {
  return agents.filter((agent) => agent.channelType === "mcp" && agentEndpoint(agent));
}

function agentEndpoint(agent: Agent): string {
  const endpoint = (agent.channelConfig as { endpoint?: unknown } | undefined)?.endpoint;
  return typeof endpoint === "string" ? endpoint.trim() : "";
}

// Row 1: API service. Health endpoint, system/info contract and session all
// have to hold; any failure fails the row.
export function apiServiceCheck(input: {
  apiBase: string;
  apiHealthMessage: string | null;
  contractIssues: readonly string[];
}): EnvCheckRow {
  const base = { detail: "", fixKeys: [], key: "api" as const };
  if (input.contractIssues.length > 0) {
    return {
      ...base,
      detail: input.contractIssues.join(", "),
      fixKeys: ["rd.envcheck.api.fixContract"],
      status: "error",
      subKey: "rd.envcheck.api.contract",
    };
  }
  if (input.apiHealthMessage) {
    return {
      ...base,
      detail: input.apiHealthMessage,
      fixKeys: ["rd.envcheck.api.fixReach"],
      status: "error",
      subKey: "rd.envcheck.api.unreachable",
    };
  }
  return { ...base, status: "ok", subKey: "rd.envcheck.api.ok", subParams: { apiBase: input.apiBase } };
}

export interface ProbeFixGuidance {
  causeKey: string;
  detail: string;
}

// The probe endpoint classifies failures with the same upstream codes the
// console already localizes, so the fix modal names the real cause
// (connection refused vs DNS vs TLS) instead of "failed".
export function probeFixGuidance(probe: TargetProbeResult | null): ProbeFixGuidance | null {
  if (!probe || probe.status !== "error") return null;
  const code = probe.errorCode?.trim() ?? "";
  const causeKey = code.startsWith("UPSTREAM_DNS")
    ? "error.upstream.dns"
    : code.startsWith("UPSTREAM_TIMEOUT")
      ? "error.upstream.timeout"
      : code.startsWith("UPSTREAM_TLS")
        ? "error.upstream.tls"
        : code.startsWith("UPSTREAM_CONNECT")
          ? "error.upstream.connect"
          : "error.upstream.generic";
  const parts = [code || "PROBE_ERROR", probe.message?.trim()].filter(Boolean);
  const http = probe.httpStatus > 0 ? ` HTTP ${probe.httpStatus}` : "";
  return { causeKey, detail: `${parts.join(": ")}${http}`.trim() };
}

// Row 2: MCP tool service, probed read-only at the preferred target.
export function mcpServiceCheck(input: {
  endpoint: string;
  noTarget: boolean;
  probe: TargetProbeResult | null;
  unsupported: boolean;
}): EnvCheckRow {
  const base = { detail: "", fixKeys: [], key: "mcp" as const };
  if (input.unsupported) {
    return { ...base, status: "unsupported", subKey: "rd.envcheck.mcp.unsupported" };
  }
  if (input.noTarget) {
    return { ...base, fixKeys: ["rd.envcheck.mcp.fixRegister"], status: "warning", subKey: "rd.envcheck.mcp.noTarget" };
  }
  const fix = probeFixGuidance(input.probe);
  if (fix) {
    return {
      ...base,
      detail: `${input.endpoint} — ${fix.detail}`,
      fixKeys: [fix.causeKey, "rd.envcheck.mcp.fixProbe"],
      status: "error",
      subKey: "rd.envcheck.mcp.error",
      subParams: { endpoint: input.endpoint },
    };
  }
  if (!input.probe) {
    return { ...base, status: "warning", subKey: "rd.envcheck.mcp.pending", subParams: { endpoint: input.endpoint } };
  }
  return {
    ...base,
    status: "ok",
    subKey: "rd.envcheck.mcp.ok",
    subParams: { durationMs: input.probe.durationMs, endpoint: input.endpoint, tools: input.probe.toolCount },
  };
}

// Row 3: core permission path. The latest application only counts as closed
// when the apply itself, its audit event, and an allowed runtime call exist.
export function corePathCheck(input: {
  allowedTrace: TraceEvent | null;
  application: PermissionPackageApplication | null;
  appliedAudit: AuditEvent | null;
}): EnvCheckRow {
  const base = { detail: "", key: "corePath" as const };
  if (!input.application) {
    return { ...base, fixKeys: ["rd.envcheck.corePath.fixApply"], status: "warning", subKey: "rd.envcheck.corePath.noApp" };
  }
  const fixKeys: string[] = [];
  if (!input.allowedTrace) fixKeys.push("rd.envcheck.corePath.missingTrace");
  if (!input.appliedAudit) fixKeys.push("rd.envcheck.corePath.missingAudit");
  if (fixKeys.length > 0) {
    return {
      ...base,
      detail: input.application.id,
      fixKeys,
      status: "error",
      subKey: "rd.envcheck.corePath.missing",
    };
  }
  return { ...base, fixKeys: [], status: "ok", subKey: "rd.envcheck.corePath.ok" };
}

// Row 4: environment health. Catalog contract issues fail the row; other
// registered targets that do not answer only warn (plan P3 note 4) so one
// retired experimental target cannot mark a healthy environment bad.
export function envHealthCheck(input: {
  catalogDetail: string | null;
  catalogIssues: readonly string[];
  unreachable: readonly { endpoint: string; name: string }[];
}): EnvCheckRow {
  const base = { key: "health" as const };
  if (input.catalogIssues.length > 0) {
    return {
      ...base,
      detail: input.catalogIssues.join(", "),
      fixKeys: ["rd.envcheck.health.fixCatalog"],
      status: "error",
      subKey: "rd.envcheck.health.catalog",
    };
  }
  if (input.catalogDetail) {
    return { ...base, detail: input.catalogDetail, fixKeys: [], status: "warning", subKey: "rd.envcheck.health.catalogWarn" };
  }
  if (input.unreachable.length > 0) {
    return {
      ...base,
      detail: input.unreachable.map((target) => `${target.name} (${target.endpoint})`).join("; "),
      fixKeys: ["rd.envcheck.health.fixUnreachable"],
      status: "warning",
      subKey: "rd.envcheck.health.unreachable",
      subParams: { count: input.unreachable.length },
    };
  }
  return { ...base, detail: "", fixKeys: [], status: "ok", subKey: "rd.envcheck.health.ok" };
}

export interface EnvCheckSummary {
  abnormal: number;
  error: number;
  warning: number;
}

// Notifications (plan P5) consume "the latest useEnvChecks result" instead of
// probing on their own 15s cadence. A module-level single-flight cache keeps
// that shared per tab: whichever surface ran the checks last wins.
export interface EnvCheckSnapshot {
  rows: readonly EnvCheckRow[];
  summary: EnvCheckSummary;
}

let latestSnapshot: EnvCheckSnapshot | null = null;
const listeners = new Set<() => void>();

export function rememberEnvCheckSnapshot(rows: readonly EnvCheckRow[]): EnvCheckSnapshot {
  latestSnapshot = { rows, summary: envCheckSummary(rows) };
  for (const listener of listeners) listener();
  return latestSnapshot;
}

export function latestEnvCheckSnapshot(): EnvCheckSnapshot | null {
  return latestSnapshot;
}

// useSyncExternalStore wiring so derived consumers re-run when a surface
// finishes its checks; the snapshot reference is stable between updates.
export function subscribeEnvCheckSnapshot(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function envCheckSummary(rows: readonly EnvCheckRow[]): EnvCheckSummary {
  const error = rows.filter((row) => row.status === "error").length;
  const warning = rows.filter((row) => row.status === "warning").length;
  return { abnormal: error + warning, error, warning };
}
