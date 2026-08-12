import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const app = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const component = readFileSync(new URL("../src/components/AccessHandoffPanel.tsx", import.meta.url), "utf8");
const hook = readFileSync(new URL("../src/hooks/useAccessHandoffController.ts", import.meta.url), "utf8");
const i18n = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");
const permissionPackages = readFileSync(new URL("../src/permissionPackages.ts", import.meta.url), "utf8");
const styles = readFileSync(new URL("../src/styles.css", import.meta.url), "utf8");

test("go-live status lazy-loads access handoff as a first-class delivery section", () => {
  assert.match(app, /lazy\(\(\) => import\("\.\/components\/AccessHandoffPanel"\)/);
  assert.match(app, /activeNav === "go-live"/);
  assert.match(app, /<AccessHandoffPanel/);
  assert.match(component, /useAccessHandoffController\(/);
  assert.match(component, /className="access-handoff"/);
  assert.match(component, /handoff\.copyArtifacts\.mcpClientConfig/);
  assert.match(component, /handoff\.copyArtifacts\.promptTemplate/);
});

test("access handoff reloads from the immutable applied permission context", () => {
  assert.match(permissionPackages, /export function permissionPackageApplicationDraftInput/);
  assert.match(permissionPackages, /requestText: application\.requestText \?\? fallback\.requestText/);
  assert.match(permissionPackages, /subjectSelector: application\.subjectSelector \?\? fallback\.subjectSelector/);
  assert.match(app, /permissionPackageApplicationDraftInput\(aiAdminApplication, goLiveAcceptanceBaseForm\)/);
  assert.match(app, /aiAdminProductionReadiness\?\.runtimeEvidence\.allowedTrace\?\.subjectId/);
  assert.match(app, /aiAdminProductionReadiness\?\.runtimeEvidence\.deniedTrace\?\.subjectId/);
  assert.match(app, /filter=\{goLiveAccessHandoffFilter\}/);
  assert.match(app, /refreshAiAdminProductionReadiness\(goLiveAcceptanceForm, \{[\s\S]*subjectId: goLiveAcceptanceSubjectId/);
  assert.match(app, /exportAiAdminAcceptanceReport\([\s\S]*goLiveAcceptanceForm,[\s\S]*goLiveAcceptanceSubjectId/);
});

test("access handoff token lifecycle keeps plaintext ephemeral and revocation explicit", () => {
  assert.match(permissionPackages, /Omit<PermissionPackageProductionReadinessFilter, "traceLimit">/);
  assert.match(hook, /setOneTimeToken\(null\);[\s\S]*if \(!enabled/);
  assert.match(hook, /setOneTimeToken\(created\)/);
  assert.match(hook, /clearOneTimeToken: \(\) => setOneTimeToken\(null\)/);
  assert.match(component, /oneTimeToken\.key/);
  assert.match(component, /accessHandoff\.oneTimeTokenDetail/);
  assert.match(component, /window\.confirm\(t\("accessHandoff\.tokenRevokeConfirm"\)\)/);
  assert.match(component, /controller\.revokeToken\(id\)/);
  assert.match(i18n, /"accessHandoff\.oneTimeTokenDetail": "Copy it now\./);
  assert.match(i18n, /"accessHandoff\.oneTimeTokenDetail": "请立即复制。/);
});

test("blocked handoffs do not render copy artifacts or token creation controls", () => {
  const readyBlock = component.slice(component.indexOf("{handoff && ready ?"));
  assert.match(component, /handoff && !ready/);
  assert.match(readyBlock, /handoff\.copyArtifacts/);
  assert.match(component, /canCreate=\{ready\}/);
  assert.match(component, /\{canCreate \? \([\s\S]*onClick=\{onCreateToken\}/);
  assert.match(component, /accessHandoff\.tokenBlockedDetail/);
  assert.match(component, /disabled=\{Boolean\(activeToken\) \|\| tokenAction !== "" \|\| !handoff\.tokenEligibility\.eligible\}/);
});

test("existing tokens remain revocable after handoff readiness is lost", () => {
  assert.match(component, /\{handoff \? \([\s\S]*<AccessHandoffTokenSection/);
  assert.match(component, /token\.status === "active"/);
  assert.match(component, /onRevokeToken\(token\.id\)/);
});

test("access handoff uses responsive product styles and shared theme tokens", () => {
  assert.match(styles, /\.access-handoff\s*\{/);
  assert.match(styles, /\.access-handoff-artifacts\s*\{[^}]*grid-template-columns:\s*repeat\(2,\s*minmax\(0,\s*1fr\)\);/s);
  assert.match(styles, /\.access-handoff-one-time-token\s*\{[^}]*var\(--success-border\)/s);
});

test("access handoff presents a three-step builder journey with progressive disclosure", () => {
  assert.match(component, /access-handoff-boundary/);
  assert.match(component, /access-handoff-connect/);
  assert.match(component, /access-handoff-token-section/);
  assert.match(component, /<details>[\s\S]*<summary>\{t\("accessHandoff\.previewArtifact"\)\}/);
  assert.match(component, /accessHandoff\.boundaryTitle/);
  assert.match(component, /accessHandoff\.connectTitle/);
  assert.match(styles, /\.access-handoff-section-heading\s*\{/);
  assert.match(styles, /\.access-handoff-artifact-heading\s*\{/);
});
