import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import test from "node:test";

import { permissionPackageTemplates } from "../src/permissionPackages.ts";
import {
  ACCESS_CONTEXT_STORAGE_KEY,
  accessContextFromApplication,
  accessContextFromApprovalRequest,
  accessContextOptions,
  accessContextRouteParams,
  emptyAccessContext,
  latestApplication,
  parseStoredAccessContext,
  readinessFilterFromContext,
  resolveAccessContext,
  serializeAccessContext,
} from "../src/redesign/model/accessContext.ts";
import {
  approvalReconcileFilter,
  approvalScopeKey,
  effectiveApproval,
  isApprovalAlreadyPendingError,
  isApprovalNotRequiredError,
  isCapabilityChangedError,
  permissionChangePresentation,
  permissionChangeState,
  pickPendingApproval,
} from "../src/redesign/model/approvalStateMachine.ts";
import { buildDecisionChain, decisionChainLayers, decisionRemediation } from "../src/redesign/model/decisionChain.ts";
import {
  absolutizeHandoffConfig,
  defaultTokenTtl,
  goLiveLegs,
  handoffShellSnippet,
  readinessCheckCount,
  shellQuote,
  tokenTtlOptions,
} from "../src/redesign/model/goLive.ts";
import { rankTemplates, recommendedTemplateId, templateMatch } from "../src/redesign/model/templateMatch.ts";
import { splitBlockedCapabilities, unclassifiedCapabilities } from "../src/redesign/model/capabilityGovernance.ts";
import { buildRuntimeValidationReadiness, classifyRuntimeValidationRun } from "../src/redesign/model/runtimeValidation.ts";
import {
  DEMO_ACTOR,
  keyStatus,
  myPermissions,
  myRequestCounts,
  myRequests,
  myResources,
  onboardingProgress,
  workbenchActor,
} from "../src/redesign/model/userWorkbench.ts";

const agents = [
  { id: "caller-a", tenantId: "t1", workspaceId: "ws1", name: "Support bot", channelType: "local", status: "active", credentialVersion: 1, createdAt: "", updatedAt: "2026-09-01T00:00:00Z" },
  { id: "target-a", tenantId: "t1", workspaceId: "ws1", name: "Ticket MCP", channelType: "mcp", status: "active", credentialVersion: 1, createdAt: "", updatedAt: "2026-09-02T00:00:00Z" },
  { id: "caller-b", tenantId: "t2", workspaceId: "ws2", name: "Other bot", channelType: "local", status: "active", credentialVersion: 1, createdAt: "", updatedAt: "2026-09-03T00:00:00Z" },
];

function capability(overrides) {
  return {
    action: "read",
    dataDomains: ["support"],
    discoveryStatus: "approved",
    displayName: overrides.key,
    enforcementMode: "gateway",
    riskLevel: "low",
    sensitivity: "internal",
    targetId: "target-a",
    type: "mcp_tool",
    version: 1,
    discoveredAt: "",
    updatedAt: "",
    ...overrides,
  };
}

const capabilities = [
  capability({ id: "cap-read", key: "search-tickets" }),
  capability({ id: "cap-write", key: "update-ticket", action: "write", riskLevel: "medium" }),
  capability({ id: "cap-delete", key: "delete-ticket", action: "delete", riskLevel: "high" }),
];

function application(overrides = {}) {
  return {
    id: "app-1",
    draftId: "d1",
    templateId: "support-ticket-triage",
    templateVersion: 2,
    tenantId: "t1",
    workspaceId: "ws1",
    targetId: "target-a",
    callerInstanceId: "caller-a",
    subjectSelector: "user:support-*",
    requestText: "Open ticket access",
    region: "east",
    allowedCapabilityIds: ["cap-read"],
    allowedCapabilityKeys: ["search-tickets"],
    tenantEntitlementIds: ["te-1"],
    workspaceAssignmentIds: ["wa-1"],
    instanceAssignmentIds: ["ia-1"],
    appliedAt: "2026-09-20T00:00:00Z",
    ...overrides,
  };
}

function approval(overrides = {}) {
  return {
    id: "apr-1",
    draftId: "d1",
    templateId: "support-ticket-triage",
    templateVersion: 2,
    policyVersion: 1,
    tenantId: "t1",
    workspaceId: "ws1",
    targetId: "target-a",
    callerInstanceId: "caller-a",
    subjectSelector: "user:support-*",
    requestText: "Open ticket access",
    region: "east",
    allowedCapabilityIds: ["cap-read", "cap-write"],
    allowedCapabilityKeys: [],
    allowedCapabilityFingerprints: [],
    policyGate: { canApplyDirectly: false, decision: "approval_required", policyVersion: 1, reasons: [], nextActions: [] },
    status: "pending",
    requestedBy: DEMO_ACTOR,
    createdAt: "2026-09-21T00:00:00Z",
    updatedAt: "2026-09-21T00:00:00Z",
    expiresAt: "2026-10-21T00:00:00Z",
    ...overrides,
  };
}

function readiness(status, checks = []) {
  return {
    status,
    summary: { readyCount: checks.filter((check) => check.severity === "passed").length, warningCount: 0, blockingCount: 0 },
    checks,
    runtimeEvidence: {},
    auditEvidence: {},
    nextActions: [],
    generatedAt: "",
  };
}

function preview({ status = "awaiting_approval", canApplyDirectly = false, applied = false, approvalRequest, latestApplication: latest, productionReadiness } = {}) {
  return {
    draft: {
      allowedCapabilities: [],
      blockedCapabilities: [],
      policyGate: { canApplyDirectly, decision: canApplyDirectly ? "allow" : "approval_required", policyVersion: 1, reasons: [], nextActions: [] },
      readiness: { canApply: status !== "needs_input", missingFields: [], warnings: [] },
      template: permissionPackageTemplates[1],
    },
    approvalRequest,
    latestApplication: latest,
    productionReadiness,
    summary: {
      status,
      primaryActionCode: "create_approval_request",
      approvalRequired: !canApplyDirectly,
      canApply: status === "ready_to_apply",
      applied,
      readinessReadyCount: 3,
      readinessTotalCount: 10,
      steps: [],
    },
    generatedAt: "",
  };
}

test("decision chain marks every node passed when access is allowed", () => {
  const chain = buildDecisionChain({
    outcome: "allowed",
    decision: { allowed: true, source: "instance_assignment", reason: "capability assignment matched" },
    evidence: decisionChainLayers.map((layer) => ({ layer, status: "matched", message: "" })),
    nextActionCodes: ["no_change_required"],
    nextActions: [],
    request: {},
    summary: "",
  });
  assert.equal(chain.nodes.length, 6);
  assert.ok(chain.nodes.every((node) => node.state === "passed"));
  assert.equal(chain.terminatingLayer, null);
  assert.equal(chain.passedCount, 6);
});

test("decision chain names the terminating node and leaves later nodes pending", () => {
  const result = {
    outcome: "denied",
    decision: { allowed: false, source: "tenant_entitlement", reason: "tenant has no entitlement for capability" },
    evidence: [
      { layer: "caller_instance", status: "matched", message: "" },
      { layer: "target", status: "matched", message: "" },
      { layer: "capability", status: "matched", message: "" },
      { layer: "tenant_entitlement", status: "missing", message: "" },
    ],
    nextActionCodes: ["use_permission_package"],
    nextActions: ["Use the permission package flow"],
    request: {},
    summary: "",
  };
  const chain = buildDecisionChain(result);
  assert.equal(chain.terminatingLayer, "tenant_entitlement");
  assert.deepEqual(chain.nodes.map((node) => node.state), ["passed", "passed", "passed", "failed", "pending", "pending"]);
  assert.equal(chain.nodes[3].labelKey, "ask.recordLayer.tenant_entitlement");
  const remediation = decisionRemediation(result);
  assert.equal(remediation.code, "use_permission_package");
  assert.equal(remediation.labelKey, "ask.nextAction.usePermissionPackage");
  assert.equal(remediation.requestable, true);
});

test("decision chain falls back to the decision source when no record fails", () => {
  const chain = buildDecisionChain({
    outcome: "denied",
    decision: { allowed: false, source: "capability", reason: "capability is not approved" },
    evidence: [{ layer: "caller_instance", status: "matched", message: "" }],
    nextActionCodes: ["approve_capability"],
    nextActions: [],
    request: {},
    summary: "",
  });
  assert.equal(chain.terminatingLayer, "capability");
  assert.deepEqual(chain.nodes.map((node) => node.state), ["passed", "pending", "failed", "pending", "pending", "pending"]);
  assert.equal(decisionRemediation({ outcome: "denied", nextActionCodes: ["approve_capability"], nextActions: [] }).requestable, false);
  assert.equal(decisionRemediation({ outcome: "allowed", nextActions: [] }).code, "no_change_required");
});

test("template match scores recommended, partial and none", () => {
  const triage = permissionPackageTemplates.find((template) => template.id === "support-ticket-triage");
  const sales = permissionPackageTemplates.find((template) => template.id === "sales-readonly");
  assert.deepEqual(
    (({ allowedCount, blockedCount, level }) => ({ allowedCount, blockedCount, level }))(templateMatch(triage, capabilities)),
    { allowedCount: 2, blockedCount: 1, level: "partial" },
  );
  assert.equal(templateMatch(sales, capabilities).level, "none");
  assert.equal(templateMatch(triage, capabilities.slice(0, 2)).level, "recommended");
  const exact = templateMatch(triage, capabilities, "cap-write");
  assert.equal(exact.level, "recommended");
  assert.equal(exact.requestedCovered, true);
  assert.equal(exact.blockedCount, 2);
  assert.equal(templateMatch(triage, capabilities, "cap-delete").level, "none");
  assert.equal(rankTemplates(permissionPackageTemplates, capabilities)[0].template.id, "support-ticket-triage");
  assert.equal(recommendedTemplateId(permissionPackageTemplates, capabilities, "cap-read"), "support-ticket-triage");
  assert.equal(recommendedTemplateId(permissionPackageTemplates, capabilities, "cap-delete"), "");
});

test("template match counts capabilities blocked only by a missing data domain", () => {
  const triage = permissionPackageTemplates.find((template) => template.id === "support-ticket-triage");
  const classified = capabilities;
  const unclassifiedRead = capability({ id: "cap-fresh", key: "search-crm", dataDomains: [] });
  // The read stays blocked until governed, but it is not a least-privilege block.
  assert.equal(templateMatch(triage, [...classified, unclassifiedRead]).missingDomainBlockedCount, 1);
  assert.equal(templateMatch(triage, classified).missingDomainBlockedCount, 0);
  // A denied-by-design capability without a domain does not count: the
  // template would refuse it anyway.
  const unclassifiedDelete = capability({ id: "cap-fresh-del", key: "purge-crm", action: "delete", riskLevel: "high", dataDomains: [] });
  assert.equal(templateMatch(triage, [unclassifiedDelete]).missingDomainBlockedCount, 0);
  // dataScopes domains count as classification too.
  const scopedRead = capability({ id: "cap-scoped", key: "search-scoped", dataDomains: [], dataScopes: [{ dataDomain: "support" }] });
  assert.equal(templateMatch(triage, [scopedRead]).missingDomainBlockedCount, 0);
});

test("blocked capabilities split into missing-domain and by-design", () => {
  const blocked = [
    capability({ id: "cap-a", key: "a", dataDomains: [] }),
    capability({ id: "cap-b", key: "b", dataDomains: ["support"] }),
    capability({ id: "cap-c", key: "c", dataDomains: [], dataScopes: [{ dataDomain: "support" }] }),
  ];
  const split = splitBlockedCapabilities(blocked);
  assert.deepEqual(split.missingDomain.map((capability) => capability.id), ["cap-a"]);
  assert.deepEqual(split.blockedByDesign.map((capability) => capability.id), ["cap-b", "cap-c"]);
  assert.deepEqual(unclassifiedCapabilities(blocked).map((capability) => capability.id), ["cap-a"]);
});

test("runtime validation readiness plans probes and lists blockers in order", () => {
  const base = {
    allowedCapabilities: [capability({ id: "cap-read", key: "search-tickets" })],
    blockedCapabilities: [capability({ id: "cap-export", key: "export-tickets", action: "export", riskLevel: "high" })],
    context: { callerInstanceId: "caller-a", subjectSelector: "user:support-*", targetId: "target-a" },
    hasApplication: true,
    liveDataAvailable: true,
    runId: "run-1",
  };
  const ready = buildRuntimeValidationReadiness(base);
  assert.deepEqual(ready.blockers, []);
  assert.equal(ready.plan.allowedCapabilityKey, "search-tickets");
  assert.equal(ready.plan.blockedCapabilityKey, "export-tickets");
  assert.equal(ready.plan.subjectId, "user:support-example");

  // No blocked capability: the denied probe is skipped, not a blocker.
  const noBlocked = buildRuntimeValidationReadiness({ ...base, blockedCapabilities: [] });
  assert.equal(noBlocked.plan.blockedCapabilityKey, null);

  const notReady = buildRuntimeValidationReadiness({ ...base, allowedCapabilities: [], hasApplication: false, liveDataAvailable: false });
  assert.deepEqual(notReady.blockers, ["requiresLiveApi", "requiresApplication", "requiresAllowedCapability"]);
  assert.equal(notReady.plan, null);

  const noSubject = buildRuntimeValidationReadiness({ ...base, context: { callerInstanceId: "caller-a", subjectSelector: "", targetId: "target-a" } });
  assert.deepEqual(noSubject.blockers, ["requiresSubject"]);
});

test("classifyRuntimeValidationRun: an unexpected deny probe is a diagnosis, not an abort", () => {
  const nominal = {
    allowedOk: true,
    allowedStatus: 200,
    blockedCapabilityKey: "export_contracts",
    deniedStatus: 403,
    toolListOk: true,
    toolListStatus: 200,
  };
  // Round 6 finding #41: a grant wider than the package lets the blocked
  // call through. The allowed probe still lands, and the outcome names the
  // capability and the observed status instead of dead-ending.
  const mismatch = classifyRuntimeValidationRun({ ...nominal, deniedStatus: 200 });
  assert.equal(mismatch.kind, "denyUnexpected");
  assert.equal(mismatch.blockedCapabilityKey, "export_contracts");
  assert.equal(mismatch.deniedStatus, 200);
  assert.equal(mismatch.allowedStatus, 200);

  assert.deepEqual(classifyRuntimeValidationRun(nominal), { allowedStatus: 200, deniedStatus: 403, kind: "completed" });
  assert.deepEqual(
    classifyRuntimeValidationRun({ ...nominal, blockedCapabilityKey: null, deniedStatus: null }),
    { allowedStatus: 200, deniedStatus: null, kind: "completed" },
  );
  assert.deepEqual(classifyRuntimeValidationRun({ ...nominal, toolListOk: false, toolListStatus: 500 }), { kind: "toolListFailed", status: 500 });
  assert.deepEqual(classifyRuntimeValidationRun({ ...nominal, allowedOk: false, allowedStatus: 502 }), { allowedStatus: 502, kind: "allowedFailed", status: 502 });
});

test("access context persists, defaults to the latest application and honours deep links", () => {
  assert.equal(ACCESS_CONTEXT_STORAGE_KEY, "agent-harbor-access-context-v1");
  const older = application({ id: "app-0", appliedAt: "2026-09-10T00:00:00Z", templateId: "audit-readonly" });
  const newer = application();
  assert.equal(latestApplication([older, newer]).id, "app-1");
  const fromApplication = resolveAccessContext({ agents, applications: [older, newer], routeParams: {}, stored: null });
  assert.equal(fromApplication.source, "application");
  assert.equal(fromApplication.context.templateId, "support-ticket-triage");
  assert.equal(fromApplication.context.requestText, "Open ticket access");
  const fromRequest = accessContextFromApprovalRequest(approval({ requestText: "Need write", region: "north" }));
  assert.equal(fromRequest.requestText, "Need write");
  assert.equal(fromRequest.region, "north");
  assert.equal(fromRequest.templateId, "support-ticket-triage");

  const stored = parseStoredAccessContext(serializeAccessContext({ ...accessContextFromApplication(older), region: " west " }));
  assert.equal(stored.region, "west");
  assert.equal(resolveAccessContext({ agents, applications: [newer], routeParams: {}, stored }).source, "stored");
  assert.equal(parseStoredAccessContext("not json"), null);
  assert.equal(parseStoredAccessContext(JSON.stringify({ callerInstanceId: "caller-a" })), null);
  const gone = { ...stored, callerInstanceId: "deleted" };
  assert.equal(resolveAccessContext({ agents, applications: [newer], routeParams: {}, stored: gone }).source, "application");

  const linked = resolveAccessContext({
    agents,
    applications: [newer],
    routeParams: { caller: "caller-b", target: "target-a", capability: "cap-write", subject: "user:x" },
    stored: null,
  });
  assert.equal(linked.source, "route");
  assert.equal(linked.context.tenantId, "t2");
  assert.equal(linked.context.workspaceId, "ws2");
  assert.equal(linked.context.requestText, "");
  assert.equal(linked.context.requestedCapabilityId, "cap-write");
  assert.deepEqual(accessContextRouteParams(linked.context), {
    template: "support-ticket-triage",
    caller: "caller-b",
    target: "target-a",
    capability: "cap-write",
    subject: "user:x",
  });
  assert.deepEqual(resolveAccessContext({ agents, applications: [], routeParams: {}, stored: null }), {
    context: emptyAccessContext,
    source: "none",
  });
  assert.equal(readinessFilterFromContext(fromApplication.context).subjectId, "user:support-example");
  const options = accessContextOptions([older, newer, application({ id: "app-2", appliedAt: "2026-09-01T00:00:00Z" })]);
  assert.deepEqual(options.map((option) => option.application.id), ["app-1", "app-0"]);
});

test("state machine covers every plan §7 row", () => {
  const pending = approval();
  const cases = [
    ["needsInput", { preview: preview({ status: "needs_input" }) }],
    ["needsInput", { preview: null }],
    ["needsInput", { preview: preview(), localMissing: ["requestText"] }],
    ["draft", { preview: preview() }],
    ["noApproval", { preview: preview({ status: "ready_to_apply", canApplyDirectly: true }) }],
    ["submitted", { preview: preview({ approvalRequest: pending }) }],
    ["submitted", { preview: preview(), approval: pending }],
    ["approved", { preview: preview({ status: "ready_to_apply", approvalRequest: approval({ status: "approved" }) }) }],
    ["rejected", { preview: preview({ approvalRequest: approval({ status: "rejected", reviewComment: "too broad" }) }) }],
    ["withdrawn", { preview: preview({ approvalRequest: approval({ status: "withdrawn" }) }) }],
    ["expired", { preview: preview(), approval: approval({ effectiveStatus: "expired", isExpired: true }) }],
    ["applied", { preview: preview({ status: "validating", applied: true, latestApplication: application() }) }],
    ["applied", { preview: preview({ status: "validating", applied: true, productionReadiness: readiness("blocked", [{ code: "runtime_allowed_trace_present", severity: "blocking" }]) }) }],
    ["applied", { preview: preview(), approval: approval({ status: "approved", consumedAt: "2026-09-22T00:00:00Z" }) }],
    ["checked", { preview: preview({ status: "production_ready", applied: true, productionReadiness: readiness("ready") }) }],
    ["needsReview", { preview: preview({ status: "validating", applied: true, productionReadiness: readiness("needs_review") }) }],
    ["blocked", { preview: preview({ status: "validating", applied: true, productionReadiness: readiness("blocked", [{ code: "application_capabilities_current", severity: "blocking" }]) }) }],
    ["blocked", { preview: preview({ status: "blocked" }) }],
    ["blocked", { preview: preview(), capabilityChanged: true }],
    ["handoff", {
      preview: preview({ status: "production_ready", applied: true, productionReadiness: readiness("ready") }),
      handoff: { status: "ready", tokens: [{ id: "tok", status: "active" }] },
    }],
  ];
  for (const [expected, input] of cases) {
    assert.equal(permissionChangeState(input), expected, JSON.stringify(Object.keys(input)));
  }
});

test("state presentation gives banner tone, actions and stepper per state", () => {
  const stepValues = (state, options) => Object.values(permissionChangePresentation(state, options).steps);
  assert.deepEqual(stepValues("needsInput", { templateSelected: false }), ["current", "pending", "pending", "pending", "pending"]);
  assert.deepEqual(stepValues("needsInput", { templateSelected: true }), ["done", "current", "pending", "pending", "pending"]);
  assert.equal(permissionChangePresentation("needsInput").primaryDisabled, true);
  assert.deepEqual(permissionChangePresentation("draft").secondary, ["changeTemplate"]);
  assert.deepEqual(stepValues("noApproval"), ["done", "done", "done", "skipped", "current"]);
  const submitted = permissionChangePresentation("submitted");
  assert.equal(submitted.tone, "warning");
  assert.equal(submitted.primary, "submitted");
  assert.equal(submitted.primaryDisabled, true);
  assert.deepEqual(submitted.secondary, ["withdraw"]);
  assert.equal(submitted.polling, true);
  assert.equal(permissionChangePresentation("approved").primary, "apply");
  assert.deepEqual(stepValues("rejected"), ["done", "done", "done", "blocked", "pending"]);
  assert.equal(permissionChangePresentation("rejected").tone, "danger");
  assert.equal(permissionChangePresentation("withdrawn").primary, "reedit");
  assert.equal(permissionChangePresentation("expired").primary, "resubmit");
  assert.equal(permissionChangePresentation("applied").primary, "goCheck");
  assert.deepEqual(permissionChangePresentation("checked").secondary, ["recheck", "viewRequests"]);
  assert.equal(permissionChangePresentation("needsReview").primary, "nextAction");
  assert.equal(permissionChangePresentation("blocked").tone, "danger");
  assert.deepEqual(permissionChangePresentation("handoff").secondary, ["createToken"]);
  for (const state of ["needsInput", "draft", "noApproval", "submitted", "approved", "rejected", "withdrawn", "expired", "applied", "checked", "needsReview", "blocked", "handoff"]) {
    const presentation = permissionChangePresentation(state);
    assert.equal(presentation.titleKey, `rd.pc.${state}.title`);
    assert.equal(presentation.polling, state === "submitted");
  }
});

test("409 reconciliation re-fetches by scope and prefers the matching snapshot", () => {
  const input = {
    callerInstanceId: "caller-a",
    region: "east",
    requestText: "Open ticket access",
    requestedCapabilityId: "cap-write",
    subjectSelector: "user:support-*",
    targetId: "target-a",
    templateId: "support-ticket-triage",
    tenantId: "t1",
    workspaceId: "ws1",
  };
  assert.deepEqual(approvalReconcileFilter(input), {
    callerInstanceId: "caller-a",
    limit: 8,
    requestedCapabilityId: "cap-write",
    targetId: "target-a",
    templateId: "support-ticket-triage",
    tenantId: "t1",
    workspaceId: "ws1",
  });
  const other = approval({ id: "apr-other", requestText: "different" });
  const match = approval({ id: "apr-match" });
  const expired = approval({ id: "apr-expired", effectiveStatus: "expired" });
  assert.equal(pickPendingApproval([expired, other, match], input).id, "apr-match");
  assert.equal(pickPendingApproval([expired, other], input).id, "apr-other");
  assert.equal(pickPendingApproval([expired, approval({ id: "done", status: "approved" })], input), null);
  assert.equal(isApprovalAlreadyPendingError({ code: "PERMISSION_PACKAGE_APPROVAL_ALREADY_PENDING", status: 409 }), true);
  assert.equal(isApprovalNotRequiredError({ status: 400, message: "permission package does not require approval" }), true);
  assert.equal(isApprovalNotRequiredError({ status: 400, message: "other" }), false);
  assert.equal(isCapabilityChangedError({ code: "PERMISSION_PACKAGE_CAPABILITY_CHANGED" }), true);
});

test("effective approval prefers the fresher copy of the same request", () => {
  const stale = approval({ updatedAt: "2026-09-21T00:00:00Z" });
  const fresh = approval({ status: "approved", updatedAt: "2026-09-22T00:00:00Z" });
  assert.equal(effectiveApproval(stale, fresh).status, "approved");
  assert.equal(effectiveApproval(fresh, stale).status, "approved");
  assert.equal(effectiveApproval(null, stale).id, "apr-1");
  const rejectedOld = approval({ id: "old", status: "rejected" });
  assert.equal(effectiveApproval(rejectedOld, stale).id, "apr-1");
});

test("go-live counts are named and come from readiness or the preview", () => {
  const checks = Array.from({ length: 11 }, (_, index) => ({ code: `c${index}`, severity: index < 4 ? "passed" : "blocking" }));
  assert.deepEqual(readinessCheckCount(readiness("blocked", checks), null), { ready: 4, total: 11 });
  assert.deepEqual(readinessCheckCount(null, preview()), { ready: 3, total: 10 });
  assert.equal(readinessCheckCount(null, null), null);
  const legs = goLiveLegs({ approval: null, connectionStatus: "ok", liveDataAvailable: true, preview: preview(), readiness: null });
  assert.equal(legs.totalCount, 4);
  assert.deepEqual(legs.checkRows.map((row) => row.key), ["connection", "permission_change", "runtime", "handoff"]);
  assert.equal(goLiveLegs({ approval: null, connectionStatus: null, liveDataAvailable: true, preview: null, readiness: null }), null);
});

test("approval scope key detaches a bound approval when snapshot fields change", () => {
  const base = {
    callerInstanceId: "agt-caller",
    region: "cn-north",
    requestedCapabilityId: "cap-1",
    requestText: "支持工单读改。",
    subjectSelector: "user:support-*",
    targetId: "agt-target",
    templateId: "support-ticket-triage",
    tenantId: "tenant",
    workspaceId: "ws",
  };
  // the approval request echoes the submitted snapshot, so context and its
  // request must hash to the same key
  assert.equal(approvalScopeKey(base), approvalScopeKey({ ...base, id: "ppar-1", status: "approved" }));
  for (const field of ["region", "requestedCapabilityId", "requestText", "subjectSelector"]) {
    assert.notEqual(approvalScopeKey({ ...base, [field]: `changed-${field}` }), approvalScopeKey(base));
  }
  // optional snapshot fields absent on both sides still match
  const sparseA = { ...base, requestedCapabilityId: undefined, subjectSelector: undefined, requestText: undefined, region: undefined };
  const sparseB = { ...base, requestedCapabilityId: undefined, subjectSelector: undefined, requestText: undefined, region: undefined, id: "ppar-2" };
  assert.equal(approvalScopeKey(sparseA), approvalScopeKey(sparseB));
});

test("handoff token TTL stays within 15, 30 and 60 minutes", () => {
  assert.deepEqual(tokenTtlOptions(null), [900, 1800, 3600]);
  const handoff = { tokenEligibility: { defaultExpiresInSeconds: 1800, maxExpiresInSeconds: 1800 } };
  assert.deepEqual(tokenTtlOptions(handoff), [900, 1800]);
  assert.equal(defaultTokenTtl(handoff), 1800);
  assert.equal(defaultTokenTtl({ tokenEligibility: { defaultExpiresInSeconds: 7200, maxExpiresInSeconds: 3600 } }), 3600);
});

test("handoff shell snippet is valid shell with one command per line and no token", () => {
  const snippet = handoffShellSnippet({
    apiBase: "http://127.0.0.1:9090/",
    comments: { call: "call one tool", token: "paste the token" },
    handoff: {
      allowedCapabilities: [{ key: "search-tickets" }, { key: "a'quoted" }],
      scope: { targetId: "target a", subjectId: "user:support-example" },
    },
  });
  const lines = snippet.split("\n");
  assert.equal(lines.length, 7);
  assert.match(lines[0], /^export AGENT_HARBOR_URL='http:\/\/127\.0\.0\.1:9090\/api\/v1\/mcp\/agents\/target%20a\/rpc'$/);
  assert.match(snippet, /Authorization: Bearer \$\{AGENT_HARBOR_TOKEN\}/);
  assert.match(snippet, /"name":"a'\\''quoted"/);
  assert.equal(shellQuote("it's"), `'it'\\''s'`);
  execFileSync("bash", ["-n"], { input: snippet });
  execFileSync("sh", ["-n"], { input: snippet });
});

test("copied MCP client config url becomes absolute with the configured API base", () => {
  const config = JSON.stringify({
    transport: "streamable-http",
    url: "/api/v1/mcp/agents/target%20a/rpc",
    headers: {
      Authorization: "Bearer ${AGENT_HARBOR_TOKEN}",
      "X-AgentHarbor-Subject-Id": "user:support-example",
    },
  });
  const absolute = JSON.parse(absolutizeHandoffConfig(config, "http://127.0.0.1:19090/"));
  assert.equal(absolute.url, "http://127.0.0.1:19090/api/v1/mcp/agents/target%20a/rpc");
  // Placeholders survive the round trip untouched.
  assert.equal(absolute.headers.Authorization, "Bearer ${AGENT_HARBOR_TOKEN}");

  // Already-absolute urls, non-JSON payloads, and bases that change nothing
  // pass through byte-for-byte.
  const absoluteUrl = JSON.stringify({ url: "https://harbor.example/rpc" });
  assert.equal(absolutizeHandoffConfig(absoluteUrl, "http://127.0.0.1:19090"), absoluteUrl);
  assert.equal(absolutizeHandoffConfig("not json", "http://127.0.0.1:19090"), "not json");
  assert.equal(absolutizeHandoffConfig(config, ""), config);
});

test("user workbench derives onboarding, requests, resources and permissions", () => {
  assert.equal(workbenchActor({ authenticated: true, requiresLogin: false }), DEMO_ACTOR);
  assert.equal(workbenchActor({ actor: "alice", authenticated: true, requiresLogin: true }), "alice");
  assert.equal(workbenchActor(null), "");

  const now = Date.parse("2026-09-27T00:00:00Z");
  const keys = [
    { id: "k1", agentId: "caller-a", name: "handoff", prefix: "ah_abcdefgh", createdForHandoffId: "h1", createdAt: "", expiresAt: "2026-09-27T01:00:00Z" },
    { id: "k2", agentId: "caller-a", name: "old", prefix: "ah_zzzzzzzz", createdAt: "", expiresAt: "2026-09-26T00:00:00Z" },
    { id: "k3", agentId: "caller-a", name: "revoked", prefix: "ah_yyyyyyyy", createdAt: "", expiresAt: "2026-09-28T00:00:00Z", revokedAt: "2026-09-26T00:00:00Z" },
  ];
  assert.deepEqual(keys.map((key) => keyStatus(key, now)), ["active", "expired", "revoked"]);
  const progress = onboardingProgress({ agents, applications: [application()], capabilities, keys, liveData: true, now });
  assert.equal(progress.doneCount, 4);
  assert.equal(progress.total, 4);
  const early = onboardingProgress({ agents, applications: [], capabilities, keys: [], liveData: true, now });
  assert.equal(early.doneCount, 2);
  assert.equal(early.nextKey, "authorize");

  const rows = myRequests([
    approval({ id: "mine-pending" }),
    approval({ id: "mine-approved", status: "approved", createdAt: "2026-09-22T00:00:00Z" }),
    approval({ id: "mine-withdrawn", status: "withdrawn" }),
    approval({ id: "theirs", requestedBy: "someone" }),
  ], DEMO_ACTOR, capabilities);
  assert.deepEqual(rows.map((row) => row.request.id), ["mine-approved", "mine-pending"]);
  assert.equal(rows[0].allowedCount, 2);
  assert.equal(rows[0].blockedCount, 1);
  assert.deepEqual(myRequestCounts(rows), { approved: 1, pending: 1, rejected: 0 });
  assert.deepEqual(myRequests(rows.map((row) => row.request), "", capabilities), []);

  const resources = myResources(agents, capabilities, [{ id: "tr", targetAgentId: "target-a", callerInstanceId: "caller-a", createdAt: "2026-09-25T00:00:00Z", decision: "allowed", routeType: "mcp" }]);
  assert.deepEqual(resources.map((row) => [row.agent.id, row.role, row.lastActivity]), [
    ["caller-b", "caller", "2026-09-03T00:00:00Z"],
    ["caller-a", "caller", "2026-09-25T00:00:00Z"],
    ["target-a", "target", "2026-09-25T00:00:00Z"],
  ]);

  const layer = { effect: "allow", status: "enabled" };
  const profile = {
    grants: [{
      tenantEntitlement: { id: "te-1", targetId: "target-a", capabilityId: "cap-read", ...layer },
      capability: capabilities[0],
      scopeStatus: "valid",
      workspaceAssignments: [{
        workspaceAssignment: { id: "wa-1", ...layer },
        scopeStatus: "valid",
        instanceAssignments: [{
          instanceAssignment: { id: "ia-1", callerInstanceId: "caller-a", ...layer },
          effectiveInstanceDataScopes: [{ dataDomain: "support" }],
          scopeStatus: "valid",
        }],
      }],
    }],
  };
  const permissions = myPermissions({
    approvals: [approval({ id: "apr-used", status: "approved", consumedByApplicationId: "app-1" })],
    applications: [application()],
    callerInstanceId: "caller-a",
    capabilities,
    keys,
    now,
    profile,
  });
  assert.deepEqual(permissions.kpis, { activeTokens: 1, allowed: 1, blocked: 2, historicalTokens: 2 });
  assert.equal(permissions.rows[0].approvalId, "apr-used");
  assert.deepEqual(permissions.rows[0].dataScopes, [{ dataDomain: "support" }]);
  assert.deepEqual(permissions.rows.slice(1).map((row) => row.decision), ["blocked", "blocked"]);
});
