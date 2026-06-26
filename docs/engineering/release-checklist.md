# Release Checklist

Use this checklist before merging larger behavior changes, cutting a tagged release, or declaring `main` ready for downstream integration.

For the current v0.2.0 local validation record, see `docs/engineering/0.2.0-local-validation-evidence.md`.

当前 v0.2.0 本地验收材料见 `docs/engineering/0.2.0-local-validation-evidence.md`。

## Required Local Checks

Run the uncached local release gate:

```bash
make release-check
```

Use the checked-in Go, Node, and pnpm pins when running release checks locally or in CI.

This covers:

- Go formatting via `make gofmt-check`
- uncached Go tests via `make test-fresh`
- `go vet` via `make vet`
- Go package build via `make build`
- production safety baseline via `make production-hardening`
- approval-required permission package journey via `make scenario-permission-package-approval`
- browser-facing AI Admin approval journey via `make ai-admin-browser-journey`
- scoped administrator tenant/workspace boundary via `make scenario-admin-tenant-boundary`
- managed administrator lifecycle via `make scenario-admin-access-management`
- tenant permission center projection via `make scenario-tenant-permission-center`
- web console production journey smoke gate via `make web-console-production-journey`
- frontend unit tests via `make frontend-test`
- frontend production build via `make frontend-build`
- scenario script syntax checks via `make scenario-scripts-lint`
- GitHub YAML configuration parse checks via `make github-config-lint`

The production safety baseline starts a local API with `AGENT_HARBOR_ADMIN_KEY` set and private upstreams disabled. It must prove that health remains public, management APIs reject missing or wrong admin keys, management routes fail closed when no admin authentication is configured, `AGENT_HARBOR_DEPLOYMENT_MODE=production` rejects development-only admin bypass, private-upstream flags, malformed, invalid-actor, weak, conflicting, reserved-actor, scoped-platform-admin, or platform-admin-missing bootstrap admin identities, malformed, unbound, or oversized approval reviewer routing, missing or weak session secrets, missing or invalid persistent storage, missing credential encryption keys, and invalid CORS origins before startup or storage initialization, permission-package and management MCP endpoints use the same admin-key protection, loopback/private MCP upstreams are rejected by default, and public HTTPS MCP targets remain registrable.

生产安全基线还必须验证 `AGENT_HARBOR_DEPLOYMENT_MODE=production` 会在启动或存储初始化前阻断开发专用的管理绕过、私有上游开关、格式错误、actor 格式无效、弱值、冲突、使用保留 actor、平台管理员携带租户范围或缺少平台管理员的引导管理员身份、格式错误、未绑定认证身份或范围过宽的审批人路由、缺失或弱值会话密钥、缺失或无效的持久化存储、缺失凭据加密密钥和无效 CORS 来源。

The approval-required permission package journey gate starts an isolated local API and mock MCP service when `BASE_URL` is not provided. It must prove draft, approval request, withdrawal, approval expiry metadata, approved preflight, approved apply, atomic one-time approval consumption, consumed-approval reuse rejection, runtime allow/deny, tenant access-profile, permission package application, impact review, readiness report, and applied audit records without adding browser automation dependencies.

需审批权限包旅程门禁在未提供 `BASE_URL` 时会启动隔离端口的本地 API 和 mock MCP 服务；它必须验证草稿、审批请求、撤回、审批过期元数据、已审批预检、已审批应用、审批一次性原子消费、已消费审批复用拒绝、运行时允许/拒绝、租户访问画像、权限包应用、影响复核、上线状态报告和应用审计记录，且不新增浏览器自动化依赖。

The browser-facing AI Admin approval journey gate starts the API, MCP tool service, and web console on isolated ports. It must prove the browser origin is served, the subject-id header is accepted by CORS, and the same approval-required permission package journey completes while the console server is live.

浏览器侧 AI Admin 旅程门禁会在隔离端口启动 API、MCP 工具服务和 Web 控制台；它必须验证浏览器来源可访问、subject-id 请求头通过 CORS，并在控制台服务运行时完成同一条需审批权限包旅程。

The managed administrator lifecycle gate must prove platform-created administrator identities cannot reuse bootstrap administrator actors, can log in with scoped boundaries, cannot manage administrators as tenant admins, cannot escape tenant/workspace scope, do not expose one-time key material in lists or audit records, invalidate pre-rotation browser sessions when keys rotate, invalidate existing browser sessions when identities are disabled, reject old and disabled keys, reject key rotation for disabled identities, reject repeated disablement without adding duplicate lifecycle audit events, expose structured application error data through management MCP tool errors, and record lifecycle audit actions.

托管管理员生命周期门禁必须验证平台创建的管理员身份不能复用引导管理员 actor、可按范围登录、租户管理员不能管理管理员、租户/工作区边界不能被扩大、列表和审计记录不会暴露一次性密钥材料、密钥轮换会让轮换前浏览器会话失效、禁用身份会让已有浏览器会话失效、旧密钥和已禁用密钥会被拒绝、已禁用身份不能再轮换密钥、重复禁用不会追加重复生命周期审计事件、Management MCP 工具错误会暴露结构化业务错误数据，并记录生命周期审计动作。

The web console production journey gate starts an isolated local API, the official SDK MCP demo service, and the web console. It must prove the console is served, primary journey routes are reachable, and the production journey, language, and navigation regression tests pass without adding browser automation dependencies.

Web 控制台生产旅程门禁会启动隔离端口的本地 API、官方 SDK MCP 演示服务和 Web 控制台；它必须验证控制台可以访问、主旅程路由可达，并在不新增浏览器自动化依赖的前提下通过生产旅程、语言和导航回归测试。

Run PostgreSQL integration when the change touches persistence, migrations, audit behavior, credential storage, route policies, or CI database wiring:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

Run end-to-end scenarios when user-visible control-plane or data-plane behavior changes. Start the API first, then run:

```bash
ADMIN_KEY=local-admin-key make scenario-all
```

For web console changes, run `make demo` and manually verify that the console opens directly on Permission Changes, the separate product-message card and old card-board UI are absent, the five-step approval and readiness path is visible, the current step is marked, advanced checks are collapsed by default, the main request form uses readable tenant/caller/target/access object choices, raw subject selectors stay in Advanced settings, and runtime checks are ready against the official SDK MCP demo service. Also verify the Self-Check workspace core permission loop can run against the local MCP service, and the non-destructive reset returns that self-check session to its default scope. Confirm the tenant access journey remains usable in both English and Simplified Chinese, including the language toggle, Tenant Permission Console, capability governance entry point, runtime records metrics, and trace records labels.

If the default demo ports are already in use, run the same browser check with isolated demo ports. `scripts/demo.sh` must automatically wire the frontend API base and local CORS from these port values:

```bash
AGENT_HARBOR_DEMO_API_PORT=19094 \
AGENT_HARBOR_DEMO_FRONTEND_PORT=15184 \
MOCK_MCP_PORT=18794 \
  make demo
```

如果默认 demo 端口已被占用，使用上面的隔离端口执行同样的浏览器检查；`scripts/demo.sh` 应根据这些端口自动配置前端 API 地址和本地 CORS。

For release-candidate validation of the browser-facing AI Admin path, run:

```bash
make ai-admin-browser-journey
```

This gate starts the API with split requester and reviewer admin identities, official SDK MCP demo service, and web console. It verifies browser CORS allows `X-AgentHarbor-Subject-Id`, rejects requester-key reviewer impersonation, and then runs the approval-required permission package journey against those services.

该门禁会用分离的申请人与审批人管理身份启动 API，同时启动官方 SDK MCP 演示服务和 Web 控制台；它会验证浏览器 CORS 允许 `X-AgentHarbor-Subject-Id`、申请人 key 冒充审批人会被拒绝，然后再跑完整的需审批权限包旅程。

If the default local ports are already in use, run the same gate with isolated ports:

```bash
AGENT_HARBOR_BROWSER_GATE_API_PORT=19090 \
AGENT_HARBOR_BROWSER_GATE_FRONTEND_PORT=15174 \
MOCK_MCP_PORT=18787 \
  make ai-admin-browser-journey
```

如果默认本地端口已被开发服务占用，可以使用上面的隔离端口运行同一个浏览器门禁。

Before declaring v0.2.0 permission-package work or a permission-package release candidate ready, `make release-check` must include the dependency-free approval-required journey. When you need SDK-service parity evidence, run the same journey against a local API with private upstreams enabled:

```bash
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
BASE_URL=http://127.0.0.1:9090 MCP_SERVER_MODE=real make scenario-permission-package-approval
```

This scenario must prove draft, approval request, withdrawal, approval expiry metadata, approved preflight, approved apply, atomic one-time approval consumption, consumed-approval reuse rejection, runtime allow/deny, tenant access-profile, permission package application, impact review, readiness report, and applied audit records.

## GitHub Checks

Before merging:

- Confirm the PR CI is green for Backend, PostgreSQL integration, and Frontend.
- Confirm no job reached its timeout budget; timeouts indicate a workflow defect or unexpectedly slow test path that needs investigation.
- Confirm the PR description lists the local verification commands that were run.
- Confirm docs and scenarios describe behavior that exists in the current diff.

After merging:

- Confirm the `main` push CI is green.
- Pull the latest `main` locally.
- Check `git status --short --branch` is clean and tracking `origin/main`.

## Release Notes

For a tagged release or downstream handoff, summarize:

- behavior changes and compatibility notes
- schema or migration changes
- security and credential handling changes
- verification evidence, including PostgreSQL and scenario coverage when relevant
- known follow-ups that should not block the release

Keep release notes short, factual, and tied to merged commits or PRs.
