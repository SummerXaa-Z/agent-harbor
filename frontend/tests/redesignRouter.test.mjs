import assert from "node:assert/strict";
import test from "node:test";

import { translationKeys } from "../src/i18n.ts";
import {
  adminFooterNav,
  adminHash,
  adminNavSections,
  adminViews,
  isRedesignHash,
  otherSurface,
  parseRedesignHash,
  redesignHash,
  userHash,
  userNavSections,
  userViews,
} from "../src/redesign/router.ts";

test("the empty hash is the redesign entry page since P5", () => {
  assert.deepEqual(parseRedesignHash(""), {
    canonicalHash: "#/",
    redirected: true,
    route: { surface: "entry" },
  });
  assert.equal(isRedesignHash(""), true);
  assert.equal(isRedesignHash("#"), true);
});

test("retired legacy hashes redirect to their redesign successors", () => {
  for (const [hash, canonicalHash] of [
    ["#ask", "#user/ask"],
    ["#/ask", "#user/ask"],
    ["#access", "#user/mine"],
    ["#getting-started", "#user/home"],
    ["#go-live", "#user/golive"],
    ["#evidence", "#user/golive"],
    ["#ai-admin", "#admin/approvals"],
    ["#admin-access", "#admin/admin"],
    ["#cockpit", "#admin/cockpit"],
    ["#traces", "#admin/traces"],
    ["#tenants", "#admin/tenants"],
    ["#registry", "#admin/registry"],
    ["#capabilities", "#admin/capabilities"],
    ["#policies", "#admin/policies"],
    ["#routes", "#admin/routes"],
  ]) {
    const parsed = parseRedesignHash(hash);
    assert.ok(parsed, hash);
    assert.equal(parsed.canonicalHash, canonicalHash, hash);
    assert.equal(parsed.redirected, true, hash);
    assert.notEqual(parsed.route.surface, "notfound", hash);
    assert.equal(isRedesignHash(hash), true, hash);
  }
  // Legacy hashes that never had a redesign successor land on the 404 page
  // with the attempted hash kept.
  for (const hash of ["#users", "#user-guide", "#administrator"]) {
    const parsed = parseRedesignHash(hash);
    assert.ok(parsed, hash);
    assert.deepEqual(parsed, { canonicalHash: hash, redirected: false, route: { surface: "notfound", attempted: hash.slice(1) } }, hash);
  }
});

test("unknown slash-shaped hashes resolve to the redesign not-found page", () => {
  for (const hash of ["#//user/home", "#/nonsense", "#/user/nope", "#/admin/nope", "#/user/nope?caller=agt_1", "#User/home", "#nonsense/foo"]) {
    const parsed = parseRedesignHash(hash);
    assert.ok(parsed, hash);
    assert.equal(parsed.route.surface, "notfound", hash);
    assert.equal(parsed.redirected, false, hash);
    assert.equal(isRedesignHash(hash), true, hash);
  }
  // The attempted path rides along so the 404 page can show it.
  assert.equal(parseRedesignHash("#/user/nope").route.attempted, "/user/nope");
  // The attempted hash is kept in the address bar rather than rewritten.
  assert.equal(parseRedesignHash("#/nonsense?x=1").canonicalHash, "#/nonsense?x=1");
});

test("entry hash opens the redesigned entry page", () => {
  assert.deepEqual(parseRedesignHash("#/"), { canonicalHash: "#/", redirected: false, route: { surface: "entry" } });
  assert.deepEqual(parseRedesignHash("#/?from=legacy"), {
    canonicalHash: "#/",
    redirected: true,
    route: { surface: "entry" },
  });
});

test("missing sub-routes redirect to each surface home", () => {
  for (const [hash, canonicalHash] of [
    ["#user", "#user/home"],
    ["#user/", "#user/home"],
    ["#admin", "#admin/cockpit"],
    ["#admin/", "#admin/cockpit"],
  ]) {
    const parsed = parseRedesignHash(hash);
    assert.ok(parsed, hash);
    assert.equal(parsed.canonicalHash, canonicalHash, hash);
    assert.equal(parsed.redirected, true, hash);
  }
});

test("slash-prefixed and over-long paths canonicalize", () => {
  const prefixed = parseRedesignHash("#/user/ask");
  assert.equal(prefixed.canonicalHash, "#user/ask");
  assert.equal(prefixed.redirected, true);
  assert.deepEqual(prefixed.route, { surface: "user", view: "ask", params: {} });

  const extra = parseRedesignHash("#admin/approvals/extra?id=apr_1");
  assert.equal(extra.canonicalHash, "#admin/approvals?id=apr_1");
  assert.equal(extra.redirected, true);

  assert.equal(parseRedesignHash("#user/home").redirected, false);
  assert.equal(parseRedesignHash("#admin/cockpit").redirected, false);
});

test("params are whitelisted per view, trimmed, and serialized in a stable order", () => {
  const parsed = parseRedesignHash(
    "#user/apply?subject=user:support-example&template=tpl_support&foo=1&caller=%20agt_caller%20&target=",
  );
  assert.deepEqual(parsed.route.params, {
    caller: "agt_caller",
    subject: "user:support-example",
    template: "tpl_support",
  });
  assert.equal(parsed.canonicalHash, "#user/apply?template=tpl_support&caller=agt_caller&subject=user:support-example");
  assert.equal(parsed.redirected, true);

  assert.deepEqual(parseRedesignHash("#user/home?caller=agt_1").route.params, {});
  assert.deepEqual(parseRedesignHash("#admin/traces?range=7d&result=denied&type=agent").route.params, {
    range: "7d",
    result: "denied",
    type: "agent",
  });
});

test("tenant tabs only accept the three confirmed tabs", () => {
  const center = parseRedesignHash("#admin/tenants?tab=center&tenant=tenant-east");
  assert.equal(center.canonicalHash, "#admin/tenants?tenant=tenant-east&tab=center");
  assert.deepEqual(parseRedesignHash("#admin/tenants?tab=bogus").route.params, {});
  assert.equal(adminHash("tenants", { tab: "profile" }), "#admin/tenants?tab=profile");
  assert.equal(adminHash("tenants", { tab: "bogus" }), "#admin/tenants");
});

test("serialized params round-trip through the parser", () => {
  const params = {
    capability: "cap with space & symbols=#?",
    caller: "agt_caller",
    subject: "user:支持组/上海+1",
    target: "agt_target",
  };
  const hash = userHash("ask", params);
  const parsed = parseRedesignHash(hash);
  assert.equal(parsed.redirected, false);
  assert.equal(parsed.canonicalHash, hash);
  assert.deepEqual(parsed.route.params, params);
  assert.equal(redesignHash({ surface: "entry" }), "#/");
});

test("every view has exactly one navigation entry with translated labels", () => {
  const userNav = userNavSections.flatMap((section) => section.items.map((item) => item.view));
  assert.deepEqual([...userNav].sort(), [...userViews].sort());

  const adminNav = [...adminNavSections.flatMap((section) => section.items.map((item) => item.view)), adminFooterNav.view];
  assert.deepEqual([...adminNav].sort(), [...adminViews].sort());

  const en = new Set(translationKeys("en"));
  const zh = new Set(translationKeys("zh-CN"));
  const labelKeys = [
    ...userNavSections.flatMap((section) => [section.labelKey, ...section.items.map((item) => item.labelKey)]),
    ...adminNavSections.flatMap((section) => [section.labelKey, ...section.items.map((item) => item.labelKey)]),
    adminFooterNav.labelKey,
  ].filter(Boolean);
  for (const key of labelKeys) {
    assert.ok(en.has(key), `missing en label ${key}`);
    assert.ok(zh.has(key), `missing zh-CN label ${key}`);
  }
});

test("the surface switch always points at the other surface", () => {
  assert.equal(otherSurface("user"), "admin");
  assert.equal(otherSurface("admin"), "user");
});
