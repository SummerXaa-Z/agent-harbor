# ui-8(P1)— 首次使用旅程与导航 IA 重构实施计划

> **For agentic workers:** 本计划使用 checkbox(`- [ ]`)逐任务跟踪。每个任务完成后运行验证命令再勾选。任务之间有依赖顺序,按编号执行。

- **日期**:2026-06-11
- **基线分支**:`codex/production-readiness-gate`(PR #74 之后)
- **来源**:[ui-review-2026-06-11.md](ui-review-2026-06-11.md) 的后续 finding。该报告的 ui-1~ui-7 已闭合,本计划是 ui-8,优先级 P1。
- **关联**:ui-7(概念负担)当时只按文档可读性修了,本计划覆盖其 UI 侧的本意。

## 问题定义

**新用户打开控制台,不知道该干什么。** 具体三层:

1. **落地页假设系统已配置完毕。** 默认视图是"权限变更"工作台——一个五步旅程(请求 → 审批 → 应用 → 运行验证 → 上线就绪)。这个旅程要先有租户、Agent、已审批能力、授权链才跑得通。新用户面对的是空系统,看到的却是流程中段:模板下拉是空的、目标选择器是空的、每个面板都是空状态。用户的第一个问题"我现在该干什么"没有任何元素回答。
2. **导航是对象清单,不是任务地图。** 9 个导航项(权限变更 / 权限画像 / 上线证据 / 运行审计 / 系统自检 / Agent 与工具 / 工具能力 / 访问策略 / 路由规则)本质是把后端资源表翻译成菜单。README 写了核心链路 `Tenant → Agent → Capability → Entitlement → Assignment → Runtime decision`,但 UI 没有把这条链画出来。用户带着"我想让某个 Agent 能调某个工具"的任务进来,需要自己脑补要按什么顺序穿过哪几个菜单。
3. **空状态会"指物"但不会"指路"。** 现有空状态(`empty.registry.detail` 等)说明了"这里将来会有什么",部分说了"做什么能填上"(如 `empty.capabilities.detail`:"Refresh an MCP target…"),但都不带可点的动作,也不告诉用户"你卡在这一步是因为上一步没做"。授权链是有严格依赖顺序的,空状态应该顺着链往回指。

**本计划不做的事**(防止范围膨胀):

- 不改后端 API、不加新端点。所需的"系统配置进度"全部从现有 `ConsoleData`(`frontend/src/types.ts:365`)字段推导。
- 不重写既有视图组件的内部布局。只动:导航结构、默认路由逻辑、新增 onboarding 视图、空状态组件增强。
- 不动权限变更工作台的五步旅程本身——它对"已配置好的系统"是对的。

---

## 目标态设计

### A. 新增"开始使用"视图(setup checklist)

一个新 `NavKey: "getting-started"`,内容是按依赖顺序排列的 6 步 checklist。每步的完成判定**纯前端推导**,数据来自已有的 `loadConsoleData()` 返回值:

| # | 步骤 | 完成判定(从 ConsoleData 推导) | 行动按钮跳转 |
|---|------|-------------------------------|--------------|
| 1 | 连接后端 API | `loadedFromApi === true` | 文案引导 `make demo` / 配置 API 地址(复用现有连接设置入口) |
| 2 | 注册租户与 Agent | `tenants.length > 0 && agents.some(a => a.status === "active")` | `#registry` |
| 3 | 发现工具能力 | `capabilities.length > 0` | `#capabilities` |
| 4 | 建立授权链 | `tenantEntitlements.length > 0` | `#ai-admin`(权限变更是建链的正路) |
| 5 | 发起一次调用,看到 allow/deny | `traces.length > 0` | `#traces`(空态时文案指向 demo 脚本 `scripts/scenario-core-journey.sh`) |
| 6 | 查看上线证据 | `evidenceRuns.length > 0` 或权限包 readiness 为 ready | `#evidence` |

呈现要求:

- 每步一行:序号圈(完成态打勾)+ 标题 + 一句话说明 + 右侧行动按钮。视觉复用现有权限变更旅程的步骤条样式语言(`permissionRequestProcessStepStatuses` 那套完成/进行中/待办三态),不新发明视觉。
- 顶部一句话产品定位(一行,不是营销段落):"注册 Agent 与工具 → 通过权限包授权 → 用运行时证据验收",下面紧跟一张极简链路图。链路图用纯 CSS/flex 渲染 6 个节点(租户 → Agent → 能力 → 授权 → 运行时 → 证据),**不引入图片资源或图表库**。
- 全部文案走 i18n,中英双语,key 前缀 `gettingStarted.*`。
- sample 模式(`loadedFromApi === false`)下,步骤 1 显示未完成并给出醒目提示,其余步骤照常按 sample 数据判定但加"示例数据"角标——和现有 `message.fallbackDataModeDetail` 的处理保持一致。

### B. 默认路由按系统状态走

修改 `frontend/src/consoleNavigation.ts` 的默认视图逻辑:

- 当前:`defaultNavKey` 硬编码 `"ai-admin"`。
- 目标:导出 `resolveDefaultNavKey(data: ConsoleData): NavKey`——上面 checklist 的步骤 1–4 **任一未完成**则返回 `"getting-started"`,否则返回 `"ai-admin"`。
- URL hash 显式指定视图时永远尊重 hash,不做重定向(深链/书签/测试都不能被劫持)。
- 已配置好的系统里,"开始使用"仍保留在导航最末(audit 组之后单独成组或放 configuration 组首),变成"系统配置概览"用途;不要在配置完成后把它藏掉。

### C. 导航分组按用户旅程重排

保持现有三组结构,调整顺序、归属和组名语义,让组序 = 使用顺序:

```text
开始使用            ← 新增,组外置顶或独立组
── 接入配置(configuration,提前到第二位)
   Agent 与工具 / 工具能力 / 访问策略 / 路由规则
── 权限运营(primary,改名以反映"日常主任务")
   权限变更 / 权限画像
── 审计与证据(audit)
   运行审计 / 上线证据 / 系统自检
```

说明:

- "上线证据"从 primary 移到 audit 组——它是验收动作,使用者(安全/验收角色)和日常变更操作者不同。
- 组内顺序按链路依赖排(Agent 先于能力,能力先于策略)。
- 仅调整 `navItems` 的 `groupKey`/顺序与组标签 i18n 文案;`NavKey` 本身、hash 路由、各视图组件**一概不动**,把回归面压到最小。
- 现有测试若断言导航顺序,同步更新断言——这是预期变更,不是回归。

### D. 空状态学会"指路"

增强 `frontend/src/components/ui.tsx` 的 `EmptyRow`:加可选 props `actionLabel` + `onAction`(或 `actionHash`),渲染为空状态下方的次级按钮。然后给**链路依赖型**空状态接上动作:

| 空状态 | 现状 | 增加的动作 |
|--------|------|-----------|
| 工具能力为空(`empty.capabilities`) | 文字说"刷新 MCP 目标" | 按钮"去注册 Agent 与工具" → `#registry`(无目标时);有目标无能力时按钮改"刷新能力"(调用既有 `refreshTargetCapabilities` 入口) |
| 授权链为空(`empty.grantChains`) | 文字说明 | 按钮"发起权限变更" → `#ai-admin` |
| 运行审计为空(`empty.auditTraces`) | 文字说明 | 按钮"查看开始使用" → `#getting-started`(指向步骤 5 的调用引导) |
| 访问策略为空 | ui-5 已给创建引导 | 不动,已达标 |
| Agent 注册表为空(`empty.registry`) | 文字说明 | 按钮"查看开始使用" → `#getting-started` |

原则:动作永远指向**依赖链的上一步**,不指向泛泛的文档。纯过滤型空状态(`empty.filteredResults`)不加动作。

---

## 实施任务

### Task 1: 抽取 setup 进度推导模块

- [x] 新建 `frontend/src/gettingStarted.ts`:输入 `ConsoleData`,输出 `GettingStartedStep[]`(含 `key` / `done: boolean` / `targetHash`)与 `isSetupComplete(data): boolean`(步骤 1–4 全完成)。纯函数,不碰 React。
- [x] 新建 `frontend/tests/gettingStarted.test.mjs`:覆盖空系统(全未完成)、部分配置、sample 模式、全配置四种数据形态下的步骤判定与 `isSetupComplete`。
- [x] 验证:`pnpm --dir frontend test`。

### Task 2: 开始使用视图

- [x] 新建 `frontend/src/components/GettingStartedView.tsx`:消费 Task 1 的输出渲染 checklist + 链路图。行动按钮统一通过现有 hash 导航跳转(`navHashFor`),不引入新路由机制。
- [x] `frontend/src/i18n.ts` 增加 `gettingStarted.*` 全部 key,EN + zh-CN 成对添加。中文文案口语直接(参照现有 `navDetail.*` 的语气),不写营销话术。
- [x] `consoleNavigation.ts`:注册 `"getting-started"` NavKey、视图映射、导航项。
- [x] `ConsoleController.tsx`:挂载新视图。注意:**只加一个 view 分支与必要的 props 透传,不在控制器里新增业务 state**——setup 进度全部由 Task 1 的纯函数从既有 `consoleData` 现算。
- [x] 新建 `frontend/tests/gettingStartedView.test.mjs`:渲染层断言(完成态打勾、行动按钮 hash 正确、sample 模式角标)。
- [x] 验证:`pnpm --dir frontend test && pnpm --dir frontend build`。

### Task 3: 默认路由与导航重排

- [x] `consoleNavigation.ts`:实现 `resolveDefaultNavKey(data)`;调整 `navItems` 分组与顺序(见目标态 C);组标签 i18n 文案更新("权限运营"/"接入配置"/"审计与证据")。
- [x] `ConsoleController.tsx`:数据加载完成后、且 URL 无显式 hash 时,应用 `resolveDefaultNavKey`。有 hash 永远尊重 hash。注意避免数据二次加载时把用户已主动切换的视图弹回去——只在首次加载决策一次。
- [x] 更新受影响的现有测试断言(导航顺序/默认视图)。
- [x] 新增测试:空系统默认落 getting-started;配置完成系统默认落 ai-admin;带 hash 进入时不被重定向;首次加载后用户切换视图不被弹回。
- [x] 验证:`pnpm --dir frontend test`。

### Task 4: 空状态指路

- [x] `frontend/src/components/ui.tsx`:`EmptyRow` 增加可选动作 props,无动作时渲染与现状完全一致(零回归)。
- [x] 按目标态 D 的表格逐个接线,i18n 增加对应 `empty.*.action` key(EN + zh-CN)。
- [x] 更新/新增空状态相关测试。
- [x] 验证:`pnpm --dir frontend test`。

### Task 5: 全量验证与文档

- [x] `pnpm --dir frontend test`(全绿)+ `pnpm --dir frontend build` + `make check` + `make release-check`。
- [x] 浏览器实测两条路径并记录:(a) 全新 `make demo` 环境,确认默认落"开始使用",逐步完成 checklist,每步打勾、行动按钮可达;(b) 已配置环境(跑完 `scripts/scenario-permission-package-approval.sh` 后),确认默认落"权限变更",原有旅程无回归。
- [x] `docs/engineering/ui-review-2026-06-11.md`:在 finding 明细后追加 ui-8 条目与闭合状态回链本计划。
- [x] `README.md` "Try the Permission Changes Console" 一节开头加一句:首次打开会落在"开始使用"页(EN + zh-CN)。
- [x] `CHANGELOG.md` 追加记录。

实施边界:当前 `ConsoleData` 不包含权限包 production readiness 字段,因此第 6 步按本计划允许的 `evidenceRuns.length > 0` 判定;隔离 `make demo` 浏览器实测中,跑完权限包审批场景后前 5 步完成、第 6 步保持当前态。该边界已回写 ui-review 闭合记录,并需在 PR 描述中说明。

---

## 验收标准

1. 全新环境(`make demo` 起、零配置)打开控制台,**不需要读任何文档**就能看到:我在哪、系统是什么链路、第一步做什么、按钮在哪。
2. checklist 六步全部可通过 UI 内的行动按钮逐步推进(步骤 5 的"发起调用"允许指向脚本说明,因为调用本身发生在控制台之外)。
3. 配置完成的系统打开,默认仍是权限变更工作台,既有 9 个视图、五步旅程、所有现有测试零回归。
4. 带 `#traces` 等显式 hash 打开,永远直达对应视图,不被重定向。
5. 全部新增文案中英双语,切换语言即时生效(沿用 ui-2 的 key+params 机制,不允许出现新的语言快照 state)。
6. 不新增后端代码;`ConsoleController.tsx` 行数不增加超过 60 行(新增 view 接线的合理上限——本计划不是给状态单体继续加码的理由)。

## 红线(违反即返工)

- 不引入新依赖(图表库、路由库、状态管理库)。链路图用现有 CSS 体系画。
- 不把 setup 判定逻辑写进 `ConsoleController.tsx`,必须留在 `gettingStarted.ts` 纯函数里。
- 不修改任何后端文件。
- 不删、不改现有 NavKey 与 hash 语义(书签兼容)。
- i18n 成对添加,任一语言缺 key 视为未完成(现有 i18n 测试会抓)。
