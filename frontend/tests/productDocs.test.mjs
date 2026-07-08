import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const readme = readFileSync(new URL("../../README.md", import.meta.url), "utf8");
const demoScript = readFileSync(new URL("../../scripts/demo.sh", import.meta.url), "utf8");
const webConsoleProductionJourneyScript = readFileSync(
  new URL("../../scripts/scenario-web-console-production-journey.sh", import.meta.url),
  "utf8"
);
const permissionPackageApprovalScript = readFileSync(
  new URL("../../scripts/scenario-permission-package-approval.sh", import.meta.url),
  "utf8"
);
const productJourney = readFileSync(new URL("../../docs/product/0.2.0-ai-admin-permission-journey.md", import.meta.url), "utf8");
const frontendDesignReference = readFileSync(new URL("../../docs/frontend-design-reference.md", import.meta.url), "utf8");
const releaseChecklist = readFileSync(new URL("../../docs/engineering/release-checklist.md", import.meta.url), "utf8");
const localValidationRecord = readFileSync(
  new URL("../../docs/engineering/0.2.0-local-validation-record.md", import.meta.url),
  "utf8"
);
const productionReportPlan = readFileSync(
  new URL("../../docs/engineering/0.2.0-production-report-plan.md", import.meta.url),
  "utf8"
);
const productionReportDesign = readFileSync(
  new URL("../../docs/engineering/0.2.0-production-report-design.md", import.meta.url),
  "utf8"
);
const changelog = readFileSync(new URL("../../CHANGELOG.md", import.meta.url), "utf8");
const capabilityGovernanceScript = readFileSync(
  new URL("../../scripts/scenario-mcp-capability-governance.sh", import.meta.url),
  "utf8"
);
const coreJourneyScript = readFileSync(
  new URL("../../scripts/scenario-core-journey.sh", import.meta.url),
  "utf8"
);

function proseWithoutCode(text) {
  return text
    .replace(/```[\s\S]*?```/g, "")
    .replace(/`[^`]*`/g, "")
    .replace(/\[[^\]]+\]\([^)]+\)/g, "");
}

test("permission changes README quickstart is split into actionable bilingual steps", () => {
  const start = readme.indexOf("## Try the Permission Changes Console");
  const end = readme.indexOf("Use the CLI scenario", start);
  const section = readme.slice(start, end);

  assert.notEqual(start, -1);
  assert.notEqual(end, -1);
  assert.match(section, /### What this validates/);
  assert.match(section, /### Run it locally/);
  assert.match(section, /### 本地运行/);
  assert.match(section, /1\. Start the local demo stack/);
  assert.match(section, /1\. 启动本地演示环境/);
  assert.match(section, /Technical overrides/);
  assert.match(section, /技术覆盖/);
  assert.doesNotMatch(section, /The Permission Changes console proves[\s\S]{1200,}?权限变更与状态检查控制台验证/);
});

test("demo script wires isolated ports without hidden frontend variables", () => {
  assert.match(demoScript, /API_BASE_URL="http:\/\/\$\{API_URL_HOST\}:\$\{API_PORT\}"/);
  assert.match(demoScript, /FRONTEND_ORIGIN="http:\/\/\$\{FRONTEND_URL_HOST\}:\$\{FRONTEND_PORT\}"/);
  assert.match(demoScript, /AGENT_HARBOR_CORS_ORIGINS="\$\{AGENT_HARBOR_CORS_ORIGINS:-\$FRONTEND_ORIGIN\}"/);
  assert.match(demoScript, /VITE_API_BASE="\$\{VITE_API_BASE:-\$API_BASE_URL\}"/);
});

test("demo script checks ports before installing dependencies", () => {
  const portPreflight = demoScript.indexOf('assert_port_free "API" "$API_PORT"');
  const frontendInstall = demoScript.indexOf('"${PNPM_CMD[@]}" --dir frontend install --frozen-lockfile');
  const realMcpInstall = demoScript.indexOf('"${PNPM_CMD[@]}" --dir scripts/real-mcp install --frozen-lockfile');
  assert.match(demoScript, /PNPM:-corepack pnpm/);
  assert.ok(portPreflight >= 0);
  assert.ok(frontendInstall > portPreflight);
  assert.ok(realMcpInstall > portPreflight);
});

test("web console production smoke uses the canonical go-live route", () => {
  assert.match(webConsoleProductionJourneyScript, /for hash in getting-started registry ask ai-admin go-live; do/);
  assert.doesNotMatch(webConsoleProductionJourneyScript, /for hash in .*evidence/);
});

test("public product docs use records wording instead of forensic-style prose", () => {
  const publicProse = proseWithoutCode([
    readme,
    productJourney,
    frontendDesignReference,
    releaseChecklist,
    localValidationRecord,
    productionReportPlan,
    productionReportDesign,
    changelog
  ].join("\n"));
  assert.doesNotMatch(publicProse, /\bEvidence\b|\bevidence\b|证据/);
});

test("primary product docs keep legacy report aliases out of the main path", () => {
  assert.doesNotMatch(readme, /export_permission_package_production_evidence/);
  assert.doesNotMatch(productJourney, /export_permission_package_production_evidence/);
});

test("public release docs keep legacy report identifiers in compatibility docs only", () => {
  assert.doesNotMatch(changelog, /export_permission_package_production_evidence/);
  assert.doesNotMatch(productJourney, /export_production_evidence/);
});

test("release scenario operator output uses report and record wording", () => {
  assert.doesNotMatch(permissionPackageApprovalScript, /production evidence report/i);
  assert.doesNotMatch(permissionPackageApprovalScript, /production report/i);
  assert.doesNotMatch(permissionPackageApprovalScript, /trace evidence verified/i);
  assert.doesNotMatch(permissionPackageApprovalScript, /audit evidence verified/i);
  assert.doesNotMatch(permissionPackageApprovalScript, /after evidence/i);
  assert.doesNotMatch(permissionPackageApprovalScript, /application evidence/i);
  assert.doesNotMatch(permissionPackageApprovalScript, /evidence = report\.get/);
  assert.doesNotMatch(capabilityGovernanceScript, /policy evidence/i);
  assert.doesNotMatch(capabilityGovernanceScript, /trace evidence verified/i);
  assert.doesNotMatch(coreJourneyScript, /trace evidence/i);
});
