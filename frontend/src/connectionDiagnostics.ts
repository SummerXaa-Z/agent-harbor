import type { HealthCheckResult } from "./api";
import type { ConsoleSession } from "./types";

export type ConnectionDiagnosticKey = "session" | "api" | "dataSource" | "mcp" | "mcpCatalog";
export type ConnectionDiagnosticStatus = "ok" | "warning" | "error";

export interface ManagementMcpCatalogDiagnostic {
  metadataVersion?: number;
  message?: string;
  status: ConnectionDiagnosticStatus;
  toolsWithAccess?: number;
  toolsWithSafety?: number;
}

export interface ManagementMcpCatalogTool {
  access?: {
    requiredRole?: string;
    reviewerBound?: boolean;
    scopeBoundary?: string;
  };
  name?: string;
  safety?: {
    approvalMode?: string;
    mutatesState?: boolean;
    operationType?: string;
    readOnly?: boolean;
  };
}

export interface ManagementMcpToolsListResult {
  metadataVersion?: number;
  tools?: ManagementMcpCatalogTool[];
}

export interface ConnectionDiagnosticRow {
  key: ConnectionDiagnosticKey;
  status: ConnectionDiagnosticStatus;
  titleKey: string;
  detailKey: string;
  detailParams?: Record<string, string | number>;
}

export interface ConnectionDiagnosticInput {
  apiHealth: HealthCheckResult;
  liveDataLoaded: boolean;
  loadError: string;
  mcpCatalog: ManagementMcpCatalogDiagnostic;
  mcpHealth: HealthCheckResult;
  session: ConsoleSession | null;
}

export function buildConnectionDiagnosticRows(input: ConnectionDiagnosticInput): ConnectionDiagnosticRow[] {
  return [
    sessionDiagnosticRow(input.session),
    apiDiagnosticRow(input.apiHealth),
    dataSourceDiagnosticRow(input.liveDataLoaded, input.loadError),
    mcpDiagnosticRow(input.mcpHealth),
    mcpCatalogDiagnosticRow(input.mcpCatalog)
  ];
}

export function connectionDiagnosticsSummaryStatus(rows: ConnectionDiagnosticRow[]): ConnectionDiagnosticStatus {
  if (rows.some((row) => row.status === "error")) return "error";
  if (rows.some((row) => row.status === "warning")) return "warning";
  return "ok";
}

export function managementMcpCatalogDiagnosticFromResult(
  result: ManagementMcpToolsListResult | null | undefined
): ManagementMcpCatalogDiagnostic {
  if (!result || result.metadataVersion === undefined) {
    return { message: "missing metadataVersion", status: "error" };
  }
  if (result.metadataVersion !== 1) {
    return { metadataVersion: result.metadataVersion, status: "warning" };
  }
  const tools = Array.isArray(result.tools) ? result.tools : [];
  if (tools.length === 0) {
    return { metadataVersion: result.metadataVersion, message: "empty management tool catalog", status: "error" };
  }
  const toolsWithSafety = tools.filter((tool) => hasSafetyMetadata(tool)).length;
  const toolsWithAccess = tools.filter((tool) => hasAccessMetadata(tool)).length;
  if (toolsWithSafety !== tools.length || toolsWithAccess !== tools.length) {
    return {
      metadataVersion: result.metadataVersion,
      message: `catalog metadata incomplete: safety ${toolsWithSafety}/${tools.length}, access ${toolsWithAccess}/${tools.length}`,
      status: "error",
      toolsWithAccess,
      toolsWithSafety
    };
  }
  return {
    metadataVersion: result.metadataVersion,
    status: "ok",
    toolsWithAccess,
    toolsWithSafety
  };
}

function sessionDiagnosticRow(session: ConsoleSession | null): ConnectionDiagnosticRow {
  if (session?.authenticated || session?.requiresLogin === false) {
    return {
      detailKey: session.requiresLogin
        ? "connectionDiagnostics.session.ok"
        : "connectionDiagnostics.session.devBypass",
      key: "session",
      status: "ok",
      titleKey: "connectionDiagnostics.session.title"
    };
  }
  return {
    detailKey: "connectionDiagnostics.session.signIn",
    key: "session",
    status: "error",
    titleKey: "connectionDiagnostics.session.title"
  };
}

function hasSafetyMetadata(tool: ManagementMcpCatalogTool): boolean {
  const safety = tool.safety;
  return Boolean(
    safety &&
    safety.operationType &&
    safety.operationType !== "unspecified" &&
    safety.approvalMode &&
    safety.approvalMode !== "unspecified" &&
    typeof safety.readOnly === "boolean" &&
    typeof safety.mutatesState === "boolean"
  );
}

function hasAccessMetadata(tool: ManagementMcpCatalogTool): boolean {
  const access = tool.access;
  return Boolean(
    access &&
    access.requiredRole &&
    access.requiredRole !== "unspecified" &&
    access.scopeBoundary &&
    access.scopeBoundary !== "unspecified" &&
    typeof access.reviewerBound === "boolean"
  );
}

function apiDiagnosticRow(apiHealth: HealthCheckResult): ConnectionDiagnosticRow {
  if (apiHealth.status === "ok") {
    return {
      detailKey: "connectionDiagnostics.api.ok",
      key: "api",
      status: "ok",
      titleKey: "connectionDiagnostics.api.title"
    };
  }
  if (apiHealth.code === "api_contract_unavailable") {
    return {
      detailKey: "message.apiContractUnavailable",
      key: "api",
      status: "error",
      titleKey: "connectionDiagnostics.api.title"
    };
  }
  if (apiHealth.code === "api_contract_incompatible") {
    return {
      detailKey: "message.apiContractIncompatible",
      detailParams: {
        capabilities: apiHealth.missingCapabilities?.join(", ") || apiHealth.message
      },
      key: "api",
      status: "error",
      titleKey: "connectionDiagnostics.api.title"
    };
  }
  return {
    detailKey: "connectionDiagnostics.api.error",
    detailParams: { detail: apiHealth.message },
    key: "api",
    status: "error",
    titleKey: "connectionDiagnostics.api.title"
  };
}

function dataSourceDiagnosticRow(liveDataLoaded: boolean, loadError: string): ConnectionDiagnosticRow {
  if (liveDataLoaded) {
    return {
      detailKey: "connectionDiagnostics.dataSource.ok",
      key: "dataSource",
      status: "ok",
      titleKey: "connectionDiagnostics.dataSource.title"
    };
  }
  if (loadError.trim()) {
    return {
      detailKey: "connectionDiagnostics.dataSource.error",
      detailParams: { detail: loadError.trim() },
      key: "dataSource",
      status: "warning",
      titleKey: "connectionDiagnostics.dataSource.title"
    };
  }
  return {
    detailKey: "connectionDiagnostics.dataSource.fallback",
    key: "dataSource",
    status: "warning",
    titleKey: "connectionDiagnostics.dataSource.title"
  };
}

function mcpDiagnosticRow(mcpHealth: HealthCheckResult): ConnectionDiagnosticRow {
  if (mcpHealth.status === "ok") {
    return {
      detailKey: "connectionDiagnostics.mcp.ok",
      key: "mcp",
      status: "ok",
      titleKey: "connectionDiagnostics.mcp.title"
    };
  }
  return {
    detailKey: "connectionDiagnostics.mcp.error",
    detailParams: { detail: mcpHealth.message },
    key: "mcp",
    status: "error",
    titleKey: "connectionDiagnostics.mcp.title"
  };
}

function mcpCatalogDiagnosticRow(catalog: ManagementMcpCatalogDiagnostic): ConnectionDiagnosticRow {
  if (catalog.status === "ok") {
    return {
      detailKey: "connectionDiagnostics.mcpCatalog.ok",
      detailParams: {
        tools: Math.min(catalog.toolsWithAccess ?? 0, catalog.toolsWithSafety ?? 0),
        version: catalog.metadataVersion ?? 1
      },
      key: "mcpCatalog",
      status: "ok",
      titleKey: "connectionDiagnostics.mcpCatalog.title"
    };
  }
  if (catalog.status === "warning") {
    return {
      detailKey: "connectionDiagnostics.mcpCatalog.versionWarning",
      detailParams: { version: catalog.metadataVersion ?? "unknown" },
      key: "mcpCatalog",
      status: "warning",
      titleKey: "connectionDiagnostics.mcpCatalog.title"
    };
  }
  return {
    detailKey: "connectionDiagnostics.mcpCatalog.error",
    detailParams: { detail: catalog.message || "catalog unavailable" },
    key: "mcpCatalog",
    status: "error",
    titleKey: "connectionDiagnostics.mcpCatalog.title"
  };
}
