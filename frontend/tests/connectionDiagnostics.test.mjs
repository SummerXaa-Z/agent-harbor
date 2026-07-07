import assert from "node:assert/strict";
import test from "node:test";

import {
  buildConnectionDiagnosticRows,
  connectionDiagnosticDetail,
  connectionDiagnosticsSummaryStatus,
  managementMcpCatalogDiagnosticFromResult,
  requiredManagementMcpToolNames
} from "../src/connectionDiagnostics.ts";

const catalogTool = {
  access: { requiredRole: "authenticated_admin", reviewerBound: false, scopeBoundary: "requested_scope" },
  name: "draft_permission_package",
  safety: { approvalMode: "none", mutatesState: false, operationType: "preview", readOnly: true }
};

function catalogToolNamed(name) {
  return { ...catalogTool, name };
}

test("connection diagnostics reports ready when session, API, live data, and MCP are ready", () => {
  const rows = buildConnectionDiagnosticRows({
    apiHealth: { status: "ok", message: "ok" },
    liveDataLoaded: true,
    loadError: "",
    mcpCatalog: { metadataVersion: 1, status: "ok", toolsWithAccess: 12, toolsWithSafety: 12 },
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
    mcpCatalog: { metadataVersion: 1, status: "ok", toolsWithAccess: 12, toolsWithSafety: 12 },
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
      contractIssues: ["managementMcpToolCatalog.requiredMetadata.access"],
      message: "system info contract issues: managementMcpToolCatalog.requiredMetadata.access",
      missingCapabilities: [],
      status: "error"
    },
    liveDataLoaded: true,
    loadError: "",
    mcpCatalog: { metadataVersion: 1, status: "ok", toolsWithAccess: 12, toolsWithSafety: 12 },
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
    mcpCatalog: { metadataVersion: 1, status: "ok", toolsWithAccess: 12, toolsWithSafety: 12 },
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
    mcpCatalog: { metadataVersion: 2, status: "warning", toolsWithAccess: 12, toolsWithSafety: 12 },
    mcpHealth: { status: "ok", message: "ok" },
    session: { actor: "admin-key", authenticated: true, requiresLogin: true }
  });

  assert.equal(connectionDiagnosticsSummaryStatus(rows), "warning");
  assert.equal(rows.find((row) => row.key === "mcpCatalog")?.detailKey, "connectionDiagnostics.mcpCatalog.versionWarning");
  assert.deepEqual(rows.find((row) => row.key === "mcpCatalog")?.detailParams, { version: 2 });
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

test("management MCP catalog diagnostic accepts versioned safety and access metadata", () => {
  const catalogTools = requiredManagementMcpToolNames.map(catalogToolNamed);

  assert.deepEqual(
    managementMcpCatalogDiagnosticFromResult({ metadataVersion: 1, tools: catalogTools }),
    { metadataVersion: 1, status: "ok", toolsWithAccess: catalogTools.length, toolsWithSafety: catalogTools.length }
  );
});

test("management MCP catalog diagnostic requires the preferred production report tool", () => {
  const legacyOnlyTools = requiredManagementMcpToolNames
    .filter((name) => name !== "export_permission_package_production_report")
    .concat("export_permission_package_production_evidence")
    .map(catalogToolNamed);
  const diagnostic = managementMcpCatalogDiagnosticFromResult({ metadataVersion: 1, tools: legacyOnlyTools });

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

test("management MCP catalog diagnostic flags incomplete or unknown metadata contracts", () => {
  assert.deepEqual(
    managementMcpCatalogDiagnosticFromResult({ metadataVersion: 2, tools: requiredManagementMcpToolNames.map(catalogToolNamed) }),
    { metadataVersion: 2, status: "warning" }
  );
  assert.equal(
    managementMcpCatalogDiagnosticFromResult({ metadataVersion: 1, tools: [{ ...catalogTool, access: undefined }] }).status,
    "error"
  );
  assert.equal(managementMcpCatalogDiagnosticFromResult({ tools: [catalogTool] }).message, "missing metadataVersion");
});
