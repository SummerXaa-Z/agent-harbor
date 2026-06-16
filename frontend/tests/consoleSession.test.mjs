import assert from "node:assert/strict";
import test from "node:test";

import { sessionExpiryDelayMs } from "../src/consoleSession.ts";

test("sessionExpiryDelayMs returns the remaining session lifetime", () => {
  assert.equal(
    sessionExpiryDelayMs("2026-06-17T10:00:10Z", Date.parse("2026-06-17T10:00:00Z")),
    10_000,
  );
});

test("sessionExpiryDelayMs treats expired sessions as immediate refresh", () => {
  assert.equal(
    sessionExpiryDelayMs("2026-06-17T10:00:00Z", Date.parse("2026-06-17T10:00:10Z")),
    0,
  );
});

test("sessionExpiryDelayMs ignores missing or invalid timestamps", () => {
  assert.equal(sessionExpiryDelayMs(undefined, Date.parse("2026-06-17T10:00:00Z")), null);
  assert.equal(sessionExpiryDelayMs("not-a-date", Date.parse("2026-06-17T10:00:00Z")), null);
});
