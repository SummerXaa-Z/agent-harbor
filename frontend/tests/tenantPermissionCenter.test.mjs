import assert from "node:assert/strict";
import test from "node:test";

import {
  buildTenantPermissionCenterViewModel,
  tenantPermissionCenterActionTarget,
} from "../src/tenantPermissionCenter.ts";

const now = "2026-06-16T12:00:00Z";
const tenant = { createdAt: now, id: "tenant-child", level: 2, name: "Customer Service", parentTenantId: "tenant-root", status: "active", updatedAt: now };
const root = { createdAt: now, id: "tenant-root", level: 1, name: "Group HQ", status: "active", updatedAt: now };

const center = {
  administrators: [
    { actor: "tenant-admin", displayName: "Tenant Admin", id: "adm-1", role: "tenant_admin", source: "managed", status: "active", tenantId: tenant.id, workspaceId: "ws-support" },
  ],
  capabilities: [
    { capabilityId: "cap-search", capabilityName: "Search tickets", dataScopes: [{ dataDomain: "support", dataset: "tickets", region: "us-east" }], effect: "allow", targetId: "target-ticket", targetName: "Ticket Tool Service", workspaceIds: ["ws-support"] },
    { capabilityId: "cap-export", capabilityName: "Export tickets", dataScopes: [], effect: "deny", targetId: "target-ticket", targetName: "Ticket Tool Service", workspaceIds: ["ws-support"] },
  ],
  generatedAt: now,
  nextActions: [
    { code: "start_permission_change", targetView: "ai-admin" },
    { code: "open_access_profile", targetView: "access" },
    { code: "manage_administrators", targetView: "admin-access" },
  ],
  operatorBoundary: { actor: "platform", canManageAdministrators: true, role: "platform_admin" },
  permissionPackages: [
    { allowedCapabilityCount: 1, blockedCapabilityCount: 1, dataScopes: [{ dataDomain: "support", region: "us-east" }], latestApplicationId: "ppa-1", status: "ready", templateId: "support-ticket-triage", templateName: "Support ticket triage" },
  ],
  scopeTenants: [root, tenant],
  tenant,
  workspaces: [{ assignmentCount: 1, callerCount: 1, targetCount: 1, workspaceId: "ws-support" }],
};

test("tenant permission center presenter summarizes ready tenant governance", () => {
  const vm = buildTenantPermissionCenterViewModel(center, { selectedWorkspaceId: "ws-support" });

  assert.equal(vm.tenantName, "Customer Service");
  assert.equal(vm.tenantPath, "Group HQ / Customer Service");
  assert.equal(vm.status, "ready");
  assert.equal(vm.metric.allowedCapabilities, 1);
  assert.equal(vm.metric.blockedCapabilities, 1);
  assert.equal(vm.metric.administrators, 1);
  assert.deepEqual(vm.capabilitySummaries.map((capability) => capability.capabilityName), ["Search tickets", "Export tickets"]);
  assert.deepEqual(vm.capabilitySummaries[0].dataScopeLabels, ["support / tickets / us-east"]);
  assert.deepEqual(vm.dataScopeLabels, ["support / us-east", "support / tickets / us-east"]);
  assert.deepEqual(vm.primaryActions.map((action) => action.code), ["start_permission_change", "open_access_profile", "manage_administrators"]);
});

test("tenant permission center presenter hides admin management for scoped admins", () => {
  const vm = buildTenantPermissionCenterViewModel({
    ...center,
    nextActions: center.nextActions.filter((action) => action.code !== "manage_administrators"),
    operatorBoundary: { actor: "tenant-admin", canManageAdministrators: false, role: "tenant_admin", tenantId: tenant.id, workspaceId: "ws-support" },
  });

  assert.equal(vm.operatorBoundaryLabel, "tenant_admin / Customer Service / ws-support");
  assert.equal(vm.canManageAdministrators, false);
  assert.equal(tenantPermissionCenterActionTarget(vm.primaryActions, "manage_administrators"), "");
});

test("tenant permission center presenter tolerates nullable array fields from older APIs", () => {
  const vm = buildTenantPermissionCenterViewModel({
    ...center,
    administrators: null,
    nextActions: null,
  });

  assert.equal(vm.metric.administrators, 0);
  assert.deepEqual(vm.primaryActions, []);
});

test("tenant permission center presenter marks empty tenant as setup needed", () => {
  const vm = buildTenantPermissionCenterViewModel({
    ...center,
    administrators: [],
    capabilities: [],
    nextActions: [{ code: "complete_setup", targetView: "getting-started" }],
    permissionPackages: [],
    workspaces: [],
  });

  assert.equal(vm.status, "blocked");
  assert.deepEqual(vm.emptyReasons, ["tenantCenter.empty.noWorkspaces", "tenantCenter.empty.noCapabilities"]);
  assert.equal(tenantPermissionCenterActionTarget(vm.primaryActions, "complete_setup"), "getting-started");
});
