import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  capabilityGrantRefreshFailedMessageKey,
  mergeCapabilityGrantChainIntoConsoleData,
  refreshAfterCapabilityGrantMutation
} from "../src/capabilityGrantRefresh.ts";
import { localizedUpstreamErrorMessage } from "../src/localizedMessages.ts";
import { createTranslator } from "../src/i18n.ts";

class UpstreamApiError extends Error {
  code;
  constructor(code, message) {
    super(message);
    this.code = code;
  }
}

const hook = readFileSync(new URL("../src/hooks/useCapabilityGovernanceController.ts", import.meta.url), "utf8");
const i18n = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");

function submitGrantBlock() {
  const start = hook.indexOf("async function submitCapabilityGrantChain(");
  assert.notEqual(start, -1, "submitCapabilityGrantChain not found");
  const next = hook.indexOf("\n  return {", start);
  return hook.slice(start, next);
}

function consoleData(overrides = {}) {
  return {
    accessGrants: [],
    agents: [],
    apiBase: "http://127.0.0.1:9090",
    auditEvents: [],
    capabilities: [],
    capabilitiesLoadedFromApi: true,
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

test("capability grant refresh helper reports reload failures without throwing", async () => {
  const refreshError = new Error("refresh unavailable");
  const result = await refreshAfterCapabilityGrantMutation({
    onRefresh: async () => {
      throw refreshError;
    }
  });

  assert.equal(result.ok, false);
  assert.equal(result.error, refreshError);
  assert.equal(capabilityGrantRefreshFailedMessageKey(), "message.grantChainCreatedRefreshFailed");
});

test("capability grant chain returned from live writes is merged before follow-up refresh", () => {
  const entitlement = {
    capabilityId: "cap_read",
    createdAt: "2026-06-17T01:00:00.000Z",
    dataScopes: ["support"],
    effect: "allow",
    id: "tent-live",
    priority: 50,
    status: "enabled",
    targetId: "agent-mcp",
    tenantId: "tenant-east",
    updatedAt: "2026-06-17T01:00:00.000Z"
  };
  const workspaceAssignment = {
    createdAt: "2026-06-17T01:00:01.000Z",
    dataScopes: ["support"],
    effect: "allow",
    id: "wsa-live",
    status: "enabled",
    tenantId: "tenant-east",
    tenantEntitlementId: entitlement.id,
    updatedAt: "2026-06-17T01:00:01.000Z",
    workspaceId: "ws-support"
  };
  const instanceAssignment = {
    callerInstanceId: "agent-caller",
    createdAt: "2026-06-17T01:00:02.000Z",
    dataScopes: ["support"],
    effect: "allow",
    id: "ia-live",
    status: "enabled",
    subjectSelector: "role:support",
    tenantId: "tenant-east",
    updatedAt: "2026-06-17T01:00:02.000Z",
    workspaceAssignmentId: workspaceAssignment.id,
    workspaceId: "ws-support"
  };
  const merged = mergeCapabilityGrantChainIntoConsoleData(consoleData(), {
    entitlement,
    instanceAssignment,
    workspaceAssignment
  });

  assert.deepEqual(merged.tenantEntitlements, [entitlement]);
  assert.deepEqual(merged.workspaceAssignments, [workspaceAssignment]);
  assert.deepEqual(merged.instanceAssignments, [instanceAssignment]);
  assert.equal(merged.grantsLoadedFromApi, true);
  assert.equal(merged.capabilityAssignmentsLoadedFromApi, true);
});

test("capability grant creation separates write success from follow-up refresh failure", () => {
  const block = submitGrantBlock();

  assert.match(block, /const instanceAssignment = await createInstanceAssignment/);
  assert.match(block, /mergeCapabilityGrantChainIntoConsoleData/);
  assert.match(block, /refreshAfterCapabilityGrantMutation\(\{ onRefresh \}\)/);
  assert.match(block, /capabilityGrantRefreshFailedMessageKey\(\)/);
  assert.doesNotMatch(block, /setMessage\(t\("message\.grantChainCreated"\)\);\s+await onRefresh\(\)/);
  assert.doesNotMatch(block, /setMessage\(localizedErrorMessage\(t, language, error, "error\.createGrantChain"\)\)[\s\S]*await onRefresh\(\)/);
});

test("capability grant refresh failure message is bilingual", () => {
  assert.match(i18n, /"message\.grantChainCreatedRefreshFailed": "Grant chain created, but the grant list could not be refreshed\. Refresh before continuing\."/);
  assert.match(i18n, /"message\.grantChainCreatedRefreshFailed": "授权链已创建，但授权清单未能刷新。继续前请先刷新。"/);
});

test("capability refresh failures surface the upstream classification and cause", () => {
  assert.match(hook, /setMessage\(localizedUpstreamErrorMessageState\(error, "error\.refreshCapabilities"\)\)/);
  assert.doesNotMatch(hook, /setMessage\(localizedErrorMessageState\(error, "error\.refreshCapabilities"\)\)/);
});

test("upstream error message composes localized classification plus technical cause", () => {
  const error = new UpstreamApiError(
    "UPSTREAM_CONNECT_ERROR",
    "upstream connection failed: dial tcp 127.0.0.1:9999: connect: connection refused"
  );
  const zh = createTranslator("zh-CN");
  const en = createTranslator("en");

  assert.equal(
    localizedUpstreamErrorMessage(zh, "zh-CN", error, "error.refreshCapabilities"),
    "无法刷新能力。上游连接失败（upstream connection failed: dial tcp 127.0.0.1:9999: connect: connection refused）"
  );
  assert.equal(
    localizedUpstreamErrorMessage(en, "en", error, "error.refreshCapabilities"),
    error.message
  );
  assert.equal(
    localizedUpstreamErrorMessage(zh, "zh-CN", new Error("boom"), "error.refreshCapabilities"),
    "无法刷新能力。"
  );
  assert.equal(
    localizedUpstreamErrorMessage(
      zh,
      "zh-CN",
      new UpstreamApiError("UPSTREAM_TIMEOUT", "upstream request timed out: context deadline exceeded"),
      "error.refreshCapabilities"
    ),
    "无法刷新能力。上游请求超时（upstream request timed out: context deadline exceeded）"
  );
});
