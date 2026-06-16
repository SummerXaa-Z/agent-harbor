import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const app = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const forms = readFileSync(new URL("../src/components/ManagementForms.tsx", import.meta.url), "utf8");
const primitives = readFileSync(new URL("../src/components/ConsolePrimitives.tsx", import.meta.url), "utf8");
const hook = readFileSync(new URL("../src/hooks/useManagementOperations.ts", import.meta.url), "utf8");
const i18n = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");

test("agent one-time keys are cleared when the key modal closes", () => {
  const createKeyAction = app.slice(
    app.indexOf("function createKeyAction"),
    app.indexOf("function rotateCredentialAction")
  );

  assert.match(primitives, /onClose\?: \(\) => void/);
  assert.match(primitives, /function closeModal\(\)/);
  assert.match(primitives, /onClose\?\.\(\)/);
  assert.match(primitives, /if \(event\.key === "Escape"\) closeModal\(\)/);
  assert.match(primitives, /if \(event\.target === event\.currentTarget\) closeModal\(\)/);
  assert.match(createKeyAction, /title=\{t\("panel\.createKey"\)\}/);
  assert.match(createKeyAction, /onClose=\{management\.clearCreatedKey\}/);
});

test("agent one-time keys have an explicit clear action next to copy", () => {
  assert.match(forms, /onDismissCreatedKey/);
  assert.match(forms, /<button className="secondary-button" type="button" onClick=\{onDismissCreatedKey\}>/);
  assert.match(forms, /t\("action\.clearOneTimeKey"\)/);
  assert.match(i18n, /"action\.clearOneTimeKey": "Clear key"/);
  assert.match(i18n, /"action\.clearOneTimeKey": "清除密钥"/);
});

test("management operations clear stale plaintext keys on form edits and new submits", () => {
  assert.match(hook, /function clearCreatedKey\(\)/);
  assert.match(hook, /function updateKeyForm\(next: KeyCreateFormState\)/);
  assert.match(hook, /setCreatedKey\(null\);\s+setKeyForm\(next\)/);
  assert.match(hook, /setCreatedKey\(null\);\s+if \(!beginManagementMutation\("create_key"\)\) return/);
  assert.match(hook, /clearCreatedKey,/);
  assert.match(hook, /setKeyForm: updateKeyForm/);
});
