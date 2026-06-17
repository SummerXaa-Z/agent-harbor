import { useState, type ReactNode } from "react";
import {
  ArrowRight,
  Boxes,
  CheckCircle2,
  DatabaseZap,
  KeyRound,
  RefreshCw,
  ShieldCheck
} from "lucide-react";

import { permissionEntityDisplayName, type Tone, type Translator } from "../consolePresenters";
import type { ManagementMutationRefreshState } from "../managementMutationRefresh";
import type { ResourceLifecycleItem, ResourceLifecycleStatus, ResourceLifecycleSummary } from "../resourceLifecycle";
import { Badge, EmptyRow } from "./ui";

export function ResourceLifecycleView({
  formatTenantName = (tenantId) => tenantId,
  formatWorkspaceName = (workspaceId) => workspaceId,
  lastRefreshedAt,
  onResourceAction,
  onRefresh,
  primaryActions,
  refreshState = { status: "idle" },
  secondaryActions,
  summary,
  t
}: {
  formatTenantName?: (tenantId: string) => string;
  formatWorkspaceName?: (workspaceId: string) => string;
  lastRefreshedAt?: Date;
  onResourceAction?: (item: ResourceLifecycleItem) => void;
  onRefresh?: () => void;
  primaryActions?: ReactNode;
  refreshState?: ManagementMutationRefreshState;
  secondaryActions?: ReactNode;
  summary: ResourceLifecycleSummary;
  t: Translator;
}) {
  const [selectedResourceId, setSelectedResourceId] = useState("");
  const hasResources = summary.items.length > 0;
  const defaultSelectedItem = summary.items.find((item) => item.status !== "ready") ?? summary.items[0];
  const selectedItem = summary.items.find((item) => item.id === selectedResourceId) ?? defaultSelectedItem;
  const metrics = [
    { label: t("resource.metric.total"), value: summary.totalResources },
    { label: t("resource.metric.active"), value: summary.activeResources },
    { label: t("resource.metric.mcpTargets"), value: summary.mcpTargets },
    { label: t("resource.metric.ready"), value: summary.readyResources },
    { label: t("resource.metric.needsAttention"), value: summary.needsAttention }
  ];
  const stages = [
    { icon: <Boxes size={15} />, label: t("resource.stage.register") },
    { icon: <KeyRound size={15} />, label: t("resource.stage.connect") },
    { icon: <DatabaseZap size={15} />, label: t("resource.stage.discover") },
    { icon: <ShieldCheck size={15} />, label: t("resource.stage.authorize") },
    { icon: <CheckCircle2 size={15} />, label: t("resource.stage.validate") }
  ];

  return (
    <div className="resource-lifecycle">
      <section className="resource-lifecycle-hero">
        <div>
          <span className="section-kicker">{t("resource.kicker")}</span>
          <h3>{t("resource.title")}</h3>
          <p>{t("resource.subtitle")}</p>
        </div>
        <div className="resource-lifecycle-metrics" aria-label={t("resource.metricsAria")}>
          {metrics.map((metric) => (
            <span key={metric.label}>
              <strong>{metric.value}</strong>
              {metric.label}
            </span>
          ))}
        </div>
      </section>

      <div className="resource-lifecycle-stages" aria-label={t("resource.stagesAria")}>
        {stages.map((stage) => (
          <span key={stage.label}>
            {stage.icon}
            {stage.label}
          </span>
        ))}
      </div>

      {primaryActions || secondaryActions ? (
        <section className="resource-lifecycle-command-center" aria-label={t("resource.commandAria")}>
          <div className="resource-lifecycle-command-copy">
            <span className="section-kicker">{t("resource.commandKicker")}</span>
            <strong>{t("resource.commandTitle")}</strong>
            <p>{t("resource.commandDetail")}</p>
          </div>
          {primaryActions}
          {secondaryActions}
          <ResourceLifecycleRefreshStatus
            lastRefreshedAt={lastRefreshedAt}
            onRefresh={onRefresh}
            refreshState={refreshState}
            t={t}
          />
        </section>
      ) : null}

      {selectedItem ? (
        <ResourceLifecycleContextPanel
          formatTenantName={formatTenantName}
          formatWorkspaceName={formatWorkspaceName}
          item={selectedItem}
          onResourceAction={onResourceAction}
          t={t}
        />
      ) : null}

      <section className={`resource-lifecycle-list${hasResources ? "" : " is-empty"}`} aria-label={t("resource.listAria")}>
        <div className="resource-lifecycle-list-title">
          <strong>{t("resource.listTitle")}</strong>
          <span>{tx(t, "resource.listCount", { count: summary.items.length })}</span>
        </div>
        {hasResources ? (
          <div className="resource-lifecycle-header" aria-hidden="true">
            <span>{t("resource.column.resource")}</span>
            <span>{t("resource.column.status")}</span>
            <span>{t("resource.column.capabilities")}</span>
            <span>{t("resource.column.permission")}</span>
            <span>{t("resource.column.runtime")}</span>
            <span>{t("resource.column.next")}</span>
          </div>
        ) : null}
        {!hasResources ? (
          <div className="resource-lifecycle-empty">
            <EmptyRow
              detail={t("resource.empty.detail")}
              title={t("resource.empty.title")}
            />
          </div>
        ) : (
          summary.items.map((item) => (
            <ResourceLifecycleRow
              item={item}
              key={item.id}
              onResourceAction={onResourceAction}
              onSelectResource={setSelectedResourceId}
              selected={item.id === selectedItem?.id}
              t={t}
            />
          ))
        )}
      </section>
    </div>
  );
}

function ResourceLifecycleRefreshStatus({
  lastRefreshedAt,
  onRefresh,
  refreshState,
  t
}: {
  lastRefreshedAt?: Date;
  onRefresh?: () => void;
  refreshState: ManagementMutationRefreshState;
  t: Translator;
}) {
  const stale = refreshState.status === "stale";
  const refreshing = refreshState.status === "refreshing";
  const refreshedAt = refreshState.status === "fresh" ? new Date(refreshState.refreshedAt) : lastRefreshedAt;
  const detail = stale
    ? refreshState.errorMessage || t("resource.refreshStatus.staleDetail")
    : refreshedAt
      ? tx(t, "resource.refreshStatus.detail", { time: formatRefreshTime(refreshedAt) })
      : t("resource.refreshStatus.idleDetail");
  const title = refreshing
    ? t("resource.refreshStatus.refreshing")
    : stale
      ? t("resource.refreshStatus.stale")
      : t("resource.refreshStatus.fresh");

  return (
    <div className={`resource-lifecycle-refresh-status is-${refreshState.status}`} role={stale ? "alert" : "status"}>
      <span>
        <small>{t("resource.refreshStatus.title")}</small>
        <strong>{title}</strong>
        <em>{detail}</em>
      </span>
      {onRefresh ? (
        <button className="secondary-button" disabled={refreshing} onClick={onRefresh} type="button">
          <RefreshCw size={14} />
          {refreshing ? t("action.loading") : t("action.refresh")}
        </button>
      ) : null}
    </div>
  );
}

function ResourceLifecycleContextPanel({
  formatTenantName,
  formatWorkspaceName,
  item,
  onResourceAction,
  t
}: {
  formatTenantName: (tenantId: string) => string;
  formatWorkspaceName: (workspaceId: string) => string;
  item: ResourceLifecycleItem;
  onResourceAction?: (item: ResourceLifecycleItem) => void;
  t: Translator;
}) {
  const requiresAction = item.status !== "ready";
  const actionClassName = `${requiresAction ? "primary-button" : "secondary-button"} resource-lifecycle-action`;

  return (
    <section className="resource-lifecycle-context-panel" aria-label={t("resource.contextTitle")}>
      <div className="resource-lifecycle-context-main">
        <span className="section-kicker">{t("resource.contextTitle")}</span>
        <strong>{permissionEntityDisplayName(item.name, t)}</strong>
        <p>{t("resource.contextDetail")}</p>
      </div>
      <div className="resource-lifecycle-context-grid">
        <span>
          <small>{t("resource.contextScope")}</small>
          <strong>{formatTenantName(item.tenantId)}</strong>
          <em>{formatWorkspaceName(item.workspaceId)}</em>
        </span>
        <span>
          <small>{t("resource.contextHealth")}</small>
          <Badge tone={statusTone(item.status)}>{t(item.statusKey)}</Badge>
          <em>{t(item.detailKey)}</em>
        </span>
        <span>
          <small>{t("resource.column.capabilities")}</small>
          <strong>{item.approvedCapabilityCount}/{item.capabilityCount}</strong>
          <em>{t("resource.stage.discover")}</em>
        </span>
        <span>
          <small>{t("resource.column.permission")}</small>
          <strong>{item.grantCount}</strong>
          <em>{t("resource.stage.authorize")}</em>
        </span>
        <span>
          <small>{t("resource.column.runtime")}</small>
          <strong>{item.runtimeDecisionCount}</strong>
          <em>{t("resource.stage.validate")}</em>
        </span>
      </div>
      <div className="resource-lifecycle-context-next">
        <span>{t("resource.contextNext")}</span>
        {onResourceAction ? (
          <button
            className={actionClassName}
            type="button"
            onClick={() => onResourceAction(item)}
          >
            {t(item.nextActionKey)}
            <ArrowRight size={14} />
          </button>
        ) : (
          <a className={actionClassName} href={item.nextActionHash}>
            {t(item.nextActionKey)}
            <ArrowRight size={14} />
          </a>
        )}
      </div>
    </section>
  );
}

function ResourceLifecycleRow({
  item,
  onResourceAction,
  onSelectResource,
  selected,
  t
}: {
  item: ResourceLifecycleItem;
  onResourceAction?: (item: ResourceLifecycleItem) => void;
  onSelectResource: (resourceId: string) => void;
  selected: boolean;
  t: Translator;
}) {
  const requiresAction = item.status !== "ready";
  const actionClassName = `${requiresAction ? "primary-button" : "secondary-button"} resource-lifecycle-action`;

  return (
    <div className={`resource-lifecycle-row${selected ? " is-selected" : ""}`}>
      <div className="resource-lifecycle-name">
        <button
          aria-pressed={selected}
          className="resource-lifecycle-resource-button"
          type="button"
          onClick={() => onSelectResource(item.id)}
        >
          <strong>{permissionEntityDisplayName(item.name, t)}</strong>
          <span>{t(item.kindKey)}</span>
          <span>{t(item.detailKey)}</span>
        </button>
      </div>
      <Badge tone={statusTone(item.status)}>{t(item.statusKey)}</Badge>
      <span>{item.approvedCapabilityCount}/{item.capabilityCount}</span>
      <span>{item.grantCount}</span>
      <span>{item.runtimeDecisionCount}</span>
      {onResourceAction ? (
        <button
          className={actionClassName}
          type="button"
          onClick={() => onResourceAction(item)}
        >
          {t(item.nextActionKey)}
          <ArrowRight size={14} />
        </button>
      ) : (
        <a className={actionClassName} href={item.nextActionHash}>
          {t(item.nextActionKey)}
          <ArrowRight size={14} />
        </a>
      )}
    </div>
  );
}

function statusTone(status: ResourceLifecycleStatus): Tone {
  if (status === "ready") return "success";
  if (status === "disabled") return "neutral";
  if (status === "needs_runtime") return "info";
  if (status === "needs_credentials") return "warning";
  return "danger";
}

function formatRefreshTime(value: Date) {
  return value.toLocaleTimeString("zh-CN", { hour12: false });
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}
