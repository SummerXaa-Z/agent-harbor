import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";
import {
  Activity,
  Boxes,
  CheckCircle2,
  CircleDot,
  ClipboardCheck,
  Copy,
  DatabaseZap,
  ExternalLink,
  FileSearch,
  Filter,
  Gauge,
  KeyRound,
  Layers3,
  LockKeyhole,
  MoreHorizontal,
  Network,
  RefreshCw,
  Route,
  Search,
  ServerCog,
  ShieldCheck,
  Sparkles,
  TriangleAlert,
  Workflow
} from "lucide-react";
import {
  applyPermissionPackage,
  approvePermissionPackageApprovalRequest,
  callMcpRpc,
  checkApiHealth,
  checkMockMcpHealth,
  checkSubjectHeaderCors,
  createAgent,
  createAgentKey,
  createInstanceAssignment,
  createPermissionPackageApprovalRequest,
  createPermissionPackageDraftFromApi,
  createRoutePolicy,
  createTenant,
  createTenantEntitlement,
  createWorkspaceAssignment,
  defaultMockMcpHealthUrl,
  disableAgent,
  disableRoutePolicy,
  fetchAccessDecisionExplanation,
  fetchAuditEvents,
  fetchPermissionPackageApplicationImpact,
  fetchPermissionPackageApprovalRequests,
  fetchPermissionPackageTemplates,
  isApiCompatibilityFallbackError,
  loadConsoleData,
  loadTenantAccessProfile,
  rejectPermissionPackageApprovalRequest,
  refreshTargetCapabilities,
  rotateAgentCredentials,
  updateAgent,
  updateCapability
} from "./api";
import {
  countInvalidAccessProfileRows,
  countInvalidGrantRows,
  parseAccessProfileTraceLimit,
  scopeStatusTone,
  summarizeDataScopes
} from "./accessProfile";
import {
  runtimeEvidenceMetric,
  type MetricTone
} from "./consoleMetrics";
import {
  createTranslator,
  resolveInitialLanguage,
  type Language
} from "./i18n";
import {
  navItems,
  viewForNav,
  type NavKey
} from "./consoleNavigation";
import {
  createCoreJourneyConfig,
  defaultCoreJourneyForm,
  evaluateCoreJourney,
  type CoreJourneyConfig,
  type CoreJourneyEvaluation,
  type CoreJourneyForm,
  type CoreJourneyStep,
  type CoreJourneyStepStatus
} from "./coreJourney";
import {
  coreJourneyPreflightCanRun,
  coreJourneyPreflightRows,
  defaultCoreJourneyPreflight,
  type CoreJourneyPreflightState,
  type CoreJourneyPreflightStatus
} from "./coreJourneyPreflight";
import {
  createAiAdminApprovalJourneyConfig,
  evaluateAiAdminApprovalJourney,
  type AiAdminApprovalJourneyConfig,
  type AiAdminApprovalJourneyEvaluation,
  type AiAdminApprovalJourneyResult,
  type AiAdminApprovalJourneyStep,
  type AiAdminApprovalJourneyStepStatus
} from "./aiAdminApprovalJourney";
import {
  aiAdminApprovalReadinessCanRun,
  aiAdminApprovalReadinessRows,
  defaultAiAdminApprovalReadiness,
  type AiAdminApprovalReadinessRow,
  type AiAdminApprovalReadinessState,
  type AiAdminApprovalReadinessStatus
} from "./aiAdminApprovalReadiness";
import {
  createPermissionPackageDraft,
  permissionPackageTemplates,
  subjectIdExampleFromSelector,
  type PermissionPackageApplication,
  type PermissionPackageApplicationImpact,
  type PermissionPackageApprovalRequest,
  type PermissionPackageDraft,
  type PermissionPackageDraftInput,
  type PermissionPackageRemediationAction,
  type PermissionPackageSimulationRow,
  type PermissionPackageTemplate
} from "./permissionPackages";
import { parseRetryFields } from "./retryForm";
import type {
  AccessProfileFilters,
  AccessProfileSummary,
  AccessDecisionExplainRequest,
  AccessDecisionExplainResult,
  Agent,
  AgentStatus,
  AuditEvent,
  Capability,
  ChannelContract,
  ConsoleData,
  CreateAgentKeyResponse,
  DataScope,
  EvidenceRun,
  InstanceAssignment,
  JsonObject,
  ManagementScope,
  RoutePolicy,
  SystemMetric,
  TenantAccessProfileData,
  TenantAccessProfileGrant,
  TenantAccessProfileInstance,
  TenantAccessProfileWorkspace,
  TenantEntitlement,
  TraceDecision,
  TraceEvent,
  TraceFilters,
  WorkspaceAssignment
} from "./types";

type Tone = MetricTone;
type Translator = ReturnType<typeof createTranslator>;

interface CoreJourneyRunResult {
  allowedStatus: number;
  callerId: string;
  deniedStatus: number;
  targetId: string;
  toolListStatus: number;
}

const emptyAccessProfileSummary: AccessProfileSummary = {
  tenantCount: 0,
  grantCount: 0,
  targetCount: 0,
  capabilityCount: 0,
  workspaceAssignmentCount: 0,
  instanceAssignmentCount: 0,
  recentAllowedTraceCount: 0,
  recentDeniedTraceCount: 0
};

const workspaceTabs = ["Prod", "Staging", "Sandbox"];
const defaultManagementScope: ManagementScope = {
  tenantId: "default",
  workspaceId: "workspace-sandbox"
};
const languageStorageKey = "agent-harbor-language";
const defaultAgentForm = {
  channelType: "local",
  credentialHeader: "",
  credentialName: "",
  credentialValue: "",
  description: "",
  endpoint: "",
  name: "",
  retryBackoffMs: "0",
  retryMaxAttempts: "1",
  status: "draft" as AgentStatus
};
const defaultKeyForm = { agentId: "", expiresInSeconds: "900", name: "console key" };
const defaultRotateForm = { agentId: "", credentialName: "apiToken", credentialValue: "" };
const defaultPolicyForm = { callerAgentId: "", effect: "allow", name: "", priority: "100", retryBackoffMs: "0", retryMaxAttempts: "1", routeKey: "", routeType: "mcp", targetAgentId: "" };
const defaultCapabilityGrantForm = {
  callerInstanceId: "",
  capabilityId: "",
  subjectSelector: "",
  targetId: "",
  tenantId: defaultManagementScope.tenantId,
  workspaceId: defaultManagementScope.workspaceId
};
const defaultAiAdminForm: PermissionPackageDraftInput = {
  callerInstanceId: "",
  region: "华东",
  requestText: "给销售助手开通当前租户的客户只读访问，禁止导出合同和访问财务字段。",
  subjectSelector: "user:*",
  targetId: "",
  templateId: "sales-readonly",
  tenantId: defaultManagementScope.tenantId,
  workspaceId: defaultManagementScope.workspaceId
};

function initialLanguage(): Language {
  if (typeof window === "undefined") {
    return "en";
  }
  const browserLanguages = Array.from(
    window.navigator.languages?.length ? window.navigator.languages : [window.navigator.language]
  ).filter(Boolean);
  try {
    return resolveInitialLanguage(window.localStorage.getItem(languageStorageKey), browserLanguages);
  } catch {
    return resolveInitialLanguage(undefined, browserLanguages);
  }
}
const defaultTraceFilters: TraceFilters = { callerAgentId: "", decision: "", runId: "", targetAgentId: "" };
const defaultAccessProfileFilters: AccessProfileFilters = {
  callerInstanceId: "",
  capabilityId: "",
  subjectId: "",
  targetId: "",
  traceLimit: "20",
  workspaceId: ""
};
const mcpRouteKeyPresets = ["initialize", "tools/list", "tools/call"];
const coreJourneyCallerName = "Core Journey Caller";
const coreJourneyTargetName = "Core Journey MCP Target";

function navIconFor(key: NavKey) {
  switch (key) {
    case "ai-admin":
      return Sparkles;
    case "registry":
      return Boxes;
    case "routes":
      return Route;
    case "policies":
      return ShieldCheck;
    case "capabilities":
      return DatabaseZap;
    case "access":
      return LockKeyhole;
    case "traces":
      return FileSearch;
    case "evidence":
      return ClipboardCheck;
    case "cockpit":
    default:
      return Gauge;
  }
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}

function isAbortError(error: unknown) {
  return error instanceof Error && error.name === "AbortError";
}

function translatedValue(t: Translator, value?: string) {
  return value ? t(`value.${value}`, value) : "";
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

function agentStatusLabel(status: AgentStatus, t: Translator) {
  if (status === "active") return t("status.agentActive");
  if (status === "disabled") return t("status.agentDisabled");
  return t("status.agentDraft");
}

function policyEffectLabel(effect: "allow" | "deny", t: Translator) {
  return effect === "allow" ? t("status.policyAllow") : t("status.policyDeny");
}

function App() {
  const [activeNav, setActiveNav] = useState("cockpit");
  const [activeWorkspace, setActiveWorkspace] = useState("Prod");
  const [adminKey, setAdminKey] = useState("");
  const [scope, setScope] = useState<ManagementScope>(defaultManagementScope);
  const [data, setData] = useState<ConsoleData | null>(null);
  const [loadError, setLoadError] = useState("");
  const [lastRefresh, setLastRefresh] = useState(new Date());
  const [agentForm, setAgentForm] = useState(defaultAgentForm);
  const [agentMessage, setAgentMessage] = useState("");
  const [keyForm, setKeyForm] = useState(defaultKeyForm);
  const [keyMessage, setKeyMessage] = useState("");
  const [createdKey, setCreatedKey] = useState<CreateAgentKeyResponse | null>(null);
  const [rotateForm, setRotateForm] = useState(defaultRotateForm);
  const [rotateMessage, setRotateMessage] = useState("");
  const [policyForm, setPolicyForm] = useState(defaultPolicyForm);
  const [policyMessage, setPolicyMessage] = useState("");
  const [capabilityForm, setCapabilityForm] = useState(defaultCapabilityGrantForm);
  const [capabilityMessage, setCapabilityMessage] = useState("");
  const [capabilityActionId, setCapabilityActionId] = useState("");
  const [cleanupActionId, setCleanupActionId] = useState("");
  const [traceFilters, setTraceFilters] = useState<TraceFilters>(defaultTraceFilters);
  const [accessFilters, setAccessFilters] = useState<AccessProfileFilters>(defaultAccessProfileFilters);
  const [accessLoading, setAccessLoading] = useState(false);
  const [accessMessage, setAccessMessage] = useState("");
  const [accessProfile, setAccessProfile] = useState<TenantAccessProfileData | null>(null);
  const [accessDecisionExplanation, setAccessDecisionExplanation] = useState<AccessDecisionExplainResult | null>(null);
  const [accessDecisionExplainLoading, setAccessDecisionExplainLoading] = useState(false);
  const [accessDecisionExplainMessage, setAccessDecisionExplainMessage] = useState("");
  const [language, setLanguage] = useState<Language>(initialLanguage);
  const [coreJourneyForm, setCoreJourneyForm] = useState<CoreJourneyForm>(defaultCoreJourneyForm);
  const [coreJourneyConfig, setCoreJourneyConfig] = useState<CoreJourneyConfig>(() => createCoreJourneyConfig());
  const [coreJourneyMessage, setCoreJourneyMessage] = useState("");
  const [coreJourneyRunning, setCoreJourneyRunning] = useState(false);
  const [coreJourneyResult, setCoreJourneyResult] = useState<CoreJourneyRunResult | null>(null);
  const [coreJourneyPreflight, setCoreJourneyPreflight] = useState<CoreJourneyPreflightState>(defaultCoreJourneyPreflight);
  const [coreJourneyPreflightChecking, setCoreJourneyPreflightChecking] = useState(false);
  const [coreJourneyPreflightMessage, setCoreJourneyPreflightMessage] = useState("");
  const [aiAdminForm, setAiAdminForm] = useState<PermissionPackageDraftInput>(defaultAiAdminForm);
  const [aiAdminMessage, setAiAdminMessage] = useState("");
  const [aiAdminApplying, setAiAdminApplying] = useState(false);
  const [aiAdminTemplates, setAiAdminTemplates] = useState<PermissionPackageTemplate[]>(permissionPackageTemplates);
  const [aiAdminServerDraft, setAiAdminServerDraft] = useState<PermissionPackageDraft | null>(null);
  const [aiAdminApplication, setAiAdminApplication] = useState<PermissionPackageApplication | null>(null);
  const [aiAdminApplicationImpact, setAiAdminApplicationImpact] =
    useState<PermissionPackageApplicationImpact | null>(null);
  const [aiAdminApplicationImpactLoading, setAiAdminApplicationImpactLoading] = useState(false);
  const [aiAdminApplicationImpactMessage, setAiAdminApplicationImpactMessage] = useState("");
  const [aiAdminApprovalRequests, setAiAdminApprovalRequests] = useState<PermissionPackageApprovalRequest[]>([]);
  const [aiAdminApprovalAction, setAiAdminApprovalAction] = useState<"" | "create" | "approve" | "reject">("");
  const [aiAdminApprovalReviewer, setAiAdminApprovalReviewer] = useState("AI Admin");
  const [aiAdminReviewerQueueLoading, setAiAdminReviewerQueueLoading] = useState(false);
  const [aiAdminReviewerQueueMessage, setAiAdminReviewerQueueMessage] = useState("");
  const [aiAdminSelectedApprovalRequestId, setAiAdminSelectedApprovalRequestId] = useState("");
  const [aiAdminAccessDecisionExplanation, setAiAdminAccessDecisionExplanation] =
    useState<AccessDecisionExplainResult | null>(null);
  const [aiAdminAccessDecisionExplainLoading, setAiAdminAccessDecisionExplainLoading] = useState(false);
  const [aiAdminAccessDecisionExplainMessage, setAiAdminAccessDecisionExplainMessage] = useState("");
  const [aiAdminApprovalJourneyConfig, setAiAdminApprovalJourneyConfig] = useState<AiAdminApprovalJourneyConfig>(() =>
    createAiAdminApprovalJourneyConfig()
  );
  const [aiAdminApprovalJourneyRunning, setAiAdminApprovalJourneyRunning] = useState(false);
  const [aiAdminApprovalJourneyMessage, setAiAdminApprovalJourneyMessage] = useState("");
  const [aiAdminApprovalJourneyResult, setAiAdminApprovalJourneyResult] =
    useState<AiAdminApprovalJourneyResult | null>(null);
  const [aiAdminApprovalAuditEvent, setAiAdminApprovalAuditEvent] = useState<AuditEvent | null>(null);
  const [aiAdminApprovalReadiness, setAiAdminApprovalReadiness] =
    useState<AiAdminApprovalReadinessState>(defaultAiAdminApprovalReadiness);
  const [aiAdminApprovalReadinessChecking, setAiAdminApprovalReadinessChecking] = useState(false);
  const [aiAdminApprovalReadinessMessage, setAiAdminApprovalReadinessMessage] = useState("");
  const t = useMemo(() => createTranslator(language), [language]);

  useEffect(() => {
    void refresh();
  }, []);

  useEffect(() => {
    void refreshCoreJourneyPreflight();
  }, []);

  useEffect(() => {
    if (typeof document !== "undefined") {
      document.documentElement.lang = language;
    }
    try {
      window.localStorage.setItem(languageStorageKey, language);
    } catch {
      // The UI still works when storage is unavailable.
    }
  }, [language]);

  useEffect(() => {
    if (activeNav === "access" && !accessProfile && !accessLoading) {
      void refreshAccessProfile();
    }
  }, [activeNav]);

  useEffect(() => {
    if (activeNav === "ai-admin") {
      void refreshAiAdminCatalog();
      void refreshAiAdminApprovalReadiness();
    }
  }, [activeNav]);

  useEffect(() => {
    if (!data) return;
    setCapabilityForm((current) => {
      const mcpTarget = data.agents.find((agent) => agent.channelType === "mcp");
      const targetId = current.targetId || mcpTarget?.id || "";
      const capability = data.capabilities.find((item) => item.id === current.capabilityId && item.targetId === targetId)
        ?? data.capabilities.find((item) => item.targetId === targetId)
        ?? data.capabilities[0];
      const caller = data.agents.find(
        (agent) =>
          agent.status === "active" &&
          agent.tenantId === current.tenantId &&
          agent.workspaceId === current.workspaceId &&
          agent.channelType === "local"
      ) ?? data.agents.find((agent) => agent.status === "active" && agent.channelType === "local");
      const next = {
        ...current,
        callerInstanceId: current.callerInstanceId || caller?.id || "",
        capabilityId: current.capabilityId || capability?.id || "",
        targetId
      };
      return shallowEqualCapabilityForm(current, next) ? current : next;
    });
  }, [data]);

  useEffect(() => {
    if (!data) return;
    setAiAdminForm((current) => {
      const requestScope = normalizedScope(scope);
      const mcpTarget = data.agents.find((agent) => agent.channelType === "mcp" && agent.status === "active")
        ?? data.agents.find((agent) => agent.channelType === "mcp");
      const targetId = current.targetId || mcpTarget?.id || "";
      const capabilityRegion = data.capabilities
        .find((capability) => capability.targetId === targetId && capability.action === "read")
        ?.dataScopes?.find((scopeItem) => scopeItem.region)?.region;
      const caller = data.agents.find(
        (agent) =>
          agent.status === "active" &&
          agent.channelType === "local" &&
          agent.tenantId === requestScope.tenantId &&
          agent.workspaceId === requestScope.workspaceId
      ) ?? data.agents.find((agent) => agent.status === "active" && agent.channelType === "local");
      const next = {
        ...current,
        callerInstanceId: current.callerInstanceId || caller?.id || "",
        region: current.region === defaultAiAdminForm.region && capabilityRegion ? capabilityRegion : current.region,
        targetId,
        tenantId: current.tenantId === defaultManagementScope.tenantId && caller?.tenantId
          ? caller.tenantId
          : current.tenantId || requestScope.tenantId,
        workspaceId: current.workspaceId === defaultManagementScope.workspaceId && caller?.workspaceId
          ? caller.workspaceId
          : current.workspaceId || requestScope.workspaceId
      };
      return shallowEqualAiAdminForm(current, next) ? current : next;
    });
  }, [data, scope]);

  useEffect(() => {
    if (activeNav !== "ai-admin" || !data?.loadedFromApi) {
      setAiAdminServerDraft(null);
      return;
    }
    const controller = new AbortController();
    createPermissionPackageDraftFromApi(aiAdminForm, adminKey, controller.signal)
      .then((draft) => setAiAdminServerDraft(draft))
      .catch((error) => {
        if (!isAbortError(error)) {
          setAiAdminServerDraft(null);
        }
      });
    return () => controller.abort();
  }, [activeNav, adminKey, aiAdminForm, data?.loadedFromApi]);

  async function refresh() {
    try {
      setLoadError("");
      const next = await loadConsoleData(
        adminKey,
        traceFilters,
        activeNav === "ai-admin" ? undefined : normalizedScope(scope)
      );
      setData(next);
      setLastRefresh(new Date());
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "console data unavailable");
    }
    if (activeNav === "access") {
      await refreshAccessProfile();
    }
  }

  async function refreshAiAdminCatalog() {
    try {
      setLoadError("");
      const [next, templates] = await Promise.all([
        loadConsoleData(adminKey, traceFilters),
        fetchPermissionPackageTemplates(adminKey).catch(() => permissionPackageTemplates)
      ]);
      setData(next);
      setAiAdminTemplates(templates.length > 0 ? templates : permissionPackageTemplates);
      setLastRefresh(new Date());
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "console data unavailable");
    }
  }

  async function refreshAccessProfile() {
    const traceLimit = parseAccessProfileTraceLimit(accessFilters.traceLimit);
    if (!traceLimit.ok) {
      setAccessMessage(traceLimit.message);
      return;
    }
    const requestScope = normalizedScope(scope);
    setAccessLoading(true);
    setAccessMessage("");
    try {
      const next = await loadTenantAccessProfile(requestScope.tenantId, adminKey, {
        ...accessFilters,
        traceLimit: traceLimit.value
      });
      setAccessProfile(next);
      setAccessMessage(next.loadedFromApi ? t("status.profileRefreshed") : t("status.profileFallback"));
    } catch (error) {
      setAccessMessage(error instanceof Error ? error.message : "Unable to load tenant access profile");
    } finally {
      setAccessLoading(false);
    }
  }

  async function explainAccessDecisionFromProfile() {
    if (!data?.loadedFromApi) {
      setAccessDecisionExplainMessage(t("message.accessDecisionExplainRequiresLiveApi"));
      return;
    }
    const requestScope = normalizedScope(scope);
    const request: AccessDecisionExplainRequest = {
      callerInstanceId: accessFilters.callerInstanceId?.trim() ?? "",
      capabilityId: accessFilters.capabilityId?.trim() ?? "",
      subjectId: accessFilters.subjectId?.trim() || undefined,
      targetId: accessFilters.targetId?.trim() ?? "",
      tenantId: requestScope.tenantId,
      workspaceId: accessFilters.workspaceId?.trim() || requestScope.workspaceId
    };
    if (!accessDecisionExplainRequestComplete(request)) {
      setAccessDecisionExplainMessage(t("message.accessDecisionExplainMissingFields"));
      return;
    }
    setAccessDecisionExplainLoading(true);
    setAccessDecisionExplainMessage("");
    try {
      const next = await fetchAccessDecisionExplanation(request, adminKey);
      setAccessDecisionExplanation(next);
      setAccessDecisionExplainMessage(t("message.accessDecisionExplainLoaded"));
    } catch (error) {
      setAccessDecisionExplainMessage(error instanceof Error ? error.message : "Unable to explain access decision");
    } finally {
      setAccessDecisionExplainLoading(false);
    }
  }

  async function explainAiAdminAccessDecision() {
    if (!data?.loadedFromApi) {
      setAiAdminAccessDecisionExplainMessage(t("message.accessDecisionExplainRequiresLiveApi"));
      return;
    }
    const capability = aiAdminDraft.allowedCapabilities[0];
    const request: AccessDecisionExplainRequest = {
      callerInstanceId: aiAdminDraft.input.callerInstanceId,
      capabilityId: capability?.id ?? "",
      subjectId: subjectIdExampleFromSelector(aiAdminDraft.input.subjectSelector),
      targetId: aiAdminDraft.input.targetId,
      tenantId: aiAdminDraft.input.tenantId,
      workspaceId: aiAdminDraft.input.workspaceId
    };
    if (!accessDecisionExplainRequestComplete(request)) {
      setAiAdminAccessDecisionExplainMessage(
        capability ? t("message.accessDecisionExplainMissingFields") : t("message.noMatchingAllowedCapabilities")
      );
      return;
    }
    setAiAdminAccessDecisionExplainLoading(true);
    setAiAdminAccessDecisionExplainMessage("");
    try {
      const next = await fetchAccessDecisionExplanation(request, adminKey);
      setAiAdminAccessDecisionExplanation(next);
      setAiAdminAccessDecisionExplainMessage(t("message.accessDecisionExplainLoaded"));
    } catch (error) {
      setAiAdminAccessDecisionExplainMessage(error instanceof Error ? error.message : "Unable to explain access decision");
    } finally {
      setAiAdminAccessDecisionExplainLoading(false);
    }
  }

  async function reviewAiAdminApplicationImpact() {
    if (!data?.loadedFromApi) {
      setAiAdminApplicationImpactMessage(t("message.permissionApplicationImpactRequiresLiveApi"));
      return;
    }
    if (!aiAdminApplication) {
      setAiAdminApplicationImpactMessage(t("message.aiAdminApprovalJourneyMissingApplication"));
      return;
    }
    setAiAdminApplicationImpactLoading(true);
    setAiAdminApplicationImpactMessage("");
    try {
      const next = await fetchPermissionPackageApplicationImpact(
        aiAdminApplication.id,
        {
          tenantId: aiAdminApplication.tenantId,
          workspaceId: aiAdminApplication.workspaceId
        },
        adminKey
      );
      setAiAdminApplicationImpact(next);
      setAiAdminApplicationImpactMessage(t("message.permissionApplicationImpactLoaded"));
    } catch (error) {
      setAiAdminApplicationImpactMessage(error instanceof Error ? error.message : "Unable to review application impact");
    } finally {
      setAiAdminApplicationImpactLoading(false);
    }
  }

  async function refreshCoreJourneyPreflight() {
    setCoreJourneyPreflightChecking(true);
    setCoreJourneyPreflightMessage(t("message.coreJourneyPreflightChecking"));
    setCoreJourneyPreflight((current) => ({
      ...current,
      api: "pending",
      mockMcp: "pending"
    }));
    const [apiHealth, mockMcpHealth] = await Promise.all([
      checkApiHealth(),
      checkMockMcpHealth(mockMcpHealthUrlFromEndpoint(coreJourneyForm.mcpEndpoint))
    ]);
    const nextPreflight: CoreJourneyPreflightState = {
      api: apiHealth.status === "ok" ? "ok" : "error",
      mockMcp: mockMcpHealth.status === "ok" ? "ok" : "error",
      privateUpstreams: "warning"
    };
    setCoreJourneyPreflight(nextPreflight);
    if (coreJourneyPreflightCanRun(nextPreflight)) {
      setCoreJourneyPreflightMessage(t("message.coreJourneyPreflightReady"));
    } else {
      const detail = [
        apiHealth.status === "ok" ? "" : `API ${apiHealth.message}`,
        mockMcpHealth.status === "ok" ? "" : `Mock MCP ${mockMcpHealth.message}`
      ].filter(Boolean).join(" · ");
      setCoreJourneyPreflightMessage(tx(t, "message.coreJourneyPreflightFailed", { detail: detail || "unknown" }));
    }
    setCoreJourneyPreflightChecking(false);
  }

  async function checkAiAdminApprovalReadiness(config: AiAdminApprovalJourneyConfig) {
    const [apiHealth, mockMcpHealth, subjectHeaderHealth] = await Promise.all([
      checkApiHealth(),
      checkMockMcpHealth(mockMcpHealthUrlFromEndpoint(config.mcpEndpoint)),
      checkSubjectHeaderCors()
    ]);
    const nextReadiness: AiAdminApprovalReadinessState = {
      api: apiHealth.status === "ok" ? "ok" : "error",
      dataSource: data?.loadedFromApi ? "ok" : "warning",
      mockMcp: mockMcpHealth.status === "ok" ? "ok" : "error",
      privateUpstreams: "warning",
      subjectHeader: subjectHeaderHealth.status === "ok" ? "ok" : "error"
    };
    const detail = [
      apiHealth.status === "ok" ? "" : `API ${apiHealth.message}`,
      mockMcpHealth.status === "ok" ? "" : `Mock MCP ${mockMcpHealth.message}`,
      subjectHeaderHealth.status === "ok" ? "" : `Subject header ${subjectHeaderHealth.message}`
    ].filter(Boolean).join(" · ");
    return {
      detail,
      message: aiAdminApprovalReadinessCanRun(nextReadiness)
        ? t("message.aiAdminReadinessReady")
        : tx(t, "message.aiAdminReadinessFailed", { detail: detail || "unknown" }),
      state: nextReadiness
    };
  }

  async function refreshAiAdminApprovalReadiness(config = aiAdminApprovalJourneyConfig) {
    setAiAdminApprovalReadinessChecking(true);
    setAiAdminApprovalReadinessMessage(t("message.aiAdminReadinessChecking"));
    setAiAdminApprovalReadiness((current) => ({
      ...current,
      api: "pending",
      dataSource: data?.loadedFromApi ? "ok" : "pending",
      mockMcp: "pending",
      subjectHeader: "pending"
    }));
    try {
      const result = await checkAiAdminApprovalReadiness(config);
      setAiAdminApprovalReadiness(result.state);
      setAiAdminApprovalReadinessMessage(result.message);
      return result;
    } finally {
      setAiAdminApprovalReadinessChecking(false);
    }
  }

  async function resetCoreJourneySession() {
    const resetScope = defaultManagementScope;
    const resetTraceFilters = defaultTraceFilters;
    const resetAccessFilters = defaultAccessProfileFilters;
    const nextConfig = createCoreJourneyConfig(coreJourneyForm);
    setCoreJourneyConfig(nextConfig);
    setCoreJourneyResult(null);
    setCoreJourneyMessage(t("message.coreJourneyReset"));
    setTraceFilters(resetTraceFilters);
    setAccessFilters(resetAccessFilters);
    setAccessProfile(null);
    setScope(resetScope);
    try {
      setLoadError("");
      const nextData = await loadConsoleData(adminKey, resetTraceFilters, normalizedScope(resetScope));
      setData(nextData);
      setLastRefresh(new Date());
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "console data unavailable");
    }
    await refreshCoreJourneyPreflight();
  }

  async function runCoreJourney() {
    const nextConfig = createCoreJourneyConfig(coreJourneyForm);
    if (!coreJourneyPreflightCanRun(coreJourneyPreflight)) {
      setCoreJourneyMessage(t("message.coreJourneyPreflightBlocked"));
      await refreshCoreJourneyPreflight();
      return;
    }
    const tenantScope: DataScope[] = [
      {
        dataDomain: "crm",
        region: "us-east",
        tenantFilter: `tenant_id = '${nextConfig.childTenantId}'`
      }
    ];
    setCoreJourneyConfig(nextConfig);
    setCoreJourneyResult(null);
    setCoreJourneyRunning(true);
    setCoreJourneyMessage(t("message.coreJourneyRunning"));
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
          workspaceAssignmentId: workspaceAssignment.id
        },
        adminKey
      );

      const toolList = await callMcpRpc(target.id, mcpToolsListPayload(), callerKey.key, nextConfig.runId, adminKey);
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
        adminKey
      );
      if (deniedCall.status !== 403) {
        throw new Error(tx(t, "message.coreJourneyDeniedUnexpected", { status: deniedCall.status }));
      }
      const allowedCall = await callMcpRpc(
        target.id,
        mcpToolCallPayload(nextConfig.allowedTool),
        callerKey.key,
        nextConfig.runId,
        adminKey
      );
      if (!allowedCall.ok) throw new Error(tx(t, "message.coreJourneyRpcUnexpected", { status: allowedCall.status }));

      const nextScope = {
        tenantId: nextConfig.childTenantId,
        workspaceId: nextConfig.workspaceId
      };
      const nextTraceFilters = {
        callerAgentId: caller.id,
        decision: "" as TraceDecision | "",
        runId: nextConfig.runId,
        targetAgentId: target.id
      };
      const nextAccessFilters = {
        callerInstanceId: caller.id,
        capabilityId: "",
        targetId: target.id,
        traceLimit: "10",
        workspaceId: nextConfig.workspaceId
      };
      const [nextData, nextProfile] = await Promise.all([
        loadConsoleData(adminKey, nextTraceFilters),
        loadTenantAccessProfile(nextConfig.childTenantId, adminKey, {
          ...nextAccessFilters,
          traceLimit: 10
        })
      ]);
      setScope(nextScope);
      setTraceFilters(nextTraceFilters);
      setAccessFilters(nextAccessFilters);
      setData(nextData);
      setAccessProfile(nextProfile);
      setLastRefresh(new Date());
      setCoreJourneyResult({
        allowedStatus: allowedCall.status,
        callerId: caller.id,
        deniedStatus: deniedCall.status,
        targetId: target.id,
        toolListStatus: toolList.status
      });
      setCoreJourneyMessage(t("message.coreJourneyComplete"));
    } catch (error) {
      setCoreJourneyMessage(error instanceof Error ? error.message : "Core journey failed");
    } finally {
      setCoreJourneyRunning(false);
    }
  }

  async function runAiAdminApprovalJourney() {
    const nextConfig = createAiAdminApprovalJourneyConfig();
    setAiAdminApprovalJourneyConfig(nextConfig);
    setAiAdminApprovalJourneyResult(null);
    setAiAdminApprovalAuditEvent(null);
    setAiAdminApplicationImpact(null);
    setAiAdminApplicationImpactMessage("");
    setAiAdminApprovalJourneyRunning(true);
    setAiAdminApprovalJourneyMessage(t("message.aiAdminApprovalJourneyRunning"));
    setAiAdminMessage("");
    try {
      const readinessResult = await refreshAiAdminApprovalReadiness(nextConfig);
      if (!aiAdminApprovalReadinessCanRun(readinessResult.state)) {
        const detail = readinessResult.detail;
        throw new Error(tx(t, "message.aiAdminApprovalJourneyPreflightFailed", { detail: detail || "unknown" }));
      }

      await createTenant(
        {
          id: nextConfig.rootTenantId,
          name: "AI Admin Approval Root",
          status: "active"
        },
        adminKey
      );
      await createTenant(
        {
          id: nextConfig.childTenantId,
          name: "AI Admin Approval Team",
          parentTenantId: nextConfig.rootTenantId,
          status: "active"
        },
        adminKey
      );
      await createTenant(
        {
          id: nextConfig.grandchildTenantId,
          name: "AI Admin Approval Project",
          parentTenantId: nextConfig.childTenantId,
          status: "active"
        },
        adminKey
      );

      const caller = await createAgent(
        {
          channelType: "local",
          description: "AI Admin approval journey browser caller",
          name: "AI Admin Approval Caller",
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
          name: "ai admin approval journey key"
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
          description: "AI Admin approval journey MCP target",
          name: "AI Admin Approval MCP Target",
          status: "active",
          tenantId: nextConfig.rootTenantId,
          workspaceId: nextConfig.workspaceId
        },
        adminKey
      );

      const discovered = await refreshTargetCapabilities(target.id, adminKey);
      const readCapability = discovered.find((capability) => capability.key === nextConfig.readTool);
      const writeCapability = discovered.find((capability) => capability.key === nextConfig.writeTool);
      const deniedCapability = discovered.find((capability) => capability.key === nextConfig.deniedTool);
      if (!readCapability || !writeCapability || !deniedCapability) {
        throw new Error(
          tx(t, "message.aiAdminApprovalJourneyMissingTools", {
            denied: nextConfig.deniedTool,
            read: nextConfig.readTool,
            write: nextConfig.writeTool
          })
        );
      }

      const nextForm: PermissionPackageDraftInput = {
        callerInstanceId: caller.id,
        region: nextConfig.region,
        requestText: nextConfig.requestText,
        subjectSelector: nextConfig.subjectSelector,
        targetId: target.id,
        templateId: nextConfig.templateId,
        tenantId: nextConfig.childTenantId,
        workspaceId: nextConfig.workspaceId
      };
      setAiAdminForm(nextForm);
      setAiAdminApplication(null);
      setAiAdminApplicationImpact(null);
      setAiAdminApplicationImpactMessage("");
      setAiAdminApprovalRequests([]);
      const draft = await createPermissionPackageDraftFromApi(nextForm, adminKey);
      setAiAdminServerDraft(draft);
      if (!draft.readiness.canApply) {
        throw new Error(tx(t, "message.permissionPackageNotReady", { detail: permissionReadinessMessages(draft.readiness, t).join(", ") }));
      }
      if (draft.policyGate.canApplyDirectly) {
        throw new Error(t("message.aiAdminApprovalJourneyApprovalGateMissing"));
      }

      const pendingApproval = await createPermissionPackageApprovalRequest(nextForm, adminKey);
      const approvedApproval = await approvePermissionPackageApprovalRequest(
        pendingApproval.id,
        {
          comment: "Approved from AI Admin approval journey",
          reviewer: "AI Admin"
        },
        adminKey
      );
      setAiAdminApprovalRequests([approvedApproval]);

      const applied = await applyPermissionPackage(
        {
          ...nextForm,
          approvalRequestId: approvedApproval.id
        },
        adminKey
      );
      const application = applied.application ?? null;
      if (!application) {
        throw new Error(t("message.aiAdminApprovalJourneyMissingApplication"));
      }
      setAiAdminApplication(application);
      setAiAdminApplicationImpact(null);
      setAiAdminApplicationImpactMessage("");

      const toolList = await callMcpRpc(
        target.id,
        mcpToolsListPayload(),
        callerKey.key,
        nextConfig.runId,
        adminKey,
        nextConfig.subjectId
      );
      if (!toolList.ok) throw new Error(tx(t, "message.aiAdminApprovalJourneyRpcUnexpected", { status: toolList.status }));
      const listedTools = toolNamesFromPayload(toolList.payload);
      if (
        !listedTools.includes(nextConfig.readTool) ||
        !listedTools.includes(nextConfig.writeTool) ||
        listedTools.includes(nextConfig.deniedTool)
      ) {
        throw new Error(t("message.aiAdminApprovalJourneyToolsListInvalid"));
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
        throw new Error(tx(t, "message.aiAdminApprovalJourneyDeniedUnexpected", { status: deniedCall.status }));
      }
      const allowedCall = await callMcpRpc(
        target.id,
        mcpToolCallPayload(nextConfig.writeTool),
        callerKey.key,
        nextConfig.runId,
        adminKey,
        nextConfig.subjectId
      );
      if (!allowedCall.ok) throw new Error(tx(t, "message.aiAdminApprovalJourneyRpcUnexpected", { status: allowedCall.status }));

      const nextScope = {
        tenantId: nextConfig.childTenantId,
        workspaceId: nextConfig.workspaceId
      };
      const nextTraceFilters = {
        callerAgentId: caller.id,
        decision: "" as TraceDecision | "",
        runId: nextConfig.runId,
        targetAgentId: target.id
      };
      const nextAccessFilters = {
        callerInstanceId: caller.id,
        capabilityId: "",
        targetId: target.id,
        traceLimit: "10",
        workspaceId: nextConfig.workspaceId
      };
      const [nextData, nextProfile, auditRows] = await Promise.all([
        loadConsoleData(adminKey, nextTraceFilters),
        loadTenantAccessProfile(nextConfig.childTenantId, adminKey, {
          ...nextAccessFilters,
          traceLimit: 10
        }),
        fetchAuditEvents(
          {
            action: "permission_package.applied",
            resourceId: application.id,
            tenantId: nextConfig.childTenantId,
            workspaceId: nextConfig.workspaceId
          },
          adminKey
        )
      ]);
      const appliedAudit = auditRows.find((event) => event.metadata?.approvalRequestId === approvedApproval.id) ?? auditRows[0] ?? null;
      setScope(nextScope);
      setTraceFilters(nextTraceFilters);
      setAccessFilters(nextAccessFilters);
      setData(appliedAudit ? { ...nextData, auditEvents: [appliedAudit, ...nextData.auditEvents.filter((event) => event.id !== appliedAudit.id)] } : nextData);
      setAccessProfile(nextProfile);
      setAiAdminApprovalAuditEvent(appliedAudit);
      setLastRefresh(new Date());
      setAiAdminApprovalJourneyResult({
        allowedStatus: allowedCall.status,
        applicationId: application.id,
        approvalRequestId: approvedApproval.id,
        callerId: caller.id,
        deniedStatus: deniedCall.status,
        targetId: target.id,
        toolListStatus: toolList.status
      });
      setAiAdminApprovalJourneyMessage(t("message.aiAdminApprovalJourneyComplete"));
      setAiAdminMessage(tx(t, "message.permissionPackageApplied", { count: applied.tenantEntitlements.length }));
    } catch (error) {
      setAiAdminApprovalJourneyMessage(error instanceof Error ? error.message : "AI Admin approval journey failed");
    } finally {
      setAiAdminApprovalJourneyRunning(false);
    }
  }

  async function submitAgent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAgentMessage("");
    try {
      const channelConfig: JsonObject = {};
      const endpoint = agentForm.endpoint.trim();
      if (endpoint) channelConfig.endpoint = endpoint;
      const retry = parseRetryFields({
        backoffMsText: agentForm.retryBackoffMs,
        maxAttemptsText: agentForm.retryMaxAttempts
      });
      if (!retry.ok) {
        setAgentMessage(retry.message);
        return;
      }
      if (retry.requested) {
        channelConfig.retry = {
          backoffMs: retry.backoffMs,
          maxAttempts: retry.maxAttempts
        };
      }
      const credentialHeader = agentForm.credentialHeader.trim();
      const credentialName = agentForm.credentialName.trim();
      const credentialValue = agentForm.credentialValue;
      const hasCredentialInput = Boolean(credentialHeader || credentialName || credentialValue.trim());
      let credentials: Record<string, string> | undefined;
      if (hasCredentialInput) {
        if (!credentialHeader || !credentialName || !credentialValue.trim()) {
          setAgentMessage(t("message.validationCredentialGroup"));
          return;
        }
        channelConfig.credentialHeaders = { [credentialHeader]: credentialName };
        credentials = { [credentialName]: credentialValue };
      }
      const requestScope = normalizedScope(scope);
      await createAgent(
        {
          channelConfig: Object.keys(channelConfig).length > 0 ? channelConfig : undefined,
          channelType: agentForm.channelType.trim() || "local",
          credentials,
          description: agentForm.description.trim() || undefined,
          name: agentForm.name.trim(),
          status: agentForm.status,
          tenantId: requestScope.tenantId,
          workspaceId: requestScope.workspaceId
        },
        adminKey
      );
      setAgentForm(defaultAgentForm);
      setAgentMessage(t("message.agentCreated"));
      await refresh();
    } catch (error) {
      setAgentMessage(error instanceof Error ? error.message : "Unable to create agent");
    }
  }

  async function handleAgentStatusChange(agent: Agent, status: AgentStatus) {
    setAgentMessage("");
    setCleanupActionId(agent.id);
    try {
      if (status === "disabled") {
        await disableAgent(agent.id, adminKey);
      } else {
        await updateAgent(agent.id, { status }, adminKey);
      }
      setAgentMessage(tx(t, "message.statusChanged", { name: agent.name, status: agentStatusLabel(status, t) }));
      await refresh();
    } catch (error) {
      setAgentMessage(error instanceof Error ? error.message : "Unable to update agent status");
    } finally {
      setCleanupActionId("");
    }
  }

  async function handleDisablePolicy(policy: RoutePolicy) {
    setPolicyMessage("");
    setCleanupActionId(policy.id);
    try {
      await disableRoutePolicy(policy.id, adminKey);
      setPolicyMessage(t("message.policyDisabled"));
      await refresh();
    } catch (error) {
      setPolicyMessage(error instanceof Error ? error.message : "Unable to disable route policy");
    } finally {
      setCleanupActionId("");
    }
  }

  async function submitKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setKeyMessage("");
    setCreatedKey(null);
    try {
      const ttl = Number(keyForm.expiresInSeconds);
      if (!Number.isInteger(ttl) || ttl < 1 || ttl > 3600) {
        setKeyMessage(t("message.validationTtl"));
        return;
      }
      const next = await createAgentKey(
        {
          agentId: keyForm.agentId,
          expiresInSeconds: ttl,
          name: keyForm.name.trim() || undefined
        },
        adminKey
      );
      setCreatedKey(next);
      setKeyMessage(t("message.keyCreated"));
      setKeyForm({ ...defaultKeyForm, agentId: keyForm.agentId });
    } catch (error) {
      setKeyMessage(error instanceof Error ? error.message : "Unable to create key");
    }
  }

  async function submitCredentialRotation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setRotateMessage("");
    try {
      const credentialName = rotateForm.credentialName.trim();
      if (!rotateForm.agentId) {
        setRotateMessage(t("message.validationRotateAgent"));
        return;
      }
      if (!credentialName || !rotateForm.credentialValue.trim()) {
        setRotateMessage(t("message.validationCredentialRequired"));
        return;
      }
      await rotateAgentCredentials(
        rotateForm.agentId,
        { credentials: { [credentialName]: rotateForm.credentialValue } },
        adminKey
      );
      setRotateForm({ ...defaultRotateForm, agentId: rotateForm.agentId, credentialName });
      setRotateMessage(t("message.credentialRotated"));
      await refresh();
    } catch (error) {
      setRotateMessage(error instanceof Error ? error.message : "Unable to rotate credential");
    }
  }

  async function submitRoutePolicy(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPolicyMessage("");
    try {
      const priority = Number(policyForm.priority);
      if (!Number.isInteger(priority) || priority < 0) {
        setPolicyMessage(t("message.validationPriority"));
        return;
      }
      const retry = parseRetryFields({
        backoffMsText: policyForm.retryBackoffMs,
        maxAttemptsText: policyForm.retryMaxAttempts
      });
      if (!retry.ok) {
        setPolicyMessage(retry.message);
        return;
      }
      await createRoutePolicy(
        {
          callerAgentId: policyForm.callerAgentId,
          effect: policyForm.effect as "allow" | "deny",
          name: policyForm.name.trim() || undefined,
          priority,
          retry: retry.requested
            ? { backoffMs: retry.backoffMs, maxAttempts: retry.maxAttempts, statusCodes: [502, 503, 504] }
            : undefined,
          routeKey: policyForm.routeKey.trim() || undefined,
          routeType: policyForm.routeType.trim(),
          targetAgentId: policyForm.targetAgentId
        },
        adminKey
      );
      setPolicyMessage(t("message.policyCreated"));
      setPolicyForm({ ...defaultPolicyForm, callerAgentId: policyForm.callerAgentId });
      await refresh();
    } catch (error) {
      setPolicyMessage(error instanceof Error ? error.message : "Unable to create route policy");
    }
  }

  async function handleRefreshTargetCapabilities() {
    const targetId = capabilityForm.targetId.trim();
    if (!targetId) {
      setCapabilityMessage(t("message.validationMcpTargetRequired"));
      return;
    }
    setCapabilityMessage("");
    setCapabilityActionId(`refresh:${targetId}`);
    try {
      const refreshed = await refreshTargetCapabilities(targetId, adminKey);
      setData((current) =>
        current
          ? {
              ...current,
              capabilities: mergeCapabilitiesForTarget(current.capabilities, refreshed, targetId),
              capabilitiesLoadedFromApi: true
            }
          : current
      );
      setCapabilityMessage(tx(t, "message.refreshedCapabilities", { count: refreshed.length }));
    } catch (error) {
      if (shouldUseLocalCapabilityFallback(error, data)) {
        setCapabilityMessage(t("message.capabilityFallback"));
        return;
      }
      setCapabilityMessage(error instanceof Error ? error.message : "Unable to refresh capabilities");
    } finally {
      setCapabilityActionId("");
    }
  }

  async function handleApproveCapability(capability: Capability) {
    setCapabilityMessage("");
    setCapabilityActionId(capability.id);
    try {
      const updated = await updateCapability(capability.id, { discoveryStatus: "approved" }, adminKey);
      setData((current) =>
        current
          ? {
              ...current,
              capabilities: current.capabilities.map((item) => (item.id === updated.id ? updated : item)),
              capabilitiesLoadedFromApi: true
          }
          : current
      );
      setCapabilityMessage(tx(t, "message.capabilityApproved", { name: capability.key }));
    } catch (error) {
      if (shouldUseLocalCapabilityFallback(error, data)) {
        setData((current) =>
          current
            ? {
                ...current,
                capabilities: current.capabilities.map((item) =>
                  item.id === capability.id
                    ? { ...item, discoveryStatus: "approved", updatedAt: new Date().toISOString() }
                    : item
                )
              }
            : current
        );
        setCapabilityMessage(tx(t, "message.capabilityApprovedFallback", { name: capability.key }));
        return;
      }
      setCapabilityMessage(error instanceof Error ? error.message : "Unable to approve capability");
    } finally {
      setCapabilityActionId("");
    }
  }

  async function submitCapabilityGrantChain(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCapabilityMessage("");
    const capability = data?.capabilities.find((item) => item.id === capabilityForm.capabilityId);
    const tenantId = capabilityForm.tenantId.trim();
    const workspaceId = capabilityForm.workspaceId.trim();
    const callerInstanceId = capabilityForm.callerInstanceId.trim();
    if (!capability) {
      setCapabilityMessage(t("message.validationCapabilityRequired"));
      return;
    }
    if (!tenantId || !workspaceId || !callerInstanceId) {
      setCapabilityMessage(t("message.validationTenantWorkspaceCaller"));
      return;
    }
    const dataScopes = capability.dataScopes ?? [];
    setCapabilityActionId(`grant:${capability.id}`);
    try {
      const entitlement = await createTenantEntitlement(
        {
          capabilityId: capability.id,
          dataScopes,
          effect: "allow",
          priority: 50,
          status: "enabled",
          targetId: capability.targetId,
          tenantId
        },
        adminKey
      );
      const workspaceAssignment = await createWorkspaceAssignment(
        {
          dataScopes,
          effect: "allow",
          status: "enabled",
          tenantEntitlementId: entitlement.id,
          workspaceId
        },
        adminKey
      );
      await createInstanceAssignment(
        {
          callerInstanceId,
          dataScopes,
          effect: "allow",
          status: "enabled",
          subjectSelector: capabilityForm.subjectSelector.trim() || undefined,
          workspaceAssignmentId: workspaceAssignment.id
        },
        adminKey
      );
      setCapabilityMessage(t("message.grantChainCreated"));
      await refresh();
    } catch (error) {
      if (shouldUseLocalCapabilityFallback(error, data) && data) {
        setData((current) =>
          current ? appendLocalCapabilityGrantChain(current, capability, capabilityForm, dataScopes) : current
        );
        setCapabilityMessage(t("message.grantChainCreatedFallback"));
        return;
      }
      setCapabilityMessage(error instanceof Error ? error.message : "Unable to create grant chain");
    } finally {
      setCapabilityActionId("");
    }
  }

  async function loadAiAdminApprovalRequestsForDraft(draft: PermissionPackageDraft, signal?: AbortSignal) {
    if (!data?.loadedFromApi || !draft.readiness.canApply || draft.policyGate.canApplyDirectly) {
      setAiAdminApprovalRequests([]);
      return [];
    }
    const rows = await fetchPermissionPackageApprovalRequests(
      {
        callerInstanceId: draft.input.callerInstanceId,
        limit: 8,
        targetId: draft.input.targetId,
        templateId: draft.template.id,
        tenantId: draft.input.tenantId,
        workspaceId: draft.input.workspaceId
      },
      adminKey,
      signal
    );
    setAiAdminApprovalRequests((current) => mergePermissionPackageApprovalRequests(rows, current));
    return rows;
  }

  function upsertAiAdminApprovalRequest(request: PermissionPackageApprovalRequest) {
    setAiAdminApprovalRequests((current) => [request, ...current.filter((item) => item.id !== request.id)]);
  }

  async function refreshAiAdminReviewerQueue(signal?: AbortSignal) {
    if (!data?.loadedFromApi) {
      setAiAdminReviewerQueueMessage(t("message.reviewerQueueRequiresLiveApi"));
      return;
    }
    const reviewer = aiAdminApprovalReviewer.trim();
    if (!reviewer) {
      setAiAdminReviewerQueueMessage(t("message.reviewerQueueReviewerRequired"));
      return;
    }
    setAiAdminReviewerQueueLoading(true);
    setAiAdminReviewerQueueMessage("");
    try {
      const rows = await fetchPermissionPackageApprovalRequests(
        {
          callerInstanceId: aiAdminDraft.input.callerInstanceId,
          limit: 10,
          reviewer,
          status: "pending",
          targetId: aiAdminDraft.input.targetId,
          templateId: aiAdminDraft.template.id,
          tenantId: aiAdminDraft.input.tenantId,
          workspaceId: aiAdminDraft.input.workspaceId
        },
        adminKey,
        signal
      );
      setAiAdminApprovalRequests((current) => mergePermissionPackageApprovalRequests(rows, current));
      setAiAdminSelectedApprovalRequestId((current) => rows.some((request) => request.id === current) ? current : rows[0]?.id ?? "");
      setAiAdminReviewerQueueMessage(tx(t, "message.reviewerQueueLoaded", { count: rows.length }));
    } catch (error) {
      if (!isAbortError(error)) {
        setAiAdminReviewerQueueMessage(error instanceof Error ? error.message : "Unable to load reviewer queue");
      }
    } finally {
      setAiAdminReviewerQueueLoading(false);
    }
  }

  async function createAiAdminApprovalRequest() {
    if (!data?.loadedFromApi) {
      setAiAdminMessage(t("message.permissionApprovalRequiresLiveApi"));
      return;
    }
    setAiAdminApprovalAction("create");
    setAiAdminMessage("");
    try {
      const request = await createPermissionPackageApprovalRequest(aiAdminForm, adminKey);
      upsertAiAdminApprovalRequest(request);
      setAiAdminMessage(tx(t, "message.permissionApprovalCreated", { id: request.id }));
    } catch (error) {
      setAiAdminMessage(error instanceof Error ? error.message : "Unable to create approval request");
    } finally {
      setAiAdminApprovalAction("");
    }
  }

  async function approveAiAdminApprovalRequest(requestId?: string) {
    const targetRequest = requestId
      ? aiAdminApprovalRequests.find((request) => request.id === requestId)
      : aiAdminApprovalRequest;
    if (!targetRequest) return;
    setAiAdminApprovalAction("approve");
    setAiAdminMessage("");
    try {
      const reviewer = aiAdminApprovalReviewer.trim();
      const request = await approvePermissionPackageApprovalRequest(
        targetRequest.id,
        {
          comment: requestId ? "Approved from AI Admin reviewer queue" : "Approved from AI Admin",
          ...(reviewer ? { reviewer } : {})
        },
        adminKey
      );
      upsertAiAdminApprovalRequest(request);
      setAiAdminMessage(tx(t, "message.permissionApprovalApproved", { id: request.id }));
    } catch (error) {
      setAiAdminMessage(error instanceof Error ? error.message : "Unable to approve request");
    } finally {
      setAiAdminApprovalAction("");
    }
  }

  async function rejectAiAdminApprovalRequest(requestId?: string) {
    const targetRequest = requestId
      ? aiAdminApprovalRequests.find((request) => request.id === requestId)
      : aiAdminApprovalRequest;
    if (!targetRequest) return;
    setAiAdminApprovalAction("reject");
    setAiAdminMessage("");
    try {
      const reviewer = aiAdminApprovalReviewer.trim();
      const request = await rejectPermissionPackageApprovalRequest(
        targetRequest.id,
        {
          comment: requestId ? "Rejected from AI Admin reviewer queue" : "Rejected from AI Admin",
          ...(reviewer ? { reviewer } : {})
        },
        adminKey
      );
      upsertAiAdminApprovalRequest(request);
      setAiAdminMessage(tx(t, "message.permissionApprovalRejected", { id: request.id }));
    } catch (error) {
      setAiAdminMessage(error instanceof Error ? error.message : "Unable to reject request");
    } finally {
      setAiAdminApprovalAction("");
    }
  }

  async function applyAiAdminPermissionPackage() {
    setAiAdminMessage("");
    setAiAdminApplicationImpact(null);
    setAiAdminApplicationImpactMessage("");
    if (!aiAdminDraft.readiness.canApply) {
      const detail = permissionReadinessMessages(aiAdminDraft.readiness, t).join(", ");
      setAiAdminMessage(tx(t, "message.permissionPackageNotReady", { detail: detail || "not ready" }));
      return;
    }
    if (!aiAdminDraft.policyGate.canApplyDirectly) {
      if (!data?.loadedFromApi) {
        setAiAdminMessage(t("message.permissionApprovalRequiresLiveApi"));
        return;
      }
      if (!aiAdminApprovalRequest) {
        const detail = permissionPolicyGateMessages(aiAdminDraft.policyGate, t).join(", ");
        setAiAdminMessage(tx(t, "message.permissionPackageApprovalRequired", { detail: detail || t("status.approvalRequired") }));
        return;
      }
      if (aiAdminApprovalRequest.status === "pending") {
        setAiAdminMessage(t("message.permissionApprovalPending"));
        return;
      }
      if (aiAdminApprovalRequest.status === "rejected") {
        setAiAdminMessage(t("message.permissionApprovalRejectedApply"));
        return;
      }
    }
    setAiAdminApplying(true);
    try {
      let appliedCount = aiAdminDraft.allowedCapabilities.length;
      let application: PermissionPackageApplication | null = null;
      try {
        const applied = await applyPermissionPackage(
          aiAdminApprovalRequest?.status === "approved"
            ? { ...aiAdminForm, approvalRequestId: aiAdminApprovalRequest.id }
            : aiAdminForm,
          adminKey
        );
        appliedCount = applied.tenantEntitlements.length;
        application = applied.application ?? null;
      } catch (error) {
        if (!isApiCompatibilityFallbackError(error) || !aiAdminDraft.policyGate.canApplyDirectly) {
          throw error;
        }
        appliedCount = await applyAiAdminPermissionPackageWithManagementChain();
      }

      const nextScope = {
        tenantId: aiAdminForm.tenantId,
        workspaceId: aiAdminForm.workspaceId
      };
      const nextAccessFilters = {
        callerInstanceId: aiAdminForm.callerInstanceId,
        capabilityId: "",
        targetId: aiAdminForm.targetId,
        traceLimit: "10",
        workspaceId: aiAdminForm.workspaceId
      };
      const [nextData, nextProfile] = await Promise.all([
        loadConsoleData(adminKey, traceFilters),
        loadTenantAccessProfile(aiAdminForm.tenantId, adminKey, {
          ...nextAccessFilters,
          traceLimit: 10
        })
      ]);
      setScope(nextScope);
      setAccessFilters(nextAccessFilters);
      setData(nextData);
      setAccessProfile(nextProfile);
      setLastRefresh(new Date());
      setAiAdminApplication(application);
      setAiAdminApplicationImpact(null);
      setAiAdminApplicationImpactMessage("");
      setAiAdminMessage(tx(t, "message.permissionPackageApplied", { count: appliedCount }));
      await loadAiAdminApprovalRequestsForDraft(aiAdminDraft).catch(() => undefined);
    } catch (error) {
      setAiAdminMessage(error instanceof Error ? error.message : "Unable to apply permission package");
    } finally {
      setAiAdminApplying(false);
    }
  }

  async function applyAiAdminPermissionPackageWithManagementChain() {
    for (const capability of aiAdminDraft.allowedCapabilities) {
      const approvedCapability = await updateCapability(
        capability.id,
        {
          dataScopes: aiAdminDraft.dataScopes,
          discoveryStatus: "approved"
        },
        adminKey
      );
      const entitlement = await createTenantEntitlement(
        {
          capabilityId: approvedCapability.id,
          dataScopes: aiAdminDraft.dataScopes,
          effect: "allow",
          priority: 40,
          status: "enabled",
          targetId: approvedCapability.targetId,
          tenantId: aiAdminForm.tenantId
        },
        adminKey
      );
      const workspaceAssignment = await createWorkspaceAssignment(
        {
          dataScopes: aiAdminDraft.dataScopes,
          effect: "allow",
          status: "enabled",
          tenantEntitlementId: entitlement.id,
          workspaceId: aiAdminForm.workspaceId
        },
        adminKey
      );
      await createInstanceAssignment(
        {
          callerInstanceId: aiAdminForm.callerInstanceId,
          dataScopes: aiAdminDraft.dataScopes,
          effect: "allow",
          status: "enabled",
          subjectSelector: aiAdminForm.subjectSelector?.trim() || undefined,
          workspaceAssignmentId: workspaceAssignment.id
        },
        adminKey
      );
    }
    return aiAdminDraft.allowedCapabilities.length;
  }

  const agents = data?.agents ?? [];
  const traces = data?.traces ?? [];
  const auditEvents = data?.auditEvents ?? [];
  const channels = data?.channels ?? [];
  const policies = data?.routePolicies ?? [];
  const capabilities = data?.capabilities ?? [];
  const tenantEntitlements = data?.tenantEntitlements ?? [];
  const workspaceAssignments = data?.workspaceAssignments ?? [];
  const instanceAssignments = data?.instanceAssignments ?? [];
  const evidenceRuns = data?.evidenceRuns ?? [];
  const metrics = data?.systemMetrics ?? [];
  const localCallers = agents.filter((agent) => agent.status === "active" && agent.channelType === "local");
  const mcpTargets = agents.filter((agent) => agent.channelType === "mcp");
  const localAiAdminDraft = useMemo(
    () => createPermissionPackageDraft(aiAdminForm, { capabilities }),
    [aiAdminForm, capabilities]
  );
  const aiAdminDraft = aiAdminServerDraft ?? localAiAdminDraft;
  const aiAdminApprovalRequest = useMemo(
    () => matchingPermissionPackageApprovalRequest(aiAdminApprovalRequests, aiAdminDraft),
    [aiAdminApprovalRequests, aiAdminDraft]
  );

  useEffect(() => {
    if (activeNav !== "ai-admin" || !data?.loadedFromApi || !aiAdminDraft.readiness.canApply || aiAdminDraft.policyGate.canApplyDirectly) {
      setAiAdminApprovalRequests([]);
      return;
    }
    const controller = new AbortController();
    loadAiAdminApprovalRequestsForDraft(aiAdminDraft, controller.signal).catch((error) => {
      if (!isAbortError(error)) {
        setAiAdminApprovalRequests([]);
      }
    });
    return () => controller.abort();
  }, [
    activeNav,
    adminKey,
    aiAdminDraft.id,
    aiAdminDraft.policyGate.canApplyDirectly,
    aiAdminDraft.readiness.canApply,
    data?.loadedFromApi
  ]);

  const channelLabels = useMemo(() => {
    return channels.reduce<Record<string, string>>((acc, item) => {
      acc[item.key] = item.label;
      return acc;
    }, {});
  }, [channels]);

  const activeAgents = agents.filter((agent) => agent.status === "active").length;
  const activePolicies = policies.filter((policy) => policy.status === "enabled").length;
  const deniedTraces = traces.filter((trace) => trace.decision === "denied").length;
  const allowedTraces = traces.filter((trace) => trace.decision === "allowed").length;
  const pendingCapabilities = capabilities.filter((capability) => capability.discoveryStatus === "pending_review").length;
  const runtimeEvidence = runtimeEvidenceMetric(allowedTraces, deniedTraces);
  const dataSourceLabel = loadError
    ? t("dataSource.apiError")
    : data?.loadedFromApi
      ? t("dataSource.runtime")
      : t("dataSource.fallback");
  const activeView = viewForNav(activeNav);
  const activeNavItem = navItems.find((item) => item.key === activeView.key) ?? navItems[0];
  const activeNavLabel = t(`nav.${activeNavItem.key}`, activeNavItem.label);
  const isCapabilitiesView = activeView.key === "capabilities";
  const isAccessView = activeView.key === "access";
  const accessSummary = accessProfile?.summary;
  const invalidAccessRows = countInvalidAccessProfileRows(accessProfile);
  const pageTitle = t(activeView.titleKey, t("app.title"));
  const coreJourneyEvaluation = useMemo(
    () => evaluateCoreJourney(data, accessProfile, coreJourneyConfig),
    [accessProfile, coreJourneyConfig, data]
  );
  const aiAdminApprovalJourneyEvaluation = useMemo(
    () =>
      evaluateAiAdminApprovalJourney({
        accessProfile,
        application: aiAdminApplication,
        approvalRequest: aiAdminApprovalRequest,
        auditEvent: aiAdminApprovalAuditEvent,
        config: aiAdminApprovalJourneyConfig,
        data,
        result: aiAdminApprovalJourneyResult
      }),
    [
      accessProfile,
      aiAdminApplication,
      aiAdminApprovalAuditEvent,
      aiAdminApprovalJourneyConfig,
      aiAdminApprovalJourneyResult,
      aiAdminApprovalRequest,
      data
    ]
  );
  const tracePanel = (className = "span-7") => (
    <Panel className={className} icon={<FileSearch size={18} />} title={t("panel.auditTraces")} action={<IconOpen title={t("action.open")} />}>
      <TraceFilterBar agents={agents} filters={traceFilters} onChange={setTraceFilters} onRefresh={refresh} t={t} />
      <TraceTable traces={traces} agents={agents} t={t} />
    </Panel>
  );
  const managementAuditPanel = (className = "span-12") => (
    <Panel className={className} icon={<ClipboardCheck size={18} />} title={t("panel.managementAudit")} action={<IconOpen title={t("action.open")} />}>
      <ManagementAuditTable events={auditEvents} t={t} />
    </Panel>
  );
  const routeGovernancePanel = (className = "span-8") => (
    <Panel className={className} icon={<Workflow size={18} />} title={t("panel.routeGovernance")} action={<IconMore title={t("action.more")} />}>
      <PolicyTable
        agents={agents}
        canDisable={Boolean(data?.routePoliciesLoadedFromApi)}
        onDisable={handleDisablePolicy}
        pendingActionId={cleanupActionId}
        policies={policies}
        t={t}
      />
    </Panel>
  );
  const evidenceRunsPanel = (className = "span-4") => (
    <Panel className={className} icon={<Sparkles size={18} />} title={t("panel.evidenceRuns")} action={<IconOpen title={t("action.open")} />}>
      <EvidenceTimeline runs={evidenceRuns} t={t} />
    </Panel>
  );
  const runtimeSignalsPanel = (className = "span-5") => (
    <Panel className={className} icon={<DatabaseZap size={18} />} title={t("panel.runtimeSignals")} action={<IconMore title={t("action.more")} />}>
      <SignalBoard metrics={metrics} t={t} />
    </Panel>
  );
  const agentRegistryPanel = (className = "span-8") => (
    <Panel className={className} icon={<Boxes size={18} />} title={t("panel.agentRegistry")} action={<IconMore title={t("action.more")} />}>
      <AgentTable
        agents={agents}
        channelLabels={channelLabels}
        onStatusChange={handleAgentStatusChange}
        pendingActionId={cleanupActionId}
        t={t}
      />
    </Panel>
  );
  const contractMatrixPanel = (className = "span-4") => (
    <Panel className={className} icon={<Layers3 size={18} />} title={t("panel.contractMatrix")} action={<IconOpen title={t("action.open")} />}>
      <ContractMatrix channels={channels} providers={data?.providers ?? []} t={t} />
    </Panel>
  );
  const capabilityGovernancePanel = (className = "span-12") => (
    <Panel className={className} icon={<DatabaseZap size={18} />} title={t("panel.mcpCapabilities")} action={<IconMore title={t("action.more")} />}>
      <CapabilityGovernanceView
        actionId={capabilityActionId}
        agents={agents}
        capabilities={capabilities}
        form={capabilityForm}
        instanceAssignments={instanceAssignments}
        message={capabilityMessage}
        mcpTargets={mcpTargets}
        onApprove={handleApproveCapability}
        onChange={setCapabilityForm}
        onCreateGrantChain={submitCapabilityGrantChain}
        onRefreshTarget={handleRefreshTargetCapabilities}
        t={t}
        tenantEntitlements={tenantEntitlements}
        workspaceAssignments={workspaceAssignments}
      />
    </Panel>
  );
  const accessProfilePanel = (
    <Panel className="span-12" icon={<LockKeyhole size={18} />} title={t("panel.accessProfile")} action={<IconOpen title={t("action.open")} />}>
      <TenantAccessProfileView
        agents={agents}
        capabilities={capabilities}
        explanation={accessDecisionExplanation}
        explanationLoading={accessDecisionExplainLoading}
        explanationMessage={accessDecisionExplainMessage}
        filters={accessFilters}
        loading={accessLoading}
        message={accessMessage}
        onChange={(filters) => {
          setAccessFilters(filters);
          setAccessDecisionExplanation(null);
          setAccessDecisionExplainMessage("");
        }}
        onExplainAccessDecision={() => void explainAccessDecisionFromProfile()}
        onRefresh={() => void refreshAccessProfile()}
        onTenantChange={(tenantId) => {
          setScope((current) => ({ ...current, tenantId }));
          setAccessProfile(null);
          setAccessDecisionExplanation(null);
          setAccessDecisionExplainMessage("");
        }}
        profile={accessProfile}
        scope={scope}
        t={t}
      />
    </Panel>
  );
  const aiAdminPanel = (
    <Panel className="span-12" icon={<Sparkles size={18} />} title={t("panel.aiAdminPermissionWorkbench")}>
      <AiAdminPermissionWorkbench
        agents={agents}
        approvalAction={aiAdminApprovalAction}
        approvalAuditEvent={aiAdminApprovalAuditEvent}
        approvalJourneyConfig={aiAdminApprovalJourneyConfig}
        approvalJourneyEvaluation={aiAdminApprovalJourneyEvaluation}
        approvalJourneyMessage={aiAdminApprovalJourneyMessage}
        approvalJourneyResult={aiAdminApprovalJourneyResult}
        approvalJourneyRunning={aiAdminApprovalJourneyRunning}
        approvalReadiness={aiAdminApprovalReadiness}
        approvalReadinessChecking={aiAdminApprovalReadinessChecking}
        approvalReadinessMessage={aiAdminApprovalReadinessMessage}
        approvalRequest={aiAdminApprovalRequest}
        approvalRequests={aiAdminApprovalRequests}
        approvalReviewer={aiAdminApprovalReviewer}
        applicationImpact={aiAdminApplicationImpact}
        applicationImpactLoading={aiAdminApplicationImpactLoading}
        applicationImpactMessage={aiAdminApplicationImpactMessage}
        accessDecisionExplanation={aiAdminAccessDecisionExplanation}
        accessDecisionExplanationLoading={aiAdminAccessDecisionExplainLoading}
        accessDecisionExplanationMessage={aiAdminAccessDecisionExplainMessage}
        applying={aiAdminApplying}
        draft={aiAdminDraft}
        form={aiAdminForm}
        application={aiAdminApplication}
        message={aiAdminMessage}
        mcpTargets={mcpTargets}
        onApply={() => void applyAiAdminPermissionPackage()}
        onApprovalReviewerChange={setAiAdminApprovalReviewer}
        onApproveApprovalRequest={(requestId) => void approveAiAdminApprovalRequest(requestId)}
        onChange={(nextForm) => {
          setAiAdminForm(nextForm);
          setAiAdminApplication(null);
          setAiAdminApplicationImpact(null);
          setAiAdminApplicationImpactMessage("");
          setAiAdminApprovalAuditEvent(null);
          setAiAdminApprovalJourneyResult(null);
          setAiAdminApprovalRequests([]);
          setAiAdminSelectedApprovalRequestId("");
          setAiAdminAccessDecisionExplanation(null);
          setAiAdminAccessDecisionExplainMessage("");
        }}
        onCreateApprovalRequest={() => void createAiAdminApprovalRequest()}
        onExplainAccessDecision={() => void explainAiAdminAccessDecision()}
        onRefreshApprovalReadiness={() => void refreshAiAdminApprovalReadiness()}
        onRefreshReviewerQueue={() => void refreshAiAdminReviewerQueue()}
        onRejectApprovalRequest={(requestId) => void rejectAiAdminApprovalRequest(requestId)}
        onReviewApplicationImpact={() => void reviewAiAdminApplicationImpact()}
        onRunApprovalJourney={() => void runAiAdminApprovalJourney()}
        onSelectApprovalRequest={(requestId) => {
          setAiAdminSelectedApprovalRequestId(requestId);
          setAiAdminApplicationImpact(null);
          setAiAdminApplicationImpactMessage("");
        }}
        reviewerQueueLoading={aiAdminReviewerQueueLoading}
        reviewerQueueMessage={aiAdminReviewerQueueMessage}
        selectedApprovalRequestId={aiAdminSelectedApprovalRequestId}
        templates={aiAdminTemplates}
        t={t}
      />
    </Panel>
  );
  const createAgentPanel = (
    <Panel className="span-4" icon={<Boxes size={18} />} title={t("panel.createAgent")}>
      <AgentCreateForm form={agentForm} message={agentMessage} onChange={setAgentForm} onSubmit={submitAgent} t={t} />
    </Panel>
  );
  const createKeyPanel = (
    <Panel className="span-4" icon={<KeyRound size={18} />} title={t("panel.createKey")}>
      <KeyCreateForm
        agents={localCallers}
        createdKey={createdKey}
        form={keyForm}
        message={keyMessage}
        onChange={setKeyForm}
        onSubmit={submitKey}
        t={t}
      />
    </Panel>
  );
  const createPolicyPanel = (
    <Panel className="span-4" icon={<Route size={18} />} title={t("panel.createPolicy")}>
      <PolicyCreateForm agents={agents} form={policyForm} message={policyMessage} onChange={setPolicyForm} onSubmit={submitRoutePolicy} t={t} />
    </Panel>
  );
  const rotateCredentialPanel = (
    <Panel className="span-4" icon={<KeyRound size={18} />} title={t("panel.rotateCredential")}>
      <CredentialRotateForm
        agents={agents}
        form={rotateForm}
        message={rotateMessage}
        onChange={setRotateForm}
        onSubmit={submitCredentialRotation}
        t={t}
      />
    </Panel>
  );
  const coreJourneyPanel = (
    <Panel className="span-12" icon={<Workflow size={18} />} title={t("panel.coreJourney")}>
      <CoreJourneyWorkbench
        config={coreJourneyConfig}
        evaluation={coreJourneyEvaluation}
        form={coreJourneyForm}
        message={coreJourneyMessage}
        onChange={setCoreJourneyForm}
        onOpen={setActiveNav}
        onRefreshPreflight={() => void refreshCoreJourneyPreflight()}
        onReset={() => void resetCoreJourneySession()}
        onRun={() => void runCoreJourney()}
        preflight={coreJourneyPreflight}
        preflightChecking={coreJourneyPreflightChecking}
        preflightMessage={coreJourneyPreflightMessage}
        result={coreJourneyResult}
        running={coreJourneyRunning}
        t={t}
      />
    </Panel>
  );
  const viewContent = (() => {
    switch (activeView.key) {
      case "ai-admin":
        return (
          <section className="content-grid">
            {aiAdminPanel}
            {accessProfilePanel}
          </section>
        );
      case "registry":
        return (
          <section className="content-grid">
            {createAgentPanel}
            {createKeyPanel}
            {rotateCredentialPanel}
            {agentRegistryPanel("span-8")}
            {contractMatrixPanel("span-4")}
          </section>
        );
      case "routes":
        return (
          <section className="content-grid">
            {createPolicyPanel}
            {routeGovernancePanel("span-8")}
            {tracePanel("span-12")}
          </section>
        );
      case "policies":
        return (
          <section className="content-grid">
            {routeGovernancePanel("span-7")}
            {managementAuditPanel("span-5")}
            {capabilityGovernancePanel("span-12")}
          </section>
        );
      case "capabilities":
        return (
          <section className="content-grid">
            {capabilityGovernancePanel("span-12")}
            {tracePanel("span-7")}
            {managementAuditPanel("span-5")}
          </section>
        );
      case "access":
        return <section className="content-grid">{accessProfilePanel}</section>;
      case "traces":
        return (
          <section className="content-grid">
            {tracePanel("span-12")}
            {managementAuditPanel("span-12")}
          </section>
        );
      case "evidence":
        return (
          <section className="content-grid">
            {evidenceRunsPanel("span-5")}
            {managementAuditPanel("span-7")}
            {runtimeSignalsPanel("span-12")}
          </section>
        );
      case "cockpit":
      default:
        return (
          <section className="content-grid">
            {coreJourneyPanel}
            {runtimeSignalsPanel("span-5")}
            {tracePanel("span-7")}
            {evidenceRunsPanel("span-4")}
            {agentRegistryPanel("span-8")}
          </section>
        );
    }
  })();

  return (
    <div className="app-shell">
      <aside className="sidebar" aria-label="Primary navigation">
        <div className="brand">
          <div className="brand-mark">
            <Network size={18} />
          </div>
          <div>
            <strong>AgentHarbor</strong>
            <span>{t("app.controlPlane")}</span>
          </div>
        </div>

        <nav className="nav-list">
          {navItems.map((item) => {
            const Icon = navIconFor(item.key);
            const itemLabel = t(`nav.${item.key}`, item.label);
            return (
              <button
                aria-label={itemLabel}
                className={activeView.key === item.key ? "nav-item active" : "nav-item"}
                key={item.key}
                onClick={() => setActiveNav(item.key)}
                title={itemLabel}
                type="button"
              >
                <Icon size={17} />
                <span>{itemLabel}</span>
              </button>
            );
          })}
        </nav>

        <div className="sidebar-section">
          <div className="section-kicker">{t("section.environments")}</div>
          <div className="env-stack">
            {workspaceTabs.map((tab) => (
              <button
                className={activeWorkspace === tab ? "env-pill selected" : "env-pill"}
                key={tab}
                onClick={() => setActiveWorkspace(tab)}
                type="button"
              >
                <CircleDot size={12} />
                {tab}
              </button>
            ))}
          </div>
        </div>

        <div className="sidebar-health">
          <div>
            <span className="health-dot" />
            <strong>{t("app.controlPlane")}</strong>
          </div>
          <span>{loadError ? t("dataSource.apiError") : data?.loadedFromApi ? t("status.controlLive") : t("status.controlFallback")}</span>
        </div>
      </aside>

      <main className="workspace">
        <header className="topbar">
          <div className="topbar-title">
            <div className="breadcrumb">{t("app.gateway")} / {activeWorkspace} / {activeNavLabel}</div>
            <h1>{pageTitle}</h1>
          </div>
          <div className="topbar-actions">
            <div className="language-toggle" aria-label={t("control.language")} role="group">
              <button
                className={language === "zh-CN" ? "selected" : ""}
                onClick={() => setLanguage("zh-CN")}
                type="button"
              >
                中文
              </button>
              <button
                className={language === "en" ? "selected" : ""}
                onClick={() => setLanguage("en")}
                type="button"
              >
                EN
              </button>
            </div>
            <label className="admin-key-box">
              <LockKeyhole size={16} />
              <input
                onChange={(event) => setAdminKey(event.target.value)}
                placeholder={t("control.adminKey")}
                type="password"
                value={adminKey}
              />
            </label>
            <label className="search-box">
              <Search size={16} />
              <input placeholder={t("control.search")} />
            </label>
            <button className="icon-button" title={t("control.filter")} type="button">
              <Filter size={17} />
            </button>
            <button className="icon-button" onClick={refresh} title={t("action.refresh")} type="button">
              <RefreshCw size={17} />
            </button>
          </div>
        </header>

        <section className="status-strip" aria-label="Runtime status">
          <div>
            <span>{t("status.api")}</span>
            <strong>{data?.apiBase ?? "http://127.0.0.1:9090"}</strong>
          </div>
          <div>
            <span>{t("status.dataSource")}</span>
            <strong>{dataSourceLabel}</strong>
          </div>
          <div>
            <span>{t("status.lastRefresh")}</span>
            <strong>{lastRefresh.toLocaleTimeString("zh-CN", { hour12: false })}</strong>
          </div>
          <div className="scope-control">
            <span>{t("status.scope")}</span>
            <div className="scope-inputs">
              <input
                aria-label={t("form.tenantId")}
                onBlur={() => void refresh()}
                onChange={(event) => setScope((current) => ({ ...current, tenantId: event.target.value }))}
                placeholder="tenantId"
                value={scope.tenantId}
              />
              <input
                aria-label={t("form.workspaceId")}
                onBlur={() => void refresh()}
                onChange={(event) => setScope((current) => ({ ...current, workspaceId: event.target.value }))}
                placeholder="workspaceId"
                value={scope.workspaceId}
              />
            </div>
          </div>
          {loadError ? <div className="strip-error">{loadError}</div> : null}
        </section>

        <section className="metric-grid" aria-label="Gateway metrics">
          <MetricCard
            icon={<ServerCog size={18} />}
            label={isAccessView ? t("metric.scopeTenants") : t("metric.managedAgents")}
            value={String(isAccessView ? accessSummary?.tenantCount ?? 0 : agents.length)}
            detail={isAccessView ? accessProfile?.tenant.name ?? normalizedScope(scope).tenantId : `${activeAgents} ${t("detail.active")}`}
            tone="info"
          />
          <MetricCard
            icon={<KeyRound size={18} />}
            label={isAccessView ? t("metric.grantChains") : t("metric.activePolicies")}
            value={String(isAccessView ? accessSummary?.grantCount ?? 0 : activePolicies)}
            detail={isAccessView ? `${accessSummary?.targetCount ?? 0} ${t("detail.targets")}` : data?.routePoliciesLoadedFromApi ? t("detail.liveRoutePolicies") : t("detail.sampleFallback")}
            tone="success"
          />
          <MetricCard
            icon={<TriangleAlert size={18} />}
            label={isAccessView ? t("metric.invalidRows") : isCapabilitiesView ? t("metric.pendingCaps") : t("metric.deniedTraces")}
            value={String(isAccessView ? invalidAccessRows : isCapabilitiesView ? pendingCapabilities : deniedTraces)}
            detail={isAccessView ? `${accessSummary?.workspaceAssignmentCount ?? 0} ${t("detail.workspaces")}` : isCapabilitiesView ? `${capabilities.length} ${t("detail.discovered")}` : `${allowedTraces} ${t("detail.allowed")}`}
            tone={(isAccessView ? invalidAccessRows : isCapabilitiesView ? pendingCapabilities : deniedTraces) > 0 ? "warning" : "success"}
          />
          <MetricCard
            icon={<ClipboardCheck size={18} />}
            label={isAccessView ? t("metric.traceEvidence") : t("metric.runtimeEvidence")}
            value={isAccessView ? String((accessSummary?.recentAllowedTraceCount ?? 0) + (accessSummary?.recentDeniedTraceCount ?? 0)) : runtimeEvidence.value}
            detail={isAccessView ? `${accessSummary?.recentDeniedTraceCount ?? 0} ${t("detail.denied")}` : runtimeEvidence.value === "0" ? t("detail.noTraces") : `${allowedTraces} ${t("detail.allowed")} / ${deniedTraces} ${t("detail.denied")}`}
            tone={isAccessView ? "neutral" : runtimeEvidence.tone}
          />
        </section>

        {viewContent}
      </main>
    </div>
  );
}

function CoreJourneyWorkbench({
  config,
  evaluation,
  form,
  message,
  onChange,
  onOpen,
  onRefreshPreflight,
  onReset,
  onRun,
  preflight,
  preflightChecking,
  preflightMessage,
  result,
  running,
  t
}: {
  config: CoreJourneyConfig;
  evaluation: CoreJourneyEvaluation;
  form: CoreJourneyForm;
  message: string;
  onChange: (form: CoreJourneyForm) => void;
  onOpen: (key: NavKey) => void;
  onRefreshPreflight: () => void;
  onReset: () => void;
  onRun: () => void;
  preflight: CoreJourneyPreflightState;
  preflightChecking: boolean;
  preflightMessage: string;
  result: CoreJourneyRunResult | null;
  running: boolean;
  t: Translator;
}) {
  const canRun = coreJourneyPreflightCanRun(preflight);
  return (
    <div className="core-journey">
      <div className="core-journey-toolbar">
        <div className="core-journey-score">
          <strong>{evaluation.completeCount}/{evaluation.totalCount}</strong>
          <span>{t("text.coreJourneyCompletion")}</span>
        </div>
        <label>
          {t("form.workspace")}
          <input
            disabled={running}
            value={form.workspaceId}
            onChange={(event) => onChange({ ...form, workspaceId: event.target.value })}
          />
        </label>
        <label>
          {t("form.endpoint")}
          <input
            disabled={running}
            value={form.mcpEndpoint}
            onChange={(event) => onChange({ ...form, mcpEndpoint: event.target.value })}
          />
        </label>
        <label>
          {t("form.allowedTool")}
          <input
            disabled={running}
            value={form.allowedTool}
            onChange={(event) => onChange({ ...form, allowedTool: event.target.value })}
          />
        </label>
        <label>
          {t("form.deniedTool")}
          <input
            disabled={running}
            value={form.deniedTool}
            onChange={(event) => onChange({ ...form, deniedTool: event.target.value })}
          />
        </label>
        <button className="primary-button" disabled={running || preflightChecking || !canRun} onClick={onRun} type="button">
          <Workflow size={14} />
          {running ? t("action.runningJourney") : t("action.runCoreJourney")}
        </button>
      </div>

      <div className="core-journey-preflight">
        <div className="core-journey-preflight-header">
          <div>
            <strong>{t("section.preflight")}</strong>
            {preflightMessage ? <span>{preflightMessage}</span> : null}
          </div>
          <div className="core-journey-preflight-actions">
            <button className="secondary-button" disabled={running || preflightChecking} onClick={onRefreshPreflight} type="button">
              <RefreshCw size={14} />
              {preflightChecking ? t("action.checkingPreflight") : t("action.checkPreflight")}
            </button>
            <button className="secondary-button" disabled={running} onClick={onReset} type="button">
              <RefreshCw size={14} />
              {t("action.resetCoreJourney")}
            </button>
          </div>
        </div>
        <div className="core-journey-preflight-grid">
          {coreJourneyPreflightRows(preflight).map((row) => (
            <article className={`core-journey-preflight-row status-${row.status}`} key={row.key}>
              <Badge tone={preflightTone(row.status)}>{preflightStatusLabel(row.status, t)}</Badge>
              <div>
                <strong>{t(row.titleKey)}</strong>
                <span>{t(row.detailKey)}</span>
              </div>
            </article>
          ))}
        </div>
      </div>

      <div className="core-journey-meta">
        <code>{config.runId}</code>
        <span>{config.childTenantId}</span>
        {result ? (
          <span>
            tools/list {result.toolListStatus} · {form.deniedTool} {result.deniedStatus} · {form.allowedTool} {result.allowedStatus}
          </span>
        ) : null}
        {message ? <strong>{message}</strong> : null}
      </div>

      <div className="core-journey-steps">
        {evaluation.steps.map((step) => (
          <CoreJourneyStepRow key={step.key} step={step} t={t} />
        ))}
      </div>

      <div className="core-journey-actions">
        <button className="secondary-button" onClick={() => onOpen("access")} type="button">
          <LockKeyhole size={14} />
          {t("action.openAccess")}
        </button>
        <button className="secondary-button" onClick={() => onOpen("capabilities")} type="button">
          <DatabaseZap size={14} />
          {t("action.openCapabilities")}
        </button>
        <button className="secondary-button" onClick={() => onOpen("traces")} type="button">
          <FileSearch size={14} />
          {t("action.openTraces")}
        </button>
      </div>
    </div>
  );
}

function AiAdminPermissionWorkbench({
  agents,
  application,
  approvalAction,
  approvalAuditEvent,
  approvalJourneyConfig,
  approvalJourneyEvaluation,
  approvalJourneyMessage,
  approvalJourneyResult,
  approvalJourneyRunning,
  approvalReadiness,
  approvalReadinessChecking,
  approvalReadinessMessage,
  approvalRequest,
  approvalRequests,
  approvalReviewer,
  applicationImpact,
  applicationImpactLoading,
  applicationImpactMessage,
  accessDecisionExplanation,
  accessDecisionExplanationLoading,
  accessDecisionExplanationMessage,
  applying,
  draft,
  form,
  message,
  mcpTargets,
  onApply,
  onApprovalReviewerChange,
  onApproveApprovalRequest,
  onChange,
  onCreateApprovalRequest,
  onExplainAccessDecision,
  onRefreshApprovalReadiness,
  onRefreshReviewerQueue,
  onRejectApprovalRequest,
  onReviewApplicationImpact,
  onRunApprovalJourney,
  onSelectApprovalRequest,
  reviewerQueueLoading,
  reviewerQueueMessage,
  selectedApprovalRequestId,
  templates,
  t
}: {
  agents: Agent[];
  application: PermissionPackageApplication | null;
  approvalAction: "" | "create" | "approve" | "reject";
  approvalAuditEvent: AuditEvent | null;
  approvalJourneyConfig: AiAdminApprovalJourneyConfig;
  approvalJourneyEvaluation: AiAdminApprovalJourneyEvaluation;
  approvalJourneyMessage: string;
  approvalJourneyResult: AiAdminApprovalJourneyResult | null;
  approvalJourneyRunning: boolean;
  approvalReadiness: AiAdminApprovalReadinessState;
  approvalReadinessChecking: boolean;
  approvalReadinessMessage: string;
  approvalRequest: PermissionPackageApprovalRequest | null;
  approvalRequests: PermissionPackageApprovalRequest[];
  approvalReviewer: string;
  applicationImpact: PermissionPackageApplicationImpact | null;
  applicationImpactLoading: boolean;
  applicationImpactMessage: string;
  accessDecisionExplanation: AccessDecisionExplainResult | null;
  accessDecisionExplanationLoading: boolean;
  accessDecisionExplanationMessage: string;
  applying: boolean;
  draft: PermissionPackageDraft;
  form: PermissionPackageDraftInput;
  message: string;
  mcpTargets: Agent[];
  onApply: () => void;
  onApprovalReviewerChange: (reviewer: string) => void;
  onApproveApprovalRequest: (requestId?: string) => void;
  onChange: (form: PermissionPackageDraftInput) => void;
  onCreateApprovalRequest: () => void;
  onExplainAccessDecision: () => void;
  onRefreshApprovalReadiness: () => void;
  onRefreshReviewerQueue: () => void;
  onRejectApprovalRequest: (requestId?: string) => void;
  onReviewApplicationImpact: () => void;
  onRunApprovalJourney: () => void;
  onSelectApprovalRequest: (requestId: string) => void;
  reviewerQueueLoading: boolean;
  reviewerQueueMessage: string;
  selectedApprovalRequestId: string;
  templates: PermissionPackageTemplate[];
  t: Translator;
}) {
  const callers = agents.filter((agent) => agent.status === "active" && agent.channelType === "local");
  const hasApprovedRequest = approvalRequest?.status === "approved";
  const canApply = draft.readiness.canApply && (draft.policyGate.canApplyDirectly || hasApprovedRequest);
  const approvalStatusTone = approvalRequest ? permissionApprovalStatusTone(approvalRequest.status) : "warning";
  const reviewerQueueRequests = approvalRequests.filter((request) => request.status === "pending");
  return (
    <div className="ai-admin-workbench">
      <section className="ai-admin-live-journey">
        <div className="ai-admin-journey-toolbar">
          <div className="core-journey-score">
            <strong>{approvalJourneyEvaluation.completeCount}/{approvalJourneyEvaluation.totalCount}</strong>
            <span>{t("text.aiAdminApprovalJourneyCompletion")}</span>
          </div>
          <div className="ai-admin-journey-meta">
            <div>
              <span>{t("detail.runId")}</span>
              <code>{approvalJourneyConfig.runId}</code>
            </div>
            <div>
              <span>{t("detail.subjectId")}</span>
              <code>{approvalJourneyConfig.subjectId}</code>
            </div>
            <div>
              <span>{t("form.permissionPackage")}</span>
              <strong>{approvalJourneyConfig.templateId}</strong>
            </div>
            <div>
              <span>{t("form.endpoint")}</span>
              <code>{approvalJourneyConfig.mcpEndpoint}</code>
            </div>
          </div>
          <button className="primary-button" disabled={approvalJourneyRunning || applying} onClick={onRunApprovalJourney} type="button">
            <Workflow size={14} />
            {approvalJourneyRunning ? t("action.runningApprovalJourney") : t("action.runApprovalJourney")}
          </button>
        </div>
        <div className="ai-admin-readiness">
          <div className="core-journey-preflight-header">
            <div>
              <strong>{t("section.aiAdminReadiness")}</strong>
              {approvalReadinessMessage ? <span>{approvalReadinessMessage}</span> : null}
            </div>
            <div className="core-journey-preflight-actions">
              <button className="secondary-button" disabled={approvalJourneyRunning || approvalReadinessChecking} onClick={onRefreshApprovalReadiness} type="button">
                <RefreshCw size={14} />
                {approvalReadinessChecking ? t("action.checkingApprovalReadiness") : t("action.checkApprovalReadiness")}
              </button>
            </div>
          </div>
          <div className="ai-admin-readiness-grid">
            {aiAdminApprovalReadinessRows(approvalReadiness).map((row) => (
              <AiAdminApprovalReadinessRow key={row.key} row={row} t={t} />
            ))}
          </div>
        </div>
        <div className="ai-admin-journey-result">
          <strong>{t("section.aiAdminApprovalJourney")}</strong>
          {approvalJourneyResult ? (
            <span>
              tools/list {approvalJourneyResult.toolListStatus} · {approvalJourneyConfig.deniedTool} {approvalJourneyResult.deniedStatus} · {approvalJourneyConfig.writeTool} {approvalJourneyResult.allowedStatus}
            </span>
          ) : (
            <span>{approvalJourneyConfig.childTenantId}</span>
          )}
          {approvalAuditEvent ? <code>{approvalAuditEvent.id}</code> : null}
          {approvalJourneyMessage ? <em>{approvalJourneyMessage}</em> : null}
        </div>
        <div className="ai-admin-journey-steps">
          {approvalJourneyEvaluation.steps.map((step) => (
            <AiAdminApprovalJourneyStepRow key={step.key} step={step} t={t} />
          ))}
        </div>
      </section>

      <div className="ai-admin-request">
        <label className="ai-admin-request-text">
          {t("form.adminRequest")}
          <textarea
            rows={4}
            value={form.requestText}
            onChange={(event) => onChange({ ...form, requestText: event.target.value })}
          />
        </label>
        <div className="ai-admin-fields">
          <label>
            {t("form.permissionPackage")}
            <select value={form.templateId} onChange={(event) => onChange({ ...form, templateId: event.target.value })}>
              {templates.map((template) => (
                <option key={template.id} value={template.id}>
                  {permissionPackageTemplateName(template, t)}
                </option>
              ))}
            </select>
          </label>
          <label>
            {t("form.tenantId")}
            <input value={form.tenantId} onChange={(event) => onChange({ ...form, tenantId: event.target.value })} />
          </label>
          <label>
            {t("form.workspaceId")}
            <input value={form.workspaceId} onChange={(event) => onChange({ ...form, workspaceId: event.target.value })} />
          </label>
          <label>
            {t("form.region")}
            <input value={form.region} onChange={(event) => onChange({ ...form, region: event.target.value })} />
          </label>
          <label>
            {t("form.callerInstance")}
            <select value={form.callerInstanceId} onChange={(event) => onChange({ ...form, callerInstanceId: event.target.value })}>
              <option value="">{t("form.selectCaller")}</option>
              {callers.map((agent) => (
                <option key={agent.id} value={agent.id}>
                  {agent.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            {t("form.target")}
            <select value={form.targetId} onChange={(event) => onChange({ ...form, targetId: event.target.value })}>
              <option value="">{t("form.allMcpTargets")}</option>
              {mcpTargets.map((agent) => (
                <option key={agent.id} value={agent.id}>
                  {agent.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            {t("form.subjectSelector")}
            <input
              placeholder={t("form.subjectSelectorPlaceholder")}
              value={form.subjectSelector ?? ""}
              onChange={(event) => onChange({ ...form, subjectSelector: event.target.value })}
            />
          </label>
        </div>
      </div>

      <div className="permission-package-grid">
        <section className="permission-package-summary">
          <div className="permission-section-title">
            <strong>{t("section.permissionDraft")}</strong>
            <Badge tone={draft.readiness.canApply ? "success" : "warning"}>
              {draft.readiness.canApply ? t("status.readyToApply") : t("status.needsReview")}
            </Badge>
          </div>
          <div className="permission-package-template">
            <strong>{permissionPackageTemplateName(draft.template, t)}</strong>
            <span>{permissionPackageTemplateSummary(draft.template, t)}</span>
          </div>
          {application ? (
            <div className="permission-application-evidence">
              <div className="permission-section-title">
                <strong>{t("section.permissionApplicationEvidence")}</strong>
                <Badge tone="success">v{application.templateVersion}</Badge>
              </div>
              <div className="permission-application-grid">
                <div>
                  <span>{t("detail.applicationId")}</span>
                  <code>{application.id}</code>
                </div>
                <div>
                  <span>{t("detail.draftId")}</span>
                  <code>{application.draftId}</code>
                </div>
                <div>
                  <span>{t("detail.createdObjects")}</span>
                  <strong>
                    {application.tenantEntitlementIds.length + application.workspaceAssignmentIds.length + application.instanceAssignmentIds.length}
                  </strong>
                </div>
                <div>
                  <span>{t("detail.dataScopes")}</span>
                  <strong>{application.dataScopes?.length ?? 0}</strong>
                </div>
              </div>
              <PermissionApplicationImpactPanel
                impact={applicationImpact}
                loading={applicationImpactLoading}
                message={applicationImpactMessage}
                onReview={onReviewApplicationImpact}
                t={t}
              />
            </div>
          ) : null}
          <div className="permission-metrics">
            <div>
              <span>{t("metric.allowedCapabilities")}</span>
              <strong>{draft.allowedCapabilities.length}</strong>
            </div>
            <div>
              <span>{t("metric.blockedCapabilities")}</span>
              <strong>{draft.blockedCapabilities.length}</strong>
            </div>
            <div>
              <span>{t("metric.simulationChecks")}</span>
              <strong>{draft.simulationRows.length}</strong>
            </div>
          </div>
          <CapabilityChipList
            capabilities={draft.allowedCapabilities}
            emptyLabel={t("empty.permissionAllowed.detail")}
            label={t("section.allowedByPackage")}
            tone="success"
            t={t}
          />
          <CapabilityChipList
            capabilities={draft.blockedCapabilities}
            emptyLabel={t("empty.permissionBlocked.detail")}
            label={t("section.blockedByPackage")}
            tone="danger"
            t={t}
          />
          <div className="permission-scope-list">
            <strong>{t("section.dataScope")}</strong>
            {draft.dataScopes.map((scope, index) => (
              <code key={`${scope.dataDomain ?? "scope"}:${index}`}>{summarizeDataScopes([scope])}</code>
            ))}
          </div>
          {draft.readiness.missingFields.length > 0 || draft.readiness.warnings.length > 0 ? (
            <div className="permission-warning">
              <TriangleAlert size={15} />
              <span>{permissionReadinessMessages(draft.readiness, t).join(" · ")}</span>
            </div>
          ) : null}
          <div className={`permission-policy-gate ${draft.policyGate.canApplyDirectly ? "is-direct" : "is-approval"}`}>
            <div className="permission-section-title">
              <strong>{t("section.permissionPolicyGate")}</strong>
              <Badge tone={draft.policyGate.canApplyDirectly ? "success" : "warning"}>
                {draft.policyGate.canApplyDirectly ? t("status.directApplyAllowed") : t("status.approvalRequired")}
              </Badge>
            </div>
            <span>
              {draft.policyGate.canApplyDirectly ? t("text.policyGateDirectDetail") : t("text.policyGateApprovalDetail")}
            </span>
            {draft.policyGate.reasons.length > 0 ? (
              <ul>
                {draft.policyGate.reasons.slice(0, 4).map((reason) => (
                  <li key={reason.id}>{permissionPolicyReasonMessage(reason, t)}</li>
                ))}
              </ul>
            ) : null}
          </div>
          <AccessDecisionExplainPanel
            explanation={accessDecisionExplanation}
            loading={accessDecisionExplanationLoading}
            message={accessDecisionExplanationMessage}
            onExplain={onExplainAccessDecision}
            t={t}
          />
          <div className="permission-reviewer-queue">
            <div className="permission-section-title">
              <strong>{t("section.permissionReviewerQueue")}</strong>
              <span>{t("text.reviewerQueueHelp")}</span>
            </div>
            <div className="permission-reviewer-controls">
              <label>
                {t("form.approvalReviewer")}
                <input
                  value={approvalReviewer}
                  onChange={(event) => onApprovalReviewerChange(event.target.value)}
                />
              </label>
              <button disabled={reviewerQueueLoading || Boolean(approvalAction)} onClick={onRefreshReviewerQueue} type="button">
                <RefreshCw size={14} />
                {reviewerQueueLoading ? t("action.loading") : t("action.refreshReviewerQueue")}
              </button>
            </div>
            {selectedApprovalRequestId ? (
              <span className="permission-reviewer-selected">
                {t("text.reviewerQueueSelected")} <code>{selectedApprovalRequestId}</code>
              </span>
            ) : null}
            {reviewerQueueMessage ? <span className="permission-reviewer-message">{reviewerQueueMessage}</span> : null}
            <div className="permission-reviewer-list">
              {reviewerQueueRequests.length === 0 ? (
                <EmptyRow title={t("section.permissionReviewerQueue")} detail={t("empty.reviewerQueue.detail")} />
              ) : null}
              {reviewerQueueRequests.map((request) => (
                <article
                  className={`permission-reviewer-row ${request.id === selectedApprovalRequestId ? "is-selected" : ""}`}
                  key={request.id}
                >
                  <button onClick={() => onSelectApprovalRequest(request.id)} type="button">
                    <strong>{request.id}</strong>
                    <span>{permissionPackageApprovalRouteLabel(request)}</span>
                    <code>{request.templateId} · {formatDate(request.expiresAt)}</code>
                  </button>
                  <Badge tone={permissionApprovalStatusTone(request.status)}>
                    {permissionApprovalStatusLabel(request.status, t)}
                  </Badge>
                  <div className="permission-reviewer-row-actions">
                    <button disabled={Boolean(approvalAction) || applying} onClick={() => onApproveApprovalRequest(request.id)} type="button">
                      <CheckCircle2 size={13} />
                      {approvalAction === "approve" ? t("action.approving") : t("action.approvePermissionRequest")}
                    </button>
                    <button disabled={Boolean(approvalAction) || applying} onClick={() => onRejectApprovalRequest(request.id)} type="button">
                      <TriangleAlert size={13} />
                      {approvalAction === "reject" ? t("action.rejecting") : t("action.rejectPermissionRequest")}
                    </button>
                  </div>
                </article>
              ))}
            </div>
          </div>
          {!draft.policyGate.canApplyDirectly ? (
            <div className="permission-approval-request">
              <div className="permission-section-title">
                <strong>{t("section.permissionApprovalRequest")}</strong>
                <Badge tone={approvalRequest ? approvalStatusTone : "warning"}>
                  {approvalRequest ? permissionApprovalStatusLabel(approvalRequest.status, t) : t("status.approvalNotRequested")}
                </Badge>
              </div>
              {approvalRequest ? (
                <div className="permission-approval-grid">
                  <div>
                    <span>{t("detail.approvalRequestId")}</span>
                    <code>{approvalRequest.id}</code>
                  </div>
                  <div>
                    <span>{t("table.version")}</span>
                    <strong>v{approvalRequest.templateVersion} / p{approvalRequest.policyVersion}</strong>
                  </div>
                  <div>
                    <span>{t("detail.createdObjects")}</span>
                    <strong>{approvalRequest.allowedCapabilityIds.length}</strong>
                  </div>
                  <div>
                    <span>{t("table.actor")}</span>
                    <strong>{approvalRequest.reviewedBy || approvalRequest.requestedBy || "-"}</strong>
                  </div>
                </div>
              ) : (
                <span>{t("text.approvalRequestEmpty")}</span>
              )}
              <div className="permission-review-actions">
                {approvalRequest?.status === "pending" ? (
                  <>
                    <button disabled={Boolean(approvalAction) || applying} onClick={() => onApproveApprovalRequest()} type="button">
                      <CheckCircle2 size={14} />
                      {approvalAction === "approve" ? t("action.approving") : t("action.approvePermissionRequest")}
                    </button>
                    <button disabled={Boolean(approvalAction) || applying} onClick={() => onRejectApprovalRequest()} type="button">
                      <TriangleAlert size={14} />
                      {approvalAction === "reject" ? t("action.rejecting") : t("action.rejectPermissionRequest")}
                    </button>
                  </>
                ) : null}
                {approvalRequest?.status !== "pending" && approvalRequest?.status !== "approved" ? (
                  <button disabled={Boolean(approvalAction) || applying || !draft.readiness.canApply} onClick={onCreateApprovalRequest} type="button">
                    <ClipboardCheck size={14} />
                    {approvalAction === "create" ? t("action.creatingApprovalRequest") : t("action.createApprovalRequest")}
                  </button>
                ) : null}
              </div>
            </div>
          ) : null}
          <div className="permission-actions">
            <button className="primary-button" disabled={applying || !canApply} onClick={onApply} type="button">
              <CheckCircle2 size={14} />
              {applying ? t("action.applyingPermissionPackage") : t("action.applyPermissionPackage")}
            </button>
            {message ? <span>{message}</span> : null}
          </div>
        </section>

        <section className="permission-simulation">
          <div className="permission-section-title">
            <strong>{t("section.permissionSimulation")}</strong>
            <span>{t("text.aiAdminSimulationHint")}</span>
          </div>
          <PermissionSimulationTable rows={draft.simulationRows} t={t} />
        </section>
      </div>
    </div>
  );
}

function PermissionApplicationImpactPanel({
  impact,
  loading,
  message,
  onReview,
  t
}: {
  impact: PermissionPackageApplicationImpact | null;
  loading: boolean;
  message: string;
  onReview: () => void;
  t: Translator;
}) {
  const rollbackBlockers = impact?.rollbackReview.blockers ?? [];
  const rollbackBlockerCodes = impact?.rollbackReview.blockerCodes ?? [];
  const rollbackSteps = impact?.rollbackReview.steps ?? [];
  const remediationPlan = impact?.remediationPlan;
  const remediationBlockers = remediationPlan?.blockers ?? [];
  const remediationBlockerCodes = remediationPlan?.blockerCodes ?? [];
  const remediationActions = remediationPlan?.actions ?? [];
  const rollbackBlockerLabels = permissionImpactBlockerLabels(rollbackBlockerCodes, rollbackBlockers, t);
  const remediationBlockerLabels = permissionImpactBlockerLabels(remediationBlockerCodes, remediationBlockers, t);
  return (
    <div className="permission-application-impact">
      <div className="permission-impact-header">
        <div>
          <strong>{t("section.permissionApplicationImpact")}</strong>
          {message ? <span>{message}</span> : null}
        </div>
        <button className="secondary-button" disabled={loading} onClick={onReview} type="button">
          <FileSearch size={14} />
          {loading ? t("action.loading") : t("action.reviewApplicationImpact")}
        </button>
      </div>

      {impact ? (
        <>
          <div className="permission-impact-metrics">
            <div>
              <span>{t("detail.createdObjects")}</span>
              <strong>{impact.summary.createdObjectCount}</strong>
            </div>
            <div>
              <span>{t("metric.activeObjects")}</span>
              <strong>{impact.summary.activeObjectCount}</strong>
            </div>
            <div>
              <span>{t("metric.missingObjects")}</span>
              <strong>{impact.summary.missingObjectCount}</strong>
            </div>
          </div>

          <div className="permission-impact-list">
            {impact.createdObjects.map((item) => (
              <article className="permission-impact-row" key={`${item.type}:${item.id}`}>
                <Badge tone={permissionImpactStatusTone(item.currentStatus)}>
                  {permissionImpactStatusLabel(item.currentStatus, t)}
                </Badge>
                <div>
                  <strong>{permissionImpactObjectLabel(item.type, t)}</strong>
                  <code>{item.id}</code>
                  {item.dataScopes?.length ? <span>{item.dataScopes.map((scope) => summarizeDataScopes([scope])).join(" · ")}</span> : null}
                </div>
                <span>{translatedValue(t, item.rollbackAction)}</span>
              </article>
            ))}
          </div>

          <div className="permission-impact-review">
            <div className="permission-section-title">
              <strong>{t("text.capabilityReview")}</strong>
              <span>{impact.capabilityReviews.length} {t("detail.capabilities")}</span>
            </div>
            <div className="permission-impact-list">
              {impact.capabilityReviews.map((item) => (
                <article className="permission-impact-row" key={item.id}>
                  <Badge tone={permissionImpactStatusTone(item.currentStatus)}>
                    {permissionImpactStatusLabel(item.currentStatus, t)}
                  </Badge>
                  <div>
                    <strong>{item.key || item.id}</strong>
                    <code>{item.id}</code>
                  </div>
                  <span>{translatedValue(t, item.rollbackAction)}</span>
                </article>
              ))}
            </div>
          </div>

          <div className="permission-impact-review">
            <div className="permission-section-title">
              <strong>{t("text.rollbackReview")}</strong>
              <Badge tone={impact.rollbackReview.ready ? "success" : "warning"}>
                {impact.rollbackReview.ready ? t("status.readyToApply") : t("status.needsReview")}
              </Badge>
            </div>
            {rollbackBlockerLabels.length > 0 ? (
              <ul className="permission-impact-blockers">
                {rollbackBlockerLabels.map((blocker) => (
                  <li key={blocker}>{blocker}</li>
                ))}
              </ul>
            ) : null}
            <ol>
              {rollbackSteps.map((step) => (
                <li key={step}>{permissionRollbackStepLabel(step, t)}</li>
              ))}
            </ol>
          </div>

          <div className="permission-impact-review permission-remediation-plan">
            <div className="permission-section-title">
              <strong>{t("text.remediationPlan")}</strong>
              <span>{remediationActions.length} {t("metric.plannedActions")}</span>
            </div>
            <div className="permission-remediation-summary">
              <Badge tone={remediationPlan?.ready ? "success" : "warning"}>
                {remediationPlan?.ready ? t("status.readyToApply") : t("status.needsReview")}
              </Badge>
              <Badge tone="info">
                {translatedValue(t, remediationPlan?.executionMode) || t("text.readOnlyPlan")}
              </Badge>
            </div>
            {remediationBlockerLabels.length > 0 ? (
              <ul className="permission-impact-blockers">
                {remediationBlockerLabels.map((blocker) => (
                  <li key={blocker}>{blocker}</li>
                ))}
              </ul>
            ) : null}
            {remediationActions.length > 0 ? (
              <div className="permission-impact-list">
                {remediationActions.map((action) => (
                  <article className="permission-impact-row permission-remediation-row" key={action.id}>
                    <Badge tone={permissionRemediationActionTone(action.action)}>
                      #{action.order}
                    </Badge>
                    <div>
                      <strong>{permissionRemediationTargetLabel(action.targetType, t)}</strong>
                      <code>{action.targetId || action.id}</code>
                      <span>{translatedValue(t, action.reason)}</span>
                    </div>
                    <span>
                      {action.readOnly ? `${t("text.readOnlyPlan")} · ` : ""}
                      {translatedValue(t, action.action)}
                    </span>
                  </article>
                ))}
              </div>
            ) : (
              <span className="permission-impact-empty">{t("empty.remediationActions.detail")}</span>
            )}
          </div>
        </>
      ) : (
        <span className="permission-impact-empty">{t("empty.permissionApplicationImpact.detail")}</span>
      )}
    </div>
  );
}

function CapabilityChipList({
  capabilities,
  emptyLabel,
  label,
  tone,
  t
}: {
  capabilities: Capability[];
  emptyLabel: string;
  label: string;
  tone: Tone;
  t: Translator;
}) {
  return (
    <div className="permission-chip-list">
      <strong>{label}</strong>
      {capabilities.length === 0 ? <span>{emptyLabel}</span> : null}
      {capabilities.map((capability) => (
        <Badge key={capability.id} tone={tone}>
          {capability.key} · {translatedValue(t, capability.action)}
        </Badge>
      ))}
    </div>
  );
}

function PermissionSimulationTable({ rows, t }: { rows: PermissionPackageSimulationRow[]; t: Translator }) {
  return (
    <div className="table-wrap">
      <table className="permission-simulation-table">
        <thead>
          <tr>
            <th>{t("table.check")}</th>
            <th>{t("table.decision")}</th>
            <th>{t("table.reason")}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.id}>
              <td>
                <strong>{row.capabilityKey}</strong>
                <span>{row.capabilityId ?? t("text.packageGuardrail")}</span>
              </td>
              <td>
                <Badge tone={row.expectedDecision === "allow" ? "success" : "danger"}>
                  {row.expectedDecision === "allow" ? t("text.decisionAllowed") : t("text.decisionDenied")}
                </Badge>
              </td>
              <td>{permissionSimulationReason(row, t)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function CoreJourneyStepRow({ step, t }: { step: CoreJourneyStep; t: Translator }) {
  return (
    <article className={`core-journey-step status-${step.status}`}>
      <Badge tone={coreJourneyStatusTone(step.status)}>{coreJourneyStatusLabel(step.status, t)}</Badge>
      <div>
        <strong>{t(`journey.step.${step.key}`)}</strong>
        <span>{step.detail}</span>
      </div>
      <code>{step.metric}</code>
    </article>
  );
}

function AiAdminApprovalJourneyStepRow({ step, t }: { step: AiAdminApprovalJourneyStep; t: Translator }) {
  return (
    <article className={`ai-admin-journey-step status-${step.status}`}>
      <Badge tone={aiAdminApprovalJourneyStatusTone(step.status)}>
        {aiAdminApprovalJourneyStatusLabel(step.status, t)}
      </Badge>
      <div>
        <strong>{t(`journey.aiAdmin.step.${step.key}`)}</strong>
        <span>{step.detail}</span>
      </div>
      <code>{step.metric}</code>
    </article>
  );
}

function AiAdminApprovalReadinessRow({ row, t }: { row: AiAdminApprovalReadinessRow; t: Translator }) {
  return (
    <article className={`ai-admin-readiness-row status-${row.status}`}>
      <Badge tone={aiAdminApprovalReadinessStatusTone(row.status)}>
        {aiAdminApprovalReadinessStatusLabel(row.status, t)}
      </Badge>
      <div>
        <strong>{t(row.titleKey)}</strong>
        <span>{t(row.detailKey)}</span>
      </div>
    </article>
  );
}

function AgentCreateForm({
  form,
  message,
  onChange,
  onSubmit,
  t
}: {
  form: typeof defaultAgentForm;
  message: string;
  onChange: (form: typeof defaultAgentForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  t: Translator;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
      <label>{t("form.name")}<input required value={form.name} onChange={(event) => onChange({ ...form, name: event.target.value })} /></label>
      <div className="form-row">
        <label>{t("form.channel")}<input value={form.channelType} onChange={(event) => onChange({ ...form, channelType: event.target.value })} /></label>
        <label>{t("table.status")}<select value={form.status} onChange={(event) => onChange({ ...form, status: event.target.value as AgentStatus })}><option value="draft">{t("status.agentDraft")}</option><option value="active">{t("status.agentActive")}</option><option value="disabled">{t("status.agentDisabled")}</option></select></label>
      </div>
      <label>{t("form.endpoint")}<input placeholder="https://api.example.com/a2a" value={form.endpoint} onChange={(event) => onChange({ ...form, endpoint: event.target.value })} /></label>
      <div className="form-row">
        <label>{t("form.credentialHeader")}<input placeholder="Authorization" value={form.credentialHeader} onChange={(event) => onChange({ ...form, credentialHeader: event.target.value })} /></label>
        <label>{t("form.credentialKey")}<input placeholder="apiToken" value={form.credentialName} onChange={(event) => onChange({ ...form, credentialName: event.target.value })} /></label>
      </div>
      <label>{t("form.secretValue")}<input placeholder="Bearer ..." type="password" value={form.credentialValue} onChange={(event) => onChange({ ...form, credentialValue: event.target.value })} /></label>
      <div className="form-row">
        <label>{t("form.retryAttempts")}<input inputMode="numeric" max={4} min={1} type="number" value={form.retryMaxAttempts} onChange={(event) => onChange({ ...form, retryMaxAttempts: event.target.value })} /></label>
        <label>{t("form.backoffMs")}<input inputMode="numeric" max={1000} min={0} type="number" value={form.retryBackoffMs} onChange={(event) => onChange({ ...form, retryBackoffMs: event.target.value })} /></label>
      </div>
      <label>{t("form.description")}<textarea rows={2} value={form.description} onChange={(event) => onChange({ ...form, description: event.target.value })} /></label>
      <FormFooter message={message} submitLabel={t("action.createAgent")} />
    </form>
  );
}

function KeyCreateForm({
  agents,
  createdKey,
  form,
  message,
  onChange,
  onSubmit,
  t
}: {
  agents: Agent[];
  createdKey: CreateAgentKeyResponse | null;
  form: typeof defaultKeyForm;
  message: string;
  onChange: (form: typeof defaultKeyForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  t: Translator;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
      <label>{t("form.caller")}<select required value={form.agentId} onChange={(event) => onChange({ ...form, agentId: event.target.value })}><option value="">{t("form.selectCaller")}</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <label>{t("form.name")}<input value={form.name} onChange={(event) => onChange({ ...form, name: event.target.value })} /></label>
      <label>{t("form.ttlSeconds")}<input inputMode="numeric" max={3600} min={1} type="number" value={form.expiresInSeconds} onChange={(event) => onChange({ ...form, expiresInSeconds: event.target.value })} /></label>
      {createdKey ? (
        <div className="one-time-key">
          <div><strong>{t("text.oneTimeKey")}</strong><span>{tx(t, "text.oneTimeKeyDetail", { expiresAt: formatDate(createdKey.expiresAt) })}</span></div>
          <code>{createdKey.key}</code>
          <button className="secondary-button" type="button" onClick={() => void navigator.clipboard?.writeText(createdKey.key)}><Copy size={14} /> {t("action.copy")}</button>
        </div>
      ) : null}
      <FormFooter message={message} submitLabel={t("action.createKey")} />
    </form>
  );
}

function CredentialRotateForm({
  agents,
  form,
  message,
  onChange,
  onSubmit,
  t
}: {
  agents: Agent[];
  form: typeof defaultRotateForm;
  message: string;
  onChange: (form: typeof defaultRotateForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  t: Translator;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
      <label>{t("form.agent")}<select required value={form.agentId} onChange={(event) => onChange({ ...form, agentId: event.target.value })}><option value="">{t("form.selectAgent")}</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <label>{t("form.credentialKey")}<input placeholder="apiToken" value={form.credentialName} onChange={(event) => onChange({ ...form, credentialName: event.target.value })} /></label>
      <label>{t("form.newSecret")}<input placeholder="Bearer ..." type="password" value={form.credentialValue} onChange={(event) => onChange({ ...form, credentialValue: event.target.value })} /></label>
      <FormFooter message={message} submitLabel={t("action.rotateCredential")} />
    </form>
  );
}

function PolicyCreateForm({
  agents,
  form,
  message,
  onChange,
  onSubmit,
  t
}: {
  agents: Agent[];
  form: typeof defaultPolicyForm;
  message: string;
  onChange: (form: typeof defaultPolicyForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  t: Translator;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
      <label>{t("form.name")}<input placeholder="Allow MCP tools/call" value={form.name} onChange={(event) => onChange({ ...form, name: event.target.value })} /></label>
      <label>{t("form.caller")}<select required value={form.callerAgentId} onChange={(event) => onChange({ ...form, callerAgentId: event.target.value })}><option value="">{t("form.selectCaller")}</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <label>{t("form.target")}<select required value={form.targetAgentId} onChange={(event) => onChange({ ...form, targetAgentId: event.target.value })}><option value="">{t("form.anyTarget")}</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <div className="route-presets" aria-label={t("form.routeKeyPresets")}>
        {mcpRouteKeyPresets.map((preset) => (
          <button
            className={form.routeType === "mcp" && form.routeKey === preset ? "selected" : ""}
            key={preset}
            onClick={() => onChange({ ...form, routeKey: preset, routeType: "mcp" })}
            type="button"
          >
            {preset}
          </button>
        ))}
        <button
          className={form.routeType === "mcp" && form.routeKey === "" ? "selected" : ""}
          onClick={() => onChange({ ...form, routeKey: "", routeType: "mcp" })}
          type="button"
        >
          {t("text.routeWildcard")}
        </button>
      </div>
      <div className="form-row">
        <label>{t("form.routeType")}<input value={form.routeType} onChange={(event) => onChange({ ...form, routeType: event.target.value })} /></label>
        <label>{t("form.routeKey")}<input value={form.routeKey} onChange={(event) => onChange({ ...form, routeKey: event.target.value })} /></label>
      </div>
      <div className="form-row">
        <label>{t("form.effect")}<select value={form.effect} onChange={(event) => onChange({ ...form, effect: event.target.value })}><option value="allow">{t("status.policyAllow")}</option><option value="deny">{t("status.policyDeny")}</option></select></label>
        <label>{t("form.priority")}<input inputMode="numeric" min={0} type="number" value={form.priority} onChange={(event) => onChange({ ...form, priority: event.target.value })} /></label>
      </div>
      <div className="form-row">
        <label>{t("form.retryAttempts")}<input inputMode="numeric" max={4} min={1} type="number" value={form.retryMaxAttempts} onChange={(event) => onChange({ ...form, retryMaxAttempts: event.target.value })} /></label>
        <label>{t("form.retryBackoffMs")}<input inputMode="numeric" max={1000} min={0} type="number" value={form.retryBackoffMs} onChange={(event) => onChange({ ...form, retryBackoffMs: event.target.value })} /></label>
      </div>
      <FormFooter message={message} submitLabel={t("action.createPolicy")} />
    </form>
  );
}

function TraceFilterBar({
  agents,
  filters,
  onChange,
  onRefresh,
  t
}: {
  agents: Agent[];
  filters: TraceFilters;
  onChange: (filters: TraceFilters) => void;
  onRefresh: () => void;
  t: Translator;
}) {
  return (
    <div className="trace-filters">
      <input placeholder="runId" value={filters.runId ?? ""} onChange={(event) => onChange({ ...filters, runId: event.target.value })} />
      <select value={filters.decision ?? ""} onChange={(event) => onChange({ ...filters, decision: event.target.value as TraceDecision | "" })}>
        <option value="">{t("form.anyDecision")}</option>
        <option value="allowed">{t("text.decisionAllowed")}</option>
        <option value="denied">{t("text.decisionDenied")}</option>
      </select>
      <select value={filters.callerAgentId ?? ""} onChange={(event) => onChange({ ...filters, callerAgentId: event.target.value })}>
        <option value="">{t("form.anyCaller")}</option>
        {agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}
      </select>
      <select value={filters.targetAgentId ?? ""} onChange={(event) => onChange({ ...filters, targetAgentId: event.target.value })}>
        <option value="">{t("form.anyTarget")}</option>
        {agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}
      </select>
      <button className="secondary-button" type="button" onClick={onRefresh}><RefreshCw size={14} /> {t("action.refresh")}</button>
    </div>
  );
}

function FormFooter({ message, submitLabel }: { message: string; submitLabel: string }) {
  return (
    <div className="form-footer">
      <button className="primary-button" type="submit">{submitLabel}</button>
      {message ? <span>{message}</span> : null}
    </div>
  );
}

function MetricCard({
  icon,
  label,
  value,
  detail,
  tone
}: {
  icon: ReactNode;
  label: string;
  value: string;
  detail: string;
  tone: Tone;
}) {
  return (
    <article className={`metric-card tone-${tone}`}>
      <div className="metric-icon">{icon}</div>
      <div>
        <span>{label}</span>
        <strong>{value}</strong>
        <small>{detail}</small>
      </div>
    </article>
  );
}

function Panel({
  title,
  icon,
  action,
  className,
  children
}: {
  title: string;
  icon: ReactNode;
  action?: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section className={`panel ${className ?? ""}`}>
      <header className="panel-header">
        <div>
          {icon}
          <h2>{title}</h2>
        </div>
        {action}
      </header>
      {children}
    </section>
  );
}

function PolicyTable({
  agents,
  canDisable,
  onDisable,
  pendingActionId,
  policies,
  t
}: {
  agents: Agent[];
  canDisable: boolean;
  onDisable: (policy: RoutePolicy) => void;
  pendingActionId: string;
  policies: RoutePolicy[];
  t: Translator;
}) {
  const names = agentNameMap(agents);

  return (
    <div className="table-wrap">
      <table className="policy-table">
        <thead>
          <tr>
            <th>{t("table.policy")}</th>
            <th>{t("table.caller")}</th>
            <th>{t("table.target")}</th>
            <th>{t("table.route")}</th>
            <th>{t("table.decision")}</th>
            <th>{t("table.action")}</th>
          </tr>
        </thead>
        <tbody>
          {policies.length === 0 ? (
            <tr>
              <td colSpan={6}>
                <EmptyRow title={t("empty.routePolicies.title")} detail={t("empty.routePolicies.detail")} />
              </td>
            </tr>
          ) : null}
          {policies.map((policy) => (
            <tr className={policy.status === "disabled" ? "row-disabled" : undefined} key={policy.id}>
              <td>
                <strong>{policy.name}</strong>
                <span>{tx(t, "text.policyPriority", { priority: policy.priority })} · {policyRetryText(policy, t)} · {tx(t, "text.policyMatched", { date: formatDate(policy.lastMatchedAt ?? policy.createdAt) })}</span>
              </td>
              <td>{names[policy.callerAgentId] ?? policy.callerAgentId}</td>
              <td>{names[policy.targetAgentId] ?? policy.targetAgentId}</td>
              <td>
                <code>{policy.routeType}:{policy.routeKey || t("text.routeWildcard")}</code>
              </td>
              <td><Badge tone={policy.effect === "allow" ? "success" : "danger"}>{policyEffectLabel(policy.effect, t)}</Badge></td>
              <td>
                {canDisable && policy.status === "enabled" ? (
                  <button
                    className="table-action"
                    disabled={pendingActionId === policy.id}
                    onClick={() => onDisable(policy)}
                    type="button"
                  >
                    <LockKeyhole size={13} />
                    {pendingActionId === policy.id ? t("action.disabling") : t("action.disable")}
                  </button>
                ) : (
                  <span className="muted-action">{policy.status === "disabled" ? t("status.agentDisabled") : t("status.sample")}</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function AgentTable({
  agents,
  channelLabels,
  onStatusChange,
  pendingActionId,
  t
}: {
  agents: Agent[];
  channelLabels: Record<string, string>;
  onStatusChange: (agent: Agent, status: AgentStatus) => void;
  pendingActionId: string;
  t: Translator;
}) {
  return (
    <div className="table-wrap">
      <table className="agent-table">
        <thead>
          <tr>
            <th>{t("table.name")}</th>
            <th>{t("table.channel")}</th>
            <th>{t("table.endpoint")}</th>
            <th>{t("table.status")}</th>
            <th>{t("table.owner")}</th>
            <th>{t("table.action")}</th>
          </tr>
        </thead>
        <tbody>
          {agents.length === 0 ? (
            <tr>
              <td colSpan={6}>
                <EmptyRow title={t("empty.registry.title")} detail={t("empty.registry.detail")} />
              </td>
            </tr>
          ) : null}
          {agents.map((agent) => (
            <tr className={agent.status === "disabled" ? "row-disabled" : undefined} key={agent.id}>
              <td>
                <strong>{agent.name}</strong>
                <span>{agent.description || agent.id}</span>
              </td>
              <td>{channelLabel(agent.channelType, channelLabels, t)}</td>
              <td className="truncate">{configText(agent, "endpoint") || t("status.localRuntime")}</td>
              <td><Badge tone={agent.status === "active" ? "success" : agent.status === "draft" ? "warning" : "neutral"}>{agentStatusLabel(agent.status, t)}</Badge></td>
              <td>{agent.ownerId || t("text.ownerPlatform")}</td>
              <td>
                {agent.status !== "disabled" ? (
                  <div className="table-action-group">
                    <button
                      className="table-action"
                      disabled={pendingActionId === agent.id}
                      onClick={() => onStatusChange(agent, agent.status === "active" ? "draft" : "active")}
                      type="button"
                    >
                      {agent.status === "active" ? <CircleDot size={13} /> : <CheckCircle2 size={13} />}
                      {pendingActionId === agent.id ? t("action.updating") : agent.status === "active" ? t("action.draft") : t("action.activate")}
                    </button>
                    <button
                      className="table-action"
                      disabled={pendingActionId === agent.id}
                      onClick={() => onStatusChange(agent, "disabled")}
                      type="button"
                    >
                      <LockKeyhole size={13} />
                      {t("action.disable")}
                    </button>
                  </div>
                ) : (
                  <span className="muted-action">{t("status.agentDisabled")}</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function EvidenceTimeline({ runs, t }: { runs: EvidenceRun[]; t: Translator }) {
  return (
    <div className="timeline">
      {runs.length === 0 ? <EmptyRow title={t("empty.evidenceRuns.title")} detail={t("empty.evidenceRuns.detail")} /> : null}
      {runs.map((run) => (
        <article className="timeline-row" key={run.id}>
          <div className={`timeline-marker tone-${toneFromStatus(run.status)}`}>
            {run.status === "passed" ? <CheckCircle2 size={14} /> : <Activity size={14} />}
          </div>
          <div>
            <div className="timeline-title">
              <strong>{run.title}</strong>
              <Badge tone={toneFromStatus(run.status)}>{evidenceStatusLabel(run.status, t)}</Badge>
            </div>
            <p>{run.checks} {t("text.checks")} · {formatDuration(evidenceDuration(run))} · {formatDate(run.completedAt ?? run.startedAt)}</p>
          </div>
        </article>
      ))}
    </div>
  );
}

function ContractMatrix({
  channels,
  providers,
  t
}: {
  channels: ChannelContract[];
  providers: Array<{ key: string; label: string; channelType: string; requiredCreds?: string[] }>;
  t: Translator;
}) {
  return (
    <div className="contract-list">
      {channels.map((channel) => (
        <div className="contract-row" key={channel.key}>
          <div>
            <strong>{channelLabel(channel.key, { [channel.key]: channel.label }, t)}</strong>
            <span>{channel.key}</span>
          </div>
          <Badge tone={channel.endpointRequiredWhenActive ? "warning" : "neutral"}>
            {channel.endpointRequiredWhenActive ? t("form.endpoint") : translatedValue(t, "local")}
          </Badge>
        </div>
      ))}
      <div className="provider-strip">
        {providers.map((provider) => (
          <span key={provider.key}>{tx(t, "text.provider", { label: provider.label, channelType: translatedValue(t, provider.channelType) })}</span>
        ))}
      </div>
    </div>
  );
}

function SignalBoard({ metrics, t }: { metrics: SystemMetric[]; t: Translator }) {
  return (
    <div className="signal-grid">
      {metrics.map((metric) => (
        <article className="signal" key={metric.id}>
          <span>{t(`signal.${metric.id}`, metric.label)}</span>
          <strong>{metric.value}{metric.unit ?? ""}</strong>
          <div className="signal-track" aria-hidden="true">
            <i style={{ width: `${metricRatio(metric)}%` }} />
          </div>
          <small>{translatedValue(t, metric.trend)} · {translatedValue(t, metric.status)}</small>
        </article>
      ))}
    </div>
  );
}

function TraceTable({ traces, agents, t }: { traces: TraceEvent[]; agents: Agent[]; t: Translator }) {
  const names = useMemo(() => agentNameMap(agents), [agents]);

  return (
    <div className="trace-list">
      {traces.length === 0 ? <EmptyRow title={t("empty.auditTraces.title")} detail={t("empty.auditTraces.detail")} /> : null}
      {traces.map((trace) => (
        <article className="trace-row" key={trace.id}>
          <div className={`trace-decision tone-${trace.decision === "allowed" ? "success" : "danger"}`}>
            {trace.decision === "allowed" ? <CheckCircle2 size={15} /> : <LockKeyhole size={15} />}
          </div>
          <div>
            <div className="trace-title-line">
              <strong>{names[trace.callerAgentId ?? ""] ?? trace.callerAgentId ?? t("text.traceAnonymous")} → {names[trace.targetAgentId] ?? trace.targetAgentId}</strong>
              <Badge tone={trace.decision === "allowed" ? "success" : "danger"}>
                {trace.decision === "allowed" ? t("text.decisionAllowed") : t("text.decisionDenied")}
              </Badge>
            </div>
            <span>
              {trace.routeType}:{trace.routeKey || t("text.traceDefaultRoute")}
              {trace.capabilityId ? ` · ${trace.capabilityId}` : ""}
              {" · "}
              {trace.reason || policyEffectLabel(trace.decision === "allowed" ? "allow" : "deny", t)}
            </span>
          </div>
          <time>{formatDate(trace.createdAt)}</time>
        </article>
      ))}
    </div>
  );
}

function TenantAccessProfileView({
  agents,
  capabilities,
  explanation,
  explanationLoading,
  explanationMessage,
  filters,
  loading,
  message,
  onChange,
  onExplainAccessDecision,
  onRefresh,
  onTenantChange,
  profile,
  scope,
  t
}: {
  agents: Agent[];
  capabilities: Capability[];
  explanation: AccessDecisionExplainResult | null;
  explanationLoading: boolean;
  explanationMessage: string;
  filters: AccessProfileFilters;
  loading: boolean;
  message: string;
  onChange: (filters: AccessProfileFilters) => void;
  onExplainAccessDecision: () => void;
  onRefresh: () => void;
  onTenantChange: (tenantId: string) => void;
  profile: TenantAccessProfileData | null;
  scope: ManagementScope;
  t: Translator;
}) {
  const names = useMemo(() => agentNameMap(agents), [agents]);
  const targetAgents = agents.filter((agent) => agent.channelType !== "local");
  const visibleCapabilities = filters.targetId
    ? capabilities.filter((capability) => capability.targetId === filters.targetId)
    : capabilities;
  const sourceLabel = profile
    ? profile.loadedFromApi
      ? t("status.sourceLive")
      : t("status.sourceFallback")
    : t("status.sourceNotLoaded");
  const profileSummary = profile?.summary ?? emptyAccessProfileSummary;
  const profileScopeTenants = profile?.scopeTenants ?? [];
  const profileGrants = profile?.grants ?? [];
  const profileRecentTraces = profile?.recentTraces ?? [];

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onRefresh();
  }

  return (
    <div className="access-profile">
      <form className="access-toolbar" onSubmit={submit}>
        <label>
          {t("form.tenant")}
          <input value={scope.tenantId} onChange={(event) => onTenantChange(event.target.value)} />
        </label>
        <label>
          {t("form.workspace")}
          <input
            placeholder={t("form.workspaceAll")}
            value={filters.workspaceId ?? ""}
            onChange={(event) => onChange({ ...filters, workspaceId: event.target.value })}
          />
        </label>
        <label>
          {t("form.target")}
          <select
            value={filters.targetId ?? ""}
            onChange={(event) => onChange({ ...filters, capabilityId: "", targetId: event.target.value })}
          >
            <option value="">{t("form.anyTarget")}</option>
            {targetAgents.map((agent) => (
              <option key={agent.id} value={agent.id}>
                {agent.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          {t("form.capability")}
          <select
            value={filters.capabilityId ?? ""}
            onChange={(event) => onChange({ ...filters, capabilityId: event.target.value })}
          >
            <option value="">{t("form.anyCapability")}</option>
            {visibleCapabilities.map((capability) => (
              <option key={capability.id} value={capability.id}>
                {capability.key}
              </option>
            ))}
          </select>
        </label>
        <label>
          {t("form.caller")}
          <select
            value={filters.callerInstanceId ?? ""}
            onChange={(event) => onChange({ ...filters, callerInstanceId: event.target.value })}
          >
            <option value="">{t("form.anyCaller")}</option>
            {agents.map((agent) => (
              <option key={agent.id} value={agent.id}>
                {agent.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          {t("form.subjectId")}
          <input
            placeholder={t("detail.subjectId")}
            value={filters.subjectId ?? ""}
            onChange={(event) => onChange({ ...filters, subjectId: event.target.value })}
          />
        </label>
        <label>
          {t("form.traces")}
          <input
            inputMode="numeric"
            max={100}
            min={0}
            type="number"
            value={String(filters.traceLimit ?? "")}
            onChange={(event) => onChange({ ...filters, traceLimit: event.target.value })}
          />
        </label>
        <button className="secondary-button" disabled={loading} type="submit">
          <RefreshCw size={14} />
          {loading ? t("action.loading") : t("action.loadProfile")}
        </button>
      </form>

      <div className="access-source-line">
        <span>{sourceLabel}</span>
        {profile ? <span>{t("status.generated")} {formatDate(profile.generatedAt)}</span> : null}
        {message ? <strong>{message}</strong> : null}
      </div>

      <AccessDecisionExplainPanel
        explanation={explanation}
        loading={explanationLoading}
        message={explanationMessage}
        onExplain={onExplainAccessDecision}
        t={t}
      />

      {!profile ? (
        <EmptyRow title={t("empty.accessProfile.title")} detail={t("empty.accessProfile.detail")} />
      ) : (
        <>
          <div className="access-summary-grid">
            <AccessSummaryCell label={t("summary.tenantScope")} value={String(profileSummary.tenantCount ?? profileScopeTenants.length)} detail={profile.tenant.name} />
            <AccessSummaryCell label={t("summary.grants")} value={String(profileSummary.grantCount ?? profileGrants.length)} detail={`${profileSummary.capabilityCount ?? 0} ${t("detail.capabilities")}`} />
            <AccessSummaryCell label={t("summary.assignments")} value={`${profileSummary.workspaceAssignmentCount ?? 0}/${profileSummary.instanceAssignmentCount ?? 0}`} detail={t("text.workspaceCaller")} />
            <AccessSummaryCell label={t("summary.recentDecisions")} value={`${profileSummary.recentAllowedTraceCount ?? 0}/${profileSummary.recentDeniedTraceCount ?? 0}`} detail={t("text.allowedDenied")} />
          </div>

          <div className="access-tenant-list" aria-label={t("summary.tenantScope")}>
            {profileScopeTenants.map((tenant) => (
              <div className="access-tenant-row" key={tenant.id}>
                <Badge tone={tenant.status === "active" ? "success" : "neutral"}>L{tenant.level}</Badge>
                <div>
                  <strong>{tenant.name}</strong>
                  <span>{tenant.id}{tenant.parentTenantId ? ` · ${t("text.parentTenant")} ${tenant.parentTenantId}` : ""}</span>
                </div>
              </div>
            ))}
          </div>

          <div className="access-layout">
            <section className="access-grant-chain" aria-label={t("section.effectiveGrantChain")}>
              <header>
                <strong>{t("section.effectiveGrantChain")}</strong>
                <span>{countInvalidAccessProfileRows(profile)} {t("text.invalidRows")}</span>
              </header>
              {profileGrants.length === 0 ? (
                <EmptyRow title={t("empty.grantChains.title")} detail={t("empty.grantChains.detail")} />
              ) : null}
              {profileGrants.map((grant) => (
                <AccessGrantRow grant={grant} key={grant.tenantEntitlement.id} t={t} />
              ))}
            </section>

            <section className="access-trace-evidence" aria-label={t("section.traceEvidence")}>
              <header>
                <strong>{t("section.traceEvidence")}</strong>
                <span>{profileRecentTraces.length} {t("text.recentTraces")}</span>
              </header>
              {profileRecentTraces.length === 0 ? (
                <EmptyRow title={t("empty.traceEvidence.title")} detail={t("empty.traceEvidence.detail")} />
              ) : null}
              {profileRecentTraces.map((trace) => (
                <article className="access-trace-row" key={trace.id}>
                  <div className={`trace-decision tone-${trace.decision === "allowed" ? "success" : "danger"}`}>
                    {trace.decision === "allowed" ? <CheckCircle2 size={15} /> : <LockKeyhole size={15} />}
                  </div>
                  <div>
                    <div className="trace-title-line">
                      <strong>{names[trace.callerAgentId ?? ""] ?? trace.callerAgentId ?? t("text.traceAnonymous")} → {names[trace.targetAgentId] ?? trace.targetAgentId}</strong>
                      <Badge tone={trace.decision === "allowed" ? "success" : "danger"}>
                        {trace.decision === "allowed" ? t("text.decisionAllowed") : t("text.decisionDenied")}
                      </Badge>
                    </div>
                    <span>{trace.capabilityId ?? `${trace.routeType}:${trace.routeKey || t("text.traceDefaultRoute")}`} · {summarizeDataScopes(trace.dataScopes)} · {trace.reason || policyEffectLabel(trace.decision === "allowed" ? "allow" : "deny", t)}</span>
                  </div>
                  <time>{formatDate(trace.createdAt)}</time>
                </article>
              ))}
            </section>
          </div>
        </>
      )}
    </div>
  );
}

function AccessSummaryCell({ label, value, detail }: { label: string; value: string; detail: string }) {
  return (
    <div className="access-summary-cell">
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{detail}</small>
    </div>
  );
}

function AccessDecisionExplainPanel({
  explanation,
  loading,
  message,
  onExplain,
  t
}: {
  explanation: AccessDecisionExplainResult | null;
  loading: boolean;
  message: string;
  onExplain: () => void;
  t: Translator;
}) {
  const dataScopes = explanation?.dataScopes ?? explanation?.decision.dataScopes ?? [];
  return (
    <section className="access-decision-explain">
      <header>
        <div>
          <strong>{t("section.accessDecisionExplain")}</strong>
          {message ? <span>{message}</span> : null}
        </div>
        <button className="secondary-button" disabled={loading} onClick={onExplain} type="button">
          <FileSearch size={14} />
          {loading ? t("action.loading") : t("action.explainAccessDecision")}
        </button>
      </header>
      {!explanation ? (
        <EmptyRow title={t("section.accessDecisionExplain")} detail={t("empty.accessDecisionExplain.detail")} />
      ) : (
        <>
          <div className="access-decision-summary">
            <Badge tone={accessDecisionOutcomeTone(explanation.outcome)}>
              {accessDecisionOutcomeLabel(explanation.outcome, t)}
            </Badge>
            <div>
              <strong>{explanation.summary}</strong>
              <span>{explanation.decision.source} · {explanation.decision.reason}</span>
            </div>
          </div>
          <div className="access-decision-evidence">
            {explanation.evidence.map((row) => (
              <article key={`${row.layer}:${row.id ?? row.status}`}>
                <Badge tone={accessDecisionEvidenceTone(row.status)}>{row.status}</Badge>
                <div>
                  <strong>{row.layer}</strong>
                  <span>{row.message}</span>
                  {row.id ? <code>{row.id}</code> : null}
                </div>
              </article>
            ))}
          </div>
          <div className="access-decision-footer">
            <div>
              <strong>{t("detail.dataScopes")}</strong>
              <span>{summarizeDataScopes(dataScopes)}</span>
            </div>
            <div>
              <strong>{t("text.nextActions")}</strong>
              <ul>
                {explanation.nextActions.map((action) => (
                  <li key={action}>{action}</li>
                ))}
              </ul>
            </div>
          </div>
        </>
      )}
    </section>
  );
}

function AccessGrantRow({ grant, t }: { grant: TenantAccessProfileGrant; t: Translator }) {
  const invalidRows = countInvalidGrantRows(grant);
  return (
    <article className={invalidRows > 0 ? "access-grant-row invalid" : "access-grant-row"}>
      <div className="access-grant-header">
        <div>
          <strong>{grant.capability?.displayName ?? grant.capability?.key ?? grant.tenantEntitlement.capabilityId}</strong>
          <span>{grant.target?.name ?? grant.tenantEntitlement.targetId}</span>
        </div>
        <div className="access-badge-group">
          <Badge tone={scopeStatusTone(grant.scopeStatus)}>{grant.scopeStatus}</Badge>
          <Badge tone={grant.tenantEntitlement.effect === "allow" ? "success" : "danger"}>{policyEffectLabel(grant.tenantEntitlement.effect, t)}</Badge>
        </div>
      </div>
      <div className="access-scope-line">
        <span>{t("text.tenantEntitlement")}</span>
        <code>{grant.tenantEntitlement.id}</code>
        <span>{summarizeDataScopes(grant.effectiveTenantDataScopes)}</span>
      </div>
      {grant.scopeReason ? <p className="access-invalid-reason">{grant.scopeReason}</p> : null}
      <div className="access-nested-list">
        {grant.workspaceAssignments.length === 0 ? (
          <EmptyRow title={t("empty.workspaceAssignments.title")} detail={t("empty.workspaceAssignments.detail")} />
        ) : null}
        {grant.workspaceAssignments.map((workspace) => (
          <AccessWorkspaceRow key={workspace.workspaceAssignment.id} workspace={workspace} t={t} />
        ))}
      </div>
    </article>
  );
}

function AccessWorkspaceRow({ workspace, t }: { workspace: TenantAccessProfileWorkspace; t: Translator }) {
  return (
    <div className="access-workspace-row">
      <div className="access-row-main">
        <div>
          <strong>{workspace.workspaceAssignment.workspaceId}</strong>
          <span>{summarizeDataScopes(workspace.effectiveWorkspaceDataScopes)}</span>
        </div>
        <Badge tone={scopeStatusTone(workspace.scopeStatus)}>{workspace.scopeStatus}</Badge>
      </div>
      {workspace.scopeReason ? <p className="access-invalid-reason">{workspace.scopeReason}</p> : null}
      <div className="access-instance-list">
        {workspace.instanceAssignments.length === 0 ? (
          <span className="access-empty-inline">{t("empty.callerInstances.title")}</span>
        ) : null}
        {workspace.instanceAssignments.map((instance) => (
          <AccessInstanceRow instance={instance} key={instance.instanceAssignment.id} t={t} />
        ))}
      </div>
    </div>
  );
}

function AccessInstanceRow({ instance, t }: { instance: TenantAccessProfileInstance; t: Translator }) {
  return (
    <div className="access-instance-row">
      <div>
        <strong>{instance.callerInstance?.name ?? instance.instanceAssignment.callerInstanceId}</strong>
        <span>{instance.instanceAssignment.subjectSelector || t("text.subjectsAll")} · {summarizeDataScopes(instance.effectiveInstanceDataScopes)}</span>
      </div>
      <Badge tone={scopeStatusTone(instance.scopeStatus)}>{instance.scopeStatus}</Badge>
      {instance.scopeReason ? <p className="access-invalid-reason">{instance.scopeReason}</p> : null}
    </div>
  );
}

function CapabilityGovernanceView({
  actionId,
  agents,
  capabilities,
  form,
  instanceAssignments,
  message,
  mcpTargets,
  onApprove,
  onChange,
  onCreateGrantChain,
  onRefreshTarget,
  t,
  tenantEntitlements,
  workspaceAssignments
}: {
  actionId: string;
  agents: Agent[];
  capabilities: Capability[];
  form: typeof defaultCapabilityGrantForm;
  instanceAssignments: InstanceAssignment[];
  message: string;
  mcpTargets: Agent[];
  onApprove: (capability: Capability) => void;
  onChange: (form: typeof defaultCapabilityGrantForm) => void;
  onCreateGrantChain: (event: FormEvent<HTMLFormElement>) => void;
  onRefreshTarget: () => void;
  t: Translator;
  tenantEntitlements: TenantEntitlement[];
  workspaceAssignments: WorkspaceAssignment[];
}) {
  const agentNames = useMemo(() => agentNameMap(agents), [agents]);
  const visibleCapabilities = useMemo(() => {
    const targetId = form.targetId.trim();
    return targetId ? capabilities.filter((capability) => capability.targetId === targetId) : capabilities;
  }, [capabilities, form.targetId]);
  const selectedCapability = capabilities.find((capability) => capability.id === form.capabilityId);
  const entitlementIdsByCapability = useMemo(() => {
    return tenantEntitlements.reduce<Record<string, string[]>>((acc, entitlement) => {
      acc[entitlement.capabilityId] = [...(acc[entitlement.capabilityId] ?? []), entitlement.id];
      return acc;
    }, {});
  }, [tenantEntitlements]);
  const workspaceIdsByEntitlement = useMemo(() => {
    return workspaceAssignments.reduce<Record<string, string[]>>((acc, assignment) => {
      acc[assignment.tenantEntitlementId] = [...(acc[assignment.tenantEntitlementId] ?? []), assignment.id];
      return acc;
    }, {});
  }, [workspaceAssignments]);
  const instancesByWorkspaceAssignment = useMemo(() => {
    return instanceAssignments.reduce<Record<string, InstanceAssignment[]>>((acc, assignment) => {
      acc[assignment.workspaceAssignmentId] = [...(acc[assignment.workspaceAssignmentId] ?? []), assignment];
      return acc;
    }, {});
  }, [instanceAssignments]);

  function handleTargetChange(targetId: string) {
    const nextCapability = capabilities.find((capability) => capability.targetId === targetId);
    onChange({
      ...form,
      capabilityId: nextCapability?.id ?? "",
      targetId
    });
  }

  function handleCapabilityChange(capabilityId: string) {
    const capability = capabilities.find((item) => item.id === capabilityId);
    onChange({
      ...form,
      capabilityId,
      targetId: capability?.targetId ?? form.targetId
    });
  }

  return (
    <div className="capability-governance">
      <div className="capability-toolbar">
        <label>
          {t("form.target")}
          <select value={form.targetId} onChange={(event) => handleTargetChange(event.target.value)}>
            <option value="">{t("form.allMcpTargets")}</option>
            {mcpTargets.map((target) => (
              <option key={target.id} value={target.id}>
                {target.name}
              </option>
            ))}
          </select>
        </label>
        <button
          className="secondary-button"
          disabled={!form.targetId || actionId === `refresh:${form.targetId}`}
          onClick={onRefreshTarget}
          type="button"
        >
          <RefreshCw size={14} />
          {actionId === `refresh:${form.targetId}` ? t("action.loading") : t("action.refresh")}
        </button>
        {message ? <span className="capability-message">{message}</span> : null}
      </div>

      <div className="capability-layout">
        <div className="capability-catalog">
          <div className="table-wrap">
            <table className="capability-table">
              <thead>
                <tr>
                  <th>{t("table.capability")}</th>
                  <th>{t("table.target")}</th>
                  <th>{t("table.action")}</th>
                  <th>{t("table.risk")}</th>
                  <th>{t("table.status")}</th>
                  <th>{t("table.grants")}</th>
                  <th>{t("table.action")}</th>
                </tr>
              </thead>
              <tbody>
                {visibleCapabilities.length === 0 ? (
                  <tr>
                    <td colSpan={7}>
                      <EmptyRow title={t("empty.capabilities.title")} detail={t("empty.capabilities.detail")} />
                    </td>
                  </tr>
                ) : null}
                {visibleCapabilities.map((capability) => {
                  const entitlementIds = entitlementIdsByCapability[capability.id] ?? [];
                  const workspaceIds = entitlementIds.flatMap((id) => workspaceIdsByEntitlement[id] ?? []);
                  const instanceCount = workspaceIds.reduce(
                    (total, id) => total + (instancesByWorkspaceAssignment[id]?.length ?? 0),
                    0
                  );
                  return (
                    <tr key={capability.id}>
                      <td>
                        <strong>{capability.displayName || capability.key}</strong>
                        <span>{dataScopeText(capability.dataScopes) || capability.description || capability.key}</span>
                      </td>
                      <td>{agentNames[capability.targetId] ?? capability.targetId}</td>
                      <td><Badge tone={capability.action === "delete" || capability.action === "admin" ? "danger" : capability.action === "export" ? "warning" : "info"}>{translatedValue(t, capability.action)}</Badge></td>
                      <td><Badge tone={riskTone(capability.riskLevel)}>{translatedValue(t, capability.riskLevel)}</Badge></td>
                      <td><Badge tone={capabilityStatusTone(capability.discoveryStatus)}>{capabilityDiscoveryStatusLabel(capability.discoveryStatus, t)}</Badge></td>
                      <td>
                        <strong>{entitlementIds.length}/{workspaceIds.length}/{instanceCount}</strong>
                        <span>{t("detail.tenantWorkspaceInstance")}</span>
                      </td>
                      <td>
                        {capability.discoveryStatus === "approved" ? (
                          <span className="muted-action">{t("status.capabilityApproved")}</span>
                        ) : (
                          <button
                            className="table-action"
                            disabled={actionId === capability.id}
                            onClick={() => onApprove(capability)}
                            type="button"
                          >
                            <ShieldCheck size={13} />
                            {actionId === capability.id ? t("action.approving") : t("action.approve")}
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>

        <form className="control-form capability-grant-form" onSubmit={onCreateGrantChain}>
          <div className="form-row">
            <label>{t("form.tenant")}<input required value={form.tenantId} onChange={(event) => onChange({ ...form, tenantId: event.target.value })} /></label>
            <label>{t("form.workspace")}<input required value={form.workspaceId} onChange={(event) => onChange({ ...form, workspaceId: event.target.value })} /></label>
          </div>
          <label>
            {t("form.capability")}
            <select required value={form.capabilityId} onChange={(event) => handleCapabilityChange(event.target.value)}>
              <option value="">{t("form.selectCapability")}</option>
              {visibleCapabilities.map((capability) => (
                <option key={capability.id} value={capability.id}>
                  {capability.key}
                </option>
              ))}
            </select>
          </label>
          <div className="form-row">
            <label>
              {t("form.callerInstance")}
              <select required value={form.callerInstanceId} onChange={(event) => onChange({ ...form, callerInstanceId: event.target.value })}>
                <option value="">{t("form.selectCaller")}</option>
                {agents.map((agent) => (
                  <option key={agent.id} value={agent.id}>
                    {agent.name}
                  </option>
                ))}
              </select>
            </label>
            <label>{t("form.subjectSelector")}<input placeholder={t("form.subjectSelectorPlaceholder")} value={form.subjectSelector} onChange={(event) => onChange({ ...form, subjectSelector: event.target.value })} /></label>
          </div>
          <div className="capability-scope-strip">
            <span>{selectedCapability?.sensitivity ?? t("text.sensitivity")}</span>
            <span>{selectedCapability?.riskLevel ?? t("text.risk")}</span>
            <span>{dataScopeText(selectedCapability?.dataScopes) || t("text.noDataScope")}</span>
          </div>
          <FormFooter
            message=""
            submitLabel={actionId === `grant:${form.capabilityId}` ? t("action.loading") : t("action.grantChain")}
          />
        </form>

        <div className="assignment-list">
          {tenantEntitlements.length === 0 ? (
            <EmptyRow title={t("empty.grantChains.title")} detail={t("empty.grantChains.assignmentDetail")} />
          ) : null}
          {tenantEntitlements.map((entitlement) => {
            const capability = capabilities.find((item) => item.id === entitlement.capabilityId);
            const children = workspaceAssignments.filter((item) => item.tenantEntitlementId === entitlement.id);
            const instanceCount = children.reduce(
              (total, item) => total + instanceAssignments.filter((instance) => instance.workspaceAssignmentId === item.id).length,
              0
            );
            return (
              <article className="assignment-row" key={entitlement.id}>
                <div>
                  <strong>{capability?.key ?? entitlement.capabilityId}</strong>
                  <span>{entitlement.tenantId} · {policyEffectLabel(entitlement.effect, t)} · {translatedValue(t, entitlement.status)}</span>
                </div>
                <div className="assignment-metrics">
                  <span>{children.length} {t("text.workspaces")}</span>
                  <span>{instanceCount} {t("text.callers")}</span>
                </div>
              </article>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function ManagementAuditTable({ events, t }: { events: AuditEvent[]; t: Translator }) {
  return (
    <div className="table-wrap">
      <table className="audit-table">
        <thead>
          <tr>
            <th>{t("table.time")}</th>
            <th>{t("table.action")}</th>
            <th>{t("table.resource")}</th>
            <th>{t("table.actor")}</th>
            <th>{t("table.version")}</th>
            <th>{t("table.summary")}</th>
          </tr>
        </thead>
        <tbody>
          {events.length === 0 ? (
            <tr>
              <td colSpan={6}>
                <EmptyRow title={t("empty.managementAudit.title")} detail={t("empty.managementAudit.detail")} />
              </td>
            </tr>
          ) : null}
          {events.map((event) => (
            <tr key={event.id}>
              <td>{formatDate(event.createdAt)}</td>
              <td><Badge tone={auditTone(event.action)}>{event.action}</Badge></td>
              <td>
                <strong>{event.resourceType}</strong>
                <span>{event.resourceId}</span>
              </td>
              <td>{event.actor || "-"}</td>
              <td>{auditCredentialVersion(event)}</td>
              <td>{event.summary || "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Badge({ tone, children }: { tone: Tone; children: ReactNode }) {
  return <span className={`badge tone-${tone}`}>{children}</span>;
}

function EmptyRow({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="empty-row">
      <strong>{title}</strong>
      <span>{detail}</span>
    </div>
  );
}

function IconMore({ title = "More" }: { title?: string }) {
  return (
    <button className="icon-button compact" title={title} type="button">
      <MoreHorizontal size={16} />
    </button>
  );
}

function IconOpen({ title = "Open" }: { title?: string }) {
  return (
    <button className="icon-button compact" title={title} type="button">
      <ExternalLink size={15} />
    </button>
  );
}

function configText(agent: Agent, key: string) {
  const value = agent.channelConfig?.[key];
  return typeof value === "string" ? value : "";
}

function normalizedScope(scope: ManagementScope): ManagementScope {
  return {
    tenantId: scope.tenantId.trim() || defaultManagementScope.tenantId,
    workspaceId: scope.workspaceId.trim() || defaultManagementScope.workspaceId
  };
}

function accessDecisionExplainRequestComplete(request: AccessDecisionExplainRequest) {
  return Boolean(
    request.tenantId.trim() &&
      request.workspaceId.trim() &&
      request.callerInstanceId.trim() &&
      request.targetId.trim() &&
      request.capabilityId.trim()
  );
}

function accessDecisionOutcomeTone(outcome: AccessDecisionExplainResult["outcome"]): Tone {
  return outcome === "allowed" ? "success" : "danger";
}

function accessDecisionOutcomeLabel(outcome: AccessDecisionExplainResult["outcome"], t: Translator) {
  return outcome === "allowed" ? t("text.decisionAllowed") : t("text.decisionDenied");
}

function accessDecisionEvidenceTone(status: string): Tone {
  if (status === "matched") return "success";
  if (status === "blocking" || status === "missing" || status === "mismatch") return "danger";
  if (status === "not_approved" || status === "inactive") return "warning";
  return "neutral";
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false
  });
}

function agentNameMap(agents: Agent[]) {
  return agents.reduce<Record<string, string>>((acc, agent) => {
    acc[agent.id] = agent.name;
    return acc;
  }, {});
}

function auditCredentialVersion(event: AuditEvent) {
  const value = event.metadata?.credentialVersion;
  if (typeof value === "number" || typeof value === "string") return String(value);
  return "-";
}

function channelLabel(channelType: string, channelLabels: Record<string, string>, t: Translator) {
  return t(`value.${channelType}`, channelLabels[channelType] ?? channelType);
}

function evidenceStatusLabel(status: EvidenceRun["status"], t: Translator) {
  if (status === "passed") return t("status.evidencePassed");
  if (status === "failed") return t("status.evidenceFailed");
  return t("status.evidenceWarning");
}

function capabilityDiscoveryStatusLabel(status: Capability["discoveryStatus"], t: Translator) {
  if (status === "approved") return t("status.capabilityApproved");
  if (status === "deprecated") return t("status.capabilityDeprecated");
  if (status === "removed") return t("status.capabilityRemoved");
  return t("status.capabilityPendingReview");
}

function policyRetryText(policy: RoutePolicy, t: Translator) {
  if (!policy.retry) return t("text.targetRetry");
  const statuses = policy.retry.statusCodes.length > 0 ? policy.retry.statusCodes.join("/") : t("text.retryNone");
  return `retry ${policy.retry.maxAttempts}x ${policy.retry.backoffMs}ms ${statuses}`;
}

function permissionPackageTemplateName(template: PermissionPackageTemplate, t: Translator) {
  return t(`permissionPackage.${template.id}.name`, template.name);
}

function permissionPackageTemplateSummary(template: PermissionPackageTemplate, t: Translator) {
  return t(`permissionPackage.${template.id}.summary`, template.summary);
}

function permissionSimulationReason(row: PermissionPackageSimulationRow, t: Translator) {
  if (!row.reasonKey) return row.reason;
  const values = Object.entries(row.reasonValues ?? {}).reduce<Record<string, string>>((acc, [key, value]) => {
    if (key === "action") {
      acc[key] = translatedValue(t, value);
    } else if (key === "packageId") {
      acc.package = t(`permissionPackage.${value}.name`, value);
    } else {
      acc[key] = value;
    }
    return acc;
  }, {});
  return tx(t, row.reasonKey, values);
}

function permissionReadinessMessages(readiness: PermissionPackageDraft["readiness"], t: Translator) {
  const fieldLabels: Record<string, string> = {
    callerInstanceId: t("form.callerInstance"),
    targetId: t("form.target"),
    tenantId: t("form.tenantId"),
    workspaceId: t("form.workspaceId")
  };
  return [
    ...readiness.missingFields.map((field) =>
      tx(t, "message.permissionPackageMissingField", { field: fieldLabels[field] ?? field })
    ),
    ...readiness.warnings.map((warning) =>
      warning === "No matching allowed capabilities for the selected target."
        ? t("message.noMatchingAllowedCapabilities")
        : warning
    )
  ];
}

function permissionPolicyGateMessages(policyGate: PermissionPackageDraft["policyGate"], t: Translator) {
  if (policyGate.canApplyDirectly) {
    return [t("text.policyGateDirectDetail")];
  }
  return policyGate.reasons.length > 0
    ? policyGate.reasons.map((reason) => permissionPolicyReasonMessage(reason, t))
    : [t("text.policyGateApprovalDetail")];
}

function permissionPolicyReasonMessage(
  reason: PermissionPackageDraft["policyGate"]["reasons"][number],
  t: Translator,
) {
  if (!reason.reasonKey) return reason.message;
  const values = Object.entries(reason.reasonValues ?? {}).reduce<Record<string, string>>((acc, [key, value]) => {
    if (key === "action" || key === "risk" || key === "sensitivity") {
      acc[key] = translatedValue(t, value);
    } else {
      acc[key] = value;
    }
    return acc;
  }, {});
  return tx(t, reason.reasonKey, values);
}

function permissionApprovalStatusLabel(status: PermissionPackageApprovalRequest["status"], t: Translator) {
  if (status === "approved") return t("status.approvalApproved");
  if (status === "rejected") return t("status.approvalRejected");
  return t("status.approvalPending");
}

function permissionApprovalStatusTone(status: PermissionPackageApprovalRequest["status"]): Tone {
  if (status === "approved") return "success";
  if (status === "rejected") return "danger";
  return "warning";
}

function permissionPackageApprovalRouteLabel(request: PermissionPackageApprovalRequest) {
  return `${request.tenantId} / ${request.workspaceId} / ${request.callerInstanceId}`;
}

function permissionImpactObjectLabel(type: string, t: Translator) {
  if (type === "tenant_entitlement") return t("text.tenantEntitlement");
  if (type === "workspace_assignment") return t("text.workspaceAssignment");
  if (type === "instance_assignment") return t("text.instanceAssignment");
  return type.replaceAll("_", " ");
}

function permissionRemediationTargetLabel(type: string, t: Translator) {
  if (type === "capability") return t("text.capability");
  if (type === "access_decision") return t("text.finalAccessVerification");
  return permissionImpactObjectLabel(type, t);
}

function permissionRemediationActionTone(action: PermissionPackageRemediationAction["action"]): Tone {
  if (action === "disable") return "warning";
  if (action === "investigate") return "danger";
  if (action === "verify") return "info";
  return "neutral";
}

function permissionRollbackStepLabel(step: string, t: Translator) {
  switch (step) {
    case "Review capability discovery status manually; shared capabilities are not automatically downgraded by rollback.":
      return t("rollbackStep.capabilityManualReview");
    case "Disable recorded instance assignments before workspace assignments.":
      return t("rollbackStep.disableInstanceAssignments");
    case "Disable recorded workspace assignments before tenant entitlements.":
      return t("rollbackStep.disableWorkspaceAssignments");
    case "Disable recorded tenant entitlements and then verify effective access decisions.":
      return t("rollbackStep.disableTenantEntitlements");
    default:
      return step;
  }
}

function permissionImpactBlockerLabels(blockerCodes: string[], blockers: string[], t: Translator) {
  if (blockerCodes.length > 0) {
    return blockerCodes.map((code) => t(`blocker.${code}`, code));
  }
  return blockers;
}

function permissionImpactStatusLabel(status: string, t: Translator) {
  if (status === "approved") return t("status.capabilityApproved");
  if (status === "deprecated") return t("status.capabilityDeprecated");
  if (status === "pending_review") return t("status.capabilityPendingReview");
  if (status === "removed") return t("status.capabilityRemoved");
  return translatedValue(t, status);
}

function permissionImpactStatusTone(status: string): Tone {
  if (status === "enabled" || status === "approved") return "success";
  if (status === "missing" || status === "removed") return "danger";
  if (status === "disabled" || status === "deprecated" || status === "pending_review") return "warning";
  return "neutral";
}

function mergePermissionPackageApprovalRequests(
  next: PermissionPackageApprovalRequest[],
  current: PermissionPackageApprovalRequest[]
) {
  const rows = new Map<string, PermissionPackageApprovalRequest>();
  [...next, ...current].forEach((request) => rows.set(request.id, request));
  return Array.from(rows.values());
}

function matchingPermissionPackageApprovalRequest(
  requests: PermissionPackageApprovalRequest[],
  draft: PermissionPackageDraft
) {
  const matching = requests.filter((request) => permissionPackageApprovalRequestMatchesDraft(request, draft));
  return matching.find((request) => request.status === "approved")
    ?? matching.find((request) => request.status === "pending")
    ?? matching.find((request) => request.status === "rejected")
    ?? null;
}

function permissionPackageApprovalRequestMatchesDraft(
  request: PermissionPackageApprovalRequest,
  draft: PermissionPackageDraft
) {
  return request.draftId === draft.id
    && request.templateId === draft.template.id
    && request.templateVersion === draft.template.version
    && request.policyVersion === draft.policyGate.policyVersion
    && request.tenantId === draft.input.tenantId
    && request.workspaceId === draft.input.workspaceId
    && request.targetId === draft.input.targetId
    && request.callerInstanceId === draft.input.callerInstanceId
    && (request.subjectSelector ?? "") === (draft.input.subjectSelector ?? "")
    && (request.requestText ?? "") === draft.input.requestText
    && (request.region ?? "") === draft.input.region
    && sameStringSet(request.allowedCapabilityIds, draft.allowedCapabilities.map((capability) => capability.id))
    && sameStringSet(request.allowedCapabilityKeys, draft.allowedCapabilities.map((capability) => capability.key))
    && sameDataScopes(request.dataScopes ?? [], draft.dataScopes);
}

function sameStringSet(left: string[], right: string[]) {
  if (left.length !== right.length) return false;
  const leftSorted = [...left].sort();
  const rightSorted = [...right].sort();
  return leftSorted.every((value, index) => value === rightSorted[index]);
}

function sameDataScopes(left: DataScope[], right: DataScope[]) {
  if (left.length !== right.length) return false;
  return left.every((scope, index) => JSON.stringify(scope) === JSON.stringify(right[index]));
}

function auditTone(action: string): Tone {
  if (action.includes("delete") || action.includes("revoke") || action.includes("disable")) return "danger";
  if (action.includes("rotate") || action.includes("credentials")) return "warning";
  if (action.includes("create") || action.includes("enable")) return "success";
  return "info";
}

function evidenceDuration(run: EvidenceRun) {
  if (!run.completedAt) return 0;
  const started = new Date(run.startedAt).getTime();
  const completed = new Date(run.completedAt).getTime();
  if (Number.isNaN(started) || Number.isNaN(completed)) return 0;
  return Math.max(0, completed - started);
}

function formatDuration(durationMs: number) {
  if (durationMs < 1000) return `${durationMs}ms`;
  return `${(durationMs / 1000).toFixed(1)}s`;
}

function metricRatio(metric: SystemMetric) {
  if (metric.unit === "%") return clamp(metric.value, 0, 100);
  if (metric.unit === "ms") return clamp(100 - metric.value, 8, 100);
  return clamp(metric.value * 10, 8, 100);
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

function toneFromStatus(status: string): Tone {
  if (status === "passed") return "success";
  if (status === "warning") return "warning";
  if (status === "failed") return "danger";
  return "info";
}

function shallowEqualCapabilityForm(
  left: typeof defaultCapabilityGrantForm,
  right: typeof defaultCapabilityGrantForm
) {
  return (
    left.callerInstanceId === right.callerInstanceId &&
    left.capabilityId === right.capabilityId &&
    left.subjectSelector === right.subjectSelector &&
    left.targetId === right.targetId &&
    left.tenantId === right.tenantId &&
    left.workspaceId === right.workspaceId
  );
}

function shallowEqualAiAdminForm(
  left: PermissionPackageDraftInput,
  right: PermissionPackageDraftInput
) {
  return (
    left.callerInstanceId === right.callerInstanceId &&
    left.region === right.region &&
    left.requestText === right.requestText &&
    left.subjectSelector === right.subjectSelector &&
    left.targetId === right.targetId &&
    left.templateId === right.templateId &&
    left.tenantId === right.tenantId &&
    left.workspaceId === right.workspaceId
  );
}

function mergeCapabilitiesForTarget(existing: Capability[], refreshed: Capability[], targetId: string) {
  return [
    ...existing.filter((capability) => capability.targetId !== targetId),
    ...refreshed
  ].sort((left, right) => `${left.targetId}:${left.key}`.localeCompare(`${right.targetId}:${right.key}`));
}

function shouldUseLocalCapabilityFallback(error: unknown, data: ConsoleData | null) {
  if (error instanceof TypeError) return true;
  return Boolean(data && (!data.loadedFromApi || !data.capabilitiesLoadedFromApi || !data.capabilityAssignmentsLoadedFromApi));
}

function appendLocalCapabilityGrantChain(
  current: ConsoleData,
  capability: Capability,
  form: typeof defaultCapabilityGrantForm,
  dataScopes: DataScope[]
): ConsoleData {
  const now = new Date().toISOString();
  const tenantId = form.tenantId.trim() || defaultManagementScope.tenantId;
  const workspaceId = form.workspaceId.trim() || defaultManagementScope.workspaceId;
  const callerInstanceId = form.callerInstanceId.trim();
  const subjectSelector = form.subjectSelector.trim() || undefined;

  const existingEntitlement = current.tenantEntitlements.find(
    (item) => item.tenantId === tenantId && item.targetId === capability.targetId && item.capabilityId === capability.id
  );
  const entitlement: TenantEntitlement = existingEntitlement
    ? {
        ...existingEntitlement,
        dataScopes,
        effect: "allow",
        status: "enabled",
        updatedAt: now
      }
    : {
        id: nextLocalId("ent", [capability.id, tenantId], current.tenantEntitlements.map((item) => item.id)),
        tenantId,
        targetId: capability.targetId,
        capabilityId: capability.id,
        effect: "allow",
        dataScopes,
        status: "enabled",
        priority: 50,
        createdAt: now,
        updatedAt: now
      };

  const tenantEntitlements = existingEntitlement
    ? current.tenantEntitlements.map((item) => (item.id === entitlement.id ? entitlement : item))
    : [...current.tenantEntitlements, entitlement];

  const existingWorkspaceAssignment = current.workspaceAssignments.find(
    (item) => item.tenantEntitlementId === entitlement.id && item.workspaceId === workspaceId
  );
  const workspaceAssignment: WorkspaceAssignment = existingWorkspaceAssignment
    ? {
        ...existingWorkspaceAssignment,
        dataScopes,
        effect: "allow",
        status: "enabled",
        updatedAt: now
      }
    : {
        id: nextLocalId(
          "wsa",
          [capability.id, tenantId, workspaceId],
          current.workspaceAssignments.map((item) => item.id)
        ),
        tenantEntitlementId: entitlement.id,
        tenantId,
        workspaceId,
        effect: "allow",
        dataScopes,
        status: "enabled",
        createdAt: now,
        updatedAt: now
      };

  const workspaceAssignments = existingWorkspaceAssignment
    ? current.workspaceAssignments.map((item) => (item.id === workspaceAssignment.id ? workspaceAssignment : item))
    : [...current.workspaceAssignments, workspaceAssignment];

  const existingInstanceAssignment = current.instanceAssignments.find(
    (item) => item.workspaceAssignmentId === workspaceAssignment.id && item.callerInstanceId === callerInstanceId
  );
  const instanceAssignment: InstanceAssignment = existingInstanceAssignment
    ? {
        ...existingInstanceAssignment,
        dataScopes,
        effect: "allow",
        status: "enabled",
        subjectSelector,
        updatedAt: now
      }
    : {
        id: nextLocalId(
          "ina",
          [capability.id, tenantId, workspaceId, callerInstanceId],
          current.instanceAssignments.map((item) => item.id)
        ),
        workspaceAssignmentId: workspaceAssignment.id,
        tenantId,
        workspaceId,
        callerInstanceId,
        subjectSelector,
        effect: "allow",
        dataScopes,
        status: "enabled",
        createdAt: now,
        updatedAt: now
      };

  const instanceAssignments = existingInstanceAssignment
    ? current.instanceAssignments.map((item) => (item.id === instanceAssignment.id ? instanceAssignment : item))
    : [...current.instanceAssignments, instanceAssignment];

  return {
    ...current,
    tenantEntitlements,
    workspaceAssignments,
    instanceAssignments,
    capabilityAssignmentsLoadedFromApi: false
  };
}

function nextLocalId(prefix: string, parts: string[], existing: string[]) {
  const base = `${prefix}_${parts.map(safeIdPart).filter(Boolean).join("_") || "local"}`;
  if (!existing.includes(base)) return base;
  let counter = 2;
  while (existing.includes(`${base}_${counter}`)) counter += 1;
  return `${base}_${counter}`;
}

function safeIdPart(value: string) {
  return value.trim().replace(/[^a-zA-Z0-9]+/g, "_").replace(/^_+|_+$/g, "").toLowerCase();
}

function dataScopeText(scopes?: DataScope[]) {
  if (!scopes || scopes.length === 0) return "";
  const labels = scopes
    .map((scope) =>
      [scope.dataDomain, scope.dataset, scope.schema, scope.table, scope.field, scope.classification]
        .filter(Boolean)
        .join("/")
    )
    .filter(Boolean);
  if (labels.length === 0) return "";
  return labels.length > 2 ? `${labels.slice(0, 2).join(", ")} +${labels.length - 2}` : labels.join(", ");
}

function riskTone(risk: Capability["riskLevel"]): Tone {
  if (risk === "critical" || risk === "high") return "danger";
  if (risk === "medium") return "warning";
  return "success";
}

function capabilityStatusTone(status: Capability["discoveryStatus"]): Tone {
  if (status === "approved") return "success";
  if (status === "pending_review") return "warning";
  if (status === "removed") return "danger";
  return "neutral";
}

function coreJourneyStatusTone(status: CoreJourneyStepStatus): Tone {
  if (status === "complete") return "success";
  if (status === "partial") return "warning";
  return "neutral";
}

function coreJourneyStatusLabel(status: CoreJourneyStepStatus, t: Translator) {
  if (status === "complete") return t("status.stepComplete");
  if (status === "partial") return t("status.stepPartial");
  return t("status.stepMissing");
}

function aiAdminApprovalJourneyStatusTone(status: AiAdminApprovalJourneyStepStatus): Tone {
  if (status === "complete") return "success";
  if (status === "partial") return "warning";
  return "neutral";
}

function aiAdminApprovalJourneyStatusLabel(status: AiAdminApprovalJourneyStepStatus, t: Translator) {
  if (status === "complete") return t("status.stepComplete");
  if (status === "partial") return t("status.stepPartial");
  return t("status.stepMissing");
}

function aiAdminApprovalReadinessStatusTone(status: AiAdminApprovalReadinessStatus): Tone {
  if (status === "ok") return "success";
  if (status === "warning") return "warning";
  if (status === "error") return "danger";
  return "neutral";
}

function aiAdminApprovalReadinessStatusLabel(status: AiAdminApprovalReadinessStatus, t: Translator) {
  if (status === "ok") return t("status.preflightOk");
  if (status === "warning") return t("status.preflightWarning");
  if (status === "error") return t("status.preflightError");
  return t("status.preflightPending");
}

function preflightTone(status: CoreJourneyPreflightStatus): Tone {
  if (status === "ok") return "success";
  if (status === "warning") return "warning";
  if (status === "error") return "danger";
  return "neutral";
}

function preflightStatusLabel(status: CoreJourneyPreflightStatus, t: Translator) {
  if (status === "ok") return t("status.preflightOk");
  if (status === "warning") return t("status.preflightWarning");
  if (status === "error") return t("status.preflightError");
  return t("status.preflightPending");
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
      arguments: {
        query: "acme",
        status: "triaged",
        ticketId: "T-1000"
      },
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
    .map((tool) => {
      if (!tool || typeof tool !== "object") return "";
      const name = "name" in tool ? (tool as { name?: unknown }).name : undefined;
      return typeof name === "string" ? name : "";
    })
    .filter(Boolean);
}

export default App;
