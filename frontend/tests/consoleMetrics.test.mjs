import assert from "node:assert/strict";
import test from "node:test";

import {
  consoleDataSourceLabel,
  runtimeEvidenceMetric
} from "../src/consoleMetrics.ts";

test("consoleDataSourceLabel does not call live API data samples", () => {
  assert.equal(consoleDataSourceLabel(undefined, true), "Go runtime");
  assert.equal(consoleDataSourceLabel(undefined, false), "Fallback dataset");
  assert.equal(consoleDataSourceLabel("failed", true), "API error");
});

test("runtimeEvidenceMetric summarizes real runtime traces", () => {
  assert.deepEqual(runtimeEvidenceMetric(0, 0), {
    label: "Runtime Records",
    value: "0",
    detail: "no traces yet",
    tone: "neutral"
  });
  assert.deepEqual(runtimeEvidenceMetric(2, 1), {
    label: "Runtime Records",
    value: "3",
    detail: "2 allowed / 1 denied",
    tone: "info"
  });
  assert.deepEqual(runtimeEvidenceMetric(2, 0), {
    label: "Runtime Records",
    value: "2",
    detail: "2 allowed / 0 denied",
    tone: "success"
  });
});
