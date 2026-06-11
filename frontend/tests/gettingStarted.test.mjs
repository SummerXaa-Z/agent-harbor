import assert from "node:assert/strict";
import test from "node:test";

import {
  gettingStartedSteps,
  isSetupComplete
} from "../src/gettingStarted.ts";

const now = "2026-06-11T09:00:00Z";

function baseConsoleData(overrides = {}) {
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
    systemMetrics: [],
    tenantEntitlements: [],
    tenants: [],
    traces: [],
    workspaceAssignments: [],
    ...overrides
  };
}

const activeAgent = {
  channelType: "local",
  createdAt: now,
  credentialVersion: 1,
  id: "agt_support",
  name: "Support Agent",
  status: "active",
  tenantId: "tenant-root",
  updatedAt: now,
  workspaceId: "ws-support"
};

const tenant = {
  createdAt: now,
  id: "tenant-root",
  level: 0,
  name: "Root Tenant",
  status: "active",
  updatedAt: now
};

const capability = {
  action: "read",
  dataDomains: ["support"],
  description: "Search support tickets.",
  discoveredAt: now,
  discoveryStatus: "approved",
  displayName: "Search ticket",
  enforcementMode: "gateway",
  id: "cap_search_ticket",
  key: "search_ticket",
  riskLevel: "low",
  sensitivity: "internal",
  targetId: "agt_support",
  type: "mcp_tool",
  updatedAt: now,
  version: 1
};

const tenantEntitlement = {
  capabilityId: capability.id,
  createdAt: now,
  effect: "allow",
  id: "ent_support",
  priority: 50,
  status: "enabled",
  targetId: activeAgent.id,
  tenantId: tenant.id,
  updatedAt: now
};

const trace = {
  createdAt: now,
  decision: "allowed",
  id: "trace_support",
  routeType: "mcp",
  targetAgentId: activeAgent.id
};

const evidenceRun = {
  agentId: activeAgent.id,
  checks: 6,
  completedAt: now,
  decision: "allowed",
  id: "evidence_support",
  runId: "run_support",
  startedAt: now,
  status: "passed",
  summary: "Ready",
  title: "Permission journey evidence"
};

test("getting started progress reports an empty live system as not configured", () => {
  const data = baseConsoleData();
  const steps = gettingStartedSteps(data);

  assert.deepEqual(steps.map((step) => [step.key, step.done, step.targetHash]), [
    ["connect-api", true, "#getting-started"],
    ["register-agents", false, "#registry"],
    ["discover-capabilities", false, "#capabilities"],
    ["create-grant-chain", false, "#ai-admin"],
    ["run-decision", false, "#traces"],
    ["review-evidence", false, "#evidence"]
  ]);
  assert.equal(isSetupComplete(data), false);
});

test("getting started progress follows the dependency chain for partial setup", () => {
  const data = baseConsoleData({
    agents: [{ ...activeAgent, status: "draft" }],
    capabilities: [capability],
    tenants: [tenant]
  });
  const steps = gettingStartedSteps(data);

  assert.deepEqual(steps.map((step) => [step.key, step.done]), [
    ["connect-api", true],
    ["register-agents", false],
    ["discover-capabilities", true],
    ["create-grant-chain", false],
    ["run-decision", false],
    ["review-evidence", false]
  ]);
  assert.equal(isSetupComplete(data), false);
});

test("sample data keeps API connection incomplete while showing sample progress", () => {
  const data = baseConsoleData({
    agents: [activeAgent],
    capabilities: [capability],
    evidenceRuns: [evidenceRun],
    loadedFromApi: false,
    tenantEntitlements: [tenantEntitlement],
    tenants: [tenant],
    traces: [trace]
  });
  const steps = gettingStartedSteps(data);

  assert.deepEqual(steps.map((step) => [step.key, step.done]), [
    ["connect-api", false],
    ["register-agents", true],
    ["discover-capabilities", true],
    ["create-grant-chain", true],
    ["run-decision", true],
    ["review-evidence", true]
  ]);
  assert.equal(isSetupComplete(data), false);
});

test("configured live systems are setup-complete before runtime and evidence are present", () => {
  const data = baseConsoleData({
    agents: [activeAgent],
    capabilities: [capability],
    tenantEntitlements: [tenantEntitlement],
    tenants: [tenant]
  });
  const steps = gettingStartedSteps(data);

  assert.deepEqual(steps.slice(0, 4).map((step) => step.done), [true, true, true, true]);
  assert.deepEqual(steps.slice(4).map((step) => step.done), [false, false]);
  assert.equal(isSetupComplete(data), true);
});
