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
});

test("createTranslator falls back to English for missing keys", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("missing.key", "Fallback"), "Fallback");
});
