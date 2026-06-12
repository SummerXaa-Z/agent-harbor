import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const controllerSource = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const authHookSource = readFileSync(new URL("../src/hooks/useConsoleAuth.ts", import.meta.url), "utf8");
const loginViewSource = readFileSync(new URL("../src/components/ConsoleLoginView.tsx", import.meta.url), "utf8");
const accessProfileHookSource = readFileSync(new URL("../src/hooks/useAccessProfileController.ts", import.meta.url), "utf8");
const coreJourneyHookSource = readFileSync(new URL("../src/hooks/useCoreJourneyController.ts", import.meta.url), "utf8");

test("console loads session before showing management views", () => {
  assert.match(controllerSource, /useConsoleAuth\(\)/);
  assert.match(controllerSource, /const consoleAccessReady = consoleAuth\.accessReady/);
  assert.match(controllerSource, /if \(!consoleAccessReady\) \{\s*return \(\s*<ConsoleLoginView/s);
  assert.match(authHookSource, /fetchConsoleSession\(controller\.signal\)/);
  assert.match(authHookSource, /loginConsole\(nextAdminKey\)/);
  assert.match(authHookSource, /logoutConsole\(\)/);
});

test("automatic management loaders are disabled until auth is ready", () => {
  assert.match(controllerSource, /enabled: consoleAccessReady/);
  assert.match(accessProfileHookSource, /if \(!enabled\) return/);
  assert.match(coreJourneyHookSource, /if \(!enabled\) return/);
});

test("login view uses a real password form and language switch", () => {
  assert.match(loginViewSource, /<form className="login-form" onSubmit=\{onSubmit\}>/);
  assert.match(loginViewSource, /autoComplete="current-password"/);
  assert.match(loginViewSource, /type="password"/);
  assert.match(loginViewSource, /onLanguageChange\("zh-CN"\)/);
  assert.match(loginViewSource, /onLanguageChange\("en"\)/);
});
