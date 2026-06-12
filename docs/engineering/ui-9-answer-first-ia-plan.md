# ui-9(P1)— 信息架构倒置:从对象管理到"问答 + 变更"双入口

> **For agentic workers:** 本计划使用 checkbox(`- [ ]`)逐任务跟踪。每个任务完成后运行验证命令再勾选。任务之间有依赖顺序,按编号执行。

- **日期**:2026-06-11
- **基线**:`codex/ui-8-first-run-journey`(PR #75 之后;若已合并则基于 main)
- **来源**:[ui-review-2026-06-11.md](ui-review-2026-06-11.md) 的后续 finding ui-9。ui-8([ui-8-first-run-journey-plan.md](ui-8-first-run-journey-plan.md))解决了"第一次打开不知道干什么";本计划解决"过了 onboarding 之后,每个页面仍是对象管理器"。

## 问题定义

README 给产品的定义是一个**问题**:"哪个租户、工作区、调用方实例和主体可以访问哪些工具或数据范围,为什么,以及对应的审批和审计证据是什么。"

但 UI 的信息架构是**对象管理**:Agent 表、能力表、策略表、路由表,按后端资源一表一页。用户带着问题来("这个 Agent 为什么调不通"/"给它开个权限"),必须自己把问题翻译成跨四五个页面的对象操作序列,且每个页面都不知道自己在整个任务里的位置。这就是"打开之后不知道怎么用"的根因——ui-8 的 checklist 教会了第一次,但教不会日常。

**关键观察:任务型产品的零件已经全部存在,只是层级摆反了。**

- **问答机已存在**:决策解释(`fetchAccessDecisionExplanation`,`frontend/src/api.ts:410`)输入五元组(tenant/workspace/callerInstance/target/capability),返回 `outcome` + `summary` + 分层 `evidence[]`(每层 `layer/status/id/message`)+ `nextActions`。但它埋在权限画像页底部,是个三级功能。
- **跨页向导已存在**:权限变更工作台从意图文本生成草案 → 预览 allow/deny → 预检 → 审批 → 应用,一条流走完建链全程。但它与"查询"没有打通——查到 deny 之后,用户得自己记住五元组,手动去工作台重新选一遍。
- **跨页带上下文机制已存在**:`AccessProfileHandoffContext`(`frontend/src/types.ts:399`)已经在"权限变更完成 → 查看画像"方向使用。缺的是反方向:"查询发现断点 → 带上下文发起变更"。

**本计划做的事,一句话:把"查 → 看见断点 → 一键带上下文去修"接成闭环,并把导航语义从对象改成任务。**

**本计划不做的事**(防止范围膨胀):

- 不改后端 API、不加新端点。决策解释、权限包、画像接口全部够用。
- 不删任何现有页面、不改任何 NavKey 的 hash 语义。配置区四页(注册表/能力/策略/路由)原样保留,只调整导航分组的呈现语义。
- 不重做权限变更工作台内部——只给它加"受控预填入口"。
- 不做自由文本搜索/NLP 查询。问答 = 结构化选择器五元组,本计划范围内不扩大。

---

## 目标态设计

### A. 新增"访问查询"视图(answer-first 首页)

新 `NavKey: "ask"`,这是配置完成系统的**新默认落地页**(取代 ai-admin;ui-8 的 `resolveDefaultNavKey` 逻辑顺延:未配置完 → getting-started,配置完 → ask)。

页面结构(从上到下):

1. **问题区**:一行式选择器组合,读作一句话——"〔调用方实例〕能否访问〔目标〕的〔能力〕?"外加 tenant/workspace 前置选择器和可选 subjectId。选择器复用权限变更工作台的业务选择器呈现(显示业务名,技术 ID 走 `TechnicalId` 折叠),数据源用已有的 `ConsoleData`(agents/capabilities/tenants),**不新增加载逻辑**。
2. **回答区**:调用既有 `fetchAccessDecisionExplanation`,渲染:
   - 结论横幅:allowed(绿)/ denied(红)+ `summary`。
   - **授权链可视化**:把 `evidence[]` 按层渲染成一条链(能力审批 → 租户授权 → 工作区分配 → 实例分配 → 数据范围),每环用现有 Badge 三态(通过/断裂/未检查),断裂环高亮并显示该层 `message`。这是 ui-8 链路图的"实例化版本":ui-8 画的是概念链,这里画的是**这一次查询的真实链**。
   - `nextActions` 渲染为列表。
3. **修复入口(本计划的核心交付)**:当 `outcome === "denied"` 时,回答区出现主按钮"发起权限变更修复"——把当前五元组转换为 `PermissionPackageDraftInput` 预填(tenantId/workspaceId/callerInstanceId/targetId 直接映射,requestText 自动生成一句意图文本如"为〔调用方〕开通〔目标〕的〔能力〕访问"),跳转 `#ai-admin` 且工作台表单已填好。用户点一下就站在了变更旅程的起点。
4. **查询历史(轻量)**:本 session 内最近 5 次查询(内存 state,不持久化),点击回填问题区。方便"改完回来再查一遍"的验证动作。

空状态(未发起任何查询时)给一行引导文案 + 示例查询按钮(用 `ConsoleData` 里第一个 active caller + 第一个 capability 组装,数据不足时降级为指向 getting-started)。

### B. 查询 → 变更的上下文交接

仿照既有 `AccessProfileHandoffContext` 模式,新增 `PermissionChangeHandoffContext`(类型放 `frontend/src/types.ts`):

```ts
interface PermissionChangeHandoffContext {
  tenantId: string
  workspaceId: string
  callerInstanceId?: string
  targetId?: string
  capabilityId?: string
  subjectId?: string
  intentText?: string   // 自动生成的意图句,用户可改
  sourceView: "ask"     // 预留扩展
}
```

交接规则:

- 控制器持有这个 context(一个 state,允许;它是跨视图交接的最小必要状态),"发起权限变更修复"按钮 set context + 跳 `#ai-admin`。
- 工作台挂载/接收到 context 时,把字段灌进 `PermissionPackageDraftInput` 表单,然后**立即清空 context**(一次性消费,防止用户后续手动进工作台时被旧 context 污染——这是既有 handoff 机制踩过的坑,见 git log `a1fecb8 fix: pin access profile handoff context`)。
- 预填后不自动提交、不自动生成草案——用户看一眼、可改,自己点"生成草案"。预填是减少抄写,不是代替决策。
- capabilityId 在 DraftInput 里没有直接字段(草案按 template 生成),处理:若五元组的 capability 能反查到某个 template 覆盖它,预选该 template;反查不到则只填四元组 + intentText 里写明能力名,让既有的意图生成路径接手。**不要为此扩展 DraftInput 结构**。

### C. 反向链接:对象页 → 查询页

授权链断在哪一环,对象页就是修哪一环的地方;反过来,对象页的行应该能"以此为条件去查询"。最小集(只做这三处,防膨胀):

| 位置 | 动作 |
|------|------|
| 注册表 Agent 行详情侧栏 | 链接"查询此调用方的访问" → 跳 ask,预填 callerInstanceId(active caller)或 targetId(mcp target) |
| 能力表行详情侧栏 | 链接"查询谁能访问此能力" → 跳 ask,预填 targetId + capabilityId |
| 权限画像页决策解释区 | 该区**整体迁移**到 ask 视图,画像页原位置放一行链接"访问判定已移至「访问查询」"(过渡期保留一个版本,避免书签断链) |

实现统一走 B 的 handoff context 机制(新增 `AskHandoffContext` 或复用同一结构加 `sourceView` 区分),不发明第二套传参。

### D. 导航语义改成任务动词

ui-8 已把分组排成"开始使用 / 接入配置 / 权限运营 / 审计与证据"。本计划在此之上:

```text
开始使用(onboarding,不变)
── 查与改(原 primary 组,改名)
   访问查询          ← 新增,组内第一位,配置完成后的默认页
   权限变更
   权限画像
── 审计与证据(audit,不变)
   运行审计 / 上线证据 / 系统自检
── 资源清单(原 configuration 组改名,整组后移到最末)
   Agent 与工具 / 工具能力 / 访问策略 / 路由规则
```

说明:

- 配置组从第二位移到最末并改名"资源清单"——它的新定位是 escape hatch / 库存盘点,日常任务从"查与改"走。ui-8 把配置组提前是为了 onboarding 阶段的链路顺序;现在 getting-started 的 checklist 行动按钮直接深链各页,不依赖导航顺序,所以这次后移不损伤 onboarding(需同步检查 getting-started 各步 targetHash 不受影响——它们是 hash 深链,本就不受导航顺序影响)。
- `navDetail.*` 文案逐条改成动词句:"查一个调用方为什么能/不能访问"(ask)、"申请并应用权限变更"(ai-admin)、"按租户盘点生效中的权限"(access)等。EN + zh-CN 成对。
- NavKey/hash 一概不动,纯分组与文案调整。

---

## 实施任务

### Task 1: 交接 context 与查询状态模块

- [x] `frontend/src/types.ts`:新增 `PermissionChangeHandoffContext` 与 `AskHandoffContext` 类型(或统一结构 + sourceView 判别,实现者按代码现状选更贴的方案,在 PR 描述说明取舍)。
- [x] 新建 `frontend/src/askJourney.ts` 纯函数模块:
  - `buildExplainRequest(...)`: 选择器 state → `AccessDecisionExplainRequest`(含完整性校验,复用 `accessDecisionExplainRequestComplete` 的语义)。
  - `buildPermissionChangeHandoff(request, consoleData)`: 查询五元组 → `PermissionChangeHandoffContext`,含 template 反查逻辑与 intentText 生成(EN/zh-CN 由调用方传 translator 生成,模块本身不持有语言)。
  - `evidenceChainRows(result)`: `AccessDecisionExplainResult` → 渲染用的链路行数组(层名 key / 三态 / message / 是否断裂环)。
- [x] 新建 `frontend/tests/askJourney.test.mjs`:覆盖完整/不完整请求、allowed/denied 证据链、断裂环定位、template 反查命中与未命中、handoff 字段映射。
- [x] 验证:`pnpm --dir frontend test`。

### Task 2: 访问查询视图

- [x] 新建 `frontend/src/components/AskAccessView.tsx`:目标态 A 的四个区。回答区证据链视觉复用 getting-started 链路图与权限旅程步骤条的既有样式语言,不新发明视觉体系;新增样式进 `styles.css` 沿用现有 token。
- [x] 新建 `frontend/src/hooks/useAskAccessController.ts`:选择器 state、explain 调用(loading/error/结果)、查询历史(最多 5 条,内存)、handoff 消费与产出。**业务判定全部 import 自 askJourney.ts,hook 只做 state 编排**——遵循既有 `useAccessProfileController` 的分层。
- [x] `consoleNavigation.ts`:注册 `"ask"` NavKey、视图映射、导航项(进 primary 组首位)。
- [x] `ConsoleController.tsx`:挂载视图 + hook,接 handoff context 的 set/consume。预算:净增 ≤ 50 行(view 分支 + hook 接线 + context 一个 state;选择器/历史/调用编排都必须在 hook 里)。
- [x] i18n:`ask.*` 全部 key,EN + zh-CN 成对;`message.*` 沿用 key+params 机制。
- [x] 新建 `frontend/tests/askAccessView.test.mjs`:denied 时修复按钮出现且 handoff 字段正确、allowed 时无修复按钮、空状态示例查询、sample 模式提示(explain 需要 live API,复用 `message.accessDecisionExplainRequiresLiveApi` 的既有处理)。
- [x] 验证:`pnpm --dir frontend test && pnpm --dir frontend build`。

### Task 3: 工作台受控预填

- [x] `ConsoleController.tsx` / `AiAdminPermissionWorkbench.tsx`:接收 `PermissionChangeHandoffContext`,灌表单、一次性消费清空。预填后表单处于普通可编辑态,不触发任何自动提交/自动草案。
- [x] 顶部给一条可关闭的上下文提示条:"已从访问查询带入:〔调用方〕→〔目标〕"(复用既有 handoff 提示的样式,参照画像页 `accessProfileAdjustScopeDetail` 那套)。
- [x] 新增测试:handoff 灌入字段正确、消费后再次进入工作台无残留、用户手动改预填值不被覆盖。
- [x] 验证:`pnpm --dir frontend test`。

### Task 4: 默认路由、导航重排与反向链接

- [x] `gettingStarted.ts` 的 `resolveDefaultNavKey`:配置完成分支从 `"ai-admin"` 改为 `"ask"`。同步更新其测试断言。
- [x] `consoleNavigation.ts`:按目标态 D 调整分组顺序与组名;`navDetail.*` 文案改任务动词句(EN + zh-CN)。更新导航相关测试断言。
- [x] 反向链接三处接线(目标态 C 表格):注册表详情侧栏、能力表详情侧栏、画像页决策解释区迁移 + 原位链接。
- [x] 画像页迁移后,`useAccessProfileController` 中 explain 相关 state 若不再被画像页使用,随迁移挪到 `useAskAccessController`,不留两份。
- [x] 验证:`pnpm --dir frontend test`。

### Task 5: 全量验证与文档

- [x] `pnpm --dir frontend test`(全绿)+ `pnpm --dir frontend build` + `make check` + `make release-check`。
- [x] 浏览器实测三条路径并记录:
  - (a) 配置完成环境(跑完 `scripts/scenario-permission-package-approval.sh`)打开 → 默认落"访问查询" → 查一个 denied 组合 → 链路断裂环高亮 → 点修复 → 工作台预填正确 → 走完变更 → 回 ask 重查同一组合变 allowed。**这条全链路是本计划的验收主线。**
  - (b) 全新 `make demo` 环境 → 仍默认落 getting-started,checklist 不受导航重排影响。
  - (c) 带 `#registry` 等显式 hash 直达,不被重定向。
- [x] `docs/engineering/ui-review-2026-06-11.md`:追加 ui-9 条目与闭合记录回链本计划。
- [x] `README.md`:控制台介绍处补一句双入口("先查后改"),EN + zh-CN。
- [x] `CHANGELOG.md` 追加记录。

**Task 5 evidence (2026-06-11):**

- Gates: `pnpm --dir frontend test`(177/177), `pnpm --dir frontend build`, `make check`, `make release-check` passed locally. Vite retained the existing >500 kB chunk-size warning.
- Browser path (a): isolated configured stack `API 9093 / MCP 8793 / Web 5176`; no-hash landing resolved to `#ask`; `update_ticket` query returned denied with `能力审批 / not_approved`; repair handoff entered `#ai-admin` with caller, target, template, and notice prefilled; approval, approve, and apply completed in UI; re-query returned allowed with all grant-chain layers matched.
- Browser path (b): isolated `make demo` stack `API 9095 / MCP 8795 / Web 5178`; no-hash landing stayed `#getting-started` and checklist remained visible. Current full data-source badge still shows sample fallback because catalog/metrics/policy compatibility endpoints are outside ui-9, but setup routing no longer depends on those non-critical endpoints.
- Browser path (c): `http://127.0.0.1:5176/#registry` stayed on `#registry` and rendered Agent Registry; no default-route redirect occurred.

---

## 验收标准

1. 配置完成的系统打开,首屏是"我能问问题"而不是任何对象表;问完一个 denied,**两次点击内**站在预填好的变更旅程起点(一次点修复、一次确认生成草案)。
2. 授权链断裂环在回答区可视化定位,不需要用户去画像页或对象页人肉排查。
3. 验收主线(Task 5a)全链路走通:查 denied → 修复 → 重查 allowed,全程不需要手动抄写任何 ID。
4. ui-8 的 getting-started 行为零回归;显式 hash 深链零回归;既有权限变更旅程在不带 handoff 时与现状完全一致。
5. 全部新增文案 EN + zh-CN,语言切换即时生效,无语言快照 state。
6. `ConsoleController.tsx` 净增 ≤ 50 行;explain 相关逻辑在画像页与 ask 页之间**不存在两份拷贝**。

## 红线(违反即返工)

- 不修改任何后端文件,不新增 API 端点。
- 不引入新依赖。链路可视化用现有 CSS 体系。
- 业务判定(请求构造/证据链转换/handoff 映射/template 反查)必须在 `askJourney.ts` 纯函数模块,hook 只编排 state,控制器只接线。
- handoff context 必须一次性消费,消费即清空;不允许残留导致后续进入工作台被污染。
- 不删、不改现有 NavKey 与 hash 语义;画像页决策解释区迁移必须留原位指路链接。
- i18n 成对添加,任一语言缺 key 视为未完成。
- 预填永不自动提交。任何写操作的触发必须是用户显式点击。
