import assert from "node:assert/strict";
import test from "node:test";

import { buildTenantOrganizationModel } from "../src/tenantOrganization.ts";

const now = "2026-06-12T10:00:00Z";

const rootTenant = { createdAt: now, id: "tenant-root", level: 0, name: "Platform", status: "active", updatedAt: now };
const childTenant = { createdAt: now, id: "tenant-child", level: 1, name: "Customer Service", parentTenantId: rootTenant.id, status: "active", updatedAt: now };
const grandchildTenant = { createdAt: now, id: "tenant-grandchild", level: 2, name: "Ticket Project", parentTenantId: childTenant.id, status: "active", updatedAt: now };

const callerAgent = {
  channelType: "local",
  createdAt: now,
  credentialVersion: 0,
  id: "caller-support",
  name: "Support Assistant",
  status: "active",
  tenantId: childTenant.id,
  updatedAt: now,
  workspaceId: "ws-support"
};

const targetAgent = {
  ...callerAgent,
  channelType: "mcp",
  id: "target-ticket",
  name: "Ticket Tool Service"
};

const disabledAgent = {
  ...callerAgent,
  id: "caller-disabled",
  name: "Disabled Caller",
  status: "disabled"
};

const capability = {
  action: "read",
  discoveredAt: now,
  discoveryStatus: "approved",
  displayName: "Search ticket",
  enforcementMode: "gateway",
  id: "cap-search",
  key: "search_ticket",
  riskLevel: "low",
  sensitivity: "internal",
  targetId: targetAgent.id,
  type: "mcp_tool",
  updatedAt: now,
  version: 1
};

const allowEntitlement = {
  capabilityId: capability.id,
  createdAt: now,
  effect: "allow",
  id: "ent-allow",
  priority: 50,
  status: "enabled",
  targetId: targetAgent.id,
  tenantId: childTenant.id,
  updatedAt: now
};

const denyEntitlement = {
  ...allowEntitlement,
  effect: "deny",
  id: "ent-deny"
};

const workspaceAssignment = {
  createdAt: now,
  effect: "allow",
  id: "wa-support",
  status: "enabled",
  tenantEntitlementId: allowEntitlement.id,
  tenantId: childTenant.id,
  updatedAt: now,
  workspaceId: "ws-support"
};

const instanceAssignment = {
  callerInstanceId: callerAgent.id,
  createdAt: now,
  effect: "allow",
  id: "ia-support",
  status: "enabled",
  tenantId: childTenant.id,
  updatedAt: now,
  workspaceAssignmentId: workspaceAssignment.id,
  workspaceId: workspaceAssignment.workspaceId
};

function model(overrides = {}) {
  return buildTenantOrganizationModel({
    agents: [callerAgent, targetAgent, disabledAgent],
    capabilities: [capability],
    instanceAssignments: [instanceAssignment],
    selectedTenantId: childTenant.id,
    tenantEntitlements: [allowEntitlement, denyEntitlement],
    tenants: [grandchildTenant, childTenant, rootTenant],
    workspaceAssignments: [workspaceAssignment],
    ...overrides
  });
}

test("tenant organization model builds a sorted tenant hierarchy", () => {
  const result = model();

  assert.deepEqual(result.nodes.map((node) => node.tenant.id), [rootTenant.id]);
  assert.deepEqual(result.nodes[0].children.map((node) => node.tenant.id), [childTenant.id]);
  assert.deepEqual(result.flatNodes.map((node) => [node.tenant.id, node.depth]), [
    [rootTenant.id, 0],
    [childTenant.id, 1],
    [grandchildTenant.id, 2]
  ]);
});

test("tenant organization model summarizes selected tenant permissions and workspaces", () => {
  const result = model();

  assert.equal(result.selectedTenantId, childTenant.id);
  assert.equal(result.selected?.tenant.name, childTenant.name);
  assert.deepEqual(result.selected?.path.map((tenant) => tenant.id), [rootTenant.id, childTenant.id]);
  assert.equal(result.selected?.activeAgentCount, 2);
  assert.equal(result.selected?.callerCount, 2);
  assert.equal(result.selected?.targetCount, 1);
  assert.equal(result.selected?.allowedPermissionCount, 1);
  assert.equal(result.selected?.deniedPermissionCount, 1);
  assert.equal(result.selected?.workspaceAssignmentCount, 1);
  assert.equal(result.selected?.instanceAssignmentCount, 1);
  assert.deepEqual(result.selected?.workspaces, [
    {
      agentCount: 3,
      assignmentCount: 1,
      callerCount: 2,
      targetCount: 1,
      workspaceId: "ws-support"
    }
  ]);
});

test("tenant organization model falls back to the active root tenant", () => {
  const result = model({ selectedTenantId: "missing" });

  assert.equal(result.selectedTenantId, rootTenant.id);
  assert.equal(result.selected?.tenant.id, rootTenant.id);
});

test("tenant organization model collapses repeated demo tenant batches to the latest batch", () => {
  const oldRoot = {
    createdAt: "2026-06-09T10:00:00Z",
    id: "tenant-root-ui-approval-old",
    level: 1,
    name: "Permission Request Approval Root",
    status: "active",
    updatedAt: "2026-06-09T10:00:00Z"
  };
  const oldChild = {
    createdAt: oldRoot.createdAt,
    id: "tenant-child-ui-approval-old",
    level: 2,
    name: "Permission Request Approval Team",
    parentTenantId: oldRoot.id,
    status: "active",
    updatedAt: oldRoot.updatedAt
  };
  const oldProject = {
    createdAt: oldRoot.createdAt,
    id: "tenant-grandchild-ui-approval-old",
    level: 3,
    name: "Permission Request Approval Project",
    parentTenantId: oldChild.id,
    status: "active",
    updatedAt: oldRoot.updatedAt
  };
  const latestRoot = {
    createdAt: "2026-06-10T10:00:00Z",
    id: "tenant-root-permission-package-approval-new",
    level: 1,
    name: "Permission Package Approval Root",
    status: "active",
    updatedAt: "2026-06-10T10:00:00Z"
  };
  const latestChild = {
    createdAt: latestRoot.createdAt,
    id: "tenant-child-permission-package-approval-new",
    level: 2,
    name: "Permission Package Approval Team",
    parentTenantId: latestRoot.id,
    status: "active",
    updatedAt: latestRoot.updatedAt
  };
  const latestProject = {
    createdAt: latestRoot.createdAt,
    id: "tenant-grandchild-permission-package-approval-new",
    level: 3,
    name: "Permission Package Approval Project",
    parentTenantId: latestChild.id,
    status: "active",
    updatedAt: latestRoot.updatedAt
  };
  const oldAgent = { ...callerAgent, id: "old-caller", tenantId: oldChild.id };
  const latestAgent = { ...callerAgent, id: "latest-caller", tenantId: latestChild.id };

  const result = model({
    agents: [oldAgent, latestAgent],
    selectedTenantId: oldChild.id,
    tenantEntitlements: [],
    tenants: [oldRoot, oldChild, oldProject, latestRoot, latestChild, latestProject],
    workspaceAssignments: []
  });

  assert.deepEqual(result.flatNodes.map((node) => node.tenant.id), [
    latestRoot.id,
    latestChild.id,
    latestProject.id
  ]);
  assert.equal(result.selectedTenantId, latestRoot.id);
  assert.equal(result.totals.tenantCount, 3);
  assert.equal(result.totals.agentCount, 1);
});
