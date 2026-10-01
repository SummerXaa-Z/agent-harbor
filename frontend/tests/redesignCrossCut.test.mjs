import assert from "node:assert/strict";
import test from "node:test";

import {
  filterPaletteItems,
  paletteItems,
  recentItems,
} from "../src/redesign/model/paletteIndex.ts";
import {
  deriveNotifications,
  effectiveReadIds,
  emptyReadState,
  parseReadState,
  serializeReadState,
  unreadCounts,
} from "../src/redesign/model/notifications.ts";
import { adminViews, userViews, isRedesignHash } from "../src/redesign/router.ts";
import { permissionPackageTemplates } from "../src/permissionPackages.ts";

function agent(overrides = {}) {
  return {
    channelConfig: { endpoint: "http://a.example/sse" },
    channelType: "mcp",
    createdAt: "2026-09-20T08:00:00Z",
    credentialVersion: 1,
    id: "agt-a",
    name: "Target A",
    status: "active",
    tenantId: "t1",
    updatedAt: "2026-09-20T08:00:00Z",
    workspaceId: "w1",
    ...overrides,
  };
}

function capability(overrides = {}) {
  return {
    action: "read",
    discoveredAt: "2026-09-20T08:00:00Z",
    discoveryStatus: "approved",
    displayName: "Cap",
    id: "cap-1",
    key: "search_customer",
    riskLevel: "low",
    sensitivity: "internal",
    targetId: "agt-a",
    type: "mcp_tool",
    updatedAt: "2026-09-20T08:00:00Z",
    version: 1,
    ...overrides,
  };
}

function approval(overrides = {}) {
  return {
    allowedCapabilityIds: ["cap-1"],
    allowedCapabilityKeys: ["search_customer"],
    allowedCapabilityFingerprints: [],
    callerInstanceId: "agt-caller",
    createdAt: "2026-09-24T08:00:00Z",
    expiresAt: "2026-09-25T08:00:00Z",
    id: "apr-1",
    policyGate: { reasons: [], required: true },
    status: "pending",
    targetId: "agt-a",
    templateId: "support-ticket-triage",
    templateVersion: 1,
    policyVersion: 1,
    tenantId: "t1",
    updatedAt: "2026-09-24T08:00:00Z",
    workspaceId: "w1",
    ...overrides,
  };
}

test("paletteItems covers every page of both surfaces plus actions", () => {
  const items = paletteItems();
  const pageIds = items.filter((item) => item.group === "pages").map((item) => item.id.split(":")[1]);
  for (const view of [...userViews, ...adminViews]) assert.ok(pageIds.includes(view), view);
  assert.equal(new Set(pageIds).size, userViews.length + adminViews.length);
  const actions = items.filter((item) => item.group === "actions");
  assert.ok(actions.length >= 8);
  assert.ok(actions.every((item) => item.hash.startsWith("#")));
  const hashes = items.map((item) => item.hash);
  assert.ok(hashes.every((hash) => isRedesignHash(hash)));
  assert.equal(new Set(items.map((item) => item.id)).size, items.length, "ids are unique");
  // Actions may share a hash with their page — that is the deep link.
});

test("recentItems interleaves agents and approvals with deep links", () => {
  const agents = [
    agent({ id: "a1", name: "Old", updatedAt: "2026-09-01T00:00:00Z" }),
    agent({ id: "a2", name: "New", updatedAt: "2026-09-26T00:00:00Z" }),
    agent({ channelConfig: {}, channelType: "local", id: "a3", name: "Caller", updatedAt: "2026-09-25T00:00:00Z" }),
  ];
  const approvals = [
    approval({ id: "apr-old", updatedAt: "2026-09-01T00:00:00Z" }),
    approval({ id: "apr-new", status: "approved", updatedAt: "2026-09-27T00:00:00Z" }),
  ];
  const items = recentItems({ agents, approvals });
  assert.equal(items.length, 5);
  // Interleaving keeps both kinds visible; agents lead at equal rank.
  assert.equal(items[0].label, "New");
  assert.equal(items[0].sub, "http://a.example/sse");
  assert.equal(items[1].id, "recent:approval:apr-new");
  assert.equal(items[1].hash, "#admin/approvals?id=apr-new");
  const callerItem = items.find((item) => item.id === "recent:agent:a3");
  assert.equal(callerItem.sub, "a3", "local callers fall back to their id");
});

test("filterPaletteItems matches labels and sub lines, case-insensitively", () => {
  const pages = paletteItems();
  const label = (item) => (item.group === "pages" ? item.id.split(":")[1] : item.group === "actions" ? item.id : item.label);
  const needle = (q) => filterPaletteItems(pages, q, label);
  assert.equal(needle("").length, pages.length);
  assert.deepEqual(needle("COCKPIT").map((i) => i.id), ["page:cockpit"]);
  assert.deepEqual(needle("action:apply").map((i) => i.id), ["action:apply"]);
  const recent = [
    { group: "recent", hash: "#x", id: "r", label: "Knowledge MCP", sub: "http://127.0.0.1:8787/mcp", surface: "admin" },
  ];
  assert.equal(filterPaletteItems(recent, "8787", (i) => i.label).length, 1, "sub line is searchable");
  assert.equal(filterPaletteItems(recent, "nope", (i) => i.label).length, 0);
});

test("deriveNotifications splits surfaces and filters by the session actor", () => {
  const base = {
    agents: [agent({}), agent({ channelConfig: {}, channelType: "local", id: "agt-caller", name: "Caller" })],
    approvals: [
      approval({ id: "apr-pending" }),
      approval({ id: "apr-mine", requestedBy: "local-dev", resolvedAt: "2026-09-26T09:00:00Z", reviewedBy: "reviewer@x", reviewComment: "ok", status: "approved", updatedAt: "2026-09-26T09:00:00Z" }),
      approval({ id: "apr-other", requestedBy: "someone-else", resolvedAt: "2026-09-26T09:00:00Z", reviewedBy: "reviewer@x", status: "rejected", updatedAt: "2026-09-26T09:00:00Z" }),
    ],
    capabilities: [capability()],
    envRows: [
      { detail: "", fixKeys: [], key: "api", status: "ok", subKey: "rd.envcheck.api.ok" },
      { detail: "refused", fixKeys: [], key: "mcp", status: "error", subKey: "rd.envcheck.mcp.error" },
    ],
    sessionActor: "local-dev",
    templates: permissionPackageTemplates,
  };
  const items = deriveNotifications(base);
  assert.deepEqual(
    items.map((item) => item.dedupeId),
    ["approval:apr-mine:approved", "approval:apr-pending:pending", "env:mcp:error"],
  );
  const pending = items[1];
  assert.equal(pending.surface, "admin");
  assert.equal(pending.hash, "#admin/approvals?id=apr-pending");
  assert.equal(pending.params.caller, "Caller");
  assert.equal(pending.params.target, "Target A");
  const mine = items[0];
  assert.equal(mine.surface, "user");
  assert.equal(mine.hash, "#user/apply?approval=apr-mine");
  assert.ok(!items.some((item) => item.dedupeId.includes("apr-other")), "other actors' requests stay out");

  const rejected = deriveNotifications({
    ...base,
    approvals: [approval({ id: "apr-r", requestedBy: "local-dev", resolvedAt: "2026-09-26T09:00:00Z", reviewedBy: "reviewer@x", reviewComment: "window", status: "rejected", updatedAt: "2026-09-26T09:00:00Z" })],
  })[0];
  assert.equal(rejected.subKey, "rd.nt.rejected.subReason");
  assert.equal(rejected.params.comment, "window");
});

test("read state: baseline, storage round-trip and unread counts", () => {
  const items = deriveNotifications({
    agents: [],
    approvals: [
      approval({ id: "apr-old", createdAt: "2026-09-01T00:00:00Z", requestedBy: "local-dev", resolvedAt: "2026-09-01T00:00:00Z", reviewedBy: "r", status: "approved", updatedAt: "2026-09-01T00:00:00Z" }),
      approval({ id: "apr-new", createdAt: "2026-09-26T00:00:00Z" }),
    ],
    capabilities: [],
    envRows: [],
    sessionActor: "local-dev",
    templates: [],
  });
  const state = { baselineAt: "2026-09-20T00:00:00Z", readIds: [] };
  const read = effectiveReadIds(items, state);
  assert.ok(read.has("approval:apr-old:approved"), "resolved items before the baseline count as read");
  assert.ok(!read.has("approval:apr-new:pending"), "pending items never auto-read");
  assert.deepEqual(unreadCounts(items, read), { admin: 1, user: 0 });

  const state2 = { baselineAt: "2026-09-20T00:00:00Z", readIds: ["approval:apr-new:pending"] };
  assert.deepEqual(unreadCounts(items, effectiveReadIds(items, state2)), { admin: 0, user: 0 });

  const roundTrip = parseReadState(serializeReadState(state), "x");
  assert.deepEqual(roundTrip, state);
  assert.deepEqual(parseReadState(null, "now"), emptyReadState("now"));
  assert.deepEqual(parseReadState("{bogus", "now"), emptyReadState("now"));
  assert.deepEqual(
    parseReadState('{"baselineAt":"b","readIds":[1,"a"]}', "now"),
    { baselineAt: "b", readIds: ["a"] },
  );
  assert.deepEqual(parseReadState('{"readIds":[1,"a"]}', "now"), emptyReadState("now"), "missing baseline resets");
});

test("token lifecycle notifications cover handoff keys only, masked", () => {
  const now = Date.parse("2026-09-27T12:00:00Z");
  const keys = [
    { id: "k-active", agentId: "agt-caller", name: "handoff", prefix: "ah_active01", createdAt: "2026-09-27T11:00:00Z", expiresAt: "2026-09-27T13:00:00Z", createdForHandoffId: "h1" },
    { id: "k-expired", agentId: "agt-caller", name: "handoff", prefix: "ah_expired1", createdAt: "2026-09-27T10:00:00Z", expiresAt: "2026-09-27T11:30:00Z", createdForHandoffId: "h2" },
    { id: "k-revoked", agentId: "agt-caller", name: "handoff", prefix: "ah_revoked1", createdAt: "2026-09-27T10:00:00Z", expiresAt: "2026-09-27T13:00:00Z", revokedAt: "2026-09-27T11:45:00Z", createdForHandoffId: "h3" },
    { id: "k-plain", agentId: "agt-caller", name: "local", prefix: "ah_plain001", createdAt: "2026-09-27T10:00:00Z", expiresAt: "2026-09-27T11:30:00Z" },
  ];
  const items = deriveNotifications({
    agents: [], approvals: [], capabilities: [], envRows: [],
    keys, now, sessionActor: "local-dev", templates: [],
  });
  assert.deepEqual(items.map((item) => item.dedupeId), ["token:k-revoked:revoked", "token:k-expired:expired"]);
  const revoked = items[0];
  assert.equal(revoked.surface, "user");
  assert.equal(revoked.hash, "#user/golive");
  assert.equal(revoked.titleKey, "rd.nt.token.revoked.title");
  assert.equal(revoked.createdAt, "2026-09-27T11:45:00Z", "revocation time beats expiry");
  assert.equal(revoked.params.prefix, "ah_revoked1…");
  assert.equal(items[1].titleKey, "rd.nt.token.expired.title");
  assert.equal(items[1].createdAt, "2026-09-27T11:30:00Z");
  // Pre-baseline token items auto-read like other resolved items, so a new
  // browser does not replay every long-dead handoff token as unread.
  const read = effectiveReadIds(items, { baselineAt: "2026-09-27T11:40:00Z", readIds: [] });
  assert.ok(read.has("token:k-expired:expired"));
  assert.ok(!read.has("token:k-revoked:revoked"));
});

test("structural empty-state env notifications follow live data, not the stale snapshot", () => {
  const envRows = [
    { detail: "", fixKeys: [], key: "mcp", status: "warning", subKey: "rd.envcheck.mcp.noTarget" },
    { detail: "", fixKeys: [], key: "corePath", status: "warning", subKey: "rd.envcheck.corePath.noApp" },
    { detail: "refused", fixKeys: [], key: "mcp", status: "error", subKey: "rd.envcheck.mcp.error", subParams: { endpoint: "http://x" } },
  ];
  const base = { agents: [agent()], approvals: [], capabilities: [], envRows, sessionActor: null, templates: [] };

  // Live counts say both conditions cleared: the two snapshot warnings drop
  // immediately while probe-dependent rows stay (round 5, finding #40).
  const cleared = deriveNotifications({ ...base, structuralEnv: { applicationCount: 1, registeredTargetCount: 1 } });
  assert.deepEqual(cleared.map((item) => item.dedupeId), ["env:mcp:error"]);

  // Empty state derives the same items and dedupe ids phase 1 produced, so
  // read state carries over.
  const empty = deriveNotifications({ ...base, structuralEnv: { applicationCount: 0, registeredTargetCount: 0 } });
  assert.deepEqual(
    empty.map((item) => item.dedupeId),
    ["env:mcp:error", "env:mcp:warning", "env:corePath:warning"],
  );
  const noTarget = empty.find((item) => item.dedupeId === "env:mcp:warning");
  assert.equal(noTarget.subKey, "rd.envcheck.mcp.noTarget");
  assert.equal(noTarget.hash, "#admin/cockpit");

  // Without structuralEnv the snapshot rows pass through unchanged.
  assert.equal(deriveNotifications(base).length, 3);
});
