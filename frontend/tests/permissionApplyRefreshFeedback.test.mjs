import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  mergePermissionApplyResultIntoConsoleData,
  permissionApplyRefreshFailedMessageKey,
  refreshAfterPermissionApply
} from "../src/permissionApplyRefresh.ts";

const app = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const i18n = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");

function functionBlock(name) {
  const start = app.indexOf(`async function ${name}(`);
  assert.notEqual(start, -1, `${name} not found`);
  const next = app.indexOf("\n  async function ", start + 1);
  return app.slice(start, next === -1 ? undefined : next);
}

function consoleData(overrides = {}) {
  return {
    accessGrants: [],
    agents: [],
    apiBase: "http://127.0.0.1:9090",
    auditEvents: [],
    capabilities: [],
    capabilitiesLoadedFromApi: false,
    capabilityAssignmentsLoadedFromApi: false,
    channels: [],
    evidenceRuns: [],
    grantsLoadedFromApi: false,
    instanceAssignments: [],
    loadedFromApi: true,
    providers: [],
    routePolicies: [],
    routePoliciesLoadedFromApi: true,
    setupLoadedFromApi: false,
    systemMetrics: [],
    tenantEntitlements: [],
    tenants: [],
    traces: [],
    workspaceAssignments: [],
    ...overrides
  };
}

test("permission apply refresh helper reports post-apply refresh failures without throwing", async () => {
  const refreshError = new Error("profile unavailable");
  const result = await refreshAfterPermissionApply({
    onRefresh: async () => {
      throw refreshError;
    }
  });

  assert.equal(result.ok, false);
  assert.equal(result.error, refreshError);
  assert.equal(permissionApplyRefreshFailedMessageKey(), "message.permissionPackageAppliedRefreshFailed");
});

test("permission apply result is merged into console data before follow-up refresh", () => {
  const capability = {
    action: "tools/call",
    dataScopes: ["support"],
    discoveredAt: "2026-06-17T02:00:00.000Z",
    discoveryStatus: "approved",
    displayName: "Export contract",
    id: "cap-export",
    key: "export_contracts",
    riskLevel: "high",
    sensitivity: "confidential",
    summary: "Export contract bundles",
    targetId: "agent-mcp",
    updatedAt: "2026-06-17T02:00:00.000Z",
    version: 1
  };
  const entitlement = {
    capabilityId: capability.id,
    createdAt: "2026-06-17T02:00:01.000Z",
    dataScopes: ["support"],
    effect: "allow",
    id: "tent-apply",
    priority: 50,
    status: "enabled",
    targetId: capability.targetId,
    tenantId: "tenant-east",
    updatedAt: "2026-06-17T02:00:01.000Z"
  };
  const workspaceAssignment = {
    createdAt: "2026-06-17T02:00:02.000Z",
    dataScopes: ["support"],
    effect: "allow",
    id: "wsa-apply",
    status: "enabled",
    tenantId: "tenant-east",
    tenantEntitlementId: entitlement.id,
    updatedAt: "2026-06-17T02:00:02.000Z",
    workspaceId: "ws-support"
  };
  const instanceAssignment = {
    callerInstanceId: "agent-caller",
    createdAt: "2026-06-17T02:00:03.000Z",
    dataScopes: ["support"],
    effect: "allow",
    id: "ia-apply",
    status: "enabled",
    subjectSelector: "role:support",
    tenantId: "tenant-east",
    updatedAt: "2026-06-17T02:00:03.000Z",
    workspaceAssignmentId: workspaceAssignment.id,
    workspaceId: "ws-support"
  };
  const merged = mergePermissionApplyResultIntoConsoleData(consoleData(), {
    draft: { allowedCapabilities: [capability] },
    instanceAssignments: [instanceAssignment],
    tenantEntitlements: [entitlement],
    workspaceAssignments: [workspaceAssignment]
  });

  assert.deepEqual(merged.capabilities, [capability]);
  assert.deepEqual(merged.tenantEntitlements, [entitlement]);
  assert.deepEqual(merged.workspaceAssignments, [workspaceAssignment]);
  assert.deepEqual(merged.instanceAssignments, [instanceAssignment]);
  assert.equal(merged.capabilitiesLoadedFromApi, true);
  assert.equal(merged.capabilityAssignmentsLoadedFromApi, true);
  assert.equal(merged.grantsLoadedFromApi, true);
});

test("permission apply separates apply success from follow-up data/profile refresh failure", () => {
  const block = functionBlock("applyAiAdminPermissionPackage");

  assert.match(block, /let appliedResult: PermissionPackageApplyResult \| null = null/);
  assert.match(block, /appliedResult = applied/);
  assert.match(block, /setData\(\(current\) =>[\s\S]*mergePermissionApplyResultIntoConsoleData/);
  assert.match(block, /refreshAfterPermissionApply/);
  assert.match(block, /permissionApplyRefreshFailedMessageKey\(\)/);
  assert.match(block, /setAiAdminApplication\(application\)/);
  assert.doesNotMatch(block, /setAiAdminMessage\(localizedErrorMessageState\(error, "error\.applyPermissionPackage"\)\)[\s\S]*loadConsoleData\(adminKey, traceFilters\)/);
});

test("permission apply refresh failure message is bilingual", () => {
  assert.match(i18n, /"message\.permissionPackageAppliedRefreshFailed": "Permissions applied to \{count\} allowed capabilities, but the console could not refresh the tenant profile\. Refresh before continuing\."/);
  assert.match(i18n, /"message\.permissionPackageAppliedRefreshFailed": "权限已应用到 \{count\} 项允许能力，但控制台未能刷新租户访问画像。继续前请先刷新。"/);
});
