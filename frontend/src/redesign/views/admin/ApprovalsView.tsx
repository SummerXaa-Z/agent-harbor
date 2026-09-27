import { CircleCheck, CircleX } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { formatDate } from "../../../consolePresenters";
import { tx } from "../../../localizedMessages";
import { permissionApprovalStatusLabel, permissionApprovalStatusTone } from "../../../permissionWorkbenchPresenters";
import { useApprovals } from "../../hooks/useApprovals";
import { usePermissionCatalog } from "../../hooks/usePermissionCatalog";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { apiErrorPresentation } from "../../model/apiErrorCategory";
import {
  approvalCapabilityRows,
  approvalList,
  approvalNoticeKey,
  approvalRiskSummary,
  approvalTabCounts,
  filterApprovalList,
  type ApprovalTab,
} from "../../model/approvalReview";
import { isSelfReview } from "../../model/demoRole";
import { adminHash } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Banner, Notice } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip } from "../../ui/Chip";
import { Field } from "../../ui/Field";
import { KvList, type KvItem } from "../../ui/KvList";
import { Modal } from "../../ui/Modal";
import { EmptyState, LoadingState } from "../../ui/StateViews";
import { agentName, templateName } from "../user/AccessContextSwitcher";
import type { AdminViewProps } from "./adminViewProps";

const tabs: readonly ApprovalTab[] = ["pending", "approved", "rejected", "all"];

function normalizeTab(value: string | undefined): ApprovalTab {
  return tabs.includes(value as ApprovalTab) ? (value as ApprovalTab) : "pending";
}

export function ApprovalsView({ data, onRetry, params, session }: AdminViewProps) {
  const { language, t } = useRedesignI18n();
  const live = data.status === "live";
  const approvals = useApprovals(live, session);
  const catalog = usePermissionCatalog(live);
  const [tab, setTab] = useState<ApprovalTab>(normalizeTab(params.status));
  const [selectedId, setSelectedId] = useState(params.id || "");
  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectReason, setRejectReason] = useState("");
  const [rejectError, setRejectError] = useState("");
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const entries = useMemo(() => approvalList(approvals.requests), [approvals.requests]);
  const counts = approvalTabCounts(entries);
  const filtered = useMemo(() => filterApprovalList(entries, tab), [entries, tab]);
  const selected = entries.find((entry) => entry.request.id === selectedId) ?? filtered[0] ?? entries[0] ?? null;
  const request = selected?.request ?? null;

  // Selection and tab live in local state; the hash is only synced (no
  // hashchange) so picking rows does not remount the page or refetch the list.
  useEffect(() => {
    const next = adminHash("approvals", {
      ...(selected?.request.id ? { id: selected.request.id } : {}),
      status: tab,
    });
    if (window.location.hash !== next) {
      window.history.replaceState(null, "", `${window.location.pathname}${window.location.search}${next}`);
    }
  }, [selected, tab]);

  const consoleData = data.data;
  const capabilityRows = request
    ? approvalCapabilityRows(request, catalog.templates.find((tpl) => tpl.id === request.templateId) ?? null, consoleData?.capabilities ?? [])
    : [];
  const riskSummary = approvalRiskSummary(capabilityRows);
  const selfReview = request ? isSelfReview(approvals.reviewer, request.requestedBy) : false;
  const pending = selected?.status === "pending";

  async function approve() {
    if (!request || acting) return;
    setActing(true);
    setActionError(null);
    const result = await approvals.act(request.id, "approve", "");
    setActing(false);
    if (!result.ok) {
      setActionError(actionErrorMessage(t, language, result.error, request.id));
    }
  }

  async function reject() {
    if (!request || acting) return;
    const reason = rejectReason.trim();
    if (!reason) {
      setRejectError(t("rd.apr.rejectReasonRequired"));
      return;
    }
    setActing(true);
    setActionError(null);
    const result = await approvals.act(request.id, "reject", reason);
    setActing(false);
    if (result.ok) {
      setRejectOpen(false);
      setRejectReason("");
      setRejectError("");
    } else {
      setRejectError(actionErrorMessage(t, language, result.error, request.id));
    }
  }

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t("rd.nav.approvals")}</h1>
          <p>{t("rd.page.approvals.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      {!live && !consoleData ? (
        <LoadingState label={t("rd.data.loading")} />
      ) : (
        <div className="apr-layout">
          <Card flush title={t("rd.apr.listTitle")}>
            <div className="tab-row" role="tablist" aria-label={t("rd.apr.listTitle")}>
              {tabs.map((key) => (
                <button
                  aria-selected={tab === key}
                  className={tab === key ? "tab-btn active" : "tab-btn"}
                  key={key}
                  onClick={() => setTab(key)}
                  role="tab"
                  type="button"
                >
                  {t(`rd.apr.tab.${key}`)}
                  <span className="tab-count">{counts[key]}</span>
                </button>
              ))}
            </div>
            <div className="apr-list">
              {!approvals.requests.length && approvals.loading ? (
                <LoadingState label={t("rd.common.loading")} />
              ) : approvals.failed ? (
                <EmptyState desc={t("rd.apr.failedDesc")} title={t("rd.apr.failed")} />
              ) : filtered.length === 0 ? (
                <EmptyState desc={t("rd.apr.emptyDesc")} title={t("rd.apr.empty")} />
              ) : (
                filtered.map((entry) => (
                  <button
                    className={entry.request.id === request?.id ? "apr-row apr-active" : "apr-row"}
                    key={entry.request.id}
                    onClick={() => setSelectedId(entry.request.id)}
                    type="button"
                  >
                    <span className="apr-main">
                      <span className="apr-name">{templateName(catalog.templates, entry.request.templateId)}</span>
                      <span className="apr-sub mono">
                        {maskId(entry.request.id)} · {entry.request.requestedBy || t("rd.common.none")} ·{" "}
                        {formatDate(entry.request.createdAt, language)}
                      </span>
                    </span>
                    <Chip tone={permissionApprovalStatusTone(entry.status)}>
                      {permissionApprovalStatusLabel(entry.status, t)}
                    </Chip>
                  </button>
                ))
              )}
            </div>
          </Card>

          <Card
            right={selected ? <Chip tone={permissionApprovalStatusTone(selected.status)}>{permissionApprovalStatusLabel(selected.status, t)}</Chip> : null}
            title={
              request
                ? tx(t, "rd.apr.detailTitle", { name: templateName(catalog.templates, request.templateId) })
                : t("rd.apr.detailTitleEmpty")
            }
          >
            {!request ? (
              <EmptyState desc={t("rd.apr.selectDesc")} title={t("rd.apr.detailTitleEmpty")} />
            ) : (
              <div className="stack">
                <KvList
                  items={[
                    {
                      key: "requester",
                      label: t("rd.apr.requester"),
                      value: request.requestedBy || t("rd.common.none"),
                    },
                    {
                      key: "subject",
                      label: t("rd.apr.subject"),
                      value: <span className="mono">{request.subjectSelector || t("rd.common.none")}</span>,
                    },
                    reviewerItem(t, approvals, session),
                    {
                      key: "scope",
                      label: t("rd.apr.scope"),
                      value: `${request.tenantId} / ${request.workspaceId}`,
                    },
                    {
                      key: "path",
                      label: t("rd.apr.path"),
                      value: tx(t, "rd.apr.pathValue", {
                        caller: agentName(consoleData?.agents ?? [], request.callerInstanceId),
                        target: agentName(consoleData?.agents ?? [], request.targetId),
                      }),
                    },
                    {
                      key: "template",
                      label: t("rd.apr.template"),
                      value: templateName(catalog.templates, request.templateId),
                    },
                    {
                      key: "requestedAt",
                      label: t("rd.apr.requestedAt"),
                      value: formatDate(request.createdAt, language),
                    },
                  ]}
                />

                <div className="cap-list">
                  {capabilityRows.map((row) => (
                    <div className="cap-row" key={row.key}>
                      <span className="cap-main">
                        <span className="cap-name">{row.capability?.displayName ?? row.key}</span>
                        <span className="cap-sub mono">{row.key}</span>
                      </span>
                      {row.isRequested ? <Chip tone="info">{t("rd.apr.requestedCapability")}</Chip> : null}
                      <Chip tone={row.allowed ? "success" : "danger"}>{row.allowed ? t("rd.apr.allow") : t("rd.apr.deny")}</Chip>
                    </div>
                  ))}
                </div>

                <Notice>{noticeText(t, approvalNoticeKey(riskSummary), riskSummary)}</Notice>

                {selected.status === "pending" ? (
                  <div className="apr-actions">
                    {selfReview ? (
                      <Notice tone="danger">{t("rd.apr.selfReview")}</Notice>
                    ) : null}
                    <div className="apr-button-row">
                      <Button disabled={acting || selfReview} icon={<CircleCheck aria-hidden="true" size={15} />} onClick={() => void approve()} variant="success">
                        {acting ? t("rd.common.loading") : t("rd.apr.approve")}
                      </Button>
                      <Button disabled={acting || selfReview} icon={<CircleX aria-hidden="true" size={15} />} onClick={() => { setRejectOpen(true); setRejectError(""); }} variant="danger-ghost">
                        {t("rd.apr.reject")}
                      </Button>
                    </div>
                  </div>
                ) : (
                  <KvList
                    items={[
                      { key: "reviewedBy", label: t("rd.apr.reviewedBy"), value: request.reviewedBy || t("rd.common.none") },
                      { key: "resolvedAt", label: t("rd.apr.resolvedAt"), value: request.resolvedAt ? formatDate(request.resolvedAt, language) : t("rd.common.none") },
                      { key: "comment", label: t("rd.apr.reviewComment"), value: request.reviewComment || t("rd.common.none") },
                    ]}
                  />
                )}

                {actionError ? <Banner desc={actionError} title={t("rd.error.other.title")} tone="danger" /> : null}
              </div>
            )}
          </Card>
        </div>
      )}

      <Modal
        footer={
          <>
            <Button onClick={() => setRejectOpen(false)} variant="ghost">{t("rd.apr.cancel")}</Button>
            <Button disabled={acting} onClick={() => void reject()} variant="danger-ghost">{t("rd.apr.rejectConfirm")}</Button>
          </>
        }
        onClose={() => setRejectOpen(false)}
        open={rejectOpen}
        size="narrow"
        title={t("rd.apr.rejectTitle")}
      >
        <Field
          error={rejectError || undefined}
          htmlFor="apr-reject-reason"
          label={t("rd.apr.rejectReasonLabel")}
        >
          <textarea
            aria-invalid={rejectError ? "true" : undefined}
            className="input"
            id="apr-reject-reason"
            onChange={(event) => setRejectReason(event.target.value)}
            rows={3}
            value={rejectReason}
          />
        </Field>
        <p className="hint">{t("rd.apr.rejectReasonHint")}</p>
      </Modal>
    </>
  );
}

function noticeText(
  t: ReturnType<typeof useRedesignI18n>["t"],
  key: ReturnType<typeof approvalNoticeKey>,
  summary: ReturnType<typeof approvalRiskSummary>,
): string {
  return tx(t, `rd.apr.notice.${key}`, { denied: summary.deniedCount, highRisk: summary.highRiskCount });
}

function reviewerItem(
  t: ReturnType<typeof useRedesignI18n>["t"],
  approvals: ReturnType<typeof useApprovals>,
  session: AdminViewProps["session"],
): KvItem {
  if (!approvals.reviewerSelectable) {
    return {
      key: "reviewer",
      label: t("rd.apr.reviewer"),
      value: approvals.reviewer || session?.actor || t("rd.common.none"),
    };
  }
  return {
    key: "reviewer",
    label: t("rd.apr.reviewer"),
    value: (
      <>
        <input
          className="input"
          list="apr-reviewer-options"
          onChange={(event) => approvals.setReviewer(event.target.value)}
          placeholder={t("rd.apr.reviewerPlaceholder")}
          value={approvals.reviewer}
        />
        <datalist id="apr-reviewer-options">
          {approvals.reviewerOptions.map((option) => (
            <option key={option} value={option} />
          ))}
        </datalist>
        <span className="hint">{t("rd.apr.reviewerHint")}</span>
      </>
    ),
  };
}

function actionErrorMessage(
  t: ReturnType<typeof useRedesignI18n>["t"],
  language: ReturnType<typeof useRedesignI18n>["language"],
  error: unknown,
  requestId: string,
): string {
  const presentation = apiErrorPresentation(t, language, error, "error.consoleDataUnavailable");
  return `${presentation.detail || presentation.next} (${maskId(requestId)})`;
}

function maskId(id: string): string {
  return id.length > 14 ? `${id.slice(0, 12)}…${id.slice(-4)}` : id;
}
