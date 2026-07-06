import assert from "node:assert/strict";
import test from "node:test";

import { healthCheckFailureDetail } from "../src/healthCheckPresentation.ts";

function t(key) {
  const messages = {
    "message.apiContractIncompatible": "API is missing capabilities: {capabilities}.",
    "message.apiContractIncompatibleManagementCatalog": "Management MCP catalog contract is incompatible.",
    "message.apiContractIncompatibleUnknown": "API compatibility contract is incompatible.",
    "message.apiContractUnavailable": "API compatibility check is unavailable.",
    "systemCapability.permissionPackageApprovalRequests": "Approval queue",
    "systemCapability.permissionPackageApplyPreflight": "Apply preflight"
  };
  return messages[key] ?? key;
}

test("health check presentation hides raw management MCP catalog contract issue keys", () => {
  const detail = healthCheckFailureDetail(t, "API", {
    code: "api_contract_incompatible",
    contractIssues: ["managementMcpToolCatalog.requiredMetadata.access"],
    message: "system info contract issues: managementMcpToolCatalog.requiredMetadata.access",
    missingCapabilities: [],
    status: "error"
  });

  assert.equal(detail, "Management MCP catalog contract is incompatible.");
});

test("health check presentation renders missing API capabilities as readable labels", () => {
  const detail = healthCheckFailureDetail(t, "API", {
    code: "api_contract_incompatible",
    message: "missing capabilities: permission_package_approval_requests, permission_package_apply_preflight",
    missingCapabilities: ["permission_package_approval_requests", "permission_package_apply_preflight"],
    status: "error"
  });

  assert.equal(detail, "API is missing capabilities: Approval queue, Apply preflight.");
  assert.doesNotMatch(detail, /permission_package_/);
});

test("health check presentation hides unknown raw API contract issue keys", () => {
  const detail = healthCheckFailureDetail(t, "API", {
    code: "api_contract_incompatible",
    contractIssues: ["futureContract.requiredField"],
    message: "system info contract issues: futureContract.requiredField",
    missingCapabilities: [],
    status: "error"
  });

  assert.equal(detail, "API compatibility contract is incompatible.");
  assert.doesNotMatch(detail, /futureContract/);
});
