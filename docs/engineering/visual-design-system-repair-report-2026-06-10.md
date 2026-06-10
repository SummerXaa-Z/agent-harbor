# 视觉设计系统修复报告

- **日期**: 2026-06-10
- **项目**: AgentHarbor
- **范围**: Web 控制台视觉设计系统首批修复
- **复审对象**: `visual-design-system-review-2026-06-10.md`
- **本轮目标**: 先修 P0/P1 级体验问题,把控制台从页面级补丁推进到可回归的设计系统约束。

## 摘要

本轮没有引入新的 UI 框架,也没有继续增加装饰性颜色或复杂组件。修复重点是把散落的视觉值压回统一 token,把焦点态和按钮尺寸收敛到可访问、可复用的组件规则,并修正几个高频页面的首屏信息架构问题。

已完成的核心变化:

- 建立更完整的设计 token 层,覆盖字号、间距、圆角、阴影、状态色、等宽字体和焦点环。
- 禁止组件层继续散落裸十六进制色值和裸 `rgba()`。
- 将审批/访问表单的焦点态从裸 `:focus` 收敛到 `:focus-visible`。
- 主按钮、次按钮、表格动作按钮、审批动作按钮统一尺寸 token。
- 全局 KPI / status strip 只保留在系统自检页,不再跨工作区复读。
- Agent 与工具页改为注册表优先,创建/密钥/轮换表单下移。
- 新增 `TechnicalId` 展示单元,长技术 ID 首尾保留、中段省略、等宽显示,并提供复制按钮。
- 将上述规则写入 `styleTheme.test.mjs`,避免后续回退。

## 修改文件

| 文件 | 作用 |
|------|------|
| `frontend/src/styles.css` | 扩展主题 token;统一按钮、焦点、阴影、技术 ID、trace 元信息样式;限制全局 KPI 展示相关样式。 |
| `frontend/src/styles/permission-workbench.css` | 审批工作台按钮 hover、上下文阴影和表单焦点态 token 化。 |
| `frontend/src/App.tsx` | 收敛 telemetry 展示范围;调整 Agent 与工具页顺序;新增 `TechnicalId`;将 Agent 表和 Trace 表接入可复制技术 ID。 |
| `frontend/tests/styleTheme.test.mjs` | 新增视觉系统回归测试,覆盖 token、裸色值、裸 rgba、焦点态、按钮尺寸、telemetry 范围、registry 顺序、technical ID。 |
| `docs/engineering/visual-design-system-review-2026-06-10.md` | 在原审查报告中追加修复闭环、验证记录和浏览器复测结论。 |
| `CHANGELOG.md` | 补充中英双语变更记录。 |
| `docs/engineering/visual-design-system-repair-report-2026-06-10.md` | 本报告,供 Claude 复审。 |

## Finding 对应修复

| Finding | 原问题 | 本轮处理 | 状态 |
|---------|--------|----------|------|
| vd-1 Token 层形同虚设 | 组件层大量硬编码色值、阴影、字重 | 扩展 `:root` token;组件层裸 hex / 裸 rgba 进入测试禁止;字重白名单限制为 `400/600/700` | 已修复首批 |
| vd-2 输入框无可见焦点态 | 表单焦点态不统一,键盘用户难以定位 | 全局 `:focus-visible` 使用 `--shadow-focus`;审批/访问表单改为 `:focus-visible` | 已修复 |
| vd-3 主按钮质感廉价 | 主按钮 padding 和高度不稳定 | 主/次按钮使用 `--control-height` + `--space-4`;表格/审批动作使用紧凑高度 token | 已修复首批 |
| vd-4 KPI 指标条全站复读 | 多个工作区重复展示与当前任务无关的 KPI/status | `showWorkspaceTelemetry = activeView.key === "cockpit"`;仅系统自检保留 KPI/status | 已修复 |
| vd-5 Agent 注册表不优先 | 创建表单占据首屏,列表被挤到后面 | registry 工作区顺序改为注册表 -> 创建 Agent -> 创建密钥 -> 轮换凭据 -> 契约矩阵 | 已修复首批 |
| vd-6 长技术 ID 无策略 | Agent / Trace 中技术 ID 直铺、截断、不可复制 | 新增 `TechnicalId`,接入 Agent 表和 Trace capability id | 已修复首批 |
| vd-7 排版/空状态细节 | 空状态、行高、标题层级仍有继续优化空间 | 空状态已完成首批组件化:统一 `EmptyRow` 图标、标题/说明结构、128px 稳定高度和紧凑变体;表格密度与标题层级仍留后续 | 部分闭环 |
| vd-8 布局节奏 | 宽屏表单阅读宽度、8pt 节奏仍可继续收敛 | 本轮通过 token 建基础,未全量重排所有页面 | 保留后续 |

## 回归测试新增规则

`frontend/tests/styleTheme.test.mjs` 当前重点约束:

- `theme tokens define one restrained brand color system`
- `component styles consume tokens instead of ad hoc visual values`
- `focus and button controls use the shared production interaction tokens`
- `workspace telemetry is scoped to system check instead of repeating on every workspace`
- `agent tools workspace prioritizes the registry before mutation forms`
- `technical identifiers use a readable copyable component in dense workspaces`

关键意义:

- 后续如果有人在组件层直接写 `#xxxxxx` 或 `rgba(...)`,测试会失败。
- 后续如果 `.approval-*` / `.access-*` 控件重新使用裸 `:focus`,测试会失败。
- 后续如果 Agent 页把创建表单重新放回注册表前面,测试会失败。
- 后续如果 Trace 或 Agent 表重新直铺技术 ID,测试会失败。

## 实测验证

已执行并通过:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
pnpm --dir frontend test
pnpm --dir frontend build
make check
make release-check
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_BROWSER_GATE_API_PORT=9093 AGENT_HARBOR_BROWSER_GATE_FRONTEND_PORT=5176 MOCK_MCP_PORT=8788 make ai-admin-browser-journey
git diff --check
```

结果:

- `pnpm --dir frontend test`: 96/96 通过。
- `pnpm --dir frontend build`: 通过。
- `make check`: 通过。
- `make release-check`: 通过。
- `make ai-admin-browser-journey`: 通过,覆盖审批、预检、应用、运行 allow/deny、访问画像、审计和上线证据。
- `git diff --check`: 干净。

浏览器复测:

- 目标地址: `http://127.0.0.1:5174/`
- 视口: `1440x900`, `846x900`, `375x800`
- 结论:
  - 三档视口均无横向溢出。
  - 权限变更、Agent 与工具、调用日志页面不再展示全局 KPI/status strip。
  - 系统自检页保留全局状态和 4 个指标卡。
  - Agent 与工具页首屏顺序为注册表优先。
  - Agent 注册表和调用日志均出现可复制 `TechnicalId` 展示单元。

## 设计取舍

本轮刻意没有做以下事情:

- 没有引入 Ant Design、shadcn 或其他 UI 框架。当前问题主要是设计系统执行不一致,不是缺框架。
- 没有继续添加新主题色或装饰渐变。主题仍保持克制企业蓝,状态色只用于 success / warning / danger。
- 没有把 Agent 创建/密钥/轮换立即改成 Drawer。列表优先已经修复首屏主次;抽屉化属于下一轮交互重构。
- 没有全量重构表格密度、标题层级和宽屏表单布局。空状态已完成首批组件化与浏览器实测,但更大范围的页面级排版仍需后续整理。

## 剩余风险与建议

建议 Claude 复审时重点看:

1. `TechnicalId` 是否应该继续下沉为独立组件文件,避免 `App.tsx` 继续膨胀。
2. Agent 注册表操作列仍是双按钮,在数据量更大时可继续改为单一操作菜单。
3. 当前测试禁止组件层裸 `rgba()`,但允许 token 内定义 rgba;这个边界是否足够。
4. `vd-7` 空状态已完成首批统一组件化;标题层级、表格密度还没有完整闭环。
5. `vd-8` 宽屏阅读宽度和 8pt 网格还需要下一轮页面级整理。

## 复审建议

建议按以下顺序审查:

1. 先读 `frontend/tests/styleTheme.test.mjs`,确认回归约束是否覆盖设计系统底线。
2. 再读 `frontend/src/styles.css` 和 `frontend/src/styles/permission-workbench.css`,确认 token 使用是否合理。
3. 再读 `frontend/src/App.tsx` 中 `showWorkspaceTelemetry`、registry case、`TechnicalId` 和 Trace/Agent 接入点。
4. 最后跑浏览器复测,重点看 846px 和 375px 是否符合 B 端生产系统的可用性要求。

---

## 独立复审记录(Claude,2026-06-10)

复审方法与提出 finding 时一致:回归测试实跑 + 4174 preview 接 9090 真实后端做 DOM 度量,不采信自述。

### 复审结论:**通过**。vd-1~vd-6 的修复全部实测属实,vd-7/vd-8 如实标注未完成。

| 验收项 | 审查时实测 | 修复后实测 | 判定 |
|--------|-----------|-----------|------|
| 字重档位(DOM 计算值) | 15 档 | **3 档(400/600/700)**;CSS 源码仅声明 600×14、700×27 | ✅ |
| 组件层硬编码色值 | styles.css 161 处 / workbench 107 处 | styles.css 49 处全部位于 `:root` token 定义内,**组件层 0**;workbench **0 hex、0 裸 rgba** | ✅ |
| 文字色种数(DOM) | 22 种 | **10 种** | ✅ 大幅收敛(离 4-5 种理想值还有距离,可下轮继续) |
| 主按钮 | 34px 高、`1px 6px` padding | **36px 高、`0 16px` padding**,命中 `--control-height`/`--space-4` | ✅ |
| KPI 条范围 | 5 页复读 | 调用日志/Agent 与工具页**无 KPI**,系统自检页**保留** | ✅ |
| Agent 页顺序 | 创建表单占首屏 | 首屏顺序:**注册表 → 创建 Agent → 创建密钥** | ✅ |
| TechnicalId | ID 直铺无策略 | 中段省略(`agt_7b6edfbe...4bffcbcc`)+ `--mono-font`(SFMono)等宽 + 带 aria-label 复制按钮;Agent 页 16 处、调用日志 24 处接入 | ✅ |
| 回归测试 | — | styleTheme **17/17**,全量 **96/96**,build 通过 | ✅ |

### 复审中的一个未决项:focus-visible 实机确认受测试手段限制

CSS 规则齐全且正确(`:where(button,input,select,textarea,summary):focus-visible` + 审批/访问表单四组规则均命中 `var(--shadow-focus)`),回归测试也已覆盖。但 preview 里用脚本 `el.focus()` 无法触发 Chromium 的 `:focus-visible` 启发式(合成 KeyboardEvent 不移动焦点),复审无法亲眼确认键盘 Tab 时焦点环渲染。**规则层判定通过,建议夏天在浏览器里按一次 Tab 做 10 秒人工确认**——这是验证手段的边界,不是修复缺陷的证据。

### 对报告中 5 个待决问题的答复

1. **TechnicalId 下沉独立组件**:同意,下轮移到 `frontend/src/components/TechnicalId.tsx`——App.tsx 还在 4000+ 行,新增单元不应再进主文件。
2. **Agent 操作列双按钮 → 单一菜单**:同意,归入 vd-5 后续批次(实测行高仍 81px,密度问题同批处理)。
3. **"token 内允许 rgba、组件层禁止"的边界**:足够。alpha 值本来就该住在 token 里,现行测试规则方向正确,不必收紧。
4. **vd-7(空状态/行高/标题层级)**:空状态首批已闭环,不再是单行灰字;浏览器实测访问画像页 3 个空态均有图标、标题/说明结构和稳定高度。表格行高与标题层级仍留后续 P1。
5. **vd-8(宽屏阅读宽度/8pt)**:确认未闭环。下轮 P2,建议与 vd-7 同批做页面级整理。
