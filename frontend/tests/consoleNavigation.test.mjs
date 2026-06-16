import assert from "node:assert/strict";
import test from "node:test";

import {
  defaultNavKey,
  navHashFor,
  navKeyFromHash,
  navGroups,
  navItems,
  viewForNav
} from "../src/consoleNavigation.ts";
import { resolveDefaultNavKey } from "../src/gettingStarted.ts";

const now = "2026-06-11T10:00:00Z";

function consoleData(overrides = {}) {
  return {
    accessGrants: [],
    agents: [],
    apiBase: "http://127.0.0.1:9090",
    auditEvents: [],
    capabilities: [],
    capabilitiesLoadedFromApi: true,
    capabilityAssignmentsLoadedFromApi: true,
    channels: [],
    evidenceRuns: [],
    grantsLoadedFromApi: true,
    instanceAssignments: [],
    loadedFromApi: true,
    providers: [],
    routePolicies: [],
    routePoliciesLoadedFromApi: true,
    setupLoadedFromApi: true,
    systemMetrics: [],
    tenantEntitlements: [],
    tenants: [],
    traces: [],
    workspaceAssignments: [],
    ...overrides
  };
}

const tenant = { createdAt: now, id: "tenant-root", level: 0, name: "Root", status: "active", updatedAt: now };
const agent = {
  channelType: "local",
  createdAt: now,
  credentialVersion: 1,
  id: "agt-support",
  name: "Support Agent",
  status: "active",
  tenantId: tenant.id,
  updatedAt: now,
  workspaceId: "ws-support"
};
const capability = {
  action: "read",
  dataDomains: ["support"],
  description: "Search tickets.",
  discoveredAt: now,
  discoveryStatus: "approved",
  displayName: "Search ticket",
  enforcementMode: "gateway",
  id: "cap-search",
  key: "search_ticket",
  riskLevel: "low",
  sensitivity: "internal",
  targetId: agent.id,
  type: "mcp_tool",
  updatedAt: now,
  version: 1
};
const entitlement = {
  capabilityId: capability.id,
  createdAt: now,
  effect: "allow",
  id: "ent-support",
  priority: 50,
  status: "enabled",
  targetId: agent.id,
  tenantId: tenant.id,
  updatedAt: now
};

test("every primary navigation item resolves to a distinct workspace", () => {
  const views = navItems.map((item) => viewForNav(item.key));
  const viewKeys = views.map((view) => view.key);

  assert.deepEqual(viewKeys, [
    "getting-started",
    "ask",
    "ai-admin",
    "access",
    "traces",
    "evidence",
    "cockpit",
    "admin-access",
    "tenants",
    "registry",
    "capabilities",
    "policies",
    "routes",
  ]);
  assert.equal(new Set(views.map((view) => view.primaryPanelKey)).size, views.length);
});

test("navigation is grouped by user task", () => {
  assert.deepEqual(navGroups.map((group) => group.key), ["onboarding", "primary", "audit", "configuration"]);
  assert.deepEqual(navItems.map((item) => item.groupKey), [
    "onboarding",
    "primary",
    "primary",
    "primary",
    "audit",
    "audit",
    "audit",
    "configuration",
    "configuration",
    "configuration",
    "configuration",
    "configuration",
    "configuration"
  ]);
  assert.ok(navItems.every((item) => item.detailKey.startsWith("navDetail.")));
});

test("default navigation opens answer-first access query", () => {
  assert.equal(defaultNavKey, "getting-started");
  assert.equal(viewForNav("unknown").key, defaultNavKey);
});

test("go-live evidence navigation points to the acceptance workflow", () => {
  assert.equal(viewForNav("evidence").primaryPanelKey, "goLiveAcceptance");
});

test("resource management is the single entry for Agent and MCP lifecycle", () => {
  assert.equal(viewForNav("registry").primaryPanelKey, "resourceLifecycle");
});

test("production journey acceptance keeps existing primary routes", () => {
  assert.equal(viewForNav("getting-started").primaryPanelKey, "gettingStarted");
  assert.equal(viewForNav("ask").primaryPanelKey, "askAccess");
  assert.equal(viewForNav("registry").primaryPanelKey, "resourceLifecycle");
  assert.equal(viewForNav("ai-admin").primaryPanelKey, "aiAdminPermissionWorkbench");
  assert.equal(viewForNav("evidence").primaryPanelKey, "goLiveAcceptance");
});

test("navigation hash preserves the current workspace across reloads", () => {
  assert.equal(navHashFor("evidence"), "#evidence");
  assert.equal(navHashFor("tenants"), "#tenants");
  assert.equal(navHashFor("getting-started"), "#getting-started");
  assert.equal(navKeyFromHash("#evidence"), "evidence");
  assert.equal(navKeyFromHash("#tenants"), "tenants");
  assert.equal(navKeyFromHash("#getting-started"), "getting-started");
  assert.equal(navKeyFromHash("evidence"), "evidence");
  assert.equal(navKeyFromHash("#unknown"), null);
});

test("default navigation is resolved from setup readiness", () => {
  assert.equal(resolveDefaultNavKey(consoleData()), "getting-started");
  assert.equal(resolveDefaultNavKey(consoleData({ loadedFromApi: false })), "getting-started");
  assert.equal(resolveDefaultNavKey(consoleData({
    agents: [agent],
    capabilities: [capability],
    tenantEntitlements: [entitlement],
    tenants: [tenant]
  })), "ask");
});
