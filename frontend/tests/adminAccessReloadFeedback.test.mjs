import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  adminAccessReloadFailedMessageKey,
  reloadAfterAdminAccessMutation
} from "../src/adminAccessMutationReload.ts";

const hook = readFileSync(new URL("../src/hooks/useAdminAccessController.ts", import.meta.url), "utf8");
const i18n = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");

function asyncFunctionBlock(name) {
  const start = hook.indexOf(`async function ${name}(`);
  assert.notEqual(start, -1, `${name} not found`);
  const next = hook.indexOf("\n  async function ", start + 1);
  return hook.slice(start, next === -1 ? undefined : next);
}

test("admin access reload helper reports follow-up reload failures without throwing", async () => {
  const reloadError = new Error("database unavailable");
  const result = await reloadAfterAdminAccessMutation({
    action: "create_admin",
    onReload: async () => {
      throw reloadError;
    }
  });

  assert.equal(result.action, "create_admin");
  assert.equal(result.ok, false);
  assert.equal(result.error, reloadError);
  assert.equal(adminAccessReloadFailedMessageKey("create_admin"), "message.adminAccessCreatedReloadFailed");
  assert.equal(adminAccessReloadFailedMessageKey("rotate_admin_key"), "message.adminAccessRotatedReloadFailed");
  assert.equal(adminAccessReloadFailedMessageKey("disable_admin"), "message.adminAccessDisabledReloadFailed");
});

test("admin access mutations separate write success from follow-up list reload failures", () => {
  const createBlock = asyncFunctionBlock("submitCreate");
  const rotateBlock = asyncFunctionBlock("submitRotate");
  const disableBlock = asyncFunctionBlock("submitDisable");

  for (const block of [createBlock, rotateBlock, disableBlock]) {
    assert.match(block, /reloadAfterAdminAccessMutation/);
    assert.match(block, /loadAdminIdentities\(undefined, \{ throwOnError: true \}\)/);
    assert.doesNotMatch(block, /message:\s*localizedAdminAccessError\(error, "error\.adminAccessLoad"\)/);
  }

  assert.match(createBlock, /oneTimeKey: created\.key/);
  assert.match(rotateBlock, /oneTimeKey: rotated\.key/);
  assert.doesNotMatch(createBlock, /await loadAdminIdentities\(\);\s*\} catch/);
  assert.doesNotMatch(rotateBlock, /await loadAdminIdentities\(\);\s*\} catch/);
  assert.doesNotMatch(disableBlock, /await loadAdminIdentities\(\);\s*\} catch/);
});

test("admin access reload failure messages are bilingual", () => {
  assert.match(i18n, /"message\.adminAccessCreatedReloadFailed": "\{actor\} administrator created, but the administrator list could not be refreshed\. Copy the one-time key now, then refresh before continuing\."/);
  assert.match(i18n, /"message\.adminAccessRotatedReloadFailed": "\{actor\} administrator key rotated, but the administrator list could not be refreshed\. Copy the new one-time key now, then refresh before continuing\."/);
  assert.match(i18n, /"message\.adminAccessDisabledReloadFailed": "\{actor\} administrator disabled, but the administrator list could not be refreshed\. Refresh before continuing\."/);
  assert.match(i18n, /"message\.adminAccessCreatedReloadFailed": "\{actor\} 已创建，但管理员列表未能刷新。请立即复制一次性管理员密钥，继续前先刷新。"/);
  assert.match(i18n, /"message\.adminAccessRotatedReloadFailed": "\{actor\} 的管理员密钥已轮换，但管理员列表未能刷新。请立即复制新密钥，继续前先刷新。"/);
  assert.match(i18n, /"message\.adminAccessDisabledReloadFailed": "\{actor\} 已停用，但管理员列表未能刷新。继续前请先刷新。"/);
});
