import type { FormEvent } from "react";
import { Copy, RefreshCw, X } from "lucide-react";

import {
  formatDate,
  permissionEntityDisplayName,
  type Translator
} from "../consolePresenters";
import type { ResourceLifecycleActionContext } from "../resourceLifecycleActionPlanner";
import type {
  Agent,
  AgentStatus,
  CreateAgentKeyResponse,
  TraceDecision,
  TraceFilters
} from "../types";

const mcpRouteKeyPresets = ["initialize", "tools/list", "tools/call"];

export interface AgentCreateFormState {
  channelType: string;
  credentialHeader: string;
  credentialName: string;
  credentialValue: string;
  description: string;
  endpoint: string;
  name: string;
  retryBackoffMs: string;
  retryMaxAttempts: string;
  status: AgentStatus;
}

export interface KeyCreateFormState {
  agentId: string;
  expiresInSeconds: string;
  name: string;
}

export interface CredentialRotateFormState {
  agentId: string;
  credentialName: string;
  credentialValue: string;
}

export interface PolicyCreateFormState {
  callerAgentId: string;
  effect: string;
  name: string;
  priority: string;
  retryBackoffMs: string;
  retryMaxAttempts: string;
  routeKey: string;
  routeType: string;
  targetAgentId: string;
}

export function AgentCreateForm({
  form,
  message,
  onChange,
  onSubmit,
  submitting = false,
  t
}: {
  form: AgentCreateFormState;
  message: string;
  onChange: (form: AgentCreateFormState) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  submitting?: boolean;
  t: Translator;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
      <label>{t("form.name")}<input required value={form.name} onChange={(event) => onChange({ ...form, name: event.target.value })} /></label>
      <div className="form-row">
        <label>{t("form.channel")}<input value={form.channelType} onChange={(event) => onChange({ ...form, channelType: event.target.value })} /></label>
        <label>{t("table.status")}<select value={form.status} onChange={(event) => onChange({ ...form, status: event.target.value as AgentStatus })}><option value="draft">{t("status.agentDraft")}</option><option value="active">{t("status.agentActive")}</option><option value="disabled">{t("status.agentDisabled")}</option></select></label>
      </div>
      <label>{t("form.endpoint")}<input placeholder="https://api.example.com/a2a" value={form.endpoint} onChange={(event) => onChange({ ...form, endpoint: event.target.value })} /></label>
      <div className="form-row">
        <label>{t("form.credentialHeader")}<input placeholder="Authorization" value={form.credentialHeader} onChange={(event) => onChange({ ...form, credentialHeader: event.target.value })} /></label>
        <label>{t("form.credentialKey")}<input placeholder="apiToken" value={form.credentialName} onChange={(event) => onChange({ ...form, credentialName: event.target.value })} /></label>
      </div>
      <label>{t("form.secretValue")}<input placeholder="Bearer ..." type="password" value={form.credentialValue} onChange={(event) => onChange({ ...form, credentialValue: event.target.value })} /></label>
      <div className="form-row">
        <label>{t("form.retryAttempts")}<input inputMode="numeric" max={4} min={1} type="number" value={form.retryMaxAttempts} onChange={(event) => onChange({ ...form, retryMaxAttempts: event.target.value })} /></label>
        <label>{t("form.backoffMs")}<input inputMode="numeric" max={1000} min={0} type="number" value={form.retryBackoffMs} onChange={(event) => onChange({ ...form, retryBackoffMs: event.target.value })} /></label>
      </div>
      <label>{t("form.description")}<textarea rows={2} value={form.description} onChange={(event) => onChange({ ...form, description: event.target.value })} /></label>
      <FormFooter message={message} submitting={submitting} submittingLabel={t("action.processing")} submitLabel={t("action.createAgent")} />
    </form>
  );
}

export function KeyCreateForm({
  agents,
  context,
  createdKey,
  form,
  message,
  onChange,
  onDismissCreatedKey,
  onSubmit,
  submitting = false,
  t
}: {
  agents: Agent[];
  context?: ResourceLifecycleActionContext | null;
  createdKey: CreateAgentKeyResponse | null;
  form: KeyCreateFormState;
  message: string;
  onChange: (form: KeyCreateFormState) => void;
  onDismissCreatedKey: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  submitting?: boolean;
  t: Translator;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
      <ResourceActionContextStrip context={context} t={t} />
      <label>{t("form.caller")}<select required value={form.agentId} onChange={(event) => onChange({ ...form, agentId: event.target.value })}><option value="">{t("form.selectCaller")}</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <label>{t("form.name")}<input value={form.name} onChange={(event) => onChange({ ...form, name: event.target.value })} /></label>
      <label>{t("form.ttlSeconds")}<input inputMode="numeric" max={3600} min={1} type="number" value={form.expiresInSeconds} onChange={(event) => onChange({ ...form, expiresInSeconds: event.target.value })} /></label>
      {createdKey ? (
        <div className="one-time-key">
          <div><strong>{t("text.oneTimeKey")}</strong><span>{tx(t, "text.oneTimeKeyDetail", { expiresAt: formatDate(createdKey.expiresAt) })}</span></div>
          <code>{createdKey.key}</code>
          <div className="one-time-key-actions">
            <button className="secondary-button" type="button" onClick={() => void navigator.clipboard?.writeText(createdKey.key)}><Copy size={14} /> {t("action.copy")}</button>
            <button className="secondary-button" type="button" onClick={onDismissCreatedKey}><X size={14} /> {t("action.clearOneTimeKey")}</button>
          </div>
        </div>
      ) : null}
      <FormFooter message={message} submitting={submitting} submittingLabel={t("action.processing")} submitLabel={t("action.createKey")} />
    </form>
  );
}

export function CredentialRotateForm({
  agents,
  context,
  form,
  message,
  onChange,
  onSubmit,
  submitting = false,
  t
}: {
  agents: Agent[];
  context?: ResourceLifecycleActionContext | null;
  form: CredentialRotateFormState;
  message: string;
  onChange: (form: CredentialRotateFormState) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  submitting?: boolean;
  t: Translator;
}) {
  return (
    <form className="control-form" onSubmit={onSubmit}>
      <ResourceActionContextStrip context={context} t={t} />
      <label>{t("form.agent")}<select required value={form.agentId} onChange={(event) => onChange({ ...form, agentId: event.target.value })}><option value="">{t("form.selectAgent")}</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <label>{t("form.credentialKey")}<input placeholder="apiToken" value={form.credentialName} onChange={(event) => onChange({ ...form, credentialName: event.target.value })} /></label>
      <label>{t("form.newSecret")}<input placeholder="Bearer ..." type="password" value={form.credentialValue} onChange={(event) => onChange({ ...form, credentialValue: event.target.value })} /></label>
      <FormFooter message={message} submitting={submitting} submittingLabel={t("action.processing")} submitLabel={t("action.rotateCredential")} />
    </form>
  );
}

export function PolicyCreateForm({
  agents,
  context,
  form,
  message,
  onChange,
  onSubmit,
  submitting = false,
  t
}: {
  agents: Agent[];
  context?: ResourceLifecycleActionContext | null;
  form: PolicyCreateFormState;
  message: string;
  onChange: (form: PolicyCreateFormState) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  submitting?: boolean;
  t: Translator;
}) {
  return (
    <form className="control-form policy-create-form" id="policy-create-form" onSubmit={onSubmit}>
      <ResourceActionContextStrip context={context} t={t} />
      <label>{t("form.name")}<input placeholder="Allow MCP tools/call" value={form.name} onChange={(event) => onChange({ ...form, name: event.target.value })} /></label>
      <label>{t("form.caller")}<select required value={form.callerAgentId} onChange={(event) => onChange({ ...form, callerAgentId: event.target.value })}><option value="">{t("form.selectCaller")}</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <label>{t("form.target")}<select required value={form.targetAgentId} onChange={(event) => onChange({ ...form, targetAgentId: event.target.value })}><option value="">{t("form.anyTarget")}</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label>
      <div className="route-presets" aria-label={t("form.routeKeyPresets")}>
        {mcpRouteKeyPresets.map((preset) => (
          <button
            className={form.routeType === "mcp" && form.routeKey === preset ? "selected" : ""}
            key={preset}
            onClick={() => onChange({ ...form, routeKey: preset, routeType: "mcp" })}
            type="button"
          >
            {preset}
          </button>
        ))}
        <button
          className={form.routeType === "mcp" && form.routeKey === "" ? "selected" : ""}
          onClick={() => onChange({ ...form, routeKey: "", routeType: "mcp" })}
          type="button"
        >
          {t("text.routeWildcard")}
        </button>
      </div>
      <div className="form-row">
        <label>{t("form.routeType")}<input value={form.routeType} onChange={(event) => onChange({ ...form, routeType: event.target.value })} /></label>
        <label>{t("form.routeKey")}<input value={form.routeKey} onChange={(event) => onChange({ ...form, routeKey: event.target.value })} /></label>
      </div>
      <div className="form-row">
        <label>{t("form.effect")}<select value={form.effect} onChange={(event) => onChange({ ...form, effect: event.target.value })}><option value="allow">{t("status.policyAllow")}</option><option value="deny">{t("status.policyDeny")}</option></select></label>
        <label>{t("form.priority")}<input inputMode="numeric" min={0} type="number" value={form.priority} onChange={(event) => onChange({ ...form, priority: event.target.value })} /></label>
      </div>
      <div className="form-row">
        <label>{t("form.retryAttempts")}<input inputMode="numeric" max={4} min={1} type="number" value={form.retryMaxAttempts} onChange={(event) => onChange({ ...form, retryMaxAttempts: event.target.value })} /></label>
        <label>{t("form.retryBackoffMs")}<input inputMode="numeric" max={1000} min={0} type="number" value={form.retryBackoffMs} onChange={(event) => onChange({ ...form, retryBackoffMs: event.target.value })} /></label>
      </div>
      <FormFooter message={message} submitting={submitting} submittingLabel={t("action.processing")} submitLabel={t("action.createPolicy")} />
    </form>
  );
}

export function TraceFilterBar({
  agents,
  filters,
  onChange,
  onRefresh,
  t
}: {
  agents: Agent[];
  filters: TraceFilters;
  onChange: (filters: TraceFilters) => void;
  onRefresh: () => void;
  t: Translator;
}) {
  return (
    <div className="trace-filters">
      <label>
        <span>{t("form.decision")}</span>
        <select value={filters.decision ?? ""} onChange={(event) => onChange({ ...filters, decision: event.target.value as TraceDecision | "" })}>
          <option value="">{t("form.anyDecision")}</option>
          <option value="allowed">{t("text.decisionAllowed")}</option>
          <option value="denied">{t("text.decisionDenied")}</option>
        </select>
      </label>
      <label>
        <span>{t("form.caller")}</span>
        <select value={filters.callerAgentId ?? ""} onChange={(event) => onChange({ ...filters, callerAgentId: event.target.value })}>
          <option value="">{t("form.anyCaller")}</option>
          {agents.map((agent) => <option key={agent.id} value={agent.id}>{permissionEntityDisplayName(agent.name, t)}</option>)}
        </select>
      </label>
      <label>
        <span>{t("form.target")}</span>
        <select value={filters.targetAgentId ?? ""} onChange={(event) => onChange({ ...filters, targetAgentId: event.target.value })}>
          <option value="">{t("form.anyTarget")}</option>
          {agents.map((agent) => <option key={agent.id} value={agent.id}>{permissionEntityDisplayName(agent.name, t)}</option>)}
        </select>
      </label>
      <button className="secondary-button" type="button" onClick={onRefresh}><RefreshCw size={14} /> {t("action.refresh")}</button>
      <details className="trace-filter-advanced">
        <summary>{t("text.filterSettings")}</summary>
        <label>
          <span>{t("form.traceRunId")}</span>
          <input placeholder={t("form.traceRunPlaceholder")} value={filters.runId ?? ""} onChange={(event) => onChange({ ...filters, runId: event.target.value })} />
        </label>
      </details>
    </div>
  );
}

function FormFooter({
  message,
  submitLabel,
  submitting,
  submittingLabel
}: {
  message: string;
  submitLabel: string;
  submitting: boolean;
  submittingLabel: string;
}) {
  return (
    <div aria-busy={submitting || undefined} className="form-footer">
      <button className="primary-button" disabled={submitting} type="submit">
        {submitting ? submittingLabel : submitLabel}
      </button>
      {message ? <span>{message}</span> : null}
    </div>
  );
}

function ResourceActionContextStrip({
  context,
  t
}: {
  context?: ResourceLifecycleActionContext | null;
  t: Translator;
}) {
  if (!context) return null;

  return (
    <div className="resource-action-context">
      <span className="section-kicker">{t("resource.actionContext.title")}</span>
      <div>
        <strong>{context.resourceName}</strong>
        <small>{t(context.resourceKindKey)}</small>
      </div>
      <p>{t("resource.actionContext.scope")}: {context.tenantName} / {context.workspaceName}</p>
    </div>
  );
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}
