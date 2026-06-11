import { useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from "react";
import {
  Activity,
  Boxes,
  CheckCircle2,
  CircleDot,
  ClipboardCheck,
  Copy,
  DatabaseZap,
  Download,
  ExternalLink,
  FileSearch,
  Gauge,
  KeyRound,
  Layers3,
  LockKeyhole,
  MoreHorizontal,
  Network,
  RefreshCw,
  Route,
  ServerCog,
  ShieldCheck,
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
  fetchPermissionPackageAccessSubjects,
  fetchPermissionPackageApplicationHealth,
  fetchPermissionPackageApplicationImpact,
  fetchPermissionPackageApprovalRequests,
  fetchPermissionPackageProductionEvidenceReport,
  fetchPermissionPackageProductionReadiness,
  fetchPermissionPackageTemplates,
  isApiCompatibilityFallbackError,
  loadConsoleData,
  loadTenantAccessProfile,
  preflightPermissionPackage,
  previewPermissionPackageWorkbench,
  rejectPermissionPackageApprovalRequest,
  refreshTargetCapabilities,
  rotateAgentCredentials,
  updateAgent,
  updateCapability,
  withdrawPermissionPackageApprovalRequest
} from "./api";
import { parseAccessProfileTraceLimit } from "./accessProfile";
import {
  runtimeEvidenceMetric
} from "./consoleMetrics";
import {
  accessTraceReasonLabel,
  agentNameMap,
  capabilityDisplayName,
  capabilityKeyDisplayName,
  capabilityDiscoveryStatusLabel,
  capabilityStatusTone,
  dataScopeText,
  formatDate,
  permissionEntityDisplayName,
  policyEffectLabel,
  readableIdentifierLabel,
  riskTone,
  translatedValue,
  type Tone,
  type Translator
} from "./consolePresenters";
import {
  accessSubjectOptions,
  normalizeAccessSubjectOptions,
  type AccessSubjectOption
} from "./accessSubjects";
import {
  createTranslator,
  resolveInitialLanguage,
  type Language
} from "./i18n";
import {
  defaultNavKey,
  navHashFor,
  navKeyFromHash,
  navGroups,
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
  summarizeAiAdminGoLiveReadiness,
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
  buildAiAdminProductionConsoleSummary,
  type AiAdminProductionConsoleStatus,
  type AiAdminProductionConsoleSummary
} from "./aiAdminProductionConsole";
import {
  createPermissionPackageDraft,
  defaultPermissionPackageDraftInput,
  permissionPackageTemplates,
  subjectIdExampleFromSelector,
  type PermissionPackageApplyInput,
  type PermissionPackageApplyPreflight,
  type PermissionPackageApplyPreflightCheck,
  type PermissionPackageApplication,
  type PermissionPackageApplicationHealth,
  type PermissionPackageApplicationHealthRow,
  type PermissionPackageApplicationHealthStatus,
  type PermissionPackageApplicationImpact,
  type PermissionPackageApprovalRequest,
  type PermissionPackageDraft,
  type PermissionPackageDraftInput,
  type PermissionPackageProductionEvidenceReport,
  type PermissionPackageProductionReadiness,
  type PermissionPackageProductionReadinessCheck,
  type PermissionPackageProductionReadinessFilter,
  type PermissionPackageProductionReadinessStatus,
  type PermissionPackageRemediationAction,
  type PermissionPackageSimulationRow,
  type PermissionPackageTemplate,
  type PermissionPackageWorkbenchPreview,
  type PermissionPackageWorkbenchStep
} from "./permissionPackages";
import {
  currentPermissionRequestWizardStep,
  type PermissionRequestWizardStep
} from "./permissionRequestJourney";
import { parseRetryFields } from "./retryForm";
import { AiAdminPermissionWorkbench } from "./components/AiAdminPermissionWorkbench";
import { CapabilityGovernanceView, type CapabilityGrantForm } from "./components/CapabilityGovernanceView";
import { TechnicalId } from "./components/TechnicalId";
import { TenantAccessProfileView } from "./components/TenantAccessProfileView";
import { Badge, EmptyRow } from "./components/ui";
import type {
  AccessProfileFilters,
  AccessProfileHandoffContext,
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
  Tenant,
  TenantAccessProfileData,
  TenantEntitlement,
  TraceDecision,
  TraceEvent,
  TraceFilters,
  WorkspaceAssignment
} from "./types";

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

const defaultManagementScope: ManagementScope = {
  tenantId: "default",
  workspaceId: "workspace-sandbox"
};
const languageStorageKey = "agent-harbor-language";
const approvalResolveCooldownMs = 1200;
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
const defaultCapabilityGrantForm: CapabilityGrantForm = {
  callerInstanceId: "",
  capabilityId: "",
  subjectSelector: "user:support-*",
  targetId: "",
  tenantId: defaultManagementScope.tenantId,
  workspaceId: defaultManagementScope.workspaceId
};
const defaultAiAdminForm: PermissionPackageDraftInput = {
  ...defaultPermissionPackageDraftInput,
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

function initialNavKey(): NavKey {
  if (typeof window === "undefined") {
    return defaultNavKey;
  }
  return navKeyFromHash(window.location.hash) ?? defaultNavKey;
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
      return ShieldCheck;
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

type LocalizedMessage =
  | {
      key: string;
      params?: Record<string, string | number>;
    }
  | {
      render: (t: Translator, language: Language) => string;
    };

function localizedMessageText(message: LocalizedMessage | null, t: Translator, language: Language) {
  if (!message) return "";
  if ("render" in message) return message.render(t, language);
  return message.params ? tx(t, message.key, message.params) : t(message.key);
}

function localizedErrorMessageState(error: unknown, fallbackKey: string): LocalizedMessage {
  return {
    render: (t, language) => localizedErrorMessage(t, language, error, fallbackKey)
  };
}

function isAbortError(error: unknown) {
  return error instanceof Error && error.name === "AbortError";
}

function localizedErrorMessage(t: Translator, language: Language, error: unknown, fallbackKey: string) {
  const fallback = t(fallbackKey);
  if (!(error instanceof Error) || !error.message.trim()) return fallback;
  if (language === "en" || /[\u4e00-\u9fa5]/.test(error.message)) {
    return error.message;
  }
  return fallback;
}

function retryFieldValidationMessage(message: string, t: Translator) {
  if (message === "Retry attempts must be an integer between 1 and 4.") {
    return t("message.validationRetryAttempts");
  }
  if (message === "Retry backoff must be an integer between 0 and 1000 ms.") {
    return t("message.validationRetryBackoff");
  }
  return message;
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

function traceRouteBusinessLabel(trace: TraceEvent, t: Translator) {
  if (trace.routeType === "mcp") {
    if (trace.routeKey === "tools/call") return t("traceRoute.mcpToolsCall");
    if (trace.routeKey === "tools/list") return t("traceRoute.mcpToolsList");
    if (trace.routeKey === "initialize") return t("traceRoute.mcpInitialize");
    if (!trace.routeKey) return t("traceRoute.mcpDefault");
  }
  return trace.routeKey ? readableIdentifierLabel(trace.routeKey) : t("text.traceDefaultRoute");
}

function App() {
  const approvalCreateInFlightRef = useRef(false);
  const approvalResolveBlockedRef = useRef(false);
  const approvalResolveCooldownTimerRef = useRef<number | null>(null);
  const [activeNav, setActiveNav] = useState<NavKey>(initialNavKey);
  const [connectionMenuOpen, setConnectionMenuOpen] = useState(false);
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
  const [accessProfileHandoffContext, setAccessProfileHandoffContext] = useState<AccessProfileHandoffContext | null>(null);
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
  const [aiAdminMessage, setAiAdminMessage] = useState<LocalizedMessage | null>(null);
  const [aiAdminApplying, setAiAdminApplying] = useState(false);
  const [aiAdminTemplates, setAiAdminTemplates] = useState<PermissionPackageTemplate[]>(permissionPackageTemplates);
  const [aiAdminAccessSubjects, setAiAdminAccessSubjects] = useState<AccessSubjectOption[]>(accessSubjectOptions);
  const [aiAdminServerDraft, setAiAdminServerDraft] = useState<PermissionPackageDraft | null>(null);
  const [aiAdminWorkbenchPreview, setAiAdminWorkbenchPreview] = useState<PermissionPackageWorkbenchPreview | null>(null);
  const [aiAdminNewDraftMode, setAiAdminNewDraftMode] = useState(false);
  const [aiAdminApplication, setAiAdminApplication] = useState<PermissionPackageApplication | null>(null);
  const [aiAdminApplicationHealth, setAiAdminApplicationHealth] =
    useState<PermissionPackageApplicationHealth | null>(null);
  const [aiAdminApplicationHealthLoading, setAiAdminApplicationHealthLoading] = useState(false);
  const [aiAdminApplicationHealthMessage, setAiAdminApplicationHealthMessage] = useState("");
  const [aiAdminApplyPreflight, setAiAdminApplyPreflight] = useState<PermissionPackageApplyPreflight | null>(null);
  const [aiAdminApplyPreflightLoading, setAiAdminApplyPreflightLoading] = useState(false);
  const [aiAdminApplyPreflightMessage, setAiAdminApplyPreflightMessage] = useState("");
  const [aiAdminApplicationImpact, setAiAdminApplicationImpact] =
    useState<PermissionPackageApplicationImpact | null>(null);
  const [aiAdminApplicationImpactLoading, setAiAdminApplicationImpactLoading] = useState(false);
  const [aiAdminApplicationImpactMessage, setAiAdminApplicationImpactMessage] = useState("");
  const [aiAdminProductionReadiness, setAiAdminProductionReadiness] =
    useState<PermissionPackageProductionReadiness | null>(null);
  const [aiAdminProductionReadinessLoading, setAiAdminProductionReadinessLoading] = useState(false);
  const [aiAdminProductionEvidenceExporting, setAiAdminProductionEvidenceExporting] = useState(false);
  const [aiAdminProductionReadinessMessage, setAiAdminProductionReadinessMessage] = useState("");
  const [aiAdminApprovalRequests, setAiAdminApprovalRequests] = useState<PermissionPackageApprovalRequest[]>([]);
  const [aiAdminApprovalAction, setAiAdminApprovalAction] = useState<"" | "create" | "approve" | "reject" | "withdraw">("");
  const [aiAdminApprovalResolutionCoolingDown, setAiAdminApprovalResolutionCoolingDown] = useState(false);
  const [aiAdminApprovalReviewer, setAiAdminApprovalReviewer] = useState("Security Reviewer");
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
  const [aiAdminApprovalJourneyAccessProfile, setAiAdminApprovalJourneyAccessProfile] =
    useState<TenantAccessProfileData | null>(null);
  const [aiAdminApprovalJourneyApprovalRequest, setAiAdminApprovalJourneyApprovalRequest] =
    useState<PermissionPackageApprovalRequest | null>(null);
  const [aiAdminApprovalReadiness, setAiAdminApprovalReadiness] =
    useState<AiAdminApprovalReadinessState>(defaultAiAdminApprovalReadiness);
  const [aiAdminApprovalReadinessChecking, setAiAdminApprovalReadinessChecking] = useState(false);
  const [aiAdminApprovalReadinessMessage, setAiAdminApprovalReadinessMessage] = useState("");
  const t = useMemo(() => createTranslator(language), [language]);
  const renderedAiAdminMessage = localizedMessageText(aiAdminMessage, t, language);

  useEffect(() => {
    void refresh();
  }, []);

  useEffect(() => () => {
    if (approvalResolveCooldownTimerRef.current !== null) {
      window.clearTimeout(approvalResolveCooldownTimerRef.current);
    }
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
    setConnectionMenuOpen(false);
  }, [activeNav]);

  const shouldLoadAiAdminCatalog = activeNav === "ai-admin" || activeNav === "evidence";

  useEffect(() => {
    if (shouldLoadAiAdminCatalog) {
      void refreshAiAdminCatalog();
      void refreshAiAdminApprovalReadiness();
    }
  }, [shouldLoadAiAdminCatalog]);

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

  const shouldLoadAiAdminWorkbenchPreview = activeNav === "ai-admin" || activeNav === "evidence";

  useEffect(() => {
    if (!shouldLoadAiAdminWorkbenchPreview || !data?.loadedFromApi) {
      setAiAdminServerDraft(null);
      setAiAdminWorkbenchPreview(null);
      return;
    }
    const controller = new AbortController();
    previewPermissionPackageWorkbench(aiAdminForm, adminKey, controller.signal)
      .then((preview) => {
        setAiAdminServerDraft(preview.draft);
        if (aiAdminNewDraftMode) {
          setAiAdminWorkbenchPreview(null);
          setAiAdminApplication(null);
          setAiAdminProductionReadiness(null);
          setAiAdminApprovalRequests([]);
          return;
        }
        setAiAdminWorkbenchPreview(preview);
        if (preview.approvalRequest) {
          upsertAiAdminApprovalRequest(preview.approvalRequest);
          setAiAdminSelectedApprovalRequestId((current) => current || preview.approvalRequest?.id || "");
        }
        setAiAdminApplication(preview.latestApplication ?? null);
        setAiAdminProductionReadiness(preview.productionReadiness ?? null);
        if (preview.productionReadiness?.preflight) {
          setAiAdminApplyPreflight(preview.productionReadiness.preflight);
        }
        if (preview.productionReadiness?.applicationHealth) {
          setAiAdminApplicationHealth({
            applications: [preview.productionReadiness.applicationHealth],
            summary: {
              drifted: preview.productionReadiness.applicationHealth.status === "drifted" ? 1 : 0,
              needsReview: preview.productionReadiness.applicationHealth.status === "needs_review" ? 1 : 0,
              ready: preview.productionReadiness.applicationHealth.status === "ready" ? 1 : 0,
              total: 1
            }
          });
        }
        if (preview.productionReadiness?.applicationImpact) {
          setAiAdminApplicationImpact(preview.productionReadiness.applicationImpact);
        }
      })
      .catch((error) => {
        if (isAbortError(error)) return;
        setAiAdminWorkbenchPreview(null);
        if (isApiCompatibilityFallbackError(error)) {
          createPermissionPackageDraftFromApi(aiAdminForm, adminKey, controller.signal)
            .then((draft) => setAiAdminServerDraft(draft))
            .catch((draftError) => {
              if (!isAbortError(draftError)) {
                setAiAdminServerDraft(null);
              }
            });
          return;
        }
        setAiAdminServerDraft(null);
      });
    return () => controller.abort();
  }, [shouldLoadAiAdminWorkbenchPreview, adminKey, aiAdminForm, aiAdminNewDraftMode, data?.loadedFromApi]);

  async function refresh() {
    try {
      setLoadError("");
      const next = await loadConsoleData(
        adminKey,
        traceFilters,
        shouldLoadAiAdminCatalog ? undefined : normalizedScope(scope)
      );
      setData(next);
      setLastRefresh(new Date());
    } catch (error) {
      setLoadError(localizedErrorMessage(t, language, error, "error.consoleDataUnavailable"));
    }
    if (activeNav === "access") {
      await refreshAccessProfile();
    }
  }

  async function refreshAiAdminCatalog() {
    try {
      setLoadError("");
      const [next, templates, accessSubjects] = await Promise.all([
        loadConsoleData(adminKey, traceFilters),
        fetchPermissionPackageTemplates(adminKey).catch(() => permissionPackageTemplates),
        fetchPermissionPackageAccessSubjects(adminKey).catch(() => accessSubjectOptions)
      ]);
      setData(next);
      setAiAdminTemplates(templates.length > 0 ? templates : permissionPackageTemplates);
      setAiAdminAccessSubjects(normalizeAccessSubjectOptions(accessSubjects));
      setLastRefresh(new Date());
    } catch (error) {
      setLoadError(localizedErrorMessage(t, language, error, "error.consoleDataUnavailable"));
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
      setAccessMessage(localizedErrorMessage(t, language, error, "error.loadTenantAccessProfile"));
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
      setAccessDecisionExplainMessage(localizedErrorMessage(t, language, error, "error.explainAccessDecision"));
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
      setAiAdminAccessDecisionExplainMessage(localizedErrorMessage(t, language, error, "error.explainAccessDecision"));
    } finally {
      setAiAdminAccessDecisionExplainLoading(false);
    }
  }

  async function refreshAiAdminApplicationHealth(
    formInput: PermissionPackageDraftInput = aiAdminForm,
    options: { requireLiveApi?: boolean } = {}
  ) {
    const requireLiveApi = options.requireLiveApi ?? true;
    if (requireLiveApi && !data?.loadedFromApi) {
      setAiAdminApplicationHealthMessage(t("message.permissionApplicationHealthRequiresLiveApi"));
      return null;
    }
    setAiAdminApplicationHealthLoading(true);
    setAiAdminApplicationHealthMessage("");
    try {
      const next = await fetchPermissionPackageApplicationHealth(
        {
          callerInstanceId: formInput.callerInstanceId,
          limit: 10,
          targetId: formInput.targetId,
          templateId: formInput.templateId,
          tenantId: formInput.tenantId,
          workspaceId: formInput.workspaceId
        },
        adminKey
      );
      setAiAdminApplicationHealth(next);
      setAiAdminApplicationHealthMessage(t("message.permissionApplicationHealthLoaded"));
      return next;
    } catch (error) {
      setAiAdminApplicationHealthMessage(localizedErrorMessage(t, language, error, "error.refreshApplicationHealth"));
      return null;
    } finally {
      setAiAdminApplicationHealthLoading(false);
    }
  }

  function aiAdminProductionReadinessFilter(
    formInput: PermissionPackageDraftInput = aiAdminForm,
    options: { approvalRequestId?: string; subjectId?: string } = {}
  ): PermissionPackageProductionReadinessFilter {
    const approvalRequestId =
      options.approvalRequestId ??
      (aiAdminApprovalRequest?.status === "approved" ? aiAdminApprovalRequest.id : undefined);
    return {
      ...(approvalRequestId ? { approvalRequestId } : {}),
      callerInstanceId: formInput.callerInstanceId,
      region: formInput.region,
      requestText: formInput.requestText,
      subjectId: options.subjectId ?? subjectIdExampleFromSelector(formInput.subjectSelector),
      subjectSelector: formInput.subjectSelector,
      targetId: formInput.targetId,
      templateId: formInput.templateId,
      tenantId: formInput.tenantId,
      traceLimit: 20,
      workspaceId: formInput.workspaceId
    };
  }

  async function refreshAiAdminProductionReadiness(
    formInput: PermissionPackageDraftInput = aiAdminForm,
    options: { approvalRequestId?: string; requireLiveApi?: boolean; subjectId?: string } = {}
  ) {
    const requireLiveApi = options.requireLiveApi ?? true;
    if (requireLiveApi && !data?.loadedFromApi) {
      setAiAdminProductionReadinessMessage(t("message.permissionProductionReadinessRequiresLiveApi"));
      return null;
    }
    setAiAdminProductionReadinessLoading(true);
    setAiAdminProductionReadinessMessage("");
    try {
      const next = await fetchPermissionPackageProductionReadiness(
        aiAdminProductionReadinessFilter(formInput, {
          approvalRequestId: options.approvalRequestId,
          subjectId: options.subjectId
        }),
        adminKey
      );
      setAiAdminProductionReadiness(next);
      if (next.latestApplication) {
        setAiAdminApplication(next.latestApplication);
      }
      if (next.applicationHealth) {
        setAiAdminApplicationHealth({
          applications: [next.applicationHealth],
          summary: {
            drifted: next.applicationHealth.status === "drifted" ? 1 : 0,
            needsReview: next.applicationHealth.status === "needs_review" ? 1 : 0,
            ready: next.applicationHealth.status === "ready" ? 1 : 0,
            total: 1
          }
        });
      }
      if (next.applicationImpact) {
        setAiAdminApplicationImpact(next.applicationImpact);
      }
      setAiAdminProductionReadinessMessage(t("message.permissionProductionReadinessLoaded"));
      return next;
    } catch (error) {
      setAiAdminProductionReadinessMessage(localizedErrorMessage(t, language, error, "error.checkProductionReadiness"));
      return null;
    } finally {
      setAiAdminProductionReadinessLoading(false);
    }
  }

  async function exportAiAdminProductionEvidence(formInput: PermissionPackageDraftInput = aiAdminForm) {
    if (!data?.loadedFromApi) {
      setAiAdminMessage({ key: "message.productionEvidenceRequiresLiveApi" });
      return null;
    }
    setAiAdminProductionEvidenceExporting(true);
    setAiAdminMessage(null);
    setAiAdminProductionReadinessMessage("");
    try {
      const report = await fetchPermissionPackageProductionEvidenceReport(
        aiAdminProductionReadinessFilter(formInput),
        adminKey
      );
      downloadJson(report, productionEvidenceReportFilename(report));
      setAiAdminMessage({ key: "message.productionEvidenceExported" });
      return report;
    } catch (error) {
      setAiAdminMessage(localizedErrorMessageState(error, "error.exportProductionEvidence"));
      return null;
    } finally {
      setAiAdminProductionEvidenceExporting(false);
    }
  }

  function openAiAdminAccessProfile() {
    const tenantPath = permissionTenantPathLabel(aiAdminForm.tenantId, tenants, t);
    const selectedCaller = agents.find((agent) => agent.id === aiAdminForm.callerInstanceId);
    const selectedTarget = agents.find((agent) => agent.id === aiAdminForm.targetId);
    const selectedCapability = aiAdminDraft.allowedCapabilities[0];
    const subjectId = subjectIdExampleFromSelector(aiAdminForm.subjectSelector);
    setScope((current) => ({ ...current, tenantId: aiAdminForm.tenantId }));
    setAccessFilters((current) => ({
      ...current,
      capabilityId: selectedCapability?.id ?? "",
      callerInstanceId: aiAdminForm.callerInstanceId,
      subjectId,
      targetId: aiAdminForm.targetId,
      traceLimit: current.traceLimit || "20",
      workspaceId: aiAdminForm.workspaceId
    }));
    setAccessProfileHandoffContext({
      capabilityId: selectedCapability?.id ?? "",
      capabilityName: selectedCapability ? capabilityDisplayName(selectedCapability, t) : "",
      callerInstanceId: aiAdminForm.callerInstanceId,
      callerName: selectedCaller ? permissionEntityDisplayName(selectedCaller.name, t) : aiAdminForm.callerInstanceId,
      targetId: aiAdminForm.targetId,
      targetName: selectedTarget ? permissionEntityDisplayName(selectedTarget.name, t) : aiAdminForm.targetId,
      tenantId: aiAdminForm.tenantId,
      tenantName: permissionTenantPathLabel(aiAdminForm.tenantId, tenants, t).primary,
      tenantPath: tenantPath.path,
      workspaceId: aiAdminForm.workspaceId,
      workspaceName: permissionWorkspaceDisplayName(aiAdminForm.workspaceId, agents, t)
    });
    setAccessProfile(null);
    setAccessMessage("");
    setAccessDecisionExplanation(null);
    setAccessDecisionExplainMessage("");
    setActiveNav("access");
  }

  function startNewAiAdminPermissionChange() {
    setAiAdminNewDraftMode(true);
    setAiAdminForm((current) => ({
      ...defaultAiAdminForm,
      callerInstanceId: current.callerInstanceId,
      region: current.region,
      subjectSelector: current.subjectSelector,
      targetId: current.targetId,
      templateId: current.templateId,
      tenantId: current.tenantId,
      workspaceId: current.workspaceId
    }));
    setAiAdminServerDraft(null);
    setAiAdminWorkbenchPreview(null);
    setAiAdminApplication(null);
    setAiAdminApplicationHealth(null);
    setAiAdminApplicationHealthMessage("");
    setAiAdminApplyPreflight(null);
    setAiAdminApplyPreflightMessage("");
    setAiAdminApplicationImpact(null);
    setAiAdminApplicationImpactMessage("");
    setAiAdminProductionReadiness(null);
    setAiAdminProductionReadinessMessage("");
    setAiAdminApprovalRequests([]);
    setAiAdminSelectedApprovalRequestId("");
    setAiAdminAccessDecisionExplanation(null);
    setAiAdminAccessDecisionExplainMessage("");
    setAiAdminApprovalJourneyMessage("");
    setAiAdminApprovalJourneyResult(null);
    setAiAdminApprovalAuditEvent(null);
    setAiAdminApprovalJourneyAccessProfile(null);
    setAiAdminApprovalJourneyApprovalRequest(null);
    setAiAdminMessage(null);
  }

  async function reviewAiAdminApplicationImpact(applicationOverride?: PermissionPackageApplication) {
    if (!data?.loadedFromApi) {
      setAiAdminApplicationImpactMessage(t("message.permissionApplicationImpactRequiresLiveApi"));
      return;
    }
    const application = applicationOverride ?? aiAdminApplication;
    if (!application) {
      setAiAdminApplicationImpactMessage(t("message.aiAdminApprovalJourneyMissingApplication"));
      return;
    }
    setAiAdminApplication(application);
    setAiAdminApplicationImpactLoading(true);
    setAiAdminApplicationImpactMessage("");
    try {
      const next = await fetchPermissionPackageApplicationImpact(
        application.id,
        {
          tenantId: application.tenantId,
          workspaceId: application.workspaceId
        },
        adminKey
      );
      setAiAdminApplicationImpact(next);
      setAiAdminApplicationImpactMessage(t("message.permissionApplicationImpactLoaded"));
    } catch (error) {
      setAiAdminApplicationImpactMessage(localizedErrorMessage(t, language, error, "error.reviewApplicationImpact"));
    } finally {
      setAiAdminApplicationImpactLoading(false);
    }
  }

  async function rehearseAiAdminApplicationDrift() {
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
          rehearsal: "grant_drift",
          tenantId: aiAdminApplication.tenantId,
          workspaceId: aiAdminApplication.workspaceId
        },
        adminKey
      );
      setAiAdminApplicationImpact(next);
      setAiAdminApplicationImpactMessage(t("message.permissionApplicationDriftRehearsalLoaded"));
    } catch (error) {
      setAiAdminApplicationImpactMessage(localizedErrorMessage(t, language, error, "error.rehearseApplicationDrift"));
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
        mockMcpHealth.status === "ok" ? "" : `MCP service ${mockMcpHealth.message}`
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
      mockMcpHealth.status === "ok" ? "" : `MCP service ${mockMcpHealth.message}`,
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
    setAccessProfileHandoffContext(null);
    setAccessProfile(null);
    setScope(resetScope);
    try {
      setLoadError("");
      const nextData = await loadConsoleData(adminKey, resetTraceFilters, normalizedScope(resetScope));
      setData(nextData);
      setLastRefresh(new Date());
    } catch (error) {
      setLoadError(localizedErrorMessage(t, language, error, "error.consoleDataUnavailable"));
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
      const [nextData, nextProfile] = await Promise.all([
        loadConsoleData(adminKey, nextTraceFilters),
        loadTenantAccessProfile(nextConfig.childTenantId, adminKey, {
          ...nextAccessFilters,
          traceLimit: 10
        })
      ]);
      setTraceFilters(nextTraceFilters);
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
      setCoreJourneyMessage(localizedErrorMessage(t, language, error, "error.coreJourneyFailed"));
    } finally {
      setCoreJourneyRunning(false);
    }
  }

  async function runAiAdminApprovalJourney() {
    if (!data?.loadedFromApi) {
      setAiAdminApprovalJourneyMessage(t("message.fallbackDataModeActionBlocked"));
      return;
    }
    const nextConfig = {
      ...createAiAdminApprovalJourneyConfig(),
      requestText: t("default.aiAdminApprovalJourneyRequestText")
    };
    setAiAdminApprovalJourneyConfig(nextConfig);
    setAiAdminApprovalJourneyResult(null);
    setAiAdminApprovalAuditEvent(null);
    setAiAdminApprovalJourneyAccessProfile(null);
    setAiAdminApprovalJourneyApprovalRequest(null);
    setAiAdminApplicationHealth(null);
    setAiAdminApplicationHealthMessage("");
    setAiAdminApplyPreflight(null);
    setAiAdminApplyPreflightMessage("");
    setAiAdminApplicationImpact(null);
    setAiAdminApplicationImpactMessage("");
    setAiAdminApprovalJourneyRunning(true);
    setAiAdminApprovalJourneyMessage(t("message.aiAdminApprovalJourneyRunning"));
    setAiAdminMessage(null);
    try {
      const readinessResult = await refreshAiAdminApprovalReadiness(nextConfig);
      if (!aiAdminApprovalReadinessCanRun(readinessResult.state)) {
        const detail = readinessResult.detail;
        throw new Error(tx(t, "message.aiAdminApprovalJourneyPreflightFailed", { detail: detail || "unknown" }));
      }

      await createTenant(
        {
          id: nextConfig.rootTenantId,
          name: "Permission Request Approval Root",
          status: "active"
        },
        adminKey
      );
      await createTenant(
        {
          id: nextConfig.childTenantId,
          name: "Permission Request Approval Team",
          parentTenantId: nextConfig.rootTenantId,
          status: "active"
        },
        adminKey
      );
      await createTenant(
        {
          id: nextConfig.grandchildTenantId,
          name: "Permission Request Approval Project",
          parentTenantId: nextConfig.childTenantId,
          status: "active"
        },
        adminKey
      );

      const caller = await createAgent(
        {
          channelType: "local",
          description: "Permission request approval browser caller",
          name: "Permission Request Approval Caller",
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
          name: "permission request approval key"
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
          description: "Permission request approval MCP target",
          name: "Permission Request Approval MCP Target",
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

      const validationForm: PermissionPackageDraftInput = {
        callerInstanceId: caller.id,
        region: nextConfig.region,
        requestText: nextConfig.requestText,
        subjectSelector: nextConfig.subjectSelector,
        targetId: target.id,
        templateId: nextConfig.templateId,
        tenantId: nextConfig.childTenantId,
        workspaceId: nextConfig.workspaceId
      };
      setAiAdminApplication(null);
      setAiAdminApplicationHealth(null);
      setAiAdminApplicationHealthMessage("");
      setAiAdminApplyPreflight(null);
      setAiAdminApplyPreflightMessage("");
      setAiAdminApplicationImpact(null);
      setAiAdminApplicationImpactMessage("");
      const draft = await createPermissionPackageDraftFromApi(validationForm, adminKey);
      if (!draft.readiness.canApply) {
        throw new Error(tx(t, "message.permissionPackageNotReady", { detail: permissionReadinessMessages(draft.readiness, t).join(", ") }));
      }
      if (draft.policyGate.canApplyDirectly) {
        throw new Error(t("message.aiAdminApprovalJourneyApprovalGateMissing"));
      }

      const pendingApproval = await createPermissionPackageApprovalRequest(validationForm, adminKey);
      const approvedApproval = await approvePermissionPackageApprovalRequest(
        pendingApproval.id,
        {
          comment: "Approved from permission package approval journey",
          reviewer: "Security Reviewer"
        },
        adminKey
      );
      setAiAdminApprovalJourneyApprovalRequest(approvedApproval);

      const journeyPreflight = await preflightPermissionPackage(
        {
          ...validationForm,
          approvalRequestId: approvedApproval.id
        },
        adminKey
      );
      setAiAdminApplyPreflight(journeyPreflight);
      setAiAdminApplyPreflightMessage(
        journeyPreflight.summary.canApply
          ? t("message.permissionPackagePreflightReady")
          : tx(t, "message.permissionPackagePreflightBlocked", {
            detail: permissionApplyPreflightCheckMessage(firstBlockingApplyPreflightCheck(journeyPreflight), t)
          })
      );
      if (!journeyPreflight.summary.canApply) {
        throw new Error(
          tx(t, "message.permissionPackagePreflightBlocked", {
            detail: permissionApplyPreflightCheckMessage(firstBlockingApplyPreflightCheck(journeyPreflight), t)
          })
        );
      }

      const applied = await applyPermissionPackage(
        {
          ...validationForm,
          approvalRequestId: approvedApproval.id
        },
        adminKey
      );
      const application = applied.application ?? null;
      if (!application) {
        throw new Error(t("message.aiAdminApprovalJourneyMissingApplication"));
      }
      setAiAdminApplication(application);
      setAiAdminApplicationHealth(null);
      setAiAdminApplicationHealthMessage("");
      setAiAdminApplicationImpact(null);
      setAiAdminApplicationImpactMessage("");
      setAiAdminProductionReadiness(null);
      setAiAdminProductionReadinessMessage("");

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

      const nextTraceFilters = {
        callerAgentId: caller.id,
        decision: "" as TraceDecision | "",
        runId: nextConfig.runId,
        targetAgentId: target.id
      };
      const validationAccessFilters = {
        callerInstanceId: caller.id,
        capabilityId: "",
        targetId: target.id,
        traceLimit: "10",
        workspaceId: nextConfig.workspaceId
      };
      const [nextData, nextProfile, auditRows] = await Promise.all([
        loadConsoleData(adminKey, nextTraceFilters),
        loadTenantAccessProfile(nextConfig.childTenantId, adminKey, {
          ...validationAccessFilters,
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
      setTraceFilters(nextTraceFilters);
      setData(appliedAudit ? { ...nextData, auditEvents: [appliedAudit, ...nextData.auditEvents.filter((event) => event.id !== appliedAudit.id)] } : nextData);
      setAiAdminApprovalJourneyAccessProfile(nextProfile);
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
      await refreshAiAdminApplicationHealth(validationForm, { requireLiveApi: false });
      await refreshAiAdminProductionReadiness(validationForm, {
        approvalRequestId: approvedApproval.id,
        requireLiveApi: false,
        subjectId: nextConfig.subjectId
      });
      setAiAdminApprovalJourneyMessage(t("message.aiAdminApprovalJourneyComplete"));
      setAiAdminMessage({ key: "message.permissionPackageApplied", params: { count: applied.tenantEntitlements.length } });
    } catch (error) {
      setAiAdminApprovalJourneyMessage(localizedErrorMessage(t, language, error, "error.permissionPackageApprovalJourneyFailed"));
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
        setAgentMessage(retryFieldValidationMessage(retry.message, t));
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
      setAgentMessage(localizedErrorMessage(t, language, error, "error.createAgent"));
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
      setAgentMessage(localizedErrorMessage(t, language, error, "error.updateAgentStatus"));
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
      setPolicyMessage(localizedErrorMessage(t, language, error, "error.disableRoutePolicy"));
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
      setKeyMessage(localizedErrorMessage(t, language, error, "error.createKey"));
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
      setRotateMessage(localizedErrorMessage(t, language, error, "error.rotateCredential"));
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
        setPolicyMessage(retryFieldValidationMessage(retry.message, t));
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
      setPolicyMessage(localizedErrorMessage(t, language, error, "error.createRoutePolicy"));
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
      setCapabilityMessage(localizedErrorMessage(t, language, error, "error.refreshCapabilities"));
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
      setCapabilityMessage(tx(t, "message.capabilityApproved", { name: capabilityDisplayName(capability, t) }));
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
        setCapabilityMessage(tx(t, "message.capabilityApprovedFallback", { name: capabilityDisplayName(capability, t) }));
        return;
      }
      setCapabilityMessage(localizedErrorMessage(t, language, error, "error.approveCapability"));
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
    const subjectSelector = capabilityForm.subjectSelector.trim();
    if (!subjectSelector || subjectSelector === "*") {
      setCapabilityMessage(t("message.validationSubjectSelectorRequired"));
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
          subjectSelector,
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
      setCapabilityMessage(localizedErrorMessage(t, language, error, "error.createGrantChain"));
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

function aiAdminPermissionPackageApplyInput(): PermissionPackageApplyInput {
    const approvalRequestId = aiAdminApprovalRequest?.status === "approved" ? aiAdminApprovalRequest.id.trim() : "";
    return approvalRequestId ? { ...aiAdminForm, approvalRequestId } : { ...aiAdminForm };
  }

  async function refreshAiAdminApplyPreflight(signal?: AbortSignal, options: { silent?: boolean } = {}) {
    if (!data?.loadedFromApi) {
      setAiAdminApplyPreflight(null);
      setAiAdminApplyPreflightMessage(t("message.permissionPackagePreflightRequiresLiveApi"));
      return null;
    }
    setAiAdminApplyPreflightLoading(true);
    if (!options.silent) {
      setAiAdminApplyPreflightMessage("");
    }
    try {
      const next = await preflightPermissionPackage(aiAdminPermissionPackageApplyInput(), adminKey, signal);
      setAiAdminApplyPreflight(next);
      setAiAdminApplyPreflightMessage(
        next.summary.canApply
          ? t("message.permissionPackagePreflightReady")
          : tx(t, "message.permissionPackagePreflightBlocked", {
            detail: permissionApplyPreflightCheckMessage(firstBlockingApplyPreflightCheck(next), t)
          })
      );
      return next;
    } catch (error) {
      if (!isAbortError(error)) {
        setAiAdminApplyPreflight(null);
        setAiAdminApplyPreflightMessage(localizedErrorMessage(t, language, error, "error.runPermissionPackagePreflight"));
      }
      return null;
    } finally {
      setAiAdminApplyPreflightLoading(false);
    }
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
        setAiAdminReviewerQueueMessage(localizedErrorMessage(t, language, error, "error.loadReviewerQueue"));
      }
    } finally {
      setAiAdminReviewerQueueLoading(false);
    }
  }

  function startApprovalResolutionCooldown() {
    approvalResolveBlockedRef.current = true;
    setAiAdminApprovalResolutionCoolingDown(true);
    if (approvalResolveCooldownTimerRef.current !== null) {
      window.clearTimeout(approvalResolveCooldownTimerRef.current);
    }
    approvalResolveCooldownTimerRef.current = window.setTimeout(() => {
      approvalResolveBlockedRef.current = false;
      approvalResolveCooldownTimerRef.current = null;
      setAiAdminApprovalResolutionCoolingDown(false);
    }, approvalResolveCooldownMs);
  }

  async function createAiAdminApprovalRequest() {
    if (approvalCreateInFlightRef.current || aiAdminApprovalAction === "create") {
      return;
    }
    if (!data?.loadedFromApi) {
      setAiAdminMessage({ key: "message.permissionApprovalRequiresLiveApi" });
      return;
    }
    if (!aiAdminForm.requestText.trim()) {
      setAiAdminMessage({ key: "message.permissionApprovalRequestTextRequired" });
      return;
    }
    if (aiAdminApprovalRequest?.status === "pending") {
      setAiAdminSelectedApprovalRequestId(aiAdminApprovalRequest.id);
      setAiAdminMessage({ key: "message.permissionApprovalAlreadyPending" });
      return;
    }
    approvalCreateInFlightRef.current = true;
    setAiAdminApprovalAction("create");
    setAiAdminMessage(null);
    try {
      const request = await createPermissionPackageApprovalRequest(aiAdminForm, adminKey);
      startApprovalResolutionCooldown();
      setAiAdminNewDraftMode(false);
      upsertAiAdminApprovalRequest(request);
      setAiAdminSelectedApprovalRequestId(request.id);
      setAiAdminWorkbenchPreview(null);
      setAiAdminApplyPreflight(null);
      setAiAdminApplyPreflightMessage("");
      setAiAdminApplication(null);
      setAiAdminApplicationHealth(null);
      setAiAdminApplicationHealthMessage("");
      setAiAdminApplicationImpact(null);
      setAiAdminApplicationImpactMessage("");
      setAiAdminProductionReadiness(null);
      setAiAdminProductionReadinessMessage("");
      setAiAdminMessage({ key: "message.permissionApprovalCreated", params: { id: request.id } });
    } catch (error) {
      setAiAdminMessage(localizedErrorMessageState(error, "error.createApprovalRequest"));
    } finally {
      approvalCreateInFlightRef.current = false;
      setAiAdminApprovalAction("");
    }
  }

  async function approveAiAdminApprovalRequest(requestId?: string, comment?: string) {
    if (approvalResolveBlockedRef.current) {
      return;
    }
    if (!data?.loadedFromApi) {
      setAiAdminMessage({ key: "message.permissionApprovalRequiresLiveApi" });
      return;
    }
    const targetRequest = requestId
      ? aiAdminApprovalRequests.find((request) => request.id === requestId)
      : aiAdminApprovalRequest;
    if (!targetRequest) return;
    const reviewerComment = comment?.trim();
    setAiAdminApprovalAction("approve");
    setAiAdminMessage(null);
    try {
      const reviewer = aiAdminApprovalReviewer.trim();
      const request = await approvePermissionPackageApprovalRequest(
        targetRequest.id,
        {
          comment: reviewerComment || (requestId ? "Approved from permission package reviewer queue" : "Approved from permission package console"),
          ...(reviewer ? { reviewer } : {})
        },
        adminKey
      );
      upsertAiAdminApprovalRequest(request);
      setAiAdminApplyPreflight(null);
      setAiAdminApplyPreflightMessage("");
      setAiAdminMessage({ key: "message.permissionApprovalApproved", params: { id: request.id } });
    } catch (error) {
      setAiAdminMessage(localizedErrorMessageState(error, "error.approveRequest"));
    } finally {
      setAiAdminApprovalAction("");
    }
  }

  async function rejectAiAdminApprovalRequest(requestId?: string, comment?: string) {
    if (approvalResolveBlockedRef.current) {
      return;
    }
    if (!data?.loadedFromApi) {
      setAiAdminMessage({ key: "message.permissionApprovalRequiresLiveApi" });
      return;
    }
    const targetRequest = requestId
      ? aiAdminApprovalRequests.find((request) => request.id === requestId)
      : aiAdminApprovalRequest;
    if (!targetRequest) return;
    const reviewerComment = comment?.trim();
    if (!reviewerComment) {
      setAiAdminMessage({ key: "message.permissionApprovalRejectReasonRequired" });
      return;
    }
    setAiAdminApprovalAction("reject");
    setAiAdminMessage(null);
    try {
      const reviewer = aiAdminApprovalReviewer.trim();
      const request = await rejectPermissionPackageApprovalRequest(
        targetRequest.id,
        {
          comment: reviewerComment,
          ...(reviewer ? { reviewer } : {})
        },
        adminKey
      );
      upsertAiAdminApprovalRequest(request);
      setAiAdminApplyPreflight(null);
      setAiAdminApplyPreflightMessage("");
      setAiAdminMessage({ key: "message.permissionApprovalRejected", params: { id: request.id } });
    } catch (error) {
      setAiAdminMessage(localizedErrorMessageState(error, "error.rejectRequest"));
    } finally {
      setAiAdminApprovalAction("");
    }
  }

  async function withdrawAiAdminApprovalRequest(comment?: string) {
    if (!data?.loadedFromApi) {
      setAiAdminMessage({ key: "message.permissionApprovalRequiresLiveApi" });
      return;
    }
    if (!aiAdminApprovalRequest || aiAdminApprovalRequest.status !== "pending") {
      setAiAdminMessage({ key: "message.permissionApprovalWithdrawUnavailable" });
      return;
    }
    const withdrawComment = comment?.trim();
    setAiAdminApprovalAction("withdraw");
    setAiAdminMessage(null);
    try {
      const request = await withdrawPermissionPackageApprovalRequest(aiAdminApprovalRequest.id, { comment: withdrawComment }, adminKey);
      upsertAiAdminApprovalRequest(request);
      setAiAdminSelectedApprovalRequestId(request.id);
      setAiAdminWorkbenchPreview(null);
      setAiAdminApplyPreflight(null);
      setAiAdminApplyPreflightMessage("");
      setAiAdminApplication(null);
      setAiAdminApplicationHealth(null);
      setAiAdminApplicationHealthMessage("");
      setAiAdminApplicationImpact(null);
      setAiAdminApplicationImpactMessage("");
      setAiAdminProductionReadiness(null);
      setAiAdminProductionReadinessMessage("");
      setAiAdminMessage({ key: "message.permissionApprovalWithdrawn" });
    } catch (error) {
      setAiAdminMessage(localizedErrorMessageState(error, "error.withdrawRequest"));
    } finally {
      setAiAdminApprovalAction("");
    }
  }

  async function applyAiAdminPermissionPackage() {
    setAiAdminMessage(null);
    setAiAdminApplicationHealth(null);
    setAiAdminApplicationHealthMessage("");
    setAiAdminApplicationImpact(null);
    setAiAdminApplicationImpactMessage("");
    setAiAdminProductionReadiness(null);
    setAiAdminProductionReadinessMessage("");
    if (!data?.loadedFromApi) {
      setAiAdminMessage({ key: "message.fallbackDataModeActionBlocked" });
      return;
    }
    if (!aiAdminDraft.readiness.canApply) {
      setAiAdminMessage({
        render: (t) => {
          const detail = permissionReadinessMessages(aiAdminDraft.readiness, t).join(", ");
          return tx(t, "message.permissionPackageNotReady", { detail: detail || "not ready" });
        }
      });
      return;
    }
    if (!aiAdminDraft.policyGate.canApplyDirectly) {
      if (!data?.loadedFromApi) {
        setAiAdminMessage({ key: "message.permissionApprovalRequiresLiveApi" });
        return;
      }
      if (!aiAdminApprovalRequest) {
        setAiAdminMessage({
          render: (t) => {
            const detail = permissionPolicyGateMessages(aiAdminDraft.policyGate, t).join(", ");
            return tx(t, "message.permissionPackageApprovalRequired", { detail: detail || t("status.approvalRequired") });
          }
        });
        return;
      }
      if (aiAdminApprovalRequest.status === "pending") {
        setAiAdminMessage({ key: "message.permissionApprovalPending" });
        return;
      }
      if (aiAdminApprovalRequest.status === "rejected") {
        setAiAdminMessage({ key: "message.permissionApprovalRejectedApply" });
        return;
      }
      if (aiAdminApprovalRequest.status === "withdrawn") {
        setAiAdminMessage({ key: "message.permissionApprovalWithdrawnApply" });
        return;
      }
    }
    if (data?.loadedFromApi) {
      const preflight = await refreshAiAdminApplyPreflight(undefined, { silent: true });
      if (!preflight) {
        setAiAdminMessage({ key: "message.permissionPackagePreflightRequiresLiveApi" });
        return;
      }
      if (!preflight.summary.canApply) {
        setAiAdminMessage({
          render: (t) => tx(t, "message.permissionPackagePreflightApplyBlocked", {
            detail: permissionApplyPreflightCheckMessage(firstBlockingApplyPreflightCheck(preflight), t)
          })
        });
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
      if (application) {
        await refreshAiAdminApplicationHealth(aiAdminForm, { requireLiveApi: false });
        await refreshAiAdminProductionReadiness(aiAdminForm, { requireLiveApi: false });
      }
      setAiAdminApplicationImpact(null);
      setAiAdminApplicationImpactMessage("");
      setAiAdminMessage({ key: "message.permissionPackageApplied", params: { count: appliedCount } });
      await loadAiAdminApprovalRequestsForDraft(aiAdminDraft).catch(() => undefined);
    } catch (error) {
      setAiAdminMessage(localizedErrorMessageState(error, "error.applyPermissionPackage"));
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
          subjectSelector: (aiAdminForm.subjectSelector ?? "").trim(),
          workspaceAssignmentId: workspaceAssignment.id
        },
        adminKey
      );
    }
    return aiAdminDraft.allowedCapabilities.length;
  }

  const agents = data?.agents ?? [];
  const tenants = data?.tenants ?? [];
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
    if (activeNav !== "ai-admin" || aiAdminNewDraftMode || !data?.loadedFromApi || !aiAdminDraft.readiness.canApply || aiAdminDraft.policyGate.canApplyDirectly) {
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
    aiAdminNewDraftMode,
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
  const runtimeEvidence = runtimeEvidenceMetric(allowedTraces, deniedTraces);
  const dataSourceLabel = loadError
    ? t("dataSource.apiError")
    : data?.loadedFromApi
      ? t("dataSource.runtime")
      : t("dataSource.fallback");
  const activeView = viewForNav(activeNav);
  const activeNavItem = navItems.find((item) => item.key === activeView.key) ?? navItems[0];
  const activeNavLabel = t(`nav.${activeNavItem.key}`, activeNavItem.label);
  const showWorkspaceTelemetry = activeView.key === "cockpit";
  const pageTitle = t(activeView.titleKey, t("app.title"));

  useEffect(() => {
    const handleHashChange = () => {
      setActiveNav(navKeyFromHash(window.location.hash) ?? defaultNavKey);
    };
    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  useEffect(() => {
    const nextHash = navHashFor(activeView.key);
    if (window.location.hash !== nextHash) {
      window.history.replaceState(null, "", `${window.location.pathname}${window.location.search}${nextHash}`);
    }
  }, [activeView.key]);

  useEffect(() => {
    window.scrollTo({ left: 0, top: 0 });
    document.querySelector(".workspace")?.scrollTo({ left: 0, top: 0 });
  }, [activeView.key]);
  const coreJourneyEvaluation = useMemo(
    () => evaluateCoreJourney(data, accessProfile, coreJourneyConfig),
    [accessProfile, coreJourneyConfig, data]
  );
  const aiAdminApprovalJourneyEvaluation = useMemo(
    () =>
      evaluateAiAdminApprovalJourney({
        accessProfile: aiAdminApprovalJourneyAccessProfile ?? accessProfile,
        application: aiAdminApplication,
        approvalRequest: aiAdminApprovalJourneyApprovalRequest ?? aiAdminApprovalRequest,
        auditEvent: aiAdminApprovalAuditEvent,
        config: aiAdminApprovalJourneyConfig,
        data,
        result: aiAdminApprovalJourneyResult
      }),
    [
      accessProfile,
      aiAdminApprovalJourneyAccessProfile,
      aiAdminApprovalJourneyApprovalRequest,
      aiAdminApplication,
      aiAdminApprovalAuditEvent,
      aiAdminApprovalJourneyConfig,
      aiAdminApprovalJourneyResult,
      aiAdminApprovalRequest,
      data
    ]
  );
  const aiAdminProductionConsoleSummary = useMemo(
    () =>
      buildAiAdminProductionConsoleSummary({
        application: aiAdminApplication,
        approvalRequest: aiAdminApprovalRequest,
        draft: aiAdminDraft,
        productionReadiness: aiAdminProductionReadiness
      }),
    [aiAdminApplication, aiAdminApprovalRequest, aiAdminDraft, aiAdminProductionReadiness]
  );
  const goLiveAcceptanceForm = aiAdminServerDraft?.input ?? aiAdminForm;
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
  const goLiveAcceptancePanel = (
    <Panel className="span-12" icon={<ClipboardCheck size={18} />} title={t("section.goLiveAcceptance")}>
      <GoLiveAcceptanceOverview
        agents={agents}
        draft={aiAdminServerDraft}
        form={aiAdminForm}
        liveDataAvailable={Boolean(data?.loadedFromApi)}
        onExportProductionEvidence={() => void exportAiAdminProductionEvidence(goLiveAcceptanceForm)}
        onOpenPermissionChange={() => setActiveNav("ai-admin")}
        onRefreshProductionReadiness={() => void refreshAiAdminProductionReadiness(goLiveAcceptanceForm)}
        productionEvidenceExporting={aiAdminProductionEvidenceExporting}
        productionReadiness={aiAdminProductionReadiness}
        productionReadinessLoading={aiAdminProductionReadinessLoading}
        productionReadinessMessage={aiAdminProductionReadinessMessage}
        productionSummary={aiAdminProductionConsoleSummary}
        templates={aiAdminTemplates}
        tenants={tenants}
        t={t}
      />
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
    <Panel className={className} icon={<ClipboardCheck size={18} />} title={t("panel.evidenceRuns")} action={<IconOpen title={t("action.open")} />}>
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
        tenants={tenants}
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
        handoffContext={accessProfileHandoffContext}
        loading={accessLoading}
        message={accessMessage}
        onChange={(filters) => {
          setAccessFilters(filters);
          setAccessProfileHandoffContext(null);
          setAccessDecisionExplanation(null);
          setAccessDecisionExplainMessage("");
        }}
        onExplainAccessDecision={() => void explainAccessDecisionFromProfile()}
        onRefresh={() => void refreshAccessProfile()}
        onTenantChange={(tenantId) => {
          setScope((current) => ({ ...current, tenantId }));
          setAccessProfileHandoffContext(null);
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
      <AiAdminPermissionWorkbench
        accessSubjects={aiAdminAccessSubjects}
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
        approvalResolutionBlocked={aiAdminApprovalResolutionCoolingDown}
        approvalRequest={aiAdminApprovalRequest}
        approvalRequests={aiAdminApprovalRequests}
        approvalReviewer={aiAdminApprovalReviewer}
        workbenchPreview={aiAdminWorkbenchPreview}
        liveDataAvailable={Boolean(data?.loadedFromApi)}
        productionSummary={aiAdminProductionConsoleSummary}
        applyPreflight={aiAdminApplyPreflight}
        applyPreflightLoading={aiAdminApplyPreflightLoading}
        applyPreflightMessage={aiAdminApplyPreflightMessage}
        applicationHealth={aiAdminApplicationHealth}
        applicationHealthLoading={aiAdminApplicationHealthLoading}
        applicationHealthMessage={aiAdminApplicationHealthMessage}
        applicationImpact={aiAdminApplicationImpact}
        applicationImpactLoading={aiAdminApplicationImpactLoading}
        applicationImpactMessage={aiAdminApplicationImpactMessage}
        productionReadiness={aiAdminProductionReadiness}
        productionEvidenceExporting={aiAdminProductionEvidenceExporting}
        productionReadinessLoading={aiAdminProductionReadinessLoading}
        productionReadinessMessage={aiAdminProductionReadinessMessage}
        accessDecisionExplanation={aiAdminAccessDecisionExplanation}
        accessDecisionExplanationLoading={aiAdminAccessDecisionExplainLoading}
        accessDecisionExplanationMessage={aiAdminAccessDecisionExplainMessage}
        applying={aiAdminApplying}
        draft={aiAdminDraft}
        form={aiAdminForm}
        application={aiAdminApplication}
        message={renderedAiAdminMessage}
        mcpTargets={mcpTargets}
        onApply={() => void applyAiAdminPermissionPackage()}
        onApprovalReviewerChange={setAiAdminApprovalReviewer}
        onApproveApprovalRequest={(requestId, comment) => void approveAiAdminApprovalRequest(requestId, comment)}
        onChange={(nextForm) => {
          setAiAdminForm(nextForm);
          setAiAdminWorkbenchPreview(null);
          setAiAdminApplication(null);
          setAiAdminApplicationHealth(null);
          setAiAdminApplicationHealthMessage("");
          setAiAdminApplyPreflight(null);
          setAiAdminApplyPreflightMessage("");
          setAiAdminApplicationImpact(null);
          setAiAdminApplicationImpactMessage("");
          setAiAdminProductionReadiness(null);
          setAiAdminProductionReadinessMessage("");
          setAiAdminApprovalAuditEvent(null);
          setAiAdminApprovalJourneyResult(null);
          setAiAdminApprovalRequests([]);
          setAiAdminSelectedApprovalRequestId("");
          setAiAdminAccessDecisionExplanation(null);
          setAiAdminAccessDecisionExplainMessage("");
        }}
        onCreateApprovalRequest={() => void createAiAdminApprovalRequest()}
        onExplainAccessDecision={() => void explainAiAdminAccessDecision()}
        onOpenAccessProfile={openAiAdminAccessProfile}
        onRefreshApplyPreflight={() => void refreshAiAdminApplyPreflight()}
        onRefreshApprovalReadiness={() => void refreshAiAdminApprovalReadiness()}
        onRefreshApplicationHealth={() => void refreshAiAdminApplicationHealth()}
        onExportProductionEvidence={() => void exportAiAdminProductionEvidence()}
        onRefreshProductionReadiness={() => void refreshAiAdminProductionReadiness()}
        onRefreshReviewerQueue={() => void refreshAiAdminReviewerQueue()}
        onRejectApprovalRequest={(requestId, comment) => void rejectAiAdminApprovalRequest(requestId, comment)}
        onRehearseApplicationDrift={() => void rehearseAiAdminApplicationDrift()}
        onReviewApplicationHealthRow={(application) => void reviewAiAdminApplicationImpact(application)}
        onReviewApplicationImpact={() => void reviewAiAdminApplicationImpact()}
        onRunApprovalJourney={() => void runAiAdminApprovalJourney()}
        onSelectApprovalRequest={(requestId) => {
          setAiAdminSelectedApprovalRequestId(requestId);
          setAiAdminApplicationImpact(null);
          setAiAdminApplicationImpactMessage("");
        }}
        onStartNewPermissionChange={startNewAiAdminPermissionChange}
        onWithdrawApprovalRequest={(comment) => void withdrawAiAdminApprovalRequest(comment)}
        reviewerQueueLoading={aiAdminReviewerQueueLoading}
        reviewerQueueMessage={aiAdminReviewerQueueMessage}
        selectedApprovalRequestId={aiAdminSelectedApprovalRequestId}
        templates={aiAdminTemplates}
        tenants={tenants}
        t={t}
      />
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
          </section>
        );
      case "registry":
        return (
          <section className="content-grid">
            {agentRegistryPanel("span-12")}
            {createAgentPanel}
            {createKeyPanel}
            {rotateCredentialPanel}
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
            {capabilityGovernancePanel()}
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
            {goLiveAcceptancePanel}
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

        <nav className="nav-list" aria-label={t("navGroup.mainNavigation")}>
          {navGroups.map((group) => {
            const groupItems = navItems.filter((item) => item.groupKey === group.key);
            return (
              <section className="nav-group" key={group.key}>
                <div className="section-kicker">{t(group.labelKey)}</div>
                <div className="nav-group-items">
                  {groupItems.map((item) => {
                    const Icon = navIconFor(item.key);
                    const itemLabel = t(`nav.${item.key}`, item.label);
                    const itemDetail = t(item.detailKey);
                    return (
                      <button
                        aria-current={activeView.key === item.key ? "page" : undefined}
                        aria-label={`${itemLabel} - ${itemDetail}`}
                        className={activeView.key === item.key ? "nav-item active" : "nav-item"}
                        key={item.key}
                        onClick={() => setActiveNav(item.key)}
                        title={`${itemLabel} - ${itemDetail}`}
                        type="button"
                      >
                        <Icon size={16} />
                        <span>
                          <strong>{itemLabel}</strong>
                          <small>{itemDetail}</small>
                        </span>
                      </button>
                    );
                  })}
                </div>
              </section>
            );
          })}
        </nav>

      </aside>

      <main className="workspace" aria-busy={!data}>
        <header className="topbar">
          <div className="topbar-title">
            <div className="breadcrumb">{t("app.gateway")} / {activeNavLabel}</div>
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
            <details className="connection-menu" onToggle={(event) => setConnectionMenuOpen(event.currentTarget.open)} open={connectionMenuOpen}>
              <summary
                aria-label={t("section.connectionSettings")}
                className="connection-trigger"
                onClick={(event) => {
                  event.preventDefault();
                  setConnectionMenuOpen((open) => !open);
                }}
                title={t("section.connectionSettings")}
              >
                <LockKeyhole size={15} />
                <span>{t("section.connectionSettings")}</span>
              </summary>
              <div className="connection-popover">
                <div className="connection-popover-header">
                  <strong>{t("section.connectionSettings")}</strong>
                  <span>{t("text.connectionSettingsDetail")}</span>
                </div>
                <label className="connection-field">
                  <span>{t("control.adminKey")}</span>
                  <input
                    onChange={(event) => setAdminKey(event.target.value)}
                    placeholder={t("control.adminKey")}
                    type="password"
                    value={adminKey}
                  />
                </label>
                <div className="connection-scope-grid">
                  <label className="connection-field">
                    <span>{t("form.tenantId")}</span>
                    <input
                      onBlur={() => void refresh()}
                      onChange={(event) => setScope((current) => ({ ...current, tenantId: event.target.value }))}
                      placeholder="tenantId"
                      value={scope.tenantId}
                    />
                  </label>
                  <label className="connection-field">
                    <span>{t("form.workspaceId")}</span>
                    <input
                      onBlur={() => void refresh()}
                      onChange={(event) => setScope((current) => ({ ...current, workspaceId: event.target.value }))}
                      placeholder="workspaceId"
                      value={scope.workspaceId}
                    />
                  </label>
                </div>
                <p className="connection-note">{t("text.adminKeyStoredLocally")}</p>
                <div className="connection-status">
                  <span className="health-dot" />
                  <div>
                    <strong>{loadError ? t("dataSource.apiError") : data?.loadedFromApi ? t("status.controlLive") : t("status.controlFallback")}</strong>
                    <span>{dataSourceLabel}</span>
                  </div>
                </div>
              </div>
            </details>
            <button aria-label={t("action.refresh")} className="icon-button" onClick={refresh} title={t("action.refresh")} type="button">
              <RefreshCw size={17} />
            </button>
          </div>
        </header>

        {showWorkspaceTelemetry ? (
          <>
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
                label={t("metric.managedAgents")}
                value={String(agents.length)}
                detail={`${activeAgents} ${t("detail.active")}`}
                tone="info"
              />
              <MetricCard
                icon={<KeyRound size={18} />}
                label={t("metric.activePolicies")}
                value={String(activePolicies)}
                detail={data?.routePoliciesLoadedFromApi ? t("detail.liveRoutePolicies") : t("detail.sampleFallback")}
                tone="success"
              />
              <MetricCard
                icon={<TriangleAlert size={18} />}
                label={t("metric.deniedTraces")}
                value={String(deniedTraces)}
                detail={`${allowedTraces} ${t("detail.allowed")}`}
                tone={deniedTraces > 0 ? "warning" : "success"}
              />
              <MetricCard
                icon={<ClipboardCheck size={18} />}
                label={t("metric.runtimeEvidence")}
                value={runtimeEvidence.value}
                detail={runtimeEvidence.value === "0" ? t("detail.noTraces") : `${allowedTraces} ${t("detail.allowed")} / ${deniedTraces} ${t("detail.denied")}`}
                tone={runtimeEvidence.tone}
              />
            </section>
          </>
        ) : null}

        {!data ? (
          <section className="workspace-loading" role="status" aria-live="polite">
            <div className="workspace-loading-copy">
              <strong>{t("status.loadingConsole")}</strong>
              <span>{t("text.loadingConsoleDetail")}</span>
            </div>
            <div className="workspace-loading-skeleton" aria-hidden="true">
              <span />
              <span />
              <span />
            </div>
          </section>
        ) : viewContent}
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
      <p className="core-journey-intro">{t("text.coreJourneyIntro")}</p>
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
      <label>
        <span>{t("form.decision")}</span>
        <select value={filters.decision ?? ""} onChange={(event) => onChange({ ...filters, decision: event.target.value as TraceDecision | "" })}>
          <option value="">{t("form.anyDecision")}</option>
          <option value="allowed">{t("text.decisionAllowed")}</option>
          <option value="denied">{t("text.decisionDenied")}</option>
        </select>
      </label>
      <label>
        <span>{t("form.caller")}</span>
        <select value={filters.callerAgentId ?? ""} onChange={(event) => onChange({ ...filters, callerAgentId: event.target.value })}>
          <option value="">{t("form.anyCaller")}</option>
          {agents.map((agent) => <option key={agent.id} value={agent.id}>{permissionEntityDisplayName(agent.name, t)}</option>)}
        </select>
      </label>
      <label>
        <span>{t("form.target")}</span>
        <select value={filters.targetAgentId ?? ""} onChange={(event) => onChange({ ...filters, targetAgentId: event.target.value })}>
          <option value="">{t("form.anyTarget")}</option>
          {agents.map((agent) => <option key={agent.id} value={agent.id}>{permissionEntityDisplayName(agent.name, t)}</option>)}
        </select>
      </label>
      <button className="secondary-button" type="button" onClick={onRefresh}><RefreshCw size={14} /> {t("action.refresh")}</button>
      <details className="trace-filter-advanced">
        <summary>{t("text.technicalDetails")}</summary>
        <label>
          <span>{t("form.traceRunId")}</span>
          <input placeholder={t("form.traceRunPlaceholder")} value={filters.runId ?? ""} onChange={(event) => onChange({ ...filters, runId: event.target.value })} />
        </label>
      </details>
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

function GoLiveAcceptanceOverview({
  agents,
  draft,
  form,
  liveDataAvailable,
  onExportProductionEvidence,
  onOpenPermissionChange,
  onRefreshProductionReadiness,
  productionEvidenceExporting,
  productionReadiness,
  productionReadinessLoading,
  productionReadinessMessage,
  productionSummary,
  templates,
  tenants,
  t
}: {
  agents: Agent[];
  draft: PermissionPackageDraft | null;
  form: PermissionPackageDraftInput;
  liveDataAvailable: boolean;
  onExportProductionEvidence: () => void;
  onOpenPermissionChange: () => void;
  onRefreshProductionReadiness: () => void;
  productionEvidenceExporting: boolean;
  productionReadiness: PermissionPackageProductionReadiness | null;
  productionReadinessLoading: boolean;
  productionReadinessMessage: string;
  productionSummary: AiAdminProductionConsoleSummary;
  templates: PermissionPackageTemplate[];
  tenants: Tenant[];
  t: Translator;
}) {
  const acceptanceInput = draft?.input ?? form;
  const acceptanceTemplate = draft?.template;
  const tenantPath = permissionTenantPathLabel(acceptanceInput.tenantId, tenants, t);
  const template = acceptanceTemplate ?? templates.find((item) => item.id === acceptanceInput.templateId);
  const caller = agents.find((agent) => agent.id === acceptanceInput.callerInstanceId);
  const target = agents.find((agent) => agent.id === acceptanceInput.targetId);
  const workspaceName = permissionWorkspaceDisplayName(acceptanceInput.workspaceId, agents, t);
  const templateName = template
    ? permissionPackageTemplateName(template, t)
    : permissionPackageTemplateNameById(acceptanceInput.templateId, t);
  const callerName = caller
    ? permissionEntityDisplayName(caller.name, t)
    : acceptanceInput.callerInstanceId
      ? t("text.selectedCallerFallback")
      : t("text.callerPendingSelection");
  const targetName = target
    ? permissionEntityDisplayName(target.name, t)
    : acceptanceInput.targetId
      ? t("text.selectedTargetFallback")
      : t("text.targetPendingSelection");
  const readinessStatusLabel = productionReadinessStatusLabel(productionReadiness?.status, t);
  const statusLabel = productionReadiness ? readinessStatusLabel : productionConsoleStatusLabel(productionSummary, t);
  const statusTone = productionReadiness
    ? productionReadinessStatusTone(productionReadiness.status)
    : productionConsoleStatusTone(productionSummary.status);
  const nextAction = productionReadiness?.nextActions[0]
    ? permissionProductionReadinessNextAction(productionReadiness.nextActions[0], t)
    : productionReadiness?.status === "ready"
      ? t("text.productionReadinessReadyDetail")
      : productionReadiness
        ? t("text.productionReadinessPendingDetail")
        : t("text.goLiveAcceptanceNoReadinessDetail");
  const readyCount = productionReadiness?.summary.readyCount
    ?? productionSummary.steps.filter((step) => step.status === "ready").length;
  const totalCount = productionReadiness?.checks.length ?? productionSummary.steps.length;
  const blockerCount = productionReadiness?.summary.blockingCount
    ?? productionSummary.steps.filter((step) => step.status === "blocked").length;
  const warningCount = productionReadiness?.summary.warningCount
    ?? productionSummary.steps.filter((step) => step.status === "needs_review" || step.status === "pending").length;
  const acceptanceReady = productionReadiness?.status === "ready";
  const statusMessage = productionReadinessMessage === t("message.permissionProductionReadinessLoaded")
    ? ""
    : productionReadinessMessage;

  return (
    <div className="go-live-acceptance">
      <section className="go-live-acceptance-main">
        <div className="go-live-acceptance-heading">
          <span>{t("text.goLiveAcceptanceTaskTitle")}</span>
          <Badge tone={statusTone}>{statusLabel}</Badge>
        </div>
        <p>{nextAction}</p>
        {!liveDataAvailable ? <p className="go-live-acceptance-warning">{t("message.fallbackDataModeDetail")}</p> : null}
        {statusMessage ? <p className="go-live-acceptance-message">{statusMessage}</p> : null}
        <div className="go-live-acceptance-actions">
          {acceptanceReady ? (
            <>
              <button className="primary-button" disabled={!liveDataAvailable || productionEvidenceExporting} onClick={onExportProductionEvidence} type="button">
                <Download size={14} />
                {productionEvidenceExporting ? t("action.exportingProductionEvidence") : t("action.exportProductionEvidence")}
              </button>
              <button className="secondary-button" disabled={!liveDataAvailable || productionReadinessLoading} onClick={onRefreshProductionReadiness} type="button">
                <RefreshCw size={14} />
                {productionReadinessLoading ? t("action.checkingProductionReadiness") : t("action.checkProductionReadiness")}
              </button>
            </>
          ) : (
            <>
              <button className="primary-button" disabled={!liveDataAvailable || productionReadinessLoading} onClick={onRefreshProductionReadiness} type="button">
                <RefreshCw size={14} />
                {productionReadinessLoading ? t("action.checkingProductionReadiness") : t("action.checkProductionReadiness")}
              </button>
              <button className="secondary-button" disabled={!liveDataAvailable || !productionReadiness || productionEvidenceExporting} onClick={onExportProductionEvidence} type="button">
                <Download size={14} />
                {productionEvidenceExporting ? t("action.exportingProductionEvidence") : t("action.exportProductionEvidence")}
              </button>
            </>
          )}
          <button className="secondary-button" onClick={onOpenPermissionChange} type="button">
            <ShieldCheck size={14} />
            {t("action.openPermissionChange")}
          </button>
        </div>
      </section>

      <aside className="go-live-acceptance-context" aria-label={t("text.goLiveAcceptanceContext")}>
        <strong>{t("text.goLiveAcceptanceContext")}</strong>
        <dl>
          <div>
            <dt>{t("form.businessTenant")}</dt>
            <dd>{tenantPath.primary}</dd>
          </div>
          <div>
            <dt>{t("form.businessWorkspace")}</dt>
            <dd>{workspaceName}</dd>
          </div>
          <div>
            <dt>{t("form.businessCaller")}</dt>
            <dd>{callerName} → {targetName}</dd>
          </div>
          <div>
            <dt>{t("form.permissionPackage")}</dt>
            <dd>{templateName}</dd>
          </div>
        </dl>
      </aside>

      <section className="go-live-acceptance-checks" aria-label={t("section.permissionRequestProcess")}>
        <div className="go-live-acceptance-score">
          <div>
            <span>{t("metric.productionReadyChecks")}</span>
            <strong>{readyCount}/{totalCount}</strong>
          </div>
          <div>
            <span>{t("metric.productionWarnings")}</span>
            <strong>{warningCount}</strong>
          </div>
          <div>
            <span>{t("metric.productionBlockers")}</span>
            <strong>{blockerCount}</strong>
          </div>
        </div>
        <ol className="go-live-step-list">
          {productionSummary.steps.map((step) => (
            <li key={step.key}>
              <span className={`go-live-step-dot tone-${productionConsoleStatusTone(step.status)}`} aria-hidden="true" />
              <div>
                <strong>{t(step.labelKey)}</strong>
                <span>{step.detailKey ? t(step.detailKey) : step.detail}</span>
              </div>
            </li>
          ))}
        </ol>
      </section>
    </div>
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
                    className="table-action is-danger"
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
                {agent.description ? <span>{agent.description}</span> : <TechnicalId copyLabel={t("action.copy")} value={agent.id} />}
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
                      className="table-action is-danger"
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
      {traces.map((trace) => {
        const traceCallerName = trace.callerAgentId
          ? permissionEntityDisplayName(names[trace.callerAgentId] ?? trace.callerAgentId, t)
          : t("text.traceAnonymous");
        const traceTargetName = permissionEntityDisplayName(names[trace.targetAgentId] ?? trace.targetAgentId, t);

        return (
          <article className="trace-row" key={trace.id}>
            <div className={`trace-decision tone-${trace.decision === "allowed" ? "success" : "danger"}`}>
              {trace.decision === "allowed" ? <CheckCircle2 size={15} /> : <LockKeyhole size={15} />}
            </div>
            <div>
              <div className="trace-title-line">
                <strong>{traceCallerName} → {traceTargetName}</strong>
                <Badge tone={trace.decision === "allowed" ? "success" : "danger"}>
                  {trace.decision === "allowed" ? t("text.decisionAllowed") : t("text.decisionDenied")}
                </Badge>
              </div>
              <div className="trace-business-line">
                <span className="trace-route-text">{traceRouteBusinessLabel(trace, t)}</span>
                <span className="trace-reason">{accessTraceReasonLabel(trace.reason, trace.decision === "allowed" ? "allow" : "deny", t)}</span>
              </div>
              <details className="trace-technical-details">
                <summary>{t("text.technicalDetails")}</summary>
                <div className="trace-technical-grid">
                  <span>
                    <span>{t("form.routeType")}</span>
                    <code>{trace.routeType}</code>
                  </span>
                  <span>
                    <span>{t("form.routeKey")}</span>
                    <code>{trace.routeKey || t("text.traceDefaultRoute")}</code>
                  </span>
                  {trace.capabilityId ? <TechnicalId copyLabel={t("action.copy")} label={t("form.capability")} value={trace.capabilityId} /> : null}
                  {trace.runId ? <TechnicalId copyLabel={t("action.copy")} label={t("form.traceRunId")} value={trace.runId} /> : null}
                </div>
              </details>
            </div>
            <time>{formatDate(trace.createdAt)}</time>
          </article>
        );
      })}
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
              <td><Badge tone={auditTone(event.action)}>{auditActionLabel(event.action, t)}</Badge></td>
              <td>
                <strong>{auditResourceTypeLabel(event.resourceType, t)}</strong>
                <details className="audit-technical">
                  <summary>{t("text.auditDetails")}</summary>
                  <TechnicalId copyLabel={t("action.copy")} value={event.resourceId} />
                </details>
              </td>
              <td>{auditActorLabel(event.actor, t)}</td>
              <td>{auditCredentialVersion(event)}</td>
              <td>{auditSummaryLabel(event.summary, t)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function IconMore({ title = "More" }: { title?: string }) {
  return (
    <button aria-label={title} className="icon-button compact" title={title} type="button">
      <MoreHorizontal size={16} />
    </button>
  );
}

function IconOpen({ title = "Open" }: { title?: string }) {
  return (
    <button aria-label={title} className="icon-button compact" title={title} type="button">
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

function permissionTenantPathLabel(tenantId: string, tenants: Tenant[], t: Translator): { path: string; primary: string } {
  const normalizedTenantId = tenantId.trim();
  if (!normalizedTenantId) return { path: "-", primary: t("text.unresolvedTenant") };
  const tenantById = tenants.reduce<Record<string, Tenant>>((acc, tenant) => {
    acc[tenant.id] = tenant;
    return acc;
  }, {});
  const selected = tenantById[normalizedTenantId];
  if (!selected) {
    if (normalizedTenantId === defaultManagementScope.tenantId) {
      const defaultTenantName = t("text.defaultTenantName");
      return { path: defaultTenantName, primary: defaultTenantName };
    }
    return {
      path: tx(t, "text.unresolvedTenantDetail", { id: normalizedTenantId }),
      primary: t("text.unresolvedTenant")
    };
  }

  const path: Tenant[] = [];
  const visited = new Set<string>();
  let current: Tenant | undefined = selected;
  while (current && !visited.has(current.id)) {
    path.unshift(current);
    visited.add(current.id);
    current = current.parentTenantId ? tenantById[current.parentTenantId] : undefined;
  }
  const names = path.map((tenant) => permissionEntityDisplayName(tenant.name.trim() || tenant.id, t));
  return {
    path: names.join(" > "),
    primary: permissionEntityDisplayName(selected.name.trim() || selected.id, t)
  };
}

function permissionWorkspaceDisplayName(workspaceId: string, agents: Agent[], t: Translator) {
  const normalizedWorkspaceId = workspaceId.trim();
  if (!normalizedWorkspaceId) return "-";
  const agentInWorkspace = agents.find((agent) => agent.workspaceId === normalizedWorkspaceId);
  if (normalizedWorkspaceId === defaultManagementScope.workspaceId || agentInWorkspace?.workspaceId === defaultManagementScope.workspaceId) {
    return t("text.defaultWorkspaceName");
  }
  if (/permission[-_]?(request|package)[-_]?approval/i.test(normalizedWorkspaceId)) {
    return t("demo.permissionRequestWorkspace");
  }
  if (/core[-_]?journey/i.test(normalizedWorkspaceId)) {
    return t("demo.coreJourneyWorkspace");
  }
  return permissionEntityDisplayName(readableIdentifierLabel(normalizedWorkspaceId), t);
}

function uniquePermissionEntityOptions<T extends { id: string }>(
  entities: T[],
  selectedId: string,
  labelFor: (entity: T) => string
): Array<{ id: string; label: string }> {
  const selected = entities.find((entity) => entity.id === selectedId);
  const ordered = selected ? [selected, ...entities.filter((entity) => entity.id !== selected.id)] : entities;
  const seen = new Set<string>();
  const options: Array<{ id: string; label: string }> = [];

  for (const entity of ordered) {
    const label = labelFor(entity).trim() || entity.id;
    const key = label.toLocaleLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    options.push({ id: entity.id, label });
  }

  return options;
}

function auditCredentialVersion(event: AuditEvent) {
  const value = event.metadata?.credentialVersion;
  if (typeof value === "number" || typeof value === "string") return String(value);
  return "-";
}

function auditActionLabel(action: string, t: Translator) {
  return t(`auditAction.${action}`, readableIdentifierLabel(action));
}

function auditActorLabel(actor: string | undefined, t: Translator) {
  if (!actor) return "-";
  return t(`auditActor.${actor}`, actor);
}

function auditResourceTypeLabel(resourceType: string, t: Translator) {
  return t(`auditResource.${resourceType}`, readableIdentifierLabel(resourceType));
}

function auditSummaryLabel(summary: string | undefined, t: Translator) {
  if (!summary) return "-";
  const key = summary.trim().replaceAll(" ", "_").toLowerCase();
  return t(`auditSummary.${key}`, summary);
}

function channelLabel(channelType: string, channelLabels: Record<string, string>, t: Translator) {
  return t(`value.${channelType}`, channelLabels[channelType] ?? channelType);
}

function evidenceStatusLabel(status: EvidenceRun["status"], t: Translator) {
  if (status === "passed") return t("status.evidencePassed");
  if (status === "failed") return t("status.evidenceFailed");
  return t("status.evidenceWarning");
}

function policyRetryText(policy: RoutePolicy, t: Translator) {
  if (!policy.retry) return t("text.targetRetry");
  const statuses = policy.retry.statusCodes.length > 0 ? policy.retry.statusCodes.join("/") : t("text.retryNone");
  return `retry ${policy.retry.maxAttempts}x ${policy.retry.backoffMs}ms ${statuses}`;
}

function permissionPackageTemplateName(template: PermissionPackageTemplate, t: Translator) {
  return t(`permissionPackage.${template.id}.name`, template.name);
}

function permissionPackageTemplateNameById(templateId: string, t: Translator) {
  return t(`permissionPackage.${templateId}.name`, templateId);
}

function permissionPackageTemplateSummary(template: PermissionPackageTemplate, t: Translator) {
  return t(`permissionPackage.${template.id}.summary`, template.summary);
}

function permissionDraftStatus(draft: PermissionPackageDraft): { labelKey: string; tone: Tone } {
  if (!draft.readiness.canApply) {
    return { labelKey: "status.needsReview", tone: "warning" };
  }
  if (!draft.policyGate.canApplyDirectly) {
    return { labelKey: "status.approvalPending", tone: "warning" };
  }
  return { labelKey: "status.readyToApply", tone: "success" };
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
    callerInstanceId: t("form.caller"),
    subjectSelector: t("form.accessSubject"),
    targetId: t("form.target"),
    tenantId: t("form.tenant"),
    workspaceId: t("form.workspace")
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
    if (key === "capability") {
      acc[key] = capabilityKeyDisplayName(value, t);
    } else if (key === "action" || key === "risk" || key === "sensitivity") {
      acc[key] = translatedValue(t, value);
    } else {
      acc[key] = value;
    }
    return acc;
  }, {});
  return tx(t, reason.reasonKey, values);
}

function firstBlockingApplyPreflightCheck(preflight: PermissionPackageApplyPreflight): PermissionPackageApplyPreflightCheck | null {
  return preflight.checks.find((check) => check.severity === "blocking") ?? preflight.checks[0] ?? null;
}

function permissionApplyPreflightCheckLabel(code: string, t: Translator) {
  return t(`permissionPreflight.${code}`, code.replaceAll("_", " "));
}

function permissionApplyPreflightCheckMessage(check: PermissionPackageApplyPreflightCheck | null, t: Translator) {
  if (!check) return t("message.permissionPackagePreflightNoDetail");
  return t(`permissionPreflight.detail.${check.code}`, check.message || permissionApplyPreflightCheckLabel(check.code, t));
}

function permissionApplyPreflightNextAction(action: string, t: Translator) {
  const keyByAction: Record<string, string> = {
    "Apply this permission package when the reviewer is ready.": "permissionPreflight.next.applyWhenReady",
    "Apply this permission request when the reviewer is ready.": "permissionPreflight.next.applyWhenReady",
    "Create and approve a permission package approval request, then preflight again with approvalRequestId.": "permissionPreflight.next.createApproval",
    "Create and approve an approval request for this permission request, then preflight again with approvalRequestId.": "permissionPreflight.next.createApproval",
    "Fix draft readiness blockers before applying this permission package.": "permissionPreflight.next.fixDraft",
    "Fix draft readiness blockers before applying this permission request.": "permissionPreflight.next.fixDraft",
    "Narrow region or data scopes so the package stays inside every capability boundary.": "permissionPreflight.next.narrowScope",
    "Refresh approval or create a new approval request for the current draft.": "permissionPreflight.next.refreshApproval",
    "Review existing grant chains before applying another permission package for the same caller and capability.": "permissionPreflight.next.reviewExistingGrants",
    "Review existing grant chains before applying another permission request for the same caller and capability.": "permissionPreflight.next.reviewExistingGrants",
    "Use an approved approvalRequestId that matches the current draft.": "permissionPreflight.next.useApprovedRequest"
  };
  return keyByAction[action] ? t(keyByAction[action], action) : action;
}

function permissionApplyPreflightSeverityLabel(severity: PermissionPackageApplyPreflightCheck["severity"], t: Translator) {
  if (severity === "blocking") return t("status.preflightBlocking");
  if (severity === "warning") return t("status.preflightWarning");
  if (severity === "passed") return t("status.preflightPassed");
  return t("status.preflightInfo");
}

function permissionApplyPreflightTone(severity: PermissionPackageApplyPreflightCheck["severity"]): Tone {
  if (severity === "blocking") return "danger";
  if (severity === "warning") return "warning";
  if (severity === "passed") return "success";
  return "info";
}

function permissionApplyPreflightSeverityRank(severity: PermissionPackageApplyPreflightCheck["severity"]) {
  if (severity === "blocking") return 4;
  if (severity === "warning") return 3;
  if (severity === "info") return 2;
  return 1;
}

function permissionApprovalStatusLabel(status: PermissionPackageApprovalRequest["status"], t: Translator) {
  if (status === "approved") return t("status.approvalApproved");
  if (status === "rejected") return t("status.approvalRejected");
  if (status === "withdrawn") return t("status.approvalWithdrawn");
  return t("status.approvalPending");
}

function permissionApprovalStatusTone(status: PermissionPackageApprovalRequest["status"]): Tone {
  if (status === "approved") return "success";
  if (status === "rejected") return "danger";
  if (status === "withdrawn") return "warning";
  return "warning";
}

function permissionApplicationHealthLabel(status: PermissionPackageApplicationHealthStatus, t: Translator) {
  if (status === "ready") return t("status.applicationHealthReady");
  if (status === "drifted") return t("status.applicationHealthDrifted");
  return t("status.applicationHealthNeedsReview");
}

function permissionApplicationHealthTone(status: PermissionPackageApplicationHealthStatus): Tone {
  if (status === "ready") return "success";
  if (status === "drifted") return "warning";
  return "info";
}

function permissionApplicationHealthRowSummary(row: PermissionPackageApplicationHealthRow, t: Translator) {
  if (row.status === "ready") return t("text.applicationHealthReadyDetail");
  if (row.status === "drifted") return t("text.applicationHealthDriftedDetail");
  return t("text.applicationHealthNeedsReviewDetail");
}

function productionReadinessStatusLabel(status: PermissionPackageProductionReadinessStatus | undefined, t: Translator) {
  if (status === "ready") return t("status.productionReady");
  if (status === "needs_review") return t("status.productionNeedsReview");
  if (status === "blocked") return t("status.productionBlocked");
  return t("status.preflightPending");
}

function productionReadinessStatusTone(status: PermissionPackageProductionReadinessStatus | undefined): Tone {
  if (status === "ready") return "success";
  if (status === "needs_review") return "warning";
  if (status === "blocked") return "danger";
  return "neutral";
}

function permissionProductionReadinessCheckLabel(code: string, t: Translator) {
  return t(`productionCheck.${code}`, code.replaceAll("_", " "));
}

function permissionProductionReadinessCheckMessage(check: PermissionPackageProductionReadinessCheck, t: Translator) {
  return t(`productionCheck.detail.${check.code}`, check.message);
}

function permissionProductionReadinessNextAction(action: string, t: Translator) {
  const known: Record<string, string> = {
    "Apply the approved permission package before production readiness.": "productionNext.applyApproved",
    "Inspect the latest permission package application scope before go-live.": "productionNext.inspectScope",
    "Production readiness evidence is complete.": "productionNext.complete",
    "Resolve apply preflight blockers before claiming production readiness.": "productionNext.resolvePreflight",
    "Resolve impact review blockers before production readiness.": "productionNext.resolveImpact",
    "Review application health and drift blockers before production readiness.": "productionNext.reviewHealth",
    "Run a denied MCP call that proves blocked tools stay blocked.": "productionNext.runDenied",
    "Run an allowed MCP call with the production subject before go-live.": "productionNext.runAllowed",
    "Verify permission package applied audit evidence before production readiness.": "productionNext.verifyAudit",
    "Verify tenant entitlement, workspace assignment, and caller assignment evidence.": "productionNext.verifyGrantChain"
  };
  const key = known[action];
  return key ? t(key) : action;
}

function productionEvidenceReportFilename(report: PermissionPackageProductionEvidenceReport) {
  const generated = safeFilenameSegment(report.generatedAt || new Date().toISOString());
  return [
    "agentharbor-production-evidence",
    safeFilenameSegment(report.scope.tenantId),
    safeFilenameSegment(report.scope.workspaceId),
    safeFilenameSegment(report.scope.templateId),
    safeFilenameSegment(report.status),
    generated
  ].filter(Boolean).join("-") + ".json";
}

function safeFilenameSegment(value: string | undefined) {
  return (value ?? "")
    .trim()
    .replace(/[^a-zA-Z0-9._-]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80);
}

function downloadJson(value: unknown, filename: string) {
  const blob = new Blob([JSON.stringify(value, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
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
    ?? matching.find((request) => request.status === "withdrawn")
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
  const subjectSelector = form.subjectSelector.trim();

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

function productionConsoleStatusTone(status: AiAdminProductionConsoleStatus): Tone {
  if (status === "ready") return "success";
  if (status === "needs_review") return "warning";
  if (status === "blocked") return "danger";
  return "neutral";
}

function permissionWorkbenchActionKey(code: string | undefined, fallback: string) {
  switch (code) {
    case "complete_request":
      return "action.completePermissionRequest";
    case "create_approval_request":
      return "action.createApprovalRequest";
    case "review_approval_request":
      return "action.reviewApprovalRequest";
    case "apply_permission_package":
      return "action.applyPermissionPackage";
    case "run_runtime_validation":
      return "action.runApprovalJourney";
    case "export_production_evidence":
      return "action.exportProductionEvidence";
    default:
      return fallback;
  }
}

function permissionWorkbenchStatusLabelKey(status: string | undefined) {
  if (!status) return "permissionWorkbench.status.pending";
  return `permissionWorkbench.status.${status}`;
}

function permissionWorkbenchStatusDetailKey(status: string | undefined) {
  if (!status) return "permissionWorkbench.statusDetail.pending";
  return `permissionWorkbench.statusDetail.${status}`;
}

function permissionWorkbenchStatusTone(status: string | undefined, fallback: AiAdminProductionConsoleStatus): Tone {
  if (status === "production_ready") return "success";
  if (status === "blocked") return "danger";
  if (status === "ready_to_apply" || status === "validating" || status === "awaiting_approval") return "warning";
  if (status === "needs_input") return "neutral";
  return productionConsoleStatusTone(fallback);
}

function permissionWorkbenchStepLabelKey(key: string) {
  return `permissionWorkbench.step.${key}`;
}

function permissionWorkbenchStepDetailKey(detailCode: string) {
  return `permissionWorkbench.detail.${detailCode}`;
}

function productionConsoleStatusLabel(summary: AiAdminProductionConsoleSummary, t: Translator) {
  if (summary.status === "ready") return t("status.productionReady");
  if (summary.status === "needs_review") return t("status.productionNeedsReview");
  if (summary.status === "blocked") return t("status.productionBlocked");
  if (summary.primaryActionKey === "action.createApprovalRequest") return t("status.approvalPending");
  return t("status.productionPending");
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
