import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  requiredConsoleCapabilities,
  systemInfoContractIssues,
} from "../src/systemInfoContract.ts";
import * as apiPaths from "../src/apiPaths.ts";
import {
  accessDecisionExplainPath,
  permissionPackageAccessHandoffPath,
  permissionPackageAccessHandoffTokenPath,
  permissionPackageApplicationHealthPath,
  permissionPackageApplicationImpactPath,
  permissionPackageApprovalRequestsPath,
  permissionPackageProductionEvidenceReportPath,
  permissionPackageProductionReportPath,
  permissionPackageProductionReadinessPath
} from "../src/apiPaths.ts";

const apiSource = readFileSync(new URL("../src/api.ts", import.meta.url), "utf8");
const systemInfoContractSource = readFileSync(new URL("../src/systemInfoContract.ts", import.meta.url), "utf8");
const typesSource = readFileSync(new URL("../src/types.ts", import.meta.url), "utf8");

test("permission package workbench preview posts the request body instead of query text", () => {
  assert.match(apiSource, /function previewPermissionPackageWorkbench\(/);
  assert.match(apiSource, /\/api\/v1\/permission-packages\/workbench:preview/);
  assert.match(apiSource, /body,\s*[\r\n\s]*signal/s);
  assert.doesNotMatch(apiSource, /workbench:preview\$\{query\}/);
});

test("permission package access subjects load from the management API", () => {
  assert.match(apiSource, /function fetchPermissionPackageAccessSubjects\(/);
  assert.match(apiSource, /\/api\/v1\/permission-packages\/access-subjects/);
});

test("management API requests include console session cookies", () => {
  assert.match(apiSource, /credentials:\s*['"]include['"]/);
});

test("console session mutations send csrf token from session state", () => {
  assert.match(typesSource, /csrfToken\?: string/);
  assert.match(apiSource, /let consoleCsrfToken = ['"]['"]/);
  assert.match(apiSource, /function setConsoleCsrfToken/);
  assert.match(apiSource, /X-AgentHarbor-CSRF/);
  assert.match(apiSource, /function shouldSendConsoleCsrf/);
  assert.match(apiSource, /setConsoleCsrfToken\(session\.csrfToken\)/);
});

test("API request errors preserve backend error codes", () => {
  assert.match(apiSource, /readonly code\?: string/);
  assert.match(apiSource, /readonly retryAfterSeconds\?: number/);
  assert.match(apiSource, /constructor\(status: number, message: string, code\?: string, retryAfterSeconds\?: number\)/);
  assert.match(apiSource, /retryAfterSeconds\?: number/);
  assert.match(apiSource, /parseRetryAfterSeconds\(response\.headers\.get\(['"]Retry-After['"]\)\)/);
  assert.match(apiSource, /new ApiRequestError\(\s*response\.status,\s*message \|\| `Request failed with status \$\{response\.status\}`,\s*isEnvelope<[^>]+>\(payload\) \? payload\.error : undefined,\s*parseRetryAfterSeconds\(response\.headers\.get\(['"]Retry-After['"]\)\),\s*\)/s);
});

test("console auth API exposes session login and logout endpoints", () => {
  assert.match(apiSource, /function fetchConsoleSession\(/);
  assert.match(apiSource, /\/api\/v1\/auth\/session/);
  assert.match(apiSource, /function loginConsole\(/);
  assert.match(apiSource, /\/api\/v1\/auth\/login/);
  assert.match(apiSource, /body:\s*\{\s*adminKey\s*\}/);
  assert.match(apiSource, /function logoutConsole\(/);
  assert.match(apiSource, /\/api\/v1\/auth\/logout/);
});

test("API health check verifies the system compatibility contract", () => {
  assert.match(systemInfoContractSource, /interface SystemInfo/);
  assert.match(systemInfoContractSource, /authRequired: boolean/);
  assert.match(systemInfoContractSource, /managementMcpToolCatalog:\s*\{\s*metadataVersion: number\s*requiredMetadata: string\[\]\s*catalogDigest: string\s*toolCount: number\s*confirmationRequiredTools: number\s*toolsWithConfirmationSchema: number\s*\}/);
  assert.match(apiSource, /function fetchSystemInfo\(/);
  assert.match(apiSource, /\/api\/v1\/system\/info/);
  assert.match(apiSource, /systemInfoContractIssues\(systemInfo\)/);
  assert.match(systemInfoContractSource, /requiredConsoleCapabilities/);
  assert.match(systemInfoContractSource, /permission_package_approval_withdraw/);
  assert.match(systemInfoContractSource, /permission_package_consumed_approval_recovery/);
  assert.match(systemInfoContractSource, /management_mcp_tools_metadata_v4/);
  assert.match(apiSource, /api_contract_unavailable/);
  assert.match(apiSource, /api_contract_incompatible/);
});

test("system info contract issues validate the management MCP catalog summary", () => {
  const compatibleInfo = {
    apiVersion: "2026-06-15",
    authRequired: true,
    capabilities: requiredConsoleCapabilities,
    managementMcpToolCatalog: {
      metadataVersion: 4,
      requiredMetadata: ["safety", "access", "lifecycle", "execution"],
      catalogDigest: "a".repeat(64),
      toolCount: 22,
      confirmationRequiredTools: 8,
      toolsWithConfirmationSchema: 8,
    },
    name: "AgentHarbor",
  };

  assert.deepEqual(systemInfoContractIssues(compatibleInfo), []);
  assert.deepEqual(
    systemInfoContractIssues({
      ...compatibleInfo,
      managementMcpToolCatalog: { ...compatibleInfo.managementMcpToolCatalog, metadataVersion: 3 },
    }),
    ["managementMcpToolCatalog.metadataVersion"],
  );
  assert.deepEqual(
    systemInfoContractIssues({
      ...compatibleInfo,
      managementMcpToolCatalog: { ...compatibleInfo.managementMcpToolCatalog, requiredMetadata: ["safety", "access", "lifecycle"] },
    }),
    ["managementMcpToolCatalog.requiredMetadata.execution"],
  );
  assert.deepEqual(
    systemInfoContractIssues({
      ...compatibleInfo,
      managementMcpToolCatalog: {
        ...compatibleInfo.managementMcpToolCatalog,
        catalogDigest: "not-a-digest",
      },
    }),
    ["managementMcpToolCatalog.catalogDigest"],
  );
  assert.deepEqual(
    systemInfoContractIssues({
      ...compatibleInfo,
      managementMcpToolCatalog: {
        ...compatibleInfo.managementMcpToolCatalog,
        toolsWithConfirmationSchema: 7,
      },
    }),
    ["managementMcpToolCatalog.toolsWithConfirmationSchema"],
  );
  assert.deepEqual(
    systemInfoContractIssues({
      ...compatibleInfo,
      managementMcpToolCatalog: undefined,
    }),
    [
      "managementMcpToolCatalog.metadataVersion",
      "managementMcpToolCatalog.requiredMetadata.safety",
      "managementMcpToolCatalog.requiredMetadata.access",
      "managementMcpToolCatalog.requiredMetadata.lifecycle",
      "managementMcpToolCatalog.requiredMetadata.execution",
      "managementMcpToolCatalog.toolCount",
      "managementMcpToolCatalog.catalogDigest",
      "managementMcpToolCatalog.confirmationRequiredTools",
      "managementMcpToolCatalog.toolsWithConfirmationSchema",
    ],
  );
});

test("permissionPackageApprovalRequestsPath includes reviewer routing query", () => {
  const path = permissionPackageApprovalRequestsPath({
    limit: 20,
    reviewer: "security-east",
    status: "pending",
    tenantId: "tenant-east",
    workspaceId: "ws-support"
  });

  const url = new URL(path, "http://127.0.0.1:9090");
  assert.equal(url.pathname, "/api/v1/permission-packages/approval-requests");
  assert.equal(url.searchParams.get("reviewer"), "security-east");
  assert.equal(url.searchParams.get("status"), "pending");
  assert.equal(url.searchParams.get("limit"), "20");
  assert.equal(url.searchParams.get("tenantId"), "tenant-east");
  assert.equal(url.searchParams.get("workspaceId"), "ws-support");
});

test("permission package approval request API exposes withdraw endpoint", () => {
  assert.match(apiSource, /function withdrawPermissionPackageApprovalRequest\(/);
  assert.match(apiSource, /permission-packages\/approval-requests\/\$\{encodeURIComponent\(id\)\}\/withdraw/);
  assert.match(apiSource, /body: \{ comment\?: string \}/);
});

test("accessDecisionExplainPath includes effective permission scope query", () => {
  const path = accessDecisionExplainPath({
    callerInstanceId: "caller-sales",
    capabilityId: "cap-search-customer",
    subjectId: "user:sales-001",
    targetId: "mcp-crm",
    tenantId: "tenant-east",
    workspaceId: "ws-sales"
  });

  const url = new URL(path, "http://127.0.0.1:9090");
  assert.equal(url.pathname, "/api/v1/access-decisions:explain");
  assert.equal(url.searchParams.get("callerInstanceId"), "caller-sales");
  assert.equal(url.searchParams.get("capabilityId"), "cap-search-customer");
  assert.equal(url.searchParams.get("subjectId"), "user:sales-001");
  assert.equal(url.searchParams.get("targetId"), "mcp-crm");
  assert.equal(url.searchParams.get("tenantId"), "tenant-east");
  assert.equal(url.searchParams.get("workspaceId"), "ws-sales");
});

test("permissionPackageApplicationImpactPath includes application id and scope query", () => {
  const path = permissionPackageApplicationImpactPath("ppa east/1", {
    tenantId: "tenant-root",
    workspaceId: "ws-sales"
  });

  const url = new URL(path, "http://127.0.0.1:9090");
  assert.equal(url.pathname, "/api/v1/permission-packages/applications/ppa%20east%2F1/impact");
  assert.equal(url.searchParams.get("tenantId"), "tenant-root");
  assert.equal(url.searchParams.get("workspaceId"), "ws-sales");
});

test("permissionPackageApplicationImpactPath can request a read-only rehearsal", () => {
  const path = permissionPackageApplicationImpactPath("ppa-1", {
    rehearsal: "grant_drift",
    tenantId: "tenant-root",
    workspaceId: "ws-sales"
  });

  const url = new URL(path, "http://127.0.0.1:9090");
  assert.equal(url.pathname, "/api/v1/permission-packages/applications/ppa-1/impact");
  assert.equal(url.searchParams.get("tenantId"), "tenant-root");
  assert.equal(url.searchParams.get("workspaceId"), "ws-sales");
  assert.equal(url.searchParams.get("rehearsal"), "grant_drift");
});

test("permissionPackageApplicationHealthPath includes application health filters", () => {
  const path = permissionPackageApplicationHealthPath({
    callerInstanceId: "caller-sales",
    limit: 10,
    targetId: "mcp-crm",
    templateId: "sales-readonly",
    tenantId: "tenant-root",
    workspaceId: "ws-sales"
  });

  const url = new URL(path, "http://127.0.0.1:9090");
  assert.equal(url.pathname, "/api/v1/permission-packages/applications/health");
  assert.equal(url.searchParams.get("callerInstanceId"), "caller-sales");
  assert.equal(url.searchParams.get("limit"), "10");
  assert.equal(url.searchParams.get("targetId"), "mcp-crm");
  assert.equal(url.searchParams.get("templateId"), "sales-readonly");
  assert.equal(url.searchParams.get("tenantId"), "tenant-root");
  assert.equal(url.searchParams.get("workspaceId"), "ws-sales");
});

test("permissionPackageProductionReadinessPath includes status-check filters", () => {
  const path = permissionPackageProductionReadinessPath({
    approvalRequestId: "ppar-1",
    callerInstanceId: "caller-sales",
    region: "us-east",
    requestText: "Allow support triage",
    subjectId: "user:sales-001",
    subjectSelector: "user:sales-*",
    targetId: "mcp-crm",
    templateId: "support-ticket-triage",
    tenantId: "tenant-east",
    traceLimit: 20,
    workspaceId: "ws-sales"
  });

  const url = new URL(path, "http://127.0.0.1:9090");
  assert.equal(url.pathname, "/api/v1/permission-packages/production-readiness");
  assert.equal(url.searchParams.get("approvalRequestId"), "ppar-1");
  assert.equal(url.searchParams.get("callerInstanceId"), "caller-sales");
  assert.equal(url.searchParams.get("region"), "us-east");
  assert.equal(url.searchParams.get("requestText"), "Allow support triage");
  assert.equal(url.searchParams.get("subjectId"), "user:sales-001");
  assert.equal(url.searchParams.get("subjectSelector"), "user:sales-*");
  assert.equal(url.searchParams.get("targetId"), "mcp-crm");
  assert.equal(url.searchParams.get("templateId"), "support-ticket-triage");
  assert.equal(url.searchParams.get("tenantId"), "tenant-east");
  assert.equal(url.searchParams.get("traceLimit"), "20");
  assert.equal(url.searchParams.get("workspaceId"), "ws-sales");
});

test("access handoff paths preserve the permission scope and encode token ids", () => {
  const path = permissionPackageAccessHandoffPath({
    callerInstanceId: "caller-1",
    subjectId: "user:support-001",
    subjectSelector: "user:support-*",
    targetId: "target-1",
    templateId: "support-ticket-triage",
    tenantId: "tenant-east",
    workspaceId: "ws-support",
  });
  assert.match(path, /^\/api\/v1\/permission-packages\/access-handoff\?/);
  assert.match(path, /callerInstanceId=caller-1/);
  assert.match(path, /subjectSelector=user%3Asupport-\*/);
  assert.equal(
    permissionPackageAccessHandoffTokenPath("key/one"),
    "/api/v1/permission-packages/access-handoff/tokens/key%2Fone",
  );
});

test("access handoff API keeps token creation and revocation on dedicated endpoints", () => {
  assert.match(apiSource, /function fetchAccessHandoff\(/);
  assert.match(apiSource, /function createAccessHandoffToken\(/);
  assert.match(apiSource, /delete requestBody\.traceLimit/);
  assert.match(apiSource, /function revokeAccessHandoffToken\(/);
  assert.match(apiSource, /permissionPackageAccessHandoffTokenPath\(id\)/);
});

test("permissionPackageAcceptanceReportPath includes acceptance report filters", () => {
  assert.equal(typeof apiPaths.permissionPackageAcceptanceReportPath, "function");

  const path = apiPaths.permissionPackageAcceptanceReportPath({
    approvalRequestId: "ppar-1",
    callerInstanceId: "caller-sales",
    region: "us-east",
    requestText: "Allow support triage",
    subjectId: "user:sales-001",
    subjectSelector: "user:sales-*",
    targetId: "mcp-crm",
    templateId: "support-ticket-triage",
    tenantId: "tenant-east",
    traceLimit: 20,
    workspaceId: "ws-sales"
  });

  const url = new URL(path, "http://127.0.0.1:9090");
  assert.equal(url.pathname, "/api/v1/permission-packages/production-readiness/report");
  assert.equal(url.searchParams.get("approvalRequestId"), "ppar-1");
  assert.equal(url.searchParams.get("callerInstanceId"), "caller-sales");
  assert.equal(url.searchParams.get("region"), "us-east");
  assert.equal(url.searchParams.get("requestText"), "Allow support triage");
  assert.equal(url.searchParams.get("subjectId"), "user:sales-001");
  assert.equal(url.searchParams.get("subjectSelector"), "user:sales-*");
  assert.equal(url.searchParams.get("targetId"), "mcp-crm");
  assert.equal(url.searchParams.get("templateId"), "support-ticket-triage");
  assert.equal(url.searchParams.get("tenantId"), "tenant-east");
  assert.equal(url.searchParams.get("traceLimit"), "20");
  assert.equal(url.searchParams.get("workspaceId"), "ws-sales");
});

test("legacy report path helpers remain compatibility aliases", () => {
  assert.equal(permissionPackageProductionReportPath, apiPaths.permissionPackageAcceptanceReportPath);
  assert.equal(permissionPackageProductionEvidenceReportPath, apiPaths.permissionPackageAcceptanceReportPath);

  const path = permissionPackageProductionReportPath({
    approvalRequestId: "ppar-1",
    callerInstanceId: "caller-sales",
    region: "us-east",
    requestText: "Allow support triage",
    subjectId: "user:sales-001",
    subjectSelector: "user:sales-*",
    targetId: "mcp-crm",
    templateId: "support-ticket-triage",
    tenantId: "tenant-east",
    traceLimit: 20,
    workspaceId: "ws-sales"
  });

  const url = new URL(path, "http://127.0.0.1:9090");
  assert.equal(url.pathname, "/api/v1/permission-packages/production-readiness/report");
  assert.equal(url.searchParams.get("approvalRequestId"), "ppar-1");
  assert.equal(url.searchParams.get("callerInstanceId"), "caller-sales");
  assert.equal(url.searchParams.get("region"), "us-east");
  assert.equal(url.searchParams.get("requestText"), "Allow support triage");
  assert.equal(url.searchParams.get("subjectId"), "user:sales-001");
  assert.equal(url.searchParams.get("subjectSelector"), "user:sales-*");
  assert.equal(url.searchParams.get("targetId"), "mcp-crm");
  assert.equal(url.searchParams.get("templateId"), "support-ticket-triage");
  assert.equal(url.searchParams.get("tenantId"), "tenant-east");
  assert.equal(url.searchParams.get("traceLimit"), "20");
  assert.equal(url.searchParams.get("workspaceId"), "ws-sales");
});

test("acceptance report fetch helper is the primary report implementation", () => {
  assert.match(apiSource, /function fetchPermissionPackageAcceptanceReport\(/);
  assert.match(apiSource, /permissionPackageAcceptanceReportPath\(filter\)/);
  assert.match(apiSource, /fetchPermissionPackageProductionReport = fetchPermissionPackageAcceptanceReport/);
  assert.match(apiSource, /fetchPermissionPackageProductionEvidenceReport = fetchPermissionPackageAcceptanceReport/);
  assert.doesNotMatch(apiSource, /function fetchPermissionPackageProductionReport\(/);
});
