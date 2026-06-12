import assert from "node:assert/strict";
import test from "node:test";

import {
  createCoreJourneyConfig,
  defaultCoreJourneyForm,
  evaluateCoreJourney
} from "../src/coreJourney.ts";

const now = "2026-06-03T04:00:00Z";

test("createCoreJourneyConfig generates a scoped tenant tree for a run", () => {
  const config = createCoreJourneyConfig(defaultCoreJourneyForm, "2026-06-03 12:00");

  assert.equal(config.runId, "ui-core-2026-06-03-12-00");
  assert.equal(config.rootTenantId, "tenant-root-ui-core-2026-06-03-12-00");
  assert.equal(config.childTenantId, "tenant-child-ui-core-2026-06-03-12-00");
  assert.equal(config.grandchildTenantId, "tenant-grandchild-ui-core-2026-06-03-12-00");
  assert.equal(config.subjectSelector, "user:ui-core-2026-06-03-12-00-*");
  assert.equal(config.subjectId, "user:ui-core-2026-06-03-12-00-operator");
});

test("evaluateCoreJourney requires tenant, capability, grant, runtime, and profile evidence", () => {
  const config = createCoreJourneyConfig(defaultCoreJourneyForm, "demo");
  const caller = {
    id: "agt_caller",
    tenantId: config.childTenantId,
    workspaceId: config.workspaceId,
    name: "Core Journey Caller",
    channelType: "local",
    credentialVersion: 0,
    status: "active",
    createdAt: now,
    updatedAt: now
  };
  const target = {
    id: "agt_target",
    tenantId: config.rootTenantId,
    workspaceId: config.workspaceId,
    name: "Core Journey MCP Target",
    channelType: "mcp",
    channelConfig: { endpoint: config.mcpEndpoint },
    credentialVersion: 0,
    status: "active",
    createdAt: now,
    updatedAt: now
  };
  const allowedCapability = {
    id: "cap_allowed",
    targetId: target.id,
    type: "mcp_tool",
    key: config.allowedTool,
    displayName: "Search Customer",
    action: "read",
    dataScopes: [{ dataDomain: "crm", region: "us-east", tenantFilter: `tenant_id = '${config.childTenantId}'` }],
    sensitivity: "confidential",
    riskLevel: "medium",
    enforcementMode: "gateway",
    discoveryStatus: "approved",
    version: 1,
    discoveredAt: now,
    updatedAt: now
  };
  const deniedCapability = {
    ...allowedCapability,
    id: "cap_denied",
    key: config.deniedTool,
    displayName: "Export Contracts",
    action: "export",
    dataScopes: [],
    discoveryStatus: "pending_review"
  };
  const tenantEntitlement = {
    id: "ent_1",
    tenantId: config.childTenantId,
    targetId: target.id,
    capabilityId: allowedCapability.id,
    effect: "allow",
    status: "enabled",
    priority: 50,
    createdAt: now,
    updatedAt: now
  };
  const workspaceAssignment = {
    id: "wsa_1",
    tenantEntitlementId: tenantEntitlement.id,
    tenantId: config.childTenantId,
    workspaceId: config.workspaceId,
    effect: "allow",
    status: "enabled",
    createdAt: now,
    updatedAt: now
  };
  const instanceAssignment = {
    id: "ina_1",
    workspaceAssignmentId: workspaceAssignment.id,
    tenantId: config.childTenantId,
    workspaceId: config.workspaceId,
    callerInstanceId: caller.id,
    effect: "allow",
    status: "enabled",
    createdAt: now,
    updatedAt: now
  };
  const data = {
    providers: [],
    channels: [],
    agents: [caller, target],
    accessGrants: [],
    capabilities: [allowedCapability, deniedCapability],
    tenantEntitlements: [tenantEntitlement],
    workspaceAssignments: [workspaceAssignment],
    instanceAssignments: [instanceAssignment],
    traces: [
      { id: "trace_allowed", runId: config.runId, targetAgentId: target.id, decision: "allowed", routeType: "mcp", routeKey: "tools/call", createdAt: now },
      { id: "trace_denied", runId: config.runId, targetAgentId: target.id, decision: "denied", routeType: "mcp", routeKey: "tools/call", createdAt: now }
    ],
    auditEvents: [],
    routePolicies: [],
    evidenceRuns: [],
    systemMetrics: [],
    loadedFromApi: true,
    grantsLoadedFromApi: true,
    capabilitiesLoadedFromApi: true,
    capabilityAssignmentsLoadedFromApi: true,
    routePoliciesLoadedFromApi: true,
    apiBase: "http://127.0.0.1:9090"
  };
  const profile = {
    tenant: { id: config.childTenantId, parentTenantId: config.rootTenantId, level: 1, name: "Core Journey Team", status: "active", createdAt: now, updatedAt: now },
    scopeTenants: [
      { id: config.childTenantId, parentTenantId: config.rootTenantId, level: 1, name: "Core Journey Team", status: "active", createdAt: now, updatedAt: now },
      { id: config.grandchildTenantId, parentTenantId: config.childTenantId, level: 2, name: "Core Journey Project", status: "active", createdAt: now, updatedAt: now }
    ],
    summary: {
      tenantCount: 2,
      grantCount: 1,
      targetCount: 1,
      capabilityCount: 1,
      workspaceAssignmentCount: 1,
      instanceAssignmentCount: 1,
      recentAllowedTraceCount: 1,
      recentDeniedTraceCount: 1
    },
    grants: [],
    recentTraces: data.traces,
    generatedAt: now,
    loadedFromApi: true,
    apiBase: "http://127.0.0.1:9090"
  };

  const evaluation = evaluateCoreJourney(data, profile, config);

  assert.equal(evaluation.completeCount, evaluation.totalCount);
  assert.deepEqual(evaluation.steps.map((step) => step.status), [
    "complete",
    "complete",
    "complete",
    "complete",
    "complete",
    "complete"
  ]);
});

test("evaluateCoreJourney marks missing evidence without an access profile", () => {
  const config = createCoreJourneyConfig(defaultCoreJourneyForm, "empty");
  const evaluation = evaluateCoreJourney(null, null, config);

  assert.equal(evaluation.completeCount, 0);
  assert.equal(evaluation.steps.find((step) => step.key === "tenantTree").status, "missing");
  assert.equal(evaluation.steps.find((step) => step.key === "runtimeEvidence").metric, "0/0");
});
