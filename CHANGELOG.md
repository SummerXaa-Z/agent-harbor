## [2026-05-30 17:30] Session: Sprint 5 Proxy Retry and Error Classification

### 完成
- 新增 Sprint 5 brief / design / implementation plan，范围聚焦在 per-target retry policy 和 upstream error classification。
- `channelConfig.retry` 支持 `maxAttempts`、`backoffMs`、`statusCodes`，默认单次调用，显式 opt-in 后才重试。
- MCP/OpenAPI proxy 会在 retryable 5xx 或网络错误时重建 upstream request 并重发，保留原 request body。
- Proxied upstream response 新增 `X-AgentHarbor-Upstream-Attempts`，成功重试和重试耗尽都能看到实际尝试次数。
- Proxy request body 缓冲上限设为 4MiB，超限返回 `413 PAYLOAD_TOO_LARGE`，避免 retry replay 带来无界内存占用。
- Retry loop 遇到 canceled context 会停止，zero-backoff 也会检查 context 状态。
- Gateway-generated upstream failures 细分为 `UPSTREAM_TIMEOUT`、`UPSTREAM_DNS_ERROR`、`UPSTREAM_TLS_ERROR`、`UPSTREAM_CONNECT_ERROR` 和 fallback `UPSTREAM_ERROR`。
- 前端 Create Agent 表单新增 retry attempts / backoff ms 输入并阻止空值/NaN payload，contracts/catalog/sample data 暴露 `retry` 字段。
- 新增 `scripts/demo-sprint5-retry-config.sh`，验证 retry config 创建成功和非法 retry config 拒绝。

### 决策
- 默认 `maxAttempts=1`，避免对非幂等 MCP tool call 造成意外重复执行。
- Sprint 5 不引入 circuit breaker、jitter、全局 retry budget；这些等真实流量需求出现再做。
- Retry 状态码限制为 5xx，避免对 4xx 调用方错误做无意义重试。

### 待办
- 下一轮把 attempts/error classification 接入 OTel spans/metrics，让 Runtime Signals 从 mock 转真实数据。
- route-level retry override 等 route policy 对象成型后再做。

### 验证
- TDD targeted tests for retry success, retry exhaustion, invalid retry config, connect/TLS/DNS/timeout classification.
- Reviewer P2 fixes targeted tests for oversized proxy body, data-plane DNS classification, canceled-context retry stop, zero-backoff context check.
- `go test -count=1 ./...`
- `go vet ./...`
- `go build ./...`
- `pnpm -C frontend build`
- `bash -n scripts/demo-governance-loop.sh scripts/demo-sprint2-cleanup.sh scripts/demo-sprint3-mcp-policy.sh scripts/demo-sprint4-credentials.sh scripts/demo-sprint5-retry-config.sh`
- `AGENT_HARBOR_TEST_DATABASE_URL=... go test ./internal/store -run TestPostgresRepositoryRoundTrip -count=1`（临时 PostgreSQL 16 容器）
- `scripts/demo-governance-loop.sh` + `scripts/demo-sprint2-cleanup.sh` + `scripts/demo-sprint3-mcp-policy.sh` + `scripts/demo-sprint4-credentials.sh` + `scripts/demo-sprint5-retry-config.sh` against local API with `AGENT_HARBOR_ADMIN_KEY`

### 影响文件
- `internal/httpapi/server.go` / `server_test.go` / `error_classification_test.go`：retry policy parsing、proxy retry loop、attempt header、error classification。
- `internal/domain/errors.go`：新增 upstream DNS/TLS/connect error helpers。
- `frontend/src/App.tsx` / `data.ts`：Create Agent retry fields 和 sample contract metadata。
- `internal/contracts/catalog.go`：channel contract 暴露 `retry` 字段。
- `scripts/demo-sprint5-retry-config.sh`、`README.md`、`docs/sprints/`、`docs/superpowers/`：Sprint 5 行为记录。

## [2026-05-30 16:10] Session: Sprint 4 Secret Header Injection

### 完成
- 后端 `POST /api/v1/agents` 新增 Agent-level `credentials` 输入，响应、读取、列表均不会回显明文密钥。
- `channelConfig.credentialHeaders` 支持声明 upstream header 到 credential key 的映射，数据面 proxy 授权通过后自动注入对应 header。
- `channelConfig.headers` 继续只允许非 secret header；`Authorization` / `Cookie` / `X-Api-Key` 等仍必须走 credentials。
- PostgreSQL 新增 `credentials_ciphertext` migration，非空 credentials 通过 AES-GCM ciphertext 持久化；`AGENT_HARBOR_CREDENTIAL_KEY` 支持 raw/base64 32-byte key。
- Credential key 收紧为短 identifier，避免把真实 secret 误填成 key 后通过 `credentialHeaders` 回显。
- PostgreSQL 模式缺少 `AGENT_HARBOR_CREDENTIAL_KEY` 时启动期 fail fast，不再等到首次写入/读取 credential 才暴露配置错误。
- 前端 Create Agent 表单新增 credential header/key/value 输入，提交后由后端脱敏保存。
- 新增 `scripts/demo-sprint4-credentials.sh` 验证 credentialed Agent 创建成功且 create/get/list 响应不泄漏 secret。

### 决策
- Sprint 4 只支持创建时写入 credentials，不做 partial update / rotate；轮换留给后续专门 API。
- `credentialHeaders` 留在 `channelConfig` 中作为非密钥映射，真正 secret 只进入 Agent-level `credentials`。
- 内存模式不需要 `AGENT_HARBOR_CREDENTIAL_KEY`；PostgreSQL 模式一律需要 key，减少配置漂移。
- AES-GCM AAD / key version 留到 credential rotation sprint 一起设计，避免这轮 migration 过度展开。

### 验证
- `go test ./internal/httpapi ./internal/store ./internal/security -count=1`
- `go test -count=1 ./...`
- `go vet ./...`
- `go build ./...`
- `pnpm -C frontend build`
- `bash -n scripts/demo-governance-loop.sh scripts/demo-sprint2-cleanup.sh scripts/demo-sprint3-mcp-policy.sh scripts/demo-sprint4-credentials.sh`
- `AGENT_HARBOR_TEST_DATABASE_URL=... go test ./internal/store -run TestPostgresRepositoryRoundTrip -count=1`（临时 PostgreSQL 16 容器，覆盖 ciphertext round-trip）
- `scripts/demo-governance-loop.sh` + `scripts/demo-sprint2-cleanup.sh` + `scripts/demo-sprint3-mcp-policy.sh` + `scripts/demo-sprint4-credentials.sh` against local API with `AGENT_HARBOR_ADMIN_KEY`

### 影响文件
- `internal/httpapi/server.go` / `server_test.go`：credentials 输入校验、响应脱敏、upstream secret header 注入。
- `internal/security/credentials.go`：AES-GCM credential encrypt/decrypt 和 key parsing。
- `internal/store/postgres.go` / `internal/db/migrations/002_sprint4_agent_credentials.sql` / `postgres_test.go`：encrypted credential persistence。
- `frontend/src/App.tsx` / `types.ts`：Create Agent credential fields 和 API request 类型。
- `scripts/demo-sprint4-credentials.sh`、`README.md`、`docs/clean-room-spec.md`、`docs/sprints/`、`docs/superpowers/`：Sprint 4 行为记录。

## [2026-05-30 14:45] Session: Sprint 3 MCP Policy Controls

### 完成
- 新增 Sprint 3 brief / design / implementation plan，范围收敛在 MCP method-level policy、proxy headers、timeout controls。
- MCP 数据面从固定 `tools/call` 改为解析 JSON-RPC `method` 作为 `AccessGrant.routeKey`，trace 也记录真实 method。
- 无效 MCP JSON 或缺失/空 `method` 返回 `400 VALIDATION_FAILED`，且不写 trace。
- `channelConfig.headers` 支持非 secret 的 string-to-string upstream header 注入，并拒绝 `Authorization`、`Cookie`、`x-api-key`、token 等 secret-like header 名。
- `channelConfig.timeoutMs` 支持 1-30000ms proxy timeout，超时返回 `504 UPSTREAM_TIMEOUT`，其他网络失败仍是 `502 UPSTREAM_ERROR`。
- 前端 Create Grant 新增 MCP route key presets：`initialize` / `tools/list` / `tools/call` / `wildcard`，保留自由输入。
- 新增 `scripts/demo-sprint3-mcp-policy.sh`，一键验证 `tools/list` grant 只允许 `tools/list`，拒绝 `tools/call`。

### 决策
- Sprint 3 不引入 encrypted credential store；headers 只允许非 secret 值，真正密钥注入留到下一轮。
- MCP method 解析在授权前完成；解析失败不写 trace，因为还没有可信 route key。
- Timeout 是 per-target channel config，默认 10s，最大 30s，避免 demo gateway 被慢 upstream 拖住。

### 血泪教训
- `x-api-key` 这种 header 名不会被只查 `api_key` 的规则挡住，secret-like 检测需要同时看原串和去掉 `-/_/space` 后的归一化形式。
- `Cookie` / `Set-Cookie` 也属于 credential-bearing header，不能只盯 Authorization / token。
- MCP JSON-RPC method 是大小写敏感的 route key，grant 匹配不能继续沿用 Sprint 0 的 `EqualFold` 宽松策略。
- 解析 MCP body 后必须把 request body 放回去，否则 Sprint 2 的 upstream proxy 会拿到空 body。
- method-level policy 会改变旧脚本的语义；demo 必须显式发送匹配的 JSON-RPC `method`。

### 验证
- `go test -count=1 ./...`
- `go vet ./...`
- `go build ./...`
- `pnpm -C frontend build`
- `bash -n scripts/demo-governance-loop.sh scripts/demo-sprint2-cleanup.sh scripts/demo-sprint3-mcp-policy.sh`
- `AGENT_HARBOR_TEST_DATABASE_URL=... go test ./internal/store -run TestPostgresRepositoryRoundTrip -count=1`（临时 PostgreSQL 16 容器）
- `scripts/demo-governance-loop.sh` + `scripts/demo-sprint2-cleanup.sh` + `scripts/demo-sprint3-mcp-policy.sh` against local API with `AGENT_HARBOR_ADMIN_KEY`
- Playwright route preset smoke against Vite dev server

### 影响文件
- `internal/httpapi/server.go` / `server_test.go`：MCP method parsing、headers/timeout validation、upstream timeout handling。
- `internal/security/validation.go`：secret-like key/header 判断增强。
- `internal/domain/errors.go`：新增 `UPSTREAM_TIMEOUT`。
- `frontend/src/App.tsx` / `styles.css`：Create Grant MCP preset 控件。
- `scripts/demo-sprint3-mcp-policy.sh`：新增 method policy demo。
- `README.md` / `docs/sprints/` / `docs/superpowers/`：Sprint 3 行为和计划记录。

## [2026-05-29 17:45] Session: Sprint 2 Governance Proxy

### 完成
- 新增 Sprint 2 brief / design / execution plan，延续“轻 PRD + 组队直接实现”的节奏。
- 后端管理读接口支持 `tenantId` / `workspaceId` scope：agents、api keys、access grants、audit traces 行为对齐 memory 与 PostgreSQL。
- 新增 `DELETE /api/v1/agents/{id}` 软禁用 Agent；禁用 caller 的旧 Agent Key 不再认证，禁用 target 也会拒绝后续数据面调用并记录 denied trace。
- 新增 `DELETE /api/v1/access-grants/{id}` 撤销授权；撤销后同一路由调用回到 403 denied。
- MCP/OpenAPI 数据面在授权通过后可按 target `channelConfig.endpoint` 真实转发，并透传 upstream status / content-type / body；无 endpoint 的目标保留 Sprint 1 stub 响应。
- OpenAPI relative path proxy 继续拒绝 traversal / URL 注入，upstream 网络失败返回 `502 UPSTREAM_ERROR` 并保留 allowed trace 证据。
- 前端控制台新增 scope 输入、列表 scoped refresh、Agent Disable、Access Grant Revoke，revoked grants 以 disabled/deny 状态展示。
- 调整两张主表桌面最小宽度，避免 1440 宽默认视野下为操作列横向滚动。
- 更新 `scripts/demo-governance-loop.sh`，避免真实 proxy 误打外部示例 endpoint；新增 `scripts/demo-sprint2-cleanup.sh` 覆盖 revoke / disable / scoped list。

### 决策
- 本 sprint 的 proxy 单测使用 `httptest`，不把 loopback endpoint 暴露给公共 create-agent API；这样不绕开现有 SSRF/unsafe endpoint 防护。
- Demo 脚本继续用无 endpoint 的 local target 验证治理闭环；真实 MCP/OpenAPI endpoint 转发由后端测试保证。
- Cleanup controls 暂时是平台管理面动作，不引入完整 RBAC 审批流。

### 血泪教训
- 一旦 endpoint 从“配置字段”变成“真实转发”，旧 demo 里的 `https://api.example.com/mcp` 就会变成不稳定外部依赖；演示脚本必须避免假 endpoint。
- UI 文本同时出现在隐藏 `<option>` 和表格里时，Playwright `get_by_text().first()` 会抓错对象；表格行为验证要用 scoped selector。
- 禁用 caller 只挡住旧 key 还不够，禁用 target 后也应该停止接流量，否则 cleanup 语义不完整。

### 验证
- `go test -count=1 ./...`
- `go vet ./...`
- `go build ./...`
- `pnpm -C frontend build`
- `bash -n scripts/demo-governance-loop.sh scripts/demo-sprint2-cleanup.sh`
- `AGENT_HARBOR_TEST_DATABASE_URL=... go test ./internal/store -run TestPostgresRepositoryRoundTrip -count=1`（临时 PostgreSQL 16 容器）
- `scripts/demo-governance-loop.sh` + `scripts/demo-sprint2-cleanup.sh` against local API with `AGENT_HARBOR_ADMIN_KEY`
- Playwright live smoke（desktop/mobile）和 backend-down fallback smoke

### 影响文件
- `internal/httpapi/server.go` / `server_test.go`：scope parsing、cleanup routes、target disable denial、upstream proxy、trace tests。
- `internal/store/memory.go` / `postgres.go` / `postgres_test.go`：scope filters、disable/revoke repository methods、PG round-trip。
- `frontend/src/`：scope strip、disable/revoke API 和控制台动作。
- `scripts/`：治理闭环 demo 兼容 Sprint 2，新增 cleanup demo。
- `README.md` / `docs/sprints/` / `docs/superpowers/`：Sprint 2 用法和计划记录。

## [2026-05-29 17:04] Session: Sprint 1 Governance Loop

### 完成
- 新增 Sprint 1 brief 和执行计划，明确“轻 PRD + 直接实现”的迭代方式。
- 后端新增 `AGENT_HARBOR_ADMIN_KEY` 管理面保护，contracts/health/data-plane 保持原语义。
- 新增 PostgreSQL 持久化：启动迁移、`pgxpool` repository、`agents` / `agent_keys` / `access_grants` / `trace_events` schema。
- 新增 `GET /api/v1/access-grants` 和 audit trace filters，控制台可读取真实 grants/traces。
- 前端控制台新增创建 Agent、创建一次性 Agent Key、创建 Access Grant、trace 过滤刷新，以及 admin key 输入。
- 新增 `scripts/demo-governance-loop.sh`，覆盖 denied → grant → allowed → trace evidence 的完整演示链路。
- 修复 `time.Time,omitempty` 不省略零值导致未撤销 grant 被前端当成 `deny` 的问题，改用 Go 1.25 `omitzero`。
- 修复移动端/窄列中表格文本挤压问题，长 agent 名称按省略和横向滚动处理。
- 根据 code review 修复 fallback 吞鉴权错误、Key TTL 前后端不一致、memory/PG list 顺序不一致、admin key 普通比较、grant 输入未 trim 等问题。

### 决策
- Sprint 1 只管研发侧闭环：持久化、最小管理保护、控制台 CRUD、可脚本化 E2E。
- 管理面先用 `X-Admin-Key` 作为本地/demo 保护，不引入完整用户/RBAC。
- PostgreSQL 使用内嵌 SQL migration + `pgx/v5/pgxpool`，不在当前 sprint 引入 sqlc/goose。

### 血泪教训
- 前端带 admin key 创建成功不代表读模型也通；agents/traces/grants 的管理读接口也必须统一带 `X-Admin-Key`。
- Go `time.Time` 加 `omitempty` 仍会序列化零值结构体，前端用 truthy 判断会误判状态；Go 1.25 要用 `omitzero`。
- 桌面表格好看不代表移动端安全，To B 表格在窄屏要给 `min-width` 和横向滚动，不要把列硬挤进 390px。
- 浏览器 smoke 的初始 401 可能只是未输入 admin key 的预期 fallback，验证脚本要区分预期失败和真实 console error。
- fallback 不能吞掉 HTTP 401/403/500；只有网络不可达才进入 mock fallback，否则会把权限错误伪装成“设计模式”。

### 待办
- 增加 tenant/workspace scoped management reads/writes。
- 增加 Access Grant revoke / Agent cleanup API，方便 demo 残留清理。
- 继续推进真实 MCP/OpenAPI upstream proxy 和 method-level policy。
- 后续补 OTel spans/metrics，让 runtime signals 从样例切到真实数据。

### 影响文件
- `internal/httpapi/server.go` / `server_test.go`：admin key middleware、grant list、trace filters、零值时间回归测试。
- `internal/store/memory.go` / `postgres.go` / `postgres_test.go`：repository 扩展与 PostgreSQL 实现。
- `internal/db/`：新增 migration runner 与 Sprint 1 schema。
- `internal/app/app.go` / `cmd/agent-harbor/main.go`：按 env 选择 memory/PostgreSQL 并释放资源。
- `frontend/src/`：控制台 CRUD、admin key、真实 grants/traces、响应式表格修复。
- `scripts/demo-governance-loop.sh`：新增本地 E2E demo。
- `README.md` / `docs/sprints/` / `docs/superpowers/plans/`：更新 Sprint 1 使用说明和计划记录。

## [2026-05-29 16:45] Session: Rename to AgentHarbor

### 完成
- 将产品展示名统一为 `AgentHarbor`，仓库标识统一为 `agent-harbor`。
- 将 Go module 改为 `github.com/SummerXaa-Z/agent-harbor`。
- 将启动入口改为 `cmd/agent-harbor`。
- 将服务地址环境变量从旧名改为 `AGENT_HARBOR_ADDR`。
- 将前端包名和页面标题改为 `agent-harbor-console` / `AgentHarbor Console`。
- 将 Agent Key 前缀从旧缩写改为 `ah_`。

### 决策
- 用户确认使用 `AgentHarbor`，不要再使用旧产品名或旧仓库名。
- GitHub 仓库目标名为 `SummerXaa-Z/agent-harbor`。

### 血泪教训
- 改名不是只改 README；Go module path、import path、cmd 目录、前端 package、文档、日志、key prefix、GitHub remote 都要一起改。
- pnpm package name 改动后需要刷新 lockfile，否则仓库元数据会残留旧名。

### 待办
- GitHub 仓库 rename 后，同步本地 remote URL 和本地目录名。

### 影响文件
- `go.mod` 与 Go imports：切换到 `github.com/SummerXaa-Z/agent-harbor`。
- `cmd/agent-harbor/main.go`：新入口、日志和环境变量。
- `frontend/package.json` / `frontend/index.html` / `frontend/src/App.tsx`：前端命名。
- `README.md` / `docs/*.md` / `CHANGELOG.md`：文档命名。

## [2026-05-29 16:34] Session: Clean-room To B Frontend Rebirth

### 完成
- 新增 `frontend/`，使用 Vite + React + TypeScript 搭建 clean-room 企业控制台。
- 实现 Agent Gateway Cockpit：左侧导航、顶部环境栏、指标卡、Route Governance、Agent Registry、Contract Matrix、Evidence Runs、Runtime Signals、Audit Traces。
- 新增 `frontend/src/api.ts`，兼容 Go API envelope，支持 `VITE_API_BASE` 和后端不可用时 mock fallback。
- 新增 `frontend/src/data.ts` / `frontend/src/types.ts`，提供虚构样例数据和前端类型模型。
- 新增本地开发 CORS allowlist，允许 Vite dev/preview 读取 Go API。
- 新增 `docs/frontend-design-reference.md`，沉淀 To B 控制台设计原则和 clean-room 约束。
- 更新 `README.md` 前后端运行说明。

### 决策
- 前端不复用旧实现 `web/` 代码，全部放在独立 GitHub 仓库 `SummerXaa-Z/agent-harbor`。
- 设计参考吸收 Ant Design Pro、Semi Design、shadcn dashboard blocks、Arco Design Pro 的通用中后台模式，但不复制模板代码。
- 后端在线时 catalog / agents / traces 走 Go runtime；route policies / evidence / runtime signals 暂用本地样例面板，状态栏写作 `Go runtime + samples`。
- GitHub 仓库保持 private，提交到 `main`，不再触碰公司 GitLab。

### 血泪教训
- `apply_patch` 没有 `workdir` 参数时会按会话默认目录写文件，曾误写到原项目仓库；后续跨仓库操作必须使用绝对路径。
- Vite 调本地 Go API 会遇到 CORS，不能只靠 curl 验证；必须用真实浏览器检查。
- API 请求成功但返回空数组时不能用 mock 替换，否则会出现 `Go runtime` 标签下展示 mock agent/trace 的混合状态。
- CORS 中间件不能对所有 `OPTIONS` 一律 204，只能短路 allowlist origin 的 preflight。
- `pnpm install` 会生成 `node_modules/`，`pnpm build` 会生成 `dist/` 和 `*.tsbuildinfo`，这些需要 `.gitignore` 防止误提交。

### 待办
- 后续补真实 route policy / evidence / runtime signal API 后，把本地样例面板替换为真实数据。
- 为前端增加组件级测试或 Playwright 脚本化回归入口。
- 后续可补 Agent 注册/Grant 创建表单、详情抽屉和审计深链接。

### 影响文件
- `.gitignore`：忽略前端依赖、构建产物和 TS build info。
- `README.md`：修正 Go 运行路径，新增 frontend 运行说明。
- `docs/frontend-design-reference.md`：新增前端设计参考和 clean-room 约束。
- `frontend/`：新增完整 Vite React TS 控制台。
- `internal/httpapi/server.go`：新增本地开发 CORS。
- `internal/httpapi/server_test.go`：新增 CORS allowlist / disallow 测试。
