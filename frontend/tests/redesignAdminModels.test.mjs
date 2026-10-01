import assert from "node:assert/strict";
import test from "node:test";

import { createTranslator } from "../src/i18n.ts";
import { approvalCapabilityRows, approvalList, approvalNoticeKey, approvalRiskSummary, approvalTabCounts, filterApprovalList } from "../src/redesign/model/approvalReview.ts";
import { auditRangeSince, auditResourceTypeLabelKey, auditResourceTypes, auditTimelineRows, filterAuditTimeline, todayAuditCount, traceResultKind } from "../src/redesign/model/auditTimeline.ts";
import { apiServiceCheck, corePathCheck, envCheckSummary, envHealthCheck, mcpServiceCheck, preferredProbeTarget, probeFixGuidance, registeredMcpTargets } from "../src/redesign/model/envChecks.ts";
import { dailyTrend, trendDenyPeak, trendDirection, trendHasData, trendPeakDayLabel, trendSummary } from "../src/redesign/model/dailyTrend.ts";

const zh = createTranslator("zh-CN");

function metricsBucket(overrides = {}) {
  return { allowedCalls: 0, auditEvents: 0, calls: 0, date: "2026-09-20", deniedCalls: 0, denyRate: null, ...overrides };
}

function dailyMetrics(buckets, totalsOverrides = {}) {
  return {
    buckets,
    generatedAt: "2026-09-27T08:00:00Z",
    totals: { allowedCalls: 0, auditEvents: 0, calls: 0, deniedCalls: 0, ...totalsOverrides },
    truncated: false,
  };
}

test("dailyTrend maps buckets and keeps null deny rates", () => {
  const trend = dailyTrend(
    dailyMetrics(
      [metricsBucket(), metricsBucket({ auditEvents: 3, calls: 4, date: "2026-09-26", deniedCalls: 1, denyRate: 0.25 })],
      { auditEvents: 3, calls: 4, deniedCalls: 1 },
    ),
  );
  assert.equal(trend.points.length, 2);
  assert.equal(trend.points[0].denyRate, null);
  assert.equal(trend.points[1].denyRate, 0.25);
  assert.deepEqual(trend.totals, { auditEvents: 3, calls: 4, denied: 1 });
  assert.equal(trend.truncated, false);
});

test("dailyTrend tolerates missing metrics", () => {
  assert.equal(dailyTrend(null), null);
  assert.equal(dailyTrend(undefined), null);
});

test("trendHasData distinguishes an empty week from traffic", () => {
  assert.equal(trendHasData(null), false);
  assert.equal(trendHasData(dailyTrend(dailyMetrics([metricsBucket()]))), false);
  assert.equal(trendHasData(dailyTrend(dailyMetrics([metricsBucket({ auditEvents: 1 })]))), true);
});

test("trendDirection applies the 10% dead zone", () => {
  const week = (first, last) => dailyTrend(dailyMetrics([
    metricsBucket({ calls: first, date: "2026-09-20" }),
    metricsBucket({ calls: Math.round((first + last) / 2), date: "2026-09-23" }),
    metricsBucket({ calls: last, date: "2026-09-26" }),
  ]));
  assert.equal(trendDirection(week(10, 11)), "flat");
  assert.equal(trendDirection(week(4, 10)), "up");
  assert.equal(trendDirection(week(10, 4)), "down");
  assert.equal(trendDirection(dailyTrend(dailyMetrics([metricsBucket({ calls: 9 })]))), "flat");
});

test("trendDenyPeak and the peak-day label", () => {
  const trend = dailyTrend(dailyMetrics([
    metricsBucket({ calls: 2, date: "2026-09-24", denyRate: 0.5 }),
    metricsBucket({ calls: 2, date: "2026-09-26", denyRate: 0.75 }),
  ]));
  assert.deepEqual(trendDenyPeak(trend), { date: "2026-09-26", denyRate: 0.75 });
  assert.equal(trendPeakDayLabel("2026-09-26"), "09-26");
  assert.equal(trendPeakDayLabel("not-a-date"), "not-a-date");
});

test("trendSummary reads one honest sentence from the data", () => {
  assert.equal(trendSummary(zh, dailyTrend(dailyMetrics([metricsBucket()]))), null);

  const clean = dailyTrend(dailyMetrics(
    [metricsBucket({ auditEvents: 5, calls: 9, date: "2026-09-26" })],
    { auditEvents: 5, calls: 9, deniedCalls: 0 },
  ));
  assert.match(trendSummary(zh, clean), /9 次调用.*无拒绝/);

  const rising = dailyTrend(dailyMetrics(
    [
      metricsBucket({ calls: 4, date: "2026-09-20", denyRate: 0.25 }),
      metricsBucket({ calls: 10, date: "2026-09-26", denyRate: 0.75 }),
    ],
    { calls: 14, deniedCalls: 3 },
  ));
  const summary = trendSummary(zh, rising);
  assert.match(summary, /14 次调用.*3 次拒绝.*拒绝率上升/);
  assert.match(summary, /峰值在 09-26\(75%\)/);
});

function agent(overrides = {}) {
  return {
    channelConfig: { endpoint: "http://a.example/sse" },
    channelType: "mcp",
    id: "agent-a",
    status: "active",
    ...overrides,
  };
}

function capability(overrides = {}) {
  return { displayName: "Cap", id: "cap-1", key: "mcp.tool.echo", riskLevel: "low", ...overrides };
}

test("preferredProbeTarget prefers active targets with approved capabilities", () => {
  const approved = capability({ discoveryStatus: "approved", id: "cap-2", targetId: "agent-b" });
  const experimental = agent({ channelConfig: { endpoint: "http://old.example/sse" }, id: "agent-x", status: "inactive" });
  const healthy = agent({ channelConfig: { endpoint: "http://b.example/sse" }, id: "agent-b" });
  const preferred = preferredProbeTarget([experimental, healthy], [approved]);
  assert.equal(preferred?.agent.id, "agent-b");
  assert.equal(preferred?.endpoint, "http://b.example/sse");

  assert.equal(preferredProbeTarget([], []), null);
  assert.deepEqual(registeredMcpTargets([experimental, healthy]).map((row) => row.id), ["agent-x", "agent-b"]);
});

test("apiServiceCheck maps contract, reachability and health", () => {
  assert.deepEqual(pick(apiServiceCheck({ apiBase: "http://api", apiHealthMessage: null, contractIssues: [] }), ["fixKeys", "status", "subKey"]), {
    fixKeys: [], status: "ok", subKey: "rd.envcheck.api.ok",
  });
  assert.equal(apiServiceCheck({ apiBase: "http://api", apiHealthMessage: null, contractIssues: ["stale field"] }).status, "error");
  const unreachable = apiServiceCheck({ apiBase: "http://api", apiHealthMessage: "connection refused", contractIssues: [] });
  assert.equal(unreachable.fixKeys[0], "rd.envcheck.api.fixReach");
  assert.equal(unreachable.detail, "connection refused");
});

function pick(row, keys) {
  return Object.fromEntries(keys.map((key) => [key, row[key]]));
}

function probe(overrides = {}) {
  return { checkedAt: "2026-09-27T08:00:00Z", durationMs: 42, endpoint: "http://b.example/sse", httpStatus: 0, status: "ok", targetId: "agent-b", toolCount: 5, ...overrides };
}

test("probeFixGuidance classifies upstream error codes", () => {
  assert.equal(probeFixGuidance(null), null);
  assert.equal(probeFixGuidance(probe()), null);
  const connect = probeFixGuidance(probe({ errorCode: "UPSTREAM_CONNECT_ECONNREFUSED", httpStatus: 502, message: "dial tcp refused", status: "error" }));
  assert.equal(connect.causeKey, "error.upstream.connect");
  assert.equal(connect.detail, "UPSTREAM_CONNECT_ECONNREFUSED: dial tcp refused HTTP 502");
  assert.equal(probeFixGuidance(probe({ errorCode: "UPSTREAM_DNS_NXDOMAIN", status: "error" })).causeKey, "error.upstream.dns");
  assert.equal(probeFixGuidance(probe({ errorCode: "UPSTREAM_TLS_HANDSHAKE", status: "error" })).causeKey, "error.upstream.tls");
  assert.equal(probeFixGuidance(probe({ errorCode: "SOMETHING_ELSE", status: "error" })).causeKey, "error.upstream.generic");
});

test("mcpServiceCheck covers unsupported, unregistered, failed and healthy probes", () => {
  assert.equal(mcpServiceCheck({ endpoint: "", noTarget: false, probe: null, unsupported: true }).status, "unsupported");
  assert.deepEqual(pick(mcpServiceCheck({ endpoint: "", noTarget: true, probe: null, unsupported: false }), ["fixKeys", "status", "subKey"]), {
    fixKeys: ["rd.envcheck.mcp.fixRegister"], status: "warning", subKey: "rd.envcheck.mcp.noTarget",
  });
  const failed = mcpServiceCheck({ endpoint: "http://b.example/sse", noTarget: false, probe: probe({ errorCode: "UPSTREAM_CONNECT_ECONNREFUSED", message: "refused", status: "error" }), unsupported: false });
  assert.equal(failed.status, "error");
  assert.deepEqual(failed.fixKeys, ["error.upstream.connect", "rd.envcheck.mcp.fixProbe"]);
  assert.match(failed.detail, /http:\/\/b\.example\/sse — UPSTREAM_CONNECT/);
  assert.equal(mcpServiceCheck({ endpoint: "http://b.example/sse", noTarget: false, probe: null, unsupported: false }).status, "warning");
  const ok = mcpServiceCheck({ endpoint: "http://b.example/sse", noTarget: false, probe: probe(), unsupported: false });
  assert.equal(ok.status, "ok");
  assert.deepEqual(ok.subParams, { durationMs: 42, endpoint: "http://b.example/sse", tools: 5 });
});

test("corePathCheck requires apply, audit and an allowed call", () => {
  const application = { appliedAt: "2026-09-26T09:00:00Z", id: "app-1" };
  const allowedTrace = { createdAt: "2026-09-26T10:00:00Z", decision: "allowed", id: "tr-1" };
  const appliedAudit = { action: "permission_package.applied", createdAt: "2026-09-26T09:30:00Z", id: "ev-1" };

  assert.equal(corePathCheck({ allowedTrace: null, application: null, appliedAudit: null }).status, "warning");
  const missing = corePathCheck({ allowedTrace: null, application, appliedAudit: null });
  assert.equal(missing.status, "error");
  assert.deepEqual(missing.fixKeys, ["rd.envcheck.corePath.missingTrace", "rd.envcheck.corePath.missingAudit"]);
  assert.equal(corePathCheck({ allowedTrace, application, appliedAudit }).status, "ok");
});

test("envHealthCheck: catalog issues fail, other targets only warn", () => {
  assert.equal(envHealthCheck({ catalogDetail: null, catalogIssues: ["stale"], unreachable: [] }).status, "error");
  assert.equal(envHealthCheck({ catalogDetail: "metadata v3", catalogIssues: [], unreachable: [] }).status, "warning");
  // Each unreachable target carries its classified probe failure so the
  // fix-guidance modal matches the registry probe modal (round 6, #42).
  const unreachable = envHealthCheck({
    catalogDetail: null,
    catalogIssues: [],
    unreachable: [
      { detail: "UPSTREAM_CONNECT_ERROR: dial tcp 127.0.0.1:9: connection refused", endpoint: "http://x", name: "x" },
      { detail: "", endpoint: "http://y", name: "y" },
    ],
  });
  assert.equal(unreachable.status, "warning");
  assert.deepEqual(unreachable.subParams, { count: 2 });
  assert.equal(
    unreachable.detail,
    "x (http://x) — UPSTREAM_CONNECT_ERROR: dial tcp 127.0.0.1:9: connection refused; y (http://y)",
  );
  assert.equal(envHealthCheck({ catalogDetail: null, catalogIssues: [], unreachable: [] }).status, "ok");
});

test("envCheckSummary counts abnormal rows", () => {
  const rows = [apiServiceCheck({ apiBase: "a", apiHealthMessage: null, contractIssues: [] }), mcpServiceCheck({ endpoint: "", noTarget: true, probe: null, unsupported: false })];
  assert.deepEqual(envCheckSummary(rows), { abnormal: 1, error: 0, warning: 1 });
});

function auditEvent(overrides = {}) {
  return { action: "agent.created", actor: "admin:op", createdAt: "2026-09-26T08:00:00Z", id: "ev-1", resourceId: "agent-a", resourceType: "agent", ...overrides };
}

function traceEvent(overrides = {}) {
  return { capabilityId: "mcp.tool.echo", createdAt: "2026-09-26T09:00:00Z", decision: "allowed", id: "tr-1", subjectId: "user:support", ...overrides };
}

test("traceResultKind separates denied policy calls from upstream failures", () => {
  assert.equal(traceResultKind(traceEvent()), "success");
  assert.equal(traceResultKind(traceEvent({ decision: "denied" })), "denied");
  assert.equal(traceResultKind(traceEvent({ upstreamError: "dial refused" })), "failed");
});

test("auditTimelineRows merges newest first", () => {
  const rows = auditTimelineRows(
    [auditEvent({ createdAt: "2026-09-26T08:00:00Z" }), auditEvent({ createdAt: "2026-09-25T08:00:00Z", id: "ev-0" })],
    [traceEvent({ createdAt: "2026-09-27T08:00:00Z" }), traceEvent({ createdAt: "2026-09-26T08:30:00Z", decision: "denied", id: "tr-0", upstreamError: "" })],
  );
  assert.deepEqual(rows.map((row) => row.id), ["tr-1", "tr-0", "ev-1", "ev-0"]);
  assert.equal(rows[0].resourceType, "trace");
  assert.equal(rows[0].operation, "call");
  assert.equal(rows[2].resource, "agent-a");
});

test("filterAuditTimeline narrows by result and type", () => {
  const rows = auditTimelineRows(
    [auditEvent()],
    [
      traceEvent({ createdAt: "2026-09-26T10:00:00Z", decision: "denied", id: "tr-denied" }),
      traceEvent({ createdAt: "2026-09-26T11:00:00Z", id: "tr-failed", upstreamError: "boom" }),
    ],
  );
  assert.deepEqual(filterAuditTimeline(rows, { result: "all", type: "all" }).map((row) => row.id), ["tr-failed", "tr-denied", "ev-1"]);
  assert.deepEqual(filterAuditTimeline(rows, { result: "denied", type: "trace" }).map((row) => row.id), ["tr-denied"]);
  assert.deepEqual(filterAuditTimeline(rows, { result: "failed", type: "trace" }).map((row) => row.id), ["tr-failed"]);
  assert.deepEqual(filterAuditTimeline(rows, { result: "success", type: "agent" }).map((row) => row.id), ["ev-1"]);
});

test("auditResourceTypes lists distinct audit kinds only", () => {
  const rows = auditTimelineRows([auditEvent(), auditEvent({ id: "ev-2", resourceType: "capability" })], [traceEvent()]);
  assert.deepEqual(auditResourceTypes(rows), ["agent", "capability"]);
});

test("todayAuditCount counts from local midnight", () => {
  const now = new Date(2026, 8, 27, 12, 0, 0);
  const rows = auditTimelineRows([
    auditEvent({ createdAt: new Date(2026, 8, 27, 1, 0, 0).toISOString(), id: "today" }),
    auditEvent({ createdAt: new Date(2026, 8, 25, 1, 0, 0).toISOString(), id: "old" }),
  ], []);
  assert.equal(todayAuditCount(rows, now), 1);
});

test("auditRangeSince anchors today at local midnight", () => {
  const now = new Date(2026, 8, 27, 15, 30, 0);
  const startOfDay = new Date(2026, 8, 27, 0, 0, 0).getTime();
  assert.equal(Date.parse(auditRangeSince("today", now)), startOfDay);
  assert.equal(Date.parse(auditRangeSince("week", now)), startOfDay - 6 * 86400000);
  assert.equal(Date.parse(auditRangeSince("month", now)), startOfDay - 29 * 86400000);
});

test("auditResourceTypeLabelKey reuses legacy labels and flags the one gap", () => {
  assert.equal(auditResourceTypeLabelKey("agent"), "auditResource.agent");
  assert.equal(auditResourceTypeLabelKey("permission_package_approval_request"), "auditResource.permission_package_approval_request");
  assert.equal(auditResourceTypeLabelKey("access_handoff_token"), "rd.audit.type.access_handoff_token");
  assert.equal(auditResourceTypeLabelKey("mystery_kind"), null);
});

function approvalRequest(overrides = {}) {
  return {
    allowedCapabilityIds: ["cap-1", "cap-2"],
    allowedCapabilityKeys: ["mcp.tool.echo", "http.invoice.write"],
    allowedCapabilityFingerprints: [],
    callerInstanceId: "agent-caller",
    createdAt: "2026-09-26T08:00:00Z",
    draftId: "draft-1",
    id: "req-1",
    policyGate: {},
    policyVersion: 1,
    status: "pending",
    targetId: "agent-target",
    templateId: "tpl-1",
    templateVersion: 1,
    tenantId: "t1",
    updatedAt: "2026-09-26T08:00:00Z",
    workspaceId: "w1",
    ...overrides,
  };
}

const template = {
  guardrails: [
    { capabilityKey: "mcp.tool.echo", expectedDecision: "allow" },
    { capabilityKey: "http.invoice.write", expectedDecision: "deny" },
  ],
  id: "tpl-1",
  name: "Support package",
  summary: "",
  version: 1,
};

const registry = [
  capability({ displayName: "Echo", id: "cap-1", key: "mcp.tool.echo" }),
  capability({ displayName: "Invoice write", id: "cap-2", key: "http.invoice.write", riskLevel: "critical" }),
];

test("approvalCapabilityRows reads allow/deny from the template guardrails", () => {
  const rows = approvalCapabilityRows(approvalRequest({ requestedCapabilityId: "cap-1" }), template, registry);
  assert.deepEqual(
    rows.map((row) => ({ allowed: row.allowed, isRequested: row.isRequested, key: row.key })),
    [
      { allowed: true, isRequested: true, key: "mcp.tool.echo" },
      { allowed: false, isRequested: false, key: "http.invoice.write" },
    ],
  );
  assert.equal(rows[0].capability?.displayName, "Echo");

  const unknownRegistry = approvalCapabilityRows(approvalRequest(), template, []);
  assert.equal(unknownRegistry[0].capability, null);
  assert.equal(unknownRegistry[0].key, "mcp.tool.echo");
});

test("approvalCapabilityRows without a template shows only allowed keys", () => {
  const rows = approvalCapabilityRows(approvalRequest(), null, registry);
  assert.equal(rows.length, 2);
  assert.ok(rows.every((row) => row.allowed));
});

test("approvalCapabilityRows merges snapshot allowed keys with deny-only guardrails", () => {
  // Built-in templates deny by guardrail only; the concrete allow list lives
  // in the request snapshot and must still reach the approver.
  const denyOnly = {
    guardrails: [{ capabilityKey: "delete-ticket", expectedDecision: "deny" }],
    id: "tpl-1",
    name: "Support package",
    summary: "",
    version: 1,
  };
  const supportRegistry = [
    capability({ displayName: "Update ticket", id: "cap-u", key: "update-ticket" }),
    capability({ displayName: "Delete ticket", id: "cap-d", key: "delete-ticket", riskLevel: "critical" }),
  ];
  const rows = approvalCapabilityRows(
    approvalRequest({
      allowedCapabilityIds: ["cap-u"],
      allowedCapabilityKeys: ["update-ticket"],
      requestedCapabilityId: "cap-u",
    }),
    denyOnly,
    supportRegistry,
  );
  assert.deepEqual(
    rows.map((row) => ({ allowed: row.allowed, isRequested: row.isRequested, key: row.key })),
    [
      { allowed: false, isRequested: false, key: "delete-ticket" },
      { allowed: true, isRequested: true, key: "update-ticket" },
    ],
  );
  assert.deepEqual(approvalRiskSummary(rows), { deniedCount: 1, highRiskCount: 1 });

  // Keys already covered by a guardrail are not duplicated from the snapshot.
  const deduped = approvalCapabilityRows(approvalRequest(), template, registry);
  assert.deepEqual(deduped.map((row) => row.key), ["mcp.tool.echo", "http.invoice.write"]);
});

test("approvalList sorts newest first and honors effectiveStatus", () => {
  const entries = approvalList([
    approvalRequest({ createdAt: "2026-09-25T08:00:00Z", id: "req-old" }),
    approvalRequest({ createdAt: "2026-09-26T08:00:00Z", id: "req-new", status: "approved" }),
    approvalRequest({ createdAt: "2026-09-26T09:00:00Z", effectiveStatus: "expired", id: "req-expired", status: "pending" }),
  ]);
  assert.deepEqual(entries.map((entry) => entry.request.id), ["req-expired", "req-new", "req-old"]);
  assert.deepEqual(entries.map((entry) => entry.status), ["expired", "approved", "pending"]);
});

test("approvalTabCounts and filterApprovalList keep every record visible", () => {
  const entries = approvalList([
    approvalRequest({ createdAt: "2026-09-23T08:00:00Z", id: "r1" }),
    approvalRequest({ createdAt: "2026-09-24T08:00:00Z", id: "r2", status: "approved" }),
    approvalRequest({ createdAt: "2026-09-25T08:00:00Z", id: "r3", status: "rejected" }),
    approvalRequest({ createdAt: "2026-09-26T08:00:00Z", effectiveStatus: "withdrawn", id: "r4" }),
  ]);
  const counts = approvalTabCounts(entries);
  assert.equal(counts.pending, 1);
  assert.equal(counts.approved, 1);
  assert.equal(counts.rejected, 1);
  assert.equal(counts.all, 4);
  assert.equal(counts.withdrawn, 1);
  assert.deepEqual(filterApprovalList(entries, "all").map((entry) => entry.request.id), ["r4", "r3", "r2", "r1"]);
  assert.deepEqual(filterApprovalList(entries, "withdrawn").map((entry) => entry.request.id), ["r4"]);
});

test("approvalRiskSummary and notice key ordering", () => {
  const rows = approvalCapabilityRows(approvalRequest(), template, registry);
  const summary = approvalRiskSummary(rows);
  assert.deepEqual(summary, { deniedCount: 1, highRiskCount: 1 });
  assert.equal(approvalNoticeKey(summary), "leastPrivilege");
  assert.equal(approvalNoticeKey({ deniedCount: 0, highRiskCount: 2 }), "highRisk");
  assert.equal(approvalNoticeKey({ deniedCount: 0, highRiskCount: 0 }), "matchesTemplate");
});
