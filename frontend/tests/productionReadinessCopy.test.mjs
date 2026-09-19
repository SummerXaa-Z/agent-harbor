import assert from "node:assert/strict";
import test from "node:test";

import { createTranslator } from "../src/i18n.ts";
import {
  permissionProductionReadinessNextAction,
  sanitizeProductionReadinessAction
} from "../src/productionReadinessCopy.ts";

test("production readiness next actions resolve copy by next action code", () => {
  const t = createTranslator("en");

  assert.equal(
    permissionProductionReadinessNextAction(
      "verify_applied_audit",
      "Verify the permission package applied audit record before production readiness.",
      t
    ),
    "Verify the permission package applied audit record before the status check."
  );
  assert.doesNotMatch(
    permissionProductionReadinessNextAction("verify_applied_audit", "", t),
    /\bevidence\b/i
  );
});

test("drift and subject-scope next actions have localized copy in both languages", () => {
  for (const language of ["en", "zh-CN"]) {
    const t = createTranslator(language);
    const reapply = permissionProductionReadinessNextAction(
      "reapply_permission_package",
      "Review the changed capability contract and apply a fresh permission package approval.",
      t
    );
    const reviewSubject = permissionProductionReadinessNextAction(
      "review_subject_scope",
      "Use the application subject selector and a production subject that it covers.",
      t
    );

    assert.notEqual(
      reapply,
      "Review the changed capability contract and apply a fresh permission package approval."
    );
    assert.notEqual(
      reviewSubject,
      "Use the application subject selector and a production subject that it covers."
    );
  }
});

test("unknown next action codes fall back to the sanitized backend message", () => {
  const t = createTranslator("en");

  assert.equal(
    permissionProductionReadinessNextAction(
      "some_future_action",
      "Collect audit evidence before go-live.",
      t
    ),
    "Collect audit records before go-live."
  );
});

test("sanitize fallback keeps evidence wording out of visible copy", () => {
  assert.equal(
    sanitizeProductionReadinessAction("Collect audit evidence before go-live."),
    "Collect audit records before go-live."
  );
  assert.equal(
    sanitizeProductionReadinessAction("补齐上线证据后再验收。"),
    "补齐上线记录后再验收。"
  );
});
