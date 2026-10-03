import { useMemo, useState } from "react";
import { KeyRound } from "lucide-react";
import { tx } from "../../../localizedMessages";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { useAdminAccess, type AdminAccessAction } from "../../hooks/useAdminAccess";
import {
  adminBoundaryRows,
  adminIdentityMutable,
  createAdminIdentityRequestReady,
  demoAdminSessionVisible,
  type AdminBoundaryRow,
  type AdminRoleKey
} from "../../model/adminBoundary";
import { apiErrorPresentation } from "../../model/apiErrorCategory";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Banner } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip } from "../../ui/Chip";
import { Field } from "../../ui/Field";
import { Modal } from "../../ui/Modal";
import { EmptyState, LoadingState } from "../../ui/StateViews";
import { Table } from "../../ui/Table";
import { useToast } from "../../ui/Toast";
import type { CreateAdminIdentityRequest, UpdateAdminIdentityOwnedAgentsRequest } from "../../../types";
import type { AdminViewProps } from "./adminViewProps";

type AdminColumnKey = "member" | "role" | "scope" | "status" | "actions";
type AdminTableRow = { kind: "demo" } | { kind: "identity"; row: AdminBoundaryRow };

const roleKeys: Record<AdminRoleKey, string> = {
  platform_admin: "rd.adm.role.platform_admin",
  security_reviewer: "rd.adm.role.security_reviewer",
  tenant_admin: "rd.adm.role.tenant_admin"
};

const roleTone: Record<AdminRoleKey, "info" | "neutral" | "warning"> = {
  platform_admin: "info",
  security_reviewer: "warning",
  tenant_admin: "neutral"
};

// Head-and-tail inline mask for one-time key panels; the prefix-only
// convention for stored key prefixes lives in model/secretMask.ts.
function maskInlineSecret(value: string): string {
  return value.length > 12 ? `${value.slice(0, 6)}…${value.slice(-4)}` : "••••";
}

export function AdminAccessView({ data, onRetry, session }: AdminViewProps) {
  const { language, t } = useRedesignI18n();
  const toast = useToast();
  const live = data.status === "live";
  const access = useAdminAccess(live);
  const consoleData = data.data;

  const identityRows = useMemo(() => adminBoundaryRows(access.identities), [access.identities]);
  const tableRows = useMemo<AdminTableRow[]>(
    () => [
      ...(demoAdminSessionVisible(session) ? [{ kind: "demo" as const }] : []),
      ...identityRows.map((row) => ({ kind: "identity" as const, row }))
    ],
    [identityRows, session]
  );

  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState<{ actor: string; displayName: string; role: AdminRoleKey; tenantId: string; workspaceId: string }>({
    actor: "",
    displayName: "",
    role: "security_reviewer",
    tenantId: "",
    workspaceId: ""
  });
  const [rotateTarget, setRotateTarget] = useState<AdminBoundaryRow | null>(null);
  const [disableTarget, setDisableTarget] = useState<AdminBoundaryRow | null>(null);
  const [editTarget, setEditTarget] = useState<AdminBoundaryRow | null>(null);
  const [editSelection, setEditSelection] = useState<Record<string, boolean>>({});
  const [oneTimeKey, setOneTimeKey] = useState<string | null>(null);
  const [actionError, setActionError] = useState("");
  const [acting, setActing] = useState(false);

  const createReady = createAdminIdentityRequestReady({ actor: createForm.actor, role: createForm.role, tenantId: createForm.tenantId });
  const localCallers = useMemo(
    () => (consoleData?.agents ?? []).filter((agent) => agent.channelType === "local"),
    [consoleData]
  );
  const editCandidates = useMemo(() => {
    if (!editTarget) return [];
    const tenantId = editTarget.identity.tenantId ?? "";
    return localCallers.filter((agent) => editTarget.identity.role === "platform_admin" || agent.tenantId === tenantId);
  }, [editTarget, localCallers]);

  function resetCreate() {
    setCreateForm({ actor: "", displayName: "", role: "security_reviewer", tenantId: "", workspaceId: "" });
    setOneTimeKey(null);
    setActionError("");
  }

  async function copyKey(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      toast(t("rd.common.copied"));
    } catch {
      toast(t("rd.common.copyFailed"), "warning");
    }
  }

  async function runAction(
    action: AdminAccessAction,
    input: { body?: CreateAdminIdentityRequest | UpdateAdminIdentityOwnedAgentsRequest; id?: string }
  ) {
    setActing(true);
    setActionError("");
    const result = await access.act(action, input);
    setActing(false);
    if (!result.ok) {
      setActionError(apiErrorPresentation(t, language, result.error, "rd.adm.actionFailed").detail || apiErrorPresentation(t, language, result.error, "rd.adm.actionFailed").next);
      return false;
    }
    const issued = result.outcome && "key" in result.outcome ? result.outcome : null;
    if (issued) {
      setOneTimeKey(issued.key);
    } else {
      toast(t("rd.adm.rotateDone"));
    }
    return true;
  }

  async function submitCreate() {
    const actor = createForm.actor.trim();
    if (!createReady || !actor) {
      setActionError(t("rd.adm.validation.actor"));
      return;
    }
    const body: CreateAdminIdentityRequest = {
      actor,
      displayName: createForm.displayName.trim() || undefined,
      role: createForm.role,
      tenantId: createForm.role === "platform_admin" ? undefined : createForm.tenantId.trim(),
      workspaceId: createForm.workspaceId.trim() || undefined
    };
    const ok = await runAction("create", { body });
    if (ok) toast(t("rd.adm.created"));
  }

  async function submitRotate() {
    if (!rotateTarget) return;
    // On success the modal stays open and swaps to the one-time key panel.
    await runAction("rotate", { id: rotateTarget.identity.id });
  }

  async function submitDisable() {
    if (!disableTarget) return;
    const ok = await runAction("disable", { id: disableTarget.identity.id });
    if (ok) {
      toast(t("rd.adm.disabled"));
      setDisableTarget(null);
    }
  }

  function openEdit(row: AdminBoundaryRow) {
    setOneTimeKey(null);
    setActionError("");
    const selected: Record<string, boolean> = {};
    for (const agentId of row.identity.ownedAgentIds ?? []) selected[agentId] = true;
    setEditSelection(selected);
    setEditTarget(row);
  }

  async function submitEdit() {
    if (!editTarget) return;
    const ownedAgentIds = Object.entries(editSelection)
      .filter(([, checked]) => checked)
      .map(([agentId]) => agentId)
      .sort();
    const ok = await runAction("updateOwnedAgents", { id: editTarget.identity.id, body: { ownedAgentIds } });
    if (ok) {
      toast(t("rd.adm.ownedUpdated"));
      setEditTarget(null);
    }
  }

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t("rd.nav.admin")}</h1>
          <p>{t("rd.page.admin.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      <div className="stack">
        {access.forbidden ? (
          <Banner desc={t("rd.adm.forbidden")} title={t("rd.adm.forbiddenTitle")} tone="warning" />
        ) : null}
        <Card
          flush
          right={
            access.forbidden ? null : (
              <Button
                disabled={access.loading}
                onClick={() => {
                  resetCreate();
                  setCreateOpen(true);
                }}
                size="sm"
                variant="primary"
              >
                {t("rd.adm.create")}
              </Button>
            )
          }
          title={t("rd.adm.tableTitle")}
        >
          <Table
            caption={t("rd.adm.tableTitle")}
            empty={
              access.loading ? (
                <LoadingState label={t("rd.common.loading")} />
              ) : access.failed ? (
                <EmptyState desc={t("rd.adm.failedDesc")} title={t("rd.adm.failed")} />
              ) : (
                <EmptyState desc={t("rd.adm.emptyDesc")} title={t("rd.adm.empty")} />
              )
            }
            renderCell={(row, column) => renderAdminCell(row, column, t, setRotateTarget, setDisableTarget, openEdit)}
            rowKey={(row) => (row.kind === "demo" ? "demo-session" : row.row.identity.id)}
            rows={tableRows}
            tableId="adminAdmins"
          />
        </Card>

        <div className="grid grid-2">
          <Card title={t("rd.adm.sepTitle")}>
            <div className="rule-list">
              {(["selfReview", "namedAdmin", "highRisk"] as const).map((key) => (
                <div className="rule-line" key={key}>
                  <div className="rule-main">
                    <div className="rule-name">{t(`rd.adm.sep.${key}.t`)}</div>
                    <div className="rule-sub">{t(`rd.adm.sep.${key}.d`)}</div>
                  </div>
                  <Chip tone="success">{t("rd.adm.sep.enforced")}</Chip>
                </div>
              ))}
            </div>
          </Card>

          <Card title={t("rd.adm.boundaryTitle")}>
            <div className="boundary-grid">
              <div>
                <h4 className="boundary-col-title">{t("rd.adm.boundary.allow")}</h4>
                <ul className="boundary-list">
                  {["query", "apply", "report", "token"].map((key) => (
                    <li key={key}>{t(`rd.adm.boundary.allow.${key}`)}</li>
                  ))}
                </ul>
              </div>
              <div>
                <h4 className="boundary-col-title">{t("rd.adm.boundary.restrict")}</h4>
                <ul className="boundary-list">
                  {["approval", "crossWorkspace", "export"].map((key) => (
                    <li key={key}>{t(`rd.adm.boundary.restrict.${key}`)}</li>
                  ))}
                </ul>
              </div>
              <div className="boundary-col-danger">
                <h4 className="boundary-col-title">{t("rd.adm.boundary.forbid")}</h4>
                <ul className="boundary-list">
                  {["selfApprove", "sharedToken", "bypass"].map((key) => (
                    <li key={key}>{t(`rd.adm.boundary.forbid.${key}`)}</li>
                  ))}
                </ul>
              </div>
            </div>
            <p className="hint">{t("rd.adm.boundary.hint")}</p>
          </Card>
        </div>
      </div>

      <Modal
        footer={
          oneTimeKey ? (
            <Button onClick={() => setCreateOpen(false)} variant="ghost">{t("rd.common.close")}</Button>
          ) : (
            <>
              <Button onClick={() => setCreateOpen(false)} variant="ghost">{t("rd.apr.cancel")}</Button>
              <Button disabled={acting || !createReady} onClick={() => void submitCreate()} variant="primary">
                {t("rd.adm.create")}
              </Button>
            </>
          )
        }
        onClose={() => setCreateOpen(false)}
        open={createOpen}
        title={t("rd.adm.createTitle")}
      >
        {oneTimeKey ? (
          <div className="stack">
            <div className="key-panel">
              <div className="key-panel-main">
                <div className="key-panel-value">{maskInlineSecret(oneTimeKey)}</div>
                <div className="key-panel-sub">{t("rd.adm.keyHint")}</div>
              </div>
              <Button onClick={() => void copyKey(oneTimeKey)} size="sm" variant="primary">
                <KeyRound aria-hidden="true" size={14} />
                {t("rd.adm.keyCopy")}
              </Button>
            </div>
          </div>
        ) : (
          <form
            className="stack"
            onSubmit={(event) => {
              event.preventDefault();
              void submitCreate();
            }}
          >
            <Field htmlFor="adm-actor" label={t("rd.adm.field.actor")}>
              <input
                className="input mono"
                id="adm-actor"
                onChange={(event) => setCreateForm({ ...createForm, actor: event.target.value })}
                value={createForm.actor}
              />
            </Field>
            <Field htmlFor="adm-display" label={t("rd.adm.field.displayName")}>
              <input
                className="input"
                id="adm-display"
                onChange={(event) => setCreateForm({ ...createForm, displayName: event.target.value })}
                value={createForm.displayName}
              />
            </Field>
            <Field htmlFor="adm-role" label={t("rd.adm.field.role")}>
              <select
                className="select"
                id="adm-role"
                onChange={(event) => setCreateForm({ ...createForm, role: event.target.value as AdminRoleKey })}
                value={createForm.role}
              >
                {(Object.keys(roleKeys) as AdminRoleKey[]).map((role) => (
                  <option key={role} value={role}>{t(roleKeys[role])}</option>
                ))}
              </select>
            </Field>
            {createForm.role !== "platform_admin" ? (
              <Field hint={t("rd.adm.field.tenantHint")} htmlFor="adm-tenant" label={t("rd.adm.field.tenant")}>
                <input
                  className="input mono"
                  id="adm-tenant"
                  onChange={(event) => setCreateForm({ ...createForm, tenantId: event.target.value })}
                  value={createForm.tenantId}
                />
              </Field>
            ) : null}
            <Field htmlFor="adm-workspace" label={t("rd.adm.field.workspace")}>
              <input
                className="input mono"
                id="adm-workspace"
                onChange={(event) => setCreateForm({ ...createForm, workspaceId: event.target.value })}
                value={createForm.workspaceId}
              />
            </Field>
            {actionError ? <Banner desc={actionError} title={t("rd.error.other.title")} tone="danger" /> : null}
          </form>
        )}
      </Modal>

      <Modal
        footer={
          oneTimeKey && rotateTarget ? (
            <Button onClick={() => setRotateTarget(null)} variant="ghost">{t("rd.common.close")}</Button>
          ) : (
            <>
              <Button onClick={() => setRotateTarget(null)} variant="ghost">{t("rd.apr.cancel")}</Button>
              <Button disabled={acting} onClick={() => void submitRotate()} variant="primary">
                {t("rd.adm.rotateConfirmAction")}
              </Button>
            </>
          )
        }
        onClose={() => setRotateTarget(null)}
        open={rotateTarget !== null}
        size="narrow"
        title={t("rd.adm.rotateTitle")}
      >
        {rotateTarget && oneTimeKey ? (
          <div className="key-panel">
            <div className="key-panel-main">
              <div className="key-panel-value">{maskInlineSecret(oneTimeKey)}</div>
              <div className="key-panel-sub">{t("rd.adm.keyHint")}</div>
            </div>
            <Button onClick={() => void copyKey(oneTimeKey)} size="sm" variant="primary">
              {t("rd.adm.keyCopy")}
            </Button>
          </div>
        ) : rotateTarget ? (
          <p className="muted">{tx(t, "rd.adm.rotateConfirm", { name: rotateTarget.displayName })}</p>
        ) : null}
      </Modal>

      <Modal
        footer={
          <>
            <Button onClick={() => setDisableTarget(null)} variant="ghost">{t("rd.apr.cancel")}</Button>
            <Button disabled={acting} onClick={() => void submitDisable()} variant="danger-ghost">
              {t("rd.adm.disableAction")}
            </Button>
          </>
        }
        onClose={() => setDisableTarget(null)}
        open={disableTarget !== null}
        size="narrow"
        title={t("rd.adm.disableTitle")}
      >
        {disableTarget ? (
          <p className="muted">{tx(t, "rd.adm.disableConfirm", { name: disableTarget.displayName })}</p>
        ) : null}
      </Modal>

      <Modal
        footer={
          <>
            <Button onClick={() => setEditTarget(null)} variant="ghost">{t("rd.apr.cancel")}</Button>
            <Button disabled={acting} onClick={() => void submitEdit()} variant="primary">
              {t("rd.adm.ownedSave")}
            </Button>
          </>
        }
        onClose={() => setEditTarget(null)}
        open={editTarget !== null}
        title={tx(t, "rd.adm.editTitle", { name: editTarget?.displayName ?? "" })}
      >
        {editTarget ? (
          <form
            className="stack"
            onSubmit={(event) => {
              event.preventDefault();
              void submitEdit();
            }}
          >
            <p className="muted">{t("rd.adm.editDesc")}</p>
            <ul className="check-list holder-picker">
              {editCandidates.map((agent) => {
                const checked = Boolean(editSelection[agent.id]);
                return (
                  <li className="check-line" key={agent.id}>
                    <input
                      aria-label={agent.name}
                      checked={checked}
                      id={`holder-${agent.id}`}
                      onChange={(event) =>
                        setEditSelection({ ...editSelection, [agent.id]: event.target.checked })
                      }
                      type="checkbox"
                    />
                    <label className="check-main" htmlFor={`holder-${agent.id}`}>
                      <span className="check-name">{agent.name}</span>
                      <span className="check-sub mono">
                        {agent.id}
                        {agent.tenantId ? ` · ${agent.tenantId}` : ""}
                      </span>
                    </label>
                  </li>
                );
              })}
            </ul>
            {editCandidates.length === 0 ? <p className="muted">{t("rd.adm.ownedEmpty")}</p> : null}
            {actionError ? <Banner desc={actionError} title={t("rd.error.other.title")} tone="danger" /> : null}
          </form>
        ) : null}
      </Modal>
    </>
  );
}

function renderAdminCell(
  row: AdminTableRow,
  column: AdminColumnKey,
  t: ReturnType<typeof useRedesignI18n>["t"],
  onRotate: (row: AdminBoundaryRow) => void,
  onDisable: (row: AdminBoundaryRow) => void,
  onEdit: (row: AdminBoundaryRow) => void
) {
  if (row.kind === "demo") {
    switch (column) {
      case "member":
        return (
          <div>
            <div className="cell-main">{t("rd.demo.localAdmin")}</div>
            <div className="cell-sub mono">{t("rd.adm.sessionSub")}</div>
          </div>
        );
      case "role":
        return <Chip tone="info">{t("rd.adm.role.platform_admin")}</Chip>;
      case "scope":
        return <span className="small">{t("rd.adm.scope.all")}</span>;
      case "status":
        return <Chip tone="neutral">{t("rd.adm.sessionRow")}</Chip>;
      case "actions":
        return <span className="small muted">—</span>;
    }
  }
  const identity = row.row.identity;
  switch (column) {
    case "member":
      return (
        <div>
          <div className="cell-main">{row.row.displayName}</div>
          <div className="cell-sub mono">{identity.actor}</div>
        </div>
      );
    case "role":
      return <Chip tone={roleTone[identity.role]}>{t(roleKeys[identity.role])}</Chip>;
    case "scope":
      return (
        <div>
          <span className="small">
            {identity.tenantId
              ? `${identity.tenantId}${identity.workspaceId ? ` / ${identity.workspaceId}` : ""}`
              : t("rd.adm.scope.all")}
          </span>
          <div className="small muted">
            {identity.ownedAgentIds?.length
              ? tx(t, "rd.adm.ownedCount", { count: identity.ownedAgentIds.length })
              : t("rd.adm.ownedNone")}
          </div>
        </div>
      );
    case "status": {
      if (identity.status === "disabled") return <Chip tone="danger">{t("rd.status.inactive")}</Chip>;
      return <Chip tone="success">{t("rd.adm.active")}</Chip>;
    }
    case "actions":
      if (!adminIdentityMutable(identity)) {
        return <span className="small muted" title={t("rd.adm.bootstrapReadonly")}>—</span>;
      }
      return (
        <div className="apr-actions">
          <Button onClick={() => onEdit(row.row)} size="sm" variant="link">
            {t("rd.adm.editOwned")}
          </Button>
          <Button disabled={identity.status === "disabled"} onClick={() => onRotate(row.row)} size="sm" variant="link">
            {t("rd.adm.rotate")}
          </Button>
          <Button
            disabled={identity.status === "disabled"}
            onClick={() => onDisable(row.row)}
            size="sm"
            variant="link"
          >
            {t("rd.adm.disable")}
          </Button>
        </div>
      );
  }
}
