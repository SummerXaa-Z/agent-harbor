import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const baseStyles = readFileSync(new URL("../src/styles.css", import.meta.url), "utf8");
const workbenchStyles = readFileSync(new URL("../src/styles/permission-workbench.css", import.meta.url), "utf8");
const styles = `${baseStyles}\n${workbenchStyles}`;
const app = readFileSync(new URL("../src/App.tsx", import.meta.url), "utf8");
const workbench = readFileSync(new URL("../src/components/AiAdminPermissionWorkbench.tsx", import.meta.url), "utf8");
const dropdown = readFileSync(new URL("../src/components/ApprovalDropdown.tsx", import.meta.url), "utf8");
const technicalId = readFileSync(new URL("../src/components/TechnicalId.tsx", import.meta.url), "utf8");
const ui = readFileSync(new URL("../src/components/ui.tsx", import.meta.url), "utf8");

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
  assert.match(styles, /\.table-action\s*\{[^}]*min-height:\s*var\(--control-height-compact\);/s);
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
  assert.match(app, /const \[connectionMenuOpen, setConnectionMenuOpen\] = useState\(false\)/);
  assert.match(app, /setConnectionMenuOpen\(false\)/);
  assert.match(app, /<details className="connection-menu"[\s\S]*open=\{connectionMenuOpen\}/);
  assert.match(app, /event\.preventDefault\(\)/);
  assert.match(app, /setConnectionMenuOpen\(\(open\) => !open\)/);
  assert.match(app, /onToggle=\{\(event\) => setConnectionMenuOpen\(event\.currentTarget\.open\)\}/);
  assert.match(styles, /\.connection-popover\s*\{[^}]*box-shadow:\s*var\(--shadow-pop\);/s);
  assert.match(styles, /\.connection-menu:not\(\[open\]\)\s+\.connection-popover\s*\{[^}]*display:\s*none;/s);
});

test("workspace telemetry is scoped to system check instead of repeating on every workspace", () => {
  assert.match(app, /const showWorkspaceTelemetry = activeView\.key === "cockpit";/);
  assert.doesNotMatch(app, /isCapabilitiesView \? "compact" : ""/);
});

test("agent tools workspace prioritizes the registry before mutation forms", () => {
  const viewSwitchStart = app.indexOf("const viewContent = (() => {");
  const registryStart = app.indexOf('case "registry":', viewSwitchStart);
  const routesStart = app.indexOf('case "routes":', registryStart);
  const registryView = app.slice(registryStart, routesStart);

  assert.notEqual(viewSwitchStart, -1);
  assert.notEqual(registryStart, -1);
  assert.notEqual(routesStart, -1);
  assert.ok(registryView.indexOf('{agentRegistryPanel("span-12")}') < registryView.indexOf("{createAgentPanel}"));
  assert.ok(registryView.indexOf("{createAgentPanel}") < registryView.indexOf("{createKeyPanel}"));
  assert.ok(registryView.indexOf("{createKeyPanel}") < registryView.indexOf("{rotateCredentialPanel}"));
});

test("technical identifiers use a readable copyable component in dense workspaces", () => {
  assert.match(technicalId, /export function shortTechnicalId\(value: string\)/);
  assert.match(technicalId, /export function TechnicalId/);
  assert.match(technicalId, /className="technical-id"/);
  assert.match(technicalId, /navigator\.clipboard\?\.writeText\(value\)/);
  assert.match(app, /import \{ TechnicalId \} from "\.\/components\/TechnicalId"/);
  assert.match(app, /<TechnicalId copyLabel=\{t\("action\.copy"\)\} value=\{agent\.id\} \/>/);
  assert.match(app, /<TechnicalId copyLabel=\{t\("action\.copy"\)\} label=\{t\("form\.capability"\)\} value=\{trace\.capabilityId\} \/>/);
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

test("capability workspace compresses metrics and prioritizes grant operations", () => {
  assert.match(app, /case "capabilities":\s*return \(\s*<section className="content-grid">\s*\{capabilityGovernancePanel\(\)\}/s);
  assert.match(styles, /\.capability-layout\s*\{[^}]*grid-template-areas:\s*"grant catalog"[\s\S]*"assignments catalog";/s);
});
