import assert from "node:assert/strict";
import test from "node:test";

import {
  createPermissionPackageDraft,
  permissionPackageTemplates
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

  assert.equal(permissionPackageTemplates.some((template) => template.id === "sales-readonly"), true);
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
