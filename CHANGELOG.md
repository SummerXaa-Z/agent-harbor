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
