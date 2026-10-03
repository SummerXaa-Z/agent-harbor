import { Activity, Boxes, ClipboardCheck, FileSearch, RefreshCw } from "lucide-react";
import { useMemo, useState } from "react";
import { formatDate } from "../../../consolePresenters";
import { tx } from "../../../localizedMessages";
import { permissionApprovalStatusTone } from "../../../permissionWorkbenchPresenters";
import { useAdminAudit } from "../../hooks/useAdminAudit";
import { useApprovals } from "../../hooks/useApprovals";
import { useDailyMetrics } from "../../hooks/useDailyMetrics";
import { useEnvChecks } from "../../hooks/useEnvChecks";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { approvalList } from "../../model/approvalReview";
import { todayAuditCount, type AuditTimelineRow } from "../../model/auditTimeline";
import { trendHasData, trendSummary, type DailyTrend } from "../../model/dailyTrend";
import type { EnvCheckRow } from "../../model/envChecks";
import { adminHash } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip } from "../../ui/Chip";
import { Modal } from "../../ui/Modal";
import { Notice } from "../../ui/Banner";
import { EmptyState } from "../../ui/StateViews";
import { Kpi } from "../../ui/Kpi";
import { Table } from "../../ui/Table";
import { agentName, templateName } from "../user/AccessContextSwitcher";
import { operationLabel } from "./TracesView";
import { usePermissionCatalog } from "../../hooks/usePermissionCatalog";
import type { AdminViewProps } from "./adminViewProps";

const recentAuditLimit = 6;

export function CockpitView({ data, onRetry, session }: AdminViewProps) {
  const { language, t } = useRedesignI18n();
  const live = data.status === "live";
  const approvals = useApprovals(live, session);
  const catalog = usePermissionCatalog(live);
  const audit = useAdminAudit(live, "week");
  const metrics = useDailyMetrics(live);
  const checks = useEnvChecks(data);
  const [fixRow, setFixRow] = useState<EnvCheckRow | null>(null);

  const entries = useMemo(() => approvalList(approvals.requests), [approvals.requests]);
  const pending = entries.filter((entry) => entry.status === "pending");
  const todayCount = metrics.trend?.points.length
    ? metrics.trend.points[metrics.trend.points.length - 1].auditEvents
    : todayAuditCount(audit.rows);
  const agents = data.data?.agents ?? [];

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t("rd.nav.cockpit")}</h1>
          <p>{t("rd.page.cockpit.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      <div className="stack">
        <div className="grid grid-4">
          <Kpi icon={<ClipboardCheck aria-hidden="true" size={17} />} label={t("rd.cockpit.kpi.pending")} value={pending.length} />
          <Kpi icon={<FileSearch aria-hidden="true" size={17} />} label={t("rd.cockpit.kpi.todayAudit")} value={todayCount} />
          <Kpi icon={<Boxes aria-hidden="true" size={17} />} label={t("rd.cockpit.kpi.agents")} value={agents.length} />
          <Kpi
            hint={checks.readinessCount ? undefined : t("rd.cockpit.kpi.checksHint")}
            icon={<Activity aria-hidden="true" size={17} />}
            label={t("rd.cockpit.kpi.checks")}
            value={
              checks.readinessCount ? (
                <>
                  {checks.readinessCount.ready}
                  <span className="kpi-frac">/{checks.readinessCount.total}</span>
                </>
              ) : (
                "—"
              )
            }
          />
        </div>

        <Card sub={metrics.unsupported ? t("rd.trend.unsupported") : t("rd.trend.subtitle")} title={t("rd.trend.title")}>
          {metrics.unsupported ? (
            <EmptyState desc={t("rd.trend.unsupportedDesc")} title={t("rd.trend.unsupported")} />
          ) : !metrics.trend || !trendHasData(metrics.trend) ? (
            <EmptyState desc={t("rd.trend.emptyDesc")} title={t("rd.trend.empty")} />
          ) : (
            <div className="stack">
              <div className="trend-grid">
                <div className="chart-box">
                  <div className="chart-legend">
                    <span><i className="leg-dot leg-calls" />{t("rd.trend.calls")}</span>
                    <span><i className="leg-dot leg-audit" />{t("rd.trend.auditEvents")}</span>
                  </div>
                  <CallsChart trend={metrics.trend} />
                </div>
                <div className="chart-box">
                  <div className="chart-legend">
                    <span><i className="leg-dot leg-deny" />{t("rd.trend.denyRate")}</span>
                  </div>
                  <DenyChart trend={metrics.trend} />
                </div>
              </div>
              <p className="trend-summary">{trendSummary(t, metrics.trend)}</p>
            </div>
          )}
        </Card>

        <div className="cockpit-grid">
          <div className="stack">
            <Card
              right={
                <>
                  {checks.summary.abnormal > 0 ? (
                    <Chip tone={checks.summary.error > 0 ? "danger" : "warning"}>
                      {tx(t, "rd.cockpit.checkAbnormal", { count: checks.summary.abnormal })}
                    </Chip>
                  ) : (
                    <Chip tone="success">{t("rd.cockpit.checkOk")}</Chip>
                  )}
                  <Button
                    disabled={checks.checking}
                    icon={<RefreshCw aria-hidden="true" className={checks.checking ? "spin" : undefined} size={14} />}
                    onClick={() => void checks.recheck()}
                    size="sm"
                    variant="ghost"
                  >
                    {t("rd.cockpit.recheck")}
                  </Button>
                </>
              }
              title={t("rd.cockpit.checksTitle")}
            >
              <ul className="check-list">
                {(["api", "mcp", "corePath", "health"] as const).map((key) => {
                  const row = checks.rows.find((candidate) => candidate.key === key);
                  if (!row) return null;
                  return (
                    <li className="check-line" key={key}>
                      <div className="check-main">
                        <div className="check-name">{t(`rd.envcheck.${key}.name`)}</div>
                        <div className="check-sub">{rowDetail(t, row)}</div>
                      </div>
                      {row.status !== "ok" && (row.fixKeys.length > 0 || row.detail) ? (
                        <Button onClick={() => setFixRow(row)} size="sm" variant="link">{t("rd.cockpit.fixGuide")}</Button>
                      ) : null}
                      <Chip tone={checkTone(row.status)}>{t(`rd.envcheck.status.${row.status}`)}</Chip>
                    </li>
                  );
                })}
              </ul>
            </Card>

            <Card
              flush
              right={<a className="text-link" href={adminHash("traces")}>{t("rd.cockpit.openTraces")}</a>}
              title={t("rd.cockpit.recentTitle")}
            >
              <Table
                caption={t("rd.cockpit.recentTitle")}
                empty={<EmptyState desc={t("rd.audit.emptyDesc")} title={t("rd.common.empty")} />}
                renderCell={(row, column) => renderRecentCell(row, column, language, t)}
                rowKey={(row) => row.id}
                rows={audit.rows.slice(0, recentAuditLimit)}
                tableId="adminRecentAudit"
              />
            </Card>
          </div>

          <div className="stack">
            <Card
              right={<Chip tone={pending.length > 0 ? "warning" : "neutral"}>{pending.length}</Chip>}
              title={t("rd.cockpit.pendingTitle")}
            >
              {pending.length === 0 ? (
                <EmptyState desc={t("rd.cockpit.pendingEmptyDesc")} title={t("rd.cockpit.pendingEmpty")} />
              ) : (
                <div className="stack">
                  {pending.slice(0, 2).map((entry) => (
                    <div key={entry.request.id}>
                      <div className="cell-main">{templateName(catalog.templates, entry.request.templateId)}</div>
                      <div className="cell-sub">
                        {tx(t, "rd.cockpit.pendingSub", {
                          caller: agentName(agents, entry.request.callerInstanceId),
                          requester: entry.request.requestedBy || t("rd.common.none"),
                          target: agentName(agents, entry.request.targetId),
                        })}
                      </div>
                      <Chip tone={permissionApprovalStatusTone(entry.status)}>{entry.status}</Chip>
                    </div>
                  ))}
                  <Button href={adminHash("approvals", { status: "pending" })} variant="primary">
                    {t("rd.cockpit.goApprovals")}
                  </Button>
                </div>
              )}
            </Card>

            <Card title={t("rd.cockpit.riskTitle")}>
              {pending.length === 0 ? (
                <p className="muted">{t("rd.cockpit.riskEmpty")}</p>
              ) : (
                <Notice tone="warn">{tx(t, "rd.cockpit.riskPending", { count: pending.length })}</Notice>
              )}
            </Card>
          </div>
        </div>
      </div>

      <Modal onClose={() => setFixRow(null)} open={fixRow !== null} size="narrow" title={t("rd.cockpit.fixTitle")}>
        {fixRow ? (
          <div className="stack">
            <div className="check-sub">{rowDetail(t, fixRow)}</div>
            {fixRow.fixKeys.length > 0 ? (
              <ul className="fix-list">
                {fixRow.fixKeys.map((key) => (
                  <li key={key}>{t(key)}</li>
                ))}
              </ul>
            ) : null}
            {fixRow.detail ? <div className="mono small">{fixRow.detail}</div> : null}
          </div>
        ) : null}
      </Modal>
    </>
  );
}

function checkTone(status: EnvCheckRow["status"]): "success" | "warning" | "danger" | "neutral" {
  if (status === "ok") return "success";
  if (status === "error") return "danger";
  if (status === "warning") return "warning";
  return "neutral";
}

export function rowDetail(t: ReturnType<typeof useRedesignI18n>["t"], row: EnvCheckRow): string {
  return row.subParams ? tx(t, row.subKey, row.subParams) : t(row.subKey);
}

type RecentColumnKey = "time" | "operation" | "resource" | "result" | "subject";

function renderRecentCell(
  row: AuditTimelineRow,
  column: RecentColumnKey,
  language: ReturnType<typeof useRedesignI18n>["language"],
  t: ReturnType<typeof useRedesignI18n>["t"],
) {
  switch (column) {
    case "time":
      return <span className="muted small">{formatDate(row.time, language)}</span>;
    case "operation":
      return operationLabel(t, row);
    case "resource":
      return <span className="mono small">{row.resource}</span>;
    case "result":
      return <Chip tone={row.result === "success" ? "success" : "danger"}>{t(`rd.audit.result.${row.result}`)}</Chip>;
    case "subject":
      return <span className="mono small">{row.subject || "—"}</span>;
  }
}

const chartWidth = 460;
const chartHeight = 150;
const chartPad = { bottom: 22, left: 8, right: 8, top: 10 };

// Grouped neutral bars (calls, audit events); data series never use brand blue.
function CallsChart({ trend }: { trend: DailyTrend }) {
  const { t } = useRedesignI18n();
  const max = Math.max(1, ...trend.points.map((point) => Math.max(point.calls, point.auditEvents)));
  const innerW = chartWidth - chartPad.left - chartPad.right;
  const innerH = chartHeight - chartPad.top - chartPad.bottom;
  const groupW = innerW / Math.max(1, trend.points.length);
  const barW = Math.min(10, groupW / 3.2);
  return (
    <svg
      className="chart-svg"
      role="img"
      aria-label={t("rd.trend.callsChartLabel")}
      viewBox={`0 0 ${chartWidth} ${chartHeight}`}
    >
      {trend.points.map((point, index) => {
        const groupX = chartPad.left + index * groupW;
        const callsH = (point.calls / max) * innerH;
        const auditH = (point.auditEvents / max) * innerH;
        const day = dayLabel(point.date);
        return (
          <g key={point.date}>
            <rect
              className="bar-calls"
              height={Math.max(point.calls > 0 ? 2 : 0, callsH)}
              rx={2}
              width={barW}
              x={groupX + groupW / 2 - barW - 1.5}
              y={chartPad.top + innerH - Math.max(point.calls > 0 ? 2 : 0, callsH)}
            >
              <title>{`${day} · ${t("rd.trend.calls")} ${point.calls}`}</title>
            </rect>
            <rect
              className="bar-audit"
              height={Math.max(point.auditEvents > 0 ? 2 : 0, auditH)}
              rx={2}
              width={barW}
              x={groupX + groupW / 2 + 1.5}
              y={chartPad.top + innerH - Math.max(point.auditEvents > 0 ? 2 : 0, auditH)}
            >
              <title>{`${day} · ${t("rd.trend.auditEvents")} ${point.auditEvents}`}</title>
            </rect>
            <text className="chart-tick" textAnchor="middle" x={groupX + groupW / 2} y={chartHeight - 6}>
              {day}
            </text>
          </g>
        );
      })}
      <line className="chart-axis" x1={chartPad.left} x2={chartWidth - chartPad.right} y1={chartPad.top + innerH} y2={chartPad.top + innerH} />
    </svg>
  );
}

// Deny-rate line with a soft area under it; days without calls (null rate)
// leave a gap instead of pretending the rate was zero.
function DenyChart({ trend }: { trend: DailyTrend }) {
  const { t } = useRedesignI18n();
  const innerW = chartWidth - chartPad.left - chartPad.right;
  const innerH = chartHeight - chartPad.top - chartPad.bottom;
  const n = Math.max(1, trend.points.length);
  const x = (index: number) => chartPad.left + (index + 0.5) * (innerW / n);
  const y = (rate: number) => chartPad.top + innerH - rate * innerH;
  const segments: { points: { index: number; rate: number }[] }[] = [];
  let current: { index: number; rate: number }[] = [];
  trend.points.forEach((point, index) => {
    if (point.denyRate === null) {
      if (current.length > 0) segments.push({ points: current });
      current = [];
    } else {
      current.push({ index, rate: point.denyRate });
    }
  });
  if (current.length > 0) segments.push({ points: current });

  return (
    <svg
      className="chart-svg"
      role="img"
      aria-label={t("rd.trend.denyChartLabel")}
      viewBox={`0 0 ${chartWidth} ${chartHeight}`}
    >
      {[0, 0.5, 1].map((grid) => (
        <line className="chart-grid" key={grid} x1={chartPad.left} x2={chartWidth - chartPad.right} y1={y(grid)} y2={y(grid)} />
      ))}
      {segments.map((segment) => {
        const line = segment.points.map((point) => `${x(point.index)},${y(point.rate)}`).join(" ");
        const first = segment.points[0];
        const last = segment.points[segment.points.length - 1];
        const area = `${x(first.index)},${y(0)} ${line} ${x(last.index)},${y(0)}`;
        return (
          <g key={`${first.index}-${last.index}`}>
            <polygon className="deny-area" points={area} />
            <polyline className="deny-line" fill="none" points={line} />
            {segment.points.map((point) => (
              <circle className="deny-dot" key={point.index} r={2.5} cx={x(point.index)} cy={y(point.rate)}>
                <title>{`${dayLabel(trend.points[point.index].date)} · ${Math.round(point.rate * 100)}%`}</title>
              </circle>
            ))}
          </g>
        );
      })}
      {[0, 50, 100].map((percent) => (
        <text className="chart-tick" key={percent} x={chartWidth - chartPad.right} textAnchor="end" y={y(percent / 100) - 3}>
          {percent}%
        </text>
      ))}
    </svg>
  );
}

function dayLabel(date: string): string {
  const parsed = new Date(`${date}T00:00:00`);
  if (Number.isNaN(parsed.getTime())) return date;
  return `${parsed.getMonth() + 1}/${parsed.getDate()}`;
}
