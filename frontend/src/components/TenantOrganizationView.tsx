import { useEffect, useMemo, useRef, useState, type CSSProperties, type FormEvent, type ReactNode } from "react";
import { ArrowRight, Building2, GitBranch, LockKeyhole, Network, ShieldCheck, UserRoundCheck, UsersRound, X } from "lucide-react";

import {
  accessSubjectOptionForIdFrom,
  accessSubjectsForWorkspace,
  type AccessSubjectKind,
  type AccessSubjectOption
} from "../accessSubjects";
import { permissionEntityDisplayName, readableIdentifierLabel, type Translator } from "../consolePresenters";
import {
  buildTenantOrganizationModel,
  type TenantOrganizationSelection
} from "../tenantOrganization";
import type {
  Agent,
  Capability,
  InstanceAssignment,
  PermissionChangeHandoffContext,
  Tenant,
  TenantEntitlement,
  WorkspaceAssignment
} from "../types";
import { Panel } from "./ConsolePrimitives";
import { ApprovalDropdown } from "./ApprovalDropdown";
import { Badge, EmptyRow } from "./ui";

interface TenantOrganizationViewProps {
  accessSubjects: AccessSubjectOption[];
  agents: Agent[];
  capabilities: Capability[];
  instanceAssignments: InstanceAssignment[];
  onOpenAccessProfile: (context: TenantWorkspaceContext) => void;
  onStartPermissionChange: (context: PermissionChangeHandoffContext) => void;
  t: Translator;
  tenantEntitlements: TenantEntitlement[];
  tenants: Tenant[];
  workspaceAssignments: WorkspaceAssignment[];
}

export interface TenantWorkspaceContext {
  tenantId: string;
  tenantName: string;
  tenantPath: string;
  workspaceId: string;
  workspaceName: string;
}

interface TenantPermissionForm {
  accessSubjectId: string;
  callerInstanceId: string;
  targetId: string;
}

export function TenantOrganizationView({
  accessSubjects,
  agents,
  capabilities,
  instanceAssignments,
  onOpenAccessProfile,
  onStartPermissionChange,
  t,
  tenantEntitlements,
  tenants,
  workspaceAssignments
}: TenantOrganizationViewProps) {
  const [selectedTenantId, setSelectedTenantId] = useState("");
  const [permissionModalOpen, setPermissionModalOpen] = useState(false);
  const [permissionForm, setPermissionForm] = useState<TenantPermissionForm>({
    accessSubjectId: "",
    callerInstanceId: "",
    targetId: ""
  });
  const closePermissionModalRef = useRef<HTMLButtonElement>(null);
  const model = useMemo(
    () => buildTenantOrganizationModel({
      agents,
      capabilities,
      instanceAssignments,
      selectedTenantId,
      tenantEntitlements,
      tenants,
      workspaceAssignments
    }),
    [agents, capabilities, instanceAssignments, selectedTenantId, tenantEntitlements, tenants, workspaceAssignments]
  );

  useEffect(() => {
    if (model.selectedTenantId && model.selectedTenantId !== selectedTenantId) {
      setSelectedTenantId(model.selectedTenantId);
    }
  }, [model.selectedTenantId, selectedTenantId]);

  const context = model.selected ? tenantWorkspaceContext(model.selected, t) : null;
  const permissionDefaults = model.selected && context
    ? tenantPermissionDefaults(model.selected, agents, context.workspaceId, t)
    : { callerInstanceId: "", callerName: "", targetId: "", targetName: "" };
  const accessSubjectCatalog = accessSubjectsForWorkspace(accessSubjects, context?.workspaceId ?? "");
  const defaultAccessSubjectId = accessSubjectCatalog[0]?.id ?? "";
  const callerAgents = model.selected && context ? tenantPermissionCallerAgents(model.selected, agents, context.workspaceId) : [];
  const targetAgents = model.selected && context ? tenantPermissionTargetAgents(model.selected, agents, context.workspaceId) : [];
  const callerOptions = agentDropdownOptions(callerAgents, permissionForm.callerInstanceId, t, t("ask.emptyOption.caller"));
  const targetOptions = agentDropdownOptions(targetAgents, permissionForm.targetId, t, t("ask.emptyOption.target"));
  const selectedAccessSubject = accessSubjectOptionForIdFrom(accessSubjectCatalog, permissionForm.accessSubjectId)
    ?? accessSubjectCatalog[0];
  const accessSubjectOptions = accessSubjectCatalog.map((option) => ({
    label: `${t(accessSubjectKindLabelKey(option.kind))} · ${t(option.labelKey)}`,
    value: option.id
  }));
  const accessSubjectGroups = accessSubjectDirectoryGroups(accessSubjectCatalog);
  const modalSubmitDisabled = !context || !permissionForm.callerInstanceId || !permissionForm.targetId || !selectedAccessSubject;

  useEffect(() => {
    if (permissionModalOpen) return;
    setPermissionForm({
      accessSubjectId: defaultAccessSubjectId,
      callerInstanceId: permissionDefaults.callerInstanceId,
      targetId: permissionDefaults.targetId
    });
  }, [
    defaultAccessSubjectId,
    model.selected?.tenant.id,
    permissionDefaults.callerInstanceId,
    permissionDefaults.targetId,
    permissionModalOpen
  ]);

  useEffect(() => {
    if (!permissionModalOpen) return;

    const activeElement = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const previousOverflow = document.body.style.overflow;
    const focusTimer = window.setTimeout(() => closePermissionModalRef.current?.focus(), 0);
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setPermissionModalOpen(false);
    };

    document.body.style.overflow = "hidden";
    document.addEventListener("keydown", onKeyDown);

    return () => {
      window.clearTimeout(focusTimer);
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", onKeyDown);
      activeElement?.focus();
    };
  }, [permissionModalOpen]);

  if (!model.selected || !context) {
    return (
      <section className="content-grid">
        <Panel className="span-12" icon={<Building2 size={18} />} title={t("panel.tenantOrganization")}>
          <EmptyRow
            actionHash="#getting-started"
            actionLabel={t("empty.tenants.action")}
            detail={t("empty.tenants.detail")}
            title={t("empty.tenants.title")}
          />
        </Panel>
      </section>
    );
  }

  const openPermissionModal = () => {
    setPermissionForm({
      accessSubjectId: permissionForm.accessSubjectId || defaultAccessSubjectId,
      callerInstanceId: permissionForm.callerInstanceId || permissionDefaults.callerInstanceId,
      targetId: permissionForm.targetId || permissionDefaults.targetId
    });
    setPermissionModalOpen(true);
  };

  const submitPermissionModal = (event: FormEvent) => {
    event.preventDefault();
    if (modalSubmitDisabled || !selectedAccessSubject) return;

    const selectedCaller = agents.find((agent) => agent.id === permissionForm.callerInstanceId);
    const selectedTarget = agents.find((agent) => agent.id === permissionForm.targetId);
    setPermissionModalOpen(false);
    onStartPermissionChange({
      ...context,
      callerInstanceId: permissionForm.callerInstanceId,
      callerName: selectedCaller ? permissionEntityDisplayName(selectedCaller.name, t) : permissionDefaults.callerName,
      intentText: t("tenantOrg.permissionIntent"),
      sourceView: "tenants",
      subjectId: selectedAccessSubject.subjectSelector,
      targetId: permissionForm.targetId,
      targetName: selectedTarget ? permissionEntityDisplayName(selectedTarget.name, t) : permissionDefaults.targetName
    });
  };

  return (
    <section className="tenant-organization content-grid">
      <Panel className="span-4" icon={<GitBranch size={18} />} title={t("panel.tenantTree")}>
        <div className="tenant-tree-summary">
          <div>
            <span>{t("tenantOrg.totalTenants")}</span>
            <strong>{model.totals.tenantCount}</strong>
          </div>
          <div>
            <span>{t("tenantOrg.activeTenants")}</span>
            <strong>{model.totals.activeTenantCount}</strong>
          </div>
        </div>
        <div className="tenant-tree-list" role="list">
          {model.flatNodes.map((node) => {
            const selected = node.tenant.id === model.selectedTenantId;
            return (
              <button
                aria-current={selected ? "true" : undefined}
                className={`tenant-tree-row ${selected ? "is-selected" : ""}`}
                key={node.tenant.id}
                onClick={() => setSelectedTenantId(node.tenant.id)}
                style={{ "--tenant-depth": node.depth } as CSSProperties}
                type="button"
              >
                <span className="tenant-tree-row-main">
                  <strong>{tenantDisplayName(node.tenant, t)}</strong>
                  <small>{t(`text.tenantLevel.${node.tenant.level}`, t("text.tenantLevelFallback"))}</small>
                </span>
                <span className="tenant-tree-row-meta">
                  {node.workspaceCount} {t("detail.workspaces")} · {node.permissionCount} {t("detail.permissions")}
                </span>
              </button>
            );
          })}
        </div>
      </Panel>

      <Panel className="span-8" icon={<Building2 size={18} />} title={t("panel.tenantOrganization")}>
        <div className="tenant-org-hero">
          <div>
            <span className="section-kicker">{t("tenantOrg.selectedTenant")}</span>
            <h2>{context.tenantName}</h2>
            <p>{context.tenantPath}</p>
          </div>
          <Badge tone={model.selected.tenant.status === "active" ? "success" : "neutral"}>
            {model.selected.tenant.status === "active" ? t("status.active") : t("status.agentDisabled")}
          </Badge>
        </div>

        <div className="tenant-org-actions" aria-label={t("tenantOrg.permissionManagement")}>
          <div>
            <strong>{t("tenantOrg.permissionManagement")}</strong>
            <span>{t("tenantOrg.permissionManagementDetail")}</span>
          </div>
          <div>
            <button
              aria-controls="tenant-permission-modal"
              aria-expanded={permissionModalOpen}
              aria-haspopup="dialog"
              className="primary-button"
              onClick={openPermissionModal}
              type="button"
            >
              <ShieldCheck size={15} />
              {t("action.startTenantPermissionChange")}
            </button>
            <button className="secondary-button" onClick={() => onOpenAccessProfile(context)} type="button">
              <LockKeyhole size={15} />
              {t("action.openTenantAccessProfile")}
            </button>
          </div>
        </div>

        {permissionModalOpen ? (
          <div
            className="action-modal-backdrop"
            role="presentation"
            onMouseDown={(event) => {
              if (event.target === event.currentTarget) setPermissionModalOpen(false);
            }}
          >
            <section
              aria-labelledby="tenant-permission-modal-title"
              aria-modal="true"
              className="action-modal-panel tenant-permission-modal"
              id="tenant-permission-modal"
              role="dialog"
            >
              <header className="action-modal-header">
                <div>
                  <ShieldCheck size={16} />
                  <h2 id="tenant-permission-modal-title">{t("tenantOrg.startPermissionModalTitle")}</h2>
                </div>
                <button
                  ref={closePermissionModalRef}
                  aria-label={t("action.close")}
                  className="icon-button compact"
                  title={t("action.close")}
                  type="button"
                  onClick={() => setPermissionModalOpen(false)}
                >
                  <X size={15} />
                </button>
              </header>
              <div className="action-modal-body">
                <form className="tenant-permission-form" onSubmit={submitPermissionModal}>
                  <p>{t("tenantOrg.startPermissionModalDetail")}</p>
                  <div className="tenant-permission-context">
                    <span>
                      <small>{t("form.tenant")}</small>
                      <strong>{context.tenantName}</strong>
                    </span>
                    <span>
                      <small>{t("form.workspace")}</small>
                      <strong>{context.workspaceName}</strong>
                    </span>
                  </div>
                  <div className="tenant-permission-fields">
                    <label>
                      <span className="approval-field-label">{t("form.caller")}</span>
                      <ApprovalDropdown
                        disabled={callerAgents.length === 0}
                        label={t("form.caller")}
                        onChange={(value) => setPermissionForm((current) => ({ ...current, callerInstanceId: value }))}
                        options={callerOptions}
                        value={permissionForm.callerInstanceId}
                      />
                    </label>
                    <label>
                      <span className="approval-field-label">{t("form.target")}</span>
                      <ApprovalDropdown
                        disabled={targetAgents.length === 0}
                        label={t("form.target")}
                        onChange={(value) => setPermissionForm((current) => ({ ...current, targetId: value }))}
                        options={targetOptions}
                        value={permissionForm.targetId}
                      />
                    </label>
                    <label className="is-wide">
                      <span className="approval-field-label">{t("form.accessSubject")}</span>
                      <ApprovalDropdown
                        label={t("form.accessSubject")}
                        onChange={(value) => setPermissionForm((current) => ({ ...current, accessSubjectId: value }))}
                        options={accessSubjectOptions}
                        value={permissionForm.accessSubjectId}
                      />
                    </label>
                  </div>
                  <div className="tenant-permission-preview">
                    <span>{t("tenantOrg.modalPreview")}</span>
                    <strong>{selectedAccessSubject ? t(selectedAccessSubject.labelKey) : "-"}</strong>
                    <small>{selectedAccessSubject ? t(selectedAccessSubject.detailKey) : t("tenantOrg.noAccessSubjects")}</small>
                  </div>
                  <footer className="tenant-permission-footer">
                    <button className="secondary-button" type="button" onClick={() => setPermissionModalOpen(false)}>
                      {t("action.cancel")}
                    </button>
                    <button className="primary-button" disabled={modalSubmitDisabled} type="submit">
                      {t("tenantOrg.startPermissionSubmit")}
                      <ArrowRight size={15} />
                    </button>
                  </footer>
                </form>
              </div>
            </section>
          </div>
        ) : null}

        <div className="tenant-org-metrics">
          <TenantOrgMetric
            icon={<Network size={16} />}
            label={t("tenantOrg.workspaces")}
            value={String(model.selected.workspaces.length)}
            detail={model.selected.workspaces.length > 0 ? context.workspaceName : t("tenantOrg.noWorkspaceMetric")}
          />
          <TenantOrgMetric icon={<UsersRound size={16} />} label={t("tenantOrg.agents")} value={String(model.selected.activeAgentCount)} detail={tx(t, "tenantOrg.agentDetail", { callers: model.selected.callerCount, targets: model.selected.targetCount })} />
          <TenantOrgMetric icon={<ShieldCheck size={16} />} label={t("tenantOrg.permissions")} value={String(model.selected.permissionCount)} detail={tx(t, "tenantOrg.permissionDetail", { allowed: model.selected.allowedPermissionCount, denied: model.selected.deniedPermissionCount })} />
          <TenantOrgMetric icon={<UserRoundCheck size={16} />} label={t("tenantOrg.assignments")} value={`${model.selected.workspaceAssignmentCount}/${model.selected.instanceAssignmentCount}`} detail={t("tenantOrg.assignmentDetail")} />
        </div>

        <section className="tenant-access-directory" aria-label={t("tenantOrg.accessDirectory")}>
          <div>
            <h3>{t("tenantOrg.accessDirectory")}</h3>
            <p>{t("tenantOrg.accessDirectoryDetail")}</p>
          </div>
          <div className="tenant-access-directory-groups">
            {accessSubjectGroups.map((group) => (
              <article key={group.kind}>
                <span>{t(accessSubjectDirectoryLabelKey(group.kind))}</span>
                {group.items.length > 0 ? (
                  <div>
                    {group.items.slice(0, 4).map((option) => (
                      <button
                        className="tenant-access-subject"
                        key={option.id}
                        onClick={() => {
                          setPermissionForm((current) => ({ ...current, accessSubjectId: option.id }));
                          setPermissionModalOpen(true);
                        }}
                        type="button"
                      >
                        <strong>{t(option.labelKey)}</strong>
                        <small>{option.email ?? t(option.detailKey)}</small>
                      </button>
                    ))}
                  </div>
                ) : (
                  <p className="tenant-org-muted">{t("tenantOrg.noAccessSubjects")}</p>
                )}
              </article>
            ))}
          </div>
        </section>

        <div className="tenant-org-detail-grid">
          <section>
            <h3>{t("tenantOrg.workspaceList")}</h3>
            {model.selected.workspaces.length > 0 ? (
              <div className="tenant-org-list">
                {model.selected.workspaces.map((workspace) => (
                  <article key={workspace.workspaceId}>
                    <div>
                      <strong>{workspaceDisplayName(workspace.workspaceId, t)}</strong>
                      <span>{workspace.agentCount} {t("detail.agents")} · {workspace.assignmentCount} {t("detail.workspaceAssignments")}</span>
                    </div>
                    <Badge tone={workspace.assignmentCount > 0 ? "success" : "neutral"}>
                      {workspace.assignmentCount > 0 ? t("status.permissionConfigured") : t("status.unconfigured")}
                    </Badge>
                  </article>
                ))}
              </div>
            ) : (
              <p className="tenant-org-muted">{t("tenantOrg.emptyWorkspaces")}</p>
            )}
          </section>

          <section>
            <h3>{t("tenantOrg.permissionBoundary")}</h3>
            <div className="tenant-org-boundary">
              <span>{t("tenantOrg.boundaryTenant")}</span>
              <strong>{context.tenantName}</strong>
              <span>{t("tenantOrg.boundaryWorkspace")}</span>
              <strong>{context.workspaceName}</strong>
              <span>{t("tenantOrg.boundaryPolicy")}</span>
              <strong>{t("tenantOrg.boundaryPolicyDetail")}</strong>
            </div>
          </section>
        </div>
      </Panel>
    </section>
  );
}

function TenantOrgMetric({
  detail,
  icon,
  label,
  value
}: {
  detail: string;
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <article className="tenant-org-metric">
      <div>{icon}</div>
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{detail}</small>
    </article>
  );
}

function tenantWorkspaceContext(selection: TenantOrganizationSelection, t: Translator): TenantWorkspaceContext {
  const workspaceId = selection.selectedWorkspaceId;
  return {
    tenantId: selection.tenant.id,
    tenantName: tenantDisplayName(selection.tenant, t),
    tenantPath: selection.path.map((tenant) => tenantDisplayName(tenant, t)).join(" / "),
    workspaceId,
    workspaceName: workspaceDisplayName(workspaceId, t)
  };
}

function tenantPermissionDefaults(
  selection: TenantOrganizationSelection,
  agents: Agent[],
  workspaceId: string,
  t: Translator
) {
  const caller = tenantPermissionCallerAgents(selection, agents, workspaceId)[0];
  const target = tenantPermissionTargetAgents(selection, agents, workspaceId)[0];

  return {
    callerInstanceId: caller?.id ?? "",
    callerName: caller ? permissionEntityDisplayName(caller.name, t) : "",
    targetId: target?.id ?? "",
    targetName: target ? permissionEntityDisplayName(target.name, t) : ""
  };
}

function tenantPermissionCallerAgents(
  selection: TenantOrganizationSelection,
  agents: Agent[],
  workspaceId: string
) {
  const tenantAgents = agents.filter((agent) =>
    agent.tenantId === selection.tenant.id
    && agent.channelType === "local"
    && agent.status === "active"
  );
  const preferredAgents = tenantAgents.filter((agent) => !workspaceId || agent.workspaceId === workspaceId);
  return preferredAgents.length > 0 ? preferredAgents : tenantAgents;
}

function tenantPermissionTargetAgents(
  selection: TenantOrganizationSelection,
  agents: Agent[],
  workspaceId: string
) {
  const activeTargets = agents.filter((agent) => agent.channelType !== "local" && agent.status === "active");
  const tenantTargets = activeTargets.filter((agent) => agent.tenantId === selection.tenant.id);
  const tenantWorkspaceTargets = tenantTargets.filter((agent) => !workspaceId || agent.workspaceId === workspaceId);
  if (tenantWorkspaceTargets.length > 0) return tenantWorkspaceTargets;
  if (tenantTargets.length > 0) return tenantTargets;
  const workspaceTargets = activeTargets.filter((agent) => !workspaceId || agent.workspaceId === workspaceId);
  return workspaceTargets.length > 0 ? workspaceTargets : activeTargets;
}

function agentDropdownOptions(agents: Agent[], value: string, t: Translator, emptyLabel: string) {
  const options = agents.map((agent) => ({
    label: permissionEntityDisplayName(agent.name, t),
    value: agent.id
  }));
  if (value && !options.some((option) => option.value === value)) {
    options.push({ label: permissionEntityDisplayName(readableIdentifierLabel(value), t), value });
  }
  return options.length > 0 ? options : [{ label: emptyLabel, value: "" }];
}

function accessSubjectDirectoryGroups(subjects: AccessSubjectOption[]) {
  return (["role", "department", "member"] as AccessSubjectKind[]).map((kind) => ({
    items: subjects.filter((subject) => subject.kind === kind),
    kind
  }));
}

function accessSubjectKindLabelKey(kind: AccessSubjectKind) {
  return `accessSubject.kind.${kind}`;
}

function accessSubjectDirectoryLabelKey(kind: AccessSubjectKind) {
  if (kind === "role") return "tenantOrg.directoryRoles";
  if (kind === "department") return "tenantOrg.directoryDepartments";
  if (kind === "member") return "tenantOrg.directoryMembers";
  return "accessSubject.kind.custom";
}

function tenantDisplayName(tenant: Tenant, t: Translator) {
  return permissionEntityDisplayName(tenant.name || tenant.id, t);
}

function workspaceDisplayName(workspaceId: string, t: Translator) {
  if (!workspaceId) return t("form.workspaceAll");
  if (workspaceId === "workspace-sandbox") return t("text.defaultWorkspaceName");
  if (/permission[-_]?(request|package)[-_]?approval/i.test(workspaceId)) return t("demo.permissionRequestWorkspace");
  if (/core[-_]?journey/i.test(workspaceId)) return t("demo.coreJourneyWorkspace");
  return permissionEntityDisplayName(readableIdentifierLabel(workspaceId), t);
}

function tx(t: Translator, key: string, params: Record<string, string | number>) {
  return Object.entries(params).reduce((text, [name, value]) => text.replaceAll(`{${name}}`, String(value)), t(key));
}
