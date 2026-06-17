import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const ui = readFileSync(new URL("../src/components/ui.tsx", import.meta.url), "utf8");
const operationalViews = readFileSync(new URL("../src/components/OperationalViews.tsx", import.meta.url), "utf8");
const capabilityGovernance = readFileSync(new URL("../src/components/CapabilityGovernanceView.tsx", import.meta.url), "utf8");
const accessProfile = readFileSync(new URL("../src/components/TenantAccessProfileView.tsx", import.meta.url), "utf8");
const runtimeEvidence = readFileSync(new URL("../src/components/RuntimeEvidenceViews.tsx", import.meta.url), "utf8");
const i18n = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");

test("empty row supports optional guidance actions without replacing the empty state", () => {
  assert.match(ui, /actionLabel\?: string/);
  assert.match(ui, /actionHash\?: string/);
  assert.match(ui, /onAction\?: \(\) => void/);
  assert.match(ui, /className="secondary-button empty-row-action"/);
  assert.match(ui, /href=\{actionHash\}/);
  assert.match(ui, /onClick=\{onAction\}/);
});

test("setup-dependent empty states point to the next useful workspace", () => {
  assert.match(operationalViews, /empty\.registry\.detail/);
  assert.doesNotMatch(operationalViews, /empty\.registry\.action/);
  assert.doesNotMatch(operationalViews, /actionHash=\{hasAgents \? undefined : "#getting-started"\}/);

  assert.match(capabilityGovernance, /empty\.capabilities\.actionRegisterAgents/);
  assert.match(capabilityGovernance, /empty\.capabilities\.actionRefresh/);
  assert.match(capabilityGovernance, /actionHash=\{capabilityEmptyActionHash\}/);
  assert.match(capabilityGovernance, /onAction=\{capabilityEmptyAction\}/);
  assert.doesNotMatch(capabilityGovernance, /empty\.filteredResults\.action/);

  assert.match(accessProfile, /empty\.grantChains\.action/);
  assert.match(accessProfile, /actionHash="#ai-admin"/);
  assert.match(runtimeEvidence, /empty\.auditTraces\.action/);
  assert.match(runtimeEvidence, /actionHash="#getting-started"/);
});

test("empty state guidance copy is bilingual", () => {
 for (const key of [
    "empty.auditTraces.action",
    "empty.capabilities.actionRefresh",
    "empty.capabilities.actionRegisterAgents",
    "empty.grantChains.action",
    "empty.registry.detail"
  ]) {
    assert.match(i18n, new RegExp(`"${key}"`), `${key} should be present in i18n`);
  }
});
