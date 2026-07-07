import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  buildConnectionDiagnosticRows,
  connectionDiagnosticDetail,
  connectionDiagnosticsSummaryStatus,
  managementMcpCatalogDiagnosticFromResult,
  requiredManagementMcpToolNames
} from "../src/connectionDiagnostics.ts";

const managementMcpSource = readFileSync(new URL("../../internal/httpapi/management_mcp.go", import.meta.url), "utf8");
const legacyManagementMcpToolNames = new Set(["export_permission_package_production_evidence"]);
const confirmationRequiredToolNames = new Set([
  "create_admin_identity",
  "rotate_admin_identity_key",
  "disable_admin_identity",
  "apply_permission_package",
  "create_permission_package_approval_request",
  "approve_permission_package_approval_request",
  "reject_permission_package_approval_request",
  "withdraw_permission_package_approval_request"
]);

const confirmationInputSchema = {
  type: "object",
  properties: {
    confirmation: {
      type: "object",
      properties: {
        confirmed: { type: "boolean" },
        reason: { type: "string", minLength: 1, maxLength: 500 }
      },
      required: ["confirmed", "reason"],
      additionalProperties: false
    }
  },
  required: ["confirmation"],
  additionalProperties: false
};

const catalogTool = {
  access: { requiredRole: "authenticated_admin", reviewerBound: false, scopeBoundary: "requested_scope" },
  execution: { confirmationRequired: false, idempotency: "safe_repeat" },
  inputSchema: { type: "object", properties: {}, required: [] },
  lifecycle: { status: "active" },
  name: "draft_permission_package",
  safety: { approvalMode: "none", mutatesState: false, operationType: "preview", readOnly: true }
};

function catalogToolNamed(name) {
  if (!confirmationRequiredToolNames.has(name)) {
    return { ...catalogTool, name };
  }
  return {
    ...catalogTool,
    execution: { confirmationRequired: true, idempotency: "not_idempotent" },
    inputSchema: confirmationInputSchema,
    name,
    safety: { approvalMode: "none", mutatesState: true, operationType: "write", readOnly: false }
  };
}

function backendManagementMcpToolNames() {
  return Array.from(managementMcpSource.matchAll(/Name:\s*"([^"]+)"/g), (match) => match[1]);
}

test("connection diagnostics reports ready when session, API, live data, and MCP are ready", () => {
  const rows = buildConnectionDiagnosticRows({
    apiHealth: { status: "ok", message: "ok" },
    liveDataLoaded: true,
    loadError: "",
    mcpCatalog: { metadataVersion: 4, status: "ok", toolsWithAccess: 12, toolsWithExecution: 12, toolsWithLifecycle: 12, toolsWithSafety: 12 },
    mcpHealth: { status: "ok", message: "ok" },
    session: { actor: "admin-key", authenticated: true, requiresLogin: true }
  });

  assert.deepEqual(rows.map((row) => [row.key, row.status]), [
    ["session", "ok"],
    ["api", "ok"],
    ["dataSource", "ok"],
    ["mcp", "ok"],
    ["mcpCatalog", "ok"]
  ]);
  assert.equal(connectionDiagnosticsSummaryStatus(rows), "ok");
});

test("connection diagnostics blocks old API or expired session before production journey", () => {
  const rows = buildConnectionDiagnosticRows({
    apiHealth: {
      code: "api_contract_incompatible",
      message: "missing capabilities",
      missingCapabilities: ["permission_package_approval_withdraw"],
      status: "error"
    },
    liveDataLoaded: false,
    loadError: "",
    mcpCatalog: { metadataVersion: 4, status: "ok", toolsWithAccess: 12, toolsWithExecution: 12, toolsWithLifecycle: 12, toolsWithSafety: 12 },
    mcpHealth: { status: "ok", message: "ok" },
    session: { authenticated: false, requiresLogin: true }
  });

  assert.equal(connectionDiagnosticsSummaryStatus(rows), "error");
  assert.equal(rows.find((row) => row.key === "session")?.detailKey, "connectionDiagnostics.session.signIn");
  const apiRow = rows.find((row) => row.key === "api");
  assert.equal(apiRow?.detailKey, "message.apiContractIncompatible");
  assert.deepEqual(apiRow?.detailParamKeys, {
    capabilities: ["systemCapability.permissionPackageApprovalWithdraw"]
  });
  assert.equal(apiRow?.detailParams, undefined);
  assert.equal(
    connectionDiagnosticDetail(apiRow, (key) => ({
      "message.apiContractIncompatible": "API is missing capabilities: {capabilities}.",
      "systemCapability.permissionPackageApprovalWithdraw": "Approval withdrawal"
    }[key] ?? key)),
    "API is missing capabilities: Approval withdrawal."
  );
  assert.equal(rows.find((row) => row.key === "dataSource")?.status, "warning");
});

test("connection diagnostics hides raw management MCP catalog contract issue keys", () => {
  const rows = buildConnectionDiagnosticRows({
    apiHealth: {
      code: "api_contract_incompatible",
      contractIssues: [
        "managementMcpToolCatalog.requiredMetadata.access",
        "managementMcpToolCatalog.catalogDigest",
        "managementMcpToolCatalog.toolsWithConfirmationSchema"
      ],
      message: "system info contract issues: managementMcpToolCatalog.requiredMetadata.access, managementMcpToolCatalog.catalogDigest, managementMcpToolCatalog.toolsWithConfirmationSchema",
      missingCapabilities: [],
      status: "error"
    },
    liveDataLoaded: true,
    loadError: "",
    mcpCatalog: { metadataVersion: 4, status: "ok", toolsWithAccess: 12, toolsWithExecution: 12, toolsWithLifecycle: 12, toolsWithSafety: 12 },
    mcpHealth: { status: "ok", message: "ok" },
    session: { actor: "admin-key", authenticated: true, requiresLogin: true }
  });

  const apiRow = rows.find((row) => row.key === "api");
  assert.equal(apiRow?.detailKey, "message.apiContractIncompatibleManagementCatalog");
  assert.equal(apiRow?.detailParams, undefined);
});

test("connection diagnostics hides unknown raw API contract issue keys", () => {
  const rows = buildConnectionDiagnosticRows({
    apiHealth: {
      code: "api_contract_incompatible",
      contractIssues: ["futureContract.requiredField"],
      message: "system info contract issues: futureContract.requiredField",
      missingCapabilities: [],
      status: "error"
    },
    liveDataLoaded: true,
    loadError: "",
    mcpCatalog: { metadataVersion: 4, status: "ok", toolsWithAccess: 12, toolsWithExecution: 12, toolsWithLifecycle: 12, toolsWithSafety: 12 },
    mcpHealth: { status: "ok", message: "ok" },
    session: { actor: "admin-key", authenticated: true, requiresLogin: true }
  });

  const apiRow = rows.find((row) => row.key === "api");
  assert.equal(apiRow?.detailKey, "message.apiContractIncompatibleUnknown");
  assert.equal(apiRow?.detailParams, undefined);
});

test("connection diagnostics treats fallback data as warning and MCP failure as error", () => {
  const rows = buildConnectionDiagnosticRows({
    apiHealth: { status: "ok", message: "ok" },
    liveDataLoaded: false,
    loadError: "API unavailable",
    mcpCatalog: { message: "HTTP 503", status: "error" },
    mcpHealth: { status: "error", message: "HTTP 503" },
    session: { actor: "local-dev", authenticated: true, requiresLogin: false }
  });

  assert.equal(connectionDiagnosticsSummaryStatus(rows), "error");
  assert.equal(rows.find((row) => row.key === "dataSource")?.status, "warning");
  assert.equal(rows.find((row) => row.key === "dataSource")?.detailKey, "connectionDiagnostics.dataSource.error");
  assert.deepEqual(rows.find((row) => row.key === "mcp")?.detailParams, { detail: "HTTP 503" });
  assert.equal(rows.find((row) => row.key === "mcpCatalog")?.status, "error");
  assert.deepEqual(rows.find((row) => row.key === "mcpCatalog")?.detailParams, { detail: "HTTP 503" });
});

test("connection diagnostics warns on unknown management MCP catalog metadata version", () => {
  const rows = buildConnectionDiagnosticRows({
    apiHealth: { status: "ok", message: "ok" },
    liveDataLoaded: true,
    loadError: "",
    mcpCatalog: { metadataVersion: 5, status: "warning", toolsWithAccess: 12, toolsWithExecution: 12, toolsWithLifecycle: 12, toolsWithSafety: 12 },
    mcpHealth: { status: "ok", message: "ok" },
    session: { actor: "admin-key", authenticated: true, requiresLogin: true }
  });

  assert.equal(connectionDiagnosticsSummaryStatus(rows), "warning");
  assert.equal(rows.find((row) => row.key === "mcpCatalog")?.detailKey, "connectionDiagnostics.mcpCatalog.versionWarning");
  assert.deepEqual(rows.find((row) => row.key === "mcpCatalog")?.detailParams, { version: 5 });
});

test("connection diagnostics blocks missing management MCP catalog metadata", () => {
  const rows = buildConnectionDiagnosticRows({
    apiHealth: { status: "ok", message: "ok" },
    liveDataLoaded: true,
    loadError: "",
    mcpCatalog: { message: "missing safety metadata", status: "error" },
    mcpHealth: { status: "ok", message: "ok" },
    session: { actor: "admin-key", authenticated: true, requiresLogin: true }
  });

  assert.equal(connectionDiagnosticsSummaryStatus(rows), "error");
  assert.equal(rows.find((row) => row.key === "mcpCatalog")?.detailKey, "connectionDiagnostics.mcpCatalog.error");
  assert.deepEqual(rows.find((row) => row.key === "mcpCatalog")?.detailParams, { detail: "missing safety metadata" });
});

test("management MCP catalog diagnostic accepts versioned safety access lifecycle execution and confirmation schema metadata", () => {
  const catalogTools = requiredManagementMcpToolNames.map(catalogToolNamed);

  assert.deepEqual(
    managementMcpCatalogDiagnosticFromResult({ metadataVersion: 4, tools: catalogTools }),
    {
      metadataVersion: 4,
      status: "ok",
      toolsWithAccess: catalogTools.length,
      toolsWithExecution: catalogTools.length,
      toolsWithLifecycle: catalogTools.length,
      toolsWithSafety: catalogTools.length
    }
  );
});

test("management MCP catalog diagnostic requires confirmation schema for write tools", () => {
  const catalogTools = requiredManagementMcpToolNames.map(catalogToolNamed);
  const unsafeTools = catalogTools.map((tool) => (
    tool.name === "apply_permission_package"
      ? { ...tool, inputSchema: { type: "object", properties: {}, required: [] } }
      : tool
  ));

  const diagnostic = managementMcpCatalogDiagnosticFromResult({ metadataVersion: 4, tools: unsafeTools });

  assert.equal(diagnostic.status, "error");
  assert.equal(diagnostic.toolsWithConfirmationSchema, confirmationRequiredToolNames.size - 1);
  assert.equal(diagnostic.confirmationRequiredTools, confirmationRequiredToolNames.size);
  assert.equal(diagnostic.message, `confirmation schema incomplete: ${confirmationRequiredToolNames.size - 1}/${confirmationRequiredToolNames.size}`);

  const rows = buildConnectionDiagnosticRows({
    apiHealth: { status: "ok", message: "ok" },
    liveDataLoaded: true,
    loadError: "",
    mcpCatalog: diagnostic,
    mcpHealth: { status: "ok", message: "ok" },
    session: { actor: "admin-key", authenticated: true, requiresLogin: true }
  });
  const mcpCatalogRow = rows.find((row) => row.key === "mcpCatalog");
  assert.equal(mcpCatalogRow?.detailKey, "connectionDiagnostics.mcpCatalog.confirmationSchema");
  assert.deepEqual(mcpCatalogRow?.detailParams, {
    ready: confirmationRequiredToolNames.size - 1,
    total: confirmationRequiredToolNames.size
  });
});

test("management MCP catalog diagnostic requires the preferred production report tool", () => {
  const legacyOnlyTools = requiredManagementMcpToolNames
    .filter((name) => name !== "export_permission_package_production_report")
    .concat("export_permission_package_production_evidence")
    .map(catalogToolNamed);
  const diagnostic = managementMcpCatalogDiagnosticFromResult({ metadataVersion: 4, tools: legacyOnlyTools });

  assert.equal(diagnostic.status, "error");
  assert.deepEqual(diagnostic.missingRequiredTools, ["export_permission_package_production_report"]);

  const rows = buildConnectionDiagnosticRows({
    apiHealth: { status: "ok", message: "ok" },
    liveDataLoaded: true,
    loadError: "",
    mcpCatalog: diagnostic,
    mcpHealth: { status: "ok", message: "ok" },
    session: { actor: "admin-key", authenticated: true, requiresLogin: true }
  });
  const mcpCatalogRow = rows.find((row) => row.key === "mcpCatalog");
  assert.equal(mcpCatalogRow?.detailKey, "connectionDiagnostics.mcpCatalog.missingTools");
  assert.equal(mcpCatalogRow?.detailParams, undefined);
});

test("management MCP catalog diagnostic required tools cover backend management tools", () => {
  const backendTools = backendManagementMcpToolNames().filter((name) => !legacyManagementMcpToolNames.has(name));
  const missing = backendTools.filter((name) => !requiredManagementMcpToolNames.includes(name));

  assert.deepEqual(missing, []);
});

test("management MCP catalog diagnostic flags incomplete or unknown metadata contracts", () => {
  assert.deepEqual(
    managementMcpCatalogDiagnosticFromResult({ metadataVersion: 5, tools: requiredManagementMcpToolNames.map(catalogToolNamed) }),
    { metadataVersion: 5, status: "warning" }
  );
  assert.equal(
    managementMcpCatalogDiagnosticFromResult({ metadataVersion: 4, tools: [{ ...catalogTool, access: undefined }] }).status,
    "error"
  );
  assert.equal(
    managementMcpCatalogDiagnosticFromResult({ metadataVersion: 4, tools: [{ ...catalogTool, lifecycle: undefined }] }).status,
    "error"
  );
  assert.equal(
    managementMcpCatalogDiagnosticFromResult({ metadataVersion: 4, tools: [{ ...catalogTool, execution: undefined }] }).status,
    "error"
  );
  assert.equal(managementMcpCatalogDiagnosticFromResult({ tools: [catalogTool] }).message, "missing metadataVersion");
});
