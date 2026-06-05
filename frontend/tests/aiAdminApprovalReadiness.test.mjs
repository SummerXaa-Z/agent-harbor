import assert from "node:assert/strict";
import test from "node:test";

import {
  aiAdminApprovalReadinessCanRun,
  aiAdminApprovalReadinessRows,
  defaultAiAdminApprovalReadiness
} from "../src/aiAdminApprovalReadiness.ts";

test("default AI Admin approval readiness blocks live journey until checks pass", () => {
  assert.equal(aiAdminApprovalReadinessCanRun(defaultAiAdminApprovalReadiness), false);
  assert.deepEqual(
    aiAdminApprovalReadinessRows(defaultAiAdminApprovalReadiness).map((row) => [row.key, row.status]),
    [
      ["api", "pending"],
      ["mockMcp", "pending"],
      ["subjectHeader", "pending"],
      ["privateUpstreams", "warning"],
      ["dataSource", "pending"]
    ]
  );
});

test("AI Admin approval readiness requires API, mock MCP, and subject header checks", () => {
  assert.equal(aiAdminApprovalReadinessCanRun({
    api: "ok",
    dataSource: "warning",
    mockMcp: "ok",
    privateUpstreams: "warning",
    subjectHeader: "ok"
  }), true);
  assert.equal(aiAdminApprovalReadinessCanRun({
    api: "ok",
    dataSource: "ok",
    mockMcp: "ok",
    privateUpstreams: "warning",
    subjectHeader: "error"
  }), false);
  assert.equal(aiAdminApprovalReadinessCanRun({
    api: "ok",
    dataSource: "ok",
    mockMcp: "error",
    privateUpstreams: "warning",
    subjectHeader: "ok"
  }), false);
});
