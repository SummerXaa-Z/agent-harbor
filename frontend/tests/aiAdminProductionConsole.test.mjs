import assert from "node:assert/strict";
import test from "node:test";

import {
  buildAiAdminProductionConsoleSummary
} from "../src/aiAdminProductionConsole.ts";

const baseDraft = {
  id: "draft-support",
  input: {
    callerInstanceId: "caller-support",
    region: "华东",
    requestText: "给客服助手开通工单处理权限。",
    subjectSelector: "user:support-*",
    targetId: "target-support-mcp",
    templateId: "support-ticket-triage",
    tenantId: "tenant-east",
    workspaceId: "ws-support"
  },
  template: {
    id: "support-ticket-triage",
    version: 1,
    name: "Support ticket triage",
    summary: "Allow ticket reads and bounded updates.",
    allowedActions: ["read", "write"],
    blockedActions: ["export", "delete", "admin"],
    blockedRisks: ["critical"],
    blockedSensitivities: ["restricted"],
    defaultDataDomain: "support",
    guardrails: []
  },
  allowedCapabilities: [{ id: "cap-search", key: "search_customer" }],
  blockedCapabilities: [{ id: "cap-export", key: "export_contracts" }],
  dataScopes: [{ dataDomain: "support", region: "华东", tenantFilter: "tenant_id = 'tenant-east'" }],
  readiness: { canApply: true, missingFields: [], warnings: [] },
  policyGate: {
    decision: "approval_required",
    canApplyDirectly: false,
    policyVersion: 1,
    reasons: [{ id: "risky-write", severity: "warning", message: "write needs approval" }],
    nextActions: ["create_approval_request"]
  },
  simulationRows: []
};

const approvedRequest = {
  id: "approval-1",
  draftId: "draft-support",
  templateId: "support-ticket-triage",
  templateVersion: 1,
  policyVersion: 1,
  tenantId: "tenant-east",
  workspaceId: "ws-support",
  targetId: "target-support-mcp",
  callerInstanceId: "caller-support",
  allowedCapabilityIds: ["cap-search"],
  allowedCapabilityKeys: ["search_customer"],
  policyGate: baseDraft.policyGate,
  status: "approved",
  createdAt: "2026-06-07T08:00:00Z",
  updatedAt: "2026-06-07T08:01:00Z",
  expiresAt: "2026-06-08T08:00:00Z"
};

const application = {
  id: "app-1",
  draftId: "draft-support",
  templateId: "support-ticket-triage",
  templateVersion: 1,
  tenantId: "tenant-east",
  workspaceId: "ws-support",
  targetId: "target-support-mcp",
  callerInstanceId: "caller-support",
  subjectSelector: "user:support-*",
  dataScopes: baseDraft.dataScopes,
  allowedCapabilityIds: ["cap-search"],
  allowedCapabilityKeys: ["search_customer"],
  tenantEntitlementIds: ["te-1"],
  workspaceAssignmentIds: ["wa-1"],
  instanceAssignmentIds: ["ia-1"],
  appliedAt: "2026-06-07T08:02:00Z"
};

test("production console summary stays pending before approval and application evidence", () => {
  const summary = buildAiAdminProductionConsoleSummary({
    application: null,
    approvalRequest: null,
    draft: baseDraft,
    productionReadiness: null
  });

  assert.equal(summary.status, "pending");
  assert.equal(summary.readyCount, 1);
  assert.equal(summary.totalCount, 5);
  assert.equal(summary.primaryActionKey, "action.createApprovalRequest");
  assert.equal(summary.steps[0].detailKey, "productionConsole.requestConfigured");
  assert.equal(summary.steps[0].detail, "support-ticket-triage");
  assert.equal(summary.steps[0].metric, "search_customer allowed / export_contracts blocked");
  assert.deepEqual(
    summary.steps.map((step) => [step.key, step.status]),
    [
      ["request", "ready"],
        ["approval", "pending"],
      ["application", "pending"],
      ["runtime", "pending"],
      ["production", "pending"]
    ]
  );
});

test("production console summary reports ready after approval application and runtime evidence", () => {
  const summary = buildAiAdminProductionConsoleSummary({
    application,
    approvalRequest: approvedRequest,
    draft: baseDraft,
    productionReadiness: {
      status: "ready",
      summary: {
        readyCount: 8,
        warningCount: 0,
        blockingCount: 0,
        hasApplication: true,
        hasAllowedTrace: true,
        hasDeniedTrace: true,
        hasAppliedAudit: true,
        accessProfileReady: true
      },
      checks: [],
      latestApplication: application,
      runtimeEvidence: {
        allowedTrace: { id: "trace-allowed" },
        deniedTrace: { id: "trace-denied" }
      },
      auditEvidence: { appliedEvent: { id: "audit-apply" } },
      nextActions: [],
      generatedAt: "2026-06-07T08:03:00Z"
    }
  });

  assert.equal(summary.status, "ready");
  assert.equal(summary.readyCount, 5);
  assert.equal(summary.primaryActionKey, "action.exportProductionEvidence");
  assert.deepEqual(summary.steps.map((step) => step.status), [
    "ready",
    "ready",
    "ready",
    "ready",
    "ready"
  ]);
  assert.equal(summary.steps.find((step) => step.key === "application")?.detail, "app-1");
  assert.equal(summary.steps.find((step) => step.key === "runtime")?.metric, "2/2");
  assert.equal(summary.steps.find((step) => step.key === "runtime")?.detailKey, "productionConsole.runtimeReady");
  assert.equal(summary.steps.find((step) => step.key === "production")?.detailKey, "productionConsole.productionReady");
});

test("production console summary treats applied production evidence as approval satisfied", () => {
  const summary = buildAiAdminProductionConsoleSummary({
    application,
    approvalRequest: null,
    draft: baseDraft,
    productionReadiness: {
      status: "ready",
      summary: {
        readyCount: 8,
        warningCount: 0,
        blockingCount: 0,
        hasApplication: true,
        hasAllowedTrace: true,
        hasDeniedTrace: true,
        hasAppliedAudit: true,
        accessProfileReady: true
      },
      checks: [],
      latestApplication: application,
      runtimeEvidence: {
        allowedTrace: { id: "trace-allowed" },
        deniedTrace: { id: "trace-denied" }
      },
      auditEvidence: { appliedEvent: { id: "audit-apply" } },
      nextActions: [],
      generatedAt: "2026-06-07T08:03:00Z"
    }
  });

  const approvalStep = summary.steps.find((step) => step.key === "approval");
  assert.equal(summary.status, "ready");
  assert.equal(approvalStep?.status, "ready");
  assert.equal(approvalStep?.detailKey, "productionConsole.approvalSatisfied");
});

test("production console summary treats direct apply packages as approval ready", () => {
  const directDraft = {
    ...baseDraft,
    policyGate: {
      decision: "allow",
      canApplyDirectly: true,
      policyVersion: 1,
      reasons: [],
      nextActions: []
    }
  };

  const summary = buildAiAdminProductionConsoleSummary({
    application: null,
    approvalRequest: null,
    draft: directDraft,
    productionReadiness: null
  });

  assert.equal(summary.steps.find((step) => step.key === "request")?.status, "ready");
  assert.equal(summary.steps.find((step) => step.key === "approval")?.status, "ready");
  assert.equal(summary.steps.find((step) => step.key === "approval")?.detailKey, "productionConsole.approvalNotRequired");
});
