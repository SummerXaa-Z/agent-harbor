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
import { createAccessGrant, createAgent, createAgentKey, disableAgent, loadConsoleData, revokeAccessGrant } from "./api";
import type {
  AccessGrant,
  Agent,
  AgentStatus,
  ChannelContract,
  ConsoleData,
  CreateAgentKeyResponse,
  EvidenceRun,
  JsonObject,
  ManagementScope,
  RoutePolicy,
  SystemMetric,
  TraceDecision,
  TraceEvent,
  TraceFilters
} from "./types";

type Tone = "success" | "warning" | "danger" | "info" | "neutral";

const navItems = [
  { key: "cockpit", label: "Cockpit", icon: Gauge },
  { key: "registry", label: "Registry", icon: Boxes },
  { key: "routes", label: "Routes", icon: Route },
  { key: "policies", label: "Policies", icon: ShieldCheck },
  { key: "traces", label: "Traces", icon: FileSearch },
  { key: "evidence", label: "Evidence", icon: ClipboardCheck }
];

const workspaceTabs = ["Prod", "Staging", "Sandbox"];
const defaultManagementScope: ManagementScope = {
  tenantId: "default",
  workspaceId: "workspace-demo"
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
const defaultGrantForm = { callerAgentId: "", routeKey: "", routeType: "mcp", targetAgentId: "" };
const defaultTraceFilters: TraceFilters = { callerAgentId: "", decision: "", runId: "", targetAgentId: "" };
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
  const [grantForm, setGrantForm] = useState(defaultGrantForm);
  const [grantMessage, setGrantMessage] = useState("");
  const [cleanupActionId, setCleanupActionId] = useState("");
  const [traceFilters, setTraceFilters] = useState<TraceFilters>(defaultTraceFilters);

  useEffect(() => {
    void refresh();
  }, []);

  async function refresh() {
    try {
      setLoadError("");
      const next = await loadConsoleData(adminKey, traceFilters, normalizedScope(scope));
      setData(next);
      setLastRefresh(new Date());
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "console data unavailable");
    }
  }

  async function submitAgent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAgentMessage("");
    try {
      const channelConfig: JsonObject = {};
      const endpoint = agentForm.endpoint.trim();
      if (endpoint) channelConfig.endpoint = endpoint;
      const retryMaxAttempts = Number.parseInt(agentForm.retryMaxAttempts, 10);
      const retryBackoffMs = Number.parseInt(agentForm.retryBackoffMs, 10);
      const retryAttemptsText = agentForm.retryMaxAttempts.trim();
      const retryBackoffText = agentForm.retryBackoffMs.trim();
      const retryRequested = retryAttemptsText !== "1" || retryBackoffText !== "0";
      if (!retryAttemptsText || Number.isNaN(retryMaxAttempts)) {
        setAgentMessage("Retry attempts must be a number.");
        return;
      }
      if (!retryBackoffText || Number.isNaN(retryBackoffMs)) {
        setAgentMessage("Retry backoff must be a number.");
        return;
      }
      if (retryRequested) {
        channelConfig.retry = {
          backoffMs: retryBackoffMs,
          maxAttempts: retryMaxAttempts
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

  async function handleDisableAgent(agent: Agent) {
    setAgentMessage("");
    setCleanupActionId(agent.id);
    try {
      await disableAgent(agent.id, adminKey);
      setAgentMessage(`${agent.name} disabled. Registry refreshed.`);
      await refresh();
    } catch (error) {
      setAgentMessage(error instanceof Error ? error.message : "Unable to disable agent");
    } finally {
      setCleanupActionId("");
    }
  }

  async function handleRevokeGrant(policy: RoutePolicy) {
    setGrantMessage("");
    setCleanupActionId(policy.id);
    try {
      await revokeAccessGrant(policy.id, adminKey);
      setGrantMessage("Access grant revoked. Route Governance refreshed.");
      await refresh();
    } catch (error) {
      setGrantMessage(error instanceof Error ? error.message : "Unable to revoke access grant");
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

  async function submitGrant(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setGrantMessage("");
    try {
      await createAccessGrant(
        {
          callerAgentId: grantForm.callerAgentId,
          routeKey: grantForm.routeKey.trim() || undefined,
          routeType: grantForm.routeType.trim() || undefined,
          targetAgentId: grantForm.targetAgentId
        },
        adminKey
      );
      setGrantMessage("Access grant created. Route Governance refreshed.");
      setGrantForm({ ...defaultGrantForm, callerAgentId: grantForm.callerAgentId });
      await refresh();
    } catch (error) {
      setGrantMessage(error instanceof Error ? error.message : "Unable to create grant");
    }
  }

  const agents = data?.agents ?? [];
  const grants = data?.accessGrants ?? [];
  const traces = data?.traces ?? [];
  const channels = data?.channels ?? [];
  const policies = useMemo(
    () => (data?.grantsLoadedFromApi ? grants.map(grantToPolicy) : data?.routePolicies ?? []),
    [data?.grantsLoadedFromApi, data?.routePolicies, grants]
  );
  const evidenceRuns = data?.evidenceRuns ?? [];
  const metrics = data?.systemMetrics ?? [];
  const localCallers = agents.filter((agent) => agent.status === "active" && agent.channelType === "local");

  const channelLabels = useMemo(() => {
    return channels.reduce<Record<string, string>>((acc, item) => {
      acc[item.key] = item.label;
      return acc;
    }, {});
  }, [channels]);

  const activeAgents = agents.filter((agent) => agent.status === "active").length;
  const activeGrants = policies.filter((policy) => policy.status === "enabled" && policy.effect === "allow").length;
  const deniedTraces = traces.filter((trace) => trace.decision === "denied").length;
  const allowedTraces = traces.filter((trace) => trace.decision === "allowed").length;
  const evidencePassed = evidenceRuns.filter((run) => run.status === "passed").length;
  const dataSourceLabel = loadError ? "API error" : data?.loadedFromApi ? "Go runtime + samples" : "Fallback dataset";

  return (
    <div className="app-shell">
      <aside className="sidebar" aria-label="Primary navigation">
        <div className="brand">
          <div className="brand-mark">
            <Network size={18} />
          </div>
          <div>
            <strong>AgentHarbor</strong>
            <span>Agent Gateway</span>
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
            <div className="breadcrumb">Gateway / {activeWorkspace} / Agent Governance</div>
            <h1>Agent Gateway Cockpit</h1>
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
          <MetricCard icon={<ServerCog size={18} />} label="Managed Agents" value={String(agents.length)} detail={`${activeAgents} active`} tone="info" />
          <MetricCard icon={<KeyRound size={18} />} label="Active Grants" value={String(activeGrants)} detail={data?.grantsLoadedFromApi ? "live access grants" : "sample fallback"} tone="success" />
          <MetricCard icon={<TriangleAlert size={18} />} label="Denied Traces" value={String(deniedTraces)} detail={`${allowedTraces} allowed`} tone={deniedTraces > 0 ? "warning" : "success"} />
          <MetricCard icon={<ClipboardCheck size={18} />} label="Evidence Health" value={`${evidencePassed}/${Math.max(evidenceRuns.length, 1)}`} detail="checks passed" tone="neutral" />
        </section>

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

          <Panel className="span-4" icon={<Route size={18} />} title="Create Grant">
            <GrantCreateForm agents={agents} form={grantForm} message={grantMessage} onChange={setGrantForm} onSubmit={submitGrant} />
          </Panel>

          <Panel className="span-8" icon={<Workflow size={18} />} title="Route Governance" action={<IconMore />}>
            <PolicyTable
              agents={agents}
              canRevoke={Boolean(data?.grantsLoadedFromApi)}
              onRevoke={handleRevokeGrant}
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
              onDisable={handleDisableAgent}
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
        </section>
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

function GrantCreateForm({
  agents,
  form,
  message,
  onChange,
  onSubmit
}: {
  agents: Agent[];
  form: typeof defaultGrantForm;
  message: string;
  onChange: (form: typeof defaultGrantForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
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
      <FormFooter message={message} submitLabel="Create grant" />
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
  canRevoke,
  onRevoke,
  pendingActionId,
  policies
}: {
  agents: Agent[];
  canRevoke: boolean;
  onRevoke: (policy: RoutePolicy) => void;
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
                <EmptyRow title="No route policies" detail="Create an access grant to populate governed routes." />
              </td>
            </tr>
          ) : null}
          {policies.map((policy) => (
            <tr className={policy.status === "disabled" ? "row-disabled" : undefined} key={policy.id}>
              <td>
                <strong>{policy.name}</strong>
                <span>priority {policy.priority} · matched {formatDate(policy.lastMatchedAt ?? policy.createdAt)}</span>
              </td>
              <td>{names[policy.callerAgentId] ?? policy.callerAgentId}</td>
              <td>{names[policy.targetAgentId] ?? policy.targetAgentId}</td>
              <td>
                <code>{policy.routeType}:{policy.routeKey || "default"}</code>
              </td>
              <td><Badge tone={policy.effect === "allow" ? "success" : "danger"}>{policy.effect}</Badge></td>
              <td>
                {canRevoke && policy.status === "enabled" ? (
                  <button
                    className="table-action"
                    disabled={pendingActionId === policy.id}
                    onClick={() => onRevoke(policy)}
                    type="button"
                  >
                    <LockKeyhole size={13} />
                    {pendingActionId === policy.id ? "Revoking" : "Revoke"}
                  </button>
                ) : (
                  <span className="muted-action">{policy.status === "disabled" ? "revoked" : "sample"}</span>
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
  onDisable,
  pendingActionId
}: {
  agents: Agent[];
  channelLabels: Record<string, string>;
  onDisable: (agent: Agent) => void;
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
                  <button
                    className="table-action"
                    disabled={pendingActionId === agent.id}
                    onClick={() => onDisable(agent)}
                    type="button"
                  >
                    <LockKeyhole size={13} />
                    {pendingActionId === agent.id ? "Disabling" : "Disable"}
                  </button>
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
        <article className="signal" key={metric.label}>
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
            <span>{trace.routeType}:{trace.routeKey || "default"} · {trace.reason || trace.decision}</span>
          </div>
          <time>{formatDate(trace.createdAt)}</time>
        </article>
      ))}
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

function grantToPolicy(grant: AccessGrant): RoutePolicy {
  return {
    callerAgentId: grant.callerAgentId,
    createdAt: grant.createdAt,
    effect: grant.revokedAt ? "deny" : "allow",
    id: grant.id,
    name: `${grant.routeType || "route"} grant`,
    priority: 10,
    routeKey: grant.routeKey,
    routeType: grant.routeType || "default",
    status: grant.revokedAt ? "disabled" : "enabled",
    targetAgentId: grant.targetAgentId
  };
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

export default App;
