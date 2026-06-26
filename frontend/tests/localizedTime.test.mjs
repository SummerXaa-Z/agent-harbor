import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import { formatConsoleTime } from "../src/consolePresenters.ts";

const controller = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const resourceLifecycleView = readFileSync(new URL("../src/components/ResourceLifecycleView.tsx", import.meta.url), "utf8");

test("console refresh timestamps use the active language time format", () => {
  const timestamp = new Date("2026-06-17T03:04:05.000Z");

  assert.equal(formatConsoleTime(timestamp, "en", { timeZone: "UTC" }), "03:04:05 AM");
  assert.equal(formatConsoleTime(timestamp, "zh-CN", { timeZone: "UTC" }), "03:04:05");
});

test("refresh timestamp UI does not hard-code zh-CN time formatting", () => {
  assert.doesNotMatch(resourceLifecycleView, /toLocaleTimeString\("zh-CN"/);
  assert.doesNotMatch(controller, /toLocaleTimeString\("zh-CN"/);
});
