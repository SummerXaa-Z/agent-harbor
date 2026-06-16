import assert from "node:assert/strict";
import test from "node:test";

import {
  managementMutationRefreshFailedMessageKey,
  managementMutationSuccessMessageKey,
  refreshAfterManagementMutation
} from "../src/managementMutationRefresh.ts";

test("post-mutation refresh reports a fresh timestamp when reload succeeds", async () => {
  let refreshCalls = 0;
  const result = await refreshAfterManagementMutation({
    action: "create_agent",
    now: () => new Date("2026-06-17T01:02:03.000Z"),
    onRefresh: async () => {
      refreshCalls += 1;
    }
  });

  assert.equal(refreshCalls, 1);
  assert.deepEqual(result, {
    action: "create_agent",
    ok: true,
    refreshedAt: "2026-06-17T01:02:03.000Z"
  });
});

test("post-mutation refresh failure is reported without throwing", async () => {
  const refreshError = new Error("api unavailable");
  const result = await refreshAfterManagementMutation({
    action: "create_policy",
    onRefresh: async () => {
      throw refreshError;
    }
  });

  assert.equal(result.action, "create_policy");
  assert.equal(result.ok, false);
  assert.equal(result.error, refreshError);
});

test("mutation message keys distinguish success from refresh failure", () => {
  assert.equal(managementMutationSuccessMessageKey("create_agent"), "message.agentCreated");
  assert.equal(managementMutationSuccessMessageKey("create_key"), "message.keyCreated");
  assert.equal(managementMutationSuccessMessageKey("rotate_credential"), "message.credentialRotated");
  assert.equal(managementMutationSuccessMessageKey("create_policy"), "message.policyCreated");

  assert.equal(managementMutationRefreshFailedMessageKey("create_agent"), "message.agentCreatedRefreshFailed");
  assert.equal(managementMutationRefreshFailedMessageKey("create_key"), "message.keyCreatedRefreshFailed");
  assert.equal(managementMutationRefreshFailedMessageKey("rotate_credential"), "message.credentialRotatedRefreshFailed");
  assert.equal(managementMutationRefreshFailedMessageKey("create_policy"), "message.policyCreatedRefreshFailed");
});

