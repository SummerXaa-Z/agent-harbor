import { useMemo, useState, type FormEvent } from "react";
import { FileSearch, RefreshCw, ShieldCheck, X } from "lucide-react";

import {
  accessSubjectOptionForId,
  accessSubjectOptionForSelector,
  accessSubjectOptions,
  customAccessSubjectOption
} from "../accessSubjects";
import {
  capabilityDisplayName,
  capabilityDiscoveryStatusLabel,
  capabilitySummaryText,
  capabilityStatusTone,
  dataScopeText,
  permissionEntityDisplayName,
  policyEffectLabel,
  readableIdentifierLabel,
  riskTone,
  translatedValue,
  type Translator
} from "../consolePresenters";
import type {
  Agent,
  AskHandoffContext,
  Capability,
  CapabilityGovernanceHandoffContext,
  InstanceAssignment,
  Tenant,
  TenantEntitlement,
  WorkspaceAssignment
} from "../types";
import { tx } from "../localizedMessages";
import { ApprovalDropdown } from "./ApprovalDropdown";
import { Badge, EmptyRow } from "./ui";

export interface CapabilityGrantForm {
  callerInstanceId: string;
  capabilityId: string;
  subjectSelector: string;
  targetId: string;
  tenantId: string;
  workspaceId: string;
}

export function CapabilityGovernanceView({
  actionId,
  agents,
  capabilities,
  form,
  handoffContext,
  instanceAssignments,
  message,
  mcpTargets,
  onApprove,
  onChange,
  onCreateGrantChain,
  onDismissHandoff,
  onQueryAccess,
  onRefreshTarget,
  t,
  tenants,
  tenantEntitlements,
  workspaceAssignments
}: {
  actionId: string;
  agents: Agent[];
  capabilities: Capability[];
  form: CapabilityGrantForm;
  handoffContext?: CapabilityGovernanceHandoffContext | null;
  instanceAssignments: InstanceAssignment[];
  message: string;
  mcpTargets: Agent[];
  onApprove: (capability: Capability) => void;
  onChange: (form: CapabilityGrantForm) => void;
  onCreateGrantChain: (event: FormEvent<HTMLFormElement>) => void;
  onDismissHandoff?: () => void;
  onQueryAccess: (context: AskHandoffContext) => void;
  onRefreshTarget: () => void;
  t: Translator;
  tenants: Tenant[];
  tenantEntitlements: TenantEntitlement[];
  workspaceAssignments: WorkspaceAssignment[];
}) {
  const [capabilityQuery, setCapabilityQuery] = useState("");
  const [capabilityStatusFilter, setCapabilityStatusFilter] = useState("");
  const [grantPanelOpen, setGrantPanelOpen] = useState(false);
  const [selectedCapabilityId, setSelectedCapabilityId] = useState("");
  const agentNames = useMemo(
    () => Object.fromEntries(agents.map((agent) => [agent.id, permissionEntityDisplayName(agent.name, t)])),
    [agents, t]
  );
  const tenantNames = useMemo(
    () => new Map(tenants.map((tenant) => [tenant.id, permissionEntityDisplayName(tenant.name, t)])),
    [tenants, t]
  );
  const targetCapabilities = useMemo(() => {
    const targetId = form.targetId.trim();
    return targetId ? capabilities.filter((capability) => capability.targetId === targetId) : capabilities;
  }, [capabilities, form.targetId]);
  const hasTargetCapabilities = targetCapabilities.length > 0;
  const visibleCapabilities = useMemo(() => {
    const query = capabilityQuery.trim().toLowerCase();
    return targetCapabilities.filter((capability) => {
      const searchable = [
        capabilityDisplayName(capability, t),
        capabilitySummaryText(capability, t),
        agentNames[capability.targetId] ?? capability.targetId,
        translatedValue(t, capability.action),
        translatedValue(t, capability.riskLevel),
        capabilityDiscoveryStatusLabel(capability.discoveryStatus, t),
        dataScopeText(capability.dataScopes, t)
      ].join(" ").toLowerCase();
      return (
        (!capabilityStatusFilter || capability.discoveryStatus === capabilityStatusFilter) &&
        (!query || searchable.includes(query))
      );
    });
  }, [agentNames, capabilityQuery, capabilityStatusFilter, targetCapabilities, t]);
  const selectedCapability = capabilities.find((capability) => capability.id === form.capabilityId);
  const selectedCatalogCapability = capabilities.find((capability) => capability.id === selectedCapabilityId) ?? null;
  const selectedAccessSubject = accessSubjectOptionForSelector(form.subjectSelector);
  const currentTargetLabel = form.targetId ? agentNames[form.targetId] ?? form.targetId : t("form.allMcpTargets");
  const capabilityEmptyActionLabel = targetCapabilities.length > 0
    ? undefined
    : mcpTargets.length === 0
      ? t("empty.capabilities.actionRegisterAgents")
      : form.targetId
        ? t("empty.capabilities.actionRefresh")
        : undefined;
  const capabilityEmptyActionHash = targetCapabilities.length === 0 && mcpTargets.length === 0 ? "#registry" : undefined;
  const capabilityEmptyAction = targetCapabilities.length === 0 && mcpTargets.length > 0 && form.targetId ? onRefreshTarget : undefined;
  const tenantOptions = [
    ...tenants.map((tenant) => ({ value: tenant.id, label: permissionEntityDisplayName(tenant.name, t) })),
    ...(form.tenantId && !tenants.some((tenant) => tenant.id === form.tenantId)
      ? [{ value: form.tenantId, label: permissionEntityDisplayName(form.tenantId, t) }]
      : [])
  ];
  const workspaceOptions = Array.from(
    new Set([
      form.workspaceId,
      ...agents.map((agent) => agent.workspaceId)
    ].filter(Boolean))
  ).map((workspaceId) => ({
    value: workspaceId,
    label: capabilityWorkspaceDisplayName(workspaceId, agents, t)
  }));
  const targetOptions = [
    { value: "", label: t("form.allMcpTargets") },
    ...mcpTargets.map((target) => ({ value: target.id, label: permissionEntityDisplayName(target.name, t) }))
  ];
  const capabilityStatusOptions = [
    { value: "", label: t("form.anyStatus") },
    { value: "pending_review", label: t("status.capabilityPendingReview") },
    { value: "approved", label: t("status.capabilityApproved") },
    { value: "deprecated", label: t("status.capabilityDeprecated") },
    { value: "removed", label: t("status.capabilityRemoved") }
  ];
  const capabilityOptions = [
    { value: "", label: t("form.selectCapability") },
    ...targetCapabilities.map((capability) => ({
      value: capability.id,
      label: capabilityDisplayName(capability, t)
    }))
  ];
  const callerOptions = [
    { value: "", label: t("form.selectCaller") },
    ...agents.map((agent) => ({ value: agent.id, label: permissionEntityDisplayName(agent.name, t) }))
  ];
  const accessSubjectDropdownOptions = [
    ...accessSubjectOptions.map((option) => ({
      value: option.id,
      label: `${t(`accessSubject.kind.${option.kind}`)} · ${t(option.labelKey)}`
    })),
    {
      value: customAccessSubjectOption.id,
      label: `${t("accessSubject.kind.custom")} · ${t(customAccessSubjectOption.labelKey)}`
    }
  ];
  const entitlementIdsByCapability = useMemo(() => {
    return tenantEntitlements.reduce<Record<string, string[]>>((acc, entitlement) => {
      acc[entitlement.capabilityId] = [...(acc[entitlement.capabilityId] ?? []), entitlement.id];
      return acc;
    }, {});
  }, [tenantEntitlements]);
  const workspaceIdsByEntitlement = useMemo(() => {
    return workspaceAssignments.reduce<Record<string, string[]>>((acc, assignment) => {
      acc[assignment.tenantEntitlementId] = [...(acc[assignment.tenantEntitlementId] ?? []), assignment.id];
      return acc;
    }, {});
  }, [workspaceAssignments]);
  const instancesByWorkspaceAssignment = useMemo(() => {
    return instanceAssignments.reduce<Record<string, InstanceAssignment[]>>((acc, assignment) => {
      acc[assignment.workspaceAssignmentId] = [...(acc[assignment.workspaceAssignmentId] ?? []), assignment];
      return acc;
    }, {});
  }, [instanceAssignments]);

  function handleTargetChange(targetId: string) {
    const nextCapability = capabilities.find((capability) => capability.targetId === targetId);
    onChange({
      ...form,
      capabilityId: nextCapability?.id ?? "",
      targetId
    });
  }

  function handleCapabilityChange(capabilityId: string) {
    const capability = capabilities.find((item) => item.id === capabilityId);
    onChange({
      ...form,
      capabilityId,
      targetId: capability?.targetId ?? form.targetId
    });
  }

  function handleAccessSubjectChange(accessSubjectId: string) {
    const option = accessSubjectOptionForId(accessSubjectId);
    if (!option) return;
    onChange({
      ...form,
      subjectSelector: option.subjectSelector
    });
  }

  function openGrantPanel(capability?: Capability) {
    if (capability) {
      onChange({
        ...form,
        capabilityId: capability.id,
        targetId: capability.targetId
      });
    }
    setGrantPanelOpen(true);
  }

  return (
    <div className="capability-governance">
      {handoffContext ? (
        <section className="capability-handoff-notice" role="status" aria-live="polite">
          <FileSearch size={16} />
          <div>
            <strong>{t("text.capabilityHandoffTitle")}</strong>
            <span>
              {tx(t, "text.capabilityHandoffDetail", {
                target: handoffContext.targetName ?? handoffContext.targetId,
                tenant: handoffContext.tenantName ?? handoffContext.tenantId,
                workspace: handoffContext.workspaceName ?? handoffContext.workspaceId
              })}
            </span>
          </div>
          {onDismissHandoff ? (
            <button className="secondary-button" onClick={onDismissHandoff} type="button">
              <X aria-hidden="true" size={14} />
              {t("action.dismiss")}
            </button>
          ) : null}
        </section>
      ) : null}

      <div className="capability-scope-bar">
        <div className="capability-scope-copy">
          <span className="section-kicker">{t("section.currentCapabilityScope")}</span>
          <strong>{currentTargetLabel}</strong>
          <p>{t("text.currentCapabilityScopeDetail")}</p>
        </div>
        <div className="capability-scope-controls">
          <label>
            {t("form.target")}
            <ApprovalDropdown
              label={t("form.target")}
              options={targetOptions}
              value={form.targetId}
              onChange={handleTargetChange}
            />
          </label>
          <button
            className="secondary-button"
            disabled={!form.targetId || actionId === `refresh:${form.targetId}`}
            onClick={onRefreshTarget}
            type="button"
          >
            <RefreshCw size={14} />
            {actionId === `refresh:${form.targetId}` ? t("action.loading") : t("action.refresh")}
          </button>
          {message ? <span className="capability-message">{message}</span> : null}
        </div>
      </div>

      <div className="capability-layout">
        <div className="capability-catalog">
          <div className="capability-catalog-heading">
            <div>
              <span className="section-kicker">{t("section.capabilityCatalog")}</span>
              <h3>{currentTargetLabel}</h3>
              <p>{t("text.capabilityCatalogHelp")}</p>
            </div>
            <div className="capability-catalog-actions">
              <span>{visibleCapabilities.length}/{targetCapabilities.length}</span>
              {hasTargetCapabilities ? (
                <button className="primary-button capability-grant-launcher" onClick={() => openGrantPanel()} type="button">
                  <ShieldCheck size={14} />
                  {t("action.grantChain")}
                </button>
              ) : (
                <button
                  className="primary-button capability-refresh-launcher"
                  disabled={!form.targetId || actionId === `refresh:${form.targetId}`}
                  onClick={onRefreshTarget}
                  type="button"
                >
                  <RefreshCw size={14} />
                  {actionId === `refresh:${form.targetId}` ? t("action.loading") : t("empty.capabilities.actionRefresh")}
                </button>
              )}
            </div>
          </div>
          {hasTargetCapabilities ? (
            <>
              <div className="table-toolbar">
                <label>
                  <span>{t("form.capability")}</span>
                  <input
                    placeholder={t("form.searchCapabilities")}
                    value={capabilityQuery}
                    onChange={(event) => setCapabilityQuery(event.target.value)}
                  />
                </label>
                <label>
                  <span>{t("table.status")}</span>
                  <ApprovalDropdown
                    label={t("table.status")}
                    options={capabilityStatusOptions}
                    value={capabilityStatusFilter}
                    onChange={setCapabilityStatusFilter}
                  />
                </label>
              </div>
              <div className="table-wrap">
                <table className="capability-table">
                  <thead>
                    <tr>
                      <th>{t("table.capability")}</th>
                      <th>{t("table.target")}</th>
                      <th>{t("table.governance")}</th>
                      <th>{t("table.grants")}</th>
                      <th>{t("table.action")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {visibleCapabilities.length === 0 ? (
                      <tr>
                        <td colSpan={5}>
                          <EmptyRow
                            title={t("empty.filteredResults.title")}
                            detail={t("empty.filteredResults.detail")}
                          />
                        </td>
                      </tr>
                    ) : null}
                    {visibleCapabilities.map((capability) => {
                      const entitlementIds = entitlementIdsByCapability[capability.id] ?? [];
                      const workspaceIds = entitlementIds.flatMap((id) => workspaceIdsByEntitlement[id] ?? []);
                      const instanceCount = workspaceIds.reduce(
                        (total, id) => total + (instancesByWorkspaceAssignment[id]?.length ?? 0),
                        0
                      );
                      return (
                        <tr key={capability.id}>
                          <td>
                            <strong>{capabilityDisplayName(capability, t)}</strong>
                            <span>{capabilitySummaryText(capability, t)}</span>
                          </td>
                          <td>{agentNames[capability.targetId] ?? capability.targetId}</td>
                          <td>
                            <div className="capability-meta-stack">
                              <Badge tone={capability.action === "delete" || capability.action === "admin" ? "danger" : capability.action === "export" ? "warning" : "info"}>{translatedValue(t, capability.action)}</Badge>
                              <Badge tone={riskTone(capability.riskLevel)}>{translatedValue(t, capability.riskLevel)}</Badge>
                              <Badge tone={capabilityStatusTone(capability.discoveryStatus)}>{capabilityDiscoveryStatusLabel(capability.discoveryStatus, t)}</Badge>
                            </div>
                          </td>
                          <td>
                            <strong>{entitlementIds.length}/{workspaceIds.length}/{instanceCount}</strong>
                            <span>{t("detail.tenantWorkspaceInstance")}</span>
                          </td>
                          <td>
                            <div className="table-action-group">
                              <button
                                className="table-action"
                                onClick={() => setSelectedCapabilityId(capability.id)}
                                type="button"
                              >
                                <FileSearch size={13} />
                                {t("action.viewDetails")}
                              </button>
                              {capability.discoveryStatus === "approved" ? (
                                <span className="muted-action">{t("status.capabilityApproved")}</span>
                              ) : (
                                <button
                                  className="table-action"
                                  disabled={actionId === capability.id}
                                  onClick={() => onApprove(capability)}
                                  type="button"
                                >
                                  <ShieldCheck size={13} />
                                  {actionId === capability.id ? t("action.approving") : t("action.approve")}
                                </button>
                              )}
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </>
          ) : (
            <div className="capability-empty-state">
              <EmptyRow
                title={t("empty.capabilities.title")}
                detail={t("empty.capabilities.detail")}
                actionLabel={capabilityEmptyActionLabel}
                actionHash={capabilityEmptyActionHash}
                onAction={capabilityEmptyAction}
              />
            </div>
          )}
          {selectedCatalogCapability ? (
            <aside className="capability-detail-panel">
              <div>
                <span className="section-kicker">{t("text.capabilityDetails")}</span>
                <h3>{capabilityDisplayName(selectedCatalogCapability, t)}</h3>
                <p>{capabilitySummaryText(selectedCatalogCapability, t)}</p>
              </div>
              <div className="table-detail-grid">
                <span>{t("table.target")}<strong>{agentNames[selectedCatalogCapability.targetId] ?? selectedCatalogCapability.targetId}</strong></span>
                <span>{t("table.action")}<strong>{translatedValue(t, selectedCatalogCapability.action)}</strong></span>
                <span>{t("table.risk")}<strong>{translatedValue(t, selectedCatalogCapability.riskLevel)}</strong></span>
                <span>{t("table.status")}<strong>{capabilityDiscoveryStatusLabel(selectedCatalogCapability.discoveryStatus, t)}</strong></span>
                <span>{t("section.dataScope")}<strong>{dataScopeText(selectedCatalogCapability.dataScopes, t) || t("text.noDataScope")}</strong></span>
                <span>{t("text.technicalDetails")}<strong>{selectedCatalogCapability.key}</strong></span>
              </div>
              <button
                className="secondary-button table-detail-action"
                onClick={() => openGrantPanel(selectedCatalogCapability)}
                type="button"
              >
                <ShieldCheck size={14} />
                {t("action.grantChain")}
              </button>
              <button
                className="secondary-button table-detail-action"
                onClick={() => onQueryAccess({
                  capabilityId: selectedCatalogCapability.id,
                  sourceView: "capabilities",
                  targetId: selectedCatalogCapability.targetId,
                  tenantId: form.tenantId,
                  workspaceId: form.workspaceId
                })}
                type="button"
              >
                <FileSearch size={14} />
                {t("action.queryCapabilityAccess")}
              </button>
            </aside>
          ) : null}
        </div>

        <div className="assignment-list">
          <div className="assignment-list-heading">
            <span className="section-kicker">{t("section.existingGrantChains")}</span>
            <strong>{tenantEntitlements.length} {t("table.grants")}</strong>
          </div>
          {tenantEntitlements.length === 0 ? (
            <EmptyRow
              title={t("empty.grantChains.title")}
              detail={t("empty.grantChains.assignmentDetail")}
              actionLabel={t("empty.grantChains.action")}
              actionHash="#ai-admin"
            />
          ) : null}
          {tenantEntitlements.map((entitlement) => {
            const capability = capabilities.find((item) => item.id === entitlement.capabilityId);
            const children = workspaceAssignments.filter((item) => item.tenantEntitlementId === entitlement.id);
            const instanceCount = children.reduce(
              (total, item) => total + instanceAssignments.filter((instance) => instance.workspaceAssignmentId === item.id).length,
              0
            );
            const tenantName = tenantNames.get(entitlement.tenantId) ?? entitlement.tenantId;
            return (
              <article className="assignment-row" key={entitlement.id}>
                <div>
                  <strong>{capability ? capabilityDisplayName(capability, t) : entitlement.capabilityId}</strong>
                  <span>{tenantName} · {policyEffectLabel(entitlement.effect, t)} · {translatedValue(t, entitlement.status)}</span>
                </div>
                <div className="assignment-metrics">
                  <span>{children.length} {t("text.workspaces")}</span>
                  <span>{instanceCount} {t("text.callers")}</span>
                </div>
              </article>
            );
          })}
        </div>
      </div>
      {grantPanelOpen ? (
        <div className="capability-grant-overlay" onClick={() => setGrantPanelOpen(false)} role="presentation">
          <aside
            aria-labelledby="capability-grant-title"
            className="capability-grant-sheet"
            onClick={(event) => event.stopPropagation()}
            role="dialog"
          >
            <div className="capability-grant-sheet-header">
              <div>
                <span className="section-kicker">{t("section.capabilityGrant")}</span>
                <h3 id="capability-grant-title">{t("action.grantChain")}</h3>
                <p>{t("text.capabilityGrantHelp")}</p>
              </div>
              <button aria-label={t("action.dismiss")} className="icon-button compact" onClick={() => setGrantPanelOpen(false)} type="button">
                <X aria-hidden="true" size={15} />
              </button>
            </div>
            <form className="control-form capability-grant-form" onSubmit={onCreateGrantChain}>
              <div className="form-row">
                <label>
                  {t("form.businessTenant")}
                  <ApprovalDropdown
                    label={t("form.businessTenant")}
                    options={tenantOptions}
                    value={form.tenantId}
                    onChange={(value) => onChange({ ...form, tenantId: value })}
                  />
                </label>
                <label>
                  {t("form.businessWorkspace")}
                  <ApprovalDropdown
                    label={t("form.businessWorkspace")}
                    options={workspaceOptions}
                    value={form.workspaceId}
                    onChange={(value) => onChange({ ...form, workspaceId: value })}
                  />
                </label>
              </div>
              <label>
                {t("form.capability")}
                <ApprovalDropdown
                  label={t("form.capability")}
                  options={capabilityOptions}
                  value={form.capabilityId}
                  onChange={handleCapabilityChange}
                />
              </label>
              <div className="form-row">
                <label>
                  {t("form.callerInstance")}
                  <ApprovalDropdown
                    label={t("form.callerInstance")}
                    options={callerOptions}
                    value={form.callerInstanceId}
                    onChange={(value) => onChange({ ...form, callerInstanceId: value })}
                  />
                </label>
                <label>
                  {t("form.accessSubject")}
                  <ApprovalDropdown
                    label={t("form.accessSubject")}
                    options={accessSubjectDropdownOptions}
                    value={selectedAccessSubject.id}
                    onChange={handleAccessSubjectChange}
                  />
                </label>
              </div>
              <details className="capability-grant-advanced" open={selectedAccessSubject.id === customAccessSubjectOption.id}>
                <summary>{t("text.technicalOverrides")}</summary>
                <label>{t("form.subjectSelector")}<input placeholder={t("form.subjectSelectorPlaceholder")} value={form.subjectSelector} onChange={(event) => onChange({ ...form, subjectSelector: event.target.value })} /></label>
              </details>
              <div className="capability-scope-strip">
                <span>{selectedCapability ? translatedValue(t, selectedCapability.sensitivity) : t("text.sensitivity")}</span>
                <span>{selectedCapability ? translatedValue(t, selectedCapability.riskLevel) : t("text.risk")}</span>
                <span>{dataScopeText(selectedCapability?.dataScopes, t) || t("text.noDataScope")}</span>
              </div>
              <FormFooter
                message=""
                submitLabel={actionId === `grant:${form.capabilityId}` ? t("action.loading") : t("action.grantChain")}
              />
            </form>
          </aside>
        </div>
      ) : null}
    </div>
  );
}

function FormFooter({ message, submitLabel }: { message: string; submitLabel: string }) {
  return (
    <div className="form-footer">
      <button className="primary-button" type="submit">{submitLabel}</button>
      {message ? <span>{message}</span> : null}
    </div>
  );
}

function capabilityWorkspaceDisplayName(workspaceId: string, agents: Agent[], t: Translator) {
  const normalized = workspaceId.trim();
  if (!normalized) return t("form.workspace");
  if (normalized === "workspace-sandbox") return t("text.defaultWorkspaceName");
  const agent = agents.find((item) => item.workspaceId === normalized);
  if (agent?.workspaceId === "workspace-sandbox") return t("text.defaultWorkspaceName");
  return t(`workspace.${normalized}.name`, readableIdentifierLabel(normalized));
}
