import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  accessDecisionSummaryLabel,
  accessDecisionRecordMessageLabel,
  accessDecisionPrimaryAction,
  accessNextActionLabel,
  askAccessScopeOptions,
  askAccessTargetVisibleToScope,
  buildExplainRequest,
  buildPermissionChangeHandoff,
  canStartPermissionChangeForAdmin,
  decisionRecordRows,
  permissionChangeHandoffDraftInput,
  resolveAskAccessSelection
} from "../src/askJourney.ts";
import {
  createTranslator
} from "../src/i18n.ts";
import { permissionPackageTemplates } from "../src/permissionPackages.ts";

const now = "2026-06-11T10:00:00Z";
const askJourneySource = readFileSync(new URL("../src/askJourney.ts", import.meta.url), "utf8");

const tenant = { createdAt: now, id: "tenant-root", level: 0, name: "Platform Ops", status: "active", updatedAt: now };
const caller = {
  channelType: "local",
  createdAt: now,
  credentialVersion: 1,
  id: "agt-support",
  name: "Support Agent",
  status: "active",
  tenantId: tenant.id,
  updatedAt: now,
  workspaceId: "ws-support"
};
const target = {
  ...caller,
  channelType: "mcp",
  id: "agt-ticket-mcp",
  name: "Ticket MCP"
};
const readCapability = {
  action: "read",
  dataDomains: ["support"],
  description: "Search tickets.",
  discoveredAt: now,
  discoveryStatus: "approved",
  displayName: "Search tickets",
  enforcementMode: "gateway",
  id: "cap-search",
  key: "search_ticket",
  riskLevel: "low",
  sensitivity: "internal",
  targetId: target.id,
  type: "mcp_tool",
  updatedAt: now,
  version: 1
};
const exportCapability = {
  ...readCapability,
  action: "export",
  displayName: "Export contracts",
  id: "cap-export",
  key: "export_contracts",
  riskLevel: "high",
  sensitivity: "confidential"
};

function consoleData(overrides = {}) {
  return {
    accessGrants: [],
    agents: [caller, target],
    apiBase: "http://127.0.0.1:9090",
    auditEvents: [],
    capabilities: [readCapability, exportCapability],
    capabilitiesLoadedFromApi: true,
    capabilityAssignmentsLoadedFromApi: true,
    channels: [],
    evidenceRuns: [],
    grantsLoadedFromApi: true,
    instanceAssignments: [],
    loadedFromApi: true,
    providers: [],
    routePolicies: [],
    routePoliciesLoadedFromApi: true,
    setupLoadedFromApi: true,
    systemMetrics: [],
    tenantEntitlements: [],
    tenants: [tenant],
    traces: [],
    workspaceAssignments: [],
    ...overrides
  };
}

const translateIntent = (key, values) => `${key}: ${values.callerName} -> ${values.targetName} / ${values.capabilityName}`;

test("buildExplainRequest validates and normalizes required access query fields", () => {
  assert.deepEqual(
    buildExplainRequest({ tenantId: "tenant-root", workspaceId: " ", callerInstanceId: "", targetId: "agt-ticket-mcp" }),
    {
      complete: false,
      missingFields: ["workspaceId", "callerInstanceId", "capabilityId"],
      request: null
    }
  );

  assert.deepEqual(
    buildExplainRequest({
      capabilityId: " cap-search ",
      callerInstanceId: " agt-support ",
      subjectId: " user:support-001 ",
      targetId: " agt-ticket-mcp ",
      tenantId: " tenant-root ",
      workspaceId: " ws-support "
    }),
    {
      complete: true,
      missingFields: [],
      request: {
        capabilityId: "cap-search",
        callerInstanceId: "agt-support",
        subjectId: "user:support-001",
        targetId: "agt-ticket-mcp",
        tenantId: "tenant-root",
        workspaceId: "ws-support"
      }
    }
  );
});

test("decisionRecordRows marks the first denied record layer as the broken ring", () => {
  const rows = decisionRecordRows({
    dataScopes: [],
    decision: { allowed: false, reason: "capability is not approved", source: "capability" },
    evidence: [
      { id: "ent-1", layer: "tenant_entitlement", message: "Tenant grant matched.", status: "matched" },
      { layer: "workspace_assignment", message: "Workspace assignment is missing.", status: "missing" },
      { layer: "instance_assignment", message: "Instance was not checked.", status: "not_checked" }
    ],
    nextActions: ["Create the missing workspace assignment."],
    outcome: "denied",
    request: {
      capabilityId: "cap-search",
      callerInstanceId: "agt-support",
      targetId: "agt-ticket-mcp",
      tenantId: "tenant-root",
      workspaceId: "ws-support"
    },
    summary: "Denied because the workspace assignment is missing."
  });

  assert.deepEqual(
    rows.map((row) => [row.layerKey, row.tone, row.isBroken]),
    [
      ["ask.recordLayer.tenant_entitlement", "success", false],
      ["ask.recordLayer.workspace_assignment", "danger", true],
      ["ask.recordLayer.instance_assignment", "neutral", false]
    ]
  );
});

test("ask access presentation no longer exports legacy evidence aliases", () => {
  assert.doesNotMatch(askJourneySource, /AskEvidenceChainRow/);
  assert.doesNotMatch(askJourneySource, /evidenceChainRows/);
  assert.doesNotMatch(askJourneySource, /accessEvidenceMessageLabel/);
  assert.doesNotMatch(askJourneySource, /function evidenceTone/);
  assert.doesNotMatch(askJourneySource, /evidenceTone\(/);
});

test("access query presentation localizes backend decision guidance at render time", () => {
  const t = createTranslator("zh-CN");
  const result = {
    dataScopes: [],
    decision: {
      allowed: false,
      capabilityId: "cap-search",
      reason: "tenant has no entitlement for capability",
      source: "tenant_entitlement"
    },
    evidence: [
      { layer: "tenant_entitlement", message: "Tenant entitlement is missing or blocking this capability.", status: "missing" },
      { id: "workspace-1", layer: "workspace_assignment", message: "Workspace assignment matched.", status: "matched" }
    ],
    nextActions: [
      "Use the permission package flow with draft_permission_package and apply_permission_package to create the tenant entitlement, workspace assignment, and caller assignment together."
    ],
    outcome: "denied",
    request: {
      capabilityId: "cap-search",
      callerInstanceId: "agt-support",
      targetId: "agt-ticket-mcp",
      tenantId: "tenant-root",
      workspaceId: "ws-support"
    },
    summary: "Denied: tenant has no entitlement for capability."
  };
  const rows = decisionRecordRows(result);

  assert.equal(accessDecisionSummaryLabel(result, t), "当前不能访问：租户尚未获得该能力授权。");
  assert.equal(accessDecisionRecordMessageLabel(rows[0], t), "租户授权缺失，或正在阻断这个工具能力。");
  assert.equal(accessDecisionRecordMessageLabel(rows[1], t), "工作区分配已匹配。");
  assert.equal(accessNextActionLabel(result.nextActions[0], t), "发起权限包变更，一次性创建租户、工作区和调用方授权。");
});

test("access query presentation sanitizes unknown technical guidance", () => {
  const t = createTranslator("en");

  assert.equal(
    accessNextActionLabel("Inspect get_tenant_access_profile and then call list_capabilities.", t),
    "Inspect tenant access profile and then call capability list."
  );
});

test("buildPermissionChangeHandoff maps a denied query into a one-time permission change context", () => {
  const result = buildPermissionChangeHandoff(
    {
      capabilityId: readCapability.id,
      callerInstanceId: caller.id,
      subjectId: "user:support-001",
      targetId: target.id,
      tenantId: tenant.id,
      workspaceId: caller.workspaceId
    },
    consoleData(),
    { templates: permissionPackageTemplates, translateIntent }
  );

  assert.deepEqual(result, {
    callerInstanceId: caller.id,
    callerName: caller.name,
    capabilityId: readCapability.id,
    capabilityName: readCapability.displayName,
    intentText: "ask.intent.openAccess: Support Agent -> Ticket MCP / Search tickets",
    sourceView: "ask",
    subjectId: "user:support-001",
    targetId: target.id,
    targetName: target.name,
    templateId: "support-ticket-triage",
    tenantId: tenant.id,
    tenantName: tenant.name,
    workspaceId: caller.workspaceId,
    workspaceName: "Support"
  });
});

test("access-query handoff pins the denied capability while non-query handoffs clear that boundary", () => {
  const current = {
    callerInstanceId: "old-caller",
    region: "华东",
    requestText: "Old request",
    requestedCapabilityId: "cap-old",
    subjectSelector: "user:old-*",
    targetId: "old-target",
    templateId: "sales-readonly",
    tenantId: "old-tenant",
    workspaceId: "old-workspace"
  };
  const askContext = buildPermissionChangeHandoff(
    {
      capabilityId: readCapability.id,
      callerInstanceId: caller.id,
      subjectId: "user:support-001",
      targetId: target.id,
      tenantId: tenant.id,
      workspaceId: caller.workspaceId
    },
    consoleData(),
    { templates: permissionPackageTemplates, translateIntent }
  );

  const askDraft = permissionChangeHandoffDraftInput(askContext, current);
  assert.equal(askDraft.requestedCapabilityId, readCapability.id);
  assert.equal(askDraft.targetId, target.id);
  assert.equal(askDraft.subjectSelector, "user:support-001");

  const tenantDraft = permissionChangeHandoffDraftInput({
    sourceView: "tenants",
    tenantId: tenant.id,
    workspaceId: caller.workspaceId
  }, askDraft);
  assert.equal(tenantDraft.requestedCapabilityId, undefined);
});

test("access query selection stays inside one active tenant and workspace", () => {
  const tenantTwo = { ...tenant, id: "tenant-two", name: "Finance" };
  const callerTwo = { ...caller, id: "agt-finance", tenantId: tenantTwo.id, workspaceId: "ws-finance" };
  const targetTwo = { ...target, id: "agt-finance-mcp", tenantId: tenantTwo.id, workspaceId: "ws-finance" };
  const capabilityTwo = { ...readCapability, id: "cap-finance", targetId: targetTwo.id };
  const inventory = {
    agents: [caller, target, callerTwo, targetTwo],
    capabilities: [readCapability, capabilityTwo],
    tenants: [tenant, tenantTwo]
  };

  const resolved = resolveAskAccessSelection({ tenantId: tenantTwo.id }, inventory);
  assert.deepEqual(resolved, {
    capabilityId: capabilityTwo.id,
    callerInstanceId: callerTwo.id,
    subjectId: "",
    targetId: targetTwo.id,
    tenantId: tenantTwo.id,
    workspaceId: "ws-finance"
  });

  const options = askAccessScopeOptions(inventory, resolved);
  assert.deepEqual(options.callers.map((agent) => agent.id), [callerTwo.id]);
  assert.deepEqual(options.targets.map((agent) => agent.id), [targetTwo.id]);
  assert.deepEqual(options.capabilities.map((capability) => capability.id), [capabilityTwo.id]);
});

test("access query target visibility follows the active ancestor chain and workspace", () => {
  const childTenant = {
    ...tenant,
    id: "tenant-child",
    level: 1,
    name: "Support child",
    parentTenantId: tenant.id
  };
  const siblingTenant = {
    ...tenant,
    id: "tenant-sibling",
    level: 1,
    name: "Support sibling",
    parentTenantId: tenant.id
  };
  const childTarget = { ...target, id: "agt-child-mcp", tenantId: childTenant.id };
  const siblingTarget = { ...target, id: "agt-sibling-mcp", tenantId: siblingTenant.id };

  assert.equal(
    askAccessTargetVisibleToScope(target, childTenant.id, target.workspaceId, [tenant, childTenant, siblingTenant]),
    true
  );
  assert.equal(
    askAccessTargetVisibleToScope(siblingTarget, childTenant.id, target.workspaceId, [tenant, childTenant, siblingTenant]),
    false
  );
  assert.equal(
    askAccessTargetVisibleToScope(childTarget, tenant.id, target.workspaceId, [tenant, childTenant, siblingTenant]),
    false
  );
  assert.equal(
    askAccessTargetVisibleToScope(target, childTenant.id, "ws-other", [tenant, childTenant]),
    false
  );
  assert.equal(
    askAccessTargetVisibleToScope(
      target,
      childTenant.id,
      target.workspaceId,
      [{ ...tenant, status: "disabled" }, childTenant]
    ),
    false
  );
  assert.equal(
    askAccessTargetVisibleToScope({ ...target, status: "disabled" }, childTenant.id, target.workspaceId, [tenant, childTenant]),
    false
  );
});

test("access query bootstraps ancestor targets without replacing explicit stale targets", () => {
  const childTenant = {
    ...tenant,
    id: "tenant-child",
    level: 1,
    name: "Support child",
    parentTenantId: tenant.id
  };
  const siblingTenant = {
    ...tenant,
    id: "tenant-sibling",
    level: 1,
    name: "Support sibling",
    parentTenantId: tenant.id
  };
  const childCaller = { ...caller, id: "agt-child-caller", tenantId: childTenant.id };
  const siblingTarget = { ...target, id: "agt-sibling-mcp", tenantId: siblingTenant.id };
  const disabledTarget = { ...target, id: "agt-disabled-mcp", status: "disabled" };
  const siblingCapability = { ...readCapability, id: "cap-sibling", targetId: siblingTarget.id };
  const disabledCapability = { ...readCapability, id: "cap-disabled", targetId: disabledTarget.id };
  const inventory = {
    agents: [childCaller, siblingTarget, disabledTarget, target],
    capabilities: [siblingCapability, disabledCapability, readCapability],
    tenants: [tenant, childTenant, siblingTenant]
  };

  const resolved = resolveAskAccessSelection({}, inventory);
  assert.deepEqual(resolved, {
    capabilityId: readCapability.id,
    callerInstanceId: childCaller.id,
    subjectId: "",
    targetId: target.id,
    tenantId: childTenant.id,
    workspaceId: childCaller.workspaceId
  });

  const options = askAccessScopeOptions(inventory, resolved);
  assert.deepEqual(options.callers.map((agent) => agent.id), [childCaller.id]);
  assert.deepEqual(options.targets.map((agent) => agent.id), [target.id]);
  assert.deepEqual(options.capabilities.map((capability) => capability.id), [readCapability.id]);

  const staleSelection = {
    capabilityId: siblingCapability.id,
    callerInstanceId: childCaller.id,
    targetId: siblingTarget.id,
    tenantId: childTenant.id,
    workspaceId: childCaller.workspaceId
  };
  assert.deepEqual(resolveAskAccessSelection(staleSelection, inventory), {
    capabilityId: "",
    callerInstanceId: childCaller.id,
    subjectId: "",
    targetId: "",
    tenantId: childTenant.id,
    workspaceId: childCaller.workspaceId
  });
  assert.deepEqual(askAccessScopeOptions(inventory, staleSelection).capabilities, []);

  assert.deepEqual(resolveAskAccessSelection({
    callerInstanceId: childCaller.id,
    tenantId: childTenant.id,
    workspaceId: childCaller.workspaceId
  }, inventory), resolved);
});

test("access query selection defaults omissions but never replaces explicit stale resources", () => {
  const disabledTenant = { ...tenant, id: "tenant-disabled", name: "Disabled", status: "disabled" };
  const disabledCaller = { ...caller, id: "agt-disabled", tenantId: disabledTenant.id, workspaceId: "ws-disabled" };
  const disabledTarget = { ...target, id: "agt-disabled-mcp", tenantId: disabledTenant.id, workspaceId: "ws-disabled" };
  const disabledCapability = { ...readCapability, id: "cap-disabled", targetId: disabledTarget.id };
  const tenantTwo = { ...tenant, id: "tenant-two", name: "Finance" };
  const callerTwo = { ...caller, id: "agt-finance", tenantId: tenantTwo.id, workspaceId: "ws-finance" };
  const targetTwo = { ...target, id: "agt-finance-mcp", tenantId: tenantTwo.id, workspaceId: "ws-finance" };
  const capabilityTwo = { ...readCapability, id: "cap-finance", targetId: targetTwo.id };
  const inventory = {
    agents: [disabledCaller, disabledTarget, caller, target, callerTwo, targetTwo],
    capabilities: [disabledCapability, readCapability, capabilityTwo],
    tenants: [disabledTenant, tenant, tenantTwo]
  };

  assert.deepEqual(resolveAskAccessSelection({}, inventory), {
    capabilityId: readCapability.id,
    callerInstanceId: caller.id,
    subjectId: "",
    targetId: target.id,
    tenantId: tenant.id,
    workspaceId: caller.workspaceId
  });
  assert.deepEqual(resolveAskAccessSelection({ callerInstanceId: callerTwo.id }, inventory), {
    capabilityId: capabilityTwo.id,
    callerInstanceId: callerTwo.id,
    subjectId: "",
    targetId: targetTwo.id,
    tenantId: tenantTwo.id,
    workspaceId: callerTwo.workspaceId
  });
  assert.deepEqual(resolveAskAccessSelection({ tenantId: tenantTwo.id, workspaceId: "" }, inventory), {
    capabilityId: "",
    callerInstanceId: "",
    subjectId: "",
    targetId: "",
    tenantId: tenantTwo.id,
    workspaceId: ""
  });
  assert.deepEqual(resolveAskAccessSelection({ tenantId: disabledTenant.id }, inventory), {
    capabilityId: "",
    callerInstanceId: "",
    subjectId: "",
    targetId: "",
    tenantId: "",
    workspaceId: ""
  });

  const stale = resolveAskAccessSelection({
    capabilityId: readCapability.id,
    callerInstanceId: caller.id,
    targetId: target.id,
    tenantId: tenantTwo.id,
    workspaceId: callerTwo.workspaceId
  }, inventory);
  assert.equal(stale.tenantId, tenantTwo.id);
  assert.equal(stale.workspaceId, callerTwo.workspaceId);
  assert.equal(stale.callerInstanceId, "");
  assert.equal(stale.targetId, "");
  assert.equal(stale.capabilityId, "");
});

test("access query primary actions route capability, permission, and boundary fixes safely", () => {
  const base = {
    dataScopes: [],
    decision: { allowed: false, reason: "capability is not approved", source: "capability" },
    evidence: [],
    nextActions: [],
    outcome: "denied",
    request: {
      capabilityId: readCapability.id,
      callerInstanceId: caller.id,
      targetId: target.id,
      tenantId: tenant.id,
      workspaceId: caller.workspaceId
    },
    summary: "Denied"
  };

  assert.deepEqual(
    accessDecisionPrimaryAction({ ...base, nextActionCodes: ["approve_capability"] }, [readCapability], permissionPackageTemplates),
    { kind: "capability_review", labelKey: "action.reviewCapabilityApproval" }
  );
  assert.deepEqual(
    accessDecisionPrimaryAction({ ...base, nextActionCodes: ["use_permission_package"] }, [readCapability], permissionPackageTemplates),
    { kind: "permission_change", labelKey: "action.fixAccessDecision" }
  );
  assert.deepEqual(
    accessDecisionPrimaryAction({ ...base, nextActionCodes: ["use_permission_package"] }, [readCapability], permissionPackageTemplates, false),
    { kind: "access_profile", labelKey: "action.reviewPlatformManagedTarget" }
  );
  assert.deepEqual(
    accessDecisionPrimaryAction({ ...base, nextActionCodes: ["use_permission_package"] }, [
      { ...readCapability, dataDomains: [], dataScopes: [] }
    ], permissionPackageTemplates),
    { kind: "capability_review", labelKey: "action.classifyCapability" }
  );
  assert.deepEqual(
    accessDecisionPrimaryAction({ ...base, nextActionCodes: ["review_deny"] }, [readCapability], permissionPackageTemplates),
    { kind: "access_profile", labelKey: "action.openAccessProfile" }
  );
  assert.deepEqual(
    accessDecisionPrimaryAction({
      ...base,
      decision: { allowed: false, reason: "tenant has no entitlement for capability", source: "tenant_entitlement" }
    }, [readCapability], permissionPackageTemplates),
    { kind: "permission_change", labelKey: "action.fixAccessDecision" }
  );
});

test("tenant admins can diagnose ancestor targets without entering an unauthorized change flow", () => {
  const request = {
    capabilityId: readCapability.id,
    callerInstanceId: caller.id,
    targetId: target.id,
    tenantId: "tenant-child",
    workspaceId: caller.workspaceId
  };
  assert.equal(canStartPermissionChangeForAdmin("platform_admin", request, [target]), true);
  assert.equal(canStartPermissionChangeForAdmin("tenant_admin", request, [target]), false);
  assert.equal(canStartPermissionChangeForAdmin("tenant_admin", request, [{ ...target, tenantId: "tenant-child" }]), true);
});

test("approved support export reviews the permission boundary when no safe package can grant it", () => {
  const denied = {
    dataScopes: [],
    decision: { allowed: false, reason: "tenant has no entitlement for capability", source: "tenant_entitlement" },
    evidence: [],
    nextActionCodes: ["use_permission_package"],
    nextActions: [],
    outcome: "denied",
    request: {
      capabilityId: exportCapability.id,
      callerInstanceId: caller.id,
      targetId: target.id,
      tenantId: tenant.id,
      workspaceId: caller.workspaceId
    },
    summary: "Denied"
  };

  assert.equal(exportCapability.discoveryStatus, "approved");
  assert.deepEqual(exportCapability.dataDomains, ["support"]);
  assert.deepEqual(
    accessDecisionPrimaryAction(denied, [exportCapability], permissionPackageTemplates),
    { kind: "access_profile", labelKey: "action.reviewPermissionBoundary" }
  );
});

test("permission handoff preserves the denied decision context", () => {
  const denied = {
    dataScopes: [],
    decision: { allowed: false, reason: "tenant has no entitlement for capability", source: "tenant_entitlement" },
    evidence: [
      { layer: "tenant_entitlement", message: "Tenant entitlement is missing or blocking this capability.", status: "missing" }
    ],
    nextActionCodes: ["use_permission_package"],
    nextActions: [],
    outcome: "denied",
    request: {
      capabilityId: readCapability.id,
      callerInstanceId: caller.id,
      targetId: target.id,
      tenantId: tenant.id,
      workspaceId: caller.workspaceId
    },
    summary: "Denied"
  };
  const result = buildPermissionChangeHandoff(denied.request, consoleData(), {
    decisionResult: denied,
    templates: permissionPackageTemplates,
    translateIntent
  });

  assert.equal(result.brokenLayer, "tenant_entitlement");
  assert.equal(result.decisionReason, denied.decision.reason);
  assert.equal(result.decisionSource, denied.decision.source);
});

test("buildPermissionChangeHandoff keeps template unset when no permission package template can safely cover the capability", () => {
  const result = buildPermissionChangeHandoff(
    {
      capabilityId: exportCapability.id,
      callerInstanceId: caller.id,
      targetId: target.id,
      tenantId: tenant.id,
      workspaceId: caller.workspaceId
    },
    consoleData(),
    { templates: permissionPackageTemplates, translateIntent }
  );

  assert.equal(result.templateId, undefined);
  assert.match(result.intentText, /Export contracts/);
});
