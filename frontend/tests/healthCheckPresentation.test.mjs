import assert from "node:assert/strict";
import test from "node:test";

import { healthCheckFailureDetail } from "../src/healthCheckPresentation.ts";

function t(key) {
  const messages = {
    "message.apiContractIncompatible": "API is missing capabilities: {capabilities}.",
    "message.apiContractIncompatibleManagementCatalog": "Management MCP catalog contract is incompatible.",
    "message.apiContractUnavailable": "API compatibility check is unavailable."
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
