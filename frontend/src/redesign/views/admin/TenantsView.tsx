import { useEffect, useMemo, useState } from "react";
import { Building2, Layers, Users } from "lucide-react";
import { createTenant } from "../../../api";
import { tx } from "../../../localizedMessages";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { usePermissionCatalog } from "../../hooks/usePermissionCatalog";
import { useTenantDetail } from "../../hooks/useTenantDetail";
import { apiErrorPresentation } from "../../model/apiErrorCategory";
import {
  normalizeTenantDetailTab,
  tenantDirectoryRows,
  tenantDirectorySummary,
  type TenantDetailTab,
  type TenantDirectoryRow
} from "../../model/tenantDirectory";
import { adminHash } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Banner } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip } from "../../ui/Chip";
import { Field } from "../../ui/Field";
import { Kpi } from "../../ui/Kpi";
import { KvList, type KvItem } from "../../ui/KvList";
import { Modal } from "../../ui/Modal";
import { EmptyState, LoadingState } from "../../ui/StateViews";
import { Table } from "../../ui/Table";
import { useToast } from "../../ui/Toast";
import type { Tenant } from "../../../types";
import type { AdminViewProps } from "./adminViewProps";

type TenantColumnKey = "tenant" | "workspace" | "resourceCount" | "grants" | "status" | "actions";

const tenantDetailTabKeys: Record<TenantDetailTab, string> = {
  center: "rd.tenants.tab.center",
  org: "rd.tenants.tab.org",
  profile: "rd.tenants.tab.profile"
};

export function TenantsView({ data, onRetry, params }: AdminViewProps) {
  const { language, t } = useRedesignI18n();
  const toast = useToast();
  const live = data.status === "live";
  const catalog = usePermissionCatalog(live);
  const consoleData = data.data;

  const rows = useMemo(
    () =>
      tenantDirectoryRows({
        agents: consoleData?.agents ?? [],
        entitlements: consoleData?.tenantEntitlements ?? [],
        tenants: consoleData?.tenants ?? [],
        workspaceAssignments: consoleData?.workspaceAssignments ?? []
      }),
    [consoleData]
  );
  const summary = tenantDirectorySummary(rows, catalog.subjects.length);

  const [selectedTenantId, setSelectedTenantId] = useState(params.tenant ?? "");
  const [tab, setTab] = useState<TenantDetailTab>(normalizeTenantDetailTab(params.tab));
  const detail = useTenantDetail(live, selectedTenantId);

  useEffect(() => {
    const next = adminHash("tenants", {
      ...(selectedTenantId ? { tenant: selectedTenantId } : {}),
      ...(selectedTenantId && tab !== "org" ? { tab } : {})
    });
    if (window.location.hash !== next) {
      window.history.replaceState(null, "", `${window.location.pathname}${window.location.search}${next}`);
    }
  }, [selectedTenantId, tab]);

  const selectedTenant = (consoleData?.tenants ?? []).find((tenant) => tenant.id === selectedTenantId) ?? null;
  const selectedRows = useMemo(
    () => rows.filter((row) => row.tenantId === selectedTenantId),
    [rows, selectedTenantId]
  );

  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ id: "", name: "", parentTenantId: "" });
  const [createError, setCreateError] = useState("");
  const [acting, setActing] = useState(false);

  async function submitCreate() {
    const id = createForm.id.trim();
    const name = createForm.name.trim();
    if (!id || !name) {
      setCreateError(t("rd.tenants.validation"));
      return;
    }
    setActing(true);
    setCreateError("");
    try {
      await createTenant({
        id,
        name,
        parentTenantId: createForm.parentTenantId.trim() || undefined
      });
      toast(tx(t, "rd.tenants.created", { name }));
      setCreateOpen(false);
      setCreateForm({ id: "", name: "", parentTenantId: "" });
      await onRetry();
    } catch (error) {
      const presentation = apiErrorPresentation(t, language, error, "rd.tenants.createFailed");
      setCreateError(presentation.detail || presentation.next);
    } finally {
      setActing(false);
    }
  }

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t("rd.nav.tenants")}</h1>
          <p>{t("rd.page.tenants.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      <div className="stack">
        <div className="grid grid-3">
          <Kpi icon={<Building2 aria-hidden="true" size={17} />} label={t("rd.tenants.kpi.tenants")} value={summary.tenants} />
          <Kpi icon={<Layers aria-hidden="true" size={17} />} label={t("rd.tenants.kpi.workspaces")} value={summary.workspaces} />
          <Kpi icon={<Users aria-hidden="true" size={17} />} label={t("rd.tenants.kpi.subjects")} value={summary.subjects} />
        </div>

        <Card
          flush
          right={<Button onClick={() => setCreateOpen(true)} size="sm" variant="primary">{t("rd.tenants.create")}</Button>}
          title={t("rd.tenants.tableTitle")}
        >
          <Table
            caption={t("rd.tenants.tableTitle")}
            empty={<EmptyState desc={t("rd.tenants.emptyDesc")} title={t("rd.tenants.empty")} />}
            renderCell={(row, column) => renderTenantCell(row, column, t, setSelectedTenantId)}
            rowKey={(row) => `${row.tenantId}:${row.workspaceId}`}
            rows={rows}
            tableId="adminTenants"
          />
        </Card>

        {selectedTenantId ? (
          <Card
            right={
              selectedTenant ? <Chip tone={selectedTenant.status === "disabled" ? "danger" : "success"}>{t(selectedTenant.status === "disabled" ? "rd.status.inactive" : "rd.status.active")}</Chip> : null
            }
            title={selectedTenant ? selectedTenant.name : selectedTenantId}
          >
            <div className="stack">
              <div className="tab-row" role="tablist">
                {(["org", "center", "profile"] as const).map((key) => (
                  <button
                    aria-selected={tab === key}
                    className={tab === key ? "tab-btn active" : "tab-btn"}
                    key={key}
                    onClick={() => setTab(key)}
                    role="tab"
                    type="button"
                  >
                    {t(tenantDetailTabKeys[key])}
                  </button>
                ))}
              </div>

              {tab === "org" ? (
                <OrgTab
                  rows={selectedRows}
                  subjects={catalog.subjects}
                  t={t}
                  tenant={selectedTenant}
                  tenantId={selectedTenantId}
                />
              ) : null}
              {tab === "center" ? <CenterTab detail={detail} t={t} /> : null}
              {tab === "profile" ? <ProfileTab detail={detail} language={language} t={t} /> : null}
            </div>
          </Card>
        ) : null}
      </div>

      <Modal
        footer={
          <>
            <Button onClick={() => setCreateOpen(false)} variant="ghost">{t("rd.apr.cancel")}</Button>
            <Button disabled={acting} onClick={() => void submitCreate()} variant="primary">{t("rd.tenants.create")}</Button>
          </>
        }
        onClose={() => setCreateOpen(false)}
        open={createOpen}
        title={t("rd.tenants.createTitle")}
      >
        <form
          className="stack"
          onSubmit={(event) => {
            event.preventDefault();
            void submitCreate();
          }}
        >
          <Field hint={t("rd.tenants.field.idHint")} htmlFor="tenant-id" label={t("rd.tenants.field.id")}>
            <input
              className="input mono"
              id="tenant-id"
              onChange={(event) => setCreateForm({ ...createForm, id: event.target.value })}
              value={createForm.id}
            />
          </Field>
          <Field htmlFor="tenant-name" label={t("rd.tenants.field.name")}>
            <input
              className="input"
              id="tenant-name"
              onChange={(event) => setCreateForm({ ...createForm, name: event.target.value })}
              value={createForm.name}
            />
          </Field>
          <Field htmlFor="tenant-parent" label={t("rd.tenants.field.parent")}>
            <select
              className="select"
              id="tenant-parent"
              onChange={(event) => setCreateForm({ ...createForm, parentTenantId: event.target.value })}
              value={createForm.parentTenantId}
            >
              <option value="">—</option>
              {(consoleData?.tenants ?? []).map((tenant) => (
                <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
              ))}
            </select>
          </Field>
          {createError ? <Banner desc={createError} title={t("rd.error.other.title")} tone="danger" /> : null}
        </form>
      </Modal>
    </>
  );
}

function OrgTab({
  rows,
  subjects,
  t,
  tenant,
  tenantId
}: {
  rows: TenantDirectoryRow[];
  subjects: { id: string; labelKey: string }[];
  t: ReturnType<typeof useRedesignI18n>["t"];
  tenant: Tenant | null;
  tenantId: string;
}) {
  const path = tenant?.parentTenantId ? `${tenant.parentTenantId} / ${tenant.name}` : tenant?.name ?? tenantId;
  const items: KvItem[] = [
    { key: "id", label: t("rd.tenants.org.id"), value: <span className="mono small">{tenantId}</span> },
    { key: "path", label: t("rd.tenants.org.path"), value: path },
    ...(tenant ? [{ key: "level", label: t("rd.tenants.org.level"), value: String(tenant.level) }] : []),
    { key: "resources", label: t("rd.tenants.org.resources"), value: String(rows.reduce((sum, row) => sum + row.agentCount, 0)) }
  ];
  return (
    <div className="stack">
      <KvList items={items} />
      <div className="section-title">{t("rd.tenants.org.workspacesTitle")}</div>
      {rows.length === 0 ? (
        <EmptyState desc={t("rd.tenants.org.workspacesEmptyDesc")} title={t("rd.tenants.org.workspacesEmpty")} />
      ) : (
        <div className="rule-list">
          {rows.map((row) => (
            <div className="rule-line" key={row.workspaceId}>
              <div className="rule-main">
                <div className="rule-name mono">{row.workspaceId}</div>
                <div className="rule-sub">{tx(t, "rd.tenants.org.workspaceSub", { count: row.agentCount })}</div>
              </div>
              <Button
                href={adminHash("registry", { tenant: row.tenantId, workspace: row.workspaceId })}
                size="sm"
                variant="ghost"
              >
                {t("rd.tenants.org.registerResource")}
              </Button>
            </div>
          ))}
        </div>
      )}
      <div className="section-title">{t("rd.tenants.org.subjectsTitle")}</div>
      <p className="muted small">{tx(t, "rd.tenants.org.subjectsSub", { count: subjects.length })}</p>
      <div className="chip-row">
        {subjects.slice(0, 8).map((subject) => (
          <Chip key={subject.id} tone="neutral">{t(subject.labelKey)}</Chip>
        ))}
      </div>
    </div>
  );
}

function CenterTab({
  detail,
  t
}: {
  detail: ReturnType<typeof useTenantDetail>;
  t: ReturnType<typeof useRedesignI18n>["t"];
}) {
  if (detail.loading) return <LoadingState label={t("rd.common.loading")} />;
  const center = detail.center;
  if (!center) return <EmptyState desc={t("rd.tenants.center.emptyDesc")} title={t("rd.tenants.center.empty")} />;
  const packages = center.permissionPackages ?? [];
  const allowed = center.capabilities.filter((capability) => capability.effect === "allow").length;
  const blocked = center.capabilities.filter((capability) => capability.effect === "deny").length;
  return (
    <div className="stack">
      <div className="grid grid-4">
        <Kpi icon={<Layers aria-hidden="true" size={17} />} label={t("rd.tenants.center.kpi.packages")} value={packages.length} />
        <Kpi icon={<Users aria-hidden="true" size={17} />} label={t("rd.tenants.center.kpi.admins")} value={center.administrators.length} />
        <Kpi icon={<Layers aria-hidden="true" size={17} />} label={t("rd.tenants.center.kpi.allowed")} value={allowed} />
        <Kpi icon={<Layers aria-hidden="true" size={17} />} label={t("rd.tenants.center.kpi.blocked")} value={blocked} />
      </div>
      <div className="section-title">{t("rd.tenants.center.capabilitiesTitle")}</div>
      {center.capabilities.length === 0 ? (
        <EmptyState desc={t("rd.tenants.center.capabilitiesEmptyDesc")} title={t("rd.tenants.center.capabilitiesEmpty")} />
      ) : (
        <div className="rule-list">
          {center.capabilities.slice(0, 8).map((capability) => (
            <div className="rule-line" key={`${capability.targetId}:${capability.capabilityId}`}>
              <div className="rule-main">
                <div className="rule-name">{capability.capabilityName}</div>
                <div className="rule-sub mono">
                  {capability.targetName} · {capability.workspaceIds.join(" / ") || "—"}
                </div>
              </div>
              <Chip tone={capability.effect === "deny" ? "danger" : "success"}>
                {capability.effect === "deny" ? t("rd.decision.blocked") : t("rd.decision.allowed")}
              </Chip>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ProfileTab({
  detail,
  language,
  t
}: {
  detail: ReturnType<typeof useTenantDetail>;
  language: ReturnType<typeof useRedesignI18n>["language"];
  t: ReturnType<typeof useRedesignI18n>["t"];
}) {
  if (detail.loading) return <LoadingState label={t("rd.common.loading")} />;
  const profile = detail.profile;
  if (!profile) return <EmptyState desc={t("rd.tenants.profile.emptyDesc")} title={t("rd.tenants.profile.empty")} />;
  return (
    <div className="stack">
      <div className="grid grid-4">
        <Kpi icon={<Layers aria-hidden="true" size={17} />} label={t("rd.tenants.profile.kpi.grants")} value={profile.summary.grantCount} />
        <Kpi icon={<Layers aria-hidden="true" size={17} />} label={t("rd.tenants.profile.kpi.workspaces")} value={profile.summary.workspaceAssignmentCount} />
        <Kpi icon={<Users aria-hidden="true" size={17} />} label={t("rd.tenants.profile.kpi.callers")} value={profile.summary.instanceAssignmentCount} />
        <Kpi
          hint={t("rd.tenants.profile.kpi.tracesHint")}
          icon={<Users aria-hidden="true" size={17} />}
          label={t("rd.tenants.profile.kpi.traces")}
          value={`${profile.summary.recentAllowedTraceCount}/${profile.summary.recentDeniedTraceCount}`}
        />
      </div>
      <div className="section-title">{t("rd.tenants.profile.chainTitle")}</div>
      {profile.grants.length === 0 ? (
        <EmptyState desc={t("rd.tenants.profile.chainEmptyDesc")} title={t("rd.tenants.profile.chainEmpty")} />
      ) : (
        <div className="cap-list">
          {profile.grants.map((grant) => (
            <div className="cap-row" key={grant.tenantEntitlement.id}>
              <div className="cap-main">
                <div className="cap-name">
                  {grant.capability?.displayName ?? grant.tenantEntitlement.capabilityId}
                  {" · "}
                  <span className="mono small">{grant.target?.name ?? grant.tenantEntitlement.targetId}</span>
                </div>
                <div className="cap-sub mono">{grant.tenantEntitlement.id}</div>
                {grant.workspaceAssignments.map((workspace) => (
                  <div className="cap-sub" key={workspace.workspaceAssignment.id}>
                    <span className="mono">{workspace.workspaceAssignment.workspaceId}</span>
                    {" · "}
                    <Chip tone={workspace.scopeStatus === "invalid" ? "danger" : "success"}>
                      {workspace.scopeStatus === "invalid" ? t("rd.tenants.profile.invalid") : t("rd.tenants.profile.valid")}
                    </Chip>
                    {workspace.instanceAssignments.map((instance) => (
                      <div className="cap-sub mono" key={instance.instanceAssignment.id}>
                        {instance.callerInstance?.name ?? instance.instanceAssignment.callerInstanceId}
                        {" → "}
                        {instance.instanceAssignment.subjectSelector}
                        {" · "}
                        <Chip tone={instance.scopeStatus === "invalid" ? "danger" : "success"}>
                          {instance.scopeStatus === "invalid" ? t("rd.tenants.profile.invalid") : t("rd.tenants.profile.valid")}
                        </Chip>
                      </div>
                    ))}
                  </div>
                ))}
              </div>
              <span className="small muted">{formatDateSafe(grant.tenantEntitlement.createdAt, language)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function formatDateSafe(value: string, language: ReturnType<typeof useRedesignI18n>["language"]): string {
  try {
    return new Date(value).toLocaleDateString(language === "zh-CN" ? "zh-CN" : "en");
  } catch {
    return value;
  }
}

function renderTenantCell(
  row: TenantDirectoryRow,
  column: TenantColumnKey,
  t: ReturnType<typeof useRedesignI18n>["t"],
  onSelect: (tenantId: string) => void
) {
  switch (column) {
    case "tenant":
      return (
        <div>
          <div className="cell-main">{row.tenantName}</div>
          <div className="cell-sub mono">{row.tenantId}</div>
        </div>
      );
    case "workspace":
      return <span className="mono small">{row.workspaceId || "—"}</span>;
    case "resourceCount":
      return <span className="mono small">{row.agentCount}</span>;
    case "grants":
      return row.allowCount + row.denyCount === 0 ? (
        <span className="muted small">—</span>
      ) : (
        <span className="small">
          {tx(t, "rd.tenants.grantsSummary", { allow: row.allowCount, deny: row.denyCount })}
        </span>
      );
    case "status":
      return <Chip tone={row.tenantStatus === "disabled" ? "danger" : "success"}>{t(row.tenantStatus === "disabled" ? "rd.status.inactive" : "rd.status.active")}</Chip>;
    case "actions":
      return (
        <Button onClick={() => onSelect(row.tenantId)} size="sm" variant="link">
          {t("rd.tenants.manage")}
        </Button>
      );
  }
}
