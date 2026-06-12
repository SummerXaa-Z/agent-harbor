import { ArrowRight, History, Search, Wrench } from "lucide-react";

import type {
  AskAccessSelection,
  AskEvidenceChainRow,
  ExplainRequestBuildResult
} from "../askJourney";
import {
  capabilityDisplayName,
  permissionEntityDisplayName,
  readableIdentifierLabel,
  type Translator
} from "../consolePresenters";
import type { AskAccessController, AskAccessHistoryEntry } from "../hooks/useAskAccessController";
import type { AccessDecisionExplainResult, Agent, Capability, Tenant } from "../types";
import { ApprovalDropdown, type ApprovalDropdownOption } from "./ApprovalDropdown";
import { Panel } from "./ConsolePrimitives";
import { Badge } from "./ui";

interface AskAccessViewProps {
  agents: Agent[]
  capabilities: Capability[]
  chainRows: AskEvidenceChainRow[]
  effectiveSelection: AskAccessSelection
  exampleAvailable: boolean
  history: AskAccessHistoryEntry[]
  liveDataAvailable: boolean
  loading: boolean
  message: string
  onChange: (selection: AskAccessSelection) => void
  onExplain: () => void
  onRunExampleQuery: () => void
  onSelectHistory: (entry: AskAccessHistoryEntry) => void
  onStartPermissionChange: () => void
  requestBuild: ExplainRequestBuildResult
  result: AccessDecisionExplainResult | null
  t: Translator
  tenants: Tenant[]
}

export function AskAccessPanel({
  agents,
  capabilities,
  controller,
  liveDataAvailable,
  t,
  tenants,
  title
}: {
  agents: Agent[]
  capabilities: Capability[]
  controller: AskAccessController
  liveDataAvailable: boolean
  t: Translator
  tenants: Tenant[]
  title: string
}) {
  return (
    <Panel className="span-12" icon={<Search size={18} />} title={title}>
      <AskAccessView
        agents={agents}
        capabilities={capabilities}
        chainRows={controller.chainRows}
        effectiveSelection={controller.effectiveSelection}
        exampleAvailable={controller.exampleAvailable}
        history={controller.history}
        liveDataAvailable={liveDataAvailable}
        loading={controller.loading}
        message={controller.message}
        onChange={controller.updateSelection}
        onExplain={() => void controller.explain()}
        onRunExampleQuery={() => void controller.runExampleQuery()}
        onSelectHistory={controller.selectHistory}
        onStartPermissionChange={controller.startPermissionChange}
        requestBuild={controller.requestBuild}
        result={controller.result}
        t={t}
        tenants={tenants}
      />
    </Panel>
  );
}

export function AskAccessView({
  agents,
  capabilities,
  chainRows,
  effectiveSelection,
  exampleAvailable,
  history: recentHistory,
  liveDataAvailable,
  loading,
  message,
  onChange,
  onExplain,
  onRunExampleQuery,
  onSelectHistory,
  onStartPermissionChange,
  requestBuild,
  result,
  t,
  tenants
}: AskAccessViewProps) {
  const tenantOptions = tenants.map((tenant) => ({
    label: permissionEntityDisplayName(tenant.name, t),
    value: tenant.id
  }));
  const workspaceOptions = uniqueOptions(
    agents.map((agent) => agent.workspaceId),
    (workspaceId) => workspaceLabel(workspaceId, t)
  );
  const callerOptions = agentOptions(
    agents.filter((agent) => agent.channelType === "local" || agent.status === "active"),
    t
  );
  const targetIds = new Set(capabilities.map((capability) => capability.targetId));
  const targetOptions = agentOptions(agents.filter((agent) => targetIds.has(agent.id)), t);
  const capabilityOptions = capabilities
    .filter((capability) => !effectiveSelection.targetId || capability.targetId === effectiveSelection.targetId)
    .map((capability) => ({
      label: capabilityDisplayName(capability, t),
      value: capability.id
    }));
  const canExplain = requestBuild.complete && liveDataAvailable && !loading;
  const answerHeading = result
    ? result.outcome === "allowed" ? t("ask.allowedTitle") : t("ask.deniedTitle")
    : t("ask.answerPendingTitle");

  return (
    <div className="ask-access">
      <header className="ask-access-header">
        <div>
          <span>{t("ask.kicker")}</span>
          <h2>{t("ask.title")}</h2>
          <p>{t("ask.subtitle")}</p>
        </div>
        <Badge tone={liveDataAvailable ? "success" : "warning"}>
          {liveDataAvailable ? t("ask.liveMode") : t("ask.sampleMode")}
        </Badge>
      </header>

      <div className="ask-workspace">
        <div className="ask-primary-column">
          <section className="ask-query-panel" aria-label={t("ask.questionLabel")}>
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
                    onChange={(tenantId) => onChange({ tenantId })}
                    options={withEmptyOption(tenantOptions, t("ask.emptyOption.tenant"))}
                    value={effectiveSelection.tenantId ?? ""}
                  />
                  <QueryField
                    label={t("ask.field.workspace")}
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
                    onChange={(callerInstanceId) => onChange({ callerInstanceId })}
                    options={withEmptyOption(callerOptions, t("ask.emptyOption.caller"))}
                    value={effectiveSelection.callerInstanceId ?? ""}
                  />
                  <QueryField
                    label={t("ask.field.target")}
                    onChange={(targetId) => onChange({ targetId })}
                    options={withEmptyOption(targetOptions, t("ask.emptyOption.target"))}
                    value={effectiveSelection.targetId ?? ""}
                  />
                  <QueryField
                    label={t("ask.field.capability")}
                    onChange={(capabilityId) => onChange({ capabilityId })}
                    options={withEmptyOption(capabilityOptions, t("ask.emptyOption.capability"))}
                    value={effectiveSelection.capabilityId ?? ""}
                  />
                </div>
              </section>
            </div>
            <div className="ask-query-footer">
              <label className="ask-subject-field">
                <span>{t("ask.field.subject")}</span>
                <input
                  autoComplete="off"
                  name="accessSubject"
                  onChange={(event) => onChange({ subjectId: event.target.value })}
                  placeholder={t("ask.subjectPlaceholder")}
                  type="text"
                  value={effectiveSelection.subjectId ?? ""}
                />
              </label>
              <button className="primary-button" disabled={!canExplain} onClick={onExplain} type="button">
                <Search aria-hidden="true" size={15} />
                {loading ? t("action.loading") : t("action.askAccess")}
              </button>
            </div>
            {message ? <p className="ask-inline-message">{message}</p> : null}
          </section>

          <section className="ask-answer" aria-label={t("ask.answerTitle")}>
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
                  <div>
                    <strong>{result.outcome === "allowed" ? t("ask.allowedTitle") : t("ask.deniedTitle")}</strong>
                    <p>{result.summary}</p>
                  </div>
                  {result.outcome === "denied" ? (
                    <button className="primary-button" onClick={onStartPermissionChange} type="button">
                      <Wrench aria-hidden="true" size={15} />
                      {t("action.fixAccessDecision")}
                    </button>
                  ) : null}
                </div>

                <div className="ask-chain">
                  <header>
                    <strong>{t("ask.chainTitle")}</strong>
                    <span>{t("ask.chainDetail")}</span>
                  </header>
                  <ol>
                    {chainRows.map((row) => (
                      <li className={row.isBroken ? "is-broken" : ""} key={`${row.layer}-${row.id ?? row.message}`}>
                        <Badge tone={row.tone}>{t(`status.${row.status}`, row.status)}</Badge>
                        <div>
                          <strong>{t(row.layerKey, readableIdentifierLabel(row.layer))}</strong>
                          <span>{row.message}</span>
                        </div>
                      </li>
                    ))}
                  </ol>
                </div>

                {result.nextActions.length > 0 ? (
                  <div className="ask-next-actions">
                    <strong>{t("ask.nextActions")}</strong>
                    <ul>
                      {result.nextActions.map((action) => <li key={action}>{action}</li>)}
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
  label,
  onChange,
  options,
  value
}: {
  label: string
  onChange: (value: string) => void
  options: ApprovalDropdownOption[]
  value: string
}) {
  return (
    <label className="ask-query-field">
      <span>{label}</span>
      <ApprovalDropdown label={label} onChange={onChange} options={options} value={value} />
    </label>
  );
}

function agentOptions(agents: Agent[], t: Translator) {
  return agents.map((agent) => ({ label: permissionEntityDisplayName(agent.name, t), value: agent.id }));
}

function uniqueOptions(values: string[], labelFor: (value: string) => string) {
  return Array.from(new Set(values.filter(Boolean))).map((value) => ({ label: labelFor(value), value }));
}

function withEmptyOption(options: ApprovalDropdownOption[], emptyLabel: string) {
  return options.length > 0 ? options : [{ label: emptyLabel, value: "" }];
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
