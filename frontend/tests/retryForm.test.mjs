import assert from "node:assert/strict";
import test from "node:test";

import { parseRetryFields } from "../src/retryForm.ts";

test("parseRetryFields omits default retry", () => {
  assert.deepEqual(parseRetryFields({ backoffMsText: "0", maxAttemptsText: "1" }), {
    ok: true,
    requested: false,
    backoffMs: 0,
    maxAttempts: 1
  });
});

test("parseRetryFields rejects blank companion fields instead of silently defaulting", () => {
  assert.deepEqual(parseRetryFields({ backoffMsText: "", maxAttemptsText: "2" }), {
    ok: false,
    message: "Retry backoff must be an integer between 0 and 1000 ms."
  });
  assert.deepEqual(parseRetryFields({ backoffMsText: "100", maxAttemptsText: "" }), {
    ok: false,
    message: "Retry attempts must be an integer between 1 and 4."
  });
});

test("parseRetryFields returns normalized retry values when requested", () => {
  assert.deepEqual(parseRetryFields({ backoffMsText: "250", maxAttemptsText: "3" }), {
    ok: true,
    requested: true,
    backoffMs: 250,
    maxAttempts: 3
  });
});
