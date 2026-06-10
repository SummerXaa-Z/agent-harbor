import assert from "node:assert/strict";
import test from "node:test";

import {
  dropdownKeyAction,
  nextDropdownActiveIndex
} from "../src/dropdownKeyboard.ts";

test("dropdown keyboard navigation moves through options without native select chrome", () => {
  assert.equal(nextDropdownActiveIndex(-1, 3, "ArrowDown"), 0);
  assert.equal(nextDropdownActiveIndex(0, 3, "ArrowDown"), 1);
  assert.equal(nextDropdownActiveIndex(2, 3, "ArrowDown"), 0);
  assert.equal(nextDropdownActiveIndex(0, 3, "ArrowUp"), 2);
  assert.equal(nextDropdownActiveIndex(2, 3, "Home"), 0);
  assert.equal(nextDropdownActiveIndex(0, 3, "End"), 2);
  assert.equal(nextDropdownActiveIndex(1, 0, "ArrowDown"), -1);
});

test("dropdown key actions distinguish open, select, close, and ignore", () => {
  assert.equal(dropdownKeyAction("ArrowDown", false), "open");
  assert.equal(dropdownKeyAction("ArrowDown", true), "move");
  assert.equal(dropdownKeyAction("Enter", true), "select");
  assert.equal(dropdownKeyAction(" ", true), "select");
  assert.equal(dropdownKeyAction("Escape", true), "close");
  assert.equal(dropdownKeyAction("Tab", true), "ignore");
});
