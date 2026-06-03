import assert from "node:assert/strict";
import test from "node:test";

import {
  createTranslator,
  normalizeLanguage,
  resolveInitialLanguage
} from "../src/i18n.ts";

test("normalizeLanguage supports English and Simplified Chinese", () => {
  assert.equal(normalizeLanguage("zh-CN"), "zh-CN");
  assert.equal(normalizeLanguage("zh-Hans-CN"), "zh-CN");
  assert.equal(normalizeLanguage("en-US"), "en");
  assert.equal(normalizeLanguage("fr-FR"), "en");
});

test("resolveInitialLanguage prefers stored language over browser language", () => {
  assert.equal(resolveInitialLanguage("en", ["zh-CN"]), "en");
  assert.equal(resolveInitialLanguage(undefined, ["zh-CN", "en-US"]), "zh-CN");
  assert.equal(resolveInitialLanguage(undefined, ["en-US", "zh-CN"]), "en");
});

test("createTranslator returns core journey Chinese labels", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("app.title"), "AgentHarbor 控制平面");
  assert.equal(t("nav.access"), "权限");
  assert.equal(t("page.access"), "租户权限控制台");
  assert.equal(t("metric.runtimeEvidence"), "运行证据");
  assert.equal(t("action.loadProfile"), "加载档案");
  assert.equal(t("panel.coreJourney"), "核心旅程工作台");
  assert.equal(t("action.runCoreJourney"), "跑通核心旅程");
  assert.equal(t("journey.step.grantChain"), "租户/工作区/实例授权");
});

test("createTranslator returns Chinese labels for operator controls", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("form.name"), "名称");
  assert.equal(t("form.channel"), "通道");
  assert.equal(t("form.credentialHeader"), "凭据 Header");
  assert.equal(t("action.createAgent"), "创建 Agent");
  assert.equal(t("action.rotateCredential"), "轮换凭据");
  assert.equal(t("table.policy"), "策略");
  assert.equal(t("status.agentDraft"), "草稿");
  assert.equal(t("status.policyAllow"), "允许");
  assert.equal(t("empty.routePolicies.title"), "暂无路由策略");
});

test("createTranslator falls back to English for missing keys", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("missing.key", "Fallback"), "Fallback");
});
