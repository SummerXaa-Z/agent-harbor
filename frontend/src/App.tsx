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
  createAgent,
  createAgentKey,
  createInstanceAssignment,
  createRoutePolicy,
  createTenantEntitlement,
  createWorkspaceAssignment,
  disableAgent,
  disableRoutePolicy,
  loadConsoleData,
  loadTenantAccessProfile,
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
import { parseRetryFields } from "./retryForm";
import type {
  AccessProfileFilters,
  AccessProfileSummary,
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
  targetId: "",
  traceLimit: "20",
  workspaceId: ""
};
const mcpRouteKeyPresets = ["initialize", "tools/list", "tools/call"];

function navIconFor(key: NavKey) {
  switch (key) {
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

function translatedValue(t: Translator, value?: string) {
  return value ? t(`value.${value}`, value) : "";
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
  const [language, setLanguage] = useState<Language>(initialLanguage);
  const t = useMemo(() => createTranslator(language), [language]);

  useEffect(() => {
    void refresh();
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

  async function refresh() {
    try {
      setLoadError("");
      const next = await loadConsoleData(adminKey, traceFilters, normalizedScope(scope));
      setData(next);
      setLastRefresh(new Date());
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "console data unavailable");
    }
    if (activeNav === "access") {
      await refreshAccessProfile();
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
        filters={accessFilters}
        loading={accessLoading}
        message={accessMessage}
        onChange={setAccessFilters}
        onRefresh={() => void refreshAccessProfile()}
        onTenantChange={(tenantId) => {
          setScope((current) => ({ ...current, tenantId }));
          setAccessProfile(null);
        }}
        profile={accessProfile}
        scope={scope}
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
  const viewContent = (() => {
    switch (activeView.key) {
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
            <strong>{names[trace.callerAgentId ?? ""] ?? trace.callerAgentId ?? t("text.traceAnonymous")} → {names[trace.targetAgentId] ?? trace.targetAgentId}</strong>
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
  filters,
  loading,
  message,
  onChange,
  onRefresh,
  onTenantChange,
  profile,
  scope,
  t
}: {
  agents: Agent[];
  capabilities: Capability[];
  filters: AccessProfileFilters;
  loading: boolean;
  message: string;
  onChange: (filters: AccessProfileFilters) => void;
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
                    <strong>{names[trace.callerAgentId ?? ""] ?? trace.callerAgentId ?? t("text.traceAnonymous")} → {names[trace.targetAgentId] ?? trace.targetAgentId}</strong>
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

export default App;
