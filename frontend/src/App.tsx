import { useEffect, useMemo, useState, type ReactNode } from "react";
import {
  Activity,
  Boxes,
  CheckCircle2,
  CircleDot,
  ClipboardCheck,
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
import { loadConsoleData } from "./api";
import type {
  Agent,
  ChannelContract,
  ConsoleData,
  EvidenceRun,
  RoutePolicy,
  SystemMetric,
  TraceEvent
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

function App() {
  const [activeNav, setActiveNav] = useState("cockpit");
  const [activeWorkspace, setActiveWorkspace] = useState("Prod");
  const [data, setData] = useState<ConsoleData | null>(null);
  const [loadError, setLoadError] = useState("");
  const [lastRefresh, setLastRefresh] = useState(new Date());

  useEffect(() => {
    void refresh();
  }, []);

  async function refresh() {
    try {
      setLoadError("");
      const next = await loadConsoleData();
      setData(next);
      setLastRefresh(new Date());
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "console data unavailable");
    }
  }

  const agents = data?.agents ?? [];
  const traces = data?.traces ?? [];
  const channels = data?.channels ?? [];
  const policies = data?.routePolicies ?? [];
  const evidenceRuns = data?.evidenceRuns ?? [];
  const metrics = data?.systemMetrics ?? [];

  const channelLabels = useMemo(() => {
    return channels.reduce<Record<string, string>>((acc, item) => {
      acc[item.key] = item.label;
      return acc;
    }, {});
  }, [channels]);

  const activeAgents = agents.filter((agent) => agent.status === "active").length;
  const deniedTraces = traces.filter((trace) => trace.decision === "denied").length;
  const allowedTraces = traces.filter((trace) => trace.decision === "allowed").length;
  const evidencePassed = evidenceRuns.filter((run) => run.status === "passed").length;
  const dataSourceLabel = data?.loadedFromApi ? "Go runtime + samples" : "Fallback dataset";

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
          <span>{data?.loadedFromApi ? "Live API" : "Mock fallback"}</span>
        </div>
      </aside>

      <main className="workspace">
        <header className="topbar">
          <div className="topbar-title">
            <div className="breadcrumb">Gateway / {activeWorkspace} / Agent Governance</div>
            <h1>Agent Gateway Cockpit</h1>
          </div>
          <div className="topbar-actions">
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
          {loadError ? <div className="strip-error">{loadError}</div> : null}
        </section>

        <section className="metric-grid" aria-label="Gateway metrics">
          <MetricCard icon={<ServerCog size={18} />} label="Managed Agents" value={String(agents.length)} detail={`${activeAgents} active`} tone="info" />
          <MetricCard icon={<KeyRound size={18} />} label="Active Grants" value={String(policies.length)} detail="route scoped" tone="success" />
          <MetricCard icon={<TriangleAlert size={18} />} label="Denied Traces" value={String(deniedTraces)} detail={`${allowedTraces} allowed`} tone={deniedTraces > 0 ? "warning" : "success"} />
          <MetricCard icon={<ClipboardCheck size={18} />} label="Evidence Health" value={`${evidencePassed}/${Math.max(evidenceRuns.length, 1)}`} detail="checks passed" tone="neutral" />
        </section>

        <section className="content-grid">
          <Panel className="span-8" icon={<Workflow size={18} />} title="Route Governance" action={<IconMore />}>
            <PolicyTable agents={agents} policies={policies} />
          </Panel>

          <Panel className="span-4" icon={<Sparkles size={18} />} title="Evidence Runs" action={<IconOpen />}>
            <EvidenceTimeline runs={evidenceRuns} />
          </Panel>

          <Panel className="span-8" icon={<Boxes size={18} />} title="Agent Registry" action={<IconMore />}>
            <AgentTable agents={agents} channelLabels={channelLabels} />
          </Panel>

          <Panel className="span-4" icon={<Layers3 size={18} />} title="Contract Matrix" action={<IconOpen />}>
            <ContractMatrix channels={channels} providers={data?.providers ?? []} />
          </Panel>

          <Panel className="span-5" icon={<DatabaseZap size={18} />} title="Runtime Signals" action={<IconMore />}>
            <SignalBoard metrics={metrics} />
          </Panel>

          <Panel className="span-7" icon={<FileSearch size={18} />} title="Audit Traces" action={<IconOpen />}>
            <TraceTable traces={traces} agents={agents} />
          </Panel>
        </section>
      </main>
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

function PolicyTable({ agents, policies }: { agents: Agent[]; policies: RoutePolicy[] }) {
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
          </tr>
        </thead>
        <tbody>
          {policies.length === 0 ? (
            <tr>
              <td colSpan={5}>
                <EmptyRow title="No route policies" detail="Create an access grant to populate governed routes." />
              </td>
            </tr>
          ) : null}
          {policies.map((policy) => (
            <tr key={policy.id}>
              <td>
                <strong>{policy.name}</strong>
                <span>priority {policy.priority} · matched {formatDate(policy.lastMatchedAt ?? policy.createdAt)}</span>
              </td>
              <td>{names[policy.callerAgentId] ?? policy.callerAgentId}</td>
              <td>{names[policy.targetAgentId] ?? policy.targetAgentId}</td>
              <td>
                <code>{policy.routeType}:{policy.routeKey}</code>
              </td>
              <td><Badge tone={policy.effect === "allow" ? "success" : "danger"}>{policy.effect}</Badge></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function AgentTable({ agents, channelLabels }: { agents: Agent[]; channelLabels: Record<string, string> }) {
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
          </tr>
        </thead>
        <tbody>
          {agents.length === 0 ? (
            <tr>
              <td colSpan={5}>
                <EmptyRow title="No agents registered" detail="Register caller and target agents to start routing traffic." />
              </td>
            </tr>
          ) : null}
          {agents.map((agent) => (
            <tr key={agent.id}>
              <td>
                <strong>{agent.name}</strong>
                <span>{agent.description || agent.id}</span>
              </td>
              <td>{channelLabels[agent.channelType] ?? agent.channelType}</td>
              <td className="truncate">{configText(agent, "endpoint") || "local runtime"}</td>
              <td><Badge tone={agent.status === "active" ? "success" : agent.status === "draft" ? "warning" : "neutral"}>{agent.status}</Badge></td>
              <td>{agent.ownerId || "platform"}</td>
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
