import { useMemo, useState } from "react";
import {
  Activity,
  CheckCircle2,
  FileSearch,
  LockKeyhole
} from "lucide-react";

import {
  accessTraceReasonLabel,
  agentNameMap,
  formatDate,
  permissionEntityDisplayName,
  readableIdentifierLabel,
  translatedValue,
  type Tone,
  type Translator
} from "../consolePresenters";
import type {
  Agent,
  AuditEvent,
  AcceptanceRun,
  SystemMetric,
  TraceEvent
} from "../types";
import { TechnicalId } from "./TechnicalId";
import { Badge, EmptyRow } from "./ui";

export function AcceptanceHistoryTimeline({ runs, t }: { runs: AcceptanceRun[]; t: Translator }) {
  return (
    <div className="timeline">
      {runs.length === 0 ? <EmptyRow title={t("empty.acceptanceHistory.title")} detail={t("empty.acceptanceHistory.detail")} /> : null}
      {runs.map((run) => (
        <article className="timeline-row" key={run.id}>
          <div className={`timeline-marker tone-${toneFromStatus(run.status)}`}>
            {run.status === "passed" ? <CheckCircle2 size={14} /> : <Activity size={14} />}
          </div>
          <div>
            <div className="timeline-title">
              <strong>{run.title}</strong>
              <Badge tone={toneFromStatus(run.status)}>{acceptanceRunStatusLabel(run.status, t)}</Badge>
            </div>
            <p>{run.checks} {t("text.checks")} · {formatDuration(acceptanceRunDuration(run))} · {formatDate(run.completedAt ?? run.startedAt)}</p>
          </div>
        </article>
      ))}
    </div>
  );
}

export function SignalBoard({ metrics, t }: { metrics: SystemMetric[]; t: Translator }) {
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

export function TraceTable({ traces, agents, t }: { traces: TraceEvent[]; agents: Agent[]; t: Translator }) {
  const names = useMemo(() => agentNameMap(agents), [agents]);
  const [traceDetailsExpanded, setTraceDetailsExpanded] = useState(false);

  return (
    <div className="trace-list">
      {traces.length === 0 ? (
        <EmptyRow
          title={t("empty.auditTraces.title")}
          detail={t("empty.auditTraces.detail")}
          actionLabel={t("empty.auditTraces.action")}
          actionHash="#getting-started"
        />
      ) : null}
      {traces.length > 0 ? (
        <div className="trace-list-header">
          <span>{tx(t, "text.visibleTraceCount", { count: traces.length })}</span>
          <button className="secondary-button trace-detail-toggle" onClick={() => setTraceDetailsExpanded(!traceDetailsExpanded)} type="button">
            <FileSearch size={14} />
            {traceDetailsExpanded ? t("action.collapseTraceDetails") : t("action.expandTraceDetails")}
          </button>
        </div>
      ) : null}
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
              <details className="trace-technical-details" open={traceDetailsExpanded}>
                <summary>{t("text.traceDetails")}</summary>
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

export function ManagementAuditTable({ events, t }: { events: AuditEvent[]; t: Translator }) {
  if (events.length === 0) {
    return (
      <div className="management-audit-empty-state">
        <EmptyRow title={t("empty.managementAudit.title")} detail={t("empty.managementAudit.detail")} />
      </div>
    );
  }

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

function traceRouteBusinessLabel(trace: TraceEvent, t: Translator) {
  if (trace.routeType === "mcp") {
    if (trace.routeKey === "tools/call") return t("traceRoute.mcpToolsCall");
    if (trace.routeKey === "tools/list") return t("traceRoute.mcpToolsList");
    if (trace.routeKey === "initialize") return t("traceRoute.mcpInitialize");
    if (!trace.routeKey) return t("traceRoute.mcpDefault");
  }
  return trace.routeKey ? readableIdentifierLabel(trace.routeKey) : t("text.traceDefaultRoute");
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

function acceptanceRunStatusLabel(status: AcceptanceRun["status"], t: Translator) {
  if (status === "passed") return t("status.evidencePassed");
  if (status === "failed") return t("status.evidenceFailed");
  return t("status.evidenceWarning");
}

function auditTone(action: string): Tone {
  if (action.includes("delete") || action.includes("revoke") || action.includes("disable")) return "danger";
  if (action.includes("rotate") || action.includes("credentials")) return "warning";
  if (action.includes("create") || action.includes("enable")) return "success";
  return "info";
}

function acceptanceRunDuration(run: AcceptanceRun) {
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

function toneFromStatus(status: string): Tone {
  if (status === "passed") return "success";
  if (status === "warning") return "warning";
  if (status === "failed") return "danger";
  return "info";
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}
