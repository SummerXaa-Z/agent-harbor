import { useEffect, useMemo, useState } from "react";
import { RefreshCw } from "lucide-react";
import {
  createInstanceAssignment,
  createTenantEntitlement,
  createWorkspaceAssignment,
  refreshTargetCapabilities,
  updateCapability,
} from "../../../api";
import {
  accessSubjectOptionForId,
  accessSubjectOptionForSelector,
  accessSubjectOptions,
  customAccessSubjectOption,
} from "../../../accessSubjects";
import {
  capabilityGrantBlockerKey,
  normalizeCapabilityGrantForm,
  validateCapabilityGrantChain,
  type CapabilityGrantForm,
} from "../../../capabilityGrantChain";
import { tx } from "../../../localizedMessages";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { usePermissionCatalog } from "../../hooks/usePermissionCatalog";
import { apiErrorPresentation } from "../../model/apiErrorCategory";
import { capabilityDomainSegments, capabilityGrantRows } from "../../model/capabilityCatalog";
import { adminHash } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Banner } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip, TagOutline } from "../../ui/Chip";
import { Field } from "../../ui/Field";
import { Modal } from "../../ui/Modal";
import { EmptyState } from "../../ui/StateViews";
import { Table } from "../../ui/Table";
import { useToast } from "../../ui/Toast";
import type { Capability } from "../../../types";
import type { AdminViewProps } from "./adminViewProps";

type CapabilityColumnKey = "capability" | "type" | "risk" | "dataScope" | "status" | "actions";

const capabilityStatusTone: Record<Capability["discoveryStatus"], "success" | "warning" | "neutral"> = {
  approved: "success",
  deprecated: "neutral",
  pending_review: "warning",
  removed: "neutral"
};

const capabilityStatusKey: Record<Capability["discoveryStatus"], string> = {
  approved: "status.capabilityApproved",
  deprecated: "status.capabilityDeprecated",
  pending_review: "status.capabilityPendingReview",
  removed: "status.capabilityRemoved"
};

export function CapabilitiesView({ data, onRetry, params }: AdminViewProps) {
  const { language, t } = useRedesignI18n();
  const toast = useToast();
  const consoleData = data.data;
  const live = data.status === "live";
  const catalog = usePermissionCatalog(live);

  const mcpTargets = useMemo(
    () => (consoleData?.agents ?? []).filter((agent) => agent.channelType === "mcp" && agent.status !== "disabled"),
    [consoleData?.agents]
  );
  const [targetId, setTargetId] = useState(() => {
    const initial = params.target ?? "";
    return mcpTargets.some((agent) => agent.id === initial) ? initial : mcpTargets[0]?.id ?? "";
  });

  useEffect(() => {
    const next = adminHash("capabilities", targetId ? { target: targetId } : {});
    if (window.location.hash !== next) {
      window.history.replaceState(null, "", `${window.location.pathname}${window.location.search}${next}`);
    }
  }, [targetId]);

  const capabilities = consoleData?.capabilities ?? [];
  const targetCapabilities = useMemo(
    () => capabilities.filter((capability) => capability.targetId === targetId),
    [capabilities, targetId]
  );
  const segments = useMemo(() => capabilityDomainSegments(targetCapabilities), [targetCapabilities]);
  const grants = useMemo(
    () =>
      capabilityGrantRows({
        entitlements: (consoleData?.tenantEntitlements ?? []).filter((item) => item.targetId === targetId),
        instanceAssignments: consoleData?.instanceAssignments ?? [],
        workspaceAssignments: consoleData?.workspaceAssignments ?? []
      }),
    [consoleData, targetId]
  );

  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState("");
  const [refreshOpen, setRefreshOpen] = useState(false);
  const [detail, setDetail] = useState<Capability | null>(null);
  const [detailRisk, setDetailRisk] = useState<Capability["riskLevel"]>("low");
  const [detailDomain, setDetailDomain] = useState("");
  const [chainOpen, setChainOpen] = useState(false);
  const [form, setForm] = useState<CapabilityGrantForm>(() => ({
    callerInstanceId: "",
    capabilityId: "",
    subjectSelector: "user:support-*",
    targetId: "",
    tenantId: "",
    workspaceId: ""
  }));
  const [chainError, setChainError] = useState("");

  const agents = consoleData?.agents ?? [];
  const tenants = consoleData?.tenants ?? [];
  const selectedCapability = capabilities.find((capability) => capability.id === form.capabilityId) ?? null;
  const blockerKey = capabilityGrantBlockerKey(form, selectedCapability);

  function openChain() {
    if (!consoleData) return;
    setForm((current) => {
      const seeded = normalizeCapabilityGrantForm({ ...current, targetId }, consoleData);
      if (seeded.tenantId !== "" && seeded.workspaceId !== "") return seeded;
      // The shared normalizer never invents a scope; in this view the natural
      // default is the pre-filled caller's own tenant and workspace.
      const caller = consoleData.agents.find((agent) => agent.id === seeded.callerInstanceId);
      return {
        ...seeded,
        tenantId: seeded.tenantId || caller?.tenantId || "",
        workspaceId: seeded.workspaceId || caller?.workspaceId || ""
      };
    });
    setChainError("");
    setChainOpen(true);
  }

  async function runRefresh() {
    if (!targetId) return;
    setActing(true);
    setActionError("");
    try {
      const refreshed = await refreshTargetCapabilities(targetId);
      toast(tx(t, "rd.cap.refreshed", { count: refreshed.length }));
      setRefreshOpen(false);
      await onRetry();
    } catch (error) {
      const presentation = apiErrorPresentation(t, language, error, "rd.cap.refreshFailed");
      setActionError(presentation.detail || presentation.next);
      setRefreshOpen(false);
    } finally {
      setActing(false);
    }
  }

  async function approveCapability(capability: Capability) {
    setActing(true);
    setActionError("");
    try {
      await updateCapability(capability.id, { discoveryStatus: "approved" });
      toast(
        (capability.dataDomains?.length ?? 0) === 0
          ? tx(t, "rd.cap.approvedNeedsDomain", { name: capability.displayName })
          : tx(t, "rd.cap.approved", { name: capability.displayName })
      );
      setDetail(null);
      await onRetry();
    } catch (error) {
      const presentation = apiErrorPresentation(t, language, error, "rd.cap.failed");
      setActionError(presentation.detail || presentation.next);
    } finally {
      setActing(false);
    }
  }

  function openDetail(capability: Capability) {
    setDetail(capability);
    setDetailRisk(capability.riskLevel);
    setDetailDomain(
      capability.dataDomains?.[0] ?? capability.dataScopes?.find((scope) => scope.dataDomain)?.dataDomain ?? ""
    );
  }

  async function saveDetail() {
    if (!detail) return;
    setActing(true);
    setActionError("");
    try {
      await updateCapability(detail.id, {
        dataDomains: detailDomain.trim() ? [detailDomain.trim()] : undefined,
        riskLevel: detailRisk
      });
      toast(tx(t, "rd.cap.governanceSaved", { name: detail.displayName }));
      setDetail(null);
      await onRetry();
    } catch (error) {
      const presentation = apiErrorPresentation(t, language, error, "rd.cap.failed");
      setActionError(presentation.detail || presentation.next);
    } finally {
      setActing(false);
    }
  }

  async function submitChain() {
    const outcome = validateCapabilityGrantChain(form, capabilities);
    if ("messageKey" in outcome) {
      setChainError(t(outcome.messageKey));
      return;
    }
    const { capability, dataScopes, tenantId, workspaceId, callerInstanceId, subjectSelector } = outcome.plan;
    setActing(true);
    setChainError("");
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
        }
      );
      const workspaceAssignment = await createWorkspaceAssignment(
        {
          dataScopes,
          effect: "allow",
          status: "enabled",
          tenantEntitlementId: entitlement.id,
          workspaceId
        }
      );
      await createInstanceAssignment(
        {
          callerInstanceId,
          dataScopes,
          effect: "allow",
          status: "enabled",
          subjectSelector,
          workspaceAssignmentId: workspaceAssignment.id
        }
      );
      toast(t("rd.cap.chainCreated"));
      setChainOpen(false);
      await onRetry();
    } catch (error) {
      const presentation = apiErrorPresentation(t, language, error, "rd.cap.chainFailed");
      setChainError(presentation.detail || presentation.next);
    } finally {
      setActing(false);
    }
  }

  const domainOptions = useMemo(
    () => [...new Set([...catalog.templates.map((template) => template.defaultDataDomain), ...segments.map((segment) => segment.domain)].filter(Boolean))],
    [catalog.templates, segments]
  );

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t("rd.nav.capabilities")}</h1>
          <p>{t("rd.page.capabilities.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      <div className="stack">
        <Card>
          <div className="form-grid form-grid-4">
            <Field htmlFor="cap-target" label={t("rd.cap.target")}>
              <select
                className="select"
                id="cap-target"
                onChange={(event) => setTargetId(event.target.value)}
                value={targetId}
              >
                {mcpTargets.map((agent) => (
                  <option key={agent.id} value={agent.id}>{agent.name}</option>
                ))}
              </select>
            </Field>
            <div className="audit-count">
              <Chip tone="neutral">{tx(t, "rd.cap.count", { count: targetCapabilities.length })}</Chip>
            </div>
            <div className="form-actions">
              <Button
                disabled={!targetId || acting}
                onClick={() => setRefreshOpen(true)}
                variant="ghost"
              >
                <RefreshCw aria-hidden="true" size={14} />
                {t("rd.cap.refresh")}
              </Button>
            </div>
            <div className="form-actions">
              <Button disabled={!targetId} onClick={openChain} variant="primary">
                {t("rd.cap.createChain")}
              </Button>
            </div>
          </div>
        </Card>

        {actionError ? <Banner desc={actionError} title={t("rd.error.other.title")} tone="danger" /> : null}

        {mcpTargets.length === 0 ? (
          <Card>
            <EmptyState desc={t("rd.cap.noTargetDesc")} title={t("rd.cap.noTarget")} />
          </Card>
        ) : (
          <div className="cockpit-grid">
            <Card
              flush
              title={t("rd.cap.tableTitle")}
            >
              <Table
                caption={t("rd.cap.tableTitle")}
                empty={<EmptyState desc={t("rd.cap.emptyDesc")} title={t("rd.cap.empty")} />}
                renderCell={(row, column) => renderCapabilityCell(row, column, t, openDetail, approveCapability, acting)}
                rowKey={(row) => row.id}
                rows={targetCapabilities}
                tableId="adminCapabilities"
              />
            </Card>

            <div className="stack">
              <Card title={t("rd.cap.domainsTitle")}>
                <DomainDonut segments={segments} t={t} />
              </Card>

              <Card
                title={t("rd.cap.grantsTitle")}
              >
                {grants.length === 0 ? (
                  <EmptyState desc={t("rd.cap.grantsEmptyDesc")} title={t("rd.cap.grantsEmpty")} />
                ) : (
                  <div className="cap-list">
                    {grants.map((grant) => (
                      <div className="cap-row" key={`${grant.tenantId}:${grant.capabilityId}`}>
                        <div className="cap-main">
                          <div className="cap-name">
                            {capabilities.find((capability) => capability.id === grant.capabilityId)?.displayName ?? grant.capabilityId}
                          </div>
                          <div className="cap-sub mono">
                            {grant.tenantId} · {tx(t, "rd.cap.grantWorkspaces", { count: grant.workspaceCount })} · {tx(t, "rd.cap.grantCallers", { count: grant.callerCount })}
                          </div>
                          {grant.subjectSelectors.length > 0 ? (
                            <div className="cap-sub mono">{grant.subjectSelectors.join(" · ")}</div>
                          ) : null}
                        </div>
                        <Chip tone={grant.effect === "deny" ? "danger" : "success"}>
                          {grant.effect === "deny" ? t("rd.decision.blocked") : t("rd.decision.allowed")}
                        </Chip>
                      </div>
                    ))}
                  </div>
                )}
              </Card>
            </div>
          </div>
        )}
      </div>

      <Modal
        footer={
          <>
            <Button onClick={() => setRefreshOpen(false)} variant="ghost">{t("rd.apr.cancel")}</Button>
            <Button disabled={acting} onClick={() => void runRefresh()} variant="primary">
              {t("rd.cap.refreshAction")}
            </Button>
          </>
        }
        onClose={() => setRefreshOpen(false)}
        open={refreshOpen}
        size="narrow"
        title={t("rd.cap.refreshTitle")}
      >
        <p className="muted">{t("rd.cap.refreshConfirm")}</p>
      </Modal>

      <Modal
        footer={
          <>
            <Button onClick={() => setDetail(null)} variant="ghost">{t("rd.common.close")}</Button>
            <Button disabled={acting} onClick={() => void saveDetail()} variant="primary">{t("rd.cap.save")}</Button>
          </>
        }
        onClose={() => setDetail(null)}
        open={detail !== null}
        title={detail ? detail.displayName : ""}
      >
        {detail ? (
          <div className="stack">
            <div className="mono small muted">{detail.key}</div>
            {detail.description ? <p className="muted small">{detail.description}</p> : null}
            {detail.discoveryStatus === "pending_review" ? (
              <Button disabled={acting} onClick={() => void approveCapability(detail)} variant="success">
                {t("rd.cap.approve")}
              </Button>
            ) : null}
            <div className="form-grid">
              <Field htmlFor="cap-risk" label={t("rd.col.risk")}>
                <select
                  className="select"
                  id="cap-risk"
                  onChange={(event) => setDetailRisk(event.target.value as Capability["riskLevel"])}
                  value={detailRisk}
                >
                  {(["low", "medium", "high", "critical"] as const).map((level) => (
                    <option key={level} value={level}>{t(`rd.risk.${level}`)}</option>
                  ))}
                </select>
              </Field>
              <Field htmlFor="cap-domain" label={t("rd.cap.domain")}>
                <select
                  className="select"
                  id="cap-domain"
                  onChange={(event) => setDetailDomain(event.target.value)}
                  value={detailDomain}
                >
                  <option value="">{t("rd.cap.domainUnset")}</option>
                  {domainOptions.map((domain) => (
                    <option key={domain} value={domain}>{domain}</option>
                  ))}
                </select>
              </Field>
            </div>
          </div>
        ) : null}
      </Modal>

      <Modal
        footer={
          <>
            <Button onClick={() => setChainOpen(false)} variant="ghost">{t("rd.apr.cancel")}</Button>
            <Button disabled={acting || blockerKey !== null} onClick={() => void submitChain()} variant="primary">
              {t("rd.cap.createChain")}
            </Button>
          </>
        }
        onClose={() => setChainOpen(false)}
        open={chainOpen}
        title={t("rd.cap.chainTitle")}
      >
        <div className="stack">
          <p className="hint">{t("rd.cap.chainHelp")}</p>
          <div className="form-grid">
            <Field htmlFor="chain-tenant" label={t("rd.cap.field.tenant")}>
              <select
                className="select"
                id="chain-tenant"
                onChange={(event) => setForm({ ...form, tenantId: event.target.value })}
                value={form.tenantId}
              >
                <option value="">—</option>
                {tenants.map((tenant) => (
                  <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
                ))}
                {form.tenantId && !tenants.some((tenant) => tenant.id === form.tenantId) ? (
                  <option value={form.tenantId}>{form.tenantId}</option>
                ) : null}
              </select>
            </Field>
            <Field htmlFor="chain-workspace" label={t("rd.cap.field.workspace")}>
              <select
                className="select"
                id="chain-workspace"
                onChange={(event) => setForm({ ...form, workspaceId: event.target.value })}
                value={form.workspaceId}
              >
                <option value="">—</option>
                {[...new Set(agents.map((agent) => agent.workspaceId).filter(Boolean))].map((workspaceId) => (
                  <option key={workspaceId} value={workspaceId}>{workspaceId}</option>
                ))}
                {form.workspaceId && !agents.some((agent) => agent.workspaceId === form.workspaceId) ? (
                  <option value={form.workspaceId}>{form.workspaceId}</option>
                ) : null}
              </select>
            </Field>
          </div>
          <Field htmlFor="chain-capability" label={t("rd.cap.field.capability")}>
            <select
              className="select"
              id="chain-capability"
              onChange={(event) => setForm({ ...form, capabilityId: event.target.value })}
              value={form.capabilityId}
            >
              <option value="">—</option>
              {targetCapabilities.map((capability) => (
                <option key={capability.id} value={capability.id}>
                  {capability.displayName} ({t(`rd.risk.${capability.riskLevel}`)})
                </option>
              ))}
            </select>
          </Field>
          <Field htmlFor="chain-caller" label={t("rd.cap.field.caller")}>
            <select
              className="select"
              id="chain-caller"
              onChange={(event) => setForm({ ...form, callerInstanceId: event.target.value })}
              value={form.callerInstanceId}
            >
              <option value="">—</option>
              {agents
                .filter((agent) => agent.status === "active" && agent.channelType === "local")
                .map((agent) => (
                  <option key={agent.id} value={agent.id}>{agent.name}</option>
                ))}
            </select>
          </Field>
          <SubjectPicker form={form} setForm={setForm} t={t} />
          {blockerKey ? <p className="field-error" role="alert">{t(blockerKey)}</p> : null}
          {chainError ? <Banner desc={chainError} title={t("rd.error.other.title")} tone="danger" /> : null}
        </div>
      </Modal>
    </>
  );
}

function SubjectPicker({
  form,
  setForm,
  t
}: {
  form: CapabilityGrantForm;
  setForm: (next: CapabilityGrantForm) => void;
  t: ReturnType<typeof useRedesignI18n>["t"];
}) {
  const selected = accessSubjectOptionForSelector(form.subjectSelector);
  const options = [...accessSubjectOptions, customAccessSubjectOption];
  return (
    <>
      <Field htmlFor="chain-subject" label={t("rd.cap.field.subject")}>
        <select
          className="select"
          id="chain-subject"
          onChange={(event) => {
            const option = accessSubjectOptionForId(event.target.value);
            if (option) setForm({ ...form, subjectSelector: option.subjectSelector });
          }}
          value={selected.id}
        >
          {options.map((option) => (
            <option key={option.id} value={option.id}>
              {t(`accessSubject.kind.${option.kind}`)} · {t(option.labelKey)}
            </option>
          ))}
        </select>
      </Field>
      {selected.id === customAccessSubjectOption.id ? (
        <Field
          hint={t("rd.cap.field.subjectCustomHint")}
          htmlFor="chain-subject-raw"
          label={t("rd.cap.field.subjectCustom")}
        >
          <input
            className="input mono"
            id="chain-subject-raw"
            onChange={(event) => setForm({ ...form, subjectSelector: event.target.value })}
            value={form.subjectSelector}
          />
        </Field>
      ) : null}
    </>
  );
}

function DomainDonut({
  segments,
  t
}: {
  segments: ReturnType<typeof capabilityDomainSegments>;
  t: ReturnType<typeof useRedesignI18n>["t"];
}) {
  const total = segments.reduce((sum, segment) => sum + segment.count, 0);
  const radius = 45;
  const center = 60;
  let offset = 0;

  return (
    <div className="donut-box">
      <svg aria-hidden="true" className="donut-svg" role="img" viewBox="0 0 120 120">
        <circle className="donut-track" cx={center} cy={center} r={radius} />
        {segments.map((segment, index) => {
          const element = (
            <circle
              className={index % 2 === 0 ? "donut-seg-1" : "donut-seg-2"}
              cx={center}
              cy={center}
              key={segment.domain || "unclassified"}
              pathLength={1}
              r={radius}
              strokeDasharray={`${segment.fraction} ${1 - segment.fraction}`}
              strokeDashoffset={-offset}
              transform={`rotate(-90 ${center} ${center})`}
            />
          );
          offset += segment.fraction;
          return element;
        })}
        <text className="donut-center-num" dominantBaseline="central" textAnchor="middle" x={center} y={center - 15}>
          {total}
        </text>
        <text className="donut-center-label" dominantBaseline="central" textAnchor="middle" x={center} y={center + 3}>
          {t("rd.cap.donutCapabilities")}
        </text>
      </svg>
      <div className="donut-legend">
        {segments.map((segment, index) => (
          <div className="chart-legend" key={segment.domain || "unclassified"}>
            <span>
              <span className={index % 2 === 0 ? "leg-dot leg-calls" : "leg-dot leg-audit"} />
              <span className="small">{segment.domain || t("rd.cap.domainUnset")}</span>
            </span>
            <span className="small muted">{segment.count}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function renderCapabilityCell(
  row: Capability,
  column: CapabilityColumnKey,
  t: ReturnType<typeof useRedesignI18n>["t"],
  onDetail: (capability: Capability) => void,
  onApprove: (capability: Capability) => void,
  acting: boolean
) {
  switch (column) {
    case "capability":
      return (
        <div>
          <div className="cell-main">{row.displayName}</div>
          <div className="cell-sub mono">{row.key}</div>
        </div>
      );
    case "type":
      return <TagOutline>{t(`value.${row.action}`)}</TagOutline>;
    case "risk":
      return (
        <Chip tone={row.riskLevel === "critical" || row.riskLevel === "high" ? "danger" : row.riskLevel === "medium" ? "warning" : "neutral"}>
          {t(`rd.risk.${row.riskLevel}`)}
        </Chip>
      );
    case "dataScope":
      return (
        <span className="small">
          {row.dataDomains?.[0] ?? row.dataScopes?.find((scope) => scope.dataDomain)?.dataDomain ?? t("rd.cap.domainUnset")}
        </span>
      );
    case "status":
      return <Chip tone={capabilityStatusTone[row.discoveryStatus]}>{t(capabilityStatusKey[row.discoveryStatus])}</Chip>;
    case "actions":
      return (
        <div className="apr-actions">
          {row.discoveryStatus === "pending_review" ? (
            <Button disabled={acting} onClick={() => onApprove(row)} size="sm" variant="link">
              {t("rd.cap.approve")}
            </Button>
          ) : null}
          <Button onClick={() => onDetail(row)} size="sm" variant="link">
            {t("rd.cap.detail")}
          </Button>
        </div>
      );
  }
}
