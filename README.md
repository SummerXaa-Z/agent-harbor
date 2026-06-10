# AgentHarbor

AgentHarbor is a tenant-first access governance and permission operations platform for AI agents, MCP tools, OpenAPI services, and governed data access.

AgentHarbor 是面向 AI Agent、MCP 工具、OpenAPI 服务和受治理数据访问的租户优先权限治理与权限运营平台。

It helps platform, security, and tenant operations teams answer one production question: which tenant, workspace, caller instance, and subject can access which tool or data scope, why, and with what approval evidence.

它帮助平台、安全和租户运营团队回答一个生产问题：哪个租户、工作区、调用方实例和主体可以访问哪些工具或数据范围，为什么可以访问，以及对应的审批和审计证据是什么。

## Positioning

AgentHarbor supports MCP gateway capabilities, but it is not positioned as another generic MCP gateway. MCP servers are one governed target type. The product identity is tenant-first access governance, AI-friendly permission operations, and evidence-backed runtime enforcement across MCP tools, APIs, agents, and data systems.

AgentHarbor 支持 MCP 网关能力，但不把自己定位成另一个通用 MCP Gateway。MCP Server 只是被治理的目标类型之一。AgentHarbor 的产品身份是租户优先的访问治理、面向 AI 的权限运营，以及覆盖 MCP 工具、API、Agent 和数据系统的证据化运行时控制。

## Key Messages

- **Clear tenant boundaries / 租户边界清楚**: three-level tenants, workspaces, and caller instances jointly decide which data an agent can access.
- **Controlled permission changes / 权限变更可控**: administrators use permission packages to request access, and risky capabilities require approval before they are applied.
- **Complete go-live evidence / 上线证据完整**: runtime allow/deny traces, tenant access profiles, and audit events support go-live readiness evidence.

## Project Status

AgentHarbor is in developer preview. It is ready for local evaluation, design feedback, and early contribution, but it is not yet recommended for production traffic.

AgentHarbor 当前处于开发者预览阶段，适合本地评估、设计反馈和早期贡献；暂不建议承载生产流量。

Open-source timing is intentionally secondary to production hardening. Before any release-readiness claim, the current standard is that the safety baseline, release checks, and primary Permission Changes journey all pass from a fresh local checkout.

开源节奏会服从生产可用性。任何发布就绪声明之前，都必须确保安全基线、发布检查和核心权限包用户旅程能在全新本地检出中通过。

## What It Provides

- **Tenant-first governance / 租户优先治理**: register a three-level tenant tree and scope management views by tenant subtree. 注册三级租户树，并按租户子树限定管理视图。
- **Agent and target registry / Agent 与目标注册**: manage caller agents, MCP targets, OpenAPI services, webhooks, credentials, and short-lived Agent Keys. 管理调用方 Agent、MCP 目标、OpenAPI 服务、Webhook、凭据和短期 Agent Key。
- **Route policy controls / 路由策略控制**: allow or deny MCP/OpenAPI routes with priority, wildcard matching, and bounded retry overrides. 通过优先级、通配匹配和有界重试覆盖来允许或拒绝 MCP/OpenAPI 路由。
- **Capability governance / 能力治理**: discover target tools, approve capabilities, and grant them through tenant, workspace, and caller-instance assignments. 发现目标工具，审批能力，并通过租户、工作区和调用方实例分配进行授权。
- **Data permission enforcement / 数据权限控制**: narrow `dataScopes` across capability, tenant entitlement, workspace assignment, and instance assignment boundaries. 在能力、租户授权、工作区分配和实例分配边界上逐层收敛 `dataScopes`。
- **AI-friendly permission operations / AI 友好的权限运营**: draft tenant-scoped permission package changes from administrator intent, preview allow/deny outcomes, run apply preflight, run approval-required packages, apply them through the grant chain, and review structured application health and impact. 从管理员意图生成租户范围的权限包草案，预览允许/拒绝结果，运行应用前预检，运行需审批权限包，通过授权链落地，并查看结构化落地状态与影响复核。
- **Approval and evidence / 审批与证据**: route approval queues to configured approvers, expire and consume approval requests, record applied package evidence, review active or missing created objects, and keep audit trails for every privileged permission change. 将审批队列路由给已配置审批人，对审批请求设置过期和一次性消费，记录权限包应用证据，复核已创建对象是否仍有效，并为高风险权限变更保留审计链。
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

The data plane uses short-lived Agent Keys. Management APIs require configured admin authentication by default: use `AGENT_HARBOR_ADMIN_KEY` for a shared local admin key or `AGENT_HARBOR_ADMIN_IDENTITIES` for named administrators and reviewers. `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true` is development-only and must not be used for production deployments.

## Quick Start

Use the repository toolchain pins before running local commands:

- Go version comes from `go.mod`.
- Node major version comes from `.node-version`.
- Frontend package manager comes from `frontend/package.json`.

```bash
make demo
```

Then open `http://127.0.0.1:5174/`. The demo command starts the API in explicit unauthenticated development mode, the official MCP TypeScript SDK demo service, and the web console together for the first browser evaluation.

The web console opens on **Permission Changes**, the primary approval and readiness workspace. Confirm the runtime checks are ready, then run the validation journey. The **Self-Check** workspace remains available for the lower-level `6/6` core permission loop validation.

Use the local release gate when you want to verify the repository rather than run the browser demo:

```bash
make check
```

`make check` installs the pinned frontend dependencies from `frontend/pnpm-lock.yaml` before running frontend tests and builds.

Use the production safety gate when you want to verify conservative runtime defaults:

```bash
make production-hardening
```

This starts a local API with `AGENT_HARBOR_ADMIN_KEY` configured and private upstreams disabled. It verifies health remains public, management APIs reject missing or wrong admin keys, permission-package and management MCP endpoints use the same admin-key protection, loopback MCP targets are rejected by default, and public HTTPS MCP targets remain registrable. The default server also rejects management routes when no admin key, named identity, or explicit development unauthenticated flag is configured.

The API listens on `:9090` by default. Override it with:

```bash
AGENT_HARBOR_ADDR=:9091 go run ./cmd/agent-harbor
```

## Try the Core Journey in 10 Minutes

This local scenario runs the most important AgentHarbor workflow: create a three-level tenant tree, register an MCP target, discover tools, approve one tool, assign it to a tenant/workspace/caller instance, run allowed and denied calls, and verify access-profile plus audit evidence.

Terminal 1:

```bash
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
```

Terminal 2:

```bash
make core-journey
```

The scenario starts the dependency-free mock MCP server automatically and points AgentHarbor at `http://127.0.0.1:8787/mcp`. The `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` flag is required only for local development scenarios that use loopback or private-network upstreams; do not enable it for production deployments.

## Try the Permission Changes Console

The Permission Changes console proves the v0.2.0 approval-required permission change path for the primary product journey: create a three-level tenant tree, register a caller and MCP target, discover read/write/export tools, start a request from the **Support ticket triage** permission package template, create, withdraw, recreate, and approve matching approval requests, run apply preflight, apply it with `approvalRequestId`, run allowed and denied MCP calls with a subject id, and verify access-profile, application, approval, trace, audit, read-only health, read-only impact, go-live readiness evidence, and the production evidence report. The first screen is now a task-first workspace for completing one permission change: current status and next action at the top, request information on the left, approval/apply/validation handling on the right, and advanced checks collapsed below. `POST /api/v1/permission-packages/workbench:preview` provides the read-only projection behind this screen so the UI does not stitch together draft, approval, application, and readiness objects as peer cards. The console is intentionally business-first: the default view asks who needs access, which permission package template should be used, whether approval is required, and what the next action is. The current step is exposed with `aria-current="step"` and localized process wording. The main journey shows readable tenant hierarchy, workspace, caller, target service, and access object labels; raw tenant, workspace, caller, and target ids plus technical subject selectors stay in folded Advanced settings or **Advanced Checks** for audit and troubleshooting. Permission drafts and instance assignments require an explicit access object selector and reject empty or bare `*` selectors. Apply preflight is read-only: it checks draft readiness, approval match, capability configuration fingerprints, data-scope fit, planned grant objects, and existing grant chains without creating entitlements, applications, audit events, or consuming approval. The console also includes an Approval queue that filters pending approvals by approver identity and the current request scope. Approval requests expire after 24 hours, reject self-approval, can be withdrawn by the original requester while still pending, snapshot per-capability permission configuration fingerprints, and are consumed transactionally by the first successful permission application. The application health panel derives ready, drifted, and needs-review status from the same impact calculation; it is not a scheduler, notification system, or rollback executor. The impact review panel resolves the created entitlement, workspace assignment, instance assignment, and capability ids so operators can inspect rollback readiness before any future rollback mutation exists. The Go-Live Readiness Status panel is the final read-only go/no-go gate: it combines preflight, latest application, health, impact, access-profile, allowed/denied runtime traces, and applied audit evidence before reporting ready, needs review, or blocked. Readiness and evidence responses include a stable `nextActionCode` for UI localization and admin-agent planning, while keeping the human-readable `nextActions` list for compatibility. The same panel can export a bounded local JSON evidence report for security handoff. The adjacent **Rehearse drift** action calls `rehearsal=grant_drift` to simulate missing/inactive grant blockers in the response only; it does not mutate permission state.

权限变更与上线就绪控制台验证 v0.2.0 需要审批的权限变更主旅程：创建三级租户树，注册调用方和 MCP 目标，发现读/写/导出工具，基于 **客服工单处理包** 权限包模板发起权限变更，创建、撤回、重新创建并批准匹配的审批请求，运行应用前预检，携带 `approvalRequestId` 应用权限，用主体 ID 跑通允许和拒绝调用，并验证访问画像、应用记录、审批、追踪、审计、只读落地状态、只读影响复核、上线就绪证据和上线证据报告。首屏现在不再展示产品主张卡片或状态看板，而是直接进入完成一单权限变更的任务工作台：顶部给出当前状态和下一步，左侧填写申请信息，右侧处理审批、应用和运行验证，高级检查默认折叠。`POST /api/v1/permission-packages/workbench:preview` 为这个界面提供只读工作台投影，避免前端把草案、审批、应用和上线就绪对象拼成同级卡片。控制台默认采用业务优先表达：先问谁需要访问、用哪个权限包模板、是否需要审批、下一步做什么；主旅程展示可读的租户层级、工作区、调用方、目标服务和访问对象，原始租户、工作区、调用方和目标 ID 以及主体选择器只收进高级设置或 **高级检查项**，用于审计和排障。应用前预检是只读安全门：它检查草案就绪、审批匹配、数据范围边界、计划授权对象和已有授权链，不会创建授权、应用记录、审计事件，也不会消费审批。审批请求 24 小时后过期，拒绝自审批，待审批状态下可由原申请人撤回。落地状态面板复用同一套影响计算派生正常、漂移、需复核状态；它不是调度器、通知系统或回滚执行器。上线就绪状态面板是最后的只读上线门禁：汇总预检、最新应用、落地状态、影响复核、访问画像、允许/拒绝运行追踪和应用审计证据后，给出可上线、需复核或阻断。上线就绪状态和证据报告会返回稳定 `nextActionCode`，便于 UI 本地化和管理 Agent 规划下一步，同时保留原有人类可读 `nextActions` 以兼容旧客户端。同一面板可以导出有边界的本地 JSON 证据报告，用于安全交接。相邻的 **演练漂移** 动作会调用 `rehearsal=grant_drift`，仅在响应中模拟缺失/未启用授权阻断，不会修改权限状态。

```bash
make demo
```

Then open `http://127.0.0.1:5174/`, switch to **Permission Changes**, click **Run validation**, confirm **Go-Live Status** reaches ready, confirm **Go-Live Readiness** says ready for go-live, confirm **Application Health** shows a healthy row, confirm **Readiness Status** reports ready, export the production evidence JSON, click that row's **Review impact** action, and optionally click **Rehearse drift** to preview the read-only blocker state.

然后打开 `http://127.0.0.1:5174/`，切换到 **权限变更**，点击 **Run validation / 执行运行验证**，确认 **Go-Live Status / 上线状态** 达到可上线，确认 **Go-Live Readiness / 上线就绪** 显示已满足上线条件，确认 **Application Health / 落地状态** 出现正常应用行，确认 **Readiness Status / 上线就绪状态** 显示可上线，并导出上线证据 JSON，再点击该行的 **Review impact / 查看影响**，也可以点击 **Rehearse drift / 演练漂移** 预览只读阻断状态。

Use the CLI scenario when you want the same path as a scriptable regression check:

Terminal 1:

```bash
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
```

Terminal 2:

```bash
make scenario-permission-package-approval
```

The default scenario starts `scripts/mock-mcp-server.py` automatically for a dependency-free regression path. To run the same journey against the official MCP TypeScript SDK demo service, start the API with private upstreams and explicit unauthenticated development mode enabled, then run `MCP_SERVER_MODE=real make scenario-permission-package-approval`; the SDK service exposes the same `search_customer`, `update_ticket`, and `export_contracts` tools over Streamable HTTP. The scenario verifies that missing approval is blocked by preflight, approved preflight passes without consuming approval, production readiness and the evidence report block before apply, and the approved request is marked consumed only after apply and cannot be reused. After allowed and denied runtime calls plus applied audit evidence are present, production readiness and the evidence report must report ready.

默认脚本会自动启动 `scripts/mock-mcp-server.py`，用于无额外依赖的回归验证。如果要用官方 MCP TypeScript SDK 演示服务跑同一条旅程，先用私有上游模式启动 API，再执行 `MCP_SERVER_MODE=real make scenario-permission-package-approval`；该 SDK 服务通过 Streamable HTTP 暴露同一组 `search_customer`、`update_ticket` 和 `export_contracts` 工具。脚本会验证缺少审批会被预检阻断、已批准预检不会消费审批、应用前上线就绪状态和证据报告仍然阻断，以及已批准请求只会在应用后被消费且不能复用。当允许/拒绝运行调用和应用审计证据都存在后，上线就绪状态和证据报告都必须显示可上线。

For release-candidate validation of the browser-facing path, run:

```bash
make ai-admin-browser-journey
```

This starts the API, the official SDK MCP demo service, and web console, verifies browser CORS allows `X-AgentHarbor-Subject-Id`, then runs the approval-required package scenario against those services.

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

Then open `http://127.0.0.1:5174/`. The web console opens on **Permission Changes**, the production approval and readiness workspace for the approval-required **Support ticket triage** path. Each validation run uses fresh `ui-approval-*` identifiers, applies permissions through live APIs, sends runtime MCP calls with `X-AgentHarbor-Subject-Id`, and surfaces the application record, application impact review, tenant access profile, traces, applied audit event, go-live readiness, and bounded evidence export in the console.

The Permission Changes console also shows runtime checks for the API, MCP tool service, browser subject-header CORS, local private-upstream mode, and current data source before validation runs. Use the **Self-Check** workspace when you need the lower-level core permission loop check; it verifies API and MCP tool service readiness before enabling the run button and keeps **Reset session** non-destructive.

`make demo` starts:

- AgentHarbor API at `http://127.0.0.1:9090`
- Official SDK MCP demo service at `http://127.0.0.1:8787/mcp`
- Web console at `http://127.0.0.1:5174`

Use `Ctrl+C` in the demo terminal to stop all demo services.

If you need to troubleshoot a single service, use the manual three-terminal path:

Terminal 1:

```bash
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
```

Terminal 2:

```bash
make real-mcp
```

Terminal 3:

```bash
cd frontend
pnpm install
pnpm dev
```

The console reads `VITE_API_BASE`; if unset, it uses `http://127.0.0.1:9090`. When the backend is unavailable during local development, the UI falls back to sample data so the console remains navigable, shows a persistent warning, and disables mutation actions that would otherwise imply a durable permission change.

The **Self-Check** workspace creates a fresh tenant tree, caller, MCP target, scoped capability grant chain, allowed call, denied call, and tenant access profile evidence through the real API. This lower-level core permission loop supports English and Simplified Chinese. The browser language is used on first load, and the visible `中文` / `EN` toggle persists the operator's choice locally.

The console also includes a **Permission Changes** workspace for the v0.2.0 permission-package journey. It lets an administrator create a permission change, select a deterministic permission package template, preview allow/deny simulation rows, run apply preflight, apply low-risk requests through the backend permission-package API, request approval for high-risk requests, and then inspect the refreshed tenant access profile. Each draft includes a policy gate: direct apply is allowed for low-risk read-oriented templates, while write, export, admin, high-risk, critical-risk, confidential, or restricted allowed capabilities require an approval request before apply. The Apply Preflight panel calls `POST /api/v1/permission-packages:preflight` and blocks live apply when draft readiness, access-object binding, approval match, capability fingerprint match, data-scope fit, or other safety checks fail. Approved requests snapshot the draft, template version, scope, allowed capabilities, data scopes, per-capability configuration fingerprints, and policy-gate reasons so apply rejects drift before writing permissions. Approval requests expire after 24 hours, reject self-approval, and are consumed in the same repository transaction or lock as the successful application, so the same approval cannot apply the request twice even under concurrent reuse. When `AGENT_HARBOR_APPROVAL_REVIEWERS` is configured, the Approval queue and approve/reject operations are scoped to the approver's configured tenant subtree and workspace; when `AGENT_HARBOR_ADMIN_IDENTITIES` is configured, the reviewer is bound to the authenticated admin key. The Permission Changes and tenant access-profile workspaces include a read-only effective permission explanation panel so operators can inspect why a tenant/workspace/caller/capability decision is currently allowed or denied, which evidence layer matched or blocked, and which remediation step comes next. Each successful permission application records the template version, draft id, created entitlement and assignment ids, capability ids, and data scopes; the applied audit event links the approval request id when one was used. The read-only application health endpoint and Permission Changes panel summarize recent applications as ready, drifted, or needs review by reusing the impact calculation and stable blocker codes. The read-only production readiness endpoint and Permission Changes panel combine preflight, latest application, health, impact, access-profile, runtime allowed/denied traces, and applied audit evidence into one final ready, needs-review, or blocked gate. The read-only impact review endpoint and Permission Changes panel resolve those recorded ids against current state, show created/active/missing object counts, mark capability rollback as manual review, and return a read-only remediation plan with ordered manual-review, disable, drift-investigation, and final verification actions without executing rollback. Drift blockers also include stable `blockerCodes` such as `missing_created_objects`, `inactive_created_objects`, and `no_allowed_capabilities` so UI and admin agents can localize and reason about unsafe states reliably. The optional `rehearsal=grant_drift` impact query and **Rehearse drift** button simulate those blockers without changing any grants, which lets operators practice remediation review safely. See [the v0.2.0 journey note](docs/product/0.2.0-ai-admin-permission-journey.md).

权限变更工作区承载 v0.2.0 权限包旅程。管理员发起的是权限变更，选择的是权限包模板；随后可以预览允许/拒绝模拟行、运行应用前预检、通过后端权限包 API 应用低风险变更、为高风险变更提交审批，并查看刷新的租户访问画像。应用前预检面板调用 `POST /api/v1/permission-packages:preflight`，当草案就绪、访问对象绑定、审批匹配、能力配置指纹、数据范围边界或其他安全检查失败时阻断实时应用。审批请求会拒绝自审批，并通过每能力配置指纹阻断审批后能力变宽；配置 `AGENT_HARBOR_ADMIN_IDENTITIES` 后，审批人来自已认证管理 key，而不是请求体自报。能力治理里的直接授权也优先选择角色、部门或成员访问对象，只有高级场景才暴露主体选择器表达式。只读落地状态端点和权限变更面板会复用影响计算，把最近的权限应用归类为正常、已漂移或需复核；它不执行调度、通知或回滚。只读上线就绪端点和权限变更面板会把预检、最新应用、落地状态、影响复核、访问画像、允许/拒绝运行追踪和应用审计证据合并成最后的可上线、需复核或阻断判断。只读影响复核端点和权限变更面板会把记录中的授权对象与当前状态对齐，展示已创建/有效/缺失对象数量，返回只读处置计划，并用稳定的 `blockerCodes` 解释缺失、未启用或无允许能力等不安全状态。可选的 `rehearsal=grant_drift` 查询和 **演练漂移** 按钮会在不修改任何授权的前提下模拟这些阻断，让操作员安全练习处置复核。

AgentHarbor also exposes the same workflow as a management MCP endpoint at `POST /api/v1/management/mcp`. Admin agents can call `tools/list` and then use tools such as `draft_permission_package`, `preflight_permission_package`, `check_permission_package_production_readiness`, `export_permission_package_production_evidence`, `create_permission_package_approval_request`, `list_permission_package_approval_requests`, `approve_permission_package_approval_request`, `reject_permission_package_approval_request`, `withdraw_permission_package_approval_request`, `apply_permission_package`, `list_permission_package_applications`, `explain_permission_package_draft`, `explain_access_decision`, `get_tenant_access_profile`, `list_agents`, and `list_capabilities`. REST also exposes `POST /api/v1/permission-packages:preflight` for read-only apply preflight, `GET /api/v1/permission-packages/production-readiness?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&subjectId=` for the read-only production gate, `GET /api/v1/permission-packages/production-readiness/report?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&subjectId=` for the bounded production evidence report, `GET /api/v1/permission-packages/applications/health?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&limit=` for read-only application health, and `GET /api/v1/permission-packages/applications/{id}/impact?tenantId=&workspaceId=` for read-only application impact review and remediation planning, plus `rehearsal=grant_drift` for response-only drift rehearsal. `list_permission_package_approval_requests` accepts an optional `reviewer` field so an admin agent can fetch only the requests that reviewer is allowed to handle; approve and reject validate the same reviewer route before status changes, while withdraw is limited to the original pending requester. This endpoint requires `X-Admin-Key` unless the API is explicitly started with `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true` for local development.

AgentHarbor 也通过 `POST /api/v1/management/mcp` 暴露同一套管理工作流。管理 Agent 可以使用 `preflight_permission_package` 在应用前执行只读预检，使用 `check_permission_package_production_readiness` 获取上线就绪状态，也可以使用 `export_permission_package_production_evidence` 生成有边界的上线证据报告。REST 端点 `POST /api/v1/permission-packages:preflight` 用于只读应用前预检；`GET /api/v1/permission-packages/production-readiness?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&subjectId=` 用于只读上线就绪门禁；`GET /api/v1/permission-packages/production-readiness/report?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&subjectId=` 用于有边界的上线证据报告；`GET /api/v1/permission-packages/applications/health?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&limit=` 用于只读落地状态巡检；`GET /api/v1/permission-packages/applications/{id}/impact?tenantId=&workspaceId=` 用于只读影响复核和处置规划，也支持 `rehearsal=grant_drift` 做仅影响响应的漂移演练。除非本地开发显式设置 `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true`，否则这些端点与其他管理 API 一样需要 `X-Admin-Key`。

## Runtime Configuration

Use `.env.example` as the local configuration template.

| Variable | Purpose |
| --- | --- |
| `AGENT_HARBOR_ADDR` | API listen address. Defaults to `:9090`. |
| `AGENT_HARBOR_ADMIN_KEY` | Optional shared management API key. Management and audit endpoints require this key, a named admin identity, or the explicit development unauthenticated flag. |
| `AGENT_HARBOR_ADMIN_IDENTITIES` | Optional named admin identities for production approvals. Use comma or semicolon separated `actor=key` entries, for example `requester=req-key;security=sec-key`. Matching keys set `requestedBy` and `reviewedBy` to the actor and prevent reviewer impersonation. 可选具名管理身份，使用 `actor=key`；匹配后审批发起人与审批人来自认证身份，而不是请求体自报。 |
| `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN` | Development-only boolean. Allows management endpoints without `X-Admin-Key` only when no admin key or named identities are configured. Defaults to `false`. |
| `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` | Development-only boolean. Allows loopback/private upstream endpoints for local scenarios when set to `true`. Defaults to `false`. |
| `AGENT_HARBOR_APPROVAL_REVIEWERS` | Optional approval reviewer routing rules. Use comma or semicolon separated `reviewer=tenantId/workspaceId` entries, for example `security-root=tenant-root/*;security-east=tenant-east/ws-support`. `*` allows any tenant or workspace. Pair this with `AGENT_HARBOR_ADMIN_IDENTITIES` in production so route checks use authenticated reviewer actors. 可选审批人路由规则，生产环境建议与 `AGENT_HARBOR_ADMIN_IDENTITIES` 配合使用，让路由校验基于已认证审批人身份。 |
| `AGENT_HARBOR_CORS_ORIGINS` | Optional comma or semicolon separated browser origins to allow in addition to the default local console origins. Use it for isolated browser gates or non-default local frontend ports. 可选浏览器来源白名单，使用逗号或分号分隔；用于隔离浏览器门禁或非默认本地前端端口。 |
| `AGENT_HARBOR_DATABASE_URL` | Optional PostgreSQL connection string. If unset, AgentHarbor uses the in-memory repository. |
| `AGENT_HARBOR_CREDENTIAL_KEY` | 32-byte raw or base64 key used to encrypt persisted agent credentials. Required with PostgreSQL. |
| `AGENT_HARBOR_TEST_DATABASE_URL` | PostgreSQL connection string used by integration tests. |
| `VITE_API_BASE` | Frontend API base URL. |

Run `make production-hardening` before any deployment-style handoff. It proves that `AGENT_HARBOR_ADMIN_KEY` protection is enforced across management APIs, that management routes fail closed without configured admin authentication, and that `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` stays disabled unless explicitly set for local development scenarios.

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
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
make core-journey
```

The approval-required permission package journey can run against the official SDK MCP demo service:

```bash
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
MCP_SERVER_MODE=real make scenario-permission-package-approval
```

This scenario verifies approval expiry metadata and one-time approval consumption in addition to the runtime allow/deny checks.

With a shared admin key instead of the local unauthenticated development flag:

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
- `GET /api/v1/permission-packages/access-subjects`
- `POST /api/v1/permission-packages/drafts`
- `POST /api/v1/permission-packages/approval-requests`
- `GET /api/v1/permission-packages/approval-requests?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&status=&reviewer=&limit=`
- `POST /api/v1/permission-packages/approval-requests/{id}/approve`
- `POST /api/v1/permission-packages/approval-requests/{id}/reject`
- `POST /api/v1/permission-packages:preflight`
- `POST /api/v1/permission-packages:apply`
- `GET /api/v1/permission-packages/applications?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&limit=`
- `GET /api/v1/permission-packages/applications/health?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&limit=`
- `GET /api/v1/permission-packages/production-readiness?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&subjectId=&traceLimit=`
- `GET /api/v1/permission-packages/production-readiness/report?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&subjectId=&traceLimit=`
- `GET /api/v1/permission-packages/applications/{id}/impact?tenantId=&workspaceId=&rehearsal=`
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
