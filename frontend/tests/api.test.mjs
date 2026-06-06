import assert from "node:assert/strict";
import test from "node:test";

import {
  accessDecisionExplainPath,
  permissionPackageApplicationImpactPath,
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
