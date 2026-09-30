import { buildAiAdminProductionConsoleSummary } from "../../aiAdminProductionConsole.ts";
import { buildProductionAcceptanceCenter, type ProductionAcceptanceCenter } from "../../productionAcceptance.ts";
import type { ConnectionDiagnosticStatus } from "../../connectionDiagnostics";
import type {
  AccessHandoff,
  PermissionPackageApprovalRequest,
  PermissionPackageProductionReadiness,
  PermissionPackageWorkbenchPreview,
} from "../../permissionPackages.ts";
import { bearerPlaceholderHeader } from "./secretMask.ts";

export interface ReadinessCheckCount {
  ready: number;
  total: number;
}

// Named "N/11 readiness checks" (N/10 without a subject); never hard-coded.
export function readinessCheckCount(
  readiness: PermissionPackageProductionReadiness | null | undefined,
  preview: PermissionPackageWorkbenchPreview | null | undefined,
): ReadinessCheckCount | null {
  if (readiness) return { ready: readiness.summary.readyCount, total: readiness.checks.length };
  if (preview && preview.summary.readinessTotalCount > 0) {
    return { ready: preview.summary.readinessReadyCount, total: preview.summary.readinessTotalCount };
  }
  return null;
}

// The four go-live legs (connection, permission change, runtime validation,
// handoff) come from the same builder the legacy go-live page uses.
export function goLiveLegs({
  approval,
  connectionStatus,
  liveDataAvailable,
  preview,
  readiness,
}: {
  approval: PermissionPackageApprovalRequest | null;
  connectionStatus: ConnectionDiagnosticStatus | null;
  liveDataAvailable: boolean;
  preview: PermissionPackageWorkbenchPreview | null;
  readiness: PermissionPackageProductionReadiness | null;
}): ProductionAcceptanceCenter | null {
  if (!preview) return null;
  const productionSummary = buildAiAdminProductionConsoleSummary({
    application: preview.latestApplication ?? null,
    approvalRequest: approval,
    draft: preview.draft,
    productionReadiness: readiness,
  });
  return buildProductionAcceptanceCenter({ connectionStatus, liveDataAvailable, productionReadiness: readiness, productionSummary });
}

export const handoffTokenTtlOptions = [900, 1800, 3600] as const;
export const defaultHandoffTokenTtl = 1800;

export function tokenTtlOptions(handoff: Pick<AccessHandoff, "tokenEligibility"> | null): number[] {
  const max = handoff?.tokenEligibility.maxExpiresInSeconds || 3600;
  return handoffTokenTtlOptions.filter((seconds) => seconds <= max);
}

export function defaultTokenTtl(handoff: Pick<AccessHandoff, "tokenEligibility"> | null): number {
  const options = tokenTtlOptions(handoff);
  const preferred = handoff?.tokenEligibility.defaultExpiresInSeconds || defaultHandoffTokenTtl;
  return options.includes(preferred) ? preferred : options[options.length - 1] ?? defaultHandoffTokenTtl;
}

export function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'\\''`)}'`;
}

export function runtimeRpcPath(targetId: string): string {
  return `/api/v1/mcp/agents/${encodeURIComponent(targetId)}/rpc`;
}

// The backend emits the MCP client config origin-agnostic (a relative url);
// only the console knows the API base the operator actually uses, so the copy
// button absolutizes the url the same way the shell snippet does. Anything
// that is not a JSON object with a relative url passes through untouched.
export function absolutizeHandoffConfig(config: string, apiBase: string): string {
  let parsed: unknown;
  try {
    parsed = JSON.parse(config);
  } catch {
    return config;
  }
  if (typeof parsed !== "object" || parsed === null) return config;
  const record = parsed as { url?: unknown };
  if (typeof record.url !== "string" || !record.url.startsWith("/")) return config;
  const absolute = `${apiBase.replace(/\/+$/, "")}${record.url}`;
  if (absolute === record.url) return config;
  record.url = absolute;
  return JSON.stringify(parsed, null, 2);
}

// One command per line; values are single-quoted so any selector or tool
// name stays literal, and the token is read from the prompt, never inlined.
export function handoffShellSnippet({
  apiBase,
  comments,
  handoff,
}: {
  apiBase: string;
  comments: { call: string; token: string };
  handoff: Pick<AccessHandoff, "allowedCapabilities" | "scope">;
}): string {
  const toolName = [...handoff.allowedCapabilities.map((capability) => capability.key)].sort()[0] ?? "<allowed-capability-key>";
  const subjectId = handoff.scope.subjectId?.trim() || "<subject-id-matching-selector>";
  const url = `${apiBase.replace(/\/+$/, "")}${runtimeRpcPath(handoff.scope.targetId)}`;
  const body = JSON.stringify({
    id: "access-handoff-call",
    jsonrpc: "2.0",
    method: "tools/call",
    params: { arguments: {}, name: toolName },
  });
  return [
    `export AGENT_HARBOR_URL=${shellQuote(url)}`,
    `export AGENT_HARBOR_SUBJECT_ID=${shellQuote(subjectId)}`,
    `# ${comments.token}`,
    "read -rs AGENT_HARBOR_TOKEN",
    "export AGENT_HARBOR_TOKEN",
    `# ${comments.call}`,
    `curl -sS "$AGENT_HARBOR_URL" -H "${bearerPlaceholderHeader}" -H "X-AgentHarbor-Subject-Id: $AGENT_HARBOR_SUBJECT_ID" -H 'Content-Type: application/json' -d ${shellQuote(body)}`,
  ].join("\n");
}
