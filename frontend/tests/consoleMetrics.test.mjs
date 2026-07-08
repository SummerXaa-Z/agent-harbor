import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  consoleDataSourceLabel,
  runtimeRecordMetric
} from "../src/consoleMetrics.ts";

const consoleMetricsSource = readFileSync(new URL("../src/consoleMetrics.ts", import.meta.url), "utf8");

test("consoleDataSourceLabel does not call live API data samples", () => {
  assert.equal(consoleDataSourceLabel(undefined, true), "Go runtime");
  assert.equal(consoleDataSourceLabel(undefined, false), "Fallback dataset");
  assert.equal(consoleDataSourceLabel("failed", true), "API error");
});

test("runtimeRecordMetric summarizes real runtime traces", () => {
  assert.deepEqual(runtimeRecordMetric(0, 0), {
    label: "Runtime Records",
    value: "0",
    detail: "no traces yet",
    tone: "neutral"
  });
  assert.deepEqual(runtimeRecordMetric(2, 1), {
    label: "Runtime Records",
    value: "3",
    detail: "2 allowed / 1 denied",
    tone: "info"
  });
  assert.deepEqual(runtimeRecordMetric(2, 0), {
    label: "Runtime Records",
    value: "2",
    detail: "2 allowed / 0 denied",
    tone: "success"
  });
});

test("runtime metric helpers no longer export legacy aliases", () => {
  assert.doesNotMatch(consoleMetricsSource, /RuntimeEvidenceMetric/);
  assert.doesNotMatch(consoleMetricsSource, /runtimeEvidenceMetric/);
});
