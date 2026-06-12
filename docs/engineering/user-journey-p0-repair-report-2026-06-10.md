# 用户旅程 P0 修复报告

- **日期**: 2026-06-10
- **范围**: `user-journey-review-2026-06-10.md` 中 j-1 / j-2 两个 critical finding
- **目标旅程**: AI Admin 配置权限变更 → 提交审批 → 审批 → 应用 → 运行验证 → 上线就绪
- **结论**: j-1 已闭环; j-2 的空申请、重复 pending、双击提交、双击后误触批准/拒绝均已闭环。撤回入口仍依赖后端取消语义,未在本轮伪造 UI。

## 修复项

### j-1: 运行验证不得覆盖用户草稿

原问题: 执行运行验证后,用户手输的"管理员需求"会被英文种子文案覆盖,租户/工作区上下文也会被验证流程切走。

本轮修复:

- `runAiAdminApprovalJourney` 改用独立 `validationForm`,不再把验证种子写回 `aiAdminForm`。
- 运行验证函数内移除 `setScope(...)` 和 `setAccessFilters(...)`,不再改当前租户/工作区上下文或权限画像筛选。
- 验证用审批请求和访问画像落到独立状态 `aiAdminApprovalJourneyApprovalRequest` / `aiAdminApprovalJourneyAccessProfile`,状态检查优先读取这份验证证据。
- 验证申请文案改为 i18n 注入,中文界面使用中文验证需求,不再泄漏英文种子申请。

### j-2: 提交审批防错

原问题: 双击"提交审批"可能产生重复 pending 审批请求;空需求也能提交;浏览器复测还发现一个更细的风险:提交成功后按钮切换,同一次双击的第二下可能落到"批准请求"。

本轮修复:

- `createAiAdminApprovalRequest` 增加同步 `approvalCreateInFlightRef` 闸门,进入网络请求前立即阻断并发提交。
- 提交前拦截空"管理员需求",中文提示为"请先填写管理员需求，再提交审批。"。
- 若当前草稿已有匹配的 pending 审批请求,不再新建,中文提示为"当前申请已有待审批请求，请先查看或批准现有请求后再继续。"。
- 创建审批成功后启动短冷却期,冷却期间批准/拒绝按钮置灰,函数层也用 `approvalResolveBlockedRef` 阻断误触发的 approve/reject API。

## 浏览器实测

- j-1: 在 5174 本地页面输入唯一文本 `Codex保留草稿验证-1781094721570`,点击"执行运行验证";验证完成后,管理员需求仍为该唯一文本,当前上下文仍显示"客户服务中心 / 客户服务工作区 / 客服助手"。
- j-2: 在 5174 本地页面输入唯一文本 `Codex双击防护验证-1781094989579`,快速双击"提交审批";UI 显示创建成功,批准/拒绝按钮处于禁用态,未出现"已批准"消息。
- j-2 后端计数: 同一唯一文本最终只有 1 条记录,状态为 `pending`,id 为 `ppar_e28c83dbff064324c068512cfeb8b40f`。

## 自动化回归

- `frontend/tests/permissionJourneySafety.test.mjs`
  - 验证运行验证函数不包含 `setAiAdminForm` / `setScope` / `setAccessFilters`。
  - 验证创建审批请求前先执行 in-flight、空文本、已有 pending 三类 guard。
  - 验证创建审批成功后启动审批冷却,批准/拒绝函数先检查冷却闸门。
- `frontend/tests/i18n.test.mjs`
  - 锁定新增中文错误提示和中文验证申请文案。

## 已执行门禁

```bash
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
make check
make release-check
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true \
AGENT_HARBOR_BROWSER_GATE_API_PORT=9190 \
AGENT_HARBOR_BROWSER_GATE_FRONTEND_PORT=5184 \
MOCK_MCP_PORT=8797 \
make ai-admin-browser-journey
```

全部通过。

## 未决边界

- j-2 中"撤回入口"未闭环: 当前后端还没有审批请求 cancel/withdraw 状态、权限边界和审计事件,因此本轮没有做假撤回按钮。
- j-3 到 j-8 仍按原报告优先级推进: 审批角色感知、完成态、异步 pending 态统一、消息降噪、跨工作区上下文延续、步骤可点击。
