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
- 前端不复用旧 AI Nexus `web/` 代码，全部放在独立 GitHub 仓库 `SummerXaa-Z/ai-nexus-go-rebirth`。
- 设计参考吸收 Ant Design Pro、Semi Design、shadcn dashboard blocks、Arco Design Pro 的通用中后台模式，但不复制模板代码。
- 后端在线时 catalog / agents / traces 走 Go runtime；route policies / evidence / runtime signals 暂用本地样例面板，状态栏写作 `Go runtime + samples`。
- GitHub 仓库保持 private，提交到 `main`，不再触碰公司 GitLab。

### 血泪教训
- `apply_patch` 没有 `workdir` 参数时会按会话默认目录写文件，曾误写到原 AI Nexus 仓库；后续跨仓库操作必须使用绝对路径。
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
