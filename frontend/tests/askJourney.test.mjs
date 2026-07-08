import assert from "node:assert/strict";
import test from "node:test";

import {
  accessDecisionSummaryLabel,
  accessEvidenceMessageLabel,
  accessNextActionLabel,
  buildExplainRequest,
  buildPermissionChangeHandoff,
  decisionRecordRows,
  evidenceChainRows
} from "../src/askJourney.ts";
import {
  createTranslator
} from "../src/i18n.ts";
import { permissionPackageTemplates } from "../src/permissionPackages.ts";

const now = "2026-06-11T10:00:00Z";

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
      ["ask.evidenceLayer.tenant_entitlement", "success", false],
      ["ask.evidenceLayer.workspace_assignment", "danger", true],
      ["ask.evidenceLayer.instance_assignment", "neutral", false]
    ]
  );
});

test("evidenceChainRows remains a compatibility alias for decisionRecordRows", () => {
  assert.equal(evidenceChainRows, decisionRecordRows);
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
  assert.equal(accessEvidenceMessageLabel(rows[0], t), "租户授权缺失，或正在阻断这个工具能力。");
  assert.equal(accessEvidenceMessageLabel(rows[1], t), "工作区分配已匹配。");
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
    templateId: "sales-readonly",
    tenantId: tenant.id,
    tenantName: tenant.name,
    workspaceId: caller.workspaceId,
    workspaceName: "Support"
  });
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
