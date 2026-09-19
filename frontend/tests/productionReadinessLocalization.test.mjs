import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import { createTranslator } from "../src/i18n.ts";
import { productionReadinessNextActionKeys } from "../src/productionReadinessCopy.ts";

// Guard for the console message localization contract
// (docs/product/0.3.x-console-message-localization.md): the backend emits
// readiness check codes and next-action codes; the frontend must keep
// localized copy for every one of them. Reading the Go source keeps this
// test honest when new checks appear server-side.
const serverSource = readFileSync(
  new URL("../../internal/httpapi/server.go", import.meta.url),
  "utf8"
);

function unique(matches) {
  return [...new Set(matches)];
}

function readinessBlockingCodes() {
  const found = [...serverSource.matchAll(
    /permissionPackageProductionReadinessCheckFor\("([a-z_]+)",\s*domain\.PermissionPackagePreflight(Blocking|Warning)/g
  )];
  return unique(found.map((match) => match[1]));
}

function readinessNextActionCodes() {
  const found = [...serverSource.matchAll(
    /permissionPackageProductionAddNextAction\(&result,\s*"([a-z_]+)"/g
  )];
  return unique(found.map((match) => match[1]));
}

test("every backend blocking readiness code has a localized blocker label", () => {
  const codes = readinessBlockingCodes();
  assert.ok(codes.length >= 10, `expected a real readiness inventory, found ${codes.length}`);
  for (const code of codes) {
    const key = `productionAcceptance.blocker.${code}`;
    for (const language of ["en", "zh-CN"]) {
      const t = createTranslator(language);
      assert.notEqual(t(key), key, `${key} is missing ${language} copy`);
    }
  }
});

test("every backend readiness next action code maps to localized copy", () => {
  const codes = readinessNextActionCodes();
  assert.ok(codes.length >= 12, `expected a real next-action inventory, found ${codes.length}`);
  for (const code of codes) {
    const key = productionReadinessNextActionKeys[code];
    assert.ok(key, `next action code ${code} has no copy mapping`);
    for (const language of ["en", "zh-CN"]) {
      const t = createTranslator(language);
      assert.notEqual(t(key), key, `${key} is missing ${language} copy`);
    }
  }
});

test("the go-live overview resolves the next action by code, not message text", () => {
  const overview = readFileSync(
    new URL("../src/components/GoLiveAcceptanceOverview.tsx", import.meta.url),
    "utf8"
  );
  assert.match(overview, /permissionProductionReadinessNextAction\(\s*productionReadiness\.nextActionCode/);
});
