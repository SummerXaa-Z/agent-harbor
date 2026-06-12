# 用户旅程 P1 修复报告

- **日期**: 2026-06-10
- **范围**: `user-journey-review-2026-06-10.md` 中 j-3 / j-6
- **目标旅程**: AI Admin 配置权限变更 → 提交审批 → 审批处理 → 应用 → 运行验证 → 上线就绪
- **结论**: j-3 审批角色感知、审批确认和拒绝理由已完成首轮闭环;j-6 主路径异步 busy guard 已统一。focused tests、前端全量测试、构建、`make check`、`make release-check` 和浏览器实看均已通过。

## 修复项

### j-3: 审批处理必须有角色感和防错

原问题: 提交审批后,同一页面直接出现"批准请求/拒绝请求",点击立即生效;界面没有说明当前审批人是谁,拒绝也不需要理由,用户很难理解这是另一个角色的审计动作。

本轮修复:

- 权限变更审批区新增审批人上下文,显示"当前审批人"和生产环境职责分离说明。
- 批准/拒绝按钮不再直接调用审批 API,改为打开确认面板。
- 批准默认带审批说明,该说明会写入审批审计记录。
- 拒绝必须填写拒绝理由;空理由会在确认面板内提示,不会触发 API。
- App 层 `rejectAiAdminApprovalRequest` 在网络调用前再次 `trim` 校验拒绝理由,防止绕过组件状态提交空理由。
- 待审批队列中的批准/拒绝也改为进入同一确认面板,保持主审批区和队列处理一致。

### j-6: 异步主操作必须有统一进行中保护

原问题: 运行验证约 2 秒内按钮没有统一禁用和进行中保护,用户容易重复点击;同类风险也出现在审批、应用、预检、上线检查和证据刷新等路径。

本轮修复:

- 新增统一 `permissionRequestBusy` guard,覆盖审批旅程运行中、审批创建/处理、应用权限、审批冷却、预检、上线检查、证据导出、权限判定、落地状态、影响复核和待审队列刷新。
- 主要会改变状态或刷新关键证据的按钮统一使用 `liveDataBlocked || permissionRequestBusy` 禁用。
- 保留只读信息展示,但避免在关键异步动作未完成时并发触发另一条关键动作。

## 用户体验变化

- 用户看到审批按钮时,会先知道"我正在以谁的身份审批"。
- 用户拒绝请求时,必须说明原因;原因会进入审计记录,便于申请人和安全人员复核。
- 用户在运行验证、应用权限、导出证据或刷新关键检查时,不会再通过另一处按钮重复触发主路径动作。
- 本地开发模式仍会提示开发审批身份,但不伪装成生产鉴权;生产环境的职责分离继续以后端配置和 API 校验为准。

## 自动化回归

已新增或扩展以下 focused tests:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs
pnpm --dir frontend exec node --test tests/i18n.test.mjs
```

覆盖内容:

- 审批区显示审批人上下文。
- 批准/拒绝先进入确认面板,不再一键直接调用处理函数。
- 拒绝理由为空时先提示,不调用拒绝 API。
- App 层拒绝函数在网络调用前校验 `comment?.trim()`。
- 主路径异步动作共享 `permissionRequestBusy`。
- 新增中英双语文案 key 保持一致。

## 浏览器实测

在当前本地页面 `http://127.0.0.1:5174/` 直接验证:

- 页面主审批区显示审批人上下文: `当前审批人：Security Reviewer` 和生产环境职责分离说明。
- 点击主审批区"批准请求"后只打开确认面板,面板内容包含 `确认批准`、`审批人：Security Reviewer`、`审批说明`、`该说明会写入审批审计记录。`;页面未直接出现新的"已批准"结果。
- 取消后点击"拒绝请求",确认面板显示 `确认拒绝`、`拒绝理由` 和 `用于审计和交接，不能为空。`。
- 不填写拒绝理由直接点"确认拒绝",面板内提示 `请先填写拒绝理由，再拒绝请求。`;页面未直接出现新的"已拒绝"结果。

## 已执行门禁

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs
pnpm --dir frontend exec node --test tests/i18n.test.mjs
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
make check
make release-check
```

全部通过。

## 未决边界

- j-4 旅程完成态仍未闭环:导出证据后还缺"本次变更已完成"终态和出口。
- j-5 消息降噪仍未闭环:加载类消息还需要收敛为低干扰提示。
- j-7 跨工作区上下文延续仍未闭环:权限变更到权限画像需要携带完整业务上下文。
- j-8 步骤指示器仍未闭环:五步流程当前还是展示型,还不能点击跳转。
- 后端已经支持审批职责分离和审批评论持久化;本轮没有新增后端 reject comment 必填校验,因为该策略需要兼容 MCP/API 调用方的迁移窗口,应作为后续 API 契约收紧项单独处理。
