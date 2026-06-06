# AgentHarbor

AgentHarbor is a tenant-first access governance and permission operations platform for AI agents, MCP tools, OpenAPI services, and governed data access.

AgentHarbor 是面向 AI Agent、MCP 工具、OpenAPI 服务和受治理数据访问的租户优先权限治理与权限运营平台。

It helps platform, security, and tenant operations teams answer one production question: which tenant, workspace, caller instance, and subject can access which tool or data scope, why, and with what approval evidence.

它帮助平台、安全和租户运营团队回答一个生产问题：哪个租户、工作区、调用方实例和主体可以访问哪些工具或数据范围，为什么可以访问，以及对应的审批和审计证据是什么。

## Positioning

AgentHarbor supports MCP gateway capabilities, but it is not positioned as another generic MCP gateway. MCP servers are one governed target type. The product identity is tenant-first access governance, AI-friendly permission operations, and evidence-backed runtime enforcement across MCP tools, APIs, agents, and data systems.

AgentHarbor 支持 MCP 网关能力，但不把自己定位成另一个通用 MCP Gateway。MCP Server 只是被治理的目标类型之一。AgentHarbor 的产品身份是租户优先的访问治理、面向 AI 的权限运营，以及覆盖 MCP 工具、API、Agent 和数据系统的证据化运行时控制。

## Project Status

AgentHarbor is in developer preview. It is ready for local evaluation, design feedback, and early contribution, but it is not yet recommended for production traffic.

AgentHarbor 当前处于开发者预览阶段，适合本地评估、设计反馈和早期贡献；暂不建议承载生产流量。

Open-source timing is intentionally secondary to production hardening. Before any release-readiness claim, the current standard is that the safety baseline, release checks, and primary AI Admin journey all pass from a fresh local checkout.

开源节奏会服从生产可用性。任何发布就绪声明之前，都必须确保安全基线、发布检查和核心 AI Admin 用户旅程能在全新本地检出中通过。

## What It Provides

- **Tenant-first governance / 租户优先治理**: register a three-level tenant tree and scope management views by tenant subtree. 注册三级租户树，并按租户子树限定管理视图。
- **Agent and target registry / Agent 与目标注册**: manage caller agents, MCP targets, OpenAPI services, webhooks, credentials, and short-lived Agent Keys. 管理调用方 Agent、MCP 目标、OpenAPI 服务、Webhook、凭据和短期 Agent Key。
- **Route policy controls / 路由策略控制**: allow or deny MCP/OpenAPI routes with priority, wildcard matching, and bounded retry overrides. 通过优先级、通配匹配和有界重试覆盖来允许或拒绝 MCP/OpenAPI 路由。
- **Capability governance / 能力治理**: discover target tools, approve capabilities, and grant them through tenant, workspace, and caller-instance assignments. 发现目标工具，审批能力，并通过租户、工作区和调用方实例分配进行授权。
- **Data permission enforcement / 数据权限控制**: narrow `dataScopes` across capability, tenant entitlement, workspace assignment, and instance assignment boundaries. 在能力、租户授权、工作区分配和实例分配边界上逐层收敛 `dataScopes`。
- **AI-friendly permission operations / AI 友好的权限运营**: draft tenant-scoped permission package changes from administrator intent, preview allow/deny outcomes, run approval-required packages, apply them through the grant chain, and review structured application impact. 从管理员意图生成租户范围的权限包草案，预览允许/拒绝结果，运行需审批权限包，通过授权链落地，并复盘结构化应用影响。
- **Approval and evidence / 审批与证据**: route approval queues to configured reviewers, expire and consume approval requests, record applied package evidence, review active or missing created objects, and keep audit trails for every privileged permission change. 将审批队列路由给已配置审查员，对审批请求设置过期和一次性消费，记录权限包应用证据，复盘已创建对象是否仍有效，并为高风险权限变更保留审计链。
- **Runtime evidence / 运行时证据**: record traces, audit events, metrics, upstream attempts, effective data scopes, and deny reasons. 记录 trace、审计事件、指标、上游尝试、有效数据范围和拒绝原因。
- **Tenant Permission Console / 租户权限控制台**: inspect each tenant's effective access profile, grant chain, invalid scope rows, and recent trace evidence. 查看每个租户的有效访问画像、授权链、无效范围行和近期运行证据。

## Core Model

```text
Tenant
  -> Agent or target service
  -> MCP/OpenAPI capability
  -> Tenant entitlement
  -> Workspace assignment
  -> Caller instance assignment
  -> Runtime decision and trace evidence
```

The tenant is the primary control boundary. A registered tenant can manage its own subtree; unregistered tenant strings keep exact-match behavior for compatibility.

The data plane uses short-lived Agent Keys. Management APIs can be protected with `AGENT_HARBOR_ADMIN_KEY`, which requires callers to send the same value in `X-Admin-Key`.

## Quick Start

Use the repository toolchain pins before running local commands:

- Go version comes from `go.mod`.
- Node major version comes from `.node-version`.
- Frontend package manager comes from `frontend/package.json`.

```bash
make demo
```

Then open `http://127.0.0.1:5174/`. The demo command starts the API, the dependency-free mock MCP server, and the web console together for the first browser evaluation.

In the Cockpit's **Core Journey Workbench**, confirm the preflight rows show the API service and Mock MCP service as ready, then run the core journey. A successful run reaches `6/6` and leaves allowed/denied runtime evidence plus the tenant access profile visible in the console.

Use the local release gate when you want to verify the repository rather than run the browser demo:

```bash
make check
```

`make check` installs the pinned frontend dependencies from `frontend/pnpm-lock.yaml` before running frontend tests and builds.

Use the production safety gate when you want to verify conservative runtime defaults:

```bash
make production-hardening
```

This starts a local API with `AGENT_HARBOR_ADMIN_KEY` configured and private upstreams disabled. It verifies health remains public, management APIs reject missing or wrong admin keys, permission-package and management MCP endpoints use the same admin-key protection, loopback MCP targets are rejected by default, and public HTTPS MCP targets remain registrable.

The API listens on `:9090` by default. Override it with:

```bash
AGENT_HARBOR_ADDR=:9091 go run ./cmd/agent-harbor
```

## Try the Core Journey in 10 Minutes

This local scenario runs the most important AgentHarbor workflow: create a three-level tenant tree, register a mock MCP target, discover tools, approve one tool, assign it to a tenant/workspace/caller instance, run allowed and denied calls, and verify access-profile plus audit evidence.

Terminal 1:

```bash
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
```

Terminal 2:

```bash
make core-journey
```

The scenario starts `scripts/mock-mcp-server.py` automatically and points AgentHarbor at `http://127.0.0.1:8787/mcp`. The `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` flag is required only for local development scenarios that use loopback or private-network upstreams; do not enable it for production deployments.

## Try the AI Admin Approval Journey

The browser workbench proves the v0.2.0 approval-required permission package path for the primary product journey: create a three-level tenant tree, register a caller and MCP target, discover read/write/export tools, draft a **Support ticket triage** permission package, create and approve a matching approval request, apply it with `approvalRequestId`, run allowed and denied MCP calls with a subject id, and verify access-profile, application, approval, trace, audit, and read-only impact evidence. The AI Admin workspace also includes a Reviewer queue that filters pending approvals by reviewer identity and the current package scope. Approval requests expire after 24 hours and are consumed transactionally by the first successful package application. The impact review panel resolves the created entitlement, workspace assignment, instance assignment, and capability ids so operators can inspect rollback readiness before any future rollback mutation exists. The adjacent **Rehearse drift** action calls `rehearsal=grant_drift` to simulate missing/inactive grant blockers in the response only; it does not mutate permission state.

浏览器工作台验证 v0.2.0 需要审批的权限包主旅程：创建三级租户树，注册调用方和 MCP 目标，发现读/写/导出工具，起草 **客服工单处理包**，创建并批准匹配的审批请求，携带 `approvalRequestId` 应用权限包，用主体 ID 跑通允许和拒绝调用，并验证访问画像、应用记录、审批、追踪、审计和只读影响证据。相邻的 **演练漂移** 动作会调用 `rehearsal=grant_drift`，仅在响应中模拟缺失/未启用授权阻断，不会修改权限状态。

```bash
make demo
```

Then open `http://127.0.0.1:5174/`, switch to **AI Admin**, run **Live Approval Journey**, click **Review impact** on the application evidence panel, and optionally click **Rehearse drift** to preview the read-only blocker state.

然后打开 `http://127.0.0.1:5174/`，切换到 **AI Admin**，运行 **Live Approval Journey / 跑通审批旅程**，在应用证据面板点击 **Review impact / 复盘影响**，也可以点击 **Rehearse drift / 演练漂移** 预览只读阻断状态。

Use the CLI scenario when you want the same path as a scriptable regression check:

Terminal 1:

```bash
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
```

Terminal 2:

```bash
make scenario-permission-package-approval
```

The scenario starts `scripts/mock-mcp-server.py` automatically and uses the mock `update_ticket` write tool to exercise the approval-required gate. It also verifies that the approved request is marked consumed after apply and cannot be reused.

For release-candidate validation of the browser-facing path, run:

```bash
make ai-admin-browser-journey
```

This starts the API, Mock MCP, and web console, verifies browser CORS allows `X-AgentHarbor-Subject-Id`, then runs the approval-required package scenario against those services.

For release-candidate validation of production defaults, run:

```bash
make production-hardening
```

`make release-check` also includes this safety baseline.

## Web Console

For the first browser evaluation, run:

```bash
make demo
```

Then open `http://127.0.0.1:5174/` and use the Cockpit's **Core Journey Workbench**. The workbench checks API and Mock MCP readiness before enabling the run button. It also includes a **Reset demo session** action that clears the current browser session state and filters without deleting backend data; each run uses fresh `ui-core-*` identifiers so historical evidence remains inspectable.

The **AI Admin** workspace includes a live approval journey workbench for the approval-required **Support ticket triage** path. Each run uses fresh `ui-approval-*` identifiers, applies the package through live APIs, sends runtime MCP calls with `X-AgentHarbor-Subject-Id`, and surfaces the run id, subject id, application record, application impact review, tenant access profile, traces, and applied audit event in the console.

The AI Admin workbench also shows first-run readiness for the API, Mock MCP, browser subject-header CORS, local private-upstream mode, and current data source before the live journey runs.

`make demo` starts:

- AgentHarbor API at `http://127.0.0.1:9090`
- Mock MCP server at `http://127.0.0.1:8787/mcp`
- Web console at `http://127.0.0.1:5174`

Use `Ctrl+C` in the demo terminal to stop all demo services.

If you need to troubleshoot a single service, use the manual three-terminal path:

Terminal 1:

```bash
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
```

Terminal 2:

```bash
make mock-mcp
```

Terminal 3:

```bash
cd frontend
pnpm install
pnpm dev
```

The console reads `VITE_API_BASE`; if unset, it uses `http://127.0.0.1:9090`. When the backend is unavailable during local development, the UI falls back to sample data so the console remains navigable.

The Core Journey Workbench creates a fresh tenant tree, caller, MCP target, scoped capability grant chain, allowed call, denied call, and tenant access profile evidence through the real API. The core console journey supports English and Simplified Chinese. The browser language is used on first load, and the visible `中文` / `EN` toggle persists the operator's choice locally.

The console also includes an **AI Admin** workspace for the v0.2.0 permission-package journey. It lets an administrator describe an access request, select a deterministic permission package template, preview allow/deny simulation rows, apply low-risk packages through the backend permission-package API, request approval for high-risk packages, and then inspect the refreshed tenant access profile. Each draft includes a policy gate: direct apply is allowed for low-risk read-oriented packages, while write, export, admin, high-risk, critical-risk, confidential, or restricted allowed capabilities require an approval request before apply. Approved requests snapshot the draft, template version, scope, allowed capabilities, data scopes, and policy-gate reasons so apply rejects drift before writing permissions. Approval requests expire after 24 hours and are consumed in the same repository transaction or lock as the successful application, so the same approval cannot apply the package twice even under concurrent reuse. When `AGENT_HARBOR_APPROVAL_REVIEWERS` is configured, the Reviewer queue and approve/reject operations are scoped to the reviewer's configured tenant subtree and workspace. The AI Admin and tenant access-profile workspaces include a read-only effective permission explanation panel so operators can inspect why a tenant/workspace/caller/capability decision is currently allowed or denied, which evidence layer matched or blocked, and which remediation step comes next. Each successful package application records the template version, draft id, created entitlement and assignment ids, capability ids, and data scopes; the applied audit event links the approval request id when one was used. The read-only impact review endpoint and AI Admin panel resolve those recorded ids against current state, show created/active/missing object counts, mark capability rollback as manual review, and return a read-only remediation plan with ordered manual-review, disable, drift-investigation, and final verification actions without executing rollback. Drift blockers also include stable `blockerCodes` such as `missing_created_objects`, `inactive_created_objects`, and `no_allowed_capabilities` so UI and admin agents can localize and reason about unsafe states reliably. The optional `rehearsal=grant_drift` impact query and AI Admin **Rehearse drift** button simulate those blockers without changing any grants, which lets operators practice remediation review safely. See [the v0.2.0 journey note](docs/product/0.2.0-ai-admin-permission-journey.md).

AI Admin 工作区还承载 v0.2.0 权限包旅程。管理员可以描述访问需求、选择确定性权限包模板、预览允许/拒绝模拟行、通过后端权限包 API 应用低风险权限包、为高风险权限包发起审批，并查看刷新的租户访问画像。只读影响复盘端点和 AI Admin 面板会把记录中的授权对象与当前状态对齐，展示已创建/有效/缺失对象数量，返回只读处置计划，并用稳定的 `blockerCodes` 解释缺失、未启用或无允许能力等不安全状态。可选的 `rehearsal=grant_drift` 查询和 **演练漂移** 按钮会在不修改任何授权的前提下模拟这些阻断，让操作员安全练习处置复盘。

AgentHarbor also exposes the same workflow as a management MCP endpoint at `POST /api/v1/management/mcp`. Admin agents can call `tools/list` and then use tools such as `draft_permission_package`, `create_permission_package_approval_request`, `list_permission_package_approval_requests`, `approve_permission_package_approval_request`, `reject_permission_package_approval_request`, `apply_permission_package`, `list_permission_package_applications`, `explain_permission_package_draft`, `explain_access_decision`, `get_tenant_access_profile`, `list_agents`, and `list_capabilities`. REST also exposes `GET /api/v1/permission-packages/applications/{id}/impact?tenantId=&workspaceId=` for read-only application impact review and remediation planning, plus `rehearsal=grant_drift` for response-only drift rehearsal. `list_permission_package_approval_requests` accepts an optional `reviewer` field so an admin agent can fetch only the requests that reviewer is allowed to handle; approve and reject validate the same reviewer route before status changes. When `AGENT_HARBOR_ADMIN_KEY` is configured, this endpoint requires `X-Admin-Key` like the rest of the management API.

AgentHarbor 也通过 `POST /api/v1/management/mcp` 暴露同一套管理工作流。REST 端点 `GET /api/v1/permission-packages/applications/{id}/impact?tenantId=&workspaceId=` 用于只读应用影响复盘和处置规划，也支持 `rehearsal=grant_drift` 做仅影响响应的漂移演练。配置 `AGENT_HARBOR_ADMIN_KEY` 后，该端点与其他管理 API 一样需要 `X-Admin-Key`。

## Runtime Configuration

Use `.env.example` as the local configuration template.

| Variable | Purpose |
| --- | --- |
| `AGENT_HARBOR_ADDR` | API listen address. Defaults to `:9090`. |
| `AGENT_HARBOR_ADMIN_KEY` | Optional management API key. When set, management and audit endpoints require `X-Admin-Key`. |
| `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` | Development-only boolean. Allows loopback/private upstream endpoints for local scenarios when set to `true`. Defaults to `false`. |
| `AGENT_HARBOR_APPROVAL_REVIEWERS` | Optional approval reviewer routing rules. Use comma or semicolon separated `reviewer=tenantId/workspaceId` entries, for example `security-root=tenant-root/*;security-east=tenant-east/ws-support`. `*` allows any tenant or workspace. When unset, the developer-preview approval flow keeps accepting any non-empty reviewer. 可选审批审查员路由规则，使用逗号或分号分隔的 `reviewer=tenantId/workspaceId`；未设置时保留开发预览兼容行为。 |
| `AGENT_HARBOR_DATABASE_URL` | Optional PostgreSQL connection string. If unset, AgentHarbor uses the in-memory repository. |
| `AGENT_HARBOR_CREDENTIAL_KEY` | 32-byte raw or base64 key used to encrypt persisted agent credentials. Required with PostgreSQL. |
| `AGENT_HARBOR_TEST_DATABASE_URL` | PostgreSQL connection string used by integration tests. |
| `VITE_API_BASE` | Frontend API base URL. |

Run `make production-hardening` before any deployment-style handoff. It proves that `AGENT_HARBOR_ADMIN_KEY` protection is enforced across management APIs and that `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` stays disabled unless explicitly set for local development scenarios.

PostgreSQL example:

```bash
AGENT_HARBOR_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
AGENT_HARBOR_CREDENTIAL_KEY='0123456789abcdef0123456789abcdef' \
  go run ./cmd/agent-harbor
```

## Local Verification

```bash
make frontend-deps
make test
make test-fresh
make vet
make build
make frontend-test
make frontend-build
make scenario-scripts-lint
make github-config-lint
```

PostgreSQL integration remains opt-in:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

## Scenario Scripts

The repository includes executable scenario scripts under `scripts/` for local end-to-end smoke coverage. Start the API first, then run:

```bash
make scenario-all
```

The core journey has its own script because it intentionally uses a local mock MCP endpoint:

```bash
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
make core-journey
```

The approval-required permission package journey also uses the local mock MCP endpoint:

```bash
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
make scenario-permission-package-approval
```

This scenario verifies approval expiry metadata and one-time approval consumption in addition to the runtime allow/deny checks.

With admin protection enabled:

```bash
AGENT_HARBOR_ADMIN_KEY=local-admin-key go run ./cmd/agent-harbor
ADMIN_KEY=local-admin-key make scenario-all
```

For MCP capability scenarios, provide a safe public test endpoint because AgentHarbor rejects loopback and private-network target endpoints by design:

```bash
MCP_ENDPOINT=https://mcp.example.test/rpc \
ALLOWED_TOOL=search_customer \
DENIED_TOOL=export_contracts \
ADMIN_KEY=local-admin-key \
  make scenario-all
```

## API Overview

### Health and Contracts

- `GET /healthz`
- `GET /api/v1/contracts/providers`
- `GET /api/v1/contracts/channels`

### Tenants and Access Profile

- `POST /api/v1/tenants`
- `GET /api/v1/tenants?tenantId=&parentTenantId=`
- `GET /api/v1/tenants/{id}`
- `GET /api/v1/tenants/{id}/access-profile?workspaceId=&targetId=&capabilityId=&callerInstanceId=&traceLimit=`

### Agents and Keys

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

### Route Policies and Legacy Grants

- `POST /api/v1/access-grants`
- `GET /api/v1/access-grants?tenantId=&workspaceId=`
- `DELETE /api/v1/access-grants/{id}`
- `POST /api/v1/route-policies`
- `GET /api/v1/route-policies?tenantId=&workspaceId=`
- `PATCH /api/v1/route-policies/{id}`
- `DELETE /api/v1/route-policies/{id}`

### Capabilities and Assignments

- `POST /api/v1/targets/{targetId}/capabilities:refresh`
- `GET /api/v1/capabilities?tenantId=&workspaceId=&targetId=&status=`
- `PATCH /api/v1/capabilities/{id}`
- `GET /api/v1/access-decisions:explain?tenantId=&workspaceId=&callerInstanceId=&targetId=&capabilityId=&subjectId=`
- `GET /api/v1/permission-packages/templates`
- `POST /api/v1/permission-packages/drafts`
- `POST /api/v1/permission-packages/approval-requests`
- `GET /api/v1/permission-packages/approval-requests?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&status=&reviewer=&limit=`
- `POST /api/v1/permission-packages/approval-requests/{id}/approve`
- `POST /api/v1/permission-packages/approval-requests/{id}/reject`
- `POST /api/v1/permission-packages:apply`
- `GET /api/v1/permission-packages/applications?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&limit=`
- `POST /api/v1/management/mcp`
- `POST /api/v1/management/mcp/rpc`
- `POST /api/v1/tenant-entitlements`
- `GET /api/v1/tenant-entitlements?tenantId=&workspaceId=&targetId=&capabilityId=`
- `POST /api/v1/workspace-assignments`
- `GET /api/v1/workspace-assignments?tenantId=&workspaceId=&entitlementId=`
- `POST /api/v1/instance-assignments`
- `GET /api/v1/instance-assignments?tenantId=&workspaceId=&callerInstanceId=&capabilityId=`

### Data Plane

- `POST /api/v1/mcp/agents/{targetId}`
- `POST /api/v1/mcp/agents/{targetId}/rpc`
- `POST /api/v1/openapi/agents/{targetId}/operations/{operationId}`
- `ANY /api/v1/openapi/agents/{targetId}/{relativePath...}`

### Audit, Traces, and Metrics

- `GET /api/v1/audit/events?tenantId=&workspaceId=&action=&resourceType=&resourceId=&limit=`
- `GET /api/v1/audit/traces?tenantId=&workspaceId=&runId=&decision=&callerAgentId=&targetAgentId=`
- `GET /api/v1/metrics/runtime?tenantId=&workspaceId=`

## Policy and Data Scope Semantics

Route policies match `routeType` and optional `routeKey`; for MCP, route keys include `initialize`, `tools/list`, and `tools/call`. Higher priority wins, `deny` wins ties, disabled policies are ignored, and direct access grants remain as a compatibility fallback when no route policy matches.

MCP capabilities must be approved before they can be granted. Tenant entitlements, workspace assignments, and caller instance assignments form the effective grant chain for capability-aware MCP calls.

`dataScopes` are hierarchical OR alternatives. A child assignment may fill an empty parent dimension, but it cannot change a fixed parent dimension such as `region` or `tenantFilter`. Runtime traces record the effective inherited scope list, and governed MCP `tools/call` forwards the same list in `X-AgentHarbor-Context`. Caller-supplied, static target, and credential-backed values for `X-AgentHarbor-Context` are reserved and not forwarded.

The tenant access profile endpoint is read-only. It returns configured grants, effective scope calculations, invalid historical scope evidence, and recent trace evidence for a registered tenant subtree. `traceLimit=0` disables recent traces.

## Project Docs

- [CONTRIBUTING.md](CONTRIBUTING.md): contribution workflow and verification expectations.
- [SECURITY.md](SECURITY.md): private vulnerability reporting and security handling.
- [.env.example](.env.example): local configuration template.
- [ROADMAP.md](ROADMAP.md): public product and contribution direction.
- [docs/engineering/](docs/engineering): release, review, dependency, and engineering workflow references.
- [CHANGELOG.md](CHANGELOG.md): public release notes and notable changes.

## License

AgentHarbor is released under the [Apache License 2.0](LICENSE).
