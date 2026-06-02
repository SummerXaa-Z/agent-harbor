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
import { parseRetryFields } from "./retryForm";
import type {
  AccessProfileFilters,
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

type Tone = "success" | "warning" | "danger" | "info" | "neutral";

const navItems = [
  { key: "cockpit", label: "Cockpit", icon: Gauge },
  { key: "registry", label: "Registry", icon: Boxes },
  { key: "routes", label: "Routes", icon: Route },
  { key: "policies", label: "Policies", icon: ShieldCheck },
  { key: "capabilities", label: "Capabilities", icon: DatabaseZap },
  { key: "access", label: "Access", icon: LockKeyhole },
  { key: "traces", label: "Traces", icon: FileSearch },
  { key: "evidence", label: "Evidence", icon: ClipboardCheck }
];

const workspaceTabs = ["Prod", "Staging", "Sandbox"];
const defaultManagementScope: ManagementScope = {
  tenantId: "default",
  workspaceId: "workspace-sandbox"
};
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
const defaultTraceFilters: TraceFilters = { callerAgentId: "", decision: "", runId: "", targetAgentId: "" };
const defaultAccessProfileFilters: AccessProfileFilters = {
  callerInstanceId: "",
  capabilityId: "",
  targetId: "",
  traceLimit: "20",
  workspaceId: ""
};
const mcpRouteKeyPresets = ["initialize", "tools/list", "tools/call"];

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

  useEffect(() => {
    void refresh();
  }, []);

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
      setAccessMessage(next.loadedFromApi ? "Access profile refreshed." : "Using fallback access profile.");
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
          setAgentMessage("Credential header, key, and value are required together.");
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
      setAgentMessage("Agent created. Registry refreshed.");
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
      setAgentMessage(`${agent.name} set to ${status}. Registry refreshed.`);
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
      setPolicyMessage("Route policy disabled. Governance refreshed.");
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
        setKeyMessage("TTL must be an integer between 1 and 3600 seconds.");
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
      setKeyMessage("Plaintext key is shown once. Copy it before leaving this view.");
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
        setRotateMessage("Select an Agent to rotate.");
        return;
      }
      if (!credentialName || !rotateForm.credentialValue.trim()) {
        setRotateMessage("Credential key and new secret are required.");
        return;
      }
      await rotateAgentCredentials(
        rotateForm.agentId,
        { credentials: { [credentialName]: rotateForm.credentialValue } },
        adminKey
      );
      setRotateForm({ ...defaultRotateForm, agentId: rotateForm.agentId, credentialName });
      setRotateMessage("Credential rotated. Secret value cleared.");
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
        setPolicyMessage("Priority must be a non-negative integer.");
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
      setPolicyMessage("Route policy created. Governance refreshed.");
      setPolicyForm({ ...defaultPolicyForm, callerAgentId: policyForm.callerAgentId });
      await refresh();
    } catch (error) {
      setPolicyMessage(error instanceof Error ? error.message : "Unable to create route policy");
    }
  }

  async function handleRefreshTargetCapabilities() {
    const targetId = capabilityForm.targetId.trim();
    if (!targetId) {
      setCapabilityMessage("Select an MCP target.");
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
      setCapabilityMessage(`Refreshed ${refreshed.length} capabilities.`);
    } catch (error) {
      if (shouldUseLocalCapabilityFallback(error, data)) {
        setCapabilityMessage("Using fallback capabilities.");
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
      setCapabilityMessage(`${capability.key} approved.`);
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
        setCapabilityMessage(`${capability.key} approved in fallback data.`);
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
      setCapabilityMessage("Select a capability.");
      return;
    }
    if (!tenantId || !workspaceId || !callerInstanceId) {
      setCapabilityMessage("Tenant, workspace, and caller are required.");
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
      setCapabilityMessage("Grant chain created.");
      await refresh();
    } catch (error) {
      if (shouldUseLocalCapabilityFallback(error, data) && data) {
        setData((current) =>
          current ? appendLocalCapabilityGrantChain(current, capability, capabilityForm, dataScopes) : current
        );
        setCapabilityMessage("Grant chain created in fallback data.");
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
  const evidencePassed = evidenceRuns.filter((run) => run.status === "passed").length;
  const dataSourceLabel = loadError ? "API error" : data?.loadedFromApi ? "Go runtime + samples" : "Fallback dataset";
  const activeNavLabel = navItems.find((item) => item.key === activeNav)?.label ?? "Cockpit";
  const isCapabilitiesView = activeNav === "capabilities";
  const isAccessView = activeNav === "access";
  const accessSummary = accessProfile?.summary;
  const invalidAccessRows = countInvalidAccessProfileRows(accessProfile);
  const pageTitle = isAccessView
    ? "Tenant Permission Console"
    : isCapabilitiesView
      ? "Capability Governance"
      : "AgentHarbor Control Plane";

  return (
    <div className="app-shell">
      <aside className="sidebar" aria-label="Primary navigation">
        <div className="brand">
          <div className="brand-mark">
            <Network size={18} />
          </div>
          <div>
            <strong>AgentHarbor</strong>
            <span>Control Plane</span>
          </div>
        </div>

        <nav className="nav-list">
          {navItems.map((item) => {
            const Icon = item.icon;
            return (
              <button
                className={activeNav === item.key ? "nav-item active" : "nav-item"}
                key={item.key}
                onClick={() => setActiveNav(item.key)}
                type="button"
              >
                <Icon size={17} />
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>

        <div className="sidebar-section">
          <div className="section-kicker">Environments</div>
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
            <strong>Control plane</strong>
          </div>
          <span>{loadError ? "API error" : data?.loadedFromApi ? "Live API" : "Mock fallback"}</span>
        </div>
      </aside>

      <main className="workspace">
        <header className="topbar">
          <div className="topbar-title">
            <div className="breadcrumb">Gateway / {activeWorkspace} / {activeNavLabel}</div>
            <h1>{pageTitle}</h1>
          </div>
          <div className="topbar-actions">
            <label className="admin-key-box">
              <LockKeyhole size={16} />
              <input
                onChange={(event) => setAdminKey(event.target.value)}
                placeholder="X-Admin-Key"
                type="password"
                value={adminKey}
              />
            </label>
            <label className="search-box">
              <Search size={16} />
              <input placeholder="Search agents, routes, traces" />
            </label>
            <button className="icon-button" title="Filter" type="button">
              <Filter size={17} />
            </button>
            <button className="icon-button" onClick={refresh} title="Refresh" type="button">
              <RefreshCw size={17} />
            </button>
          </div>
        </header>

        <section className="status-strip" aria-label="Runtime status">
          <div>
            <span>API</span>
            <strong>{data?.apiBase ?? "http://127.0.0.1:9090"}</strong>
          </div>
          <div>
            <span>Data source</span>
            <strong>{dataSourceLabel}</strong>
          </div>
          <div>
            <span>Last refresh</span>
            <strong>{lastRefresh.toLocaleTimeString("zh-CN", { hour12: false })}</strong>
          </div>
          <div className="scope-control">
            <span>Scope</span>
            <div className="scope-inputs">
              <input
                aria-label="Tenant ID"
                onBlur={() => void refresh()}
                onChange={(event) => setScope((current) => ({ ...current, tenantId: event.target.value }))}
                placeholder="tenantId"
                value={scope.tenantId}
              />
              <input
                aria-label="Workspace ID"
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
            label={isAccessView ? "Scope Tenants" : "Managed Agents"}
            value={String(isAccessView ? accessSummary?.tenantCount ?? 0 : agents.length)}
            detail={isAccessView ? accessProfile?.tenant.name ?? normalizedScope(scope).tenantId : `${activeAgents} active`}
            tone="info"
          />
          <MetricCard
            icon={<KeyRound size={18} />}
            label={isAccessView ? "Grant Chains" : "Active Policies"}
            value={String(isAccessView ? accessSummary?.grantCount ?? 0 : activePolicies)}
            detail={isAccessView ? `${accessSummary?.targetCount ?? 0} targets` : data?.routePoliciesLoadedFromApi ? "live route policies" : "sample fallback"}
            tone="success"
          />
          <MetricCard
            icon={<TriangleAlert size={18} />}
            label={isAccessView ? "Invalid Rows" : isCapabilitiesView ? "Pending Caps" : "Denied Traces"}
            value={String(isAccessView ? invalidAccessRows : isCapabilitiesView ? pendingCapabilities : deniedTraces)}
            detail={isAccessView ? `${accessSummary?.workspaceAssignmentCount ?? 0} workspaces` : isCapabilitiesView ? `${capabilities.length} discovered` : `${allowedTraces} allowed`}
            tone={(isAccessView ? invalidAccessRows : isCapabilitiesView ? pendingCapabilities : deniedTraces) > 0 ? "warning" : "success"}
          />
          <MetricCard
            icon={<ClipboardCheck size={18} />}
            label={isAccessView ? "Trace Evidence" : "Evidence Health"}
            value={isAccessView ? String((accessSummary?.recentAllowedTraceCount ?? 0) + (accessSummary?.recentDeniedTraceCount ?? 0)) : `${evidencePassed}/${Math.max(evidenceRuns.length, 1)}`}
            detail={isAccessView ? `${accessSummary?.recentDeniedTraceCount ?? 0} denied` : "checks passed"}
            tone="neutral"
          />
        </section>

        {isAccessView ? (
          <section className="content-grid">
            <Panel className="span-12" icon={<LockKeyhole size={18} />} title="Tenant Access Profile" action={<IconOpen />}>
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
              />
            </Panel>
          </section>
        ) : isCapabilitiesView ? (
          <section className="content-grid">
            <Panel className="span-12" icon={<DatabaseZap size={18} />} title="MCP Capabilities" action={<IconMore />}>
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
                tenantEntitlements={tenantEntitlements}
                workspaceAssignments={workspaceAssignments}
              />
            </Panel>

            <Panel className="span-7" icon={<FileSearch size={18} />} title="Audit Traces" action={<IconOpen />}>
              <TraceFilterBar agents={agents} filters={traceFilters} onChange={setTraceFilters} onRefresh={refresh} />
              <TraceTable traces={traces} agents={agents} />
            </Panel>

            <Panel className="span-5" icon={<ClipboardCheck size={18} />} title="Management Audit" action={<IconOpen />}>
              <ManagementAuditTable events={auditEvents} />
            </Panel>
          </section>
        ) : (
        <section className="content-grid">
          <Panel className="span-4" icon={<Boxes size={18} />} title="Create Agent">
            <AgentCreateForm form={agentForm} message={agentMessage} onChange={setAgentForm} onSubmit={submitAgent} />
          </Panel>

          <Panel className="span-4" icon={<KeyRound size={18} />} title="Create Key">
            <KeyCreateForm
              agents={localCallers}
              createdKey={createdKey}
              form={keyForm}
              message={keyMessage}
              onChange={setKeyForm}
              onSubmit={submitKey}
            />
          </Panel>

          <Panel className="span-4" icon={<Route size={18} />} title="Create Policy">
            <PolicyCreateForm agents={agents} form={policyForm} message={policyMessage} onChange={setPolicyForm} onSubmit={submitRoutePolicy} />
          </Panel>

          <Panel className="span-4" icon={<KeyRound size={18} />} title="Rotate Credential">
            <CredentialRotateForm
              agents={agents}
              form={rotateForm}
              message={rotateMessage}
              onChange={setRotateForm}
              onSubmit={submitCredentialRotation}
            />
          </Panel>

          <Panel className="span-8" icon={<Workflow size={18} />} title="Route Governance" action={<IconMore />}>
            <PolicyTable
              agents={agents}
              canDisable={Boolean(data?.routePoliciesLoadedFromApi)}
              onDisable={handleDisablePolicy}
              pendingActionId={cleanupActionId}
              policies={policies}
            />
          </Panel>

          <Panel className="span-4" icon={<Sparkles size={18} />} title="Evidence Runs" action={<IconOpen />}>
            <EvidenceTimeline runs={evidenceRuns} />
          </Panel>

          <Panel className="span-8" icon={<Boxes size={18} />} title="Agent Registry" action={<IconMore />}>
            <AgentTable
              agents={agents}
              channelLabels={channelLabels}
              onStatusChange={handleAgentStatusChange}
              pendingActionId={cleanupActionId}
            />
          </Panel>

          <Panel className="span-4" icon={<Layers3 size={18} />} title="Contract Matrix" action={<IconOpen />}>
            <ContractMatrix channels={channels} providers={data?.providers ?? []} />
          </Panel>

          <Panel className="span-5" icon={<DatabaseZap size={18} />} title="Runtime Signals" action={<IconMore />}>
            <SignalBoard metrics={metrics} />
          </Panel>

          <Panel className="span-7" icon={<FileSearch size={18} />} title="Audit Traces" action={<IconOpen />}>
            <TraceFilterBar agents={agents} filters={traceFilters} onChange={setTraceFilters} onRefresh={refresh} />
            <TraceTable traces={traces} agents={agents} />
          </Panel>

          <Panel className="span-12" icon={<ClipboardCheck size={18} />} title="Management Audit" action={<IconOpen />}>
            <ManagementAuditTable events={auditEvents} />
          </Panel>
        </section>
        )}
      </main>
    </div>
  );
}

function AgentCreateForm({
  form,
  message,
  onChange,
  onSubmit
}: {
  form: typeof defaultAgentForm;
  message: string;
  onChange: (form: typeof defaultAgentForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
      <label>Name<input required value={form.name} onChange={(event) => onChange({ ...form, name: event.target.value })} /></label>
      <div className="form-row">
        <label>Channel<input value={form.channelType} onChange={(event) => onChange({ ...form, channelType: event.target.value })} /></label>
        <label>Status<select value={form.status} onChange={(event) => onChange({ ...form, status: event.target.value as AgentStatus })}><option value="draft">draft</option><option value="active">active</option><option value="disabled">disabled</option></select></label>
      </div>
      <label>Endpoint<input placeholder="https://api.example.com/a2a" value={form.endpoint} onChange={(event) => onChange({ ...form, endpoint: event.target.value })} /></label>
      <div className="form-row">
        <label>Credential header<input placeholder="Authorization" value={form.credentialHeader} onChange={(event) => onChange({ ...form, credentialHeader: event.target.value })} /></label>
        <label>Credential key<input placeholder="apiToken" value={form.credentialName} onChange={(event) => onChange({ ...form, credentialName: event.target.value })} /></label>
      </div>
      <label>Secret value<input placeholder="Bearer ..." type="password" value={form.credentialValue} onChange={(event) => onChange({ ...form, credentialValue: event.target.value })} /></label>
      <div className="form-row">
        <label>Retry attempts<input inputMode="numeric" max={4} min={1} type="number" value={form.retryMaxAttempts} onChange={(event) => onChange({ ...form, retryMaxAttempts: event.target.value })} /></label>
        <label>Backoff ms<input inputMode="numeric" max={1000} min={0} type="number" value={form.retryBackoffMs} onChange={(event) => onChange({ ...form, retryBackoffMs: event.target.value })} /></label>
      </div>
      <label>Description<textarea rows={2} value={form.description} onChange={(event) => onChange({ ...form, description: event.target.value })} /></label>
      <FormFooter message={message} submitLabel="Create agent" />
    </form>
  );
}

function KeyCreateForm({
  agents,
  createdKey,
  form,
  message,
  onChange,
  onSubmit
}: {
  agents: Agent[];
  createdKey: CreateAgentKeyResponse | null;
  form: typeof defaultKeyForm;
  message: string;
  onChange: (form: typeof defaultKeyForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
      <label>Active local caller<select required value={form.agentId} onChange={(event) => onChange({ ...form, agentId: event.target.value })}><option value="">Select caller</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <label>Name<input value={form.name} onChange={(event) => onChange({ ...form, name: event.target.value })} /></label>
      <label>TTL seconds<input inputMode="numeric" max={3600} min={1} type="number" value={form.expiresInSeconds} onChange={(event) => onChange({ ...form, expiresInSeconds: event.target.value })} /></label>
      {createdKey ? (
        <div className="one-time-key">
          <div><strong>One-time key</strong><span>Copy now. Expires {formatDate(createdKey.expiresAt)}.</span></div>
          <code>{createdKey.key}</code>
          <button className="secondary-button" type="button" onClick={() => void navigator.clipboard?.writeText(createdKey.key)}><Copy size={14} /> Copy</button>
        </div>
      ) : null}
      <FormFooter message={message} submitLabel="Create key" />
    </form>
  );
}

function CredentialRotateForm({
  agents,
  form,
  message,
  onChange,
  onSubmit
}: {
  agents: Agent[];
  form: typeof defaultRotateForm;
  message: string;
  onChange: (form: typeof defaultRotateForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
      <label>Agent<select required value={form.agentId} onChange={(event) => onChange({ ...form, agentId: event.target.value })}><option value="">Select Agent</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <label>Credential key<input placeholder="apiToken" value={form.credentialName} onChange={(event) => onChange({ ...form, credentialName: event.target.value })} /></label>
      <label>New secret<input placeholder="Bearer ..." type="password" value={form.credentialValue} onChange={(event) => onChange({ ...form, credentialValue: event.target.value })} /></label>
      <FormFooter message={message} submitLabel="Rotate credential" />
    </form>
  );
}

function PolicyCreateForm({
  agents,
  form,
  message,
  onChange,
  onSubmit
}: {
  agents: Agent[];
  form: typeof defaultPolicyForm;
  message: string;
  onChange: (form: typeof defaultPolicyForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
      <label>Name<input placeholder="Allow MCP tools/call" value={form.name} onChange={(event) => onChange({ ...form, name: event.target.value })} /></label>
      <label>Caller<select required value={form.callerAgentId} onChange={(event) => onChange({ ...form, callerAgentId: event.target.value })}><option value="">Select caller</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <label>Target<select required value={form.targetAgentId} onChange={(event) => onChange({ ...form, targetAgentId: event.target.value })}><option value="">Select target</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <div className="route-presets" aria-label="MCP route key presets">
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
          wildcard
        </button>
      </div>
      <div className="form-row">
        <label>Route type<input value={form.routeType} onChange={(event) => onChange({ ...form, routeType: event.target.value })} /></label>
        <label>Route key<input value={form.routeKey} onChange={(event) => onChange({ ...form, routeKey: event.target.value })} /></label>
      </div>
      <div className="form-row">
        <label>Effect<select value={form.effect} onChange={(event) => onChange({ ...form, effect: event.target.value })}><option value="allow">allow</option><option value="deny">deny</option></select></label>
        <label>Priority<input inputMode="numeric" min={0} type="number" value={form.priority} onChange={(event) => onChange({ ...form, priority: event.target.value })} /></label>
      </div>
      <div className="form-row">
        <label>Retry attempts<input inputMode="numeric" max={4} min={1} type="number" value={form.retryMaxAttempts} onChange={(event) => onChange({ ...form, retryMaxAttempts: event.target.value })} /></label>
        <label>Retry backoff ms<input inputMode="numeric" max={1000} min={0} type="number" value={form.retryBackoffMs} onChange={(event) => onChange({ ...form, retryBackoffMs: event.target.value })} /></label>
      </div>
      <FormFooter message={message} submitLabel="Create policy" />
    </form>
  );
}

function TraceFilterBar({
  agents,
  filters,
  onChange,
  onRefresh
}: {
  agents: Agent[];
  filters: TraceFilters;
  onChange: (filters: TraceFilters) => void;
  onRefresh: () => void;
}) {
  return (
    <div className="trace-filters">
      <input placeholder="runId" value={filters.runId ?? ""} onChange={(event) => onChange({ ...filters, runId: event.target.value })} />
      <select value={filters.decision ?? ""} onChange={(event) => onChange({ ...filters, decision: event.target.value as TraceDecision | "" })}>
        <option value="">Any decision</option>
        <option value="allowed">allowed</option>
        <option value="denied">denied</option>
      </select>
      <select value={filters.callerAgentId ?? ""} onChange={(event) => onChange({ ...filters, callerAgentId: event.target.value })}>
        <option value="">Any caller</option>
        {agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}
      </select>
      <select value={filters.targetAgentId ?? ""} onChange={(event) => onChange({ ...filters, targetAgentId: event.target.value })}>
        <option value="">Any target</option>
        {agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}
      </select>
      <button className="secondary-button" type="button" onClick={onRefresh}><RefreshCw size={14} /> Refresh</button>
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
  policies
}: {
  agents: Agent[];
  canDisable: boolean;
  onDisable: (policy: RoutePolicy) => void;
  pendingActionId: string;
  policies: RoutePolicy[];
}) {
  const names = agentNameMap(agents);

  return (
    <div className="table-wrap">
      <table className="policy-table">
        <thead>
          <tr>
            <th>Policy</th>
            <th>Caller</th>
            <th>Target</th>
            <th>Route</th>
            <th>Decision</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {policies.length === 0 ? (
            <tr>
              <td colSpan={6}>
                <EmptyRow title="No route policies" detail="Create a route policy to populate governed routes." />
              </td>
            </tr>
          ) : null}
          {policies.map((policy) => (
            <tr className={policy.status === "disabled" ? "row-disabled" : undefined} key={policy.id}>
              <td>
                <strong>{policy.name}</strong>
                <span>priority {policy.priority} · {policyRetryText(policy)} · matched {formatDate(policy.lastMatchedAt ?? policy.createdAt)}</span>
              </td>
              <td>{names[policy.callerAgentId] ?? policy.callerAgentId}</td>
              <td>{names[policy.targetAgentId] ?? policy.targetAgentId}</td>
              <td>
                <code>{policy.routeType}:{policy.routeKey || "wildcard"}</code>
              </td>
              <td><Badge tone={policy.effect === "allow" ? "success" : "danger"}>{policy.effect}</Badge></td>
              <td>
                {canDisable && policy.status === "enabled" ? (
                  <button
                    className="table-action"
                    disabled={pendingActionId === policy.id}
                    onClick={() => onDisable(policy)}
                    type="button"
                  >
                    <LockKeyhole size={13} />
                    {pendingActionId === policy.id ? "Disabling" : "Disable"}
                  </button>
                ) : (
                  <span className="muted-action">{policy.status === "disabled" ? "disabled" : "sample"}</span>
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
  pendingActionId
}: {
  agents: Agent[];
  channelLabels: Record<string, string>;
  onStatusChange: (agent: Agent, status: AgentStatus) => void;
  pendingActionId: string;
}) {
  return (
    <div className="table-wrap">
      <table className="agent-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Channel</th>
            <th>Endpoint</th>
            <th>Status</th>
            <th>Owner</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {agents.length === 0 ? (
            <tr>
              <td colSpan={6}>
                <EmptyRow title="No agents registered" detail="Register caller and target agents to start routing traffic." />
              </td>
            </tr>
          ) : null}
          {agents.map((agent) => (
            <tr className={agent.status === "disabled" ? "row-disabled" : undefined} key={agent.id}>
              <td>
                <strong>{agent.name}</strong>
                <span>{agent.description || agent.id}</span>
              </td>
              <td>{channelLabels[agent.channelType] ?? agent.channelType}</td>
              <td className="truncate">{configText(agent, "endpoint") || "local runtime"}</td>
              <td><Badge tone={agent.status === "active" ? "success" : agent.status === "draft" ? "warning" : "neutral"}>{agent.status}</Badge></td>
              <td>{agent.ownerId || "platform"}</td>
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
                      {pendingActionId === agent.id ? "Updating" : agent.status === "active" ? "Draft" : "Activate"}
                    </button>
                    <button
                      className="table-action"
                      disabled={pendingActionId === agent.id}
                      onClick={() => onStatusChange(agent, "disabled")}
                      type="button"
                    >
                      <LockKeyhole size={13} />
                      Disable
                    </button>
                  </div>
                ) : (
                  <span className="muted-action">disabled</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function EvidenceTimeline({ runs }: { runs: EvidenceRun[] }) {
  return (
    <div className="timeline">
      {runs.length === 0 ? <EmptyRow title="No evidence runs" detail="Policy checks will appear here after traffic is evaluated." /> : null}
      {runs.map((run) => (
        <article className="timeline-row" key={run.id}>
          <div className={`timeline-marker tone-${toneFromStatus(run.status)}`}>
            {run.status === "passed" ? <CheckCircle2 size={14} /> : <Activity size={14} />}
          </div>
          <div>
            <div className="timeline-title">
              <strong>{run.title}</strong>
              <Badge tone={toneFromStatus(run.status)}>{run.status}</Badge>
            </div>
            <p>{run.checks} checks · {formatDuration(evidenceDuration(run))} · {formatDate(run.completedAt ?? run.startedAt)}</p>
          </div>
        </article>
      ))}
    </div>
  );
}

function ContractMatrix({
  channels,
  providers
}: {
  channels: ChannelContract[];
  providers: Array<{ key: string; label: string; channelType: string; requiredCreds?: string[] }>;
}) {
  return (
    <div className="contract-list">
      {channels.map((channel) => (
        <div className="contract-row" key={channel.key}>
          <div>
            <strong>{channel.label}</strong>
            <span>{channel.key}</span>
          </div>
          <Badge tone={channel.endpointRequiredWhenActive ? "warning" : "neutral"}>
            {channel.endpointRequiredWhenActive ? "endpoint" : "local"}
          </Badge>
        </div>
      ))}
      <div className="provider-strip">
        {providers.map((provider) => (
          <span key={provider.key}>{provider.label} · {provider.channelType}</span>
        ))}
      </div>
    </div>
  );
}

function SignalBoard({ metrics }: { metrics: SystemMetric[] }) {
  return (
    <div className="signal-grid">
      {metrics.map((metric) => (
        <article className="signal" key={metric.id}>
          <span>{metric.label}</span>
          <strong>{metric.value}{metric.unit ?? ""}</strong>
          <div className="signal-track" aria-hidden="true">
            <i style={{ width: `${metricRatio(metric)}%` }} />
          </div>
          <small>{metric.trend} · {metric.status}</small>
        </article>
      ))}
    </div>
  );
}

function TraceTable({ traces, agents }: { traces: TraceEvent[]; agents: Agent[] }) {
  const names = useMemo(() => agentNameMap(agents), [agents]);

  return (
    <div className="trace-list">
      {traces.length === 0 ? <EmptyRow title="No audit traces" detail="Allowed and denied data-plane calls will be listed here." /> : null}
      {traces.map((trace) => (
        <article className="trace-row" key={trace.id}>
          <div className={`trace-decision tone-${trace.decision === "allowed" ? "success" : "danger"}`}>
            {trace.decision === "allowed" ? <CheckCircle2 size={15} /> : <LockKeyhole size={15} />}
          </div>
          <div>
            <strong>{names[trace.callerAgentId ?? ""] ?? trace.callerAgentId ?? "anonymous"} → {names[trace.targetAgentId] ?? trace.targetAgentId}</strong>
            <span>
              {trace.routeType}:{trace.routeKey || "default"}
              {trace.capabilityId ? ` · ${trace.capabilityId}` : ""}
              {" · "}
              {trace.reason || trace.decision}
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
  scope
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
}) {
  const names = useMemo(() => agentNameMap(agents), [agents]);
  const targetAgents = agents.filter((agent) => agent.channelType !== "local");
  const visibleCapabilities = filters.targetId
    ? capabilities.filter((capability) => capability.targetId === filters.targetId)
    : capabilities;
  const sourceLabel = profile ? (profile.loadedFromApi ? "Live access profile" : "Fallback access profile") : "Not loaded";

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onRefresh();
  }

  return (
    <div className="access-profile">
      <form className="access-toolbar" onSubmit={submit}>
        <label>
          Tenant
          <input value={scope.tenantId} onChange={(event) => onTenantChange(event.target.value)} />
        </label>
        <label>
          Workspace
          <input
            placeholder="all workspaces"
            value={filters.workspaceId ?? ""}
            onChange={(event) => onChange({ ...filters, workspaceId: event.target.value })}
          />
        </label>
        <label>
          Target
          <select
            value={filters.targetId ?? ""}
            onChange={(event) => onChange({ ...filters, capabilityId: "", targetId: event.target.value })}
          >
            <option value="">Any target</option>
            {targetAgents.map((agent) => (
              <option key={agent.id} value={agent.id}>
                {agent.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Capability
          <select
            value={filters.capabilityId ?? ""}
            onChange={(event) => onChange({ ...filters, capabilityId: event.target.value })}
          >
            <option value="">Any capability</option>
            {visibleCapabilities.map((capability) => (
              <option key={capability.id} value={capability.id}>
                {capability.key}
              </option>
            ))}
          </select>
        </label>
        <label>
          Caller
          <select
            value={filters.callerInstanceId ?? ""}
            onChange={(event) => onChange({ ...filters, callerInstanceId: event.target.value })}
          >
            <option value="">Any caller</option>
            {agents.map((agent) => (
              <option key={agent.id} value={agent.id}>
                {agent.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Traces
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
          {loading ? "Loading" : "Load profile"}
        </button>
      </form>

      <div className="access-source-line">
        <span>{sourceLabel}</span>
        {profile ? <span>Generated {formatDate(profile.generatedAt)}</span> : null}
        {message ? <strong>{message}</strong> : null}
      </div>

      {!profile ? (
        <EmptyRow title="No access profile loaded" detail="Load a tenant profile to inspect effective permissions." />
      ) : (
        <>
          <div className="access-summary-grid">
            <AccessSummaryCell label="Tenant Scope" value={String(profile.summary.tenantCount)} detail={profile.tenant.name} />
            <AccessSummaryCell label="Grants" value={String(profile.summary.grantCount)} detail={`${profile.summary.capabilityCount} capabilities`} />
            <AccessSummaryCell label="Assignments" value={`${profile.summary.workspaceAssignmentCount}/${profile.summary.instanceAssignmentCount}`} detail="workspace / caller" />
            <AccessSummaryCell label="Recent Decisions" value={`${profile.summary.recentAllowedTraceCount}/${profile.summary.recentDeniedTraceCount}`} detail="allowed / denied" />
          </div>

          <div className="access-tenant-list" aria-label="Tenant scope">
            {profile.scopeTenants.map((tenant) => (
              <div className="access-tenant-row" key={tenant.id}>
                <Badge tone={tenant.status === "active" ? "success" : "neutral"}>L{tenant.level}</Badge>
                <div>
                  <strong>{tenant.name}</strong>
                  <span>{tenant.id}{tenant.parentTenantId ? ` · parent ${tenant.parentTenantId}` : ""}</span>
                </div>
              </div>
            ))}
          </div>

          <div className="access-layout">
            <section className="access-grant-chain" aria-label="Effective grant chain">
              <header>
                <strong>Effective Grant Chain</strong>
                <span>{countInvalidAccessProfileRows(profile)} invalid rows</span>
              </header>
              {profile.grants.length === 0 ? (
                <EmptyRow title="No grant chains" detail="No tenant entitlement matched the current profile filters." />
              ) : null}
              {profile.grants.map((grant) => (
                <AccessGrantRow grant={grant} key={grant.tenantEntitlement.id} />
              ))}
            </section>

            <section className="access-trace-evidence" aria-label="Recent trace evidence">
              <header>
                <strong>Trace Evidence</strong>
                <span>{profile.recentTraces.length} recent traces</span>
              </header>
              {profile.recentTraces.length === 0 ? (
                <EmptyRow title="No trace evidence" detail="Set trace limit above 0 to include recent runtime decisions." />
              ) : null}
              {profile.recentTraces.map((trace) => (
                <article className="access-trace-row" key={trace.id}>
                  <div className={`trace-decision tone-${trace.decision === "allowed" ? "success" : "danger"}`}>
                    {trace.decision === "allowed" ? <CheckCircle2 size={15} /> : <LockKeyhole size={15} />}
                  </div>
                  <div>
                    <strong>{names[trace.callerAgentId ?? ""] ?? trace.callerAgentId ?? "anonymous"} → {names[trace.targetAgentId] ?? trace.targetAgentId}</strong>
                    <span>{trace.capabilityId ?? `${trace.routeType}:${trace.routeKey || "default"}`} · {summarizeDataScopes(trace.dataScopes)} · {trace.reason || trace.decision}</span>
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

function AccessGrantRow({ grant }: { grant: TenantAccessProfileGrant }) {
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
          <Badge tone={grant.tenantEntitlement.effect === "allow" ? "success" : "danger"}>{grant.tenantEntitlement.effect}</Badge>
        </div>
      </div>
      <div className="access-scope-line">
        <span>Tenant entitlement</span>
        <code>{grant.tenantEntitlement.id}</code>
        <span>{summarizeDataScopes(grant.effectiveTenantDataScopes)}</span>
      </div>
      {grant.scopeReason ? <p className="access-invalid-reason">{grant.scopeReason}</p> : null}
      <div className="access-nested-list">
        {grant.workspaceAssignments.length === 0 ? (
          <EmptyRow title="No workspace assignments" detail="No workspace assignment matched this entitlement." />
        ) : null}
        {grant.workspaceAssignments.map((workspace) => (
          <AccessWorkspaceRow key={workspace.workspaceAssignment.id} workspace={workspace} />
        ))}
      </div>
    </article>
  );
}

function AccessWorkspaceRow({ workspace }: { workspace: TenantAccessProfileWorkspace }) {
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
          <span className="access-empty-inline">No caller instances</span>
        ) : null}
        {workspace.instanceAssignments.map((instance) => (
          <AccessInstanceRow instance={instance} key={instance.instanceAssignment.id} />
        ))}
      </div>
    </div>
  );
}

function AccessInstanceRow({ instance }: { instance: TenantAccessProfileInstance }) {
  return (
    <div className="access-instance-row">
      <div>
        <strong>{instance.callerInstance?.name ?? instance.instanceAssignment.callerInstanceId}</strong>
        <span>{instance.instanceAssignment.subjectSelector || "all subjects"} · {summarizeDataScopes(instance.effectiveInstanceDataScopes)}</span>
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
          Target
          <select value={form.targetId} onChange={(event) => handleTargetChange(event.target.value)}>
            <option value="">All MCP targets</option>
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
          {actionId === `refresh:${form.targetId}` ? "Refreshing" : "Refresh"}
        </button>
        {message ? <span className="capability-message">{message}</span> : null}
      </div>

      <div className="capability-layout">
        <div className="capability-catalog">
          <div className="table-wrap">
            <table className="capability-table">
              <thead>
                <tr>
                  <th>Capability</th>
                  <th>Target</th>
                  <th>Action</th>
                  <th>Risk</th>
                  <th>Status</th>
                  <th>Grants</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {visibleCapabilities.length === 0 ? (
                  <tr>
                    <td colSpan={7}>
                      <EmptyRow title="No capabilities" detail="Refresh an MCP target to populate tool capabilities." />
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
                      <td><Badge tone={capability.action === "delete" || capability.action === "admin" ? "danger" : capability.action === "export" ? "warning" : "info"}>{capability.action}</Badge></td>
                      <td><Badge tone={riskTone(capability.riskLevel)}>{capability.riskLevel}</Badge></td>
                      <td><Badge tone={capabilityStatusTone(capability.discoveryStatus)}>{capability.discoveryStatus}</Badge></td>
                      <td>
                        <strong>{entitlementIds.length}/{workspaceIds.length}/{instanceCount}</strong>
                        <span>tenant/workspace/instance</span>
                      </td>
                      <td>
                        {capability.discoveryStatus === "approved" ? (
                          <span className="muted-action">approved</span>
                        ) : (
                          <button
                            className="table-action"
                            disabled={actionId === capability.id}
                            onClick={() => onApprove(capability)}
                            type="button"
                          >
                            <ShieldCheck size={13} />
                            {actionId === capability.id ? "Approving" : "Approve"}
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
            <label>Tenant<input required value={form.tenantId} onChange={(event) => onChange({ ...form, tenantId: event.target.value })} /></label>
            <label>Workspace<input required value={form.workspaceId} onChange={(event) => onChange({ ...form, workspaceId: event.target.value })} /></label>
          </div>
          <label>
            Capability
            <select required value={form.capabilityId} onChange={(event) => handleCapabilityChange(event.target.value)}>
              <option value="">Select capability</option>
              {visibleCapabilities.map((capability) => (
                <option key={capability.id} value={capability.id}>
                  {capability.key}
                </option>
              ))}
            </select>
          </label>
          <div className="form-row">
            <label>
              Caller instance
              <select required value={form.callerInstanceId} onChange={(event) => onChange({ ...form, callerInstanceId: event.target.value })}>
                <option value="">Select caller</option>
                {agents.map((agent) => (
                  <option key={agent.id} value={agent.id}>
                    {agent.name}
                  </option>
                ))}
              </select>
            </label>
            <label>Subject selector<input placeholder="optional, e.g. user:*" value={form.subjectSelector} onChange={(event) => onChange({ ...form, subjectSelector: event.target.value })} /></label>
          </div>
          <div className="capability-scope-strip">
            <span>{selectedCapability?.sensitivity ?? "sensitivity"}</span>
            <span>{selectedCapability?.riskLevel ?? "risk"}</span>
            <span>{dataScopeText(selectedCapability?.dataScopes) || "no data scope"}</span>
          </div>
          <FormFooter
            message=""
            submitLabel={actionId === `grant:${form.capabilityId}` ? "Granting" : "Grant chain"}
          />
        </form>

        <div className="assignment-list">
          {tenantEntitlements.length === 0 ? (
            <EmptyRow title="No grant chains" detail="Tenant, workspace, and instance assignments will appear here." />
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
                  <span>{entitlement.tenantId} · {entitlement.effect} · {entitlement.status}</span>
                </div>
                <div className="assignment-metrics">
                  <span>{children.length} workspaces</span>
                  <span>{instanceCount} callers</span>
                </div>
              </article>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function ManagementAuditTable({ events }: { events: AuditEvent[] }) {
  return (
    <div className="table-wrap">
      <table className="audit-table">
        <thead>
          <tr>
            <th>Time</th>
            <th>Action</th>
            <th>Resource</th>
            <th>Actor</th>
            <th>Version</th>
            <th>Summary</th>
          </tr>
        </thead>
        <tbody>
          {events.length === 0 ? (
            <tr>
              <td colSpan={6}>
                <EmptyRow title="No management audit events" detail="Control-plane changes will appear here after they are recorded." />
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

function IconMore() {
  return (
    <button className="icon-button compact" title="More" type="button">
      <MoreHorizontal size={16} />
    </button>
  );
}

function IconOpen() {
  return (
    <button className="icon-button compact" title="Open" type="button">
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

function policyRetryText(policy: RoutePolicy) {
  if (!policy.retry) return "target retry";
  const statuses = policy.retry.statusCodes.length > 0 ? policy.retry.statusCodes.join("/") : "none";
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
