import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import { createTranslator } from "../src/i18n.ts";
import { knownDecisionReasons } from "../src/askJourney.ts";

// Phase 3 guards for the console message localization contract
// (docs/product/0.3.x-console-message-localization.md): keep the long tail
// covered — access decision reasons stay mapped, and the readiness/decision
// chain render paths must not regress to raw backend message text.

function readSource(relativePath) {
  return readFileSync(new URL(relativePath, import.meta.url), "utf8");
}

test("every backend access decision reason has localized copy", () => {
  const storeSources = [
    readSource("../../internal/store/memory.go"),
    readSource("../../internal/store/postgres.go")
  ];
  const backendReasons = new Set(
    storeSources.flatMap((source) =>
      [...source.matchAll(/Reason:\s*"([a-z][a-z ]+)"/g)].map((match) => match[1])
    )
  );
  const mappedReasons = new Set(Object.keys(knownDecisionReasons));

  assert.ok(backendReasons.size >= 14, `expected a real reason inventory, found ${backendReasons.size}`);
  for (const reason of backendReasons) {
    assert.ok(
      mappedReasons.has(reason),
      `backend reason "${reason}" has no frontend mapping — add an ask.reason.* entry`
    );
    const t = createTranslator("zh-CN");
    assert.notEqual(t(knownDecisionReasons[reason]), knownDecisionReasons[reason], `"${reason}" copy missing zh`);
  }
  for (const reason of mappedReasons) {
    assert.ok(
      backendReasons.has(reason),
      `mapped reason "${reason}" no longer exists backend-side — remove the stale entry`
    );
  }
});

test("the go-live readiness path renders blocker labels by key, not raw check messages", () => {
  const presentation = readSource("../src/productionAcceptance.ts");
  const overview = readSource("../src/components/GoLiveAcceptanceOverview.tsx");

  assert.match(presentation, /labelKey: `productionAcceptance\.blocker\.\$\{check\.code\}`/);
  // check.message is only allowed as the unknown-code fallback detail.
  assert.doesNotMatch(presentation, /labelKey: check\.message/);
  assert.doesNotMatch(overview, />\{[^}]*blockers\[0\]\.detail\}</);
  assert.doesNotMatch(overview, /labelKey:\s*[^,]*\.message/);
});

test("the ask decision chain renders record messages through the label resolver", () => {
  const view = readSource("../src/components/AskAccessView.tsx");

  assert.match(view, /accessDecisionRecordMessageLabel\(row, t\)/);
  for (const forbidden of [">{row.message}<", ">{record.message}<", ">{result.summary}<", ">{evidence.message}<"]) {
    assert.ok(!view.includes(forbidden), `raw backend text rendered: ${forbidden}`);
  }
});

test("tenant permission center errors resolve through the localized error path", () => {
  const controller = readSource("../src/ConsoleController.tsx");

  assert.match(controller, /permissionCenterError: localizedErrorMessage\(/);
  assert.doesNotMatch(controller, /permissionCenterError: error instanceof Error/);
});
