import assert from "node:assert/strict";
import test from "node:test";

import {
  createAiAdminApprovalJourneyConfig,
  evaluateAiAdminApprovalJourney
} from "../src/aiAdminApprovalJourney.ts";

const now = "2026-06-05T09:00:00Z";

test("createAiAdminApprovalJourneyConfig generates deterministic approval run ids", () => {
  const config = createAiAdminApprovalJourneyConfig("Demo Run");

  assert.equal(config.runId, "ui-approval-demo-run");
  assert.equal(config.rootTenantId, "tenant-root-ui-approval-demo-run");
  assert.equal(config.childTenantId, "tenant-child-ui-approval-demo-run");
  assert.equal(config.grandchildTenantId, "tenant-grandchild-ui-approval-demo-run");
  assert.equal(config.workspaceId, "ws-ai-admin-approval");
  assert.equal(config.mcpEndpoint, "http://127.0.0.1:8787/mcp");
  assert.equal(config.subjectSelector, "user:support-*");
  assert.equal(config.subjectId, "user:support-001");
  assert.equal(config.templateId, "support-ticket-triage");
});

test("evaluateAiAdminApprovalJourney reports missing evidence before the live run", () => {
  const config = createAiAdminApprovalJourneyConfig("missing");
  const evaluation = evaluateAiAdminApprovalJourney({
    accessProfile: null,
    application: null,
    approvalRequest: null,
    auditEvent: null,
    config,
    data: null,
    result: null
  });

  assert.equal(evaluation.totalCount, 8);
  assert.equal(evaluation.completeCount, 0);
  assert.deepEqual(new Set(evaluation.steps.map((step) => step.status)), new Set(["missing"]));
});

test("evaluateAiAdminApprovalJourney completes with approval, runtime, profile, and audit evidence", () => {
  const config = createAiAdminApprovalJourneyConfig("complete");
  const caller = agent({
    channelType: "local",
    id: "agt_caller",
    name: "AI Admin Approval Caller",
    tenantId: config.childTenantId,
    workspaceId: config.workspaceId
  });
  const target = agent({
    channelConfig: { endpoint: config.mcpEndpoint, transport: "streamable-http" },
    channelType: "mcp",
    id: "agt_target",
    name: "AI Admin Approval MCP Target",
    tenantId: config.rootTenantId,
    workspaceId: config.workspaceId
  });
  const readCapability = capability({
    action: "read",
    id: "cap_read",
    key: config.readTool,
    targetId: target.id
  });
  const writeCapability = capability({
    action: "write",
    id: "cap_write",
    key: config.writeTool,
    targetId: target.id
  });
  const deniedCapability = capability({
    action: "export",
    discoveryStatus: "pending_review",
    id: "cap_denied",
    key: config.deniedTool,
    targetId: target.id
  });
  const approvalRequest = {
    allowedCapabilityIds: [readCapability.id, writeCapability.id],
    allowedCapabilityKeys: [config.readTool, config.writeTool],
    callerInstanceId: caller.id,
    createdAt: now,
    dataScopes: supportScopes(config),
    draftId: "draft_1",
    id: "approval_1",
    policyGate: { canApplyDirectly: false, decision: "approval_required", nextActions: [], policyVersion: 1, reasons: [] },
    policyVersion: 1,
    region: config.region,
    requestText: config.requestText,
    status: "approved",
    subjectSelector: config.subjectSelector,
    targetId: target.id,
    templateId: config.templateId,
    templateVersion: 1,
    tenantId: config.childTenantId,
    updatedAt: now,
    workspaceId: config.workspaceId
  };
  const application = {
    allowedCapabilityIds: [readCapability.id, writeCapability.id],
    allowedCapabilityKeys: [config.readTool, config.writeTool],
    appliedAt: now,
    callerInstanceId: caller.id,
    dataScopes: supportScopes(config),
    draftId: "draft_1",
    id: "application_1",
    instanceAssignmentIds: ["ia_read", "ia_write"],
    region: config.region,
    requestText: config.requestText,
    subjectSelector: config.subjectSelector,
    targetId: target.id,
    templateId: config.templateId,
    templateVersion: 1,
    tenantEntitlementIds: ["te_read", "te_write"],
    tenantId: config.childTenantId,
    workspaceAssignmentIds: ["wa_read", "wa_write"],
    workspaceId: config.workspaceId
  };
  const auditEvent = {
    action: "permission_package.applied",
    actor: "system",
    createdAt: now,
    id: "audit_1",
    metadata: {
      applicationId: application.id,
      approvalRequestId: approvalRequest.id
    },
    resourceId: application.id,
    resourceType: "permission_package",
    tenantId: config.childTenantId,
    workspaceId: config.workspaceId
  };
  const data = {
    accessGrants: [],
    agents: [caller, target],
    apiBase: "http://127.0.0.1:9090",
    auditEvents: [auditEvent],
    capabilities: [readCapability, writeCapability, deniedCapability],
    capabilityAssignmentsLoadedFromApi: true,
    capabilitiesLoadedFromApi: true,
    channels: [],
    evidenceRuns: [],
    grantsLoadedFromApi: true,
    instanceAssignments: [],
    loadedFromApi: true,
    providers: [],
    routePolicies: [],
    routePoliciesLoadedFromApi: true,
    systemMetrics: [],
    tenantEntitlements: [],
    traces: [
      trace({ capabilityId: writeCapability.id, decision: "allowed", id: "trace_allowed", runId: config.runId, targetAgentId: target.id }),
      trace({ capabilityId: deniedCapability.id, decision: "denied", id: "trace_denied", runId: config.runId, targetAgentId: target.id })
    ],
    workspaceAssignments: []
  };
  const accessProfile = {
    apiBase: "http://127.0.0.1:9090",
    generatedAt: now,
    grants: [],
    loadedFromApi: true,
    recentTraces: data.traces,
    scopeTenants: [
      tenant({ id: config.childTenantId, parentTenantId: config.rootTenantId }),
      tenant({ id: config.grandchildTenantId, parentTenantId: config.childTenantId })
    ],
    summary: {
      capabilityCount: 2,
      grantCount: 2,
      instanceAssignmentCount: 2,
      recentAllowedTraceCount: 1,
      recentDeniedTraceCount: 1,
      targetCount: 1,
      tenantCount: 2,
      workspaceAssignmentCount: 2
    },
    tenant: tenant({ id: config.childTenantId, parentTenantId: config.rootTenantId })
  };
  const evaluation = evaluateAiAdminApprovalJourney({
    accessProfile,
    application,
    approvalRequest,
    auditEvent,
    config,
    data,
    result: {
      allowedStatus: 200,
      applicationId: application.id,
      approvalRequestId: approvalRequest.id,
      callerId: caller.id,
      deniedStatus: 403,
      targetId: target.id,
      toolListStatus: 200
    }
  });

  assert.equal(evaluation.totalCount, 8);
  assert.equal(evaluation.completeCount, 8);
  assert.deepEqual(evaluation.steps.map((step) => step.status), Array.from({ length: 8 }, () => "complete"));
  assert.equal(evaluation.readCapability?.id, readCapability.id);
  assert.equal(evaluation.writeCapability?.id, writeCapability.id);
  assert.equal(evaluation.deniedCapability?.id, deniedCapability.id);
});

function agent(overrides) {
  return {
    channelType: "local",
    createdAt: now,
    credentialVersion: 1,
    id: "agent",
    name: "Agent",
    status: "active",
    tenantId: "tenant",
    updatedAt: now,
    workspaceId: "workspace",
    ...overrides
  };
}

function capability(overrides) {
  return {
    action: "read",
    dataDomains: ["support"],
    dataScopes: supportScopes(createAiAdminApprovalJourneyConfig("complete")),
    description: "MCP tool",
    discoveredAt: now,
    discoveryStatus: "approved",
    displayName: overrides.key,
    enforcementMode: "gateway",
    id: "capability",
    key: "search_customer",
    riskLevel: "low",
    sensitivity: "internal",
    targetId: "target",
    type: "mcp_tool",
    updatedAt: now,
    version: 1,
    ...overrides
  };
}

function tenant(overrides) {
  return {
    createdAt: now,
    id: "tenant",
    level: 1,
    name: overrides.id,
    status: "active",
    updatedAt: now,
    ...overrides
  };
}

function trace(overrides) {
  return {
    createdAt: now,
    decision: "allowed",
    id: "trace",
    routeKey: "tools/call",
    routeType: "mcp",
    targetAgentId: "target",
    ...overrides
  };
}

function supportScopes(config) {
  return [
    {
      dataDomain: "support",
      region: config.region,
      tenantFilter: `tenant_id = '${config.childTenantId}'`
    }
  ];
}
