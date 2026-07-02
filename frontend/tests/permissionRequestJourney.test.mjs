import assert from "node:assert/strict";
import test from "node:test";

import {
  currentPermissionRequestWizardStep,
  permissionRequestProcessStepStatuses
} from "../src/permissionRequestJourney.ts";

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

test("permission request journey points to approval before an approved request exists", () => {
  assert.equal(currentPermissionRequestWizardStep({
    application: null,
    approvalRequest: null,
    draft: baseDraft,
    productionReadiness: null
  }), "approval");
});

test("permission request journey points to apply after approval", () => {
  assert.equal(currentPermissionRequestWizardStep({
    application: null,
    approvalRequest: approvedRequest,
    draft: baseDraft,
    productionReadiness: null
  }), "apply");
});

test("permission request journey returns to approval when stored approval has expired", () => {
  assert.equal(currentPermissionRequestWizardStep({
    application: null,
    approvalRequest: { ...approvedRequest, effectiveStatus: "expired", isExpired: true },
    draft: baseDraft,
    productionReadiness: null
  }), "approval");
});

test("permission request journey points to go-live after application", () => {
  assert.equal(currentPermissionRequestWizardStep({
    application,
    approvalRequest: approvedRequest,
    draft: baseDraft,
    productionReadiness: {
      status: "blocked",
      summary: {
        readyCount: 6,
        warningCount: 0,
        blockingCount: 1,
        hasApplication: true,
        hasAllowedTrace: true,
        hasDeniedTrace: false,
        hasAppliedAudit: true,
        accessProfileReady: true
      },
      checks: [],
      latestApplication: application,
      runtimeEvidence: { allowedTrace: { id: "trace-allowed" } },
      auditEvidence: { appliedEvent: { id: "audit-apply" } },
      nextActionCode: "run_denied_runtime_call",
      nextActions: ["Run a denied MCP call that proves blocked tools stay blocked."],
      generatedAt: "2026-06-07T08:03:00Z"
    }
  }), "goLive");
});

test("permission request process states expose one current operator step", () => {
  const steps = [
    { key: "scope", complete: true },
    { key: "template", complete: true },
    { key: "approval", complete: false },
    { key: "apply", complete: false },
    { key: "goLive", complete: false }
  ];
  const statuses = permissionRequestProcessStepStatuses(steps, "approval");

  assert.deepEqual(statuses.map((step) => step.status), ["complete", "complete", "current", "waiting", "waiting"]);
  assert.equal(statuses.filter((step) => step.status === "current").length, 1);
});

test("permission request process states keep completed production journeys complete", () => {
  const statuses = permissionRequestProcessStepStatuses([
    { key: "scope", complete: true },
    { key: "template", complete: true },
    { key: "approval", complete: true },
    { key: "apply", complete: true },
    { key: "goLive", complete: true }
  ], "goLive");

  assert.deepEqual(statuses.map((step) => step.status), ["complete", "complete", "complete", "complete", "complete"]);
});
