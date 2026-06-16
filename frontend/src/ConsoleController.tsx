import { useEffect, useMemo, useReducer, useRef, useState, type FormEvent, type ReactNode } from "react";
import {
  Building2,
  Boxes,
  ClipboardCheck,
  DatabaseZap,
  FileSearch,
  Gauge,
  KeyRound,
  LogOut,
  Layers3,
  LockKeyhole,
  Network,
  RefreshCw,
  Route,
  ShieldCheck,
  Workflow
} from "lucide-react";
import {
  ApiRequestError,
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
  createTenant,
  createTenantEntitlement,
  createWorkspaceAssignment,
  defaultMockMcpHealthUrl,
  fetchAccessDecisionExplanation,
  fetchAuditEvents,
  fetchPermissionPackageAccessSubjects,
  fetchPermissionPackageApplicationHealth,
  fetchPermissionPackageApplicationImpact,
  fetchPermissionPackageApprovalRequests,
  fetchPermissionPackageProductionEvidenceReport,
  fetchPermissionPackageProductionReadiness,
  fetchPermissionPackageTemplates,
  fetchTenantPermissionCenter,
  isApiCompatibilityFallbackError,
  loadConsoleData,
  loadTenantAccessProfile,
  preflightPermissionPackage,
  previewPermissionPackageWorkbench,
  rejectPermissionPackageApprovalRequest,
  refreshTargetCapabilities,
  updateAgent,
  updateCapability,
  withdrawPermissionPackageApprovalRequest
} from "./api";
import {
  runtimeEvidenceMetric
} from "./consoleMetrics";
import type { ConnectionDiagnosticRow } from "./connectionDiagnostics";
import {
  capabilityDisplayName,
  capabilityKeyDisplayName,
  capabilityDiscoveryStatusLabel,
  capabilityStatusTone,
  dataScopeText,
  permissionEntityDisplayName,
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
  healthCheckFailureDetail
} from "./healthCheckPresentation";
import {
  resourcePermissionIntent
} from "./permissionWorkbenchPresenters";
import {
  buildResourceLifecycleSummary,
  type ResourceLifecycleItem
} from "./resourceLifecycle";
import {
  planResourceLifecycleAction,
  type ResourceLifecycleActionContext,
  type ResourceLifecycleModal
} from "./resourceLifecycleActionPlanner";
import {
  deriveProductionJourney
} from "./productionJourney";
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
  evaluateCoreJourney,
  type CoreJourneyEvaluation
} from "./coreJourney";
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
  type AiAdminProductionConsoleStatus
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
  type PermissionPackageProductionReadinessFilter,
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
import { AiAdminPermissionWorkbench } from "./components/AiAdminPermissionWorkbench";
import { CapabilityGovernanceView } from "./components/CapabilityGovernanceView";
import { ActionModalButton, IconMore, IconOpen, Panel } from "./components/ConsolePrimitives";
import { ProductionJourneyCheckpoint } from "./components/ProductionJourneyCheckpoint";
import {
  AccessView,
  AdminAccessView,
  AiAdminView,
  AskView,
  CapabilitiesView,
  CockpitView,
  EvidenceView,
  GettingStartedConsoleView,
  PoliciesView,
  RegistryView,
  RoutesView,
  TenantsView,
  TracesView
} from "./components/ConsoleViews";
import { AdminAccessManagementView } from "./components/AdminAccessManagementView";
import { AskAccessPanel } from "./components/AskAccessView";
import { CoreJourneyWorkbench } from "./components/CoreJourneyWorkbench";
import { GettingStartedView } from "./components/GettingStartedView";
import { ConsoleLoginView } from "./components/ConsoleLoginView";
import { ResourceLifecycleView } from "./components/ResourceLifecycleView";
import {
  AgentCreateForm,
  CredentialRotateForm,
  KeyCreateForm,
  PolicyCreateForm,
  TraceFilterBar
} from "./components/ManagementForms";
import { GoLiveAcceptanceOverview } from "./components/GoLiveAcceptanceOverview";
import {
  AgentTable,
  ContractMatrix,
  PolicyTable
} from "./components/OperationalViews";
import {
  EvidenceTimeline,
  ManagementAuditTable,
  SignalBoard,
  TraceTable
} from "./components/RuntimeEvidenceViews";
import { TechnicalId } from "./components/TechnicalId";
import { TenantAccessProfileView } from "./components/TenantAccessProfileView";
import { TenantOrganizationView, type TenantWorkspaceContext } from "./components/TenantOrganizationView";
import { Badge } from "./components/ui";
import { useAccessProfileController } from "./hooks/useAccessProfileController";
import { useAdminAccessController } from "./hooks/useAdminAccessController";
import { useAskAccessController } from "./hooks/useAskAccessController";
import { useCapabilityGovernanceController } from "./hooks/useCapabilityGovernanceController";
import { useConnectionDiagnostics } from "./hooks/useConnectionDiagnostics";
import { useConsoleAuth } from "./hooks/useConsoleAuth";
import { useCoreJourneyController } from "./hooks/useCoreJourneyController";
import { useManagementOperations } from "./hooks/useManagementOperations";
import { gettingStartedSteps, resolveDefaultNavKey } from "./gettingStarted";
import type {
  AccessProfileFilters,
  AccessProfileHandoffContext,
  AccessProfileSummary,
  AccessDecisionExplainRequest,
  AccessDecisionExplainResult,
  AskHandoffContext,
  Agent,
  AgentStatus,
  AuditEvent,
  Capability,
  ConsoleSession,
  ConsoleData,
  CreateAgentKeyResponse,
  DataScope,
  InstanceAssignment,
  JsonObject,
  ManagementScope,
  PermissionChangeHandoffContext,
  RoutePolicy,
  Tenant,
  TenantAccessProfileData,
  TenantEntitlement,
  TenantPermissionCenterResponse,
  TraceDecision,
  TraceFilters,
  WorkspaceAssignment
} from "./types";

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
const defaultAiAdminForm: PermissionPackageDraftInput = {
  ...defaultPermissionPackageDraftInput,
  tenantId: defaultManagementScope.tenantId,
  workspaceId: defaultManagementScope.workspaceId
};

interface TenantOrganizationConsoleState {
  permissionCenter: TenantPermissionCenterResponse | null;
  permissionCenterError: string;
  permissionCenterLoading: boolean;
  selectedTenantId: string;
}

const defaultTenantOrganizationConsoleState: TenantOrganizationConsoleState = {
  permissionCenter: null,
  permissionCenterError: "",
  permissionCenterLoading: false,
  selectedTenantId: ""
};

function tenantOrganizationConsoleStateReducer(
  current: TenantOrganizationConsoleState,
  patch: Partial<TenantOrganizationConsoleState>
): TenantOrganizationConsoleState {
  const changed = (Object.keys(patch) as Array<keyof TenantOrganizationConsoleState>).some(
    (key) => current[key] !== patch[key]
  );
  return changed ? { ...current, ...patch } : current;
}

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

function initialHashNavKey(): NavKey | null {
  if (typeof window === "undefined") {
    return null;
  }
  return navKeyFromHash(window.location.hash);
}

function initialNavKey(): NavKey {
  return initialHashNavKey() ?? defaultNavKey;
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
type ResourceActionModal = "" | ResourceLifecycleModal;

interface ResourceActionRequest {
  context: ResourceLifecycleActionContext | null;
  modal: ResourceActionModal;
  openToken: number;
}

const defaultResourceActionRequest: ResourceActionRequest = { context: null, modal: "", openToken: 0 };

function resourceActionRequestReducer(
  current: ResourceActionRequest,
  action: { context?: ResourceLifecycleActionContext | null; modal: ResourceActionModal }
): ResourceActionRequest {
  return { context: action.context ?? null, modal: action.modal, openToken: current.openToken + 1 };
}

function navIconFor(key: NavKey) {
  switch (key) {
    case "getting-started":
      return Workflow;
    case "ask":
      return FileSearch;
    case "ai-admin":
      return ShieldCheck;
    case "admin-access":
      return KeyRound;
    case "tenants":
      return Building2;
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

function sessionRoleLabel(session: ConsoleSession | null, t: Translator) {
  const role = session?.role?.trim() || "platform_admin";
  return t(`auth.role.${role}`, role);
}

function sessionScopeLabel(session: ConsoleSession | null, t: Translator) {
  const tenantId = session?.tenantId?.trim();
  const workspaceId = session?.workspaceId?.trim();
  if (!tenantId && !workspaceId) {
    return t("auth.scope.allTenants");
  }
  if (tenantId && workspaceId) {
    return tx(t, "auth.scope.tenantWorkspace", { tenantId, workspaceId });
  }
  return tenantId || workspaceId || t("auth.scope.allTenants");
}

function connectionDiagnosticDetail(row: ConnectionDiagnosticRow, t: Translator) {
  return row.detailParams ? tx(t, row.detailKey, row.detailParams) : t(row.detailKey);
}

function connectionDiagnosticSummaryLabel(status: "ok" | "warning" | "error", t: Translator) {
  if (status === "ok") return t("connectionDiagnostics.summaryOk");
  if (status === "warning") return t("connectionDiagnostics.summaryWarning");
  return t("connectionDiagnostics.summaryError");
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

function isConsumedApprovalRetryError(error: unknown) {
  return error instanceof ApiRequestError && error.code === "PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED";
}

function localizedErrorMessage(t: Translator, language: Language, error: unknown, fallbackKey: string) {
  const fallback = t(fallbackKey);
  if (!(error instanceof Error) || !error.message.trim()) return fallback;
  if (language === "en" || /[\u4e00-\u9fa5]/.test(error.message)) {
    return error.message;
  }
  return fallback;
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

export function ConsoleController() {
  const approvalCreateInFlightRef = useRef(false);
  const approvalResolveBlockedRef = useRef(false);
  const approvalResolveCooldownTimerRef = useRef<number | null>(null);
  const defaultNavResolvedRef = useRef(initialHashNavKey() !== null);
  const userSelectedNavRef = useRef(false);
  const [activeNav, setActiveNav] = useState<NavKey>(initialNavKey);
  const [connectionMenuOpen, setConnectionMenuOpen] = useState(false);
  const [adminKey, setAdminKey] = useState("");
  const consoleAuth = useConsoleAuth();
  const [scope, setScope] = useState<ManagementScope>(defaultManagementScope);
  const [data, setData] = useState<ConsoleData | null>(null);
  const [loadError, setLoadError] = useState("");
  const [lastRefresh, setLastRefresh] = useState(new Date());
  const [traceFilters, setTraceFilters] = useState<TraceFilters>(defaultTraceFilters);
  const [language, setLanguage] = useState<Language>(initialLanguage);
  const [handoffContexts, setHandoffContexts] = useState<{ ask: AskHandoffContext | null; permissionChange: PermissionChangeHandoffContext | null; permissionNotice: PermissionChangeHandoffContext | null }>({ ask: null, permissionChange: null, permissionNotice: null });
  const [resourceActionRequest, dispatchResourceActionRequest] = useReducer(
    resourceActionRequestReducer,
    defaultResourceActionRequest
  );
  const resourceActionContext = resourceActionRequest.context;
  const resourceActionModal = resourceActionRequest.modal;
  const resourceActionOpenToken = resourceActionRequest.openToken;
  const [tenantOrganizationState, setTenantOrganizationState] = useReducer(
    tenantOrganizationConsoleStateReducer,
    defaultTenantOrganizationConsoleState
  );
  const tenantOrganizationSelectedTenantId = tenantOrganizationState.selectedTenantId;
  const tenantPermissionCenter = tenantOrganizationState.permissionCenter;
  const tenantPermissionCenterLoading = tenantOrganizationState.permissionCenterLoading;
  const tenantPermissionCenterError = tenantOrganizationState.permissionCenterError;
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
  const connectionDiagnostics = useConnectionDiagnostics({
    liveDataLoaded: Boolean(data?.loadedFromApi),
    loadError,
    mcpEndpoint: aiAdminApprovalJourneyConfig.mcpEndpoint
  });
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
  const renderedConsoleLoginMessage = localizedMessageText(consoleAuth.loginMessage, t, language);
  const consoleAccessReady = consoleAuth.accessReady;
  function setTenantOrganizationSelectedTenantId(tenantId: string) {
    setTenantOrganizationState({ selectedTenantId: tenantId });
  }
  function selectActiveNav(key: NavKey) {
    userSelectedNavRef.current = true;
    setActiveNav(key);
  }
  function openAskAccess(context: AskHandoffContext) {
    setHandoffContexts((current) => ({ ...current, ask: context }));
    userSelectedNavRef.current = true;
    setActiveNav("ask");
  }
  function openTenantPermissionChange(context: PermissionChangeHandoffContext) {
    setHandoffContexts((current) => ({ ...current, permissionChange: context }));
    userSelectedNavRef.current = true;
    setActiveNav("ai-admin");
  }
  function openResourceActionModal(modal: ResourceActionModal, context: ResourceLifecycleActionContext | null = null) {
    dispatchResourceActionRequest({ context, modal });
  }
  const management = useManagementOperations({
    adminKey,
    defaultScope: defaultManagementScope,
    language,
    onRefresh: refresh,
    scope,
    t
  });
  const adminAccess = useAdminAccessController({
    adminKey,
    enabled: consoleAccessReady
  });
  const capabilityGovernance = useCapabilityGovernanceController({
    adminKey,
    data,
    defaultScope: defaultManagementScope,
    language,
    onRefresh: refresh,
    setData,
    t
  });
  function handleResourceLifecycleAction(item: ResourceLifecycleItem) {
    const plan = planResourceLifecycleAction({
      agents,
      formatEntityName: (name) => permissionEntityDisplayName(name, t),
      formatPermissionIntent: (targetName) => resourcePermissionIntent(targetName, t),
      formatTenantName: (tenantId) => permissionTenantPathLabel(tenantId, tenants, t).primary,
      formatWorkspaceName: (workspaceId) => permissionWorkspaceDisplayName(workspaceId, agents, t),
      item,
      localCallers,
      mcpTargets
    });
    if (plan.kind === "open_modal") {
      if (plan.modal === "create_key" && plan.agentId) {
        management.setKeyForm({
          ...management.keyForm,
          agentId: plan.agentId
        });
      }
      if (plan.modal === "rotate_credential" && plan.agentId) {
        management.setRotateForm({
          ...management.rotateForm,
          agentId: plan.agentId
        });
      }
      if (plan.modal === "create_policy") {
        management.setPolicyForm({
          ...management.policyForm,
          callerAgentId: plan.callerAgentId ?? management.policyForm.callerAgentId,
          targetAgentId: plan.targetAgentId ?? management.policyForm.targetAgentId
        });
      }
      openResourceActionModal(plan.modal, plan.context);
      return;
    }
    if (plan.kind === "capability_prefill") {
      capabilityGovernance.setForm({
        ...capabilityGovernance.form,
        targetId: plan.targetId
      });
      userSelectedNavRef.current = true;
      setActiveNav(plan.navKey);
      return;
    }
    if (plan.kind === "permission_handoff") {
      openTenantPermissionChange(plan.context);
      return;
    }
    if (plan.kind === "runtime_filters") {
      setTraceFilters((current) => ({ ...current, ...plan.traceFilters }));
      userSelectedNavRef.current = true;
      setActiveNav(plan.navKey);
      return;
    }
    userSelectedNavRef.current = true;
    setActiveNav(plan.navKey);
  }
  const accessProfileController = useAccessProfileController({
    activeNav,
    adminKey,
    defaultScope: defaultManagementScope,
    enabled: consoleAccessReady,
    language,
    scope,
    setScope,
    t
  });
  const askAccess = useAskAccessController({
    adminKey, consoleData: data, handoffContext: handoffContexts.ask, language, liveDataAvailable: Boolean(data?.loadedFromApi),
    onConsumeHandoff: () => setHandoffContexts((current) => ({ ...current, ask: null })),
    onStartPermissionChange: (context) => { setHandoffContexts((current) => ({ ...current, permissionChange: context })); userSelectedNavRef.current = true; setActiveNav("ai-admin"); },
    t, templates: aiAdminTemplates
  });
  useEffect(() => {
    const context = handoffContexts.permissionChange;
    if (!context) return;
    setAiAdminNewDraftMode(true);
    setAiAdminForm((current) => ({
      ...current,
      callerInstanceId: context.callerInstanceId ?? current.callerInstanceId,
      requestText: context.intentText ?? current.requestText,
      subjectSelector: context.subjectId ?? current.subjectSelector,
      targetId: context.targetId ?? current.targetId,
      templateId: context.templateId ?? current.templateId,
      tenantId: context.tenantId,
      workspaceId: context.workspaceId
    }));
    setAiAdminServerDraft(null);
    setAiAdminWorkbenchPreview(null);
    setAiAdminApplication(null);
    setAiAdminApplyPreflight(null);
    setAiAdminProductionReadiness(null);
    setAiAdminApprovalRequests([]);
    setAiAdminSelectedApprovalRequestId("");
    setAiAdminAccessDecisionExplanation(null);
    setAiAdminMessage(null);
    setHandoffContexts((current) => ({ ...current, permissionChange: null, permissionNotice: context }));
  }, [handoffContexts.permissionChange]);
  const coreJourney = useCoreJourneyController({
    adminKey,
    defaultAccessFilters: defaultAccessProfileFilters,
    defaultScope: defaultManagementScope,
    defaultTraceFilters,
    enabled: consoleAccessReady,
    language,
    setAccessFilters: accessProfileController.updateFilters,
    setAccessProfile: accessProfileController.setProfile,
    setData,
    setLastRefresh,
    setLoadError,
    setScope,
    setTraceFilters,
    t
  });

  useEffect(() => {
    if (!consoleAccessReady) return;
    void refresh();
  }, [consoleAccessReady]);

  useEffect(() => () => {
    if (approvalResolveCooldownTimerRef.current !== null) {
      window.clearTimeout(approvalResolveCooldownTimerRef.current);
    }
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
    setConnectionMenuOpen(false);
  }, [activeNav]);

  const shouldLoadAiAdminCatalog =
    consoleAccessReady && (activeNav === "ask" || activeNav === "ai-admin" || activeNav === "evidence" || activeNav === "tenants");

  useEffect(() => {
    if (shouldLoadAiAdminCatalog) {
      void refreshAiAdminCatalog();
      void refreshAiAdminApprovalReadiness();
    }
  }, [shouldLoadAiAdminCatalog]);

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

  const shouldLoadAiAdminWorkbenchPreview = consoleAccessReady && (activeNav === "ai-admin" || activeNav === "evidence");

  useEffect(() => {
    if (!shouldLoadAiAdminWorkbenchPreview || !data?.loadedFromApi || aiAdminNewDraftMode) {
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

  function handleConsoleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void consoleAuth.login(() => {
      setData(null);
      setLoadError("");
    });
  }

  function handleConsoleLogout() {
    void consoleAuth.logout((nextSession) => {
      if (nextSession.requiresLogin && !nextSession.authenticated) {
        setData(null);
        setLoadError("");
      }
      connectionDiagnostics.reset();
    });
  }

  async function refresh() {
    if (!consoleAccessReady) return;
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
      await accessProfileController.refresh();
    }
  }

  async function refreshAiAdminCatalog() {
    if (!consoleAccessReady) return;
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
    accessProfileController.clearForPermissionChangeHandoff({
      ...accessProfileController.filters,
      capabilityId: selectedCapability?.id ?? "",
      callerInstanceId: aiAdminForm.callerInstanceId,
      subjectId,
      targetId: aiAdminForm.targetId,
      traceLimit: accessProfileController.filters.traceLimit || "20",
      workspaceId: aiAdminForm.workspaceId
    }, {
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
    setActiveNav("access");
  }

  function openTenantAccessProfile(context: TenantWorkspaceContext) {
    setScope((current) => ({
      ...current,
      tenantId: context.tenantId,
      workspaceId: context.workspaceId
    }));
    accessProfileController.clearForPermissionChangeHandoff({
      ...accessProfileController.filters,
      traceLimit: accessProfileController.filters.traceLimit || "20",
      workspaceId: context.workspaceId
    }, {
      tenantId: context.tenantId,
      tenantName: context.tenantName,
      tenantPath: context.tenantPath,
      workspaceId: context.workspaceId,
      workspaceName: context.workspaceName
    });
    userSelectedNavRef.current = true;
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
    setHandoffContexts((current) => ({ ...current, permissionChange: null, permissionNotice: null }));
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
      apiHealth.status === "ok" ? "" : healthCheckFailureDetail(t, t("readiness.aiAdmin.api.title"), apiHealth),
      mockMcpHealth.status === "ok" ? "" : healthCheckFailureDetail(t, t("readiness.aiAdmin.mockMcp.title"), mockMcpHealth),
      subjectHeaderHealth.status === "ok" ? "" : healthCheckFailureDetail(t, t("readiness.aiAdmin.subjectHeader.title"), subjectHeaderHealth)
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
      accessProfileController.updateFilters(nextAccessFilters);
      setData(nextData);
      accessProfileController.setProfile(nextProfile);
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
      if (isConsumedApprovalRetryError(error)) {
        await Promise.allSettled([
          refreshAiAdminApplicationHealth(aiAdminForm, { requireLiveApi: false }),
          refreshAiAdminProductionReadiness(aiAdminForm, { requireLiveApi: false }),
          loadAiAdminApprovalRequestsForDraft(aiAdminDraft)
        ]);
        setAiAdminMessage({ key: "message.permissionApprovalAlreadyConsumedRecovery" });
        return;
      }
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
  const tenantOrganizationEffectiveTenantId = tenantOrganizationSelectedTenantId || scope.tenantId || tenants[0]?.id || "";

  useEffect(() => {
    if (tenants.length === 0) {
      setTenantOrganizationSelectedTenantId("");
      return;
    }
    if (tenantOrganizationSelectedTenantId && tenants.some((tenant) => tenant.id === tenantOrganizationSelectedTenantId)) {
      return;
    }
    const scopedTenant = scope.tenantId && tenants.some((tenant) => tenant.id === scope.tenantId)
      ? scope.tenantId
      : "";
    setTenantOrganizationSelectedTenantId(scopedTenant || tenants[0].id);
  }, [scope.tenantId, tenantOrganizationSelectedTenantId, tenants]);

  useEffect(() => {
    if (!consoleAccessReady || activeNav !== "tenants" || tenants.length === 0 || !tenantOrganizationEffectiveTenantId) {
      setTenantOrganizationState({ permissionCenterLoading: false });
      return;
    }

    const controller = new AbortController();
    let active = true;
    setTenantOrganizationState({
      permissionCenter: null,
      permissionCenterError: "",
      permissionCenterLoading: true
    });
    fetchTenantPermissionCenter(tenantOrganizationEffectiveTenantId, undefined, adminKey, controller.signal)
      .then((center) => {
        if (active) setTenantOrganizationState({ permissionCenter: center });
      })
      .catch((error) => {
        if (error instanceof Error && error.name === "AbortError") return;
        if (!active) return;
        setTenantOrganizationState({
          permissionCenter: null,
          permissionCenterError: error instanceof Error
            ? error.message
            : createTranslator(language)("error.loadTenantPermissionCenter")
        });
      })
      .finally(() => {
        if (active) setTenantOrganizationState({ permissionCenterLoading: false });
      });
    return () => {
      active = false;
      controller.abort();
    };
  }, [activeNav, adminKey, consoleAccessReady, language, tenantOrganizationEffectiveTenantId, tenants.length]);

  const setupSteps = data ? gettingStartedSteps(data) : [];
  const resourceLifecycleSummary = useMemo(
    () =>
      buildResourceLifecycleSummary({
        agents,
        capabilities,
        instanceAssignments,
        routePolicies: policies,
        tenantEntitlements,
        traces,
        workspaceAssignments
      }),
    [agents, capabilities, instanceAssignments, policies, tenantEntitlements, traces, workspaceAssignments]
  );
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
  const connectionDiagnosticsStatus = connectionDiagnostics.status;
  const activeView = viewForNav(activeNav);
  const activeNavItem = navItems.find((item) => item.key === activeView.key) ?? navItems[0];
  const activeNavLabel = t(`nav.${activeNavItem.key}`, activeNavItem.label);
  const showWorkspaceTelemetry = activeView.key === "cockpit";
  const pageTitle = t(activeView.titleKey, t("app.title"));

  useEffect(() => {
    const handleHashChange = () => {
      const hashNav = navKeyFromHash(window.location.hash);
      defaultNavResolvedRef.current = hashNav !== null;
      userSelectedNavRef.current = hashNav !== null;
      setActiveNav(hashNav ?? defaultNavKey);
    };
    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  useEffect(() => {
    if (!data?.setupLoadedFromApi || defaultNavResolvedRef.current || userSelectedNavRef.current) return;
    setActiveNav(resolveDefaultNavKey(data));
    defaultNavResolvedRef.current = true;
  }, [data]);

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
    () => evaluateCoreJourney(data, accessProfileController.profile, coreJourney.config),
    [accessProfileController.profile, coreJourney.config, data]
  );
  const aiAdminApprovalJourneyEvaluation = useMemo(
    () =>
      evaluateAiAdminApprovalJourney({
        accessProfile: aiAdminApprovalJourneyAccessProfile ?? accessProfileController.profile,
        application: aiAdminApplication,
        approvalRequest: aiAdminApprovalJourneyApprovalRequest ?? aiAdminApprovalRequest,
        auditEvent: aiAdminApprovalAuditEvent,
        config: aiAdminApprovalJourneyConfig,
        data,
        result: aiAdminApprovalJourneyResult
      }),
    [
      accessProfileController.profile,
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
  const productionJourney = useMemo(
    () =>
      data
        ? deriveProductionJourney({
            accessOutcome: askAccess.result?.outcome ?? null,
            activeNav,
            data,
            hasPermissionChangeContext: Boolean(handoffContexts.permissionNotice || handoffContexts.permissionChange),
            permissionBlocked: aiAdminProductionReadiness?.status === "blocked" || aiAdminProductionConsoleSummary.status === "blocked",
            permissionReady: aiAdminProductionReadiness?.status === "ready" || aiAdminProductionConsoleSummary.status === "ready"
          })
        : null,
    [
      activeNav,
      aiAdminProductionConsoleSummary.status,
      aiAdminProductionReadiness?.status,
      askAccess.result?.outcome,
      data,
      handoffContexts.permissionChange,
      handoffContexts.permissionNotice
    ]
  );
  const productionJourneyCheckpoint = (
    productionJourney ? <ProductionJourneyCheckpoint journey={productionJourney} t={t} /> : null
  );
  const goLiveAcceptanceForm = aiAdminServerDraft?.input ?? aiAdminForm;
  const sessionActorLabel = consoleAuth.session?.actor
    ? t(`auditActor.${consoleAuth.session.actor}`, consoleAuth.session.actor)
    : t("auth.unknownActor");
  const sessionRole = sessionRoleLabel(consoleAuth.session, t);
  const sessionScope = sessionScopeLabel(consoleAuth.session, t);

  if (consoleAuth.sessionLoading) {
    return (
      <section className="workspace-loading login-loading" role="status" aria-live="polite">
        <div className="workspace-loading-copy">
          <strong>{t("status.loadingConsole")}</strong>
          <span>{t("auth.sessionLoading")}</span>
        </div>
        <div className="workspace-loading-skeleton" aria-hidden="true">
          <span />
          <span />
          <span />
        </div>
      </section>
    );
  }

  if (!consoleAccessReady) {
    return (
      <ConsoleLoginView
        adminKey={consoleAuth.loginKey}
        language={language}
        loading={consoleAuth.loginSubmitting}
        message={renderedConsoleLoginMessage}
        onAdminKeyChange={consoleAuth.setLoginKey}
        onLanguageChange={setLanguage}
        onSubmit={handleConsoleLogin}
        t={t}
      />
    );
  }

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
        connectionDiagnosticsChecking={connectionDiagnostics.checking}
        connectionStatus={connectionDiagnosticsStatus}
        draft={aiAdminServerDraft}
        form={aiAdminForm}
        liveDataAvailable={Boolean(data?.loadedFromApi)}
        onExportProductionEvidence={() => void exportAiAdminProductionEvidence(goLiveAcceptanceForm)}
        onOpenPermissionChange={() => setActiveNav("ai-admin")}
        onRunConnectionDiagnostics={() => void connectionDiagnostics.run()}
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
  function createPolicyAction(variant: "compact" | "command" = "compact", tone: "primary" | "secondary" = "secondary") {
    return (
      <ActionModalButton
        closeLabel={t("action.dismiss")}
        icon={<Route size={16} />}
        id="policy-create-panel"
        openLabel={t("action.open")}
        openToken={resourceActionModal === "create_policy" ? resourceActionOpenToken : undefined}
        tone={tone}
        title={t("panel.createPolicy")}
        variant={variant}
      >
        <PolicyCreateForm
          agents={agents}
          context={resourceActionContext}
          form={management.policyForm}
          message={management.policyMessage}
          onChange={management.setPolicyForm}
          onSubmit={management.submitRoutePolicy}
          t={t}
        />
      </ActionModalButton>
    );
  }
  const routeGovernancePanel = (className = "span-8", action: ReactNode = <IconMore title={t("action.more")} />) => (
    <Panel className={className} icon={<Workflow size={18} />} title={t("panel.routeGovernance")} action={action}>
      <PolicyTable
        agents={agents}
        canDisable={Boolean(data?.routePoliciesLoadedFromApi)}
        onDisable={management.handleDisablePolicy}
        pendingActionId={management.cleanupActionId}
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
  function resourceLifecyclePanel() {
    return (
      <Panel className="span-12" icon={<Network size={18} />} title={t("panel.resourceLifecycle")}>
        <ResourceLifecycleView
          formatTenantName={(tenantId) => permissionTenantPathLabel(tenantId, tenants, t).primary}
          formatWorkspaceName={(workspaceId) => permissionWorkspaceDisplayName(workspaceId, agents, t)}
          onResourceAction={handleResourceLifecycleAction}
          primaryActions={resourceLifecyclePrimaryActions}
          secondaryActions={resourceLifecycleSecondaryActions}
          summary={resourceLifecycleSummary}
          t={t}
        />
      </Panel>
    );
  }
  const agentRegistryPanel = (className = "span-8", action?: ReactNode) => (
    <Panel className={className} icon={<Boxes size={18} />} title={t("panel.agentRegistry")} action={action}>
      <AgentTable
        agents={agents}
        channelLabels={channelLabels}
        onQueryAccess={openAskAccess}
        onStatusChange={management.handleAgentStatusChange}
        pendingActionId={management.cleanupActionId}
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
        actionId={capabilityGovernance.actionId}
        agents={agents}
        capabilities={capabilities}
        form={capabilityGovernance.form}
        instanceAssignments={instanceAssignments}
        message={capabilityGovernance.message}
        mcpTargets={mcpTargets}
        onApprove={capabilityGovernance.handleApproveCapability}
        onChange={capabilityGovernance.setForm}
        onCreateGrantChain={capabilityGovernance.submitCapabilityGrantChain}
        onQueryAccess={openAskAccess}
        onRefreshTarget={capabilityGovernance.handleRefreshTargetCapabilities}
        t={t}
        tenants={tenants}
        tenantEntitlements={tenantEntitlements}
        workspaceAssignments={workspaceAssignments}
      />
    </Panel>
  );
  const askAccessPanel = (
    <AskAccessPanel agents={agents} capabilities={capabilities} controller={askAccess} liveDataAvailable={Boolean(data?.loadedFromApi)} t={t} tenants={tenants} title={t("panel.askAccess")} />
  );
  const accessProfilePanel = (
    <Panel className="span-12" icon={<LockKeyhole size={18} />} title={t("panel.accessProfile")} action={<IconOpen title={t("action.open")} />}>
      <TenantAccessProfileView
        agents={agents}
        capabilities={capabilities}
        filters={accessProfileController.filters}
        handoffContext={accessProfileController.handoffContext}
        loading={accessProfileController.loading}
        message={accessProfileController.message}
        onChange={accessProfileController.updateFilters}
        onQueryAccess={openAskAccess}
        onRefresh={() => void accessProfileController.refresh()}
        onTenantChange={accessProfileController.changeTenant}
        profile={accessProfileController.profile}
        scope={scope}
        t={t}
      />
    </Panel>
  );
  const tenantOrganizationPanel = (
    <TenantOrganizationView
      accessSubjects={aiAdminAccessSubjects}
      agents={agents}
      capabilities={capabilities}
      instanceAssignments={instanceAssignments}
      onOpenAccessProfile={openTenantAccessProfile}
      onSelectedTenantIdChange={setTenantOrganizationSelectedTenantId}
      onStartPermissionChange={openTenantPermissionChange}
      selectedTenantId={tenantOrganizationEffectiveTenantId}
      t={t}
      tenantEntitlements={tenantEntitlements}
      tenantPermissionCenter={tenantPermissionCenter}
      tenantPermissionCenterError={tenantPermissionCenterError}
      tenantPermissionCenterLoading={tenantPermissionCenterLoading}
      tenants={tenants}
      workspaceAssignments={workspaceAssignments}
    />
  );
  const adminAccessPanel = (
    <AdminAccessManagementView controller={adminAccess} t={t} />
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
        onDismissPermissionHandoff={() => setHandoffContexts((current) => ({ ...current, permissionNotice: null }))}
        onWithdrawApprovalRequest={(comment) => void withdrawAiAdminApprovalRequest(comment)}
        permissionHandoffContext={handoffContexts.permissionNotice}
        reviewerQueueLoading={aiAdminReviewerQueueLoading}
        reviewerQueueMessage={aiAdminReviewerQueueMessage}
        selectedApprovalRequestId={aiAdminSelectedApprovalRequestId}
        templates={aiAdminTemplates}
        tenants={tenants}
        t={t}
      />
  );
  function createAgentAction(variant: "compact" | "command" = "compact", tone: "primary" | "secondary" = "secondary") {
    return (
      <ActionModalButton
        closeLabel={t("action.dismiss")}
        icon={<Boxes size={16} />}
        openLabel={t("action.open")}
        tone={tone}
        title={t("panel.createAgent")}
        variant={variant}
      >
        <AgentCreateForm
          form={management.agentForm}
          message={management.agentMessage}
          onChange={management.setAgentForm}
          onSubmit={management.submitAgent}
          t={t}
        />
      </ActionModalButton>
    );
  }
  function createKeyAction(variant: "compact" | "command" = "compact", tone: "primary" | "secondary" = "secondary") {
    return (
      <ActionModalButton
        closeLabel={t("action.dismiss")}
        icon={<KeyRound size={16} />}
        openLabel={t("action.open")}
        openToken={resourceActionModal === "create_key" ? resourceActionOpenToken : undefined}
        tone={tone}
        title={t("panel.createKey")}
        variant={variant}
      >
        <KeyCreateForm
          agents={localCallers}
          context={resourceActionContext}
          createdKey={management.createdKey}
          form={management.keyForm}
          message={management.keyMessage}
          onChange={management.setKeyForm}
          onSubmit={management.submitKey}
          t={t}
        />
      </ActionModalButton>
    );
  }
  function rotateCredentialAction(variant: "compact" | "command" = "compact", tone: "primary" | "secondary" = "secondary") {
    return (
      <ActionModalButton
        closeLabel={t("action.dismiss")}
        icon={<KeyRound size={16} />}
        openLabel={t("action.open")}
        openToken={resourceActionModal === "rotate_credential" ? resourceActionOpenToken : undefined}
        tone={tone}
        title={t("panel.rotateCredential")}
        variant={variant}
      >
        <CredentialRotateForm
          agents={agents}
          context={resourceActionContext}
          form={management.rotateForm}
          message={management.rotateMessage}
          onChange={management.setRotateForm}
          onSubmit={management.submitCredentialRotation}
          t={t}
        />
      </ActionModalButton>
    );
  }
  const resourceLifecyclePrimaryActions = (
    <div className="resource-lifecycle-command-actions">
      {createAgentAction("command", "primary")}
      {agents.length > 0 ? (
        <>
          {createKeyAction("command", "secondary")}
          {rotateCredentialAction("command", "secondary")}
          {createPolicyAction("command", "secondary")}
        </>
      ) : null}
    </div>
  );
  const resourceLifecycleSecondaryActions = (
    <div className="resource-lifecycle-secondary-actions">
      <a className="secondary-button" href="#capabilities">
        <DatabaseZap size={14} />
        {t("resource.action.reviewCapabilities")}
      </a>
      <a className="secondary-button" href="#ai-admin">
        <ShieldCheck size={14} />
        {t("resource.action.startPermissionChange")}
      </a>
      <a className="secondary-button" href="#traces">
        <FileSearch size={14} />
        {t("resource.action.reviewRuntime")}
      </a>
    </div>
  );
  const coreJourneyPanel = (
    <Panel className="span-12" icon={<Workflow size={18} />} title={t("panel.coreJourney")}>
      <CoreJourneyWorkbench
        config={coreJourney.config}
        evaluation={coreJourneyEvaluation}
        form={coreJourney.form}
        message={coreJourney.message}
        onChange={coreJourney.setForm}
        onOpen={setActiveNav}
        onRefreshPreflight={() => void coreJourney.refreshPreflight()}
        onReset={() => void coreJourney.resetSession()}
        onRun={() => void coreJourney.run()}
        preflight={coreJourney.preflight}
        preflightChecking={coreJourney.preflightChecking}
        preflightMessage={coreJourney.preflightMessage}
        result={coreJourney.result}
        running={coreJourney.running}
        t={t}
      />
    </Panel>
  );
  const viewContent = (() => {
    switch (activeView.key) {
      case "ask":
        return <AskView askAccessPanel={askAccessPanel} journeyCheckpoint={productionJourneyCheckpoint} />;
      case "ai-admin":
        return <AiAdminView aiAdminPanel={aiAdminPanel} journeyCheckpoint={productionJourneyCheckpoint} />;
      case "getting-started":
        return (
          <GettingStartedConsoleView
            gettingStartedPanel={(
              <GettingStartedView
                setupDataAvailable={Boolean(data?.setupLoadedFromApi)}
                steps={setupSteps}
                t={t}
              />
            )}
            journeyCheckpoint={productionJourneyCheckpoint}
          />
        );
      case "registry":
        return (
          <RegistryView
            agentRegistryPanel={agentRegistryPanel("span-8")}
            contractMatrixPanel={contractMatrixPanel("span-4")}
            journeyCheckpoint={productionJourneyCheckpoint}
            resourceLifecyclePanel={resourceLifecyclePanel()}
          />
        );
      case "routes":
        return (
          <RoutesView
            routeGovernancePanel={routeGovernancePanel("span-12")}
            tracePanel={tracePanel("span-12")}
          />
        );
      case "policies":
        return (
          <PoliciesView
            capabilityGovernancePanel={capabilityGovernancePanel("span-12")}
            managementAuditPanel={managementAuditPanel("span-12")}
            policies={policies}
            routeGovernancePanel={routeGovernancePanel("span-12")}
            t={t}
          />
        );
      case "capabilities":
        return <CapabilitiesView capabilityGovernancePanel={capabilityGovernancePanel()} />;
      case "tenants":
        return <TenantsView tenantOrganizationPanel={tenantOrganizationPanel} />;
      case "admin-access":
        return <AdminAccessView adminAccessPanel={adminAccessPanel} />;
      case "access":
        return <AccessView accessProfilePanel={accessProfilePanel} journeyCheckpoint={productionJourneyCheckpoint} />;
      case "traces":
        return (
          <TracesView
            managementAuditPanel={managementAuditPanel("span-12")}
            tracePanel={tracePanel("span-12")}
          />
        );
      case "evidence":
        return (
          <EvidenceView
            evidenceRunsPanel={evidenceRunsPanel("span-5")}
            goLiveAcceptancePanel={goLiveAcceptancePanel}
            journeyCheckpoint={productionJourneyCheckpoint}
            managementAuditPanel={managementAuditPanel("span-7")}
            runtimeSignalsPanel={runtimeSignalsPanel("span-12")}
          />
        );
      case "cockpit":
      default:
        return (
          <CockpitView
            agentRegistryPanel={agentRegistryPanel("span-8")}
            coreJourneyPanel={coreJourneyPanel}
            evidenceRunsPanel={evidenceRunsPanel("span-4")}
            runtimeSignalsPanel={runtimeSignalsPanel("span-5")}
            tracePanel={tracePanel("span-7")}
          />
        );
    }
  })();
  const viewCanRenderWithoutConsoleData = activeView.key === "admin-access";

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
                        onClick={() => selectActiveNav(item.key)}
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
            <div
              className="session-chip"
              title={tx(t, "auth.sessionScopeTitle", {
                actor: sessionActorLabel,
                role: sessionRole,
                scope: sessionScope
              })}
            >
              <LockKeyhole size={14} />
              <span className="session-chip-main">{sessionActorLabel}</span>
              <span className="session-chip-scope">{sessionRole} · {sessionScope}</span>
              {consoleAuth.session?.requiresLogin ? (
                <button onClick={() => void handleConsoleLogout()} type="button">
                  <LogOut size={14} />
                  {t("action.signOut")}
                </button>
              ) : null}
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
                  <span>{t("control.adminKeyOverride")}</span>
                  <input
                    onChange={(event) => setAdminKey(event.target.value)}
                    placeholder={t("control.adminKeyOverridePlaceholder")}
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
                <button
                  className="connection-diagnostics-action primary-button"
                  disabled={connectionDiagnostics.checking}
                  onClick={() => void connectionDiagnostics.run()}
                  type="button"
                >
                  {connectionDiagnostics.checking ? t("connectionDiagnostics.checking") : t("connectionDiagnostics.action")}
                </button>
                {connectionDiagnosticsStatus ? (
                  <div className={`connection-diagnostics-panel tone-${connectionDiagnosticsStatus}`} role="status" aria-live="polite">
                    <div className="connection-diagnostics-summary">
                      <strong>{t("connectionDiagnostics.title")}</strong>
                      <span>{connectionDiagnosticSummaryLabel(connectionDiagnosticsStatus, t)}</span>
                    </div>
                    <ul className="connection-diagnostics-list">
                      {connectionDiagnostics.rows.map((row) => (
                        <li className={`connection-diagnostics-row tone-${row.status}`} key={row.key}>
                          <span className="health-dot" />
                          <div>
                            <strong>{t(row.titleKey)}</strong>
                            <span>{connectionDiagnosticDetail(row, t)}</span>
                          </div>
                        </li>
                      ))}
                    </ul>
                    {connectionDiagnostics.checkedAt ? (
                      <span className="connection-diagnostics-time">
                        {tx(t, "connectionDiagnostics.checkedAt", {
                          time: connectionDiagnostics.checkedAt.toLocaleTimeString("zh-CN", { hour12: false })
                        })}
                      </span>
                    ) : null}
                  </div>
                ) : null}
              </div>
            </details>
            <button aria-label={t("action.refresh")} className="icon-button" onClick={refresh} title={t("action.refresh")} type="button">
              <RefreshCw size={17} />
            </button>
          </div>
        </header>

        {showWorkspaceTelemetry ? (
          <section className="system-check-context" aria-label={pageTitle}>
            <div className="system-check-context-main">
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
                <div className="scope-values">
                  <span title={scope.tenantId}>{scope.tenantId}</span>
                  <span title={scope.workspaceId}>{scope.workspaceId}</span>
                </div>
              </div>
            </div>
            <div className="system-check-signals" aria-label="Gateway metrics">
              <span className="system-check-signal">
                <span>{t("metric.managedAgents")}</span>
                <strong>{agents.length}</strong>
                <small>{activeAgents} {t("detail.active")}</small>
              </span>
              <span className="system-check-signal">
                <span>{t("metric.activePolicies")}</span>
                <strong>{activePolicies}</strong>
                <small>{data?.routePoliciesLoadedFromApi ? t("detail.liveRoutePolicies") : t("detail.sampleFallback")}</small>
              </span>
              <span className={`system-check-signal tone-${deniedTraces > 0 ? "warning" : "success"}`}>
                <span>{t("metric.deniedTraces")}</span>
                <strong>{deniedTraces}</strong>
                <small>{allowedTraces} {t("detail.allowed")}</small>
              </span>
              <span className={`system-check-signal tone-${runtimeEvidence.tone}`}>
                <span>{t("metric.runtimeEvidence")}</span>
                <strong>{runtimeEvidence.value}</strong>
                <small>{runtimeEvidence.value === "0" ? t("detail.noTraces") : `${allowedTraces} ${t("detail.allowed")} / ${deniedTraces} ${t("detail.denied")}`}</small>
              </span>
            </div>
            {loadError ? <div className="strip-error">{loadError}</div> : null}
          </section>
        ) : null}

        {!data && !viewCanRenderWithoutConsoleData ? (
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

export default ConsoleController;
