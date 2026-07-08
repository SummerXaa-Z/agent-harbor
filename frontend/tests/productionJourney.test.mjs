import assert from "node:assert/strict";
import test from "node:test";

import {
  deriveProductionJourney,
  productionJourneyStages
} from "../src/productionJourney.ts";

const now = "2026-06-15T10:00:00Z";

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

const tenant = { createdAt: now, id: "tenant-root", level: 0, name: "Customer Service", status: "active", updatedAt: now };
const caller = {
  channelType: "local",
  createdAt: now,
  credentialVersion: 1,
  id: "agent-caller",
  name: "Support Assistant",
  status: "active",
  tenantId: tenant.id,
  updatedAt: now,
  workspaceId: "ws-support"
};
const target = { ...caller, channelType: "mcp", id: "agent-mcp", name: "Ticket Tool Service" };
const capability = {
  action: "read",
  dataDomains: ["support"],
  description: "Search tickets.",
  discoveredAt: now,
  discoveryStatus: "approved",
  displayName: "Search tickets",
  enforcementMode: "gateway",
  id: "cap-search",
  key: "search_ticket",
  riskLevel: "low",
  sensitivity: "internal",
  targetId: target.id,
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
  targetId: target.id,
  tenantId: tenant.id,
  updatedAt: now
};

function configuredData(overrides = {}) {
  return consoleData({
    agents: [caller, target],
    capabilities: [capability],
    tenantEntitlements: [entitlement],
    tenants: [tenant],
    ...overrides
  });
}

test("production journey exposes a stable stage order", () => {
  assert.deepEqual(productionJourneyStages.map((stage) => stage.key), [
    "setup",
    "resources",
    "access_query",
    "permission_change",
    "go_live_status",
    "handoff"
  ]);
});

test("empty live systems start at setup and point to getting started", () => {
  const journey = deriveProductionJourney({ data: consoleData(), activeNav: "getting-started" });

  assert.equal(journey.state, "empty");
  assert.equal(journey.currentStageKey, "setup");
  assert.equal(journey.nextActionHash, "#getting-started");
  assert.equal(journey.nextActionKey, "productionJourney.next.setup");
  assert.deepEqual(journey.completedStageKeys, []);
});

test("configured systems default to access query before a decision exists", () => {
  const journey = deriveProductionJourney({ data: configuredData(), activeNav: "ask" });

  assert.equal(journey.state, "configured");
  assert.equal(journey.currentStageKey, "access_query");
  assert.equal(journey.nextActionHash, "#ask");
  assert.equal(journey.nextActionKey, "productionJourney.next.ask");
  assert.deepEqual(journey.completedStageKeys, ["setup", "resources"]);
});

test("denied access query points to permission change without marking it complete", () => {
  const journey = deriveProductionJourney({
    accessOutcome: "denied",
    data: configuredData(),
    activeNav: "ask"
  });

  assert.equal(journey.state, "denied");
  assert.equal(journey.currentStageKey, "access_query");
  assert.equal(journey.nextActionHash, "#ai-admin");
  assert.equal(journey.nextActionKey, "productionJourney.next.fixDenied");
  assert.deepEqual(journey.completedStageKeys, ["setup", "resources"]);
});

test("permission change handoff marks the journey as in change", () => {
  const journey = deriveProductionJourney({
    data: configuredData(),
    hasPermissionChangeContext: true,
    activeNav: "ai-admin"
  });

  assert.equal(journey.state, "in_change");
  assert.equal(journey.currentStageKey, "permission_change");
  assert.equal(journey.nextActionHash, "#ai-admin");
  assert.equal(journey.nextActionKey, "productionJourney.next.completePermissionChange");
});

test("ready permission changes point to go-live status and handoff", () => {
  const journey = deriveProductionJourney({
    data: configuredData({ traces: [{ createdAt: now, decision: "allowed", id: "trace-1", routeType: "mcp", targetAgentId: target.id }] }),
    permissionReady: true,
    activeNav: "ai-admin"
  });

  assert.equal(journey.state, "ready");
  assert.equal(journey.currentStageKey, "go_live_status");
  assert.equal(journey.nextActionHash, "#go-live");
  assert.equal(journey.nextActionKey, "productionJourney.next.confirmGoLive");
  assert.deepEqual(journey.completedStageKeys, ["setup", "resources", "access_query", "permission_change"]);
});

test("blocked permission changes stay on the permission change workspace", () => {
  const journey = deriveProductionJourney({
    data: configuredData(),
    permissionBlocked: true,
    activeNav: "go-live"
  });

  assert.equal(journey.state, "blocked");
  assert.equal(journey.currentStageKey, "permission_change");
  assert.equal(journey.nextActionHash, "#ai-admin");
  assert.equal(journey.nextActionKey, "productionJourney.next.resolveBlocker");
});
