import { useMemo, type FormEvent } from "react";
import { CheckCircle2, FileSearch, LockKeyhole, RefreshCw } from "lucide-react";

import { accessSubjectOptionForSelector } from "../accessSubjects";
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
  accessTraceReasonLabel,
  agentNameMap,
  capabilityDisplayName,
  dataScopeValueLabels,
  formatDate,
  permissionEntityDisplayName,
  policyEffectLabel,
  type Translator
} from "../consolePresenters";
import type {
  AccessDecisionExplainResult,
  AccessProfileFilters,
  AccessProfileHandoffContext,
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
import { TechnicalId } from "./TechnicalId";
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
  handoffContext,
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
  handoffContext?: AccessProfileHandoffContext | null;
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
  const selectedCapability = capabilities.find((capability) => capability.id === filters.capabilityId);
  const sourceLabel = profile
    ? profile.loadedFromApi
      ? t("status.sourceLive")
      : t("status.sourceFallback")
    : t("status.sourceNotLoaded");
  const profileSummary = profile?.summary ?? emptyAccessProfileSummary;
  const profileScopeTenants = profile?.scopeTenants ?? [];
  const profileGrants = profile?.grants ?? [];
  const profileRecentTraces = profile?.recentTraces ?? [];
  const tenantNameById = useMemo(
    () => new Map(profileScopeTenants.map((tenant) => [tenant.id, permissionEntityDisplayName(tenant.name, t)])),
    [profileScopeTenants, t]
  );
  const capabilityNameById = useMemo(
    () => new Map(capabilities.map((capability) => [capability.id, capabilityDisplayName(capability, t)])),
    [capabilities, t]
  );
  const dataScopeLabels = useMemo(() => dataScopeValueLabels(t), [t]);
  const tenantLabel = handoffContext?.tenantName
    ?? (profile?.tenant ? permissionEntityDisplayName(profile.tenant.name || profile.tenant.id, t) : permissionEntityDisplayName(scope.tenantId, t));
  const workspaceLabel = handoffContext?.workspaceName ?? (filters.workspaceId?.trim() || t("form.workspaceAll"));
  const targetLabel = handoffContext?.targetName ?? (filters.targetId ? names[filters.targetId] ?? filters.targetId : t("form.anyTarget"));
  const capabilityLabel = handoffContext?.capabilityName ?? (selectedCapability ? capabilityDisplayName(selectedCapability, t) : filters.capabilityId?.trim() || t("form.anyCapability"));
  const callerLabel = handoffContext?.callerName ?? (filters.callerInstanceId ? names[filters.callerInstanceId] ?? filters.callerInstanceId : t("form.anyCaller"));
  const traceLimitLabel = String(filters.traceLimit ?? 20);
  const targetDropdownOptions = [
    { value: "", label: t("form.anyTarget") },
    ...targetAgents.map((agent) => ({ value: agent.id, label: permissionEntityDisplayName(agent.name, t) }))
  ];
  const capabilityDropdownOptions = [
    { value: "", label: t("form.anyCapability") },
    ...visibleCapabilities.map((capability) => ({ value: capability.id, label: capabilityDisplayName(capability, t) }))
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

      {handoffContext ? (
        <section className="access-handoff-context" aria-label={t("text.accessProfileHandoffContext")}>
          <CheckCircle2 size={16} />
          <div>
            <strong>{t("text.accessProfileHandoffTitle")}</strong>
            <span>{t("text.accessProfileHandoffDetail")}</span>
          </div>
        </section>
      ) : null}

      <section className="access-selected-scope" aria-label={t("text.selectedScope")}>
        <div>
          <span>{t("form.tenant")}</span>
          <strong>{tenantLabel}</strong>
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
        <details className="access-advanced-filters" open={!handoffContext}>
          <summary>{t("text.technicalDetails")}</summary>
          <div className="access-query-grid is-technical">
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
          </div>
        </details>
        <div className="access-query-summary">
          <span>{capabilityLabel}</span>
          <span>{callerLabel}</span>
          <span>{targetLabel}</span>
        </div>
      </form>

      <AccessDecisionExplainPanel
        dataScopeLabels={dataScopeLabels}
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
            <AccessSummaryCell label={t("summary.tenantScope")} value={String(profileSummary.tenantCount ?? profileScopeTenants.length)} detail={permissionEntityDisplayName(profile.tenant.name || profile.tenant.id, t)} />
            <AccessSummaryCell label={t("summary.grants")} value={String(profileSummary.grantCount ?? profileGrants.length)} detail={`${profileSummary.capabilityCount ?? 0} ${t("detail.capabilities")}`} />
            <AccessSummaryCell label={t("summary.assignments")} value={`${profileSummary.workspaceAssignmentCount ?? 0}/${profileSummary.instanceAssignmentCount ?? 0}`} detail={t("text.workspaceCaller")} />
            <AccessSummaryCell label={t("summary.recentDecisions")} value={`${profileSummary.recentAllowedTraceCount ?? 0}/${profileSummary.recentDeniedTraceCount ?? 0}`} detail={t("text.allowedDenied")} />
          </div>

          <div className="access-tenant-list" aria-label={t("summary.tenantScope")}>
            {profileScopeTenants.map((tenant) => {
              const tenantName = permissionEntityDisplayName(tenant.name || tenant.id, t);
              const parentTenantName = tenant.parentTenantId ? tenantNameById.get(tenant.parentTenantId) ?? t("text.parentTenantOutsideScope") : "";

              return (
                <div className="access-tenant-row" key={tenant.id}>
                  <Badge tone={tenant.status === "active" ? "success" : "neutral"}>{tenantLevelLabel(tenant.level, t)}</Badge>
                  <div>
                    <strong>{tenantName}</strong>
                    <span>{tenant.parentTenantId ? `${t("text.parentTenant")} ${parentTenantName}` : t("text.rootTenant")}</span>
                  </div>
                  <details className="access-tenant-technical">
                    <summary>{t("text.technicalDetails")}</summary>
                    <div className="access-tenant-technical-grid">
                      <TechnicalId label={t("form.tenantId")} value={tenant.id} />
                      {tenant.parentTenantId ? <TechnicalId label={t("text.parentTenant")} value={tenant.parentTenantId} /> : null}
                    </div>
                  </details>
                </div>
              );
            })}
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
                <AccessGrantRow dataScopeLabels={dataScopeLabels} grant={grant} key={grant.tenantEntitlement.id} t={t} />
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
              {profileRecentTraces.map((trace) => {
                const capabilityName = trace.capabilityId
                  ? capabilityNameById.get(trace.capabilityId) ?? trace.capabilityId
                  : `${trace.routeType}:${trace.routeKey || t("text.traceDefaultRoute")}`;
                const traceCallerName = trace.callerAgentId
                  ? permissionEntityDisplayName(names[trace.callerAgentId] ?? trace.callerAgentId, t)
                  : t("text.traceAnonymous");
                const traceTargetName = permissionEntityDisplayName(names[trace.targetAgentId] ?? trace.targetAgentId, t);

                return (
                  <article className="access-trace-row" key={trace.id}>
                    <div className={`trace-decision tone-${trace.decision === "allowed" ? "success" : "danger"}`}>
                      {trace.decision === "allowed" ? <CheckCircle2 size={15} /> : <LockKeyhole size={15} />}
                    </div>
                    <div>
                      <div className="trace-title-line">
                        <strong>{traceCallerName} → {traceTargetName}</strong>
                        <Badge tone={trace.decision === "allowed" ? "success" : "danger"}>
                          {trace.decision === "allowed" ? t("text.decisionAllowed") : t("text.decisionDenied")}
                        </Badge>
                      </div>
                      <span>{capabilityName} · {summarizeDataScopes(trace.dataScopes, t("text.noDataScope"), dataScopeLabels)} · {accessTraceReasonLabel(trace.reason, trace.decision === "allowed" ? "allow" : "deny", t)}</span>
                    </div>
                    <time>{formatDate(trace.createdAt)}</time>
                  </article>
                );
              })}
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

function tenantLevelLabel(level: number, t: Translator) {
  return t(`text.tenantLevel.${level}`, `L${level}`);
}

function AccessDecisionExplainPanel({
  dataScopeLabels,
  explanation,
  loading,
  message,
  t
}: {
  dataScopeLabels: Record<string, string>;
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
              <span>{summarizeDataScopes(dataScopes, t("text.noDataScope"), dataScopeLabels)}</span>
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

function AccessGrantRow({
  dataScopeLabels,
  grant,
  t
}: {
  dataScopeLabels: Record<string, string>;
  grant: TenantAccessProfileGrant;
  t: Translator;
}) {
  const invalidRows = countInvalidGrantRows(grant);
  const capabilityName = grant.capability
    ? capabilityDisplayName(grant.capability, t)
    : grant.tenantEntitlement.capabilityId;
  const targetName = grant.target ? permissionEntityDisplayName(grant.target.name, t) : t("text.unknownTarget");
  return (
    <article className={invalidRows > 0 ? "access-grant-row invalid" : "access-grant-row"}>
      <div className="access-grant-header">
        <div>
          <strong>{capabilityName}</strong>
          <span>{targetName}</span>
        </div>
        <div className="access-badge-group">
          <Badge tone={scopeStatusTone(grant.scopeStatus)}>{grant.scopeStatus}</Badge>
          <Badge tone={grant.tenantEntitlement.effect === "allow" ? "success" : "danger"}>{policyEffectLabel(grant.tenantEntitlement.effect, t)}</Badge>
        </div>
      </div>
      <div className="access-scope-line">
        <span>{t("text.tenantEntitlement")}</span>
        <span>{summarizeDataScopes(grant.effectiveTenantDataScopes, t("text.noDataScope"), dataScopeLabels)}</span>
      </div>
      <details className="access-grant-technical">
        <summary>{t("text.technicalDetails")}</summary>
        <div className="access-technical-grid">
          <TechnicalId label={t("text.tenantEntitlement")} value={grant.tenantEntitlement.id} />
          <TechnicalId label={t("form.target")} value={grant.tenantEntitlement.targetId} />
          <TechnicalId label={t("form.capability")} value={grant.tenantEntitlement.capabilityId} />
        </div>
      </details>
      {grant.scopeReason ? <p className="access-invalid-reason">{grant.scopeReason}</p> : null}
      <div className="access-nested-list">
        {grant.workspaceAssignments.length === 0 ? (
          <EmptyRow title={t("empty.workspaceAssignments.title")} detail={t("empty.workspaceAssignments.detail")} />
        ) : null}
        {grant.workspaceAssignments.map((workspace) => (
          <AccessWorkspaceRow dataScopeLabels={dataScopeLabels} key={workspace.workspaceAssignment.id} workspace={workspace} t={t} />
        ))}
      </div>
    </article>
  );
}

function AccessWorkspaceRow({
  dataScopeLabels,
  workspace,
  t
}: {
  dataScopeLabels: Record<string, string>;
  workspace: TenantAccessProfileWorkspace;
  t: Translator;
}) {
  const workspaceLabel = t("text.workspaceAssignment");
  return (
    <div className="access-workspace-row">
      <div className="access-row-main">
        <div>
          <strong>{workspaceLabel}</strong>
          <span>{summarizeDataScopes(workspace.effectiveWorkspaceDataScopes, t("text.noDataScope"), dataScopeLabels)}</span>
        </div>
        <Badge tone={scopeStatusTone(workspace.scopeStatus)}>{workspace.scopeStatus}</Badge>
      </div>
      <details className="access-workspace-technical">
        <summary>{t("text.technicalDetails")}</summary>
        <div className="access-technical-grid">
          <TechnicalId label={t("text.workspaceAssignment")} value={workspace.workspaceAssignment.id} />
          <TechnicalId label={t("form.workspaceId")} value={workspace.workspaceAssignment.workspaceId} />
          <TechnicalId label={t("text.tenantEntitlement")} value={workspace.workspaceAssignment.tenantEntitlementId} />
        </div>
      </details>
      {workspace.scopeReason ? <p className="access-invalid-reason">{workspace.scopeReason}</p> : null}
      <div className="access-instance-list">
        {workspace.instanceAssignments.length === 0 ? (
          <span className="access-empty-inline">{t("empty.callerInstances.title")}</span>
        ) : null}
        {workspace.instanceAssignments.map((instance) => (
          <AccessInstanceRow dataScopeLabels={dataScopeLabels} instance={instance} key={instance.instanceAssignment.id} t={t} />
        ))}
      </div>
    </div>
  );
}

function AccessInstanceRow({
  dataScopeLabels,
  instance,
  t
}: {
  dataScopeLabels: Record<string, string>;
  instance: TenantAccessProfileInstance;
  t: Translator;
}) {
  const callerName = instance.callerInstance
    ? permissionEntityDisplayName(instance.callerInstance.name, t)
    : permissionEntityDisplayName(instance.instanceAssignment.callerInstanceId, t);
  const subjectLabel = subjectSelectorDisplayName(instance.instanceAssignment.subjectSelector, t);

  return (
    <div className="access-instance-row">
      <div className="access-instance-main">
        <strong>{callerName}</strong>
        <span>{subjectLabel} · {summarizeDataScopes(instance.effectiveInstanceDataScopes, t("text.noDataScope"), dataScopeLabels)}</span>
      </div>
      <Badge tone={scopeStatusTone(instance.scopeStatus)}>{instance.scopeStatus}</Badge>
      <details className="access-instance-technical">
        <summary>{t("text.technicalDetails")}</summary>
        <div className="access-technical-grid">
          <TechnicalId label={t("text.instanceAssignment")} value={instance.instanceAssignment.id} />
          <TechnicalId label={t("form.callerInstance")} value={instance.instanceAssignment.callerInstanceId} />
          <TechnicalId label={t("form.subjectSelector")} value={instance.instanceAssignment.subjectSelector} />
        </div>
      </details>
      {instance.scopeReason ? <p className="access-invalid-reason">{instance.scopeReason}</p> : null}
    </div>
  );
}

function subjectSelectorDisplayName(subjectSelector: string | undefined, t: Translator) {
  if (!subjectSelector?.trim()) return t("text.subjectsAll");
  const option = accessSubjectOptionForSelector(subjectSelector);
  return option.id === "custom" ? t("text.customSubjectScope") : t(option.labelKey);
}
