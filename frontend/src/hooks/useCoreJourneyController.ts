import { useEffect, useState, type Dispatch, type SetStateAction } from "react";
import {
  callMcpRpc,
  checkApiHealth,
  checkMockMcpHealth,
  createAgent,
  createAgentKey,
  createInstanceAssignment,
  createTenant,
  createTenantEntitlement,
  createWorkspaceAssignment,
  defaultMockMcpHealthUrl,
  loadConsoleData,
  loadTenantAccessProfile,
  refreshTargetCapabilities,
  updateCapability
} from "../api";
import {
  healthCheckFailureDetail
} from "../healthCheckPresentation";
import {
  createCoreJourneyConfig,
  defaultCoreJourneyForm,
  type CoreJourneyConfig,
  type CoreJourneyForm
} from "../coreJourney";
import {
  coreJourneyPreflightCanRun,
  defaultCoreJourneyPreflight,
  type CoreJourneyPreflightState
} from "../coreJourneyPreflight";
import type { Translator } from "../consolePresenters";
import type { Language } from "../i18n";
import {
  journeyCompletionRefreshFailedMessageKey,
  refreshAfterJourneyCompletion
} from "../journeyCompletionRefresh";
import type {
  AccessProfileFilters,
  ConsoleData,
  DataScope,
  ManagementScope,
  TenantAccessProfileData,
  TraceDecision,
  TraceFilters
} from "../types";

interface CoreJourneyRunResult {
  allowedStatus: number;
  callerId: string;
  deniedStatus: number;
  targetId: string;
  toolListStatus: number;
}

const coreJourneyCallerName = "Core Journey Caller";
const coreJourneyTargetName = "Core Journey MCP Target";

interface UseCoreJourneyControllerArgs {
  adminKey: string;
  defaultAccessFilters: AccessProfileFilters;
  defaultScope: ManagementScope;
  defaultTraceFilters: TraceFilters;
  enabled: boolean;
  language: Language;
  setAccessFilters: (filters: AccessProfileFilters) => void;
  setAccessProfile: Dispatch<SetStateAction<TenantAccessProfileData | null>>;
  setData: Dispatch<SetStateAction<ConsoleData | null>>;
  setLastRefresh: Dispatch<SetStateAction<Date>>;
  setLoadError: Dispatch<SetStateAction<string>>;
  setScope: Dispatch<SetStateAction<ManagementScope>>;
  setTraceFilters: Dispatch<SetStateAction<TraceFilters>>;
  t: Translator;
}

export function useCoreJourneyController({
  adminKey,
  defaultAccessFilters,
  defaultScope,
  defaultTraceFilters,
  enabled,
  language,
  setAccessFilters,
  setAccessProfile,
  setData,
  setLastRefresh,
  setLoadError,
  setScope,
  setTraceFilters,
  t
}: UseCoreJourneyControllerArgs) {
  const [form, setForm] = useState<CoreJourneyForm>(defaultCoreJourneyForm);
  const [config, setConfig] = useState<CoreJourneyConfig>(() => createCoreJourneyConfig());
  const [message, setMessage] = useState("");
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<CoreJourneyRunResult | null>(null);
  const [preflight, setPreflight] = useState<CoreJourneyPreflightState>(defaultCoreJourneyPreflight);
  const [preflightChecking, setPreflightChecking] = useState(false);
  const [preflightMessage, setPreflightMessage] = useState("");

  useEffect(() => {
    if (!enabled) return;
    void refreshPreflight();
  }, [enabled]);

  async function refreshPreflight() {
    if (!enabled) return;
    setPreflightChecking(true);
    setPreflightMessage(t("message.coreJourneyPreflightChecking"));
    setPreflight((current) => ({
      ...current,
      api: "pending",
      mockMcp: "pending"
    }));
    const [apiHealth, mockMcpHealth] = await Promise.all([
      checkApiHealth(),
      checkMockMcpHealth(mockMcpHealthUrlFromEndpoint(form.mcpEndpoint))
    ]);
    const nextPreflight: CoreJourneyPreflightState = {
      api: apiHealth.status === "ok" ? "ok" : "error",
      mockMcp: mockMcpHealth.status === "ok" ? "ok" : "error",
      privateUpstreams: "warning"
    };
    setPreflight(nextPreflight);
    if (coreJourneyPreflightCanRun(nextPreflight)) {
      setPreflightMessage(t("message.coreJourneyPreflightReady"));
    } else {
      const detail = [
        apiHealth.status === "ok" ? "" : healthCheckFailureDetail(t, t("preflight.api.title"), apiHealth),
        mockMcpHealth.status === "ok" ? "" : healthCheckFailureDetail(t, t("preflight.mockMcp.title"), mockMcpHealth)
      ].filter(Boolean).join(" · ");
      setPreflightMessage(tx(t, "message.coreJourneyPreflightFailed", { detail: detail || "unknown" }));
    }
    setPreflightChecking(false);
  }

  async function resetSession() {
    const nextConfig = createCoreJourneyConfig(form);
    setConfig(nextConfig);
    setResult(null);
    setMessage(t("message.coreJourneyReset"));
    setTraceFilters(defaultTraceFilters);
    setAccessFilters(defaultAccessFilters);
    setAccessProfile(null);
    setScope(defaultScope);
    try {
      setLoadError("");
      const nextData = await loadConsoleData(adminKey, defaultTraceFilters, normalizedScope(defaultScope, defaultScope));
      setData(nextData);
      setLastRefresh(new Date());
    } catch (error) {
      setLoadError(localizedErrorMessage(t, language, error, "error.consoleDataUnavailable"));
    }
    await refreshPreflight();
  }

  async function run() {
    const nextConfig = createCoreJourneyConfig(form);
    if (!coreJourneyPreflightCanRun(preflight)) {
      setMessage(t("message.coreJourneyPreflightBlocked"));
      await refreshPreflight();
      return;
    }
    const tenantScope: DataScope[] = [
      {
        dataDomain: "crm",
        region: "us-east",
        tenantFilter: `tenant_id = '${nextConfig.childTenantId}'`
      }
    ];
    setConfig(nextConfig);
    setResult(null);
    setRunning(true);
    setMessage(t("message.coreJourneyRunning"));
    try {
      await createTenant(
        {
          id: nextConfig.rootTenantId,
          name: "Core Journey Root",
          status: "active"
        },
        adminKey
      );
      await createTenant(
        {
          id: nextConfig.childTenantId,
          name: "Core Journey Team",
          parentTenantId: nextConfig.rootTenantId,
          status: "active"
        },
        adminKey
      );
      await createTenant(
        {
          id: nextConfig.grandchildTenantId,
          name: "Core Journey Project",
          parentTenantId: nextConfig.childTenantId,
          status: "active"
        },
        adminKey
      );

      const caller = await createAgent(
        {
          channelType: "local",
          description: "Core journey browser caller",
          name: coreJourneyCallerName,
          status: "active",
          tenantId: nextConfig.childTenantId,
          workspaceId: nextConfig.workspaceId
        },
        adminKey
      );
      const callerKey = await createAgentKey(
        {
          agentId: caller.id,
          expiresInSeconds: 900,
          name: "core journey key"
        },
        adminKey
      );
      const target = await createAgent(
        {
          channelConfig: {
            endpoint: nextConfig.mcpEndpoint,
            transport: "streamable-http"
          },
          channelType: "mcp",
          description: "Core journey MCP target",
          name: coreJourneyTargetName,
          status: "active",
          tenantId: nextConfig.rootTenantId,
          workspaceId: nextConfig.workspaceId
        },
        adminKey
      );

      const discovered = await refreshTargetCapabilities(target.id, adminKey);
      const allowedCapability = discovered.find((capability) => capability.key === nextConfig.allowedTool);
      const deniedCapability = discovered.find((capability) => capability.key === nextConfig.deniedTool);
      if (!allowedCapability || !deniedCapability) {
        throw new Error(tx(t, "message.coreJourneyMissingTools", { allowed: nextConfig.allowedTool, denied: nextConfig.deniedTool }));
      }
      const scopedCapability = await updateCapability(
        allowedCapability.id,
        {
          dataScopes: tenantScope,
          discoveryStatus: "approved"
        },
        adminKey
      );
      const entitlement = await createTenantEntitlement(
        {
          capabilityId: scopedCapability.id,
          effect: "allow",
          priority: 50,
          status: "enabled",
          targetId: target.id,
          tenantId: nextConfig.childTenantId
        },
        adminKey
      );
      const workspaceAssignment = await createWorkspaceAssignment(
        {
          dataScopes: [{ table: "accounts" }],
          effect: "allow",
          status: "enabled",
          tenantEntitlementId: entitlement.id,
          workspaceId: nextConfig.workspaceId
        },
        adminKey
      );
      await createInstanceAssignment(
        {
          callerInstanceId: caller.id,
          dataScopes: [{ field: "email" }],
          effect: "allow",
          status: "enabled",
          subjectSelector: nextConfig.subjectSelector,
          workspaceAssignmentId: workspaceAssignment.id
        },
        adminKey
      );

      const toolList = await callMcpRpc(
        target.id,
        mcpToolsListPayload(),
        callerKey.key,
        nextConfig.runId,
        adminKey,
        nextConfig.subjectId
      );
      if (!toolList.ok) throw new Error(tx(t, "message.coreJourneyRpcUnexpected", { status: toolList.status }));
      const listedTools = toolNamesFromPayload(toolList.payload);
      if (!listedTools.includes(nextConfig.allowedTool) || listedTools.includes(nextConfig.deniedTool)) {
        throw new Error(t("message.coreJourneyToolsListInvalid"));
      }
      const deniedCall = await callMcpRpc(
        target.id,
        mcpToolCallPayload(nextConfig.deniedTool),
        callerKey.key,
        nextConfig.runId,
        adminKey,
        nextConfig.subjectId
      );
      if (deniedCall.status !== 403) {
        throw new Error(tx(t, "message.coreJourneyDeniedUnexpected", { status: deniedCall.status }));
      }
      const allowedCall = await callMcpRpc(
        target.id,
        mcpToolCallPayload(nextConfig.allowedTool),
        callerKey.key,
        nextConfig.runId,
        adminKey,
        nextConfig.subjectId
      );
      if (!allowedCall.ok) throw new Error(tx(t, "message.coreJourneyRpcUnexpected", { status: allowedCall.status }));

      const nextTraceFilters = {
        callerAgentId: caller.id,
        decision: "" as TraceDecision | "",
        runId: nextConfig.runId,
        targetAgentId: target.id
      };
      const nextAccessFilters = {
        callerInstanceId: caller.id,
        capabilityId: "",
        subjectId: nextConfig.subjectId,
        targetId: target.id,
        traceLimit: "10",
        workspaceId: nextConfig.workspaceId
      };
      setTraceFilters(nextTraceFilters);
      setResult({
        allowedStatus: allowedCall.status,
        callerId: caller.id,
        deniedStatus: deniedCall.status,
        targetId: target.id,
        toolListStatus: toolList.status
      });
      const refreshResult = await refreshAfterJourneyCompletion({
        onRefresh: async () => {
          const [nextData, nextProfile] = await Promise.all([
            loadConsoleData(adminKey, nextTraceFilters),
            loadTenantAccessProfile(nextConfig.childTenantId, adminKey, {
              ...nextAccessFilters,
              traceLimit: 10
            })
          ]);
          return { nextData, nextProfile };
        }
      });
      if (refreshResult.ok) {
        setData(refreshResult.value.nextData);
        setAccessProfile(refreshResult.value.nextProfile);
        setLastRefresh(new Date());
      }
      setMessage(
        refreshResult.ok
          ? t("message.coreJourneyComplete")
          : t(journeyCompletionRefreshFailedMessageKey("core_journey"))
      );
    } catch (error) {
      setMessage(localizedErrorMessage(t, language, error, "error.coreJourneyFailed"));
    } finally {
      setRunning(false);
    }
  }

  return {
    config,
    form,
    message,
    preflight,
    preflightChecking,
    preflightMessage,
    refreshPreflight,
    resetSession,
    result,
    run,
    running,
    setForm
  };
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}

function localizedErrorMessage(t: Translator, language: Language, error: unknown, fallbackKey: string) {
  const fallback = t(fallbackKey);
  if (!(error instanceof Error) || !error.message.trim()) return fallback;
  if (language === "en" || /[\u4e00-\u9fa5]/.test(error.message)) {
    return error.message;
  }
  return fallback;
}

function normalizedScope(scope: ManagementScope, defaultScope: ManagementScope): ManagementScope {
  return {
    tenantId: scope.tenantId.trim() || defaultScope.tenantId,
    workspaceId: scope.workspaceId.trim() || defaultScope.workspaceId
  };
}

function mockMcpHealthUrlFromEndpoint(endpointValue: string) {
  try {
    const endpointUrl = new URL(endpointValue);
    endpointUrl.pathname = "/healthz";
    endpointUrl.search = "";
    endpointUrl.hash = "";
    return endpointUrl.toString();
  } catch {
    return defaultMockMcpHealthUrl;
  }
}

function mcpToolsListPayload() {
  return {
    id: "tools-list",
    jsonrpc: "2.0",
    method: "tools/list"
  };
}

function mcpToolCallPayload(toolName: string) {
  return {
    id: `call-${toolName}`,
    jsonrpc: "2.0",
    method: "tools/call",
    params: {
      arguments: {},
      name: toolName
    }
  };
}

function toolNamesFromPayload(payload: unknown) {
  if (!payload || typeof payload !== "object") return [];
  const result = "result" in payload ? (payload as { result?: unknown }).result : undefined;
  if (!result || typeof result !== "object") return [];
  const tools = "tools" in result ? (result as { tools?: unknown }).tools : undefined;
  if (!Array.isArray(tools)) return [];
  return tools
    .map((tool) => (tool && typeof tool === "object" && "name" in tool ? String((tool as { name?: unknown }).name) : ""))
    .filter(Boolean);
}
