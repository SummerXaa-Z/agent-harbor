# Resource Action Planner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move Resource Management row-action decision logic into a pure planner while preserving PR #95 behavior.

**Architecture:** Add `resourceLifecycleActionPlanner.ts` with a discriminated action plan. `ConsoleController.tsx` executes plans but no longer chooses same-scope callers, targets, runtime filters, or fallback navigation inline.

**Tech Stack:** React, TypeScript, existing Node test runner, existing console presenter helpers.

---

## File Map

- Create: `frontend/src/resourceLifecycleActionPlanner.ts` — pure action-planning module.
- Create: `frontend/tests/resourceLifecycleActionPlanner.test.mjs` — behavior tests for action plans.
- Modify: `frontend/src/ConsoleController.tsx` — replace inline row-action branching with plan execution.
- Modify: `frontend/src/permissionWorkbenchPresenters.ts` — expose permission-intent formatting for controller injection.
- Modify: `frontend/tests/styleTheme.test.mjs` — guard that resource action decision logic is no longer embedded in the controller.

---

### Task 1: Add Planner Contract Tests

**Files:**
- Create: `frontend/tests/resourceLifecycleActionPlanner.test.mjs`

- [x] **Step 1: Write failing planner tests**

Create `frontend/tests/resourceLifecycleActionPlanner.test.mjs`:

```js
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

  assert.deepEqual(plan, { agentId: "target-a", kind: "open_modal", modal: "rotate_credential" });
});

test("capability blockers navigate with the target preselected", () => {
  const plan = planResourceLifecycleAction({
    agents: [callerA, targetA],
    ...formatters,
    item: item(targetA, "review_capabilities"),
    localCallers: [callerA],
    mcpTargets: [targetA]
  });

  assert.deepEqual(plan, { kind: "capability_prefill", navKey: "capabilities", targetId: "target-a" });
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
```

- [x] **Step 2: Run test and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/resourceLifecycleActionPlanner.test.mjs
```

Expected: FAIL because `resourceLifecycleActionPlanner.ts` does not exist.

---

### Task 2: Implement Planner And Use It In Controller

**Files:**
- Create: `frontend/src/resourceLifecycleActionPlanner.ts`
- Modify: `frontend/src/ConsoleController.tsx`

- [x] **Step 1: Add planner implementation**

Create `frontend/src/resourceLifecycleActionPlanner.ts`:

```ts
import type { NavKey } from "./consoleNavigation";
import type { ResourceLifecycleItem } from "./resourceLifecycle";
import type { Agent, PermissionChangeHandoffContext, TraceFilters } from "./types";

export type ResourceLifecycleModal = "rotate_credential" | "create_policy" | "create_key";

export type ResourceLifecycleActionPlan =
  | { agentId: string; kind: "open_modal"; modal: ResourceLifecycleModal }
  | { kind: "capability_prefill"; navKey: "capabilities"; targetId: string }
  | { context: PermissionChangeHandoffContext; kind: "permission_handoff" }
  | { kind: "runtime_filters"; navKey: "traces"; traceFilters: TraceFilters }
  | { kind: "navigate"; navKey: NavKey };

export interface ResourceLifecycleActionPlanInput {
  agents: Agent[];
  formatEntityName: (name: string) => string;
  formatPermissionIntent: (targetName: string) => string;
  formatTenantName: (tenantId: string) => string;
  formatWorkspaceName: (workspaceId: string) => string;
  item: ResourceLifecycleItem;
  localCallers: Agent[];
  mcpTargets: Agent[];
}

export function planResourceLifecycleAction({
  agents,
  formatEntityName,
  formatPermissionIntent,
  formatTenantName,
  formatWorkspaceName,
  item,
  localCallers,
  mcpTargets
}: ResourceLifecycleActionPlanInput): ResourceLifecycleActionPlan {
  if (item.nextActionKind === "rotate_credential") {
    return { agentId: item.id, kind: "open_modal", modal: "rotate_credential" };
  }
  if (item.nextActionKind === "review_capabilities") {
    return { kind: "capability_prefill", navKey: "capabilities", targetId: item.id };
  }
  if (item.nextActionKind === "start_permission_change") {
    return {
      context: permissionHandoffContext({
        agents,
        formatEntityName,
        formatPermissionIntent,
        formatTenantName,
        formatWorkspaceName,
        item,
        localCallers,
        mcpTargets
      }),
      kind: "permission_handoff"
    };
  }
  if (item.nextActionKind === "review_runtime") {
    return {
      kind: "runtime_filters",
      navKey: "traces",
      traceFilters: item.kind === "caller" ? { callerAgentId: item.id } : { targetAgentId: item.id }
    };
  }
  return { kind: "navigate", navKey: "registry" };
}

function permissionHandoffContext({
  agents,
  formatEntityName,
  formatPermissionIntent,
  formatTenantName,
  formatWorkspaceName,
  item,
  localCallers,
  mcpTargets
}: ResourceLifecycleActionPlanInput): PermissionChangeHandoffContext {
  const resourceAgent = agents.find((agent) => agent.id === item.id);
  const sameScopeCaller = localCallers.find((agent) => sameScope(agent, item));
  const sameScopeTarget = mcpTargets.find((agent) => sameScope(agent, item));
  const caller = item.kind === "caller" ? resourceAgent : sameScopeCaller ?? localCallers[0];
  const target = item.kind === "caller" ? sameScopeTarget ?? mcpTargets[0] : resourceAgent;
  const targetName = target ? formatEntityName(target.name) : item.name;

  return {
    callerInstanceId: caller?.id,
    callerName: caller ? formatEntityName(caller.name) : undefined,
    intentText: formatPermissionIntent(targetName),
    sourceView: "registry",
    targetId: target?.id ?? item.id,
    targetName,
    tenantId: item.tenantId,
    tenantName: formatTenantName(item.tenantId),
    workspaceId: item.workspaceId,
    workspaceName: formatWorkspaceName(item.workspaceId)
  };
}

function sameScope(agent: Agent, item: ResourceLifecycleItem) {
  return agent.tenantId === item.tenantId && agent.workspaceId === item.workspaceId;
}
```

- [x] **Step 2: Replace inline controller decision logic**

In `frontend/src/ConsoleController.tsx` import the planner:

```ts
import {
  planResourceLifecycleAction,
  type ResourceLifecycleModal
} from "./resourceLifecycleActionPlanner";
```

Replace `ResourceActionModal` with:

```ts
type ResourceActionModal = "" | ResourceLifecycleModal;
```

Replace `handleResourceLifecycleAction` with:

```ts
function handleResourceLifecycleAction(item: ResourceLifecycleItem) {
  const plan = planResourceLifecycleAction({
    agents,
    formatEntityName: (name) => permissionEntityDisplayName(name, t),
    formatPermissionIntent: (targetName) => resourcePermissionIntent(targetName, t),
    formatTenantName: (tenantId) => permissionTenantPathLabel(tenantId, tenants, t).primary,
    formatWorkspaceName: (workspaceId) => permissionWorkspaceDisplayName(workspaceId, agents, t),
    item,
    localCallers,
    mcpTargets
  });
  if (plan.kind === "open_modal") {
    management.setRotateForm({
      ...management.rotateForm,
      agentId: plan.agentId
    });
    openResourceActionModal(plan.modal);
    return;
  }
  if (plan.kind === "capability_prefill") {
    capabilityGovernance.setForm({
      ...capabilityGovernance.form,
      targetId: plan.targetId
    });
    userSelectedNavRef.current = true;
    setActiveNav(plan.navKey);
    return;
  }
  if (plan.kind === "permission_handoff") {
    openTenantPermissionChange(plan.context);
    return;
  }
  if (plan.kind === "runtime_filters") {
    setTraceFilters((current) => ({ ...current, ...plan.traceFilters }));
    userSelectedNavRef.current = true;
    setActiveNav(plan.navKey);
    return;
  }
  userSelectedNavRef.current = true;
  setActiveNav(plan.navKey);
}
```

- [x] **Step 3: Run planner test and confirm GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/resourceLifecycleActionPlanner.test.mjs
```

Expected: PASS.

---

### Task 3: Strengthen Structure Guards And Verify

**Files:**
- Modify: `frontend/tests/styleTheme.test.mjs`

- [x] **Step 1: Add controller anti-regression guard**

In `styleTheme.test.mjs`, add:

```js
const resourceLifecycleActionPlanner = readExistingFile(new URL("../src/resourceLifecycleActionPlanner.ts", import.meta.url));
```

Inside `agent tools workspace balances registry layout and hides inactive empty-state actions`, add:

```js
assert.match(app, /planResourceLifecycleAction/);
assert.match(resourceLifecycleActionPlanner, /export function planResourceLifecycleAction/);
assert.doesNotMatch(app, /sameScopeCaller|sameScopeTarget|resource\\.permissionIntent/);
```

- [x] **Step 2: Run focused tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/resourceLifecycleActionPlanner.test.mjs tests/styleTheme.test.mjs
```

Expected: PASS.

- [x] **Step 3: Run release gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
make check
make release-check
```

Expected: PASS.

- [x] **Step 4: Commit, push, and open PR**

Run:

```bash
git add docs/superpowers/plans/2026-06-17-resource-action-planner.md docs/superpowers/specs/2026-06-17-resource-action-planner-design.md frontend/src frontend/tests
git commit -m "Extract resource lifecycle action planner"
git push -u origin codex/resource-action-planner
gh pr create --base main --head codex/resource-action-planner --title "Extract resource lifecycle action planner" --body "..."
```
