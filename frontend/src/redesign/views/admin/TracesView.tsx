import { useEffect, useMemo, useState } from "react";
import { accessTraceReasonLabel, formatDate } from "../../../consolePresenters";
import { tx } from "../../../localizedMessages";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { useAdminAudit, normalizeAuditRange, type AuditRangeKey } from "../../hooks/useAdminAudit";
import {
  auditResourceTypeLabelKey,
  filterAuditTimeline,
  traceSource,
  type AuditResultKind,
  type AuditTimelineRow,
} from "../../model/auditTimeline";
import { adminHash } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip } from "../../ui/Chip";
import { CodeBlock } from "../../ui/CodeBlock";
import { Switch } from "../../ui/Field";
import { Modal } from "../../ui/Modal";
import { EmptyState, LoadingState } from "../../ui/StateViews";
import { Table } from "../../ui/Table";
import type { TraceEvent } from "../../../types";
import type { AdminViewProps } from "./adminViewProps";
import { TracePolicyDetail } from "./TracePolicyDetail";

const resultFilters: readonly (AuditResultKind | "all")[] = ["all", "success", "denied", "failed"];

function normalizeResult(value: string | undefined): AuditResultKind | "all" {
  return resultFilters.includes(value as AuditResultKind) ? (value as AuditResultKind) : "all";
}

const resultTone: Record<AuditResultKind, "success" | "danger"> = {
  denied: "danger",
  failed: "danger",
  success: "success",
};

export function TracesView({ data, onRetry, params }: AdminViewProps) {
  const { language, t } = useRedesignI18n();
  const live = data.status === "live";
  const [range, setRange] = useState<AuditRangeKey>(normalizeAuditRange(params.range));
  const [type, setType] = useState(params.type || "all");
  const [result, setResult] = useState<AuditResultKind | "all">(normalizeResult(params.result));
  const [highRisk, setHighRisk] = useState(false);
  const [search, setSearch] = useState("");
  const [detail, setDetail] = useState<AuditTimelineRow | null>(null);
  const audit = useAdminAudit(live, range, { highRiskOnly: highRisk });

  // Filters stay local (no remount churn); the hash is kept in sync for deep
  // links and refreshes.
  useEffect(() => {
    const next = adminHash("traces", {
      ...(result !== "all" ? { result } : {}),
      ...(range !== "week" ? { range } : {}),
      ...(type !== "all" ? { type } : {}),
    });
    if (window.location.hash !== next) {
      window.history.replaceState(null, "", `${window.location.pathname}${window.location.search}${next}`);
    }
  }, [range, result, type]);

  const resourceTypes = useMemo(
    () => [...new Set(audit.rows.map((row) => row.resourceType).filter(Boolean))],
    [audit.rows],
  );
  const rows = useMemo(() => {
    let filtered = filterAuditTimeline(audit.rows, { result, type });
    if (highRisk) {
      filtered = filtered.filter((row) => {
        if (row.kind !== "trace") return false;
        const flags = (row.event as TraceEvent).riskFlags ?? [];
        return flags.length > 0;
      });
    }
    const needle = search.trim().toLowerCase();
    if (needle) {
      filtered = filtered.filter(
        (row) =>
          row.resource.toLowerCase().includes(needle)
          || (row.subject || "").toLowerCase().includes(needle)
          || (row.kind === "trace" ? accessTraceReasonLabel(
            (row.event as TraceEvent).reason,
            (row.event as TraceEvent).decision === "allowed" ? "allow" : "deny",
            t,
          ) : row.operation).toLowerCase().includes(needle),
      );
    }
    return filtered;
  }, [audit.rows, result, type, highRisk, search, t]);

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t("rd.nav.traces")}</h1>
          <p>{t("rd.page.traces.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      <div className="stack">
        <Card title={t("rd.audit.filtersTitle")}>
          <div className="form-grid form-grid-4">
            <div>
              <label className="field-label" htmlFor="audit-range">{t("rd.audit.range")}</label>
              <select
                className="select"
                id="audit-range"
                onChange={(event) => setRange(normalizeAuditRange(event.target.value))}
                value={range}
              >
                {(["today", "week", "month"] as const).map((key) => (
                  <option key={key} value={key}>{t(`rd.audit.range.${key}`)}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="field-label" htmlFor="audit-type">{t("rd.audit.typeFilter")}</label>
              <select className="select" id="audit-type" onChange={(event) => setType(event.target.value)} value={type}>
                <option value="all">{t("rd.audit.type.all")}</option>
                {resourceTypes.map((resourceType) => (
                  <option key={resourceType} value={resourceType}>{resourceTypeLabel(t, resourceType)}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="field-label" htmlFor="audit-result">{t("rd.audit.resultFilter")}</label>
              <select
                className="select"
                id="audit-result"
                onChange={(event) => setResult(normalizeResult(event.target.value))}
                value={result}
              >
                {resultFilters.map((key) => (
                  <option key={key} value={key}>{key === "all" ? t("rd.audit.result.all") : t(`rd.audit.result.${key}`)}</option>
                ))}
              </select>
            </div>
            <div className="audit-count">
              <Chip tone="neutral">{tx(t, "rd.audit.count", { count: rows.length })}</Chip>
            </div>
          </div>
          <div className="form-grid form-grid-4">
            <div>
              <label className="field-label" htmlFor="audit-search">{t("rd.audit.searchPlaceholder")}</label>
              <input
                aria-label={t("rd.audit.searchPlaceholder")}
                className="input"
                id="audit-search"
                onChange={(event) => setSearch(event.target.value)}
                placeholder={t("rd.audit.searchPlaceholder")}
                type="search"
                value={search}
              />
            </div>
            <div className="apr-button-row">
              <Switch checked={highRisk} label={t("rd.audit.highRiskOnly")} onChange={setHighRisk} />
            </div>
          </div>
        </Card>

        <Card
          flush
          right={
            audit.loading ? (
              <Chip icon={false} tone="neutral">{t("rd.common.loading")}</Chip>
            ) : (
              <Chip tone="neutral">{t("rd.audit.merged")}</Chip>
            )
          }
          title={t("rd.audit.tableTitle")}
        >
          <Table
            caption={t("rd.audit.tableTitle")}
            empty={
              audit.loading ? (
                <LoadingState label={t("rd.common.loading")} />
              ) : audit.failed ? (
                <EmptyState desc={t("rd.audit.failedDesc")} title={t("rd.audit.failed")} />
              ) : (
                <EmptyState desc={t("rd.audit.emptyDesc")} title={t("rd.common.empty")} />
              )
            }
            renderCell={(row, column) => renderAuditCell(row, column, language, t, setDetail)}
            rowKey={(row) => row.id}
            rows={rows}
            tableId="adminTraces"
          />
        </Card>
      </div>
      <Modal onClose={() => setDetail(null)} open={detail !== null} size="wide" title={detail ? operationLabel(t, detail) : ""}>
        {detail ? (
          <div className="stack">
            <div className="mono small">{formatDate(detail.time, language)} · {detail.resource}</div>
            {detail.kind === "trace" ? (
              <TracePolicyDetail trace={detail.event as TraceEvent} t={t} />
            ) : null}
            <CodeBlock code={JSON.stringify(detail.event, null, 2)} label={t("rd.audit.detailJson")} />
          </div>
        ) : null}
      </Modal>
    </>
  );
}

type AuditColumnKey = "time" | "operation" | "resource" | "subject" | "result" | "summary";

function renderAuditCell(
  row: AuditTimelineRow,
  column: AuditColumnKey,
  language: ReturnType<typeof useRedesignI18n>["language"],
  t: ReturnType<typeof useRedesignI18n>["t"],
  onDetail: (row: AuditTimelineRow) => void,
) {
  switch (column) {
    case "time":
      return <span className="muted small">{formatDate(row.time, language)}</span>;
    case "operation":
      return operationLabel(t, row);
    case "resource":
      return <span className="mono small">{row.resource}</span>;
    case "subject":
      return <span className="mono small">{row.subject || "—"}</span>;
    case "result":
      return <Chip tone={resultTone[row.result]}>{t(`rd.audit.result.${row.result}`)}</Chip>;
    case "summary":
      if (row.kind === "trace") {
        const trace = row.event as TraceEvent;
        return (
          <div className="trace-summary">
            <Chip tone="neutral" icon={false}>{t(`rd.audit.source.${traceSource(trace)}`)}</Chip>
            <span className="small muted">
              {accessTraceReasonLabel(trace.reason, trace.decision === "allowed" ? "allow" : "deny", t)}
            </span>
            {trace.riskFlags && trace.riskFlags.length > 0 ? (
              <span className="apr-button-row">
                {trace.riskFlags.map((flag) => (
                  <Chip key={flag} tone="danger">{flag}</Chip>
                ))}
              </span>
            ) : null}
            {trace.ruleRef ? <span className="mono small muted">{trace.ruleRef}</span> : null}
            <Button onClick={() => onDetail(row)} size="sm" variant="link">
              {t("rd.audit.detail")}
            </Button>
          </div>
        );
      }
      return (
        <Button onClick={() => onDetail(row)} size="sm" variant="link">
          {t("rd.audit.detail")}
        </Button>
      );
  }
}

export function operationLabel(t: ReturnType<typeof useRedesignI18n>["t"], row: AuditTimelineRow): string {
  if (row.kind === "trace") return t(`rd.audit.call.${row.result}`);
  return t(`auditAction.${row.operation}`, row.operation);
}

function resourceTypeLabel(t: ReturnType<typeof useRedesignI18n>["t"], resourceType: string): string {
  if (resourceType === "trace") return t("rd.audit.type.trace");
  const key = auditResourceTypeLabelKey(resourceType);
  return key ? t(key) : resourceType;
}
