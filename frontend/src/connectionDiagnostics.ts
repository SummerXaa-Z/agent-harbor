import type { HealthCheckResult } from "./api";
import type { ConsoleSession } from "./types";

export type ConnectionDiagnosticKey = "session" | "api" | "dataSource" | "mcp";
export type ConnectionDiagnosticStatus = "ok" | "warning" | "error";

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
  mcpHealth: HealthCheckResult;
  session: ConsoleSession | null;
}

export function buildConnectionDiagnosticRows(input: ConnectionDiagnosticInput): ConnectionDiagnosticRow[] {
  return [
    sessionDiagnosticRow(input.session),
    apiDiagnosticRow(input.apiHealth),
    dataSourceDiagnosticRow(input.liveDataLoaded, input.loadError),
    mcpDiagnosticRow(input.mcpHealth)
  ];
}

export function connectionDiagnosticsSummaryStatus(rows: ConnectionDiagnosticRow[]): ConnectionDiagnosticStatus {
  if (rows.some((row) => row.status === "error")) return "error";
  if (rows.some((row) => row.status === "warning")) return "warning";
  return "ok";
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
