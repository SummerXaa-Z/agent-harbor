# 用户旅程完成态与消息降噪修复报告

- **日期**: 2026-06-10
- **范围**: `user-journey-review-2026-06-10.md` 中 j-4 / j-5
- **目标旅程**: AI Admin 配置权限变更 → 审批 → 应用 → 运行验证 → 上线就绪 → 完成交接
- **结论**: j-4 旅程完成态和三个出口已完成首轮闭环;j-5 高级区成功加载消息已降噪。focused tests、前端全量测试、构建、`make check` 和 `make release-check` 已通过。

## 修复项

### j-4: 旅程必须有明确终点

原问题: 权限变更达到生产 ready 后,页面仍只提示"导出证据",没有告诉用户本次变更已经完成,也没有后续出口。

本轮修复:

- 工作台新增完成态区块 `approval-completion`,在生产 ready 后显示。
- 完成态使用生产 ready 信号收口: `productionReadiness.status === "ready"`、`workbenchPreview.summary.productionReady` 或 `productionSummary.status === "ready"` 任一成立即可进入完成态。
- 完成态标题为"本次权限变更已生效",详情说明权限已应用、运行验证和上线证据已完成。
- 完成时间使用 `productionReadiness.generatedAt`。
- 完成态提供三个出口:
  - `导出证据`: 复用现有上线证据 JSON 导出。
  - `查看权限画像`: 跳转到权限画像工作区,并携带当前租户、工作区、调用方、目标和能力上下文。
  - `新建权限变更`: 保留当前业务上下文,清空本次审批、应用、运行验证和证据状态。

### j-5: 结果消息不能堆成系统日志

原问题: 用户执行应用、检查、验证后,页面同时堆出多条"已加载/已通过"类内部状态消息,降低主路径可读性。

本轮修复:

- 新增 `shouldShowAdvancedStatusMessage`,高级检查区只展示 error/warning 消息。
- 以下成功加载类消息不再直接铺在高级区:
  - 环境检查已通过
  - 预检通过
  - 待审批请求已加载
  - 上线就绪状态已加载
  - 权限判定已加载
  - 落地状态已加载
  - 影响复核已加载
- 导出证据成功、导出证据失败和缺少实时 API 属于用户直接动作结果,改为写入主路径 `aiAdminMessage`,不会被高级区降噪吞掉。

## 自动化回归

已新增或扩展以下 focused tests:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs
pnpm --dir frontend exec node --test tests/i18n.test.mjs
```

覆盖内容:

- 完成态使用生产 ready 信号,而不只依赖运行验证 demo 状态。
- 完成态包含标题、详情、完成时间和三个出口。
- 查看权限画像会携带租户、工作区、调用方、目标和能力上下文。
- 开始新变更会清空当前变更的应用、上线就绪、审批队列、运行验证、影响复核和消息状态。
- 高级区成功加载消息被过滤,错误/阻断消息保留。
- 导出证据结果写入主路径消息。
- 新增中英双语 key 保持一致。

## 实测与门禁

已执行并通过:

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

浏览器实测:

- 在 `http://127.0.0.1:5174/` 当前权限变更页确认主流程出现"本次权限变更已生效"完成态,包含"权限已应用，运行验证和上线证据已完成。"、完成时间、`导出证据`、`查看权限画像`、`新建权限变更`。
- 点击 `查看权限画像` 后,页面切换到 `权限画像` 工作区,并带入当前调用方、目标和能力上下文,可直接查看最终权限与运行证据。
- 高级检查区不再铺出成功加载类消息,主路径仍保留与用户动作直接相关的证据导出结果。

## 未决边界

- j-2 撤回入口仍未闭环,需要后端取消/撤回审批请求的状态、权限边界和审计语义。
- j-7 已被"查看权限画像"出口部分缓解:完成态跳转会携带当前上下文;但普通左侧导航直接切到权限画像时,仍需要更完整的跨工作区上下文策略。
- j-8 步骤指示器仍未闭环:五步流程当前还是展示型,还不能点击跳转到对应区块。
