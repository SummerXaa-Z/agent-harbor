import {
  Download,
  RefreshCw,
  ShieldCheck
} from "lucide-react";

import {
  permissionEntityDisplayName,
  readableIdentifierLabel,
  type Tone,
  type Translator
} from "../consolePresenters";
import type { AiAdminProductionConsoleStatus, AiAdminProductionConsoleSummary } from "../aiAdminProductionConsole";
import type {
  PermissionPackageDraft,
  PermissionPackageDraftInput,
  PermissionPackageProductionReadiness,
  PermissionPackageTemplate
} from "../permissionPackages";
import type {
  Agent,
  Tenant
} from "../types";
import { Badge } from "./ui";

const defaultTenantId = "default";
const defaultWorkspaceId = "workspace-sandbox";

export function GoLiveAcceptanceOverview({
  agents,
  draft,
  form,
  liveDataAvailable,
  onExportProductionEvidence,
  onOpenPermissionChange,
  onRefreshProductionReadiness,
  productionEvidenceExporting,
  productionReadiness,
  productionReadinessLoading,
  productionReadinessMessage,
  productionSummary,
  templates,
  tenants,
  t
}: {
  agents: Agent[];
  draft: PermissionPackageDraft | null;
  form: PermissionPackageDraftInput;
  liveDataAvailable: boolean;
  onExportProductionEvidence: () => void;
  onOpenPermissionChange: () => void;
  onRefreshProductionReadiness: () => void;
  productionEvidenceExporting: boolean;
  productionReadiness: PermissionPackageProductionReadiness | null;
  productionReadinessLoading: boolean;
  productionReadinessMessage: string;
  productionSummary: AiAdminProductionConsoleSummary;
  templates: PermissionPackageTemplate[];
  tenants: Tenant[];
  t: Translator;
}) {
  const acceptanceInput = draft?.input ?? form;
  const acceptanceTemplate = draft?.template;
  const tenantPath = permissionTenantPathLabel(acceptanceInput.tenantId, tenants, t);
  const template = acceptanceTemplate ?? templates.find((item) => item.id === acceptanceInput.templateId);
  const caller = agents.find((agent) => agent.id === acceptanceInput.callerInstanceId);
  const target = agents.find((agent) => agent.id === acceptanceInput.targetId);
  const workspaceName = permissionWorkspaceDisplayName(acceptanceInput.workspaceId, agents, t);
  const templateName = template
    ? permissionPackageTemplateName(template, t)
    : permissionPackageTemplateNameById(acceptanceInput.templateId, t);
  const callerName = caller
    ? permissionEntityDisplayName(caller.name, t)
    : acceptanceInput.callerInstanceId
      ? t("text.selectedCallerFallback")
      : t("text.callerPendingSelection");
  const targetName = target
    ? permissionEntityDisplayName(target.name, t)
    : acceptanceInput.targetId
      ? t("text.selectedTargetFallback")
      : t("text.targetPendingSelection");
  const readinessStatusLabel = productionReadinessStatusLabel(productionReadiness?.status, t);
  const statusLabel = productionReadiness ? readinessStatusLabel : productionConsoleStatusLabel(productionSummary, t);
  const statusTone = productionReadiness
    ? productionReadinessStatusTone(productionReadiness.status)
    : productionConsoleStatusTone(productionSummary.status);
  const nextAction = productionReadiness?.nextActions[0]
    ? permissionProductionReadinessNextAction(productionReadiness.nextActions[0], t)
    : productionReadiness?.status === "ready"
      ? t("text.productionReadinessReadyDetail")
      : productionReadiness
        ? t("text.productionReadinessPendingDetail")
        : t("text.goLiveAcceptanceNoReadinessDetail");
  const readyCount = productionReadiness?.summary.readyCount
    ?? productionSummary.steps.filter((step) => step.status === "ready").length;
  const totalCount = productionReadiness?.checks.length ?? productionSummary.steps.length;
  const blockerCount = productionReadiness?.summary.blockingCount
    ?? productionSummary.steps.filter((step) => step.status === "blocked").length;
  const warningCount = productionReadiness?.summary.warningCount
    ?? productionSummary.steps.filter((step) => step.status === "needs_review" || step.status === "pending").length;
  const acceptanceReady = productionReadiness?.status === "ready";
  const statusMessage = productionReadinessMessage === t("message.permissionProductionReadinessLoaded")
    ? ""
    : productionReadinessMessage;

  return (
    <div className="go-live-acceptance">
      <section className="go-live-acceptance-main">
        <div className="go-live-acceptance-heading">
          <span>{t("text.goLiveAcceptanceTaskTitle")}</span>
          <Badge tone={statusTone}>{statusLabel}</Badge>
        </div>
        <p>{nextAction}</p>
        {!liveDataAvailable ? <p className="go-live-acceptance-warning">{t("message.fallbackDataModeDetail")}</p> : null}
        {statusMessage ? <p className="go-live-acceptance-message">{statusMessage}</p> : null}
        <div className="go-live-acceptance-actions">
          {acceptanceReady ? (
            <>
              <button className="primary-button" disabled={!liveDataAvailable || productionEvidenceExporting} onClick={onExportProductionEvidence} type="button">
                <Download size={14} />
                {productionEvidenceExporting ? t("action.exportingProductionEvidence") : t("action.exportProductionEvidence")}
              </button>
              <button className="secondary-button" disabled={!liveDataAvailable || productionReadinessLoading} onClick={onRefreshProductionReadiness} type="button">
                <RefreshCw size={14} />
                {productionReadinessLoading ? t("action.checkingProductionReadiness") : t("action.checkProductionReadiness")}
              </button>
            </>
          ) : (
            <>
              <button className="primary-button" disabled={!liveDataAvailable || productionReadinessLoading} onClick={onRefreshProductionReadiness} type="button">
                <RefreshCw size={14} />
                {productionReadinessLoading ? t("action.checkingProductionReadiness") : t("action.checkProductionReadiness")}
              </button>
              <button className="secondary-button" disabled={!liveDataAvailable || !productionReadiness || productionEvidenceExporting} onClick={onExportProductionEvidence} type="button">
                <Download size={14} />
                {productionEvidenceExporting ? t("action.exportingProductionEvidence") : t("action.exportProductionEvidence")}
              </button>
            </>
          )}
          <button className="secondary-button" onClick={onOpenPermissionChange} type="button">
            <ShieldCheck size={14} />
            {t("action.openPermissionChange")}
          </button>
        </div>
      </section>

      <aside className="go-live-acceptance-context" aria-label={t("text.goLiveAcceptanceContext")}>
        <strong>{t("text.goLiveAcceptanceContext")}</strong>
        <dl>
          <div>
            <dt>{t("form.businessTenant")}</dt>
            <dd>{tenantPath.primary}</dd>
          </div>
          <div>
            <dt>{t("form.businessWorkspace")}</dt>
            <dd>{workspaceName}</dd>
          </div>
          <div>
            <dt>{t("form.businessCaller")}</dt>
            <dd>{callerName} → {targetName}</dd>
          </div>
          <div>
            <dt>{t("form.permissionPackage")}</dt>
            <dd>{templateName}</dd>
          </div>
        </dl>
      </aside>

      <section className="go-live-acceptance-checks" aria-label={t("section.permissionRequestProcess")}>
        <div className="go-live-acceptance-score">
          <div>
            <span>{t("metric.productionReadyChecks")}</span>
            <strong>{readyCount}/{totalCount}</strong>
          </div>
          <div>
            <span>{t("metric.productionWarnings")}</span>
            <strong>{warningCount}</strong>
          </div>
          <div>
            <span>{t("metric.productionBlockers")}</span>
            <strong>{blockerCount}</strong>
          </div>
        </div>
        <ol className="go-live-step-list">
          {productionSummary.steps.map((step) => (
            <li key={step.key}>
              <span className={`go-live-step-dot tone-${productionConsoleStatusTone(step.status)}`} aria-hidden="true" />
              <div>
                <strong>{t(step.labelKey)}</strong>
                <span>{step.detailKey ? t(step.detailKey) : step.detail}</span>
              </div>
            </li>
          ))}
        </ol>
      </section>
    </div>
  );
}

function permissionTenantPathLabel(tenantId: string, tenants: Tenant[], t: Translator): { path: string; primary: string } {
  const normalizedTenantId = tenantId.trim();
  if (!normalizedTenantId) return { path: "-", primary: t("text.unresolvedTenant") };
  const tenantById = tenants.reduce<Record<string, Tenant>>((acc, tenant) => {
    acc[tenant.id] = tenant;
    return acc;
  }, {});
  const selected = tenantById[normalizedTenantId];
  if (!selected) {
    if (normalizedTenantId === defaultTenantId) {
      const defaultTenantName = t("text.defaultTenantName");
      return { path: defaultTenantName, primary: defaultTenantName };
    }
    return {
      path: tx(t, "text.unresolvedTenantDetail", { id: normalizedTenantId }),
      primary: t("text.unresolvedTenant")
    };
  }

  const path: Tenant[] = [];
  const visited = new Set<string>();
  let current: Tenant | undefined = selected;
  while (current && !visited.has(current.id)) {
    path.unshift(current);
    visited.add(current.id);
    current = current.parentTenantId ? tenantById[current.parentTenantId] : undefined;
  }
  const names = path.map((tenant) => permissionEntityDisplayName(tenant.name.trim() || tenant.id, t));
  return {
    path: names.join(" > "),
    primary: permissionEntityDisplayName(selected.name.trim() || selected.id, t)
  };
}

function permissionWorkspaceDisplayName(workspaceId: string, agents: Agent[], t: Translator) {
  const normalizedWorkspaceId = workspaceId.trim();
  if (!normalizedWorkspaceId) return "-";
  const agentInWorkspace = agents.find((agent) => agent.workspaceId === normalizedWorkspaceId);
  if (normalizedWorkspaceId === defaultWorkspaceId || agentInWorkspace?.workspaceId === defaultWorkspaceId) {
    return t("text.defaultWorkspaceName");
  }
  if (/permission[-_]?(request|package)[-_]?approval/i.test(normalizedWorkspaceId)) {
    return t("demo.permissionRequestWorkspace");
  }
  if (/core[-_]?journey/i.test(normalizedWorkspaceId)) {
    return t("demo.coreJourneyWorkspace");
  }
  return permissionEntityDisplayName(readableIdentifierLabel(normalizedWorkspaceId), t);
}

function permissionPackageTemplateName(template: PermissionPackageTemplate, t: Translator) {
  return t(`permissionPackage.${template.id}.name`, template.name);
}

function permissionPackageTemplateNameById(templateId: string, t: Translator) {
  return t(`permissionPackage.${templateId}.name`, templateId);
}

function productionReadinessStatusLabel(status: PermissionPackageProductionReadiness["status"] | undefined, t: Translator) {
  if (status === "ready") return t("status.productionReady");
  if (status === "needs_review") return t("status.productionNeedsReview");
  if (status === "blocked") return t("status.productionBlocked");
  return t("status.preflightPending");
}

function productionReadinessStatusTone(status: PermissionPackageProductionReadiness["status"] | undefined): Tone {
  if (status === "ready") return "success";
  if (status === "needs_review") return "warning";
  if (status === "blocked") return "danger";
  return "neutral";
}

function permissionProductionReadinessNextAction(action: string, t: Translator) {
  const known: Record<string, string> = {
    "Apply the approved permission package before production readiness.": "productionNext.applyApproved",
    "Inspect the latest permission package application scope before go-live.": "productionNext.inspectScope",
    "Production readiness evidence is complete.": "productionNext.complete",
    "Resolve apply preflight blockers before claiming production readiness.": "productionNext.resolvePreflight",
    "Resolve impact review blockers before production readiness.": "productionNext.resolveImpact",
    "Review application health and drift blockers before production readiness.": "productionNext.reviewHealth",
    "Run a denied MCP call that proves blocked tools stay blocked.": "productionNext.runDenied",
    "Run an allowed MCP call with the production subject before go-live.": "productionNext.runAllowed",
    "Verify permission package applied audit evidence before production readiness.": "productionNext.verifyAudit",
    "Verify tenant entitlement, workspace assignment, and caller assignment evidence.": "productionNext.verifyGrantChain"
  };
  const key = known[action];
  return key ? t(key) : action;
}

function productionConsoleStatusTone(status: AiAdminProductionConsoleStatus): Tone {
  if (status === "ready") return "success";
  if (status === "needs_review") return "warning";
  if (status === "blocked") return "danger";
  return "neutral";
}

function productionConsoleStatusLabel(summary: AiAdminProductionConsoleSummary, t: Translator) {
  if (summary.status === "ready") return t("status.productionReady");
  if (summary.status === "needs_review") return t("status.productionNeedsReview");
  if (summary.status === "blocked") return t("status.productionBlocked");
  if (summary.primaryActionKey === "action.createApprovalRequest") return t("status.approvalPending");
  return t("status.productionPending");
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}
