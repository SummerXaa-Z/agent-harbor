import { ClipboardCheck, SlidersHorizontal, X } from "lucide-react";

import type { AccessSubjectOption } from "../accessSubjects";
import { summarizeDataScopes } from "../accessProfile";
import { capabilityDisplayName } from "../consolePresenters";
import { dataScopeValueLabels } from "../consolePresenters";
import {
  permissionRequestStepSectionId,
  permissionPackageTemplateName,
  permissionPackageTemplateSummary,
  type Tone,
  type Translator
} from "../permissionWorkbenchPresenters";
import { applyPermissionRequestAccessSubject } from "../permissionRequestForm";
import type { PermissionPackageDraft, PermissionPackageDraftInput } from "../permissionPackages";
import type { Agent, Capability } from "../types";
import { ApprovalDropdown } from "./ApprovalDropdown";
import { Badge } from "./ui";

type DropdownOption = { label: string; value: string };

export interface PermissionChangeDraftSheetProps {
  accessSubjectCatalog: AccessSubjectOption[];
  accessSubjectDropdownOptions: DropdownOption[];
  callerDropdownOptions: DropdownOption[];
  draft: PermissionPackageDraft;
  form: PermissionPackageDraftInput;
  isActiveLocked: boolean;
  isLocked: boolean;
  isOpen: boolean;
  lockedDetailKey: string;
  onChange: (form: PermissionPackageDraftInput) => void;
  onClose: () => void;
  onOpenDraftSheet: () => void;
  onStartNewPermissionChange: () => void;
  requestFormHelpKey: string;
  requestFormTitleKey: string;
  selectedAccessSubject: AccessSubjectOption;
  selectedCaller: Agent | undefined;
  selectedTarget: Agent | undefined;
  statusLabel: string;
  statusTone: Tone;
  templateDropdownOptions: DropdownOption[];
  targetDropdownOptions: DropdownOption[];
  tenantDropdownOptions: DropdownOption[];
  tenantPath: { path: string; primary: string };
  t: Translator;
  workspaceName: string;
}

export function PermissionChangeDraftSheet({
  accessSubjectCatalog,
  accessSubjectDropdownOptions,
  callerDropdownOptions,
  draft,
  form,
  isActiveLocked,
  isLocked,
  isOpen,
  lockedDetailKey,
  onChange,
  onClose,
  onOpenDraftSheet,
  onStartNewPermissionChange,
  requestFormHelpKey,
  requestFormTitleKey,
  selectedAccessSubject,
  selectedCaller,
  selectedTarget,
  statusLabel,
  statusTone,
  templateDropdownOptions,
  targetDropdownOptions,
  tenantDropdownOptions,
  tenantPath,
  t,
  workspaceName
}: PermissionChangeDraftSheetProps) {
  const dataScopeLabels = dataScopeValueLabels(t);
  return (
    <>
      <button className="permission-draft-command" id={permissionRequestStepSectionId("scope")} onClick={onOpenDraftSheet} type="button">
        <span className="permission-draft-command-icon">
          <SlidersHorizontal size={16} />
        </span>
        <span className="permission-draft-command-copy">
          <strong>{t("text.permissionDraftCommandTitle")}</strong>
          <small>{t("text.permissionDraftCommandDetail")}</small>
        </span>
        <span className="permission-draft-command-context">
          <em>{tenantPath.primary}</em>
          <em>{permissionPackageTemplateName(draft.template, t)}</em>
        </span>
        <span className="permission-draft-command-action">
          {isLocked ? t("action.reviewPermissionChangeDraft") : t("action.editPermissionChangeDraft")}
        </span>
      </button>

      {isOpen ? (
        <div className="permission-draft-sheet-backdrop" role="presentation">
          <section className="permission-draft-sheet" aria-label={t("text.permissionDraftSheetTitle")} role="dialog" aria-modal="true">
            <header className="permission-draft-sheet-header">
              <div>
                <span>{t(requestFormTitleKey)}</span>
                <h3>{t("text.permissionDraftSheetTitle")}</h3>
                <p>{t("text.permissionDraftSheetDetail")}</p>
              </div>
              <div className="permission-draft-sheet-header-actions">
                <Badge tone={statusTone}>{statusLabel}</Badge>
                <button className="icon-button" aria-label={t("action.closePermissionChangeDraft")} onClick={onClose} type="button">
                  <X aria-hidden="true" size={16} />
                </button>
              </div>
            </header>

            <div className="permission-draft-sheet-body">
              <p className="permission-draft-sheet-help">{t(requestFormHelpKey)}</p>
              {isLocked ? (
                <div className="approval-lock-notice">
                  <div>
                    <strong>{t("text.permissionRequestLockedTitle")}</strong>
                    <span>{t(lockedDetailKey)}</span>
                  </div>
                  {isActiveLocked ? (
                    <button className="secondary-button" onClick={onStartNewPermissionChange} type="button">
                      <ClipboardCheck size={14} />
                      {t("action.startPermissionApproval")}
                    </button>
                  ) : null}
                </div>
              ) : null}

              <details className="approval-concept-guide">
                <summary>{t("section.permissionConceptGuide")}</summary>
                <div className="approval-concept-grid">
                  <article>
                    <strong>{t("concept.tenant")}</strong>
                    <span>{t("concept.tenant.detail")}</span>
                  </article>
                  <article>
                    <strong>{t("concept.caller")}</strong>
                    <span>{t("concept.caller.detail")}</span>
                  </article>
                  <article>
                    <strong>{t("concept.permissionPackage")}</strong>
                    <span>{t("concept.permissionPackage.detail")}</span>
                  </article>
                  <article>
                    <strong>{t("concept.acceptanceMaterials")}</strong>
                    <span>{t("concept.acceptanceMaterials.detail")}</span>
                  </article>
                </div>
              </details>

              <label className="approval-request">
                {t("form.adminRequest")}
                <textarea
                  disabled={isLocked}
                  rows={3}
                  value={form.requestText}
                  onChange={(event) => onChange({ ...form, requestText: event.target.value })}
                />
              </label>
              <div className="approval-form-grid">
                <div className="approval-field is-wide">
                  <span className="approval-field-label">{t("form.businessTenant")}</span>
                  <ApprovalDropdown
                    disabled={isLocked}
                    label={t("form.businessTenant")}
                    options={tenantDropdownOptions}
                    value={form.tenantId}
                    onChange={(value) => onChange({ ...form, tenantId: value })}
                  />
                  <span>{tenantPath.path}</span>
                </div>
                <div className="approval-readonly-field is-wide">
                  <span>{t("form.businessWorkspace")}</span>
                  <strong>{workspaceName}</strong>
                  <small>{t("text.workspaceResolvedDetail")}</small>
                </div>
                <div className="approval-field">
                  <span className="approval-field-label">{t("form.businessCaller")}</span>
                  <ApprovalDropdown
                    disabled={isLocked}
                    label={t("form.businessCaller")}
                    options={callerDropdownOptions}
                    value={form.callerInstanceId}
                    onChange={(value) => onChange({ ...form, callerInstanceId: value })}
                  />
                </div>
                <div className="approval-field">
                  <span className="approval-field-label">{t("form.target")}</span>
                  <ApprovalDropdown
                    disabled={isLocked}
                    label={t("form.target")}
                    options={targetDropdownOptions}
                    value={form.targetId}
                    onChange={(value) => onChange({ ...form, targetId: value })}
                  />
                </div>
                <div className="approval-field approval-subject-field is-wide">
                  <span className="approval-field-label">{t("form.accessSubject")}</span>
                  <ApprovalDropdown
                    disabled={isLocked}
                    label={t("form.accessSubject")}
                    options={accessSubjectDropdownOptions}
                    value={selectedAccessSubject.id}
                    onChange={(value) => onChange(applyPermissionRequestAccessSubject(form, accessSubjectCatalog, value))}
                  />
                  <small>{t(selectedAccessSubject.detailKey)}</small>
                </div>
                <label>
                  {t("form.region")}
                  <input disabled={isLocked} value={form.region} onChange={(event) => onChange({ ...form, region: event.target.value })} />
                </label>
                <div className="approval-select is-wide">
                  <span className="approval-field-label">{t("form.permissionPackage")}</span>
                  <ApprovalDropdown
                    disabled={isLocked}
                    label={t("form.permissionPackage")}
                    options={templateDropdownOptions}
                    value={form.templateId}
                    onChange={(value) => onChange({ ...form, templateId: value })}
                  />
                </div>
              </div>
              <div className="approval-package-preview" id={permissionRequestStepSectionId("template")}>
                <div>
                  <span>{t("section.permissionWizardTemplate")}</span>
                  <strong>{permissionPackageTemplateName(draft.template, t)}</strong>
                  <p>{permissionPackageTemplateSummary(draft.template, t)}</p>
                </div>
                <div className="approval-capability-columns">
                  <CapabilityChipList
                    capabilities={draft.allowedCapabilities}
                    emptyLabel={t("empty.permissionAllowed.detail")}
                    label={t("section.allowedByPackage")}
                    tone="success"
                    t={t}
                  />
                  <CapabilityChipList
                    capabilities={draft.blockedCapabilities}
                    emptyLabel={t("empty.permissionBlocked.detail")}
                    label={t("section.blockedByPackage")}
                    tone="danger"
                    t={t}
                  />
                </div>
                <div className="approval-scope">
                  <span>{t("section.dataScope")}</span>
                  <code>{summarizeDataScopes(draft.dataScopes, t("text.noDataScope"), dataScopeLabels)}</code>
                </div>
              </div>
              <details className="approval-details">
                <summary>{t("text.technicalOverrides")}</summary>
                <label>
                  {t("form.workspaceId")}
                  <input disabled={isLocked} value={form.workspaceId} onChange={(event) => onChange({ ...form, workspaceId: event.target.value })} />
                </label>
                <label>
                  {t("form.subjectSelector")}
                  <input
                    disabled={isLocked}
                    placeholder={t("form.subjectSelectorPlaceholder")}
                    value={form.subjectSelector ?? ""}
                    onChange={(event) => onChange({ ...form, subjectSelector: event.target.value })}
                  />
                  <small>{t("text.subjectSelectorAdvancedHelp")}</small>
                </label>
                <dl>
                  <div>
                    <dt>{t("form.tenantId")}</dt>
                    <dd>{form.tenantId || "-"}</dd>
                  </div>
                  <div>
                    <dt>{t("form.workspaceId")}</dt>
                    <dd>{form.workspaceId || "-"}</dd>
                  </div>
                  <div>
                    <dt>{t("form.callerInstance")}</dt>
                    <dd>{selectedCaller?.id || form.callerInstanceId || "-"}</dd>
                  </div>
                  <div>
                    <dt>{t("form.target")}</dt>
                    <dd>{selectedTarget?.id || form.targetId || "-"}</dd>
                  </div>
                </dl>
              </details>
            </div>

            <footer className="permission-draft-sheet-footer">
              <button className="secondary-button" onClick={onClose} type="button">
                <X aria-hidden="true" size={14} />
                {t("action.closePermissionChangeDraft")}
              </button>
            </footer>
          </section>
        </div>
      ) : null}
    </>
  );
}

export function CapabilityChipList({
  capabilities,
  emptyLabel,
  label,
  tone,
  t
}: {
  capabilities: Capability[];
  emptyLabel: string;
  label: string;
  tone: Tone;
  t: Translator;
}) {
  return (
    <div className={`approval-capability-list tone-${tone}`}>
      <strong>{label}</strong>
      {capabilities.length === 0 ? <span>{emptyLabel}</span> : null}
      <div>
        {capabilities.map((capability) => (
          <span key={capability.id}>
            {capabilityDisplayName(capability, t)} · {t(`value.${capability.action}`, capability.action)}
          </span>
        ))}
      </div>
    </div>
  );
}
