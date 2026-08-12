import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import { createTranslator, translationKeys } from "../src/i18n.ts";

const capabilityView = readFileSync(new URL("../src/components/CapabilityGovernanceView.tsx", import.meta.url), "utf8");
const workbench = readFileSync(new URL("../src/components/AiAdminPermissionWorkbench.tsx", import.meta.url), "utf8");
const askView = readFileSync(new URL("../src/components/AskAccessView.tsx", import.meta.url), "utf8");

test("capability governance labels do not look like permission-change approval", () => {
  const en = createTranslator("en");
  const zh = createTranslator("zh-CN");

  assert.match(capabilityView, /t\("table\.capabilityGovernanceStatus"\)/);
  assert.equal(en("table.capabilityGovernanceStatus"), "Governance status");
  assert.equal(zh("table.capabilityGovernanceStatus"), "能力治理状态");
  assert.equal(zh("status.capabilityPendingReview"), "能力待复核");
  assert.equal(zh("status.capabilityApproved"), "能力已批准");
  assert.match(zh("text.capabilityCatalogHelp"), /不代表某次运行时调用已获准/);
});

test("permission change approval labels stay scoped to the matching change", () => {
  const en = createTranslator("en");
  const zh = createTranslator("zh-CN");

  assert.match(workbench, /t\("section\.permissionChangeApproval"\)/);
  assert.equal(en("section.permissionChangeApproval"), "Change Approval");
  assert.equal(zh("section.permissionChangeApproval"), "变更审批");
  assert.equal(zh("status.approvalPending"), "变更待审批");
  assert.equal(zh("status.approvalApproved"), "变更已批准");
  assert.match(zh("text.permissionRequestApprovalHelp"), /只授权这份匹配的权限变更/);
});

test("access query presents a runtime result instead of a catalog or approval state", () => {
  const en = createTranslator("en");
  const zh = createTranslator("zh-CN");

  assert.match(askView, /t\("ask\.outcomeAllowed"\)/);
  assert.match(askView, /t\("ask\.outcomeDenied"\)/);
  assert.equal(en("ask.answerTitle"), "This access result");
  assert.equal(zh("ask.answerTitle"), "本次访问结论");
  assert.equal(zh("ask.outcomeAllowed"), "本次允许");
  assert.equal(zh("ask.outcomeDenied"), "本次拒绝");
  assert.equal(zh("ask.recordLayer.capability"), "能力治理");
  assert.match(zh("ask.subtitle"), /不同于能力治理状态和变更审批状态/);
});

test("state-semantics copy is available in both supported languages", () => {
  const english = new Set(translationKeys("en"));
  const chinese = new Set(translationKeys("zh-CN"));
  const keys = [
    "ask.answerTitle",
    "ask.outcomeAllowed",
    "ask.outcomeDenied",
    "section.capabilityGovernance",
    "section.permissionChangeApproval",
    "status.capabilityApproved",
    "status.capabilityPendingReview",
    "status.approvalApproved",
    "status.approvalPending",
    "table.capabilityGovernanceStatus",
    "text.capabilityCatalogHelp",
    "text.permissionRequestApprovalHelp"
  ];

  for (const key of keys) {
    assert.equal(english.has(key), true, `${key} missing in English`);
    assert.equal(chinese.has(key), true, `${key} missing in zh-CN`);
  }
});
