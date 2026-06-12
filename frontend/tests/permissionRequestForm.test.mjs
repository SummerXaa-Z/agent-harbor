import assert from "node:assert/strict";
import test from "node:test";

import { accessSubjectOptions } from "../src/accessSubjects.ts";
import { applyPermissionRequestAccessSubject } from "../src/permissionRequestForm.ts";

const form = {
  callerInstanceId: "agt_support",
  region: "华东",
  requestText: "给客服助手开通工单处理权限。",
  subjectSelector: "user:support-*",
  targetId: "agt_ticket_mcp",
  templateId: "support-ticket-triage",
  tenantId: "tenant-east",
  workspaceId: "ws-support"
};

test("permission request access-object selection derives the subject selector", () => {
  const next = applyPermissionRequestAccessSubject(form, accessSubjectOptions, "role:support-lead");

  assert.equal(next.subjectSelector, "user:support-lead-*");
  assert.equal(next.tenantId, form.tenantId);
  assert.notEqual(next, form);
});

test("permission request access-object selection ignores custom advanced selectors", () => {
  const next = applyPermissionRequestAccessSubject(form, accessSubjectOptions, "custom");

  assert.equal(next.subjectSelector, form.subjectSelector);
  assert.equal(next, form);
});
