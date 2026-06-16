import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import { translationKeys } from "../src/i18n.ts";

const controller = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const hook = readFileSync(new URL("../src/hooks/useAskAccessController.ts", import.meta.url), "utf8");
const navigation = readFileSync(new URL("../src/consoleNavigation.ts", import.meta.url), "utf8");
const types = readFileSync(new URL("../src/types.ts", import.meta.url), "utf8");
const view = readFileSync(new URL("../src/components/AskAccessView.tsx", import.meta.url), "utf8");
const styles = readFileSync(new URL("../src/styles.css", import.meta.url), "utf8");
const workbench = readFileSync(new URL("../src/components/AiAdminPermissionWorkbench.tsx", import.meta.url), "utf8");
const operationalViews = readFileSync(new URL("../src/components/OperationalViews.tsx", import.meta.url), "utf8");
const capabilityView = readFileSync(new URL("../src/components/CapabilityGovernanceView.tsx", import.meta.url), "utf8");
const accessProfileView = readFileSync(new URL("../src/components/TenantAccessProfileView.tsx", import.meta.url), "utf8");
const accessProfileHook = readFileSync(new URL("../src/hooks/useAccessProfileController.ts", import.meta.url), "utf8");
const presenters = readFileSync(new URL("../src/consolePresenters.ts", import.meta.url), "utf8");
const resourceLifecycleActionPlanner = readFileSync(new URL("../src/resourceLifecycleActionPlanner.ts", import.meta.url), "utf8");

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
  assert.match(types, /sourceView: 'ask' \| 'tenants' \| 'registry'/);
  assert.match(resourceLifecycleActionPlanner, /sourceView: "registry"/);
  assert.match(workbench, /permissionHandoffContext\?\.sourceView === "registry"/);
  assert.match(workbench, /className="permission-handoff-notice"/);
  assert.match(workbench, /onDismissPermissionHandoff/);
  assert.doesNotMatch(controller, /handoffContexts\.permissionChange[\s\S]{0,900}createPermissionPackageApprovalRequest/);
  assert.doesNotMatch(controller, /handoffContexts\.permissionChange[\s\S]{0,900}applyPermissionPackage/);
});

test("ask access copy is bilingual", () => {
  const english = new Set(translationKeys("en"));
  const chinese = new Set(translationKeys("zh-CN"));
  for (const key of [
    "ask.answerPendingTitle",
    "ask.answerTitle",
    "ask.chainTitle",
    "ask.dataSourceTitle",
    "ask.emptyDetail",
    "ask.group.access",
    "ask.group.context",
    "ask.intent.openAccess",
    "ask.liveMode",
    "ask.questionTitle",
    "text.permissionHandoffDetail",
    "text.permissionHandoffRegistryTitle",
    "text.permissionHandoffTitle",
    "resource.permissionIntent",
    "nav.ask",
    "navDetail.ask",
    "page.ask"
  ]) {
    assert.equal(english.has(key), true, `${key} missing in English`);
    assert.equal(chinese.has(key), true, `${key} missing in zh-CN`);
  }
});

test("ask access view keeps the primary path answer-first and business-readable", () => {
  assert.match(view, /className="ask-workspace"/);
  assert.match(view, /className="ask-context-column"/);
  assert.match(view, /className="ask-query-groups"/);
  assert.match(view, /className="ask-query-grid ask-query-grid-access"/);
  assert.match(view, /className="ask-answer-empty"/);
  assert.match(view, /permissionEntityDisplayName\(tenant\.name, t\)/);
  assert.match(view, /function agentOptions\(agents: Agent\[], t: Translator\)/);
  assert.doesNotMatch(view, /EmptyRow/);
  assert.doesNotMatch(view, /ask-sentence-text/);
  assert.match(presenters, /"Policy Router": t\("demo\.policyRouterTarget"\)/);
  assert.match(presenters, /"Sandbox": t\("demo\.workspaceSandbox"\)/);
});

test("ask access shows the data-source mode once in the context card", () => {
  const liveModeReferences = view.match(/t\("ask\.liveMode"\)/g) ?? [];
  const sampleModeReferences = view.match(/t\("ask\.sampleMode"\)/g) ?? [];

  assert.equal(liveModeReferences.length, 1);
  assert.equal(sampleModeReferences.length, 1);
  assert.match(view, /<strong>\{t\("ask\.dataSourceTitle"\)\}<\/strong>[\s\S]*<Badge tone=\{liveDataAvailable \? "success" : "warning"\}>/);
});

test("ask access form controls use one coherent control treatment", () => {
  assert.match(view, /name="accessSubject"/);
  assert.match(view, /autoComplete="off"/);
  assert.match(styles, /--ask-control-height:\s*40px/);
  assert.match(styles, /\.ask-query-field \.approval-dropdown-trigger:focus-visible,\s*\.ask-subject-field input:focus-visible\s*\{/s);
  assert.match(styles, /\.ask-query-field \.approval-dropdown-trigger:hover,\s*\.ask-subject-field input:hover:not\(:focus-visible\)\s*\{/s);
  assert.match(styles, /\.ask-query-grid-access\s*\{[^}]*grid-template-columns:\s*repeat\(3,\s*minmax\(0,\s*1fr\)\)/s);
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
