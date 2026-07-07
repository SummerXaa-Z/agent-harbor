import type { HealthCheckResult } from "./api";
import type { Translator } from "./consolePresenters";
import { systemCapabilityLabelKeys } from "./systemCapabilityLabels.ts";
import { isManagementMcpToolCatalogContractIssue } from "./systemInfoContract.ts";
import type { ConsoleSession, JsonObject, JsonValue } from "./types";

export type ConnectionDiagnosticKey = "session" | "api" | "dataSource" | "mcp" | "mcpCatalog";
export type ConnectionDiagnosticStatus = "ok" | "warning" | "error";

export interface ManagementMcpCatalogDiagnostic {
  confirmationRequiredTools?: number;
  metadataVersion?: number;
  message?: string;
  missingRequiredTools?: string[];
  status: ConnectionDiagnosticStatus;
  toolsWithConfirmationSchema?: number;
  toolsWithAccess?: number;
  toolsWithExecution?: number;
  toolsWithLifecycle?: number;
  toolsWithSafety?: number;
}

export interface ManagementMcpCatalogTool {
  access?: {
    requiredRole?: string;
    reviewerBound?: boolean;
    scopeBoundary?: string;
  };
  execution?: {
    auditResourceType?: string;
    confirmationRequired?: boolean;
    idempotency?: string;
    preflightTool?: string;
    returnsSecret?: boolean;
  };
  inputSchema?: JsonObject;
  lifecycle?: {
    preferredName?: string;
    status?: string;
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

export const requiredManagementMcpToolNames = [
  "list_admin_identities",
  "create_admin_identity",
  "rotate_admin_identity_key",
  "disable_admin_identity",
  "list_permission_package_templates",
  "draft_permission_package",
  "preflight_permission_package",
  "apply_permission_package",
  "create_permission_package_approval_request",
  "list_permission_package_approval_requests",
  "approve_permission_package_approval_request",
  "reject_permission_package_approval_request",
  "withdraw_permission_package_approval_request",
  "list_permission_package_applications",
  "check_permission_package_production_readiness",
  "export_permission_package_production_report",
  "explain_permission_package_draft",
  "explain_access_decision",
  "get_tenant_access_profile",
  "list_agents",
  "list_capabilities"
] as const;

const managementMcpWriteConfirmationReasonMaxLength = 500;

export interface ConnectionDiagnosticRow {
  key: ConnectionDiagnosticKey;
  status: ConnectionDiagnosticStatus;
  titleKey: string;
  detailKey: string;
  detailParamKeys?: Record<string, string[]>;
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

export function connectionDiagnosticDetail(row: ConnectionDiagnosticRow | undefined, t: Translator) {
  if (!row) return "";
  const detailParams: Record<string, string | number> = { ...(row.detailParams ?? {}) };
  for (const [name, keys] of Object.entries(row.detailParamKeys ?? {})) {
    detailParams[name] = keys.map((key) => t(key)).join(", ");
  }
  return Object.keys(detailParams).length > 0 ? tx(t, row.detailKey, detailParams) : t(row.detailKey);
}

export function managementMcpCatalogDiagnosticFromResult(
  result: ManagementMcpToolsListResult | null | undefined
): ManagementMcpCatalogDiagnostic {
  if (!result || result.metadataVersion === undefined) {
    return { message: "missing metadataVersion", status: "error" };
  }
  if (result.metadataVersion !== 4) {
    return { metadataVersion: result.metadataVersion, status: "warning" };
  }
  const tools = Array.isArray(result.tools) ? result.tools : [];
  if (tools.length === 0) {
    return { metadataVersion: result.metadataVersion, message: "empty management tool catalog", status: "error" };
  }
  const toolsWithSafety = tools.filter((tool) => hasSafetyMetadata(tool)).length;
  const toolsWithAccess = tools.filter((tool) => hasAccessMetadata(tool)).length;
  const toolsWithLifecycle = tools.filter((tool) => hasLifecycleMetadata(tool)).length;
  const toolsWithExecution = tools.filter((tool) => hasExecutionMetadata(tool)).length;
  if (
    toolsWithSafety !== tools.length ||
    toolsWithAccess !== tools.length ||
    toolsWithLifecycle !== tools.length ||
    toolsWithExecution !== tools.length
  ) {
    return {
      metadataVersion: result.metadataVersion,
      message: `catalog metadata incomplete: safety ${toolsWithSafety}/${tools.length}, access ${toolsWithAccess}/${tools.length}, lifecycle ${toolsWithLifecycle}/${tools.length}, execution ${toolsWithExecution}/${tools.length}`,
      status: "error",
      toolsWithAccess,
      toolsWithExecution,
      toolsWithLifecycle,
      toolsWithSafety
    };
  }
  const toolsRequiringConfirmation = tools.filter((tool) => managementMcpToolRequiresConfirmation(tool));
  const toolsWithConfirmationSchema = toolsRequiringConfirmation.filter((tool) => hasWriteConfirmationInputSchema(tool)).length;
  if (toolsWithConfirmationSchema !== toolsRequiringConfirmation.length) {
    return {
      confirmationRequiredTools: toolsRequiringConfirmation.length,
      metadataVersion: result.metadataVersion,
      message: `confirmation schema incomplete: ${toolsWithConfirmationSchema}/${toolsRequiringConfirmation.length}`,
      status: "error",
      toolsWithAccess,
      toolsWithConfirmationSchema,
      toolsWithExecution,
      toolsWithLifecycle,
      toolsWithSafety
    };
  }
  const toolNames = new Set(tools.map((tool) => tool.name).filter((name): name is string => Boolean(name)));
  const missingRequiredTools = requiredManagementMcpToolNames.filter((name) => !toolNames.has(name));
  if (missingRequiredTools.length > 0) {
    return {
      metadataVersion: result.metadataVersion,
      missingRequiredTools,
      status: "error",
      toolsWithAccess,
      toolsWithExecution,
      toolsWithLifecycle,
      toolsWithSafety
    };
  }
  return {
    metadataVersion: result.metadataVersion,
    status: "ok",
    toolsWithAccess,
    toolsWithExecution,
    toolsWithLifecycle,
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

function hasLifecycleMetadata(tool: ManagementMcpCatalogTool): boolean {
  const lifecycle = tool.lifecycle;
  if (!lifecycle || !lifecycle.status || lifecycle.status === "unspecified") return false;
  if (lifecycle.status === "compatibility_alias") return Boolean(lifecycle.preferredName);
  return lifecycle.status === "active";
}

function hasExecutionMetadata(tool: ManagementMcpCatalogTool): boolean {
  const execution = tool.execution;
  if (!execution || !execution.idempotency || execution.idempotency === "unspecified") return false;
  if (!["safe_repeat", "conditional_repeat", "not_idempotent"].includes(execution.idempotency)) return false;
  if (typeof execution.confirmationRequired !== "boolean") return false;
  if (execution.preflightTool !== undefined && execution.preflightTool.trim() === "") return false;
  if (execution.auditResourceType !== undefined && execution.auditResourceType.trim() === "") return false;
  if (execution.returnsSecret !== undefined && typeof execution.returnsSecret !== "boolean") return false;
  return true;
}

function managementMcpToolRequiresConfirmation(tool: ManagementMcpCatalogTool): boolean {
  return Boolean(
    tool.execution?.confirmationRequired ||
    tool.safety?.mutatesState ||
    tool.safety?.operationType === "write"
  );
}

function hasWriteConfirmationInputSchema(tool: ManagementMcpCatalogTool): boolean {
  const inputSchema = tool.inputSchema;
  if (!jsonObjectHasStringValue(inputSchema, "type", "object")) return false;
  if (!jsonSchemaRequiredIncludes(inputSchema, "confirmation")) return false;
  const confirmationSchema = jsonObjectProperty(jsonObjectProperty(inputSchema, "properties"), "confirmation");
  if (!jsonObjectHasStringValue(confirmationSchema, "type", "object")) return false;
  if (!jsonSchemaRequiredIncludes(confirmationSchema, "confirmed")) return false;
  if (!jsonSchemaRequiredIncludes(confirmationSchema, "reason")) return false;
  const confirmationProperties = jsonObjectProperty(confirmationSchema, "properties");
  const confirmedSchema = jsonObjectProperty(confirmationProperties, "confirmed");
  const reasonSchema = jsonObjectProperty(confirmationProperties, "reason");
  return Boolean(
    jsonObjectHasStringValue(confirmedSchema, "type", "boolean") &&
    jsonObjectHasStringValue(reasonSchema, "type", "string") &&
    jsonObjectHasNumberValue(reasonSchema, "minLength", 1) &&
    jsonObjectHasNumberValue(reasonSchema, "maxLength", managementMcpWriteConfirmationReasonMaxLength)
  );
}

function jsonObjectProperty(object: JsonObject | undefined, key: string): JsonObject | undefined {
  const value = object?.[key];
  return isJsonObject(value) ? value : undefined;
}

function jsonObjectHasStringValue(object: JsonObject | undefined, key: string, expected: string): boolean {
  return object?.[key] === expected;
}

function jsonObjectHasNumberValue(object: JsonObject | undefined, key: string, expected: number): boolean {
  return object?.[key] === expected;
}

function jsonSchemaRequiredIncludes(schema: JsonObject | undefined, field: string): boolean {
  const required = schema?.required;
  return Array.isArray(required) && required.some((value) => value === field);
}

function isJsonObject(value: JsonValue | undefined): value is JsonObject {
  return typeof value === "object" && value !== null && !Array.isArray(value);
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
    if (hasOnlyManagementMcpCatalogContractIssues(apiHealth.contractIssues, apiHealth.missingCapabilities)) {
      return {
        detailKey: "message.apiContractIncompatibleManagementCatalog",
        key: "api",
        status: "error",
        titleKey: "connectionDiagnostics.api.title"
      };
    }
    const capabilityLabelKeys = systemCapabilityLabelKeys(apiHealth.missingCapabilities);
    if (capabilityLabelKeys.length > 0) {
      return {
        detailKey: "message.apiContractIncompatible",
        detailParamKeys: {
          capabilities: capabilityLabelKeys
        },
        key: "api",
        status: "error",
        titleKey: "connectionDiagnostics.api.title"
      };
    }
    return {
      detailKey: "message.apiContractIncompatibleUnknown",
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
        tools: Math.min(
          catalog.toolsWithAccess ?? 0,
          catalog.toolsWithExecution ?? 0,
          catalog.toolsWithLifecycle ?? 0,
          catalog.toolsWithSafety ?? 0
        ),
        version: catalog.metadataVersion ?? 4
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
  if (catalog.missingRequiredTools?.length) {
    return {
      detailKey: "connectionDiagnostics.mcpCatalog.missingTools",
      key: "mcpCatalog",
      status: "error",
      titleKey: "connectionDiagnostics.mcpCatalog.title"
    };
  }
  if (catalog.toolsWithConfirmationSchema !== undefined && catalog.confirmationRequiredTools !== undefined) {
    return {
      detailKey: "connectionDiagnostics.mcpCatalog.confirmationSchema",
      detailParams: {
        ready: catalog.toolsWithConfirmationSchema,
        total: catalog.confirmationRequiredTools
      },
      key: "mcpCatalog",
      status: "error",
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

function hasOnlyManagementMcpCatalogContractIssues(
  contractIssues: string[] | undefined,
  missingCapabilities: string[] | undefined
): boolean {
  const issues = Array.isArray(contractIssues) ? contractIssues : [];
  const capabilities = Array.isArray(missingCapabilities) ? missingCapabilities : [];
  return capabilities.length === 0 && issues.length > 0 && issues.every(isManagementMcpToolCatalogContractIssue);
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}
