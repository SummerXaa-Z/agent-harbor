import { useState, type ReactNode } from "react";
import {
  CheckCircle2,
  CircleDot,
  ClipboardCheck,
  FileSearch,
  LockKeyhole,
  MoreHorizontal,
  Route
} from "lucide-react";

import {
  agentNameMap,
  formatDate,
  permissionEntityDisplayName,
  policyEffectLabel,
  translatedValue,
  type Translator
} from "../consolePresenters";
import type {
  Agent,
  AgentStatus,
  AskHandoffContext,
  ChannelContract,
  RoutePolicy
} from "../types";
import { TechnicalId } from "./TechnicalId";
import { Badge, EmptyRow } from "./ui";

export function PolicyTable({
  agents,
  canDisable,
  onDisable,
  pendingActionId,
  policies,
  t
}: {
  agents: Agent[];
  canDisable: boolean;
  onDisable: (policy: RoutePolicy) => void;
  pendingActionId: string;
  policies: RoutePolicy[];
  t: Translator;
}) {
  const names = agentNameMap(agents);

  return (
    <div className="table-wrap">
      <table className="policy-table">
        <thead>
          <tr>
            <th>{t("table.policy")}</th>
            <th>{t("table.caller")}</th>
            <th>{t("table.target")}</th>
            <th>{t("table.route")}</th>
            <th>{t("table.decision")}</th>
            <th>{t("table.action")}</th>
          </tr>
        </thead>
        <tbody>
          {policies.length === 0 ? (
            <tr>
              <td colSpan={6}>
                <EmptyRow title={t("empty.routePolicies.title")} detail={t("empty.routePolicies.detail")} />
              </td>
            </tr>
          ) : null}
          {policies.map((policy) => (
            <tr className={policy.status === "disabled" ? "row-disabled" : undefined} key={policy.id}>
              <td>
                <strong>{policy.name}</strong>
                <span>{tx(t, "text.policyPriority", { priority: policy.priority })} · {policyRetryText(policy, t)} · {tx(t, "text.policyMatched", { date: formatDate(policy.lastMatchedAt ?? policy.createdAt) })}</span>
              </td>
              <td>{names[policy.callerAgentId] ?? policy.callerAgentId}</td>
              <td>{names[policy.targetAgentId] ?? policy.targetAgentId}</td>
              <td>
                <code>{policy.routeType}:{policy.routeKey || t("text.routeWildcard")}</code>
              </td>
              <td><Badge tone={policy.effect === "allow" ? "success" : "danger"}>{policyEffectLabel(policy.effect, t)}</Badge></td>
              <td>
                {canDisable && policy.status === "enabled" ? (
                  <button
                    className="table-action is-danger"
                    disabled={pendingActionId === policy.id}
                    onClick={() => onDisable(policy)}
                    type="button"
                  >
                    <LockKeyhole size={13} />
                    {pendingActionId === policy.id ? t("action.disabling") : t("action.disable")}
                  </button>
                ) : (
                  <span className="muted-action">{policy.status === "disabled" ? t("status.agentDisabled") : t("status.sample")}</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function AccessPolicyWorkspace({
  createPolicyPanel,
  managementAuditPanel,
  policies,
  routeGovernancePanel,
  t
}: {
  createPolicyPanel: ReactNode;
  managementAuditPanel: ReactNode;
  policies: RoutePolicy[];
  routeGovernancePanel: ReactNode;
  t: Translator;
}) {
  const [auditCollapsed, setAuditCollapsed] = useState(true);

  function focusPolicyForm() {
    const trigger = document.querySelector<HTMLButtonElement>("#policy-create-panel .action-modal-trigger");
    trigger?.scrollIntoView({ behavior: "smooth", block: "center" });
    trigger?.click();
  }

  return (
    <>
      {policies.length === 0 ? (
        <section className="policy-workspace span-12">
          <div>
            <span className="section-kicker">{t("nav.policies")}</span>
            <h3>{t("text.policyEmptyTitle")}</h3>
            <p>{t("text.policyEmptyDetail")}</p>
          </div>
          <button className="policy-empty-action" type="button" onClick={focusPolicyForm}>
            <Route size={14} />
            {t("action.createFirstPolicy")}
          </button>
        </section>
      ) : null}
      {createPolicyPanel}
      {routeGovernancePanel}
      <section className="policy-audit-disclosure span-12">
        <button
          className="secondary-button"
          onClick={() => setAuditCollapsed((collapsed) => !collapsed)}
          type="button"
        >
          <ClipboardCheck size={14} />
          {auditCollapsed ? t("action.showAudit") : t("action.hideAudit")}
        </button>
        {!auditCollapsed ? managementAuditPanel : null}
      </section>
    </>
  );
}

export function AgentTable({
  agents,
  channelLabels,
  onStatusChange,
  onQueryAccess,
  pendingActionId,
  t
}: {
  agents: Agent[];
  channelLabels: Record<string, string>;
  onStatusChange: (agent: Agent, status: AgentStatus) => void;
  onQueryAccess: (context: AskHandoffContext) => void;
  pendingActionId: string;
  t: Translator;
}) {
  const [agentQuery, setAgentQuery] = useState("");
  const [agentStatusFilter, setAgentStatusFilter] = useState<AgentStatus | "">("");
  const [selectedAgentId, setSelectedAgentId] = useState("");
  const normalizedAgentQuery = agentQuery.trim().toLowerCase();
  const visibleAgents = agents.filter((agent) => {
    const agentStatus = agentStatusLabel(agent.status, t);
    const searchable = [
      agent.name,
      agent.description ?? "",
      channelLabel(agent.channelType, channelLabels, t),
      agentStatus,
      configText(agent, "endpoint"),
      agent.ownerId ?? ""
    ].join(" ").toLowerCase();
    return (!agentStatusFilter || agent.status === agentStatusFilter) && (!normalizedAgentQuery || searchable.includes(normalizedAgentQuery));
  });
  const selectedAgent = agents.find((agent) => agent.id === selectedAgentId) ?? null;
  const hasAgents = agents.length > 0;

  return (
    <div className="agent-registry">
      {hasAgents ? (
        <div className="table-toolbar">
          <label>
            <span>{t("form.agent")}</span>
            <input
              placeholder={t("form.searchAgents")}
              value={agentQuery}
              onChange={(event) => setAgentQuery(event.target.value)}
            />
          </label>
          <label>
            <span>{t("table.status")}</span>
            <select
              value={agentStatusFilter}
              onChange={(event) => setAgentStatusFilter(event.target.value as AgentStatus | "")}
            >
              <option value="">{t("form.anyStatus")}</option>
              <option value="active">{t("status.agentActive")}</option>
              <option value="draft">{t("status.agentDraft")}</option>
              <option value="disabled">{t("status.agentDisabled")}</option>
            </select>
          </label>
          <span>{tx(t, "text.visibleRowCount", { count: visibleAgents.length, total: agents.length })}</span>
        </div>
      ) : null}
      {visibleAgents.length === 0 ? (
        <div className="registry-empty-state">
          <EmptyRow
            title={hasAgents ? t("empty.filteredResults.title") : t("empty.registry.title")}
            detail={hasAgents ? t("empty.filteredResults.detail") : t("empty.registry.detail")}
            actionLabel={hasAgents ? undefined : t("empty.registry.action")}
            actionHash={hasAgents ? undefined : "#getting-started"}
          />
        </div>
      ) : (
        <div className="table-wrap">
          <table className="agent-table">
            <thead>
              <tr>
                <th>{t("table.name")}</th>
                <th>{t("table.channel")}</th>
                <th>{t("table.endpoint")}</th>
                <th>{t("table.status")}</th>
                <th>{t("table.owner")}</th>
                <th>{t("table.action")}</th>
              </tr>
            </thead>
            <tbody>
              {visibleAgents.map((agent) => (
                <tr className={agent.status === "disabled" ? "row-disabled" : undefined} key={agent.id}>
                  <td>
                    <strong>{permissionEntityDisplayName(agent.name, t)}</strong>
                    {agent.description ? <span>{agent.description}</span> : <TechnicalId copyLabel={t("action.copy")} value={agent.id} />}
                  </td>
                  <td>{channelLabel(agent.channelType, channelLabels, t)}</td>
                  <td className="truncate">{configText(agent, "endpoint") || t("status.localRuntime")}</td>
                  <td><Badge tone={agent.status === "active" ? "success" : agent.status === "draft" ? "warning" : "neutral"}>{agentStatusLabel(agent.status, t)}</Badge></td>
                  <td>{agent.ownerId || t("text.ownerPlatform")}</td>
                  <td>
                    <div className="table-action-group">
                      <button
                        className="table-action"
                        onClick={() => setSelectedAgentId(agent.id)}
                        type="button"
                      >
                        <FileSearch size={13} />
                        {t("action.viewDetails")}
                      </button>
                      {agent.status !== "disabled" ? (
                        <details className="row-action-menu">
                          <summary className="table-action">
                            <MoreHorizontal size={13} />
                            {t("action.more")}
                          </summary>
                          <div className="row-action-menu-list">
                            <button
                              className="table-action"
                              disabled={pendingActionId === agent.id}
                              onClick={() => onStatusChange(agent, agent.status === "active" ? "draft" : "active")}
                              type="button"
                            >
                              {agent.status === "active" ? <CircleDot size={13} /> : <CheckCircle2 size={13} />}
                              {pendingActionId === agent.id ? t("action.updating") : agent.status === "active" ? t("action.draft") : t("action.activate")}
                            </button>
                            <button
                              className="table-action is-danger"
                              disabled={pendingActionId === agent.id}
                              onClick={() => onStatusChange(agent, "disabled")}
                              type="button"
                            >
                              <LockKeyhole size={13} />
                              {t("action.disable")}
                            </button>
                          </div>
                        </details>
                      ) : (
                        <span className="muted-action">{t("status.agentDisabled")}</span>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {selectedAgent ? (
        <aside className="table-detail-panel">
          <div>
            <span className="section-kicker">{t("text.agentDetails")}</span>
            <h3>{permissionEntityDisplayName(selectedAgent.name, t)}</h3>
            {selectedAgent.description ? <p>{selectedAgent.description}</p> : null}
          </div>
          <div className="table-detail-grid">
            <span>{t("table.status")}<strong>{agentStatusLabel(selectedAgent.status, t)}</strong></span>
            <span>{t("table.channel")}<strong>{channelLabel(selectedAgent.channelType, channelLabels, t)}</strong></span>
            <span>{t("table.endpoint")}<strong>{configText(selectedAgent, "endpoint") || t("status.localRuntime")}</strong></span>
            <span>{t("table.owner")}<strong>{selectedAgent.ownerId || t("text.ownerPlatform")}</strong></span>
            <span>{t("form.workspace")}<strong>{selectedAgent.workspaceId || t("text.notSpecified")}</strong></span>
            <span>{t("text.technicalDetails")}<TechnicalId copyLabel={t("action.copy")} value={selectedAgent.id} /></span>
          </div>
          <button
            className="secondary-button table-detail-action"
            onClick={() => onQueryAccess({
              callerInstanceId: selectedAgent.channelType === "local" ? selectedAgent.id : undefined,
              sourceView: "registry",
              targetId: selectedAgent.channelType !== "local" ? selectedAgent.id : undefined,
              tenantId: selectedAgent.tenantId,
              workspaceId: selectedAgent.workspaceId
            })}
            type="button"
          >
            <FileSearch size={14} />
            {selectedAgent.channelType === "local" ? t("action.queryThisCallerAccess") : t("action.queryThisTargetAccess")}
          </button>
        </aside>
      ) : null}
    </div>
  );
}

export function ContractMatrix({
  channels,
  providers,
  t
}: {
  channels: ChannelContract[];
  providers: Array<{ key: string; label: string; channelType: string; requiredCreds?: string[] }>;
  t: Translator;
}) {
  return (
    <div className="contract-list">
      {channels.map((channel) => (
        <div className="contract-row" key={channel.key}>
          <div>
            <strong>{channelLabel(channel.key, { [channel.key]: channel.label }, t)}</strong>
            <span>{channel.key}</span>
          </div>
          <Badge tone={channel.endpointRequiredWhenActive ? "warning" : "neutral"}>
            {channel.endpointRequiredWhenActive ? t("form.endpoint") : translatedValue(t, "local")}
          </Badge>
        </div>
      ))}
      <div className="provider-strip">
        {providers.map((provider) => (
          <span key={provider.key}>{tx(t, "text.provider", { label: provider.label, channelType: translatedValue(t, provider.channelType) })}</span>
        ))}
      </div>
    </div>
  );
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}

function agentStatusLabel(status: AgentStatus, t: Translator) {
  if (status === "active") return t("status.agentActive");
  if (status === "disabled") return t("status.agentDisabled");
  return t("status.agentDraft");
}

function configText(agent: Agent, key: string) {
  const value = agent.channelConfig?.[key];
  return typeof value === "string" ? value : "";
}

function channelLabel(channelType: string, channelLabels: Record<string, string>, t: Translator) {
  return t(`value.${channelType}`, channelLabels[channelType] ?? channelType);
}

function policyRetryText(policy: RoutePolicy, t: Translator) {
  if (!policy.retry) return t("text.targetRetry");
  const statuses = policy.retry.statusCodes.length > 0 ? policy.retry.statusCodes.join("/") : t("text.retryNone");
  return `retry ${policy.retry.maxAttempts}x ${policy.retry.backoffMs}ms ${statuses}`;
}
