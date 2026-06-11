import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  navHashFor,
  navItems,
  viewForNav
} from "../src/consoleNavigation.ts";

const gettingStartedView = readFileSync(new URL("../src/components/GettingStartedView.tsx", import.meta.url), "utf8");
const controller = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const i18n = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");
const styles = readFileSync(new URL("../src/styles.css", import.meta.url), "utf8");

test("getting started workspace is registered as a console view", () => {
  assert.equal(navHashFor("getting-started"), "#getting-started");
  assert.equal(viewForNav("getting-started").primaryPanelKey, "gettingStarted");
  assert.ok(navItems.some((item) => item.key === "getting-started" && item.detailKey === "navDetail.getting-started"));
  assert.match(controller, /import \{ GettingStartedView \} from "\.\/components\/GettingStartedView"/);
  assert.match(controller, /gettingStartedSteps\(data\)/);
  assert.match(controller, /case "getting-started":/);
});

test("getting started view renders checklist, chain, actions, and sample badges", () => {
  assert.match(gettingStartedView, /export function GettingStartedView/);
  assert.match(gettingStartedView, /className="getting-started-chain"/);
  assert.match(gettingStartedView, /className=\{`getting-started-step status-\$\{status\}`\}/);
  assert.match(gettingStartedView, /href=\{step\.targetHash\}/);
  assert.match(gettingStartedView, /gettingStarted\.sampleBadge/);
  assert.match(gettingStartedView, /gettingStarted\.sampleNotice/);
  assert.match(gettingStartedView, /steps\.findIndex\(\(step\) => !step\.done\)/);
});

test("getting started copy is bilingual and token-styled", () => {
  for (const key of [
    "nav.getting-started",
    "navDetail.getting-started",
    "page.gettingStarted",
    "gettingStarted.title",
    "gettingStarted.lead",
    "gettingStarted.sampleBadge",
    "gettingStarted.sampleNotice",
    "gettingStarted.step.connect-api.title",
    "gettingStarted.step.register-agents.title",
    "gettingStarted.step.discover-capabilities.title",
    "gettingStarted.step.create-grant-chain.title",
    "gettingStarted.step.run-decision.title",
    "gettingStarted.step.review-evidence.title",
    "gettingStarted.chain.tenant",
    "gettingStarted.chain.evidence"
  ]) {
    assert.match(i18n, new RegExp(`"${key}"`), `${key} should be present in i18n`);
  }

  assert.match(styles, /\.getting-started\s*\{/);
  assert.match(styles, /\.getting-started-step\.status-current\s+\.getting-started-step-index\s*\{/);
  assert.match(styles, /\.getting-started-chain\s*\{/);
});

test("console resolves first-run default navigation once after data loads", () => {
  assert.match(controller, /resolveDefaultNavKey/);
  assert.match(controller, /function initialHashNavKey\(\): NavKey \| null/);
  assert.match(controller, /const defaultNavResolvedRef = useRef\(initialHashNavKey\(\) !== null\)/);
  assert.match(controller, /const userSelectedNavRef = useRef\(false\)/);
  assert.match(controller, /if \(!data \|\| defaultNavResolvedRef\.current \|\| userSelectedNavRef\.current\) return/);
  assert.match(controller, /setActiveNav\(resolveDefaultNavKey\(data\)\)/);
  assert.match(controller, /defaultNavResolvedRef\.current = true/);
  assert.match(controller, /onClick=\{\(\) => selectActiveNav\(item\.key\)\}/);
});
