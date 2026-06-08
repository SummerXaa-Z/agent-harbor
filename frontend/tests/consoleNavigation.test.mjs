import assert from "node:assert/strict";
import test from "node:test";

import {
  defaultNavKey,
  navItems,
  viewForNav
} from "../src/consoleNavigation.ts";

test("every primary navigation item resolves to a distinct workspace", () => {
  const views = navItems.map((item) => viewForNav(item.key));
  const viewKeys = views.map((view) => view.key);

  assert.deepEqual(viewKeys, [
    "ai-admin",
    "access",
    "traces",
    "evidence",
    "cockpit",
    "registry",
    "routes",
    "policies",
    "capabilities"
  ]);
  assert.equal(new Set(views.map((view) => view.primaryPanelKey)).size, views.length);
});

test("default navigation opens the permission package production journey", () => {
  assert.equal(defaultNavKey, "ai-admin");
  assert.equal(navItems[0].key, defaultNavKey);
  assert.equal(viewForNav("unknown").key, defaultNavKey);
});
