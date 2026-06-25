# AgentHarbor

AgentHarbor is a tenant-first access governance and permission operations platform for AI agents, MCP tools, OpenAPI services, and governed data access.

AgentHarbor 是面向 AI Agent、MCP 工具、OpenAPI 服务和受治理数据访问的租户优先权限治理与权限运营平台。

It helps platform, security, and tenant operations teams answer one production question: which tenant, workspace, caller instance, and subject can access which tool or data scope, why, and with what approval record.

它帮助平台、安全和租户运营团队回答一个生产问题：哪个租户、工作区、调用方实例和主体可以访问哪些工具或数据范围，为什么可以访问，以及对应的审批和审计记录是什么。

## Positioning

AgentHarbor supports MCP gateway capabilities, but it is not positioned as another generic MCP gateway. MCP servers are one governed target type. The product identity is tenant-first access governance, AI-friendly permission operations, and audit-ready runtime enforcement across MCP tools, APIs, agents, and data systems.

AgentHarbor 支持 MCP 网关能力，但不把自己定位成另一个通用 MCP Gateway。MCP Server 只是被治理的目标类型之一。AgentHarbor 的产品身份是租户优先的访问治理、面向 AI 的权限运营，以及覆盖 MCP 工具、API、Agent 和数据系统的可审计运行时控制。

## Key Messages

- **Clear tenant boundaries / 租户边界清楚**: three-level tenants, workspaces, and caller instances jointly decide which data an agent can access.
- **Controlled permission changes / 权限变更可控**: administrators use permission packages to request access, and risky capabilities require approval before they are applied.
- **Clear go-live status / 上线状态清楚**: runtime allow/deny traces, tenant access profiles, and audit events support go-live decisions.

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
- **Approval and audit trail / 审批与审计**: route approval queues to configured approvers, expire and consume approval requests, record applied package evidence, review active or missing created objects, and keep audit trails for every privileged permission change. 将审批队列路由给已配置审批人，对审批请求设置过期和一次性消费，记录权限包应用记录，复核已创建对象是否仍有效，并为高风险权限变更保留审计链。
- **Runtime records / 运行记录**: record traces, audit events, metrics, upstream attempts, effective data scopes, and deny reasons. 记录 trace、审计事件、指标、上游尝试、有效数据范围和拒绝原因。
- **Access Profile / 权限画像**: inspect each tenant's effective access profile, grant chain, invalid scope rows, and recent trace records. 查看每个租户的有效访问画像、授权链、无效范围行和近期运行记录。

## Core Model

```text
Tenant
  -> Agent or target service
  -> MCP/OpenAPI capability
  -> Tenant entitlement
  -> Workspace assignment
  -> Caller instance assignment
  -> Runtime decision and trace records
```

The tenant is the primary control boundary. A registered tenant can manage its own subtree; unregistered tenant strings keep exact-match behavior for compatibility.

The data plane uses short-lived Agent Keys. Management APIs require configured admin authentication by default: use `AGENT_HARBOR_ADMIN_KEY` for a shared local admin key or `AGENT_HARBOR_ADMIN_IDENTITIES` for named administrators and reviewers. Named identities can also carry role, tenant, and workspace boundaries, for example `AGENT_HARBOR_ADMIN_IDENTITIES="platform=platform-admin-key-32|role=platform_admin;east=east-admin-key-32|role=tenant_admin|tenant=tenant-east|workspace=ws-support"`. Actor names must be stable machine identifiers using 1-80 letters, numbers, dots, underscores, hyphens, or at signs. Scoped tenant admins can only read or mutate their allowed tenant subtree and workspace across REST management APIs, permission-package operations, and the management MCP endpoint. Production deployment mode rejects short or common weak bootstrap admin keys, rejects a shared admin key that matches any named bootstrap identity key, and reserves the `admin-key` and `local-dev` actor names for built-in bootstrap/development identities, so keep shared and named bootstrap keys long, distinct, and stored outside the product. The web console signs in through `/api/v1/auth/login` and exchanges the admin key for an HttpOnly `agent_harbor_session` cookie; direct `X-Admin-Key` remains available for API clients and advanced local overrides. Console session cookies set `Secure` for direct TLS requests and for trusted loopback/private proxy requests that carry `X-Forwarded-Proto: https` or `Forwarded: proto=https`; forwarded scheme headers from public clients are ignored. Set `AGENT_HARBOR_SESSION_SECRET` for deployment-style environments so console sessions are signed with a stable, high-entropy secret; production mode blocks missing, short, or common weak session secrets. `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true` is development-only and must not be used for production deployments.

数据面使用短期 Agent Key。管理 API 默认要求配置管理员认证：可以使用 `AGENT_HARBOR_ADMIN_KEY` 作为共享本地管理员密钥，也可以使用 `AGENT_HARBOR_ADMIN_IDENTITIES` 配置具名管理员和审批人。具名身份也可以绑定角色、租户和工作区边界，例如 `AGENT_HARBOR_ADMIN_IDENTITIES="platform=platform-admin-key-32|role=platform_admin;east=east-admin-key-32|role=tenant_admin|tenant=tenant-east|workspace=ws-support"`。actor 名必须是稳定机器标识，只能使用 1-80 位字母、数字、点、下划线、连字符或 at 符号。租户管理员只能在被授权的租户子树和工作区内读取或变更 REST 管理 API、权限包操作和管理 MCP 端点。生产部署模式会拒绝过短或常见弱值的引导管理员密钥，也会拒绝共享管理员密钥与任一具名引导身份密钥相同的配置，并把 `admin-key` 和 `local-dev` 作为内置引导/开发身份的保留 actor 名，因此共享密钥和具名引导密钥都应保持足够长度、彼此不同，并存放在产品外。Web 控制台通过 `/api/v1/auth/login` 登录，并把管理员密钥换成 HttpOnly `agent_harbor_session` Cookie；直接 `X-Admin-Key` 仍保留给 API 客户端和本地高级覆盖。控制台会话 Cookie 会在真实 TLS 请求，或可信本机/私网代理携带 `X-Forwarded-Proto: https` 或 `Forwarded: proto=https` 时设置 `Secure`；来自公网客户端的转发协议头会被忽略。部署式环境必须设置 `AGENT_HARBOR_SESSION_SECRET`，确保控制台会话使用稳定高熵密钥签名；生产模式会阻断缺失、过短或常见弱值会话密钥。`AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true` 仅用于开发，不得用于生产部署。

The **Tenant Permission Center** turns each registered tenant into a governance workspace: platform administrators can review assigned administrators, workspaces, permission packages, allowed and blocked capabilities, data scopes, and safe next actions from one tenant detail page. Tenant administrators see the same page bounded to their assigned tenant/workspace and cannot manage administrator identities.

**租户权限中心** 会把每个已注册租户变成一个治理工作区：平台管理员可以在租户详情页查看负责管理员、工作区、权限包、允许和禁止能力、数据范围以及下一步动作。租户管理员只能看到自己被分配的租户/工作区范围，不能管理管理员身份。

### Administrator Identity Management

Bootstrap administrator identities from `AGENT_HARBOR_ADMIN_KEY` and `AGENT_HARBOR_ADMIN_IDENTITIES` are read-only in the product and should be kept as production break-glass access. Day-to-day administrators should be managed in the web console under **Administrators & Boundaries**: platform administrators can create managed identities, scope them to a role, tenant, and workspace, rotate their keys, disable them, and review lifecycle audit records. Managed administrator actors cannot reuse bootstrap administrator actors, keeping sessions, audit rows, and approval routing unambiguous. Managed administrator keys are shown once on create or rotate and are never returned by list or audit APIs. Rotating a managed administrator key invalidates browser sessions issued before the rotation, and disabling a managed administrator invalidates that administrator's existing browser sessions, so either action can be used as an immediate containment step. Tenant administrators cannot manage administrator identities; they can only operate inside their assigned tenant/workspace boundary. For production recovery, keep at least one bootstrap platform administrator configured outside the product.

### 管理员身份管理

通过 `AGENT_HARBOR_ADMIN_KEY` 和 `AGENT_HARBOR_ADMIN_IDENTITIES` 配置的引导管理员在产品内是只读身份，应保留为生产 break-glass 入口。日常管理员建议在 Web 控制台的 **管理员与边界** 中管理：平台管理员可以创建托管身份，按角色、租户和工作区设定范围，轮换密钥，禁用身份，并查看生命周期审计记录。托管管理员 actor 不能复用引导管理员 actor，避免会话主体、审计记录和审批路由含混。托管管理员密钥只会在创建或轮换时展示一次，列表和审计 API 不会再次返回明文；轮换托管管理员密钥会让轮换前签发的浏览器会话失效，禁用托管管理员也会让该管理员已有浏览器会话失效，因此两者都可作为即时止损动作。租户管理员不能管理管理员身份，只能在被分配的租户/工作区边界内操作。生产恢复路径上，应至少保留一个产品外配置的引导平台管理员。

## Quick Start

Use the repository toolchain pins before running local commands:

- Go version comes from `go.mod`.
- Node major version comes from `.node-version`.
- Frontend package manager comes from `frontend/package.json`.

```bash
make demo
```

Then open `http://127.0.0.1:5174/`. The demo command starts the API in explicit unauthenticated development mode, the official MCP TypeScript SDK demo service, and the web console together for the first browser evaluation.

The web console opens on **Getting Started** when the live system is not configured yet, then defaults back to **Permission Changes** once tenant, Agent, capability, and grant-chain setup is complete. Confirm the runtime checks are ready, then run the validation journey. The **Self-Check** workspace remains available for the lower-level `6/6` core permission loop validation.

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

This local scenario runs the most important AgentHarbor workflow: create a three-level tenant tree, register an MCP target, discover tools, approve one tool, assign it to a tenant/workspace/caller instance, run allowed and denied calls, and verify access-profile plus audit records.

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

First-time users start on **Getting Started**, a six-step setup checklist that explains the chain from tenant to Agent, capability, grant, runtime records, and go-live status. Once the first four setup steps are complete, the console opens directly on **Permission Changes** for daily operations.

首次打开控制台时，如果系统尚未完成配置，会先进入 **开始使用**：一个六步检查清单，说明从租户、Agent、能力、授权、运行记录到上线检查的链路。前四步完成后，控制台会默认进入日常操作的 **权限变更**。

### What this validates

The console exercises the v0.2.0 permission-change journey end to end:

1. Create a three-level tenant tree, a caller, and an MCP target.
2. Discover read, write, and export tools from the target.
3. Start from the **Support ticket triage** permission package template.
4. Create, withdraw, recreate, and approve scoped approval requests.
5. Run read-only apply preflight, apply with `approvalRequestId`, and verify allowed and denied MCP calls.
6. Review access profile, application health, impact, trace, audit, status-check records, and the production acceptance report.

The UI is intentionally task-first. It asks who needs access, which permission package template should apply, whether approval is required, and what the next safe action is. Technical IDs and subject selectors stay in **Technical overrides**; go-live proof stays in **Acceptance Details**. The status-check and status APIs return stable `nextActionCode` values so the UI and admin agents can localize next actions without parsing English text.

### 这会验证什么

权限变更控制台用于验证 v0.2.0 的审批型权限变更主旅程:

1. 创建三级租户树、调用方和 MCP 目标。
2. 从目标服务发现读、写、导出工具。
3. 基于 **客服工单处理包** 权限包模板发起变更。
4. 创建、撤回、重新创建并批准匹配的审批请求。
5. 执行只读应用前预检，携带 `approvalRequestId` 应用权限，并验证允许/拒绝 MCP 调用。
6. 复核访问画像、落地状态、影响范围、追踪、审计、状态检查记录和上线状态报告。

界面默认服务一个任务: 谁需要访问、使用哪个权限包模板、是否需要审批、下一步做什么。技术 ID 和主体选择器收进 **技术覆盖**，上线检查收进 **验收明细**。状态检查和状态报告返回稳定的 `nextActionCode`，便于 UI 和管理 Agent 本地化下一步动作。

### Run it locally

```bash
make demo
```

1. Start the local demo stack with `make demo`.
2. Open `http://127.0.0.1:5174/`. A fresh system lands on **Getting Started**; a configured system lands on **Access Query** so operators can ask why a caller is allowed or denied before starting a change.
3. From **Access Query**, use **Start permission fix** to carry a denied decision into **Permission Changes**, or open **Permission Changes** directly to run validation.
4. Confirm **Status Check** reaches ready and **Application Health** shows a healthy row.
5. Export the production acceptance JSON.
6. Open **Review impact** or **Rehearse drift** when you want to inspect read-only impact or drift blockers.

### 本地运行

1. 启动本地演示环境: `make demo`。
2. 打开 `http://127.0.0.1:5174/`。全新系统会进入 **开始使用**，已配置系统会进入 **访问查询**，先回答调用方为什么能或不能访问，再决定是否发起变更。
3. 在 **访问查询** 中用 **发起权限修复** 把拒绝判定带入 **权限变更**，也可以直接打开 **权限变更** 执行运行验证。
4. 确认 **Status Check / 状态检查** 达到可上线，并确认 **Application Health / 落地状态** 出现正常应用行。
5. 导出上线状态 JSON。
6. 需要复核影响或演练漂移时，再打开 **Review impact / 查看影响** 或 **Rehearse drift / 演练漂移**。

Use the CLI scenario when you want the same path as a scriptable regression check:

Terminal 1:

```bash
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
```

Terminal 2:

```bash
make scenario-permission-package-approval
```

The default scenario starts `scripts/mock-mcp-server.py` automatically for a dependency-free regression path. To run the same journey against the official MCP TypeScript SDK demo service, start the API with private upstreams and explicit unauthenticated development mode enabled, then run `MCP_SERVER_MODE=real make scenario-permission-package-approval`; the SDK service exposes the same `search_customer`, `update_ticket`, and `export_contracts` tools over Streamable HTTP. The scenario verifies that missing approval is blocked by preflight, approved preflight passes without consuming approval, production readiness and the acceptance report block before apply, and the approved request is marked consumed only after apply and cannot be reused. After allowed and denied runtime calls plus applied audit records are present, production readiness and the acceptance report must report ready.

默认脚本会自动启动 `scripts/mock-mcp-server.py`，用于无额外依赖的回归验证。如果要用官方 MCP TypeScript SDK 演示服务跑同一条旅程，先用私有上游模式启动 API，再执行 `MCP_SERVER_MODE=real make scenario-permission-package-approval`；该 SDK 服务通过 Streamable HTTP 暴露同一组 `search_customer`、`update_ticket` 和 `export_contracts` 工具。脚本会验证缺少审批会被预检阻断、已批准预检不会消费审批、应用前上线就绪状态和验收报告仍然阻断，以及已批准请求只会在应用后被消费且不能复用。当允许/拒绝运行调用和应用审计记录都存在后，上线就绪状态和验收报告都必须显示可上线。

For release-candidate validation of the browser-facing path, run:

```bash
make ai-admin-browser-journey
```

This starts the API with split requester and reviewer admin identities, the official SDK MCP demo service, and web console. It verifies browser CORS allows `X-AgentHarbor-Subject-Id`, verifies requester-key reviewer impersonation is rejected, then runs the approval-required package scenario against those services.

该门禁会用分离的申请人与审批人管理身份启动 API、官方 SDK MCP 演示服务和 Web 控制台；它会验证浏览器 CORS 允许 `X-AgentHarbor-Subject-Id`，验证申请人 key 冒充审批人会被拒绝，然后跑完整的需审批权限包场景。

For release-candidate validation of the served web console production journey, run:

```bash
make web-console-production-journey
```

This starts an isolated local API, the official SDK MCP demo service, and the web console, then verifies the production journey smoke signals, go-live check center contract, connection diagnostics contract, system-info auth metadata, and route-level console tests without adding browser automation dependencies.

如果要验证已启动 Web 控制台上的生产旅程路径，可以运行 `make web-console-production-journey`。它会启动隔离端口的本地 API、官方 SDK MCP 演示服务和 Web 控制台，并在不新增浏览器自动化依赖的前提下验证生产旅程 smoke 信号、上线检查中心合约、连接诊断合约、系统信息认证元数据和路由级控制台测试。

For release-candidate validation of production defaults, run:

```bash
make production-hardening
```

`make release-check` also includes the production safety baseline, scoped admin tenant boundary gate, and the web console production journey smoke gate.

`make release-check` 同时包含生产安全基线、范围化管理员租户边界门禁和 Web 控制台生产旅程 smoke gate。

## Web Console

For the first browser evaluation, run:

```bash
make demo
```

Then open `http://127.0.0.1:5174/`. If the live system is empty, the web console opens on **Getting Started** and shows the setup chain before any permission-change work. After tenant, Agent, capability, and grant-chain setup is complete, the same URL opens on **Access Query**: operators first ask whether a caller can access a target capability, review the decision chain, and then use **Start permission fix** to prefill **Permission Changes** without copying technical IDs. **Permission Changes** remains the production approval and readiness workspace for the approval-required **Support ticket triage** path. The **Go-Live Check** workspace turns connection diagnostics, the current permission change, runtime validation, and handoff status into one ready/blocked decision with explicit next actions. Each validation run uses fresh `ui-approval-*` identifiers, applies permissions through live APIs, sends runtime MCP calls with `X-AgentHarbor-Subject-Id`, and surfaces the application record, application impact review, tenant access profile, traces, applied audit event, go-live readiness, and bounded acceptance export in the console.

打开 `http://127.0.0.1:5174/` 后，如果实时系统为空，Web 控制台会进入 **开始使用** 并先展示配置链路；当租户、Agent、能力和授权链完成后，同一个地址会进入 **访问查询**。管理员先查询某个调用方能否访问目标能力，查看判定链路，再通过 **发起权限修复** 把上下文预填到 **权限变更**，不需要复制技术 ID。**权限变更** 仍然负责审批、应用和状态检查；**上线检查** 会把连接诊断、当前权限变更、运行验证和交接状态收束成一个可上线/已阻断判断，并给出明确下一步。

The global **Connection Settings** popover includes **Run diagnostics** for the production path: it checks the browser session, API compatibility contract, live data source, and MCP tool-service health in one compact panel. The Permission Changes console also shows runtime checks for the API, MCP tool service, browser subject-header CORS, local private-upstream mode, and current data source before validation runs. Use the **Self-Check** workspace when you need the lower-level core permission loop check; it verifies API and MCP tool service readiness before enabling the run button and keeps **Reset session** non-destructive.

全局 **连接设置** 弹层提供 **运行诊断**，用于一次性检查浏览器会话、API 兼容合约、实时数据源和 MCP 工具服务健康状态。权限变更控制台在运行验证前也会展示 API、MCP 工具服务、浏览器主体 Header CORS、本地私有上游模式和当前数据源检查。需要更底层的核心权限闭环检查时，可以使用 **系统自检** 工作区。

`make demo` starts:

- AgentHarbor API at `http://127.0.0.1:9090`
- Official SDK MCP demo service at `http://127.0.0.1:8787/mcp`
- Web console at `http://127.0.0.1:5174`

Use `Ctrl+C` in the demo terminal to stop all demo services.

If those ports are already in use, set only the demo ports; the script wires the frontend API base and local browser CORS automatically:

```bash
AGENT_HARBOR_DEMO_API_PORT=19094 \
AGENT_HARBOR_DEMO_FRONTEND_PORT=15184 \
MOCK_MCP_PORT=18794 \
  make demo
```

如果默认端口已被本机其他开发服务占用，只需要切换上面的三个端口；脚本会自动把前端连接地址和本地浏览器 CORS 配好。

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

The console reads `VITE_API_BASE`; if unset, it uses `http://127.0.0.1:9090`. When the backend is unavailable during local development, the UI falls back to a read-only sample preview so the console remains navigable, shows a persistent warning, and disables mutation actions that would otherwise imply a durable permission change.

The **Self-Check** workspace creates a fresh tenant tree, caller, MCP target, scoped capability grant chain, allowed call, denied call, and tenant access profile records through the real API. This lower-level core permission loop supports English and Simplified Chinese. The browser language is used on first load, and the visible `中文` / `EN` toggle persists the operator's choice locally.

The console also includes a **Permission Changes** workspace for the v0.2.0 permission-package journey. It lets an administrator create a permission change, select a deterministic permission package template, preview allow/deny simulation rows, run apply preflight, apply low-risk requests through the backend permission-package API, request approval for high-risk requests, and then inspect the refreshed tenant access profile. Each draft includes a policy gate: direct apply is allowed for low-risk read-oriented templates, while write, export, admin, high-risk, critical-risk, confidential, or restricted allowed capabilities require an approval request before apply. The Apply Preflight panel calls `POST /api/v1/permission-packages:preflight` and blocks live apply when draft readiness, access-object binding, approval match, capability fingerprint match, data-scope fit, or other safety checks fail. Approved requests snapshot the draft, template version, scope, allowed capabilities, data scopes, per-capability configuration fingerprints, and policy-gate reasons so apply rejects drift before writing permissions. Approval requests expire after 24 hours, reject self-approval, and are consumed in the same repository transaction or lock as the successful application, so the same approval cannot apply the request twice even under concurrent reuse. When `AGENT_HARBOR_APPROVAL_REVIEWERS` is configured, the Approval queue and approve/reject operations are scoped to the approver's configured tenant subtree and workspace; when `AGENT_HARBOR_ADMIN_IDENTITIES` is configured, the reviewer is bound to the authenticated admin key. Non-platform administrators who omit `reviewer` on approval-list reads are automatically scoped to their authenticated actor's routed queue, while platform administrators can omit `reviewer` for an all-queue operational view. The Permission Changes and tenant access-profile workspaces include a read-only effective permission explanation panel so operators can inspect why a tenant/workspace/caller/capability decision is currently allowed or denied, which decision layer matched or blocked, and which remediation step comes next. Each successful permission application records the template version, draft id, created entitlement and assignment ids, capability ids, and data scopes; the applied audit event links the approval request id when one was used. The read-only application health endpoint and Permission Changes panel summarize recent applications as ready, drifted, or needs review by reusing the impact calculation and stable blocker codes. The read-only production readiness endpoint and Permission Changes panel combine preflight, latest application, health, impact, access-profile, runtime allowed/denied traces, and applied audit records into one final ready, needs-review, or blocked gate. The read-only impact review endpoint and Permission Changes panel resolve those recorded ids against current state, show created/active/missing object counts, mark capability rollback as manual review, and return a read-only remediation plan with ordered manual-review, disable, drift-investigation, and final verification actions without executing rollback. Drift blockers also include stable `blockerCodes` such as `missing_created_objects`, `inactive_created_objects`, and `no_allowed_capabilities` so UI and admin agents can localize and reason about unsafe states reliably. The optional `rehearsal=grant_drift` impact query and **Rehearse drift** button simulate those blockers without changing any grants, which lets operators practice remediation review safely. See [the v0.2.0 journey note](docs/product/0.2.0-ai-admin-permission-journey.md).

权限变更工作区承载 v0.2.0 权限包旅程。管理员发起的是权限变更，选择的是权限包模板；随后可以预览允许/拒绝模拟行、运行应用前预检、通过后端权限包 API 应用低风险变更、为高风险变更提交审批，并查看刷新的租户访问画像。应用前预检面板调用 `POST /api/v1/permission-packages:preflight`，当草案就绪、访问对象绑定、审批匹配、能力配置指纹、数据范围边界或其他安全检查失败时阻断实时应用。审批请求会拒绝自审批，并通过每能力配置指纹阻断审批后能力变宽；配置 `AGENT_HARBOR_ADMIN_IDENTITIES` 后，审批人来自已认证管理 key，而不是请求体自报。配置审批人路由后，非平台管理员读取审批队列即使省略 `reviewer`，也会默认限定到当前认证 actor 的可审批队列；平台管理员省略 `reviewer` 时仍可查看全局队列。能力治理里的直接授权也优先选择角色、部门或成员访问对象，只有高级场景才暴露主体选择器表达式。只读落地状态端点和权限变更面板会复用影响计算，把最近的权限应用归类为正常、已漂移或需复核；它不执行调度、通知或回滚。只读上线就绪端点和权限变更面板会把预检、最新应用、落地状态、影响复核、访问画像、允许/拒绝运行追踪和应用审计记录合并成最后的可上线、需复核或阻断判断。只读影响复核端点和权限变更面板会把记录中的授权对象与当前状态对齐，展示已创建/有效/缺失对象数量，返回只读处置计划，并用稳定的 `blockerCodes` 解释缺失、未启用或无允许能力等不安全状态。可选的 `rehearsal=grant_drift` 查询和 **演练漂移** 按钮会在不修改任何授权的前提下模拟这些阻断，让操作员安全练习处置复核。

AgentHarbor also exposes the same workflow as a management MCP endpoint at `POST /api/v1/management/mcp`. Admin agents can call `tools/list` and then use tools such as `draft_permission_package`, `preflight_permission_package`, `check_permission_package_production_readiness`, `export_permission_package_production_evidence`, `create_permission_package_approval_request`, `list_permission_package_approval_requests`, `approve_permission_package_approval_request`, `reject_permission_package_approval_request`, `withdraw_permission_package_approval_request`, `apply_permission_package`, `list_permission_package_applications`, `explain_permission_package_draft`, `explain_access_decision`, `get_tenant_access_profile`, `list_agents`, and `list_capabilities`. REST also exposes `POST /api/v1/permission-packages:preflight` for read-only apply preflight, `GET /api/v1/permission-packages/production-readiness?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&subjectId=` for the read-only production gate, `GET /api/v1/permission-packages/production-readiness/report?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&subjectId=` for the bounded production acceptance report, `GET /api/v1/permission-packages/applications/health?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&limit=` for read-only application health, and `GET /api/v1/permission-packages/applications/{id}/impact?tenantId=&workspaceId=` for read-only application impact review and remediation planning, plus `rehearsal=grant_drift` for response-only drift rehearsal. `list_permission_package_approval_requests` accepts an optional `reviewer` field so an admin agent can fetch only the requests that reviewer is allowed to handle; with reviewer routing configured, omitting `reviewer` defaults non-platform administrators to their authenticated actor's queue. Approve and reject validate the same reviewer route before status changes, while withdraw is limited to the original pending requester. This endpoint requires `X-Admin-Key` unless the API is explicitly started with `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true` for local development.

AgentHarbor 也通过 `POST /api/v1/management/mcp` 暴露同一套管理工作流。管理 Agent 可以使用 `preflight_permission_package` 在应用前执行只读预检，使用 `check_permission_package_production_readiness` 获取上线就绪状态，也可以使用 `export_permission_package_production_evidence` 生成有边界的上线状态报告。REST 端点 `POST /api/v1/permission-packages:preflight` 用于只读应用前预检；`GET /api/v1/permission-packages/production-readiness?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&subjectId=` 用于只读上线就绪门禁；`GET /api/v1/permission-packages/production-readiness/report?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&subjectId=` 用于有边界的上线状态报告；`GET /api/v1/permission-packages/applications/health?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&limit=` 用于只读落地状态巡检；`GET /api/v1/permission-packages/applications/{id}/impact?tenantId=&workspaceId=` 用于只读影响复核和处置规划，也支持 `rehearsal=grant_drift` 做仅影响响应的漂移演练。审批队列过滤以及 REST/管理 MCP 的审批通过或拒绝都会把 `reviewer` 绑定到已认证管理员身份；配置审批人路由后，非平台管理员省略 `reviewer` 也会默认读取自己的审批队列，避免调用方通过参数或省略参数扩大查看范围。除非本地开发显式设置 `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true`，否则这些端点与其他管理 API 一样需要 `X-Admin-Key`。

## Runtime Configuration

Use `.env.example` as the local configuration template.

| Variable | Purpose |
| --- | --- |
| `AGENT_HARBOR_ADDR` | API listen address. Defaults to `:9090`. |
| `AGENT_HARBOR_DEPLOYMENT_MODE` | Optional deployment mode. Set to `production` for deployment-style preflight checks that block development-only admin bypass, private-upstream flags, malformed, invalid-actor, weak, reserved-actor, or misleading bootstrap admin identities, production configs without a bootstrap platform administrator, conflicting bootstrap admin keys, malformed or unauthenticated approval reviewer routing, missing or weak session secrets, missing or invalid persistent storage, and missing or weak credential encryption keys before startup, and disable default local CORS origins. 可选部署模式；设置为 `production` 后，启动前会执行部署预检并阻断开发专用的管理绕过、私有上游开关、格式错误、actor 格式无效、弱值、使用保留 actor 或表达误导的引导管理员身份、缺少引导平台管理员的生产配置、冲突的引导管理员密钥、格式错误或没有可认证审批人身份的审批人路由、缺失或弱值会话密钥、缺失或无效的持久化存储，以及缺失或弱值凭据加密密钥，并关闭默认本地 CORS 来源。 |
| `AGENT_HARBOR_ADMIN_KEY` | Optional shared management API key. Management and audit endpoints require this key, a named admin identity, or the explicit development unauthenticated flag. Production mode requires at least 16 characters and rejects common weak values such as `admin`, `secret`, `password`, `test-admin`, or `local-admin-key`. 可选共享管理 API 密钥；生产模式要求至少 16 个字符，并拒绝常见弱值。 |
| `AGENT_HARBOR_ADMIN_IDENTITIES` | Optional named admin identities for production approvals and scoped administration. Use comma or semicolon separated entries: `actor=key` for a platform admin, or `actor=key\|role=tenant_admin\|tenant=tenant-east\|workspace=ws-support` for a tenant/workspace-scoped admin. Supported roles are `platform_admin`, `tenant_admin`, and `security_reviewer`; scoped roles must include `tenant`, while `platform_admin` entries must not include `tenant` or `workspace` because platform administrators are intentionally unscoped. Actors must use 1-80 letters, numbers, dots, underscores, hyphens, or at signs; actors and keys must be unique. Production preflight validates the full configured syntax before storage initialization, including duplicate actors/keys, reserved actors (`admin-key` and `local-dev`), role names, role/scope consistency, scoped tenant requirements, 16-character key length, and common weak values. Production must include `AGENT_HARBOR_ADMIN_KEY` or at least one `role=platform_admin` named identity so recovery administration remains possible. In production, a named identity key must also be different from `AGENT_HARBOR_ADMIN_KEY`. Matching keys set `requestedBy` and `reviewedBy` to the actor, prevent reviewer impersonation, and restrict scoped admins across REST management APIs and management MCP tools. 可选具名管理身份，用于生产审批和范围化管理；`actor=key` 表示平台管理员，`actor=key\|role=tenant_admin\|tenant=tenant-east\|workspace=ws-support` 表示限定租户/工作区的管理员。支持的角色为 `platform_admin`、`tenant_admin` 和 `security_reviewer`；范围化角色必须包含 `tenant`，而 `platform_admin` 不能包含 `tenant` 或 `workspace`，因为平台管理员按设计不受租户范围限制。actor 必须使用 1-80 位字母、数字、点、下划线、连字符或 at 符号；actor 和 key 必须唯一。生产预检会在存储初始化前完整校验已配置语法，包括重复 actor/key、保留 actor（`admin-key` 和 `local-dev`）、角色名、角色与范围一致性、范围化租户要求、16 字符 key 长度和常见弱值。生产环境必须包含 `AGENT_HARBOR_ADMIN_KEY` 或至少一个 `role=platform_admin` 具名身份，确保恢复管理入口仍然可用，并要求具名身份 key 与 `AGENT_HARBOR_ADMIN_KEY` 不同。匹配后审批发起人与审批人来自认证身份，并且范围化管理员会在 REST 管理 API 与管理 MCP 工具中受到同一边界限制。 |
| `AGENT_HARBOR_SESSION_SECRET` | Optional in development and required in production for web-console HttpOnly session signing. Production mode requires at least 32 characters and rejects common weak values. Web 控制台 HttpOnly 会话签名密钥；开发模式可选，生产模式必填，且必须至少 32 个字符并避免常见弱值。 |
| `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN` | Development-only boolean. Allows management endpoints without `X-Admin-Key` only when no admin key or named identities are configured. Defaults to `false`. |
| `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` | Development-only boolean. Allows loopback/private upstream endpoints for local scenarios when set to `true`. Defaults to `false`. |
| `AGENT_HARBOR_APPROVAL_REVIEWERS` | Optional approval reviewer routing rules. Use comma or semicolon separated `reviewer=tenantId/workspaceId` entries, for example `security-root=tenant-root/*;security-east=tenant-east/ws-support`. `*` allows any tenant or workspace. Production preflight validates the configured format before storage initialization, and each configured reviewer must match `admin-key` or an `AGENT_HARBOR_ADMIN_IDENTITIES` actor with `role=security_reviewer` or `role=platform_admin`. For scoped reviewer identities, the approval route tenant must match the identity tenant, and a workspace-scoped identity cannot use a wider `*` workspace route. 可选审批人路由规则；生产预检会在存储初始化前校验已配置的格式，并要求每个审批人匹配 `admin-key` 或一个 `AGENT_HARBOR_ADMIN_IDENTITIES` actor，且角色必须是 `security_reviewer` 或 `platform_admin`。对于带范围的审批人身份，审批路由租户必须与身份租户一致；如果身份已限定工作区，则不能使用更宽的 `*` 工作区路由。 |
| `AGENT_HARBOR_CORS_ORIGINS` | Optional comma or semicolon separated browser origins. Entries must be full `http` or `https` origins with scheme, host, and optional port only; wildcard, path, query, fragment, user-info, or non-HTTP entries are rejected. Development mode adds these origins to the default local console origins; production deployment mode allows only the origins listed here. Use it for isolated browser gates, non-default local frontend ports, or production console domains. 可选浏览器来源白名单，使用逗号或分号分隔。每一项必须是完整的 `http` 或 `https` Origin，只能包含 scheme、host 和可选端口；通配、路径、查询、片段、用户信息或非 HTTP 来源都会被拒绝。开发模式会把这些来源追加到默认本地控制台来源；生产部署模式只允许这里显式列出的来源。可用于隔离浏览器门禁、非默认本地前端端口或生产控制台域名。 |
| `AGENT_HARBOR_DATABASE_URL` | Optional in development and required in production PostgreSQL connection string. Production preflight validates that it is parseable before PostgreSQL initialization and does not echo credentials in startup errors. If unset outside production, AgentHarbor uses the in-memory repository for local evaluation only. PostgreSQL 连接串；开发模式可选，生产模式必填。生产预检会在 PostgreSQL 初始化前校验其可解析性，并且启动错误不会回显凭据。非生产未设置时会使用内存仓库，仅适合本地评估。 |
| `AGENT_HARBOR_CREDENTIAL_KEY` | 32 random raw bytes or a base64-encoded 32-byte random key used to encrypt persisted agent credentials. Required with PostgreSQL; repeated, low-diversity, or common sample keys are rejected. 用于加密持久化 Agent 凭据的 32 字节随机原始值或 base64 编码随机 key；配置 PostgreSQL 时必填，重复、低多样性或常见示例 key 会被拒绝。 |
| `AGENT_HARBOR_TEST_DATABASE_URL` | PostgreSQL connection string used by integration tests. |
| `VITE_API_BASE` | Frontend API base URL. |

Run `make production-hardening` before any deployment-style handoff. It proves that `AGENT_HARBOR_ADMIN_KEY` protection is enforced across management APIs, that management routes fail closed without configured admin authentication, that console login behind a trusted HTTPS-forwarding proxy emits a Secure session cookie, that `AGENT_HARBOR_DEPLOYMENT_MODE=production` rejects development-only flags, weak or conflicting bootstrap admin keys, malformed approval reviewer routing, missing or weak session secrets, missing or invalid persistent storage, missing credential encryption keys, and invalid CORS origins before startup or storage initialization, and that `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` stays disabled unless explicitly set for local development scenarios.

部署式交付前请先运行 `make production-hardening`。它会验证管理 API 的管理员密钥保护、未配置管理员认证时的失败关闭行为、可信 HTTPS 转发代理后的控制台登录会设置 Secure 会话 Cookie、生产模式会在启动或存储初始化前拒绝开发专用开关、弱或冲突的引导管理员密钥、缺失或弱值会话密钥、缺失或无效的持久化存储、缺失凭据加密密钥和无效 CORS 来源，并确认私有上游访问默认保持关闭。

PostgreSQL example:

```bash
export AGENT_HARBOR_CREDENTIAL_KEY="$(openssl rand -base64 32)"
AGENT_HARBOR_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
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
- `GET /api/v1/system/info`
- `GET /api/v1/contracts/providers`
- `GET /api/v1/contracts/channels`

The web console uses `GET /api/v1/system/info` after `/healthz` to verify API compatibility before running permission changes. The same compatibility metadata also reports whether console authentication is required, so the Connection diagnostics panel can distinguish deployment-style login requirements from explicit local development bypass. If the endpoint is unavailable or required capabilities are missing, the console blocks runtime validation with an upgrade prompt instead of surfacing late-stage business errors.

Web 控制台会在 `/healthz` 之后读取 `GET /api/v1/system/info`，先确认 API 兼容信息，再执行权限变更。同一份兼容信息也会返回控制台是否要求登录，因此连接诊断可以区分部署式登录要求和显式本地开发绕过。如果端点不可用或缺少必要能力，控制台会在运行验证前提示升级 API，而不是让管理员在后续流程里遇到零散业务错误。

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

The tenant access profile endpoint is read-only. It returns configured grants, effective scope calculations, invalid historical scope evidence, and recent trace records for a registered tenant subtree. `traceLimit=0` disables recent traces.

## Project Docs

- [CONTRIBUTING.md](CONTRIBUTING.md): contribution workflow and verification expectations.
- [SECURITY.md](SECURITY.md): private vulnerability reporting and security handling.
- [.env.example](.env.example): local configuration template.
- [ROADMAP.md](ROADMAP.md): public product and contribution direction.
- [docs/engineering/](docs/engineering): release, review, dependency, and engineering workflow references.
- [CHANGELOG.md](CHANGELOG.md): public release notes and notable changes.

## License

AgentHarbor is released under the [Apache License 2.0](LICENSE).
