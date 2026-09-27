import { useCallback, useEffect, useRef, useState } from "react";
import {
  checkApiHealth,
  fetchManagementMcpToolsCatalog,
  fetchPermissionPackageApplications,
  fetchPermissionPackageProductionReadiness,
  probeTarget,
} from "../../api";
import { subjectIdExampleFromSelector } from "../../permissionPackages";
import { managementMcpCatalogDiagnosticFromResult, type ManagementMcpCatalogDiagnostic } from "../../connectionDiagnostics";
import type { PermissionPackageApplication } from "../../permissionPackages";
import type { Agent, TargetProbeResult } from "../../types";
import type { RedesignData } from "./useRedesignData";
import { apiServiceCheck, corePathCheck, envCheckSummary, envHealthCheck, mcpServiceCheck, preferredProbeTarget, rememberEnvCheckSnapshot, registeredMcpTargets, type EnvCheckRow, type EnvCheckSummary } from "../model/envChecks";
import { readinessCheckCount, type ReadinessCheckCount } from "../model/goLive";

export interface UnreachableTarget {
  endpoint: string;
  name: string;
}

export interface EnvChecksState {
  checking: boolean;
  latestApplication: PermissionPackageApplication | null;
  readinessCount: ReadinessCheckCount | null;
  recheck: () => Promise<void>;
  rows: readonly EnvCheckRow[];
  summary: EnvCheckSummary;
  unreachableTargets: readonly UnreachableTarget[];
}

interface EnvChecksResult {
  latestApplication: PermissionPackageApplication | null;
  readinessCount: ReadinessCheckCount | null;
  rows: readonly EnvCheckRow[];
  unreachableTargets: readonly UnreachableTarget[];
}

const emptySummary: EnvCheckSummary = { abnormal: 0, error: 0, warning: 0 };
const idleResult: EnvChecksResult = {
  latestApplication: null,
  readinessCount: null,
  rows: [],
  unreachableTargets: [],
};

// Runs the four cockpit checks (plan P3 note): API contract, an MCP probe at
// the preferred registered target, the core permission path of the latest
// application, and environment health (management catalog plus the other
// registered targets, which only warn). Probes are read-only.
export function useEnvChecks(data: RedesignData): EnvChecksState {
  const [result, setResult] = useState<EnvChecksResult>(idleResult);
  const [checking, setChecking] = useState(false);
  const runIdRef = useRef(0);

  const snapshot = data.data;
  const probeSupported = data.capabilities.has("target_probe_v1");

  const run = useCallback(async () => {
    const runId = ++runIdRef.current;
    setChecking(true);
    const stale = () => runId !== runIdRef.current;

    const [healthResult, catalogResult, applicationsResult] = await Promise.allSettled([
      checkApiHealth(),
      fetchManagementMcpToolsCatalog(),
      fetchPermissionPackageApplications({ limit: 5 }),
    ]);
    if (stale()) return;
    const health = healthResult.status === "fulfilled" ? healthResult.value : null;
    const catalog = catalogResult.status === "fulfilled" ? catalogResult.value : null;
    const latestApplication = latestApplicationOf(
      applicationsResult.status === "fulfilled" ? applicationsResult.value : [],
    );

    const readiness = latestApplication
      ? await fetchPermissionPackageProductionReadiness(readinessFilterFor(latestApplication)).catch(() => null)
      : null;

    const agents = snapshot?.agents ?? [];
    const capabilities = snapshot?.capabilities ?? [];
    const preferred = preferredProbeTarget(agents, capabilities);
    const others = preferred ? registeredMcpTargets(agents).filter((agent) => agent.id !== preferred.agent.id) : [];
    const probe = preferred && probeSupported ? await probeTarget(preferred.agent.id).catch(() => null) : null;
    const otherProbes = probeSupported
      ? await Promise.all(others.slice(0, 4).map((agent) => probeTarget(agent.id).catch(() => null)))
      : [];
    if (stale()) return;

    const unreachableTargets = otherProbes
      .map((probeResult, index) => ({ agent: others[index], probeResult }))
      .filter((entry) => entry.probeResult?.status === "error")
      .map((entry) => ({ endpoint: agentEndpoint(entry.agent), name: entry.agent.name }));
    const catalogDiagnostic: ManagementMcpCatalogDiagnostic | null = catalog
      ? managementMcpCatalogDiagnosticFromResult(catalog)
      : null;

    const rows: EnvCheckRow[] = [
      apiServiceCheck({
        apiBase: snapshot?.apiBase ?? "",
        apiHealthMessage: health && health.status === "error" ? health.message : null,
        contractIssues: health?.contractIssues ?? (health ? [] : ["api"]),
      }),
      mcpServiceCheck({
        endpoint: preferred?.endpoint ?? "",
        noTarget: !preferred,
        probe,
        unsupported: !probeSupported,
      }),
      corePathCheck({
        allowedTrace: (snapshot?.traces ?? []).find(
          (trace) => trace.decision === "allowed"
            && (!latestApplication || trace.callerInstanceId === latestApplication.callerInstanceId),
        ) ?? null,
        application: latestApplication,
        appliedAudit: (snapshot?.auditEvents ?? []).find((event) => event.action === "permission_package.applied") ?? null,
      }),
      envHealthCheck({
        catalogDetail: catalogDiagnosticDetail(catalogDiagnostic),
        catalogIssues: catalogDiagnosticIssues(catalogDiagnostic),
        unreachable: unreachableTargets,
      }),
    ];

    setChecking(false);
    rememberEnvCheckSnapshot(rows);
    setResult({ latestApplication, readinessCount: readinessCheckCount(readiness, null), rows, unreachableTargets });
  }, [probeSupported, snapshot]);

  useEffect(() => {
    if (!snapshot) return;
    void run();
  }, [run, snapshot]);

  return {
    ...result,
    checking,
    recheck: run,
    summary: result.rows.length > 0 ? envCheckSummary(result.rows) : emptySummary,
  };
}

function latestApplicationOf(applications: readonly PermissionPackageApplication[]): PermissionPackageApplication | null {
  if (applications.length === 0) return null;
  return [...applications].sort((a, b) => Date.parse(b.appliedAt) - Date.parse(a.appliedAt))[0] ?? null;
}

function readinessFilterFor(application: PermissionPackageApplication) {
  return {
    callerInstanceId: application.callerInstanceId,
    region: application.region || undefined,
    requestText: application.requestText || undefined,
    requestedCapabilityId: application.requestedCapabilityId,
    subjectId: subjectIdExampleFromSelector(application.subjectSelector) || undefined,
    subjectSelector: application.subjectSelector || undefined,
    targetId: application.targetId,
    templateId: application.templateId,
    tenantId: application.tenantId,
    workspaceId: application.workspaceId,
  };
}

function agentEndpoint(agent: Agent): string {
  const endpoint = (agent.channelConfig as { endpoint?: unknown } | undefined)?.endpoint;
  return typeof endpoint === "string" ? endpoint : "";
}

function catalogDiagnosticIssues(catalog: ManagementMcpCatalogDiagnostic | null): string[] {
  if (!catalog) return [];
  const issues: string[] = [];
  if (catalog.status === "error") issues.push(catalog.message ?? "management tool catalog unavailable");
  if (catalog.missingRequiredTools && catalog.missingRequiredTools.length > 0) {
    issues.push(`missing tools: ${catalog.missingRequiredTools.join(", ")}`);
  }
  return issues;
}

function catalogDiagnosticDetail(catalog: ManagementMcpCatalogDiagnostic | null): string | null {
  if (!catalog || catalog.status === "ok") return null;
  if (catalog.metadataVersion !== undefined) return `metadataVersion ${catalog.metadataVersion}`;
  return catalog.message ?? `status ${catalog.status}`;
}
