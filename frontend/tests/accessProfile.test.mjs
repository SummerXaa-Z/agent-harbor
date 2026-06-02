import assert from "node:assert/strict";
import test from "node:test";

import {
  countInvalidAccessProfileRows,
  normalizeAccessProfileFilters,
  parseAccessProfileTraceLimit,
  scopeStatusTone,
  summarizeDataScopes
} from "../src/accessProfile.ts";

test("normalizeAccessProfileFilters trims optional query fields", () => {
  assert.deepEqual(
    normalizeAccessProfileFilters({
      callerInstanceId: " caller-1 ",
      capabilityId: "",
      targetId: " target-1 ",
      traceLimit: "5",
      workspaceId: " ws-1 "
    }),
    {
      callerInstanceId: "caller-1",
      capabilityId: undefined,
      targetId: "target-1",
      traceLimit: "5",
      workspaceId: "ws-1"
    }
  );
});

test("parseAccessProfileTraceLimit accepts API bounds and rejects invalid values", () => {
  assert.deepEqual(parseAccessProfileTraceLimit(undefined), { ok: true, value: 20 });
  assert.deepEqual(parseAccessProfileTraceLimit("0"), { ok: true, value: 0 });
  assert.deepEqual(parseAccessProfileTraceLimit(100), { ok: true, value: 100 });
  assert.deepEqual(parseAccessProfileTraceLimit("101"), {
    ok: false,
    message: "Trace limit must be an integer between 0 and 100."
  });
  assert.deepEqual(parseAccessProfileTraceLimit("1.5"), {
    ok: false,
    message: "Trace limit must be an integer between 0 and 100."
  });
});

test("summarizeDataScopes produces compact data-scope labels", () => {
  assert.equal(
    summarizeDataScopes([
      { dataDomain: "crm", dataset: "customers", classification: "internal" },
      { dataDomain: "contracts", dataset: "packages", classification: "confidential" },
      { dataDomain: "finance", dataset: "invoices", classification: "restricted" }
    ]),
    "crm/customers/internal, contracts/packages/confidential +1"
  );
  assert.equal(summarizeDataScopes([], "empty"), "empty");
});

test("scopeStatusTone highlights invalid scope rows", () => {
  assert.equal(scopeStatusTone("valid"), "success");
  assert.equal(scopeStatusTone("invalid"), "danger");
});

test("countInvalidAccessProfileRows includes grant, workspace, and instance rows", () => {
  assert.equal(
    countInvalidAccessProfileRows({
      generatedAt: "2026-06-02T00:00:00Z",
      grants: [
        {
          scopeStatus: "invalid",
          tenantEntitlement: {},
          workspaceAssignments: [
            {
              instanceAssignments: [
                { instanceAssignment: {}, scopeStatus: "invalid" },
                { instanceAssignment: {}, scopeStatus: "valid" }
              ],
              scopeStatus: "valid",
              workspaceAssignment: {}
            },
            {
              instanceAssignments: [],
              scopeStatus: "invalid",
              workspaceAssignment: {}
            }
          ]
        }
      ],
      recentTraces: [],
      scopeTenants: [],
      summary: {},
      tenant: {}
    }),
    3
  );
});

test("countInvalidAccessProfileRows tolerates null collection fields", () => {
  assert.equal(
    countInvalidAccessProfileRows({
      generatedAt: "2026-06-02T00:00:00Z",
      grants: null,
      recentTraces: null,
      scopeTenants: null,
      summary: {},
      tenant: {}
    }),
    0
  );
  assert.equal(
    countInvalidAccessProfileRows({
      generatedAt: "2026-06-02T00:00:00Z",
      grants: [
        {
          scopeStatus: "invalid",
          tenantEntitlement: {},
          workspaceAssignments: null
        }
      ],
      recentTraces: [],
      scopeTenants: [],
      summary: {},
      tenant: {}
    }),
    1
  );
});
