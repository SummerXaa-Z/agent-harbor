import assert from "node:assert/strict";
import test from "node:test";

import { planResourceLifecycleAction } from "../src/resourceLifecycleActionPlanner.ts";

const tenants = [
  { id: "tenant-a", level: 1, name: "客户服务中心", status: "active", createdAt: "", updatedAt: "" }
];
const callerA = agent("caller-a", "tenant-a", "ws-a", "客服助手", "local");
const callerB = agent("caller-b", "tenant-b", "ws-b", "其他助手", "local");
const targetA = agent("target-a", "tenant-a", "ws-a", "工单工具服务", "mcp");
const targetB = agent("target-b", "tenant-b", "ws-b", "其他工具服务", "mcp");
const formatters = {
  formatEntityName: (name) => name,
  formatPermissionIntent: (targetName) => `为 ${targetName} 创建授权。`,
  formatTenantName: (tenantId) => tenants.find((tenant) => tenant.id === tenantId)?.name ?? tenantId,
  formatWorkspaceName: (workspaceId) => workspaceId
};

test("credential blockers open the rotate credential modal", () => {
  const plan = planResourceLifecycleAction({
    agents: [callerA, targetA],
    ...formatters,
    item: item(targetA, "rotate_credential"),
    localCallers: [callerA],
    mcpTargets: [targetA]
  });

  assert.deepEqual(plan, {
    agentId: "target-a",
    context: {
      resourceKindKey: "resource.kind.mcpTarget",
      resourceName: "工单工具服务",
      tenantName: "客户服务中心",
      workspaceName: "ws-a"
    },
    kind: "open_modal",
    modal: "rotate_credential"
  });
});

test("key creation from a caller opens with the caller preselected", () => {
  const plan = planResourceLifecycleAction({
    agents: [callerA, targetA],
    ...formatters,
    item: { ...item(callerA, "create_key"), kind: "caller", kindKey: "resource.kind.caller" },
    localCallers: [callerA],
    mcpTargets: [targetA]
  });

  assert.equal(plan.kind, "open_modal");
  assert.equal(plan.modal, "create_key");
  assert.equal(plan.agentId, "caller-a");
  assert.equal(plan.context.resourceName, "客服助手");
  assert.equal(plan.context.resourceKindKey, "resource.kind.caller");
});

test("policy creation preselects same-scope caller and target", () => {
  const plan = planResourceLifecycleAction({
    agents: [callerB, callerA, targetA],
    ...formatters,
    item: item(targetA, "create_policy"),
    localCallers: [callerB, callerA],
    mcpTargets: [targetA]
  });

  assert.equal(plan.kind, "open_modal");
  assert.equal(plan.modal, "create_policy");
  assert.equal(plan.callerAgentId, "caller-a");
  assert.equal(plan.targetAgentId, "target-a");
  assert.equal(plan.context.resourceName, "工单工具服务");
});

test("capability blockers navigate with the target preselected", () => {
  const plan = planResourceLifecycleAction({
    agents: [callerA, targetA],
    ...formatters,
    item: item(targetA, "review_capabilities"),
    localCallers: [callerA],
    mcpTargets: [targetA]
  });

  assert.deepEqual(plan, {
    context: {
      sourceView: "registry",
      targetId: "target-a",
      targetName: "工单工具服务",
      tenantId: "tenant-a",
      tenantName: "客户服务中心",
      workspaceId: "ws-a",
      workspaceName: "ws-a"
    },
    kind: "capability_prefill",
    navKey: "capabilities",
    targetId: "target-a"
  });
});

test("permission blockers for a target choose the same-scope caller", () => {
  const plan = planResourceLifecycleAction({
    agents: [callerB, callerA, targetA],
    ...formatters,
    item: item(targetA, "start_permission_change"),
    localCallers: [callerB, callerA],
    mcpTargets: [targetA]
  });

  assert.equal(plan.kind, "permission_handoff");
  assert.equal(plan.context.sourceView, "registry");
  assert.equal(plan.context.callerInstanceId, "caller-a");
  assert.equal(plan.context.targetId, "target-a");
  assert.equal(plan.context.intentText, "为 工单工具服务 创建授权。");
});

test("permission blockers for a caller choose the same-scope target", () => {
  const plan = planResourceLifecycleAction({
    agents: [callerA, targetB, targetA],
    ...formatters,
    item: { ...item(callerA, "start_permission_change"), kind: "caller", kindKey: "resource.kind.caller" },
    localCallers: [callerA],
    mcpTargets: [targetB, targetA]
  });

  assert.equal(plan.kind, "permission_handoff");
  assert.equal(plan.context.callerInstanceId, "caller-a");
  assert.equal(plan.context.targetId, "target-a");
});

test("permission blockers never prefill a cross-scope caller", () => {
  const plan = planResourceLifecycleAction({
    agents: [callerB, targetA],
    ...formatters,
    item: item(targetA, "start_permission_change"),
    localCallers: [callerB],
    mcpTargets: [targetA]
  });

  assert.equal(plan.kind, "permission_handoff");
  assert.equal(plan.context.callerInstanceId, undefined);
  assert.equal(plan.context.callerName, undefined);
  assert.equal(plan.context.targetId, "target-a");
  assert.equal(plan.context.targetName, "工单工具服务");
});

test("permission blockers never prefill a cross-scope target", () => {
  const plan = planResourceLifecycleAction({
    agents: [callerA, targetB],
    ...formatters,
    item: { ...item(callerA, "start_permission_change"), kind: "caller", kindKey: "resource.kind.caller" },
    localCallers: [callerA],
    mcpTargets: [targetB]
  });

  assert.equal(plan.kind, "permission_handoff");
  assert.equal(plan.context.callerInstanceId, "caller-a");
  assert.equal(plan.context.callerName, "客服助手");
  assert.equal(plan.context.targetId, undefined);
  assert.equal(plan.context.targetName, undefined);
  assert.equal(plan.context.intentText, undefined);
});

test("runtime plans filter caller rows by caller and target rows by target", () => {
  const callerPlan = planResourceLifecycleAction({
    agents: [callerA, targetA],
    ...formatters,
    item: { ...item(callerA, "review_runtime"), kind: "caller", kindKey: "resource.kind.caller" },
    localCallers: [callerA],
    mcpTargets: [targetA]
  });
  const targetPlan = planResourceLifecycleAction({
    agents: [callerA, targetA],
    ...formatters,
    item: item(targetA, "review_runtime"),
    localCallers: [callerA],
    mcpTargets: [targetA]
  });

  assert.deepEqual(callerPlan, { kind: "runtime_filters", navKey: "traces", traceFilters: { callerAgentId: "caller-a" } });
  assert.deepEqual(targetPlan, { kind: "runtime_filters", navKey: "traces", traceFilters: { targetAgentId: "target-a" } });
});

test("disabled resources navigate back to resource management", () => {
  const plan = planResourceLifecycleAction({
    agents: [callerA, targetA],
    ...formatters,
    item: item(targetA, "review_resource"),
    localCallers: [callerA],
    mcpTargets: [targetA]
  });

  assert.deepEqual(plan, { kind: "navigate", navKey: "registry" });
});

function agent(id, tenantId, workspaceId, name, channelType) {
  return {
    channelConfig: {},
    channelType,
    createdAt: "",
    credentialVersion: 1,
    id,
    name,
    status: "active",
    tenantId,
    updatedAt: "",
    workspaceId
  };
}

function item(agentRow, nextActionKind) {
  return {
    approvedCapabilityCount: 0,
    capabilityCount: 0,
    credentialVersion: agentRow.credentialVersion,
    detailKey: "resource.detail.needsApproval",
    grantCount: 0,
    id: agentRow.id,
    kind: "mcp_target",
    kindKey: "resource.kind.mcpTarget",
    name: agentRow.name,
    nextActionHash: "#registry",
    nextActionKey: "resource.nextAction.registry",
    nextActionKind,
    runtimeDecisionCount: 0,
    status: "needs_approval",
    statusKey: "resource.status.needsApproval",
    tenantId: agentRow.tenantId,
    workspaceId: agentRow.workspaceId
  };
}
