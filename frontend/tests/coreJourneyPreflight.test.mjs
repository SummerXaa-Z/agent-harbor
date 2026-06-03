import assert from "node:assert/strict";
import test from "node:test";

import {
  coreJourneyPreflightCanRun,
  coreJourneyPreflightRows,
  defaultCoreJourneyPreflight,
} from "../src/coreJourneyPreflight.ts";

test("defaultCoreJourneyPreflight starts as pending and cannot run", () => {
  assert.equal(coreJourneyPreflightCanRun(defaultCoreJourneyPreflight), false);
  assert.deepEqual(
    coreJourneyPreflightRows(defaultCoreJourneyPreflight).map((row) => row.status),
    ["pending", "pending", "warning"],
  );
});

test("coreJourneyPreflightCanRun requires API and mock MCP health", () => {
  assert.equal(coreJourneyPreflightCanRun({ api: "ok", mockMcp: "ok", privateUpstreams: "warning" }), true);
  assert.equal(coreJourneyPreflightCanRun({ api: "error", mockMcp: "ok", privateUpstreams: "warning" }), false);
  assert.equal(coreJourneyPreflightCanRun({ api: "ok", mockMcp: "error", privateUpstreams: "warning" }), false);
});
