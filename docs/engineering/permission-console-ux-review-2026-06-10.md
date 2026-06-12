# 权限控制台 UX 实测审查报告

- **日期**:2026-06-10
- **分支**:`codex/production-readiness-gate`
- **范围**:Web 控制台真机体验审查(权限变更 / 权限画像 / 能力治理三个工作区,桌面 846px + 移动 375px)
- **方法**:独立 preview 实例(端口 4174,不干扰本地 5174 开发会话)连接真实 Go 后端(127.0.0.1:9090),逐页截图 + DOM 度量(导航宽度 / 按钮尺寸 / 文本溢出),非源码 grep
- **证据基线(实跑)**:`make release-check` 通过;隔离浏览器主旅程门禁 `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_BROWSER_GATE_API_PORT=9093 AGENT_HARBOR_BROWSER_GATE_FRONTEND_PORT=5176 MOCK_MCP_PORT=8788 make ai-admin-browser-journey` 通过;本报告 UX finding 来自 4174 preview 实测,不是自动测试失败。
- **与 6 月 9 日审查的关系**:[permission-changes-review-2026-06-09.md](permission-changes-review-2026-06-09.md) 及其[修复报告](permission-changes-review-repair-report-2026-06-10.md)关的是**缺陷**(安全 / 一致性 / 可访问性),全部已收口;本报告关的是**好不好用**——这是另一个标准,两者不互相替代。

> 总体判断:功能闭环完整、安全边界正确、测试全绿,但体验停留在"AI 生成 B 端模板"水位。没有 blocker;**2 个 high、3 个 medium**,全部是高频使用路径上的体验损耗。

## 结论总览

| # | 审查面 | 结论 | 最高 severity |
|---|--------|------|---------------|
| 1 | 导航与信息架构 | 纯图标侧边栏 + 竖排分组标题,记忆成本拉满 | high |
| 2 | 文本与表单密度 | 截断省略号满屏,关键信息看不全 | high |
| 3 | 信息重复 | 同一上下文同屏渲染三次,无权威视图 | medium |
| 4 | 状态语义 | 四个状态信号同屏打架,归属不清 | medium |
| 5 | 视觉层级 | 卡片套卡片全平级,操作区被 KPI 大卡挤到底部 | medium |

## 优先级行动清单

| 优先级 | 事项 | Finding | 一句话方案 |
|--------|------|---------|-----------|
| **P1** | 侧边栏改 ~200px 图标+文字导航,9 项分 3 组,砍竖排分组标题 | ux-1 | 收益最高的一刀;移动端已有文字标签可直接复用文案 |
| **P1** | 下拉与说明宽度按内容自适应,省略号只允许出现在次要列;去重复 label | ux-2 | 权限选择器必须全文可见 |
| P2 | 租户/工作区/调用方收敛为页头一条 context bar,全页唯一,表单内不再重复;补上空灰格或改 3 列布局 | ux-3 | 一处权威,处处引用 |
| P2 | 状态只保留页头一个权威 badge + 下一步动作;子卡片状态改为流程内步骤态 | ux-4 | 把"对象不同"显式讲出来 |
| P2 | 建立 2-3 级卡片层级(页面区块 > 表单组 > 字段);能力治理 KPI 卡压缩为单行指标条,操作表单上移 | ux-5 | 干活的区域优先于看的区域 |

## 修复闭环(2026-06-10)

本轮已按上表完成 P1/P2 首批收束,目标不是增加视觉装饰,而是把控制台拉回"正常 B 端生产系统"的可读、可操作、可回归状态。

| Finding | 修复状态 | 实际调整 | 回归约束 |
|---------|----------|----------|----------|
| ux-1 | 已修复 | 1120px 以下桌面断点不再压成 72px 图标栏,改为 200px 图标+文字导航;分组标题禁止折行;移动端继续使用横向文字导航。 | `desktop sidebar keeps text labels at review viewport widths` |
| ux-2 | 已修复 | 权限变更表单把租户、工作区、访问对象、权限包模板升为宽字段;核心下拉值允许换行,不再强制 `ellipsis`;只把技术 ID 保留在高级设置。 | `permission request core selectors avoid forced ellipsis at review widths` |
| ux-3 | 已修复 | 租户/工作区/调用方只在页头 context bar 作为权威上下文展示;状态摘要去掉重复租户卡,改为权限包、计划对象、上线就绪三项。空灰格补丁:2 列断点下 context bar 末格(奇数项)横跨整行(`styles.css` 1120px 媒体查询),846px 实测调用方格 555px 占满,无空格残留。 | `permission request overview keeps context in one authoritative bar` |
| ux-4 | 已收束 | 页头继续承载当前状态与下一步动作;右侧处理区保留流程步骤态,不再用重复上下文卡制造多状态冲突。 | `permission request journey exposes the active step for production operators` |
| ux-5 | 已修复 | 能力治理 KPI 压缩为单行紧凑指标条;授权链表单移动到主操作区顶部,能力列表和授权链列表跟随其后。 | `capability workspace compresses metrics and prioritizes grant operations` |

### 修复后实测

- 846px 桌面视口:侧边栏宽度为 200px,9 个导航项均保留中文名称和说明,不再依赖 tooltip 记忆。
- 846px 权限变更页:核心选择器的 `white-space=normal`, `text-overflow=clip`,上下文只在页头权威展示。
- 846px 能力治理页:KPI 为紧凑单行指标条,授权链表单位于能力治理主内容最前,不再被大卡片挤到底部。

### 独立验收复测(4174 preview + 9090 真实后端,2026-06-10 第二轮)

逐项 DOM 度量复核 codex 修复,与上表自述一致:

- ux-1:侧边栏 200px,导航项含文字(实测按钮文案"权限变更 配置、审批、应用…"),分组标题"主流程/运行审计/系统配置"`white-space: nowrap` 横排不折行;移动端 375px 顶部文字导航正常。
- ux-2:五个核心选择器(租户/调用方/目标/访问对象/权限包模板)实测 527px 宽,`white-space: normal` + `text-overflow: clip`,无截断(scrollWidth ≤ offsetWidth)。
- ux-3:全页"客户服务中心"仅 3 处——context bar 权威值、下拉选中项、租户层级说明("集团总部 > 客户服务中心"),语义各不相同,重复卡已消除;空灰格在验收中发现仍残留(2 列断点 3 格),已补 CSS 修复并实测确认末格横跨整行。
- ux-4:同屏状态 badge 从 4 个降为 3 个("生产可用"+"待审批"+流程步骤态),重复上下文卡撤掉后状态归属可读;"还差 8"为就绪面板进度,保留合理。
- ux-5:能力治理 KPI 为单行四格紧凑指标条,授权链表单位于主操作区顶部,实测截图确认。

复测后回归:`pnpm --dir frontend test` 91/91 通过,`pnpm --dir frontend build` 通过,`git diff --check` 干净。

### 验证记录

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/permissionFlowLayout.test.mjs
pnpm --dir frontend test
make check
make release-check
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_BROWSER_GATE_API_PORT=9093 AGENT_HARBOR_BROWSER_GATE_FRONTEND_PORT=5176 MOCK_MCP_PORT=8788 make ai-admin-browser-journey
git diff --check
```

---

## Finding 明细

### ux-1(high)— 侧边栏纯图标 + 分组标题竖排折行

实测 DOM:侧边栏宽 **72px**,9 个导航项均为 **44×52px 纯图标按钮**,文字只存在于 `title` tooltip 中。分组标题在 72px 宽度内竖排折行,"运行审计"渲染为"运行审␣计","系统配置"渲染为"系统配␣置"(桌面截图可见)。

影响:权限治理控制台是管理员日常工具,9 个图标(权限变更 / 权限画像 / 上线证据 / 调用日志 / 系统自检 / Agent 与工具 / 工具能力 / 访问策略 / 路由规则)无文字锚点,首周学习成本和误点率都高;竖排折行的分组标题在任何 B 端设计规范里都不该出现。

### ux-2(high)— 截断文本满屏,关键信息靠省略号

实测(权限变更页,846px 桌面宽度):

- 下拉框:"客户服务..." "客服工单..." "工单工具..." "角色 · 客..." ——租户、模板、目标、访问对象四个核心选择器全部截断
- 说明文字:"执行运行验证,系统会准..." "集团总部 > 客户服务..." "由所选租户和调用方..."
- 表单中"租户"标签渲染两遍(分组 label 与字段 label 重复),"访问对象"同样重复

影响:管理员在提交审批前无法在不 hover / 不展开的情况下确认自己选的是什么——这恰好是权限变更最不该模糊的环节。

### ux-3(medium)— 同一上下文同屏重复三次,且有空灰格

"客户服务中心 / 客户服务工作区 / 客服助手"在同一屏出现三处:① 页头下方的上下文信息条;② "当前状态"右侧的租户卡;③ 申请信息表单。三处无主从关系,改了表单另外两处才跟着变,首次使用无法判断哪个是权威值。上下文信息条右下角还有一块**无内容的灰色格子**(2×2 网格只填了 3 格),视觉上像渲染 bug。

### ux-4(medium)— 状态信号同屏打架

同一屏(本地演练数据)并存:"生产可用"(当前状态卡)、"待审批"(申请信息 badge)、"未发起"(处理流程 badge / 提交审批 badge)、"还差 8"(上线就绪 badge)。每个都是真数据,但页面没有讲清楚**状态归属的对象不同**(上一次已完成的变更 vs 正在草拟的新申请)。管理员第一反应是"到底能不能上线"。

### ux-5(medium)— 视觉层级扁平,操作区被指标卡挤压

所有内容都是同一种圆角白卡,字号字重几乎一致,无主次。能力治理页:四张 KPI 大数字卡(托管 Agent 10 / 启用策略 0 / 待审能力 5 / 运行证据 15)占据首屏,真正的操作对象——授权链表单与能力列表——沉到第二屏以下。指标是看的,授权链是干活的,主次颠倒。

## 正面清单(已验证,不要在迭代中弄丢)

- 配色克制:单一企业蓝 + 独立状态色,无装饰色污染
- 中英文切换完整,主路径无技术 ID 泄漏
- `aria-busy` / `aria-current` / combobox 键盘语义齐全(6 月 10 日修复批次成果)
- 兜底数据常驻警告 + 禁用写操作,fail-closed 鉴权提示准确
- 移动端 375px 断点可用,导航转为顶部水平文字标签(移动端反而有文字,佐证 ux-1 桌面端应补文字)

## 验证方式(复现实测)

```bash
# 后端已在 9090 运行的前提下,起独立前端实例(不动 5174)
pnpm --dir frontend dev --port 4174 --host 127.0.0.1
# 4174 在后端 CORS 默认白名单内(internal/httpapi/server.go localDevCORS),
# 5178 等其他端口不在,会全量 net::ERR_FAILED——本次实测踩过,留此提醒
```

逐项核对:侧边栏 DOM 宽度与按钮尺寸、四个核心下拉的截断、上下文三处重复、四个状态 badge 并存、能力治理首屏构成,桌面与 375px 移动断点各走一遍。

## 边界说明

- 本报告基于本地演练数据(客户服务中心租户),未覆盖大数据量(长租户树 / 百级能力列表)下的表格与下拉表现
- 未做正式可用性测试(无真实管理员参与),finding 来自启发式走查,severity 为单人判断
- 调用日志 / 访问策略 / 路由规则三个低频工作区只做了导航可达性确认,未逐页审查
