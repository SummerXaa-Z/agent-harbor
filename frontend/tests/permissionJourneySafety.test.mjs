import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const app = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const api = readFileSync(new URL("../src/api.ts", import.meta.url), "utf8");
const managementHook = readFileSync(new URL("../src/hooks/useManagementOperations.ts", import.meta.url), "utf8");

function functionBlock(name, source = app) {
  const start = source.indexOf(`async function ${name}(`);
  assert.notEqual(start, -1, `${name} function should exist`);
  return blockFromIndex(name, start, source);
}

function syncFunctionBlock(name, source = app) {
  const start = source.indexOf(`function ${name}(`);
  assert.notEqual(start, -1, `${name} function should exist`);
  return blockFromIndex(name, start, source);
}

function blockFromIndex(name, start, source) {
  const braceStart = source.indexOf("{", start);
  assert.notEqual(braceStart, -1, `${name} function should have a body`);
  let depth = 0;
  for (let index = braceStart; index < source.length; index += 1) {
    const char = source[index];
    if (char === "{") depth += 1;
    if (char === "}") {
      depth -= 1;
      if (depth === 0) return source.slice(start, index + 1);
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

test("approval request list responses are normalized before controller merge", () => {
  const block = functionBlock("fetchPermissionPackageApprovalRequests", api);

  assert.match(block, /request<unknown>/);
  assert.match(block, /Array\.isArray\(rows\)/);
  assert.match(block, /PermissionPackageApprovalRequest\[\]/);
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
  assert.match(block, /setAiAdminMessage\(\{ key: "message\.permissionApprovalWithdrawn" \}\)/);
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
  assert.match(resetBlock, /setAiAdminMessage\(null\)/);
});

test("new permission change ignores historical preview evidence until submitted", () => {
  const resetBlock = syncFunctionBlock("startNewAiAdminPermissionChange");
  const createBlock = functionBlock("createAiAdminApprovalRequest");

  assert.match(app, /const \[aiAdminNewDraftMode, setAiAdminNewDraftMode\] = useState\(false\)/);
  assert.match(resetBlock, /setAiAdminNewDraftMode\(true\)/);
  assert.match(app, /if \(aiAdminNewDraftMode\) \{\s*setAiAdminWorkbenchPreview\(null\);\s*setAiAdminApplication\(null\);\s*setAiAdminProductionReadiness\(null\);\s*setAiAdminApprovalRequests\(\[\]\);\s*return;/s);
  assert.match(app, /activeNav !== "ai-admin" \|\| aiAdminNewDraftMode \|\| !data\?\.loadedFromApi/);
  assert.match(createBlock, /setAiAdminNewDraftMode\(false\)/);
  assert.match(createBlock, /setAiAdminSelectedApprovalRequestId\(request\.id\)/);
  assert.match(createBlock, /setAiAdminWorkbenchPreview\(null\)/);
  assert.match(createBlock, /setAiAdminApplication\(null\)/);
  assert.match(createBlock, /setAiAdminApplicationHealth\(null\)/);
  assert.match(createBlock, /setAiAdminApplicationImpact\(null\)/);
  assert.match(createBlock, /setAiAdminProductionReadiness\(null\)/);
});

test("production evidence export reports the result on the main permission journey", () => {
  const block = functionBlock("exportAiAdminProductionEvidence");
  assert.match(block, /setAiAdminMessage\(\{ key: "message\.productionEvidenceRequiresLiveApi" \}\)/);
  assert.match(block, /setAiAdminMessage\(\{ key: "message\.productionEvidenceExported" \}\)/);
  assert.match(block, /setAiAdminMessage\(localizedErrorMessageState\(error, "error\.exportProductionEvidence"\)\)/);
});

test("permission journey messages store translation keys instead of rendered language snapshots", () => {
  const createBlock = functionBlock("createAiAdminApprovalRequest");
  const applyBlock = functionBlock("applyAiAdminPermissionPackage");
  const exportBlock = functionBlock("exportAiAdminProductionEvidence");

  assert.match(app, /type LocalizedMessage =/);
  assert.match(app, /function localizedMessageText\(message: LocalizedMessage \| null, t: Translator, language: Language\)/);
  assert.match(app, /const \[aiAdminMessage, setAiAdminMessage\] = useState<LocalizedMessage \| null>\(null\)/);
  assert.match(app, /const renderedAiAdminMessage = localizedMessageText\(aiAdminMessage, t, language\)/);
  assert.match(app, /message=\{renderedAiAdminMessage\}/);
  assert.match(createBlock, /setAiAdminMessage\(\{ key: "message\.permissionApprovalCreated", params: \{ id: request\.id \} \}\)/);
  assert.match(applyBlock, /setAiAdminMessage\(\{ key: "message\.permissionPackageApplied", params: \{ count: appliedCount \} \}\)/);
  assert.match(exportBlock, /setAiAdminMessage\(\{ key: "message\.productionEvidenceExported" \}\)/);
  assert.doesNotMatch(app, /setAiAdminMessage\(tx\(t,/);
});

test("permission apply consumed approval retry shows recovery guidance", () => {
  const block = functionBlock("applyAiAdminPermissionPackage");

  assert.match(app, /function isConsumedApprovalRetryError\(error: unknown\)/);
  assert.match(app, /PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED/);
  assert.match(block, /if \(isConsumedApprovalRetryError\(error\)\)/);
  assert.match(block, /refreshAiAdminApplicationHealth\(aiAdminForm, \{ requireLiveApi: false \}\)/);
  assert.match(block, /refreshAiAdminProductionReadiness\(aiAdminForm, \{ requireLiveApi: false \}\)/);
  assert.match(block, /message\.permissionApprovalAlreadyConsumedRecovery/);
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

test("permission readiness messages use business field names on the main journey", () => {
  const block = syncFunctionBlock("permissionReadinessMessages");

  assert.match(block, /callerInstanceId:\s*t\("form\.caller"\)/);
  assert.match(block, /subjectSelector:\s*t\("form\.accessSubject"\)/);
  assert.match(block, /tenantId:\s*t\("form\.tenant"\)/);
  assert.match(block, /workspaceId:\s*t\("form\.workspace"\)/);
  assert.doesNotMatch(block, /tenantId:\s*t\("form\.tenantId"\)/);
  assert.doesNotMatch(block, /workspaceId:\s*t\("form\.workspaceId"\)/);
  assert.doesNotMatch(block, /callerInstanceId:\s*t\("form\.callerInstance"\)/);
  assert.doesNotMatch(block, /fieldLabels\[field\] \?\? "subjectSelector"/);
});

test("retry validation messages are localized before reaching operator panels", () => {
  const agentBlock = functionBlock("submitAgent", managementHook);
  const routeBlock = functionBlock("submitRoutePolicy", managementHook);
  const helperBlock = syncFunctionBlock("retryFieldValidationMessage", managementHook);

  assert.match(helperBlock, /message\.validationRetryAttempts/);
  assert.match(helperBlock, /message\.validationRetryBackoff/);
  assert.match(agentBlock, /setAgentMessage\(retryFieldValidationMessage\(retry\.message, t\)\)/);
  assert.match(routeBlock, /setPolicyMessage\(retryFieldValidationMessage\(retry\.message, t\)\)/);
  assert.doesNotMatch(agentBlock, /setAgentMessage\(retry\.message\)/);
  assert.doesNotMatch(routeBlock, /setPolicyMessage\(retry\.message\)/);
  assert.doesNotMatch(app, /async function submitAgent\(/);
  assert.doesNotMatch(app, /async function submitRoutePolicy\(/);
});
