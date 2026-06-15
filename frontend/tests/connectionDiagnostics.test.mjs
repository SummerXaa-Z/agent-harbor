import assert from "node:assert/strict";
import test from "node:test";

import {
  buildConnectionDiagnosticRows,
  connectionDiagnosticsSummaryStatus
} from "../src/connectionDiagnostics.ts";

test("connection diagnostics reports ready when session, API, live data, and MCP are ready", () => {
  const rows = buildConnectionDiagnosticRows({
    apiHealth: { status: "ok", message: "ok" },
    liveDataLoaded: true,
    loadError: "",
    mcpHealth: { status: "ok", message: "ok" },
    session: { actor: "admin-key", authenticated: true, requiresLogin: true }
  });

  assert.deepEqual(rows.map((row) => [row.key, row.status]), [
    ["session", "ok"],
    ["api", "ok"],
    ["dataSource", "ok"],
    ["mcp", "ok"]
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
    mcpHealth: { status: "ok", message: "ok" },
    session: { authenticated: false, requiresLogin: true }
  });

  assert.equal(connectionDiagnosticsSummaryStatus(rows), "error");
  assert.equal(rows.find((row) => row.key === "session")?.detailKey, "connectionDiagnostics.session.signIn");
  assert.equal(rows.find((row) => row.key === "api")?.detailKey, "message.apiContractIncompatible");
  assert.deepEqual(rows.find((row) => row.key === "api")?.detailParams, {
    capabilities: "permission_package_approval_withdraw"
  });
  assert.equal(rows.find((row) => row.key === "dataSource")?.status, "warning");
});

test("connection diagnostics treats fallback data as warning and MCP failure as error", () => {
  const rows = buildConnectionDiagnosticRows({
    apiHealth: { status: "ok", message: "ok" },
    liveDataLoaded: false,
    loadError: "API unavailable",
    mcpHealth: { status: "error", message: "HTTP 503" },
    session: { actor: "local-dev", authenticated: true, requiresLogin: false }
  });

  assert.equal(connectionDiagnosticsSummaryStatus(rows), "error");
  assert.equal(rows.find((row) => row.key === "dataSource")?.status, "warning");
  assert.equal(rows.find((row) => row.key === "dataSource")?.detailKey, "connectionDiagnostics.dataSource.error");
  assert.deepEqual(rows.find((row) => row.key === "mcp")?.detailParams, { detail: "HTTP 503" });
});
