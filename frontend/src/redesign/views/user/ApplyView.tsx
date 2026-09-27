import { ArrowRight, Check, FileDown, RefreshCw, RotateCcw, Send, Undo2 } from "lucide-react";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { fetchPermissionPackageApprovalRequests } from "../../../api";
import { accessSubjectsForWorkspace } from "../../../accessSubjects";
import { askAccessScopeOptions } from "../../../askJourney";
import { tx } from "../../../localizedMessages";
import { permissionPackageApprovalEffectiveStatus } from "../../../permissionPackages";
import {
  permissionApprovalStatusLabel,
  permissionApprovalStatusTone,
  permissionPolicyGateMessages,
  permissionReadinessMessages,
} from "../../../permissionWorkbenchPresenters";
import { permissionProductionReadinessNextAction } from "../../../productionReadinessCopy";
import { useAccessContext } from "../../hooks/useAccessContext";
import { usePermissionCatalog } from "../../hooks/usePermissionCatalog";
import { usePermissionChangeFlow, type PermissionChangeFlow } from "../../hooks/usePermissionChangeFlow";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { accessContextFromApprovalRequest, accessContextRouteParams, type AccessContext } from "../../model/accessContext";
import { permissionChangeStepKeys, type PermissionChangeAction } from "../../model/approvalStateMachine";
import { rankTemplates } from "../../model/templateMatch";
import { workbenchActor } from "../../model/userWorkbench";
import { userHash, viewLabelKey } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Banner, Notice } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip, TagOutline } from "../../ui/Chip";
import { Field } from "../../ui/Field";
import { KvList } from "../../ui/KvList";
import { EmptyState, LoadingState } from "../../ui/StateViews";
import { Stepper } from "../../ui/Stepper";
import { AccessContextSwitcher, templateName } from "./AccessContextSwitcher";
import { dataScopeLabel } from "./AskView";
import type { UserViewProps } from "./userViewProps";

const templateLevelTone = { none: "neutral", partial: "warning", recommended: "success" } as const;
const directApplyCodes = new Set(["apply_permission_package", "reapply_permission_package"]);

export function ApplyView({ data, onRetry, params, session }: UserViewProps) {
  const { t } = useRedesignI18n();
  const live = Boolean(data.data?.loadedFromApi);
  const consoleData = data.data;
  const accessContext = useAccessContext(data, params);
  const catalog = usePermissionCatalog(live);
  const context = accessContext.context;
  const flow = usePermissionChangeFlow({
    context,
    live,
    onApplied: () => {
      accessContext.reloadApplications();
      void data.reload();
    },
  });
  const { presentation, preview } = flow;
  const { bindApproval } = flow;
  const replaceContext = accessContext.replace;
  const [boundRequestId, setBoundRequestId] = useState("");

  useEffect(() => {
    const requestId = params.approval ?? "";
    if (!live || !requestId || boundRequestId === requestId) return;
    const controller = new AbortController();
    fetchPermissionPackageApprovalRequests(
      { callerInstanceId: params.caller, limit: 50, targetId: params.target, templateId: params.template },
      "",
      controller.signal,
    )
      .then((rows) => {
        const request = rows.find((row) => row.id === requestId);
        setBoundRequestId(requestId);
        if (!request) return;
        replaceContext(accessContextFromApprovalRequest(request));
        bindApproval(request);
      })
      .catch(() => undefined);
    return () => controller.abort();
  }, [bindApproval, boundRequestId, live, params.approval, params.caller, params.target, params.template, replaceContext]);
  const state = presentation.state;
  const locked = state === "submitted";
  const agents = useMemo(() => consoleData?.agents ?? [], [consoleData]);

  const callers = agents.filter((agent) => agent.channelType === "local" && agent.status === "active");
  const targets = consoleData
    ? askAccessScopeOptions(consoleData, { tenantId: context.tenantId, workspaceId: context.workspaceId }).targets
    : [];
  const targetCapabilities = useMemo(
    () => (consoleData?.capabilities ?? []).filter((capability) => capability.targetId === context.targetId),
    [consoleData, context.targetId],
  );
  const rankedTemplates = useMemo(
    () => rankTemplates(catalog.templates, targetCapabilities, context.requestedCapabilityId),
    [catalog.templates, context.requestedCapabilityId, targetCapabilities],
  );
  const subjects = accessSubjectsForWorkspace(catalog.subjects, context.workspaceId);
  const approval = flow.approval;
  const approvalStatus = approval ? permissionPackageApprovalEffectiveStatus(approval) : null;
  const scopeParams = accessContextRouteParams(context);
  const golivePath = userHash("golive", approval ? { ...scopeParams, approval: approval.id } : scopeParams);
  const readinessNext = preview?.productionReadiness?.nextActions?.[0] ?? "";
  const nextActionCode = preview?.productionReadiness?.nextActionCode ?? preview?.summary.nextActionCode ?? "";

  function update(patch: Partial<AccessContext>) {
    accessContext.update(patch);
  }

  function selectCaller(callerId: string) {
    const caller = agents.find((agent) => agent.id === callerId);
    update({ callerInstanceId: callerId, tenantId: caller?.tenantId ?? "", workspaceId: caller?.workspaceId ?? "" });
  }

  function applyTemplate(templateId: string) {
    update({
      subjectSelector: context.subjectSelector || subjects[0]?.subjectSelector || "",
      templateId,
    });
  }

  function scrollToTemplates() {
    document.getElementById("rd-apply-templates")?.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function actionButton(action: PermissionChangeAction, primary: boolean): ReactNode {
    const variant = primary ? (state === "approved" ? "success" : "primary") : "ghost";
    const disabled = flow.busy !== "" || (primary && presentation.primaryDisabled);
    switch (action) {
      case "submit":
        return (
          <Button disabled={disabled} icon={<Send aria-hidden="true" size={15} />} key={action} onClick={() => void flow.submit()} variant={variant}>
            {flow.busy === "submit" ? t("rd.pc.action.submitting") : t("rd.pc.action.submit")}
          </Button>
        );
      case "submitted":
        return (
          <Button disabled icon={<Check aria-hidden="true" size={15} />} key={action} onClick={flow.refresh} variant={variant}>
            {t("rd.pc.action.submitted")}
          </Button>
        );
      case "resubmit":
        return (
          <Button disabled={disabled} icon={<Send aria-hidden="true" size={15} />} key={action} onClick={() => void flow.submit()} variant={variant}>
            {t("rd.pc.action.resubmit")}
          </Button>
        );
      case "apply":
        return (
          <Button disabled={disabled} icon={<Check aria-hidden="true" size={15} />} key={action} onClick={() => void flow.apply()} variant={variant}>
            {flow.busy === "apply" ? t("rd.pc.action.applying") : t("rd.pc.action.apply")}
          </Button>
        );
      case "withdraw":
        return (
          <Button disabled={flow.busy !== ""} icon={<Undo2 aria-hidden="true" size={15} />} key={action} onClick={() => void flow.withdraw()} variant="danger-ghost">
            {t("rd.pc.action.withdraw")}
          </Button>
        );
      case "reedit":
        return (
          <Button icon={<RotateCcw aria-hidden="true" size={15} />} key={action} onClick={flow.reedit} variant={variant}>
            {t("rd.pc.action.reedit")}
          </Button>
        );
      case "changeTemplate":
        return (
          <Button key={action} onClick={scrollToTemplates} variant="ghost">
            {t("rd.pc.action.changeTemplate")}
          </Button>
        );
      case "goCheck":
        return (
          <Button href={golivePath} icon={<ArrowRight aria-hidden="true" size={15} />} key={action} variant={variant}>
            {t("rd.pc.action.goCheck")}
          </Button>
        );
      case "exportReport":
        return (
          <Button disabled={flow.busy !== ""} icon={<FileDown aria-hidden="true" size={15} />} key={action} onClick={() => void flow.exportReport()} variant={variant}>
            {t("action.exportAcceptanceReport")}
          </Button>
        );
      case "recheck":
        return (
          <Button icon={<RefreshCw aria-hidden="true" size={15} />} key={action} onClick={flow.refresh} variant="ghost">
            {t("rd.pc.action.recheck")}
          </Button>
        );
      case "viewRequests":
        return (
          <Button href={userHash("home")} key={action} variant="ghost">
            {t("rd.pc.action.viewRequests")}
          </Button>
        );
      case "nextAction":
        if (directApplyCodes.has(nextActionCode)) {
          return (
            <Button disabled={flow.busy !== ""} key={action} onClick={() => void flow.apply()} variant={variant}>
              {permissionProductionReadinessNextAction(nextActionCode, readinessNext, t)}
            </Button>
          );
        }
        if (state === "blocked" && !preview?.summary.applied) {
          return (
            <Button icon={<RotateCcw aria-hidden="true" size={15} />} key={action} onClick={flow.reedit} variant={variant}>
              {t("rd.pc.action.reedit")}
            </Button>
          );
        }
        return (
          <Button href={golivePath} key={action} variant={variant}>
            {nextActionCode ? permissionProductionReadinessNextAction(nextActionCode, readinessNext, t) : t("rd.pc.action.goCheck")}
          </Button>
        );
      case "copyConfig":
      case "createToken":
        return (
          <Button href={golivePath} key={action} variant={variant}>
            {t(`rd.pc.action.${action}`)}
          </Button>
        );
    }
  }

  const bannerDesc = (() => {
    if (state === "rejected" && approval?.reviewComment) {
      return tx(t, "rd.apply.rejectedReason", { reason: approval.reviewComment });
    }
    if (state === "needsInput") {
      const missing = preview ? permissionReadinessMessages(preview.draft.readiness, t) : [];
      if (!context.requestText.trim()) missing.push(t("message.permissionApprovalRequestTextRequired"));
      return missing.length > 0 ? missing.join(" ") : t(presentation.descKey);
    }
    return t(presentation.descKey);
  })();

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t(viewLabelKey("apply"))}</h1>
          <p>{t("rd.page.apply.desc")}</p>
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
          <Banner
            actions={
              <>
                {presentation.primary ? actionButton(presentation.primary, true) : null}
                {presentation.secondary.map((action) => actionButton(action, false))}
              </>
            }
            desc={bannerDesc}
            title={t(presentation.titleKey)}
            tone={presentation.tone}
          />
          <Card>
            <Stepper
              label={t("rd.apply.stepsLabel")}
              steps={permissionChangeStepKeys.map((key) => ({
                key,
                label: key === "approval" && presentation.steps.approval === "skipped" ? t("rd.apply.step.noApproval") : t(`rd.apply.step.${key}`),
                state: presentation.steps[key],
              }))}
            />
          </Card>

          <Card
            right={<AccessContextSwitcher accessContext={accessContext} agents={agents} templates={catalog.templates} />}
            sub={context.templateId ? tx(t, "rd.apply.basedOn", { template: templateName(catalog.templates, context.templateId) }) : undefined}
            title={t("rd.apply.scopeTitle")}
          >
            <fieldset className="plain-fieldset" disabled={locked}>
              <div className="form-grid form-grid-4">
                <Field htmlFor="rd-apply-caller" label={t("rd.ask.caller")}>
                  <select className="select" id="rd-apply-caller" onChange={(event) => selectCaller(event.target.value)} value={context.callerInstanceId}>
                    <option value="">{t("rd.common.choose")}</option>
                    {callers.map((agent) => (
                      <option key={agent.id} value={agent.id}>
                        {agent.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field htmlFor="rd-apply-target" label={t("rd.ask.target")}>
                  <select
                    className="select"
                    id="rd-apply-target"
                    onChange={(event) => update({ requestedCapabilityId: "", targetId: event.target.value })}
                    value={context.targetId}
                  >
                    <option value="">{t("rd.common.choose")}</option>
                    {targets.map((agent) => (
                      <option key={agent.id} value={agent.id}>
                        {agent.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field hint={t("rd.apply.capabilityHint")} htmlFor="rd-apply-capability" label={t("rd.apply.requestedCapability")}>
                  <select
                    className="select"
                    id="rd-apply-capability"
                    onChange={(event) => update({ requestedCapabilityId: event.target.value })}
                    value={context.requestedCapabilityId}
                  >
                    <option value="">{t("rd.apply.anyCapability")}</option>
                    {targetCapabilities.map((capability) => (
                      <option key={capability.id} value={capability.id}>
                        {capability.displayName || capability.key}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field hint={t("rd.apply.subjectHint")} htmlFor="rd-apply-subject" label={t("rd.apply.subject")}>
                  <input
                    className="input mono"
                    id="rd-apply-subject"
                    list="rd-apply-subjects"
                    onChange={(event) => update({ subjectSelector: event.target.value })}
                    placeholder="user:support-*"
                    value={context.subjectSelector}
                  />
                  <datalist id="rd-apply-subjects">
                    {subjects.map((subject) => (
                      <option key={subject.id} label={t(subject.labelKey, subject.subjectSelector)} value={subject.subjectSelector} />
                    ))}
                  </datalist>
                </Field>
              </div>
              <div className="form-grid form-grid-4">
                <Field htmlFor="rd-apply-region" label={t("rd.apply.region")}>
                  <input className="input" id="rd-apply-region" onChange={(event) => update({ region: event.target.value })} value={context.region} />
                </Field>
                <div className="span-3">
                  <Field
                    error={!context.requestText.trim() ? t("message.permissionApprovalRequestTextRequired") : undefined}
                    htmlFor="rd-apply-request"
                    label={t("rd.apply.requestText")}
                  >
                    <textarea
                      aria-invalid={!context.requestText.trim()}
                      className="input"
                      id="rd-apply-request"
                      onChange={(event) => update({ requestText: event.target.value })}
                      value={context.requestText}
                    />
                  </Field>
                </div>
              </div>
            </fieldset>
            {locked ? <Notice>{t("rd.apply.lockedHint")}</Notice> : null}
          </Card>

          <section className="stack" id="rd-apply-templates">
            <h2 className="section-title">{t("rd.apply.templatesTitle")}</h2>
            {context.targetId ? (
              <div className="tpl-grid">
                {rankedTemplates.map((match) => {
                  const selected = match.template.id === context.templateId;
                  return (
                    <div className={selected ? "card tpl-card tpl-card-active" : "card tpl-card"} key={match.template.id}>
                      <div className="tpl-head">
                        <span className="tpl-name">{match.template.name}</span>
                        <Chip tone={templateLevelTone[match.level]}>{t(`rd.apply.match.${match.level}`)}</Chip>
                      </div>
                      <p className="tpl-desc">{match.template.summary}</p>
                      <div className="tpl-meta">
                        <span>{tx(t, "rd.apply.matchCounts", { allowed: match.allowedCount, blocked: match.blockedCount })}</span>
                        {selected ? (
                          <Chip tone="info">{t("rd.apply.templateInUse")}</Chip>
                        ) : (
                          <Button disabled={locked} onClick={() => applyTemplate(match.template.id)} size="sm" variant="ghost">
                            {t("rd.apply.useTemplate")}
                          </Button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <Card>
                <EmptyState desc={t("rd.apply.pickTargetFirst")} title={t("rd.apply.templatesTitle")} />
              </Card>
            )}
          </section>

          <div className="grid grid-2">
            <Card sub={preview ? tx(t, "rd.apply.plannedObjects", { count: preview.summary.plannedObjectCount }) : undefined} title={t("rd.apply.summaryTitle")}>
              {flow.previewLoading && !preview ? (
                <LoadingState label={t("rd.apply.previewLoading")} />
              ) : preview ? (
                <div className="stack">
                  <div>
                    <div className="field-label">{tx(t, "rd.apply.allowedCaps", { count: preview.draft.allowedCapabilities.length })}</div>
                    <div className="chip-row">
                      {preview.draft.allowedCapabilities.length === 0 ? <span className="muted small">{t("rd.common.none")}</span> : null}
                      {preview.draft.allowedCapabilities.map((capability) => (
                        <Chip key={capability.id} tone="success">
                          {capability.displayName || capability.key}
                        </Chip>
                      ))}
                    </div>
                  </div>
                  <div>
                    <div className="field-label">{tx(t, "rd.apply.blockedCaps", { count: preview.draft.blockedCapabilities.length })}</div>
                    <div className="chip-row">
                      {preview.draft.blockedCapabilities.length === 0 ? <span className="muted small">{t("rd.common.none")}</span> : null}
                      {preview.draft.blockedCapabilities.map((capability) => (
                        <TagOutline key={capability.id}>{capability.displayName || capability.key}</TagOutline>
                      ))}
                    </div>
                  </div>
                  <KvList
                    items={[
                      {
                        key: "scope",
                        label: t("rd.col.dataScope"),
                        value: preview.draft.dataScopes.map(dataScopeLabel).filter(Boolean).join("; ") || t("rd.common.none"),
                      },
                      {
                        key: "readiness",
                        label: t("text.readinessChecks"),
                        value: `${preview.summary.readinessReadyCount}/${preview.summary.readinessTotalCount}`,
                      },
                    ]}
                  />
                </div>
              ) : (
                <EmptyState desc={flow.previewError ? t("rd.apply.previewFailed") : t("rd.apply.previewNeedsScope")} title={t("rd.apply.summaryTitle")} />
              )}
            </Card>
            <ApprovalNotes actor={workbenchActor(session)} flow={flow} status={approvalStatus} />
          </div>
        </div>
      )}
    </>
  );
}

function ApprovalNotes({
  actor,
  flow,
  status,
}: {
  actor: string;
  flow: PermissionChangeFlow;
  status: ReturnType<typeof permissionPackageApprovalEffectiveStatus> | null;
}) {
  const { t } = useRedesignI18n();
  const approval = flow.approval;
  const gate = flow.preview?.draft.policyGate;
  return (
    <Card title={t("rd.apply.approvalTitle")}>
      <div className="stack">
        <KvList
          items={[
            {
              key: "gate",
              label: t("rd.apply.policyGate"),
              value: gate ? (gate.canApplyDirectly ? t("rd.apply.gateDirect") : t("status.approvalRequired")) : t("rd.common.none"),
            },
            {
              key: "status",
              label: t("rd.col.status"),
              value: approval && status ? (
                <Chip tone={permissionApprovalStatusTone(status)}>{permissionApprovalStatusLabel(status, t)}</Chip>
              ) : (
                t("status.approvalNotRequested")
              ),
            },
            { key: "id", label: t("rd.col.approval"), value: approval ? <span className="mono">{approval.id}</span> : t("rd.common.none") },
            { key: "requester", label: t("rd.apply.requester"), value: approval?.requestedBy || actor || t("rd.common.none") },
            { key: "reviewer", label: t("rd.apply.reviewer"), value: approval?.reviewedBy || t("rd.apply.reviewerRole") },
          ]}
        />
        {gate ? (
          <ul className="plain-list small">
            {permissionPolicyGateMessages(gate, t).map((message) => (
              <li key={message}>{message}</li>
            ))}
          </ul>
        ) : null}
        <Notice>{t("rd.apply.separationOfDuties")}</Notice>
        {flow.presentation.polling ? <p className="hint">{t("rd.apply.polling")}</p> : null}
      </div>
    </Card>
  );
}
