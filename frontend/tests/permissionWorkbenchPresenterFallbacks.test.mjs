import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const presenter = readFileSync(new URL("../src/permissionWorkbenchPresenters.ts", import.meta.url), "utf8");

function functionBlock(name) {
  const start = presenter.indexOf(`export function ${name}(`);
  assert.notEqual(start, -1, `${name} function should exist`);
  const braceStart = presenter.indexOf("{", start);
  assert.notEqual(braceStart, -1, `${name} function should have a body`);
  let depth = 0;
  for (let index = braceStart; index < presenter.length; index += 1) {
    const char = presenter[index];
    if (char === "{") depth += 1;
    if (char === "}") {
      depth -= 1;
      if (depth === 0) return presenter.slice(start, index + 1);
    }
  }
  throw new Error(`could not parse ${name} function body`);
}

test("production readiness fallback copy does not expose raw backend check messages", () => {
  const labelBlock = functionBlock("permissionProductionReadinessCheckLabel");
  const messageBlock = functionBlock("permissionProductionReadinessCheckMessage");

  assert.match(labelBlock, /knownLabel\(t, `productionCheck\.\$\{code\}`, "productionCheck\.unknown"\)/);
  assert.match(messageBlock, /knownLabel\(t, `productionCheck\.detail\.\$\{check\.code\}`, "productionCheck\.detail\.unknown"\)/);
  assert.doesNotMatch(messageBlock, /check\.message/);
});

test("permission preflight and policy fallbacks do not expose raw service wording", () => {
  const preflightBlock = functionBlock("permissionApplyPreflightCheckMessage");
  const nextActionCodeBlock = functionBlock("permissionApplyPreflightNextActionByCode");
  const nextActionBlock = functionBlock("permissionApplyPreflightNextAction");
  const policyBlock = functionBlock("permissionPolicyReasonMessage");
  const readinessBlock = functionBlock("permissionReadinessMessages");

  assert.match(preflightBlock, /knownLabel\(t, `permissionPreflight\.detail\.\$\{check\.code\}`, "permissionPreflight\.detail\.unknown"\)/);
  assert.doesNotMatch(preflightBlock, /check\.message/);
  assert.match(nextActionCodeBlock, /review_current_application: "permissionPreflight\.next\.reviewAlreadyApplied"/);
  assert.match(nextActionCodeBlock, /apply_permission_package: "permissionPreflight\.next\.applyWhenReady"/);
  assert.match(nextActionBlock, /if \(code\) return permissionApplyPreflightNextActionByCode\(code, t\)/);
  assert.match(nextActionBlock, /t\("permissionPreflight\.next\.unknown"\)/);
  assert.match(nextActionBlock, /Review the latest permission request status before applying the same permission request again\./);
  assert.match(nextActionBlock, /permissionPreflight\.next\.reviewAlreadyApplied/);
  assert.doesNotMatch(nextActionBlock, /: action/);
  assert.match(policyBlock, /if \(!reason\.reasonKey\) return t\("permissionPolicy\.unknownReason"\)/);
  assert.doesNotMatch(policyBlock, /return reason\.message/);
  assert.match(readinessBlock, /t\("message\.permissionPackageReadinessWarning"\)/);
  assert.doesNotMatch(readinessBlock, /: warning/);
});
