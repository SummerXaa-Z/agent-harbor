import { useMemo, type FormEvent } from "react";
import { RefreshCw, ShieldCheck } from "lucide-react";

import {
  accessSubjectOptionForSelector,
  accessSubjectOptions,
  customAccessSubjectOption
} from "../accessSubjects";
import {
  agentNameMap,
  capabilityDiscoveryStatusLabel,
  capabilityStatusTone,
  dataScopeText,
  policyEffectLabel,
  riskTone,
  translatedValue,
  type Translator
} from "../consolePresenters";
import type {
  Agent,
  Capability,
  InstanceAssignment,
  TenantEntitlement,
  WorkspaceAssignment
} from "../types";
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
  instanceAssignments,
  message,
  mcpTargets,
  onApprove,
  onChange,
  onCreateGrantChain,
  onRefreshTarget,
  t,
  tenantEntitlements,
  workspaceAssignments
}: {
  actionId: string;
  agents: Agent[];
  capabilities: Capability[];
  form: CapabilityGrantForm;
  instanceAssignments: InstanceAssignment[];
  message: string;
  mcpTargets: Agent[];
  onApprove: (capability: Capability) => void;
  onChange: (form: CapabilityGrantForm) => void;
  onCreateGrantChain: (event: FormEvent<HTMLFormElement>) => void;
  onRefreshTarget: () => void;
  t: Translator;
  tenantEntitlements: TenantEntitlement[];
  workspaceAssignments: WorkspaceAssignment[];
}) {
  const agentNames = useMemo(() => agentNameMap(agents), [agents]);
  const visibleCapabilities = useMemo(() => {
    const targetId = form.targetId.trim();
    return targetId ? capabilities.filter((capability) => capability.targetId === targetId) : capabilities;
  }, [capabilities, form.targetId]);
  const selectedCapability = capabilities.find((capability) => capability.id === form.capabilityId);
  const selectedAccessSubject = accessSubjectOptionForSelector(form.subjectSelector);
  const targetOptions = [
    { value: "", label: t("form.allMcpTargets") },
    ...mcpTargets.map((target) => ({ value: target.id, label: target.name }))
  ];
  const capabilityOptions = [
    { value: "", label: t("form.selectCapability") },
    ...visibleCapabilities.map((capability) => ({
      value: capability.id,
      label: capability.displayName || capability.key
    }))
  ];
  const callerOptions = [
    { value: "", label: t("form.selectCaller") },
    ...agents.map((agent) => ({ value: agent.id, label: agent.name }))
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
    const option = accessSubjectOptions.find((item) => item.id === accessSubjectId);
    if (!option) return;
    onChange({
      ...form,
      subjectSelector: option.subjectSelector
    });
  }

  return (
    <div className="capability-governance">
      <div className="capability-toolbar">
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

      <div className="capability-layout">
        <div className="capability-catalog">
          <div className="table-wrap">
            <table className="capability-table">
              <thead>
                <tr>
                  <th>{t("table.capability")}</th>
                  <th>{t("table.target")}</th>
                  <th>{t("table.action")}</th>
                  <th>{t("table.risk")}</th>
                  <th>{t("table.status")}</th>
                  <th>{t("table.grants")}</th>
                  <th>{t("table.action")}</th>
                </tr>
              </thead>
              <tbody>
                {visibleCapabilities.length === 0 ? (
                  <tr>
                    <td colSpan={7}>
                      <EmptyRow title={t("empty.capabilities.title")} detail={t("empty.capabilities.detail")} />
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
                        <strong>{capability.displayName || capability.key}</strong>
                        <span>{dataScopeText(capability.dataScopes) || capability.description || capability.key}</span>
                      </td>
                      <td>{agentNames[capability.targetId] ?? capability.targetId}</td>
                      <td><Badge tone={capability.action === "delete" || capability.action === "admin" ? "danger" : capability.action === "export" ? "warning" : "info"}>{translatedValue(t, capability.action)}</Badge></td>
                      <td><Badge tone={riskTone(capability.riskLevel)}>{translatedValue(t, capability.riskLevel)}</Badge></td>
                      <td><Badge tone={capabilityStatusTone(capability.discoveryStatus)}>{capabilityDiscoveryStatusLabel(capability.discoveryStatus, t)}</Badge></td>
                      <td>
                        <strong>{entitlementIds.length}/{workspaceIds.length}/{instanceCount}</strong>
                        <span>{t("detail.tenantWorkspaceInstance")}</span>
                      </td>
                      <td>
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
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>

        <form className="control-form capability-grant-form" onSubmit={onCreateGrantChain}>
          <div className="form-row">
            <label>{t("form.tenant")}<input required value={form.tenantId} onChange={(event) => onChange({ ...form, tenantId: event.target.value })} /></label>
            <label>{t("form.workspace")}<input required value={form.workspaceId} onChange={(event) => onChange({ ...form, workspaceId: event.target.value })} /></label>
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
          {selectedAccessSubject.id === customAccessSubjectOption.id ? (
            <label>{t("form.subjectSelector")}<input placeholder={t("form.subjectSelectorPlaceholder")} value={form.subjectSelector} onChange={(event) => onChange({ ...form, subjectSelector: event.target.value })} /></label>
          ) : null}
          <div className="capability-scope-strip">
            <span>{selectedCapability?.sensitivity ?? t("text.sensitivity")}</span>
            <span>{selectedCapability?.riskLevel ?? t("text.risk")}</span>
            <span>{dataScopeText(selectedCapability?.dataScopes) || t("text.noDataScope")}</span>
          </div>
          <FormFooter
            message=""
            submitLabel={actionId === `grant:${form.capabilityId}` ? t("action.loading") : t("action.grantChain")}
          />
        </form>

        <div className="assignment-list">
          {tenantEntitlements.length === 0 ? (
            <EmptyRow title={t("empty.grantChains.title")} detail={t("empty.grantChains.assignmentDetail")} />
          ) : null}
          {tenantEntitlements.map((entitlement) => {
            const capability = capabilities.find((item) => item.id === entitlement.capabilityId);
            const children = workspaceAssignments.filter((item) => item.tenantEntitlementId === entitlement.id);
            const instanceCount = children.reduce(
              (total, item) => total + instanceAssignments.filter((instance) => instance.workspaceAssignmentId === item.id).length,
              0
            );
            return (
              <article className="assignment-row" key={entitlement.id}>
                <div>
                  <strong>{capability?.key ?? entitlement.capabilityId}</strong>
                  <span>{entitlement.tenantId} · {policyEffectLabel(entitlement.effect, t)} · {translatedValue(t, entitlement.status)}</span>
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
