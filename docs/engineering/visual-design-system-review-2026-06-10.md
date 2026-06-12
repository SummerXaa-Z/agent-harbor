# 视觉设计系统审查与改造方案

- **日期**:2026-06-10
- **分支**:`codex/production-readiness-gate`
- **范围**:全控制台视觉设计与体验设计(权限变更 / 权限画像 / 上线证据 / 调用日志 / 系统自检 / Agent 与工具 / 工具能力,1440×900 桌面实测)
- **方法**:4174 独立 preview + 9090 真实后端,逐页 DOM 度量(字号 / 字重 / 色值 / 圆角 / 间距 / 阴影 / 焦点态)+ CSS 源码静态统计
- **与前序报告的关系**:[permission-console-ux-review-2026-06-10.md](permission-console-ux-review-2026-06-10.md) 关的是单页结构性问题(导航 / 截断 / 重复),已闭环;本报告往下钻一层——**设计系统层面的系统性缺陷**。结构修了,但"为什么看起来还是廉价"的答案在这里。

> 总体判断:没有设计系统,只有 CSS 堆积。20 个 CSS 变量定义了却大面积不用(styles.css 硬编码色值 161 处 vs var() 110 处,workbench.css 硬编码 107 处 vs var() 19 处);字重 15 档、文字色 22 种、背景色 15 种、圆角 5 种——这些数字本身就是诊断书。**单独修任何一页都治不了,要建 token 层再往下压。**

## 实测数字总览(1440×900,权限变更页为主)

| 度量项 | 实测值 | 健康基准 | 判定 |
|--------|--------|----------|------|
| 字号档位 | 9 档(11/12/13/14/15/16/18/20/24) | 5-6 档 | ⚠️ 偏多 |
| 字重档位 | **15 档**(400/520/560/600/620/640/650/670/680/690/700/720/740/760/780/800) | 3-4 档 | 🔴 失控 |
| 正文文字色 | **22 种** | 4-5 种(主/次/弱/反白) | 🔴 失控 |
| 背景色 | 15 种 | 6-8 种 | ⚠️ 偏多 |
| 圆角 | 5 种(5/7/8px/50%/999px) | 3 种(小/卡片/圆) | ⚠️ 不统一 |
| 按钮高度 | 30/34/38px 三种并存同屏 | 2 种(默认/紧凑) | ⚠️ 不统一 |
| 按钮水平 padding | `1px 6px`(主按钮!) / `0 9px` / `0 10px` / `0 11px` | 统一 token | 🔴 主按钮 padding 6px 离谱 |
| 输入框焦点态 | `outline: none`,`box-shadow: none`,边框色不变 | 可见 focus ring | 🔴 键盘用户致盲 |
| 表格行高 | 77px(双行内容无密度选项) | 40-48px 紧凑档 | ⚠️ 浪费 |
| 内容最大宽度 | `max-width: none`(1205px 全铺) | 表单列 ~720px 上限 | ⚠️ 宽屏下表单拉满 |
| 阴影 | 3 种各自手写 | 1-2 个 token | ⚠️ |
| 网格 gap | 1/10/12/14/20px 混用 | 4 的倍数刻度 | ⚠️ |

CSS 源码佐证:

```
styles.css            硬编码色值 161 处 vs var(--*) 110 处
permission-workbench  硬编码色值 107 处 vs var(--*)  19 处
font-weight 分布      700×12 720×7 650×5 760×4 670×2 800/780/740/690/680/640/620/600/560/520 各1
:root 已定义 20 个 token(--ink/--muted/--brand/--radius/--shadow…)——定义了,没人用
```

## 诊断:为什么"看起来差劲"

### vd-1(critical)— Token 层形同虚设,每个组件自带一套视觉

`:root` 定义了 `--ink`、`--muted`、`--radius`、`--shadow` 等 20 个变量,但 styles.css 里 161 处、workbench.css 里 107 处直接写死色值。结果是 22 种文字色、15 档字重——**不是设计师选了 22 种灰,是每次写组件随手挑一个**。字重 520/620/640/650/670/680/690 这种 10 位数递增就是"觉得 600 不够粗、700 太粗"时拍出来的,Inter 是可变字体所以都渲染得出来,视觉上却造成"说不出哪里不对的杂"。

这是其余所有问题的根因:不先收口 token,修任何一页,下一页又长回来。

### vd-2(critical)— 输入框无可见焦点态

实测 `input:focus`:`outline: rgb(29,37,45) none 0px`、`box-shadow: none`、边框色保持 `#d8e0e7` 不变。`--focus-ring` token 定义了、也有 4 处 `box-shadow: 0 0 0 3px var(--focus-ring)` 规则,但实测主表单输入框没命中。键盘 Tab 导航时完全不知道焦点在哪——对一个강调 aria 语义的权限管理产品,这是可访问性硬伤(WCAG 2.4.7 Focus Visible 直接不过)。

### vd-3(high)— 主按钮质感廉价:`padding: 1px 6px`

主操作按钮("导出证据"/"应用权限")实测 `padding: 1px 6px`、高 34px;次按钮 30px、表单控件 38px,三种高度同屏。文字两侧 6px 空隙让主 CTA 看起来像个紧巴巴的标签而不是按钮。按钮是 B 端产品手感的第一来源,这一项对"廉价感"的贡献最大。

### vd-4(high)— KPI 指标条全站复读,占用每页首屏

"托管 Agent 10 / 启用策略 0 / 拒绝追踪 5 / 运行证据 15"这条指标条 + "API/数据源/最近刷新/范围"状态条,在调用日志、系统自检、Agent 与工具、工具能力、上线证据**五个页面顶部原样重复**,每页吃掉 ~180px 首屏高度。调用日志页用户要看的是追踪列表,Agent 注册表页要看的是表格——全站复读的 KPI 和当前任务无关,是"仪表盘思维"残留。状态条信息(API 地址/数据源/刷新时间)属于连接诊断,应该收进"连接设置"弹层,不该常驻每页。

### vd-5(high)— Agent 注册表:创建表单永久占首屏,表格行高 77px 且操作列截断

Agent 与工具页首屏是三张并排的"创建 Agent / 创建密钥 / 轮换凭据"表单,**注册表本体被压到第二屏**。管理场景下"看列表"频率远高于"创建"——创建应是按钮+抽屉,不是常驻三栏。表格本身:行高 77px(名称+ID 双行堆叠),无斑马纹、无 hover 态实测差异,操作列"转为草稿/禁用"按钮实测溢出截断(scrollWidth > offsetWidth)。10 行数据要滚两屏。

### vd-6(medium)— 信息密度无档位,长 ID 无处理策略

`agt_7b6edfbee6819a04da8f654...`、`cap_87d06aa4dc389f29e1dbadc27e0ca66` 这类技术 ID 在表格、追踪列表中整段平铺,既截断又占宽。缺中间態处理:等宽字体 + 首尾保留中段省略 + 点击复制。同时全站只有一种密度,审计日志这种千行场景没有紧凑模式。

### vd-7(medium)— 排版细节:正文 16px 无 line-height、标题字重漂移、空状态简陋

- body/label 16px 但 `line-height: normal`(≈1.2),中文正文应 1.5-1.6
- h1 700 / h2 720 / 各卡片标题 650-780 漂移,无 scale
- "暂无证据运行"空状态:一行灰字居中,无图形、无引导动作,白板一块
- 权限画像页"权限判定说明"标题连续渲染两遍(卡片头 + 内容区各一个 `<strong>`)

### vd-8(medium)— 布局节奏:无 8pt 网格,宽屏无阅读宽度上限

gap 1/10/12/14/20px、padding 9/11/14px——奇数值混用说明没有间距刻度。1440px 视口下表单列拉满 1205px,单行输入框宽逾 1100px,远超舒适阅读/扫视宽度。

## 改造方案:三步走,先建系统再扫页面

### 第一步(P0):立 Design Token 层,一次性收口

在 `styles.css` `:root` 重定义并**强制全站只用 token**(把现有 268 处硬编码逐步替换):

```css
:root {
  /* 字号 6 档 */
  --text-xs: 11px;   /* 徽标、表格辅助 */
  --text-sm: 12px;   /* 辅助说明 */
  --text-base: 13px; /* 表单控件、表格正文 */
  --text-md: 14px;   /* 正文 */
  --text-lg: 16px;   /* 卡片标题 */
  --text-xl: 20px;   /* 页标题 */
  /* 字重 3 档:400 / 600 / 700,删除其余 12 档 */
  /* 行高:正文 1.6,标题 1.3 */
  /* 文字色 4 种 */
  --ink: #1d252d; --ink-2: #4b5563; --ink-3: #8b98a5; --ink-inverse: #fff;
  /* 间距 8pt 刻度:4 / 8 / 12 / 16 / 24 / 32 */
  /* 圆角 3 档:--r-sm: 6px(控件) --r-md: 10px(卡片) --r-full: 999px(徽标) */
  /* 阴影 2 档 + 焦点环 */
  --shadow-card: 0 1px 2px rgba(16,24,40,.05);
  --shadow-pop: 0 12px 24px -6px rgba(16,24,40,.12);
  --focus-ring: 0 0 0 3px rgba(0,113,227,.18);
}
```

验收标准(可写成回归测试,沿用现有 styleTheme.test.mjs 的源码扫描模式):
- styles.css + workbench.css 硬编码十六进制色值 ≤ 20 处(仅 token 定义处)
- `font-weight` 只允许 400/600/700
- 全部 `input/select/textarea:focus-visible` 必须命中 `--focus-ring`

### 第二步(P1):组件级统一

| 组件 | 现状 | 目标 |
|------|------|------|
| 按钮 | 30/34/38px 三高度,`1px 6px` padding | 统一 36px(紧凑 30px),padding `0 16px`,主/次/危险三态,hover/active/disabled 全定义 |
| 输入框 | 38px,无焦点态 | 36px 对齐按钮;`:focus-visible` 边框变 brand + focus-ring |
| 表格 | 77px 行高,无 hover,操作列截断 | 名称+ID 合并为主格(ID 等宽小字 + 中段省略 + copy);行高 52px;hover 背景;操作列收进 `⋯` 菜单,根治截断 |
| 徽标 | 多套写法 | 一个 `.badge` 组件,4 个语义色变体 |
| 空状态 | 一行灰字 | 统一空状态组件:线性插图 + 一句话 + 主动作按钮 |
| KPI 条 | 五页复读 | 只保留在系统自检(它本来就是健康页);其余页顶部状态条收进"连接设置"弹层 |

### 第三步(P2):页面级布局节奏

- 表单列 `max-width: 720px`,说明文字 `max-width: 60ch`;宽屏多余空间留白,不拉伸
- Agent 与工具页改"列表优先":注册表置顶,创建/密钥/轮换收进右上角按钮 + 抽屉(Drawer)
- 调用日志:追踪列表升首屏,加紧凑密度(行高 40px)和 runId 等宽展示
- 卡片层级两档:页面区块(带 `--shadow-card`)> 表单分组(仅分隔线,不再套卡片),消除"卡片套卡片"
- 全站间距走 8pt:卡片 padding 16/24,区块间 24,组内 12

## 不要做的事

- 不要加新颜色、渐变、玩具图标——现有"克制企业蓝"方向是对的,问题在执行精度不在风格选型
- 不要为视觉重写组件结构——上一轮 ux-1~5 的结构修复保持原样
- 不要引入 UI 框架(antd/shadcn)整包替换——268 处硬编码先压进 token,成本远低于换框架且不破坏现有 91 条测试

## 验证计划

```bash
pnpm --dir frontend test          # 现有 91 条 + 新增 token 回归(硬编码扫描/字重白名单/焦点态)
pnpm --dir frontend build
make check && make release-check
# 浏览器复测:1440/846/375 三档,焦点 Tab 走查 + 各页首屏构成截图
```

## 修复闭环(2026-06-10)

本轮按 P0/P1 先修设计系统和高频工作区首屏节奏,目标是把控制台从"页面级修补"推进到"可持续约束"。未引入 UI 框架,也没有新增装饰色,而是把现有视觉值压回 token 和组件规则。

| Finding | 修复状态 | 实际调整 | 回归约束 |
|---------|----------|----------|----------|
| vd-1 Token 层形同虚设 | 已修复首批 | `:root` 扩展为统一字号、间距、圆角、阴影、状态色、等宽字体和玻璃/阴影 token;组件层禁止裸十六进制色值和裸 `rgba()`;字重收敛到 600/700。 | `component styles consume tokens instead of ad hoc visual values` |
| vd-2 输入框无可见焦点态 | 已修复 | 全局 `button/input/select/textarea/summary:focus-visible` 使用 `--shadow-focus`;审批与访问表单从裸 `:focus` 改为 `:focus-visible`,统一键盘焦点语义。 | `focus and button controls use the shared production interaction tokens` |
| vd-3 主按钮质感廉价 | 已修复首批 | 主/次按钮统一 `--control-height` 与 `--space-4`;表格动作和审批动作统一紧凑高度 `--control-height-compact`;危险/主操作继续保留语义色。 | `focus and button controls use the shared production interaction tokens` |
| vd-4 KPI 指标条全站复读 | 已修复 | `showWorkspaceTelemetry` 仅在系统自检页展示;其他工作区不再重复全局 KPI/status strip,连接上下文继续放在右上角连接设置。 | `workspace telemetry is scoped to system check instead of repeating on every workspace` |
| vd-5 Agent 注册表列表不优先 | 已修复首批 | Agent 与工具工作区先展示注册表,创建 Agent、创建密钥、轮换凭据表单下移到列表之后。抽屉化留作下一轮更大交互改造。 | `agent tools workspace prioritizes the registry before mutation forms` |
| vd-6 长技术 ID 无处理策略 | 已修复首批 | 新增 `TechnicalId` 展示单元:首尾保留、中段省略、等宽字体、图标复制按钮;先应用到 Agent 注册表和调用日志 capability id。 | `technical identifiers use a readable copyable component in dense workspaces` |

### 当前验证记录

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
pnpm --dir frontend test
pnpm --dir frontend build
make check
make release-check
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_BROWSER_GATE_API_PORT=9093 AGENT_HARBOR_BROWSER_GATE_FRONTEND_PORT=5176 MOCK_MCP_PORT=8788 make ai-admin-browser-journey
git diff --check
```

浏览器复测(127.0.0.1:5174,2026-06-10):

- 1440×900 / 846×900 / 375×800 三档无横向溢出。
- 权限变更、Agent 与工具、调用日志页面不再显示全局 KPI/status strip。
- 系统自检页保留全局状态和 4 个指标卡,符合"只在健康检查页展示"的目标。
- Agent 与工具页首屏顺序为 Agent 注册表 -> 创建 Agent -> 创建密钥 -> 轮换凭据 -> 契约矩阵。
- Agent 注册表和调用日志均出现可复制 `TechnicalId` 展示单元,技术 ID 不再作为普通长文本直铺。

## 边界说明

- 本报告全部数字来自 1440×900 实测与源码统计,可复现(4174 preview,方法同前序报告)
- 未覆盖暗色模式(当前无)、打印样式、Windows 字体渲染(Inter 在 Win 低分屏的字重表现需单独验证)
- "健康基准"列为行业惯例(8pt grid / WCAG 2.4.7 / 主流 B 端设计系统的 type scale),非客户硬性要求
