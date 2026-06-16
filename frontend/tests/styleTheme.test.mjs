import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { readFileSync } from "node:fs";
import test from "node:test";

const baseStyles = readFileSync(new URL("../src/styles.css", import.meta.url), "utf8");
const workbenchStyles = readFileSync(new URL("../src/styles/permission-workbench.css", import.meta.url), "utf8");
const styles = `${baseStyles}\n${workbenchStyles}`;
const appEntry = readFileSync(new URL("../src/App.tsx", import.meta.url), "utf8");
const app = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const workbench = readFileSync(new URL("../src/components/AiAdminPermissionWorkbench.tsx", import.meta.url), "utf8");
const permissionWorkbenchPresenters = readExistingFile(new URL("../src/permissionWorkbenchPresenters.ts", import.meta.url));
const permissionWorkbenchParts = readExistingFile(new URL("../src/components/PermissionWorkbenchParts.tsx", import.meta.url));
const permissionApprovalDecisionHook = readExistingFile(new URL("../src/hooks/usePermissionApprovalDecision.ts", import.meta.url));
const capabilityGovernanceView = readFileSync(new URL("../src/components/CapabilityGovernanceView.tsx", import.meta.url), "utf8");
const consoleViews = readFileSync(new URL("../src/components/ConsoleViews.tsx", import.meta.url), "utf8");
const consolePrimitives = readFileSync(new URL("../src/components/ConsolePrimitives.tsx", import.meta.url), "utf8");
const goLiveAcceptanceOverview = readFileSync(new URL("../src/components/GoLiveAcceptanceOverview.tsx", import.meta.url), "utf8");
const managementForms = readFileSync(new URL("../src/components/ManagementForms.tsx", import.meta.url), "utf8");
const operationalViews = readFileSync(new URL("../src/components/OperationalViews.tsx", import.meta.url), "utf8");
const runtimeEvidenceViews = readFileSync(new URL("../src/components/RuntimeEvidenceViews.tsx", import.meta.url), "utf8");
const dropdown = readFileSync(new URL("../src/components/ApprovalDropdown.tsx", import.meta.url), "utf8");
const technicalId = readFileSync(new URL("../src/components/TechnicalId.tsx", import.meta.url), "utf8");
const tenantOrganizationView = readFileSync(new URL("../src/components/TenantOrganizationView.tsx", import.meta.url), "utf8");
const resourceLifecycleActionPlanner = readExistingFile(new URL("../src/resourceLifecycleActionPlanner.ts", import.meta.url));
const resourceLifecycleView = readFileSync(new URL("../src/components/ResourceLifecycleView.tsx", import.meta.url), "utf8");
const productionJourney = readExistingFile(new URL("../src/productionJourney.ts", import.meta.url));
const productionJourneyCheckpoint = readExistingFile(new URL("../src/components/ProductionJourneyCheckpoint.tsx", import.meta.url));
const productionAcceptance = readExistingFile(new URL("../src/productionAcceptance.ts", import.meta.url));
const ui = readFileSync(new URL("../src/components/ui.tsx", import.meta.url), "utf8");
const managementOperationsHookUrl = new URL("../src/hooks/useManagementOperations.ts", import.meta.url);
const capabilityGovernanceHookUrl = new URL("../src/hooks/useCapabilityGovernanceController.ts", import.meta.url);
const accessProfileHookUrl = new URL("../src/hooks/useAccessProfileController.ts", import.meta.url);
const coreJourneyHookUrl = new URL("../src/hooks/useCoreJourneyController.ts", import.meta.url);
const connectionDiagnosticsHookUrl = new URL("../src/hooks/useConnectionDiagnostics.ts", import.meta.url);

function readExistingFile(url) {
  assert.equal(existsSync(url), true, `${url.pathname} should exist`);
  return readFileSync(url, "utf8");
}

function stylesWithoutRootTokens(source) {
  return source.replace(/:root\s*\{[\s\S]*?\n\}/, "");
}

test("theme tokens define one restrained brand color system", () => {
  assert.match(styles, /--text-xs:\s*11px;/);
  assert.match(styles, /--text-base:\s*13px;/);
  assert.match(styles, /--space-4:\s*16px;/);
  assert.match(styles, /--radius-control:\s*6px;/);
  assert.match(styles, /--brand:\s*#0071e3;/);
  assert.match(styles, /--brand-strong:\s*#0057b8;/);
  assert.match(styles, /--brand-soft:\s*#edf5ff;/);
  assert.match(styles, /--brand-border:\s*#b9d7fb;/);
  assert.match(styles, /--shadow-focus:\s*0 0 0 3px rgba\(0,\s*113,\s*227,\s*0\.18\);/);
});

test("app entry stays as a thin production shell", () => {
  assert.ok(appEntry.split("\n").length <= 20, "App.tsx should only delegate to the console controller");
  assert.match(appEntry, /import \{ ConsoleController \} from "\.\/ConsoleController"/);
  assert.match(appEntry, /return <ConsoleController \/>/);
  assert.doesNotMatch(appEntry, /useState|useEffect|fetch|<section|<details|navGroups|AgentCreateForm|AiAdminPermissionWorkbench/);
});

test("console controller delegates non-ai-admin state domains to hooks", () => {
  const managementOperations = readExistingFile(managementOperationsHookUrl);
  const capabilityGovernance = readExistingFile(capabilityGovernanceHookUrl);
  const accessProfile = readExistingFile(accessProfileHookUrl);
  const coreJourney = readExistingFile(coreJourneyHookUrl);
  const localStateCells = app.match(/\buseState(?:<|\()/g) ?? [];

  assert.match(app, /from "\.\/hooks\/useManagementOperations"/);
  assert.match(app, /from "\.\/hooks\/useCapabilityGovernanceController"/);
  assert.match(app, /from "\.\/hooks\/useAccessProfileController"/);
  assert.match(app, /from "\.\/hooks\/useCoreJourneyController"/);
  assert.ok(localStateCells.length <= 52, `ConsoleController still owns ${localStateCells.length} local useState cells`);

  assert.match(managementOperations, /export function useManagementOperations/);
  assert.match(capabilityGovernance, /export function useCapabilityGovernanceController/);
  assert.match(accessProfile, /export function useAccessProfileController/);
  assert.match(coreJourney, /export function useCoreJourneyController/);

  for (const movedHandler of [
    "async function submitAgent",
    "async function handleAgentStatusChange",
    "async function submitKey",
    "async function submitCredentialRotation",
    "async function submitRoutePolicy",
    "async function handleRefreshTargetCapabilities",
    "async function handleApproveCapability",
    "async function submitCapabilityGrantChain",
    "async function refreshAccessProfile",
    "async function explainAccessDecisionFromProfile",
    "async function refreshCoreJourneyPreflight",
    "async function resetCoreJourneySession",
    "async function runCoreJourney"
  ]) {
    assert.equal(app.includes(movedHandler), false, `${movedHandler} should live in a domain hook`);
  }
});

test("ai admin workbench has a growth guard while controller hooks are split", () => {
  assert.ok(
    workbench.split("\n").length <= 1300,
    "AiAdminPermissionWorkbench should delegate approval decision state before adding more UI"
  );
  assert.match(app, /from "\.\/components\/AiAdminPermissionWorkbench"/);
  assert.match(workbench, /from "\.\.\/hooks\/usePermissionApprovalDecision"/);
  assert.match(workbench, /from "\.\.\/permissionWorkbenchPresenters"/);
  assert.match(workbench, /from "\.\/PermissionWorkbenchParts"/);
  assert.match(permissionApprovalDecisionHook, /export function usePermissionApprovalDecision/);
  assert.match(permissionApprovalDecisionHook, /export type ApprovalDecisionAction = "approve" \| "reject" \| "withdraw"/);
  assert.match(permissionWorkbenchPresenters, /export function resolvePermissionJourneyStatus/);
  assert.match(permissionWorkbenchPresenters, /export function permissionWorkbenchStepDisplayDetailCode/);
  assert.match(permissionWorkbenchPresenters, /export function permissionWorkbenchStepDisplayStatus/);
  assert.match(permissionWorkbenchParts, /export function CapabilityChipList/);
  assert.doesNotMatch(workbench, /import \{ useState \} from "react"/);
  assert.doesNotMatch(workbench, /setPendingApprovalDecision/);
  assert.doesNotMatch(workbench, /function resolvePermissionJourneyStatus/);
  assert.doesNotMatch(workbench, /function permissionWorkbenchStepDisplayDetailCode/);
  assert.doesNotMatch(workbench, /function permissionWorkbenchStepDisplayStatus/);
  assert.doesNotMatch(workbench, /function CapabilityChipList/);
  assert.doesNotMatch(workbench, /useManagementOperations|useCoreJourneyController|useAccessProfileController/);
});

test("component styles consume tokens instead of ad hoc visual values", () => {
  const componentStyles = stylesWithoutRootTokens(styles);
  const rawHexValues = componentStyles.match(/#[0-9a-fA-F]{3,8}\b/g) ?? [];
  const rawRgbaValues = componentStyles.match(/rgba\([^)]+\)/g) ?? [];
  const fontWeights = Array.from(componentStyles.matchAll(/font-weight:\s*([^;]+);/g), (match) => match[1].trim());
  const unexpectedWeights = fontWeights.filter((weight) => !["400", "600", "700"].includes(weight));

  assert.equal(rawHexValues.length, 0, `raw component colors: ${rawHexValues.join(", ")}`);
  assert.equal(rawRgbaValues.length, 0, `raw component rgba values: ${rawRgbaValues.join(", ")}`);
  assert.deepEqual([...new Set(unexpectedWeights)].sort(), []);
});

test("focus and button controls use the shared production interaction tokens", () => {
  const bareScopedFocusSelectors = Array.from(
    styles.matchAll(/(?:^|})\s*([^{}]*(?:\.approval-|\.access-)[^{}]*:focus(?!-visible)[^{}]*)\{/g),
    (match) => match[1].trim()
  );

  assert.match(styles, /:where\(button,\s*input,\s*select,\s*textarea,\s*summary\):focus-visible\s*\{[^}]*box-shadow:\s*var\(--shadow-focus\);/s);
  assert.match(styles, /\.primary-button,\s*\n\.secondary-button\s*\{[^}]*min-height:\s*var\(--control-height\);/s);
  assert.match(styles, /\.primary-button,\s*\n\.secondary-button\s*\{[^}]*padding:\s*0 var\(--space-4\);/s);
  assert.match(styles, /\.primary-button\s*\{[^}]*box-shadow:\s*var\(--shadow-primary\);/s);
  assert.match(styles, /\.primary-button:hover:not\(:disabled\)\s*\{[^}]*background:\s*var\(--brand-strong\);/s);
  assert.match(styles, /\.table-action\s*\{[^}]*min-height:\s*var\(--control-height-compact\);/s);
  assert.match(styles, /\.table-action\s*\{[^}]*background:\s*var\(--surface\);/s);
  assert.match(styles, /\.approval-action-button\s*\{[^}]*min-height:\s*var\(--control-height-compact\);/s);
  assert.deepEqual(bareScopedFocusSelectors, []);
});

test("legacy cyan token is not used as a competing theme color", () => {
  assert.equal(styles.includes("--cyan"), false);
  assert.equal(styles.includes("var(--cyan"), false);
});

test("approval decision actions use styled semantic buttons", () => {
  assert.match(workbench, /className="approval-action-button is-primary"/);
  assert.match(workbench, /className="approval-action-button is-danger"/);
  assert.match(styles, /\.approval-action-button\s*\{/);
  assert.match(styles, /\.approval-action-button\.is-primary\s*\{/);
  assert.match(styles, /\.approval-action-button\.is-danger\s*\{/);
});

test("approval dropdown renders as an in-app menu instead of native select chrome", () => {
  assert.match(dropdown, /const menuOpen = open && !disabled/);
  assert.match(dropdown, /className=\{`approval-dropdown \$\{menuOpen \? "is-open" : ""\} \$\{disabled \? "is-disabled" : ""\}`\}/);
  assert.match(dropdown, /role="listbox"/);
  assert.match(styles, /\.approval-dropdown-menu\s*\{[^}]*background:\s*var\(--surface\);/s);
  assert.match(styles, /\.approval-dropdown-menu\s*\{[^}]*box-shadow:\s*var\(--shadow-pop\);/s);
  assert.match(styles, /\.approval-dropdown-option\.is-selected\s*\{[^}]*background:\s*var\(--brand-soft\);/s);
});

test("product shell removes demo controls and scopes connection settings", () => {
  assert.equal(app.includes("className=\"search-box\""), false);
  assert.equal(app.includes("className=\"admin-key-box\""), false);
  assert.equal(app.includes("t(\"control.filter\")"), false);
  assert.equal(styles.includes(".search-box"), false);
  assert.equal(styles.includes(".admin-key-box"), false);
  assert.equal(app.includes("workspaceTabs"), false);
  assert.equal(app.includes("activeWorkspace"), false);
  assert.equal(app.includes("className=\"workspace-switcher\""), false);
  assert.equal(app.includes("className=\"workspace-tab"), false);
  assert.equal(styles.includes(".workspace-switcher"), false);
  assert.equal(styles.includes(".workspace-tab"), false);
  assert.match(app, /className="connection-menu"/);
  assert.match(app, /className="connection-trigger"/);
  assert.match(app, /className="connection-scope-grid"/);
  assert.match(app, /useConnectionDiagnostics/);
  assert.match(app, /connection-diagnostics-action/);
  assert.match(app, /connection-diagnostics-list/);
  assert.match(readExistingFile(connectionDiagnosticsHookUrl), /async function run\(\)/);
  assert.match(app, /className="scope-values"/);
  assert.equal(app.includes("className=\"scope-inputs\""), false);
  assert.match(app, /const \[connectionMenuOpen, setConnectionMenuOpen\] = useState\(false\)/);
  assert.match(app, /setConnectionMenuOpen\(false\)/);
  assert.match(app, /<details className="connection-menu"[\s\S]*open=\{connectionMenuOpen\}/);
  assert.match(app, /event\.preventDefault\(\)/);
  assert.match(app, /setConnectionMenuOpen\(\(open\) => !open\)/);
  assert.match(app, /onToggle=\{\(event\) => setConnectionMenuOpen\(event\.currentTarget\.open\)\}/);
  assert.match(styles, /\.connection-popover\s*\{[^}]*box-shadow:\s*var\(--shadow-pop\);/s);
  assert.match(styles, /\.connection-diagnostics-action\s*\{[^}]*width:\s*100%;/s);
  assert.match(styles, /\.connection-diagnostics-list\s*\{/);
  assert.match(styles, /\.scope-values span\s*\{[^}]*background:\s*var\(--surface-raised\);/s);
  assert.match(styles, /\.connection-menu:not\(\[open\]\)\s+\.connection-popover\s*\{[^}]*display:\s*none;/s);
});

test("workspace telemetry is scoped to system check instead of repeating on every workspace", () => {
  assert.match(app, /const showWorkspaceTelemetry = activeView\.key === "cockpit";/);
  assert.match(app, /className="system-check-context"/);
  assert.match(app, /className="system-check-context-main"/);
  assert.match(app, /className="system-check-signals"/);
  assert.equal(app.includes("<MetricCard"), false);
  assert.match(styles, /\.system-check-context\s*\{[^}]*background:\s*var\(--surface\);/s);
  assert.match(styles, /\.system-check-signals\s*\{[^}]*grid-template-columns:\s*repeat\(4,\s*minmax\(0,\s*1fr\)\);/s);
  assert.doesNotMatch(app, /isCapabilitiesView \? "compact" : ""/);
});

test("agent tools workspace balances registry layout and hides inactive empty-state actions", () => {
  const registryStart = consoleViews.indexOf("export function RegistryView");
  const routesStart = consoleViews.indexOf("export function RoutesView", registryStart);
  const registryView = consoleViews.slice(registryStart, routesStart);

  assert.notEqual(registryStart, -1);
  assert.notEqual(routesStart, -1);
  assert.match(app, /from "\.\/resourceLifecycle"/);
  assert.match(app, /from "\.\/components\/ResourceLifecycleView"/);
  assert.match(consoleViews, /resourceLifecyclePanel/);
  assert.match(app, /resourceLifecyclePanel=\{resourceLifecyclePanel\(\)\}/);
  assert.match(resourceLifecycleView, /primaryActions/);
  assert.match(resourceLifecycleView, /secondaryActions/);
  assert.match(resourceLifecycleView, /useState/);
  assert.match(resourceLifecycleView, /selectedResourceId/);
  assert.match(resourceLifecycleView, /defaultSelectedItem/);
  assert.match(resourceLifecycleView, /formatTenantName/);
  assert.match(resourceLifecycleView, /formatWorkspaceName/);
  assert.match(resourceLifecycleView, /resource-lifecycle-context-panel/);
  assert.match(resourceLifecycleView, /resource-lifecycle-resource-button/);
  assert.match(resourceLifecycleView, /resource-lifecycle-row\$\{selected \? " is-selected" : ""\}/);
  assert.match(app, /const resourceLifecyclePrimaryActions = \(\s*<div className="resource-lifecycle-command-actions">/);
  assert.match(app, /const resourceLifecycleSecondaryActions = \(\s*<div className="resource-lifecycle-secondary-actions">/);
  assert.match(app, /<ResourceLifecycleView[\s\S]*formatTenantName=\{\(tenantId\) => permissionTenantPathLabel\(tenantId, tenants, t\)\.primary\}[\s\S]*formatWorkspaceName=\{\(workspaceId\) => permissionWorkspaceDisplayName\(workspaceId, agents, t\)\}/);
  assert.match(app, /<ResourceLifecycleView[\s\S]*primaryActions=\{resourceLifecyclePrimaryActions\}[\s\S]*secondaryActions=\{resourceLifecycleSecondaryActions\}/);
  assert.match(consolePrimitives, /tone = "secondary"/);
  assert.match(consolePrimitives, /tone\?: "primary" \| "secondary"/);
  assert.match(app, /createAgentAction\("command", "primary"\)/);
  assert.match(app, /createKeyAction\("command", "secondary"\)/);
  assert.match(app, /rotateCredentialAction\("command", "secondary"\)/);
  assert.match(app, /createPolicyAction\("command", "secondary"\)/);
  assert.match(app, /agentRegistryPanel=\{agentRegistryPanel\("span-8"\)\}/);
  assert.doesNotMatch(app, /const agentRegistryActions =/);
  assert.match(app, /contractMatrixPanel=\{contractMatrixPanel\("span-4"\)\}/);
  assert.match(styles, /\.content-grid\s*\{[^}]*align-items:\s*start;/s);
  assert.match(styles, /\.resource-lifecycle\s*\{/);
  assert.match(styles, /\.resource-lifecycle-command-center\s*\{/);
  assert.match(styles, /\.resource-lifecycle-context-panel\s*\{/);
  assert.match(styles, /\.resource-lifecycle-context-grid\s*\{/);
  assert.match(styles, /\.resource-lifecycle-resource-button\s*\{/);
  assert.match(styles, /\.resource-lifecycle-row\.is-selected\s*\{/);
  assert.match(styles, /\.action-modal-trigger-command\.is-primary\s*\{/);
  assert.match(styles, /\.action-modal-trigger-command\.is-secondary\s*\{/);
  assert.match(styles, /\.action-modal-body \.control-form\s*\{/);
  assert.doesNotMatch(styles, /\.action-modal-trigger-command\s*\{[^}]*box-shadow:\s*var\(--shadow-card\);/s);
  assert.match(resourceLifecycleView, /summary\.items\.map/);
  assert.match(resourceLifecycleView, /onResourceAction/);
  assert.match(resourceLifecycleView, /onSelectResource/);
  assert.match(resourceLifecycleView, /type="button"[\s\S]*onClick=\{\(\) => onResourceAction\(item\)\}/);
  assert.match(resourceLifecycleView, /t\(item\.detailKey\)/);
  assert.doesNotMatch(resourceLifecycleView, /ManagementForms/);
  assert.doesNotMatch(resourceLifecycleView, /TechnicalId/);
  assert.match(consolePrimitives, /openToken\?: number/);
  assert.match(consolePrimitives, /useEffect\(\(\) => \{[\s\S]*if \(openToken === undefined\) return;[\s\S]*setOpen\(true\)/);
  assert.match(app, /resourceActionModal/);
  assert.match(app, /handleResourceLifecycleAction/);
  assert.match(app, /planResourceLifecycleAction/);
  assert.match(resourceLifecycleActionPlanner, /export function planResourceLifecycleAction/);
  assert.doesNotMatch(app, /sameScopeCaller|sameScopeTarget|resource\.permissionIntent/);
  assert.match(app, /openToken=\{resourceActionModal === "rotate_credential" \? resourceActionOpenToken : undefined\}/);
  assert.doesNotMatch(registryView, /createAgentPanel|createKeyPanel|rotateCredentialPanel/);
  assert.doesNotMatch(app, /RoutesView[\s\S]*routeGovernancePanel=\{routeGovernancePanel\("span-12", createPolicyAction\(\)\)\}/);
  assert.doesNotMatch(app, /PoliciesView[\s\S]*routeGovernancePanel=\{routeGovernancePanel\("span-12", createPolicyAction\(\)\)\}/);
  assert.match(operationalViews, /href="#registry"/);
  assert.doesNotMatch(operationalViews, /querySelector<HTMLButtonElement>\("#policy-create-panel/);
});

test("production journey checkpoint stays compact and model-driven", () => {
  assert.match(productionJourney, /export function deriveProductionJourney/);
  assert.match(productionJourney, /export const productionJourneyStages/);
  assert.match(productionJourneyCheckpoint, /export function ProductionJourneyCheckpoint/);
  assert.match(productionJourneyCheckpoint, /productionJourneyStages\.map/);
  assert.match(productionJourneyCheckpoint, /className="production-journey-checkpoint"/);
  assert.match(productionJourneyCheckpoint, /href=\{journey\.nextActionHash\}/);
  assert.doesNotMatch(productionJourneyCheckpoint, /useState|useEffect|fetch\(/);
  assert.match(styles, /\.production-journey-checkpoint\s*\{/);
  assert.match(styles, /\.production-journey-checkpoint\s*\{[^}]*display:\s*grid;/s);
  assert.doesNotMatch(styles, /\.production-journey-checkpoint\s*\{[^}]*box-shadow:/s);
});

test("production journey checkpoint is wired through primary journey views", () => {
  assert.match(app, /from "\.\/productionJourney"/);
  assert.match(app, /from "\.\/components\/ProductionJourneyCheckpoint"/);
  assert.match(app, /deriveProductionJourney\(/);
  assert.match(app, /const productionJourneyCheckpoint = \(/);
  assert.match(consoleViews, /journeyCheckpoint/);
  assert.match(consoleViews, /GettingStartedConsoleView[\s\S]*journeyCheckpoint/);
  assert.match(consoleViews, /AskView[\s\S]*journeyCheckpoint/);
  assert.match(consoleViews, /RegistryView[\s\S]*journeyCheckpoint/);
  assert.match(consoleViews, /AiAdminView[\s\S]*journeyCheckpoint/);
  assert.match(consoleViews, /EvidenceView[\s\S]*journeyCheckpoint/);
});

test("management mutation forms open from resource command modals", () => {
  assert.match(consolePrimitives, /export function ActionModalButton/);
  assert.doesNotMatch(consolePrimitives, /export function ActionModalPanel/);
  assert.match(consolePrimitives, /aria-haspopup="dialog"/);
  assert.match(consolePrimitives, /aria-label=\{`\$\{title\} \$\{openLabel\}`\}/);
  assert.match(consolePrimitives, /aria-modal="true"/);
  assert.match(consolePrimitives, /role="dialog"/);
  assert.match(consolePrimitives, /event\.key === "Escape"/);
  assert.match(consolePrimitives, /document\.body\.style\.overflow = "hidden"/);
  assert.match(consolePrimitives, /variant = "compact"/);
  assert.match(consolePrimitives, /action-modal-trigger-command/);
  assert.match(consolePrimitives, /"action-modal-trigger action-modal-trigger-compact"/);
  assert.match(consolePrimitives, /triggerClassName=\{triggerClassName\}/);
  assert.match(app, /<ActionModalButton[\s\S]*title=\{t\("panel\.createAgent"\)\}/);
  assert.match(app, /<ActionModalButton[\s\S]*title=\{t\("panel\.createKey"\)\}/);
  assert.match(app, /<ActionModalButton[\s\S]*id="policy-create-panel"[\s\S]*title=\{t\("panel\.createPolicy"\)\}/);
  assert.match(app, /<ActionModalButton[\s\S]*title=\{t\("panel\.rotateCredential"\)\}/);
  assert.match(app, /ResourceLifecycleActionContext/);
  assert.match(app, /context: ResourceLifecycleActionContext \| null/);
  assert.match(app, /const resourceActionContext = resourceActionRequest\.context/);
  assert.match(app, /openResourceActionModal\(plan\.modal, plan\.context\)/);
  assert.match(app, /management\.setKeyForm\(\{[\s\S]*agentId: plan\.agentId/s);
  assert.match(app, /management\.setRotateForm\(\{[\s\S]*agentId: plan\.agentId/s);
  assert.match(app, /management\.setPolicyForm\(\{[\s\S]*callerAgentId: plan\.callerAgentId/s);
  assert.match(app, /management\.setPolicyForm\(\{[\s\S]*targetAgentId: plan\.targetAgentId/s);
  assert.match(app, /<KeyCreateForm[\s\S]*context=\{resourceActionContext\}/);
  assert.match(app, /<CredentialRotateForm[\s\S]*context=\{resourceActionContext\}/);
  assert.match(app, /<PolicyCreateForm[\s\S]*context=\{resourceActionContext\}/);
  assert.match(managementForms, /ResourceActionContextStrip/);
  assert.match(managementForms, /className="resource-action-context"/);
  assert.match(managementForms, /t\("resource\.actionContext\.title"\)/);
  assert.match(styles, /\.resource-action-context\s*\{/);
  assert.match(app, /createAgentAction\("command", "primary"\)/);
  assert.match(app, /createKeyAction\("command", "secondary"\)/);
  assert.match(app, /rotateCredentialAction\("command", "secondary"\)/);
  assert.match(app, /createPolicyAction\("command", "secondary"\)/);
  assert.match(app, /routeGovernancePanel=\{routeGovernancePanel\("span-12"\)\}/);
  assert.match(app, /routeGovernancePanel=\{routeGovernancePanel\("span-12"\)\}[\s\S]*t=\{t\}/);
  assert.match(operationalViews, /href="#registry"/);
  assert.doesNotMatch(operationalViews, /querySelector<HTMLButtonElement>\("#policy-create-panel \.action-modal-trigger"\)/);
  assert.match(styles, /\.panel-action-group\s*\{/);
  assert.match(styles, /\.action-modal-trigger\s*\{[\s\S]*cursor:\s*pointer;/);
  assert.match(styles, /\.action-modal-trigger-compact\s*\{[\s\S]*width:\s*auto;/);
  assert.match(styles, /\.action-modal-backdrop\s*\{[\s\S]*position:\s*fixed;[\s\S]*overscroll-behavior:\s*contain;/);
  assert.match(styles, /\.action-modal-panel\s*\{[\s\S]*width:\s*min\(680px,\s*calc\(100vw - 48px\)\);/);
  assert.doesNotMatch(consoleViews, /createPolicyPanel=\{createPolicyPanel\}/);
  assert.doesNotMatch(styles, /\.action-modal-entry/);
  assert.doesNotMatch(styles, /\.action-disclosure-panel/);
});

test("agent registry provides search status filtering and a details entry", () => {
  const agentTableStart = operationalViews.indexOf("export function AgentTable");
  const contractMatrixStart = operationalViews.indexOf("export function ContractMatrix", agentTableStart);
  const agentTable = operationalViews.slice(agentTableStart, contractMatrixStart);

  assert.match(operationalViews, /const \[agentQuery, setAgentQuery\] = useState\(""\)/);
  assert.match(operationalViews, /const \[agentStatusFilter, setAgentStatusFilter\] = useState<AgentStatus \| "">\(""\)/);
  assert.match(operationalViews, /const \[selectedAgentId, setSelectedAgentId\] = useState\(""\)/);
  assert.match(operationalViews, /const hasAgents = agents\.length > 0/);
  assert.match(operationalViews, /const visibleAgents = agents\.filter/);
  assert.match(operationalViews, /className="table-toolbar"/);
  assert.match(operationalViews, /className="registry-empty-state"/);
  assert.doesNotMatch(agentTable, /<td colSpan=\{6\}>/);
  assert.match(operationalViews, /placeholder=\{t\("form\.searchAgents"\)\}/);
  assert.match(operationalViews, /className="table-detail-panel"/);
  assert.match(operationalViews, /setSelectedAgentId\(agent\.id\)/);
  assert.match(styles, /\.table-toolbar\s*\{/);
  assert.match(styles, /\.registry-empty-state\s*\{/);
  assert.match(styles, /\.table-detail-panel\s*\{/);
});

test("table actions distinguish neutral state changes from destructive actions", () => {
  assert.match(operationalViews, /className="table-action is-danger"[\s\S]*onClick=\{\(\) => onDisable\(policy\)\}/);
  assert.match(operationalViews, /className="table-action"[\s\S]*onClick=\{\(\) => onStatusChange\(agent, agent\.status === "active" \? "draft" : "active"\)\}/);
  assert.match(operationalViews, /className="table-action is-danger"[\s\S]*onClick=\{\(\) => onStatusChange\(agent, "disabled"\)\}/);
  assert.match(styles, /\.table-action\.is-danger\s*\{[^}]*background:\s*var\(--danger-soft\);/s);
});

test("technical identifiers use a readable copyable component in dense workspaces", () => {
  assert.match(technicalId, /export function shortTechnicalId\(value: string\)/);
  assert.match(technicalId, /export function TechnicalId/);
  assert.match(technicalId, /className="technical-id"/);
  assert.match(technicalId, /navigator\.clipboard\?\.writeText\(value\)/);
  assert.match(app, /import \{ TechnicalId \} from "\.\/components\/TechnicalId"/);
  assert.match(operationalViews, /<TechnicalId copyLabel=\{t\("action\.copy"\)\} value=\{agent\.id\} \/>/);
  assert.match(runtimeEvidenceViews, /<TechnicalId copyLabel=\{t\("action\.copy"\)\} label=\{t\("form\.capability"\)\} value=\{trace\.capabilityId\} \/>/);
  assert.match(styles, /\.technical-id\s*\{/);
  assert.match(styles, /\.technical-id code\s*\{[^}]*font-family:\s*var\(--mono-font\);/s);
});

test("sidebar navigation shows grouped task labels with descriptions", () => {
  assert.match(app, /navGroups\.map/);
  assert.match(app, /className="nav-group"/);
  assert.match(app, /aria-current=\{activeView\.key === item\.key \? "page" : undefined\}/);
  assert.match(app, /<small>\{itemDetail\}<\/small>/);
  assert.match(styles, /\.nav-list\s*\{[^}]*gap:\s*16px;/s);
  assert.match(styles, /\.nav-item\s*\{[^}]*grid-template-columns:\s*22px minmax\(0,\s*1fr\);/s);
  assert.match(styles, /\.nav-item small\s*\{[^}]*font-size:\s*11px;/s);
});

test("tenant organization workspace is a first-class resource entry", () => {
  assert.match(app, /import \{ TenantOrganizationView, type TenantWorkspaceContext \} from "\.\/components\/TenantOrganizationView"/);
  assert.match(app, /case "tenants":/);
  assert.match(app, /<TenantsView tenantOrganizationPanel=\{tenantOrganizationPanel\} \/>/);
  assert.match(app, /function openTenantPermissionChange\(context: PermissionChangeHandoffContext\)/);
  assert.match(app, /function openTenantAccessProfile\(context: TenantWorkspaceContext\)/);
  assert.match(app, /accessSubjects=\{aiAdminAccessSubjects\}/);
  assert.match(consoleViews, /export function TenantsView/);
  assert.match(tenantOrganizationView, /buildTenantOrganizationModel/);
  assert.match(tenantOrganizationView, /buildTenantPermissionCenterViewModel/);
  assert.match(tenantOrganizationView, /className="tenant-organization content-grid"/);
  assert.match(tenantOrganizationView, /className="primary-button"/);
  assert.match(tenantOrganizationView, /tenant-permission-modal/);
  assert.match(tenantOrganizationView, /role="dialog"/);
  assert.match(tenantOrganizationView, /aria-haspopup="dialog"/);
  assert.match(tenantOrganizationView, /tenantCenter\.snapshot/);
  assert.match(tenantOrganizationView, /tenantCenter\.adminBoundary/);
  assert.match(tenantOrganizationView, /tenantCenter\.capabilityDetail/);
  assert.match(tenantOrganizationView, /tenantCenter\.dataScopes/);
  assert.match(tenantOrganizationView, /centerViewModel\.capabilitySummaries/);
  assert.match(tenantOrganizationView, /className="tenant-access-directory"/);
  assert.match(app, /selectedTenantId=\{tenantOrganizationEffectiveTenantId\}/);
  assert.match(tenantOrganizationView, /href="#admin-access"/);
  assert.match(app, /fetchTenantPermissionCenter/);
  assert.match(tenantOrganizationView, /accessSubjectsForWorkspace/);
  assert.match(tenantOrganizationView, /ApprovalDropdown/);
  assert.match(tenantOrganizationView, /sourceView: "tenants"/);
  assert.match(styles, /\.tenant-organization\s*\{/);
  assert.match(styles, /\.tenant-center-panel\s*\{/);
  assert.match(styles, /\.tenant-center-metrics\s*\{/);
  assert.match(styles, /\.tenant-center-scope\s*\{/);
  assert.match(styles, /\.tenant-center-capability-list\s*\{/);
  assert.match(styles, /\.tenant-center-scope-tags\s*\{/);
  assert.match(styles, /\.tenant-center-actions\s*\{/);
  assert.match(styles, /\.tenant-permission-modal\s*\{/);
  assert.match(styles, /\.tenant-access-directory\s*\{/);
  assert.match(styles, /\.tenant-org-actions\s*\{[^}]*background:\s*var\(--brand-soft\);/s);
  assert.match(styles, /\.tenant-tree-row\.is-selected\s*\{[^}]*background:\s*var\(--brand-soft\);/s);
});

test("desktop sidebar keeps text labels at review viewport widths", () => {
  assert.match(styles, /\.app-shell\s*\{[^}]*grid-template-columns:\s*220px minmax\(0,\s*1fr\);/s);
  assert.match(styles, /@media \(max-width: 1120px\)\s*\{[\s\S]*\.app-shell\s*\{[^}]*grid-template-columns:\s*200px minmax\(0,\s*1fr\);/s);
  assert.doesNotMatch(styles, /@media \(max-width: 1120px\)\s*\{[\s\S]*\.nav-item span\s*\{[^}]*display:\s*none;/s);
  assert.match(styles, /\.section-kicker\s*\{[^}]*white-space:\s*nowrap;/s);
});

test("workspace exposes an initial loading state before data is resolved", () => {
  assert.match(app, /aria-busy=\{!data\}/);
  assert.match(app, /className="workspace-loading"/);
  assert.match(styles, /\.workspace-loading\s*\{/);
  assert.match(styles, /\.workspace-loading-skeleton\s*\{/);
});

test("empty states use a consistent production component instead of single-line placeholders", () => {
  assert.match(ui, /import \{ Inbox \} from "lucide-react"/);
  assert.match(ui, /className="empty-row-icon"/);
  assert.match(ui, /className="empty-row-copy"/);
  assert.match(styles, /\.empty-row\s*\{[^}]*grid-template-columns:\s*36px minmax\(0,\s*380px\);/s);
  assert.match(styles, /\.empty-row\s*\{[^}]*min-height:\s*128px;/s);
  assert.match(styles, /\.empty-row\s*\{[^}]*padding:\s*var\(--space-5\);/s);
  assert.match(styles, /\.empty-row-icon\s*\{[^}]*border:\s*1px solid var\(--line-muted\);/s);
  assert.match(styles, /\.empty-row strong\s*\{[^}]*font-weight:\s*600;/s);
  assert.match(styles, /\.empty-row span\s*\{[^}]*line-height:\s*18px;/s);
  assert.match(styles, /\.access-decision-explain \.empty-row\s*\{[^}]*justify-content:\s*start;/s);
});

test("approval messages expose semantic status styling", () => {
  assert.match(workbench, /approval-inline-message status-\$\{messageTone\}/);
  assert.match(workbench, /approval-inline-message status-\$\{approvalReadinessMessageTone\}/);
  assert.match(styles, /\.approval-inline-message\.status-error\s*\{/);
  assert.match(styles, /\.approval-inline-message\.status-success\s*\{/);
});

test("removed demo-era selectors stay out of the production shell", () => {
  for (const selector of [".permission-package-grid", ".approval-check", ".approval-checklist", ".access-toolbar"]) {
    assert.equal(styles.includes(selector), false, `${selector} should not be present`);
  }
});

test("mobile shell keeps navigation labels readable", () => {
  assert.match(styles, /@media \(max-width: 760px\)\s*\{[\s\S]*\.nav-item\s*\{[^}]*min-width:\s*max-content;/s);
  assert.match(styles, /@media \(max-width: 760px\)\s*\{[\s\S]*\.nav-item span\s*\{[^}]*display:\s*block;/s);
  assert.match(styles, /@media \(max-width: 760px\)\s*\{[\s\S]*\.nav-item small\s*\{[^}]*display:\s*none;/s);
  assert.match(styles, /@media \(max-width: 760px\)\s*\{[\s\S]*\.connection-popover\s*\{[^}]*position:\s*fixed;/s);
});

test("capability workspace compresses metrics and opens grant operations on demand", () => {
  assert.match(app, /<CapabilitiesView capabilityGovernancePanel=\{capabilityGovernancePanel\(\)\} \/>/);
  assert.match(consoleViews, /export function CapabilitiesView[\s\S]*<section className="content-grid">[\s\S]*\{capabilityGovernancePanel\}/);
  assert.match(styles, /\.capability-layout\s*\{[^}]*grid-template-areas:\s*"catalog assignments";/s);
  assert.match(capabilityGovernanceView, /className="primary-button capability-grant-launcher"/);
  assert.match(capabilityGovernanceView, /className="capability-grant-sheet"/);
});

test("capability catalog provides search status filtering and a details entry", () => {
  assert.match(capabilityGovernanceView, /const \[capabilityQuery, setCapabilityQuery\] = useState\(""\)/);
  assert.match(capabilityGovernanceView, /const \[capabilityStatusFilter, setCapabilityStatusFilter\] = useState\(""\)/);
  assert.match(capabilityGovernanceView, /const \[selectedCapabilityId, setSelectedCapabilityId\] = useState\(""\)/);
  assert.match(capabilityGovernanceView, /const visibleCapabilities = useMemo/);
  assert.match(capabilityGovernanceView, /placeholder=\{t\("form\.searchCapabilities"\)\}/);
  assert.match(capabilityGovernanceView, /className="capability-detail-panel"/);
  assert.match(capabilityGovernanceView, /setSelectedCapabilityId\(capability\.id\)/);
  assert.match(styles, /\.capability-detail-panel\s*\{/);
});

test("access policy page routes creation back to resource management", () => {
  const policiesStart = consoleViews.indexOf("export function PoliciesView");
  const capabilitiesStart = consoleViews.indexOf("export function CapabilitiesView", policiesStart);
  const policiesView = consoleViews.slice(policiesStart, capabilitiesStart);

  assert.match(operationalViews, /function AccessPolicyWorkspace/);
  assert.match(policiesView, /<AccessPolicyWorkspace/);
  assert.match(operationalViews, /className="policy-empty-action"/);
  assert.match(operationalViews, /href="#registry"/);
  assert.match(operationalViews, /t\("action\.openResourceManagement"\)/);
  assert.match(operationalViews, /auditCollapsed/);
  assert.doesNotMatch(app, /\{managementAuditPanel\("span-5"\)\}/);
  assert.match(styles, /\.policy-workspace\s*\{/);
  assert.match(styles, /\.policy-empty-action\s*\{/);
});

test("operational list components are split from the app shell", () => {
  assert.match(app, /from "\.\/components\/OperationalViews"/);
  assert.match(app, /from "\.\/components\/ConsoleViews"/);
  assert.doesNotMatch(app, /function AgentTable/);
  assert.doesNotMatch(app, /function PolicyTable/);
  assert.doesNotMatch(app, /function ContractMatrix/);
  assert.doesNotMatch(app, /function AccessPolicyWorkspace/);
  assert.match(consoleViews, /export function RegistryView/);
  assert.match(consoleViews, /export function PoliciesView/);
  assert.match(operationalViews, /export function AgentTable/);
  assert.match(operationalViews, /export function PolicyTable/);
  assert.match(operationalViews, /export function ContractMatrix/);
  assert.match(operationalViews, /export function AccessPolicyWorkspace/);
});

test("management forms and console primitives are split from the app shell", () => {
  assert.match(app, /from "\.\/components\/ManagementForms"/);
  assert.match(app, /from "\.\/components\/ConsolePrimitives"/);
  assert.doesNotMatch(app, /function AgentCreateForm/);
  assert.doesNotMatch(app, /function KeyCreateForm/);
  assert.doesNotMatch(app, /function CredentialRotateForm/);
  assert.doesNotMatch(app, /function PolicyCreateForm/);
  assert.doesNotMatch(app, /function TraceFilterBar/);
  assert.doesNotMatch(app, /function MetricCard/);
  assert.doesNotMatch(app, /function Panel/);
  assert.match(managementForms, /export function AgentCreateForm/);
  assert.match(managementForms, /export function KeyCreateForm/);
  assert.match(managementForms, /export function CredentialRotateForm/);
  assert.match(managementForms, /export function PolicyCreateForm/);
  assert.match(managementForms, /export function TraceFilterBar/);
  assert.match(consolePrimitives, /export function MetricCard/);
  assert.match(consolePrimitives, /export function Panel/);
});

test("runtime evidence views are split from the app shell", () => {
  assert.match(app, /from "\.\/components\/RuntimeEvidenceViews"/);
  assert.doesNotMatch(app, /function EvidenceTimeline/);
  assert.doesNotMatch(app, /function SignalBoard/);
  assert.doesNotMatch(app, /function TraceTable/);
  assert.doesNotMatch(app, /function ManagementAuditTable/);
  assert.doesNotMatch(app, /function auditCredentialVersion/);
  assert.doesNotMatch(app, /function metricRatio/);
  assert.match(runtimeEvidenceViews, /export function EvidenceTimeline/);
  assert.match(runtimeEvidenceViews, /export function SignalBoard/);
  assert.match(runtimeEvidenceViews, /export function TraceTable/);
  assert.match(runtimeEvidenceViews, /export function ManagementAuditTable/);
  assert.match(consolePrimitives, /export function IconMore/);
  assert.match(consolePrimitives, /export function IconOpen/);
});

test("go-live acceptance overview is split from the app shell", () => {
  assert.match(app, /from "\.\/components\/GoLiveAcceptanceOverview"/);
  assert.doesNotMatch(app, /function GoLiveAcceptanceOverview/);
  assert.doesNotMatch(app, /function productionReadinessStatusLabel/);
  assert.doesNotMatch(app, /function permissionProductionReadinessNextAction/);
  assert.match(goLiveAcceptanceOverview, /from "\.\.\/productionAcceptance"/);
  assert.match(productionAcceptance, /export function buildProductionAcceptanceCenter/);
  assert.match(productionAcceptance, /export type ProductionAcceptanceStatus/);
  assert.match(goLiveAcceptanceOverview, /export function GoLiveAcceptanceOverview/);
  assert.match(goLiveAcceptanceOverview, /className="go-live-acceptance"/);
});
