import assert from "node:assert/strict";
import test from "node:test";

import { createTranslator } from "../src/i18n.ts";
import { setupGapAgentDraft } from "../src/resourceSetupGapAgentDraft.ts";

const tZh = createTranslator("zh-CN");
const tEn = createTranslator("en");

test("setup gap caller draft pre-fills only safe caller fields", () => {
  const draft = setupGapAgentDraft("caller", tZh);

  assert.equal(draft.name, "客服助手");
  assert.equal(draft.channelType, "local");
  assert.equal(draft.status, "active");
  assert.equal(draft.description, "当前工作区用于发起访问查询和工具调用的本地调用方 Agent。");
  assert.equal(draft.retryMaxAttempts, "1");
  assert.equal(draft.retryBackoffMs, "0");
  assert.equal(draft.endpoint, "");
  assert.equal(draft.credentialHeader, "");
  assert.equal(draft.credentialName, "");
  assert.equal(draft.credentialValue, "");
});

test("setup gap target draft pre-fills only safe MCP target fields", () => {
  const draft = setupGapAgentDraft("target", tEn);

  assert.equal(draft.name, "Ticket tool service");
  assert.equal(draft.channelType, "mcp");
  assert.equal(draft.status, "active");
  assert.equal(draft.description, "MCP or service Agent that exposes tools for permission governance.");
  assert.equal(draft.retryMaxAttempts, "1");
  assert.equal(draft.retryBackoffMs, "0");
  assert.equal(draft.endpoint, "");
  assert.equal(draft.credentialHeader, "");
  assert.equal(draft.credentialName, "");
  assert.equal(draft.credentialValue, "");
});
