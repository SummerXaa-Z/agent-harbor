import assert from "node:assert/strict";
import test from "node:test";

import {
  navItems,
  viewForNav
} from "../src/consoleNavigation.ts";

test("every primary navigation item resolves to a distinct workspace", () => {
  const views = navItems.map((item) => viewForNav(item.key));
  const viewKeys = views.map((view) => view.key);

  assert.deepEqual(viewKeys, [
    "cockpit",
    "ai-admin",
    "registry",
    "routes",
    "policies",
    "capabilities",
    "access",
    "traces",
    "evidence"
  ]);
  assert.equal(new Set(views.map((view) => view.primaryPanelKey)).size, views.length);
});

test("unknown navigation keys fall back to cockpit", () => {
  assert.equal(viewForNav("unknown").key, "cockpit");
});
