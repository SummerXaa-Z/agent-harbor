import { History, KeyRound, ShieldCheck, ShieldX } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { fetchTenantAccessProfile } from "../../../api";
import { formatDate } from "../../../consolePresenters";
import type { CapabilityRisk, TenantAccessProfile } from "../../../types";
import { useAccessContext } from "../../hooks/useAccessContext";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { useUserRecords } from "../../hooks/useUserRecords";
import { apiErrorPresentation } from "../../model/apiErrorCategory";
import { maskSecret } from "../../model/secretMask";
import { keyStatus, myPermissions } from "../../model/userWorkbench";
import { userHash, viewLabelKey } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Notice } from "../../ui/Banner";
import { Card } from "../../ui/Card";
import { Chip, Risk, TagOutline, type RiskLevel } from "../../ui/Chip";
import { Kpi } from "../../ui/Kpi";
import { KvList } from "../../ui/KvList";
import { EmptyState, LoadingState } from "../../ui/StateViews";
import { Table } from "../../ui/Table";
import { dataScopeLabel } from "./AskView";
import type { UserViewProps } from "./userViewProps";

const riskLevel: Record<CapabilityRisk, RiskLevel> = { critical: "high", high: "high", low: "low", medium: "mid" };
const keyStatusTone = { active: "success", expired: "neutral", revoked: "danger" } as const;

export function MineView({ data, onRetry, params }: UserViewProps) {
  const { language, t } = useRedesignI18n();
  const live = Boolean(data.data?.loadedFromApi);
  const consoleData = data.data;
  const accessContext = useAccessContext(data, {});
  const records = useUserRecords(live);
  const agents = useMemo(() => consoleData?.agents ?? [], [consoleData]);
  const callers = agents.filter((agent) => agent.channelType === "local");
  const callerId = params.caller || accessContext.context.callerInstanceId || callers[0]?.id || "";
  const caller = agents.find((agent) => agent.id === callerId);
  const [profile, setProfile] = useState<TenantAccessProfile | null>(null);
  const [profileError, setProfileError] = useState<unknown>(null);
  const [profileLoading, setProfileLoading] = useState(false);

  useEffect(() => {
    if (!live || !caller) return;
    const controller = new AbortController();
    setProfileLoading(true);
    setProfileError(null);
    fetchTenantAccessProfile(caller.tenantId, "", { callerInstanceId: caller.id, traceLimit: 10, workspaceId: caller.workspaceId }, controller.signal)
      .then(setProfile)
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        setProfile(null);
        setProfileError(error);
      })
      .finally(() => {
        if (!controller.signal.aborted) setProfileLoading(false);
      });
    return () => controller.abort();
  }, [caller, live]);

  const permissions = useMemo(
    () =>
      myPermissions({
        approvals: records.approvals,
        applications: accessContext.applications,
        callerInstanceId: callerId,
        capabilities: consoleData?.capabilities ?? [],
        keys: records.keys,
        profile,
      }),
    [accessContext.applications, callerId, consoleData, profile, records.approvals, records.keys],
  );
  const callerKeys = records.keys.filter((key) => key.agentId === callerId);
  const targetNames = [...new Set(permissions.rows.map((row) => row.capability.targetId))]
    .map((id) => agents.find((agent) => agent.id === id)?.name ?? id);
  const subjectSelectors = [...new Set(callerKeys.map((key) => key.subjectSelector).filter(Boolean))];

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t(viewLabelKey("mine"))}</h1>
          <p>{t("rd.page.mine.desc")}</p>
        </div>
        <div className="page-head-aside">
          {callers.length > 1 ? (
            <select
              aria-label={t("rd.ask.caller")}
              className="select"
              onChange={(event) => {
                window.location.hash = userHash("mine", { caller: event.target.value });
              }}
              value={callerId}
            >
              {callers.map((agent) => (
                <option key={agent.id} value={agent.id}>
                  {agent.name}
                </option>
              ))}
            </select>
          ) : null}
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      {!consoleData ? (
        <LoadingState label={t("rd.data.loading")} />
      ) : !caller ? (
        <Card>
          <EmptyState desc={t("rd.mine.noCallerDesc")} title={t("rd.mine.noCaller")} />
        </Card>
      ) : (
        <div className="stack">
          <div className="grid grid-4">
            <Kpi icon={<ShieldCheck aria-hidden="true" size={17} />} label={t("rd.mine.kpi.allowed")} value={permissions.kpis.allowed} />
            <Kpi icon={<ShieldX aria-hidden="true" size={17} />} label={t("rd.mine.kpi.blocked")} value={permissions.kpis.blocked} />
            <Kpi icon={<KeyRound aria-hidden="true" size={17} />} label={t("rd.mine.kpi.activeTokens")} value={permissions.kpis.activeTokens} />
            <Kpi icon={<History aria-hidden="true" size={17} />} label={t("rd.mine.kpi.historicalTokens")} value={permissions.kpis.historicalTokens} />
          </div>

          {profileError ? <Notice tone="danger">{apiErrorPresentation(t, language, profileError, "error.loadTenantAccessProfile").detail}</Notice> : null}

          <Card flush title={t("rd.mine.capabilitiesTitle")}>
            {profileLoading && !profile ? (
              <div className="card-b">
                <LoadingState label={t("rd.common.loading")} />
              </div>
            ) : (
              <Table
                caption={t("rd.mine.capabilitiesTitle")}
                empty={<EmptyState desc={t("rd.mine.capabilitiesEmptyDesc")} title={t("rd.mine.capabilitiesEmpty")} />}
                renderCell={(row, column) => {
                  switch (column) {
                    case "capability":
                      return (
                        <>
                          <div className="cell-main">{row.capability.displayName || row.capability.key}</div>
                          <div className="cell-sub mono">{row.capability.key}</div>
                        </>
                      );
                    case "type":
                      return <TagOutline>{t(`rd.capType.${row.capability.type}`, row.capability.type)}</TagOutline>;
                    case "risk":
                      return (
                        <Risk
                          glyph={t(`rd.risk.glyph.${row.capability.riskLevel}`)}
                          label={t(`rd.risk.${row.capability.riskLevel}`)}
                          level={riskLevel[row.capability.riskLevel] ?? "mid"}
                        />
                      );
                    case "dataScope":
                      return (
                        <span className="small">{row.dataScopes.map(dataScopeLabel).filter(Boolean).join("; ") || t("rd.common.none")}</span>
                      );
                    case "decision":
                      return row.decision === "allowed" ? (
                        <Chip tone="success">{t("rd.decision.allowed")}</Chip>
                      ) : (
                        <Chip tone="danger">{t("rd.decision.blocked")}</Chip>
                      );
                    case "approval":
                      return row.approvalId ? <span className="mono small">{row.approvalId}</span> : <span className="muted">{t("rd.common.none")}</span>;
                  }
                }}
                rowKey={(row) => row.id}
                rows={permissions.rows}
                tableId="userCapabilities"
              />
            )}
          </Card>

          <div className="grid grid-2">
            <Card title={t("rd.mine.identityTitle")}>
              <div className="stack">
                <KvList
                  items={[
                    { key: "caller", label: t("rd.ask.caller"), value: <>{caller.name} <span className="mono small muted">{caller.id}</span></> },
                    { key: "tenant", label: t("rd.col.tenant"), value: <span className="mono">{caller.tenantId}</span> },
                    { key: "workspace", label: t("rd.col.workspace"), value: <span className="mono">{caller.workspaceId}</span> },
                    { key: "targets", label: t("rd.mine.targets"), value: targetNames.join(", ") || t("rd.common.none") },
                    {
                      key: "subjects",
                      label: t("rd.col.subject"),
                      value: subjectSelectors.length > 0 ? <span className="mono">{subjectSelectors.join(", ")}</span> : t("rd.common.none"),
                    },
                  ]}
                />
                <Notice>{t("rd.mine.boundary")}</Notice>
              </div>
            </Card>
            <Card flush title={t("rd.mine.credentialsTitle")}>
              <Table
                caption={t("rd.mine.credentialsTitle")}
                renderCell={(row, column) => {
                  const status = keyStatus(row);
                  switch (column) {
                    case "token":
                      return <span className="mono small">{maskSecret(row.prefix)}</span>;
                    case "status":
                      return <Chip tone={keyStatusTone[status]}>{t(`rd.tokenStatus.${status}`)}</Chip>;
                    case "expiresAt":
                      return formatDate(row.expiresAt, language);
                    case "subject":
                      return row.subjectSelector ? <span className="mono small">{row.subjectSelector}</span> : <span className="muted">{t("rd.common.none")}</span>;
                    case "actions":
                      return row.createdForHandoffId && status === "active" ? (
                        <a className="text-link" href={userHash("golive")}>
                          {t("rd.mine.manageToken")}
                        </a>
                      ) : (
                        <span className="muted">{t("rd.common.none")}</span>
                      );
                  }
                }}
                rowKey={(row) => row.id}
                rows={callerKeys}
                tableId="userTokens"
              />
            </Card>
          </div>
        </div>
      )}
    </>
  );
}
