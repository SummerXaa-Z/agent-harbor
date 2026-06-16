import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  mergeManagementAgentIntoConsoleData,
  mergeManagementRoutePolicyIntoConsoleData
} from "../src/managementLocalData.ts";

const hook = readFileSync(new URL("../src/hooks/useManagementOperations.ts", import.meta.url), "utf8");

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
    routePoliciesLoadedFromApi: false,
    setupLoadedFromApi: true,
    systemMetrics: [],
    tenantEntitlements: [],
    tenants: [],
    traces: [],
    workspaceAssignments: [],
    ...overrides
  };
}

function agent(overrides = {}) {
  return {
    channelType: "local",
    createdAt: "2026-06-17T08:00:00.000Z",
    credentialVersion: 1,
    id: "agent-support",
    name: "Support Agent",
    status: "active",
    tenantId: "tenant-support",
    updatedAt: "2026-06-17T08:00:00.000Z",
    workspaceId: "ws-support",
    ...overrides
  };
}

function policy(overrides = {}) {
  return {
    callerAgentId: "agent-caller",
    createdAt: "2026-06-17T08:10:00.000Z",
    effect: "allow",
    id: "policy-tools-call",
    priority: 100,
    routeKey: "tools/call",
    routeType: "mcp",
    status: "enabled",
    targetAgentId: "agent-target",
    updatedAt: "2026-06-17T08:10:00.000Z",
    ...overrides
  };
}

function functionBlock(name) {
  const start = hook.indexOf(`async function ${name}(`);
  assert.notEqual(start, -1, `${name} not found`);
  const next = hook.indexOf("\n  async function ", start + 1);
  return hook.slice(start, next === -1 ? undefined : next);
}

test("management local data helpers upsert returned agents and route policies", () => {
  const createdAgent = agent();
  const updatedAgent = agent({ credentialVersion: 2, status: "disabled", updatedAt: "2026-06-17T09:00:00.000Z" });
  const createdPolicy = policy();
  const disabledPolicy = policy({ status: "disabled", updatedAt: "2026-06-17T09:10:00.000Z" });

  const withAgent = mergeManagementAgentIntoConsoleData(consoleData(), createdAgent);
  assert.deepEqual(withAgent.agents, [createdAgent]);

  const withUpdatedAgent = mergeManagementAgentIntoConsoleData(withAgent, updatedAgent);
  assert.deepEqual(withUpdatedAgent.agents, [updatedAgent]);

  const withPolicy = mergeManagementRoutePolicyIntoConsoleData(consoleData(), createdPolicy);
  assert.deepEqual(withPolicy.routePolicies, [createdPolicy]);
  assert.equal(withPolicy.routePoliciesLoadedFromApi, true);

  const withDisabledPolicy = mergeManagementRoutePolicyIntoConsoleData(withPolicy, disabledPolicy);
  assert.deepEqual(withDisabledPolicy.routePolicies, [disabledPolicy]);
  assert.equal(withDisabledPolicy.routePoliciesLoadedFromApi, true);
});

test("management writes patch local console data before follow-up refresh", () => {
  const submitAgent = functionBlock("submitAgent");
  const statusChange = functionBlock("handleAgentStatusChange");
  const rotateCredential = functionBlock("submitCredentialRotation");
  const submitPolicy = functionBlock("submitRoutePolicy");
  const disablePolicy = functionBlock("handleDisablePolicy");

  assert.match(hook, /onDataPatch\?: \(updater: \(current: ConsoleData\) => ConsoleData\) => void/);
  assert.match(hook, /mergeManagementAgentIntoConsoleData/);
  assert.match(hook, /mergeManagementRoutePolicyIntoConsoleData/);

  assert.match(submitAgent, /const created = await createAgent/);
  assert.match(submitAgent, /patchConsoleData\(\(current\) => mergeManagementAgentIntoConsoleData\(current, created\)\)/);
  assert.ok(
    submitAgent.indexOf("patchConsoleData((current) => mergeManagementAgentIntoConsoleData(current, created))") <
      submitAgent.indexOf('finishManagementMutation("create_agent"'),
    "created Agent should be visible before refresh starts"
  );

  assert.match(statusChange, /const updated = status === "disabled"/);
  assert.match(statusChange, /patchConsoleData\(\(current\) => mergeManagementAgentIntoConsoleData\(current, updated\)\)/);
  assert.ok(
    statusChange.indexOf("patchConsoleData((current) => mergeManagementAgentIntoConsoleData(current, updated))") <
      statusChange.indexOf('finishManagementMutation("update_agent_status"'),
    "status change should be visible before refresh starts"
  );

  assert.match(rotateCredential, /const updated = await rotateAgentCredentials/);
  assert.match(rotateCredential, /patchConsoleData\(\(current\) => mergeManagementAgentIntoConsoleData\(current, updated\)\)/);
  assert.ok(
    rotateCredential.indexOf("patchConsoleData((current) => mergeManagementAgentIntoConsoleData(current, updated))") <
      rotateCredential.indexOf('finishManagementMutation("rotate_credential"'),
    "credential rotation should be visible before refresh starts"
  );

  assert.match(submitPolicy, /const created = await createRoutePolicy/);
  assert.match(submitPolicy, /patchConsoleData\(\(current\) => mergeManagementRoutePolicyIntoConsoleData\(current, created\)\)/);
  assert.ok(
    submitPolicy.indexOf("patchConsoleData((current) => mergeManagementRoutePolicyIntoConsoleData(current, created))") <
      submitPolicy.indexOf('finishManagementMutation("create_policy"'),
    "created policy should be visible before refresh starts"
  );

  assert.match(disablePolicy, /const disabled = await disableRoutePolicy/);
  assert.match(disablePolicy, /patchConsoleData\(\(current\) => mergeManagementRoutePolicyIntoConsoleData\(current, disabled\)\)/);
  assert.ok(
    disablePolicy.indexOf("patchConsoleData((current) => mergeManagementRoutePolicyIntoConsoleData(current, disabled))") <
      disablePolicy.indexOf('finishManagementMutation("disable_policy"'),
    "disabled policy should be visible before refresh starts"
  );
});
