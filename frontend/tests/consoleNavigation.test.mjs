import assert from "node:assert/strict";
import test from "node:test";

import {
  defaultNavKey,
  navGroups,
  navItems,
  viewForNav
} from "../src/consoleNavigation.ts";

test("every primary navigation item resolves to a distinct workspace", () => {
  const views = navItems.map((item) => viewForNav(item.key));
  const viewKeys = views.map((view) => view.key);

  assert.deepEqual(viewKeys, [
    "ai-admin",
    "access",
    "evidence",
    "traces",
    "cockpit",
    "registry",
    "capabilities",
    "policies",
    "routes",
  ]);
  assert.equal(new Set(views.map((view) => view.primaryPanelKey)).size, views.length);
});

test("navigation is grouped by user task", () => {
  assert.deepEqual(navGroups.map((group) => group.key), ["primary", "audit", "configuration"]);
  assert.deepEqual(navItems.map((item) => item.groupKey), [
    "primary",
    "primary",
    "primary",
    "audit",
    "audit",
    "configuration",
    "configuration",
    "configuration",
    "configuration"
  ]);
  assert.ok(navItems.every((item) => item.detailKey.startsWith("navDetail.")));
});

test("default navigation opens the permission package production journey", () => {
  assert.equal(defaultNavKey, "ai-admin");
  assert.equal(navItems[0].key, defaultNavKey);
  assert.equal(viewForNav("unknown").key, defaultNavKey);
});
