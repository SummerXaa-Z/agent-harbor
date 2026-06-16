import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const controllerSource = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const authHookSource = readFileSync(new URL("../src/hooks/useConsoleAuth.ts", import.meta.url), "utf8");
const i18nSource = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");
const loginViewSource = readFileSync(new URL("../src/components/ConsoleLoginView.tsx", import.meta.url), "utf8");
const typesSource = readFileSync(new URL("../src/types.ts", import.meta.url), "utf8");
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

test("console session keeps admin role and tenant boundary visible", () => {
  assert.match(typesSource, /role\?: string/);
  assert.match(typesSource, /tenantId\?: string/);
  assert.match(typesSource, /workspaceId\?: string/);
  assert.match(controllerSource, /sessionScopeLabel\(consoleAuth\.session, t\)/);
  assert.match(controllerSource, /className="session-chip-scope"/);
  assert.match(i18nSource, /"auth\.role\.tenant_admin": "Tenant admin"/);
  assert.match(i18nSource, /"auth\.role\.tenant_admin": "租户管理员"/);
  assert.match(i18nSource, /"auth\.sessionScopeTitle"/);
});

test("login view uses a real password form and language switch", () => {
  assert.match(loginViewSource, /<form className="login-form" onSubmit=\{onSubmit\}>/);
  assert.match(loginViewSource, /autoComplete="current-password"/);
  assert.match(loginViewSource, /type="password"/);
  assert.match(loginViewSource, /onLanguageChange\("zh-CN"\)/);
  assert.match(loginViewSource, /onLanguageChange\("en"\)/);
});

test("rate limited console login shows localized retry guidance", () => {
  assert.match(authHookSource, /ApiRequestError/);
  assert.match(authHookSource, /error\.code === "RATE_LIMITED"/);
  assert.match(authHookSource, /error\.consoleLoginRateLimited/);
  assert.match(authHookSource, /retryAfterSeconds/);
  assert.match(i18nSource, /"error\.consoleLoginRateLimited": "Too many failed sign-in attempts\. Try again in about \{seconds\} seconds\."/);
  assert.match(i18nSource, /"error\.consoleLoginRateLimited": "登录失败次数过多，请约 \{seconds\} 秒后再试。"/);
});
