import assert from "node:assert/strict";
import test from "node:test";

import {
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
