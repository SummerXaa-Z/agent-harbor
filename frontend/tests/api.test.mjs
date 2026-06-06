import assert from "node:assert/strict";
import test from "node:test";

import {
  accessDecisionExplainPath,
  permissionPackageApprovalRequestsPath
} from "../src/apiPaths.ts";

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
