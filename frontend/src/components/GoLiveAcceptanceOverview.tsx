import {
  Download,
  RefreshCw,
  ShieldCheck
} from "lucide-react";

import {
  formatDate,
  permissionEntityDisplayName,
  readableIdentifierLabel,
  type Tone,
  type Translator
} from "../consolePresenters";
import type { AiAdminProductionConsoleSummary } from "../aiAdminProductionConsole";
import type { ConnectionDiagnosticStatus } from "../connectionDiagnostics";
import {
  buildProductionAcceptanceCenter,
  type ProductionAcceptanceAction,
  type ProductionAcceptanceStatus
} from "../productionAcceptance";
import {
  permissionProductionReadinessNextAction
} from "../productionReadinessCopy";
import type {
  PermissionPackageDraft,
  PermissionPackageDraftInput,
  PermissionPackageProductionEvidenceReport,
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
  connectionDiagnosticsChecking,
  connectionStatus,
  draft,
  form,
  liveDataAvailable,
  onExportProductionEvidence,
  onOpenPermissionChange,
  onRunConnectionDiagnostics,
  onRefreshProductionReadiness,
  productionEvidenceExporting,
  productionReport,
  productionReadiness,
  productionReadinessLoading,
  productionReadinessMessage,
  productionSummary,
  templates,
  tenants,
  t
}: {
  agents: Agent[];
  connectionDiagnosticsChecking: boolean;
  connectionStatus: ConnectionDiagnosticStatus | null;
  draft: PermissionPackageDraft | null;
  form: PermissionPackageDraftInput;
  liveDataAvailable: boolean;
  onExportProductionEvidence: () => void;
  onOpenPermissionChange: () => void;
  onRunConnectionDiagnostics: () => void;
  onRefreshProductionReadiness: () => void;
  productionEvidenceExporting: boolean;
  productionReport: PermissionPackageProductionEvidenceReport | null;
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
  const matchingProductionReport = reportMatchesAcceptanceScope(productionReport, acceptanceInput, productionReadiness)
    ? productionReport
    : null;
  const acceptanceCenter = buildProductionAcceptanceCenter({
    connectionStatus,
    liveDataAvailable,
    productionReadiness,
    productionSummary
  });
  const statusLabel = t(`productionAcceptance.status.${acceptanceCenter.status}`);
  const statusTone = productionAcceptanceStatusTone(acceptanceCenter.status);
  const nextAction = acceptanceCenter.blockers[0]
    ? t(acceptanceCenter.blockers[0].labelKey, acceptanceCenter.blockers[0].detail)
    : productionReadiness?.nextActions[0]
    ? permissionProductionReadinessNextAction(productionReadiness.nextActions[0], t)
    : productionReadiness?.status === "ready"
      ? t("text.productionReadinessReadyDetail")
      : productionReadiness
        ? t("text.productionReadinessPendingDetail")
        : t("text.goLiveAcceptanceNoReadinessDetail");
  const readyCount = acceptanceCenter.readyCount;
  const totalCount = acceptanceCenter.totalCount;
  const blockerCount = acceptanceCenter.blockingCount;
  const warningCount = acceptanceCenter.checkRows.filter((row) => row.status === "attention").length;
  const acceptanceReady = acceptanceCenter.status === "ready";
  const statusMessage = productionReadinessMessage === t("message.permissionProductionReadinessLoaded")
    ? ""
    : productionReadinessMessage;
  const primaryAction = renderProductionAcceptanceAction({
    action: acceptanceCenter.primaryAction,
    connectionDiagnosticsChecking,
    liveDataAvailable,
    onExportProductionEvidence,
    onOpenPermissionChange,
    onRefreshProductionReadiness,
    onRunConnectionDiagnostics,
    productionEvidenceExporting,
    productionReadiness,
    productionReadinessLoading,
    t
  });

  return (
    <div className="go-live-acceptance">
      <section className="go-live-acceptance-main">
        <div className="go-live-acceptance-decision">
          <div className="go-live-acceptance-copy">
            <div className="go-live-acceptance-heading">
              <span>{t("productionAcceptance.title")}</span>
              <Badge tone={statusTone}>{statusLabel}</Badge>
            </div>
            <strong className="go-live-acceptance-headline">{t(acceptanceCenter.headlineKey)}</strong>
            <p>{nextAction}</p>
            {!liveDataAvailable ? <p className="go-live-acceptance-warning">{t("message.fallbackDataModeDetail")}</p> : null}
            {statusMessage ? <p className="go-live-acceptance-message">{statusMessage}</p> : null}
          </div>
          <div className="go-live-acceptance-actions">
            {primaryAction}
            {acceptanceReady ? (
              <>
                {acceptanceCenter.primaryAction !== "run_status_check" ? <button className="secondary-button" disabled={!liveDataAvailable || productionReadinessLoading} onClick={onRefreshProductionReadiness} type="button">
                  <RefreshCw size={14} />
                  {productionReadinessLoading ? t("action.checkingProductionReadiness") : t("action.checkProductionReadiness")}
                </button> : null}
              </>
            ) : (
              <>
                {acceptanceCenter.primaryAction !== "export_acceptance_report" ? <button className="secondary-button" disabled={!liveDataAvailable || !productionReadiness || productionEvidenceExporting} onClick={onExportProductionEvidence} type="button">
                  <Download size={14} />
                  {productionEvidenceExporting ? t("action.exportingProductionEvidence") : t("action.exportProductionEvidence")}
                </button> : null}
              </>
            )}
            {acceptanceCenter.primaryAction !== "open_permission_change" ? <button className="secondary-button" onClick={onOpenPermissionChange} type="button">
              <ShieldCheck size={14} />
              {t("action.openPermissionChange")}
            </button> : null}
          </div>
        </div>

        <section className="go-live-acceptance-blockers" aria-label={t("productionAcceptance.blockers")}>
          <strong>{acceptanceCenter.blockers.length > 0 ? t("productionAcceptance.blockers") : t("productionAcceptance.noBlockers")}</strong>
          {acceptanceCenter.blockers.length > 0 ? (
            <ul>
              {acceptanceCenter.blockers.map((blocker) => (
                <li key={blocker.key}>{t(blocker.labelKey, blocker.detail)}</li>
              ))}
            </ul>
          ) : (
            <span>{t("productionAcceptance.noBlockersDetail")}</span>
          )}
        </section>

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
            {acceptanceCenter.checkRows.map((row) => (
              <li key={row.key}>
                <span className={`go-live-step-dot tone-${productionAcceptanceStatusTone(row.status)}`} aria-hidden="true" />
                <div>
                  <strong>{t(row.labelKey)}</strong>
                  <span>{row.detailKey ? t(row.detailKey) : row.detail}</span>
                </div>
              </li>
            ))}
          </ol>
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
            {matchingProductionReport ? (
              <div className="go-live-acceptance-report">
                <dt>{t("productionAcceptance.report")}</dt>
                <dd>
                  {tx(t, "productionAcceptance.reportExportedBy", {
                    actor: permissionEntityDisplayName(matchingProductionReport.generatedBy, t),
                    date: formatDate(matchingProductionReport.generatedAt)
                  })}
                </dd>
              </div>
            ) : null}
          </dl>
        </aside>
      </section>
    </div>
  );
}

function reportMatchesAcceptanceScope(
  report: PermissionPackageProductionEvidenceReport | null,
  input: PermissionPackageDraftInput,
  readiness: PermissionPackageProductionReadiness | null
) {
  if (!report) return false;
  if (readiness && report.readinessGeneratedAt !== readiness.generatedAt) return false;
  return report.scope.tenantId === input.tenantId &&
    report.scope.workspaceId === input.workspaceId &&
    report.scope.templateId === input.templateId &&
    report.scope.targetId === input.targetId &&
    report.scope.callerInstanceId === input.callerInstanceId;
}

function renderProductionAcceptanceAction({
  action,
  connectionDiagnosticsChecking,
  liveDataAvailable,
  onExportProductionEvidence,
  onOpenPermissionChange,
  onRefreshProductionReadiness,
  onRunConnectionDiagnostics,
  productionEvidenceExporting,
  productionReadiness,
  productionReadinessLoading,
  t
}: {
  action: ProductionAcceptanceAction;
  connectionDiagnosticsChecking: boolean;
  liveDataAvailable: boolean;
  onExportProductionEvidence: () => void;
  onOpenPermissionChange: () => void;
  onRefreshProductionReadiness: () => void;
  onRunConnectionDiagnostics: () => void;
  productionEvidenceExporting: boolean;
  productionReadiness: PermissionPackageProductionReadiness | null;
  productionReadinessLoading: boolean;
  t: Translator;
}) {
  if (action === "run_diagnostics") {
    return (
      <button className="primary-button" disabled={connectionDiagnosticsChecking} onClick={onRunConnectionDiagnostics} type="button">
        <RefreshCw size={14} />
        {connectionDiagnosticsChecking ? t("connectionDiagnostics.checking") : t("connectionDiagnostics.action")}
      </button>
    );
  }
  if (action === "open_permission_change") {
    return (
      <button className="primary-button" onClick={onOpenPermissionChange} type="button">
        <ShieldCheck size={14} />
        {t("action.openPermissionChange")}
      </button>
    );
  }
  if (action === "export_acceptance_report") {
    return (
      <button className="primary-button" disabled={!liveDataAvailable || productionEvidenceExporting} onClick={onExportProductionEvidence} type="button">
        <Download size={14} />
        {productionEvidenceExporting ? t("action.exportingProductionEvidence") : t("action.exportProductionEvidence")}
      </button>
    );
  }
  return (
    <button className="primary-button" disabled={!liveDataAvailable || productionReadinessLoading} onClick={onRefreshProductionReadiness} type="button">
      <RefreshCw size={14} />
      {productionReadinessLoading
        ? t("action.checkingProductionReadiness")
        : productionReadiness
          ? t("action.checkProductionReadiness")
          : t("productionAcceptance.action.runStatus")}
    </button>
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

function productionAcceptanceStatusTone(status: ProductionAcceptanceStatus): Tone {
  if (status === "ready") return "success";
  if (status === "attention") return "warning";
  if (status === "blocked") return "danger";
  return "neutral";
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}
