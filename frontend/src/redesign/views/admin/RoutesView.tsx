import { useState } from "react";
import {
  createRoutePolicy,
  updateRoutePolicy,
} from "../../../api";
import { apiErrorPresentation } from "../../model/apiErrorCategory";
import { parsePriorityInput, routePolicyRows, routeTypeOptions } from "../../model/routePolicyCatalog";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Banner } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Field } from "../../ui/Field";
import { Modal } from "../../ui/Modal";
import { Switch } from "../../ui/Switch";
import { Table } from "../../ui/Table";
import { EmptyState } from "../../ui/StateViews";
import { useToast } from "../../ui/Toast";
import type { RoutePolicy } from "../../../types";
import type { AdminViewProps } from "./adminViewProps";

type RouteColumnKey = "rule" | "match" | "target" | "priority" | "status" | "actions";

interface RouteFormState {
  callerAgentId: string;
  name: string;
  priority: string;
  routeKey: string;
  routeType: string;
  targetAgentId: string;
}

function routeFormFrom(policy: RoutePolicy | null, fallbackType: string): RouteFormState {
  return {
    callerAgentId: policy?.callerAgentId ?? "",
    name: policy?.name ?? "",
    priority: policy ? String(policy.priority) : "50",
    routeKey: policy?.routeKey ?? "",
    routeType: policy?.routeType ?? fallbackType,
    targetAgentId: policy?.targetAgentId ?? ""
  };
}

export function RoutesView({ data, onRetry }: AdminViewProps) {
  const { language, t } = useRedesignI18n();
  const toast = useToast();
  const consoleData = data.data;
  const agents = consoleData?.agents ?? [];
  const policies = consoleData?.routePolicies ?? [];

  const agentNames = new Map(agents.map((agent) => [agent.id, agent.name]));
  const agentName = (agentId: string) => agentNames.get(agentId) ?? agentId;
  const rows = routePolicyRows(policies, agentName);
  const typeOptions = routeTypeOptions(policies);

  const [editing, setEditing] = useState<RoutePolicy | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [form, setForm] = useState<RouteFormState>(() => routeFormFrom(null, "mcp"));
  const [formError, setFormError] = useState("");
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<unknown>(null);

  function openCreate() {
    setEditing(null);
    setForm(routeFormFrom(null, typeOptions[0] ?? "mcp"));
    setFormError("");
    setFormOpen(true);
  }

  function openEdit(policy: RoutePolicy) {
    setEditing(policy);
    setForm(routeFormFrom(policy, typeOptions[0] ?? "mcp"));
    setFormError("");
    setFormOpen(true);
  }

  async function submitForm() {
    const name = form.name.trim();
    if (!name) {
      setFormError(t("rd.rt.validation.name"));
      return;
    }
    if (!form.callerAgentId || !form.targetAgentId) {
      setFormError(t("rd.rt.validation.pair"));
      return;
    }
    const priority = parsePriorityInput(form.priority);
    if (!priority.ok) {
      setFormError(t("rd.rt.validation.priority"));
      return;
    }
    setActing(true);
    setFormError("");
    try {
      if (editing) {
        // PATCH rejects caller/target with INVALID_JSON: the pair is fixed at
        // creation, only name/route fields/priority may change.
        await updateRoutePolicy(editing.id, {
          name,
          priority: priority.value,
          routeKey: form.routeKey.trim() || undefined,
          routeType: form.routeType
        });
        toast(t("rd.rt.updated"));
      } else {
        await createRoutePolicy({
          callerAgentId: form.callerAgentId,
          name,
          priority: priority.value,
          routeKey: form.routeKey.trim() || undefined,
          routeType: form.routeType,
          targetAgentId: form.targetAgentId
        });
        toast(t("rd.rt.created"));
      }
      setFormOpen(false);
      await onRetry();
    } catch (error) {
      setFormError(apiErrorPresentation(t, language, error, "rd.rt.failed").detail || apiErrorPresentation(t, language, error, "rd.rt.failed").next);
    } finally {
      setActing(false);
    }
  }

  async function toggleStatus(policy: RoutePolicy, enabled: boolean) {
    setActionError(null);
    try {
      await updateRoutePolicy(policy.id, { status: enabled ? "enabled" : "disabled" });
      toast(enabled ? t("rd.rt.enabled") : t("rd.rt.disabled"));
      await onRetry();
    } catch (error) {
      setActionError(error);
    }
  }

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t("rd.nav.routes")}</h1>
          <p>{t("rd.page.routes.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      <div className="stack">
        {actionError ? (
          <Banner
            desc={apiErrorPresentation(t, language, actionError, "rd.rt.failed").detail}
            title={t("rd.error.other.title")}
            tone="danger"
          />
        ) : null}
        <Card
          flush
          right={<Button onClick={openCreate} size="sm" variant="primary">{t("rd.rt.create")}</Button>}
          title={t("rd.rt.tableTitle")}
        >
          <Table
            caption={t("rd.rt.tableTitle")}
            empty={<EmptyState desc={t("rd.rt.emptyDesc")} title={t("rd.rt.empty")} />}
            renderCell={(row, column) => renderRouteCell(row, column, t, agentName, toggleStatus, openEdit)}
            rowKey={(row) => row.policy.id}
            rows={rows}
            tableId="adminRoutes"
          />
        </Card>
      </div>

      <Modal
        footer={
          <>
            <Button onClick={() => setFormOpen(false)} variant="ghost">{t("rd.apr.cancel")}</Button>
            <Button disabled={acting} onClick={() => void submitForm()} variant="primary">
              {editing ? t("rd.rt.save") : t("rd.rt.create")}
            </Button>
          </>
        }
        onClose={() => setFormOpen(false)}
        open={formOpen}
        title={editing ? t("rd.rt.editTitle") : t("rd.rt.createTitle")}
      >
        <form
          className="stack"
          onSubmit={(event) => {
            event.preventDefault();
            void submitForm();
          }}
        >
          <Field error={formError || undefined} htmlFor="rt-name" label={t("rd.rt.field.name")}>
            <input
              className="input"
              id="rt-name"
              onChange={(event) => setForm({ ...form, name: event.target.value })}
              value={form.name}
            />
          </Field>
          <div className="form-grid">
            <Field hint={editing ? t("rd.rt.field.pairLocked") : undefined} htmlFor="rt-caller" label={t("rd.rt.field.caller")}>
              <select
                className="select"
                disabled={Boolean(editing)}
                id="rt-caller"
                onChange={(event) => setForm({ ...form, callerAgentId: event.target.value })}
                value={form.callerAgentId}
              >
                <option value="">{t("rd.rt.match.any")}</option>
                {agents.map((agent) => (
                  <option key={agent.id} value={agent.id}>{agent.name}</option>
                ))}
              </select>
            </Field>
            <Field hint={editing ? t("rd.rt.field.pairLocked") : undefined} htmlFor="rt-target" label={t("rd.rt.field.target")}>
              <select
                className="select"
                disabled={Boolean(editing)}
                id="rt-target"
                onChange={(event) => setForm({ ...form, targetAgentId: event.target.value })}
                value={form.targetAgentId}
              >
                <option value="">—</option>
                {agents.map((agent) => (
                  <option key={agent.id} value={agent.id}>{agent.name}</option>
                ))}
              </select>
            </Field>
          </div>
          <div className="form-grid">
            <Field htmlFor="rt-type" label={t("rd.rt.field.routeType")}>
              <select
                className="select"
                id="rt-type"
                onChange={(event) => setForm({ ...form, routeType: event.target.value })}
                value={form.routeType}
              >
                {typeOptions.map((type) => (
                  <option key={type} value={type}>{type}</option>
                ))}
              </select>
            </Field>
            <Field hint={t("rd.rt.field.routeKeyHint")} htmlFor="rt-key" label={t("rd.rt.field.routeKey")}>
              <input
                className="input"
                id="rt-key"
                list="rt-key-presets"
                onChange={(event) => setForm({ ...form, routeKey: event.target.value })}
                value={form.routeKey}
              />
              <datalist id="rt-key-presets">
                {["initialize", "tools/list", "tools/call"].map((preset) => (
                  <option key={preset} value={preset} />
                ))}
              </datalist>
            </Field>
          </div>
          <Field hint={t("rd.rt.field.priorityHint")} htmlFor="rt-priority" label={t("rd.rt.field.priority")}>
            <input
              className="input"
              id="rt-priority"
              inputMode="numeric"
              onChange={(event) => setForm({ ...form, priority: event.target.value })}
              value={form.priority}
            />
          </Field>
        </form>
      </Modal>
    </>
  );
}

function renderRouteCell(
  row: ReturnType<typeof routePolicyRows>[number],
  column: RouteColumnKey,
  t: ReturnType<typeof useRedesignI18n>["t"],
  agentName: (agentId: string) => string,
  onToggle: (policy: RoutePolicy, enabled: boolean) => void,
  onEdit: (policy: RoutePolicy) => void
) {
  switch (column) {
    case "rule":
      return (
        <div>
          <div className="cell-main">{row.policy.name || row.policy.id}</div>
          <div className="cell-sub mono">{row.policy.id}</div>
        </div>
      );
    case "match":
      return (
        <span className="mono small">
          {row.callerName || t("rd.rt.match.any")}
          {" → "}
          {row.targetName}
          {row.routeKey ? ` (${row.routeType}:${row.routeKey})` : row.routeType ? ` (${row.routeType})` : ""}
        </span>
      );
    case "target":
      return <span className="small">{agentName(row.policy.targetAgentId) || row.policy.targetAgentId}</span>;
    case "priority":
      return <span className="mono small">{row.policy.priority}</span>;
    case "status":
      return (
        <Switch
          checked={row.policy.status === "enabled"}
          label={row.policy.status === "enabled" ? t("rd.rt.enabled") : t("rd.rt.disabled")}
          onChange={(checked) => onToggle(row.policy, checked)}
        />
      );
    case "actions":
      return (
        <Button onClick={() => onEdit(row.policy)} size="sm" variant="link">
          {t("rd.rt.edit")}
        </Button>
      );
  }
}
