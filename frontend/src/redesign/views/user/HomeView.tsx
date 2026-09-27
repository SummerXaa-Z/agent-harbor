import { ArrowRight, Check, FilePlus2, Rocket, Search } from "lucide-react";
import { useMemo, type ReactNode } from "react";
import { formatDate } from "../../../consolePresenters";
import { tx } from "../../../localizedMessages";
import { permissionApprovalStatusLabel, permissionApprovalStatusTone } from "../../../permissionWorkbenchPresenters";
import { useAccessContext } from "../../hooks/useAccessContext";
import { usePermissionCatalog } from "../../hooks/usePermissionCatalog";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { useUserRecords } from "../../hooks/useUserRecords";
import { accessContextFromApprovalRequest, accessContextRouteParams } from "../../model/accessContext";
import {
  myRequestCounts,
  myRequests,
  myResources,
  onboardingProgress,
  workbenchActor,
  type OnboardingStepKey,
} from "../../model/userWorkbench";
import { adminHash, userHash } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Card } from "../../ui/Card";
import { Chip, TagOutline } from "../../ui/Chip";
import { EmptyState, LoadingState } from "../../ui/StateViews";
import { Table } from "../../ui/Table";
import { agentName, templateName } from "./AccessContextSwitcher";
import type { UserViewProps } from "./userViewProps";

const onboardingHref: Record<OnboardingStepKey, string> = {
  authorize: userHash("apply"),
  connect: adminHash("cockpit"),
  handoff: userHash("golive"),
  register: adminHash("registry"),
};

const quickCards: { href: string; icon: ReactNode; key: string }[] = [
  { href: userHash("ask"), icon: <Search aria-hidden="true" size={18} />, key: "ask" },
  { href: userHash("apply"), icon: <FilePlus2 aria-hidden="true" size={18} />, key: "apply" },
  { href: userHash("golive"), icon: <Rocket aria-hidden="true" size={18} />, key: "golive" },
];

const requestListLimit = 6;

export function HomeView({ data, onRetry, session }: UserViewProps) {
  const { language, t } = useRedesignI18n();
  const live = Boolean(data.data?.loadedFromApi);
  const consoleData = data.data;
  const accessContext = useAccessContext(data, {});
  const catalog = usePermissionCatalog(live);
  const records = useUserRecords(live);
  const actor = workbenchActor(session);
  const agents = useMemo(() => consoleData?.agents ?? [], [consoleData]);
  const capabilities = useMemo(() => consoleData?.capabilities ?? [], [consoleData]);

  const progress = onboardingProgress({
    agents,
    applications: accessContext.applications,
    capabilities,
    keys: records.keys,
    liveData: live,
  });
  const requests = useMemo(() => myRequests(records.approvals, actor, capabilities), [actor, capabilities, records.approvals]);
  const counts = myRequestCounts(requests);
  const resources = useMemo(
    () => myResources(agents, capabilities, consoleData?.traces ?? []),
    [agents, capabilities, consoleData],
  );

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{actor ? tx(t, "rd.home.greeting", { name: actor }) : t("rd.home.greetingAnonymous")}</h1>
          <p>{t("rd.page.home.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      {!consoleData ? (
        <LoadingState label={t("rd.data.loading")} />
      ) : (
        <div className="stack">
          <Card sub={tx(t, "rd.home.onboardingCount", { done: progress.doneCount, total: progress.total })} title={t("rd.home.onboardingTitle")}>
            <ol className="onb-list">
              {progress.steps.map((step, index) => {
                const current = step.key === progress.nextKey;
                return (
                  <li className={step.done ? "onb-item onb-done" : current ? "onb-item onb-current" : "onb-item"} key={step.key}>
                    <span className="onb-dot">{step.done ? <Check aria-hidden="true" size={14} /> : index + 1}</span>
                    <div className="onb-main">
                      <div className="onb-title">{t(`rd.home.step.${step.key}.title`)}</div>
                      <div className="onb-desc">{t(`rd.home.step.${step.key}.desc`)}</div>
                    </div>
                    <span className="onb-state">
                      {step.done ? (
                        t("rd.step.done")
                      ) : (
                        <a className="text-link" href={onboardingHref[step.key]}>
                          {t(`rd.home.step.${step.key}.go`)}
                          <ArrowRight aria-hidden="true" size={14} />
                        </a>
                      )}
                    </span>
                  </li>
                );
              })}
            </ol>
          </Card>

          <div className="grid grid-3">
            {quickCards.map((card) => (
              <a className="card quick-card" href={card.href} key={card.key}>
                <span className="kpi-ico">{card.icon}</span>
                <span className="quick-main">
                  <span className="quick-title">{t(`rd.home.quick.${card.key}.title`)}</span>
                  <span className="quick-desc">{t(`rd.home.quick.${card.key}.desc`)}</span>
                </span>
                <ArrowRight aria-hidden="true" size={16} />
              </a>
            ))}
          </div>

          <Card
            right={
              <div className="chip-row">
                <Chip tone="warning">{tx(t, "rd.home.requests.pending", { count: counts.pending })}</Chip>
                <Chip tone="success">{tx(t, "rd.home.requests.approved", { count: counts.approved })}</Chip>
                <Chip tone="danger">{tx(t, "rd.home.requests.rejected", { count: counts.rejected })}</Chip>
              </div>
            }
            title={t("rd.home.requestsTitle")}
          >
            {!records.loaded && live ? (
              <LoadingState label={t("rd.common.loading")} />
            ) : requests.length === 0 ? (
              <EmptyState desc={t("rd.home.requestsEmptyDesc")} title={t("rd.home.requestsEmpty")} />
            ) : (
              <ul className="req-list">
                {requests.slice(0, requestListLimit).map((row) => {
                  const context = accessContextFromApprovalRequest(row.request);
                  return (
                    <li className="req-item" key={row.request.id}>
                      <div className="req-main">
                        <a className="cell-main" href={userHash("apply", { ...accessContextRouteParams(context), approval: row.request.id })}>
                          {tx(t, "rd.context.option", {
                            caller: agentName(agents, row.request.callerInstanceId),
                            target: agentName(agents, row.request.targetId),
                            template: templateName(catalog.templates, row.request.templateId),
                          })}
                        </a>
                        <div className="cell-sub">
                          <span className="mono">{row.request.id}</span> · {formatDate(row.request.createdAt, language)}
                        </div>
                      </div>
                      <span className="req-counts">{tx(t, "rd.home.requests.counts", { allowed: row.allowedCount, blocked: row.blockedCount })}</span>
                      <Chip tone={permissionApprovalStatusTone(row.status)}>{permissionApprovalStatusLabel(row.status, t)}</Chip>
                    </li>
                  );
                })}
              </ul>
            )}
          </Card>

          <Card flush title={t("rd.home.resourcesTitle")}>
            <Table
              caption={t("rd.home.resourcesTitle")}
              renderCell={(row, column) => {
                switch (column) {
                  case "resource":
                    return (
                      <>
                        <div className="cell-main">{row.agent.name}</div>
                        <div className="cell-sub mono">{row.agent.id}</div>
                      </>
                    );
                  case "type":
                    return <TagOutline>{t(`rd.home.role.${row.role}`)}</TagOutline>;
                  case "status":
                    return (
                      <Chip tone={row.agent.status === "active" ? "success" : "neutral"}>
                        {row.agent.status === "active" ? t("rd.status.active") : t("rd.status.inactive")}
                      </Chip>
                    );
                  case "lastActivity":
                    return formatDate(row.lastActivity, language);
                }
              }}
              rowKey={(row) => row.agent.id}
              rows={resources}
              tableId="userResources"
            />
          </Card>
        </div>
      )}
    </>
  );
}
