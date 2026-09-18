import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  aiAdminRuntimeValidationBlockerMessageKey,
  buildAiAdminRuntimeValidationReadiness,
  countUnclassifiedTargetCapabilities
} from "../src/aiAdminRuntimeValidation.ts";

const appSource = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const workbenchSource = readFileSync(
  new URL("../src/components/AiAdminPermissionWorkbench.tsx", import.meta.url),
  "utf8"
);

const form = {
  callerInstanceId: "agt-caller",
  region: "华东",
  requestText: "给客服助手开通工单处理权限。",
  subjectSelector: "user:support-*",
  targetId: "agt-target",
  templateId: "support-ticket-triage",
  tenantId: "tenant-east",
  workspaceId: "ws-support"
};

const capability = (overrides) => ({
  id: `cap-${overrides.key}`,
  targetId: "agt-target",
  type: "tool",
  key: overrides.key,
  displayName: overrides.key,
  action: overrides.action ?? "read",
  sensitivity: "internal",
  riskLevel: "medium",
  enforcementMode: "enforce",
  discoveryStatus: "approved",
  version: 1,
  discoveredAt: "2026-09-19T00:00:00.000Z",
  updatedAt: "2026-09-19T00:00:00.000Z",
  ...overrides
});

const allowedCapabilities = [
  capability({ key: "search_customer", action: "read", dataDomains: ["support"] }),
  capability({ key: "update_ticket", action: "write", dataDomains: ["support"] })
];
const blockedCapabilities = [
  capability({ key: "export_contracts", action: "export", dataDomains: ["support"] })
];

const baseInput = {
  allowedCapabilities,
  blockedCapabilities,
  form,
  hasApplication: true,
  liveDataAvailable: true,
  runId: "ui-validation-test"
};

test("runtime validation builds a plan from the applied permission change", () => {
  const readiness = buildAiAdminRuntimeValidationReadiness(baseInput);

  assert.deepEqual(readiness.blockers, []);
  assert.equal(readiness.plan.allowedCapabilityKey, "search_customer");
  assert.equal(readiness.plan.blockedCapabilityKey, "export_contracts");
  assert.equal(readiness.plan.callerInstanceId, "agt-caller");
  assert.equal(readiness.plan.targetId, "agt-target");
  assert.equal(readiness.plan.runId, "ui-validation-test");
  assert.equal(readiness.plan.subjectId, "user:support-example");
});

test("runtime validation falls back to the first allowed capability when no read action exists", () => {
  const readiness = buildAiAdminRuntimeValidationReadiness({
    ...baseInput,
    allowedCapabilities: [capability({ key: "update_ticket", action: "write", dataDomains: ["support"] })]
  });

  assert.deepEqual(readiness.blockers, []);
  assert.equal(readiness.plan.allowedCapabilityKey, "update_ticket");
});

test("runtime validation reports every missing prerequisite as a blocker", () => {
  const readiness = buildAiAdminRuntimeValidationReadiness({
    ...baseInput,
    allowedCapabilities: [],
    blockedCapabilities: [],
    form: { ...form, subjectSelector: "" },
    hasApplication: false,
    liveDataAvailable: false
  });

  assert.deepEqual(readiness.blockers, [
    "requires_live_api",
    "requires_application",
    "requires_allowed_capability",
    "requires_blocked_capability",
    "requires_subject"
  ]);
  assert.equal(readiness.plan, null);
});

test("runtime validation ignores capabilities of other targets", () => {
  const readiness = buildAiAdminRuntimeValidationReadiness({
    ...baseInput,
    allowedCapabilities: [capability({ key: "search_other", targetId: "agt-other", dataDomains: ["support"] })]
  });

  assert.deepEqual(readiness.blockers, ["requires_allowed_capability"]);
  assert.equal(readiness.plan, null);
});

test("runtime validation uses a concrete subject selector as the subject", () => {
  const readiness = buildAiAdminRuntimeValidationReadiness({
    ...baseInput,
    form: { ...form, subjectSelector: "user:support-001" }
  });

  assert.equal(readiness.plan.subjectId, "user:support-001");
});

test("runtime validation blockers map to actionable messages", () => {
  assert.equal(aiAdminRuntimeValidationBlockerMessageKey("requires_application"), "message.aiAdminRuntimeValidationRequiresApplication");
  assert.equal(aiAdminRuntimeValidationBlockerMessageKey("requires_allowed_capability"), "message.aiAdminRuntimeValidationNoAllowedCapability");
  assert.equal(aiAdminRuntimeValidationBlockerMessageKey("requires_blocked_capability"), "message.aiAdminRuntimeValidationNoBlockedCapability");
});

test("unclassified capability counting drives the data-domain guidance", () => {
  const capabilities = [
    ...allowedCapabilities,
    blockedCapabilities[0],
    capability({ key: "unclassified_tool", dataDomains: [] }),
    capability({ key: "other_target_tool", targetId: "agt-other", dataDomains: [] })
  ];

  assert.equal(countUnclassifiedTargetCapabilities(capabilities, "agt-target"), 1);
  assert.equal(countUnclassifiedTargetCapabilities(capabilities, ""), 0);
});

test("runtime validation button no longer routes to a scripted demo journey", () => {
  assert.doesNotMatch(appSource, /runAiAdminApprovalJourney/);
  assert.match(appSource, /onRunRuntimeValidation=\{\(\) => void runAiAdminRuntimeValidation\(\)\}/);
  assert.match(appSource, /permissionPackageApplicationDraftInput\(aiAdminApplication, aiAdminForm\)/);
});

test("permission workbench surfaces unclassified capability guidance with a jump action", () => {
  assert.match(workbenchSource, /unclassifiedCapabilityCount > 0 && !application/);
  assert.match(workbenchSource, /text\.permissionUnclassifiedCapabilityHint/);
  assert.match(workbenchSource, /onClick=\{onOpenCapabilityGovernance\}/);
});
