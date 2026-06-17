import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const presenter = readFileSync(new URL("../src/permissionWorkbenchPresenters.ts", import.meta.url), "utf8");

test("permission journey asks operators to complete configuration before approval when draft fields are missing", () => {
  const needsInputStart = presenter.indexOf('args.draft.readiness.missingFields.length > 0 || args.workbenchStatus === "needs_input"');
  const blockedStart = presenter.indexOf('args.workbenchStatus === "blocked"', needsInputStart);
  assert.notEqual(needsInputStart, -1);
  assert.notEqual(blockedStart, -1);

  const needsInputBranch = presenter.slice(needsInputStart, blockedStart);
  assert.match(needsInputBranch, /nextActionKey:\s*"action\.completePermissionRequest"/);
  assert.doesNotMatch(needsInputBranch, /nextActionKey:\s*"action\.createApprovalRequest"/);
});

test("permission journey still sends complete requests to approval after configuration is ready", () => {
  const fallbackStart = presenter.indexOf('labelKey: "permissionJourney.status.needsApproval"');
  assert.notEqual(fallbackStart, -1);
  const fallback = presenter.slice(fallbackStart, fallbackStart + 240);

  assert.match(fallback, /labelKey:\s*"permissionJourney\.status\.needsApproval"/);
  assert.match(fallback, /nextActionKey:\s*"action\.createApprovalRequest"/);
});
