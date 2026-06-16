import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import { translationKeys } from "../src/i18n.ts";

const capabilityView = readFileSync(new URL("../src/components/CapabilityGovernanceView.tsx", import.meta.url), "utf8");
const controller = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const styles = readFileSync(new URL("../src/styles.css", import.meta.url), "utf8");

test("capability governance page prioritizes catalog and opens grant creation on demand", () => {
  assert.match(capabilityView, /className="capability-scope-bar"/);
  assert.match(capabilityView, /className="capability-catalog-heading"/);
  assert.match(capabilityView, /className="primary-button capability-grant-launcher"/);
  assert.match(capabilityView, /const \[grantPanelOpen, setGrantPanelOpen\] = useState\(false\)/);
  assert.match(capabilityView, /className="capability-grant-sheet"/);
  assert.match(capabilityView, /className="assignment-list-heading"/);
  assert.ok(
    capabilityView.indexOf('className="capability-catalog"') < capabilityView.indexOf('className="assignment-list"'),
    "capability catalog should be rendered before grant records"
  );
  assert.match(styles, /\.capability-layout\s*\{[^}]*grid-template-areas:\s*"catalog assignments";/s);
  assert.match(styles, /@media \(max-width: 1120px\)\s*\{[\s\S]*\.capability-layout\s*\{[^}]*grid-template-areas:\s*"catalog"[\s\S]*"assignments";/s);
  assert.doesNotMatch(styles, /grid-template-areas:\s*"grant"[\s\S]*"catalog"/);
});

test("capability governance copy is bilingual", () => {
  const english = new Set(translationKeys("en"));
  const chinese = new Set(translationKeys("zh-CN"));
  for (const key of [
    "section.capabilityCatalog",
    "section.capabilityGrant",
    "section.currentCapabilityScope",
    "section.existingGrantChains",
    "text.capabilityCatalogHelp",
    "text.capabilityGrantHelp",
    "text.currentCapabilityScopeDetail",
    "message.grantChainCreatedRefreshFailed"
  ]) {
    assert.equal(english.has(key), true, `${key} missing in English`);
    assert.equal(chinese.has(key), true, `${key} missing in zh-CN`);
  }
});

test("navigation descriptions stay compact by default", () => {
  assert.match(styles, /\.nav-item small\s*\{[^}]*display:\s*none;/s);
  assert.match(styles, /\.nav-item\.active small\s*\{[^}]*display:\s*-webkit-box;/s);
  assert.match(styles, /@media \(max-width: 760px\)\s*\{[\s\S]*\.nav-item small\s*\{[^}]*display:\s*none;/s);
});

test("capability panel avoids protocol-first title copy", () => {
  assert.match(controller, /title=\{t\("panel\.mcpCapabilities"\)\}/);
});
