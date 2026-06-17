import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";
import { Copy, KeyRound, Plus, RefreshCw, RotateCcw, ShieldCheck, Trash2, X } from "lucide-react";

import {
  adminIdentityRoleKey,
  adminIdentityScopeText,
  adminIdentitySourceKey,
  adminIdentitySourceTone,
  adminIdentityStatusKey,
  adminIdentityStatusTone,
  summarizeAdminIdentities
} from "../adminAccess";
import type { Translator } from "../consolePresenters";
import { formatDate } from "../consolePresenters";
import type { AdminAccessController } from "../hooks/useAdminAccessController";
import type { AdminIdentity, AdminIdentityRole, CreateAdminIdentityRequest } from "../types";
import { Panel } from "./ConsolePrimitives";
import { Badge, EmptyRow } from "./ui";

interface AdminAccessManagementViewProps {
  controller: AdminAccessController;
  t: Translator;
}

const defaultForm: CreateAdminIdentityRequest = {
  actor: "",
  displayName: "",
  role: "tenant_admin",
  tenantId: "",
  workspaceId: ""
};

const roleOptions: AdminIdentityRole[] = ["platform_admin", "tenant_admin", "security_reviewer"];

export function AdminAccessManagementView({ controller, t }: AdminAccessManagementViewProps) {
  const [form, setForm] = useState<CreateAdminIdentityRequest>(defaultForm);
  const summary = useMemo(() => summarizeAdminIdentities(controller.identities), [controller.identities]);
  const messageText = controller.message
    ? controller.message.params
      ? tx(t, controller.message.key, controller.message.params)
      : t(controller.message.key)
    : "";
  const selected = controller.selected;
  const createDisabled =
    controller.creating ||
    controller.forbidden ||
    !form.actor.trim() ||
    ((form.role === "tenant_admin" || form.role === "security_reviewer") && !form.tenantId?.trim());

  useEffect(() => {
    if (controller.modal === "create") setForm(defaultForm);
  }, [controller.modal]);

  function submitCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void controller.submitCreate({
      actor: form.actor.trim(),
      displayName: form.displayName?.trim() || undefined,
      role: form.role,
      tenantId: form.role === "platform_admin" ? undefined : form.tenantId?.trim(),
      workspaceId: form.role === "platform_admin" ? undefined : form.workspaceId?.trim()
    });
  }

  return (
    <Panel
      action={(
        <div className="admin-access-actions">
          <button className="secondary-button" disabled={controller.loading} onClick={() => void controller.loadAdminIdentities()} type="button">
            <RefreshCw size={14} />
            {controller.loading ? t("action.loading") : t("action.refresh")}
          </button>
          {!controller.forbidden ? (
            <button className="primary-button" onClick={controller.openCreate} type="button">
              <Plus size={14} />
              {t("adminAccess.create")}
            </button>
          ) : null}
        </div>
      )}
      className="span-12 admin-access-panel"
      icon={<ShieldCheck size={18} />}
      title={t("adminAccess.title")}
    >
      <div className="admin-access-workspace">
        <header className="admin-access-intro">
          <div>
            <span className="section-kicker">{t("page.adminAccess")}</span>
            <h3>{t("adminAccess.heading")}</h3>
            <p>{t("adminAccess.subtitle")}</p>
          </div>
        </header>

        <div className="admin-access-summary" aria-label={t("adminAccess.summary")}>
          <AdminAccessMetric label={t("adminAccess.metric.active")} value={summary.active} />
          <AdminAccessMetric label={t("adminAccess.metric.scoped")} value={summary.scoped} />
          <AdminAccessMetric label={t("adminAccess.metric.bootstrap")} value={summary.bootstrap} />
          <AdminAccessMetric label={t("adminAccess.metric.disabled")} value={summary.disabled} />
        </div>

        {messageText ? <div className="admin-access-message">{messageText}</div> : null}

        {controller.oneTimeKey ? (
          <section className="admin-access-key-panel" aria-label={t("adminAccess.oneTimeKey")}>
            <div>
              <span className="section-kicker">{t("adminAccess.oneTimeKey")}</span>
              <strong>{controller.oneTimeKey}</strong>
              <p>{t("adminAccess.oneTimeKeyDetail")}</p>
            </div>
            <div className="admin-access-key-actions">
              <button className="secondary-button" onClick={() => void navigator.clipboard?.writeText(controller.oneTimeKey)} type="button">
                <Copy size={14} />
                {t("action.copy")}
              </button>
              <button className="secondary-button" onClick={controller.clearOneTimeKey} type="button">
                <X size={14} />
                {t("action.dismiss")}
              </button>
            </div>
          </section>
        ) : null}

        {controller.identities.length === 0 ? (
          <div className="admin-access-empty-state">
            <EmptyRow
              actionLabel={controller.forbidden ? undefined : t("adminAccess.create")}
              detail={controller.forbidden ? t("adminAccess.forbiddenDetail") : t("adminAccess.emptyDetail")}
              onAction={controller.forbidden ? undefined : controller.openCreate}
              title={controller.forbidden ? t("adminAccess.forbiddenTitle") : t("adminAccess.emptyTitle")}
            />
          </div>
        ) : (
          <div className="table-wrap admin-access-table">
            <table>
              <thead>
                <tr>
                  <th>{t("adminAccess.column.admin")}</th>
                  <th>{t("adminAccess.column.role")}</th>
                  <th>{t("adminAccess.column.scope")}</th>
                  <th>{t("adminAccess.column.source")}</th>
                  <th>{t("adminAccess.column.status")}</th>
                  <th>{t("adminAccess.column.key")}</th>
                  <th>{t("adminAccess.column.lastUsed")}</th>
                  <th>{t("adminAccess.column.actions")}</th>
                </tr>
              </thead>
              <tbody>
                {controller.identities.map((identity) => (
                  <AdminIdentityRow
                    identity={identity}
                    key={identity.id}
                    onDisable={controller.openDisable}
                    onRotate={controller.openRotate}
                    t={t}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {controller.modal === "create" ? (
        <AdminAccessModal closeLabel={t("action.dismiss")} onClose={controller.closeModal} title={t("adminAccess.create")}>
          <form className="admin-access-form" onSubmit={submitCreate}>
            <label>
              {t("adminAccess.form.actor")}
              <input value={form.actor} onChange={(event) => setForm({ ...form, actor: event.target.value })} placeholder={t("adminAccess.form.actorPlaceholder")} />
            </label>
            <label>
              {t("adminAccess.form.displayName")}
              <input value={form.displayName ?? ""} onChange={(event) => setForm({ ...form, displayName: event.target.value })} placeholder={t("adminAccess.form.displayNamePlaceholder")} />
            </label>
            <div className="admin-access-role-picker" role="group" aria-label={t("adminAccess.form.role")}>
              {roleOptions.map((role) => (
                <button
                  className={form.role === role ? "is-selected" : ""}
                  key={role}
                  onClick={() => setForm({ ...form, role })}
                  type="button"
                >
                  {t(adminIdentityRoleKey(role))}
                </button>
              ))}
            </div>
            {form.role !== "platform_admin" ? (
              <div className="form-row">
                <label>
                  {t("adminAccess.form.tenant")}
                  <input value={form.tenantId ?? ""} onChange={(event) => setForm({ ...form, tenantId: event.target.value })} placeholder={t("adminAccess.form.tenantPlaceholder")} />
                </label>
                <label>
                  {t("adminAccess.form.workspace")}
                  <input value={form.workspaceId ?? ""} onChange={(event) => setForm({ ...form, workspaceId: event.target.value })} placeholder={t("adminAccess.form.workspacePlaceholder")} />
                </label>
              </div>
            ) : null}
            <div className="admin-access-modal-footer">
              <button className="secondary-button" onClick={controller.closeModal} type="button">{t("action.cancel")}</button>
              <button className="primary-button" disabled={createDisabled} type="submit">
                <Plus size={14} />
                {controller.creating ? t("action.loading") : t("adminAccess.create")}
              </button>
            </div>
          </form>
        </AdminAccessModal>
      ) : null}

      {controller.modal === "rotate" && selected ? (
        <AdminAccessModal closeLabel={t("action.dismiss")} onClose={controller.closeModal} title={t("adminAccess.rotate")}>
          <ConfirmCopy identity={selected} t={t} textKey="adminAccess.rotateConfirm" />
          <div className="admin-access-modal-footer">
            <button className="secondary-button" onClick={controller.closeModal} type="button">{t("action.cancel")}</button>
            <button className="primary-button" disabled={controller.creating} onClick={() => void controller.submitRotate()} type="button">
              <RotateCcw size={14} />
              {controller.creating ? t("action.loading") : t("adminAccess.rotate")}
            </button>
          </div>
        </AdminAccessModal>
      ) : null}

      {controller.modal === "disable" && selected ? (
        <AdminAccessModal closeLabel={t("action.dismiss")} onClose={controller.closeModal} title={t("adminAccess.disable")}>
          <ConfirmCopy identity={selected} t={t} textKey="adminAccess.disableConfirm" />
          <div className="admin-access-modal-footer">
            <button className="secondary-button" onClick={controller.closeModal} type="button">{t("action.cancel")}</button>
            <button className="danger-button" disabled={controller.creating} onClick={() => void controller.submitDisable()} type="button">
              <Trash2 size={14} />
              {controller.creating ? t("action.loading") : t("adminAccess.disable")}
            </button>
          </div>
        </AdminAccessModal>
      ) : null}
    </Panel>
  );
}

function AdminIdentityRow({
  identity,
  onDisable,
  onRotate,
  t
}: {
  identity: AdminIdentity;
  onDisable: (identity: AdminIdentity) => void;
  onRotate: (identity: AdminIdentity) => void;
  t: Translator;
}) {
  const scope = adminIdentityScopeText(identity);
  return (
    <tr>
      <td>
        <div className="admin-access-admin-cell">
          <strong>{identity.displayName || identity.actor}</strong>
          <span>{identity.actor}</span>
        </div>
      </td>
      <td>{t(adminIdentityRoleKey(identity.role))}</td>
      <td>{scope === "all" ? t("adminAccess.scopeAll") : scope}</td>
      <td><Badge tone={adminIdentitySourceTone(identity.source)}>{t(adminIdentitySourceKey(identity.source))}</Badge></td>
      <td><Badge tone={adminIdentityStatusTone(identity.status)}>{t(adminIdentityStatusKey(identity.status))}</Badge></td>
      <td>{identity.keyPrefix || "-"}</td>
      <td>{identity.lastUsedAt ? formatDate(identity.lastUsedAt) : t("adminAccess.neverUsed")}</td>
      <td>
        {identity.source === "managed" ? (
          <div className="table-action-group">
            <button className="table-action" onClick={() => onRotate(identity)} type="button">
              <KeyRound size={13} />
              {t("adminAccess.rotate")}
            </button>
            <button className="table-action is-danger" disabled={identity.status === "disabled"} onClick={() => onDisable(identity)} type="button">
              <Trash2 size={13} />
              {t("adminAccess.disable")}
            </button>
          </div>
        ) : (
          <span className="admin-access-readonly">{t("adminAccess.readOnly")}</span>
        )}
      </td>
    </tr>
  );
}

function AdminAccessMetric({ label, value }: { label: string; value: number }) {
  return (
    <div className="admin-access-metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function AdminAccessModal({ children, closeLabel, onClose, title }: { children: ReactNode; closeLabel: string; onClose: () => void; title: string }) {
  return (
    <div className="admin-access-modal-backdrop" onMouseDown={(event) => {
      if (event.target === event.currentTarget) onClose();
    }} role="presentation">
      <section aria-modal="true" className="admin-access-modal" role="dialog">
        <header>
          <h3>{title}</h3>
          <button aria-label={closeLabel} className="icon-button compact" onClick={onClose} type="button">
            <X size={15} />
          </button>
        </header>
        {children}
      </section>
    </div>
  );
}

function ConfirmCopy({ identity, t, textKey }: { identity: AdminIdentity; t: Translator; textKey: string }) {
  return (
    <div className="admin-access-confirm">
      <p>{tx(t, textKey, { actor: identity.displayName || identity.actor })}</p>
      <dl>
        <div>
          <dt>{t("adminAccess.column.role")}</dt>
          <dd>{t(adminIdentityRoleKey(identity.role))}</dd>
        </div>
        <div>
          <dt>{t("adminAccess.column.scope")}</dt>
          <dd>{adminIdentityScopeText(identity) === "all" ? t("adminAccess.scopeAll") : adminIdentityScopeText(identity)}</dd>
        </div>
      </dl>
    </div>
  );
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}
