import assert from "node:assert/strict";
import test from "node:test";

import {
  defaultPermissionPackageDraftInput,
  createPermissionPackageDraft,
  permissionPackageTemplates,
  subjectIdExampleFromSelector
} from "../src/permissionPackages.ts";

const now = "2026-06-04T09:00:00Z";

const capabilities = [
  {
    id: "cap_search_customer",
    targetId: "agt_crm_mcp",
    type: "mcp_tool",
    key: "search_customer",
    displayName: "search_customer",
    description: "Search customer records.",
    action: "read",
    dataDomains: ["crm"],
    dataScopes: [{ dataDomain: "crm", dataset: "customers", classification: "internal" }],
    sensitivity: "internal",
    riskLevel: "low",
    enforcementMode: "gateway",
    discoveryStatus: "approved",
    version: 1,
    discoveredAt: now,
    updatedAt: now
  },
  {
    id: "cap_export_contracts",
    targetId: "agt_crm_mcp",
    type: "mcp_tool",
    key: "export_contracts",
    displayName: "export_contracts",
    description: "Export contract packages.",
    action: "export",
    dataDomains: ["contracts"],
    dataScopes: [{ dataDomain: "contracts", dataset: "contract_packages", classification: "confidential" }],
    sensitivity: "confidential",
    riskLevel: "high",
    enforcementMode: "gateway",
    discoveryStatus: "pending_review",
    version: 1,
    discoveredAt: now,
    updatedAt: now
  }
];

test("default permission package draft opens the approval-required support journey", () => {
  assert.equal(defaultPermissionPackageDraftInput.templateId, "support-ticket-triage");
  assert.equal(defaultPermissionPackageDraftInput.subjectSelector, "user:support-*");
  assert.match(defaultPermissionPackageDraftInput.requestText, /客服|工单/);
});

test("sales read-only package drafts allowed reads, blocked exports, and simulation rows", () => {
  const draft = createPermissionPackageDraft(
    {
      callerInstanceId: "agt_sales_assistant",
      region: "华东",
      requestText: "给华东租户的销售助手开通客户查询权限，但不能导出合同。",
      subjectSelector: "user:sales-*",
      targetId: "agt_crm_mcp",
      templateId: "sales-readonly",
      tenantId: "tenant-east",
      workspaceId: "ws-sales"
    },
    { capabilities }
  );

  assert.equal(permissionPackageTemplates.some((template) => template.id === "sales-readonly" && template.version === 1), true);
  assert.deepEqual(draft.allowedCapabilities.map((capability) => capability.key), ["search_customer"]);
  assert.deepEqual(draft.blockedCapabilities.map((capability) => capability.key), ["export_contracts"]);
  assert.deepEqual(draft.dataScopes, [
    {
      dataDomain: "crm",
      region: "华东",
      tenantFilter: "tenant_id = 'tenant-east'"
    }
  ]);
  assert.equal(draft.readiness.canApply, true);
  assert.equal(draft.policyGate.decision, "allow");
  assert.equal(draft.policyGate.canApplyDirectly, true);
  assert.equal(draft.policyGate.policyVersion, 1);
  assert.equal(draft.policyGate.reasons.length, 0);
  assert.deepEqual(
    draft.simulationRows.map((row) => [row.expectedDecision, row.capabilityKey]),
    [
      ["allow", "search_customer"],
      ["deny", "export_contracts"],
      ["deny", "cross-region-data"],
      ["deny", "sensitive-finance-fields"]
    ]
  );
});

test("draft cannot be applied when no allowed capability matches the selected target", () => {
  const draft = createPermissionPackageDraft(
    {
      callerInstanceId: "agt_sales_assistant",
      region: "华东",
      requestText: "开通销售只读。",
      subjectSelector: "user:sales-*",
      targetId: "agt_unknown",
      templateId: "sales-readonly",
      tenantId: "tenant-east",
      workspaceId: "ws-sales"
    },
    { capabilities }
  );

  assert.equal(draft.allowedCapabilities.length, 0);
  assert.equal(draft.readiness.canApply, false);
  assert.match(draft.readiness.warnings.join(" "), /No matching allowed capabilities/);
});

test("draft cannot be applied without a bounded access subject", () => {
  for (const subjectSelector of ["", " ", "*"]) {
    const draft = createPermissionPackageDraft(
      {
        callerInstanceId: "agt_sales_assistant",
        region: "华东",
        requestText: "开通销售只读。",
        subjectSelector,
        targetId: "agt_crm_mcp",
        templateId: "sales-readonly",
        tenantId: "tenant-east",
        workspaceId: "ws-sales"
      },
      { capabilities }
    );

    assert.equal(draft.readiness.canApply, false);
    assert.deepEqual(draft.readiness.missingFields, ["subjectSelector"]);
  }
});

test("support ticket triage package requires approval for risky allowed writes", () => {
  const draft = createPermissionPackageDraft(
    {
      callerInstanceId: "agt_support_assistant",
      region: "华东",
      requestText: "给客服助手开通工单更新权限。",
      subjectSelector: "user:support-*",
      targetId: "agt_crm_mcp",
      templateId: "support-ticket-triage",
      tenantId: "tenant-east",
      workspaceId: "ws-support"
    },
    {
      capabilities: [
        ...capabilities,
        {
          id: "cap_update_ticket",
          targetId: "agt_crm_mcp",
          type: "mcp_tool",
          key: "update_ticket",
          displayName: "update_ticket",
          description: "Update support tickets.",
          action: "write",
          dataDomains: ["support"],
          dataScopes: [{ dataDomain: "support", dataset: "tickets", classification: "confidential" }],
          sensitivity: "confidential",
          riskLevel: "high",
          enforcementMode: "gateway",
          discoveryStatus: "pending_review",
          version: 1,
          discoveredAt: now,
          updatedAt: now
        }
      ]
    }
  );

  assert.deepEqual(draft.allowedCapabilities.map((capability) => capability.key), ["search_customer", "update_ticket"]);
  assert.equal(draft.readiness.canApply, true);
  assert.equal(draft.policyGate.decision, "approval_required");
  assert.equal(draft.policyGate.canApplyDirectly, false);
  assert.equal(draft.policyGate.policyVersion, 1);
  assert.equal(draft.policyGate.reasons.some((reason) => reason.capabilityKey === "update_ticket"), true);
  assert.match(draft.policyGate.nextActions.join(" "), /approval/i);
});

test("subjectIdExampleFromSelector returns a matching concrete subject", () => {
  assert.equal(subjectIdExampleFromSelector(""), undefined);
  assert.equal(subjectIdExampleFromSelector(" user:sales-001 "), "user:sales-001");
  assert.equal(subjectIdExampleFromSelector("user:*"), "user:example");
  assert.equal(subjectIdExampleFromSelector("user:sales-*"), "user:sales-example");
});
