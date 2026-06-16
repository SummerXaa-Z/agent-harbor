import assert from "node:assert/strict";
import test from "node:test";

import {
  createTranslator
} from "../src/i18n.ts";
import {
  permissionProductionReadinessNextAction,
  sanitizeProductionReadinessAction
} from "../src/productionReadinessCopy.ts";

test("production readiness next actions avoid evidence wording for known backend strings", () => {
  const t = createTranslator("en");
  const translated = permissionProductionReadinessNextAction(
    "Verify permission package applied audit evidence before production readiness.",
    t
  );

  assert.equal(translated, "Verify the permission package applied audit record before the status check.");
  assert.doesNotMatch(translated, /\bevidence\b/i);
});

test("production readiness next action fallback sanitizes unknown evidence wording", () => {
  assert.equal(
    sanitizeProductionReadinessAction("Collect audit evidence before go-live."),
    "Collect audit records before go-live."
  );
  assert.equal(
    sanitizeProductionReadinessAction("补齐上线证据后再验收。"),
    "补齐上线记录后再验收。"
  );
});
