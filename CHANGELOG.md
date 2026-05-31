## [2026-05-31 13:41] Session: Sprint 9 Ready Evidence

### 完成
- 在 `codex/sprint-9-route-policies` 分支补跑 focused verification，并将证据写入 PR #9。
- 将 PR #9 从 Draft 标记为 ready for review，并把标题从 `Draft: Sprint 9 route policy objects` 改为 `Sprint 9 route policy objects`。
- 验证 Sprint 9 的 route policy CRUD/disable、allow/deny priority precedence、legacy access grant fallback、cross-scope policy handling、Sprint 1-8 demo 回归、PostgreSQL route_policies round-trip、Go 后端和前端 build。

### 决策
- 保留 #9 无 GitHub checks 的事实说明：该分支早于 CI workflow；本轮以本地 focused verification 作为 ready evidence。
- 不修改 Sprint 9 代码和历史，只更新 GitHub PR 元数据。
- #2-#9 lower stack 已全部补齐 ready evidence；下一步可以回到顶层 #1 处理 integration PR 的 ready 边界。

### 血泪教训
- route policy 类 PR 的 ready evidence 必须同时证明优先级和回退路径；只验证 allow/deny 不够，还要证明 disabled policy 会被忽略并回退到 legacy grant。

### 待办
- 下一步检查顶层 PR #1 的 scope 和验证证据，决定是否继续保持 Draft 或拆出/标记 ready。

### 影响文件
- `CHANGELOG.md`：记录 PR #9 ready evidence 和 lower stack ready 里程碑。

## [2026-05-31 13:30] Session: Sprint 8 Ready Evidence

### 完成
- 在 `codex/sprint-8-management-audit` 分支补跑 focused verification，并将证据写入 PR #8。
- 将 PR #8 从 Draft 标记为 ready for review，并把标题从 `Draft: Sprint 8 management audit` 改为 `Sprint 8 management audit`。
- 验证 Sprint 8 的 management audit demo、credentialVersion 递增、audit event 过滤/排序、Sprint 1-7 demo 回归、PostgreSQL audit_events round-trip、Go 后端和前端 build。

### 决策
- 保留 #8 无 GitHub checks 的事实说明：该分支早于 CI workflow；本轮以本地 focused verification 作为 ready evidence。
- 不修改 Sprint 8 代码和历史，只更新 GitHub PR 元数据。

### 血泪教训
- management audit 类 PR 的 ready evidence 必须证明 metadata 只含非 secret 证据；`credentialVersion=2` 和 `credentialKeys=["apiToken"]` 有用，但 plaintext secret 必须从 response 与 audit list 中同时缺席。

### 待办
- 下一步从 #9 开始补 focused verification evidence，再决定是否 ready。

### 影响文件
- `CHANGELOG.md`：记录 PR #8 ready evidence 和验证结果。

## [2026-05-31 12:45] Session: Sprint 7 Ready Evidence

### 完成
- 在 `codex/sprint-7-credential-rotation` 分支补跑 focused verification，并将证据写入 PR #7。
- 将 PR #7 从 Draft 标记为 ready for review，并把标题从 `Draft: Sprint 7 credential rotation` 改为 `Sprint 7 credential rotation`。
- 验证 Sprint 7 的 PATCH Agent、credential rotation demo、Sprint 1-6 demo 回归、PostgreSQL rotated credential ciphertext、Go 后端和前端 build。

### 决策
- 保留 #7 无 GitHub checks 的事实说明：该分支早于 CI workflow；本轮以本地 focused verification 作为 ready evidence。
- 不修改 Sprint 7 代码和历史，只更新 GitHub PR 元数据。
- 修正 PR #7 的 review focus：Sprint 7 不引入 credential version history 或 key version metadata，rotation 语义是替换完整 credential bag。

### 血泪教训
- credential rotation 类 PR 的 ready evidence 必须证明“下一次 proxy call 用新 secret”而不只是 rotate endpoint 返回 200；否则无法确认运行时真的切换到了新凭据。

### 待办
- 下一步从 #8 开始补 focused verification evidence，再决定是否 ready。

### 影响文件
- `CHANGELOG.md`：记录 PR #7 ready evidence、验证结果和 credentialVersion 边界修正。

## [2026-05-31 04:52] Session: Sprint 6 Ready Evidence

### 完成
- 在 `codex/sprint-6-runtime-metrics` 分支补跑 focused verification，并将证据写入 PR #6。
- 将 PR #6 从 Draft 标记为 ready for review，并把标题从 `Draft: Sprint 6 runtime metrics` 改为 `Sprint 6 runtime metrics`。
- 验证 Sprint 6 的 runtime metrics demo、Sprint 1-5 demo 回归、PostgreSQL trace metrics round-trip、Go 后端和前端 build。

### 决策
- 保留 #6 无 GitHub checks 的事实说明：该分支早于 CI workflow；本轮以本地 focused verification 作为 ready evidence。
- 不修改 Sprint 6 代码和历史，只更新 GitHub PR 元数据。

### 血泪教训
- runtime metrics 类 PR 的 ready evidence 必须同时证明聚合结果和原始 trace 字段；`calls=2 allowed=50%` 证明 API 聚合，attempts/status/error/duration 证明可观测性来源。

### 待办
- 下一步从 #7 开始补 focused verification evidence，再决定是否 ready。

### 影响文件
- `CHANGELOG.md`：记录 PR #6 ready evidence 和验证结果。

## [2026-05-31 04:49] Session: Sprint 5 Ready Evidence

### 完成
- 在 `codex/sprint-5-proxy-retry-classification` 分支补跑 focused verification，并将证据写入 PR #5。
- 将 PR #5 从 Draft 标记为 ready for review，并把标题从 `Draft: Sprint 5 proxy retry classification` 改为 `Sprint 5 proxy retry classification`。
- 验证 Sprint 5 的 retry config demo、Sprint 1-4 demo 回归、PostgreSQL store integration、Go 后端和前端 build。

### 决策
- 保留 #5 无 GitHub checks 的事实说明：该分支早于 CI workflow；本轮以本地 focused verification 作为 ready evidence。
- 不修改 Sprint 5 代码和历史，只更新 GitHub PR 元数据。

### 血泪教训
- retry/classification 类 PR 的 ready evidence 要同时包含 targeted unit tests 和真实 demo；attempt header、最终 retryable response、DNS/TLS/connect error code、配置校验分别证明不同风险面。

### 待办
- 下一步从 #6 开始补 focused verification evidence，再决定是否 ready。

### 影响文件
- `CHANGELOG.md`：记录 PR #5 ready evidence 和验证结果。

## [2026-05-31 04:29] Session: Sprint 4 Ready Evidence

### 完成
- 在 `codex/sprint-4-secret-header-injection` 分支补跑 focused verification，并将证据写入 PR #4。
- 将 PR #4 从 Draft 标记为 ready for review，并把标题从 `Draft: Sprint 4 secret header injection` 改为 `Sprint 4 secret header injection`。
- 验证 Sprint 4 的 credential response redaction demo、governance loop demo、PostgreSQL credential encryption integration、Go 后端和前端 build。

### 决策
- 保留 #4 无 GitHub checks 的事实说明：该分支早于 CI workflow；本轮以本地 focused verification 作为 ready evidence。
- 不修改 Sprint 4 代码和历史，只更新 GitHub PR 元数据。

### 血泪教训
- secret/header 类 PR 的 ready evidence 不能只说“已脱敏”；必须跑真实 API demo，让 create/get/list response redacted 和 PostgreSQL ciphertext 不含明文 secret 都成为可复核证据。

### 待办
- 下一步从 #5 开始补 focused verification evidence，再决定是否 ready。

### 影响文件
- `CHANGELOG.md`：记录 PR #4 ready evidence 和验证结果。

## [2026-05-31 04:25] Session: Sprint 3 Ready Evidence

### 完成
- 在 `codex/sprint-3-mcp-policy-controls` 分支补跑 focused verification，并将证据写入 PR #3。
- 将 PR #3 从 Draft 标记为 ready for review，并把标题从 `Draft: Sprint 3 MCP method policy controls` 改为 `Sprint 3 MCP method policy controls`。
- 验证 Sprint 3 的 MCP method policy demo、governance loop demo、PostgreSQL store integration、Go 后端和前端 build。

### 决策
- 保留 #3 无 GitHub checks 的事实说明：该分支早于 CI workflow；本轮以本地 focused verification 作为 ready evidence。
- 不修改 Sprint 3 代码和历史，只更新 GitHub PR 元数据。

### 血泪教训
- 方法级授权类 PR 的 ready evidence 必须包含真实 demo；单靠单元测试不能给 reviewer 直观看到 `tools/list` allowed / `tools/call` denied 的行为证据。

### 待办
- 下一步从 #4 开始补 focused verification evidence，再决定是否 ready。

### 影响文件
- `CHANGELOG.md`：记录 PR #3 ready evidence 和验证结果。

## [2026-05-31 04:21] Session: Sprint 2 Ready Evidence

### 完成
- 在 `codex/sprint-2-governance-proxy` 分支补跑 focused verification，并将证据写入 PR #2。
- 将 PR #2 从 Draft 标记为 ready for review，并把标题从 `Draft: Sprint 2 governance proxy cleanup` 改为 `Sprint 2 governance proxy cleanup`。
- 验证 Sprint 2 的 governance loop demo、cleanup demo、PostgreSQL store integration、Go 后端和前端 build。

### 决策
- 保留 #2 无 GitHub checks 的事实说明：该分支早于 CI workflow；本轮以本地 focused verification 作为 ready evidence。
- 不修改 Sprint 2 代码和历史，只更新 GitHub PR 元数据。

### 血泪教训
- lower stack PR 标 ready 时，标题也要去掉 `Draft:` 前缀；否则 GitHub 状态和人类阅读信号会互相打架。

### 待办
- 下一步从 #3 开始补 focused verification evidence，再决定是否 ready。

### 影响文件
- `CHANGELOG.md`：记录 PR #2 ready evidence 和验证结果。

## [2026-05-31 04:17] Session: Stacked PR Runbook

### 完成
- 新增 `docs/engineering/stacked-pr-runbook.md`，记录 #1-#9 的当前 Draft PR stack、base/head 关系和 review scope。
- 明确 lower PR #2-#9 当前 checks=0 的原因：CI workflow 是在顶部 PR 分支中引入的。
- 写入 ready order、CI status caveat、CI-first review 可选策略和 bottom-up merge discipline。

### 决策
- 暂不重写或 force-push 已存在分支；先用 runbook 固化当前低扰动 stacked review 方案。
- 将 #1 保持为完整 integration signal，lower PR 进入 ready 前需要补本地 focused verification 证据。

### 血泪教训
- PR stack 拆好以后，还必须写明 checks 缺口；否则 reviewer 看到 lower PR 没有 checks 时会以为 CI 配置坏了。

### 待办
- 如需自动化 lower PR checks，需要单独执行 CI-first stack strategy：先落 CI foundation，再重排或合并 sprint branches。
- 下一步可从 #2 开始补 focused verification evidence，并决定是否 ready for review。

### 影响文件
- `docs/engineering/stacked-pr-runbook.md`：新增当前 PR stack 操作手册。
- `CHANGELOG.md`：记录 stack runbook 和 lower PR checks caveat。

## [2026-05-31 04:13] Session: Stacked PR Boundary Split

### 完成
- 为既有线性 Sprint 分支创建 Draft PR stack：#2 Sprint 2、#3 Sprint 3、#4 Sprint 4、#5 Sprint 5、#6 Sprint 6、#7 Sprint 7、#8 Sprint 8、#9 Sprint 9。
- 将 Draft PR #1 的 base 从 `main` 改为 `codex/sprint-9-route-policies`，使其只覆盖 Sprint 10、Sprint 11、CI workflow 和 review governance。
- 重写 PR #1 正文，补充 lower stack 列表、顶部 PR review order、风险关注点和验证证据。
- 验证分支祖先关系为线性：`main -> sprint2 -> ... -> sprint9 -> sprint10/11`。

### 决策
- 不重写历史、不 rebase、不拆提交；本轮只调整 GitHub PR 拓扑，降低对已有分支的扰动。
- 保留所有 PR 为 Draft，作为 review stack 和协作入口，而不是立即进入 ready-to-merge 状态。

### 血泪教训
- 大型集成 PR 最安全的第一步不是改代码，而是先把 base/head 拓扑调窄；这样 reviewer 可以从底层 Sprint 逐层审，而不是面对全量 diff。

### 待办
- 如果后续要逐个 ready，需要在各 Sprint PR 上补对应的最新验证证据；较早分支没有当前 CI workflow，需要按需回补或在合并后依赖主线 CI。
- 决定最终合并策略：逐 PR merge、squash、还是保留 #1 作为 integration checkpoint。

### 影响文件
- `CHANGELOG.md`：记录 Draft PR stack 创建和 #1 retarget 决策。

## [2026-05-31 04:08] Session: PR Review Governance

### 完成
- 新增 `.github/pull_request_template.md`，固定 Summary、Scope、Review Boundary、Verification、Data And Security、Follow-Ups 六个 PR 栏目。
- 新增 `docs/engineering/review-guidelines.md`，沉淀 AgentHarbor sprint stack 的 review 顺序、验证证据、安全治理检查和拆分条件。
- 新增 `docs/superpowers/plans/2026-05-31-pr-review-governance.md`，记录本次协作治理底座的实施计划。

### 决策
- PR 模板保持轻量，不引入强制 code owners 或 branch protection，避免在当前 stacked draft PR 阶段制造流程摩擦。
- Review guide 明确区分 Draft integration checkpoint 和 Ready PR：前者可大，后者必须有窄 review path。

### 血泪教训
- 只有 CI 不够；大型工程分支还需要 review boundary 文档，否则绿色检查会掩盖“难以审”的协作风险。

### 待办
- 后续可在 PR #1 拆分前，用该模板重写各子 PR 描述。
- 如团队成员增加，再考虑补 CODEOWNERS 和 branch protection。

### 影响文件
- `.github/pull_request_template.md`：新增未来 PR 的结构化描述模板。
- `docs/engineering/review-guidelines.md`：新增工程 review 指南。
- `docs/superpowers/plans/2026-05-31-pr-review-governance.md`：新增实施计划。
- `CHANGELOG.md`：记录协作治理底座落地。

## [2026-05-31 04:03] Session: CI Workflow Foundation

### 完成
- 新增 `.github/workflows/ci.yml`，在 PR 和 `main` / `codex/**` push 上运行 CI。
- CI 拆为 Backend、PostgreSQL integration、Frontend 三个 job，分别覆盖 Go test/vet/build、PostgreSQL store integration、pnpm test/build。
- 新增 `docs/superpowers/plans/2026-05-31-ci-workflow.md`，记录 workflow 结构和验证步骤。

### 决策
- 使用官方当前主版本 action：`actions/checkout@v6`、`actions/setup-go@v6`、`actions/setup-node@v6`、`pnpm/action-setup@v6`。
- PostgreSQL integration 独立成 job，避免普通 backend job 依赖服务容器，也让失败边界更清楚。
- pnpm 只由 `pnpm/action-setup` 安装，依赖缓存交给 `setup-node` 的 `cache: pnpm` 处理，避免重复缓存。

### 血泪教训
- GitHub Actions workflow 需要和本地验证命令一一对应；否则 PR 上的绿色检查容易给出比本地更弱的信号。

### 待办
- 推送后观察 Draft PR #1 的 CI checks，若 action 版本或 runner 环境有兼容问题，按失败日志修复。

### 影响文件
- `.github/workflows/ci.yml`：新增 PR/push 自动验证 workflow。
- `docs/superpowers/plans/2026-05-31-ci-workflow.md`：新增 CI workflow 实施计划。
- `CHANGELOG.md`：记录 CI 底座落地和验证策略。

## [2026-05-31 03:57] Session: Draft PR Integration Checkpoint

### 完成
- 创建 Draft PR [#1](https://github.com/SummerXaa-Z/agent-harbor/pull/1)，作为 AgentHarbor governance foundation through Sprint 11 的协作入口。
- PR 正文标注这是 large stacked branch，并写明建议 review 顺序、已验证命令和拆分前不要直接 merge 的语境。
- 确认 PR base 为 `main`，head 为 `codex/sprint-10-route-policy-retry-overrides`，当前状态为 open draft。

### 决策
- 继续采用低风险集成策略：先用 Draft PR 建立协作与 CI 入口，再决定是否按 Sprint 拆成更小 PR。
- 不在本轮做本地 merge，也不清理工作区或删除分支。

### 血泪教训
- 多个 Sprint 堆叠在一条分支上时，PR 正文必须把 review boundary 写清楚，否则 reviewer 很容易把它当成一个普通可合并改动。

### 待办
- 观察 GitHub CI；如果仓库没有 Actions，需要补 `.github/workflows` 作为后续工程底座任务。
- 后续可把 PR #1 拆成按 Sprint/能力域分层的 review stack。

### 影响文件
- `CHANGELOG.md`：记录 Draft PR 创建、集成策略和后续 PR 边界治理事项。

## [2026-05-31 03:51] Session: Sprint 11 Final Verification

### 完成
- 完成 Sprint 11 Task 5 inline code-quality review，确认 demo script、README、brief、CHANGELOG 与 transactional audit 目标一致。
- 补充 README Next Milestones，将已落地的 transactional audit 改为后续 external audit outbox/export 方向。
- 完成最终验证：后端聚焦测试、全量 Go 测试、vet/build、前端 test/build、静态检查、Sprint 11 demo against local API、临时 PostgreSQL integration。

### 决策
- 避免继续关闭已卡住的 reviewer subagent，改为当前线程 inline review 和验证，保证主线不被 graceful shutdown 阻塞。
- 保留现有 Sprint 10 dirty worktree，不将其混入 Sprint 11 final verification 提交。

### 血泪教训
- `close_agent` 对异常或长尾子进程表现为无超时 graceful shutdown，可能卡住当前线程；遇到这种情况应停止重复 close，改用 inline verification 或新 reviewer 继续推进。

### 待办
- 下一步可整理 Sprint 10 dirty worktree，拆分 retry hardening 与 Sprint 11 transactional audit 的 PR 边界。

### 影响文件
- `CHANGELOG.md`：记录 Sprint 11 final verification 和智能体关闭卡住的处理经验。
- `README.md`：更新 Sprint 11 后续路线图措辞。

## [2026-05-31] Session: Sprint 11 Transactional Management Audit

### 完成
- 新增 audited repository mutation 方法，覆盖 Agent、Agent Key、Access Grant、Route Policy 管理写入。
- Memory store 在同一把写锁内完成业务变更和 audit append。
- PostgreSQL store 使用单事务提交业务变更和 `audit_events` 写入。
- HTTP 管理 mutation 从 best-effort audit 改为事务化 audit 写入。
- 新增 Sprint 11 demo，验证 audit 可见性和 credential redaction。

### 决策
- Sprint 11 不引入异步 outbox worker；`audit_events` 先作为本地事务化事件日志。
- data-plane trace append 保持非阻塞，不纳入本次事务化范围。

### 验证
- `go test ./internal/httpapi -run 'TestManagementAuditFailureBlocksAgentCreateAndUpdate|TestManagementAuditEvents|TestRoutePolicyCRUDAndAudit' -count=1`
- `go test ./internal/store -count=1`
- `go test ./...`
- `go vet ./...`
- `go build ./...`
- `pnpm --dir frontend test`
- `pnpm --dir frontend build`
- `git diff --check`
- `bash -n scripts/demo-sprint11-transactional-audit.sh`

### 影响文件
- `internal/store/memory.go` / `internal/store/postgres.go`：事务化 audited mutations。
- `internal/httpapi/server.go` / `server_test.go`：HTTP 管理写入改用 audited repository 方法并补失败注入测试。
- `internal/store/postgres_test.go`：PostgreSQL rollback regression。
- `scripts/demo-sprint11-transactional-audit.sh`、`README.md`、`docs/sprints/`：Sprint 11 文档与 demo。

## [2026-05-31 01:47] Session: Sprint 10 Review Hardening

### 完成
- 根据 review 风险补强 retry 表单校验：新增 `frontend/src/retryForm.ts`，Agent / Route Policy 表单统一拒绝空 retry companion 字段，避免空值被静默序列化成 `0` / `null`。
- 新增 `frontend/tests/retryForm.test.mjs` 和 `frontend/package.json` 的 `test` script，覆盖默认 retry omit、空字段拒绝、正常 retry 归一化。
- 新增 `TestProxyDoesNotRetryCanceledContext`，锁定 data-plane 在 `context.Canceled` 时即使配置 retry 也只尝试一次。
- 确认既有 Sprint 10 dirty worktree 已包含 4MiB proxy body cap、DNS failure、status retry success/exhaustion 等 regression tests。

### 决策
- 前端 retry 校验统一使用整数边界：`maxAttempts` 1-4，`backoffMs` 0-1000；默认 `1/0` 不发送 retry override。
- Node 内置 `node:test` 测试放在 `frontend/tests/`，避免进入 Vite/tsc 的应用源码编译范围。

### 血泪教训
- 跨仓库续接时 `apply_patch` 会按当前线程 cwd 落文件；本次第一次把前端测试误落到 `ai-nexus/frontend/src/`，已删除。后续对 `agent-harbor` 打补丁优先使用绝对路径。

### 验证
- `go test ./...`
- `go vet ./...`
- `go build ./...`
- `pnpm --dir frontend test`
- `pnpm --dir frontend build`
- `git diff --check`
- `bash -n scripts/demo-governance-loop.sh ... scripts/demo-sprint10-route-policy-retry.sh`
- `scripts/demo-sprint10-route-policy-retry.sh` against local API on `127.0.0.1:9090`

### 待办
- 如需提交 PR，建议下一轮 review 聚焦 retry precedence、body cap trace 形态、PostgreSQL JSONB retry persistence、前端 policy retry UX。

### 影响文件
- `frontend/src/retryForm.ts`：新增 retry 表单解析/校验 helper。
- `frontend/tests/retryForm.test.mjs` / `frontend/package.json`：新增前端 regression test 和 test script。
- `frontend/src/App.tsx`：Agent / Policy 表单复用 retry helper。
- `internal/httpapi/server_test.go`：新增 cancel 不重试 regression test。

## [2026-05-30 22:20] Session: Sprint 10 Route Policy Retry Overrides

### 完成
- 新增 Sprint 10 brief / design / implementation plan，范围聚焦 RoutePolicy 级 retry override。
- `RoutePolicy` 新增可选 `retry`，包含 `maxAttempts`、`backoffMs`、`statusCodes`。
- `POST /api/v1/route-policies` 支持创建 retry override；`PATCH /api/v1/route-policies/{id}` 支持替换或清空 retry。
- `EvaluateRouteAccess` 在命中的 allow policy 上返回 retry override；deny policy 和 legacy access grant 不返回 override。
- 数据面 proxy retry 解析顺序改为 policy retry → target Agent `channelConfig.retry` → 默认不重试。
- PostgreSQL 新增 `006_sprint10_route_policy_retry.sql`，`route_policies.retry` 使用 JSONB 存储。
- 前端 Create Policy 新增 retry attempts/backoff 字段，Route Governance 表显示 policy retry summary。
- 新增 `scripts/demo-sprint10-route-policy-retry.sh`，覆盖 policy retry shape、非法参数拒绝和 `PATCH retry:null` 清空。

### 决策
- RoutePolicy retry 使用和 target Agent retry 相同的边界：`maxAttempts` 1-4、`backoffMs` 0-1000、`statusCodes` 只允许 5xx。
- 创建 policy 时 retry 对象内缺失字段会归一化：attempts=1、backoff=0、statusCodes=[502,503,504]。
- policy 没有 retry 时不改变既有行为，继续使用 target Agent retry；legacy access grant 不增加 retry override。

### 验证
- TDD red/green: policy retry 起初被 JSON unknown field 拒绝或不会影响 proxy → 现在 `maxAttempts=2` 可让 upstream 503 后第二次 202 成功。
- TDD red/green: route policy retry invalid attempts/status 起初不会被校验 → 现在 create/patch 返回 400。
- TDD red/green: PostgreSQL route policy retry 起初无类型/无字段 → 现在持久化并随 access decision 返回。
- `go test ./...`
- `npm --prefix frontend run build`
- `ADMIN_KEY=local-admin-key bash scripts/demo-sprint10-route-policy-retry.sh` against local API
- `go vet ./... && go build ./... && git diff --check`
- `AGENT_HARBOR_TEST_DATABASE_URL=... go test ./internal/store -run TestPostgresRepositoryRoundTrip -count=1`（临时 PostgreSQL 16 容器）
- Playwright smoke for Create Policy retry fields / Route Governance / Active Policies / Management Audit against live local backend + Vite dev server

### 影响文件
- `internal/domain/types.go`：新增 RoutePolicy retry 类型和 access decision retry。
- `internal/httpapi/server.go` / `server_test.go`：policy retry validation、patch parsing、proxy retry precedence、HTTP tests。
- `internal/store/memory.go` / `postgres.go` / `postgres_test.go`：policy retry persistence/evaluation。
- `internal/db/migrations/006_sprint10_route_policy_retry.sql`：新增 retry JSONB 列。
- `frontend/src/App.tsx` / `types.ts` / `data.ts`：Create Policy retry 字段和表格展示。
- `scripts/demo-sprint10-route-policy-retry.sh`、`README.md`、`docs/sprints/`、`docs/superpowers/`：Sprint 10 行为记录。

## [2026-05-30 21:58] Session: Sprint 9 Route Policy Objects

### 完成
- 新增 Sprint 9 brief / design / implementation plan，范围聚焦 route policy objects、优先级、显式 allow/deny 和 legacy grant fallback。
- 新增 `RoutePolicy` domain model、create/update request、`RouteAccessDecision`，并扩展 repository contract。
- 新增 `POST/GET/PATCH/DELETE /api/v1/route-policies` 管理接口；`DELETE` 语义为 disable，保留治理表稳定性。
- 数据面从 `HasGrant` 直查改为 `EvaluateRouteAccess`：enabled route policy 优先，按 priority 排序，deny tie-break，未命中时回退 access grant。
- Memory / PostgreSQL repository 支持 route policy CRUD、排序评估、scope listing；新增 `005_sprint9_route_policies.sql`。
- 管理审计新增 `route_policy.created`、`route_policy.updated`、`route_policy.disabled`，metadata 只记录 caller/target/route/effect/status/priority。
- 根据 review 修复：RoutePolicy 创建拒绝跨 tenant/workspace caller-target 组合，数据面评估也会忽略直接写入的跨 scope policy，避免 target 侧治理盲区。
- 根据二轮 review 修复：`POST /route-policies` 使用 `*int` 区分 omitted priority 和显式 `0`，避免把最低优先级策略静默提升为默认 100。
- 前端 Route Governance 改为直接读写 `/route-policies`，Create Grant 改为 Create Policy，表格动作改为 Disable policy。
- 新增 `scripts/demo-sprint9-route-policies.sh`，覆盖无策略拒绝、allow 放行、高优先级 deny 拦截、降低 deny priority 后 allow 胜出、disable allow 后 deny 生效。

### 决策
- RoutePolicy scope 使用 caller Agent 的 `tenantId/workspaceId`，便于控制面按发起方治理归属过滤。
- 不迁移已有 access grants；Sprint 9 保持 legacy fallback，避免破坏之前 demo 和现有数据面调用。
- Sprint 9 暂不支持跨 tenant/workspace route policy；如果后续需要跨 workspace/tenant 路由，要显式建模 source/target scope 和双方可见性。
- 策略排序采用 higher priority wins；同优先级 deny wins；再按 createdAt / id 固定顺序，保证 memory 和 PostgreSQL 行为一致。

### 血泪教训
- 本次一开始用相对路径写 Sprint 9 文档，误落到旧 `ai-nexus` 工作区；已删除误落文件并用绝对路径补回 AgentHarbor。后续跨仓库上下文里新增文件必须优先用绝对路径确认目标。

### 验证
- TDD red/green: `/api/v1/route-policies` 404 → CRUD + audit 通过。
- TDD red/green: legacy grant 先 allow → 高优先级 deny policy 拦截 → 更高优先级 allow policy 放行 → disable allow 后 deny 生效。
- Review fix red/green: 跨 scope policy 起初可创建并在数据面生效 → 现在 API 返回 400，直接写 repo 的跨 scope policy 也不会匹配。
- Review fix red/green: 显式 `priority: 0` 起初会被 create 默认成 100 → 现在保存为 0，`priority=50` 的 deny 可以正确压过 `priority=0` 的 allow。
- `go test ./...`
- `npm --prefix frontend run build`
- `ADMIN_KEY=local-admin-key bash scripts/demo-sprint9-route-policies.sh` against local API
- `go vet ./... && go build ./... && git diff --check`
- `AGENT_HARBOR_TEST_DATABASE_URL=... go test ./internal/store -run TestPostgresRepositoryRoundTrip -count=1`（临时 PostgreSQL 16 容器）
- Playwright smoke for Create Policy / Route Governance / Active Policies / Management Audit against live local backend + Vite dev server

### 影响文件
- `internal/domain/types.go`：新增 route policy 和 route access decision 类型。
- `internal/httpapi/server.go` / `server_test.go`：route policy API、数据面鉴权切换、HTTP tests。
- `internal/store/memory.go` / `postgres.go` / `postgres_test.go`：route policy persistence/evaluation 和 PostgreSQL round-trip。
- `internal/db/migrations/005_sprint9_route_policies.sql`：新增 route_policies 表和索引。
- `frontend/src/App.tsx` / `api.ts` / `types.ts` / `data.ts`：Route Governance 读写 policy API。
- `scripts/demo-sprint9-route-policies.sh`、`README.md`、`docs/sprints/`、`docs/superpowers/`：Sprint 9 行为记录。

## [2026-05-30 21:10] Session: Sprint 8 Management Audit and Credential Versions

### 完成
- 新增 Sprint 8 brief / design / implementation plan，范围聚焦 management audit events 和 Agent credentialVersion。
- `Agent` 响应新增 `credentialVersion`：无凭据为 0，创建时带凭据为 1，credential rotation 后递增。
- 新增 `AuditEvent` domain model，Memory / PostgreSQL repository 支持 append/list/filter。
- 新增 `GET /api/v1/audit/events?tenantId=&workspaceId=&action=&resourceType=&resourceId=` 管理审计查询接口。
- Agent create/update/disable、credential rotate、Agent Key create/revoke、Access Grant create/revoke 均写入管理审计事件。
- PostgreSQL 新增 `004_sprint8_management_audit.sql`，包含 `agents.credential_version` 和 `audit_events` 表与索引。
- 新增 `scripts/demo-sprint8-management-audit.sh`，验证 create/patch/rotate 的 credentialVersion 和审计事件不泄露 secret values。
- 根据 review 修复：已有 PostgreSQL credential rows backfill 到 version 1、rotation 版本递增下沉为 repository 原子更新、空 credential rotation 拒绝、audit listing 加 `limit`、前端 Disable 改走 DELETE 以记录 `agent.disabled`。

### 决策
- Sprint 8 只记录小型、无 secret 的 metadata，例如 credential key names 和 credentialVersion，不记录 credential values 或 Agent Key plaintext。
- 控制面审计先使用 best-effort append，避免已提交副作用后因为 audit 写入失败让 API 返回 500；事务化 outbox 留到后续迭代。
- Grant 审计事件使用 caller Agent 的 tenant/workspace 作为 scope，metadata 内记录 target Agent。

### 验证
- TDD red/green: `credentialVersion` 初始为 0 → credentialed Agent create 返回 1、rotation 返回 2。
- TDD red/green: `/api/v1/audit/events` 404 → lifecycle 审计事件和过滤通过。
- Review fix red/green: `limit=1` 起初被忽略 → audit event list 现在按 limit 截断。
- Review fix red/green: 空 credential rotation 起初可让无凭据 Agent 版本变为 1 → 现在返回 400 且版本保持 0。
- `go test ./internal/httpapi -run 'TestManagementAuditEvents|TestRotateAgentCredentials' -count=1`
- `go test ./internal/store -count=1`
- `go test ./internal/httpapi ./internal/store ./internal/db -count=1`
- `go test -count=1 ./...`
- `go vet ./... && go build ./... && git diff --check`
- `npm --prefix frontend run build`
- `scripts/demo-sprint8-management-audit.sh` against local API with `AGENT_HARBOR_ADMIN_KEY`
- `AGENT_HARBOR_TEST_DATABASE_URL=... go test ./internal/store -run TestPostgresRepositoryRoundTrip -count=1`（临时 PostgreSQL 16 容器）
- Playwright smoke for Management Audit table against live local API

### 影响文件
- `internal/domain/types.go`：新增 `CredentialVersion` 和 `AuditEvent`。
- `internal/httpapi/server.go` / `server_test.go`：管理审计写入、查询 endpoint、credentialVersion 行为测试。
- `internal/store/memory.go` / `postgres.go` / `postgres_test.go`：AuditEvent persistence/filtering 和 credential_version round-trip。
- `internal/db/migrations/004_sprint8_management_audit.sql`：PostgreSQL schema。
- `scripts/demo-sprint8-management-audit.sh`、`README.md`、`docs/sprints/`、`docs/superpowers/`：Sprint 8 行为记录。

## [2026-05-30 19:40] Session: Sprint 7 Agent Update and Credential Rotation

### 完成
- 新增 Sprint 7 brief / design / implementation plan，范围聚焦 Agent partial update 和 credential rotation。
- 新增 `PATCH /api/v1/agents/{id}`，支持更新 name / description / ownerId / status / full `channelConfig` replacement。
- 新增 `POST /api/v1/agents/{id}/credentials:rotate`，完整替换 Agent credential bag，响应继续不回显明文。
- 更新与轮换复用 create-time 校验：status、channel catalog、endpoint required、SSRF、headers、credentialHeaders、timeout、retry。
- Memory / PostgreSQL repository 新增 `UpdateAgent`，PostgreSQL 继续用 `credentials_ciphertext` AES-GCM 路径保存轮换后的密钥。
- 前端新增 credential rotation 表单，并把 Registry 状态动作切到 `PATCH`。
- 根据 review 修复 `channelConfig` secret-like key 在 array nesting 中可绕过并被回显的问题。
- 新增 `scripts/demo-sprint7-credential-rotation.sh`，验证 patch、rotate、get 响应均不泄露 credential。

### 决策
- Sprint 7 不支持 channel type / tenant / workspace 迁移，避免 update API 变成隐式 re-register。
- `channelConfig` 更新采用 full replacement，不做 deep merge，避免 nested headers / credentialHeaders 语义歧义。
- Credential rotation 是 full replacement，不做 per-key merge；缺失 `credentialHeaders` 引用会直接拒绝。

### 验证
- TDD red/green: `PATCH /api/v1/agents/{id}` 405 → mutable fields update + unsafe config rejection 通过。
- TDD red/green: rotate endpoint 404 → 下一次 proxy 使用新 Authorization 通过。
- Review fix red/green: array-nested `authorization` in `channelConfig.metadata` 先泄露回显 → 现在拒绝。
- `go test ./internal/httpapi ./internal/store -count=1`
- `pnpm -C frontend build`
- `go test -count=1 ./...`
- `go vet ./...`
- `go build ./...`
- `AGENT_HARBOR_TEST_DATABASE_URL=... go test ./internal/store -run TestPostgresRepositoryRoundTrip -count=1`（临时 PostgreSQL 16 容器）
- `scripts/demo-sprint7-credential-rotation.sh` against local API with `AGENT_HARBOR_ADMIN_KEY`
- Playwright smoke for Rotate Credential / Agent Registry against live local API

### 影响文件
- `internal/httpapi/server.go` / `server_test.go`：update/rotate handlers、共享 Agent validation、HTTP tests。
- `internal/store/memory.go` / `postgres.go` / `postgres_test.go`：Agent update persistence 和 encrypted credential round-trip。
- `internal/domain/types.go`：新增 update/rotate request types。
- `frontend/src/api.ts` / `types.ts` / `App.tsx` / `styles.css`：update/rotate API client 和控制台操作入口。
- `scripts/demo-sprint7-credential-rotation.sh`、`README.md`、`docs/sprints/`、`docs/superpowers/`：Sprint 7 行为记录。

## [2026-05-30 18:45] Session: Sprint 6 Runtime Metrics

### 完成
- 新增 Sprint 6 brief / design / implementation plan，范围聚焦 Runtime Signals 从 mock 转真实后端指标。
- `TraceEvent` 新增 `durationMs`、`upstreamAttempts`、`upstreamStatus`、`upstreamError`，用于记录数据面 proxy 结果。
- PostgreSQL 新增 `003_sprint6_runtime_metrics.sql`，memory/PostgreSQL trace 读写路径均保留新增指标字段。
- MCP/OpenAPI proxied calls 在写响应前记录 allowed trace，包含最终 attempts、upstream status、gateway upstream error code 和 duration。
- 新增 `GET /api/v1/metrics/runtime?tenantId=&workspaceId=`，按 scoped traces 聚合 gateway calls、allowed rate、upstream error rate、avg latency。
- 前端 Signal Board 改为优先读取 runtime metrics API，只有网络不可达时才回落本地 sample metrics。
- 根据 review 修复 proxied upstream 已成功后 trace 写入失败会覆盖成 500 的风险；此时保留 upstream response，trace 写入退化为 best-effort。
- 收窄前端 fallback，只对浏览器 fetch 网络失败回落 sample data，不再吞所有 `TypeError`。
- 新增 `scripts/demo-sprint6-runtime-metrics.sh`，验证 denied + allowed 数据面调用会反映到 runtime metrics。

### 决策
- Sprint 6 先做 trace-derived metrics API，不引入外部 OpenTelemetry exporter；同一批字段后续可直接映射到 spans/counters。
- Trace storage 保持 append-only，不新增 update trace 接口；proxied allowed trace 改为 proxy outcome 已知后写入。
- Metrics 聚合先放在 HTTP 层内存计算，等 trace 量增长后再加 time window 和 SQL 聚合。

### 验证
- TDD red/green: runtime metrics endpoint 404 → scoped metrics 聚合通过。
- TDD red/green: proxied trace metrics fields 为空 → attempts/status/duration 写入通过。
- `go test ./internal/httpapi -run 'TestRuntimeMetricsSummarizeDataPlaneTraces|TestProxyTraceMetricsRecordAttemptsStatusAndDuration' -count=1`
- `go test ./internal/store -run TestPostgresRepositoryRoundTrip -count=1`
- `pnpm -C frontend build`
- Review fix targeted test: `TestProxySuccessDoesNotFailWhenTraceAppendFails`

### 影响文件
- `internal/httpapi/server.go` / `server_test.go`：proxy outcome trace recording、runtime metrics endpoint 和聚合测试。
- `internal/domain/types.go`：TraceEvent 指标字段和 SystemMetric 响应形状。
- `internal/store/postgres.go` / `internal/db/migrations/003_sprint6_runtime_metrics.sql` / `postgres_test.go`：trace metric persistence。
- `frontend/src/api.ts` / `App.tsx` / `types.ts`：Runtime Signal API 接入和 trace 字段类型。
- `scripts/demo-sprint6-runtime-metrics.sh`、`README.md`、`docs/sprints/`、`docs/superpowers/`：Sprint 6 行为记录。

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
