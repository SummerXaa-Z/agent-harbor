import assert from "node:assert/strict";
import test from "node:test";

import {
  appendLocalCapabilityGrantChain,
  capabilityGrantBlockerKey,
  defaultCapabilityGrantForm,
  mergeCapabilitiesForTarget,
  normalizeCapabilityGrantForm,
  shallowEqualCapabilityForm,
  shouldUseLocalCapabilityFallback,
  validateCapabilityGrantChain,
} from "../src/capabilityGrantChain.ts";
import { capabilityDomainSegments, capabilityGrantRows, normalizeRiskLevel } from "../src/redesign/model/capabilityCatalog.ts";
import { adminBoundaryRows, adminBoundarySummary, createAdminIdentityRequestReady, demoAdminSessionVisible } from "../src/redesign/model/adminBoundary.ts";
import { normalizeTemplateId, templateRuleRows } from "../src/redesign/model/policyTemplates.ts";
import { parsePriorityInput, routePolicyRows, routeTypeOptions } from "../src/redesign/model/routePolicyCatalog.ts";
import { agentEndpoint, registryRows, registrySummary } from "../src/redesign/model/registryCatalog.ts";
import {
  normalizeTenantDetailTab,
  tenantDirectoryRows,
  tenantDirectorySummary,
} from "../src/redesign/model/tenantDirectory.ts";
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

function consoleData(overrides = {}) {
  return {
    accessGrants: [],
    agents: [],
    apiBase: "http://127.0.0.1:9090",
    auditEvents: [],
    capabilityAssignmentsLoadedFromApi: false,
    capabilities: [],
    capabilitiesLoadedFromApi: false,
    channels: [],
    entitlements: [],
    evidenceRuns: [],
    grantsLoadedFromApi: false,
    instanceAssignments: [],
    loadedFromApi: false,
    providers: [],
    routePolicies: [],
    routePoliciesLoadedFromApi: false,
    setupLoadedFromApi: false,
    systemMetrics: [],
    tenantEntitlements: [],
    tenants: [],
    traces: [],
    workspaceAssignments: [],
    ...overrides,
  };
}

test("defaultCapabilityGrantForm seeds tenant and workspace from the scope", () => {
  assert.deepEqual(defaultCapabilityGrantForm({ tenantId: "t1", workspaceId: "w1" }), {
    callerInstanceId: "",
    capabilityId: "",
    subjectSelector: "user:support-*",
    targetId: "",
    tenantId: "t1",
    workspaceId: "w1",
  });
});

test("normalizeCapabilityGrantForm fills target, capability and caller in order", () => {
  const caller = agent({ channelConfig: {}, channelType: "local", id: "agt-caller", name: "Caller", tenantId: "t1", workspaceId: "w1" });
  const other = agent({ channelConfig: {}, channelType: "local", id: "agt-other", name: "Other", tenantId: "t9", workspaceId: "w9" });
  const target = agent({ id: "agt-b", name: "Target B" });
  const caps = [
    capability({ id: "cap-orphan", targetId: "agt-z" }),
    capability({ displayName: "B1", id: "cap-b1", targetId: "agt-b" }),
    capability({ displayName: "B2", id: "cap-b2", targetId: "agt-b" }),
  ];
  const next = normalizeCapabilityGrantForm(
    { callerInstanceId: "", capabilityId: "", subjectSelector: "user:support-*", targetId: "", tenantId: "t1", workspaceId: "w1" },
    consoleData({ agents: [other, target, caller], capabilities: caps }),
  );
  assert.equal(next.targetId, "agt-b");
  assert.equal(next.capabilityId, "cap-b1");
  assert.equal(next.callerInstanceId, "agt-caller");
});

test("normalizeCapabilityGrantForm keeps user-chosen values and dedupes equal forms", () => {
  const target = agent({});
  const chosen = capability({ id: "cap-2", targetId: "agt-a" });
  const data = consoleData({
    agents: [target],
    capabilities: [capability({ id: "cap-1" }), chosen],
  });
  const form = { callerInstanceId: "", capabilityId: "cap-2", subjectSelector: "s", targetId: "agt-a", tenantId: "t1", workspaceId: "w1" };
  const next = normalizeCapabilityGrantForm(form, data);
  assert.equal(next.capabilityId, "cap-2");
  assert.equal(next, form, "an unchanged form returns the same reference");
  assert.ok(shallowEqualCapabilityForm(form, next));
});

test("validateCapabilityGrantChain enforces order and rejects wildcard subjects", () => {
  const cap = capability();
  const base = { callerInstanceId: "agt-caller", capabilityId: "cap-1", subjectSelector: "user:support-*", targetId: "agt-a", tenantId: "t1", workspaceId: "w1" };
  assert.deepEqual(
    validateCapabilityGrantChain({ ...base, capabilityId: "missing" }, [cap]),
    { messageKey: "message.validationCapabilityRequired" },
  );
  assert.deepEqual(
    validateCapabilityGrantChain({ ...base, workspaceId: " " }, [cap]),
    { messageKey: "message.validationTenantWorkspaceCaller" },
  );
  assert.deepEqual(
    validateCapabilityGrantChain({ ...base, subjectSelector: "*" }, [cap]),
    { messageKey: "message.validationSubjectSelectorRequired" },
  );
  const ok = validateCapabilityGrantChain(base, [cap]);
  assert.equal("plan" in ok && ok.plan.capability.id, "cap-1");
  assert.equal("plan" in ok && ok.plan.dataScopes.length, 0);
});

test("capabilityGrantBlockerKey splits the caller out of the tenant/workspace message", () => {
  const cap = capability();
  const base = { callerInstanceId: "c", capabilityId: "cap-1", subjectSelector: "s", targetId: "agt-a", tenantId: "t1", workspaceId: "w1" };
  assert.equal(capabilityGrantBlockerKey(base, null), "message.validationCapabilityRequired");
  assert.equal(capabilityGrantBlockerKey({ ...base, tenantId: "" }, cap), "message.validationTenantWorkspaceCaller");
  assert.equal(capabilityGrantBlockerKey({ ...base, callerInstanceId: "" }, cap), "message.capabilityGrantCallerRequired");
  assert.equal(capabilityGrantBlockerKey({ ...base, subjectSelector: "*" }, cap), "message.validationSubjectSelectorRequired");
  assert.equal(capabilityGrantBlockerKey(base, cap), null);
});

test("mergeCapabilitiesForTarget replaces only the refreshed target", () => {
  const mine = capability({ id: "cap-old", targetId: "agt-a" });
  const other = capability({ id: "cap-other", targetId: "agt-b" });
  const fresh = capability({ displayName: "Fresh", id: "cap-new", targetId: "agt-a" });
  assert.deepEqual(
    mergeCapabilitiesForTarget([mine, other], [fresh], "agt-a").map((item) => item.id),
    ["cap-other", "cap-new"],
  );
});

test("shouldUseLocalCapabilityFallback only fires for sample capabilities", () => {
  const networkError = new Error("Failed to fetch");
  assert.equal(shouldUseLocalCapabilityFallback(networkError, consoleData()), true);
  assert.equal(shouldUseLocalCapabilityFallback(networkError, consoleData({ capabilitiesLoadedFromApi: true })), false);
  assert.equal(shouldUseLocalCapabilityFallback(new Error("boom"), consoleData()), false);
  assert.equal(shouldUseLocalCapabilityFallback(networkError, null), true, "null data keeps the legacy fallback verdict; the hook gates on data");
});

test("appendLocalCapabilityGrantChain is idempotent and approves the capability", () => {
  const cap = capability({ discoveryStatus: "pending_review", id: "cap-1" });
  const data = consoleData({ agents: [agent({}), capability].length ? [agent({})] : [], capabilities: [cap] });
  const form = { callerInstanceId: "agt-caller", capabilityId: "cap-1", subjectSelector: "user:support-*", targetId: "agt-a", tenantId: "t1", workspaceId: "w1" };
  const scope = { tenantId: "t-default", workspaceId: "w-default" };

  const first = appendLocalCapabilityGrantChain(data, cap, form, [], scope);
  assert.equal(first.tenantEntitlements.length, 1);
  assert.equal(first.workspaceAssignments.length, 1);
  assert.equal(first.instanceAssignments.length, 1);
  assert.equal(first.capabilities[0].discoveryStatus, "approved");
  const entitlementId = first.tenantEntitlements[0].id;

  const second = appendLocalCapabilityGrantChain(first, cap, form, [], scope);
  assert.equal(second.tenantEntitlements.length, 1);
  assert.equal(second.workspaceAssignments.length, 1);
  assert.equal(second.instanceAssignments.length, 1);
  assert.equal(second.tenantEntitlements[0].id, entitlementId, "the same entitlement is reused, not duplicated");
});

test("appendLocalCapabilityGrantChain falls back to the default scope", () => {
  const cap = capability();
  const data = consoleData({ capabilities: [cap] });
  const form = { callerInstanceId: "agt-caller", capabilityId: "cap-1", subjectSelector: "s", targetId: "", tenantId: "", workspaceId: "" };
  const scope = { tenantId: "t-default", workspaceId: "w-default" };
  const next = appendLocalCapabilityGrantChain(data, cap, form, [], scope);
  assert.equal(next.tenantEntitlements[0].tenantId, "t-default");
  assert.equal(next.workspaceAssignments[0].workspaceId, "w-default");
});

test("tenantDirectoryRows derives tenant x workspace pairs with grant counts", () => {
  const rows = tenantDirectoryRows({
    agents: [
      agent({ id: "a1", tenantId: "t1", workspaceId: "w1" }),
      agent({ channelConfig: {}, channelType: "local", id: "a2", tenantId: "t1", workspaceId: "w1" }),
    ],
    entitlements: [
      { capabilityId: "c1", createdAt: "", effect: "allow", id: "e1", priority: 50, status: "enabled", targetId: "agt-a", tenantId: "t1", updatedAt: "" },
      { capabilityId: "c2", createdAt: "", effect: "deny", id: "e2", priority: 50, status: "enabled", targetId: "agt-a", tenantId: "t1", updatedAt: "" },
    ],
    tenants: [
      { createdAt: "", id: "t1", level: 0, name: "One", status: "active", updatedAt: "" },
      { createdAt: "", id: "t2", level: 0, name: "Two", status: "active", updatedAt: "" },
    ],
    workspaceAssignments: [
      { createdAt: "", effect: "allow", id: "wa1", status: "enabled", tenantEntitlementId: "e1", tenantId: "t1", updatedAt: "", workspaceId: "w2" },
    ],
  });
  assert.deepEqual(
    rows.map((row) => [row.tenantName, row.workspaceId, row.agentCount, row.allowCount, row.denyCount]),
    [["One", "w1", 2, 1, 1], ["One", "w2", 0, 1, 1], ["Two", "", 0, 0, 0]],
  );
  assert.deepEqual(tenantDirectorySummary(rows, 12), { subjects: 12, tenants: 2, workspaces: 2 });
  assert.equal(normalizeTenantDetailTab("profile"), "profile");
  assert.equal(normalizeTenantDetailTab("bogus"), "org");
});

test("registryRows classifies callers and targets with endpoints", () => {
  const rows = registryRows([
    agent({ channelConfig: {}, channelType: "local", id: "caller", name: "Zed caller" }),
    agent({ id: "tgt", name: "Alpha target" }),
    agent({ channelConfig: {}, channelType: "local", id: "caller2", name: "Ann caller", status: "disabled" }),
  ]);
  assert.deepEqual(rows.map((row) => [row.kind, row.agent.name]), [
    ["caller", "Ann caller"],
    ["caller", "Zed caller"],
    ["target", "Alpha target"],
  ]);
  assert.equal(agentEndpoint(rows[2].agent), "http://a.example/sse");
  // Active counts exclude disabled agents so the header chip can say how many
  // registered entries are actually live (round 4, #31).
  assert.deepEqual(registrySummary(rows), { activeCallers: 1, activeTargets: 1, callers: 2, targets: 1 });
});

test("capabilityDomainSegments sorts domains and buckets the unclassified", () => {
  const segments = capabilityDomainSegments([
    capability({ dataDomains: ["support"] }),
    capability({ dataDomains: ["support"], id: "cap-2" }),
    capability({ dataScopes: [{ dataDomain: "crm" }], id: "cap-3" }),
    capability({ id: "cap-4" }),
  ]);
  assert.deepEqual(
    segments.map((segment) => [segment.domain, segment.count, segment.fraction]),
    [["support", 2, 0.5], ["crm", 1, 0.25], ["", 1, 0.25]],
  );
  assert.deepEqual(capabilityDomainSegments([]), []);
  assert.equal(normalizeRiskLevel("high"), "high");
  assert.equal(normalizeRiskLevel("extreme"), null);
});

test("capabilityGrantRows collapses the three grant-chain layers", () => {
  const rows = capabilityGrantRows({
    entitlements: [
      { capabilityId: "cap-1", createdAt: "", effect: "allow", id: "e1", priority: 50, status: "enabled", targetId: "agt-a", tenantId: "t1", updatedAt: "" },
    ],
    instanceAssignments: [
      { callerInstanceId: "c1", createdAt: "", effect: "allow", id: "i1", status: "enabled", subjectSelector: "user:support-*", tenantId: "t1", updatedAt: "", workspaceAssignmentId: "wa1", workspaceId: "w1" },
      { callerInstanceId: "c1", createdAt: "", effect: "allow", id: "i2", status: "enabled", subjectSelector: "user:support-*", tenantId: "t1", updatedAt: "", workspaceAssignmentId: "wa1", workspaceId: "w1" },
      { callerInstanceId: "c2", createdAt: "", effect: "allow", id: "i3", status: "enabled", subjectSelector: "role:agent", tenantId: "t1", updatedAt: "", workspaceAssignmentId: "wa2", workspaceId: "w2" },
    ],
    workspaceAssignments: [
      { createdAt: "", effect: "allow", id: "wa1", status: "enabled", tenantEntitlementId: "e1", tenantId: "t1", updatedAt: "", workspaceId: "w1" },
      { createdAt: "", effect: "allow", id: "wa2", status: "enabled", tenantEntitlementId: "e1", tenantId: "t1", updatedAt: "", workspaceId: "w2" },
    ],
  });
  assert.equal(rows.length, 1);
  assert.equal(rows[0].workspaceCount, 2);
  assert.equal(rows[0].callerCount, 2);
  assert.deepEqual(rows[0].subjectSelectors, ["user:support-*", "role:agent"]);
});

test("templateRuleRows and normalizeTemplateId", () => {
  const template = permissionPackageTemplates[0];
  const rules = templateRuleRows(template);
  assert.equal(rules.length, template.guardrails.length);
  assert.equal(typeof rules[0].capabilityKey, "string");
  assert.ok(rules.every((rule) => rule.decision === "allow" || rule.decision === "deny"));
  assert.equal(normalizeTemplateId(undefined, permissionPackageTemplates), permissionPackageTemplates[0].id);
  assert.equal(normalizeTemplateId("nope", permissionPackageTemplates), permissionPackageTemplates[0].id);
  const last = permissionPackageTemplates[permissionPackageTemplates.length - 1].id;
  assert.equal(normalizeTemplateId(last, permissionPackageTemplates), last);
  assert.equal(normalizeTemplateId("x", []), null);
});

test("routePolicyRows sorts by priority and parses priorities", () => {
  const policies = [
    { callerAgentId: "c1", createdAt: "", effect: "allow", id: "r-low", priority: 10, routeType: "mcp", status: "enabled", targetAgentId: "tgt", tenantId: "t1", updatedAt: "", workspaceId: "w1" },
    { callerAgentId: "c1", createdAt: "", effect: "allow", id: "r-top", priority: 100, routeKey: "tools/call", routeType: "mcp", status: "enabled", targetAgentId: "tgt", tenantId: "t1", updatedAt: "", workspaceId: "w1" },
  ];
  const name = (id) => (id === "c1" ? "Caller" : id === "tgt" ? "Target" : id);
  const rows = routePolicyRows(policies, name);
  assert.deepEqual(rows.map((row) => row.policy.id), ["r-top", "r-low"]);
  assert.equal(rows[0].callerName, "Caller");
  assert.equal(rows[0].targetName, "Target");
  assert.equal(rows[0].routeKey, "tools/call");
  assert.deepEqual(parsePriorityInput("80"), { ok: true, value: 80 });
  assert.equal(parsePriorityInput("-1").ok, false);
  assert.equal(parsePriorityInput("1.5").ok, false);
  assert.equal(parsePriorityInput("").ok, false);
  assert.deepEqual(routeTypeOptions(policies), ["mcp", "openapi"]);
});

test("adminBoundary rows, summary and demo session gating", () => {
  const identities = [
    { actor: "admin:op", createdAt: "", displayName: "Ops admin", id: "id-1", role: "platform_admin", source: "managed", status: "active", updatedAt: "" },
    { actor: "admin:boot", createdAt: "", displayName: "", id: "id-2", role: "tenant_admin", source: "bootstrap", status: "active", tenantId: "t1", updatedAt: "" },
    { actor: "admin:x", createdAt: "", displayName: "X", id: "id-3", role: "security_reviewer", source: "managed", status: "disabled", tenantId: "t1", updatedAt: "" },
  ];
  const rows = adminBoundaryRows(identities);
  assert.deepEqual(rows.map((row) => row.displayName), ["admin:boot", "Ops admin", "X"]);
  assert.deepEqual(adminBoundarySummary(identities), { active: 2, bootstrap: 1, disabled: 1, managed: 2 });
  assert.equal(demoAdminSessionVisible({ authenticated: false, requiresLogin: false }), true);
  assert.equal(demoAdminSessionVisible({ authenticated: true, requiresLogin: false }), true, "local-dev is auto-authenticated in demo mode");
  assert.equal(demoAdminSessionVisible({ authenticated: true, requiresLogin: true }), false);
  assert.equal(demoAdminSessionVisible(null), false);
  assert.equal(createAdminIdentityRequestReady({ actor: " a ", role: "platform_admin", tenantId: "" }), true);
  assert.equal(createAdminIdentityRequestReady({ actor: "a", role: "tenant_admin", tenantId: "" }), false);
  assert.equal(createAdminIdentityRequestReady({ actor: "", role: "tenant_admin", tenantId: "t1" }), false);
});
