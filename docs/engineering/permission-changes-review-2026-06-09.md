# Permission Changes 未提交 Diff 审查报告

- **日期**:2026-06-09
- **分支**:`codex/production-readiness-gate`
- **范围**:工作树未提交改动(30 改动文件 +3933/-3076,外加 8 个未跟踪文件)
- **方法**:7 维度 fan-out 审查 + 对所有 blocker/high 结论做对抗式验证(每条独立 agent 尝试 refute,确认或修正 severity)
- **证据基线(实跑)**:`go vet ./...` 通过、`go test ./...` 全 `ok`、前端 `node --test` **71/71 pass**;当前 diff 不破坏编译与测试。

> 经对抗验证后,**没有 blocker 级问题**;有 **5 个确认成立的 high**(其中 3 个在安全维度)、若干 medium。下表 severity 为验证后修正值。

## 结论总览

| # | 维度 | 结论 | 最高 severity |
|---|------|------|---------------|
| 1 | UI 是否生产 B 端 | 主体已生产级,1 个 high 拖回 Demo 味 | high |
| 2 | 访问对象选择 | 模型正确,1 个 med 缺口(同 d6-2 根因) | medium |
| 3 | i18n / 技术 ID 泄漏 | 主路径干净,仅 2 处 low | low |
| 4 | App.tsx / styles.css 过大 | 确认债,须拆,已给最小拆分计划 | medium |
| 5 | "上线判断" 文案 | 须改,且存在术语混用 high | high |
| 6 | 安全 / 竞态 / 绕过 | 原子性扎实,3 个 high + 1 个 med | high |
| 7 | 测试覆盖 | 后端强,前端一半真一半 grep 戏法 | medium |

## 优先级行动清单

| 优先级 | 事项 | 维度 | Finding |
|--------|------|------|---------|
| **P0** | 审批人绑认证身份 + 拒自审批(SoD) | 安全 | d6-1 |
| **P0** | 拒绝空 / 裸 `*` subjectSelector | 安全 | d6-2 / d2-1 |
| **P0** | 审批快照加每能力配置 hash,挡能力漂移 | 安全 | d6-3 |
| P1 | 兜底模式常驻警告横幅,禁用主操作 | UI 真实性 | d1-1 |
| P1 | 统一 "上线就绪/检查/验收" 术语 | 文案 | d5-6 |
| P2 | 拆 workbench + presenters(先机械 1500 行) | 结构 | d4-2 / d4-4 |

---

## 维度 1 — UI 是否生产 B 端

**结论:主体已是任务优先的生产工作台,但一个 high 把它拽回 Demo 味。**

正面(已验证):
- 信息架构任务优先,顶部单一主操作 + 当前状态/下一步,重证据折叠进 `<details>` —— [App.tsx:3156-3230](../../frontend/src/App.tsx)
- 单一企业蓝主题,无玩具/装饰色 —— [styles.css:1-31](../../frontend/src/styles.css)
- 无假 Prod/Staging 切换、无占位搜索框,环境类控件都是真 scope 输入 —— [App.tsx:2656-2674](../../frontend/src/App.tsx)

| ID | severity | 问题 | 证据 |
|----|----------|------|------|
| d1-1 | **high** | 示例数据静默兜底,本页无任何警告 | `loadConsoleData` 把每个 fetch 包进 `withFallback`,后端挂了不抛错铺满 sample(api.ts:756-785);`showWorkspaceTelemetry = activeView.key !== "ai-admin"` 恰好把状态条/数据源指示在本页隐藏(App.tsx:2173);唯一线索藏在折叠的"连接设置"里 |
| d1-3 | medium | 无初始 loading 态,首屏闪空数据再翻 sample | `useState<ConsoleData\|null>(null)` + 无 aria-busy/skeleton(App.tsx:429-431) |
| d1-4 | medium | 错误与成功共用灰色样式槽,失败看着像提示 | `.approval-inline-message{color:#6e6e73}` 无 error 变体,且直出后端原始英文(App.tsx:1916,styles.css:2326) |
| d1-5 | medium | 主导航无 `aria-current="page"`(步骤条却有) | App.tsx:2560-2566 vs App.tsx:3385 |
| d1-6 | medium | 自定义 `ApprovalDropdown` 未关联字段标签,展开列表无方向键漫游 | App.tsx:3251-3306 |

**d1-1 修法**:`!data.loadedFromApi` 时在工作台正文渲染常驻可关闭警告横幅("正在展示示例数据 —— 后端不可达,操作不会持久化"),并在兜底模式禁用 Apply/Approve;不要只靠折叠菜单里一行字。

---

## 维度 2 — 访问对象选择是否合理

**结论:模型正确。** 业务对象(角色/部门/成员)确定性映射到 subjectSelector,主表单与高级设置共用同一字段、last-write-wins、无歧义([App.tsx:3287-3290](../../frontend/src/App.tsx));后端 5 条候选与前端 catalog 完全一致([access_subjects.go:5-43](../../internal/permissionpack/access_subjects.go))。

| ID | severity | 问题 | 证据 |
|----|----------|------|------|
| d2-1 | medium(验证从 high 下调) | 空 subjectSelector = 匹配全部主体,且 readiness 不拦 | `buildReadiness` 不检查 subjectSelector(permissionpack.go:208-235),空值一路写进 grant,运行时 `selector=="" → true`(memory.go:1374) |
| d2-2 | medium | `createInstanceAssignment` 不校验空 selector | server.go:3348-3352 仅查 workspaceAssignmentId/callerInstanceId |

> 下调理由:需操作员主动留空(非远程利用),且租户/工作区/region/调用方实例仍约束授权,只放宽 subject 维度。但仍是 least-privilege 缺口,与 d6-2 同根因,合并到维度 6 治理。

---

## 维度 3 — i18n / 技术 ID / 后端概念泄漏

**结论:主路径干净。** 主路径所有 `t()` key 在 en/zh 双语**全部定义齐全**,无 used-but-undefined;原始租户/工作区/调用方 ID 全部关在 Advanced `<details>` 内([App.tsx:3353-3365](../../frontend/src/App.tsx)),主路径用可读名(`tenantPath.primary`/`workspaceName`)。

| ID | severity | 问题 | 证据 |
|----|----------|------|------|
| d3-2 | low | catch 块英文 fallback 串在 zh 下露英文 | `"Unable to apply permission package"`(App.tsx:2052,1453) |
| d3-3 | low | 数据范围行 `dataDomain` 后端枚举原样显示 | `<code>{summarizeDataScopes(...)}</code>`(App.tsx:3332) |

---

## 维度 4 — App.tsx / styles.css 过大

**结论:确认的可维护性债,须拆。**

- `App()` 单函数 **2373 行**(L344-2716):74 个 `useState`、10 个 `useEffect`、~33 个 async handler、16 个内联面板闭包、9 屏 `switch` 路由 —— [App.tsx:344-2716](../../frontend/src/App.tsx)
- `styles.css` **3433 行**:零注释、扁平、4 个确认死选择器(`.permission-package-grid`/`.approval-check`/`.approval-checklist`/`.access-toolbar`)—— [styles.css](../../frontend/src/styles.css)
- 本次 diff 在原地重写 App.tsx 约 40%(+1224/-1208),每次加功能都搅动整文件,放大 review/merge 成本。

**最小拆分计划(无框架变更,沿用既有 sibling 模块惯例,按收益排序):**

1. `AiAdminPermissionWorkbench`(L2928-3711,~784 行,纯 props 驱动、零 hook)→ `components/AiAdminPermissionWorkbench.tsx`,机械搬迁零逻辑改动。
   **⚠️ 修正**:`ApprovalDropdown` 被 `TenantAccessProfileView` 也用(App.tsx:4472),**不能**跟着搬进 workbench,须留作共享组件。
2. 底部 `TenantAccessProfileView`(271行)/`CapabilityGovernanceView`(238行)/各 Table + L5061 起 ~80 个纯 helper → `components/` + `consolePresenters.ts`,约 -1500 行。
3. L572-2104 的 handler 群 + state 提进 `useConsoleController()` hook,App() 收到 ~400 行(shell+switch+wiring)。
4. CSS 按特性切 partial(base/nav/tables/approval/access/responsive),选择器名不变,TSX 不动;删 4 个死选择器。

> 建议把拆分当前置项而非后续项:在下一个功能落进这两个文件前先落 1、2,后续 diff 才能收敛到小模块。

---

## 维度 5 — "上线判断" 文案

**结论:须改,但不是改成"上线状态检查"。**

"上线判断"([i18n.ts:1437](../../frontend/src/i18n.ts))填的是 metric tile 的**状态槽**,值为"可上线/需复核/阻断"——它是状态名词,不是动作。"上线状态检查"是动作短语,塞状态槽更别扭。状态槽建议用 **"上线就绪 / 上线就绪状态"**。

| ID | severity | 问题 | 证据 |
|----|----------|------|------|
| d5-6 | **high** | 同屏混用"验收/验证/判断"指同一条自动检查流 | "检查上线验收"按钮(i18n.ts:805)填"最终上线判断"区块(i18n.ts:1182),区块坐在"上线验证"标题下(App.tsx:3473)——一个 handler、一条数据流、三个词根 |
| d5-1 | medium | "上线判断"词性错位(状态槽填动作名词) | i18n.ts:1437 |
| d5-5 | medium | ready 状态四种说法 | 生产可用/可以进入生产/已满足上线条件/可上线 |

**统一术语集(建议全量套用):**

- 动作 = **上线就绪检查**(按钮"检查上线就绪")
- 状态 = **上线就绪 / 上线就绪状态**
- 结果 = **可上线 / 需复核 / 阻断**(保留,够好)
- **"验收"只留给真正的人工签收/交接证据**(`nav.evidence` 上线证据),不用于自动检查;"验证"作为独立词从本流移除。

---

## 维度 6 — 安全 / 状态竞态 / 绕过(最高优先)

**先说扎实的部分(全部已验证):**

| ID | 结论 | 证据 |
|----|------|------|
| d6-5 | 内存 store 消费原子性安全:单 mutex 覆盖"可消费再检查 + 写授权",无 TOCTOU | memory.go:735-799 |
| d6-6 | Postgres `consumed_at is null` 条件 UPDATE + 同事务,双消费不可能 | postgres.go:923-984 |
| d6-7 | 过期在任何写入前拦截,两 store 一致 | memory.go:752,postgres.go:1116 |
| d6-8 | preflight / workbench preview 真只读,无副作用 | server.go:2611-2714 |
| d6-9 | 租户子树在 draft build 强制,apply 不能越租户建授权 | server.go:4863-4897 |

**确认成立的问题:**

### d6-1 (high) — 审批人自报、不绑身份,可自审批

reviewer 直接从请求体读,空了 fallback 成常量 `"admin-key"`([server.go:2764-2772](../../internal/httpapi/server.go));校验只把"自报的"reviewer 字符串匹配配置规则([server.go:5035](../../internal/httpapi/server.go))。任何持 admin key 者都能在 body 填 `reviewer:"security-east"` 通过;requester 与 reviewer 同为 `"admin-key"`,无 `reviewer != requestedBy` 检查。`AGENT_HARBOR_APPROVAL_REVIEWERS` 当前是**路由提示,非授权边界**。

**修**:reviewer 绑独立认证身份(每人一 key/JWT),加 SoD 拒自审批。

### d6-2 (high) — 空 / 裸 `*` subjectSelector → 运行时匹配全部主体

空值通过 readiness、可被审批、原样写入([server.go:2891](../../internal/httpapi/server.go));运行时 `selector==""` 或 `"*"` 恒真,两 store 一致([memory.go:1374-1384](../../internal/store/memory.go),[postgres.go:1262](../../internal/store/postgres.go))。被省略的字段静默成最大权限。

**修**:draft/preflight/apply 拒绝空与裸 `*`,或对 match-all 单独走高风险审批。

### d6-3 (high) — 审批快照不含每能力风险/范围,改宽的能力可在旧审批下落地

apply 用 `buildPermissionPackageDraft` 从**实时**能力状态重建 draft([server.go:2807](../../internal/httpapi/server.go)),审批校验只比对 DraftID/版本/scope/能力 **ID 集合**,不比每能力 RiskLevel/DataScopes([server.go:5171-5184](../../internal/httpapi/server.go));`draftID` 只由 `{Template,Tenant,Workspace,CallerInstance,Target}` 算,`policyVersion` 静态常量 1。

**可构造攻击序列(已验证)**:审批通过 → `PATCH /capabilities/{id}` 把某能力 DataScopes 改宽(ID/Key/Version 不变、DiscoveryStatus 不重置,[server.go:1154](../../internal/httpapi/server.go))→ apply,逐字段比对全过,reviewer 没见过的更宽授权被批下。

**修**:每能力有效配置(风险/动作/敏感度/dataScopes 或内容 hash)快照进审批记录并在 apply 时比对,或折进 draftID。

### d6-4 (medium) — 无 admin key 时 `requireAdmin` 静默 no-op,变更端点全开

`if s.adminKey == "" { next.ServeHTTP(...); return }`([server.go:231-244](../../internal/httpapi/server.go))。误配置部署把 approve/apply 全暴露且无告警。

**修**:无 key 时拒绝启动或拒绝服务变更路由,除非显式开 dev flag。

---

## 维度 7 — 测试覆盖

**结论:后端强,前端一半真一半 grep 戏法。**

后端是套件最强部分(已验证,均为既有用例):过期拒绝、双 apply "already consumed"、审批人越权 403 + 状态不变、scope drift "does not match" 全覆盖,断言真实拒绝路径与持久化副作用 —— [server_test.go:3953-3999](../../internal/httpapi/server_test.go)。

| ID | severity | 问题 | 证据 |
|----|----------|------|------|
| d7-5 | medium | 访问对象→subjectSelector 派生(安全字段)从未被执行 | 真逻辑在 dropdown onChange(App.tsx:3287-3289),测试只测纯 helper |
| d7-7 | medium | 版本漂移在 preview/apply 边界未测(只测了 scope 漂移) | 守卫在 server.go:5172-5174,无对应用例 |
| d7-2 | medium | 5 步流程 `aria-current` 只 grep 源码,无渲染驱动状态迁移 | permissionFlowLayout.test.mjs:62 |
| d7-1/3/4 | low | i18n/styleTheme/api 新增多为 `assert.equal(常量, 常量)` 变更探测器 | i18n.test.mjs:79-272 等 |

**补测建议**:把 onChange 派生抽成纯 reducer 单测(d7-5);加 template/policy version bump 后 preview/apply 应 needs-reapproval/400 的用例(d7-7);i18n 改为断言 en/zh key 集合一致 + 主路径 key 双语存在(替代逐串相等)。

---

## 附:验证修正记录(体现审查颗粒度)

| Finding | 报告原 severity | 验证后 | 修正原因 |
|---------|-----------------|--------|----------|
| d2-1 | high | medium | 需操作员主动留空,非远程利用 |
| d4-1 | high | medium | 结构债,无 correctness bug |
| d4-2 | high | low | 纯 props 组件,且推荐里 ApprovalDropdown 归属判断有误(应共享) |
| d6-1 / d6-2 / d6-3 | high | high(维持) | 均给出确切证据/攻击序列 |
| d5-6 | high | high(维持) | 单屏跨按钮/标题/徽章的术语混用 |
