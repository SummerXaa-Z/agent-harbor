# AgentHarbor API Reference

Endpoint reference for the management plane, data plane, and read-only observability surfaces. Extracted from the README so the front door stays short. Configuration and security context live in [architecture.md](architecture.md) and [.env.example](../../.env.example).

## Health and Contracts

- `GET /healthz`
- `GET /api/v1/system/info`
- `GET /api/v1/contracts/providers`
- `GET /api/v1/contracts/channels`

The web console reads `GET /api/v1/system/info` after `/healthz` to verify API compatibility before running permission changes. The same metadata reports whether console authentication is required, so connection diagnostics can distinguish deployment-style login requirements from explicit local development bypass. If the endpoint is unavailable or required capabilities are missing, the console blocks runtime validation with an upgrade prompt instead of surfacing late-stage business errors.

Web 控制台会在 `/healthz` 之后读取 `GET /api/v1/system/info`，先确认 API 兼容信息，再执行权限变更。同一份兼容信息也会返回控制台是否要求登录，因此连接诊断可以区分部署式登录要求和显式本地开发绕过。如果端点不可用或缺少必要能力，控制台会在运行验证前提示升级 API，而不是让管理员在后续流程里遇到零散业务错误。

## Tenants and Access Profile

- `POST /api/v1/tenants`
- `GET /api/v1/tenants?tenantId=&parentTenantId=`
- `GET /api/v1/tenants/{id}`
- `GET /api/v1/tenants/{id}/access-profile?workspaceId=&targetId=&capabilityId=&callerInstanceId=&traceLimit=`

The tenant access profile endpoint is read-only: configured grants, effective scope calculations, invalid historical scope records, and recent trace records for a registered tenant subtree. `traceLimit=0` disables recent traces.

## Agents and Keys

- `POST /api/v1/agents`
- `GET /api/v1/agents?tenantId=&workspaceId=`
- `GET /api/v1/agents/{id}`
- `PATCH /api/v1/agents/{id}`
- `DELETE /api/v1/agents/{id}`
- `POST /api/v1/agents/{id}/credentials:rotate`
- `POST /api/v1/agent-keys`
- `GET /api/v1/api-keys?tenantId=&workspaceId=`
- `POST /api/v1/api-keys`
- `DELETE /api/v1/api-keys/{id}`

## Route Policies and Legacy Grants

- `POST /api/v1/access-grants`
- `GET /api/v1/access-grants?tenantId=&workspaceId=`
- `DELETE /api/v1/access-grants/{id}`
- `POST /api/v1/route-policies`
- `GET /api/v1/route-policies?tenantId=&workspaceId=`
- `PATCH /api/v1/route-policies/{id}`
- `DELETE /api/v1/route-policies/{id}`

## Capabilities, Assignments, and Permission Packages

- `POST /api/v1/targets/{targetId}/capabilities:refresh`
- `POST /api/v1/targets/{targetId}:probe`
- `GET /api/v1/capabilities?tenantId=&workspaceId=&targetId=&status=`
- `PATCH /api/v1/capabilities/{id}`
- `GET /api/v1/access-decisions:explain?tenantId=&workspaceId=&callerInstanceId=&targetId=&capabilityId=&subjectId=`
- `GET /api/v1/permission-packages/templates`
- `GET /api/v1/permission-packages/access-subjects`
- `POST /api/v1/permission-packages/drafts`
- `POST /api/v1/permission-packages/approval-requests`
- `GET /api/v1/permission-packages/approval-requests?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&requestedCapabilityId=&status=&reviewer=&limit=`
- `POST /api/v1/permission-packages/approval-requests/{id}/approve`
- `POST /api/v1/permission-packages/approval-requests/{id}/reject`
- `POST /api/v1/permission-packages:preflight`
- `POST /api/v1/permission-packages:apply`
- `GET /api/v1/permission-packages/applications?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&requestedCapabilityId=&limit=`
- `GET /api/v1/permission-packages/applications/health?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&requestedCapabilityId=&limit=`
- `GET /api/v1/permission-packages/production-readiness?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&requestedCapabilityId=&subjectId=&traceLimit=`
- `GET /api/v1/permission-packages/production-readiness/report?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&requestedCapabilityId=&subjectId=&traceLimit=`
- `GET /api/v1/permission-packages/access-handoff?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&requestedCapabilityId=&subjectId=&traceLimit=`
- `POST /api/v1/permission-packages/access-handoff/tokens`
- `POST /api/v1/permission-packages/access-handoff/events` (`action` is `config_viewed` or `config_copied`, for audit)
- `DELETE /api/v1/permission-packages/access-handoff/tokens/{id}`
- `GET /api/v1/permission-packages/applications/{id}/impact?tenantId=&workspaceId=&rehearsal=`
- `POST /api/v1/management/mcp`
- `POST /api/v1/management/mcp/rpc`
- `POST /api/v1/tenant-entitlements`
- `GET /api/v1/tenant-entitlements?tenantId=&workspaceId=&targetId=&capabilityId=`
- `DELETE /api/v1/tenant-entitlements/{id}` (disables; `409 GRANT_CHAIN_CHILDREN_ACTIVE` while enabled workspace assignments reference it)
- `POST /api/v1/workspace-assignments`
- `GET /api/v1/workspace-assignments?tenantId=&workspaceId=&entitlementId=`
- `DELETE /api/v1/workspace-assignments/{id}` (disables; `409` while enabled instance assignments reference it)
- `POST /api/v1/instance-assignments`
- `GET /api/v1/instance-assignments?tenantId=&workspaceId=&callerInstanceId=&capabilityId=`
- `DELETE /api/v1/instance-assignments/{id}` (disables)

Management MCP exposes the same permission-package workflow as JSON-RPC tools (`draft_permission_package`, `preflight_permission_package`, `check_permission_package_production_readiness`, `export_permission_package_production_report`, approval-request tools, `apply_permission_package`, `explain_*`, `get_tenant_access_profile`, `list_agents`, `list_capabilities`, …), guarded by the same admin authentication and audit trail. Legacy report-export aliases remain; see [Management MCP compatibility aliases](management-mcp-compatibility-aliases.md).

### Exact-capability semantics (v0.3.0+) / 精确能力语义

- A non-empty `requestedCapabilityId` means an **exact one-capability request**: clients must carry the same value through draft, approval, apply, application/health lookup, production readiness/report, Access Handoff, and token creation. An omitted or empty value intentionally preserves the legacy template-bundle behavior. / 非空 `requestedCapabilityId` 表示**只申请这一项能力**，调用方必须在草案、审批、应用、落地记录/健康查询、上线就绪/报告、接入交付和 Token 创建阶段持续传递同一个值；省略或传空值保留旧版模板整包语义。
- Readiness and Access Handoff may recognize a legacy application with an empty stored `requestedCapabilityId` as exact-equivalent only when its `allowedCapabilityIds` contains exactly the requested capability and all current scope, template-version, data-scope, and drift checks still pass (read-time compatibility only; no provenance backfill). / 只有当旧应用记录的允许能力集合恰好只有被查询的这一项，且当前范围、模板版本、数据范围和漂移检查全部通过时，才会被视为精确请求的等价记录；该规则只用于读取兼容，不回填来源语义。
- Built-in templates are version 2: a capability must declare at least one explicit data domain (`dataDomains` or `dataScopes[].dataDomain`), every non-empty declared domain must equal the template's `defaultDataDomain`, and missing, unsupported, or mixed domains fail closed. / 内置模板已升级到 v2：能力必须显式声明至少一个数据域，且所有非空声明都必须与所选模板的 `defaultDataDomain` 一致；缺失、未知或混合数据域都会默认阻断。

## Data Plane

- `GET /api/v1/self/access-profile`
- `POST /api/v1/mcp/agents/{targetId}`
- `POST /api/v1/mcp/agents/{targetId}/rpc`
- `POST /api/v1/openapi/agents/{targetId}/operations/{operationId}`
- `ANY /api/v1/openapi/agents/{targetId}/{relativePath...}`

`GET /api/v1/self/access-profile` is the caller-facing counterpart of the tenant profile: any agent key (with `X-AgentHarbor-Subject-Id` when applicable) reads its own effective boundary — caller identity, key kind and expiry, and per-target capability lists evaluated exactly like the governed data plane. Access Handoff tokens are bounded by their application binding; a hash-matched but dead token reports whether it was revoked or expired, while guessed tokens keep the generic invalid message.

## Audit, Traces, and Metrics

- `GET /api/v1/audit/events?tenantId=&workspaceId=&action=&resourceType=&resourceId=&since=&until=&limit=`
- `GET /api/v1/audit/traces?tenantId=&workspaceId=&runId=&decision=&callerAgentId=&targetAgentId=&since=&until=&limit=`
- `GET /api/v1/metrics/runtime?tenantId=&workspaceId=`
- `GET /api/v1/metrics/daily?tenantId=&workspaceId=&days=&tzOffsetMinutes=`

Audit and trace lists return rows in ascending time order. `limit` keeps the newest rows (audit events default to 100, traces are unbounded unless `limit` is set, both cap at 500); `since` is inclusive and `until` exclusive, both RFC3339. Daily metrics count gateway decisions and audit events per local day (`days` 1–30, `tzOffsetMinutes` −720–840) with the same visibility rules as the lists, zero-filled days, `denyRate: null` on days without calls, and `truncated: true` when a window exceeds the server row cap.

## Target Probe

`POST /api/v1/targets/{targetId}:probe` sends the same `tools/list` request as a capability refresh but writes neither capabilities nor audit events. A reachability failure is returned as a `200` result with `status: "error"` and an `UPSTREAM_*` `errorCode` (connect, DNS, TLS, timeout, or generic); only an unknown target (`404`), a target outside the caller's management scope (`403`), a non-MCP target, or an invalid agent configuration (`400`) is an HTTP error.
