import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const app = readFileSync(new URL("../src/App.tsx", import.meta.url), "utf8");

function functionBlock(name) {
  const start = app.indexOf(`async function ${name}(`);
  assert.notEqual(start, -1, `${name} function should exist`);
  return blockFromIndex(name, start);
}

function syncFunctionBlock(name) {
  const start = app.indexOf(`function ${name}(`);
  assert.notEqual(start, -1, `${name} function should exist`);
  return blockFromIndex(name, start);
}

function blockFromIndex(name, start) {
  const braceStart = app.indexOf("{", start);
  assert.notEqual(braceStart, -1, `${name} function should have a body`);
  let depth = 0;
  for (let index = braceStart; index < app.length; index += 1) {
    const char = app[index];
    if (char === "{") depth += 1;
    if (char === "}") {
      depth -= 1;
      if (depth === 0) return app.slice(start, index + 1);
    }
  }
  throw new Error(`could not parse ${name} function body`);
}

test("runtime validation keeps the edited permission request draft isolated", () => {
  const block = functionBlock("runAiAdminApprovalJourney");

  assert.match(block, /const validationForm: PermissionPackageDraftInput/);
  assert.match(block, /createPermissionPackageDraftFromApi\(validationForm/);
  assert.doesNotMatch(block, /setAiAdminForm\(/);
  assert.doesNotMatch(block, /setScope\(/);
  assert.doesNotMatch(block, /setAccessFilters\(/);
});

test("approval request creation blocks blank and duplicate submissions before the network call", () => {
  const block = functionBlock("createAiAdminApprovalRequest");
  const actionIndex = block.indexOf('setAiAdminApprovalAction("create")');
  const blankTextIndex = block.indexOf("aiAdminForm.requestText.trim()");
  const inFlightIndex = block.indexOf("approvalCreateInFlightRef.current");
  const pendingIndex = block.indexOf('aiAdminApprovalRequest?.status === "pending"');

  assert.notEqual(actionIndex, -1, "create action state should still be set for real submissions");
  assert.ok(inFlightIndex >= 0 && inFlightIndex < actionIndex, "in-flight guard must run before action state");
  assert.ok(blankTextIndex >= 0 && blankTextIndex < actionIndex, "blank request guard must run before action state");
  assert.ok(pendingIndex >= 0 && pendingIndex < actionIndex, "pending duplicate guard must run before action state");
  assert.match(block, /message\.permissionApprovalRequestTextRequired/);
  assert.match(block, /message\.permissionApprovalAlreadyPending/);
  assert.match(block, /createPermissionPackageApprovalRequest\(aiAdminForm, adminKey\)/);
});

test("approval resolution ignores the follow-up click from a submit double-click", () => {
  const createBlock = functionBlock("createAiAdminApprovalRequest");
  assert.match(createBlock, /startApprovalResolutionCooldown\(\)/);

  [
    ["approveAiAdminApprovalRequest", "approve"],
    ["rejectAiAdminApprovalRequest", "reject"],
  ].forEach(([functionName, action]) => {
    const block = functionBlock(functionName);
    const guardIndex = block.indexOf("approvalResolveBlockedRef.current");
    const actionIndex = block.indexOf(`setAiAdminApprovalAction("${action}")`);

    assert.notEqual(actionIndex, -1, `${functionName} should still set action state for real submissions`);
    assert.ok(guardIndex >= 0 && guardIndex < actionIndex, `${functionName} must check the post-create cooldown before resolving`);
  });
});

test("approval rejection requires a trimmed reviewer reason before the network call", () => {
  const block = functionBlock("rejectAiAdminApprovalRequest");
  const trimIndex = block.indexOf("const reviewerComment = comment?.trim()");
  const messageIndex = block.indexOf("message.permissionApprovalRejectReasonRequired");
  const actionIndex = block.indexOf('setAiAdminApprovalAction("reject")');
  const networkIndex = block.indexOf("rejectPermissionPackageApprovalRequest(");

  assert.match(block, /async function rejectAiAdminApprovalRequest\(requestId\?: string, comment\?: string\)/);
  assert.ok(trimIndex >= 0 && trimIndex < actionIndex, "reject reason should be normalized before action state");
  assert.ok(messageIndex >= 0 && messageIndex < actionIndex, "empty reason should be reported before action state");
  assert.ok(actionIndex >= 0 && actionIndex < networkIndex, "real submissions should still set action state before network call");
  assert.match(block, /comment: reviewerComment/);
});

test("approval withdraw uses the requester action and resets approval-dependent evidence", () => {
  const block = functionBlock("withdrawAiAdminApprovalRequest");
  const trimIndex = block.indexOf("const withdrawComment = comment?.trim()");
  const actionIndex = block.indexOf('setAiAdminApprovalAction("withdraw")');
  const networkIndex = block.indexOf("withdrawPermissionPackageApprovalRequest(");

  assert.match(block, /async function withdrawAiAdminApprovalRequest\(comment\?: string\)/);
  assert.ok(trimIndex >= 0 && trimIndex < actionIndex, "withdraw comment should be normalized before action state");
  assert.ok(actionIndex >= 0 && actionIndex < networkIndex, "real withdraw should set action state before network call");
  assert.match(block, /withdrawPermissionPackageApprovalRequest\(aiAdminApprovalRequest\.id, \{ comment: withdrawComment \}, adminKey\)/);
  assert.match(block, /upsertAiAdminApprovalRequest\(request\)/);
  assert.match(block, /setAiAdminWorkbenchPreview\(null\)/);
  assert.match(block, /setAiAdminApplyPreflight\(null\)/);
  assert.match(block, /setAiAdminProductionReadiness\(null\)/);
  assert.match(block, /setAiAdminMessage\(t\("message\.permissionApprovalWithdrawn"\)\)/);
});

test("go-live completion exits preserve context or reset the current permission change", () => {
  const openAccessBlock = syncFunctionBlock("openAiAdminAccessProfile");
  assert.match(openAccessBlock, /setScope\(\(current\) => \(\{ \.\.\.current, tenantId: aiAdminForm\.tenantId \}\)\)/);
  assert.match(openAccessBlock, /workspaceId: aiAdminForm\.workspaceId/);
  assert.match(openAccessBlock, /callerInstanceId: aiAdminForm\.callerInstanceId/);
  assert.match(openAccessBlock, /targetId: aiAdminForm\.targetId/);
  assert.match(openAccessBlock, /setActiveNav\("access"\)/);

  const resetBlock = syncFunctionBlock("startNewAiAdminPermissionChange");
  assert.match(resetBlock, /setAiAdminForm\(/);
  assert.match(resetBlock, /setAiAdminApplication\(null\)/);
  assert.match(resetBlock, /setAiAdminProductionReadiness\(null\)/);
  assert.match(resetBlock, /setAiAdminApprovalRequests\(\[\]\)/);
  assert.match(resetBlock, /setAiAdminMessage\(""\)/);
});

test("production evidence export reports the result on the main permission journey", () => {
  const block = functionBlock("exportAiAdminProductionEvidence");
  assert.match(block, /setAiAdminMessage\(t\("message\.productionEvidenceRequiresLiveApi"\)\)/);
  assert.match(block, /setAiAdminMessage\(t\("message\.productionEvidenceExported"\)\)/);
  assert.match(block, /setAiAdminMessage\(localizedErrorMessage\(t, language, error, "error\.exportProductionEvidence"\)\)/);
});

test("permission journey mutation handlers require live API before network writes", () => {
  [
    ["runAiAdminApprovalJourney", "message.fallbackDataModeActionBlocked", "createTenant("],
    ["createAiAdminApprovalRequest", "message.permissionApprovalRequiresLiveApi", "createPermissionPackageApprovalRequest("],
    ["approveAiAdminApprovalRequest", "message.permissionApprovalRequiresLiveApi", "approvePermissionPackageApprovalRequest("],
    ["rejectAiAdminApprovalRequest", "message.permissionApprovalRequiresLiveApi", "rejectPermissionPackageApprovalRequest("],
    ["withdrawAiAdminApprovalRequest", "message.permissionApprovalRequiresLiveApi", "withdrawPermissionPackageApprovalRequest("],
    ["applyAiAdminPermissionPackage", "message.fallbackDataModeActionBlocked", "applyPermissionPackage("],
    ["exportAiAdminProductionEvidence", "message.productionEvidenceRequiresLiveApi", "fetchPermissionPackageProductionEvidenceReport("],
  ].forEach(([functionName, liveApiMessage, networkCall]) => {
    const block = functionBlock(functionName);
    const liveApiGuardIndex = block.indexOf("!data?.loadedFromApi");
    const liveApiMessageIndex = block.indexOf(liveApiMessage);
    const networkCallIndex = block.indexOf(networkCall);

    assert.ok(liveApiGuardIndex >= 0, `${functionName} should check live API data`);
    assert.ok(liveApiMessageIndex > liveApiGuardIndex, `${functionName} should explain the live API requirement`);
    assert.ok(networkCallIndex > liveApiMessageIndex, `${functionName} should guard before ${networkCall}`);
  });
});
