import assert from "node:assert/strict";
import test from "node:test";

import {
  accessSubjectOptionForId,
  accessSubjectOptionForSelector,
  accessSubjectOptions,
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
    subjectSelector: "user:support-*"
  }]);

  assert.equal(serverOptions.length, 1);
  assert.equal(serverOptions[0].id, "role:support-agent");
  assert.equal(normalizeAccessSubjectOptions([])[0].id, "role:support-agent");
});
