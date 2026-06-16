import type { ReactNode } from "react";
import {
  ArrowRight,
  Boxes,
  CheckCircle2,
  DatabaseZap,
  KeyRound,
  ShieldCheck
} from "lucide-react";

import { permissionEntityDisplayName, type Tone, type Translator } from "../consolePresenters";
import type { ResourceLifecycleItem, ResourceLifecycleStatus, ResourceLifecycleSummary } from "../resourceLifecycle";
import { Badge, EmptyRow } from "./ui";

export function ResourceLifecycleView({
  onResourceAction,
  primaryActions,
  secondaryActions,
  summary,
  t
}: {
  onResourceAction?: (item: ResourceLifecycleItem) => void;
  primaryActions?: ReactNode;
  secondaryActions?: ReactNode;
  summary: ResourceLifecycleSummary;
  t: Translator;
}) {
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
        </section>
      ) : null}

      <section className="resource-lifecycle-list" aria-label={t("resource.listAria")}>
        <div className="resource-lifecycle-header" aria-hidden="true">
          <span>{t("resource.column.resource")}</span>
          <span>{t("resource.column.status")}</span>
          <span>{t("resource.column.capabilities")}</span>
          <span>{t("resource.column.permission")}</span>
          <span>{t("resource.column.runtime")}</span>
          <span>{t("resource.column.next")}</span>
        </div>
        {summary.items.length === 0 ? (
          <EmptyRow
            actionHash="#getting-started"
            actionLabel={t("empty.registry.action")}
            detail={t("resource.empty.detail")}
            title={t("resource.empty.title")}
          />
        ) : (
          summary.items.map((item) => (
            <ResourceLifecycleRow
              item={item}
              key={item.id}
              onResourceAction={onResourceAction}
              t={t}
            />
          ))
        )}
      </section>
    </div>
  );
}

function ResourceLifecycleRow({
  item,
  onResourceAction,
  t
}: {
  item: ResourceLifecycleItem;
  onResourceAction?: (item: ResourceLifecycleItem) => void;
  t: Translator;
}) {
  const requiresAction = item.status !== "ready";
  const actionClassName = `${requiresAction ? "primary-button" : "secondary-button"} resource-lifecycle-action`;

  return (
    <div className="resource-lifecycle-row">
      <div className="resource-lifecycle-name">
        <strong>{permissionEntityDisplayName(item.name, t)}</strong>
        <span>{t(item.kindKey)}</span>
        <span>{t(item.detailKey)}</span>
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
