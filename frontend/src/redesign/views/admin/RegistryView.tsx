import { useMemo, useState } from "react";
import { RefreshCw } from "lucide-react";
import {
  ApiRequestError,
  createAgent,
  createAgentKey,
  disableAgent,
  probeTarget,
  rotateAgentCredentials,
  updateAgent,
} from "../../../api";
import { formatDate } from "../../../consolePresenters";
import { tx } from "../../../localizedMessages";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { apiErrorPresentation } from "../../model/apiErrorCategory";
import { agentEndpoint, registryRows, registrySummary } from "../../model/registryCatalog";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Banner } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip } from "../../ui/Chip";
import { Field } from "../../ui/Field";
import { Modal } from "../../ui/Modal";
import { EmptyState } from "../../ui/StateViews";
import { Table } from "../../ui/Table";
import { useToast } from "../../ui/Toast";
import type { Agent, TargetProbeResult } from "../../../types";
import type { AdminViewProps } from "./adminViewProps";

type RegistryColumnKey = "resource" | "type" | "endpoint" | "status" | "actions";
type RegisterType = "local" | "mcp";

interface RegisterFormState {
  name: string;
  tenantId: string;
  type: RegisterType;
  workspaceId: string;
}

interface EditFormState {
  description: string;
  endpoint: string;
  name: string;
  status: "active" | "draft";
}

interface RotateFormState {
  credentialName: string;
  credentialValue: string;
}

interface OneTimeKey {
  expiresAt: string;
  key: string;
}

const keyTtlOptions = [15, 30, 60] as const;

function maskSecret(value: string): string {
  return value.length > 12 ? `${value.slice(0, 6)}…${value.slice(-4)}` : "••••";
}

export function RegistryView({ data, onRetry, params }: AdminViewProps) {
  const { language, t } = useRedesignI18n();
  const toast = useToast();
  const consoleData = data.data;
  const agents = consoleData?.agents ?? [];
  const rows = useMemo(() => registryRows(agents), [agents]);
  const summary = useMemo(() => registrySummary(rows), [rows]);
  const probeSupported = data.capabilities.has("target_probe_v1");

  const [registerOpen, setRegisterOpen] = useState(false);
  const [registerForm, setRegisterForm] = useState<RegisterFormState>({
    name: "",
    tenantId: params.tenant ?? "",
    type: "mcp",
    workspaceId: params.workspace ?? ""
  });
  const [registerError, setRegisterError] = useState("");
  const [registerEndpoint, setRegisterEndpoint] = useState("");
  const [acting, setActing] = useState(false);

  const [editing, setEditing] = useState<RegistryRow | null>(null);
  const [editForm, setEditForm] = useState<EditFormState>({ description: "", endpoint: "", name: "", status: "active" });
  const [editError, setEditError] = useState("");
  const [rotateForm, setRotateForm] = useState<RotateFormState>({ credentialName: "", credentialValue: "" });
  const [rotateDone, setRotateDone] = useState<{ copied: string; version: number } | null>(null);
  const [oneTimeKey, setOneTimeKey] = useState<OneTimeKey | null>(null);
  const [keyTtl, setKeyTtl] = useState<number>(30);
  const [disableOpen, setDisableOpen] = useState(false);

  const [probe, setProbe] = useState<{ agent: Agent; result: TargetProbeResult } | null>(null);
  const [probing, setProbing] = useState<Agent | null>(null);

  function openCreate() {
    setRegisterForm((current) => ({
      ...current,
      name: "",
      type: "mcp"
    }));
    setRegisterEndpoint("");
    setRegisterError("");
    setRegisterOpen(true);
  }

  async function submitRegister() {
    const name = registerForm.name.trim();
    const workspaceId = registerForm.workspaceId.trim();
    if (!name || !workspaceId) {
      setRegisterError(t("rd.registry.validation.name"));
      return;
    }
    const endpoint = registerEndpoint.trim();
    if (registerForm.type === "mcp" && !endpoint) {
      setRegisterError(t("rd.registry.validation.endpoint"));
      return;
    }
    setActing(true);
    setRegisterError("");
    try {
      await createAgent({
        channelConfig: registerForm.type === "mcp" ? { endpoint } : undefined,
        channelType: registerForm.type,
        name,
        tenantId: registerForm.tenantId.trim() || undefined,
        workspaceId
      });
      toast(t("rd.registry.created"));
      setRegisterOpen(false);
      await onRetry();
    } catch (error) {
      setRegisterError(duplicateAwareMessage(t, language, error, "rd.registry.failed"));
    } finally {
      setActing(false);
    }
  }

  function openEdit(row: RegistryRow) {
    setEditing(row);
    setEditForm({
      description: row.agent.description ?? "",
      endpoint: row.endpoint,
      name: row.agent.name,
      status: row.agent.status === "draft" ? "draft" : "active"
    });
    setEditError("");
    setRotateForm({ credentialName: "", credentialValue: "" });
    setRotateDone(null);
    setOneTimeKey(null);
    setDisableOpen(false);
  }

  async function submitEdit() {
    if (!editing) return;
    const name = editForm.name.trim();
    if (!name) {
      setEditError(t("rd.registry.validation.name"));
      return;
    }
    const endpoint = editForm.endpoint.trim();
    if (editing.kind === "target" && !endpoint) {
      setEditError(t("rd.registry.validation.endpoint"));
      return;
    }
    setActing(true);
    setEditError("");
    try {
      await updateAgent(editing.agent.id, {
        channelConfig: editing.kind === "target" ? { ...editing.agent.channelConfig, endpoint } : undefined,
        description: editForm.description.trim() || undefined,
        name,
        status: editForm.status
      });
      toast(t("rd.registry.updated"));
      setEditing(null);
      await onRetry();
    } catch (error) {
      setEditError(duplicateAwareMessage(t, language, error, "rd.registry.failed"));
    } finally {
      setActing(false);
    }
  }

  async function submitRotate() {
    if (!editing) return;
    const credentialName = rotateForm.credentialName.trim();
    const credentialValue = rotateForm.credentialValue;
    if (!credentialName || !credentialValue.trim()) {
      setEditError(t("rd.registry.validation.credential"));
      return;
    }
    setActing(true);
    setEditError("");
    try {
      const updated = await rotateAgentCredentials(editing.agent.id, {
        credentials: { [credentialName]: credentialValue }
      });
      setRotateDone({ copied: credentialValue, version: updated.credentialVersion });
      setRotateForm({ credentialName, credentialValue: "" });
      toast(t("rd.registry.rotated"));
      await onRetry();
    } catch (error) {
      setEditError(apiErrorPresentation(t, language, error, "rd.registry.failed").detail);
    } finally {
      setActing(false);
    }
  }

  async function submitCreateKey() {
    if (!editing) return;
    setActing(true);
    setEditError("");
    try {
      const issued = await createAgentKey({
        agentId: editing.agent.id,
        expiresInSeconds: keyTtl * 60
      });
      setOneTimeKey({ expiresAt: issued.expiresAt, key: issued.key });
    } catch (error) {
      setEditError(apiErrorPresentation(t, language, error, "rd.registry.keyFailed").detail);
    } finally {
      setActing(false);
    }
  }

  async function submitDisable() {
    if (!editing) return;
    setActing(true);
    try {
      await disableAgent(editing.agent.id);
      toast(t("rd.registry.disabled"));
      setEditing(null);
      setDisableOpen(false);
      await onRetry();
    } catch (error) {
      setEditError(apiErrorPresentation(t, language, error, "rd.registry.failed").detail);
      setDisableOpen(false);
    } finally {
      setActing(false);
    }
  }

  async function runProbe(agent: Agent) {
    setProbing(agent);
    try {
      const result = await probeTarget(agent.id);
      setProbe({ agent, result });
    } catch {
      setProbe({
        agent,
        result: {
          checkedAt: new Date().toISOString(),
          durationMs: 0,
          endpoint: agentEndpoint(agent),
          httpStatus: 0,
          status: "error",
          targetId: agent.id,
          toolCount: 0
        }
      });
    } finally {
      setProbing(null);
    }
  }

  async function copySecret(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      toast(t("rd.common.copied"));
    } catch {
      toast(t("rd.common.copyFailed"), "warning");
    }
  }

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t("rd.nav.registry")}</h1>
          <p>{t("rd.page.registry.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      <div className="stack">
        <Card
          flush
          right={
            <>
              <Chip tone="neutral">{tx(t, "rd.registry.countCallers", { count: summary.callers })} · {tx(t, "rd.registry.countTargets", { count: summary.targets })}</Chip>
              <Button onClick={openCreate} size="sm" variant="primary">{t("rd.registry.register")}</Button>
            </>
          }
          title={t("rd.registry.tableTitle")}
        >
          <Table
            caption={t("rd.registry.tableTitle")}
            empty={<EmptyState desc={t("rd.registry.emptyDesc")} title={t("rd.registry.empty")} />}
            renderCell={(row, column) => renderRegistryCell(row, column, t, probeSupported, probing, runProbe, openEdit)}
            rowKey={(row) => row.agent.id}
            rows={rows}
            tableId="adminRegistry"
          />
        </Card>
      </div>

      <Modal
        footer={
          <>
            <Button onClick={() => setRegisterOpen(false)} variant="ghost">{t("rd.apr.cancel")}</Button>
            <Button disabled={acting} onClick={() => void submitRegister()} variant="primary">{t("rd.registry.register")}</Button>
          </>
        }
        onClose={() => setRegisterOpen(false)}
        open={registerOpen}
        title={t("rd.registry.registerTitle")}
      >
        <form
          className="stack"
          onSubmit={(event) => {
            event.preventDefault();
            void submitRegister();
          }}
        >
          <div className="form-grid">
            <Field htmlFor="reg-type" label={t("rd.registry.field.type")}>
              <select
                className="select"
                id="reg-type"
                onChange={(event) => setRegisterForm({ ...registerForm, type: event.target.value as RegisterType })}
                value={registerForm.type}
              >
                <option value="mcp">{t("rd.registry.type.target")}</option>
                <option value="local">{t("rd.registry.type.caller")}</option>
              </select>
            </Field>
            <Field htmlFor="reg-name" label={t("rd.registry.field.name")}>
              <input
                className="input"
                id="reg-name"
                onChange={(event) => setRegisterForm({ ...registerForm, name: event.target.value })}
                value={registerForm.name}
              />
            </Field>
          </div>
          <div className="form-grid">
            <Field hint={t("rd.registry.field.workspaceHint")} htmlFor="reg-workspace" label={t("rd.registry.field.workspace")}>
              <input
                className="input"
                id="reg-workspace"
                onChange={(event) => setRegisterForm({ ...registerForm, workspaceId: event.target.value })}
                value={registerForm.workspaceId}
              />
            </Field>
            <Field hint={t("rd.registry.field.tenantHint")} htmlFor="reg-tenant" label={t("rd.registry.field.tenant")}>
              <input
                className="input"
                id="reg-tenant"
                onChange={(event) => setRegisterForm({ ...registerForm, tenantId: event.target.value })}
                value={registerForm.tenantId}
              />
            </Field>
          </div>
          {registerForm.type === "mcp" ? (
            <Field hint={t("rd.registry.field.endpointHint")} htmlFor="reg-endpoint" label={t("rd.registry.field.endpoint")}>
              <input
                className="input mono"
                id="reg-endpoint"
                onChange={(event) => setRegisterEndpoint(event.target.value)}
                placeholder="http://127.0.0.1:8787/mcp"
                value={registerEndpoint}
              />
            </Field>
          ) : null}
          {registerError ? <Banner desc={registerError} title={t("rd.error.other.title")} tone="danger" /> : null}
        </form>
      </Modal>

      <Modal
        footer={
          <>
            <Button onClick={() => setEditing(null)} variant="ghost">{t("rd.common.close")}</Button>
            <Button disabled={acting} onClick={() => void submitEdit()} variant="primary">{t("rd.registry.save")}</Button>
          </>
        }
        onClose={() => setEditing(null)}
        open={editing !== null}
        size="wide"
        title={editing ? `${t("rd.registry.editTitle")} · ${editing.agent.name}` : ""}
      >
        {editing ? (
          <div className="stack">
            <form
              className="stack"
              onSubmit={(event) => {
                event.preventDefault();
                void submitEdit();
              }}
            >
              <div className="form-grid">
                <Field htmlFor="edit-name" label={t("rd.registry.field.name")}>
                  <input
                    className="input"
                    id="edit-name"
                    onChange={(event) => setEditForm({ ...editForm, name: event.target.value })}
                    value={editForm.name}
                  />
                </Field>
                <Field htmlFor="edit-status" label={t("rd.col.status")}>
                  <select
                    className="select"
                    id="edit-status"
                    onChange={(event) => setEditForm({ ...editForm, status: event.target.value === "draft" ? "draft" : "active" })}
                    value={editForm.status}
                  >
                    <option value="active">{t("rd.status.active")}</option>
                    <option value="draft">{t("rd.registry.status.draft")}</option>
                  </select>
                </Field>
              </div>
              <Field htmlFor="edit-desc" label={t("rd.registry.field.description")}>
                <input
                  className="input"
                  id="edit-desc"
                  onChange={(event) => setEditForm({ ...editForm, description: event.target.value })}
                  value={editForm.description}
                />
              </Field>
              {editing.kind === "target" ? (
                <Field htmlFor="edit-endpoint" label={t("rd.registry.field.endpoint")}>
                  <input
                    className="input mono"
                    id="edit-endpoint"
                    onChange={(event) => setEditForm({ ...editForm, endpoint: event.target.value })}
                    value={editForm.endpoint}
                  />
                </Field>
              ) : null}
              <div className="mono small muted">{editing.agent.id} · {tx(t, "rd.registry.credentialVersion", { version: editing.agent.credentialVersion })}</div>
            </form>

            <div className="section-title">{t("rd.registry.section.credentials")}</div>
            <form
              className="stack"
              onSubmit={(event) => {
                event.preventDefault();
                void submitRotate();
              }}
            >
              <div className="form-grid">
                <Field htmlFor="rot-name" label={t("rd.registry.rotateField.name")}>
                  <input
                    className="input"
                    id="rot-name"
                    onChange={(event) => setRotateForm({ ...rotateForm, credentialName: event.target.value })}
                    value={rotateForm.credentialName}
                  />
                </Field>
                <Field htmlFor="rot-value" label={t("rd.registry.rotateField.value")}>
                  <input
                    autoComplete="off"
                    className="input"
                    id="rot-value"
                    onChange={(event) => setRotateForm({ ...rotateForm, credentialValue: event.target.value })}
                    type="password"
                    value={rotateForm.credentialValue}
                  />
                </Field>
              </div>
              <div className="form-actions">
                <Button disabled={acting} type="submit" variant="ghost">{t("rd.registry.rotate")}</Button>
              </div>
            </form>
            {rotateDone ? (
              <div className="key-panel">
                <div className="key-panel-main">
                  <div className="key-panel-value">{maskSecret(rotateDone.copied)}</div>
                  <div className="key-panel-sub">
                    {tx(t, "rd.registry.rotateCopyHint", { version: rotateDone.version })}
                  </div>
                </div>
                <Button onClick={() => void copySecret(rotateDone.copied)} size="sm" variant="ghost">
                  {t("rd.registry.copyCredential")}
                </Button>
              </div>
            ) : null}

            <div className="section-title">{t("rd.registry.section.keys")}</div>
            <div className="form-grid">
              <Field htmlFor="key-ttl" label={t("rd.registry.keyTtl")}>
                <select
                  className="select"
                  id="key-ttl"
                  onChange={(event) => setKeyTtl(Number(event.target.value))}
                  value={keyTtl}
                >
                  {keyTtlOptions.map((minutes) => (
                    <option key={minutes} value={minutes}>{tx(t, "rd.registry.keyTtlMinutes", { minutes })}</option>
                  ))}
                </select>
              </Field>
              <div className="form-actions">
                <Button disabled={acting} onClick={() => void submitCreateKey()} variant="ghost">
                  {t("rd.registry.createKey")}
                </Button>
              </div>
            </div>
            {oneTimeKey ? (
              <div className="key-panel">
                <div className="key-panel-main">
                  <div className="key-panel-value">{maskSecret(oneTimeKey.key)}</div>
                  <div className="key-panel-sub">
                    {t("rd.registry.keyMaskedHint")} · {tx(t, "rd.registry.keyExpires", { time: formatDate(oneTimeKey.expiresAt, language) })}
                  </div>
                </div>
                <Button onClick={() => void copySecret(oneTimeKey.key)} size="sm" variant="primary">
                  {t("rd.registry.keyCopy")}
                </Button>
              </div>
            ) : null}

            {editError ? <Banner desc={editError} title={t("rd.error.other.title")} tone="danger" /> : null}

            <div className="form-actions">
              <Button
                disabled={acting || editing.agent.status === "disabled"}
                onClick={() => setDisableOpen(true)}
                variant="danger-ghost"
              >
                {t("rd.registry.disable")}
              </Button>
            </div>
          </div>
        ) : null}
      </Modal>

      <Modal
        footer={
          <>
            <Button onClick={() => setDisableOpen(false)} variant="ghost">{t("rd.apr.cancel")}</Button>
            <Button disabled={acting} onClick={() => void submitDisable()} variant="danger-ghost">
              {t("rd.registry.disableAction")}
            </Button>
          </>
        }
        onClose={() => setDisableOpen(false)}
        open={disableOpen && editing !== null}
        size="narrow"
        title={t("rd.registry.disableTitle")}
      >
        {editing ? (
          <p className="muted">
            {tx(t, "rd.registry.disableConfirm", { name: editing.agent.name })}
          </p>
        ) : null}
      </Modal>

      <Modal
        footer={<Button onClick={() => setProbe(null)} variant="ghost">{t("rd.common.close")}</Button>}
        onClose={() => setProbe(null)}
        open={probe !== null}
        size="narrow"
        title={probe ? tx(t, "rd.registry.probe.title", { name: probe.agent.name }) : ""}
      >
        {probe ? (
          <ProbeResult result={probe.result} t={t} />
        ) : null}
      </Modal>
    </>
  );
}

function duplicateAwareMessage(
  t: ReturnType<typeof useRedesignI18n>["t"],
  language: ReturnType<typeof useRedesignI18n>["language"],
  error: unknown,
  fallbackKey: string
): string {
  if (error instanceof ApiRequestError && error.code === "DUPLICATE_RESOURCE_MUTATION") {
    return t("rd.registry.duplicate");
  }
  const presentation = apiErrorPresentation(t, language, error, fallbackKey);
  return presentation.detail || presentation.next;
}

function ProbeResult({
  result,
  t
}: {
  result: TargetProbeResult;
  t: ReturnType<typeof useRedesignI18n>["t"];
}) {
  if (result.status === "ok") {
    return (
      <div className="stack">
        <Banner
          tone="success"
          title={tx(t, "rd.registry.probe.okTitle", { status: result.httpStatus })}
        />
        <div className="mono small muted">{result.endpoint}</div>
        <p className="muted small">
          {tx(t, "rd.registry.probe.okDetail", { ms: result.durationMs, tools: result.toolCount ?? 0 })}
        </p>
      </div>
    );
  }
  return (
    <div className="stack">
      <Banner tone="danger" title={t("rd.registry.probe.failTitle")} />
      <div className="mono small muted">{result.endpoint}</div>
      <p className="mono small">
        {result.errorCode ?? ""} {result.message ?? ""}
      </p>
    </div>
  );
}

type RegistryRow = ReturnType<typeof registryRows>[number];

function renderRegistryCell(
  row: RegistryRow,
  column: RegistryColumnKey,
  t: ReturnType<typeof useRedesignI18n>["t"],
  probeSupported: boolean,
  probing: Agent | null,
  onProbe: (agent: Agent) => void,
  onEdit: (row: RegistryRow) => void
) {
  switch (column) {
    case "resource":
      return (
        <div>
          <div className="cell-main">{row.agent.name}</div>
          <div className="cell-sub mono">{row.agent.id}</div>
        </div>
      );
    case "type":
      return <Chip tone={row.kind === "caller" ? "info" : "neutral"}>{row.kind === "caller" ? t("rd.registry.type.caller") : t("rd.registry.type.target")}</Chip>;
    case "endpoint":
      return <span className="mono small">{row.endpoint || t("rd.registry.endpointLocal")}</span>;
    case "status":
      if (row.agent.status === "disabled") return <Chip tone="danger">{t("rd.status.inactive")}</Chip>;
      if (row.agent.status === "draft") return <Chip tone="neutral">{t("rd.registry.status.draft")}</Chip>;
      return <Chip tone="success">{t("rd.status.active")}</Chip>;
    case "actions":
      return (
        <div className="apr-actions">
          {row.kind === "target" ? (
            <Button
              disabled={!probeSupported}
              onClick={() => onProbe(row.agent)}
              size="sm"
              title={probeSupported ? undefined : t("rd.registry.probe.unsupported")}
              variant="link"
            >
              {probing?.id === row.agent.id ? <RefreshCw aria-hidden="true" className="spin" size={13} /> : null}
              {t("rd.registry.test")}
            </Button>
          ) : null}
          <Button onClick={() => onEdit(row)} size="sm" variant="link">
            {t("rd.registry.edit")}
          </Button>
        </div>
      );
  }
}
