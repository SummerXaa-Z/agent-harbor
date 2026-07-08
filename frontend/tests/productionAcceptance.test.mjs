import assert from "node:assert/strict";
import test from "node:test";

import {
  buildProductionAcceptanceCenter,
  productionAcceptanceReportFilename
} from "../src/productionAcceptance.ts";

function productionSummary(overrides = {}) {
  return {
    primaryActionKey: "action.checkProductionReadiness",
    readyCount: 4,
    status: "pending",
    steps: [
      { detail: "", detailKey: "productionConsole.requestConfigured", key: "request", labelKey: "productionConsole.request", status: "ready" },
      { detail: "", detailKey: "productionConsole.approvalSatisfied", key: "approval", labelKey: "productionConsole.approval", status: "ready" },
      { detail: "", detailKey: "productionConsole.applicationRecorded", key: "application", labelKey: "productionConsole.application", status: "ready" },
      { detail: "", detailKey: "productionConsole.runtimeReady", key: "runtime", labelKey: "productionConsole.runtime", status: "ready" },
      { detail: "", detailKey: "productionConsole.productionPending", key: "production", labelKey: "productionConsole.productionReadiness", status: "pending" }
    ],
    totalCount: 5,
    ...overrides
  };
}

function productionReadiness(overrides = {}) {
  return {
    auditEvidence: { appliedEvent: { id: "audit-1" } },
    checks: [
      { code: "application_present", message: "Application is present.", severity: "info" },
      { code: "runtime_allowed_trace_present", message: "Allowed runtime trace is present.", severity: "info" }
    ],
    generatedAt: "2026-06-15T10:00:00Z",
    nextActions: ["Production readiness is complete."],
    runtimeEvidence: {
      allowedTrace: { id: "trace-allow" },
      deniedTrace: { id: "trace-deny" }
    },
    status: "ready",
    summary: {
      accessProfileReady: true,
      blockingCount: 0,
      hasAllowedTrace: true,
      hasAppliedAudit: true,
      hasApplication: true,
      hasDeniedTrace: true,
      readyCount: 9,
      warningCount: 0
    },
    ...overrides
  };
}

test("production acceptance center exports the report when status and connection are ready", () => {
  const center = buildProductionAcceptanceCenter({
    connectionStatus: "ok",
    liveDataAvailable: true,
    productionReadiness: productionReadiness(),
    productionSummary: productionSummary({
      primaryActionKey: "action.exportAcceptanceReport",
      readyCount: 5,
      status: "ready",
      steps: productionSummary().steps.map((step) => ({ ...step, status: "ready" }))
    })
  });

  assert.equal(center.status, "ready");
  assert.equal(center.primaryAction, "export_acceptance_report");
  assert.equal(center.readyCount, center.totalCount);
  assert.equal(center.blockingCount, 0);
  assert.deepEqual(center.blockers, []);
});

test("production acceptance center sends blocked permission changes back to the change workspace", () => {
  const center = buildProductionAcceptanceCenter({
    connectionStatus: "ok",
    liveDataAvailable: true,
    productionReadiness: productionReadiness({
      checks: [
        { code: "runtime_denied_trace_present", message: "Run a denied MCP call.", severity: "blocking" }
      ],
      nextActions: ["Run a denied MCP call that proves blocked tools stay blocked."],
      status: "blocked",
      summary: {
        accessProfileReady: true,
        blockingCount: 1,
        hasAllowedTrace: true,
        hasAppliedAudit: true,
        hasApplication: true,
        hasDeniedTrace: false,
        readyCount: 8,
        warningCount: 0
      }
    }),
    productionSummary: productionSummary({
      status: "blocked",
      steps: productionSummary().steps.map((step) => step.key === "runtime" ? { ...step, status: "blocked" } : step)
    })
  });

  assert.equal(center.status, "blocked");
  assert.equal(center.primaryAction, "open_permission_change");
  assert.equal(center.blockingCount, 1);
  assert.equal(center.blockers[0]?.labelKey, "productionAcceptance.blocker.runtime_denied_trace_present");
});

test("production acceptance center routes connection warnings to diagnostics", () => {
  const center = buildProductionAcceptanceCenter({
    connectionStatus: "warning",
    liveDataAvailable: true,
    productionReadiness: productionReadiness(),
    productionSummary: productionSummary({ status: "ready" })
  });

  assert.equal(center.status, "attention");
  assert.equal(center.primaryAction, "run_diagnostics");
  assert.equal(center.blockers[0]?.key, "connection");
});

test("production acceptance center blocks fallback data from production actions", () => {
  const center = buildProductionAcceptanceCenter({
    connectionStatus: "ok",
    liveDataAvailable: false,
    productionReadiness: productionReadiness(),
    productionSummary: productionSummary({ status: "ready" })
  });

  assert.equal(center.status, "blocked");
  assert.equal(center.primaryAction, "run_diagnostics");
  assert.equal(center.blockers[0]?.labelKey, "productionAcceptance.blocker.liveData");
});

test("production acceptance report filename uses go-live status wording", () => {
  const filename = productionAcceptanceReportFilename({
    generatedAt: "2026-06-15T10:00:00Z",
    reportDigest: "8bf0a7f6d58485e4d0f2359d6b4d0f77b3a7f62a64866f5d9cbdd73efc4ad123",
    scope: {
      callerInstanceId: "agt-caller",
      targetId: "agt-target",
      templateId: "sales-readonly",
      tenantId: "tenant-support",
      workspaceId: "ws-support"
    },
    status: "ready"
  });

  assert.match(filename, /^agentharbor-go-live-status-/);
  assert.match(filename, /-ready-/);
  assert.match(filename, /-digest-8bf0a7f6d584-/);
  assert.match(filename, /\.json$/);
  assert.doesNotMatch(filename, /evidence/i);
});
