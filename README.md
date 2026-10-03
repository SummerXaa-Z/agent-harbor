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
- **Bounded access handoff / 接入交付有边界**: ready applications produce copyable configuration, prompt guidance, one-time short-lived tokens, revocation, and audit references without expanding live authorization.

## Project Status

AgentHarbor is at v0.6.0: the redesigned dual-surface console, notification phases, and per-holder views. It keeps developer-preview positioning — ready for local evaluation, design feedback, and early integration work; deployment-style handoffs must pass the documented production preflight before any production traffic.

AgentHarbor 当前版本为 v0.6.0：重设计双面控制台、通知阶段与持有者视图。仍保持开发者预览定位——适合本地评估、设计反馈和早期集成；部署式交付必须先通过文档化的生产预检，再承载生产流量。

Open-source timing is intentionally secondary to production hardening. Before any release-readiness claim, the current standard is that the safety baseline, release checks, and primary Permission Changes journey all pass from a fresh local checkout.

开源节奏会服从生产可用性。任何发布就绪声明之前，都必须确保安全基线、发布检查和核心权限包用户旅程能在全新本地检出中通过。

## Architecture / 技术图谱

One Go binary serves three planes — management REST, Management MCP, and the governed data-plane proxy — backed by an in-memory (default) or PostgreSQL store, with a Vite + React dual-surface console on top. The full component map, domain model, authentication surfaces, and verification topology live in [docs/engineering/architecture.md](docs/engineering/architecture.md).

单个 Go 二进制承载三个面——管理 REST、管理 MCP 和受治理的数据面代理——底层默认使用内存存储、可选 PostgreSQL,上层是 Vite + React 双面控制台。完整组件图、领域模型、认证面和验证拓扑见 [docs/engineering/architecture.md](docs/engineering/architecture.md)。

Core grant chain:

```text
Tenant
  -> Agent or target service
  -> MCP/OpenAPI capability
  -> Tenant entitlement
  -> Workspace assignment
  -> Caller instance assignment
  -> Runtime decision and trace records
```

The tenant is the primary control boundary; `dataScopes` narrow layer by layer and runtime decisions record the effective inherited scope. Management APIs require admin authentication (`AGENT_HARBOR_ADMIN_KEY` shared key, `AGENT_HARBOR_ADMIN_IDENTITIES` named roles, or a console session); the data plane uses short-lived Agent Keys plus optional `X-AgentHarbor-Subject-Id`. Day-to-day administrators are managed identities created in the console (keys shown once, rotation and disable act as immediate containment); bootstrap env identities stay read-only break-glass. Non-platform identities can additionally be bound to owned caller agents (`ownedAgentIds`), narrowing their user surface server-side (`403 HOLDER_SCOPE_DENIED`). Security headers, body limits, credential encryption, and production preflight rules are summarized in the [architecture doc](docs/engineering/architecture.md) and enforced by `make production-hardening`.

租户是主控制边界;`dataScopes` 逐层收敛,运行时决策记录最终生效的范围。管理 API 要求管理员认证(`AGENT_HARBOR_ADMIN_KEY` 共享密钥、`AGENT_HARBOR_ADMIN_IDENTITIES` 具名角色或控制台会话);数据面使用短期 Agent Key 和可选的 `X-AgentHarbor-Subject-Id`。日常管理员建议使用控制台内创建的托管身份(密钥只展示一次,轮换和禁用即可即时止损);环境变量引导身份保持只读 break-glass。非平台身份还可以绑定持有的调用方(`ownedAgentIds`),在服务端收窄其用户面(`403 HOLDER_SCOPE_DENIED`)。安全响应头、请求体限制、凭据加密和生产预检规则摘录见[架构文档](docs/engineering/architecture.md),由 `make production-hardening` 强制验证。

## Quick Start

Use the repository toolchain pins (Go from `go.mod`; Node from `.node-version`, 24–26; frontend pnpm from `frontend/package.json`).

```bash
make demo
```

Then open `http://127.0.0.1:5174/`. The demo starts the API (`:9090`, explicit unauthenticated development mode), the official MCP TypeScript SDK demo service (`:8787/mcp`), and the web console (`:5174`) together. `Ctrl+C` stops all demo services. If ports are busy, set `AGENT_HARBOR_DEMO_API_PORT`, `AGENT_HARBOR_DEMO_FRONTEND_PORT`, and `MOCK_MCP_PORT`; the script wires the frontend API base and local browser CORS automatically.

A fresh system opens the console on **Getting Started** (six-step setup checklist); once tenant, agent, capability, and grant-chain setup is complete it opens on **Access Query**, and daily changes happen in **Permission Changes**. The console reads `VITE_API_BASE` (default `http://127.0.0.1:9090`); when the backend is unavailable it falls back to a read-only sample preview with a persistent warning and disabled mutations. Journey detail lives in [docs/product/0.2.0-ai-admin-permission-journey.md](docs/product/0.2.0-ai-admin-permission-journey.md) and the evaluation archive in [docs/product/0.4.0-console-eval.md](docs/product/0.4.0-console-eval.md).

For manual troubleshooting, the three-terminal path: `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run`, then `make real-mcp`, then `cd frontend && pnpm install && pnpm dev`.

## Verify

```bash
make check                    # backend + frontend + static wiring (fast daily gate)
make release-check            # uncached Go tests + all release scenario gates
make production-hardening     # conservative runtime defaults baseline
make evaluation-readiness     # external-evaluator pack (walkthrough, snapshot, feedback log)
```

`make release-check` includes the production safety baseline, the approval-required permission package journey, the browser-facing AI-admin journey, scoped-admin tenant boundary, managed-admin lifecycle, tenant permission center, and web-console production journey gates. See [docs/engineering/release-checklist.md](docs/engineering/release-checklist.md) for the full checklist.

## Core Journey

The most important workflow as a scriptable regression check — tenant tree, MCP target, tool discovery, capability approval, grant chain, allowed/denied calls, access profile, and audit records:

Terminal 1:

```bash
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
```

Terminal 2:

```bash
make core-journey
```

The scenario starts the dependency-free mock MCP server automatically. `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` is required only for local loopback/private upstreams and must not be enabled in production. All scenario scripts (`make scenario-all`, approval journeys, `MCP_SERVER_MODE=real`, shared-admin-key mode, public-endpoint MCP scenarios) are documented inline in `scripts/` and run the same journeys as executable gates.

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

The default scenario starts `scripts/mock-mcp-server.py` automatically for a dependency-free regression path. To run the same journey against the official MCP TypeScript SDK demo service, run `MCP_SERVER_MODE=real make scenario-permission-package-approval` against an API started with private upstreams and explicit unauthenticated development mode enabled.

## Runtime Configuration

Use [.env.example](.env.example) as the template — it documents every variable with production-mode validation rules. Summary:

| Variable | Purpose |
| --- | --- |
| `AGENT_HARBOR_ADDR` | API listen address (default `:9090`). |
| `AGENT_HARBOR_DEPLOYMENT_MODE` | `production` enables deployment preflight and blocks development-only flags/weak configs. 生产模式启用部署预检。 |
| `AGENT_HARBOR_ADMIN_KEY` | Shared management API key (production: ≥16 chars, weak values rejected). 共享管理密钥。 |
| `AGENT_HARBOR_ADMIN_IDENTITIES` | Named admin identities with optional role/tenant/workspace scope. 具名管理身份,支持角色与租户/工作区范围。 |
| `AGENT_HARBOR_SESSION_SECRET` | Console session signing key (production: required, ≥32 chars). 控制台会话签名密钥,生产必填。 |
| `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN` | Development-only unauthenticated management bypass. 仅限开发。 |
| `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` | Development-only loopback/private upstream targets. 仅限开发。 |
| `AGENT_HARBOR_APPROVAL_REVIEWERS` | Approval reviewer routing (`reviewer=tenantId/workspaceId`). 审批人路由。 |
| `AGENT_HARBOR_CORS_ORIGINS` | Explicit browser origins (production allowlist). 浏览器来源白名单。 |
| `AGENT_HARBOR_DATABASE_URL` | PostgreSQL connection string (production: required). PostgreSQL 连接串,生产必填。 |
| `AGENT_HARBOR_CREDENTIAL_KEY` | 32-byte key encrypting persisted agent credentials (required with PostgreSQL). 凭据加密密钥。 |
| `AGENT_HARBOR_TEST_DATABASE_URL` | PostgreSQL connection string for integration tests. 集成测试用连接串。 |
| `VITE_API_BASE` | Frontend API base URL. |

PostgreSQL example:

```bash
export AGENT_HARBOR_CREDENTIAL_KEY="$(openssl rand -base64 32)"
AGENT_HARBOR_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  go run ./cmd/agent-harbor
```

## API and Semantics

Full endpoint reference (management, data plane, audit/traces/metrics, target probe): [docs/engineering/api-reference.md](docs/engineering/api-reference.md). Grant-chain, route-policy, data-scope, and permission-package lifecycle semantics: [docs/engineering/architecture.md](docs/engineering/architecture.md).

完整端点参考见 [docs/engineering/api-reference.md](docs/engineering/api-reference.md);授权链、路由策略、数据范围与权限包生命周期语义见 [docs/engineering/architecture.md](docs/engineering/architecture.md)。

## Project Docs

- [docs/engineering/architecture.md](docs/engineering/architecture.md): technical map / 技术图谱.
- [docs/engineering/api-reference.md](docs/engineering/api-reference.md): endpoint reference.
- [CONTRIBUTING.md](CONTRIBUTING.md): contribution workflow and verification expectations.
- [SECURITY.md](SECURITY.md): private vulnerability reporting and security handling.
- [.env.example](.env.example): local configuration template.
- [ROADMAP.md](ROADMAP.md): public product and contribution direction.
- [CHANGELOG.md](CHANGELOG.md): public release notes and notable changes.
- [docs/engineering/](docs/engineering): release, review, dependency, and engineering workflow references.
- [docs/product/](docs/product): PRDs, journey notes, and the external evaluation archive.

## License

AgentHarbor is released under the [Apache License 2.0](LICENSE).
