import { AlertCircle, ArrowRight, FileSearch, History, Search, Wrench } from "lucide-react";

import { summarizeDataScopes } from "../accessProfile";
import type {
  AskAccessSelection,
  AskDecisionRecordRow,
  ExplainRequestBuildResult
} from "../askJourney";
import {
  accessDecisionPrimaryAction,
  accessDecisionSummaryLabel,
  accessDecisionRecordMessageLabel,
  accessNextActionLabel,
  askAccessScopeOptions
} from "../askJourney";
import {
  capabilityDisplayName,
  dataScopeValueLabels,
  permissionEntityDisplayName,
  readableIdentifierLabel,
  type Translator
} from "../consolePresenters";
import type { AskAccessController, AskAccessHistoryEntry } from "../hooks/useAskAccessController";
import type { PermissionPackageTemplate } from "../permissionPackages";
import type { AccessDecisionExplainResult, Agent, Capability, Tenant } from "../types";
import { ApprovalDropdown, type ApprovalDropdownOption } from "./ApprovalDropdown";
import { Panel } from "./ConsolePrimitives";
import { Badge } from "./ui";

interface AskAccessViewProps {
  agents: Agent[]
  capabilities: Capability[]
  effectiveSelection: AskAccessSelection
  exampleAvailable: boolean
  history: AskAccessHistoryEntry[]
  liveDataAvailable: boolean
  loading: boolean
  message: string
  permissionChangeAvailable: boolean
  onChange: (selection: AskAccessSelection) => void
  onExplain: () => void
  onRunExampleQuery: () => void
  onOpenAccessProfile: (request: AccessDecisionExplainResult["request"]) => void
  onReviewCapability: (request: AccessDecisionExplainResult["request"]) => void
  onSelectHistory: (entry: AskAccessHistoryEntry) => void
  onStartPermissionChange: () => void
  recordRows: AskDecisionRecordRow[]
  requestBuild: ExplainRequestBuildResult
  result: AccessDecisionExplainResult | null
  t: Translator
  templates: PermissionPackageTemplate[]
  tenants: Tenant[]
}

export function AskAccessPanel({
  agents,
  capabilities,
  controller,
  liveDataAvailable,
  onOpenAccessProfile,
  onReviewCapability,
  t,
  tenants,
  title
}: {
  agents: Agent[]
  capabilities: Capability[]
  controller: AskAccessController
  liveDataAvailable: boolean
  onOpenAccessProfile: (request: AccessDecisionExplainResult["request"]) => void
  onReviewCapability: (request: AccessDecisionExplainResult["request"]) => void
  t: Translator
  tenants: Tenant[]
  title: string
}) {
  return (
    <Panel className="span-12" icon={<Search size={18} />} title={title}>
      <AskAccessView
        agents={agents}
        capabilities={capabilities}
        effectiveSelection={controller.effectiveSelection}
        exampleAvailable={controller.exampleAvailable}
        history={controller.history}
        liveDataAvailable={liveDataAvailable}
        loading={controller.loading}
        message={controller.message}
        permissionChangeAvailable={controller.permissionChangeAvailable}
        onChange={controller.updateSelection}
        onExplain={() => void controller.explain()}
        onOpenAccessProfile={onOpenAccessProfile}
        onReviewCapability={onReviewCapability}
        onRunExampleQuery={() => void controller.runExampleQuery()}
        onSelectHistory={controller.selectHistory}
        onStartPermissionChange={controller.startPermissionChange}
        recordRows={controller.recordRows}
        requestBuild={controller.requestBuild}
        result={controller.result}
        t={t}
        templates={controller.templates}
        tenants={tenants}
      />
    </Panel>
  );
}

export function AskAccessView({
  agents,
  capabilities,
  effectiveSelection,
  exampleAvailable,
  history: recentHistory,
  liveDataAvailable,
  loading,
  message,
  permissionChangeAvailable,
  onChange,
  onExplain,
  onOpenAccessProfile,
  onReviewCapability,
  onRunExampleQuery,
  onSelectHistory,
  onStartPermissionChange,
  recordRows,
  requestBuild,
  result,
  t,
  templates,
  tenants
}: AskAccessViewProps) {
  const scopedOptions = askAccessScopeOptions({ agents, capabilities, tenants }, effectiveSelection);
  const tenantOptions = scopedOptions.tenants.map((tenant) => ({
    label: permissionEntityDisplayName(tenant.name, t),
    value: tenant.id
  }));
  const workspaceOptions = uniqueOptions(
    scopedOptions.workspaceIds,
    (workspaceId) => workspaceLabel(workspaceId, t)
  );
  const callerOptions = agentOptions(scopedOptions.callers, t);
  const targetOptions = agentOptions(scopedOptions.targets, t);
  const capabilityOptions = scopedOptions.capabilities.map((capability) => ({
    label: capabilityDisplayName(capability, t),
    value: capability.id
  }));
  const canExplain = requestBuild.complete && liveDataAvailable && !loading;
  const setupBlocker = askSetupBlocker({
    callerOptions,
    capabilityOptions,
    liveDataAvailable,
    targetOptions,
    tenantOptions
  });
  const answerHeading = result
    ? result.outcome === "allowed" ? t("ask.allowedTitle") : t("ask.deniedTitle")
    : t("ask.answerPendingTitle");
  const primaryAction = result ? accessDecisionPrimaryAction(result, capabilities, templates, permissionChangeAvailable) : null;
  const dataScopeLabels = dataScopeValueLabels(t);

  return (
    <div className="ask-access">
      <header className="ask-access-header">
        <div>
          <span>{t("ask.kicker")}</span>
          <h2>{t("ask.title")}</h2>
          <p>{t("ask.subtitle")}</p>
        </div>
      </header>

      <div className="ask-workspace">
        <div className="ask-primary-column">
          <form
            aria-busy={loading}
            aria-label={t("ask.questionLabel")}
            className="ask-query-panel"
            onSubmit={(event) => {
              event.preventDefault();
              if (canExplain) onExplain();
            }}
          >
            <div className="ask-query-panel-heading">
              <span>{t("ask.questionLabel")}</span>
              <strong>{t("ask.questionTitle")}</strong>
            </div>
            <div className="ask-query-groups">
              <section className="ask-query-group" aria-label={t("ask.group.context")}>
                <span className="ask-query-group-title">{t("ask.group.context")}</span>
                <div className="ask-query-grid ask-query-grid-context">
                  <QueryField
                    label={t("ask.field.tenant")}
                    disabled={loading}
                    onChange={(tenantId) => onChange({ tenantId })}
                    options={withEmptyOption(tenantOptions, t("ask.emptyOption.tenant"))}
                    value={effectiveSelection.tenantId ?? ""}
                  />
                  <QueryField
                    label={t("ask.field.workspace")}
                    disabled={loading}
                    onChange={(workspaceId) => onChange({ workspaceId })}
                    options={withEmptyOption(workspaceOptions, t("ask.emptyOption.workspace"))}
                    value={effectiveSelection.workspaceId ?? ""}
                  />
                </div>
              </section>
              <section className="ask-query-group" aria-label={t("ask.group.access")}>
                <span className="ask-query-group-title">{t("ask.group.access")}</span>
                <div className="ask-query-grid ask-query-grid-access">
                  <QueryField
                    label={t("ask.field.caller")}
                    disabled={loading}
                    onChange={(callerInstanceId) => onChange({ callerInstanceId })}
                    options={withEmptyOption(callerOptions, t("ask.emptyOption.caller"))}
                    value={effectiveSelection.callerInstanceId ?? ""}
                  />
                  <QueryField
                    label={t("ask.field.target")}
                    disabled={loading}
                    onChange={(targetId) => onChange({ targetId })}
                    options={withEmptyOption(targetOptions, t("ask.emptyOption.target"))}
                    value={effectiveSelection.targetId ?? ""}
                  />
                  <QueryField
                    label={t("ask.field.capability")}
                    disabled={loading}
                    onChange={(capabilityId) => onChange({ capabilityId })}
                    options={withEmptyOption(capabilityOptions, t("ask.emptyOption.capability"))}
                    value={effectiveSelection.capabilityId ?? ""}
                  />
                </div>
              </section>
            </div>
            {setupBlocker ? (
              <div className="ask-setup-blocker" role="note">
                <AlertCircle aria-hidden="true" size={16} />
                <div>
                  <strong>{t(setupBlocker.titleKey)}</strong>
                  <span>{t(setupBlocker.detailKey)}</span>
                </div>
                <a className="secondary-button" href={setupBlocker.href}>
                  {t(setupBlocker.actionKey)}
                  <ArrowRight aria-hidden="true" size={14} />
                </a>
              </div>
            ) : null}
            <div className="ask-query-footer">
              <label className="ask-subject-field">
                <span>{t("ask.field.subject")}</span>
                <input
                  autoComplete="off"
                  disabled={loading}
                  name="accessSubject"
                  onChange={(event) => onChange({ subjectId: event.target.value })}
                  placeholder={t("ask.subjectPlaceholder")}
                  type="text"
                  value={effectiveSelection.subjectId ?? ""}
                />
              </label>
              <button className="primary-button" disabled={!canExplain} type="submit">
                <Search aria-hidden="true" size={15} />
                {loading ? t("action.loading") : t("action.askAccess")}
              </button>
            </div>
            {message ? <p className="ask-inline-message" role="status">{message}</p> : null}
          </form>

          <section
            aria-busy={loading}
            aria-label={t("ask.answerTitle")}
            className="ask-answer"
          >
            <header className="ask-answer-heading">
              <span>{t("ask.answerTitle")}</span>
              <strong>{answerHeading}</strong>
            </header>
            {!result ? (
              <div className="ask-answer-empty">
                <Search aria-hidden="true" size={18} />
                <div>
                  <strong>{t("ask.emptyTitle")}</strong>
                  <span>{liveDataAvailable ? t("ask.emptyDetail") : t("ask.liveApiRequiredDetail")}</span>
                </div>
                {exampleAvailable ? (
                  <button className="secondary-button" onClick={onRunExampleQuery} type="button">
                    {t("action.runExampleQuery")}
                  </button>
                ) : null}
              </div>
            ) : (
              <>
                <div className={`ask-answer-banner status-${result.outcome}`}>
                  <Badge tone={result.outcome === "allowed" ? "success" : "danger"}>
                    {result.outcome === "allowed" ? t("text.decisionAllowed") : t("text.decisionDenied")}
                  </Badge>
                  <div aria-atomic="true" aria-live="polite" className="ask-answer-summary" role="status">
                    <strong>{result.outcome === "allowed" ? t("ask.allowedTitle") : t("ask.deniedTitle")}</strong>
                    <p>{accessDecisionSummaryLabel(result, t)}</p>
                  </div>
                  {primaryAction?.kind === "permission_change" ? (
                    <button className="primary-button" onClick={onStartPermissionChange} type="button">
                      <Wrench aria-hidden="true" size={15} />
                      {t(primaryAction.labelKey)}
                    </button>
                  ) : primaryAction?.kind === "capability_review" ? (
                    <button className="primary-button" onClick={() => onReviewCapability(result.request)} type="button">
                      <FileSearch aria-hidden="true" size={15} />
                      {t(primaryAction.labelKey)}
                    </button>
                  ) : primaryAction?.kind === "access_profile" ? (
                    <button className="primary-button" onClick={() => onOpenAccessProfile(result.request)} type="button">
                      <FileSearch aria-hidden="true" size={15} />
                      {t(primaryAction.labelKey)}
                    </button>
                  ) : null}
                </div>

                <ResultContext request={result.request} agents={agents} capabilities={capabilities} t={t} tenants={tenants} />

                {result.outcome === "allowed" ? (
                  <div className="ask-data-scope">
                    <strong>{t("ask.effectiveDataScope")}</strong>
                    <span>{summarizeDataScopes(result.dataScopes, t("ask.noEffectiveDataScope"), dataScopeLabels)}</span>
                  </div>
                ) : null}

                <div className="ask-chain">
                  <header>
                    <strong>{t("ask.chainTitle")}</strong>
                    <span>{t("ask.chainDetail")}</span>
                  </header>
                  <ol>
                    {recordRows.map((row) => (
                      <li className={row.isBroken ? "is-broken" : ""} key={`${row.layer}-${row.id ?? row.message}`}>
                        <Badge tone={row.tone}>{t(`status.${row.status}`, row.status)}</Badge>
                        <div>
                          <strong>{t(row.layerKey, readableIdentifierLabel(row.layer))}</strong>
                          <span>{accessDecisionRecordMessageLabel(row, t)}</span>
                        </div>
                      </li>
                    ))}
                  </ol>
                </div>

                {result.nextActions.length > 0 ? (
                  <div className="ask-next-actions">
                    <strong>{t("ask.nextActions")}</strong>
                    <ul>
                      {result.nextActions.map((action) => <li key={action}>{accessNextActionLabel(action, t)}</li>)}
                    </ul>
                  </div>
                ) : null}
              </>
            )}
          </section>
        </div>

        <aside className="ask-context-column">
          <section className="ask-context-card" aria-label={t("ask.dataSourceTitle")}>
            <header>
              <strong>{t("ask.dataSourceTitle")}</strong>
              <Badge tone={liveDataAvailable ? "success" : "warning"}>
                {liveDataAvailable ? t("ask.liveMode") : t("ask.sampleMode")}
              </Badge>
            </header>
            <p>{liveDataAvailable ? t("ask.liveModeDetail") : t("ask.sampleModeDetail")}</p>
          </section>

          <section className="ask-context-card ask-history" aria-label={t("ask.historyTitle")}>
            <header>
              <History aria-hidden="true" size={16} />
              <strong>{t("ask.historyTitle")}</strong>
            </header>
            {recentHistory.length === 0 ? (
              <span>{t("ask.noHistory")}</span>
            ) : (
              <div className="ask-history-list">
                {recentHistory.map((entry) => (
                  <button key={entry.id} onClick={() => onSelectHistory(entry)} type="button">
                    <span>{historyLabel(entry.request, agents, capabilities, t)}</span>
                    <ArrowRight aria-hidden="true" size={14} />
                  </button>
                ))}
              </div>
            )}
          </section>
        </aside>
      </div>
    </div>
  );
}

function QueryField({
  disabled,
  label,
  onChange,
  options,
  value
}: {
  disabled: boolean
  label: string
  onChange: (value: string) => void
  options: ApprovalDropdownOption[]
  value: string
}) {
  return (
    <label className="ask-query-field">
      <span>{label}</span>
      <ApprovalDropdown disabled={disabled} label={label} onChange={onChange} options={options} value={value} />
    </label>
  );
}

function ResultContext({
  agents,
  capabilities,
  request,
  t,
  tenants
}: {
  agents: Agent[]
  capabilities: Capability[]
  request: AccessDecisionExplainResult["request"]
  t: Translator
  tenants: Tenant[]
}) {
  const rows = [
    [t("ask.field.tenant"), permissionEntityDisplayName(tenants.find((tenant) => tenant.id === request.tenantId)?.name ?? request.tenantId, t)],
    [t("ask.field.workspace"), workspaceLabel(request.workspaceId, t)],
    [t("ask.field.caller"), permissionEntityDisplayName(agents.find((agent) => agent.id === request.callerInstanceId)?.name ?? request.callerInstanceId, t)],
    [t("ask.field.target"), permissionEntityDisplayName(agents.find((agent) => agent.id === request.targetId)?.name ?? request.targetId, t)],
    [t("ask.field.capability"), capabilityLabel(request.capabilityId, capabilities, t)],
    ...(request.subjectId ? [[t("ask.field.subject"), request.subjectId]] : [])
  ];
  return (
    <div className="ask-result-context" aria-label={t("ask.resultContextTitle")}>
      {rows.map(([label, value]) => (
        <div key={label}>
          <span>{label}</span>
          <strong>{value}</strong>
        </div>
      ))}
    </div>
  );
}

function capabilityLabel(capabilityId: string, capabilities: Capability[], t: Translator) {
  const capability = capabilities.find((item) => item.id === capabilityId);
  return capability ? capabilityDisplayName(capability, t) : capabilityId;
}

function agentOptions(agents: Agent[], t: Translator) {
  return agents.map((agent) => ({ label: permissionEntityDisplayName(agent.name, t), value: agent.id }));
}

function uniqueOptions(values: string[], labelFor: (value: string) => string) {
  return Array.from(new Set(values.filter(Boolean))).map((value) => ({ label: labelFor(value), value }));
}

function withEmptyOption(options: ApprovalDropdownOption[], emptyLabel: string) {
  return [{ label: emptyLabel, value: "" }, ...options];
}

interface AskSetupBlockerInput {
  callerOptions: ApprovalDropdownOption[]
  capabilityOptions: ApprovalDropdownOption[]
  liveDataAvailable: boolean
  targetOptions: ApprovalDropdownOption[]
  tenantOptions: ApprovalDropdownOption[]
}

interface AskSetupBlocker {
  actionKey: string
  detailKey: string
  href: "#registry" | "#capabilities"
  titleKey: string
}

function askSetupBlocker({
  callerOptions,
  capabilityOptions,
  liveDataAvailable,
  targetOptions,
  tenantOptions
}: AskSetupBlockerInput): AskSetupBlocker | null {
  if (!liveDataAvailable) {
    return null;
  }
  if (tenantOptions.length === 0 || callerOptions.length === 0 || targetOptions.length === 0) {
    return {
      actionKey: "ask.setupBlocker.resources.action",
      detailKey: "ask.setupBlocker.resources.detail",
      href: "#registry",
      titleKey: "ask.setupBlocker.resources.title"
    };
  }
  if (capabilityOptions.length === 0) {
    return {
      actionKey: "ask.setupBlocker.capabilities.action",
      detailKey: "ask.setupBlocker.capabilities.detail",
      href: "#capabilities",
      titleKey: "ask.setupBlocker.capabilities.title"
    };
  }
  return null;
}

function workspaceLabel(workspaceId: string, t: Translator) {
  return permissionEntityDisplayName(readableIdentifierLabel(workspaceId), t);
}

function historyLabel(
  request: AccessDecisionExplainResult["request"],
  agents: Agent[],
  capabilities: Capability[],
  t: Translator
) {
  const caller = permissionEntityDisplayName(
    agents.find((agent) => agent.id === request.callerInstanceId)?.name ?? request.callerInstanceId,
    t
  );
  const target = permissionEntityDisplayName(
    agents.find((agent) => agent.id === request.targetId)?.name ?? request.targetId,
    t
  );
  const capability = capabilities.find((item) => item.id === request.capabilityId);
  const capabilityName = capability ? capabilityDisplayName(capability, t) : request.capabilityId;
  return `${caller} -> ${target} / ${capabilityName}`;
}
