import assert from "node:assert/strict";
import test from "node:test";

import { buildResourceLifecycleSummary } from "../src/resourceLifecycle.ts";

const now = "2026-06-12T10:00:00Z";

function agent(overrides = {}) {
  return {
    channelType: "mcp",
    createdAt: now,
    credentialVersion: 1,
    id: "agt-target",
    name: "Ticket Tool Service",
    status: "active",
    tenantId: "tenant-support",
    updatedAt: now,
    workspaceId: "ws-support",
    ...overrides
  };
}

function capability(overrides = {}) {
  return {
    action: "read",
    dataDomains: ["support"],
    discoveredAt: now,
    discoveryStatus: "approved",
    displayName: "Search ticket",
    enforcementMode: "gateway",
    id: "cap-search",
    key: "search_ticket",
    riskLevel: "low",
    sensitivity: "internal",
    targetId: "agt-target",
    type: "mcp_tool",
    updatedAt: now,
    version: 1,
    ...overrides
  };
}

function entitlement(overrides = {}) {
  return {
    capabilityId: "cap-search",
    createdAt: now,
    effect: "allow",
    id: "ent-support",
    priority: 40,
    status: "enabled",
    targetId: "agt-target",
    tenantId: "tenant-support",
    updatedAt: now,
    ...overrides
  };
}

function workspaceAssignment(overrides = {}) {
  return {
    createdAt: now,
    effect: "allow",
    id: "wsa-support",
    status: "enabled",
    tenantEntitlementId: "ent-support",
    tenantId: "tenant-support",
    updatedAt: now,
    workspaceId: "ws-support",
    ...overrides
  };
}

function instanceAssignment(overrides = {}) {
  return {
    callerInstanceId: "agt-caller",
    createdAt: now,
    effect: "allow",
    id: "ia-support",
    status: "enabled",
    tenantId: "tenant-support",
    updatedAt: now,
    workspaceAssignmentId: "wsa-support",
    workspaceId: "ws-support",
    ...overrides
  };
}

function trace(overrides = {}) {
  return {
    callerAgentId: "agt-caller",
    createdAt: now,
    decision: "allowed",
    id: "trace-1",
    routeType: "mcp",
    targetAgentId: "agt-target",
    tenantId: "tenant-support",
    workspaceId: "ws-support",
    ...overrides
  };
}

test("resource lifecycle summarizes callers, MCP targets, permissions, and runtime state", () => {
  const summary = buildResourceLifecycleSummary({
    agents: [
      agent({ channelType: "local", id: "agt-caller", name: "Support Assistant" }),
      agent()
    ],
    capabilities: [capability()],
    instanceAssignments: [instanceAssignment()],
    routePolicies: [],
    tenantEntitlements: [entitlement()],
    traces: [trace()],
    workspaceAssignments: [workspaceAssignment()]
  });

  assert.equal(summary.totalResources, 2);
  assert.equal(summary.activeResources, 2);
  assert.equal(summary.callers, 1);
  assert.equal(summary.mcpTargets, 1);
  assert.equal(summary.readyResources, 2);
  assert.equal(summary.needsAttention, 0);
  assert.deepEqual(summary.items.map((item) => item.status), ["ready", "ready"]);
  assert.equal(summary.items.find((item) => item.id === "agt-target").capabilityCount, 1);
});

test("resource lifecycle directs incomplete MCP targets to the right next action", () => {
  const summary = buildResourceLifecycleSummary({
    agents: [
      agent({ credentialVersion: 0, id: "agt-missing-credential", name: "Billing Tool" }),
      agent({ id: "agt-missing-capability", name: "Contract Tool" }),
      agent({ id: "agt-needs-approval", name: "Ticket Export" }),
      agent({ id: "agt-disabled", name: "Legacy Tool", status: "disabled" })
    ],
    capabilities: [
      capability({ id: "cap-export", key: "export_contracts", targetId: "agt-needs-approval" })
    ],
    instanceAssignments: [],
    routePolicies: [],
    tenantEntitlements: [],
    traces: [],
    workspaceAssignments: []
  });

  const byId = Object.fromEntries(summary.items.map((item) => [item.id, item]));

  assert.equal(byId["agt-missing-credential"].status, "needs_credentials");
  assert.equal(byId["agt-missing-credential"].detailKey, "resource.detail.needsCredentials");
  assert.equal(byId["agt-missing-credential"].nextActionKind, "rotate_credential");
  assert.equal(byId["agt-missing-credential"].nextActionHash, "#registry");
  assert.equal(byId["agt-missing-capability"].status, "needs_capabilities");
  assert.equal(byId["agt-missing-capability"].detailKey, "resource.detail.needsCapabilities");
  assert.equal(byId["agt-missing-capability"].nextActionKind, "review_capabilities");
  assert.equal(byId["agt-missing-capability"].nextActionHash, "#capabilities");
  assert.equal(byId["agt-needs-approval"].status, "needs_approval");
  assert.equal(byId["agt-needs-approval"].detailKey, "resource.detail.needsApproval");
  assert.equal(byId["agt-needs-approval"].nextActionKind, "start_permission_change");
  assert.equal(byId["agt-needs-approval"].nextActionHash, "#ai-admin");
  assert.equal(byId["agt-disabled"].status, "disabled");
  assert.equal(byId["agt-disabled"].detailKey, "resource.detail.disabled");
  assert.equal(byId["agt-disabled"].nextActionKind, "review_resource");
  assert.equal(summary.needsAttention, 4);
});

test("resource lifecycle exposes access query setup gaps", () => {
  const targetOnly = buildResourceLifecycleSummary({
    agents: [agent()],
    capabilities: [],
    instanceAssignments: [],
    routePolicies: [],
    tenantEntitlements: [],
    traces: [],
    workspaceAssignments: []
  });
  const callerOnly = buildResourceLifecycleSummary({
    agents: [agent({ channelType: "local", id: "agt-caller", name: "Support Assistant" })],
    capabilities: [],
    instanceAssignments: [],
    routePolicies: [],
    tenantEntitlements: [],
    traces: [],
    workspaceAssignments: []
  });
  const queryReady = buildResourceLifecycleSummary({
    agents: [
      agent({ channelType: "local", id: "agt-caller", name: "Support Assistant" }),
      agent()
    ],
    capabilities: [capability()],
    instanceAssignments: [],
    routePolicies: [],
    tenantEntitlements: [],
    traces: [],
    workspaceAssignments: []
  });

  assert.deepEqual(targetOnly.setupGaps.map((gap) => gap.kind), ["caller", "capability"]);
  assert.equal(targetOnly.setupGaps[0].titleKey, "resource.setupGap.caller.title");
  assert.equal(targetOnly.setupGaps[1].actionHash, "#capabilities");
  assert.deepEqual(callerOnly.setupGaps.map((gap) => gap.kind), ["target"]);
  assert.equal(callerOnly.setupGaps[0].detailKey, "resource.setupGap.target.detail");
  assert.deepEqual(queryReady.setupGaps, []);
});

test("resource lifecycle setup gaps honor the current management scope", () => {
  const scopedGaps = buildResourceLifecycleSummary({
    agents: [
      agent({ channelType: "local", id: "agt-east-caller", name: "East Caller", tenantId: "tenant-east", workspaceId: "ws-east" }),
      agent({ id: "agt-east-target", name: "East Target", tenantId: "tenant-east", workspaceId: "ws-east" })
    ],
    capabilities: [capability({ id: "cap-east", targetId: "agt-east-target" })],
    instanceAssignments: [],
    routePolicies: [],
    scope: { tenantId: "tenant-west", workspaceId: "ws-west" },
    tenantEntitlements: [],
    traces: [],
    workspaceAssignments: []
  });
  const scopedCallerOnly = buildResourceLifecycleSummary({
    agents: [
      agent({ channelType: "local", id: "agt-west-caller", name: "West Caller", tenantId: "tenant-west", workspaceId: "ws-west" }),
      agent({ id: "agt-east-target", name: "East Target", tenantId: "tenant-east", workspaceId: "ws-east" })
    ],
    capabilities: [capability({ id: "cap-east", targetId: "agt-east-target" })],
    instanceAssignments: [],
    routePolicies: [],
    scope: { tenantId: "tenant-west", workspaceId: "ws-west" },
    tenantEntitlements: [],
    traces: [],
    workspaceAssignments: []
  });
  const scopedReady = buildResourceLifecycleSummary({
    agents: [
      agent({ channelType: "local", id: "agt-west-caller", name: "West Caller", tenantId: "tenant-west", workspaceId: "ws-west" }),
      agent({ id: "agt-west-target", name: "West Target", tenantId: "tenant-west", workspaceId: "ws-west" })
    ],
    capabilities: [capability({ id: "cap-west", targetId: "agt-west-target" })],
    instanceAssignments: [],
    routePolicies: [],
    scope: { tenantId: "tenant-west", workspaceId: "ws-west" },
    tenantEntitlements: [],
    traces: [],
    workspaceAssignments: []
  });

  assert.deepEqual(scopedGaps.setupGaps.map((gap) => gap.kind), ["caller", "target"]);
  assert.deepEqual(scopedCallerOnly.setupGaps.map((gap) => gap.kind), ["target"]);
  assert.deepEqual(scopedReady.setupGaps, []);
});

test("resource lifecycle treats approved but unverified resources as runtime follow-up", () => {
  const summary = buildResourceLifecycleSummary({
    agents: [agent({ id: "agt-target" })],
    capabilities: [capability()],
    instanceAssignments: [instanceAssignment()],
    routePolicies: [],
    tenantEntitlements: [entitlement()],
    traces: [],
    workspaceAssignments: [workspaceAssignment()]
  });

  assert.equal(summary.items[0].status, "needs_runtime");
  assert.equal(summary.items[0].detailKey, "resource.detail.needsRuntime");
  assert.equal(summary.items[0].nextActionKind, "review_runtime");
  assert.equal(summary.items[0].nextActionHash, "#traces");
  assert.equal(summary.items[0].approvedCapabilityCount, 1);
  assert.equal(summary.items[0].grantCount, 1);
  assert.equal(summary.items[0].runtimeDecisionCount, 0);
});
