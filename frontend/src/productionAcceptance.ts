import type { ConnectionDiagnosticStatus } from "./connectionDiagnostics";
import type { AiAdminProductionConsoleSummary } from "./aiAdminProductionConsole";
import type {
  PermissionPackageAcceptanceReport,
  PermissionPackageProductionReadiness,
  PermissionPackageProductionReadinessCheck
} from "./permissionPackages";

export type ProductionAcceptanceStatus = "ready" | "blocked" | "attention" | "pending";
export type ProductionAcceptanceAction =
  | "export_acceptance_report"
  | "open_permission_change"
  | "run_status_check"
  | "run_diagnostics";

export interface ProductionAcceptanceBlocker {
  key: string;
  labelKey: string;
  detail: string;
}

export interface ProductionAcceptanceCheckRow {
  key: string;
  labelKey: string;
  status: ProductionAcceptanceStatus;
  detailKey?: string;
  detail?: string;
}

export interface ProductionAcceptanceCenter {
  blockers: ProductionAcceptanceBlocker[];
  blockingCount: number;
  checkRows: ProductionAcceptanceCheckRow[];
  headlineKey: string;
  primaryAction: ProductionAcceptanceAction;
  readyCount: number;
  status: ProductionAcceptanceStatus;
  totalCount: number;
}

export interface ProductionAcceptanceInput {
  connectionStatus: ConnectionDiagnosticStatus | null;
  liveDataAvailable: boolean;
  productionReadiness: PermissionPackageProductionReadiness | null;
  productionSummary: AiAdminProductionConsoleSummary;
}

export function buildProductionAcceptanceCenter({
  connectionStatus,
  liveDataAvailable,
  productionReadiness,
  productionSummary
}: ProductionAcceptanceInput): ProductionAcceptanceCenter {
  const connectionRow = connectionCheckRow(connectionStatus, liveDataAvailable);
  const statusRow = productionStatusRow(productionReadiness, productionSummary);
  const runtimeRow = runtimeCheckRow(productionReadiness, productionSummary);
  const handoffRow = handoffCheckRow(productionReadiness);
  const checkRows = [connectionRow, statusRow, runtimeRow, handoffRow];
  const blockers = [
    ...connectionBlockers(connectionStatus, liveDataAvailable),
    ...readinessBlockers(productionReadiness)
  ];

  const status = overallStatus({
    blockers,
    checkRows,
    connectionStatus,
    liveDataAvailable,
    productionReadiness,
    productionSummary
  });

  return {
    blockers,
    blockingCount: blockers.length,
    checkRows,
    headlineKey: headlineKey(status),
    primaryAction: primaryAction(status, connectionStatus, liveDataAvailable, productionReadiness, productionSummary),
    readyCount: checkRows.filter((row) => row.status === "ready").length,
    status,
    totalCount: checkRows.length
  };
}

function connectionCheckRow(
  connectionStatus: ConnectionDiagnosticStatus | null,
  liveDataAvailable: boolean
): ProductionAcceptanceCheckRow {
  if (!liveDataAvailable) {
    return {
      detailKey: "productionAcceptance.check.connectionFallback",
      key: "connection",
      labelKey: "productionAcceptance.check.connection",
      status: "blocked"
    };
  }
  if (connectionStatus === "ok") {
    return {
      detailKey: "productionAcceptance.check.connectionReady",
      key: "connection",
      labelKey: "productionAcceptance.check.connection",
      status: "ready"
    };
  }
  if (connectionStatus === "warning") {
    return {
      detailKey: "productionAcceptance.check.connectionAttention",
      key: "connection",
      labelKey: "productionAcceptance.check.connection",
      status: "attention"
    };
  }
  if (connectionStatus === "error") {
    return {
      detailKey: "productionAcceptance.check.connectionBlocked",
      key: "connection",
      labelKey: "productionAcceptance.check.connection",
      status: "blocked"
    };
  }
  return {
    detailKey: "productionAcceptance.check.connectionUnknown",
    key: "connection",
    labelKey: "productionAcceptance.check.connection",
    status: "attention"
  };
}

function productionStatusRow(
  productionReadiness: PermissionPackageProductionReadiness | null,
  productionSummary: AiAdminProductionConsoleSummary
): ProductionAcceptanceCheckRow {
  return {
    detailKey: productionReadiness
      ? `status.production${capitalizeProductionStatus(productionReadiness.status)}`
      : productionSummary.primaryActionKey,
    key: "permission_change",
    labelKey: "productionAcceptance.check.permissionChange",
    status: readinessStatus(productionReadiness, productionSummary)
  };
}

function runtimeCheckRow(
  productionReadiness: PermissionPackageProductionReadiness | null,
  productionSummary: AiAdminProductionConsoleSummary
): ProductionAcceptanceCheckRow {
  const runtimeStep = productionSummary.steps.find((step) => step.key === "runtime");
  const hasRuntime = Boolean(productionReadiness?.runtimeEvidence.allowedTrace && productionReadiness.runtimeEvidence.deniedTrace);
  return {
    detailKey: hasRuntime ? "productionAcceptance.check.runtimeReady" : runtimeStep?.detailKey ?? "productionAcceptance.check.runtimePending",
    key: "runtime",
    labelKey: "productionAcceptance.check.runtime",
    status: hasRuntime ? "ready" : productionStepStatus(runtimeStep?.status)
  };
}

function handoffCheckRow(productionReadiness: PermissionPackageProductionReadiness | null): ProductionAcceptanceCheckRow {
  return {
    detailKey: productionReadiness?.status === "ready"
      ? "productionAcceptance.check.handoffReady"
      : "productionAcceptance.check.handoffPending",
    key: "handoff",
    labelKey: "productionAcceptance.check.handoff",
    status: productionReadiness?.status === "ready" ? "ready" : "pending"
  };
}

function connectionBlockers(
  connectionStatus: ConnectionDiagnosticStatus | null,
  liveDataAvailable: boolean
): ProductionAcceptanceBlocker[] {
  if (!liveDataAvailable) {
    return [{
      detail: "",
      key: "live_data",
      labelKey: "productionAcceptance.blocker.liveData"
    }];
  }
  if (connectionStatus && connectionStatus !== "ok") {
    return [{
      detail: "",
      key: "connection",
      labelKey: connectionStatus === "error"
        ? "productionAcceptance.blocker.connectionError"
        : "productionAcceptance.blocker.connectionWarning"
    }];
  }
  if (!connectionStatus) {
    return [{
      detail: "",
      key: "connection",
      labelKey: "productionAcceptance.blocker.connectionUnknown"
    }];
  }
  return [];
}

function readinessBlockers(
  productionReadiness: PermissionPackageProductionReadiness | null
): ProductionAcceptanceBlocker[] {
  if (!productionReadiness) return [];
  return productionReadiness.checks
    .filter((check) => check.severity === "blocking")
    .map((check) => ({
      detail: check.message,
      key: check.code,
      labelKey: `productionAcceptance.blocker.${check.code}`
    }));
}

function overallStatus({
  blockers,
  checkRows,
  connectionStatus,
  liveDataAvailable,
  productionReadiness,
  productionSummary
}: {
  blockers: ProductionAcceptanceBlocker[];
  checkRows: ProductionAcceptanceCheckRow[];
  connectionStatus: ConnectionDiagnosticStatus | null;
  liveDataAvailable: boolean;
  productionReadiness: PermissionPackageProductionReadiness | null;
  productionSummary: AiAdminProductionConsoleSummary;
}): ProductionAcceptanceStatus {
  if (!liveDataAvailable || blockers.some((blocker) => blocker.key !== "connection")) return "blocked";
  if (connectionStatus === "warning" || connectionStatus === "error" || connectionStatus === null) return "attention";
  if (productionReadiness?.status === "ready" || productionSummary.status === "ready") {
    return checkRows.every((row) => row.status === "ready") ? "ready" : "attention";
  }
  if (productionReadiness?.status === "blocked" || productionSummary.status === "blocked") return "blocked";
  if (productionReadiness?.status === "needs_review" || productionSummary.status === "needs_review") return "attention";
  return "pending";
}

function primaryAction(
  status: ProductionAcceptanceStatus,
  connectionStatus: ConnectionDiagnosticStatus | null,
  liveDataAvailable: boolean,
  productionReadiness: PermissionPackageProductionReadiness | null,
  productionSummary: AiAdminProductionConsoleSummary
): ProductionAcceptanceAction {
  if (!liveDataAvailable || connectionStatus !== "ok") return "run_diagnostics";
  if (status === "ready" && productionReadiness?.status === "ready") return "export_acceptance_report";
  if (productionReadiness?.status === "blocked" || productionSummary.status === "blocked") return "open_permission_change";
  return "run_status_check";
}

function readinessStatus(
  productionReadiness: PermissionPackageProductionReadiness | null,
  productionSummary: AiAdminProductionConsoleSummary
): ProductionAcceptanceStatus {
  if (productionReadiness?.status === "ready" || productionSummary.status === "ready") return "ready";
  if (productionReadiness?.status === "blocked" || productionSummary.status === "blocked") return "blocked";
  if (productionReadiness?.status === "needs_review" || productionSummary.status === "needs_review") return "attention";
  return "pending";
}

function productionStepStatus(status: string | undefined): ProductionAcceptanceStatus {
  if (status === "ready") return "ready";
  if (status === "blocked") return "blocked";
  if (status === "needs_review") return "attention";
  return "pending";
}

function headlineKey(status: ProductionAcceptanceStatus) {
  if (status === "ready") return "productionAcceptance.headline.ready";
  if (status === "blocked") return "productionAcceptance.headline.blocked";
  if (status === "attention") return "productionAcceptance.headline.attention";
  return "productionAcceptance.headline.pending";
}

export function productionAcceptanceReportFilename(
  report: Pick<PermissionPackageAcceptanceReport, "generatedAt" | "scope" | "status"> &
    Partial<Pick<PermissionPackageAcceptanceReport, "reportDigest">>
) {
  const generated = safeFilenameSegment(report.generatedAt || new Date().toISOString());
  const digest = safeFilenameSegment(shortReportDigest(report.reportDigest));
  return [
    "agentharbor-go-live-status",
    safeFilenameSegment(report.scope.tenantId),
    safeFilenameSegment(report.scope.workspaceId),
    safeFilenameSegment(report.scope.templateId),
    safeFilenameSegment(report.status),
    digest ? "digest" : "",
    digest,
    generated
  ].filter(Boolean).join("-") + ".json";
}

function shortReportDigest(digest: string | undefined) {
  return digest?.trim().slice(0, 12) ?? "";
}

function safeFilenameSegment(value: string | undefined) {
  return (value ?? "")
    .trim()
    .replace(/[^a-zA-Z0-9._-]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80);
}

function capitalizeProductionStatus(status: PermissionPackageProductionReadiness["status"]) {
  if (status === "needs_review") return "NeedsReview";
  if (status === "blocked") return "Blocked";
  return "Ready";
}
