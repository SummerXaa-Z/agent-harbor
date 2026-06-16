import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  journeyCompletionRefreshFailedMessageKey,
  refreshAfterJourneyCompletion
} from "../src/journeyCompletionRefresh.ts";

const app = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const coreJourneyHook = readFileSync(new URL("../src/hooks/useCoreJourneyController.ts", import.meta.url), "utf8");
const i18n = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");

function functionBlock(name, source) {
  const start = source.indexOf(`async function ${name}(`);
  assert.notEqual(start, -1, `${name} not found`);
  const next = source.indexOf("\n  async function ", start + 1);
  return source.slice(start, next === -1 ? undefined : next);
}

test("journey completion refresh helper reports refresh failures without throwing", async () => {
  const refreshError = new Error("profile unavailable");
  const result = await refreshAfterJourneyCompletion({
    onRefresh: async () => {
      throw refreshError;
    }
  });

  assert.equal(result.ok, false);
  assert.equal(result.error, refreshError);
  assert.equal(journeyCompletionRefreshFailedMessageKey("ai_admin_approval"), "message.aiAdminApprovalJourneyCompleteRefreshFailed");
  assert.equal(journeyCompletionRefreshFailedMessageKey("core_journey"), "message.coreJourneyCompleteRefreshFailed");
});

test("AI Admin approval journey commits runtime result before follow-up refresh", () => {
  const block = functionBlock("runAiAdminApprovalJourney", app);
  const resultIndex = block.indexOf("setAiAdminApprovalJourneyResult({");
  const refreshIndex = block.indexOf("refreshAfterJourneyCompletion");

  assert.ok(resultIndex >= 0, "runtime result should be committed");
  assert.ok(refreshIndex > resultIndex, "follow-up refresh should run after runtime result is committed");
  assert.match(block, /journeyCompletionRefreshFailedMessageKey\("ai_admin_approval"\)/);
  assert.match(block, /setAiAdminApprovalJourneyMessage\(\s*\{\s*key: refreshResult\.ok \? "message\.aiAdminApprovalJourneyComplete" : journeyCompletionRefreshFailedMessageKey\("ai_admin_approval"\)\s*\}/);
  assert.ok(
    block.indexOf("const [nextData, nextProfile, auditRows] = await Promise.all") > refreshIndex,
    "AI Admin follow-up refresh calls should be contained inside completion refresh helper"
  );
});

test("core journey commits runtime result before follow-up refresh", () => {
  const block = functionBlock("run", coreJourneyHook);
  const resultIndex = block.indexOf("setResult({");
  const refreshIndex = block.indexOf("refreshAfterJourneyCompletion");

  assert.ok(resultIndex >= 0, "runtime result should be committed");
  assert.ok(refreshIndex > resultIndex, "follow-up refresh should run after runtime result is committed");
  assert.match(block, /journeyCompletionRefreshFailedMessageKey\("core_journey"\)/);
  assert.match(block, /setMessage\(\s*\{\s*key: refreshResult\.ok \? "message\.coreJourneyComplete" : journeyCompletionRefreshFailedMessageKey\("core_journey"\)\s*\}/);
  assert.ok(
    block.indexOf("const [nextData, nextProfile] = await Promise.all") > refreshIndex,
    "Core journey follow-up refresh calls should be contained inside completion refresh helper"
  );
});

test("journey completion refresh failure messages are bilingual", () => {
  assert.match(i18n, /"message\.aiAdminApprovalJourneyCompleteRefreshFailed": "Runtime validation completed, but tenant profile, audit, or runtime records could not be refreshed\. Refresh before go-live confirmation\."/);
  assert.match(i18n, /"message\.coreJourneyCompleteRefreshFailed": "Core permission loop self-check completed, but runtime records or tenant profile could not be refreshed\. Refresh before continuing\."/);
  assert.match(i18n, /"message\.aiAdminApprovalJourneyCompleteRefreshFailed": "运行验证已完成，但租户访问画像、审计或运行记录未能刷新。上线确认前请先刷新。"/);
  assert.match(i18n, /"message\.coreJourneyCompleteRefreshFailed": "核心权限链路自检已完成，但运行记录或租户访问画像未能刷新。继续前请先刷新。"/);
});
