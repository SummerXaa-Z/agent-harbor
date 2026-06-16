import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const hook = readFileSync(new URL("../src/hooks/useManagementOperations.ts", import.meta.url), "utf8");
const refresh = readFileSync(new URL("../src/managementMutationRefresh.ts", import.meta.url), "utf8");
const i18n = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");

function functionBlock(name, source = hook) {
  const start = source.indexOf(`async function ${name}(`);
  assert.notEqual(start, -1, `${name} not found`);
  const next = source.indexOf("\n  async function ", start + 1);
  return source.slice(start, next === -1 ? undefined : next);
}

test("resource cleanup actions are part of typed post-mutation refresh feedback", () => {
  assert.match(refresh, /"update_agent_status"/);
  assert.match(refresh, /"disable_policy"/);
  assert.match(refresh, /update_agent_status: "message\.agentStatusChangedRefreshFailed"/);
  assert.match(refresh, /disable_policy: "message\.policyDisabledRefreshFailed"/);
});

test("agent status changes separate mutation success from follow-up refresh failure", () => {
  const block = functionBlock("handleAgentStatusChange");

  assert.match(block, /const successMessage = tx\(t, "message\.statusChanged"/);
  assert.match(block, /finishManagementMutation\("update_agent_status", setAgentMessage, successMessage\)/);
  assert.doesNotMatch(block, /setAgentMessage\(successMessage\);\s+await onRefresh\(\)/);
  assert.doesNotMatch(block, /setAgentMessage\(tx\(t, "message\.statusChanged"[\s\S]*await onRefresh\(\)/);
});

test("policy disable separates mutation success from follow-up refresh failure", () => {
  const block = functionBlock("handleDisablePolicy");

  assert.match(block, /finishManagementMutation\("disable_policy", setPolicyMessage, t\("message\.policyDisabled"\)\)/);
  assert.doesNotMatch(block, /setPolicyMessage\(t\("message\.policyDisabled"\)\);\s+await onRefresh\(\)/);
});

test("cleanup refresh failure messages are bilingual", () => {
  assert.match(i18n, /"message\.agentStatusChangedRefreshFailed": "Agent status changed, but resource status could not be refreshed\. Refresh before continuing\."/);
  assert.match(i18n, /"message\.policyDisabledRefreshFailed": "Policy disabled, but resource status could not be refreshed\. Refresh before continuing\."/);
  assert.match(i18n, /"message\.agentStatusChangedRefreshFailed": "Agent 状态已变更，但资源状态未能刷新。继续前请先刷新。"/);
  assert.match(i18n, /"message\.policyDisabledRefreshFailed": "策略已禁用，但资源状态未能刷新。继续前请先刷新。"/);
});
