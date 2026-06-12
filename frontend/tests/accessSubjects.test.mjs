import assert from "node:assert/strict";
import test from "node:test";

import {
  accessSubjectOptionForId,
  accessSubjectOptionForSelector,
  accessSubjectOptions,
  accessSubjectsForWorkspace,
  normalizeAccessSubjectOptions
} from "../src/accessSubjects.ts";

test("access subject catalog defaults role selection to compatible selectors", () => {
  const supportAgent = accessSubjectOptionForId("role:support-agent");

  assert.equal(supportAgent?.kind, "role");
  assert.equal(supportAgent?.subjectSelector, "user:support-*");
  assert.equal(accessSubjectOptions[0].id, "role:support-agent");
});

test("access subject catalog can resolve selectors back to business choices", () => {
  assert.equal(accessSubjectOptionForSelector("user:support-*").id, "role:support-agent");
  assert.equal(accessSubjectOptionForSelector("user:support-001").id, "member:support-001");
});

test("access subject catalog keeps custom selectors in advanced mode", () => {
  const option = accessSubjectOptionForSelector("user:finance-*");

  assert.equal(option.id, "custom");
  assert.equal(option.kind, "custom");
  assert.equal(option.subjectSelector, "user:finance-*");
});

test("access subject catalog normalizes server options with a local fallback", () => {
  const serverOptions = normalizeAccessSubjectOptions([{
    detailKey: "accessSubject.supportAgent.detail",
    id: "role:support-agent",
    kind: "role",
    labelKey: "accessSubject.supportAgent.name",
    workspaceId: "ws-support",
    subjectSelector: "user:support-*"
  }]);

  assert.equal(serverOptions.length, 1);
  assert.equal(serverOptions[0].id, "role:support-agent");
  assert.equal(serverOptions[0].workspaceId, "ws-support");
  assert.equal(normalizeAccessSubjectOptions([])[0].id, "role:support-agent");
});

test("access subject catalog includes concrete user accounts for the support workspace", () => {
  const subjects = normalizeAccessSubjectOptions(accessSubjectOptions);
  const memberNames = subjects
    .filter((option) => option.kind === "member")
    .map((option) => option.labelKey);

  assert.deepEqual(memberNames.slice(0, 3), [
    "accessSubject.support001.name",
    "accessSubject.support002.name",
    "accessSubject.supportLead001.name"
  ]);
  assert.equal(subjects.find((option) => option.id === "member:support-002")?.workspaceId, "ws-permission-package-approval");
  assert.equal(subjects.find((option) => option.id === "member:support-lead-001")?.subjectSelector, "user:support-lead-001");
});

test("access subject catalog can focus the directory on the current workspace", () => {
  const subjects = accessSubjectsForWorkspace(accessSubjectOptions, "ws-permission-package-approval");
  const memberIds = subjects.filter((option) => option.kind === "member").map((option) => option.id);

  assert.deepEqual(memberIds, [
    "member:support-001",
    "member:support-002",
    "member:support-lead-001",
    "member:security-reviewer-001"
  ]);
  assert.equal(accessSubjectsForWorkspace(accessSubjectOptions, "ws-unknown")[0].id, "role:support-agent");
});
