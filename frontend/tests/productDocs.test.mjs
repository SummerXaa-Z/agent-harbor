import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const readme = readFileSync(new URL("../../README.md", import.meta.url), "utf8");
const demoScript = readFileSync(new URL("../../scripts/demo.sh", import.meta.url), "utf8");
const productJourney = readFileSync(new URL("../../docs/product/0.2.0-ai-admin-permission-journey.md", import.meta.url), "utf8");
const releaseChecklist = readFileSync(new URL("../../docs/engineering/release-checklist.md", import.meta.url), "utf8");
const changelog = readFileSync(new URL("../../CHANGELOG.md", import.meta.url), "utf8");

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
  const frontendInstall = demoScript.indexOf("pnpm --dir frontend install --frozen-lockfile");
  const realMcpInstall = demoScript.indexOf("pnpm --dir scripts/real-mcp install --frozen-lockfile");
  assert.ok(portPreflight >= 0);
  assert.ok(frontendInstall > portPreflight);
  assert.ok(realMcpInstall > portPreflight);
});

test("public product docs use records wording instead of forensic-style prose", () => {
  const publicProse = proseWithoutCode([readme, productJourney, releaseChecklist, changelog].join("\n"));
  assert.doesNotMatch(publicProse, /\bEvidence\b|\bevidence\b|证据/);
});
