import { ArrowRight, Play, Search } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import {
  accessDecisionReasonLabel,
  accessDecisionRecordMessageLabel,
  askAccessScopeOptions,
} from "../../../askJourney";
import { useAskAccessController } from "../../../hooks/useAskAccessController";
import { tx } from "../../../localizedMessages";
import { subjectIdExampleFromSelector } from "../../../permissionPackages";
import type { AskHandoffContext, DataScope, PermissionChangeHandoffContext } from "../../../types";
import { useAccessContext } from "../../hooks/useAccessContext";
import { usePermissionCatalog } from "../../hooks/usePermissionCatalog";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { accessContextRouteParams, normalizeAccessContext, type AccessContext } from "../../model/accessContext";
import { buildDecisionChain, decisionRemediation } from "../../model/decisionChain";
import { userHash, viewLabelKey } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Banner, Notice } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip } from "../../ui/Chip";
import { DecisionChain } from "../../ui/DecisionChain";
import { Field } from "../../ui/Field";
import { KvList } from "../../ui/KvList";
import { LoadingState } from "../../ui/StateViews";
import { AccessContextSwitcher, agentName } from "./AccessContextSwitcher";
import type { UserViewProps } from "./userViewProps";

function askHandoffFromContext(context: AccessContext): AskHandoffContext | null {
  if (!context.callerInstanceId && !context.targetId) return null;
  return {
    callerInstanceId: context.callerInstanceId || undefined,
    capabilityId: context.requestedCapabilityId || undefined,
    sourceView: "access",
    subjectId: subjectIdExampleFromSelector(context.subjectSelector),
    targetId: context.targetId || undefined,
    tenantId: context.tenantId || undefined,
    workspaceId: context.workspaceId || undefined,
  };
}

export function dataScopeLabel(scope: DataScope): string {
  return [scope.dataDomain, scope.dataset, scope.table, scope.region, scope.classification].filter(Boolean).join(" · ");
}

export function AskView({ data, onRetry, params }: UserViewProps) {
  const { language, t } = useRedesignI18n();
  const live = Boolean(data.data?.loadedFromApi);
  const consoleData = data.data;
  const accessContext = useAccessContext(data, params);
  const catalog = usePermissionCatalog(live);
  const [handoff, setHandoff] = useState<AskHandoffContext | null>(null);
  const [seeded, setSeeded] = useState(false);

  const controller = useAskAccessController({
    adminKey: "",
    consoleData,
    handoffContext: handoff,
    language,
    liveDataAvailable: live,
    onConsumeHandoff: () => setHandoff(null),
    onStartPermissionChange: startPermissionChange,
    t,
    templates: catalog.templates,
  });

  const hasRouteScope = Boolean(params.caller || params.target);
  useEffect(() => {
    if (seeded || !consoleData) return;
    if (!hasRouteScope && !accessContext.applicationsLoaded && live) return;
    setSeeded(true);
    const seed = askHandoffFromContext(accessContext.context);
    if (!seed) return;
    setHandoff({ ...seed, subjectId: params.subject || seed.subjectId });
  }, [accessContext.applicationsLoaded, accessContext.context, consoleData, hasRouteScope, live, params.subject, seeded]);

  function startPermissionChange(change: PermissionChangeHandoffContext) {
    const next = normalizeAccessContext({
      callerInstanceId: change.callerInstanceId,
      region: "",
      requestedCapabilityId: change.capabilityId,
      requestText: change.intentText,
      subjectSelector: change.subjectId,
      targetId: change.targetId,
      templateId: change.templateId,
      tenantId: change.tenantId,
      workspaceId: change.workspaceId,
    });
    accessContext.replace(next);
    window.location.hash = userHash("apply", accessContextRouteParams(next));
  }

  const selection = controller.effectiveSelection;
  const scopeOptions = useMemo(
    () => (consoleData ? askAccessScopeOptions(consoleData, selection) : null),
    [consoleData, selection],
  );
  const subjectMissing = !selection.subjectId?.trim();
  const result = controller.result;
  const chain = useMemo(() => (result ? buildDecisionChain(result) : null), [result]);
  const remediation = result ? decisionRemediation(result) : null;
  const agents = consoleData?.agents ?? [];
  const capabilityName = (id: string) => {
    const capability = consoleData?.capabilities.find((item) => item.id === id);
    return capability?.displayName || capability?.key || id;
  };

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t(viewLabelKey("ask"))}</h1>
          <p>{t("rd.page.ask.desc")}</p>
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
          <Card
            right={
              <AccessContextSwitcher
                accessContext={accessContext}
                agents={agents}
                onSelect={(context) => {
                  const seed = askHandoffFromContext(context);
                  if (seed) setHandoff(seed);
                }}
                templates={catalog.templates}
              />
            }
            title={t("rd.ask.queryTitle")}
          >
            <form
              onSubmit={(event) => {
                event.preventDefault();
                if (!subjectMissing) void controller.explain();
              }}
            >
              <div className="form-grid">
                <Field htmlFor="rd-ask-tenant" label={t("rd.ask.tenant")}>
                  <select
                    className="select"
                    id="rd-ask-tenant"
                    onChange={(event) => controller.updateSelection({ tenantId: event.target.value })}
                    value={selection.tenantId ?? ""}
                  >
                    {scopeOptions?.tenants.map((tenant) => (
                      <option key={tenant.id} value={tenant.id}>
                        {tenant.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field htmlFor="rd-ask-workspace" label={t("rd.ask.workspace")}>
                  <select
                    className="select"
                    id="rd-ask-workspace"
                    onChange={(event) => controller.updateSelection({ workspaceId: event.target.value })}
                    value={selection.workspaceId ?? ""}
                  >
                    {scopeOptions?.workspaceIds.map((workspaceId) => (
                      <option key={workspaceId} value={workspaceId}>
                        {workspaceId}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field htmlFor="rd-ask-caller" label={t("rd.ask.caller")}>
                  <select
                    className="select"
                    id="rd-ask-caller"
                    onChange={(event) => controller.updateSelection({ callerInstanceId: event.target.value })}
                    value={selection.callerInstanceId ?? ""}
                  >
                    {scopeOptions?.callers.map((agent) => (
                      <option key={agent.id} value={agent.id}>
                        {agent.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field htmlFor="rd-ask-target" label={t("rd.ask.target")}>
                  <select
                    className="select"
                    id="rd-ask-target"
                    onChange={(event) => controller.updateSelection({ targetId: event.target.value })}
                    value={selection.targetId ?? ""}
                  >
                    {scopeOptions?.targets.map((agent) => (
                      <option key={agent.id} value={agent.id}>
                        {agent.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field htmlFor="rd-ask-capability" label={t("rd.ask.capability")}>
                  <select
                    className="select"
                    id="rd-ask-capability"
                    onChange={(event) => controller.updateSelection({ capabilityId: event.target.value })}
                    value={selection.capabilityId ?? ""}
                  >
                    {scopeOptions?.capabilities.map((capability) => (
                      <option key={capability.id} value={capability.id}>
                        {capability.displayName || capability.key}
                      </option>
                    ))}
                  </select>
                </Field>
              </div>
              <div className="form-row">
                <Field
                  error={subjectMissing ? t("rd.ask.subjectRequired") : undefined}
                  hint={t("rd.ask.subjectHint")}
                  htmlFor="rd-ask-subject"
                  label={t("rd.ask.subject")}
                >
                  <input
                    aria-invalid={subjectMissing}
                    className="input mono"
                    id="rd-ask-subject"
                    onChange={(event) => controller.updateSelection({ subjectId: event.target.value })}
                    placeholder="user:support-example"
                    value={selection.subjectId ?? ""}
                  />
                </Field>
                <div className="form-actions">
                  <Button disabled={subjectMissing || controller.loading} icon={<Search aria-hidden="true" size={15} />} type="submit">
                    {controller.loading ? t("rd.ask.querying") : t("rd.ask.query")}
                  </Button>
                  {controller.exampleAvailable ? (
                    <Button icon={<Play aria-hidden="true" size={15} />} onClick={() => void controller.runExampleQuery()} variant="ghost">
                      {t("rd.ask.example")}
                    </Button>
                  ) : null}
                </div>
              </div>
            </form>
            {controller.message && !result ? <Notice>{controller.message}</Notice> : null}
          </Card>

          {result && chain && remediation ? (
            <>
              {result.outcome === "allowed" ? (
                <Banner
                  actions={<Chip tone="success">{t("rd.ask.noChange")}</Chip>}
                  desc={accessDecisionReasonLabel(result.decision.reason, t)}
                  title={tx(t, "rd.ask.allowedTitle", {
                    caller: agentName(agents, result.request.callerInstanceId),
                    capability: capabilityName(result.request.capabilityId),
                  })}
                  tone="success"
                />
              ) : (
                <Banner
                  actions={
                    remediation.requestable && controller.permissionChangeAvailable ? (
                      <Button icon={<ArrowRight aria-hidden="true" size={15} />} onClick={controller.startPermissionChange}>
                        {t("rd.ask.goApply")}
                      </Button>
                    ) : undefined
                  }
                  desc={accessDecisionReasonLabel(result.decision.reason, t)}
                  title={tx(t, "rd.ask.deniedTitle", {
                    layer: chain.terminatingLayer ? t(`ask.recordLayer.${chain.terminatingLayer}`) : t("rd.ask.unknownLayer"),
                  })}
                  tone="danger"
                />
              )}
              <Card sub={tx(t, "rd.ask.chainCount", { passed: chain.passedCount, total: chain.nodes.length })} title={t("rd.ask.chainTitle")}>
                <div className="stack">
                  <DecisionChain chain={chain} label={t("rd.ask.chainTitle")} />
                  <KvList
                    items={chain.nodes
                      .filter((node) => node.evidence && node.state !== "pending")
                      .map((node) => ({
                        key: node.layer,
                        label: t(node.labelKey),
                        value: node.evidence ? accessDecisionRecordMessageLabel(node.evidence, t) : "",
                      }))}
                  />
                </div>
              </Card>
              <div className="grid grid-2">
                <Card title={t("rd.ask.remediationTitle")}>
                  <p>{remediation.labelKey ? t(remediation.labelKey, remediation.fallback) : remediation.fallback}</p>
                  <p className="hint mono">{remediation.code}</p>
                </Card>
                <Card title={t("rd.ask.dataScopeTitle")}>
                  {result.outcome === "allowed" && result.dataScopes && result.dataScopes.length > 0 ? (
                    <ul className="plain-list">
                      {result.dataScopes.map((scope, index) => (
                        <li className="mono small" key={index}>
                          {dataScopeLabel(scope) || t("rd.common.none")}
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="muted">{result.outcome === "allowed" ? t("rd.ask.dataScopeNone") : t("rd.ask.dataScopeDenied")}</p>
                  )}
                </Card>
              </div>
            </>
          ) : null}
        </div>
      )}
    </>
  );
}
