import assert from "node:assert/strict";
import test from "node:test";

import { createTranslator, translationKeys } from "../src/i18n.ts";
import {
  apiErrorCategory,
  apiErrorCopy,
  apiErrorPresentation,
} from "../src/redesign/model/apiErrorCategory.ts";
import {
  DEFAULT_DEMO_REVIEWER,
  isSelfReview,
  resolveReviewer,
  reviewerOptions,
  sessionIdentity,
} from "../src/redesign/model/demoRole.ts";
import {
  consoleDataStatus,
  metricsDailyCapability,
  systemCapabilitySet,
  targetProbeCapability,
} from "../src/redesign/model/dataStatus.ts";
import {
  AGENT_HARBOR_TOKEN_PLACEHOLDER,
  bearerPlaceholderHeader,
  containsUnmaskedToken,
  maskSecret,
  redactTokens,
  subjectHeaderExample,
} from "../src/redesign/model/secretMask.ts";
import { tableColumns } from "../src/redesign/model/tableColumns.ts";

class FakeApiError extends Error {
  constructor(status, message, code) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
    this.code = code;
  }
}

// Built at runtime so this file never contains a full-length key literal.
const fullAgentKey = `ah_${"Rw".repeat(21)}Q`;
const fullAdminKey = `ahadm_${"Kx".repeat(21)}Q`;

test("api errors fall into the four actionable categories", () => {
  assert.equal(apiErrorCategory(new TypeError("Failed to fetch")), "network");
  assert.equal(apiErrorCategory(new TypeError("Load failed")), "network");
  assert.equal(apiErrorCategory(new TypeError("NetworkError when attempting to fetch resource.")), "network");
  assert.equal(apiErrorCategory(new FakeApiError(401, "session expired", "UNAUTHORIZED")), "forbidden");
  assert.equal(apiErrorCategory(new FakeApiError(403, "out of scope", "FORBIDDEN")), "forbidden");
  assert.equal(apiErrorCategory(new FakeApiError(404, "not found", "NOT_FOUND")), "notFound");
  for (const status of [502, 503, 504]) {
    assert.equal(apiErrorCategory(new FakeApiError(status, "bad gateway")), "unavailable", String(status));
  }
  assert.equal(apiErrorCategory(new FakeApiError(400, "connection refused", "UPSTREAM_CONNECT_ERROR")), "unavailable");
  assert.equal(apiErrorCategory(new FakeApiError(400, "invalid", "VALIDATION_FAILED")), "other");
  assert.equal(apiErrorCategory(new TypeError("x is not a function")), "other");
  assert.equal(apiErrorCategory(new Error("boom")), "other");
  assert.equal(apiErrorCategory("boom"), "other");
  assert.equal(apiErrorCategory(null), "other");
});

test("every error category has a localized title and next step", () => {
  for (const language of ["en", "zh-CN"]) {
    const keys = new Set(translationKeys(language));
    for (const category of ["network", "forbidden", "notFound", "unavailable", "other"]) {
      assert.ok(keys.has(`rd.error.${category}.title`), `${language} rd.error.${category}.title`);
      assert.ok(keys.has(`rd.error.${category}.next`), `${language} rd.error.${category}.next`);
    }
  }
  const zh = createTranslator("zh-CN");
  assert.deepEqual(apiErrorCopy(zh, "network"), {
    next: zh("rd.error.network.next"),
    title: "网络错误",
  });
});

test("error presentation keeps the server code and cause as detail", () => {
  const zh = createTranslator("zh-CN");
  const en = createTranslator("en");

  const upstream = new FakeApiError(502, "dial tcp 127.0.0.1:8787: connect: connection refused", "UPSTREAM_CONNECT_ERROR");
  const zhUpstream = apiErrorPresentation(zh, "zh-CN", upstream, "error.consoleDataUnavailable");
  assert.equal(zhUpstream.category, "unavailable");
  assert.equal(zhUpstream.title, zh("rd.error.unavailable.title"));
  assert.match(zhUpstream.detail, /connection refused/);
  assert.ok(zhUpstream.detail.startsWith(zh("error.consoleDataUnavailable")));

  const validation = new FakeApiError(400, "reason is required", "VALIDATION_FAILED");
  const zhValidation = apiErrorPresentation(zh, "zh-CN", validation, "error.consoleDataUnavailable");
  assert.equal(zhValidation.category, "other");
  assert.equal(zhValidation.detail, `${zh("error.consoleDataUnavailable")}（VALIDATION_FAILED: reason is required）`);

  const enNetwork = apiErrorPresentation(en, "en", new TypeError("Failed to fetch"), "error.consoleDataUnavailable");
  assert.equal(enNetwork.category, "network");
  assert.equal(enNetwork.title, "Network error");
  assert.equal(enNetwork.detail, "Failed to fetch");
});

test("every table declares unique columns that fill exactly 100%", () => {
  const enKeys = new Set(translationKeys("en"));
  const zhKeys = new Set(translationKeys("zh-CN"));
  assert.equal(Object.keys(tableColumns).length, 10);
  for (const [tableId, columns] of Object.entries(tableColumns)) {
    const total = columns.reduce((sum, column) => sum + column.width, 0);
    assert.equal(total, 100, `${tableId} widths sum to ${total}`);
    assert.equal(new Set(columns.map((column) => column.key)).size, columns.length, `${tableId} keys are unique`);
    for (const column of columns) {
      assert.ok(column.width > 0, `${tableId}.${column.key} width`);
      assert.ok(enKeys.has(column.labelKey), `${tableId}.${column.key} en label ${column.labelKey}`);
      assert.ok(zhKeys.has(column.labelKey), `${tableId}.${column.key} zh label ${column.labelKey}`);
    }
  }
});

test("audit operation and row actions use different column labels", () => {
  const zh = createTranslator("zh-CN");
  assert.notEqual("rd.col.operation", "rd.col.actions");
  assert.equal(zh("rd.col.operation"), "操作");
  assert.equal(zh("rd.col.actions"), "操作");
  const en = createTranslator("en");
  assert.notEqual(en("rd.col.operation"), en("rd.col.actions"));
});

test("secrets are only ever shown as masked prefixes", () => {
  assert.equal(AGENT_HARBOR_TOKEN_PLACEHOLDER, "$" + "{AGENT_HARBOR_TOKEN}");
  assert.equal(bearerPlaceholderHeader, "Authorization: Bearer $" + "{AGENT_HARBOR_TOKEN}");
  assert.equal(subjectHeaderExample, "X-AgentHarbor-Subject-Id: user:support-example");

  assert.equal(maskSecret(fullAgentKey), `${fullAgentKey.slice(0, 12)}…`);
  assert.equal(maskSecret("  ah_short  ", 4), "ah_s…");
  assert.equal(maskSecret(""), "");

  assert.equal(containsUnmaskedToken(`export AGENT_HARBOR_TOKEN="${fullAgentKey}"`), true);
  assert.equal(containsUnmaskedToken(`admin ${fullAdminKey}`), true);
  assert.equal(containsUnmaskedToken(`token ${maskSecret(fullAgentKey)}`), false);
  assert.equal(containsUnmaskedToken(bearerPlaceholderHeader), false);
  assert.equal(containsUnmaskedToken("ah_RwGSwRbWi…"), false);
  assert.equal(containsUnmaskedToken("support_ah_x"), false);
  // Repeated calls must not depend on the global regex cursor.
  assert.equal(containsUnmaskedToken(fullAgentKey), true);
  assert.equal(containsUnmaskedToken(fullAgentKey), true);
});

test("redaction cuts full keys down to their stored prefixes", () => {
  const text = `curl -H "Authorization: Bearer ${fullAgentKey}" # admin ${fullAdminKey}`;
  const redacted = redactTokens(text);
  assert.equal(containsUnmaskedToken(redacted), false);
  assert.ok(redacted.includes(`${fullAgentKey.slice(0, 12)}…`));
  assert.ok(redacted.includes(`${fullAdminKey.slice(0, 14)}…`));
  assert.equal(redactTokens(bearerPlaceholderHeader), bearerPlaceholderHeader);
});

test("demo reviewer is selectable only when login is not required", () => {
  const demoSession = { actor: "local-dev", authenticated: false, requiresLogin: false, role: "platform_admin" };
  assert.deepEqual(resolveReviewer(demoSession, null), { reviewer: DEFAULT_DEMO_REVIEWER, selectable: true });
  assert.deepEqual(resolveReviewer(demoSession, "  reviewer-a "), { reviewer: "reviewer-a", selectable: true });
  assert.deepEqual(resolveReviewer(demoSession, "   "), { reviewer: DEFAULT_DEMO_REVIEWER, selectable: true });

  const loginSession = { actor: "alice", authenticated: true, requiresLogin: true, role: "security_reviewer" };
  assert.deepEqual(resolveReviewer(loginSession, "reviewer-a"), { reviewer: "alice", selectable: false });
  assert.deepEqual(resolveReviewer({ authenticated: false, requiresLogin: true }, "reviewer-a"), {
    reviewer: "",
    selectable: false,
  });
  assert.deepEqual(resolveReviewer(null, "reviewer-a"), { reviewer: "", selectable: false });
});

test("reviewer options add active security reviewers once", () => {
  const admin = (actor, role, status = "active") => ({ actor, role, status });
  assert.deepEqual(
    reviewerOptions([
      admin("carol", "security_reviewer"),
      admin("dave", "platform_admin"),
      admin("erin", "security_reviewer", "disabled"),
      admin(DEFAULT_DEMO_REVIEWER, "security_reviewer"),
      admin(" carol ", "security_reviewer"),
    ]),
    [DEFAULT_DEMO_REVIEWER, "carol"],
  );
  assert.deepEqual(reviewerOptions([]), [DEFAULT_DEMO_REVIEWER]);
});

test("self review is detected after trimming both sides", () => {
  assert.equal(isSelfReview("local-dev", "local-dev"), true);
  assert.equal(isSelfReview(" local-dev ", "local-dev"), true);
  assert.equal(isSelfReview("security-reviewer", "local-dev"), false);
  assert.equal(isSelfReview("", ""), false);
  assert.equal(isSelfReview("local-dev", undefined), false);
  assert.equal(isSelfReview(undefined, "local-dev"), false);
});

test("session identity names demo mode and only offers sign-out to signed-in sessions", () => {
  assert.deepEqual(sessionIdentity(null), { actor: "", canSignOut: false, demo: false, holderAgentCount: 0, roleKey: "" });
  assert.deepEqual(sessionIdentity({ actor: "local-dev", authenticated: false, requiresLogin: false, role: "platform_admin" }), {
    actor: "",
    canSignOut: false,
    demo: true,
    holderAgentCount: 0,
    roleKey: "auth.role.platform_admin",
  });
  assert.deepEqual(sessionIdentity({ actor: " alice ", authenticated: true, requiresLogin: true, role: "security_reviewer" }), {
    actor: "alice",
    canSignOut: true,
    demo: false,
    holderAgentCount: 0,
    roleKey: "auth.role.security_reviewer",
  });
  assert.equal(sessionIdentity({ authenticated: false, requiresLogin: true }).canSignOut, false);
  assert.equal(sessionIdentity({ authenticated: true, requiresLogin: true, role: "operator" }).roleKey, "");

  const keys = new Set(translationKeys("en"));
  for (const role of ["platform_admin", "security_reviewer", "tenant_admin"]) {
    const { roleKey } = sessionIdentity({ authenticated: true, requiresLogin: true, role });
    assert.ok(keys.has(roleKey), roleKey);
  }
});

test("console data status never presents sample rows as live", () => {
  assert.equal(consoleDataStatus({ hasData: false, hasError: false, loading: true }), "loading");
  assert.equal(consoleDataStatus({ hasData: false, hasError: false, loading: false }), "loading");
  assert.equal(consoleDataStatus({ hasData: false, hasError: true, loading: true }), "loading");
  assert.equal(consoleDataStatus({ hasData: false, hasError: true, loading: false }), "error");
  assert.equal(consoleDataStatus({ hasData: true, hasError: false, loadedFromApi: true, loading: false }), "live");
  assert.equal(consoleDataStatus({ hasData: true, hasError: false, loadedFromApi: false, loading: false }), "sample");
  assert.equal(consoleDataStatus({ hasData: true, hasError: false, loading: false }), "sample");
  assert.equal(consoleDataStatus({ hasData: true, hasError: true, loadedFromApi: true, loading: true }), "live");
});

test("system capabilities gate optional endpoints", () => {
  assert.equal(metricsDailyCapability, "metrics_daily_v1");
  assert.equal(targetProbeCapability, "target_probe_v1");
  assert.deepEqual([...systemCapabilitySet(undefined)], []);
  assert.deepEqual([...systemCapabilitySet(null)], []);
  assert.deepEqual([...systemCapabilitySet({ capabilities: "metrics_daily_v1" })], []);
  const capabilities = systemCapabilitySet({ capabilities: ["metrics_daily_v1", "", 3, "target_probe_v1", "metrics_daily_v1"] });
  assert.deepEqual([...capabilities], ["metrics_daily_v1", "target_probe_v1"]);
  assert.ok(capabilities.has(metricsDailyCapability));
});
