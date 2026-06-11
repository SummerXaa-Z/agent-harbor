import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import { translationKeys } from "../src/i18n.ts";

const controller = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const hook = readFileSync(new URL("../src/hooks/useAskAccessController.ts", import.meta.url), "utf8");
const navigation = readFileSync(new URL("../src/consoleNavigation.ts", import.meta.url), "utf8");
const view = readFileSync(new URL("../src/components/AskAccessView.tsx", import.meta.url), "utf8");
const workbench = readFileSync(new URL("../src/components/AiAdminPermissionWorkbench.tsx", import.meta.url), "utf8");
const operationalViews = readFileSync(new URL("../src/components/OperationalViews.tsx", import.meta.url), "utf8");
const capabilityView = readFileSync(new URL("../src/components/CapabilityGovernanceView.tsx", import.meta.url), "utf8");
const accessProfileView = readFileSync(new URL("../src/components/TenantAccessProfileView.tsx", import.meta.url), "utf8");
const accessProfileHook = readFileSync(new URL("../src/hooks/useAccessProfileController.ts", import.meta.url), "utf8");

test("answer-first access query is registered as a first-class console workspace", () => {
  assert.match(navigation, /\|\s+"ask"/);
  assert.match(navigation, /key:\s*"ask"/);
  assert.match(navigation, /primaryPanelKey:\s*"askAccess"/);
  assert.match(controller, /useAskAccessController/);
  assert.match(controller, /case "ask":/);
});

test("ask access state delegates access-query business rules to askJourney pure functions", () => {
  assert.match(hook, /buildExplainRequest/);
  assert.match(hook, /buildPermissionChangeHandoff/);
  assert.match(hook, /evidenceChainRows/);
  assert.match(hook, /fetchAccessDecisionExplanation/);
  assert.match(hook, /slice\(0,\s*5\)/);
});

test("ask access view exposes denied-to-fix handoff without automatic submission", () => {
  assert.match(view, /result\.outcome === "denied"/);
  assert.match(view, /onStartPermissionChange/);
  assert.doesNotMatch(view, /onApply/);
  assert.doesNotMatch(hook, /applyPermissionPackage|createPermissionPackageDraftFromApi|createPermissionPackageApprovalRequest/);
});

test("permission change handoff is consumed once and only pre-fills the editable workbench", () => {
  assert.match(controller, /permissionChange:\s*PermissionChangeHandoffContext \| null/);
  assert.match(controller, /permissionNotice:\s*PermissionChangeHandoffContext \| null/);
  assert.match(controller, /setAiAdminNewDraftMode\(true\)/);
  assert.match(controller, /subjectSelector:\s*context\.subjectId \?\? current\.subjectSelector/);
  assert.match(controller, /permissionChange:\s*null,\s*permissionNotice:\s*context/);
  assert.match(controller, /permissionHandoffContext=\{handoffContexts\.permissionNotice\}/);
  assert.match(controller, /onDismissPermissionHandoff/);
  assert.match(workbench, /className="permission-handoff-notice"/);
  assert.match(workbench, /onDismissPermissionHandoff/);
  assert.doesNotMatch(controller, /handoffContexts\.permissionChange[\s\S]{0,900}createPermissionPackageApprovalRequest/);
  assert.doesNotMatch(controller, /handoffContexts\.permissionChange[\s\S]{0,900}applyPermissionPackage/);
});

test("ask access copy is bilingual", () => {
  const english = new Set(translationKeys("en"));
  const chinese = new Set(translationKeys("zh-CN"));
  for (const key of [
    "ask.answerTitle",
    "ask.chainTitle",
    "ask.emptyDetail",
    "ask.intent.openAccess",
    "text.permissionHandoffDetail",
    "text.permissionHandoffTitle",
    "nav.ask",
    "navDetail.ask",
    "page.ask"
  ]) {
    assert.equal(english.has(key), true, `${key} missing in English`);
    assert.equal(chinese.has(key), true, `${key} missing in zh-CN`);
  }
});

test("resource pages hand off access questions to the answer-first workspace", () => {
  assert.match(operationalViews, /onQueryAccess: \(context: AskHandoffContext\) => void/);
  assert.match(operationalViews, /sourceView:\s*"registry"/);
  assert.match(capabilityView, /onQueryAccess: \(context: AskHandoffContext\) => void/);
  assert.match(capabilityView, /sourceView:\s*"capabilities"/);
  assert.match(accessProfileView, /AccessDecisionMovedPanel/);
  assert.match(accessProfileView, /sourceView:\s*"access"/);
  assert.match(accessProfileView, /action\.openAccessQuery/);
  assert.doesNotMatch(accessProfileHook, /fetchAccessDecisionExplanation/);
  assert.doesNotMatch(accessProfileHook, /decisionExplanation|decisionExplainMessage|decisionExplainLoading/);
});
