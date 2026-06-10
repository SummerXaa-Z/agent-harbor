import { useMemo, type FormEvent } from "react";
import { CheckCircle2, FileSearch, LockKeyhole, RefreshCw } from "lucide-react";

import {
  countInvalidAccessProfileRows,
  countInvalidGrantRows,
  scopeStatusTone,
  summarizeDataScopes
} from "../accessProfile";
import {
  accessDecisionEvidenceTone,
  accessDecisionOutcomeLabel,
  accessDecisionOutcomeTone,
  agentNameMap,
  formatDate,
  permissionEntityDisplayName,
  policyEffectLabel,
  type Translator
} from "../consolePresenters";
import type {
  AccessDecisionExplainResult,
  AccessProfileFilters,
  AccessProfileSummary,
  Agent,
  Capability,
  ManagementScope,
  TenantAccessProfileData,
  TenantAccessProfileGrant,
  TenantAccessProfileInstance,
  TenantAccessProfileWorkspace
} from "../types";
import { ApprovalDropdown } from "./ApprovalDropdown";
import { Badge, EmptyRow } from "./ui";

const emptyAccessProfileSummary: AccessProfileSummary = {
  tenantCount: 0,
  grantCount: 0,
  targetCount: 0,
  capabilityCount: 0,
  workspaceAssignmentCount: 0,
  instanceAssignmentCount: 0,
  recentAllowedTraceCount: 0,
  recentDeniedTraceCount: 0
};

export function TenantAccessProfileView({
  agents,
  capabilities,
  explanation,
  explanationLoading,
  explanationMessage,
  filters,
  loading,
  message,
  onChange,
  onExplainAccessDecision,
  onRefresh,
  onTenantChange,
  profile,
  scope,
  t
}: {
  agents: Agent[];
  capabilities: Capability[];
  explanation: AccessDecisionExplainResult | null;
  explanationLoading: boolean;
  explanationMessage: string;
  filters: AccessProfileFilters;
  loading: boolean;
  message: string;
  onChange: (filters: AccessProfileFilters) => void;
  onExplainAccessDecision: () => void;
  onRefresh: () => void;
  onTenantChange: (tenantId: string) => void;
  profile: TenantAccessProfileData | null;
  scope: ManagementScope;
  t: Translator;
}) {
  const names = useMemo(() => agentNameMap(agents), [agents]);
  const targetAgents = agents.filter((agent) => agent.channelType !== "local");
  const visibleCapabilities = filters.targetId
    ? capabilities.filter((capability) => capability.targetId === filters.targetId)
    : capabilities;
  const sourceLabel = profile
    ? profile.loadedFromApi
      ? t("status.sourceLive")
      : t("status.sourceFallback")
    : t("status.sourceNotLoaded");
  const profileSummary = profile?.summary ?? emptyAccessProfileSummary;
  const profileScopeTenants = profile?.scopeTenants ?? [];
  const profileGrants = profile?.grants ?? [];
  const profileRecentTraces = profile?.recentTraces ?? [];
  const workspaceLabel = filters.workspaceId?.trim() || t("form.workspaceAll");
  const targetLabel = filters.targetId ? names[filters.targetId] ?? filters.targetId : t("form.anyTarget");
  const capabilityLabel = filters.capabilityId?.trim() || t("form.anyCapability");
  const callerLabel = filters.callerInstanceId ? names[filters.callerInstanceId] ?? filters.callerInstanceId : t("form.anyCaller");
  const subjectLabel = filters.subjectId?.trim() || t("text.notSpecified");
  const traceLimitLabel = String(filters.traceLimit ?? 20);
  const targetDropdownOptions = [
    { value: "", label: t("form.anyTarget") },
    ...targetAgents.map((agent) => ({ value: agent.id, label: permissionEntityDisplayName(agent.name, t) }))
  ];
  const capabilityDropdownOptions = [
    { value: "", label: t("form.anyCapability") },
    ...visibleCapabilities.map((capability) => ({ value: capability.id, label: capability.key }))
  ];
  const callerDropdownOptions = [
    { value: "", label: t("form.anyCaller") },
    ...agents.map((agent) => ({ value: agent.id, label: permissionEntityDisplayName(agent.name, t) }))
  ];

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onRefresh();
  }

  return (
    <div className="access-profile">
      <section className="access-hero">
        <div>
          <span>{t("section.accessProfileTask")}</span>
          <h2>{t("text.accessProfileTaskTitle")}</h2>
          <p>{t("text.accessProfileTaskBody")}</p>
        </div>
        <div className="access-hero-status">
          <span>{sourceLabel}</span>
          <strong>{profile ? `${t("status.generated")} ${formatDate(profile.generatedAt)}` : t("empty.accessProfile.title")}</strong>
          {message ? <p>{message}</p> : <p>{profile ? t("text.accessProfileLoadedDetail") : t("empty.accessProfile.detail")}</p>}
          <button className="primary-button" disabled={loading} onClick={onRefresh} type="button">
            <RefreshCw size={14} />
            {loading ? t("action.loading") : t("action.loadProfile")}
          </button>
        </div>
      </section>

      <section className="access-selected-scope" aria-label={t("text.selectedScope")}>
        <div>
          <span>{t("form.tenant")}</span>
          <strong>{profile?.tenant.name ?? scope.tenantId}</strong>
        </div>
        <div>
          <span>{t("form.workspace")}</span>
          <strong>{workspaceLabel}</strong>
        </div>
        <div>
          <span>{t("form.caller")}</span>
          <strong>{callerLabel}</strong>
        </div>
        <div>
          <span>{t("form.target")}</span>
          <strong>{targetLabel}</strong>
        </div>
      </section>

      <form className="access-query-panel" onSubmit={submit}>
        <header>
          <div>
            <strong>{t("section.accessProfileFilters")}</strong>
            <p>{t("text.accessProfileFiltersDetail")}</p>
          </div>
          <span>{t("form.traces")}: {traceLimitLabel}</span>
        </header>
        <div className="access-query-grid">
          <label className="access-field">
            <span>{t("form.tenant")}</span>
            <input name="tenantId" value={scope.tenantId} onChange={(event) => onTenantChange(event.target.value)} />
          </label>
          <label className="access-field">
            <span>{t("form.workspace")}</span>
            <input
              name="workspaceId"
              placeholder={t("form.workspaceAll")}
              value={filters.workspaceId ?? ""}
              onChange={(event) => onChange({ ...filters, workspaceId: event.target.value })}
            />
          </label>
          <label className="access-field">
            <span>{t("form.target")}</span>
            <ApprovalDropdown
              label={t("form.target")}
              options={targetDropdownOptions}
              value={filters.targetId ?? ""}
              onChange={(value) => onChange({ ...filters, capabilityId: "", targetId: value })}
            />
          </label>
          <label className="access-field">
            <span>{t("form.capability")}</span>
            <ApprovalDropdown
              label={t("form.capability")}
              options={capabilityDropdownOptions}
              value={filters.capabilityId ?? ""}
              onChange={(value) => onChange({ ...filters, capabilityId: value })}
            />
          </label>
          <label className="access-field">
            <span>{t("form.caller")}</span>
            <ApprovalDropdown
              label={t("form.caller")}
              options={callerDropdownOptions}
              value={filters.callerInstanceId ?? ""}
              onChange={(value) => onChange({ ...filters, callerInstanceId: value })}
            />
          </label>
          <label className="access-field">
            <span>{t("form.subjectId")}</span>
            <input
              name="subjectId"
              placeholder={t("detail.subjectId")}
              value={filters.subjectId ?? ""}
              onChange={(event) => onChange({ ...filters, subjectId: event.target.value })}
            />
          </label>
          <label className="access-field is-compact">
            <span>{t("form.traces")}</span>
            <input
              inputMode="numeric"
              max={100}
              min={0}
              name="traceLimit"
              type="number"
              value={String(filters.traceLimit ?? "")}
              onChange={(event) => onChange({ ...filters, traceLimit: event.target.value })}
            />
          </label>
          <div className="access-query-actions">
            <button className="secondary-button" disabled={loading} type="submit">
              <RefreshCw size={14} />
              {loading ? t("action.loading") : t("action.loadProfile")}
            </button>
            <button className="secondary-button" disabled={explanationLoading} onClick={onExplainAccessDecision} type="button">
              <FileSearch size={14} />
              {explanationLoading ? t("action.loading") : t("action.explainAccessDecision")}
            </button>
          </div>
        </div>
        <div className="access-query-summary">
          <span>{capabilityLabel}</span>
          <span>{subjectLabel}</span>
          <span>{targetLabel}</span>
        </div>
      </form>

      <AccessDecisionExplainPanel
        explanation={explanation}
        loading={explanationLoading}
        message={explanationMessage}
        t={t}
      />

      {!profile ? (
        <EmptyRow title={t("empty.accessProfile.title")} detail={t("empty.accessProfile.detail")} />
      ) : (
        <>
          <div className="access-summary-grid">
            <AccessSummaryCell label={t("summary.tenantScope")} value={String(profileSummary.tenantCount ?? profileScopeTenants.length)} detail={profile.tenant.name} />
            <AccessSummaryCell label={t("summary.grants")} value={String(profileSummary.grantCount ?? profileGrants.length)} detail={`${profileSummary.capabilityCount ?? 0} ${t("detail.capabilities")}`} />
            <AccessSummaryCell label={t("summary.assignments")} value={`${profileSummary.workspaceAssignmentCount ?? 0}/${profileSummary.instanceAssignmentCount ?? 0}`} detail={t("text.workspaceCaller")} />
            <AccessSummaryCell label={t("summary.recentDecisions")} value={`${profileSummary.recentAllowedTraceCount ?? 0}/${profileSummary.recentDeniedTraceCount ?? 0}`} detail={t("text.allowedDenied")} />
          </div>

          <div className="access-tenant-list" aria-label={t("summary.tenantScope")}>
            {profileScopeTenants.map((tenant) => (
              <div className="access-tenant-row" key={tenant.id}>
                <Badge tone={tenant.status === "active" ? "success" : "neutral"}>L{tenant.level}</Badge>
                <div>
                  <strong>{tenant.name}</strong>
                  <span>{tenant.id}{tenant.parentTenantId ? ` · ${t("text.parentTenant")} ${tenant.parentTenantId}` : ""}</span>
                </div>
              </div>
            ))}
          </div>

          <div className="access-layout">
            <section className="access-grant-chain" aria-label={t("section.effectiveGrantChain")}>
              <header>
                <strong>{t("section.effectiveGrantChain")}</strong>
                <span>{countInvalidAccessProfileRows(profile)} {t("text.invalidRows")}</span>
              </header>
              {profileGrants.length === 0 ? (
                <EmptyRow title={t("empty.grantChains.title")} detail={t("empty.grantChains.detail")} />
              ) : null}
              {profileGrants.map((grant) => (
                <AccessGrantRow grant={grant} key={grant.tenantEntitlement.id} t={t} />
              ))}
            </section>

            <section className="access-trace-evidence" aria-label={t("section.traceEvidence")}>
              <header>
                <strong>{t("section.traceEvidence")}</strong>
                <span>{profileRecentTraces.length} {t("text.recentTraces")}</span>
              </header>
              {profileRecentTraces.length === 0 ? (
                <EmptyRow title={t("empty.traceEvidence.title")} detail={t("empty.traceEvidence.detail")} />
              ) : null}
              {profileRecentTraces.map((trace) => (
                <article className="access-trace-row" key={trace.id}>
                  <div className={`trace-decision tone-${trace.decision === "allowed" ? "success" : "danger"}`}>
                    {trace.decision === "allowed" ? <CheckCircle2 size={15} /> : <LockKeyhole size={15} />}
                  </div>
                  <div>
                    <div className="trace-title-line">
                      <strong>{names[trace.callerAgentId ?? ""] ?? trace.callerAgentId ?? t("text.traceAnonymous")} → {names[trace.targetAgentId] ?? trace.targetAgentId}</strong>
                      <Badge tone={trace.decision === "allowed" ? "success" : "danger"}>
                        {trace.decision === "allowed" ? t("text.decisionAllowed") : t("text.decisionDenied")}
                      </Badge>
                    </div>
                    <span>{trace.capabilityId ?? `${trace.routeType}:${trace.routeKey || t("text.traceDefaultRoute")}`} · {summarizeDataScopes(trace.dataScopes)} · {trace.reason || policyEffectLabel(trace.decision === "allowed" ? "allow" : "deny", t)}</span>
                  </div>
                  <time>{formatDate(trace.createdAt)}</time>
                </article>
              ))}
            </section>
          </div>
        </>
      )}
    </div>
  );
}

function AccessSummaryCell({ label, value, detail }: { label: string; value: string; detail: string }) {
  return (
    <div className="access-summary-cell">
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{detail}</small>
    </div>
  );
}

function AccessDecisionExplainPanel({
  explanation,
  loading,
  message,
  t
}: {
  explanation: AccessDecisionExplainResult | null;
  loading: boolean;
  message: string;
  t: Translator;
}) {
  const dataScopes = explanation?.dataScopes ?? explanation?.decision.dataScopes ?? [];
  return (
    <section className="access-decision-explain">
      <header>
        <div>
          <strong>{t("section.accessDecisionExplain")}</strong>
          {loading ? <span>{t("action.loading")}</span> : message ? <span>{message}</span> : null}
        </div>
      </header>
      {!explanation ? (
        <EmptyRow title={t("section.accessDecisionExplain")} detail={t("empty.accessDecisionExplain.detail")} />
      ) : (
        <>
          <div className="access-decision-summary">
            <Badge tone={accessDecisionOutcomeTone(explanation.outcome)}>
              {accessDecisionOutcomeLabel(explanation.outcome, t)}
            </Badge>
            <div>
              <strong>{explanation.summary}</strong>
              <span>{explanation.decision.source} · {explanation.decision.reason}</span>
            </div>
          </div>
          <div className="access-decision-evidence">
            {explanation.evidence.map((row) => (
              <article key={`${row.layer}:${row.id ?? row.status}`}>
                <Badge tone={accessDecisionEvidenceTone(row.status)}>{row.status}</Badge>
                <div>
                  <strong>{row.layer}</strong>
                  <span>{row.message}</span>
                  {row.id ? <code>{row.id}</code> : null}
                </div>
              </article>
            ))}
          </div>
          <div className="access-decision-footer">
            <div>
              <strong>{t("detail.dataScopes")}</strong>
              <span>{summarizeDataScopes(dataScopes)}</span>
            </div>
            <div>
              <strong>{t("text.nextActions")}</strong>
              <ul>
                {explanation.nextActions.map((action) => (
                  <li key={action}>{action}</li>
                ))}
              </ul>
            </div>
          </div>
        </>
      )}
    </section>
  );
}

function AccessGrantRow({ grant, t }: { grant: TenantAccessProfileGrant; t: Translator }) {
  const invalidRows = countInvalidGrantRows(grant);
  return (
    <article className={invalidRows > 0 ? "access-grant-row invalid" : "access-grant-row"}>
      <div className="access-grant-header">
        <div>
          <strong>{grant.capability?.displayName ?? grant.capability?.key ?? grant.tenantEntitlement.capabilityId}</strong>
          <span>{grant.target?.name ?? grant.tenantEntitlement.targetId}</span>
        </div>
        <div className="access-badge-group">
          <Badge tone={scopeStatusTone(grant.scopeStatus)}>{grant.scopeStatus}</Badge>
          <Badge tone={grant.tenantEntitlement.effect === "allow" ? "success" : "danger"}>{policyEffectLabel(grant.tenantEntitlement.effect, t)}</Badge>
        </div>
      </div>
      <div className="access-scope-line">
        <span>{t("text.tenantEntitlement")}</span>
        <code>{grant.tenantEntitlement.id}</code>
        <span>{summarizeDataScopes(grant.effectiveTenantDataScopes)}</span>
      </div>
      {grant.scopeReason ? <p className="access-invalid-reason">{grant.scopeReason}</p> : null}
      <div className="access-nested-list">
        {grant.workspaceAssignments.length === 0 ? (
          <EmptyRow title={t("empty.workspaceAssignments.title")} detail={t("empty.workspaceAssignments.detail")} />
        ) : null}
        {grant.workspaceAssignments.map((workspace) => (
          <AccessWorkspaceRow key={workspace.workspaceAssignment.id} workspace={workspace} t={t} />
        ))}
      </div>
    </article>
  );
}

function AccessWorkspaceRow({ workspace, t }: { workspace: TenantAccessProfileWorkspace; t: Translator }) {
  return (
    <div className="access-workspace-row">
      <div className="access-row-main">
        <div>
          <strong>{workspace.workspaceAssignment.workspaceId}</strong>
          <span>{summarizeDataScopes(workspace.effectiveWorkspaceDataScopes)}</span>
        </div>
        <Badge tone={scopeStatusTone(workspace.scopeStatus)}>{workspace.scopeStatus}</Badge>
      </div>
      {workspace.scopeReason ? <p className="access-invalid-reason">{workspace.scopeReason}</p> : null}
      <div className="access-instance-list">
        {workspace.instanceAssignments.length === 0 ? (
          <span className="access-empty-inline">{t("empty.callerInstances.title")}</span>
        ) : null}
        {workspace.instanceAssignments.map((instance) => (
          <AccessInstanceRow instance={instance} key={instance.instanceAssignment.id} t={t} />
        ))}
      </div>
    </div>
  );
}

function AccessInstanceRow({ instance, t }: { instance: TenantAccessProfileInstance; t: Translator }) {
  return (
    <div className="access-instance-row">
      <div>
        <strong>{instance.callerInstance?.name ?? instance.instanceAssignment.callerInstanceId}</strong>
        <span>{instance.instanceAssignment.subjectSelector || t("text.subjectsAll")} · {summarizeDataScopes(instance.effectiveInstanceDataScopes)}</span>
      </div>
      <Badge tone={scopeStatusTone(instance.scopeStatus)}>{instance.scopeStatus}</Badge>
      {instance.scopeReason ? <p className="access-invalid-reason">{instance.scopeReason}</p> : null}
    </div>
  );
}
